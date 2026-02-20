package main

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sitaware/api/internal/config"
)

// connectDB establishes a connection pool to PostgreSQL with retry logic.
func connectDB(ctx context.Context, cfg config.DBConfig) (*pgxpool.Pool, error) {
	var pool *pgxpool.Pool
	var err error

	maxRetries := 10
	for i := range maxRetries {
		pool, err = pgxpool.New(ctx, cfg.DSN())
		if err == nil {
			if pingErr := pool.Ping(ctx); pingErr == nil {
				slog.Info("connected to PostgreSQL",
					"host", cfg.Host,
					"port", cfg.Port,
					"database", cfg.Name,
				)
				return pool, nil
			}
		}

		slog.Warn("waiting for PostgreSQL",
			"attempt", i+1,
			"max", maxRetries,
			"error", err,
		)
		time.Sleep(2 * time.Second)
	}

	return nil, fmt.Errorf("failed to connect to PostgreSQL after %d attempts: %w", maxRetries, err)
}
