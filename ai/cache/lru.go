package cache

import (
	"container/list"
	"strings"
	"sync"
	"time"
)

// LRUCache implements an LRU cache with TTL support and generics.
// LRUCache 实现支持 TTL 和泛型的 LRU 缓存。
type LRUCache[K comparable, V any] struct {
	cache      map[K]*entry[K, V]
	order      *list.List
	capacity   int
	defaultTTL time.Duration
	mu         sync.RWMutex
}

type entry[K comparable, V any] struct {
	expiresAt time.Time
	element   *list.Element
	key       K
	value     V
}

// NewLRUCache creates a new LRU cache.
func NewLRUCache[K comparable, V any](capacity int, defaultTTL time.Duration) *LRUCache[K, V] {
	if capacity <= 0 {
		capacity = 1000
	}
	if defaultTTL <= 0 {
		defaultTTL = 5 * time.Minute
	}

	return &LRUCache[K, V]{
		capacity:   capacity,
		defaultTTL: defaultTTL,
		cache:      make(map[K]*entry[K, V]),
		order:      list.New(),
	}
}

// Get retrieves a value from the cache.
func (c *LRUCache[K, V]) Get(key K) (V, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	e, ok := c.cache[key]
	if !ok {
		var zero V
		return zero, false
	}

	// Check expiration
	if time.Now().After(e.expiresAt) {
		c.removeEntry(e)
		var zero V
		return zero, false
	}

	// Move to front (most recently used)
	c.order.MoveToFront(e.element)
	return e.value, true
}

// Set stores a value in the cache.
func (c *LRUCache[K, V]) Set(key K, value V, ttl time.Duration) {
	if ttl <= 0 {
		ttl = c.defaultTTL
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// Defensive check: if capacity is invalid, silently reject the write
	// This prevents infinite loop if cache was created without proper initialization
	if c.capacity <= 0 {
		return
	}

	// Update existing entry
	if e, ok := c.cache[key]; ok {
		e.value = value
		e.expiresAt = time.Now().Add(ttl)
		c.order.MoveToFront(e.element)
		return
	}

	// Evict if at capacity
	for len(c.cache) >= c.capacity {
		c.evictOldest()
	}

	// Create new entry
	e := &entry[K, V]{
		key:       key,
		value:     value,
		expiresAt: time.Now().Add(ttl),
	}
	e.element = c.order.PushFront(e)
	c.cache[key] = e
}

// SetWithDefaultTTL stores a value using the default TTL.
func (c *LRUCache[K, V]) SetWithDefaultTTL(key K, value V) {
	c.Set(key, value, c.defaultTTL)
}

// Invalidate removes entries matching the pattern.
// Supports * wildcard at the end (e.g., "user:123:*").
// Note: This method only works for string keys. For other key types, use Remove.
func (c *LRUCache[K, V]) Invalidate(pattern string) int {
	c.mu.Lock()
	defer c.mu.Unlock()

	count := 0

	// Check if K is string type using type assertion on a zero value
	var zero K
	if _, isString := any(zero).(string); !isString {
		return 0 // Non-string keys don't support pattern matching
	}

	// Exact match for string keys
	if !strings.Contains(pattern, "*") {
		key := any(pattern).(K) //nolint:errcheck // Safe: we verified K is string above
		if e, ok := c.cache[key]; ok {
			c.removeEntry(e)
			return 1
		}
		return 0
	}

	// Wildcard match (suffix only)
	prefix := strings.TrimSuffix(pattern, "*")
	for key, e := range c.cache {
		if keyStr, ok := any(key).(string); ok {
			if strings.HasPrefix(keyStr, prefix) {
				c.removeEntry(e)
				count++
			}
		}
	}

	return count
}

// Remove removes a specific entry from the cache.
func (c *LRUCache[K, V]) Remove(key K) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	if e, ok := c.cache[key]; ok {
		c.removeEntry(e)
		return true
	}
	return false
}

// Size returns the number of entries in the cache.
func (c *LRUCache[K, V]) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.cache)
}

// Clear removes all entries from the cache.
func (c *LRUCache[K, V]) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cache = make(map[K]*entry[K, V])
	c.order.Init()
}

// evictOldest removes the least recently used entry.
// Must be called with lock held.
func (c *LRUCache[K, V]) evictOldest() {
	if c.order.Len() == 0 {
		return
	}

	// Get the oldest entry (back of list)
	oldest := c.order.Back()
	if oldest == nil {
		return
	}

	e, ok := oldest.Value.(*entry[K, V])
	if !ok {
		return
	}
	c.removeEntry(e)
}

// removeEntry removes an entry from the cache.
// Must be called with lock held.
func (c *LRUCache[K, V]) removeEntry(e *entry[K, V]) {
	c.order.Remove(e.element)
	delete(c.cache, e.key)
}

// CleanupExpired removes all expired entries.
// Returns the number of entries removed.
func (c *LRUCache[K, V]) CleanupExpired() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Collect expired entries first to avoid modifying map during iteration
	var toDelete []*entry[K, V]
	now := time.Now()

	for _, e := range c.cache {
		if now.After(e.expiresAt) {
			toDelete = append(toDelete, e)
		}
	}

	// Remove collected entries
	for _, e := range toDelete {
		c.removeEntry(e)
	}

	return len(toDelete)
}

// Capacity returns the maximum capacity of the cache.
func (c *LRUCache[K, V]) Capacity() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.capacity
}

// Contains checks if a key exists in the cache (without updating access order).
func (c *LRUCache[K, V]) Contains(key K) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if e, ok := c.cache[key]; ok {
		return !time.Now().After(e.expiresAt)
	}
	return false
}

// ============================================================================
// Type alias for backward compatibility with non-generic LRUCache
// ============================================================================

// ByteLRUCache is a type alias for LRUCache with string keys and []byte values.
// Provided for backward compatibility with existing code that uses []byte values.
type ByteLRUCache = LRUCache[string, []byte]

// NewByteLRUCache creates a new LRU cache with string keys and []byte values.
// This is a convenience function for the common case of caching byte slices.
func NewByteLRUCache(capacity int, defaultTTL time.Duration) *ByteLRUCache {
	return NewLRUCache[string, []byte](capacity, defaultTTL)
}

// StringLRUCache is a type alias for LRUCache with string keys and string values.
type StringLRUCache = LRUCache[string, string]

// NewStringLRUCache creates a new LRU cache with string keys and string values.
func NewStringLRUCache(capacity int, defaultTTL time.Duration) *StringLRUCache {
	return NewLRUCache[string, string](capacity, defaultTTL)
}
