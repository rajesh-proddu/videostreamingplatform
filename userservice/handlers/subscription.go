package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/yourusername/videostreamingplatform/userservice/bl"
	"github.com/yourusername/videostreamingplatform/utils/middleware"
)

// SubscriptionHandler serves subscribe and current-subscription endpoints. These
// routes are mounted behind JWTAuth, so the caller's identity comes from the
// token, not the request body.
type SubscriptionHandler struct {
	svc *bl.BillingService
}

// NewSubscriptionHandler constructs a SubscriptionHandler.
func NewSubscriptionHandler(svc *bl.BillingService) *SubscriptionHandler {
	return &SubscriptionHandler{svc: svc}
}

type subscribeRequest struct {
	Plan string `json:"plan"`
}

// Subscribe starts a subscription and returns the hosted payment URL.
func (h *SubscriptionHandler) Subscribe(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}
	var req subscribeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Plan == "" {
		writeError(w, http.StatusBadRequest, "plan is required")
		return
	}
	res, err := h.svc.Subscribe(r.Context(), claims.Subject, req.Plan)
	if errors.Is(err, bl.ErrPlanNotFound) {
		writeError(w, http.StatusNotFound, "unknown plan")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not start subscription")
		return
	}
	writeJSON(w, http.StatusCreated, res)
}

// GetCurrent returns the caller's active subscription, if any.
func (h *SubscriptionHandler) GetCurrent(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}
	sub, err := h.svc.GetCurrentSubscription(r.Context(), claims.Subject)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not load subscription")
		return
	}
	if sub == nil {
		writeJSON(w, http.StatusOK, map[string]any{"active": false})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"active": true, "subscription": sub})
}
