package bl

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/google/uuid"

	"github.com/yourusername/videostreamingplatform/userservice/dl"
	"github.com/yourusername/videostreamingplatform/userservice/models"
	"github.com/yourusername/videostreamingplatform/userservice/payment"
)

// BillingService owns plans, subscriptions, payments, webhook processing, and
// the reconciliation/sweeper jobs. Keeping them on one type lets the webhook and
// subscribe paths share the activation/state-machine helpers.
type BillingService struct {
	store     dl.Store
	provider  payment.Provider
	publicURL string
	logger    *log.Logger
}

// timeNow is indirected so tests can control time.
var timeNow = time.Now

// NewBillingService constructs a BillingService.
func NewBillingService(store dl.Store, provider payment.Provider, publicURL string, logger *log.Logger) *BillingService {
	return &BillingService{store: store, provider: provider, publicURL: publicURL, logger: logger}
}

// SubscribeResult is returned from Subscribe.
type SubscribeResult struct {
	SubscriptionID string `json:"subscription_id"`
	PaymentURL     string `json:"payment_url"` // empty when activated immediately (free plan)
	Status         string `json:"status"`      // subscription status
	AlreadyActive  bool   `json:"already_active,omitempty"`
}

// Subscribe starts (or resumes) a subscription to planName for the user. For a
// paid plan it creates a hosted payment link and returns its URL; the
// subscription is only activated when the payment is captured via webhook. For a
// zero-price plan it activates immediately. The "one open subscription per
// user+plan" guard makes a repeated call idempotent.
func (s *BillingService) Subscribe(ctx context.Context, userID, planName string) (*SubscribeResult, error) {
	plan, err := s.store.GetPlanByName(ctx, planName)
	if errors.Is(err, dl.ErrPlanNotFound) {
		return nil, ErrPlanNotFound
	}
	if err != nil {
		return nil, err
	}

	// Guard: reuse an existing open subscription rather than creating a duplicate.
	if open, err := s.store.GetOpenSubscription(ctx, userID, plan.ID); err == nil {
		if open.Status == models.SubActive {
			return &SubscribeResult{SubscriptionID: open.ID, Status: string(open.Status), AlreadyActive: true}, nil
		}
		// PENDING_PAYMENT — return the existing payment link.
		pay, err := s.store.GetPaymentBySubscriptionID(ctx, open.ID)
		if err == nil {
			return &SubscribeResult{SubscriptionID: open.ID, PaymentURL: pay.PaymentURL, Status: string(open.Status)}, nil
		}
	} else if !errors.Is(err, dl.ErrSubscriptionNotFound) {
		return nil, err
	}

	sub := &models.Subscription{
		ID:     uuid.NewString(),
		UserID: userID,
		PlanID: plan.ID,
		Status: models.SubPendingPayment,
	}
	if err := s.store.CreateSubscription(ctx, sub); err != nil {
		return nil, err
	}

	// Free plan: activate immediately, no payment needed.
	if plan.AmountMinor == 0 {
		if err := s.activate(ctx, sub, plan); err != nil {
			return nil, err
		}
		return &SubscribeResult{SubscriptionID: sub.ID, Status: string(models.SubActive)}, nil
	}

	referenceID := uuid.NewString() // unique per payment link; our idempotency key
	order, err := s.provider.CreateOrder(ctx, payment.OrderRequest{
		AmountMinor: plan.AmountMinor,
		Currency:    plan.Currency,
		ReferenceID: referenceID,
		Description: "Subscription: " + plan.Name,
		Metadata:    map[string]string{"user_id": userID, "plan": plan.Name, "subscription_id": sub.ID},
		CallbackURL: s.publicURL + "/subscriptions/return",
	})
	if err != nil {
		return nil, err
	}

	pay := &models.Payment{
		ID:              uuid.NewString(),
		UserID:          userID,
		SubscriptionID:  sub.ID,
		AmountMinor:     plan.AmountMinor,
		Currency:        plan.Currency,
		Status:          models.PaymentCreated,
		Provider:        s.provider.Name(),
		ProviderOrderID: order.ProviderOrderID,
		PaymentURL:      order.PaymentURL,
		IdempotencyKey:  referenceID,
	}
	if err := s.store.CreatePayment(ctx, pay); err != nil {
		return nil, err
	}

	return &SubscribeResult{SubscriptionID: sub.ID, PaymentURL: order.PaymentURL, Status: string(models.SubPendingPayment)}, nil
}

// GetCurrentSubscription returns the user's active subscription, or nil if none.
func (s *BillingService) GetCurrentSubscription(ctx context.Context, userID string) (*models.Subscription, error) {
	sub, err := s.store.GetActiveSubscription(ctx, userID)
	if errors.Is(err, dl.ErrSubscriptionNotFound) {
		return nil, nil
	}
	return sub, err
}

// activate moves a subscription to ACTIVE and sets its access window.
func (s *BillingService) activate(ctx context.Context, sub *models.Subscription, plan *models.Plan) error {
	end := time.Now().Add(time.Duration(plan.PeriodDays) * 24 * time.Hour)
	sub.Status = models.SubActive
	sub.CurrentPeriodEnd = &end
	return s.store.UpdateSubscription(ctx, sub)
}

// paymentTransitions is the allowed payment state machine. CAPTURED and FAILED
// are terminal except CAPTURED→REFUNDED.
var paymentTransitions = map[models.PaymentStatus]map[models.PaymentStatus]bool{
	models.PaymentCreated:  {models.PaymentPending: true, models.PaymentCaptured: true, models.PaymentFailed: true},
	models.PaymentPending:  {models.PaymentCaptured: true, models.PaymentFailed: true},
	models.PaymentCaptured: {models.PaymentRefunded: true},
	models.PaymentFailed:   {},
	models.PaymentRefunded: {},
}

// canTransition reports whether a payment may move from→to.
func canTransition(from, to models.PaymentStatus) bool {
	return paymentTransitions[from][to]
}
