package payment

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// razorpayAPIBase is the Razorpay REST base URL. Test vs live mode is selected by
// the API key, not the URL.
const razorpayAPIBase = "https://api.razorpay.com/v1"

// RazorpayProvider integrates Razorpay Payment Links + webhooks. Card/UPI data
// never touches this service — the hosted link collects it.
//
// Wire contract verified against Razorpay docs (2026-05-29):
//   - POST /v1/payment_links  (amount in smallest unit, reference_id unique)
//   - webhook signature: HMAC-SHA256(raw body) in X-Razorpay-Signature
//   - activation event: payment_link.paid
type RazorpayProvider struct {
	keyID         string
	keySecret     string
	webhookSecret string
	publicURL     string
	http          *http.Client
}

// NewRazorpayProvider constructs a Razorpay provider.
func NewRazorpayProvider(keyID, keySecret, webhookSecret, publicURL string) *RazorpayProvider {
	return &RazorpayProvider{
		keyID:         keyID,
		keySecret:     keySecret,
		webhookSecret: webhookSecret,
		publicURL:     publicURL,
		http:          &http.Client{Timeout: 15 * time.Second},
	}
}

func (p *RazorpayProvider) Name() string { return "razorpay" }

// CreateOrder creates a Payment Link and returns its short_url.
func (p *RazorpayProvider) CreateOrder(ctx context.Context, r OrderRequest) (Order, error) {
	reqBody := map[string]any{
		"amount":          r.AmountMinor,
		"currency":        r.Currency,
		"reference_id":    r.ReferenceID,
		"description":     r.Description,
		"notes":           r.Metadata,
		"callback_url":    r.CallbackURL,
		"callback_method": "get",
	}
	if r.CustomerEmail != "" {
		reqBody["customer"] = map[string]string{"email": r.CustomerEmail}
	}

	var resp struct {
		ID       string `json:"id"`
		ShortURL string `json:"short_url"`
	}
	if err := p.do(ctx, http.MethodPost, "/payment_links", reqBody, &resp); err != nil {
		return Order{}, err
	}
	return Order{ProviderOrderID: resp.ID, PaymentURL: resp.ShortURL}, nil
}

func (p *RazorpayProvider) VerifyWebhookSignature(payload []byte, signature string) error {
	return verifyHMACSHA256(payload, signature, p.webhookSecret)
}

func (p *RazorpayProvider) ParseWebhookEvent(payload []byte) (WebhookEvent, error) {
	return parsePaymentWebhook(payload)
}

// Refund reverses a captured payment.
func (p *RazorpayProvider) Refund(ctx context.Context, providerPaymentID string, amountMinor int64) error {
	body := map[string]any{"amount": amountMinor}
	return p.do(ctx, http.MethodPost, "/payments/"+providerPaymentID+"/refund", body, nil)
}

// GetOrderStatus fetches a Payment Link's status for reconciliation.
func (p *RazorpayProvider) GetOrderStatus(ctx context.Context, providerOrderID string) (RemoteStatus, error) {
	var resp struct {
		Status string `json:"status"`
	}
	if err := p.do(ctx, http.MethodGet, "/payment_links/"+providerOrderID, nil, &resp); err != nil {
		return "", err
	}
	switch resp.Status {
	case "paid":
		return StatusCaptured, nil
	case "cancelled", "expired":
		return StatusFailed, nil
	default:
		return StatusPending, nil
	}
}

// do performs an authenticated Razorpay API call, decoding into out (if non-nil).
func (p *RazorpayProvider) do(ctx context.Context, method, path string, body any, out any) error {
	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, razorpayAPIBase+path, reader)
	if err != nil {
		return err
	}
	req.SetBasicAuth(p.keyID, p.keySecret)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := p.http.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("razorpay %s %s: status %d: %s", method, path, resp.StatusCode, string(respBody))
	}
	if out != nil {
		if err := json.Unmarshal(respBody, out); err != nil {
			return fmt.Errorf("decode razorpay response: %w", err)
		}
	}
	return nil
}
