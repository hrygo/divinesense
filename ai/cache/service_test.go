package cache

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLRUCache_BasicOperations(t *testing.T) {
	cache := NewLRUCache(100, time.Minute)

	t.Run("SetAndGet", func(t *testing.T) {
		cache.Set("key1", []byte("value1"), 0)

		val, ok := cache.Get("key1")
		assert.True(t, ok)
		assert.Equal(t, []byte("value1"), val)
	})

	t.Run("GetNonExistent", func(t *testing.T) {
		val, ok := cache.Get("nonexistent")
		assert.False(t, ok)
		assert.Nil(t, val)
	})

	t.Run("UpdateExisting", func(t *testing.T) {
		cache.Set("key2", []byte("original"), 0)
		cache.Set("key2", []byte("updated"), 0)

		val, ok := cache.Get("key2")
		assert.True(t, ok)
		assert.Equal(t, []byte("updated"), val)
	})
}

func TestLRUCache_Expiration(t *testing.T) {
	cache := NewLRUCache(100, 50*time.Millisecond)

	cache.Set("expiring", []byte("value"), 50*time.Millisecond)

	// Should exist immediately
	val, ok := cache.Get("expiring")
	assert.True(t, ok)
	assert.Equal(t, []byte("value"), val)

	// Wait for expiration
	time.Sleep(60 * time.Millisecond)

	// Should be expired
	val, ok = cache.Get("expiring")
	assert.False(t, ok)
	assert.Nil(t, val)
}

func TestLRUCache_Eviction(t *testing.T) {
	cache := NewLRUCache(3, time.Minute)

	// Fill cache
	cache.Set("key1", []byte("1"), 0)
	cache.Set("key2", []byte("2"), 0)
	cache.Set("key3", []byte("3"), 0)
	assert.Equal(t, 3, cache.Size())

	// Access key1 to make it recently used
	cache.Get("key1")

	// Add new entry, should evict key2 (LRU)
	cache.Set("key4", []byte("4"), 0)
	assert.Equal(t, 3, cache.Size())

	// key2 should be evicted
	_, ok := cache.Get("key2")
	assert.False(t, ok)

	// key1 should still exist
	_, ok = cache.Get("key1")
	assert.True(t, ok)
}

func TestLRUCache_Invalidate(t *testing.T) {
	cache := NewLRUCache(100, time.Minute)

	t.Run("ExactMatch", func(t *testing.T) {
		cache.Set("user:1", []byte("1"), 0)
		cache.Set("user:2", []byte("2"), 0)

		count := cache.Invalidate("user:1")
		assert.Equal(t, 1, count)

		_, ok := cache.Get("user:1")
		assert.False(t, ok)

		_, ok = cache.Get("user:2")
		assert.True(t, ok)
	})

	t.Run("WildcardPattern", func(t *testing.T) {
		cache.Clear()
		cache.Set("user:1:profile", []byte("1"), 0)
		cache.Set("user:1:settings", []byte("2"), 0)
		cache.Set("user:2:profile", []byte("3"), 0)

		count := cache.Invalidate("user:1:*")
		assert.Equal(t, 2, count)

		_, ok := cache.Get("user:1:profile")
		assert.False(t, ok)

		_, ok = cache.Get("user:2:profile")
		assert.True(t, ok)
	})
}

func TestLRUCache_ConcurrentAccess(t *testing.T) {
	cache := NewLRUCache(1000, time.Minute)
	var wg sync.WaitGroup

	// Concurrent writes
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			key := string(rune('a' + n%26))
			cache.Set(key, []byte{byte(n)}, 0)
		}(i)
	}

	// Concurrent reads
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			key := string(rune('a' + n%26))
			cache.Get(key)
		}(i)
	}

	wg.Wait()
	// Should not panic
}

func TestService_BasicOperations(t *testing.T) {
	svc := NewService(ServiceConfig{
		Capacity:        100,
		DefaultTTL:      time.Minute,
		CleanupInterval: time.Hour, // Disable auto cleanup for tests
	})
	defer svc.Close()

	ctx := context.Background()

	t.Run("SetAndGet", func(t *testing.T) {
		err := svc.Set(ctx, "key1", []byte("value1"), 0)
		require.NoError(t, err)

		val, ok := svc.Get(ctx, "key1")
		assert.True(t, ok)
		assert.Equal(t, []byte("value1"), val)
	})

	t.Run("Invalidate", func(t *testing.T) {
		err := svc.Set(ctx, "user:1:data", []byte("data"), 0)
		require.NoError(t, err)

		err = svc.Invalidate(ctx, "user:1:*")
		require.NoError(t, err)

		_, ok := svc.Get(ctx, "user:1:data")
		assert.False(t, ok)
	})
}

func TestService_Close(t *testing.T) {
	svc := NewService(DefaultServiceConfig())

	// Should not panic
	svc.Close()
}

func TestService_CleanupExpired(t *testing.T) {
	svc := NewService(ServiceConfig{
		Capacity:        100,
		DefaultTTL:      50 * time.Millisecond,
		CleanupInterval: 30 * time.Millisecond,
	})
	defer svc.Close()

	ctx := context.Background()
	_ = svc.Set(ctx, "temp", []byte("data"), 50*time.Millisecond)

	assert.Equal(t, 1, svc.Size())

	// Wait for cleanup
	time.Sleep(100 * time.Millisecond)

	assert.Equal(t, 0, svc.Size())
}
