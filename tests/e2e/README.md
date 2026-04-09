# E2E Tests — Video Streaming Agent

End-to-end tests that act as a video uploading/downloading agent against the
running platform.

## Tests

| Test | Description |
|------|-------------|
| `TestUploadAgent` | Full upload lifecycle: create video → initiate → chunk upload → progress tracking → complete |
| `TestDownloadAgent` | Upload a file, download it back, verify SHA-256 checksum integrity |
| `TestConcurrentAgents` | 5 parallel agents uploading + downloading simultaneously |
| `TestVideoCRUDLifecycle` | Metadata CRUD: create → get → update → list → delete → verify gone |

## Running

### Against local docker-compose (default)

```bash
# Start the stack first
docker compose -f build/docker-compose.yml up -d

# Run all e2e tests
go test -v -count=1 ./tests/e2e/...

# Run a specific test
go test -v -count=1 -run TestUploadAgent ./tests/e2e/...
```

### Against a remote / production setup

```bash
E2E_METADATA_URL=https://meta.example.com \
E2E_DATA_URL=https://data.example.com \
go test -v -count=1 ./tests/e2e/...
```

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `E2E_METADATA_URL` | `http://localhost:8080` | Metadata service base URL |
| `E2E_DATA_URL` | `http://localhost:8081` | Data service base URL |

## Notes

- Tests auto-skip if services are unreachable (no hard failure).
- Each test is independent and creates its own test data.
- The download test verifies data integrity via SHA-256 checksums.
- Concurrent test runs 5 agents in parallel to stress-test the platform.
