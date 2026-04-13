// Package events defines versioned event schemas for Kafka publishing.
// Deprecated: These JSON schemas will be replaced with generated code from
// videostreamingplatform-schemas repo (Avro/Protobuf) once codegen is set up.
package events

import (
	"encoding/json"
	"time"
)

// Video event types
const (
	VideoCreated = "video.created"
	VideoUpdated = "video.updated"
	VideoDeleted = "video.deleted"
)

// VideoEvent is a versioned envelope for video lifecycle events.
type VideoEvent struct {
	Version   string    `json:"version"`
	Type      string    `json:"type"`
	Timestamp time.Time `json:"timestamp"`
	Payload   any       `json:"payload"`
}

// NewVideoEvent creates a new VideoEvent with the current timestamp.
func NewVideoEvent(eventType string, payload any) *VideoEvent {
	return &VideoEvent{
		Version:   "1.0",
		Type:      eventType,
		Timestamp: time.Now().UTC(),
		Payload:   payload,
	}
}

// Marshal serializes the event to JSON.
func (e *VideoEvent) Marshal() ([]byte, error) {
	return json.Marshal(e)
}
