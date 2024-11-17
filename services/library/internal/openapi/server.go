package openapi

import (
	"context"
	"fmt"
	"github.com/jmoiron/sqlx"
	"github.com/muhomorfus/ds-lab-02/services/library/internal/generated"
	"github.com/samber/lo"
	"log/slog"
)

type Server struct {
	db *sqlx.DB
}

func New(db *sqlx.DB) *Server {
	return &Server{db: db}
}

func (s *Server) Health(ctx context.Context, request generated.HealthRequestObject) (generated.HealthResponseObject, error) {
	return generated.Health200Response{}, nil
}

func (s *Server) GetBook(ctx context.Context, request generated.GetBookRequestObject) (generated.GetBookResponseObject, error) {
	logger := slog.With("handler", "GetBook")
	query := `select * from books where book_uid = $1`

	var books []book
	if err := s.db.SelectContext(ctx, &books, query, request.BookUid); err != nil {
		logger.Error("select books from db", "error", err)
		return nil, fmt.Errorf("select book from db: %w", err)
	}

	if len(books) == 0 {
		logger.Error("book not found")
		return nil, fmt.Errorf("book not found")
	}

	return generated.GetBook200JSONResponse{
		Author:  books[0].Author,
		BookUid: books[0].BookUID,
		Genre:   books[0].Genre,
		Name:    books[0].Name,
	}, nil
}

func (s *Server) ListLibraries(ctx context.Context, request generated.ListLibrariesRequestObject) (generated.ListLibrariesResponseObject, error) {
	logger := slog.With("handler", "ListLibraries")
	query := pagination(`select * from library where city = $1`, request.Params.Page, request.Params.Size)

	var libraries []library
	if err := s.db.SelectContext(ctx, &libraries, query, request.Params.City); err != nil {
		logger.Error("select libraries from db", "error", err)
		return nil, fmt.Errorf("select libraries from db: %w", err)
	}

	query = `select count(*) from library where city = $1`
	var count int
	if err := s.db.QueryRow(query, request.Params.City).Scan(&count); err != nil {
		logger.Error("select count from db", "error", err)
		return nil, fmt.Errorf("select count from db: %w", err)
	}

	return generated.ListLibraries200JSONResponse{
		Items: lo.Map(libraries, func(item library, _ int) generated.LibraryResponse {
			return generated.LibraryResponse{
				Address:    item.Address,
				City:       item.City,
				LibraryUid: item.LibraryUID,
				Name:       item.Name,
			}
		}),
		Page:          request.Params.Page,
		PageSize:      request.Params.Size,
		TotalElements: count,
	}, nil
}

func (s *Server) GetLibrary(ctx context.Context, request generated.GetLibraryRequestObject) (generated.GetLibraryResponseObject, error) {
	logger := slog.With("handler", "GetLibrary")
	query := `select * from library where library_uid = $1`

	var libraries []library
	if err := s.db.SelectContext(ctx, &libraries, query, request.LibraryUid); err != nil {
		logger.Error("select library from db", "error", err)
		return nil, fmt.Errorf("select library from db: %w", err)
	}

	if len(libraries) == 0 {
		logger.Error("library not found")
		return nil, fmt.Errorf("library not found")
	}

	return generated.GetLibrary200JSONResponse{
		Address:    libraries[0].Address,
		City:       libraries[0].City,
		LibraryUid: libraries[0].LibraryUID,
		Name:       libraries[0].Name,
	}, nil
}

func (s *Server) ListBooks(ctx context.Context, request generated.ListBooksRequestObject) (generated.ListBooksResponseObject, error) {
	logger := slog.With("handler", "ListBooks")
	dontShowNotAvailableFilter := " and lb.available_count > 0"
	if lo.FromPtr(request.Params.ShowAll) {
		dontShowNotAvailableFilter = ""
	}

	query := pagination(`
	select b.*, lb.available_count from 
		books b 
			join library_books lb on b.id = lb.book_id 
			join library l on l.id = lb.library_id 
		where l.library_uid = $1`+dontShowNotAvailableFilter,
		request.Params.Page,
		request.Params.Size,
	)

	var books []libraryBook
	if err := s.db.SelectContext(ctx, &books, query, request.LibraryUid); err != nil {
		logger.Error("select books from db", "error", err)
		return nil, fmt.Errorf("select books from db: %w", err)
	}

	query = `select count(*) from
		books b
		join library_books lb on b.id = lb.book_id
		join library l on l.id = lb.library_id
		where l.library_uid = $1` + dontShowNotAvailableFilter
	var count int
	if err := s.db.QueryRow(query, request.LibraryUid).Scan(&count); err != nil {
		logger.Error("select count from db", "error", err)
		return nil, fmt.Errorf("select count from db: %w", err)
	}

	return generated.ListBooks200JSONResponse{
		Items: lo.Map(books, func(item libraryBook, _ int) generated.LibraryBookResponse {
			return generated.LibraryBookResponse{
				Author:         item.Author,
				AvailableCount: item.AvailableCount,
				BookUid:        item.BookUID,
				Condition:      generated.LibraryBookResponseCondition(item.Condition),
				Genre:          item.Genre,
				Name:           item.Name,
			}
		}),
		Page:          request.Params.Page,
		PageSize:      request.Params.Size,
		TotalElements: count,
	}, nil
}

func (s *Server) TakeBook(ctx context.Context, request generated.TakeBookRequestObject) (generated.TakeBookResponseObject, error) {
	logger := slog.With("handler", "TakeBook")
	query := `select l.id as library_id, b.id as book_id, lb.available_count from
		books b
		join library_books lb on b.id = lb.book_id
		join library l on l.id = lb.library_id
		where l.library_uid = $1 and b.book_uid = $2`

	var libraryBooks []libraryBookRaw
	if err := s.db.SelectContext(ctx, &libraryBooks, query, request.LibraryUid, request.BookUid); err != nil {
		logger.Error("select library books from db", "error", err)
		return nil, fmt.Errorf("select library books from db: %w", err)
	}

	if len(libraryBooks) == 0 {
		logger.Warn("no book presented in library")
		return generated.TakeBook400JSONResponse{
			Message: "book not presented in this library",
		}, nil
	}

	if libraryBooks[0].AvailableCount == 0 {
		logger.Warn("0 available books in library")
		return generated.TakeBook400JSONResponse{
			Message: "there is 0 available books in library",
		}, nil
	}

	query = `update library_books set available_count = available_count - 1 where library_id = $1 and book_id = $2`
	if _, err := s.db.ExecContext(ctx, query, libraryBooks[0].LibraryID, libraryBooks[0].BookID); err != nil {
		logger.Error("update library books table in db", "error", err)
		return nil, fmt.Errorf("update library books table in db: %w", err)
	}

	return generated.TakeBook204Response{}, nil
}

func (s *Server) ReturnBook(ctx context.Context, request generated.ReturnBookRequestObject) (generated.ReturnBookResponseObject, error) {
	logger := slog.With("handler", "ReturnBook")
	query := `select l.id as library_id, b.id as book_id, lb.available_count from
		books b
		join library_books lb on b.id = lb.book_id
		join library l on l.id = lb.library_id
		where l.library_uid = $1 and b.book_uid = $2`

	var libraryBooks []libraryBookRaw
	if err := s.db.SelectContext(ctx, &libraryBooks, query, request.LibraryUid, request.BookUid); err != nil {
		logger.Error("select library books from db", "error", err)
		return nil, fmt.Errorf("select library books from db: %w", err)
	}

	if len(libraryBooks) == 0 {
		logger.Warn("no book presented in library")
		return generated.ReturnBook400JSONResponse{
			Message: "book not presented in this library",
		}, nil
	}

	query = `update library_books set available_count = available_count + 1 where library_id = $1 and book_id = $2`
	if _, err := s.db.ExecContext(ctx, query, libraryBooks[0].LibraryID, libraryBooks[0].BookID); err != nil {
		logger.Error("update library books table in db", "error", err)
		return nil, fmt.Errorf("update library books table in db: %w", err)
	}

	query = `select * from books where book_uid = $1`

	var books []book
	if err := s.db.SelectContext(ctx, &books, query, request.BookUid); err != nil {
		logger.Error("select books from db", "error", err)
		return nil, fmt.Errorf("select book from db: %w", err)
	}

	if len(books) == 0 {
		logger.Error("book not found")
		return nil, fmt.Errorf("book not found")
	}

	return generated.ReturnBook200JSONResponse{
		Violation: books[0].Condition != string(request.Body.Condition),
	}, nil
}

func pagination(query string, page, pageSize *int) string {
	if pageSize == nil {
		return query
	}

	if page == nil {
		page = new(int)
		*page = 1
	}

	return fmt.Sprintf("%s limit %d offset %d", query, (*pageSize), (*pageSize)*(*page-1))
}
