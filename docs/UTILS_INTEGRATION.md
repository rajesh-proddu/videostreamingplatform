# Utils Module Integration Guide

## Overview
This document outlines how the utility modules under `utils/` are integrated with both Data Service and Metadata Service.

## Project Structure
```
videostreamingplatform/
├── utils/                          # Shared utility packages
│   ├── config/                     # Configuration management
│   ├── errors/                     # Custom error types
│   ├── middleware/                 # HTTP middleware
│   └── observability/              # Logging and observability
├── dataservice/                    # Data Service (HTTP/2 + gRPC)
│   ├── main.go                     # Uses utils packages
│   ├── bl/                         # Business logic
│   ├── dl/                         # Data layer
│   ├── handlers/                   # HTTP/gRPC handlers
│   ├── server/                     # gRPC server setup
│   ├── storage/                    # S3 storage
│   └── streaming/                  # Upload streaming utilities
└── metadataservice/                # Metadata Service (HTTP REST)
    ├── main.go                     # Uses utils packages
    ├── bl/                         # Business logic
    ├── dl/                         # Data layer
    ├── handlers/                   # HTTP handlers
    ├── db/                         # MySQL database
    └── models/                     # Domain models
```

## Utils Modules Integration

### 1. Config Module (`utils/config/`)

**Purpose**: Centralized configuration management following 12-factor app principles

**Integration Points**:
- **dataservice/main.go**: 
  ```go
  cfg := config.New("dataservice")
  cfg.Validate()
  // Uses: cfg.HTTPPort, cfg.GRPCPort, cfg.S3Region, cfg.S3Bucket, etc.
  ```
- **metadataservice/main.go**:
  ```go
  cfg := config.New("metadataservice")
  cfg.Validate()
  // Uses: cfg.HTTPPort, cfg.MySQLHost, cfg.MySQLPort, etc.
  ```

**Configuration Keys**:
- `HTTP_PORT` - HTTP server port (default: 8080)
- `GRPC_PORT` - gRPC server port (default: 50051)
- `ENVIRONMENT` - Environment (dev/staging/prod)
- `LOG_LEVEL` - Log level (debug/info/warn/error)
- `MYSQL_*` - Database connection parameters
- `S3_*` - S3 storage configuration
- `OTEL_*` - OpenTelemetry configuration

### 2. Observability Module (`utils/observability/`)

**Purpose**: Structured logging and tracing framework

**Integration Points**:
- **dataservice/main.go**:
  ```go
  logger := observability.NewLogger("DataService")
  logger.Printf("Starting DataService in %s environment", cfg.Envir)
  logger.Info("message")
  logger.Warn("warning")
  logger.Error("error")
  logger.Debug("debug")
  ```
- **metadataservice/main.go**:
  ```go
  logger := observability.NewLogger("MetadataService")
  // Same logging interface
  ```

**Features**:
- Structured log messages with timestamps
- Service name prefix (e.g., `[DataService]`)
- Log levels: Info, Warn, Error, Debug
- Context support ready for OpenTelemetry integration
- Compatible with existing `*log.Logger` interface

### 3. Middleware Module (`utils/middleware/`)

**Purpose**: HTTP request/response processing and error handling

**Integration Points**:
- **dataservice/main.go**:
  ```go
  httpHandler := middleware.ChainMiddleware(
    mux,
    func(next http.Handler) http.Handler {
      return middleware.LoggingMiddleware(logger, next)
    },
    func(next http.Handler) http.Handler {
      return middleware.ErrorHandlingMiddleware(logger, next)
    },
  )
  ```
- **metadataservice/main.go**: Same middleware chain pattern

**Middleware Components**:
- **LoggingMiddleware**: Logs all HTTP requests/responses with timing
  - Logs method, path, remote address
  - Captures response status code
  - Measures request duration
  
- **ErrorHandlingMiddleware**: Panic recovery and error handling
  - Catches panics and returns 500 error
  - Structured error responses
  
- **ChainMiddleware**: Utility to combine multiple middleware handlers

**Error Response Format**:
```json
{
  "type": "ERROR_TYPE",
  "message": "Human readable message",
  "status_code": 400
}
```

### 4. Errors Module (`utils/errors/`)

**Purpose**: Custom error types with HTTP status mapping

**Integration Points**:
- Ready for use in handlers and business logic
- Provides functions: `Validation()`, `NotFound()`, `Conflict()`, `Internal()`, `Unauthorized()`, `Forbidden()`, `ServiceError()`

**Usage Example**:
```go
if video == nil {
  return nil, errors.NotFound("video not found: " + id)
}

if req.Title == "" {
  return nil, errors.Validation("title is required")
}
```

**Error Types & HTTP Status**:
- `VALIDATION_ERROR` → 400 Bad Request
- `NOT_FOUND` → 404 Not Found
- `CONFLICT` → 409 Conflict
- `UNAUTHORIZED` → 401 Unauthorized
- `FORBIDDEN` → 403 Forbidden
- `INTERNAL_ERROR` → 500 Internal Server Error
- `SERVICE_ERROR` → 503 Service Unavailable

## Service Initialization Flow

### Data Service
```
main()
  ↓
config.New("dataservice") - Load config from env
  ↓
config.Validate() - Validate configuration
  ↓
observability.NewLogger() - Initialize logger
  ↓
Initialize S3, Repository, Service
  ↓
middleware.ChainMiddleware() - Apply HTTP middleware
  ↓
Start HTTP/2 server on port 8081
Start gRPC server on port 50051
```

### Metadata Service
```
main()
  ↓
config.New("metadataservice") - Load config from env
  ↓
config.Validate() - Validate configuration
  ↓
observability.NewLogger() - Initialize logger
  ↓
Connect to MySQL database
  ↓
middleware.ChainMiddleware() - Apply HTTP middleware
  ↓
Start HTTP server on port 8080
```

## Configuration Example

### Environment Variables (`.env`)
```bash
# Common
ENVIRONMENT=dev
LOG_LEVEL=info
DEBUG_MODE=false

# Data Service
HTTP_PORT=8081
GRPC_PORT=50051
S3_REGION=us-east-1
S3_BUCKET=video-platform-storage

# Metadata Service
HTTP_PORT=8080
MYSQL_HOST=localhost
MYSQL_PORT=3306
MYSQL_USER=videouser
MYSQL_PASSWORD=videopass
MYSQL_DATABASE=videoplatform

# Observability
OTEL_ENABLED=false
OTEL_JAEGER_URL=http://localhost:14268/api/traces
```

## How Services Use Utils Modules

### Data Service (`dataservice/main.go`)
| Module | Usage |
|--------|-------|
| config | HTTP_PORT, GRPC_PORT, S3 settings |
| observability | Structured logging throughout service |
| middleware | Request logging, panic recovery, error handling |
| errors | Ready for use in handlers |

### Metadata Service (`metadataservice/main.go`)
| Module | Usage |
|--------|-------|
| config | HTTP_PORT, MYSQL credentials, timeouts |
| observability | Structured logging throughout service |
| middleware | Request logging, panic recovery, error handling |
| errors | Ready for use in handlers |

## Benefits of Utils Integration

1. **Centralized Configuration**: Single source of truth for all service configuration
2. **Consistent Logging**: Both services use same logging interface and format
3. **Standardized Middleware**: Uniform HTTP middleware across services
4. **Error Handling**: Consistent error types with proper HTTP status codes
5. **Scalability**: Easy to add new services that follow same pattern
6. **12-Factor Compliance**: Configuration from environment, stateless processes
7. **Observability Ready**: Foundation for OpenTelemetry integration
8. **DevOps Friendly**: Docker Compose can configure via environment variables

## Future Enhancements

1. **OpenTelemetry Integration**: Full distributed tracing support
2. **Metrics Export**: Prometheus metrics via middleware
3. **Request Tracing**: Correlation ID propagation across services
4. **Rate Limiting**: Middleware for request rate limiting
5. **Authentication/Authorization**: Auth middleware integration
6. **Circuit Breaker**: Resilience patterns in middleware
7. **Health Checks**: Enhanced health check endpoint with dependencies

## Running Services with Utils Integration

### Local Development
```bash
# Using docker-compose (loads env vars)
docker-compose up -d mysql jaeger prometheus grafana metadataservice dataservice

# Check logs
docker logs videostreamingplatform-dataservice
docker logs videostreamingplatform-metadataservice

# View OpenTelemetry traces
http://localhost:16686  # Jaeger UI
```

### Manual Testing
```bash
# Set environment variables
export ENVIRONMENT=dev
export LOG_LEVEL=debug
export HTTP_PORT=8081
export GRPC_PORT=50051
export S3_REGION=us-east-1

# Run data service
go run ./dataservice

# In another terminal, run metadata service
export HTTP_PORT=8080
export MYSQL_HOST=localhost
go run ./metadataservice
```

### Build & Deploy
```bash
# Build with utils modules integrated
make build

# Run tests
make test

# Docker build
make docker-build-all

# Deploy
make deploy-local
make deploy-prod
```

## Verification Checklist

- ✅ Both services compile with utils integration
- ✅ Tests pass (dataservice/bl tests)
- ✅ Config module loads from environment variables
- ✅ Observability logger works in both services
- ✅ Middleware chain applies to HTTP handlers
- ✅ Error handling ready for use in handlers
- ✅ Docker Compose provides environment configuration
- ✅ Makefile build/test/deploy targets work
- ✅ Health check endpoints functional
- ✅ Graceful shutdown timeouts configured

## Next Steps

1. Update service handlers to use `utils.errors` for consistent error handling
2. Add request tracing middleware with correlation IDs
3. Integrate Prometheus metrics export
4. Add full OpenTelemetry with Jaeger support
5. Implement circuit breaker pattern for external service calls
6. Add health check probes for all dependencies
7. Implement request rate limiting
8. Add API documentation with Swagger/OpenAPI

