package config

import (
	"encoding/json"
	"os"
	"path"
)

type Config struct {
	Server   ServerConfig   `json:"server"`
	Database DatabaseConfig `json:"database"`
	Blob     BlobConfig     `json:"blob"`
}

type ServerConfig struct {
	Port       int `json:"port"`
	MaxWorkers int `json:"maxWorkers,omitempty"`
}

type DatabaseConfig struct {
	Type string `json:"type"`          // "memory" or "postgres"
	DSN  string `json:"dsn,omitempty"` // connection string for postgres
}

type BlobConfig struct {
	Type     string `json:"type"`               // "memory", "disk", or "s3"
	Path     string `json:"path,omitempty"`     // path for disk storage
	Bucket   string `json:"bucket,omitempty"`   // S3 bucket name
	Region   string `json:"region,omitempty"`   // AWS region
	Endpoint string `json:"endpoint,omitempty"` // custom S3 endpoint
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

	return &cfg, nil
}

func DefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Port: 3331,
		},
		Database: DatabaseConfig{
			Type: "memory",
		},
		Blob: BlobConfig{
			Type: "memory",
		},
	}
}
