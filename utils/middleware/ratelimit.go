package middleware

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// RateLimiter implements a per-IP sliding window rate limiter.
// When Redis is available, counters are shared across all pods.
// Falls back to in-memory counters if Redis is nil or unreachable.
type RateLimiter struct {
	redis    *redis.Client
	rate     int           // max requests per window
	window   time.Duration // window size
	burst    int           // burst capacity (used only for in-memory fallback)

	// In-memory fallback when Redis is unavailable
	mu       sync.Mutex
	visitors map[string]*bucket
}

type bucket struct {
	tokens   int
	lastSeen time.Time
}

// NewRateLimiter creates a distributed rate limiter backed by Redis.
// If redisClient is nil, falls back to in-memory (per-pod) limiting.
// Example: NewRateLimiter(redisClient, 100, time.Minute, 150)
// allows 100 requests per minute with a burst of 150 (in-memory fallback).
func NewRateLimiter(redisClient *redis.Client, rate int, window time.Duration, burst int) *RateLimiter {
	rl := &RateLimiter{
		redis:    redisClient,
		rate:     rate,
		window:   window,
		burst:    burst,
		visitors: make(map[string]*bucket),
	}
	go rl.cleanup()
	return rl
}

// allow checks if the IP is allowed to make a request.
// Returns (allowed, remaining requests, seconds until reset).
func (rl *RateLimiter) allow(ctx context.Context, ip string) (bool, int, int) {
	if rl.redis != nil {
		allowed, remaining, retryAfter, err := rl.allowRedis(ctx, ip)
		if err == nil {
			return allowed, remaining, retryAfter
		}
		// Redis error — fall through to in-memory fallback
	}
	return rl.allowLocal(ip)
}

// allowRedis uses Redis INCR + EXPIRE for a sliding window counter.
// This is the same algorithm used by Stripe, GitHub, and Cloudflare.
//
// How it works:
//   - Key: ratelimit:{ip}:{window_start}
//   - INCR the key on each request
//   - SET TTL = window duration (so keys auto-expire)
//   - If count > rate limit → reject
func (rl *RateLimiter) allowRedis(ctx context.Context, ip string) (bool, int, int, error) {
	windowSeconds := int(rl.window.Seconds())
	now := time.Now().Unix()
	windowStart := now - (now % int64(windowSeconds))
	key := fmt.Sprintf("ratelimit:%s:%d", ip, windowStart)

	count, err := rl.redis.Incr(ctx, key).Result()
	if err != nil {
		return false, 0, 0, err
	}

	// Set TTL only on first request in this window
	if count == 1 {
		rl.redis.Expire(ctx, key, rl.window+time.Second)
	}

	remaining := rl.rate - int(count)
	retryAfter := int(int64(windowSeconds) - (now - windowStart))

	if int(count) > rl.rate {
		return false, 0, retryAfter, nil
	}

	return true, remaining, retryAfter, nil
}

// allowLocal is the in-memory fallback using token bucket (per-pod only).
func (rl *RateLimiter) allowLocal(ip string) (bool, int, int) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	b, exists := rl.visitors[ip]
	now := time.Now()
	retryAfter := int(rl.window.Seconds())

	if !exists {
		rl.visitors[ip] = &bucket{tokens: rl.burst - 1, lastSeen: now}
		return true, rl.burst - 1, retryAfter
	}

	// Refill tokens based on elapsed time
	elapsed := now.Sub(b.lastSeen)
	refill := int(elapsed/rl.window) * rl.rate
	if refill > 0 {
		b.tokens = min(rl.burst, b.tokens+refill)
		b.lastSeen = now
	}

	if b.tokens <= 0 {
		return false, 0, retryAfter
	}

	b.tokens--
	b.lastSeen = now
	return true, b.tokens, retryAfter
}

// cleanup removes stale in-memory entries every 5 minutes.
func (rl *RateLimiter) cleanup() {
	for {
		time.Sleep(5 * time.Minute)
		rl.mu.Lock()
		for ip, b := range rl.visitors {
			if time.Since(b.lastSeen) > 10*time.Minute {
				delete(rl.visitors, ip)
			}
		}
		rl.mu.Unlock()
	}
}

// Middleware returns an HTTP middleware that rate-limits by client IP.
func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := clientIP(r)
		allowed, remaining, retryAfter := rl.allow(r.Context(), ip)

		// Always set rate limit headers (standard draft RFC)
		w.Header().Set("X-RateLimit-Limit", strconv.Itoa(rl.rate))
		w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(remaining))
		w.Header().Set("Retry-After", strconv.Itoa(retryAfter))

		if !allowed {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = fmt.Fprintf(w, `{"type":"RATE_LIMITED","message":"Too many requests","status_code":429}`)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// clientIP extracts the client IP, preferring X-Forwarded-For for proxied requests.
func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// Take the first IP (original client)
		for i := 0; i < len(xff); i++ {
			if xff[i] == ',' {
				return xff[:i]
			}
		}
		return xff
	}
	if xff := r.Header.Get("X-Real-IP"); xff != "" {
		return xff
	}
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}
