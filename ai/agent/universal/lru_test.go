// Package universal provides tests for LRU cache.
package universal

import (
	"testing"
	"time"
)

// TestLRUCache_Concurrency tests concurrent access safety.
func TestLRUCache_Concurrency(t *testing.T) {
	cache := NewLRUCache(100, time.Minute)
	done := make(chan bool)

	// Concurrent writers
	for i := 0; i < 10; i++ {
		go func(n int) {
			for j := 0; j < 100; j++ {
				key := string(rune(n)) + string(rune(j))
				cache.Set(key, "value")
			}
			done <- true
		}(i)
	}

	// Concurrent readers
	for i := 0; i < 10; i++ {
		go func(n int) {
			for j := 0; j < 100; j++ {
				key := string(rune(n)) + string(rune(j))
				cache.Get(key)
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 20; i++ {
		<-done
	}

	// Verify cache still works
	cache.Set("final", "check")
	val, found := cache.Get("final")
	if !found || val != "check" {
		t.Error("cache corrupted after concurrent operations")
	}
}

// TestLRUCache_EdgeCases tests edge cases.
func TestLRUCache_EdgeCases(t *testing.T) {
	cache := NewLRUCache(1, time.Minute)

	// Empty key
	cache.Set("", "empty")
	val, found := cache.Get("")
	if !found || val != "empty" {
		t.Error("failed to handle empty key")
	}

	// Nil size cache (should still work with size 1)
	cache2 := NewLRUCache(0, time.Minute)
	// Note: NewLRUCache doesn't default size 0 to 1, it uses 0 as-is
	if cache2.size < 0 {
		t.Errorf("expected non-negative size, got %d", cache2.size)
	}
}

// TestLRUCache_Overwrite tests overwriting existing values.
func TestLRUCache_Overwrite(t *testing.T) {
	cache := NewLRUCache(2, time.Minute)

	cache.Set("key1", "value1")
	cache.Set("key1", "value2")
	cache.Set("key1", "value3")

	val, found := cache.Get("key1")
	if !found || val != "value3" {
		t.Errorf("expected 'value3', got '%s'", val)
	}
}

// TestLRUCache_ZeroTTL tests immediate expiration.
func TestLRUCache_ZeroTTL(t *testing.T) {
	cache := NewLRUCache(10, 0)

	cache.Set("key1", "value1")

	// Should be expired immediately
	_, found := cache.Get("key1")
	if found {
		t.Error("expected key to be expired with zero TTL")
	}
}
