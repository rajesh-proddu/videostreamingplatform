// Package config provides centralized configuration management following 12-factor app principles
package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds all application configuration from environment variables
type Config struct {
	// Service configuration
	ServiceName string
	Envir       string // dev, staging, prod
	LogLevel    string

	// Server configuration
	HTTPPort         int
	HTTPReadTimeout  time.Duration
	HTTPWriteTimeout time.Duration
	HTTPIdleTimeout  time.Duration
	GRPCPort         int

	// Database configuration
	MySQLHost     string
	MySQLPort     string
	MySQLUser     string
	MySQLPassword string
	MySQLDatabase string
	MySQLMaxConn  int

	// S3 configuration
	S3Region string
	S3Bucket string

	// OpenTelemetry configuration
	OTelEnabled   bool
	OTelJaegerURL string

	// Feature flags
	DebugMode bool

	// Kafka configuration
	KafkaBrokers    string
	KafkaVideoTopic string
	KafkaWatchTopic string

	// Upload store selector (memory or mysql)
	UploadStore string

	// Recommendation service URL (empty = disabled)
	RecommendationServiceURL string

	// Redis cache configuration
	RedisAddr     string
	RedisPassword string
	RedisDB       int

	// Cache TTL configuration (in seconds)
	CacheTTLGetVideo   int // single video cache TTL
	CacheTTLListVideos int // list videos cache TTL

	// Rate limiting configuration
	RateLimitPerMin int // requests per minute per IP
	RateLimitBurst  int // burst capacity

	// CDN configuration
	CDNDistributionID string // CloudFront distribution ID (empty = CDN invalidation disabled)
}

// New creates a new Config instance from environment variables
func New(serviceName string) *Config {
	return &Config{
		ServiceName:              serviceName,
		Envir:                    getEnvOrDefault("ENVIRONMENT", "dev"),
		LogLevel:                 getEnvOrDefault("LOG_LEVEL", "info"),
		HTTPPort:                 getEnvAsInt("HTTP_PORT", 8080),
		HTTPReadTimeout:          getEnvAsDuration("HTTP_READ_TIMEOUT", 10*time.Second),
		HTTPWriteTimeout:         getEnvAsDuration("HTTP_WRITE_TIMEOUT", 30*time.Second),
		HTTPIdleTimeout:          getEnvAsDuration("HTTP_IDLE_TIMEOUT", 120*time.Second),
		GRPCPort:                 getEnvAsInt("GRPC_PORT", 50051),
		MySQLHost:                getEnvOrDefault("MYSQL_HOST", "localhost"),
		MySQLPort:                getEnvOrDefault("MYSQL_PORT", "3306"),
		MySQLUser:                getEnvOrDefault("MYSQL_USER", "videouser"),
		MySQLPassword:            getEnvOrDefault("MYSQL_PASSWORD", "videopass"),
		MySQLDatabase:            getEnvOrDefault("MYSQL_DATABASE", "videoplatform"),
		MySQLMaxConn:             getEnvAsInt("MYSQL_MAX_CONN", 50),
		S3Region:                 getEnvOrDefault("S3_REGION", "us-east-1"),
		S3Bucket:                 getEnvOrDefault("S3_BUCKET", "video-platform-storage"),
		OTelEnabled:              getEnvAsBool("OTEL_ENABLED", false),
		OTelJaegerURL:            getEnvOrDefault("OTEL_JAEGER_URL", "http://localhost:14268/api/traces"),
		DebugMode:                getEnvAsBool("DEBUG_MODE", false),
		KafkaBrokers:             getEnvOrDefault("KAFKA_BROKERS", "localhost:9092"),
		KafkaVideoTopic:          getEnvOrDefault("KAFKA_VIDEO_TOPIC", "video-events"),
		KafkaWatchTopic:          getEnvOrDefault("KAFKA_WATCH_TOPIC", "watch-events"),
		UploadStore:              getEnvOrDefault("UPLOAD_STORE", "mysql"),
		RecommendationServiceURL: getEnvOrDefault("RECOMMENDATION_SERVICE_URL", ""),
		RedisAddr:                getEnvOrDefault("REDIS_ADDR", ""),
		RedisPassword:            getEnvOrDefault("REDIS_PASSWORD", ""),
		RedisDB:                  getEnvAsInt("REDIS_DB", 0),
		CacheTTLGetVideo:         getEnvAsInt("CACHE_TTL_GET_VIDEO", 300),  // 5 minutes default
		CacheTTLListVideos:       getEnvAsInt("CACHE_TTL_LIST_VIDEOS", 60), // 1 minute default
		RateLimitPerMin:          getEnvAsInt("RATE_LIMIT_PER_MIN", 60),    // 60 req/min default
		RateLimitBurst:           getEnvAsInt("RATE_LIMIT_BURST", 100),     // burst of 100 default
		CDNDistributionID:        getEnvOrDefault("CDN_DISTRIBUTION_ID", ""),
	}
}

// Validate checks if configuration is valid
func (c *Config) Validate() error {
	if c.ServiceName == "" {
		return fmt.Errorf("service name is required")
	}
	if c.HTTPPort < 1 || c.HTTPPort > 65535 {
		return fmt.Errorf("invalid HTTP port: %d", c.HTTPPort)
	}
	if c.GRPCPort < 1 || c.GRPCPort > 65535 {
		return fmt.Errorf("invalid gRPC port: %d", c.GRPCPort)
	}
	return nil
}

// Helper functions
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	valStr := os.Getenv(key)
	if val, err := strconv.Atoi(valStr); err == nil {
		return val
	}
	return defaultValue
}

func getEnvAsBool(key string, defaultValue bool) bool {
	val := os.Getenv(key)
	if val != "" {
		return val == "true" || val == "1" || val == "yes"
	}
	return defaultValue
}

func getEnvAsDuration(key string, defaultValue time.Duration) time.Duration {
	valStr := os.Getenv(key)
	if val, err := time.ParseDuration(valStr); err == nil {
		return val
	}
	return defaultValue
}
