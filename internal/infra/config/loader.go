package config

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/cheatsnake/icecube/internal/infra/kafka"
	"github.com/cheatsnake/icecube/internal/infra/postgres"
	"github.com/cheatsnake/icecube/internal/infra/s3"
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
	switch c.Database.Type {
	case "postgres":
		logger.Info("Using PostgreSQL for job store")
		return jobstore.NewJobStorePostgres(pool, logger), nil
	case "memory":
		logger.Info("Using in-memory job store")
		return jobstore.NewJobStoreMemory(), nil
	default:
		return nil, fmt.Errorf("unsupported database type: %s", c.Database.Type)
	}
}

func (c *Config) loadMetadataStore(pool *pgxpool.Pool, logger *slog.Logger) (imagestore.MetadataStore, error) {
	switch c.Database.Type {
	case "postgres":
		logger.Info("Using PostgreSQL for metadata store")
		return imagestore.NewMetadataStorePostgres(pool), nil
	case "memory":
		logger.Info("Using in-memory metadata store")
		return imagestore.NewMetadataStoreMemory(), nil
	default:
		return nil, fmt.Errorf("unsupported database type: %s", c.Database.Type)
	}
}

func (c *Config) loadBlobStore(logger *slog.Logger) (imagestore.BlobStore, error) {
	switch c.Blob.Type {
	case "memory":
		logger.Info("Using in-memory blob store")
		return imagestore.NewBlobStoreMemory(), nil
	case "disk":
		if c.Blob.DiskPath == "" {
			c.Blob.DiskPath = "./images"
		}
		if err := os.MkdirAll(c.Blob.DiskPath, 0755); err != nil {
			return nil, fmt.Errorf("create blob directory: %w", err)
		}
		logger.Info("Using disk blob store", "path", c.Blob.DiskPath)
		return imagestore.NewBlobStoreDisk(c.Blob.DiskPath), nil
	case "s3":
		if c.Blob.Bucket == "" || c.Blob.Region == "" {
			return nil, fmt.Errorf("s3 bucket and region required")
		}
		logger.Info("Using S3 blob store", "bucket", c.Blob.Bucket, "region", c.Blob.Region)
		client, err := s3.NewS3ClientWithEndpoint(
			c.Blob.Region,
			c.Blob.Endpoint,
			false,
			os.Getenv("AWS_ACCESS_KEY_ID"),
			os.Getenv("AWS_SECRET_ACCESS_KEY"),
		)
		if err != nil {
			return nil, fmt.Errorf("create S3 client: %w", err)
		}
		return imagestore.NewBlobStoreS3(client, c.Blob.Bucket, ""), nil
	default:
		return nil, fmt.Errorf("unsupported blob type: %s", c.Blob.Type)
	}
}

func (c *Config) loadKafkaProducer(logger *slog.Logger) *kafka.Producer {
	producer := kafka.NewProducer(c.Kafka.Brokers, c.Kafka.Topic, logger)
	if producer != nil {
		logger.Info("Kafka producer initialized", "brokers", c.Kafka.Brokers, "topic", c.Kafka.Topic)
	}
	return producer
}
