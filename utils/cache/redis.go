// Package cache provides a Redis-backed caching layer for read-heavy data.
package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// Cache wraps a Redis client for application-level caching.
type Cache struct {
	client *redis.Client
}

// New creates a new Redis cache. Returns nil (no-op cache) if addr is empty.
func New(addr, password string, db int) *Cache {
	if addr == "" {
		return nil
	}
	client := redis.NewClient(&redis.Options{
		Addr:         addr,
		Password:     password,
		DB:           db,
		PoolSize:     20,
		MinIdleConns: 5,
		ReadTimeout:  500 * time.Millisecond,
		WriteTimeout: 500 * time.Millisecond,
	})
	return &Cache{client: client}
}

// Get retrieves a cached value and unmarshals it into dest.
// Returns false if the key is missing or on any error (cache miss).
func (c *Cache) Get(ctx context.Context, key string, dest any) bool {
	if c == nil {
		return false
	}
	val, err := c.client.Get(ctx, key).Bytes()
	if err != nil {
		return false
	}
	return json.Unmarshal(val, dest) == nil
}

// Set marshals value and stores it with the given TTL. Errors are silently ignored
// (cache is best-effort).
func (c *Cache) Set(ctx context.Context, key string, value any, ttl time.Duration) {
	if c == nil {
		return
	}
	data, err := json.Marshal(value)
	if err != nil {
		return
	}
	_ = c.client.Set(ctx, key, data, ttl).Err()
}

// Delete removes one or more keys.
func (c *Cache) Delete(ctx context.Context, keys ...string) error {
	if c == nil {
		return nil
	}
	return c.client.Del(ctx, keys...).Err()
}

// DeletePattern removes all keys matching a glob pattern (e.g. "videos:list:*").
func (c *Cache) DeletePattern(ctx context.Context, pattern string) error {
	if c == nil {
		return nil
	}
	var firstErr error
	iter := c.client.Scan(ctx, 0, pattern, 100).Iterator()
	for iter.Next(ctx) {
		if err := c.client.Del(ctx, iter.Val()).Err(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	if err := iter.Err(); err != nil && firstErr == nil {
		firstErr = err
	}
	return firstErr
}

// RedisClient returns the underlying Redis client, or nil if cache is disabled.
// Used to share the connection with other Redis-backed subsystems (e.g. rate limiter).
func (c *Cache) RedisClient() *redis.Client {
	if c == nil {
		return nil
	}
	return c.client
}

// Ping checks the Redis connection.
func (c *Cache) Ping(ctx context.Context) error {
	if c == nil {
		return fmt.Errorf("cache is disabled (no Redis address configured)")
	}
	return c.client.Ping(ctx).Err()
}

// Close closes the Redis connection.
func (c *Cache) Close() error {
	if c == nil {
		return nil
	}
	return c.client.Close()
}

// VideoKey returns the cache key for a single video.
func VideoKey(id string) string {
	return "video:" + id
}

// ListKey returns the cache key for a paginated video list.
func ListKey(limit, offset int) string {
	return fmt.Sprintf("videos:list:%d:%d", limit, offset)
}
