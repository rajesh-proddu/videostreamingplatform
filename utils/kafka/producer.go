// Package kafka provides a thin Kafka producer wrapper using segmentio/kafka-go.
package kafka

import (
	"context"
	"fmt"
	"time"

	kafkago "github.com/segmentio/kafka-go"
)

// Producer defines the interface for publishing messages to Kafka.
type Producer interface {
	Publish(ctx context.Context, key, value []byte) error
	Close() error
}

// KafkaProducer is the concrete implementation using segmentio/kafka-go.
type KafkaProducer struct {
	writer *kafkago.Writer
}

// NewProducer creates a new Kafka producer for the given brokers and topic.
func NewProducer(brokers []string, topic string) *KafkaProducer {
	w := &kafkago.Writer{
		Addr:         kafkago.TCP(brokers...),
		Topic:        topic,
		Balancer:     &kafkago.LeastBytes{},
		BatchTimeout: 10 * time.Millisecond,
		RequiredAcks: kafkago.RequireOne,
	}
	return &KafkaProducer{writer: w}
}

// Publish sends a message to Kafka with the given key and value.
func (p *KafkaProducer) Publish(ctx context.Context, key, value []byte) error {
	msg := kafkago.Message{
		Key:   key,
		Value: value,
	}
	if err := p.writer.WriteMessages(ctx, msg); err != nil {
		return fmt.Errorf("kafka publish: %w", err)
	}
	return nil
}

// Close closes the Kafka writer.
func (p *KafkaProducer) Close() error {
	return p.writer.Close()
}
