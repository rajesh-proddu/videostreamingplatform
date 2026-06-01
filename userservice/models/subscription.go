package models

import "time"

// SubscriptionStatus is the lifecycle state of a subscription.
type SubscriptionStatus string

const (
	SubPendingPayment SubscriptionStatus = "PENDING_PAYMENT"
	SubActive         SubscriptionStatus = "ACTIVE"
	SubExpired        SubscriptionStatus = "EXPIRED"
	SubCancelled      SubscriptionStatus = "CANCELLED"
)

// Subscription ties a user to a plan for a fixed access window. The window is
// activated only when the associated payment is captured (via webhook).
type Subscription struct {
	ID               string             `json:"id"`
	UserID           string             `json:"user_id"`
	PlanID           string             `json:"plan_id"`
	Status           SubscriptionStatus `json:"status"`
	CurrentPeriodEnd *time.Time         `json:"current_period_end,omitempty"`
	CreatedAt        time.Time          `json:"created_at"`
	UpdatedAt        time.Time          `json:"updated_at"`
}

// IsActive reports whether the subscription currently grants access.
func (s *Subscription) IsActive(now time.Time) bool {
	return s.Status == SubActive && s.CurrentPeriodEnd != nil && s.CurrentPeriodEnd.After(now)
}
