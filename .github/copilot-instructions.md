# Copilot Instructions

## Build, test, and lint commands

- Main automation lives in `build/Makefile`, so run targets as `make -f build/Makefile <target>`.
- Build both services: `make -f build/Makefile build`
- Build one service: `make -f build/Makefile build-dataservice` or `make -f build/Makefile build-metadataservice`
- Run both test suites tracked by the Makefile: `make -f build/Makefile test`
- Run one service test suite: `make -f build/Makefile test-dataservice` or `make -f build/Makefile test-metadataservice`
- Run the same broad Go test command used in CI: `go test -v -race -coverprofile=coverage.out -covermode=atomic ./...`
- Run a single unit test: `go test -v -run TestCompleteUpload ./dataservice/bl`
- Run all e2e tests against the local stack: `docker compose -f build/docker-compose.yml up -d && go test -v -count=1 ./tests/e2e/...`
- Run one e2e test: `go test -v -count=1 -run TestUploadAgent ./tests/e2e/...`
- Lint exactly like the repo tooling expects: `make -f build/Makefile lint` (`go vet ./...` + `golangci-lint run ./...`)
- Format Go code: `make -f build/Makefile fmt`

## High-level architecture

- This repository is split into two Go services with a small shared `utils/` layer. `metadataservice` owns video CRUD and persists records in MySQL. `dataservice` owns upload/download execution, stores objects in S3-compatible storage, and also exposes a gRPC API alongside HTTP.
- The upload flow spans both services: metadata is created in `metadataservice`, upload sessions are initiated in `dataservice`, each chunk is written to object storage as `videos/<videoID>/chunk_<n>`, and completion merges those chunk objects into the final `videos/<videoID>` object. The e2e suite in `tests/e2e` exercises that full cross-service path.
- Local development is centered on `build/docker-compose.yml`, which brings up MySQL, LocalStack S3, Jaeger, Prometheus, Grafana, `metadataservice` on `:8080`, and `dataservice` on `:8081` plus gRPC on `:50051`.
- `metadataservice` uses MySQL-backed repositories in `metadataservice/db` and `metadataservice/dl`. `dataservice` currently keeps upload session state in memory via `dataservice/dl/memory.go`, so upload progress is process-local even though chunk data is stored remotely.

## Key conventions

- Each service follows the same wiring pattern from `main.go`: load shared env config from `utils/config`, initialize shared observability helpers from `utils/observability`, construct `dl` repositories, pass them into `bl` services, then inject those services into HTTP handlers.
- Both HTTP servers use the standard library `http.ServeMux` with method-aware patterns such as `POST /uploads/initiate` and path extraction via `r.PathValue(...)`; do not introduce a different router style unless the repo is already moving that way.
- Shared HTTP concerns are wrapped the same way in both services: build the mux, apply `middleware.ChainMiddleware(...)` with logging and panic recovery, then wrap the final handler with `otelhttp.NewHandler(...)`. Keep `/health` and `/metrics` endpoints present when touching service startup.
- Configuration is env-driven and centralized in `utils/config`. Both services default `HTTP_PORT` to `8080`, so `dataservice` must override `HTTP_PORT=8081` when both services run together. Default gRPC port is `50051`.
- Upload chunk sizing is a repository-wide assumption: HTTP handlers, gRPC handlers, and tests all use fixed 5 MiB chunks. Preserve that expectation unless you update the full flow together.
- E2E tests are endpoint-driven through `E2E_METADATA_URL` and `E2E_DATA_URL`, defaulting to `http://localhost:8080` and `http://localhost:8081`. Use those variables instead of hard-coding alternate hosts in tests or helper scripts.
