package cache

import (
	"context"
	"testing"
)

func TestNilCache_GetReturnsFalse(t *testing.T) {
	t.Parallel()
	var c *Cache
	var dest string
	if c.Get(context.Background(), "any-key", &dest) {
		t.Error("nil cache Get should return false")
	}
}

func TestNilCache_SetDoesNotPanic(t *testing.T) {
	t.Parallel()
	var c *Cache
	c.Set(context.Background(), "key", "value", 0)
}

func TestNilCache_DeleteDoesNotPanic(t *testing.T) {
	t.Parallel()
	var c *Cache
	err := c.Delete(context.Background(), "key1", "key2")
	if err != nil {
		t.Errorf("nil cache Delete should return nil, got %v", err)
	}
}

func TestNilCache_DeletePatternDoesNotPanic(t *testing.T) {
	t.Parallel()
	var c *Cache
	err := c.DeletePattern(context.Background(), "videos:*")
	if err != nil {
		t.Errorf("nil cache DeletePattern should return nil, got %v", err)
	}
}

func TestNilCache_PingReturnsError(t *testing.T) {
	t.Parallel()
	var c *Cache
	if err := c.Ping(context.Background()); err == nil {
		t.Error("nil cache Ping should return error")
	}
}

func TestNilCache_CloseReturnsNil(t *testing.T) {
	t.Parallel()
	var c *Cache
	if err := c.Close(); err != nil {
		t.Errorf("nil cache Close should return nil, got %v", err)
	}
}

func TestNilCache_RedisClientReturnsNil(t *testing.T) {
	t.Parallel()
	var c *Cache
	if c.RedisClient() != nil {
		t.Error("nil cache RedisClient should return nil")
	}
}

func TestNew_EmptyAddr_ReturnsNil(t *testing.T) {
	t.Parallel()
	c := New("", "", 0)
	if c != nil {
		t.Error("New with empty addr should return nil")
	}
}

func TestNew_WithAddr_ReturnsNonNil(t *testing.T) {
	t.Parallel()
	c := New("localhost:6379", "", 0)
	if c == nil {
		t.Fatal("New with addr should return non-nil")
	}
	// Don't ping — Redis may not be running in unit tests
	_ = c.Close()
}

func TestVideoKey(t *testing.T) {
	t.Parallel()
	key := VideoKey("abc-123")
	if key != "video:abc-123" {
		t.Errorf("VideoKey = %q, want video:abc-123", key)
	}
}

func TestListKey(t *testing.T) {
	t.Parallel()
	key := ListKey(20, 40)
	if key != "videos:list:20:40" {
		t.Errorf("ListKey = %q, want videos:list:20:40", key)
	}
}
