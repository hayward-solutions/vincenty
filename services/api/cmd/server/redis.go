package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/sitaware/api/internal/config"
)

// connectRedis establishes a connection to Redis with retry logic.
func connectRedis(ctx context.Context, cfg config.RedisConfig) (*redis.Client, error) {
	opts := &redis.Options{
		Addr:     cfg.Addr(),
		Password: cfg.Password,
		DB:       0,
	}
	if cfg.TLS {
		opts.TLSConfig = &tls.Config{}
	}
	rdb := redis.NewClient(opts)

	maxRetries := 10
	for i := range maxRetries {
		if err := rdb.Ping(ctx).Err(); err == nil {
			slog.Info("connected to Redis", "addr", cfg.Addr())
			return rdb, nil
		}

		slog.Warn("waiting for Redis",
			"attempt", i+1,
			"max", maxRetries,
		)
		time.Sleep(2 * time.Second)
	}

	return nil, fmt.Errorf("failed to connect to Redis after %d attempts", maxRetries)
}
