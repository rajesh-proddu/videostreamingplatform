package bl

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"sync"
	"testing"
	"time"

	"github.com/yourusername/videostreamingplatform/userservice/dl"
	"github.com/yourusername/videostreamingplatform/userservice/models"
	"github.com/yourusername/videostreamingplatform/userservice/payment"
	"github.com/yourusername/videostreamingplatform/utils/events"
)

// fakeProducer captures published messages for assertions.
type fakeProducer struct {
	mu       sync.Mutex
	messages [][]byte
}

func (f *fakeProducer) Publish(_ context.Context, _, value []byte) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	cp := make([]byte, len(value))
	copy(cp, value)
	f.messages = append(f.messages, cp)
	return nil
}

func (f *fakeProducer) Close() error { return nil }

func (f *fakeProducer) decoded(t *testing.T) []events.SubscriptionEvent {
	t.Helper()
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []events.SubscriptionEvent
	for _, raw := range f.messages {
		var ev events.SubscriptionEvent
		if err := json.Unmarshal(raw, &ev); err != nil {
			t.Fatalf("unmarshal event: %v", err)
		}
		out = append(out, ev)
	}
	return out
}

func newBillingWithProducer(store *dl.InMemoryStore, p *fakeProducer) *BillingService {
	mock := payment.NewMockProvider("test-webhook-secret", "http://localhost")
	logger := log.New(io.Discard, "", 0)
	return NewBillingService(store, mock, "http://localhost", logger, WithKafkaProducer(p))
}

func TestActivateEmitsActivatedEvent(t *testing.T) {
	ctx := context.Background()
	store := dl.NewInMemoryStore()
	fp := &fakeProducer{}
	billing := newBillingWithProducer(store, fp)
	authSvc := NewAuthService(store, testSecret, 15*time.Minute, 24*time.Hour)

	u, err := authSvc.Register(ctx, "act@example.com", "hunter2")
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	// Free plan activates immediately, exercising activate().
	if _, err := billing.Subscribe(ctx, u.ID, "free"); err != nil {
		t.Fatalf("subscribe: %v", err)
	}

	evs := fp.decoded(t)
	if len(evs) != 1 {
		t.Fatalf("expected 1 event, got %d", len(evs))
	}
	ev := evs[0]
	if ev.Type != events.SubscriptionActivated {
		t.Fatalf("expected %s, got %s", events.SubscriptionActivated, ev.Type)
	}
	if ev.Payload.UserID != u.ID {
		t.Fatalf("expected user_id %s, got %s", u.ID, ev.Payload.UserID)
	}
	if ev.Payload.Status != string(models.SubActive) {
		t.Fatalf("expected status ACTIVE, got %s", ev.Payload.Status)
	}
}

func TestScanExpiringEmitsOnlyForSubsInWindow(t *testing.T) {
	ctx := context.Background()
	store := dl.NewInMemoryStore()
	fp := &fakeProducer{}
	billing := newBillingWithProducer(store, fp)

	now := time.Now()
	inWindow := now.Add(3 * 24 * time.Hour)
	outOfWindow := now.Add(30 * 24 * time.Hour)
	past := now.Add(-1 * time.Hour)

	mustCreate := func(id, userID string, status models.SubscriptionStatus, end *time.Time) {
		if err := store.CreateSubscription(ctx, &models.Subscription{
			ID: id, UserID: userID, PlanID: "premium", Status: status, CurrentPeriodEnd: end,
		}); err != nil {
			t.Fatalf("create %s: %v", id, err)
		}
	}
	mustCreate("s-in", "u-in", models.SubActive, &inWindow)
	mustCreate("s-out", "u-out", models.SubActive, &outOfWindow)
	mustCreate("s-past", "u-past", models.SubActive, &past)                   // already lapsed → out of [now, now+window]
	mustCreate("s-pending", "u-pending", models.SubPendingPayment, &inWindow) // not ACTIVE

	if err := billing.ScanExpiring(ctx, 7*24*time.Hour); err != nil {
		t.Fatalf("scan expiring: %v", err)
	}

	evs := fp.decoded(t)
	if len(evs) != 1 {
		t.Fatalf("expected 1 expiring event, got %d", len(evs))
	}
	if evs[0].Type != events.SubscriptionExpiring {
		t.Fatalf("expected %s, got %s", events.SubscriptionExpiring, evs[0].Type)
	}
	if evs[0].Payload.UserID != "u-in" {
		t.Fatalf("expected user u-in, got %s", evs[0].Payload.UserID)
	}
}

func TestNilProducerEmitIsNoOp(t *testing.T) {
	ctx := context.Background()
	store := dl.NewInMemoryStore()
	logger := log.New(io.Discard, "", 0)
	mock := payment.NewMockProvider("test-webhook-secret", "http://localhost")
	billing := NewBillingService(store, mock, "http://localhost", logger) // no producer
	authSvc := NewAuthService(store, testSecret, 15*time.Minute, 24*time.Hour)

	u, err := authSvc.Register(ctx, "nilp@example.com", "hunter2")
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	// Must not panic with a nil producer.
	if _, err := billing.Subscribe(ctx, u.ID, "free"); err != nil {
		t.Fatalf("subscribe: %v", err)
	}
	if err := billing.ScanExpiring(ctx, 7*24*time.Hour); err != nil {
		t.Fatalf("scan expiring: %v", err)
	}
}
