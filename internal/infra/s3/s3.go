package s3

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// NewS3Client creates an S3 client for AWS (production use)
// It uses standard AWS credential chain: env vars, config file, or IAM role
func NewS3Client(region string) (*s3.Client, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(region),
	)
	if err != nil {
		return nil, err
	}

	return s3.NewFromConfig(cfg), nil
}

// NewS3ClientWithEndpoint creates an S3 client with a custom endpoint
// Useful for testing with local S3-compatible services or custom endpoints
func NewS3ClientWithEndpoint(region, endpoint string, usePathStyle bool, accessKey, secretKey string) (*s3.Client, error) {
	opts := []func(*config.LoadOptions) error{
		config.WithRegion(region),
	}

	// Use static credentials if provided, otherwise use default chain
	if accessKey != "" && secretKey != "" {
		opts = append(opts, config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(accessKey, secretKey, ""),
		))
	}

	cfg, err := config.LoadDefaultConfig(context.TODO(), opts...)
	if err != nil {
		return nil, err
	}

	return s3.NewFromConfig(cfg, func(o *s3.Options) {
		if endpoint != "" {
			o.BaseEndpoint = aws.String(endpoint)
		}
		o.UsePathStyle = usePathStyle
	}), nil
}
