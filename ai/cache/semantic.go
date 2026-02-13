// Package cache provides semantic caching for AI agents.
// Issue #91: 语义缓存层实现 - 基于 Embedding 相似度匹配
package cache

import (
	"container/list"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"math"
	"sync"
	"time"
)

// EmbeddingService defines the interface for generating vector embeddings.
// This is a local interface to avoid circular dependencies.
type EmbeddingService interface {
	// Embed generates a vector embedding for the given text.
	Embed(ctx context.Context, text string) ([]float32, error)

	// EmbedBatch generates vector embeddings for multiple texts.
	EmbedBatch(ctx context.Context, texts []string) ([][]float32, error)
}

// SemanticCacheConfig configures the semantic cache.
type SemanticCacheConfig struct {
	// MaxEntries is the maximum number of entries in the cache.
	MaxEntries int

	// SimilarityThreshold is the minimum cosine similarity for a match (0-1).
	SimilarityThreshold float32

	// TTL is the time-to-live for cache entries.
	TTL time.Duration

	// EmbeddingService is the vector embedding service.
	EmbeddingService EmbeddingService
}

// DefaultSemanticCacheConfig returns the default configuration.
func DefaultSemanticCacheConfig() SemanticCacheConfig {
	return SemanticCacheConfig{
		MaxEntries:          1000,
		SimilarityThreshold: 0.95,
		TTL:                 24 * time.Hour,
	}
}

// semanticCacheEntry represents a cached entry with embedding.
type semanticCacheEntry struct {
	key       string
	text      string
	embedding []float32
	expiresAt time.Time
	element   *list.Element
}

// SemanticCacheStats represents cache statistics.
type SemanticCacheStats struct {
	ExactHits              int64
	ExactMisses            int64
	SemanticHits           int64
	SemanticMisses         int64
	SemanticSize           int
	SimilarityDistribution map[string]int64
}

// SemanticCache provides two-layer caching: exact match (SHA256) and semantic match (cosine similarity).
type SemanticCache struct {
	cfg SemanticCacheConfig

	// Exact match cache (SHA256 hash key)
	exactCache *ByteLRUCache

	// Semantic match cache (vector similarity)
	semanticCache struct {
		sync.RWMutex
		entries map[string]*semanticCacheEntry
		lruList *list.List
	}

	// Statistics
	stats   SemanticCacheStats
	statsMu sync.Mutex
}

// NewSemanticCache creates a new semantic cache.
func NewSemanticCache(cfg SemanticCacheConfig) *SemanticCache {
	if cfg.MaxEntries <= 0 {
		cfg.MaxEntries = 1000
	}
	if cfg.SimilarityThreshold <= 0 || cfg.SimilarityThreshold > 1 {
		cfg.SimilarityThreshold = 0.95
	}
	if cfg.TTL <= 0 {
		cfg.TTL = 24 * time.Hour
	}

	return &SemanticCache{
		cfg:        cfg,
		exactCache: NewByteLRUCache(cfg.MaxEntries, cfg.TTL),
		semanticCache: struct {
			sync.RWMutex
			entries map[string]*semanticCacheEntry
			lruList *list.List
		}{
			entries: make(map[string]*semanticCacheEntry),
			lruList: list.New(),
		},
		stats: SemanticCacheStats{
			SimilarityDistribution: make(map[string]int64),
		},
	}
}

// Get retrieves a cached embedding.
// Returns (embedding, found, similarity, isExactMatch).
func (c *SemanticCache) Get(ctx context.Context, text string) ([]float32, bool, float32, bool) {
	exactKey := c.hashKey(text)

	// Layer 1: Exact match (SHA256)
	if data, found := c.exactCache.Get(exactKey); found {
		c.recordExactHit()
		embedding, err := c.decodeEmbedding(data)
		if err == nil {
			return embedding, true, 1.0, true
		}
	}
	c.recordExactMiss()

	// Layer 2: Semantic match (cosine similarity)
	// Only attempt if embedding service is available
	if c.cfg.EmbeddingService == nil {
		return nil, false, 0, false
	}

	queryEmbedding, err := c.cfg.EmbeddingService.Embed(ctx, text)
	if err != nil {
		c.recordSemanticMiss()
		return nil, false, 0, false
	}

	bestMatch, similarity := c.findSimilar(queryEmbedding)
	if similarity >= c.cfg.SimilarityThreshold {
		c.recordSemanticHit()
		c.recordSimilarity(similarity)
		return bestMatch.embedding, true, similarity, false
	}

	c.recordSemanticMiss()
	return nil, false, 0, false
}

// Set stores a text and its embedding in the cache.
func (c *SemanticCache) Set(ctx context.Context, text string, embedding []float32) error {
	exactKey := c.hashKey(text)

	// Write to exact cache
	encoded, err := c.encodeEmbedding(embedding)
	if err != nil {
		return err
	}
	c.exactCache.Set(exactKey, encoded, c.cfg.TTL)

	// Write to semantic cache
	c.semanticCache.Lock()
	defer c.semanticCache.Unlock()

	entry := &semanticCacheEntry{
		key:       exactKey,
		text:      text,
		embedding: embedding,
		expiresAt: time.Now().Add(c.cfg.TTL),
	}
	entry.element = c.semanticCache.lruList.PushFront(entry)
	c.semanticCache.entries[exactKey] = entry

	// LRU eviction
	if c.semanticCache.lruList.Len() > c.cfg.MaxEntries {
		c.evictLRU()
	}

	return nil
}

// findSimilar finds the most similar cached entry.
func (c *SemanticCache) findSimilar(query []float32) (*semanticCacheEntry, float32) {
	c.semanticCache.RLock()
	defer c.semanticCache.RUnlock()

	var bestMatch *semanticCacheEntry
	var bestSimilarity float32 = 0

	now := time.Now()
	for _, entry := range c.semanticCache.entries {
		// Skip expired entries
		if now.After(entry.expiresAt) {
			continue
		}

		sim := cosineSimilarity(query, entry.embedding)
		if sim > bestSimilarity {
			bestSimilarity = sim
			bestMatch = entry
		}
	}

	return bestMatch, bestSimilarity
}

// evictLRU removes the least recently used entry from the semantic cache.
func (c *SemanticCache) evictLRU() {
	elem := c.semanticCache.lruList.Back()
	if elem != nil {
		entry := elem.Value.(*semanticCacheEntry) //nolint:errcheck // type assertion is safe
		delete(c.semanticCache.entries, entry.key)
		c.semanticCache.lruList.Remove(elem)
	}
}

// hashKey generates a SHA256 hash key for text.
func (c *SemanticCache) hashKey(text string) string {
	hash := sha256.Sum256([]byte(text))
	return "semantic:" + hex.EncodeToString(hash[:8])
}

// encodeEmbedding encodes a vector to bytes.
func (c *SemanticCache) encodeEmbedding(vec []float32) ([]byte, error) {
	buf := make([]byte, len(vec)*4)
	for i, v := range vec {
		binary.LittleEndian.PutUint32(buf[i*4:], math.Float32bits(v))
	}
	return buf, nil
}

// decodeEmbedding decodes bytes to a vector.
func (c *SemanticCache) decodeEmbedding(data []byte) ([]float32, error) {
	vec := make([]float32, len(data)/4)
	for i := 0; i < len(vec); i++ {
		bits := binary.LittleEndian.Uint32(data[i*4 : (i+1)*4])
		vec[i] = math.Float32frombits(bits)
	}
	return vec, nil
}

// recordExactHit records an exact cache hit.
func (c *SemanticCache) recordExactHit() {
	c.statsMu.Lock()
	defer c.statsMu.Unlock()
	c.stats.ExactHits++
}

// recordExactMiss records an exact cache miss.
func (c *SemanticCache) recordExactMiss() {
	c.statsMu.Lock()
	defer c.statsMu.Unlock()
	c.stats.ExactMisses++
}

// recordSemanticHit records a semantic cache hit.
func (c *SemanticCache) recordSemanticHit() {
	c.statsMu.Lock()
	defer c.statsMu.Unlock()
	c.stats.SemanticHits++
}

// recordSemanticMiss records a semantic cache miss.
func (c *SemanticCache) recordSemanticMiss() {
	c.statsMu.Lock()
	defer c.statsMu.Unlock()
	c.stats.SemanticMisses++
}

// recordSimilarity records the similarity distribution.
func (c *SemanticCache) recordSimilarity(sim float32) {
	c.statsMu.Lock()
	defer c.statsMu.Unlock()

	// Bucket: 0.00-0.90, 0.90-0.95, 0.95-1.00
	bucket := "0.00-0.90"
	if sim >= 0.95 {
		bucket = "0.95-1.00"
	} else if sim >= 0.90 {
		bucket = "0.90-0.95"
	}

	if c.stats.SimilarityDistribution == nil {
		c.stats.SimilarityDistribution = make(map[string]int64)
	}
	c.stats.SimilarityDistribution[bucket]++
}

// GetStats returns the cache statistics.
func (c *SemanticCache) GetStats() SemanticCacheStats {
	c.statsMu.Lock()
	defer c.statsMu.Unlock()

	// Copy the stats to avoid race conditions
	stats := c.stats
	stats.SemanticSize = c.semanticCache.lruList.Len()

	return stats
}

// Clear clears all cache entries.
func (c *SemanticCache) Clear() {
	c.exactCache.Clear()

	c.semanticCache.Lock()
	defer c.semanticCache.Unlock()

	c.semanticCache.entries = make(map[string]*semanticCacheEntry)
	c.semanticCache.lruList.Init()

	c.statsMu.Lock()
	c.stats = SemanticCacheStats{
		SimilarityDistribution: make(map[string]int64),
	}
	c.statsMu.Unlock()
}

// cosineSimilarity calculates the cosine similarity between two vectors.
func cosineSimilarity(a, b []float32) float32 {
	if len(a) != len(b) {
		return 0
	}

	var dotProduct float32
	var normA, normB float32

	for i := range a {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dotProduct / (float32(math.Sqrt(float64(normA))) * float32(math.Sqrt(float64(normB))))
}

// MarshalJSON implements json.Marshaler for stats.
func (s SemanticCacheStats) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"exact_hits":              s.ExactHits,
		"exact_misses":            s.ExactMisses,
		"semantic_hits":           s.SemanticHits,
		"semantic_misses":         s.SemanticMisses,
		"semantic_size":           s.SemanticSize,
		"similarity_distribution": s.SimilarityDistribution,
	})
}
