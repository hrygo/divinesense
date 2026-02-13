// Package routing provides routing result caching for performance optimization.
package routing

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	"github.com/hrygo/divinesense/ai/cache"
)

// CacheEntry represents a cached routing result.
type CacheEntry struct {
	Intent     Intent  `json:"intent"`
	Confidence float32 `json:"confidence"`
	Source     string  `json:"source"` // "rule", "history", "llm"
	Timestamp  int64   `json:"timestamp"`
}

// RouterCache provides LRU caching for routing decisions.
// Key benefits:
// - Eliminates redundant LLM calls (~400ms) for repeated queries
// - Reduces history matching overhead (~10ms) for similar inputs
// - Improves perceived latency for common queries
type RouterCache struct {
	cache          *cache.ByteLRUCache
	defaultTTL     time.Duration
	llmResultTTL   time.Duration // Longer TTL for LLM results (expensive)
	hitCount       int64
	missCount      int64
	lastStatsReset time.Time
	statsMu        sync.Mutex
}

// CacheConfig contains configuration for RouterCache.
type CacheConfig struct {
	Capacity     int           // Maximum number of entries (default: 500)
	DefaultTTL   time.Duration // TTL for rule/history matches (default: 5min)
	LLMResultTTL time.Duration // TTL for LLM results (default: 30min)
}

// NewRouterCache creates a new router cache with specified configuration.
func NewRouterCache(cfg CacheConfig) *RouterCache {
	if cfg.Capacity <= 0 {
		cfg.Capacity = 500
	}
	if cfg.DefaultTTL <= 0 {
		cfg.DefaultTTL = 5 * time.Minute
	}
	if cfg.LLMResultTTL <= 0 {
		cfg.LLMResultTTL = 30 * time.Minute
	}

	return &RouterCache{
		cache:          cache.NewByteLRUCache(cfg.Capacity, cfg.DefaultTTL),
		defaultTTL:     cfg.DefaultTTL,
		llmResultTTL:   cfg.LLMResultTTL,
		lastStatsReset: time.Now(),
	}
}

// Get retrieves a cached routing result.
// Returns (intent, confidence, found).
func (c *RouterCache) Get(input string) (Intent, float32, bool) {
	key := c.hashKey(input)
	data, found := c.cache.Get(key)
	if !found {
		c.incrementMiss()
		return IntentUnknown, 0, false
	}

	var entry CacheEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		slog.Debug("failed to unmarshal cache entry", "error", err)
		c.incrementMiss()
		return IntentUnknown, 0, false
	}

	c.incrementHit()
	slog.Debug("router cache hit", "input", truncate(input, 50), "intent", entry.Intent, "source", entry.Source)
	return entry.Intent, entry.Confidence, true
}

// Set stores a routing result in the cache.
// The source determines the TTL: LLM results get longer TTL.
func (c *RouterCache) Set(input string, intent Intent, confidence float32, source string) {
	key := c.hashKey(input)

	entry := CacheEntry{
		Intent:     intent,
		Confidence: confidence,
		Source:     source,
		Timestamp:  time.Now().Unix(),
	}

	data, err := json.Marshal(entry)
	if err != nil {
		slog.Warn("failed to marshal cache entry", "error", err)
		return
	}

	// Use longer TTL for LLM results (expensive computations)
	ttl := c.defaultTTL
	if source == "llm" {
		ttl = c.llmResultTTL
	}

	c.cache.Set(key, data, ttl)
	slog.Debug("router cache set", "input", truncate(input, 50), "intent", intent, "source", source, "ttl", ttl)
}

// Invalidate removes a specific entry from the cache.
func (c *RouterCache) Invalidate(input string) {
	key := c.hashKey(input)
	c.cache.Invalidate(key)
}

// Clear removes all entries from the cache.
func (c *RouterCache) Clear() {
	c.cache.Clear()
	c.resetStats()
}

// Stats returns cache statistics.
type Stats struct {
	Hits      int64   `json:"hits"`
	Misses    int64   `json:"misses"`
	HitRate   float64 `json:"hit_rate"`
	Size      int     `json:"size"`
	Capacity  int     `json:"capacity"`
	UptimeSec int64   `json:"uptime_sec"`
}

// GetStats returns current cache statistics.
func (c *RouterCache) GetStats() Stats {
	c.statsMu.Lock()
	defer c.statsMu.Unlock()

	hitCount := c.hitCount
	missCount := c.missCount
	total := hitCount + missCount
	hitRate := 0.0
	if total > 0 {
		hitRate = float64(hitCount) / float64(total)
	}

	return Stats{
		Hits:      hitCount,
		Misses:    missCount,
		HitRate:   hitRate,
		Size:      c.cache.Size(),
		Capacity:  c.cache.Capacity(),
		UptimeSec: int64(time.Since(c.lastStatsReset).Seconds()),
	}
}

// ResetStats resets hit/miss counters.
func (c *RouterCache) ResetStats() {
	c.resetStats()
}

// Capacity returns the cache capacity.
func (c *RouterCache) Capacity() int {
	return c.cache.Size()
}

// CleanupExpired triggers cleanup of expired entries.
func (c *RouterCache) CleanupExpired() int {
	return c.cache.CleanupExpired()
}

// hashKey creates a stable hash key for input.
// Using SHA256 for minimal collision probability.
func (c *RouterCache) hashKey(input string) string {
	hash := sha256.Sum256([]byte(input))
	return "route:" + hex.EncodeToString(hash[:8]) // First 8 bytes (64 bits) sufficient
}

// incrementHit atomically increments hit counter.
func (c *RouterCache) incrementHit() {
	c.statsMu.Lock()
	c.hitCount++
	c.statsMu.Unlock()
}

// incrementMiss atomically increments miss counter.
func (c *RouterCache) incrementMiss() {
	c.statsMu.Lock()
	c.missCount++
	c.statsMu.Unlock()
}

// resetStats resets statistics counters.
func (c *RouterCache) resetStats() {
	c.statsMu.Lock()
	c.hitCount = 0
	c.missCount = 0
	c.lastStatsReset = time.Now()
	c.statsMu.Unlock()
}
