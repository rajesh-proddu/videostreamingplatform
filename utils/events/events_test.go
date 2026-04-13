package events

import (
	"encoding/json"
	"testing"
)

func TestNewWatchEvent_Marshal(t *testing.T) {
	evt := NewWatchEvent(WatchStarted, WatchPayload{
		VideoID:   "vid-1",
		UserID:    "user-1",
		SessionID: "sess-1",
		BytesRead: 0,
	})

	data, err := evt.Marshal()
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var parsed WatchEvent
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if parsed.Version != "1.0" {
		t.Errorf("expected version 1.0, got %s", parsed.Version)
	}
	if parsed.Type != WatchStarted {
		t.Errorf("expected type %s, got %s", WatchStarted, parsed.Type)
	}
	if parsed.Payload.VideoID != "vid-1" {
		t.Errorf("expected video_id vid-1, got %s", parsed.Payload.VideoID)
	}
}

func TestNewVideoEvent_Marshal(t *testing.T) {
	payload := map[string]string{"id": "v-42", "title": "Test"}
	evt := NewVideoEvent(VideoCreated, payload)

	data, err := evt.Marshal()
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var parsed VideoEvent
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if parsed.Type != VideoCreated {
		t.Errorf("expected type %s, got %s", VideoCreated, parsed.Type)
	}
	if parsed.Version != "1.0" {
		t.Errorf("expected version 1.0, got %s", parsed.Version)
	}
}
