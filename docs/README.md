# Video Streaming Platform Documentation

Welcome to the Video Streaming Platform documentation. This folder contains all project documentation, guides, and specification files.

## Documentation Index

### Core Documentation
- **[PRD (Product Requirements Document)](prd.md)** - Project requirements, architecture, and specifications
- **[DEPLOYMENT.md](DEPLOYMENT.md)** - Deployment guide and infrastructure setup

### Engineering Excellence
- **[ENGINEERING_EXCELLENCE.md](ENGINEERING_EXCELLENCE.md)** - Comprehensive guide on 7 engineering excellence guidelines implementation
- **[ENGINEERING_EXCELLENCE_SUMMARY.md](ENGINEERING_EXCELLENCE_SUMMARY.md)** - Executive summary of engineering excellence features

### Architecture & Implementation
- **[HTTP2_STREAMING.md](HTTP2_STREAMING.md)** - HTTP/2 streaming implementation details and API specification
- **[SERVICE_INTEGRATION_GUIDE.md](SERVICE_INTEGRATION_GUIDE.md)** - Step-by-step guide for integrating observability packages into services

### Testing
- **[TESTS.md](TESTS.md)** - Comprehensive test documentation and coverage overview
- **[UNIT_TESTS_COMPLETE.md](UNIT_TESTS_COMPLETE.md)** - Unit test implementation summary and results

### Session Records
- **[SESSION_SUMMARY.md](SESSION_SUMMARY.md)** - Session summary and progress tracking

---

## Quick Navigation

### For New Developers
1. Start with [PRD](prd.md) to understand project scope
2. Review [DEPLOYMENT.md](DEPLOYMENT.md) to set up development environment
3. Check [SERVICE_INTEGRATION_GUIDE.md](SERVICE_INTEGRATION_GUIDE.md) for service setup

### For Implementation Details
- HTTP/2 Streaming: See [HTTP2_STREAMING.md](HTTP2_STREAMING.md)
- Testing Strategy: See [TESTS.md](TESTS.md)
- Health Checks & Observability: See [ENGINEERING_EXCELLENCE.md](ENGINEERING_EXCELLENCE.md)

### For Code Quality & Standards
- Engineering Excellence: See [ENGINEERING_EXCELLENCE.md](ENGINEERING_EXCELLENCE.md)
- SOLID Principles: Section 1 of [ENGINEERING_EXCELLENCE.md](ENGINEERING_EXCELLENCE.md)
- DevOps Practices: Section 6 of [ENGINEERING_EXCELLENCE.md](ENGINEERING_EXCELLENCE.md)

---

## Project Structure

```
videostreamingplatform/
├── docs/                    # All documentation (this folder)
├── cmd/                     # Service entry points
│   ├── data-service/        # Video data streaming service
│   └── metadata-service/    # Video metadata service
├── internal/                # Internal packages
│   ├── bootstrap/           # Service initialization
│   ├── config/              # Configuration management
│   ├── db/                  # Database layer
│   ├── events/              # Event schemas
│   ├── handlers/            # HTTP handlers
│   ├── health/              # Health checks
│   ├── interfaces/          # Interface definitions
│   ├── logger/              # Structured logging
│   ├── metrics/             # Metrics collection
│   ├── middleware/          # HTTP middleware
│   ├── models/              # Data models
│   ├── progress/            # Progress tracking
│   ├── retry/               # Retry logic
│   ├── storage/             # Storage layer
│   ├── streaming/           # Streaming implementation
│   ├── trace/               # Distributed tracing
│   └── docs/                # API documentation
├── k8s/                     # Kubernetes manifests
├── terraform/               # Infrastructure as Code
├── .github/                 # GitHub Actions workflows
└── Makefile                 # Build automation
```

---

## Key Files Reference

| File | Purpose |
|------|---------|
| `go.mod` | Go module dependencies |
| `docker-compose.yaml` | Local development environment |
| `Makefile` | Build and deployment commands |
| `.github/workflows/` | CI/CD pipelines |

---

## Getting Started

### Local Development
```bash
# 1. Clone repository
git clone <repo-url>
cd videostreamingplatform

# 2. Set up environment
cp .env.example .env
docker-compose up

# 3. Run services
make run-data-service
make run-metadata-service
```

### Running Tests
```bash
make test
make test-coverage
```

### Building for Production
```bash
make build-docker
make deploy-aws
```

See [DEPLOYMENT.md](DEPLOYMENT.md) for detailed instructions.

---

## Documentation Standards

All documentation follows these conventions:
- **Markdown Format**: All docs are in `.md` format
- **Headers**: Use H1 for titles, H2 for sections, H3 for subsections
- **Code Blocks**: Use fenced code blocks with language specification
- **Links**: Relative paths within docs folder

---

## Contributing to Documentation

When adding new documentation:
1. Create `.md` file in `/docs` folder
2. Update this README with a link and description
3. Follow Markdown formatting standards
4. Include code examples where applicable
5. Keep technical content clear and concise

---

## Support

For questions about documentation or project:
- Check the relevant `.md` file in this folder
- Review code comments in `internal/` packages
- Check GitHub Issues for known problems

---

**Last Updated**: April 7, 2026
**Project Version**: 0.1.0
