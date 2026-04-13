// Package recommendations provides an HTTP client for the recommendation service.
package recommendations

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Recommendation represents a single video recommendation.
type Recommendation struct {
	VideoID string  `json:"video_id"`
	Title   string  `json:"title"`
	Score   float64 `json:"score"`
	Reason  string  `json:"reason"`
}

// Response represents the recommendation service response.
type Response struct {
	UserID          string           `json:"user_id"`
	Recommendations []Recommendation `json:"recommendations"`
	Query           string           `json:"query,omitempty"`
}

// Client communicates with the recommendation service.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient creates a recommendation client. Pass empty baseURL to disable.
func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Enabled returns true if the recommendation service is configured.
func (c *Client) Enabled() bool {
	return c.baseURL != ""
}

// GetRecommendations fetches personalized video recommendations for a user.
func (c *Client) GetRecommendations(ctx context.Context, userID string, query string, limit int) (*Response, error) {
	if !c.Enabled() {
		return nil, fmt.Errorf("recommendation service not configured")
	}

	reqBody, err := json.Marshal(map[string]any{
		"user_id": userID,
		"query":   query,
		"limit":   limit,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/v1/recommend", bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	var result Response
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &result, nil
}
