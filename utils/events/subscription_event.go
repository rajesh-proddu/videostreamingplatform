package events

import (
	"encoding/json"
	"time"
)

// Subscription event types. The wire `type` uses the dotted form (matching
// video.* events); the Avro/Proto enum symbols are SUBSCRIPTION_* in the
// videostreamingplatform-schemas repo.
const (
	SubscriptionActivated = "subscription.activated"
	SubscriptionExpiring  = "subscription.expiring"
	SubscriptionExpired   = "subscription.expired"
	SubscriptionCancelled = "subscription.cancelled"
)

// SubscriptionPayload describes the subscription a SubscriptionEvent is about.
type SubscriptionPayload struct {
	UserID           string     `json:"user_id"`
	SubscriptionID   string     `json:"subscription_id"`
	PlanID           string     `json:"plan_id,omitempty"`
	Status           string     `json:"status,omitempty"`
	CurrentPeriodEnd *time.Time `json:"current_period_end,omitempty"`
}

// SubscriptionEvent is a versioned envelope for subscription lifecycle events.
type SubscriptionEvent struct {
	Version   string              `json:"version"`
	Type      string              `json:"type"`
	Timestamp time.Time           `json:"timestamp"`
	Payload   SubscriptionPayload `json:"payload"`
}

// NewSubscriptionEvent creates a new SubscriptionEvent with the current timestamp.
func NewSubscriptionEvent(eventType string, payload SubscriptionPayload) *SubscriptionEvent {
	return &SubscriptionEvent{
		Version:   "1.0",
		Type:      eventType,
		Timestamp: time.Now().UTC(),
		Payload:   payload,
	}
}

// Marshal serializes the event to JSON.
func (e *SubscriptionEvent) Marshal() ([]byte, error) {
	return json.Marshal(e)
}
