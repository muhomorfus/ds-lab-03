package openapi

import (
	"context"
	"fmt"
	"github.com/muhomorfus/ds-lab-02/services/gateway/internal/clients/library"
	"github.com/muhomorfus/ds-lab-02/services/gateway/internal/clients/rating"
	"github.com/muhomorfus/ds-lab-02/services/gateway/internal/clients/reservation"
	"github.com/muhomorfus/ds-lab-02/services/gateway/internal/generated"
	"github.com/samber/lo"
	"log/slog"
	"net/http"
)

type Server struct {
	library     *library.ClientWithResponses
	reservation *reservation.ClientWithResponses
	rating      *rating.ClientWithResponses
}

func New(library *library.ClientWithResponses, reservation *reservation.ClientWithResponses, rating *rating.ClientWithResponses) *Server {
	return &Server{library: library, reservation: reservation, rating: rating}
}

func (s *Server) ListLibraries(ctx context.Context, request generated.ListLibrariesRequestObject) (generated.ListLibrariesResponseObject, error) {
	logger := slog.With("handler", "ListLibraries")

	resp, err := s.library.ListLibrariesWithResponse(ctx, &library.ListLibrariesParams{
		Page: request.Params.Page,
		Size: request.Params.Size,
		City: request.Params.City,
	})
	if err != nil {
		logger.Error("list libraries", "error", err)
		return nil, fmt.Errorf("list libraries: %w", err)
	}

	if resp.JSON200 == nil {
		logger.Error("list libraries unknown status", "status", resp.StatusCode())
		return nil, fmt.Errorf("list libraries: %s", string(resp.Body))
	}

	return generated.ListLibraries200JSONResponse{
		Items: lo.Map(resp.JSON200.Items, func(item library.LibraryResponse, _ int) generated.LibraryResponse {
			return generated.LibraryResponse(item)
		}),
		Page:          resp.JSON200.Page,
		PageSize:      resp.JSON200.PageSize,
		TotalElements: resp.JSON200.TotalElements,
	}, nil
}

func (s *Server) ListBooks(ctx context.Context, request generated.ListBooksRequestObject) (generated.ListBooksResponseObject, error) {
	logger := slog.With("handler", "ListBooks")

	resp, err := s.library.ListBooksWithResponse(ctx, request.LibraryUid, &library.ListBooksParams{
		Page:    request.Params.Page,
		Size:    request.Params.Size,
		ShowAll: request.Params.ShowAll,
	})
	if err != nil {
		logger.Error("list books", "error", err)
		return nil, fmt.Errorf("list books: %w", err)
	}

	if resp.JSON200 == nil {
		logger.Error("list books unknown status", "status", resp.StatusCode())
		return nil, fmt.Errorf("list books: %s", string(resp.Body))
	}

	return generated.ListBooks200JSONResponse{
		Items: lo.Map(resp.JSON200.Items, func(item library.LibraryBookResponse, _ int) generated.LibraryBookResponse {
			return generated.LibraryBookResponse{
				Author:         item.Author,
				AvailableCount: item.AvailableCount,
				BookUid:        item.BookUid,
				Condition:      generated.LibraryBookResponseCondition(item.Condition),
				Genre:          item.Genre,
				Name:           item.Name,
			}
		}),
		Page:          resp.JSON200.Page,
		PageSize:      resp.JSON200.PageSize,
		TotalElements: resp.JSON200.TotalElements,
	}, nil
}

func (s *Server) GetRating(ctx context.Context, request generated.GetRatingRequestObject) (generated.GetRatingResponseObject, error) {
	logger := slog.With("handler", "GetRating")

	resp, err := s.rating.GetWithResponse(ctx, &rating.GetParams{XUserName: request.Params.XUserName})
	if err != nil {
		logger.Error("get rating", "error", err)
		return nil, fmt.Errorf("get rating: %w", err)
	}

	if resp.JSON200 == nil {
		logger.Error("get rating unknown status", "status", resp.StatusCode())
		return nil, fmt.Errorf("get user rating: %s", string(resp.Body))
	}

	return generated.GetRating200JSONResponse{
		Stars: resp.JSON200.Stars,
	}, nil
}

func (s *Server) ListReservations(ctx context.Context, request generated.ListReservationsRequestObject) (generated.ListReservationsResponseObject, error) {
	logger := slog.With("handler", "ListReservations")

	resp, err := s.reservation.ListWithResponse(ctx, &reservation.ListParams{XUserName: request.Params.XUserName})
	if err != nil {
		logger.Error("list reservations", "error", err)
		return nil, fmt.Errorf("list reservations: %w", err)
	}

	if resp.JSON200 == nil {
		logger.Error("list reservations unknown status", "status", resp.StatusCode())
		return nil, fmt.Errorf("list reservations: %s", string(resp.Body))
	}

	var result []generated.BookReservationResponse
	for _, r := range *resp.JSON200 {
		book := generated.BookInfo{
			BookUid: r.BookUid,
		}

		bookResp, err := s.library.GetBookWithResponse(ctx, r.BookUid)
		if err == nil && bookResp.JSON200 != nil {
			book = generated.BookInfo{
				Author:  bookResp.JSON200.Author,
				BookUid: bookResp.JSON200.BookUid,
				Genre:   bookResp.JSON200.Genre,
				Name:    bookResp.JSON200.Name,
			}
		}

		lib := generated.LibraryResponse{
			LibraryUid: r.LibraryUid,
		}

		libraryResp, err := s.library.GetLibraryWithResponse(ctx, r.LibraryUid)
		if err == nil && libraryResp.JSON200 != nil {
			lib = generated.LibraryResponse{
				Address:    libraryResp.JSON200.Address,
				City:       libraryResp.JSON200.City,
				LibraryUid: libraryResp.JSON200.LibraryUid,
				Name:       libraryResp.JSON200.Name,
			}
		}

		result = append(result, generated.BookReservationResponse{
			Book:           book,
			Library:        lib,
			ReservationUid: r.ReservationUid,
			StartDate:      r.StartDate,
			Status:         generated.BookReservationResponseStatus(r.Status),
			TillDate:       r.TillDate,
		})
	}

	return generated.ListReservations200JSONResponse(result), nil
}

func (s *Server) TakeBook(ctx context.Context, request generated.TakeBookRequestObject) (generated.TakeBookResponseObject, error) {
	logger := slog.With("handler", "TakeBook")

	reservationResp, err := s.reservation.ListWithResponse(ctx, &reservation.ListParams{XUserName: request.Params.XUserName})
	if err != nil {
		logger.Error("get user reservation", "error", err)
		return nil, fmt.Errorf("get user reservation: %w", err)
	}

	if reservationResp.JSON200 == nil {
		logger.Error("get user reservation unknown response", "status", reservationResp.StatusCode())
		return nil, fmt.Errorf("get user reservation: empty response")
	}

	reserved := lo.Reduce(*reservationResp.JSON200, func(agg int, item reservation.BookReservationResponse, _ int) int {
		if item.Status == reservation.BookReservationResponseStatusRENTED {
			return agg + 1
		}

		return agg
	}, 0)

	ratingResp, err := s.rating.GetWithResponse(ctx, &rating.GetParams{XUserName: request.Params.XUserName})
	if err != nil {
		logger.Error("get user rating", "error", err)
		return nil, fmt.Errorf("get user rating: %w", err)
	}

	if ratingResp.JSON200 == nil {
		logger.Error("get user rating unknown response", "status", ratingResp.StatusCode())
		return nil, fmt.Errorf("get user rating: empty response")
	}

	canReserve := ratingResp.JSON200.Stars - reserved
	if canReserve < 0 {
		return generated.TakeBook400JSONResponse{
			Message: "too many taken books",
		}, nil
	}

	reservedResp, err := s.reservation.CreateWithResponse(ctx, &reservation.CreateParams{
		XUserName: request.Params.XUserName,
	}, reservation.CreateJSONRequestBody{
		BookUid:    request.Body.BookUid,
		LibraryUid: request.Body.LibraryUid,
		TillDate:   request.Body.TillDate,
	})
	if err != nil {
		logger.Error("reserve book", "error", err)
		return nil, fmt.Errorf("reserve book: %w", err)
	}

	if reservedResp.JSON400 != nil {
		return generated.TakeBook400JSONResponse{
			Message: reservedResp.JSON400.Message,
		}, nil
	}

	if reservedResp.JSON200 == nil {
		logger.Error("reserve book unknown status", "status", reservedResp.StatusCode())
		return nil, fmt.Errorf("reserve book: %s", string(reservedResp.Body))
	}

	bookResp, err := s.library.TakeBookWithResponse(ctx, request.Body.LibraryUid, request.Body.BookUid)
	if err != nil {
		logger.Error("take book", "error", err)
		return nil, fmt.Errorf("decrease book: %w", err)
	}

	if bookResp.JSON400 != nil {
		return generated.TakeBook400JSONResponse{
			Message: bookResp.JSON400.Message,
		}, nil
	}

	if bookResp.StatusCode() != http.StatusNoContent {
		logger.Error("take book unknown status", "status", bookResp.StatusCode())
		return nil, fmt.Errorf("decrease book: %s", string(bookResp.Body))
	}

	book := generated.BookInfo{
		BookUid: reservedResp.JSON200.BookUid,
	}

	bookRespInfo, err := s.library.GetBookWithResponse(ctx, reservedResp.JSON200.BookUid)
	if err == nil && bookRespInfo.JSON200 != nil {
		book = generated.BookInfo{
			Author:  bookRespInfo.JSON200.Author,
			BookUid: bookRespInfo.JSON200.BookUid,
			Genre:   bookRespInfo.JSON200.Genre,
			Name:    bookRespInfo.JSON200.Name,
		}
	}

	lib := generated.LibraryResponse{
		LibraryUid: reservedResp.JSON200.LibraryUid,
	}

	libraryResp, err := s.library.GetLibraryWithResponse(ctx, reservedResp.JSON200.LibraryUid)
	if err == nil && libraryResp.JSON200 != nil {
		lib = generated.LibraryResponse{
			Address:    libraryResp.JSON200.Address,
			City:       libraryResp.JSON200.City,
			LibraryUid: libraryResp.JSON200.LibraryUid,
			Name:       libraryResp.JSON200.Name,
		}
	}

	return generated.TakeBook200JSONResponse{
		Book:    book,
		Library: lib,
		Rating: generated.UserRatingResponse{
			Stars: ratingResp.JSON200.Stars,
		},
		ReservationUid: reservedResp.JSON200.ReservationUid,
		StartDate:      reservedResp.JSON200.StartDate,
		Status:         generated.TakeBookResponseStatus(reservedResp.JSON200.Status),
		TillDate:       reservedResp.JSON200.TillDate,
	}, nil
}

func (s *Server) ReturnBook(ctx context.Context, request generated.ReturnBookRequestObject) (generated.ReturnBookResponseObject, error) {
	logger := slog.With("handler", "ReturnBook")

	reservationResp, err := s.reservation.GetWithResponse(ctx, request.ReservationUid, &reservation.GetParams{XUserName: request.Params.XUserName})
	if err != nil {
		logger.Error("get user reservation", "error", err)
		return nil, fmt.Errorf("get user reservation: %w", err)
	}

	if reservationResp.JSON200 == nil {
		logger.Error("get user reservation unknown status", "status", reservationResp.StatusCode())
		return nil, fmt.Errorf("get user reservation: %s", string(reservationResp.Body))
	}

	violations := 0

	unreservedResp, err := s.reservation.FinishWithResponse(ctx, request.ReservationUid, &reservation.FinishParams{XUserName: request.Params.XUserName}, reservation.FinishJSONRequestBody{Date: request.Body.Date})
	if err != nil {
		logger.Error("finish reservation", "error", err)
		return nil, fmt.Errorf("finish reservation: %w", err)
	}

	if unreservedResp.JSON404 != nil {
		return generated.ReturnBook404JSONResponse{
			Message: unreservedResp.JSON404.Message,
		}, nil
	}

	if unreservedResp.JSON200 == nil {
		logger.Error("finish reservation unknown status", "status", unreservedResp.StatusCode())
		return nil, fmt.Errorf("finish reservation: %s", string(unreservedResp.Body))
	}

	if unreservedResp.JSON200.Violation {
		violations++
	}

	makeAvailableResp, err := s.library.ReturnBookWithResponse(ctx, reservationResp.JSON200.LibraryUid, reservationResp.JSON200.BookUid, library.ReturnBookJSONRequestBody{
		Condition: library.ReturnBookRequestCondition(request.Body.Condition),
	})
	if err != nil {
		logger.Error("return book", "error", err)
		return nil, fmt.Errorf("return book: %w", err)
	}

	if makeAvailableResp.JSON200 == nil {
		logger.Error("return book unknown status", "status", makeAvailableResp.StatusCode())
		return nil, fmt.Errorf("return book: %s", string(makeAvailableResp.Body))
	}

	if makeAvailableResp.JSON200.Violation {
		violations++
	}

	changeRatingResp, err := s.rating.SaveViolationsWithResponse(ctx, &rating.SaveViolationsParams{XUserName: request.Params.XUserName, Count: violations})
	if err != nil {
		logger.Error("save violations", "error", err)
		return nil, fmt.Errorf("save violations: %w", err)
	}

	if changeRatingResp.StatusCode() != http.StatusNoContent {
		logger.Error("save violations unknown status", "status", changeRatingResp.StatusCode())
		return nil, fmt.Errorf("save violations: %s", string(changeRatingResp.Body))
	}

	return generated.ReturnBook204Response{}, nil
}

func (s *Server) Health(ctx context.Context, request generated.HealthRequestObject) (generated.HealthResponseObject, error) {
	return generated.Health200Response{}, nil
}
