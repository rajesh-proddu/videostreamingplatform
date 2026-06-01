package models

import "time"

// PaymentStatus is the lifecycle state of a payment.
type PaymentStatus string

const (
	PaymentCreated  PaymentStatus = "CREATED"
	PaymentPending  PaymentStatus = "PENDING"
	PaymentCaptured PaymentStatus = "CAPTURED"
	PaymentFailed   PaymentStatus = "FAILED"
	PaymentRefunded PaymentStatus = "REFUNDED"
)

// Payment records a single charge attempt. It holds only opaque provider
// references and amounts — never card/UPI data (PCI scope stays with the
// gateway).
type Payment struct {
	ID                string        `json:"id"`
	UserID            string        `json:"user_id"`
	SubscriptionID    string        `json:"subscription_id"`
	AmountMinor       int64         `json:"amount_minor"`
	Currency          string        `json:"currency"`
	Status            PaymentStatus `json:"status"`
	Provider          string        `json:"provider"`
	ProviderOrderID   string        `json:"provider_order_id,omitempty"`
	ProviderPaymentID string        `json:"provider_payment_id,omitempty"`
	PaymentURL        string        `json:"payment_url,omitempty"` // hosted checkout link (returned on idempotent re-request)
	IdempotencyKey    string        `json:"idempotency_key"`
	CreatedAt         time.Time     `json:"created_at"`
	UpdatedAt         time.Time     `json:"updated_at"`
}
