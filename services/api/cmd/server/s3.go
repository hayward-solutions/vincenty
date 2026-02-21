package main

import (
	"context"
	"log/slog"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/sitaware/api/internal/config"
)

// connectS3 creates an S3 client configured for either AWS S3 or Minio.
// When S3_ACCESS_KEY and S3_SECRET_KEY are set, static credentials are used
// (for Minio / local development). When they are empty, the default credential
// chain is used, which picks up the ECS task role on Fargate automatically.
func connectS3(ctx context.Context, cfg config.S3Config) (*s3.Client, error) {
	opts := []func(*awsconfig.LoadOptions) error{
		awsconfig.WithRegion(cfg.Region),
	}
	if cfg.AccessKey != "" && cfg.SecretKey != "" {
		opts = append(opts, awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(cfg.AccessKey, cfg.SecretKey, ""),
		))
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, err
	}

	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(cfg.Endpoint)
		o.UsePathStyle = cfg.UsePathStyle
	})

	slog.Info("connected to object storage",
		"endpoint", cfg.Endpoint,
		"bucket", cfg.Bucket,
	)

	return client, nil
}
