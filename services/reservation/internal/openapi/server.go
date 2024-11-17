package openapi

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/muhomorfus/ds-lab-02/services/reservation/internal/generated"
	"github.com/samber/lo"
	"log/slog"
	"time"
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

func (s *Server) Get(ctx context.Context, request generated.GetRequestObject) (generated.GetResponseObject, error) {
	logger := slog.With("handler", "Get")
	query := `select * from reservation where username = $1 and reservation_uid = $2`

	var reservations []reservation
	if err := s.db.SelectContext(ctx, &reservations, query, request.Params.XUserName, request.ReservationUid); err != nil {
		logger.Error("select reservation from db", "error", err)
		return nil, fmt.Errorf("select reservtion from db: %w", err)
	}

	if len(reservations) == 0 {
		return generated.Get404JSONResponse{
			Message: "reservation not found",
		}, nil
	}

	return generated.Get200JSONResponse{
		BookUid:        reservations[0].BookUid,
		LibraryUid:     reservations[0].LibraryUid,
		ReservationUid: reservations[0].ReservationUid,
		StartDate:      reservations[0].StartDate.Format(time.DateOnly),
		Status:         generated.BookReservationResponseStatus(reservations[0].Status),
		TillDate:       reservations[0].TillDate.Format(time.DateOnly),
	}, nil
}

func (s *Server) List(ctx context.Context, request generated.ListRequestObject) (generated.ListResponseObject, error) {
	logger := slog.With("handler", "List")

	query := `select * from reservation where username = $1`

	var reservations []reservation
	if err := s.db.SelectContext(ctx, &reservations, query, request.Params.XUserName); err != nil {
		logger.Error("select reservations from db", "error", err)
		return nil, fmt.Errorf("select reservtions from db: %w", err)
	}

	return generated.List200JSONResponse(lo.Map(reservations, func(r reservation, _ int) generated.BookReservationResponse {
		return generated.BookReservationResponse{
			BookUid:        r.BookUid,
			LibraryUid:     r.LibraryUid,
			ReservationUid: r.ReservationUid,
			StartDate:      r.StartDate.Format(time.DateOnly),
			Status:         generated.BookReservationResponseStatus(r.Status),
			TillDate:       r.TillDate.Format(time.DateOnly),
		}
	})), nil
}

func (s *Server) Create(ctx context.Context, request generated.CreateRequestObject) (generated.CreateResponseObject, error) {
	logger := slog.With("handler", "Create")

	now := time.Now()
	till, err := time.Parse(time.DateOnly, request.Body.TillDate)
	if err != nil {
		logger.Error("parse time", "error", err)
		return generated.Create400JSONResponse{
			Errors:  nil,
			Message: "invalid till date format",
		}, nil
	}

	r := reservation{
		BookUid:        request.Body.BookUid,
		LibraryUid:     request.Body.LibraryUid,
		ReservationUid: uuid.New(),
		StartDate:      now,
		Status:         rented,
		TillDate:       till,
		Username:       request.Params.XUserName,
	}

	query := `insert into reservation 
    (reservation_uid, username, book_uid, library_uid, status, start_date, till_date)
    values (:reservation_uid, :username, :book_uid, :library_uid, :status, :start_date, :till_date)`

	if _, err := s.db.NamedExecContext(ctx, query, r); err != nil {
		logger.Error("create reservation", "error", err)
		return nil, fmt.Errorf("create reservation: %w", err)
	}

	return generated.Create200JSONResponse{
		BookUid:        r.BookUid,
		LibraryUid:     r.LibraryUid,
		ReservationUid: r.ReservationUid,
		StartDate:      r.StartDate.Format(time.DateOnly),
		Status:         generated.TakeBookResponseStatus(r.Status),
		TillDate:       r.TillDate.Format(time.DateOnly),
	}, nil
}

func (s *Server) Finish(ctx context.Context, request generated.FinishRequestObject) (generated.FinishResponseObject, error) {
	logger := slog.With("handler", "Finish")

	query := `select * from reservation where reservation_uid = $1 and username = $2`

	var reservations []reservation
	if err := s.db.SelectContext(ctx, &reservations, query, request.ReservationUid, request.Params.XUserName); err != nil {
		logger.Error("select reservations from db", "error", err)
		return nil, fmt.Errorf("select reservtions from db: %w", err)
	}

	if len(reservations) == 0 {
		logger.Error("reservations not found")
		return generated.Finish404JSONResponse{
			Message: "reservation not found",
		}, nil
	}

	status := returned
	date, err := time.Parse(time.DateOnly, request.Body.Date)
	if err != nil {
		logger.Error("parse return date", "error", err)
		return nil, fmt.Errorf("invalid date format: %w", err)
	}

	if date.After(reservations[0].TillDate) {
		status = expired
	}

	query = `update reservation set status = $1 where reservation_uid = $2`
	if _, err := s.db.ExecContext(ctx, query, status, request.ReservationUid); err != nil {
		logger.Error("update reservation status", "error", err)
		return nil, fmt.Errorf("update reservation status: %w", err)
	}

	return generated.Finish200JSONResponse{
		Violation: status == expired,
	}, nil
}
