package main

import (
	"context"
	"fmt"

	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/kms"
)

// connectKMS creates an AWS KMS client using the default credential chain.
func connectKMS(ctx context.Context, region string) (*kms.Client, error) {
	cfg, err := awsConfig.LoadDefaultConfig(ctx, awsConfig.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("load AWS config: %w", err)
	}
	return kms.NewFromConfig(cfg), nil
}
