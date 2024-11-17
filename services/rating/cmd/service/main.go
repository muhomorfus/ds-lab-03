package main

import (
	"context"
	"fmt"
	"github.com/jmoiron/sqlx"
	"github.com/kelseyhightower/envconfig"
	"github.com/labstack/echo/v4"
	_ "github.com/lib/pq"
	"github.com/muhomorfus/ds-lab-02/services/rating/deployments/migrations"
	"github.com/muhomorfus/ds-lab-02/services/rating/internal/generated"
	"github.com/muhomorfus/ds-lab-02/services/rating/internal/openapi"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	if err := run(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func run() error {
	var cfg config
	if err := envconfig.Process("", &cfg); err != nil {
		return fmt.Errorf("read config: %w", err)
	}

	db, err := sqlx.Connect("postgres", cfg.dsn())
	if err != nil {
		return fmt.Errorf("connect to db: %w", err)
	}

	if err := migrations.Migrate(db); err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}

	server := openapi.New(db)
	router := echo.New()
	generated.RegisterHandlers(router, generated.NewStrictHandler(server, nil))

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	go func() {
		<-ctx.Done()

		_ = db.Close()
		_ = router.Close()
	}()

	if err := router.Start(cfg.listerAddress()); err != nil {
		return fmt.Errorf("listen http server: %w", err)
	}

	return nil
}

type config struct {
	PostgresHost     string `envconfig:"PGHOST" required:"true"`
	PostgresPort     int    `envconfig:"PGPORT" required:"true"`
	PostgresUser     string `envconfig:"PGUSER" required:"true"`
	PostgresPassword string `envconfig:"PGPASSWORD" required:"true"`
	PostgresDB       string `envconfig:"PGDB" required:"true"`
	PostgresSSL      bool   `envconfig:"PGSSL" default:"false"`
	Port             string `envconfig:"PORT" required:"true"`
}

func (c config) dsn() string {
	sslMode := ""
	if !c.PostgresSSL {
		sslMode = "sslmode=disable"
	}

	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s %s", c.PostgresHost, c.PostgresPort, c.PostgresUser, c.PostgresPassword, c.PostgresDB, sslMode)
}

func (c config) listerAddress() string {
	return fmt.Sprintf("0.0.0.0:%s", c.Port)
}
