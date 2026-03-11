package storage

import (
	"context"
	"io"
	"time"
)

// Storage abstracts object‐storage operations (S3, MinIO, local FS, etc.).
// Services accept this interface so that tests can substitute lightweight
// in‐memory mocks.
type Storage interface {
	Upload(ctx context.Context, key string, body io.Reader, contentType string, size int64) error
	Download(ctx context.Context, key string) (io.ReadCloser, string, int64, error)
	GetPresignedURL(ctx context.Context, key string, expiry time.Duration) (string, error)
	Delete(ctx context.Context, key string) error
}

// Compile-time check: StorageService implements Storage.
var _ Storage = (*StorageService)(nil)
