package e2e

import (
	"os"
	"testing"
)

// Config holds the target environment endpoints.
type Config struct {
	MetadataURL string // e.g. http://localhost:8080
	DataURL     string // e.g. http://localhost:8081
	UserURL     string // e.g. http://localhost:8082 (auth + subscriptions)
}

// LoadConfig reads endpoint URLs from environment variables,
// falling back to local docker-compose defaults.
func LoadConfig(t *testing.T) Config {
	t.Helper()

	cfg := Config{
		MetadataURL: envOr("E2E_METADATA_URL", "http://localhost:8080"),
		DataURL:     envOr("E2E_DATA_URL", "http://localhost:8081"),
		UserURL:     envOr("E2E_USER_URL", "http://localhost:8082"),
	}

	t.Logf("Target: metadata=%s  data=%s  user=%s", cfg.MetadataURL, cfg.DataURL, cfg.UserURL)
	return cfg
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
