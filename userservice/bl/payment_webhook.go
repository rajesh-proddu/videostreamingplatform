package bl

import (
	"context"
	"errors"

	"github.com/yourusername/videostreamingplatform/userservice/dl"
	"github.com/yourusername/videostreamingplatform/userservice/models"
	"github.com/yourusername/videostreamingplatform/userservice/payment"
)

// ProcessWebhook verifies, deduplicates, and applies a gateway webhook. It is the
// source of truth for subscription activation. The signature is checked over the
// raw body; the eventID (provider-unique) is the dedupe key. Returning nil means
// "acknowledge" — the handler then replies 2xx.
func (s *BillingService) ProcessWebhook(ctx context.Context, body []byte, signature, eventID string) error {
	if err := s.provider.VerifyWebhookSignature(body, signature); err != nil {
		return ErrInvalidSignature
	}

	// Dedupe: at-least-once delivery means the same event id can arrive twice.
	// This is an optimization — the apply path below is idempotent regardless, so
	// we only record the event id *after* a successful apply. That way a transient
	// failure (returning non-2xx) lets the gateway retry instead of being
	// permanently swallowed.
	if seen, err := s.store.IsWebhookEventProcessed(ctx, eventID); err != nil {
		return err
	} else if seen {
		return nil
	}

	ev, err := s.provider.ParseWebhookEvent(body)
	if err != nil {
		return err
	}
	if ev.Type == payment.EventIgnored || ev.ReferenceID == "" {
		return nil // not a payment event we act on
	}

	pay, err := s.store.GetPaymentByIdempotencyKey(ctx, ev.ReferenceID)
	if errors.Is(err, dl.ErrPaymentNotFound) {
		return nil // unknown reference — ack and move on
	}
	if err != nil {
		return err
	}

	switch ev.Type {
	case payment.EventPaymentCaptured:
		if err := s.applyCapture(ctx, pay, ev.ProviderOrderID, ev.ProviderPaymentID); err != nil {
			return err
		}
	case payment.EventPaymentFailed:
		if err := s.applyFailure(ctx, pay); err != nil {
			return err
		}
	}

	return s.store.MarkWebhookEventProcessed(ctx, eventID)
}

// applyCapture marks a payment captured and activates its subscription. The state
// machine makes a repeated capture a no-op; a capture arriving for an
// already-active subscription is treated as a genuine double charge and refunded.
func (s *BillingService) applyCapture(ctx context.Context, pay *models.Payment, orderID, paymentID string) error {
	if pay.Status == models.PaymentCaptured || pay.Status == models.PaymentRefunded {
		return nil // idempotent: already settled
	}

	sub, err := s.store.GetSubscriptionByID(ctx, pay.SubscriptionID)
	if err != nil {
		return err
	}

	// Genuine double charge: the subscription is already active from a prior
	// capture, so refund this extra payment rather than extending the window.
	if sub.IsActive(timeNow()) {
		pay.ProviderOrderID = orderID
		pay.ProviderPaymentID = paymentID
		if err := s.settle(ctx, pay, models.PaymentCaptured); err != nil {
			return err
		}
		if err := s.provider.Refund(ctx, paymentID, pay.AmountMinor); err != nil {
			s.logger.Printf("WARNING: refund failed for payment %s: %v", pay.ID, err)
			return err
		}
		return s.settle(ctx, pay, models.PaymentRefunded)
	}

	plan, err := s.store.GetPlanByID(ctx, sub.PlanID)
	if err != nil {
		return err
	}
	pay.ProviderOrderID = orderID
	pay.ProviderPaymentID = paymentID
	if err := s.settle(ctx, pay, models.PaymentCaptured); err != nil {
		return err
	}
	return s.activate(ctx, sub, plan)
}

// applyFailure marks a payment failed; the subscription stays PENDING_PAYMENT.
func (s *BillingService) applyFailure(ctx context.Context, pay *models.Payment) error {
	if pay.Status == models.PaymentCaptured || pay.Status == models.PaymentRefunded {
		return nil // never fail a settled payment
	}
	return s.settle(ctx, pay, models.PaymentFailed)
}

// settle transitions a payment if the state machine allows it (no-op otherwise).
func (s *BillingService) settle(ctx context.Context, pay *models.Payment, to models.PaymentStatus) error {
	if pay.Status == to {
		return nil
	}
	if !canTransition(pay.Status, to) {
		s.logger.Printf("WARNING: illegal payment transition %s→%s for %s", pay.Status, to, pay.ID)
		return nil
	}
	pay.Status = to
	return s.store.UpdatePayment(ctx, pay)
}
