package payment

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// MockProvider is an in-process gateway for local dev and tests. It needs no
// external account and emulates Razorpay's webhook envelope + HMAC signature so
// the webhook handler is exercised exactly as in production.
type MockProvider struct {
	secret  string
	baseURL string
}

// NewMockProvider returns a mock gateway. secret keys the webhook HMAC; baseURL
// is used to build the stub hosted-checkout URL.
func NewMockProvider(secret, baseURL string) *MockProvider {
	if secret == "" {
		secret = "mock_webhook_secret"
	}
	return &MockProvider{secret: secret, baseURL: baseURL}
}

func (m *MockProvider) Name() string { return "mock" }

// CreateOrder returns a deterministic order id and a stub checkout URL served by
// the userservice (GET /mock/checkout) when PAYMENT_PROVIDER=mock.
func (m *MockProvider) CreateOrder(_ context.Context, r OrderRequest) (Order, error) {
	return Order{
		ProviderOrderID: "mock_plink_" + r.ReferenceID,
		PaymentURL:      fmt.Sprintf("%s/mock/checkout?ref=%s", m.baseURL, r.ReferenceID),
	}, nil
}

func (m *MockProvider) VerifyWebhookSignature(payload []byte, signature string) error {
	return verifyHMACSHA256(payload, signature, m.secret)
}

func (m *MockProvider) ParseWebhookEvent(payload []byte) (WebhookEvent, error) {
	return parsePaymentWebhook(payload)
}

// Refund is a no-op for the mock.
func (m *MockProvider) Refund(_ context.Context, _ string, _ int64) error { return nil }

// GetOrderStatus cannot poll a real gateway, so it reports pending.
// Reconciliation is meaningfully exercised against the Razorpay provider.
func (m *MockProvider) GetOrderStatus(_ context.Context, _ string) (RemoteStatus, error) {
	return StatusPending, nil
}

// BuildSignedWebhook constructs a Razorpay-shaped webhook body and its HMAC
// signature, for the mock checkout endpoint and tests. paid=true emits
// payment_link.paid; paid=false emits payment.failed.
func (m *MockProvider) BuildSignedWebhook(referenceID, paymentID string, paid bool) (body []byte, signature string) {
	var env razorpayEnvelope
	env.Entity = "event"
	env.CreatedAt = time.Now().Unix()
	if paid {
		env.Event = "payment_link.paid"
		env.Payload.PaymentLink.Entity.Status = "paid"
		env.Payload.Payment.Entity.Status = "captured"
	} else {
		env.Event = "payment.failed"
		env.Payload.PaymentLink.Entity.Status = "cancelled"
		env.Payload.Payment.Entity.Status = "failed"
	}
	env.Payload.PaymentLink.Entity.ID = "mock_plink_" + referenceID
	env.Payload.PaymentLink.Entity.ReferenceID = referenceID
	env.Payload.Payment.Entity.ID = paymentID

	body, _ = json.Marshal(&env)
	return body, signHMACSHA256(body, m.secret)
}

// Secret exposes the HMAC secret so the mock checkout handler can sign its
// simulated webhook. Not part of the Provider interface.
func (m *MockProvider) Secret() string { return m.secret }
