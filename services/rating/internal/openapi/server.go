package openapi

import (
	"context"
	"fmt"
	"github.com/jmoiron/sqlx"
	"github.com/muhomorfus/ds-lab-02/services/rating/internal/generated"
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

func (s *Server) Get(ctx context.Context, request generated.GetRequestObject) (generated.GetResponseObject, error) {
	logger := slog.With("handler", "Get")
	stars, err := s.get(ctx, request.Params.XUserName)
	if err != nil {
		logger.Error("get user rating", "error", err)
		return nil, fmt.Errorf("get user rating: %w", err)
	}

	return generated.Get200JSONResponse{
		Stars: stars,
	}, nil
}

func (s *Server) SaveViolations(ctx context.Context, request generated.SaveViolationsRequestObject) (generated.SaveViolationsResponseObject, error) {
	logger := slog.With("handler", "SaveViolations")

	stars, err := s.get(ctx, request.Params.XUserName)
	if err != nil {
		logger.Error("get user rating", "error", err)
		return nil, fmt.Errorf("get user rating: %w", err)
	}

	if request.Params.Count > 0 {
		stars -= 10 * request.Params.Count
	} else {
		stars++
	}

	if err := s.save(ctx, request.Params.XUserName, stars); err != nil {
		logger.Error("save user rating", "error", err)
		return nil, fmt.Errorf("save user rating: %w", err)
	}

	return generated.SaveViolations204Response{}, nil
}

func (s *Server) get(ctx context.Context, username string) (int, error) {
	query := `select stars from rating where username = $1`
	var stars []int
	if err := s.db.SelectContext(ctx, &stars, query, username); err != nil {
		return -1, fmt.Errorf("select user rating: %w", err)
	}

	if len(stars) > 0 {
		return stars[0], nil
	}

	query = `insert into rating (username, stars) values ($1, $2)`
	if _, err := s.db.ExecContext(ctx, query, username, 1); err != nil {
		return -1, fmt.Errorf("insert rating: %w", err)
	}

	return 1, nil
}

func (s *Server) save(ctx context.Context, username string, stars int) error {
	if stars > 100 {
		stars = 100
	}

	if stars < 1 {
		stars = 1
	}

	query := `update rating set stars = $2 where username = $1`
	if _, err := s.db.ExecContext(ctx, query, username, stars); err != nil {
		return fmt.Errorf("update rating: %w", err)
	}

	return nil
}
