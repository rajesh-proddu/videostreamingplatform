package bl

import (
	"context"
	"io"
	"log"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/yourusername/videostreamingplatform/userservice/dl"
	"github.com/yourusername/videostreamingplatform/userservice/models"
	"github.com/yourusername/videostreamingplatform/userservice/payment"
	"github.com/yourusername/videostreamingplatform/utils/auth"
)

const testSecret = "test-secret"

type testEnv struct {
	store   *dl.InMemoryStore
	mock    *payment.MockProvider
	authSvc *AuthService
	billing *BillingService
}

func newTestEnv() *testEnv {
	store := dl.NewInMemoryStore()
	mock := payment.NewMockProvider("test-webhook-secret", "http://localhost")
	logger := log.New(io.Discard, "", 0)
	return &testEnv{
		store:   store,
		mock:    mock,
		authSvc: NewAuthService(store, testSecret, 15*time.Minute, 24*time.Hour),
		billing: NewBillingService(store, mock, "http://localhost", logger),
	}
}

func (e *testEnv) registerUser(t *testing.T, email string) string {
	t.Helper()
	u, err := e.authSvc.Register(context.Background(), email, "hunter2")
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	return u.ID
}

func (e *testEnv) referenceID(t *testing.T, subID string) string {
	t.Helper()
	pay, err := e.store.GetPaymentBySubscriptionID(context.Background(), subID)
	if err != nil {
		t.Fatalf("get payment: %v", err)
	}
	return pay.IdempotencyKey
}

func TestRegisterAndLogin(t *testing.T) {
	e := newTestEnv()
	ctx := context.Background()

	if _, err := e.authSvc.Register(ctx, "a@example.com", "pw"); err != nil {
		t.Fatalf("register: %v", err)
	}
	if _, err := e.authSvc.Register(ctx, "a@example.com", "pw"); err != ErrEmailTaken {
		t.Fatalf("expected ErrEmailTaken, got %v", err)
	}

	pair, err := e.authSvc.Login(ctx, "a@example.com", "pw")
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	claims, err := auth.Parse(testSecret, pair.AccessToken)
	if err != nil {
		t.Fatalf("parse access token: %v", err)
	}
	if claims.Entitled {
		t.Fatal("new user should not be entitled")
	}

	if _, err := e.authSvc.Login(ctx, "a@example.com", "wrong"); err != ErrInvalidCredentials {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestSubscribeCreatesPendingPaymentLink(t *testing.T) {
	e := newTestEnv()
	ctx := context.Background()
	uid := e.registerUser(t, "b@example.com")

	res, err := e.billing.Subscribe(ctx, uid, "premium")
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}
	if res.PaymentURL == "" {
		t.Fatal("expected a payment URL")
	}
	if res.Status != string(models.SubPendingPayment) {
		t.Fatalf("expected PENDING_PAYMENT, got %s", res.Status)
	}

	// Re-subscribe is idempotent: same subscription, same link, no new payment.
	res2, err := e.billing.Subscribe(ctx, uid, "premium")
	if err != nil {
		t.Fatalf("re-subscribe: %v", err)
	}
	if res2.SubscriptionID != res.SubscriptionID || res2.PaymentURL != res.PaymentURL {
		t.Fatalf("re-subscribe not idempotent: %+v vs %+v", res, res2)
	}
}

func TestFreePlanDoesNotGrantEntitlement(t *testing.T) {
	e := newTestEnv()
	ctx := context.Background()
	e.registerUser(t, "free@example.com")

	// Subscribing to the free plan activates immediately but must NOT grant the
	// paid entitlement claim, otherwise it bypasses the paywall.
	res, err := e.billing.Subscribe(ctx, mustUserID(t, e, "free@example.com"), "free")
	if err != nil {
		t.Fatalf("subscribe free: %v", err)
	}
	if res.Status != string(models.SubActive) {
		t.Fatalf("expected free plan ACTIVE, got %s", res.Status)
	}

	pair, err := e.authSvc.Login(ctx, "free@example.com", "hunter2")
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	claims, _ := auth.Parse(testSecret, pair.AccessToken)
	if claims.Entitled {
		t.Fatal("free plan must not grant paid entitlement")
	}
}

func mustUserID(t *testing.T, e *testEnv, email string) string {
	t.Helper()
	u, err := e.store.GetUserByEmail(context.Background(), email)
	if err != nil {
		t.Fatalf("get user: %v", err)
	}
	return u.ID
}

func TestSubscribeUnknownPlan(t *testing.T) {
	e := newTestEnv()
	uid := e.registerUser(t, "c@example.com")
	if _, err := e.billing.Subscribe(context.Background(), uid, "platinum"); err != ErrPlanNotFound {
		t.Fatalf("expected ErrPlanNotFound, got %v", err)
	}
}

func TestWebhookActivatesSubscription(t *testing.T) {
	e := newTestEnv()
	ctx := context.Background()
	uid := e.registerUser(t, "d@example.com")

	res, err := e.billing.Subscribe(ctx, uid, "premium")
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}
	ref := e.referenceID(t, res.SubscriptionID)

	body, sig := e.mock.BuildSignedWebhook(ref, "pay_1", true)
	if err := e.billing.ProcessWebhook(ctx, body, sig, "evt_1"); err != nil {
		t.Fatalf("process webhook: %v", err)
	}

	sub, err := e.store.GetSubscriptionByID(ctx, res.SubscriptionID)
	if err != nil {
		t.Fatalf("get sub: %v", err)
	}
	if sub.Status != models.SubActive {
		t.Fatalf("expected ACTIVE, got %s", sub.Status)
	}
	if sub.CurrentPeriodEnd == nil || !sub.CurrentPeriodEnd.After(time.Now()) {
		t.Fatal("expected a future current_period_end")
	}

	// Entitlement now flows into a freshly issued access token.
	pair, err := e.authSvc.Login(ctx, "d@example.com", "hunter2")
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	claims, _ := auth.Parse(testSecret, pair.AccessToken)
	if !claims.Entitled || claims.Plan != "premium" {
		t.Fatalf("expected entitled premium claim, got entitled=%v plan=%s", claims.Entitled, claims.Plan)
	}
}

func TestWebhookIsIdempotent(t *testing.T) {
	e := newTestEnv()
	ctx := context.Background()
	uid := e.registerUser(t, "e@example.com")
	res, _ := e.billing.Subscribe(ctx, uid, "premium")
	ref := e.referenceID(t, res.SubscriptionID)

	body, sig := e.mock.BuildSignedWebhook(ref, "pay_1", true)
	if err := e.billing.ProcessWebhook(ctx, body, sig, "evt_1"); err != nil {
		t.Fatalf("first webhook: %v", err)
	}
	sub1, _ := e.store.GetSubscriptionByID(ctx, res.SubscriptionID)
	firstEnd := *sub1.CurrentPeriodEnd

	// Same event id (duplicate delivery) — must be a no-op.
	if err := e.billing.ProcessWebhook(ctx, body, sig, "evt_1"); err != nil {
		t.Fatalf("duplicate webhook: %v", err)
	}
	// Different event id but already captured — also a no-op (no window extension).
	if err := e.billing.ProcessWebhook(ctx, body, sig, "evt_2"); err != nil {
		t.Fatalf("second webhook: %v", err)
	}
	sub2, _ := e.store.GetSubscriptionByID(ctx, res.SubscriptionID)
	if !sub2.CurrentPeriodEnd.Equal(firstEnd) {
		t.Fatal("period end changed on duplicate webhook")
	}
}

func TestWebhookFailureLeavesPending(t *testing.T) {
	e := newTestEnv()
	ctx := context.Background()
	uid := e.registerUser(t, "f@example.com")
	res, _ := e.billing.Subscribe(ctx, uid, "premium")
	ref := e.referenceID(t, res.SubscriptionID)

	body, sig := e.mock.BuildSignedWebhook(ref, "pay_1", false)
	if err := e.billing.ProcessWebhook(ctx, body, sig, "evt_1"); err != nil {
		t.Fatalf("process webhook: %v", err)
	}

	sub, _ := e.store.GetSubscriptionByID(ctx, res.SubscriptionID)
	if sub.Status != models.SubPendingPayment {
		t.Fatalf("expected PENDING_PAYMENT, got %s", sub.Status)
	}
	pay, _ := e.store.GetPaymentByIdempotencyKey(ctx, ref)
	if pay.Status != models.PaymentFailed {
		t.Fatalf("expected FAILED payment, got %s", pay.Status)
	}
}

func TestWebhookInvalidSignature(t *testing.T) {
	e := newTestEnv()
	ctx := context.Background()
	uid := e.registerUser(t, "g@example.com")
	res, _ := e.billing.Subscribe(ctx, uid, "premium")
	ref := e.referenceID(t, res.SubscriptionID)

	body, _ := e.mock.BuildSignedWebhook(ref, "pay_1", true)
	if err := e.billing.ProcessWebhook(ctx, body, "bad-signature", "evt_1"); err != ErrInvalidSignature {
		t.Fatalf("expected ErrInvalidSignature, got %v", err)
	}
}

func TestDoubleChargeRefundsExtraPayment(t *testing.T) {
	e := newTestEnv()
	ctx := context.Background()
	uid := e.registerUser(t, "h@example.com")
	res, _ := e.billing.Subscribe(ctx, uid, "premium")
	ref1 := e.referenceID(t, res.SubscriptionID)

	// First payment activates the subscription.
	body, sig := e.mock.BuildSignedWebhook(ref1, "pay_1", true)
	if err := e.billing.ProcessWebhook(ctx, body, sig, "evt_1"); err != nil {
		t.Fatalf("first webhook: %v", err)
	}

	// A second, distinct payment lands on the already-active subscription.
	ref2 := uuid.NewString()
	extra := &models.Payment{
		ID: uuid.NewString(), UserID: uid, SubscriptionID: res.SubscriptionID,
		AmountMinor: 29900, Currency: "INR", Status: models.PaymentCreated,
		Provider: "mock", ProviderOrderID: "mock_plink_" + ref2, IdempotencyKey: ref2,
	}
	if err := e.store.CreatePayment(ctx, extra); err != nil {
		t.Fatalf("create extra payment: %v", err)
	}
	body2, sig2 := e.mock.BuildSignedWebhook(ref2, "pay_2", true)
	if err := e.billing.ProcessWebhook(ctx, body2, sig2, "evt_2"); err != nil {
		t.Fatalf("second webhook: %v", err)
	}

	got, _ := e.store.GetPaymentByIdempotencyKey(ctx, ref2)
	if got.Status != models.PaymentRefunded {
		t.Fatalf("expected extra payment REFUNDED, got %s", got.Status)
	}
}

func TestSweepExpiresStalePending(t *testing.T) {
	e := newTestEnv()
	ctx := context.Background()
	uid := e.registerUser(t, "i@example.com")
	res, _ := e.billing.Subscribe(ctx, uid, "premium")

	// Pretend time has advanced past the pending grace window.
	orig := timeNow
	timeNow = func() time.Time { return time.Now().Add(2 * time.Hour) }
	defer func() { timeNow = orig }()

	if err := e.billing.Sweep(ctx, 30*time.Minute); err != nil {
		t.Fatalf("sweep: %v", err)
	}
	sub, _ := e.store.GetSubscriptionByID(ctx, res.SubscriptionID)
	if sub.Status != models.SubExpired {
		t.Fatalf("expected EXPIRED, got %s", sub.Status)
	}
}
