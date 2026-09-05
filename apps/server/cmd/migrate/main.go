package main

import (
	"context"
	"database/sql"
	"log/slog"
	"os"
	"time"

	"example.com/encounter/apps/server/internal/migrations"
	_ "github.com/jackc/pgx/v5/stdlib"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	if err := run(); err != nil {
		slog.Error("migration failed", "error", err)
		os.Exit(1)
	}
}

func run() error {
	db, err := sql.Open("pgx", os.Getenv("DATABASE_URL"))
	if err != nil {
		return err
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	if err := migrations.Up(ctx, db); err != nil {
		return err
	}
	slog.Info("migrations applied")
	return nil
}
