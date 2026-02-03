package agent

import (
	"container/list"
	"fmt"
	"sync"
	"time"
)

// LRUCache is a thread-safe LRU (Least Recently Used) cache with TTL support.
// LRUCache 是一个线程安全的 LRU（最近最少使用）缓存，支持 TTL。
type LRUCache struct {
	entries    map[string]*list.Element
	lruList    *list.List
	maxEntries int
	ttl        time.Duration
	hits       int64
	misses     int64
	mutex      sync.RWMutex
}

// cacheEntry represents a cache entry with value and expiration.
// cacheEntry 表示包含值和过期时间的缓存条目。
type cacheEntry struct {
	expiration time.Time
	value      interface{}
	key        string
}

// CacheEntry represents a cached value with metadata.
// CacheEntry 表示带有元数据的缓存值。
type CacheEntry struct {
	Value      interface{} `json:"value"`
	Key        string      `json:"key"`
	ExpiresAt  int64       `json:"expires_at"`
	SizeBytes  int         `json:"size_bytes"`
	AccessTime int64       `json:"access_time"`
}

// CacheStats represents cache statistics.
// CacheStats 表示缓存统计信息。
type CacheStats struct {
	Size       int     `json:"size"`
	MaxEntries int     `json:"max_entries"`
	Hits       int64   `json:"hits"`
	Misses     int64   `json:"misses"`
	HitRate    float64 `json:"hit_rate"`
}

// NewLRUCache creates a new LRUCache.
// NewLRUCache 创建一个新的 LRUCache。
//
// Parameters:
//   - maxEntries: Maximum number of entries in the cache (must be > 0)
//   - ttl: Time-to-live for cache entries (0 = no expiration)
//
// Example:
//
//	// Create a cache with max 100 entries and 5 minute TTL
//	cache := NewLRUCache(100, 5*time.Minute)
func NewLRUCache(maxEntries int, ttl time.Duration) *LRUCache {
	if maxEntries <= 0 {
		maxEntries = 100 // Default to 100 if invalid
	}

	return &LRUCache{
		maxEntries: maxEntries,
		ttl:        ttl,
		entries:    make(map[string]*list.Element),
		lruList:    list.New(),
	}
}

// Get retrieves a value from the cache.
// Get 从缓存中检索值。
//
// Returns:
//   - value: The cached value (or nil if not found/expired)
//   - found: Whether the value was found and not expired
func (c *LRUCache) Get(key string) (interface{}, bool) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// Find the element
	elem, exists := c.entries[key]
	if !exists {
		c.misses++
		return nil, false
	}

	entry := elem.Value.(*cacheEntry) //nolint:errcheck // type assertion is safe

	// Check expiration
	if !entry.expiration.IsZero() && time.Now().After(entry.expiration) {
		// Entry expired, remove it
		c.removeElement(elem)
		c.misses++
		return nil, false
	}

	// Move to front (most recently used)
	c.lruList.MoveToFront(elem)
	c.hits++

	return entry.value, true
}

// Set stores a value in the cache.
// Set 在缓存中存储值。
//
// If the key already exists, the value is updated and the entry is moved to the front.
// If the cache is full, the least recently used entry is evicted.
func (c *LRUCache) Set(key string, value interface{}) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// Check if key already exists
	if elem, exists := c.entries[key]; exists {
		// Update existing entry
		entry := elem.Value.(*cacheEntry) //nolint:errcheck // type assertion is safe
		entry.value = value
		entry.expiration = c.calculateExpiration()
		c.lruList.MoveToFront(elem)
		return
	}

	// Create new entry
	entry := &cacheEntry{
		key:        key,
		value:      value,
		expiration: c.calculateExpiration(),
	}

	// Add to front of LRU list
	elem := c.lruList.PushFront(entry)
	c.entries[key] = elem

	// Check if cache is full
	if c.lruList.Len() > c.maxEntries {
		// Evict least recently used (back of list)
		c.evictLRU()
	}
}

// Delete removes a key from the cache.
// Delete 从缓存中删除键。
func (c *LRUCache) Delete(key string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if elem, exists := c.entries[key]; exists {
		c.removeElement(elem)
	}
}

// Clear removes all entries from the cache.
// Clear 从缓存中移除所有条目。
func (c *LRUCache) Clear() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.entries = make(map[string]*list.Element)
	c.lruList.Init()
	c.hits = 0
	c.misses = 0
}

// Size returns the current number of entries in the cache.
// Size 返回缓存中的当前条目数。
func (c *LRUCache) Size() int {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.lruList.Len()
}

// Stats returns cache statistics.
// Stats 返回缓存统计信息。
func (c *LRUCache) Stats() CacheStats {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	total := c.hits + c.misses
	hitRate := 0.0
	if total > 0 {
		hitRate = float64(c.hits) / float64(total)
	}

	return CacheStats{
		Size:       c.lruList.Len(),
		MaxEntries: c.maxEntries,
		Hits:       c.hits,
		Misses:     c.misses,
		HitRate:    hitRate,
	}
}

// calculateExpiration calculates the expiration time for a new entry.
// calculateExpiration 计算新条目的过期时间。
func (c *LRUCache) calculateExpiration() time.Time {
	if c.ttl <= 0 {
		return time.Time{} // Zero time means no expiration
	}
	return time.Now().Add(c.ttl)
}

// evictLRU removes the least recently used entry from the cache.
// evictLRU 从缓存中移除最近最少使用的条目。
func (c *LRUCache) evictLRU() {
	elem := c.lruList.Back()
	if elem != nil {
		c.removeElement(elem)
	}
}

// removeElement removes an element from the cache.
// removeElement 从缓存中移除元素。
func (c *LRUCache) removeElement(elem *list.Element) {
	entry := elem.Value.(*cacheEntry) //nolint:errcheck // type assertion is safe
	delete(c.entries, entry.key)
	c.lruList.Remove(elem)
}

// String returns a string representation of the cache.
// String 返回缓存的字符串表示。
func (c *LRUCache) String() string {
	stats := c.Stats()
	return fmt.Sprintf("LRUCache{size=%d/%d, hits=%d, misses=%d, hit_rate=%.2f%%}",
		stats.Size, stats.MaxEntries, stats.Hits, stats.Misses, stats.HitRate*100)
}

// GenericCache is a generic type-safe cache wrapper.
// GenericCache 是泛型类型安全的缓存包装器。
type GenericCache[T any] struct {
	cache *LRUCache
}

// NewGenericCache creates a new generic cache.
// NewGenericCache 创建一个新的泛型缓存。
func NewGenericCache[T any](maxEntries int, ttl time.Duration) *GenericCache[T] {
	return &GenericCache[T]{
		cache: NewLRUCache(maxEntries, ttl),
	}
}

// Get retrieves a value from the generic cache.
// Get 从泛型缓存中检索值。
func (g *GenericCache[T]) Get(key string) (T, bool) {
	value, found := g.cache.Get(key)
	if !found {
		var zero T
		return zero, false
	}
	typed, ok := value.(T)
	if !ok {
		var zero T
		return zero, false
	}
	return typed, true
}

// Set stores a value in the generic cache.
// Set 在泛型缓存中存储值。
func (g *GenericCache[T]) Set(key string, value T) {
	g.cache.Set(key, value)
}

// Delete removes a key from the generic cache.
// Delete 从泛型缓存中删除键。
func (g *GenericCache[T]) Delete(key string) {
	g.cache.Delete(key)
}

// Clear removes all entries from the generic cache.
// Clear 从泛型缓存中移除所有条目。
func (g *GenericCache[T]) Clear() {
	g.cache.Clear()
}

// Size returns the current number of entries in the generic cache.
// Size 返回泛型缓存中的当前条目数。
func (g *GenericCache[T]) Size() int {
	return g.cache.Size()
}

// Stats returns cache statistics for the generic cache.
// Stats 返回泛型缓存的缓存统计信息。
func (g *GenericCache[T]) Stats() CacheStats {
	return g.cache.Stats()
}
