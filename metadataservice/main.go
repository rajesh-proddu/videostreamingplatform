// Package main provides the entry point for the metadata service
package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

	httpSwagger "github.com/swaggo/http-swagger/v2"

	"github.com/yourusername/videostreamingplatform/metadataservice/bl"
	"github.com/yourusername/videostreamingplatform/metadataservice/db"
	"github.com/yourusername/videostreamingplatform/metadataservice/dl"
	"github.com/yourusername/videostreamingplatform/metadataservice/handlers"

	"github.com/yourusername/videostreamingplatform/utils/cache"
	"github.com/yourusername/videostreamingplatform/utils/config"
	"github.com/yourusername/videostreamingplatform/utils/kafka"
	"github.com/yourusername/videostreamingplatform/utils/middleware"
	"github.com/yourusername/videostreamingplatform/utils/observability"
	"github.com/yourusername/videostreamingplatform/utils/recommendations"
)

func main() {
	// Initialize configuration
	cfg := config.New("metadataservice")
	if err := cfg.Validate(); err != nil {
		panic(fmt.Sprintf("Invalid configuration: %v", err))
	}

	// Initialize logger
	logger := observability.NewLogger("MetadataService")

	logger.Printf("Starting MetadataService in %s environment", cfg.Envir)
	logger.Printf("Config - HTTP: :%d, MySQL: %s:%s", cfg.HTTPPort, cfg.MySQLHost, cfg.MySQLPort)

	// Initialize OpenTelemetry tracing
	if otelEndpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"); otelEndpoint != "" {
		shutdown, err := observability.InitTracer(context.Background(), "metadataservice", otelEndpoint)
		if err != nil {
			logger.Printf("WARNING: Failed to initialize tracing: %v", err)
		} else {
			defer func() { _ = shutdown(context.Background()) }()
			logger.Printf("OpenTelemetry tracing enabled → %s", otelEndpoint)
		}
	}

	// Initialize database
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true",
		cfg.MySQLUser, cfg.MySQLPassword, cfg.MySQLHost, cfg.MySQLPort, cfg.MySQLDatabase)

	database, err := db.NewMySQL(dsn)
	if err != nil {
		logger.Fatalf("Failed to connect to MySQL: %v", err)
	}
	defer func() { _ = database.Close() }()

	// Check database connection
	if err := database.Ping(context.Background()); err != nil {
		logger.Fatalf("Failed to ping MySQL: %v", err)
	}

	logger.Println("Connected to MySQL database")

	// Initialize service layers
	repo := dl.NewVideoRepository(database)

	// Initialize Redis cache (best-effort: nil if not configured)
	redisCache := cache.New(cfg.RedisAddr, cfg.RedisPassword, cfg.RedisDB)
	if redisCache != nil {
		if err := redisCache.Ping(context.Background()); err != nil {
			logger.Printf("WARNING: Redis not reachable: %v (caching disabled)", err)
			redisCache = nil
		} else {
			defer func() { _ = redisCache.Close() }()
			logger.Printf("Redis cache enabled → %s", cfg.RedisAddr)
		}
	}

	var serviceOpts []bl.VideoServiceOption

	// Add cache option if Redis is available
	if redisCache != nil {
		serviceOpts = append(serviceOpts, bl.WithCache(
			redisCache,
			time.Duration(cfg.CacheTTLGetVideo)*time.Second,
			time.Duration(cfg.CacheTTLListVideos)*time.Second,
		))
	}

	if cfg.KafkaBrokers != "" {
		brokers := strings.Split(cfg.KafkaBrokers, ",")
		videoProducer := kafka.NewProducer(brokers, cfg.KafkaVideoTopic)
		defer func() { _ = videoProducer.Close() }()
		serviceOpts = append(serviceOpts, bl.WithKafkaProducer(videoProducer, logger.Logger))
		logger.Printf("Kafka video producer enabled → %s (topic: %s)", cfg.KafkaBrokers, cfg.KafkaVideoTopic)
	}

	videoService := bl.NewVideoService(repo, serviceOpts...)

	// Initialize HTTP handlers
	handler := handlers.NewVideoHandler(videoService)
	recoClient := recommendations.NewClient(cfg.RecommendationServiceURL)
	recoHandler := handlers.NewRecommendationHandler(recoClient)

	// Set up routes
	mux := http.NewServeMux()
	mux.HandleFunc("/health", healthHandler)

	// Prometheus metrics endpoint
	metricsHandler, err := observability.InitMetrics("metadataservice")
	if err != nil {
		logger.Printf("WARNING: Failed to initialize metrics: %v", err)
	} else {
		mux.Handle("/metrics", metricsHandler)
	}

	mux.HandleFunc("POST /videos", handler.CreateVideo)
	mux.HandleFunc("GET /videos/{id}", handler.GetVideo)
	mux.HandleFunc("PUT /videos/{id}", handler.UpdateVideo)
	mux.HandleFunc("DELETE /videos/{id}", handler.DeleteVideo)
	mux.HandleFunc("GET /videos", handler.ListVideos)
	mux.HandleFunc("GET /recommendations", recoHandler.GetRecommendations)
	mux.Handle("/swagger/", httpSwagger.WrapHandler)

	// Initialize rate limiter (uses Redis if available, falls back to in-memory per-pod)
	var redisClient = redisCache.RedisClient()
	rateLimiter := middleware.NewRateLimiter(redisClient, cfg.RateLimitPerMin, time.Minute, cfg.RateLimitBurst)
	if redisClient != nil {
		logger.Printf("Rate limiter enabled (distributed/Redis): %d req/min", cfg.RateLimitPerMin)
	} else {
		logger.Printf("Rate limiter enabled (in-memory fallback): %d req/min, burst %d", cfg.RateLimitPerMin, cfg.RateLimitBurst)
	}

	// Apply middleware
	httpHandler := middleware.ChainMiddleware(
		mux,
		rateLimiter.Middleware,
		func(next http.Handler) http.Handler {
			return middleware.LoggingMiddleware(logger, next)
		},
		func(next http.Handler) http.Handler {
			return middleware.ErrorHandlingMiddleware(logger, next)
		},
	)

	// Start HTTP/2 server
	addr := fmt.Sprintf(":%d", cfg.HTTPPort)
	server := &http.Server{
		Addr:           addr,
		Handler:        otelhttp.NewHandler(httpHandler, "metadataservice"),
		ReadTimeout:    cfg.HTTPReadTimeout,
		WriteTimeout:   cfg.HTTPWriteTimeout,
		IdleTimeout:    cfg.HTTPIdleTimeout,
		MaxHeaderBytes: 1 << 20, // 1MB
	}

	logger.Printf("Metadata Service starting on %s", addr)
	if err := server.ListenAndServe(); err != nil {
		logger.Fatalf("Server error: %v", err)
	}
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"healthy"}`))
}
