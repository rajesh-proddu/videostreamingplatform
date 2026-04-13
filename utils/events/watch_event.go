package events

import (
	"encoding/json"
	"time"
)

// Watch event types
const (
	WatchStarted   = "watch.started"
	WatchCompleted = "watch.completed"
)

// WatchEvent is a versioned envelope for watch/download telemetry events.
type WatchEvent struct {
	Version   string       `json:"version"`
	Type      string       `json:"type"`
	Timestamp time.Time    `json:"timestamp"`
	Payload   WatchPayload `json:"payload"`
}

// WatchPayload contains the data for a watch event.
type WatchPayload struct {
	VideoID   string `json:"video_id"`
	UserID    string `json:"user_id"`
	SessionID string `json:"session_id"`
	BytesRead int64  `json:"bytes_read"`
}

// NewWatchEvent creates a new WatchEvent with the current timestamp.
func NewWatchEvent(eventType string, payload WatchPayload) *WatchEvent {
	return &WatchEvent{
		Version:   "1.0",
		Type:      eventType,
		Timestamp: time.Now().UTC(),
		Payload:   payload,
	}
}

// Marshal serializes the event to JSON.
func (e *WatchEvent) Marshal() ([]byte, error) {
	return json.Marshal(e)
}
