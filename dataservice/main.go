// Package main provides the entry point for the data service
package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

	httpSwagger "github.com/swaggo/http-swagger/v2"

	"github.com/yourusername/videostreamingplatform/dataservice/bl"
	"github.com/yourusername/videostreamingplatform/dataservice/db"
	"github.com/yourusername/videostreamingplatform/dataservice/dl"
	"github.com/yourusername/videostreamingplatform/dataservice/handlers"
	"github.com/yourusername/videostreamingplatform/dataservice/server"

	"github.com/yourusername/videostreamingplatform/dataservice/pb"
	"github.com/yourusername/videostreamingplatform/dataservice/storage"

	"github.com/yourusername/videostreamingplatform/utils/config"
	"github.com/yourusername/videostreamingplatform/utils/kafka"
	"github.com/yourusername/videostreamingplatform/utils/middleware"
	"github.com/yourusername/videostreamingplatform/utils/observability"

	"google.golang.org/grpc"
)

func main() {
	// Initialize configuration
	cfg := config.New("dataservice")
	if err := cfg.Validate(); err != nil {
		panic(fmt.Sprintf("Invalid configuration: %v", err))
	}

	// Initialize logger
	logger := observability.NewLogger("DataService")

	logger.Printf("Starting DataService in %s environment", cfg.Envir)
	logger.Printf("Config - HTTP: :%d, gRPC: :%d", cfg.HTTPPort, cfg.GRPCPort)

	// Initialize OpenTelemetry tracing
	if otelEndpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"); otelEndpoint != "" {
		shutdown, err := observability.InitTracer(context.Background(), "dataservice", otelEndpoint)
		if err != nil {
			logger.Printf("WARNING: Failed to initialize tracing: %v", err)
		} else {
			defer func() { _ = shutdown(context.Background()) }()
			logger.Printf("OpenTelemetry tracing enabled → %s", otelEndpoint)
		}
	}

	// Initialize S3 client
	s3Client, err := storage.NewS3Client(context.Background())
	if err != nil {
		logger.Fatalf("Failed to initialize S3 client: %v", err)
	}

	// Initialize repository (MySQL or in-memory based on config)
	var uploadRepo dl.UploadRepository
	if cfg.UploadStore == "memory" {
		logger.Println("Using in-memory upload repository")
		uploadRepo = dl.NewInMemoryUploadRepository()
	} else {
		dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true",
			cfg.MySQLUser, cfg.MySQLPassword, cfg.MySQLHost, cfg.MySQLPort, cfg.MySQLDatabase)
		database, err := db.NewMySQL(dsn, cfg.MySQLMaxConn)
		if err != nil {
			logger.Fatalf("Failed to connect to MySQL: %v", err)
		}
		defer func() { _ = database.Close() }()
		logger.Println("Connected to MySQL database")
		uploadRepo = dl.NewMySQLUploadRepository(database.DB())
	}

	// Initialize Kafka producer for watch events (best-effort)
	var watchProducer kafka.Producer
	if cfg.KafkaBrokers != "" {
		brokers := strings.Split(cfg.KafkaBrokers, ",")
		watchProducer = kafka.NewProducer(brokers, cfg.KafkaWatchTopic)
		defer func() { _ = watchProducer.Close() }()
		logger.Printf("Kafka watch producer enabled → %s (topic: %s)", cfg.KafkaBrokers, cfg.KafkaWatchTopic)
	}

	// Initialize service
	uploadService := bl.NewUploadService(uploadRepo, logger.Logger)

	// Initialize handlers
	uploadHandler := handlers.NewUploadHandler(uploadService, s3Client, watchProducer, logger)

	// Set up HTTP routes for streaming
	mux := http.NewServeMux()
	mux.HandleFunc("/health", healthHandler)

	// Prometheus metrics endpoint
	metricsHandler, err := observability.InitMetrics("dataservice")
	if err != nil {
		logger.Printf("WARNING: Failed to initialize metrics: %v", err)
	} else {
		mux.Handle("/metrics", metricsHandler)
	}

	mux.HandleFunc("POST /uploads/initiate", uploadHandler.InitiateUpload)
	mux.HandleFunc("POST /uploads/{uploadId}/chunks", uploadHandler.Upload)
	mux.HandleFunc("GET /uploads/{uploadId}/progress", uploadHandler.GetUploadProgress)
	mux.HandleFunc("POST /uploads/{uploadId}/complete", uploadHandler.CompleteUpload)
	mux.HandleFunc("GET /videos/{id}/download", uploadHandler.Download)
	mux.Handle("/swagger/", httpSwagger.WrapHandler)

	// Apply middleware to HTTP server
	httpHandler := middleware.ChainMiddleware(
		mux,
		func(next http.Handler) http.Handler {
			return middleware.LoggingMiddleware(logger, next)
		},
		func(next http.Handler) http.Handler {
			return middleware.ErrorHandlingMiddleware(logger, next)
		},
	)

	// Start HTTP/2 server in goroutine
	httpAddr := fmt.Sprintf(":%d", cfg.HTTPPort)
	go func() {
		httpServer := &http.Server{
			Addr:           httpAddr,
			Handler:        otelhttp.NewHandler(httpHandler, "dataservice"),
			ReadTimeout:    cfg.HTTPReadTimeout,
			WriteTimeout:   cfg.HTTPWriteTimeout,
			IdleTimeout:    cfg.HTTPIdleTimeout,
			MaxHeaderBytes: 1 << 20, // 1MB
		}
		logger.Printf("Data Service HTTP/2 server starting on %s", httpAddr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("HTTP server error: %v", err)
		}
	}()

	// Initialize gRPC server
	grpcAddr := fmt.Sprintf(":%d", cfg.GRPCPort)
	listener, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		logger.Fatalf("Failed to listen on gRPC port %d: %v", cfg.GRPCPort, err)
	}
	defer func() { _ = listener.Close() }()

	grpcServer := grpc.NewServer()
	dataServiceServer := server.NewDataServiceServer(uploadService, logger.Logger)
	pb.RegisterDataServiceServer(grpcServer, dataServiceServer)

	logger.Printf("Data Service gRPC server listening on port %d", cfg.GRPCPort)

	if err := grpcServer.Serve(listener); err != nil {
		logger.Fatalf("Failed to serve gRPC server: %v", err)
	}
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"healthy"}`))
}
