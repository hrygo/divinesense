// Package cache provides unit tests for LRU cache implementation.
package cache

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLRUCache_Creation tests cache creation with various configurations.
func TestLRUCache_Creation(t *testing.T) {
	testCases := []struct {
		name       string
		capacity   int
		defaultTTL time.Duration
		expectCap  int
		expectTTL  time.Duration
	}{
		{"default values", 0, 0, 1000, 5 * time.Minute},
		{"custom capacity", 500, 0, 500, 5 * time.Minute},
		{"custom TTL", 0, 10 * time.Minute, 1000, 10 * time.Minute},
		{"both custom", 200, 15 * time.Minute, 200, 15 * time.Minute},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cache := NewLRUCache(tc.capacity, tc.defaultTTL)
			assert.Equal(t, tc.expectCap, cache.Capacity())
			assert.Equal(t, 0, cache.Size())
		})
	}
}

// TestLRUCache_BasicSetGet tests basic Set and Get operations.
func TestLRUCache_BasicSetGet(t *testing.T) {
	cache := NewLRUCache(100, time.Minute)

	t.Run("Set and Get returns value", func(t *testing.T) {
		key := "test-key"
		value := []byte("test-value")

		cache.Set(key, value, 0)
		result, ok := cache.Get(key)

		require.True(t, ok, "expected key to exist")
		assert.Equal(t, value, result)
	})

	t.Run("Get non-existent key returns false", func(t *testing.T) {
		_, ok := cache.Get("non-existent")
		assert.False(t, ok)
	})

	t.Run("Set with TTL", func(t *testing.T) {
		key := "ttl-key"
		value := []byte("ttl-value")

		cache.Set(key, value, 100*time.Millisecond)
		result, ok := cache.Get(key)

		require.True(t, ok)
		assert.Equal(t, value, result)
	})

	t.Run("Update existing key", func(t *testing.T) {
		key := "update-key"
		value1 := []byte("value1")
		value2 := []byte("value2")

		cache.Set(key, value1, 0)
		cache.Set(key, value2, 0)

		result, ok := cache.Get(key)
		require.True(t, ok)
		assert.Equal(t, value2, result)
	})
}

// TestLRUCache_TTLExpiration tests TTL-based expiration.
func TestLRUCache_TTLExpiration(t *testing.T) {
	cache := NewLRUCache(100, 50*time.Millisecond)

	t.Run("value expires after TTL", func(t *testing.T) {
		key := "expiring-key"
		value := []byte("expiring-value")

		cache.Set(key, value, 50*time.Millisecond)

		// Should exist immediately
		_, ok := cache.Get(key)
		assert.True(t, ok, "key should exist immediately after Set")

		// Wait for expiration
		time.Sleep(60 * time.Millisecond)

		_, ok = cache.Get(key)
		assert.False(t, ok, "key should be expired after TTL")
	})

	t.Run("custom TTL overrides default", func(t *testing.T) {
		cache := NewLRUCache(100, 10*time.Millisecond)

		// Set with longer TTL
		cache.Set("long", []byte("long"), 100*time.Millisecond)

		// Default TTL expires
		time.Sleep(20 * time.Millisecond)

		// Long TTL key should still exist
		_, ok := cache.Get("long")
		assert.True(t, ok, "key with custom TTL should persist after default TTL")
	})
}

// TestLRUCache_LRUEviction tests LRU eviction policy.
func TestLRUCache_LRUEviction(t *testing.T) {
	cache := NewLRUCache(3, time.Minute)

	t.Run("evicts least recently used when full", func(t *testing.T) {
		// Fill cache
		cache.Set("key1", []byte("1"), 0)
		cache.Set("key2", []byte("2"), 0)
		cache.Set("key3", []byte("3"), 0)

		assert.Equal(t, 3, cache.Size(), "cache should be at capacity")

		// Access key1 to make it recently used
		cache.Get("key1")

		// Add new entry - should evict key2 (LRU)
		cache.Set("key4", []byte("4"), 0)

		assert.Equal(t, 3, cache.Size(), "cache size should remain at capacity")

		// key2 should be evicted
		_, ok := cache.Get("key2")
		assert.False(t, ok, "LRU key should be evicted")

		// key1 should still exist (was accessed)
		_, ok = cache.Get("key1")
		assert.True(t, ok, "recently accessed key should exist")
	})

	t.Run("eviction respects update time", func(t *testing.T) {
		cache := NewLRUCache(3, time.Minute)

		cache.Set("key1", []byte("1"), 0)
		cache.Set("key2", []byte("2"), 0)
		cache.Set("key3", []byte("3"), 0)

		// Update key2 to make it more recent
		cache.Set("key2", []byte("2-updated"), 0)

		// Add new entry - should evict key1 (oldest)
		cache.Set("key4", []byte("4"), 0)

		_, ok := cache.Get("key1")
		assert.False(t, ok, "oldest key should be evicted")

		_, ok = cache.Get("key2")
		assert.True(t, ok, "updated key should exist")
	})
}

// TestLRUCache_Invalidation tests cache invalidation.
func TestLRUCache_Invalidation(t *testing.T) {
	cache := NewLRUCache(100, time.Minute)

	t.Run("invalidate exact key", func(t *testing.T) {
		cache.Set("user:1", []byte("1"), 0)
		cache.Set("user:2", []byte("2"), 0)

		count := cache.Invalidate("user:1")
		assert.Equal(t, 1, count, "should invalidate 1 entry")

		_, ok := cache.Get("user:1")
		assert.False(t, ok, "invalidated key should not exist")

		_, ok = cache.Get("user:2")
		assert.True(t, ok, "other keys should remain")
	})

	t.Run("invalidate with wildcard pattern", func(t *testing.T) {
		cache.Set("user:1:profile", []byte("1"), 0)
		cache.Set("user:1:settings", []byte("2"), 0)
		cache.Set("user:2:profile", []byte("3"), 0)
		cache.Set("user:2:settings", []byte("4"), 0)

		count := cache.Invalidate("user:1:*")
		assert.Equal(t, 2, count, "should invalidate 2 entries")

		_, ok := cache.Get("user:1:profile")
		assert.False(t, ok)

		_, ok = cache.Get("user:1:settings")
		assert.False(t, ok)

		_, ok = cache.Get("user:2:profile")
		assert.True(t, ok, "entries not matching pattern should remain")
	})

	t.Run("invalidate non-existent key returns 0", func(t *testing.T) {
		count := cache.Invalidate("non-existent")
		assert.Equal(t, 0, count)
	})
}

// TestLRUCache_Clearing tests clearing the cache.
func TestLRUCache_Clearing(t *testing.T) {
	cache := NewLRUCache(100, time.Minute)

	// Add entries
	for i := 0; i < 10; i++ {
		cache.Set(string(rune('a'+i)), []byte{byte(i)}, 0)
	}

	assert.Equal(t, 10, cache.Size())

	cache.Clear()

	assert.Equal(t, 0, cache.Size(), "cache should be empty after Clear")

	// Verify all keys are gone
	for i := 0; i < 10; i++ {
		_, ok := cache.Get(string(rune('a' + i)))
		assert.False(t, ok, "all entries should be cleared")
	}
}

// TestLRUCache_ExpiredCleanup tests cleanup of expired entries.
func TestLRUCache_ExpiredCleanup(t *testing.T) {
	cache := NewLRUCache(100, 50*time.Millisecond)

	// Add entries with different TTLs
	cache.Set("expired1", []byte("1"), 50*time.Millisecond)
	cache.Set("expired2", []byte("2"), 50*time.Millisecond)
	cache.Set("valid", []byte("3"), 200*time.Millisecond)
	cache.Set("long", []byte("4"), 300*time.Millisecond) // Explicitly set long TTL

	// Wait for short TTL entries to expire
	time.Sleep(60 * time.Millisecond)

	removed := cache.CleanupExpired()
	assert.GreaterOrEqual(t, removed, 2, "should remove at least 2 expired entries")

	// Verify expired entries are gone
	_, ok := cache.Get("expired1")
	assert.False(t, ok)

	_, ok = cache.Get("expired2")
	assert.False(t, ok)

	// Verify valid entries remain
	_, ok = cache.Get("valid")
	assert.True(t, ok)

	_, ok = cache.Get("long")
	assert.True(t, ok)
}

// TestLRUCache_ThreadSafety tests thread safety.
func TestLRUCache_ThreadSafety(t *testing.T) {
	cache := NewLRUCache(1000, time.Minute)
	var wg sync.WaitGroup

	// Concurrent writers
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			key := string(rune('a' + n%26))
			cache.Set(key, []byte{byte(n)}, 0)
		}(i)
	}

	// Concurrent readers
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			key := string(rune('a' + n%26))
			cache.Get(key)
		}(i)
	}

	// Concurrent deleters
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			cache.Invalidate("user:*")
		}(i)
	}

	wg.Wait()
	// Should not panic or deadlock
}

// TestLRUCache_SizeMethod tests Size method.
func TestLRUCache_SizeMethod(t *testing.T) {
	cache := NewLRUCache(100, time.Minute)

	assert.Equal(t, 0, cache.Size(), "new cache should be empty")

	for i := 0; i < 10; i++ {
		cache.Set(string(rune('a'+i)), []byte{byte(i)}, 0)
	}

	assert.Equal(t, 10, cache.Size())
}

// TestLRUCache_CapacityMethod tests Capacity method.
func TestLRUCache_CapacityMethod(t *testing.T) {
	testCases := []int{10, 100, 1000}

	for _, cap := range testCases {
		cache := NewLRUCache(cap, time.Minute)
		assert.Equal(t, cap, cache.Capacity())
	}
}

// TestLRUCache_ZeroCapacityHandling tests behavior with zero capacity.
func TestLRUCache_ZeroCapacityHandling(t *testing.T) {
	cache := NewLRUCache(0, time.Minute)

	cache.Set("key", []byte("value"), 0)
	_, ok := cache.Get("key")

	// With zero capacity (defaulted to 1000), should work
	assert.True(t, ok, "cache with default capacity should store values")
}

// TestLRUCache_GetPromotion tests that Get promotes entry to front.
func TestLRUCache_GetPromotion(t *testing.T) {
	cache := NewLRUCache(3, time.Minute)

	// Fill cache
	cache.Set("key1", []byte("1"), 0)
	cache.Set("key2", []byte("2"), 0)
	cache.Set("key3", []byte("3"), 0)

	// Access key1 to promote it
	cache.Get("key1")

	// Add new entry - should evict key2 (not key1)
	cache.Set("key4", []byte("4"), 0)

	_, ok := cache.Get("key1")
	assert.True(t, ok, "promoted entry should exist")

	_, ok = cache.Get("key2")
	assert.False(t, ok, "LRU entry should be evicted")
}

// BenchmarkLRUCache_Set benchmarks Set operation.
func BenchmarkLRUCache_Set(b *testing.B) {
	cache := NewLRUCache(10000, time.Minute)
	value := []byte("test-value")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := string(rune('a' + i%26))
		cache.Set(key, value, 0)
	}
}

// BenchmarkLRUCache_Get benchmarks Get operation.
func BenchmarkLRUCache_Get(b *testing.B) {
	cache := NewLRUCache(10000, time.Minute)
	cache.Set("test-key", []byte("test-value"), 0)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Get("test-key")
	}
}

// BenchmarkLRUCache_SetAndEvict benchmarks Set with eviction.
func BenchmarkLRUCache_SetAndEvict(b *testing.B) {
	cache := NewLRUCache(100, time.Minute)
	value := []byte("test-value")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := string(rune('a' + i%100))
		cache.Set(key, value, 0)
	}
}
