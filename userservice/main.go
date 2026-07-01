// Package main is the entry point for the user service: accounts, auth,
// subscriptions, and payments.
package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

	"github.com/yourusername/videostreamingplatform/userservice/bl"
	"github.com/yourusername/videostreamingplatform/userservice/db"
	"github.com/yourusername/videostreamingplatform/userservice/dl"
	"github.com/yourusername/videostreamingplatform/userservice/handlers"
	"github.com/yourusername/videostreamingplatform/userservice/payment"

	"github.com/yourusername/videostreamingplatform/utils/config"
	"github.com/yourusername/videostreamingplatform/utils/kafka"
	"github.com/yourusername/videostreamingplatform/utils/middleware"
	"github.com/yourusername/videostreamingplatform/utils/observability"
)

func main() {
	cfg := config.New("userservice")
	if err := cfg.Validate(); err != nil {
		panic(fmt.Sprintf("Invalid configuration: %v", err))
	}

	logger := observability.NewLogger("UserService")
	logger.Printf("Starting UserService in %s environment (payment provider: %s)", cfg.Envir, cfg.PaymentProvider)

	// JWT signing secret. A dev default keeps local runs working; production must
	// set JWT_SIGNING_SECRET.
	jwtSecret := cfg.JWTSigningSecret
	if jwtSecret == "" {
		jwtSecret = "dev-insecure-jwt-secret"
		logger.Println("WARNING: JWT_SIGNING_SECRET not set — using an insecure dev secret")
	}

	// Storage: MySQL by default, in-memory for quick local runs (USER_STORE=memory).
	var store dl.Store
	if os.Getenv("USER_STORE") == "memory" {
		logger.Println("Using in-memory store (USER_STORE=memory)")
		store = dl.NewInMemoryStore()
	} else {
		dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true",
			cfg.MySQLUser, cfg.MySQLPassword, cfg.MySQLHost, cfg.MySQLPort, cfg.MySQLDatabase)
		database, err := db.NewMySQL(dsn, cfg.MySQLMaxConn)
		if err != nil {
			logger.Fatalf("Failed to connect to MySQL: %v", err)
		}
		defer func() { _ = database.Close() }()
		logger.Println("Connected to MySQL database")
		store = dl.NewMySQLStore(database.DB())
	}

	// Payment provider selection.
	var provider payment.Provider
	var mockProvider *payment.MockProvider
	switch cfg.PaymentProvider {
	case "razorpay":
		provider = payment.NewRazorpayProvider(cfg.RazorpayKeyID, cfg.RazorpayKeySecret, cfg.RazorpayWebhookSecret, cfg.PublicBaseURL)
		logger.Println("Razorpay payment provider enabled")
	default:
		mockProvider = payment.NewMockProvider(cfg.RazorpayWebhookSecret, cfg.PublicBaseURL)
		provider = mockProvider
		logger.Println("Mock payment provider enabled")
	}

	// Optional Kafka producer for subscription lifecycle events. Skipped when
	// KAFKA_BROKERS is unset, exactly like metadata/data services.
	var billingOpts []bl.BillingOption
	if cfg.KafkaBrokers != "" {
		brokers := strings.Split(cfg.KafkaBrokers, ",")
		subProducer := kafka.NewProducer(brokers, cfg.KafkaSubscriptionTopic)
		defer func() { _ = subProducer.Close() }()
		billingOpts = append(billingOpts, bl.WithKafkaProducer(subProducer))
		logger.Printf("Kafka subscription producer enabled → %s (topic: %s)", cfg.KafkaBrokers, cfg.KafkaSubscriptionTopic)
	}

	// Services.
	authService := bl.NewAuthService(store, jwtSecret, cfg.JWTAccessTTL, cfg.JWTRefreshTTL)
	billingService := bl.NewBillingService(store, provider, cfg.PublicBaseURL, logger.Logger, billingOpts...)

	// Background reconciliation + sweeper + expiring-subscription scan.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go billingService.RunBackgroundJobs(ctx, 1*time.Minute, 30*time.Minute, 24*time.Hour, 7*24*time.Hour)

	// Handlers.
	authHandler := handlers.NewAuthHandler(authService)
	subHandler := handlers.NewSubscriptionHandler(billingService)
	webhookHandler := handlers.NewWebhookHandler(billingService)

	authMW := middleware.JWTAuth(jwtSecret)

	mux := http.NewServeMux()
	mux.HandleFunc("/health", healthHandler)

	if metricsHandler, err := observability.InitMetrics("userservice"); err != nil {
		logger.Printf("WARNING: Failed to initialize metrics: %v", err)
	} else {
		mux.Handle("/metrics", metricsHandler)
	}

	// Public auth endpoints.
	mux.HandleFunc("POST /auth/register", authHandler.Register)
	mux.HandleFunc("POST /auth/login", authHandler.Login)
	mux.HandleFunc("POST /auth/refresh", authHandler.Refresh)

	// Authenticated subscription endpoints.
	mux.Handle("POST /subscriptions", authMW(http.HandlerFunc(subHandler.Subscribe)))
	mux.Handle("GET /subscriptions/me", authMW(http.HandlerFunc(subHandler.GetCurrent)))

	// Payment webhook (signature-verified, not JWT-protected).
	mux.HandleFunc("POST /webhooks/payment", webhookHandler.PaymentWebhook)

	// Mock hosted checkout (only when the mock provider is active).
	if mockProvider != nil {
		mockCheckout := handlers.NewMockCheckoutHandler(billingService, mockProvider)
		mux.HandleFunc("GET /mock/checkout", mockCheckout.Checkout)
	}

	httpHandler := middleware.ChainMiddleware(
		mux,
		func(next http.Handler) http.Handler {
			return middleware.LoggingMiddleware(logger, next)
		},
		func(next http.Handler) http.Handler {
			return middleware.ErrorHandlingMiddleware(logger, next)
		},
	)

	addr := fmt.Sprintf(":%d", cfg.HTTPPort)
	server := &http.Server{
		Addr:           addr,
		Handler:        otelhttp.NewHandler(httpHandler, "userservice"),
		ReadTimeout:    cfg.HTTPReadTimeout,
		WriteTimeout:   cfg.HTTPWriteTimeout,
		IdleTimeout:    cfg.HTTPIdleTimeout,
		MaxHeaderBytes: 1 << 20,
	}

	logger.Printf("User Service starting on %s", addr)
	if err := server.ListenAndServe(); err != nil {
		logger.Fatalf("Server error: %v", err)
	}
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"healthy"}`))
}
