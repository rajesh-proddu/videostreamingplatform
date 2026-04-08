# Video Streaming Platform - Modular Monolith Architecture

## Project Overview

This document describes the restructuring of the Video Streaming Platform into a modular monolith architecture using independent microservices. The first service implemented is the **Data Service** for managing video uploads.

## Modular Monolith Approach

The platform is being restructured into loosely-coupled, independently deployable services while maintaining shared infrastructure:

```
┌─────────────────────────────────────┐
│   API Gateway / Load Balancer       │
└──────────────┬──────────────────────┘
               │
        ┌──────┴──────┬──────────┬──────────┐
        │             │          │          │
   ┌────▼─────┐  ┌───▼──────┐  │         ...
   │  API     │  │  Data    │  │
   │ Service  │  │ Service  │  │
   └──────────┘  └──────────┘  │
        │             │        │
   ┌────▼─────────────▼────┬───▼─────┐
   │   Shared Database     │ Storage │
   │   Caching Layer       │ (S3)    │
   │   Message Queue       │         │
   └───────────────────────┴─────────┘
```

## Implemented Services

### 1. Data Service ✓
**Location**: `dataservice/`
**Status**: Complete and tested
**Responsibility**: Upload session management and progress tracking

**Components**:
- Upload session lifecycle management
- Progress tracking with speed and ETA
- Chunk upload recording
- In-memory storage (ready for database)
- gRPC API

**Key Classes**:
- `UploadService`: Business logic
- `InMemoryUploadRepository`: Data access
- `DataServiceServer`: gRPC implementation

## Planned Services

### 2. Metadata Service
- Video metadata management
- User profile and settings
- Playlist management
- Video discovery

### 3. Streaming Service
- Video streaming delivery
- Transcoding orchestration
- Adaptive bitrate selection
- CDN integration

### 4. User Service
- Authentication and authorization
- User management
- Session management
- Integration with identity provider

### 5. Recommendation Service
- Personalized recommendations
- Trending videos
- Search functionality
- Analytics

## Architecture Benefits

### Scalability
- Each service can scale independently
- Services deployed across multiple instances
- Load balancing per service

### Maintainability
- Clear separation of concerns
- Each service has well-defined APIs
- Smaller, focused codebases
- Easier to understand and modify

### Deployability
- Services deployed independently
- No synchronization required
- Blue-green deployments possible
- Canary releases supported

### Team Autonomy
- Teams can develop services independently
- Clear API contracts
- Different technology stacks per service possible
- Parallel development

## Communication Patterns

### Synchronous Communication
- **gRPC**: Used by Data Service
- **HTTP/REST**: Alternative for services
- Point-to-point calls
- Suitable for immediate responses

### Asynchronous Communication (Future)
- **Event-driven**: Using message queues
- **Event streaming**: Using Kafka/Pulsar
- Decoupled services
- Better resilience

## Data Management

### Current Approach
- In-memory storage per service
- Each service manages its own data
- Database integration in progress

### Future Approach
- Separate database per service (if needed)
- Shared read replicas for analytics
- Event sourcing for critical data
- Eventual consistency pattern

## Integration Points

### API Gateway
- Routes requests to appropriate services
- Authentication middleware
- Rate limiting
- API versioning

### Service Discovery
- Services register on startup
- Health checks
- Load balancing
- Failover handling

### Monitoring & Observability
- Distributed tracing across services
- Centralized logging
- Metrics collection
- Performance monitoring

## Build & Deployment

### Development Environment
```bash
# Clone and setup
git clone <repository>
cd videostreamingplatform

# Build all services
go build -o bin/data-service ./dataservice

# Run services locally
./bin/data-service -port 50051
```

### Docker Deployment
```bash
# Build Docker image
docker build -t data-service:latest ./dataservice

# Run with Docker Compose
docker-compose up
```

### Kubernetes Deployment (Future)
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: data-service
spec:
  replicas: 3
  selector:
    matchLabels:
      app: data-service
  template:
    metadata:
      labels:
        app: data-service
    spec:
      containers:
      - name: data-service
        image: data-service:latest
        ports:
        - containerPort: 50051
```

## Data Service Details

### Upload Workflow

```
1. Client initiates upload
   → InitiateUpload() creates session
   ← Returns upload_id and chunk_size

2. Client uploads chunks sequentially
   → UploadChunk() for each chunk
   ← Progress updated
   ← ETA calculated

3. Client monitors progress
   → GetUploadProgress() gets status
   ← Returns % complete, speed, ETA

4. Client marks upload complete
   → CompleteUpload() finalizes
   ← Upload marked as COMPLETED
```

### Data Models

**Upload Session**:
- ID: Unique identifier
- VideoID: Associated video
- UserID: Owner of upload
- Status: PENDING | IN_PROGRESS | COMPLETED | FAILED
- Progress: Bytes uploaded, chunks, percentage
- Metrics: Speed (MB/s), estimated time remaining
- Timestamps: Created, updated, completed

## Testing Strategy

### Unit Tests
- Service layer tests
- Business logic validation
- Error handling

### Integration Tests
- End-to-end workflows
- API contract testing
- Database integration

### Load Testing
- Concurrent upload simulation
- Performance benchmarks
- Resource utilization

## Documentation Structure

- `README.md` - Service-specific documentation
- `DATASERVICE_QUICKSTART.md` - Quick start guide
- `DATASERVICE_IMPLEMENTATION.md` - Implementation details
- This file - Architecture overview

## Performance Targets

- **Upload initiation**: < 100ms
- **Chunk upload**: < 500ms (for 5MB)
- **Progress query**: < 50ms
- **Concurrent uploads**: 1000+
- **Memory per upload**: < 10KB

## Security Considerations

### Current Stage
- No authentication (local development)
- No encryption for uploads
- No rate limiting

### Production Requirements
- OAuth/JWT authentication
- TLS encryption
- Rate limiting per user
- Input validation
- Audit logging
- DDoS protection

## Monitoring & Observability

### Logs
- Structured logging (JSON format)
- Log levels: DEBUG, INFO, WARN, ERROR
- Correlation IDs for tracing

### Metrics
- Service health
- Request latency
- Error rates
- Resource utilization
- Upload statistics

### Tracing
- Distributed tracing across services
- Request flow visualization
- Performance profiling

## Roadmap

### Phase 1 (Current) ✓
- Data Service implemented
- In-memory storage
- Basic gRPC API
- Unit tests

### Phase 2
- Database integration (PostgreSQL)
- Metadata Service
- Authentication service
- API Gateway

### Phase 3
- Streaming Service
- Video transcoding
- Storage optimization
- Advanced monitoring

### Phase 4
- Recommendation engine
- Search functionality
- Analytics
- Performance optimization

## Contributing Guidelines

### Code Standards
- Follow Go conventions
- Clear naming and documentation
- Comprehensive testing
- Error handling

### Development Workflow
1. Create feature branch
2. Implement feature with tests
3. Code review
4. Merge to main
5. Deploy (with CI/CD)

### Service Addition Checklist
- [ ] Define gRPC/REST API
- [ ] Implement service logic
- [ ] Add unit tests
- [ ] Write documentation
- [ ] Create Docker image
- [ ] Test integration
- [ ] Deploy to staging
- [ ] Production release

## References

### External Documentation
- gRPC: https://grpc.io/docs/
- Go: https://golang.org/doc/
- Protocol Buffers: https://developers.google.com/protocol-buffers
- Docker: https://docs.docker.com/

### Internal Documentation
- [Data Service README](./dataservice/README.md)
- [Quick Start Guide](./DATASERVICE_QUICKSTART.md)
- [Implementation Details](./DATASERVICE_IMPLEMENTATION.md)

## Contact & Support

- Architecture Lead: [TBD]
- Data Service Owner: [TBD]
- Team: Video Streaming Platform Team

---

**Last Updated**: 2024
**Version**: 1.0
**Status**: In Development
