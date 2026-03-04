// Package s3 provides S3 URI parsing and file download utilities
// for the CLI tool. When running on ECS Fargate, credentials are
// obtained automatically from the task role via the default
// credential chain.
package s3

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// IsS3URI returns true if the path is an S3 URI (s3://bucket/key).
func IsS3URI(path string) bool {
	return strings.HasPrefix(path, "s3://")
}

// ParseURI extracts the bucket and key from an S3 URI.
// Format: s3://bucket/key
func ParseURI(uri string) (bucket, key string, err error) {
	if !IsS3URI(uri) {
		return "", "", fmt.Errorf("not an S3 URI: %q", uri)
	}
	trimmed := strings.TrimPrefix(uri, "s3://")
	parts := strings.SplitN(trimmed, "/", 2)
	if len(parts) < 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("invalid S3 URI %q: expected s3://bucket/key", uri)
	}
	return parts[0], parts[1], nil
}

// Download fetches an object from S3 and saves it to a temporary file.
// The caller is responsible for removing the file when done.
// Returns the path to the downloaded temporary file.
func Download(ctx context.Context, uri string) (string, error) {
	bucket, key, err := ParseURI(uri)
	if err != nil {
		return "", err
	}

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return "", fmt.Errorf("load AWS config: %w", err)
	}

	client := s3.NewFromConfig(cfg)

	result, err := client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: &bucket,
		Key:    &key,
	})
	if err != nil {
		return "", fmt.Errorf("get s3://%s/%s: %w", bucket, key, err)
	}
	defer result.Body.Close()

	// Preserve the file extension so track.Load detects the format.
	ext := filepath.Ext(key)
	if ext == "" {
		ext = ".gpx"
	}

	tmpFile, err := os.CreateTemp("", "sitaware-*"+ext)
	if err != nil {
		return "", fmt.Errorf("create temp file: %w", err)
	}

	if _, err := io.Copy(tmpFile, result.Body); err != nil {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
		return "", fmt.Errorf("download to temp file: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		os.Remove(tmpFile.Name())
		return "", fmt.Errorf("close temp file: %w", err)
	}

	return tmpFile.Name(), nil
}
