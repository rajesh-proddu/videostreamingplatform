// Package payment defines a provider-agnostic abstraction over payment gateways.
// The concrete Razorpay implementation and an in-process mock both satisfy
// Provider, so the rest of the service never depends on a gateway's wire format.
package payment

import (
	"context"
	"errors"
)

// ErrInvalidSignature is returned when a webhook signature fails verification.
var ErrInvalidSignature = errors.New("invalid webhook signature")

// OrderRequest is what we hand the gateway to start a payment.
type OrderRequest struct {
	AmountMinor   int64             // amount in the smallest currency unit (e.g. paise)
	Currency      string            // ISO 4217, e.g. "INR"
	ReferenceID   string            // our correlation/idempotency key (unique per order)
	Description   string            // human-readable line item
	CustomerEmail string            // pre-fills the hosted page
	Metadata      map[string]string // echoed back via notes (user_id, plan, subscription_id)
	CallbackURL   string            // UX-only browser return after payment
}

// Order is the gateway's response: an opaque id plus a hosted page to redirect to.
type Order struct {
	ProviderOrderID string
	PaymentURL      string // hosted checkout URL (browser is 302'd here)
}

// WebhookEventType is the normalized event the handler reacts to. Gateway-specific
// event names are collapsed into these by ParseWebhookEvent.
type WebhookEventType string

const (
	EventPaymentCaptured WebhookEventType = "payment_captured"
	EventPaymentFailed   WebhookEventType = "payment_failed"
	EventIgnored         WebhookEventType = "ignored"
)

// WebhookEvent is the normalized form of a gateway webhook. EventID is carried
// separately (header), so it is not set here — it is the dedupe key.
type WebhookEvent struct {
	Type              WebhookEventType
	ReferenceID       string // correlates back to our payment.idempotency_key
	ProviderOrderID   string
	ProviderPaymentID string
}

// RemoteStatus is the normalized status returned by GetPayment (used by the
// reconciliation poll to recover missed webhooks).
type RemoteStatus string

const (
	StatusPending  RemoteStatus = "pending"
	StatusCaptured RemoteStatus = "captured"
	StatusFailed   RemoteStatus = "failed"
)

// Provider is a payment gateway. One concrete provider is selected at startup
// via the PAYMENT_PROVIDER env var.
type Provider interface {
	// Name identifies the provider (stored on the payment row).
	Name() string

	// CreateOrder starts a payment and returns a hosted page to redirect to.
	CreateOrder(ctx context.Context, r OrderRequest) (Order, error)

	// VerifyWebhookSignature validates the raw body against the signature header.
	VerifyWebhookSignature(payload []byte, signature string) error

	// ParseWebhookEvent normalizes a verified webhook body.
	ParseWebhookEvent(payload []byte) (WebhookEvent, error)

	// Refund reverses a captured payment (used on genuine double-charge).
	Refund(ctx context.Context, providerPaymentID string, amountMinor int64) error

	// GetOrderStatus fetches the current status of an order/payment-link from the
	// gateway, keyed by ProviderOrderID (reconciliation of missed webhooks).
	GetOrderStatus(ctx context.Context, providerOrderID string) (RemoteStatus, error)
}
