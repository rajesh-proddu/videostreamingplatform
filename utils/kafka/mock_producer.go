package kafka

import (
	"context"
	"sync"
)

// MockProducer is a test double that records published messages.
type MockProducer struct {
	mu       sync.Mutex
	Messages []MockMessage
	Err      error // if set, Publish returns this error
}

// MockMessage stores a published message for test assertions.
type MockMessage struct {
	Key   []byte
	Value []byte
}

// NewMockProducer creates a mock producer for testing.
func NewMockProducer() *MockProducer {
	return &MockProducer{}
}

// Publish records the message (or returns the configured error).
func (m *MockProducer) Publish(_ context.Context, key, value []byte) error {
	if m.Err != nil {
		return m.Err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Messages = append(m.Messages, MockMessage{Key: key, Value: value})
	return nil
}

// Close is a no-op for the mock.
func (m *MockProducer) Close() error { return nil }

// Reset clears recorded messages.
func (m *MockProducer) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Messages = nil
	m.Err = nil
}
