package migrations

import (
	"embed"
	"fmt"
	"github.com/jmoiron/sqlx"
	"github.com/pressly/goose/v3"
)

//go:embed *.sql
var migrationFiles embed.FS

func Migrate(db *sqlx.DB) error {
	goose.SetBaseFS(migrationFiles)

	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("set goose dialect: %w", err)
	}

	if err := goose.Up(db.DB, "."); err != nil {
		return fmt.Errorf("up migrations: %w", err)
	}

	return nil
}
