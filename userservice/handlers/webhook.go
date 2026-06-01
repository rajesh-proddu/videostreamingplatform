package handlers

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"net/http"

	"github.com/google/uuid"

	"github.com/yourusername/videostreamingplatform/userservice/bl"
	"github.com/yourusername/videostreamingplatform/userservice/payment"
)

// Razorpay webhook header names (the mock emulates these).
const (
	headerSignature = "X-Razorpay-Signature"
	headerEventID   = "X-Razorpay-Event-Id"
)

// WebhookHandler processes gateway payment webhooks.
type WebhookHandler struct {
	svc *bl.BillingService
}

// NewWebhookHandler constructs a WebhookHandler.
func NewWebhookHandler(svc *bl.BillingService) *WebhookHandler {
	return &WebhookHandler{svc: svc}
}

// PaymentWebhook verifies, deduplicates, and applies a payment webhook. It reads
// the raw body first (signature is over raw bytes), then hands off to the billing
// service. A 2xx tells the gateway the event is acknowledged; anything else
// triggers a retry.
func (h *WebhookHandler) PaymentWebhook(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		writeError(w, http.StatusBadRequest, "could not read body")
		return
	}
	signature := r.Header.Get(headerSignature)
	eventID := r.Header.Get(headerEventID)
	if eventID == "" {
		// Fallback dedupe key for providers that don't send an event id header.
		sum := sha256.Sum256(body)
		eventID = hex.EncodeToString(sum[:])
	}

	if err := h.svc.ProcessWebhook(r.Context(), body, signature, eventID); err != nil {
		if errors.Is(err, bl.ErrInvalidSignature) {
			writeError(w, http.StatusBadRequest, "invalid signature")
			return
		}
		// Transient failure — return non-2xx so the gateway retries.
		writeError(w, http.StatusInternalServerError, "processing error")
		return
	}
	w.WriteHeader(http.StatusOK)
}

// MockCheckoutHandler simulates the hosted payment page for PAYMENT_PROVIDER=mock.
// Visiting it fires a signed webhook into the billing service, exactly as a real
// gateway would after the user pays.
type MockCheckoutHandler struct {
	svc  *bl.BillingService
	mock *payment.MockProvider
}

// NewMockCheckoutHandler constructs a MockCheckoutHandler.
func NewMockCheckoutHandler(svc *bl.BillingService, mock *payment.MockProvider) *MockCheckoutHandler {
	return &MockCheckoutHandler{svc: svc, mock: mock}
}

// Checkout simulates a payment. Query params: ref (reference id), outcome
// ("paid" default, or "failed").
func (h *MockCheckoutHandler) Checkout(w http.ResponseWriter, r *http.Request) {
	ref := r.URL.Query().Get("ref")
	if ref == "" {
		writeError(w, http.StatusBadRequest, "ref is required")
		return
	}
	paid := r.URL.Query().Get("outcome") != "failed"
	paymentID := "mock_pay_" + uuid.NewString()

	body, signature := h.mock.BuildSignedWebhook(ref, paymentID, paid)
	if err := h.svc.ProcessWebhook(r.Context(), body, signature, uuid.NewString()); err != nil {
		writeError(w, http.StatusInternalServerError, "simulated payment failed: "+err.Error())
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	status := "captured"
	if !paid {
		status = "failed"
	}
	_, _ = w.Write([]byte("<html><body><h2>Mock payment " + status + "</h2><p>You may close this page.</p></body></html>"))
}
