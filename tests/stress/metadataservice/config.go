package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strconv"
	"time"
)

const (
	defaultBaseURL        = "http://localhost:8080"
	defaultTargetQPS      = 10000
	defaultDuration       = time.Minute
	defaultWorkers        = 256
	defaultSeedVideos     = 2000
	defaultSeedWorkers    = 32
	defaultListLimit      = 20
	defaultRequestTimeout = 5 * time.Second
)

type operationProfile struct {
	ReadByIDPct int
	ListPct     int
	CreatePct   int
	UpdatePct   int
}

func (p operationProfile) validate() error {
	if p.ReadByIDPct < 0 || p.ListPct < 0 || p.CreatePct < 0 || p.UpdatePct < 0 {
		return errors.New("operation percentages must be non-negative")
	}

	if p.ReadByIDPct+p.ListPct+p.CreatePct+p.UpdatePct != 100 {
		return fmt.Errorf(
			"operation percentages must sum to 100, got %d",
			p.ReadByIDPct+p.ListPct+p.CreatePct+p.UpdatePct,
		)
	}

	return nil
}

type config struct {
	BaseURL        string
	TargetQPS      int
	Duration       time.Duration
	Workers        int
	SeedVideos     int
	SeedWorkers    int
	ListLimit      int
	RequestTimeout time.Duration
	Profile        operationProfile
	RunTag         string
}

func loadConfig(args []string) (config, error) {
	fs := flag.NewFlagSet("metadata-stress", flag.ContinueOnError)

	cfg := config{}

	fs.StringVar(&cfg.BaseURL, "base-url", envOrString("METADATA_STRESS_BASE_URL", defaultBaseURL), "metadata service base URL")
	fs.IntVar(&cfg.TargetQPS, "target-qps", envOrInt("METADATA_STRESS_TARGET_QPS", defaultTargetQPS), "target total QPS across all operations")
	fs.DurationVar(&cfg.Duration, "duration", envOrDuration("METADATA_STRESS_DURATION", defaultDuration), "load generation duration")
	fs.IntVar(&cfg.Workers, "workers", envOrInt("METADATA_STRESS_WORKERS", defaultWorkers), "number of concurrent workers")
	fs.IntVar(&cfg.SeedVideos, "seed-videos", envOrInt("METADATA_STRESS_SEED_VIDEOS", defaultSeedVideos), "number of metadata records to pre-create before the run")
	fs.IntVar(&cfg.SeedWorkers, "seed-workers", envOrInt("METADATA_STRESS_SEED_WORKERS", defaultSeedWorkers), "number of concurrent workers for seeding")
	fs.IntVar(&cfg.ListLimit, "list-limit", envOrInt("METADATA_STRESS_LIST_LIMIT", defaultListLimit), "limit used for GET /videos requests")
	fs.DurationVar(&cfg.RequestTimeout, "request-timeout", envOrDuration("METADATA_STRESS_REQUEST_TIMEOUT", defaultRequestTimeout), "per-request timeout")
	fs.IntVar(&cfg.Profile.ReadByIDPct, "read-by-id-pct", envOrInt("METADATA_STRESS_READ_BY_ID_PCT", 60), "percentage of GET /videos/{id}")
	fs.IntVar(&cfg.Profile.ListPct, "list-pct", envOrInt("METADATA_STRESS_LIST_PCT", 20), "percentage of GET /videos")
	fs.IntVar(&cfg.Profile.CreatePct, "create-pct", envOrInt("METADATA_STRESS_CREATE_PCT", 15), "percentage of POST /videos")
	fs.IntVar(&cfg.Profile.UpdatePct, "update-pct", envOrInt("METADATA_STRESS_UPDATE_PCT", 5), "percentage of PUT /videos/{id}")
	fs.StringVar(&cfg.RunTag, "run-tag", envOrString("METADATA_STRESS_RUN_TAG", time.Now().UTC().Format("20060102-150405")), "tag added to created metadata titles for easier identification")

	if err := fs.Parse(args); err != nil {
		return config{}, err
	}

	if cfg.BaseURL == "" {
		return config{}, errors.New("base-url must not be empty")
	}
	if cfg.TargetQPS <= 0 {
		return config{}, errors.New("target-qps must be > 0")
	}
	if cfg.Duration <= 0 {
		return config{}, errors.New("duration must be > 0")
	}
	if cfg.Workers <= 0 {
		return config{}, errors.New("workers must be > 0")
	}
	if cfg.SeedVideos < 0 {
		return config{}, errors.New("seed-videos must be >= 0")
	}
	if cfg.SeedWorkers <= 0 {
		return config{}, errors.New("seed-workers must be > 0")
	}
	if cfg.ListLimit <= 0 {
		return config{}, errors.New("list-limit must be > 0")
	}
	if cfg.RequestTimeout <= 0 {
		return config{}, errors.New("request-timeout must be > 0")
	}
	if err := cfg.Profile.validate(); err != nil {
		return config{}, err
	}

	return cfg, nil
}

func envOrString(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func envOrInt(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}

	return parsed
}

func envOrDuration(key string, fallback time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	parsed, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}

	return parsed
}
