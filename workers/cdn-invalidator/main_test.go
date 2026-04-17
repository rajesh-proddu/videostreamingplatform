package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"os"
	"testing"

	kafkago "github.com/segmentio/kafka-go"

	"github.com/yourusername/videostreamingplatform/utils/events"
)

// mockInvalidator records calls and optionally returns an error.
type mockInvalidator struct {
	calls []string
	err   error
}

func (m *mockInvalidator) InvalidateVideo(_ context.Context, videoID string) error {
	m.calls = append(m.calls, videoID)
	return m.err
}

func testLogger() *log.Logger {
	return log.New(os.Stderr, "TEST: ", 0)
}

func makeMessage(t *testing.T, eventType string, payload any) kafkago.Message {
	t.Helper()
	evt := map[string]any{
		"version": "1.0",
		"type":    eventType,
		"payload": payload,
	}
	data, err := json.Marshal(evt)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return kafkago.Message{Value: data}
}

func TestProcessMessage_VideoDeleted_InvalidatesCDN(t *testing.T) {
	t.Parallel()
	inv := &mockInvalidator{}
	msg := makeMessage(t, events.VideoDeleted, map[string]string{"id": "vid-123"})

	processMessage(context.Background(), msg, inv, testLogger())

	if len(inv.calls) != 1 {
		t.Fatalf("expected 1 invalidation call, got %d", len(inv.calls))
	}
	if inv.calls[0] != "vid-123" {
		t.Errorf("invalidated video = %q, want vid-123", inv.calls[0])
	}
}

func TestProcessMessage_VideoCreated_Ignored(t *testing.T) {
	t.Parallel()
	inv := &mockInvalidator{}
	msg := makeMessage(t, events.VideoCreated, map[string]string{"id": "vid-1"})

	processMessage(context.Background(), msg, inv, testLogger())

	if len(inv.calls) != 0 {
		t.Errorf("expected no invalidation for VideoCreated, got %d calls", len(inv.calls))
	}
}

func TestProcessMessage_VideoUpdated_Ignored(t *testing.T) {
	t.Parallel()
	inv := &mockInvalidator{}
	msg := makeMessage(t, events.VideoUpdated, map[string]string{"id": "vid-1"})

	processMessage(context.Background(), msg, inv, testLogger())

	if len(inv.calls) != 0 {
		t.Errorf("expected no invalidation for VideoUpdated, got %d calls", len(inv.calls))
	}
}

func TestProcessMessage_EmptyVideoID_Skipped(t *testing.T) {
	t.Parallel()
	inv := &mockInvalidator{}
	msg := makeMessage(t, events.VideoDeleted, map[string]string{"id": ""})

	processMessage(context.Background(), msg, inv, testLogger())

	if len(inv.calls) != 0 {
		t.Errorf("expected no invalidation for empty ID, got %d calls", len(inv.calls))
	}
}

func TestProcessMessage_InvalidJSON_Skipped(t *testing.T) {
	t.Parallel()
	inv := &mockInvalidator{}
	msg := kafkago.Message{Value: []byte("not valid json")}

	processMessage(context.Background(), msg, inv, testLogger())

	if len(inv.calls) != 0 {
		t.Errorf("expected no invalidation for bad JSON, got %d calls", len(inv.calls))
	}
}

func TestProcessMessage_InvalidPayload_Skipped(t *testing.T) {
	t.Parallel()
	inv := &mockInvalidator{}
	// Valid outer JSON but payload is a string, not an object
	evt := map[string]any{
		"version": "1.0",
		"type":    events.VideoDeleted,
		"payload": "not-an-object",
	}
	data, _ := json.Marshal(evt)
	msg := kafkago.Message{Value: data}

	processMessage(context.Background(), msg, inv, testLogger())

	if len(inv.calls) != 0 {
		t.Errorf("expected no invalidation for bad payload, got %d calls", len(inv.calls))
	}
}

func TestProcessMessage_InvalidationError_DoesNotPanic(t *testing.T) {
	t.Parallel()
	inv := &mockInvalidator{err: errors.New("cloudfront error")}
	msg := makeMessage(t, events.VideoDeleted, map[string]string{"id": "vid-fail"})

	// Should not panic
	processMessage(context.Background(), msg, inv, testLogger())

	if len(inv.calls) != 1 {
		t.Fatalf("expected 1 call even on error, got %d", len(inv.calls))
	}
}

func TestProcessMessage_UnknownEventType_Ignored(t *testing.T) {
	t.Parallel()
	inv := &mockInvalidator{}
	msg := makeMessage(t, "video.archived", map[string]string{"id": "vid-1"})

	processMessage(context.Background(), msg, inv, testLogger())

	if len(inv.calls) != 0 {
		t.Errorf("expected no invalidation for unknown type, got %d calls", len(inv.calls))
	}
}

func TestVideoEvent_Struct(t *testing.T) {
	t.Parallel()
	raw := `{"version":"1.0","type":"video.deleted","payload":{"id":"abc"}}`
	var evt videoEvent
	if err := json.Unmarshal([]byte(raw), &evt); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if evt.Type != "video.deleted" {
		t.Errorf("type = %q, want video.deleted", evt.Type)
	}
	if evt.Version != "1.0" {
		t.Errorf("version = %q, want 1.0", evt.Version)
	}

	var payload deletePayload
	if err := json.Unmarshal(evt.Payload, &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if payload.ID != "abc" {
		t.Errorf("payload.ID = %q, want abc", payload.ID)
	}
}
