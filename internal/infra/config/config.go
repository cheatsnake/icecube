package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"strconv"
)

const envPrefix = "ICECUBE_"

type Config struct {
	Server   ServerConfig   `json:"server"`
	Database DatabaseConfig `json:"database"`
	Blob     BlobConfig     `json:"blob"`
	Kafka    KafkaConfig    `json:"kafka"`
}

type ServerConfig struct {
	Port       int `json:"port"`
	MaxWorkers int `json:"maxWorkers,omitempty"`
}

type DatabaseConfig struct {
	Type string `json:"type"`          // "memory" or "postgres"
	URI  string `json:"uri,omitempty"` // connection string for database
}

type BlobConfig struct {
	Type     string `json:"type"`               // "memory", "disk", or "s3"
	DiskPath string `json:"diskPath,omitempty"` // path for disk storage
	Bucket   string `json:"bucket,omitempty"`   // S3 bucket name
	Region   string `json:"region,omitempty"`   // AWS region
	Endpoint string `json:"endpoint,omitempty"` // custom S3 endpoint
}

type KafkaConfig struct {
	Brokers string `json:"brokers,omitempty"` // comma-separated list of broker addresses
	Topic   string `json:"topic,omitempty"`   // topic name for notifications
}

var DefaultConfigPath = path.Join("config", "icecube.json")

func Load(path string) (*Config, error) {
	if path == "" {
		path = DefaultConfigPath
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultConfig(), nil
		}
		return nil, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	applyEnvOverrides(&cfg) // Apply environment variables on top of config

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// applyEnvOverrides applies environment variables with ICECUBE_ prefix to config
func applyEnvOverrides(cfg *Config) {
	// Server config
	if v := os.Getenv(envPrefix + "SERVER_PORT"); v != "" {
		if port, err := strconv.Atoi(v); err == nil {
			cfg.Server.Port = port
		}
	}
	if v := os.Getenv(envPrefix + "SERVER_MAX_WORKERS"); v != "" {
		if maxWorkers, err := strconv.Atoi(v); err == nil {
			cfg.Server.MaxWorkers = maxWorkers
		}
	}

	// Database config
	if v := os.Getenv(envPrefix + "DATABASE_TYPE"); v != "" {
		cfg.Database.Type = v
	}
	if v := os.Getenv(envPrefix + "DATABASE_URI"); v != "" {
		cfg.Database.URI = v
	}

	// Blob config
	if v := os.Getenv(envPrefix + "BLOB_TYPE"); v != "" {
		cfg.Blob.Type = v
	}
	// Filesystem storage
	if v := os.Getenv(envPrefix + "BLOB_DISK_PATH"); v != "" {
		cfg.Blob.DiskPath = v
	}
	// S3 storage
	if v := os.Getenv(envPrefix + "BLOB_S3_BUCKET"); v != "" {
		cfg.Blob.Bucket = v
	}
	if v := os.Getenv(envPrefix + "BLOB_S3_REGION"); v != "" {
		cfg.Blob.Region = v
	}
	if v := os.Getenv(envPrefix + "BLOB_S3_ENDPOINT"); v != "" {
		cfg.Blob.Endpoint = v
	}

	// Kafka config
	if v := os.Getenv(envPrefix + "KAFKA_BROKERS"); v != "" {
		cfg.Kafka.Brokers = v
	}
	if v := os.Getenv(envPrefix + "KAFKA_TOPIC"); v != "" {
		cfg.Kafka.Topic = v
	}
}

func DefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Port:       3331,
			MaxWorkers: 4,
		},
		Database: DatabaseConfig{
			Type: "memory",
		},
		Blob: BlobConfig{
			Type: "memory",
		},
	}
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.Database.Type == "postgres" && c.Database.URI == "" {
		return fmt.Errorf("database type is postgres but URI is not set. Use ICECUBE_DATABASE_URI env var")
	}

	if c.Blob.Type == "disk" && c.Blob.DiskPath == "" {
		return fmt.Errorf("blob type is disk but path is not set")
	}
	if c.Blob.Type == "s3" && (c.Blob.Bucket == "" || c.Blob.Region == "") {
		return fmt.Errorf("blob type is s3 but bucket or region is not set")
	}

	return nil
}
