# Build Directory Structure

This directory contains all build-related artifacts, configuration, and scripts for the Video Streaming Platform.

## Directory Structure

```
build/
├── docker/                         # Docker containerization
│   ├── dataservice.Dockerfile     # Data Service container image
│   ├── metadataservice.Dockerfile # Metadata Service container image
│   ├── build.sh                   # Docker build automation script
│   └── prometheus.yml             # Prometheus monitoring config
├── scripts/                       # Build and deployment scripts
│   └── ...                        # Deployment automation
├── Makefile                       # Build automation targets
├── docker-compose.yml             # Docker Compose for local development
└── docker-compose.yaml            # Docker Compose alternative
```

## Build Automation

### Makefile

The root Makefile has been moved to `build/Makefile` and contains targets for:

```bash
make build              # Build all service binaries
make test              # Run all tests
make docker-build-all  # Build all Docker images
make run               # Run services locally
make deploy-local      # Deploy to local Kubernetes
make deploy-prod       # Deploy to production
```

Usage from project root:
```bash
make -f build/Makefile build
make -f build/Makefile test
```

Or from build directory:
```bash
cd build
make build
make test
```

### Docker Images

#### Building Images

Manually with Docker:
```bash
docker build -f build/docker/dataservice.Dockerfile -t videostreamingplatform/dataservice:latest .
docker build -f build/docker/metadataservice.Dockerfile -t videostreamingplatform/metadataservice:latest .
```

Or use the build script:
```bash
bash build/docker/build.sh
```

Or use Makefile (if available):
```bash
make docker-build-all
```

#### Image Details

**Data Service** (`dataservice.Dockerfile`):
- Multi-stage build for minimal image size
- Base: Alpine Linux 3.18
- Ports: 8081 (HTTP/2), 50051 (gRPC)
- Health check included
- Non-root user execution

**Metadata Service** (`metadataservice.Dockerfile`):
- Multi-stage build for minimal image size
- Base: Alpine Linux 3.18
- Port: 8080 (HTTP REST)
- Health check included
- Non-root user execution

### Docker Compose

The `docker-compose.yml` file provides a complete local development stack:

```bash
# Start all services and dependencies
docker-compose -f build/docker-compose.yml up -d

# Logs
docker-compose -f build/docker-compose.yml logs -f

# Stop
docker-compose -f build/docker-compose.yml down
```

**Services in Compose**:
- `mysql` - Database (port 3306)
- `localstack` - AWS S3 mock (port 4566)
- `jaeger` - Distributed tracing UI (port 16686)
- `prometheus` - Metrics (port 9090)
- `grafana` - Dashboards (port 3000)
- `metadataservice` - Metadata Service (port 8080)
- `dataservice` - Data Service (ports 8081, 50051)

## Binary Artifacts

The `build/` directory may contain compiled binaries after building:
- `dataservice` - Compiled Data Service binary
- `metadataservice` - Compiled Metadata Service binary
- `metadata-service` - Legacy binary name (deprecated)
- `data-service` - Legacy binary name (deprecated)

These are generated during build and can be safely deleted with `make clean`.

## Scripts

The `scripts/` directory contains:
- Kubernetes deployment automation
- CI/CD pipeline scripts
- Infrastructure setup scripts

## Configuration Files

### Prometheus Configuration

`docker/prometheus.yml` - Prometheus scrape configuration for metrics collection.

## Best Practices

1. **All Docker images** should be built from this directory
2. **No Dockerfiles** in service directories
3. **No Makefiles** in service directories
4. **All build artifacts** consolidated in `build/` folder
5. **Docker Compose** for local development only (same stack as production)
6. **Build scripts** in `build/scripts/` for automation

## Development Workflow

### Local Development
```bash
# Build binaries
make -f build/Makefile build

# Run tests
make -f build/Makefile test

# Start services with dependencies
docker-compose -f build/docker-compose.yml up -d

# View logs
docker-compose -f build/docker-compose.yml logs -f metadataservice
docker-compose -f build/docker-compose.yml logs -f dataservice
```

### Building Docker Images
```bash
# Build all images
bash build/docker/build.sh

# Or individual images
docker build -f build/docker/dataservice.Dockerfile -t videostreamingplatform/dataservice:v1.0.0 .
docker build -f build/docker/metadataservice.Dockerfile -t videostreamingplatform/metadataservice:v1.0.0 .
```

### Production Deployment
```bash
# Build and tag images
make -f build/Makefile docker-build-all VERSION=v1.0.0

# Deploy to AWS EKS
make -f build/Makefile deploy-prod ENVIRONMENT=prod

# Or manually with kubectl
kubectl apply -f build/scripts/k8s-manifests/
```

## CI/CD Integration

The build system is designed to work with CI/CD platforms:

### GitHub Actions
```yaml
- name: Build binaries
  run: make -f build/Makefile build

- name: Run tests
  run: make -f build/Makefile test

- name: Build Docker images
  run: bash build/docker/build.sh
```

### Environment Variables

- `REGISTRY` - Docker registry (default: videostreamingplatform)
- `VERSION` - Image/binary version (default: latest)
- `ENVIRONMENT` - Deployment environment (dev/staging/prod)

## Troubleshooting

### Build Failures

Check if all dependencies are installed:
```bash
go mod download
go mod tidy
```

### Docker Build Issues

Verify Dockerfile paths:
```bash
ls -la build/docker/*.Dockerfile
```

### Docker Compose Issues

Check service connectivity:
```bash
docker network ls
docker-compose -f build/docker-compose.yml ps
```

## References

- See parent README.md for project overview
- See docs/ folder for detailed documentation
- See individual service folders for service-specific implementation
