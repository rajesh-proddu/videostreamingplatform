# Component Tests

This folder contains integration and component-level tests for the Video Streaming Platform.

## Test Scripts

### test-streaming.sh
Full end-to-end HTTP/2 streaming test with:
- Service health checks
- Video metadata creation
- Upload session initiation
- Chunked file upload
- Upload progress tracking
- File download with resume support
- Checksum verification of uploaded/downloaded files

**Usage:**
```bash
./test-streaming.sh
```

**Test File Size:** 10MB (configurable)
**Chunk Size:** 5MB (configurable)

### integration-test.sh
Quick integration test focusing on:
- Service health endpoints
- Video creation API
- Upload session initiation
- Progress endpoint verification

**Usage:**
```bash
./integration-test.sh
```

**Duration:** ~30 seconds
**Purpose:** Quick smoke test to verify core functionality

### concurrent-test.sh
Concurrent transfer test simulating:
- 5 simultaneous uploads
- 60-second duration with progress monitoring
- Performance metrics collection
- Parallel file handling

**Usage:**
```bash
./concurrent-test.sh
```

**Configuration:**
- Concurrent uploads: 5
- Test duration: 60 seconds
- Progress monitoring: Every 10 seconds
- File size per upload: 5MB

## Prerequisites

1. **Services Running**
   ```bash
   # Start services
   docker-compose up
   # Or run manually
   go run cmd/data-service/main.go &
   go run cmd/metadata-service/main.go &
   ```

2. **Environment Variables**
   ```bash
   export MYSQL_HOST=localhost
   export MYSQL_PORT=3306
   export S3_ENDPOINT=http://localhost:9000
   export S3_BUCKET=videostreamingplatform
   ```

3. **Build Services**
   ```bash
   make build
   ```

## Running Tests

### Individual Test
```bash
cd tests/component

# Full end-to-end test
./test-streaming.sh

# Quick smoke test
./integration-test.sh

# Concurrent test
./concurrent-test.sh
```

### All Tests
```bash
# From root directory
make test-integration

# Or run all tests:
./tests/component/test-streaming.sh && \
./tests/component/integration-test.sh && \
./tests/component/concurrent-test.sh
```

## Expected Results

### test-streaming.sh
```
✅ All tests passed!
- Video creation: ✓
- Upload initiation: ✓
- Chunk upload: ✓
- Progress tracking: ✓
- Download: ✓
- Checksum verification: ✓
```

### integration-test.sh
```
✅ All tests passed!
✓ Health endpoints working
✓ Video creation working
✓ Upload initiation working
✓ Progress tracking working
```

### concurrent-test.sh
```
✅ Concurrent transfer test completed
- All 5 uploads initiated
- Progress monitored
- Services handled concurrent requests
```

## Troubleshooting

### Services Not Responding
```bash
# Check if services are running
curl http://localhost:8080/health
curl http://localhost:8081/health

# Check service logs
tail -f /tmp/metadata.log
tail -f /tmp/data.log
```

### Test Fails with Connection Error
```bash
# Ensure services are started
docker-compose up

# Or if running manually:
make run-metadata-service &
make run-data-service &
```

### File Upload Fails
```bash
# Verify storage is accessible
aws s3 ls s3://videostreamingplatform/ --endpoint-url http://localhost:9000

# Check storage permissions
echo $AWS_ACCESS_KEY_ID
echo $AWS_SECRET_ACCESS_KEY
```

## Performance Metrics

Tests will output metrics like:
- **Upload Speed:** Mbps
- **Progress Percentage:** %
- **Upload Duration:** seconds
- **Checksum Match:** yes/no

## Test Coverage

These component tests cover:
- ✅ HTTP/2 streaming protocol
- ✅ Chunked upload/download
- ✅ Resume support (range requests)
- ✅ Checksum verification
- ✅ Progress tracking
- ✅ Concurrent operations
- ✅ Error handling
- ✅ Health checks

## Adding New Tests

To add new component tests:

1. Create new shell script in this folder
2. Follow naming convention: `*-test.sh`
3. Include:
   - Service startup/health checks
   - Test setup
   - Test execution with logging
   - Cleanup
   - Exit code handling

4. Add to test runner in Makefile:
   ```makefile
   test-integration:
   	./tests/component/new-test.sh
   ```

## Continuous Integration

These tests are executed in GitHub Actions on:
- Every pull request (quick smoke test)
- Merge to main (full test suite)
- Scheduled daily runs

See `.github/workflows/test.yml` for CI configuration.
