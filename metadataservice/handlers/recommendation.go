package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/yourusername/videostreamingplatform/utils/recommendations"
)

// RecommendationHandler proxies requests to the recommendation service.
type RecommendationHandler struct {
	client *recommendations.Client
}

// NewRecommendationHandler creates a new recommendation handler.
func NewRecommendationHandler(client *recommendations.Client) *RecommendationHandler {
	return &RecommendationHandler{client: client}
}

// GetRecommendations godoc
// @Summary Get video recommendations for a user
// @Description Returns personalized video recommendations from the AI recommendation service
// @Tags recommendations
// @Accept json
// @Produce json
// @Param user_id query string true "User ID"
// @Param query query string false "Optional search query to bias recommendations"
// @Param limit query int false "Maximum number of recommendations" default(10)
// @Success 200 {object} recommendations.Response
// @Failure 400 {object} map[string]string
// @Failure 503 {object} map[string]string
// @Router /recommendations [get]
func (h *RecommendationHandler) GetRecommendations(w http.ResponseWriter, r *http.Request) {
	if !h.client.Enabled() {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{"error": "recommendation service not configured"})
		return
	}

	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "user_id is required"})
		return
	}

	query := r.URL.Query().Get("query")
	limit := 10

	resp, err := h.client.GetRecommendations(r.Context(), userID, query, limit)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{"error": "recommendation service unavailable"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
