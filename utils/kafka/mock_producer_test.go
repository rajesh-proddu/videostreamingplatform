package kafka

import (
	"context"
	"testing"
)

func TestMockProducer_Publish(t *testing.T) {
	mock := NewMockProducer()

	err := mock.Publish(context.Background(), []byte("key1"), []byte("value1"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(mock.Messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(mock.Messages))
	}
	if string(mock.Messages[0].Key) != "key1" {
		t.Errorf("expected key 'key1', got '%s'", mock.Messages[0].Key)
	}
}

func TestMockProducer_PublishError(t *testing.T) {
	mock := NewMockProducer()
	mock.Err = context.DeadlineExceeded

	err := mock.Publish(context.Background(), []byte("key"), []byte("val"))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if len(mock.Messages) != 0 {
		t.Fatalf("expected 0 messages on error, got %d", len(mock.Messages))
	}
}

func TestMockProducer_Reset(t *testing.T) {
	mock := NewMockProducer()
	_ = mock.Publish(context.Background(), []byte("k"), []byte("v"))
	mock.Reset()

	if len(mock.Messages) != 0 {
		t.Fatalf("expected 0 messages after reset, got %d", len(mock.Messages))
	}
}
