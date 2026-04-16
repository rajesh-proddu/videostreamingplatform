// Package main runs a Kafka consumer that invalidates CDN cache on video deletion.
//
// Event flow:
//
//	metadataservice → DELETE /videos/{id}
//	  → publishes VIDEO_DELETED to Kafka (video-events topic)
//	  → this worker consumes the event
//	  → calls CloudFront CreateInvalidation for /videos/{id}
//
// In local (Kind) deployments, this worker calls the nginx cache purge endpoint instead.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	kafkago "github.com/segmentio/kafka-go"

	"github.com/yourusername/videostreamingplatform/utils/cdn"
	"github.com/yourusername/videostreamingplatform/utils/config"
	"github.com/yourusername/videostreamingplatform/utils/events"
	"github.com/yourusername/videostreamingplatform/utils/observability"
)

func main() {
	cfg := config.New("cdn-invalidator")
	logger := observability.NewLogger("CDNInvalidator")

	// Initialize CDN invalidator
	invalidator, err := cdn.NewCloudFrontInvalidator(context.Background(), cfg.CDNDistributionID, logger.Logger)
	if err != nil {
		logger.Fatalf("Failed to initialize CDN invalidator: %v", err)
	}
	logger.Printf("CDN invalidator initialized (distribution: %s)", cfg.CDNDistributionID)

	// Initialize Kafka reader
	if cfg.KafkaBrokers == "" {
		logger.Fatalf("KAFKA_BROKERS is required")
	}
	brokers := strings.Split(cfg.KafkaBrokers, ",")
	reader := kafkago.NewReader(kafkago.ReaderConfig{
		Brokers:  brokers,
		Topic:    cfg.KafkaVideoTopic,
		GroupID:  "cdn-invalidator",
		MinBytes: 1,
		MaxBytes: 10e6,
	})
	defer func() { _ = reader.Close() }()

	logger.Printf("Listening on Kafka topic %s (brokers: %s)", cfg.KafkaVideoTopic, cfg.KafkaBrokers)

	// Graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		logger.Println("Shutting down...")
		cancel()
	}()

	// Consume loop
	for {
		msg, err := reader.ReadMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				break // shutdown
			}
			logger.Printf("Error reading message: %v", err)
			continue
		}

		processMessage(ctx, msg, invalidator, logger.Logger)
	}

	logger.Println("CDN invalidator stopped")
}

// videoEvent mirrors the event structure for deserialization.
type videoEvent struct {
	Version   string          `json:"version"`
	Type      string          `json:"type"`
	Payload   json.RawMessage `json:"payload"`
}

// deletePayload is the payload for VIDEO_DELETED events.
type deletePayload struct {
	ID string `json:"id"`
}

func processMessage(ctx context.Context, msg kafkago.Message, invalidator cdn.Invalidator, logger *log.Logger) {
	var evt videoEvent
	if err := json.Unmarshal(msg.Value, &evt); err != nil {
		logger.Printf("WARN: failed to unmarshal event: %v", err)
		return
	}

	// Only process delete events — other events don't need CDN invalidation
	if evt.Type != events.VideoDeleted {
		return
	}

	var payload deletePayload
	if err := json.Unmarshal(evt.Payload, &payload); err != nil {
		logger.Printf("WARN: failed to unmarshal delete payload: %v", err)
		return
	}

	if payload.ID == "" {
		logger.Printf("WARN: VIDEO_DELETED event with empty ID, skipping")
		return
	}

	if err := invalidator.InvalidateVideo(ctx, payload.ID); err != nil {
		// Log but don't crash — Kafka consumer group will retry on next rebalance
		// if we don't commit, but for best-effort CDN invalidation this is acceptable
		logger.Printf("ERROR: CDN invalidation failed for video %s: %v", payload.ID, err)
		return
	}

	logger.Printf("CDN invalidated for video %s", payload.ID)

	// Metrics counter would go here (e.g. cdn_invalidations_total)
	_ = fmt.Sprintf("invalidated:%s", payload.ID)
}
