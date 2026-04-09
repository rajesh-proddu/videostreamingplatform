// Package e2e contains end-to-end tests that exercise the video streaming
// platform as a real client would — creating metadata, uploading binary data
// in chunks, tracking progress, completing the upload, and downloading the
// result for verification.
//
// Target environment is configured via environment variables:
//
//	E2E_METADATA_URL  (default http://localhost:8080)
//	E2E_DATA_URL      (default http://localhost:8081)
//
// Run against local docker-compose:
//
//	go test -v -tags=e2e -count=1 ./tests/e2e/...
//
// Run against a remote/prod setup:
//
//	E2E_METADATA_URL=https://meta.example.com \
//	E2E_DATA_URL=https://data.example.com \
//	go test -v -tags=e2e -count=1 ./tests/e2e/...
package e2e
