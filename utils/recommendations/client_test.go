package recommendations

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewClient_SetsBaseURL(t *testing.T) {
	t.Parallel()

	c := NewClient("http://localhost:8000")
	if c.baseURL != "http://localhost:8000" {
		t.Errorf("baseURL = %q, want %q", c.baseURL, "http://localhost:8000")
	}
}

func TestEnabled_True(t *testing.T) {
	t.Parallel()

	c := NewClient("http://localhost:8000")
	if !c.Enabled() {
		t.Error("Enabled() should return true for non-empty baseURL")
	}
}

func TestEnabled_False(t *testing.T) {
	t.Parallel()

	c := NewClient("")
	if c.Enabled() {
		t.Error("Enabled() should return false for empty baseURL")
	}
}

func TestGetRecommendations_Disabled(t *testing.T) {
	t.Parallel()

	c := NewClient("")
	_, err := c.GetRecommendations(context.Background(), "user1", "action", 5)
	if err == nil {
		t.Error("expected error when service is disabled")
	}
}

func TestGetRecommendations_Success(t *testing.T) {
	t.Parallel()

	expected := Response{
		UserID: "user1",
		Recommendations: []Recommendation{
			{VideoID: "v1", Title: "Video 1", Score: 0.9, Reason: "popular"},
			{VideoID: "v2", Title: "Video 2", Score: 0.8, Reason: "trending"},
		},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/api/v1/recommend" {
			t.Errorf("path = %s, want /api/v1/recommend", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(expected)
	}))
	defer srv.Close()

	c := NewClient(srv.URL)
	resp, err := c.GetRecommendations(context.Background(), "user1", "action", 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.UserID != "user1" {
		t.Errorf("UserID = %q, want %q", resp.UserID, "user1")
	}
	if len(resp.Recommendations) != 2 {
		t.Fatalf("got %d recommendations, want 2", len(resp.Recommendations))
	}
	if resp.Recommendations[0].VideoID != "v1" {
		t.Errorf("first recommendation VideoID = %q, want %q", resp.Recommendations[0].VideoID, "v1")
	}
}

func TestGetRecommendations_ServerError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := NewClient(srv.URL)
	_, err := c.GetRecommendations(context.Background(), "user1", "", 5)
	if err == nil {
		t.Error("expected error for 500 response")
	}
}

func TestGetRecommendations_InvalidJSON(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{invalid json`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL)
	_, err := c.GetRecommendations(context.Background(), "user1", "", 5)
	if err == nil {
		t.Error("expected error for invalid JSON response")
	}
}
