// Package universal provides tests for LRU cache integration.
package universal

import (
	"testing"
	"time"

	"github.com/hrygo/divinesense/ai/cache"
)

// TestLRUCache_Concurrency tests concurrent access safety.
func TestLRUCache_Concurrency(t *testing.T) {
	lru := cache.NewStringLRUCache(100, time.Minute)
	done := make(chan bool)

	// Concurrent writers
	for i := 0; i < 10; i++ {
		go func(n int) {
			for j := 0; j < 100; j++ {
				key := string(rune(n)) + string(rune(j))
				lru.SetWithDefaultTTL(key, "value")
			}
			done <- true
		}(i)
	}

	// Concurrent readers
	for i := 0; i < 10; i++ {
		go func(n int) {
			for j := 0; j < 100; j++ {
				key := string(rune(n)) + string(rune(j))
				lru.Get(key)
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 20; i++ {
		<-done
	}

	// Verify cache still works
	lru.SetWithDefaultTTL("final", "check")
	val, found := lru.Get("final")
	if !found || val != "check" {
		t.Error("cache corrupted after concurrent operations")
	}
}

// TestLRUCache_EdgeCases tests edge cases.
func TestLRUCache_EdgeCases(t *testing.T) {
	lru := cache.NewStringLRUCache(1, time.Minute)

	// Empty key
	lru.SetWithDefaultTTL("", "empty")
	val, found := lru.Get("")
	if !found || val != "empty" {
		t.Error("failed to handle empty key")
	}

	// Zero size cache (should default to 1000)
	lru2 := cache.NewStringLRUCache(0, time.Minute)
	if lru2.Capacity() != 1000 {
		t.Errorf("expected default capacity 1000, got %d", lru2.Capacity())
	}
}

// TestLRUCache_Overwrite tests overwriting existing values.
func TestLRUCache_Overwrite(t *testing.T) {
	lru := cache.NewStringLRUCache(2, time.Minute)

	lru.SetWithDefaultTTL("key1", "value1")
	lru.SetWithDefaultTTL("key1", "value2")
	lru.SetWithDefaultTTL("key1", "value3")

	val, found := lru.Get("key1")
	if !found || val != "value3" {
		t.Errorf("expected 'value3', got '%s'", val)
	}
}

// TestLRUCache_TTL tests TTL expiration.
func TestLRUCache_TTL(t *testing.T) {
	lru := cache.NewStringLRUCache(10, 50*time.Millisecond)

	lru.SetWithDefaultTTL("key1", "value1")

	// Should be found immediately
	val, found := lru.Get("key1")
	if !found || val != "value1" {
		t.Error("expected key to be found")
	}

	// Wait for expiration
	time.Sleep(100 * time.Millisecond)

	// Should be expired
	_, found = lru.Get("key1")
	if found {
		t.Error("expected key to be expired")
	}
}
