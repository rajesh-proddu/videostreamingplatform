package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}

func TestRateLimiter_AllowsRequestsUnderLimit(t *testing.T) {
	t.Parallel()

	rl := NewRateLimiter(nil, 10, time.Minute, 10)
	handler := rl.Middleware(okHandler())

	for i := 0; i < 10; i++ {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "1.2.3.4:1234"
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("request %d: status = %d, want 200", i, rec.Code)
		}
	}
}

func TestRateLimiter_BlocksOverLimit(t *testing.T) {
	t.Parallel()

	rl := NewRateLimiter(nil, 5, time.Minute, 5)
	handler := rl.Middleware(okHandler())

	// Exhaust the limit
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "10.0.0.1:1234"
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
	}

	// Next request should be rate limited
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.0.0.1:1234"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusTooManyRequests)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}
	if rec.Header().Get("Retry-After") == "" {
		t.Error("missing Retry-After header")
	}
}

func TestRateLimiter_DifferentIPsAreIndependent(t *testing.T) {
	t.Parallel()

	rl := NewRateLimiter(nil, 2, time.Minute, 2)
	handler := rl.Middleware(okHandler())

	// Exhaust IP 1
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "10.0.0.1:1234"
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
	}

	// IP 2 should still work
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.0.0.2:5678"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("different IP should not be rate limited, got %d", rec.Code)
	}
}

func TestRateLimiter_SetsRateLimitHeaders(t *testing.T) {
	t.Parallel()

	rl := NewRateLimiter(nil, 100, time.Minute, 100)
	handler := rl.Middleware(okHandler())

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "192.168.1.1:9999"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Header().Get("X-RateLimit-Limit") != "100" {
		t.Errorf("X-RateLimit-Limit = %q, want 100", rec.Header().Get("X-RateLimit-Limit"))
	}
	if rec.Header().Get("X-RateLimit-Remaining") == "" {
		t.Error("missing X-RateLimit-Remaining header")
	}
}

func TestRateLimiter_SkipsProbeAndOpsPaths(t *testing.T) {
	t.Parallel()

	rl := NewRateLimiter(nil, 1, time.Minute, 1)
	handler := rl.Middleware(okHandler())

	for _, path := range []string{"/health", "/livez", "/readyz", "/metrics"} {
		for i := 0; i < 5; i++ {
			req := httptest.NewRequest(http.MethodGet, path, nil)
			req.RemoteAddr = "10.0.0.99:1234"
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("%s request %d: status = %d, want 200 (probe paths must bypass rate limit)", path, i, rec.Code)
			}
		}
	}
}

func TestClientIP_XForwardedFor(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Forwarded-For", "203.0.113.50, 70.41.3.18, 150.172.238.178")

	ip := clientIP(req)
	if ip != "203.0.113.50" {
		t.Errorf("clientIP = %q, want 203.0.113.50", ip)
	}
}

func TestClientIP_XRealIP(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Real-IP", "198.51.100.178")

	ip := clientIP(req)
	if ip != "198.51.100.178" {
		t.Errorf("clientIP = %q, want 198.51.100.178", ip)
	}
}

func TestClientIP_RemoteAddr(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "172.16.0.5:54321"

	ip := clientIP(req)
	if ip != "172.16.0.5" {
		t.Errorf("clientIP = %q, want 172.16.0.5", ip)
	}
}
