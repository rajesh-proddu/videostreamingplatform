# Service Integration Guide

This guide shows step-by-step how to integrate the new engineering excellence packages into the main services.

## Quick Start Integration

The easiest way to integrate is using the `bootstrap` package:

### Data Service (`cmd/data-service/main.go`)

```go
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	httpSwagger "github.com/swaggo/http-swagger"
	"github.com/yourusername/videostreamingplatform/internal/bootstrap"
	"github.com/yourusername/videostreamingplatform/internal/db"
	"github.com/yourusername/videostreamingplatform/internal/handlers"
	"github.com/yourusername/videostreamingplatform/internal/metrics"
	"github.com/yourusername/videostreamingplatform/internal/middleware"
	"github.com/yourusername/videostreamingplatform/internal/storage"
)

func main() {
	ctx := context.Background()

	// Initialize all service components (config, logger, metrics, traces, health)
	svc, err := bootstrap.Initialize(ctx)
	if err != nil {
		log.Fatalf("Failed to initialize service: %v", err)
	}
	defer svc.Shutdown(ctx)

	// Initialize database with health check
	database, err := db.NewMySQL(svc.Config.Database.DSN())
	if err != nil {
		svc.Logger.Error(ctx, "Failed to connect to database", "error", err.Error())
		log.Fatalf("Database connection failed: %v", err)
	}
	defer database.Close()

	// Register database health check
	svc.RegisterDatabaseHealthCheck(database.Ping)

	// Check database connection
	if err := database.Ping(ctx); err != nil {
		svc.Logger.Error(ctx, "Database ping failed", "error", err.Error())
		log.Fatalf("Database ping failed: %v", err)
	}
	svc.Logger.Info(ctx, "Database connected successfully")

	// Initialize S3 client with health check
	s3Client, err := storage.NewS3Client(ctx)
	if err != nil {
		svc.Logger.Error(ctx, "Failed to initialize S3 client", "error", err.Error())
		log.Fatalf("S3 initialization failed: %v", err)
	}

	// Register storage health check
	svc.RegisterStorageHealthCheck(func(checkCtx context.Context) error {
		exists, err := s3Client.BucketExists(checkCtx, svc.Config.Storage.Bucket)
		if err != nil {
			return err
		}
		if !exists {
			return fmt.Errorf("bucket does not exist: %s", svc.Config.Storage.Bucket)
		}
		return nil
	})
	svc.Logger.Info(ctx, "S3 client initialized successfully")

	// Initialize streaming handler
	streamingHandler := handlers.NewStreamingDataHandler(s3Client, database)

	// Set up HTTP routes
	mux := http.NewServeMux()

	// Health and metrics endpoints
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		healthStatus := svc.GetHealthStatus(r.Context())
		w.Header().Set("Content-Type", "application/json")
		
		// Determine HTTP status based on overall health
		if status, ok := healthStatus["status"].(string); ok && status == "unhealthy" {
			w.WriteHeader(http.StatusServiceUnavailable)
		} else {
			w.WriteHeader(http.StatusOK)
		}
		
		// Write JSON response
		if jsonData, err := json.Marshal(healthStatus); err == nil {
			w.Write(jsonData)
		}
	})

	// Prometheus metrics endpoint
	mux.Handle("GET /metrics", metrics.PrometheusHandler())

	// Swagger documentation
	mux.Handle("GET /swagger/", httpSwagger.WrapHandler)

	// Streaming upload endpoints
	mux.HandleFunc("POST /uploads/initiate", streamingHandler.InitiateUpload)
	mux.HandleFunc("POST /uploads/{uploadId}/chunks", streamingHandler.UploadChunk)
	mux.HandleFunc("GET /uploads/{uploadId}/progress", streamingHandler.GetUploadProgress)
	mux.HandleFunc("POST /uploads/{uploadId}/complete", streamingHandler.CompleteUpload)

	// Streaming download endpoint
	mux.HandleFunc("GET /videos/{id}/download", streamingHandler.StreamDownload)

	// Apply middleware chain to all handlers
	handler := middleware.CombinedMiddleware(
		svc.Logger,
		svc.Metrics,
		svc.Config.Streaming.MaxUploadSize,
		svc.Config.Streaming.UploadTimeout,
	)(mux)

	// Create HTTP/2 server with configuration
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", svc.Config.Port),
		Handler:      handler,
		ReadTimeout:  svc.Config.Database.Timeout,
		WriteTimeout: svc.Config.Database.Timeout,
		IdleTimeout:  30 * time.Second,
	}

	svc.Logger.Info(ctx, "Data Service starting",
		"port", svc.Config.Port,
		"env", svc.Config.Environment,
	)

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		svc.Logger.Info(ctx, "Received signal", "signal", sig.String())
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		server.Shutdown(shutdownCtx)
	}()

	// Start server
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		svc.Logger.Error(ctx, "Server error", "error", err.Error())
		log.Fatalf("Server error: %v", err)
	}

	svc.Logger.Info(ctx, "Data Service stopped")
}
```

### Metadata Service (`cmd/metadata-service/main.go`)

```go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	httpSwagger "github.com/swaggo/http-swagger"
	"github.com/yourusername/videostreamingplatform/internal/bootstrap"
	"github.com/yourusername/videostreamingplatform/internal/db"
	"github.com/yourusername/videostreamingplatform/internal/handlers"
	"github.com/yourusername/videostreamingplatform/internal/metrics"
	"github.com/yourusername/videostreamingplatform/internal/middleware"
)

func main() {
	ctx := context.Background()

	// Initialize all service components
	svc, err := bootstrap.Initialize(ctx)
	if err != nil {
		log.Fatalf("Failed to initialize service: %v", err)
	}
	defer svc.Shutdown(ctx)

	// Initialize database
	database, err := db.NewMySQL(svc.Config.Database.DSN())
	if err != nil {
		svc.Logger.Error(ctx, "Failed to connect to database", "error", err.Error())
		log.Fatalf("Database connection failed: %v", err)
	}
	defer database.Close()

	// Register database health check
	svc.RegisterDatabaseHealthCheck(database.Ping)

	// Check database connection
	if err := database.Ping(ctx); err != nil {
		svc.Logger.Error(ctx, "Database ping failed", "error", err.Error())
		log.Fatalf("Database ping failed: %v", err)
	}
	svc.Logger.Info(ctx, "Database connected successfully")

	// Initialize HTTP handlers
	handler := handlers.NewMetadataHandler(database)

	// Set up HTTP routes
	mux := http.NewServeMux()

	// Health and metrics endpoints
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		healthStatus := svc.GetHealthStatus(r.Context())
		w.Header().Set("Content-Type", "application/json")
		
		if status, ok := healthStatus["status"].(string); ok && status == "unhealthy" {
			w.WriteHeader(http.StatusServiceUnavailable)
		} else {
			w.WriteHeader(http.StatusOK)
		}
		
		if jsonData, err := json.Marshal(healthStatus); err == nil {
			w.Write(jsonData)
		}
	})

	// Prometheus metrics endpoint
	mux.Handle("GET /metrics", metrics.PrometheusHandler())

	// Swagger documentation
	mux.Handle("GET /swagger/", httpSwagger.WrapHandler)

	// Metadata CRUD endpoints
	mux.HandleFunc("POST /videos", handler.CreateVideo)
	mux.HandleFunc("GET /videos/{id}", handler.GetVideo)
	mux.HandleFunc("PUT /videos/{id}", handler.UpdateVideo)
	mux.HandleFunc("DELETE /videos/{id}", handler.DeleteVideo)
	mux.HandleFunc("GET /videos", handler.ListVideos)

	// Apply middleware
	handlerWithMiddleware := middleware.CombinedMiddleware(
		svc.Logger,
		svc.Metrics,
		100*1024*1024, // 100MB max body
		30*time.Second, // 30 second timeout
	)(mux)

	// Create HTTP/2 server
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", svc.Config.Port),
		Handler:      handlerWithMiddleware,
		ReadTimeout:  svc.Config.Database.Timeout,
		WriteTimeout: svc.Config.Database.Timeout,
		IdleTimeout:  30 * time.Second,
	}

	svc.Logger.Info(ctx, "Metadata Service starting",
		"port", svc.Config.Port,
		"env", svc.Config.Environment,
	)

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		svc.Logger.Info(ctx, "Received signal", "signal", sig.String())
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		server.Shutdown(shutdownCtx)
	}()

	// Start server
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		svc.Logger.Error(ctx, "Server error", "error", err.Error())
		log.Fatalf("Server error: %v", err)
	}

	svc.Logger.Info(ctx, "Metadata Service stopped")
}
```

## Environment Variables

Create `.env` file in project root:

```bash
# Service Configuration
SERVICE_NAME=videostreamingplatform
SERVICE_VERSION=0.1.0
ENVIRONMENT=local
LOG_LEVEL=info

# Database Configuration
MYSQL_HOST=localhost
MYSQL_PORT=3306
MYSQL_USER=videouser
MYSQL_PASSWORD=videopass
MYSQL_DATABASE=videoplatform
DB_MAX_CONNS=20
DB_TIMEOUT=30s

# Storage Configuration
STORAGE_TYPE=s3
S3_BUCKET=videostreamingplatform
AWS_REGION=us-east-1
AWS_ACCESS_KEY_ID=minioadmin
AWS_SECRET_ACCESS_KEY=minioadmin

# Streaming Configuration
CHUNK_SIZE_MB=5
MAX_UPLOAD_SIZE_GB=100
UPLOAD_TIMEOUT=30m
DOWNLOAD_TIMEOUT=30m

# Observability
LOG_LEVEL=info
OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4317
METRICS_ENABLED=true
TRACES_ENABLED=true
PROMETHEUS_PORT=9090

# Retry Configuration
RETRY_MAX_ATTEMPTS=3
RETRY_INITIAL_BACKOFF=100ms
RETRY_MAX_BACKOFF=10s
RETRY_BACKOFF_MULTIPLIER=2.0

# HTTP Configuration
PORT=8080 # 8080 for metadata-service, 8081 for data-service
```

## Load Environment Variables

Add to shell initialization or use a tool like `direnv`:

```bash
# Load .env before running services
set -a
source .env
set +a

# Run the service
go run cmd/data-service/main.go
```

## Docker Compose Integration

Update `docker-compose.yaml` to set environment variables:

```yaml
services:
  metadata-service:
    build:
      context: .
      dockerfile: Dockerfile.metadata
    ports:
      - "8080:8080"
    environment:
      - SERVICE_NAME=videostreamingplatform
      - ENVIRONMENT=local
      - MYSQL_HOST=mysql
      - MYSQL_PORT=3306
      - S3_BUCKET=videostreamingplatform
      - AWS_ACCESS_KEY_ID=minioadmin
      - AWS_SECRET_ACCESS_KEY=minioadmin
      - S3_ENDPOINT=http://minio:9000
      - LOG_LEVEL=info
      - OTEL_EXPORTER_OTLP_ENDPOINT=http://otel-collector:4317
    depends_on:
      - mysql
      - minio
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/health"]
      interval: 10s
      timeout: 3s
      retries: 3

  data-service:
    build:
      context: .
      dockerfile: Dockerfile.data
    ports:
      - "8081:8081"
    environment:
      - SERVICE_NAME=videostreamingplatform
      - ENVIRONMENT=local
      - PORT=8081
      - MYSQL_HOST=mysql
      - S3_BUCKET=videostreamingplatform
      - AWS_ACCESS_KEY_ID=minioadmin
      - AWS_SECRET_ACCESS_KEY=minioadmin
      - S3_ENDPOINT=http://minio:9000
      - LOG_LEVEL=info
      - OTEL_EXPORTER_OTLP_ENDPOINT=http://otel-collector:4317
    depends_on:
      - mysql
      - minio
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8081/health"]
      interval: 10s
      timeout: 3s
      retries: 3
```

## Testing Integration

Verify the services start correctly:

```bash
# Start services
docker-compose up

# Check health
curl http://localhost:8080/health
curl http://localhost:8081/health

# Check metrics
curl http://localhost:8080/metrics
curl http://localhost:8081/metrics

# Check Swagger docs
open http://localhost:8080/swagger/
```

## Key Integration Points

1. **Configuration**: All services use `config.NewConfig()` to load from env
2. **Logging**: Use `svc.Logger` for structured logging
3. **Metrics**: Recording handled by middleware automatically
4. **Tracing**: Trace IDs added by middleware to all requests
5. **Health**: Register DB and storage checks via `svc.Register*HealthCheck()`
6. **Shutdown**: Defer `svc.Shutdown(ctx)` to cleanup resources gracefully

## Troubleshooting

**Service won't start**
- Check environment variables: `echo $LOG_LEVEL`
- Verify database connection: `mysql -h $MYSQL_HOST -u $MYSQL_USER -p`
- Check ports are available: `lsof -i :8080`

**No metrics recorded**
- Verify `METRICS_ENABLED=true`
- Check `curl http://localhost:8080/metrics`

**Traces not exported**
- Verify `TRACES_ENABLED=true`
- Check OTLP endpoint: `OTEL_EXPORTER_OTLP_ENDPOINT`
- Verify OpenTelemetry collector is running

**Health check failing**
- Check database connectivity from service
- Check health endpoint: `curl http://localhost:8080/health`
