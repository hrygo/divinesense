// Package tools provides unit tests for tool result caching.
package tools

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCacheKey(t *testing.T) {
	key := NewCacheKey("schedule_query", 123, `{"start":"2026-01-01"}`)

	assert.Equal(t, "schedule_query", key.ToolName)
	assert.Equal(t, int32(123), key.UserID)
	assert.NotEmpty(t, key.InputHash)
	assert.Contains(t, key.String(), "tool:schedule_query:user:123:hash:")
}

func TestCacheKey_Consistency(t *testing.T) {
	input := `{"start":"2026-01-01","end":"2026-01-02"}`
	key1 := NewCacheKey("test", 1, input)
	key2 := NewCacheKey("test", 1, input)

	// Same input should produce same key
	assert.Equal(t, key1.InputHash, key2.InputHash)
	assert.Equal(t, key1.String(), key2.String())
}

func TestCacheKey_DifferentInputs(t *testing.T) {
	key1 := NewCacheKey("test", 1, `{"start":"2026-01-01"}`)
	key2 := NewCacheKey("test", 1, `{"start":"2026-01-02"}`)

	// Different inputs should produce different hashes
	assert.NotEqual(t, key1.InputHash, key2.InputHash)
}

func TestToolResultCache_SetDefaultTTLs(t *testing.T) {
	cache := NewToolResultCache(100)

	// Check default TTLs are set
	assert.Equal(t, 30*time.Second, cache.GetTTL("schedule_query"))
	assert.Equal(t, 5*time.Minute, cache.GetTTL("memo_search"))
	assert.Equal(t, 1*time.Minute, cache.GetTTL("find_free_time"))

	// Write operations should not be cacheable
	assert.Equal(t, time.Duration(0), cache.GetTTL("schedule_add"))
	assert.Equal(t, time.Duration(0), cache.GetTTL("schedule_update"))
}

func TestToolResultCache_IsCacheable(t *testing.T) {
	cache := NewToolResultCache(100)

	assert.True(t, cache.IsCacheable("schedule_query"))
	assert.True(t, cache.IsCacheable("memo_search"))
	assert.True(t, cache.IsCacheable("find_free_time"))
	assert.False(t, cache.IsCacheable("schedule_add"))
	assert.False(t, cache.IsCacheable("schedule_update"))
	assert.False(t, cache.IsCacheable("unknown_tool"))
}

func TestToolResultCache_SetTTL(t *testing.T) {
	cache := NewToolResultCache(100)

	// Override default TTL
	cache.SetTTL("schedule_query", 60*time.Second)
	assert.Equal(t, 60*time.Second, cache.GetTTL("schedule_query"))

	// Set new tool TTL
	cache.SetTTL("new_tool", 10*time.Second)
	assert.Equal(t, 10*time.Second, cache.GetTTL("new_tool"))
}

func TestToolResultCache_GetSet(t *testing.T) {
	cache := NewToolResultCache(100)

	key := NewCacheKey("schedule_query", 123, `{"start":"2026-01-01"}`)
	output := "Found 2 schedules"

	// Initially should miss
	result, found := cache.Get(key)
	assert.False(t, found)
	assert.Nil(t, result)

	// Set and then hit
	cache.Set(key, output)
	result, found = cache.Get(key)
	assert.True(t, found)
	assert.Equal(t, output, result.Output)
}

func TestToolResultCache_Expiration(t *testing.T) {
	cache := NewToolResultCache(100)
	cache.SetTTL("test_tool", 100*time.Millisecond)

	key := NewCacheKey("test_tool", 123, `{"test":"input"}`)
	output := "test result"

	cache.Set(key, output)

	// Should hit immediately
	result, found := cache.Get(key)
	assert.True(t, found)
	assert.Equal(t, output, result.Output)

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	// Should miss after expiration
	result, found = cache.Get(key)
	assert.False(t, found)
	assert.Nil(t, result)
}

func TestToolResultCache_NotCacheable(t *testing.T) {
	cache := NewToolResultCache(100)

	key := NewCacheKey("schedule_add", 123, `{"title":"test"}`)
	output := "Schedule created"

	// Set should not cache (TTL = 0)
	cache.Set(key, output)

	// Should still miss
	result, found := cache.Get(key)
	assert.False(t, found)
	assert.Nil(t, result)
}

func TestToolResultCache_Statistics(t *testing.T) {
	cache := NewToolResultCache(100)

	key := NewCacheKey("schedule_query", 123, `{"test":"input"}`)
	output := "test result"

	// Miss
	cache.Get(key)

	// Set and hit
	cache.Set(key, output)
	cache.Get(key)

	// Check stats
	stats := cache.GetStats()
	var scheduleStats *CacheStats
	for _, s := range stats {
		if s.ToolName == "schedule_query" {
			scheduleStats = &s
			break
		}
	}

	require.NotNil(t, scheduleStats)
	assert.Equal(t, int64(1), scheduleStats.Hits)
	assert.Equal(t, int64(1), scheduleStats.Misses)
	assert.InDelta(t, 0.5, scheduleStats.HitRate, 0.01)
}

func TestToolResultCache_EnableDisable(t *testing.T) {
	cache := NewToolResultCache(100)

	assert.True(t, cache.IsEnabled())

	// Disable
	cache.Disable()
	assert.False(t, cache.IsEnabled())

	// Should not cache when disabled
	key := NewCacheKey("schedule_query", 123, `{"test":"input"}`)
	cache.Set(key, "output")
	result, found := cache.Get(key)
	assert.False(t, found)
	assert.Nil(t, result)

	// Enable
	cache.Enable()
	assert.True(t, cache.IsEnabled())

	// Should cache when enabled
	cache.Set(key, "output")
	result, found = cache.Get(key)
	assert.True(t, found)
	assert.Equal(t, "output", result.Output)
}

func TestToolResultCache_Clear(t *testing.T) {
	cache := NewToolResultCache(100)

	key := NewCacheKey("schedule_query", 123, `{"test":"input"}`)
	cache.Set(key, "output")

	// Verify cached - records a hit
	_, found := cache.Get(key)
	assert.True(t, found)

	// Clear
	cache.Clear()

	// Stats should be reset immediately after Clear
	stats := cache.GetStats()
	for _, s := range stats {
		assert.Equal(t, int64(0), s.Hits)
		assert.Equal(t, int64(0), s.Misses)
	}

	// Should miss after clear - records a miss
	_, found = cache.Get(key)
	assert.False(t, found)
}

func TestToolResultCache_Invalidate(t *testing.T) {
	cache := NewToolResultCache(100)

	userID := int32(123)

	// Add multiple cache entries for same user
	cache.Set(NewCacheKey("schedule_query", userID, `{"input":"1"}`), "result1")
	cache.Set(NewCacheKey("schedule_query", userID, `{"input":"2"}`), "result2")
	cache.Set(NewCacheKey("memo_search", userID, `{"input":"3"}`), "result3")

	// Verify they exist
	_, found := cache.Get(NewCacheKey("schedule_query", userID, `{"input":"1"}`))
	assert.True(t, found)

	// Invalidate schedule_query for this user
	removed := cache.Invalidate("schedule_query", userID)
	assert.GreaterOrEqual(t, removed, 1)

	// Should miss for schedule_query but hit for memo_search
	_, found = cache.Get(NewCacheKey("schedule_query", userID, `{"input":"1"}`))
	assert.False(t, found)

	_, found = cache.Get(NewCacheKey("memo_search", userID, `{"input":"3"}`))
	assert.True(t, found)
}

func TestToolResultCache_Size(t *testing.T) {
	cache := NewToolResultCache(10)

	assert.Equal(t, 0, cache.Size())

	// Set TTL for test tool (not cacheable by default)
	cache.SetTTL("test", 5*time.Minute)

	cache.Set(NewCacheKey("test", 1, "a"), "result1")
	assert.Equal(t, 1, cache.Size())

	cache.Set(NewCacheKey("test", 1, "b"), "result2")
	assert.Equal(t, 2, cache.Size())

	cache.Clear()
	assert.Equal(t, 0, cache.Size())
}

func TestToolResultCache_LRUEviction(t *testing.T) {
	cache := NewToolResultCache(3) // Small cache size

	// Set TTL for test tool (not cacheable by default)
	cache.SetTTL("test", 5*time.Minute)

	// Fill cache to capacity
	cache.Set(NewCacheKey("test", 1, "a"), "result1")
	cache.Set(NewCacheKey("test", 1, "b"), "result2")
	cache.Set(NewCacheKey("test", 1, "c"), "result3")

	// All should be present
	_, found := cache.Get(NewCacheKey("test", 1, "a"))
	assert.True(t, found)

	// "b" is still present (middle of LRU list)
	_, found = cache.Get(NewCacheKey("test", 1, "b"))
	assert.True(t, found)

	// Add one more - should evict the oldest ("c" since "b" was just accessed, moving "c" to back)
	// Actually: Get("a") → list: [a, c, b]; Get("b") → list: [b, a, c]; Set("d") → list: [d, b, a] → evicts "c"
	cache.Set(NewCacheKey("test", 1, "d"), "result4")

	// "c" should be evicted (was at the back)
	_, found = cache.Get(NewCacheKey("test", 1, "c"))
	assert.False(t, found)

	// "d" should be present
	_, found = cache.Get(NewCacheKey("test", 1, "d"))
	assert.True(t, found)

	// "a" and "b" should still be present
	_, found = cache.Get(NewCacheKey("test", 1, "a"))
	assert.True(t, found)

	_, found = cache.Get(NewCacheKey("test", 1, "b"))
	assert.True(t, found)
}
