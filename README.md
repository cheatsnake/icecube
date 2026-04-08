<br>
<p align="center">
  <a href="https://github.com/cheatsnake/airstation">
    <img src="https://i.ibb.co/M5DZp0Mh/icecube.png" alt="logo" height="192">
  </a>
</p>
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

<img alt="Architecture" src="./docs/workflow.png" />

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

---

### Health

<details>
<summary><code>GET</code> <code><b>/api/v1/health</b></code> <code>(Check if the service is running)</code></summary>

**Response**

| http code | content-type | response |
|-----------|--------------|----------|
| `200` | `application/json` | `{"message": "Service is healthy"}` |

**Example cURL**

```bash
curl -X GET http://localhost:3331/api/v1/health
```

</details>

---

### Images

<details>
<summary><code>POST</code> <code><b>/api/v1/images</b></code> <code>(Upload one or more images)</code></summary>

**Request**

`multipart/form-data` with a `file` field containing image file(s).

**Responses**

| http code | content-type | response |
|-----------|--------------|----------|
| `201` | `application/json` | `[{id, originalName, format, width, height, byteSize}]` |

**Example cURL**

```bash
curl -X POST -F "file=@photo.jpg" http://localhost:3331/api/v1/images
```

</details>

<details>
<summary><code>GET</code> <code><b>/api/v1/image/{id}/metadata</b></code> <code>(Get metadata for a specific image)</code></summary>

**Parameters**

| name | type | data type | description |
|------|------|-----------|-------------|
| `id` | required | string (UUID) | The unique image identifier |

**Responses**

| http code | content-type | response |
|-----------|--------------|----------|
| `200` | `application/json` | `{id, originalName, format, width, height, byteSize}` |
| `404` | `application/json` | `{"error": "Image not found"}` |

**Example cURL**

```bash
curl -X GET http://localhost:3331/api/v1/image/1a2b3c4d-5e6f-7a8b-9c0d-e1f2a3b4c5d6/metadata
```

</details>

<details>
<summary><code>GET</code> <code><b>/image/{id}</b></code> <code>(Download the image file)</code></summary>

**Parameters**

| name | type | data type | description |
|------|------|-----------|-------------|
| `id` | required | string (UUID) | The unique image identifier |

**Responses**

| http code | content-type | response |
|-----------|--------------|----------|
| `200` | `image/*` | Binary image data with `Content-Type` header set to the image format |
| `404` | `application/json` | `{"error": "Image not found"}` |

**Example cURL**

```bash
curl -X GET http://localhost:3331/image/1a2b3c4d-5e6f-7a8b-9c0d-e1f2a3b4c5d6 -o image.jpg
```

</details>

---

### Jobs

<details>
<summary><code>POST</code> <code><b>/api/v1/job</b></code> <code>(Create a job to process an image)</code></summary>

**Request**

| name | type | data type | description |
|------|------|-----------|-------------|
| `originalID` | required | string (UUID) | The source image ID |
| `options` | required | array | Array of processing options |

**Image Processing Options**

| name | type | data type | description |
|------|------|-----------|-------------|
| `format` | optional | string | Output format: `jpeg`, `png`, `webp` |
| `maxDimension` | optional | int | Maximum width or height in pixels (0 = no resize) |
| `quality` | optional | int | Quality level 1-100 (100 = best quality, largest file) |
| `keepMetadata` | optional | bool | Preserve original image metadata (default: false) |

Responses

| http code | content-type | response |
|-----------|--------------|----------|
| `201` | `application/json` | `{id, status, originalID, tasks, createdAt}` |
| `400` | `application/json` | `{"error": "Invalid request"}` |
| `404` | `application/json` | `{"error": "Image not found"}` |

**Job Statuses**

| Value | Description |
|-------|-------------|
| `pending` | Job is waiting to be processed |
| `processing` | Job is currently being processed |
| `completed` | Job finished successfully |
| `failed` | Job failed (see `reason` field) |

**Example cURL**

```bash
curl -X POST -H "Content-Type: application/json" \
  -d '{"originalID":"1a2b3c4d-5e6f-7a8b-9c0d-e1f2a3b4c5d6","options":[{"format":"webp","quality":80,"maxDimension":1000,"keepMetadata":false}]}' \
  http://localhost:3331/api/v1/job
```

</details>

<details>
<summary><code>GET</code> <code><b>/api/v1/job/{id}</b></code> <code>(Get the status of a processing job)</code></summary>

**Parameters**

| name | type | data type | description |
|------|------|-----------|-------------|
| `id` | required | string (UUID) | The unique job identifier |

**Responses**

| http code | content-type | response |
|-----------|--------------|----------|
| `200` | `application/json` | `{id, status, reason, originalID, tasks, createdAt}` |
| `404` | `application/json` | `{"error": "Job not found"}` |

**Example cURL**

```bash
curl -X GET http://localhost:3331/api/v1/job/7b8c9d0e-1f2a-3b4c-5d6e-f7a8b9c0d1e2
```

</details>

---

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
