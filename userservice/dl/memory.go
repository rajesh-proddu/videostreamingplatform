package dl

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/yourusername/videostreamingplatform/userservice/models"
)

// InMemoryStore is an in-memory Store for tests and the UPLOAD_STORE=memory-style
// local mode. It seeds the same default plans as init-db.sql.
type InMemoryStore struct {
	mu            sync.Mutex
	users         map[string]*models.User // keyed by id
	plans         map[string]*models.Plan // keyed by id
	subscriptions map[string]*models.Subscription
	payments      map[string]*models.Payment
	webhookEvents map[string]struct{}
}

// NewInMemoryStore returns a store pre-seeded with the free/premium plans.
func NewInMemoryStore() *InMemoryStore {
	now := time.Now()
	s := &InMemoryStore{
		users:         map[string]*models.User{},
		plans:         map[string]*models.Plan{},
		subscriptions: map[string]*models.Subscription{},
		payments:      map[string]*models.Payment{},
		webhookEvents: map[string]struct{}{},
	}
	s.plans["00000000-0000-0000-0000-000000000001"] = &models.Plan{
		ID: "00000000-0000-0000-0000-000000000001", Name: "free", AmountMinor: 0, Currency: "INR", PeriodDays: 36500, CreatedAt: now,
	}
	s.plans["00000000-0000-0000-0000-000000000002"] = &models.Plan{
		ID: "00000000-0000-0000-0000-000000000002", Name: "premium", AmountMinor: 29900, Currency: "INR", PeriodDays: 30, CreatedAt: now,
	}
	return s
}

// --- Users ---

func (s *InMemoryStore) CreateUser(_ context.Context, u *models.User) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, existing := range s.users {
		if existing.Email == u.Email {
			return &DuplicateError{Field: "email"}
		}
	}
	now := time.Now()
	cp := *u
	cp.CreatedAt, cp.UpdatedAt = now, now
	s.users[u.ID] = &cp
	return nil
}

func (s *InMemoryStore) GetUserByEmail(_ context.Context, email string) (*models.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, u := range s.users {
		if u.Email == email {
			cp := *u
			return &cp, nil
		}
	}
	return nil, ErrUserNotFound
}

func (s *InMemoryStore) GetUserByID(_ context.Context, id string) (*models.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if u, ok := s.users[id]; ok {
		cp := *u
		return &cp, nil
	}
	return nil, ErrUserNotFound
}

// --- Plans ---

func (s *InMemoryStore) GetPlanByName(_ context.Context, name string) (*models.Plan, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, p := range s.plans {
		if p.Name == name {
			cp := *p
			return &cp, nil
		}
	}
	return nil, ErrPlanNotFound
}

func (s *InMemoryStore) GetPlanByID(_ context.Context, id string) (*models.Plan, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if p, ok := s.plans[id]; ok {
		cp := *p
		return &cp, nil
	}
	return nil, ErrPlanNotFound
}

// --- Subscriptions ---

func (s *InMemoryStore) CreateSubscription(_ context.Context, sub *models.Subscription) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	cp := *sub
	cp.CreatedAt, cp.UpdatedAt = now, now
	s.subscriptions[sub.ID] = &cp
	return nil
}

func (s *InMemoryStore) GetSubscriptionByID(_ context.Context, id string) (*models.Subscription, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if sub, ok := s.subscriptions[id]; ok {
		cp := *sub
		return &cp, nil
	}
	return nil, ErrSubscriptionNotFound
}

func (s *InMemoryStore) UpdateSubscription(_ context.Context, sub *models.Subscription) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	existing, ok := s.subscriptions[sub.ID]
	if !ok {
		return ErrSubscriptionNotFound
	}
	existing.Status = sub.Status
	existing.CurrentPeriodEnd = sub.CurrentPeriodEnd
	existing.UpdatedAt = time.Now()
	return nil
}

func (s *InMemoryStore) GetActiveSubscription(_ context.Context, userID string) (*models.Subscription, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	var best *models.Subscription
	for _, sub := range s.subscriptions {
		if sub.UserID == userID && sub.IsActive(now) {
			if best == nil || sub.CurrentPeriodEnd.After(*best.CurrentPeriodEnd) {
				best = sub
			}
		}
	}
	if best == nil {
		return nil, ErrSubscriptionNotFound
	}
	cp := *best
	return &cp, nil
}

func (s *InMemoryStore) GetOpenSubscription(_ context.Context, userID, planID string) (*models.Subscription, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var best *models.Subscription
	for _, sub := range s.subscriptions {
		if sub.UserID == userID && sub.PlanID == planID &&
			(sub.Status == models.SubPendingPayment || sub.Status == models.SubActive) {
			if best == nil || sub.CreatedAt.After(best.CreatedAt) {
				best = sub
			}
		}
	}
	if best == nil {
		return nil, ErrSubscriptionNotFound
	}
	cp := *best
	return &cp, nil
}

func (s *InMemoryStore) ListStalePendingSubscriptions(_ context.Context, cutoff time.Time) ([]*models.Subscription, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []*models.Subscription
	for _, sub := range s.subscriptions {
		if sub.Status == models.SubPendingPayment && sub.CreatedAt.Before(cutoff) {
			cp := *sub
			out = append(out, &cp)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })
	return out, nil
}

func (s *InMemoryStore) ListSubscriptionsExpiringBetween(_ context.Context, from, to time.Time) ([]*models.Subscription, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []*models.Subscription
	for _, sub := range s.subscriptions {
		if sub.Status == models.SubActive && sub.CurrentPeriodEnd != nil &&
			!sub.CurrentPeriodEnd.Before(from) && !sub.CurrentPeriodEnd.After(to) {
			cp := *sub
			out = append(out, &cp)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CurrentPeriodEnd.Before(*out[j].CurrentPeriodEnd) })
	return out, nil
}

// --- Payments ---

func (s *InMemoryStore) CreatePayment(_ context.Context, p *models.Payment) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, existing := range s.payments {
		if existing.IdempotencyKey == p.IdempotencyKey {
			return &DuplicateError{Field: "idempotency_key"}
		}
	}
	now := time.Now()
	cp := *p
	cp.CreatedAt, cp.UpdatedAt = now, now
	s.payments[p.ID] = &cp
	return nil
}

func (s *InMemoryStore) GetPaymentByID(_ context.Context, id string) (*models.Payment, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if p, ok := s.payments[id]; ok {
		cp := *p
		return &cp, nil
	}
	return nil, ErrPaymentNotFound
}

func (s *InMemoryStore) GetPaymentByIdempotencyKey(_ context.Context, key string) (*models.Payment, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, p := range s.payments {
		if p.IdempotencyKey == key {
			cp := *p
			return &cp, nil
		}
	}
	return nil, ErrPaymentNotFound
}

func (s *InMemoryStore) GetPaymentBySubscriptionID(_ context.Context, subID string) (*models.Payment, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var best *models.Payment
	for _, p := range s.payments {
		if p.SubscriptionID == subID {
			if best == nil || p.CreatedAt.After(best.CreatedAt) {
				best = p
			}
		}
	}
	if best == nil {
		return nil, ErrPaymentNotFound
	}
	cp := *best
	return &cp, nil
}

func (s *InMemoryStore) UpdatePayment(_ context.Context, p *models.Payment) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	existing, ok := s.payments[p.ID]
	if !ok {
		return ErrPaymentNotFound
	}
	existing.Status = p.Status
	existing.ProviderOrderID = p.ProviderOrderID
	existing.ProviderPaymentID = p.ProviderPaymentID
	existing.UpdatedAt = time.Now()
	return nil
}

func (s *InMemoryStore) ListPendingPayments(_ context.Context) ([]*models.Payment, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []*models.Payment
	for _, p := range s.payments {
		if p.Status == models.PaymentCreated || p.Status == models.PaymentPending {
			cp := *p
			out = append(out, &cp)
		}
	}
	return out, nil
}

// --- Webhook dedupe ---

func (s *InMemoryStore) IsWebhookEventProcessed(_ context.Context, eventID string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, seen := s.webhookEvents[eventID]
	return seen, nil
}

func (s *InMemoryStore) MarkWebhookEventProcessed(_ context.Context, eventID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.webhookEvents[eventID] = struct{}{}
	return nil
}

// DuplicateError signals a unique-constraint violation in the in-memory store,
// mirroring dl.IsDuplicateKey for MySQL.
type DuplicateError struct{ Field string }

func (e *DuplicateError) Error() string { return "duplicate " + e.Field }
