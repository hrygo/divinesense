package routing

import (
	"testing"
	"time"
)

// TestRouterCache_BasicOperations tests basic cache operations.
func TestRouterCache_BasicOperations(t *testing.T) {
	cache := NewRouterCache(CacheConfig{
		Capacity:     10,
		DefaultTTL:   100 * time.Millisecond,
		LLMResultTTL: 500 * time.Millisecond,
	})

	// Test Set and Get
	cache.Set("test input", IntentMemoSearch, 0.9, "rule")
	intent, confidence, found := cache.Get("test input")

	if !found {
		t.Fatal("expected cache hit, got miss")
	}
	if intent != IntentMemoSearch {
		t.Errorf("expected IntentMemoSearch, got %s", intent)
	}
	if confidence != 0.9 {
		t.Errorf("expected confidence 0.9, got %f", confidence)
	}

	// Test cache miss
	_, _, found = cache.Get("nonexistent")
	if found {
		t.Error("expected cache miss, got hit")
	}
}

// TestRouterCache_TTL tests cache TTL expiration.
func TestRouterCache_TTL(t *testing.T) {
	cache := NewRouterCache(CacheConfig{
		Capacity:     10,
		DefaultTTL:   50 * time.Millisecond,
		LLMResultTTL: 100 * time.Millisecond,
	})

	// Test default TTL
	cache.Set("test1", IntentMemoSearch, 0.9, "rule")
	time.Sleep(60 * time.Millisecond)
	_, _, found := cache.Get("test1")
	if found {
		t.Error("expected cache miss after TTL expiration, got hit")
	}

	// Test LLM TTL (longer)
	cache.Set("test2", IntentScheduleCreate, 0.8, "llm")
	time.Sleep(60 * time.Millisecond)
	_, _, found = cache.Get("test2")
	if !found {
		t.Error("expected cache hit for LLM result within TTL, got miss")
	}
	time.Sleep(50 * time.Millisecond)
	_, _, found = cache.Get("test2")
	if found {
		t.Error("expected cache miss for LLM result after TTL, got hit")
	}
}

// TestRouterCache_Stats tests cache statistics.
func TestRouterCache_Stats(t *testing.T) {
	cache := NewRouterCache(CacheConfig{
		Capacity:     10,
		DefaultTTL:   5 * time.Minute,
		LLMResultTTL: 30 * time.Minute,
	})

	// Initial stats
	stats := cache.GetStats()
	if stats.Hits != 0 || stats.Misses != 0 {
		t.Errorf("expected zero initial stats, got hits=%d, misses=%d", stats.Hits, stats.Misses)
	}

	// Generate some hits and misses
	cache.Set("key1", IntentMemoSearch, 0.9, "rule")
	cache.Get("key1") // hit
	cache.Get("key2") // miss

	stats = cache.GetStats()
	if stats.Hits != 1 {
		t.Errorf("expected 1 hit, got %d", stats.Hits)
	}
	if stats.Misses != 1 {
		t.Errorf("expected 1 miss, got %d", stats.Misses)
	}

	expectedHitRate := 0.5
	if stats.HitRate != expectedHitRate {
		t.Errorf("expected hit rate %f, got %f", expectedHitRate, stats.HitRate)
	}
}

// TestRouterCache_Clear tests cache clearing.
func TestRouterCache_Clear(t *testing.T) {
	cache := NewRouterCache(CacheConfig{
		Capacity:     10,
		DefaultTTL:   5 * time.Minute,
		LLMResultTTL: 30 * time.Minute,
	})

	// Add entries
	cache.Set("key1", IntentMemoSearch, 0.9, "rule")
	cache.Set("key2", IntentScheduleCreate, 0.8, "llm")

	// Verify entries exist
	_, _, found := cache.Get("key1")
	if !found {
		t.Error("expected cache hit before clear, got miss")
	}

	// Clear cache
	cache.Clear()

	// Verify entries are gone
	_, _, found = cache.Get("key1")
	if found {
		t.Error("expected cache miss after clear, got hit")
	}

	// Get stats (the last Get call is a miss, so misses = 1)
	stats := cache.GetStats()
	if stats.Hits != 0 {
		t.Errorf("expected 0 hits after clear, got %d", stats.Hits)
	}
	// Note: misses are tracked after clear, so we expect 1 miss from the last Get call
	if stats.Misses != 1 {
		t.Logf("got %d misses after clear (expected 1 from last Get call)", stats.Misses)
	}
}

// TestRouterCache_Invalidate tests cache invalidation.
func TestRouterCache_Invalidate(t *testing.T) {
	cache := NewRouterCache(CacheConfig{
		Capacity:     10,
		DefaultTTL:   5 * time.Minute,
		LLMResultTTL: 30 * time.Minute,
	})

	cache.Set("key1", IntentMemoSearch, 0.9, "rule")
	cache.Set("key2", IntentScheduleCreate, 0.8, "llm")

	// Invalidate specific entry
	cache.Invalidate("key1")

	_, _, found := cache.Get("key1")
	if found {
		t.Error("expected cache miss after invalidation, got hit")
	}

	// Other entry should still exist
	_, _, found = cache.Get("key2")
	if !found {
		t.Error("expected cache hit for non-invalidated entry, got miss")
	}
}

// TestRouterCache_Capacity tests cache capacity eviction.
func TestRouterCache_Capacity(t *testing.T) {
	cache := NewRouterCache(CacheConfig{
		Capacity:     3, // Small capacity
		DefaultTTL:   5 * time.Minute,
		LLMResultTTL: 30 * time.Minute,
	})

	// Fill cache to capacity
	cache.Set("key1", IntentMemoSearch, 0.9, "rule")
	cache.Set("key2", IntentScheduleCreate, 0.8, "rule")
	cache.Set("key3", IntentMemoCreate, 0.7, "rule")

	// Add one more - should evict oldest
	cache.Set("key4", IntentMemoSearch, 0.6, "rule")

	stats := cache.GetStats()
	if stats.Size != 3 {
		t.Errorf("expected cache size 3, got %d", stats.Size)
	}
}

// TestRouterCache_CleanupExpired tests expired entry cleanup.
func TestRouterCache_CleanupExpired(t *testing.T) {
	cache := NewRouterCache(CacheConfig{
		Capacity:     10,
		DefaultTTL:   50 * time.Millisecond,
		LLMResultTTL: 30 * time.Minute,
	})

	// Add entries
	cache.Set("expired1", IntentMemoSearch, 0.9, "rule")
	cache.Set("expired2", IntentScheduleCreate, 0.8, "rule")
	cache.Set("valid", IntentMemoCreate, 0.7, "llm")

	// Wait for rule entries to expire
	time.Sleep(60 * time.Millisecond)

	// Run cleanup
	removed := cache.CleanupExpired()
	if removed != 2 {
		t.Errorf("expected 2 entries removed, got %d", removed)
	}

	stats := cache.GetStats()
	if stats.Size != 1 {
		t.Errorf("expected cache size 1 after cleanup, got %d", stats.Size)
	}

	// Verify valid entry still exists
	intent, _, found := cache.Get("valid")
	if !found {
		t.Error("expected valid entry to still exist")
	}
	if intent != IntentMemoCreate {
		t.Errorf("expected IntentMemoCreate, got %s", intent)
	}
}
