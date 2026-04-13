package config

import (
	"testing"
	"time"
)

func TestNew_Defaults(t *testing.T) {
	cfg := New("test-svc")

	if cfg.ServiceName != "test-svc" {
		t.Errorf("ServiceName = %q, want %q", cfg.ServiceName, "test-svc")
	}
	if cfg.Envir != "dev" {
		t.Errorf("Envir = %q, want %q", cfg.Envir, "dev")
	}
	if cfg.LogLevel != "info" {
		t.Errorf("LogLevel = %q, want %q", cfg.LogLevel, "info")
	}
	if cfg.HTTPPort != 8080 {
		t.Errorf("HTTPPort = %d, want 8080", cfg.HTTPPort)
	}
	if cfg.HTTPReadTimeout != 10*time.Second {
		t.Errorf("HTTPReadTimeout = %v, want %v", cfg.HTTPReadTimeout, 10*time.Second)
	}
	if cfg.HTTPWriteTimeout != 30*time.Second {
		t.Errorf("HTTPWriteTimeout = %v, want %v", cfg.HTTPWriteTimeout, 30*time.Second)
	}
	if cfg.HTTPIdleTimeout != 120*time.Second {
		t.Errorf("HTTPIdleTimeout = %v, want %v", cfg.HTTPIdleTimeout, 120*time.Second)
	}
	if cfg.GRPCPort != 50051 {
		t.Errorf("GRPCPort = %d, want 50051", cfg.GRPCPort)
	}
	if cfg.MySQLHost != "localhost" {
		t.Errorf("MySQLHost = %q, want %q", cfg.MySQLHost, "localhost")
	}
	if cfg.MySQLPort != "3306" {
		t.Errorf("MySQLPort = %q, want %q", cfg.MySQLPort, "3306")
	}
	if cfg.MySQLUser != "videouser" {
		t.Errorf("MySQLUser = %q, want %q", cfg.MySQLUser, "videouser")
	}
	if cfg.MySQLPassword != "videopass" {
		t.Errorf("MySQLPassword = %q, want %q", cfg.MySQLPassword, "videopass")
	}
	if cfg.MySQLDatabase != "videoplatform" {
		t.Errorf("MySQLDatabase = %q, want %q", cfg.MySQLDatabase, "videoplatform")
	}
	if cfg.MySQLMaxConn != 25 {
		t.Errorf("MySQLMaxConn = %d, want 25", cfg.MySQLMaxConn)
	}
	if cfg.S3Region != "us-east-1" {
		t.Errorf("S3Region = %q, want %q", cfg.S3Region, "us-east-1")
	}
	if cfg.S3Bucket != "video-platform-storage" {
		t.Errorf("S3Bucket = %q, want %q", cfg.S3Bucket, "video-platform-storage")
	}
	if cfg.OTelEnabled {
		t.Error("OTelEnabled should default to false")
	}
	if cfg.OTelJaegerURL != "http://localhost:14268/api/traces" {
		t.Errorf("OTelJaegerURL = %q, want default", cfg.OTelJaegerURL)
	}
	if cfg.DebugMode {
		t.Error("DebugMode should default to false")
	}
	if cfg.KafkaBrokers != "localhost:9092" {
		t.Errorf("KafkaBrokers = %q, want %q", cfg.KafkaBrokers, "localhost:9092")
	}
	if cfg.KafkaVideoTopic != "video-events" {
		t.Errorf("KafkaVideoTopic = %q, want %q", cfg.KafkaVideoTopic, "video-events")
	}
	if cfg.KafkaWatchTopic != "watch-events" {
		t.Errorf("KafkaWatchTopic = %q, want %q", cfg.KafkaWatchTopic, "watch-events")
	}
	if cfg.UploadStore != "mysql" {
		t.Errorf("UploadStore = %q, want %q", cfg.UploadStore, "mysql")
	}
	if cfg.RecommendationServiceURL != "" {
		t.Errorf("RecommendationServiceURL = %q, want empty", cfg.RecommendationServiceURL)
	}
}

func TestNew_EnvOverrides(t *testing.T) {
	t.Setenv("HTTP_PORT", "9090")
	t.Setenv("GRPC_PORT", "50052")
	t.Setenv("MYSQL_HOST", "db.example.com")
	t.Setenv("MYSQL_PORT", "3307")
	t.Setenv("ENVIRONMENT", "prod")
	t.Setenv("LOG_LEVEL", "debug")
	t.Setenv("OTEL_ENABLED", "true")
	t.Setenv("DEBUG_MODE", "1")
	t.Setenv("HTTP_READ_TIMEOUT", "5s")
	t.Setenv("KAFKA_BROKERS", "kafka1:9092,kafka2:9092")
	t.Setenv("UPLOAD_STORE", "memory")
	t.Setenv("RECOMMENDATION_SERVICE_URL", "http://rec:8000")

	cfg := New("override-svc")

	if cfg.HTTPPort != 9090 {
		t.Errorf("HTTPPort = %d, want 9090", cfg.HTTPPort)
	}
	if cfg.GRPCPort != 50052 {
		t.Errorf("GRPCPort = %d, want 50052", cfg.GRPCPort)
	}
	if cfg.MySQLHost != "db.example.com" {
		t.Errorf("MySQLHost = %q, want %q", cfg.MySQLHost, "db.example.com")
	}
	if cfg.MySQLPort != "3307" {
		t.Errorf("MySQLPort = %q, want %q", cfg.MySQLPort, "3307")
	}
	if cfg.Envir != "prod" {
		t.Errorf("Envir = %q, want %q", cfg.Envir, "prod")
	}
	if cfg.LogLevel != "debug" {
		t.Errorf("LogLevel = %q, want %q", cfg.LogLevel, "debug")
	}
	if !cfg.OTelEnabled {
		t.Error("OTelEnabled should be true")
	}
	if !cfg.DebugMode {
		t.Error("DebugMode should be true")
	}
	if cfg.HTTPReadTimeout != 5*time.Second {
		t.Errorf("HTTPReadTimeout = %v, want 5s", cfg.HTTPReadTimeout)
	}
	if cfg.KafkaBrokers != "kafka1:9092,kafka2:9092" {
		t.Errorf("KafkaBrokers = %q, want %q", cfg.KafkaBrokers, "kafka1:9092,kafka2:9092")
	}
	if cfg.UploadStore != "memory" {
		t.Errorf("UploadStore = %q, want %q", cfg.UploadStore, "memory")
	}
	if cfg.RecommendationServiceURL != "http://rec:8000" {
		t.Errorf("RecommendationServiceURL = %q, want %q", cfg.RecommendationServiceURL, "http://rec:8000")
	}
}

func TestValidate_Valid(t *testing.T) {
	cfg := &Config{
		ServiceName: "valid-svc",
		HTTPPort:    8080,
		GRPCPort:    50051,
	}
	if err := cfg.Validate(); err != nil {
		t.Errorf("Validate() returned error for valid config: %v", err)
	}
}

func TestValidate_EmptyServiceName(t *testing.T) {
	cfg := &Config{
		ServiceName: "",
		HTTPPort:    8080,
		GRPCPort:    50051,
	}
	if err := cfg.Validate(); err == nil {
		t.Error("Validate() should return error for empty ServiceName")
	}
}

func TestValidate_InvalidHTTPPort(t *testing.T) {
	tests := []struct {
		name string
		port int
	}{
		{"zero port", 0},
		{"too high", 99999},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				ServiceName: "test",
				HTTPPort:    tt.port,
				GRPCPort:    50051,
			}
			if err := cfg.Validate(); err == nil {
				t.Errorf("Validate() should return error for HTTP port %d", tt.port)
			}
		})
	}
}

func TestValidate_InvalidGRPCPort(t *testing.T) {
	cfg := &Config{
		ServiceName: "test",
		HTTPPort:    8080,
		GRPCPort:    -1,
	}
	if err := cfg.Validate(); err == nil {
		t.Error("Validate() should return error for gRPC port -1")
	}
}
