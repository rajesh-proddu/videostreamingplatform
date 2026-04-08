# Video Streaming Platform

HTTP/2 enabled video streaming platform with resumable uploads/downloads, distributed tracing, and comprehensive observability.

**Project Status**: 🚀 Production-Ready Architecture with Engineering Excellence

---

## 📋 Documentation

📚 **All documentation has been moved to the [/docs](docs/) folder**

### Key Documentation Files
- **[Project Requirements](docs/prd.md)** - Feature requirements and architecture
- **[Deployment Guide](docs/DEPLOYMENT.md)** - Infrastructure & deployment setup
- **[Engineering Excellence](docs/ENGINEERING_EXCELLENCE.md)** - 7 guidelines implementation (SOLID, 12-factor, observability)
- **[HTTP/2 Streaming](docs/HTTP2_STREAMING.md)** - API specification and implementation details
- **[Service Integration](docs/SERVICE_INTEGRATION_GUIDE.md)** - How to integrate new packages into services
- **[Testing Documentation](docs/TESTS.md)** - Test strategy and coverage

👉 **[Browse all documentation in /docs](docs/README.md)**

---

## 🎯 Quick Start

### Local Development
```bash
# Clone and setup
git clone <repo-url>
cd videostreamingplatform
cp .env.example .env

# Start services
docker-compose up

# Run tests
make test

# Start data service
make run-data-service

# Start metadata service
make run-metadata-service
```

### Health Check
```bash
curl http://localhost:8080/health
curl http://localhost:8081/health
```

### Metrics & Monitoring
```bash
# Prometheus metrics
curl http://localhost:8080/metrics

# API Documentation
open http://localhost:8080/swagger/
```

---

## 🏗️ Architecture

```
Video Streaming Platform
├── Metadata Service (Port 8080)
│   ├── Video CRUD operations
│   ├── Metadata management
│   └── MySQL backend
│
└── Data Service (Port 8081)
    ├── HTTP/2 streaming upload/download
    ├── Chunked transfer support
    ├── Progress tracking
    └── S3/MinIO backend
```

### Deployment
- **Local**: Docker Compose
- **Development**: Kubernetes (Kind)
- **Production**: AWS EKS with Terraform IaC

---

## ✨ Features

### Video Streaming
- ✅ HTTP/2 protocol support
- ✅ Resumable uploads (chunked transfer)
- ✅ Resumable downloads (range requests)
- ✅ MD5 checksum verification
- ✅ Real-time progress tracking with speed/ETA

### Reliability
- ✅ Exponential backoff retry logic
- ✅ Automatic error recovery
- ✅ Context-aware timeouts
- ✅ Request deduplication

### Observability
- ✅ Structured logging with trace ID propagation
- ✅ OpenTelemetry distributed tracing
- ✅ Prometheus metrics (20+ metrics)
- ✅ Health checks for orchestration
- ✅ Graceful error handling

### Engineering Excellence
- ✅ SOLID design principles
- ✅ 12-factor app compliance
- ✅ Interface-based architecture
- ✅ Comprehensive API documentation
- ✅ Event schema versioning

---

## 📦 Technology Stack

| Component | Technology | Version |
|-----------|-----------|---------|
| Language (Services) | Go | 1.24 |
| Language (Data) | Python | 3.11+ |
| Streaming | HTTP/2 | Protocol |
| Database | MySQL | 8.0 |
| Storage | S3/MinIO | AWS SDK v2 |
| Streaming Queue | Kafka | 3.0+ |
| Container | Docker | 20.10+ |
| Orchestration | Kubernetes | 1.28+ |
| IaC | Terraform | 1.0+ |
| Local K8s | Kind | Latest |
| CI/CD | GitHub Actions | Native |
| Metrics | Prometheus | 2.45+ |
| Tracing | OpenTelemetry | 1.20+ |
| Visualization | Grafana | 10+ |

---

## 📁 Project Structure

```
videostreamingplatform/
├── docs/                    # 📚 All documentation (9 guides)
├── cmd/                     # 🚀 Service entry points
│   ├── data-service/
│   └── metadata-service/
├── internal/                # 🔧 Internal packages (16 modules)
│   ├── bootstrap/           # Service initialization
│   ├── config/              # 12-factor config
│   ├── handlers/            # HTTP handlers
│   ├── streaming/           # HTTP/2 streaming
│   ├── logger/              # Structured logging
│   ├── trace/               # Distributed tracing
│   ├── metrics/             # Prometheus metrics
│   ├── health/              # Health checks
│   ├── interfaces/          # Interface definitions
│   ├── events/              # Event schemas
│   ├── middleware/          # HTTP middleware
│   ├── db/                  # Database layer
│   ├── storage/             # S3 storage
│   ├── progress/            # Progress tracking
│   └── retry/               # Retry logic
├── k8s/                     # Kubernetes manifests (8 files)
├── terraform/               # Infrastructure as Code (5 modules)
├── .github/workflows/       # CI/CD pipelines (4 workflows)
├── tests/                   # Integration tests
└── Makefile                 # Build automation
```

---

## 🧪 Testing

```bash
# Run all tests
make test

# Run specific test suite
go test ./internal/streaming -v
go test ./internal/retry -v
go test ./internal/progress -v

# Run with coverage
make test-coverage

# Integration tests
./integration-test.sh

# Concurrent streaming test
./concurrent-test.sh
```

**Test Coverage**: 44+ unit tests across 3 packages with 100% pass rate

---

## 🚀 Deployment

### Quick Deploy
```bash
# Local (Docker Compose)
docker-compose up

# Kubernetes (Kind)
make deploy-kind

# AWS (EKS)
make deploy-aws
```

For detailed instructions, see [DEPLOYMENT.md](docs/DEPLOYMENT.md)

---

## 📊 Monitoring & Observability

### Health Checks
```bash
# Service health
curl http://localhost:8080/health

# Detailed component health
{
  "status": "healthy",
  "timestamp": "2026-04-07T10:30:00Z",
  "components": [
    {"name": "database", "status": "healthy"},
    {"name": "storage", "status": "healthy"},
    {"name": "service", "status": "healthy"}
  ]
}
```

### Metrics
- **HTTP Requests**: 4 metrics (count, duration, request size, response size)
- **Upload Operations**: 5 metrics (count, bytes, duration, errors)
- **Download Operations**: 2 metrics (count, bytes)
- **Database Queries**: 2 metrics (duration, errors)
- **Retry Logic**: 2 metrics (attempts, successes)

### Tracing
- OpenTelemetry OTLP HTTP exporter
- Trace ID propagation across services
- Span creation for all operations

### Logs
- Structured JSON-like format
- Service, version, environment metadata
- Trace ID and Span ID in every log entry

---

## 🛠️ Development

### Required Tools
```bash
# Go 1.24
make install-go

# Docker & Docker Compose
make install-docker

# Kubernetes tools
make install-k8s
```

### Code Organization Guidelines
- **Packages**: Single responsibility per package
- **Interfaces**: Composition over inheritance
- **Errors**: Wrapped with context
- **Logging**: Structured logging with context
- **Testing**: Table-driven tests where applicable

---

## 🔄 Development Workflow

### 1. Create Feature Branch
```bash
git checkout -b feature/my-feature
```

### 2. Implement with Tests
```bash
# Make changes
vi internal/mypackage/myfile.go

# Run tests
make test

# Check coverage
make test-coverage
```

### 3. Commit & Push
```bash
git add .
git commit -m "feat: add my feature"
git push origin feature/my-feature
```

### 4. Pull Request
- PR runs automated tests via GitHub Actions
- Reviews for code quality and engineering excellence
- Merges to main trigger build and deploy

---

## 📝 API Endpoints

### Metadata Service (Port 8080)
| Method | Endpoint | Purpose |
|--------|----------|---------|
| GET | `/health` | Health check |
| GET | `/metrics` | Prometheus metrics |
| POST | `/videos` | Create video metadata |
| GET | `/videos` | List videos |
| GET | `/videos/{id}` | Get video details |
| PUT | `/videos/{id}` | Update video |
| DELETE | `/videos/{id}` | Delete video |

### Data Service (Port 8081)
| Method | Endpoint | Purpose |
|--------|----------|---------|
| GET | `/health` | Health check |
| GET | `/metrics` | Prometheus metrics |
| POST | `/uploads/initiate` | Start chunked upload |
| POST | `/uploads/{id}/chunks` | Upload chunk |
| GET | `/uploads/{id}/progress` | Get upload progress |
| POST | `/uploads/{id}/complete` | Finalize upload |
| GET | `/videos/{id}/download` | Stream download |

See [HTTP2_STREAMING.md](docs/HTTP2_STREAMING.md) for detailed API docs.

---

## 🔐 Security

- ✅ No hardcoded credentials (12-factor compliance)
- ✅ IAM role-based AWS access
- ✅ Network policies in Kubernetes
- ✅ Input validation on all endpoints
- ✅ Error messages don't leak sensitive info

---

## 📈 Performance Characteristics

- **1K requests/sec** target (per PRD)
- **Chunked uploads**: 5MB default chunk size
- **Streaming**: HTTP/2 multiplexing
- **Retry**: Exponential backoff (100ms - 10s)
- **Timeouts**: Configurable per environment

---

## 🐛 Troubleshooting

### Services Won't Start
```bash
# Check environment variables
printenv | grep MYSQL

# Verify database connection
mysql -h localhost -u videouser -p

# Check port availability
lsof -i :8080
```

### Health Check Failing
```bash
# Check database
curl http://localhost:8080/health | jq .components

# Check storage
aws s3 ls s3://videostreamingplatform/
```

### Metrics Not Recording
```bash
# Verify metrics enabled
echo $METRICS_ENABLED

# Check Prometheus endpoint
curl http://localhost:8080/metrics
```

For more troubleshooting, see [DEPLOYMENT.md](docs/DEPLOYMENT.md#troubleshooting)

---

## 📞 Support & Contributing

### Documentation
- 📚 See [/docs](docs/) for comprehensive guides
- 💬 GitHub Issues for questions
- 🔍 GitHub Discussions for ideas

### Contributing
1. Fork repository
2. Create feature branch
3. Make changes with tests
4. Submit pull request
5. Code review and merge

---

## 📜 License

Proprietary - All rights reserved

---

## 🎉 Key Achievements

✅ **16 internal packages** implementing enterprise patterns
✅ **44 unit tests** with 100% pass rate
✅ **7 engineering excellence guidelines** fully implemented
✅ **9 documentation files** covering all aspects
✅ **4 CI/CD workflows** for automated testing & deployment
✅ **Production-ready architecture** with Kubernetes & Terraform

---

## Version Info

- **Project Version**: 0.1.0
- **Go Version**: 1.24
- **API Version**: v1
- **Last Updated**: April 7, 2026

---

**[→ Browse Documentation](docs/README.md)**
