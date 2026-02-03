package cache

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
	"time"

	"log/slog"

	"github.com/hrygo/divinesense/ai"
	"github.com/pkg/errors"
)

// TieredCache implements a three-tier caching strategy:
// - L1: In-memory cache (fast, small, DEFAULT)
// - L2: Redis cache (moderate, shared, OPTIONAL)
// - L3: Database callback (slow, persistent)
//
// DEFAULT BEHAVIOR (personal system):
//   - L1 memory cache enabled (1000 items, 5min TTL)
//   - L2 Redis disabled
//
// TO ENABLE REDIS (multi-instance):
//   - Set MEMOS_CACHE_REDIS_ADDR environment variable
type TieredCache struct {
	l1        *Cache
	l2        RedisCacheInterface
	l1Enabled bool
	l2Enabled bool
}

// L3Fetcher is the function to fetch data from the database (L3).
type L3Fetcher func(ctx context.Context, key string) (any, error)

// TieredCacheConfig holds the configuration for the tiered cache.
type TieredCacheConfig struct {
	L1MaxItems int           // Max items in L1 memory cache
	L1TTL      time.Duration // TTL for L1 cache entries
	L2TTL      time.Duration // TTL for L2 Redis cache entries
	EnableL1   bool          // Enable L1 memory cache (default: true)
	EnableL2   bool          // Enable L2 Redis cache (default: false, auto-enabled if MEMOS_CACHE_REDIS_ADDR set)
}

// DefaultTieredConfig returns the default tiered cache configuration.
// For personal systems: L1 enabled, L2 disabled.
func DefaultTieredConfig() *TieredCacheConfig {
	return &TieredCacheConfig{
		L1MaxItems: 1000,
		L1TTL:      30 * time.Minute,
		L2TTL:      30 * time.Minute,
		EnableL1:   true,             // Memory cache ON by default
		EnableL2:   IsRedisEnabled(), // Auto-enable Redis if configured
	}
}

// NewTieredCache creates a new three-tier cache.
func NewTieredCache(config *TieredCacheConfig) (*TieredCache, error) {
	if config == nil {
		config = DefaultTieredConfig()
	}

	tc := &TieredCache{
		l1Enabled: config.EnableL1,
		l2Enabled: config.EnableL2,
	}

	// Initialize L1 cache
	if config.EnableL1 {
		tc.l1 = New(Config{
			DefaultTTL:      config.L1TTL,
			CleanupInterval: 1 * time.Minute,
			MaxItems:        config.L1MaxItems,
		})
	}

	// Initialize L2 cache (optional)
	if config.EnableL2 {
		// For now, use NilRedisCache (no-op)
		// To enable Redis, uncomment and use proper implementation
		// l2, err := NewRedisCache(&RedisConfig{...})
		tc.l2 = NewNilRedisCache()
		tc.l2Enabled = true
	}

	return tc, nil
}

// Get retrieves a value from the cache, checking L1, then L2, then L3.
func (t *TieredCache) Get(ctx context.Context, key string, fetcher L3Fetcher) (any, bool) {
	// Try L1 first
	if t.l1Enabled && t.l1 != nil {
		if value, found := t.l1.Get(ctx, key); found {
			return value, true
		}
	}

	// Try L2
	if t.l2Enabled && t.l2 != nil {
		if value, found := t.l2.Get(ctx, key); found {
			// Promote to L1
			if t.l1Enabled && t.l1 != nil {
				t.l1.Set(ctx, key, value)
			}
			return value, true
		}
	}

	// Fetch from L3
	if fetcher != nil {
		value, err := fetcher(ctx, key)
		if err != nil {
			return nil, false
		}

		// Store in L1 and L2
		if t.l1Enabled && t.l1 != nil {
			t.l1.Set(ctx, key, value)
		}
		if t.l2Enabled && t.l2 != nil {
			t.l2.Set(ctx, key, value)
		}

		return value, true
	}

	return nil, false
}

// Set stores a value in both L1 and L2.
func (t *TieredCache) Set(ctx context.Context, key string, value any) {
	if t.l1Enabled && t.l1 != nil {
		t.l1.Set(ctx, key, value)
	}
	if t.l2Enabled && t.l2 != nil {
		t.l2.Set(ctx, key, value)
	}
}

// SetWithTTL stores a value with custom TTL.
func (t *TieredCache) SetWithTTL(ctx context.Context, key string, value any, ttl time.Duration) {
	if t.l1Enabled && t.l1 != nil {
		t.l1.SetWithTTL(ctx, key, value, ttl)
	}
	if t.l2Enabled && t.l2 != nil {
		t.l2.SetWithTTL(ctx, key, value, ttl)
	}
}

// Delete removes a value from both L1 and L2.
func (t *TieredCache) Delete(ctx context.Context, key string) {
	if t.l1Enabled && t.l1 != nil {
		t.l1.Delete(ctx, key)
	}
	if t.l2Enabled && t.l2 != nil {
		t.l2.Delete(ctx, key)
	}
}

// Invalidate removes a value and optionally refreshes it.
func (t *TieredCache) Invalidate(ctx context.Context, key string, fetcher L3Fetcher) error {
	t.Delete(ctx, key)

	if fetcher != nil {
		value, err := fetcher(ctx, key)
		if err != nil {
			return err
		}
		t.Set(ctx, key, value)
	}

	return nil
}

// Clear clears all caches.
func (t *TieredCache) Clear(ctx context.Context) {
	if t.l1Enabled && t.l1 != nil {
		t.l1.Clear(ctx)
	}
	if t.l2Enabled && t.l2 != nil {
		t.l2.Clear(ctx)
	}
}

// Stats returns cache statistics.
func (t *TieredCache) Stats() map[string]interface{} {
	stats := make(map[string]interface{})

	if t.l1Enabled && t.l1 != nil {
		stats["l1_size"] = t.l1.Size()
		stats["l1_enabled"] = true
	} else {
		stats["l1_enabled"] = false
	}

	if t.l2Enabled && t.l2 != nil {
		stats["l2_enabled"] = true
		// L2 stats would need to be added to the interface
	} else {
		stats["l2_enabled"] = false
	}

	return stats
}

// Close closes all cache connections.
func (t *TieredCache) Close() error {
	var errs []error

	if t.l2 != nil {
		if err := t.l2.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	if t.l1 != nil {
		if err := t.l1.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return errors.Errorf("multiple errors: %v", errs)
	}

	return nil
}

// SemanticCache provides semantic caching based on vector similarity.
// It caches query results and can return semantically similar cached results.
type SemanticCache struct {
	l1        *Cache
	embedding ai.EmbeddingService
	threshold float32 // Similarity threshold for semantic matching (0.0-1.0)
	mu        sync.RWMutex
}

// SemanticCacheEntry represents a cached semantic search result.
type SemanticCacheEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Results   any       `json:"results"`
	Query     string    `json:"query"`
	Embedding []float32 `json:"embedding"`
}

// NewSemanticCache creates a new semantic cache.
func NewSemanticCache(embedding ai.EmbeddingService, threshold float32, maxItems int) *SemanticCache {
	if threshold <= 0 {
		threshold = 0.95 // Default threshold
	}

	return &SemanticCache{
		l1:        New(Config{MaxItems: maxItems, DefaultTTL: 30 * time.Minute}),
		embedding: embedding,
		threshold: threshold,
	}
}

// Get retrieves a cached result, either by exact key or semantic similarity.
func (s *SemanticCache) Get(ctx context.Context, query string) (any, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// First try exact match
	if value, found := s.l1.Get(ctx, s.hashKey(query)); found {
		if entry, ok := value.(*SemanticCacheEntry); ok {
			return entry.Results, true
		}
	}

	// Then try semantic matching
	if s.embedding != nil {
		embedding, err := s.embedding.Embed(ctx, query)
		if err != nil {
			return nil, false
		}

		// Search for similar queries
		var bestMatch *SemanticCacheEntry
		var bestScore float32

		s.l1.data.Range(func(key, value any) bool {
			if entry, ok := value.(*SemanticCacheEntry); ok {
				score := cosineSimilarity(embedding, entry.Embedding)
				if score > s.threshold && score > bestScore {
					bestMatch = entry
					bestScore = score
				}
			}
			return true
		})

		if bestMatch != nil {
			slog.Debug("semantic cache hit", "query", query, "matched_query", bestMatch.Query, "score", bestScore)
			return bestMatch.Results, true
		}
	}

	return nil, false
}

// Set stores a query result in the semantic cache.
func (s *SemanticCache) Set(ctx context.Context, query string, results any) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var embedding []float32
	var err error

	if s.embedding != nil {
		embedding, err = s.embedding.Embed(ctx, query)
		if err != nil {
			return errors.Wrap(err, "failed to embed query")
		}
	}

	entry := &SemanticCacheEntry{
		Query:     query,
		Embedding: embedding,
		Results:   results,
		Timestamp: time.Now(),
	}

	s.l1.Set(ctx, s.hashKey(query), entry)
	return nil
}

// Delete removes a query from the cache.
func (s *SemanticCache) Delete(ctx context.Context, query string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.l1.Delete(ctx, s.hashKey(query))
}

// hashKey generates a hash key for the query.
func (s *SemanticCache) hashKey(query string) string {
	h := sha256.Sum256([]byte(query))
	return "semantic:" + hex.EncodeToString(h[:])[:16]
}

// cosineSimilarity calculates the cosine similarity between two vectors.
func cosineSimilarity(a, b []float32) float32 {
	if len(a) != len(b) {
		return 0
	}

	var dotProduct float32
	var normA float32
	var normB float32

	for i := 0; i < len(a); i++ {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dotProduct / (sqrt32(normA) * sqrt32(normB))
}

// sqrt32 calculates square root for float32.
func sqrt32(x float32) float32 {
	return float32(sqrt(float64(x)))
}

// sqrt is a simple square root implementation.
func sqrt(x float64) float64 {
	// Newton's method
	z := 1.0
	for i := 0; i < 10; i++ {
		z -= (z*z - x) / (2 * z)
	}
	return z
}

// GenerateQueryKey generates a consistent cache key from query components.
func GenerateQueryKey(userID int, query string, limit int, strategy string) string {
	components := []string{
		fmt.Sprintf("user:%d", userID),
		"query:" + strings.ToLower(strings.TrimSpace(query)),
		fmt.Sprintf("limit:%d", limit),
		"strategy:" + strategy,
	}

	key := strings.Join(components, "|")
	h := sha256.Sum256([]byte(key))
	return "q:" + hex.EncodeToString(h[:])[:12]
}

// CacheStats represents combined cache statistics.
type CacheStats struct {
	Metadata    map[string]interface{} `json:"metadata"`
	L1Size      int64                  `json:"l1_size"`
	L2Size      int64                  `json:"l2_size"`
	L1HitRate   float64                `json:"l1_hit_rate"`
	L2HitRate   float64                `json:"l2_hit_rate"`
	SemanticHit int64                  `json:"semantic_hits"`
	TotalHits   int64                  `json:"total_hits"`
	TotalMisses int64                  `json:"total_misses"`
}

// GetCacheStats returns statistics from all cache layers.
// Note: Hit/miss tracking is not currently implemented; returns 0 for those fields.
// To enable hit/miss tracking, add atomic counters to Cache struct.
func GetCacheStats(tiered *TieredCache, semantic *SemanticCache) *CacheStats {
	stats := &CacheStats{
		Metadata: make(map[string]interface{}),
	}

	if tiered != nil {
		if tiered.l1 != nil {
			stats.L1Size = tiered.l1.Size()
		}
		stats.Metadata["l1_enabled"] = tiered.l1Enabled
		stats.Metadata["l2_enabled"] = tiered.l2Enabled

		// L2 size: Redis cache doesn't expose size through the interface
		// Would require adding Size() method to RedisCacheInterface
		if tiered.l2 != nil && tiered.l2Enabled {
			stats.L2Size = -1 // -1 indicates enabled but size unavailable
		}
	}

	if semantic != nil && semantic.l1 != nil {
		stats.Metadata["semantic_size"] = semantic.l1.Size()
		stats.Metadata["semantic_threshold"] = semantic.threshold
	}

	// Hit/miss tracking: Requires adding atomic counters to Cache struct
	// Leaving as 0 to avoid breaking API
	stats.L1HitRate = 0
	stats.L2HitRate = 0
	stats.SemanticHit = 0
	stats.TotalHits = 0
	stats.TotalMisses = 0

	return stats
}
