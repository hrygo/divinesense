package cache

import (
	"container/list"
	"strings"
	"sync"
	"time"
)

// LRUCache implements an LRU cache with TTL support.
type LRUCache struct {
	cache      map[string]*entry
	order      *list.List
	capacity   int
	defaultTTL time.Duration
	mu         sync.RWMutex
}

type entry struct {
	expiresAt time.Time
	element   *list.Element
	key       string
	value     []byte
}

// NewLRUCache creates a new LRU cache.
func NewLRUCache(capacity int, defaultTTL time.Duration) *LRUCache {
	if capacity <= 0 {
		capacity = 1000
	}
	if defaultTTL <= 0 {
		defaultTTL = 5 * time.Minute
	}

	return &LRUCache{
		capacity:   capacity,
		defaultTTL: defaultTTL,
		cache:      make(map[string]*entry),
		order:      list.New(),
	}
}

// Get retrieves a value from the cache.
func (c *LRUCache) Get(key string) ([]byte, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	e, ok := c.cache[key]
	if !ok {
		return nil, false
	}

	// Check expiration
	if time.Now().After(e.expiresAt) {
		c.removeEntry(e)
		return nil, false
	}

	// Move to front (most recently used)
	c.order.MoveToFront(e.element)
	return e.value, true
}

// Set stores a value in the cache.
func (c *LRUCache) Set(key string, value []byte, ttl time.Duration) {
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
	e := &entry{
		key:       key,
		value:     value,
		expiresAt: time.Now().Add(ttl),
	}
	e.element = c.order.PushFront(e)
	c.cache[key] = e
}

// Invalidate removes entries matching the pattern.
// Supports * wildcard at the end (e.g., "user:123:*").
func (c *LRUCache) Invalidate(pattern string) int {
	c.mu.Lock()
	defer c.mu.Unlock()

	count := 0

	// Exact match
	if !strings.Contains(pattern, "*") {
		if e, ok := c.cache[pattern]; ok {
			c.removeEntry(e)
			count = 1
		}
		return count
	}

	// Wildcard match (suffix only)
	prefix := strings.TrimSuffix(pattern, "*")
	for key, e := range c.cache {
		if strings.HasPrefix(key, prefix) {
			c.removeEntry(e)
			count++
		}
	}

	return count
}

// Size returns the number of entries in the cache.
func (c *LRUCache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.cache)
}

// Clear removes all entries from the cache.
func (c *LRUCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cache = make(map[string]*entry)
	c.order.Init()
}

// evictOldest removes the least recently used entry.
// Must be called with lock held.
func (c *LRUCache) evictOldest() {
	if c.order.Len() == 0 {
		return
	}

	// Get the oldest entry (back of list)
	oldest := c.order.Back()
	if oldest == nil {
		return
	}

	e, ok := oldest.Value.(*entry)
	if !ok {
		return
	}
	c.removeEntry(e)
}

// removeEntry removes an entry from the cache.
// Must be called with lock held.
func (c *LRUCache) removeEntry(e *entry) {
	c.order.Remove(e.element)
	delete(c.cache, e.key)
}

// CleanupExpired removes all expired entries.
// Returns the number of entries removed.
func (c *LRUCache) CleanupExpired() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Collect expired entries first to avoid modifying map during iteration
	var toDelete []*entry
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
