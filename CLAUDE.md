# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Run Commands

All `make` targets are defined in `build/Makefile` and must be run from the repo root (they resolve the repo root internally via `REPO_ROOT`):

```bash
# Build both services
cd build && make build

# Run services locally (requires env vars from .env)
cd build && make run-dataservice        # port 8081 HTTP, 50051 gRPC
cd build && make run-metadataservice    # port 8080 HTTP

# Alternatively from repo root
go run ./dataservice
go run ./metadataservice
```

## Testing

```bash
# Run all tests (from repo root or via make)
cd build && make test

# Single service tests
go test -v -race -coverprofile=coverage-dataservice.out ./dataservice/...
go test -v -race -coverprofile=coverage-metadataservice.out ./metadataservice/...

# Single package test
go test -v -run TestUploadService ./dataservice/bl/...

# E2E / integration tests (require running infrastructure)
cd tests/e2e && go test -v ./...
```

## Lint & Format

```bash
go vet ./...
golangci-lint run ./...   # config in .golangci.yml
go fmt ./...
```

The linter excludes `dataservice/pb/` (generated gRPC code). Enabled linters: `errcheck`, `govet`, `ineffassign`, `staticcheck`, `unused`, `gocritic`, plus `gofmt`/`goimports` formatters.

## Local Infrastructure

```bash
# Start MySQL, Redis, LocalStack (S3), Jaeger, Prometheus, Grafana
cd build && docker-compose up

# Copy and edit environment config
cp .env.example .env
```

## Architecture

### Module & Service Layout

Module path: `github.com/yourusername/videostreamingplatform`

Two services share a single Go module:

| Service | Entry Point | Ports | Storage |
|---------|-------------|-------|---------|
| `metadataservice` | `metadataservice/main.go` | 8080 HTTP | MySQL |
| `dataservice` | `dataservice/main.go` | 8081 HTTP + 50051 gRPC | MySQL + S3/MinIO |

### Layered Architecture (both services)

Each service follows a strict 3-layer pattern:

```
handlers/    → HTTP handlers, request/response parsing, no business logic
bl/          → Business logic, orchestration, service structs
dl/          → Data layer interfaces + implementations (MySQL, in-memory)
db/          → Raw database connection setup
models/      → Shared domain structs for the service
```

Dependency direction: `handlers → bl → dl → db`. The `bl` layer depends on `dl` interfaces, not concrete implementations—enabling in-memory substitution for tests.

### Shared `utils/` Packages

All utilities live in `utils/` and are imported by both services:

- **`utils/config`** — All config from env vars via `config.New(serviceName)`. Key vars: `ENVIRONMENT`, `MYSQL_*`, `S3_*`, `KAFKA_BROKERS`, `KAFKA_VIDEO_TOPIC`, `KAFKA_WATCH_TOPIC`, `UPLOAD_STORE` (`memory`|`mysql`), `OTEL_EXPORTER_OTLP_ENDPOINT`, `RECOMMENDATION_SERVICE_URL`, `REDIS_ADDR`, `REDIS_PASSWORD`, `REDIS_DB`, `CACHE_TTL_*`, `RATE_LIMIT_*`.
- **`utils/observability`** — Logger, OTel tracing (`InitTracer`), Prometheus metrics (`InitMetrics`). Tracing only initializes when `OTEL_EXPORTER_OTLP_ENDPOINT` is set. Metrics only when explicitly initialized.
- **`utils/kafka`** — `Producer` interface + `KafkaProducer` impl (segmentio/kafka-go). Kafka is **optional**—both services skip it gracefully if `KAFKA_BROKERS` is empty.
- **`utils/events`** — Avro event structs: `video_event.go` (VIDEO_CREATED/UPDATED/DELETED), `watch_event.go` (WATCH_STARTED/WATCH_COMPLETED).
- **`utils/cache`** — Redis-backed cache (`cache.New(addr, pass, db)`). Nil-safe: all methods no-op if `c==nil` or `addr=""`. Used in `metadataservice` for video metadata caching. Key helpers: `VideoKey(id)` → `video:{id}`, `ListKey(limit, offset)` → `videos:list:{limit}:{offset}`. Env vars: `REDIS_ADDR`, `REDIS_PASSWORD`, `REDIS_DB`, `CACHE_TTL_GET_VIDEO`, `CACHE_TTL_LIST_VIDEOS`.
- **`utils/middleware`** — `ChainMiddleware`, `LoggingMiddleware`, `ErrorHandlingMiddleware`, `RateLimiter` (per-IP token bucket). Env vars: `RATE_LIMIT_PER_MIN`, `RATE_LIMIT_BURST`.
- **`utils/recommendations`** — HTTP client for calling the recommendations service (Python/FastAPI). Only wired in `metadataservice`.
- **`utils/errors`** — Shared error types.

### dataservice Specifics

- Exposes both **HTTP** (upload/download REST endpoints + Swagger) and **gRPC** (`DataService` service defined in `dataservice/proto/dataservice.proto`, generated code in `dataservice/pb/`).
- `UPLOAD_STORE=memory` uses an in-memory repository (no MySQL needed)—useful for local testing without Docker.
- The `dataservice/streaming` package defines `UploadSession` and `DefaultChunkSize` (5MB). Streaming logic for HTTP chunked upload is in `dataservice/handlers/upload.go`.
- Watch events are published to Kafka on download completion (best-effort, non-fatal).

### metadataservice Specifics

- HTTP only, no gRPC.
- Publishes `VideoEvent` to Kafka (`video-events` topic) on create/update/delete via functional option `bl.WithKafkaProducer(...)`.
- Redis caching via `bl.WithCache(...)`: caches `GetVideo`/`ListVideos`, invalidates on `CreateVideo`/`UpdateVideo`/`DeleteVideo`.
- Routes `/recommendations` to the external recommendations service via `utils/recommendations.Client`.

### Event Flow

```
metadataservice → kafka[video-events] → videostreamingplatform-analytics (Kafka→ES consumer)
dataservice     → kafka[watch-events] → videostreamingplatform-analytics (Spark→Iceberg)
                                      → videostreamingplatform-recommendations
```

### Kubernetes & Deployment

- `k8s/local/` — Kind cluster manifests
- `k8s/aws/` — EKS manifests + Terraform modules
- `build/docker/` — Dockerfiles per service
- CI/CD via GitHub Actions: `build.yml`, `test.yml`, `deploy-local.yml`, `deploy-aws-k8s.yml`, `deploy-aws-terraform.yml`
