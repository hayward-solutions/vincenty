package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/vincenty/api/internal/config"
)

// connectRedis establishes a connection to Redis with retry logic.
// When cfg.Cluster is true it creates a ClusterClient that discovers
// topology via a single configuration endpoint (e.g. ElastiCache cluster
// mode). Otherwise it creates a standalone Client.
func connectRedis(ctx context.Context, cfg config.RedisConfig) (redis.UniversalClient, error) {
	var rdb redis.UniversalClient

	if cfg.Cluster {
		opts := &redis.ClusterOptions{
			Addrs:    []string{cfg.Addr()},
			Password: cfg.Password,
		}
		if cfg.TLS {
			opts.TLSConfig = &tls.Config{}
		}
		rdb = redis.NewClusterClient(opts)
	} else {
		opts := &redis.Options{
			Addr:     cfg.Addr(),
			Password: cfg.Password,
			DB:       0,
		}
		if cfg.TLS {
			opts.TLSConfig = &tls.Config{}
		}
		rdb = redis.NewClient(opts)
	}

	maxRetries := 10
	for i := range maxRetries {
		if err := rdb.Ping(ctx).Err(); err == nil {
			mode := "standalone"
			if cfg.Cluster {
				mode = "cluster"
			}
			slog.Info("connected to Redis", "addr", cfg.Addr(), "mode", mode)
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
