package bl

import (
	"context"
	"time"

	"github.com/yourusername/videostreamingplatform/userservice/models"
	"github.com/yourusername/videostreamingplatform/userservice/payment"
	"github.com/yourusername/videostreamingplatform/utils/events"
)

// Reconcile is the backstop for missed webhooks: it polls the gateway for the
// status of every non-terminal payment and applies the outcome. Safe to run
// repeatedly — the state machine makes re-application a no-op.
func (s *BillingService) Reconcile(ctx context.Context) error {
	payments, err := s.store.ListPendingPayments(ctx)
	if err != nil {
		return err
	}
	for _, pay := range payments {
		if pay.ProviderOrderID == "" {
			continue // nothing to poll yet
		}
		status, err := s.provider.GetOrderStatus(ctx, pay.ProviderOrderID)
		if err != nil {
			s.logger.Printf("reconcile: GetOrderStatus(%s): %v", pay.ProviderOrderID, err)
			continue
		}
		switch status {
		case payment.StatusCaptured:
			if err := s.applyCapture(ctx, pay, pay.ProviderOrderID, pay.ProviderPaymentID); err != nil {
				s.logger.Printf("reconcile: applyCapture(%s): %v", pay.ID, err)
			}
		case payment.StatusFailed:
			if err := s.applyFailure(ctx, pay); err != nil {
				s.logger.Printf("reconcile: applyFailure(%s): %v", pay.ID, err)
			}
		}
	}
	return nil
}

// Sweep expires subscriptions that have been awaiting payment longer than maxAge,
// so abandoned checkouts don't linger as PENDING_PAYMENT.
func (s *BillingService) Sweep(ctx context.Context, maxAge time.Duration) error {
	cutoff := timeNow().Add(-maxAge)
	stale, err := s.store.ListStalePendingSubscriptions(ctx, cutoff)
	if err != nil {
		return err
	}
	for _, sub := range stale {
		sub.Status = models.SubExpired
		if err := s.store.UpdateSubscription(ctx, sub); err != nil {
			s.logger.Printf("sweep: expire subscription %s: %v", sub.ID, err)
		}
	}
	return nil
}

// ScanExpiring emits a SUBSCRIPTION_EXPIRING event for every active subscription
// whose access window ends within `window` from now — the "your plan is about to
// expire" notification. Downstream delivery is idempotent on a deterministic
// dedupe key, so re-running the scan does not produce duplicate notifications.
func (s *BillingService) ScanExpiring(ctx context.Context, window time.Duration) error {
	now := timeNow()
	subs, err := s.store.ListSubscriptionsExpiringBetween(ctx, now, now.Add(window))
	if err != nil {
		return err
	}
	for _, sub := range subs {
		s.emitSubscriptionEvent(ctx, events.SubscriptionExpiring, sub)
	}
	return nil
}

// RunBackgroundJobs runs Reconcile + Sweep on the `interval` ticker and the
// expiring-subscription scan on the `expiringScanInterval` ticker, until ctx is
// cancelled. The expiring scan also runs once at startup so a redeploy doesn't
// leave a full interval with no scan.
func (s *BillingService) RunBackgroundJobs(ctx context.Context, interval, pendingMaxAge, expiringScanInterval, expiringWindow time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	expiringTicker := time.NewTicker(expiringScanInterval)
	defer expiringTicker.Stop()

	if err := s.ScanExpiring(ctx, expiringWindow); err != nil {
		s.logger.Printf("expiring scan (startup): %v", err)
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := s.Reconcile(ctx); err != nil {
				s.logger.Printf("reconcile job: %v", err)
			}
			if err := s.Sweep(ctx, pendingMaxAge); err != nil {
				s.logger.Printf("sweep job: %v", err)
			}
		case <-expiringTicker.C:
			if err := s.ScanExpiring(ctx, expiringWindow); err != nil {
				s.logger.Printf("expiring scan job: %v", err)
			}
		}
	}
}
