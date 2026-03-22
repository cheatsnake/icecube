<h2 align="center">icecube</h2>
<p align="center">Microservice for processing images</p>
<p align="center">
    📦 <a href="#installation">Installation</a>
    &nbsp; ⚙️ <a href="#configuration">Configuration</a>
    &nbsp; 📡 <a href="#api-documentation">HTTP API</a>
    &nbsp; 🚨 <a href="https://github.com/cheatsnake/icecube/issues/new">Bug report</a>
</p>
<br />

Icecube is an image processing microservice written in Go. With this service, you can easily compress, convert, and resize images. It provides a RESTful API with support for multiple storage backends and asynchronous job processing. Packed in a lightweight Docker container for easy deployment.

## Architecture

The project architecture can be divided into several modules. 

<img alt="Architecture" src="./docs/architecture.png" />

**Image Store** is the module responsible for storing image blobs and metadata. It supports both disk storage and optionally external S3-compatible storage. Metadata is stored in PostgreSQL. 

**Job Store** is the module responsible for storing image processing jobs. It uses the same PostgreSQL database. 

**Image Processor** is the key module responsible for image processing (compression, resize, conversion). It uses external utilities such as ImageMagick, jpegoptim, libwebp, oxipng, and pngquant. All of them are lightweight and already included in the main Alpine image. 

**Worker Pool** is the module responsible for asynchronous job processing. Optionally, it can notify about completed jobs to a Kafka topic. However, this is not required, since notifications about new jobs come from the Job Store module as soon as they appear. 

A simple HTTP API interface is used for interacting with the service.

## Installation

### Docker (Recommended)

The fastest way to get started:

```bash
# Clone the repository
git clone https://github.com/cheatsnake/icecube.git
cd icecube

# Start with PostgreSQL and disk storage
docker compose --profile prod-postgres up
```

**Available profiles:**

| Profile | Database | Storage | Description |
|---------|----------|---------|-------------|
| `dev` | Memory | Memory | Quick local development |
| `dev-postgres` | PostgreSQL | Disk | Development with persistence |
| `prod` | PostgreSQL | Disk | Production deployment |
| `prod-s3` | PostgreSQL | S3 | Production with S3 storage |
| `prod-postgres` | PostgreSQL | Disk | Production with managed DB |
| `prod-postgres-s3` | PostgreSQL | S3 | Full production setup |

### Build from Source

**Build prerequisites:**

- Go 1.25 or later

**Runtime dependencies:**

For image processing, the following utilities must be installed in the system:

- [ImageMagick](https://github.com/ImageMagick/ImageMagick)
- [jpegoptim](https://github.com/tjko/jpegoptim)
- [libwebp](https://github.com/webmproject/libwebp)
- [oxipng](https://github.com/shssoichiro/oxipng)
- [pngquant](https://github.com/kornelski/pngquant)

**Build:**

```bash
# Clone the repository
git clone https://github.com/cheatsnake/icecube.git
cd icecube

# Build the server
go build -o bin/server ./cmd/server
```

**Run:**

Development/test setup (in-memory):

```bash
./bin/server -config config/icecube.dev.json
```

Production setup (postgres + disk):
```bash
./bin/server -config config/icecube.json
```

## Configuration

Icecube can be configured using environment variables or a JSON config file.

### Environment Variables

All environment variables use the `ICECUBE_` prefix:

| Variable | Default | Description |
|----------|---------|-------------|
| `ICECUBE_SERVER_PORT` | `3331` | HTTP server port |
| `ICECUBE_SERVER_MAX_WORKERS` | `4` | Number of image processing workers |
| `ICECUBE_LOG_LEVEL` | `info` | Log level: debug, info, warn, error |
| `ICECUBE_DATABASE_TYPE` | `memory` | Database type: memory, **postgres** |
| `ICECUBE_DATABASE_URI` | - | PostgreSQL connection URI (**required when DATABASE_TYPE=postgres**) |
| `ICECUBE_BLOB_TYPE` | `memory` | Blob storage type: memory, disk, **s3** |
| `ICECUBE_BLOB_DISK_PATH` | `/app/data/images` | Path for disk storage |
| `ICECUBE_BLOB_S3_BUCKET` | - | S3 bucket name (**required when BLOB_TYPE=s3**) |
| `ICECUBE_BLOB_S3_REGION` | - | S3 region (**required when BLOB_TYPE=s3**) |
| `ICECUBE_BLOB_S3_ENDPOINT` | - | S3 endpoint (for S3-compatible storage) |
| `ICECUBE_KAFKA_BROKERS` | - | Kafka broker addresses (**required when using Kafka**) |
| `ICECUBE_KAFKA_TOPIC` | - | Kafka topic for job notifications |

### External Services Configuration

When using external services, configure their credentials via standard environment variables:

**PostgreSQL:**
| Variable | Description |
|----------|-------------|
| `POSTGRES_DB` | Database name |
| `POSTGRES_USER` | Database user |
| `POSTGRES_PASSWORD` | Database password |

**AWS S3:**
| Variable | Description |
|----------|-------------|
| `AWS_ACCESS_KEY_ID` | AWS access key |
| `AWS_SECRET_ACCESS_KEY` | AWS secret key |

**Kafka:**
Uses `ICECUBE_KAFKA_BROKERS` and `ICECUBE_KAFKA_TOPIC` from the main table.

### Config File

The config file is JSON with four main sections:

```json
{
  "server": {
    "port": 3331,
    "maxWorkers": 4,
    "logLevel": "info"
  },
  "database": {
    "type": "",
    "uri": ""
  },
  "blob": {
    "type": "",
    "diskPath": "",
    "bucket": "",
    "region": "",
    "endpoint": ""
  },
  "kafka": {
    "brokers": "",
    "topic": ""
  }
}
```

**Fields:**

| Section | Field | Type | Description |
|---------|-------|------|-------------|
| `server` | `port` | int | HTTP server port (default: 3331) |
| `server` | `maxWorkers` | int | Number of image processing workers (default: 4) |
| `server` | `logLevel` | string | Logging level: debug, info, warn, error (default: info) |
| `database` | `type` | string | Storage type: "memory" or "postgres" |
| `database` | `uri` | string | PostgreSQL connection URI (required when type=postgres) |
| `blob` | `type` | string | Blob storage: "memory", "disk", or "s3" |
| `blob` | `diskPath` | string | Path for disk storage (required when type=disk) |
| `blob` | `bucket` | string | S3 bucket name (required when type=s3) |
| `blob` | `region` | string | AWS region (required when type=s3) |
| `blob` | `endpoint` | string | Custom S3 endpoint for S3-compatible storage |
| `kafka` | `brokers` | string | Comma-separated Kafka broker addresses |
| `kafka` | `topic` | string | Kafka topic for job notifications |

> **Note:** Environment variables take precedence over config file values. If a value is set both in the config file and as an environment variable, the environment variable wins.

### Example .env File

```env
ICECUBE_DATABASE_TYPE=postgres
ICECUBE_DATABASE_URI=postgres://postgres:password@localhost:5432/icecube?sslmode=disable
POSTGRES_DB=icecube
POSTGRES_USER=postgres
POSTGRES_PASSWORD=password

ICECUBE_BLOB_TYPE=disk
ICECUBE_BLOB_DISK_PATH=/app/data/images

# S3 (optional)
# ICECUBE_BLOB_TYPE=s3
# ICECUBE_BLOB_S3_BUCKET=images
# ICECUBE_BLOB_S3_REGION=us-east-1
# AWS_ACCESS_KEY_ID=your-key
# AWS_SECRET_ACCESS_KEY=your-secret

# Kafka (optional)
# ICECUBE_KAFKA_BROKERS=localhost:9092
# ICECUBE_KAFKA_TOPIC=image-jobs
```

## API Documentation

Base URL: `http://localhost:3331`

### Health Check

**GET** `/api/v1/health`

Check if the service is running.

Response:
```json
{
  "message": "Service is healthy"
}
```

### Upload Images

**POST** `/api/v1/images`

Upload one or more images.

Request: `multipart/form-data` with a `file` field containing image file(s).

Response (201 Created):
```json
[
  {
    "id": "1a2b3c4d-5e6f-7a8b-9c0d-e1f2a3b4c5d6",
    "originalName": "photo1234",
    "format": "jpeg",
    "width": 1920,
    "height": 1080,
    "byteSize": 245000
  }
]
```

### Get Image Metadata

**GET** `/api/v1/image/{id}/metadata`

Get metadata for a specific image.

Response:
```json
{
  "id": "1a2b3c4d-5e6f-7a8b-9c0d-e1f2a3b4c5d6",
  "originalName": "photo1234",
  "format": "jpeg",
  "width": 1920,
  "height": 1080,
  "byteSize": 245000
}
```

### Download Image

**GET** `/image/{id}`

Download the image file.

Response: Binary image data with `Content-Type` header set to the image format.

### Create Job

**POST** `/api/v1/job`

Create a job to process an image.

Request:
```json
{
  "originalID": "1a2b3c4d-5e6f-7a8b-9c0d-e1f2a3b4c5d6",
  "options": [
    {
      "format": "webp",
      "quality": 80,
      "maxDimension": 1000,
      "keepMetadata": false
    }
  ]
}
```

Response (201 Created):
```json
{
  "id": "7b8c9d0e-1f2a-3b4c-5d6e-f7a8b9c0d1e2",
  "status": "pending",
  "reason": null,
  "originalID": "1a2b3c4d-5e6f-7a8b-9c0d-e1f2a3b4c5d6",
  "tasks": [
    {
      "id": "0189a2b0-0a6c-7a8e-9c4d-2f5e8b3a1d7c",
      "options": {
        "format": "webp",
        "quality": 80,
        "maxDimension": 1000,
        "keepMetadata": false
      },
      "variantID": null
    }
  ],
  "createdAt": "2026-03-20T10:30:00Z"
}
```

**Image Processing Options**

| Option | Type | Description |
|--------|------|-------------|
| `format` | string | Output format: `jpeg`, `png`, `webp` |
| `maxDimension` | int | Maximum width or height in pixels (0 = no resize) |
| `quality` | int | Quality level 1-100 (100 = best quality, largest file) |
| `keepMetadata` | bool | Preserve original image metadata |


**Job Statuses:**

| Value | Description |
|-------|-------------|
| `pending` | Job is waiting to be processed |
| `processing` | Job is currently being processed |
| `completed` | Job finished successfully |
| `failed` | Job failed (see `reason` field) |

### Get Job

**GET** `/api/v1/job/{id}`

Get the status of a processing job.

Response:
```json
{
  "id": "7b8c9d0e-1f2a-3b4c-5d6e-f7a8b9c0d1e2",
  "status": "completed",
  "originalID": "1a2b3c4d-5e6f-7a8b-9c0d-e1f2a3b4c5d6",
  "tasks": [
    {
      "id": "0189a2b0-0a6c-7a8e-9c4d-2f5e8b3a1d7c",
      "options": {
        "format": "webp",
        "quality": 80,
        "maxDimension": 1000,
        "keepMetadata": false
      },
      "variantID": "2b3c4d5e-6f7a-8b9c-0d1e-f2a3b4c5d6e7"
    }
  ],
  "createdAt": "2026-03-20T10:30:00Z"
}
```

## CLI Tool

The CLI tool allows you to process images locally:

```bash
./bin/cli -input photo.jpg -format webp -max-dimension 1000 -quality 80
```

**CLI Options:**

| Flag | Description |
|------|-------------|
| `-input` | Input image file path |
| `-format` | Output format (jpeg, png, webp) |
| `-quality` | Compression level (1-100) |
| `-max-dimension` | Maximum width or height in pixels |
| `-keep-metadata` | Preserve original metadata |
| `-output` | Output file path (default: input file with new extension) |

## Development

### Project Structure

```
icecube/
├── cmd/
│   ├── server/          # HTTP API server entry point
│   └── cli/             # CLI tool entry point
├── internal/
│   ├── domain/          # Business logic entities
│   │   ├── image/       # Image format types and validation
│   │   ├── jobs/        # Job and task entities
│   │   └── processing/  # Processing options
│   ├── infra/           # Infrastructure implementations
│   │   ├── config/      # Configuration loading
│   │   ├── kafka/       # Kafka producer
│   │   ├── postgres/    # PostgreSQL integration
│   │   └── s3/          # S3 blob storage
│   ├── service/         # Business logic services
│   │   ├── imagestore/  # Image storage (memory, disk, S3)
│   │   ├── jobstore/    # Job persistence
│   │   └── processor/   # Image processing
│   └── transport/
│       └── http/        # HTTP server and handlers
├── config/              # Config files
├── Makefile             # Build and run commands
├── Dockerfile           # Container build
└── docker-compose.yml   # Multi-profile Docker setup
```

### Running Locally with Docker Compose

```bash
# Development with in-memory storage
docker compose --profile dev up

# Development with PostgreSQL
docker compose --profile dev-postgres up
```

### Makefile Commands

```bash
make build           # Build the server
make build-cli       # Build the CLI tool
make run             # Run the server
make test            # Run tests
make clean           # Remove build artifacts
make docker-build    # Build Docker image
make docker-up-dev   # Start Docker container (dev mode)
make docker-up-prod  # Start Docker container (prod mode)
make docker-down     # Stop Docker containers
```

## License

MIT License
