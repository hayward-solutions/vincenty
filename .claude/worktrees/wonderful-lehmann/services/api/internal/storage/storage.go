package storage

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// StorageService wraps an S3-compatible client for file upload/download.
type StorageService struct {
	client *s3.Client
	bucket string
}

// NewStorageService creates a new StorageService.
func NewStorageService(client *s3.Client, bucket string) *StorageService {
	return &StorageService{client: client, bucket: bucket}
}

// Upload stores an object in the bucket.
func (s *StorageService) Upload(ctx context.Context, key string, body io.Reader, contentType string, size int64) error {
	input := &s3.PutObjectInput{
		Bucket:        aws.String(s.bucket),
		Key:           aws.String(key),
		Body:          body,
		ContentType:   aws.String(contentType),
		ContentLength: aws.Int64(size),
	}

	_, err := s.client.PutObject(ctx, input)
	if err != nil {
		return fmt.Errorf("upload to %s: %w", key, err)
	}
	return nil
}

// GetPresignedURL returns a time-limited download URL for an object.
func (s *StorageService) GetPresignedURL(ctx context.Context, key string, expiry time.Duration) (string, error) {
	presigner := s3.NewPresignClient(s.client)
	req, err := presigner.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	}, s3.WithPresignExpires(expiry))
	if err != nil {
		return "", fmt.Errorf("presign %s: %w", key, err)
	}
	return req.URL, nil
}

// Download retrieves an object from the bucket, returning its body as an
// io.ReadCloser along with the content type and content length. The caller
// is responsible for closing the returned reader.
func (s *StorageService) Download(ctx context.Context, key string) (io.ReadCloser, string, int64, error) {
	out, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, "", 0, fmt.Errorf("download %s: %w", key, err)
	}

	contentType := "application/octet-stream"
	if out.ContentType != nil {
		contentType = *out.ContentType
	}

	var contentLength int64
	if out.ContentLength != nil {
		contentLength = *out.ContentLength
	}

	return out.Body, contentType, contentLength, nil
}

// Delete removes an object from the bucket.
func (s *StorageService) Delete(ctx context.Context, key string) error {
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("delete %s: %w", key, err)
	}
	return nil
}
