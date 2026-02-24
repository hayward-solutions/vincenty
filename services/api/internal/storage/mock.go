package storage

import (
	"context"
	"io"
	"time"
)

// MockStorage is an in-memory mock implementation of the Storage interface
// for unit testing. Each method delegates to a configurable function field.
type MockStorage struct {
	UploadFn          func(ctx context.Context, key string, body io.Reader, contentType string, size int64) error
	DownloadFn        func(ctx context.Context, key string) (io.ReadCloser, string, int64, error)
	GetPresignedURLFn func(ctx context.Context, key string, expiry time.Duration) (string, error)
	DeleteFn          func(ctx context.Context, key string) error
}

func (m *MockStorage) Upload(ctx context.Context, key string, body io.Reader, contentType string, size int64) error {
	return m.UploadFn(ctx, key, body, contentType, size)
}

func (m *MockStorage) Download(ctx context.Context, key string) (io.ReadCloser, string, int64, error) {
	return m.DownloadFn(ctx, key)
}

func (m *MockStorage) GetPresignedURL(ctx context.Context, key string, expiry time.Duration) (string, error) {
	return m.GetPresignedURLFn(ctx, key, expiry)
}

func (m *MockStorage) Delete(ctx context.Context, key string) error {
	return m.DeleteFn(ctx, key)
}

// Compile-time check: MockStorage implements Storage.
var _ Storage = (*MockStorage)(nil)
