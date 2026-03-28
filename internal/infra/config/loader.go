package config

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/cheatsnake/icecube/internal/infra/kafka"
	"github.com/cheatsnake/icecube/internal/infra/postgres"
	imagestores3 "github.com/cheatsnake/icecube/internal/infra/s3"
	"github.com/cheatsnake/icecube/internal/service/imagestore"
	"github.com/cheatsnake/icecube/internal/service/jobstore"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Stores struct {
	JobStore           jobstore.Store
	ImageBlobStore     imagestore.BlobStore
	ImageMetadataStore imagestore.MetadataStore
	KafkaProducer      *kafka.Producer
}

func (c *Config) LoadStores(ctx context.Context, logger *slog.Logger) (*Stores, error) {
	stores := &Stores{}

	// Initialize database pool (shared between job store and metadata store)
	var dbPool *pgxpool.Pool
	var err error

	if c.Database.Type == "postgres" {
		dbPool, err = postgres.NewPool(c.Database.URI, logger)
		if err != nil {
			return nil, fmt.Errorf("database pool: %w", err)
		}
	}

	jobStore, err := c.loadJobStore(dbPool, logger)
	if err != nil {
		return nil, fmt.Errorf("job store: %w", err)
	}
	stores.JobStore = jobStore

	blobStore, err := c.loadBlobStore(logger)
	if err != nil {
		return nil, fmt.Errorf("blob store: %w", err)
	}
	stores.ImageBlobStore = blobStore

	metadataStore, err := c.loadMetadataStore(dbPool, logger)
	if err != nil {
		return nil, fmt.Errorf("metadata store: %w", err)
	}
	stores.ImageMetadataStore = metadataStore

	stores.KafkaProducer = c.loadKafkaProducer(logger)

	return stores, nil
}

func (c *Config) loadJobStore(pool *pgxpool.Pool, logger *slog.Logger) (jobstore.Store, error) {
	logger.Info("Using job store", "type", c.Database.Type)
	return jobstore.New(jobstore.Config{Type: c.Database.Type}, pool, logger)
}

func (c *Config) loadMetadataStore(pool *pgxpool.Pool, logger *slog.Logger) (imagestore.MetadataStore, error) {
	logger.Info("Using metadata store", "type", c.Database.Type)
	return imagestore.NewMetadataStore(imagestore.MetadataStoreConfig{Type: c.Database.Type}, pool, logger)
}

func (c *Config) loadBlobStore(logger *slog.Logger) (imagestore.BlobStore, error) {
	var s3Client *s3.Client

	if c.Blob.Type == "s3" {
		if c.Blob.Bucket == "" || c.Blob.Region == "" {
			return nil, fmt.Errorf("s3 bucket and region required")
		}
		logger.Info("Using S3 blob store", "bucket", c.Blob.Bucket, "region", c.Blob.Region)
		var err error
		s3Client, err = imagestores3.NewS3ClientWithEndpoint(
			c.Blob.Region,
			c.Blob.Endpoint,
			false,
			os.Getenv("AWS_ACCESS_KEY_ID"),
			os.Getenv("AWS_SECRET_ACCESS_KEY"),
		)
		if err != nil {
			return nil, fmt.Errorf("create S3 client: %w", err)
		}
	} else {
		logger.Info("Using blob store", "type", c.Blob.Type)
	}

	if c.Blob.Type == "disk" && c.Blob.DiskPath == "" {
		c.Blob.DiskPath = "./images"
	}
	if c.Blob.DiskPath != "" {
		if err := os.MkdirAll(c.Blob.DiskPath, 0755); err != nil {
			return nil, fmt.Errorf("create blob directory: %w", err)
		}
	}

	return imagestore.NewBlobStore(imagestore.BlobStoreConfig{
		Type:     c.Blob.Type,
		DiskPath: c.Blob.DiskPath,
		Bucket:   c.Blob.Bucket,
		Region:   c.Blob.Region,
		Endpoint: c.Blob.Endpoint,
	}, s3Client, logger)
}

func (c *Config) loadKafkaProducer(logger *slog.Logger) *kafka.Producer {
	producer := kafka.NewProducer(c.Kafka.Brokers, c.Kafka.Topic, logger)
	if producer != nil {
		logger.Info("Kafka producer initialized", "brokers", c.Kafka.Brokers, "topic", c.Kafka.Topic)
	}
	return producer
}
