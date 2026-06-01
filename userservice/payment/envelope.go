package payment

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
)

// razorpayEnvelope is the webhook body shape published by Razorpay. The mock
// provider deliberately emulates the same envelope + HMAC scheme so a single
// handler and parser serve both.
//
// Field paths verified against Razorpay docs (2026-05-29):
//
//	payload.payment_link.entity.{id,status,reference_id,amount,amount_paid}
//	payload.payment.entity.{id,status,order_id,amount}
type razorpayEnvelope struct {
	Entity  string `json:"entity"`
	Event   string `json:"event"`
	Payload struct {
		PaymentLink struct {
			Entity struct {
				ID          string `json:"id"`
				Status      string `json:"status"`
				ReferenceID string `json:"reference_id"`
				Amount      int64  `json:"amount"`
				AmountPaid  int64  `json:"amount_paid"`
			} `json:"entity"`
		} `json:"payment_link"`
		Payment struct {
			Entity struct {
				ID      string `json:"id"`
				Status  string `json:"status"`
				OrderID string `json:"order_id"`
				Amount  int64  `json:"amount"`
			} `json:"entity"`
		} `json:"payment"`
	} `json:"payload"`
	CreatedAt int64 `json:"created_at"`
}

// parsePaymentWebhook normalizes a Razorpay-shaped webhook body.
func parsePaymentWebhook(payload []byte) (WebhookEvent, error) {
	var env razorpayEnvelope
	if err := json.Unmarshal(payload, &env); err != nil {
		return WebhookEvent{}, fmt.Errorf("parse webhook payload: %w", err)
	}

	ev := WebhookEvent{
		ReferenceID:       env.Payload.PaymentLink.Entity.ReferenceID,
		ProviderOrderID:   env.Payload.PaymentLink.Entity.ID,
		ProviderPaymentID: env.Payload.Payment.Entity.ID,
	}

	switch env.Event {
	case "payment_link.paid", "order.paid", "payment.captured":
		ev.Type = EventPaymentCaptured
	case "payment.failed", "payment_link.expired", "payment_link.cancelled":
		ev.Type = EventPaymentFailed
	default:
		ev.Type = EventIgnored
	}
	return ev, nil
}

// signHMACSHA256 returns the hex-encoded HMAC-SHA256 of payload keyed by secret.
func signHMACSHA256(payload []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	return hex.EncodeToString(mac.Sum(nil))
}

// verifyHMACSHA256 constant-time compares the expected signature.
func verifyHMACSHA256(payload []byte, signature, secret string) error {
	expected := signHMACSHA256(payload, secret)
	if !hmac.Equal([]byte(expected), []byte(signature)) {
		return ErrInvalidSignature
	}
	return nil
}
