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
	GracefulShutdown time.Duration

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
	OTelEnabled        bool
	OTelTracesEnabled  bool
	OTelMetricsEnabled bool
	OTelJaegerURL      string

	// Feature flags
	DebugMode bool

	// Kafka configuration
	KafkaBrokers    string
	KafkaVideoTopic string
	KafkaWatchTopic string

	// Elasticsearch configuration
	ElasticsearchURL string

	// Upload store selector (memory or mysql)
	UploadStore string

	// Recommendation service URL (empty = disabled)
	RecommendationServiceURL string
}

// New creates a new Config instance from environment variables
func New(serviceName string) *Config {
	return &Config{
		ServiceName:        serviceName,
		Envir:              getEnvOrDefault("ENVIRONMENT", "dev"),
		LogLevel:           getEnvOrDefault("LOG_LEVEL", "info"),
		HTTPPort:           getEnvAsInt("HTTP_PORT", 8080),
		HTTPReadTimeout:    getEnvAsDuration("HTTP_READ_TIMEOUT", 10*time.Second),
		HTTPWriteTimeout:   getEnvAsDuration("HTTP_WRITE_TIMEOUT", 30*time.Second),
		HTTPIdleTimeout:    getEnvAsDuration("HTTP_IDLE_TIMEOUT", 120*time.Second),
		GRPCPort:           getEnvAsInt("GRPC_PORT", 50051),
		GracefulShutdown:   getEnvAsDuration("GRACEFUL_SHUTDOWN_TIMEOUT", 30*time.Second),
		MySQLHost:          getEnvOrDefault("MYSQL_HOST", "localhost"),
		MySQLPort:          getEnvOrDefault("MYSQL_PORT", "3306"),
		MySQLUser:          getEnvOrDefault("MYSQL_USER", "videouser"),
		MySQLPassword:      getEnvOrDefault("MYSQL_PASSWORD", "videopass"),
		MySQLDatabase:      getEnvOrDefault("MYSQL_DATABASE", "videoplatform"),
		MySQLMaxConn:       getEnvAsInt("MYSQL_MAX_CONN", 25),
		S3Region:           getEnvOrDefault("S3_REGION", "us-east-1"),
		S3Bucket:           getEnvOrDefault("S3_BUCKET", "video-platform-storage"),
		OTelEnabled:        getEnvAsBool("OTEL_ENABLED", false),
		OTelTracesEnabled:  getEnvAsBool("OTEL_TRACES_ENABLED", false),
		OTelMetricsEnabled: getEnvAsBool("OTEL_METRICS_ENABLED", false),
		OTelJaegerURL:      getEnvOrDefault("OTEL_JAEGER_URL", "http://localhost:14268/api/traces"),
		DebugMode:          getEnvAsBool("DEBUG_MODE", false),
		KafkaBrokers:       getEnvOrDefault("KAFKA_BROKERS", "localhost:9092"),
		KafkaVideoTopic:    getEnvOrDefault("KAFKA_VIDEO_TOPIC", "video-events"),
		KafkaWatchTopic:    getEnvOrDefault("KAFKA_WATCH_TOPIC", "watch-events"),
		ElasticsearchURL:   getEnvOrDefault("ELASTICSEARCH_URL", "http://localhost:9200"),
		UploadStore:              getEnvOrDefault("UPLOAD_STORE", "mysql"),
		RecommendationServiceURL: getEnvOrDefault("RECOMMENDATION_SERVICE_URL", ""),
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
