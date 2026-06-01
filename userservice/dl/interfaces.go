// Package dl defines data-persistence interfaces and implementations for the
// user service (users, plans, subscriptions, payments, webhook dedupe).
package dl

import (
	"context"
	"errors"
	"time"

	"github.com/yourusername/videostreamingplatform/userservice/models"
)

// Sentinel errors returned by repositories.
var (
	ErrUserNotFound         = errors.New("user not found")
	ErrPlanNotFound         = errors.New("plan not found")
	ErrSubscriptionNotFound = errors.New("subscription not found")
	ErrPaymentNotFound      = errors.New("payment not found")
)

// IsDuplicate reports whether err is a unique-constraint violation from either
// the MySQL or in-memory store.
func IsDuplicate(err error) bool {
	var de *DuplicateError
	if errors.As(err, &de) {
		return true
	}
	return IsDuplicateKey(err)
}

// Store is the aggregate persistence interface for the user service. A single
// interface keeps service wiring simple; MySQL and in-memory both implement it.
type Store interface {
	// Users
	CreateUser(ctx context.Context, u *models.User) error
	GetUserByEmail(ctx context.Context, email string) (*models.User, error)
	GetUserByID(ctx context.Context, id string) (*models.User, error)

	// Plans
	GetPlanByName(ctx context.Context, name string) (*models.Plan, error)
	GetPlanByID(ctx context.Context, id string) (*models.Plan, error)

	// Subscriptions
	CreateSubscription(ctx context.Context, s *models.Subscription) error
	GetSubscriptionByID(ctx context.Context, id string) (*models.Subscription, error)
	UpdateSubscription(ctx context.Context, s *models.Subscription) error
	// GetActiveSubscription returns the user's current ACTIVE subscription, or
	// ErrSubscriptionNotFound. Used for entitlement claims.
	GetActiveSubscription(ctx context.Context, userID string) (*models.Subscription, error)
	// GetOpenSubscription returns an existing PENDING_PAYMENT or ACTIVE
	// subscription for user+plan (the "one open sub per user+plan" guard).
	GetOpenSubscription(ctx context.Context, userID, planID string) (*models.Subscription, error)
	// ListStalePendingSubscriptions returns PENDING_PAYMENT subscriptions created
	// before the cutoff (sweeper).
	ListStalePendingSubscriptions(ctx context.Context, cutoff time.Time) ([]*models.Subscription, error)

	// Payments
	CreatePayment(ctx context.Context, p *models.Payment) error
	GetPaymentByID(ctx context.Context, id string) (*models.Payment, error)
	GetPaymentByIdempotencyKey(ctx context.Context, key string) (*models.Payment, error)
	GetPaymentBySubscriptionID(ctx context.Context, subID string) (*models.Payment, error)
	UpdatePayment(ctx context.Context, p *models.Payment) error
	// ListPendingPayments returns payments in a non-terminal state (reconcile).
	ListPendingPayments(ctx context.Context) ([]*models.Payment, error)

	// Webhook dedupe. IsWebhookEventProcessed is a read-only check; Mark records
	// the event id (called only after a webhook is successfully applied, so a
	// transient failure doesn't permanently swallow the event).
	IsWebhookEventProcessed(ctx context.Context, eventID string) (bool, error)
	MarkWebhookEventProcessed(ctx context.Context, eventID string) error
}
