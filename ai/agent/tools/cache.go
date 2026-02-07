// Package tools provides tool-level result caching for AI agents.
// Issue #92: 工具级检索结果缓存 - 减少重复数据库查询
package tools

import (
	"container/list"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"log/slog"
)

// CacheKey represents a unique identifier for tool result caching.
// CacheKey 表示工具结果缓存的唯一标识符。
type CacheKey struct {
	ToolName  string // 工具名称 (schedule_query, memo_search, etc.)
	UserID    int32  // 用户 ID (多租户隔离)
	InputHash string // 输入参数的 SHA256 哈希
}

// String returns the string representation of the cache key.
func (k CacheKey) String() string {
	return fmt.Sprintf("tool:%s:user:%d:hash:%s", k.ToolName, k.UserID, k.InputHash)
}

// NewCacheKey creates a new CacheKey from tool name, user ID, and input.
func NewCacheKey(toolName string, userID int32, input string) CacheKey {
	hash := sha256.Sum256([]byte(input))
	return CacheKey{
		ToolName:  toolName,
		UserID:    userID,
		InputHash: hex.EncodeToString(hash[:]),
	}
}

// CachedResult represents a cached tool execution result with metadata.
// CachedResult 表示带元数据的缓存工具执行结果。
type CachedResult struct {
	Output    string // 工具返回的文本结果
	Timestamp int64  // 缓存时间戳 (Unix)
	ExpiresAt int64  // 过期时间戳 (Unix)
}

// toolCacheEntry represents a cache entry with value and expiration.
type toolCacheEntry struct {
	expiration time.Time
	value      interface{}
	key        string
}

// ToolResultCache manages caching of tool execution results.
// ToolResultCache 管理工具执行结果的缓存。
type ToolResultCache struct {
	entries    map[string]*list.Element
	lruList    *list.List
	maxEntries int
	mu         sync.RWMutex
	ttlMap     map[string]time.Duration
	enabled    bool

	// Statistics
	hits    map[string]int64
	misses  map[string]int64
	statsMu sync.Mutex
}

// NewToolResultCache creates a new tool result cache with default TTLs.
func NewToolResultCache(maxEntries int) *ToolResultCache {
	if maxEntries <= 0 {
		maxEntries = 100
	}

	cache := &ToolResultCache{
		entries:    make(map[string]*list.Element),
		lruList:    list.New(),
		maxEntries: maxEntries,
		ttlMap:     make(map[string]time.Duration),
		enabled:    true,
		hits:       make(map[string]int64),
		misses:     make(map[string]int64),
	}

	// 设置默认 TTL 策略
	cache.SetDefaultTTLs()

	return cache
}

// SetDefaultTTLs configures TTL for different tool types.
func (c *ToolResultCache) SetDefaultTTLs() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.ttlMap = map[string]time.Duration{
		// 只读工具 (Read-only tools)
		"schedule_query": 30 * time.Second, // 日程查询: 30 秒
		"memo_search":    5 * time.Minute,  // 笔记搜索: 5 分钟
		"find_free_time": 1 * time.Minute,  // 空闲时间: 1 分钟

		// 写操作工具 (Write operations) - 不缓存 (TTL = 0)
		"schedule_add":    0,
		"schedule_update": 0,
	}
}

// GetTTL returns the TTL for a specific tool, or 0 if not cacheable.
func (c *ToolResultCache) GetTTL(toolName string) time.Duration {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.ttlMap[toolName]
}

// SetTTL allows runtime configuration of tool TTL.
func (c *ToolResultCache) SetTTL(toolName string, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.ttlMap[toolName] = ttl
}

// IsCacheable checks if a tool's result should be cached.
func (c *ToolResultCache) IsCacheable(toolName string) bool {
	return c.GetTTL(toolName) > 0
}

// Get retrieves a cached result if available and not expired.
func (c *ToolResultCache) Get(key CacheKey) (*CachedResult, bool) {
	if !c.enabled {
		return nil, false
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	elem, exists := c.entries[key.String()]
	if !exists {
		c.recordMiss(key.ToolName)
		return nil, false
	}

	entry := elem.Value.(*toolCacheEntry) //nolint:errcheck // type assertion is safe

	// Check expiration
	if !entry.expiration.IsZero() && time.Now().After(entry.expiration) {
		// Entry expired, remove it
		c.removeElement(elem)
		c.recordMiss(key.ToolName)
		return nil, false
	}

	// Move to front (most recently used)
	c.lruList.MoveToFront(elem)
	c.recordHit(key.ToolName)

	// Extract result from entry
	if result, ok := entry.value.(*CachedResult); ok {
		return result, true
	}

	return nil, false
}

// Set stores a tool result in the cache.
func (c *ToolResultCache) Set(key CacheKey, output string) {
	if !c.enabled {
		return
	}

	ttl := c.GetTTL(key.ToolName)
	if ttl <= 0 {
		// 不缓存写操作
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	result := &CachedResult{
		Output:    output,
		Timestamp: now.Unix(),
		ExpiresAt: now.Add(ttl).Unix(),
	}

	// Create new entry
	entry := &toolCacheEntry{
		key:        key.String(),
		value:      result,
		expiration: now.Add(ttl),
	}

	// Add to front of LRU list
	elem := c.lruList.PushFront(entry)
	c.entries[key.String()] = elem

	// Check if cache is full
	if c.lruList.Len() > c.maxEntries {
		// Evict least recently used (back of list)
		c.evictLRU()
	}

	slog.Debug("tool result cached",
		"tool", key.ToolName,
		"user_id", key.UserID,
		"ttl_seconds", ttl.Seconds(),
	)
}

// Invalidate removes cache entries matching the pattern.
func (c *ToolResultCache) Invalidate(toolName string, userID int32) int {
	c.mu.Lock()
	defer c.mu.Unlock()

	removed := 0
	prefix := fmt.Sprintf("tool:%s:user:%d:", toolName, userID)

	// Iterate through keys and delete matching ones
	for key, elem := range c.entries {
		if len(key) >= len(prefix) && key[:len(prefix)] == prefix {
			c.removeElement(elem)
			removed++
		}
	}

	slog.Info("tool cache invalidated",
		"tool", toolName,
		"user_id", userID,
		"removed_count", removed,
	)
	return removed
}

// Clear removes all entries from the cache.
func (c *ToolResultCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries = make(map[string]*list.Element)
	c.lruList.Init()

	c.statsMu.Lock()
	c.hits = make(map[string]int64)
	c.misses = make(map[string]int64)
	c.statsMu.Unlock()
}

// Enable enables the cache.
func (c *ToolResultCache) Enable() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.enabled = true
}

// Disable disables the cache.
func (c *ToolResultCache) Disable() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.enabled = false
}

// IsEnabled returns whether the cache is enabled.
func (c *ToolResultCache) IsEnabled() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.enabled
}

// recordHit records a cache hit for statistics.
func (c *ToolResultCache) recordHit(toolName string) {
	c.statsMu.Lock()
	defer c.statsMu.Unlock()
	c.hits[toolName]++
}

// recordMiss records a cache miss for statistics.
func (c *ToolResultCache) recordMiss(toolName string) {
	c.statsMu.Lock()
	defer c.statsMu.Unlock()
	c.misses[toolName]++
}

// evictLRU removes the least recently used entry from the cache.
func (c *ToolResultCache) evictLRU() {
	elem := c.lruList.Back()
	if elem != nil {
		c.removeElement(elem)
	}
}

// removeElement removes an element from the cache.
func (c *ToolResultCache) removeElement(elem *list.Element) {
	entry := elem.Value.(*toolCacheEntry) //nolint:errcheck // type assertion is safe
	delete(c.entries, entry.key)
	c.lruList.Remove(elem)
}

// CacheStats represents cache statistics for a specific tool.
type CacheStats struct {
	ToolName string  `json:"tool_name"`
	Hits     int64   `json:"hits"`
	Misses   int64   `json:"misses"`
	HitRate  float64 `json:"hit_rate"`
}

// GetStats returns cache statistics for all tools.
func (c *ToolResultCache) GetStats() []CacheStats {
	c.statsMu.Lock()
	defer c.statsMu.Unlock()

	stats := make([]CacheStats, 0, len(c.ttlMap))

	// Collect all tool names from ttlMap and hits/misses
	toolNames := make(map[string]bool)
	for name := range c.ttlMap {
		toolNames[name] = true
	}
	for name := range c.hits {
		toolNames[name] = true
	}
	for name := range c.misses {
		toolNames[name] = true
	}

	for toolName := range toolNames {
		hits := c.hits[toolName]
		misses := c.misses[toolName]
		total := hits + misses
		hitRate := 0.0
		if total > 0 {
			hitRate = float64(hits) / float64(total)
		}

		stats = append(stats, CacheStats{
			ToolName: toolName,
			Hits:     hits,
			Misses:   misses,
			HitRate:  hitRate,
		})
	}

	return stats
}

// Size returns the current number of entries in the cache.
func (c *ToolResultCache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.lruList.Len()
}
