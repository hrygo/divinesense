// Package cache provides unit tests for semantic caching.
package cache

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockEmbeddingService is a mock implementation of EmbeddingService.
type mockEmbeddingService struct {
	embeddings map[string][]float32
}

func newMockEmbeddingService() *mockEmbeddingService {
	return &mockEmbeddingService{
		embeddings: make(map[string][]float32),
	}
}

func (m *mockEmbeddingService) Embed(ctx context.Context, text string) ([]float32, error) {
	// Return a deterministic vector based on text
	if vec, ok := m.embeddings[text]; ok {
		return vec, nil
	}
	// Generate a simple vector (for testing)
	vec := make([]float32, 128)
	for i := range vec {
		vec[i] = 0.1
	}
	m.embeddings[text] = vec
	return vec, nil
}

func (m *mockEmbeddingService) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	result := make([][]float32, len(texts))
	for i, text := range texts {
		vec, err := m.Embed(ctx, text)
		if err != nil {
			return nil, err
		}
		result[i] = vec
	}
	return result, nil
}

func (m *mockEmbeddingService) Dimensions() int {
	return 128
}

func TestNewSemanticCache(t *testing.T) {
	cfg := SemanticCacheConfig{
		MaxEntries:          100,
		SimilarityThreshold: 0.95,
		TTL:                 time.Hour,
		EmbeddingService:    newMockEmbeddingService(),
	}

	cache := NewSemanticCache(cfg)

	assert.NotNil(t, cache)
	assert.NotNil(t, cache.exactCache)
	assert.NotNil(t, cache.semanticCache.entries)
}

func TestSemanticCache_GetSet(t *testing.T) {
	ctx := context.Background()
	mockSvc := newMockEmbeddingService()

	cfg := SemanticCacheConfig{
		MaxEntries:          100,
		SimilarityThreshold: 0.95,
		TTL:                 time.Hour,
		EmbeddingService:    mockSvc,
	}

	cache := NewSemanticCache(cfg)

	text := "test query"
	vec, err := mockSvc.Embed(ctx, text)
	require.NoError(t, err)

	// Initially should miss
	_, found, sim, exact := cache.Get(ctx, text)
	assert.False(t, found)
	assert.Zero(t, sim)
	assert.False(t, exact)

	// Set and then hit (exact match)
	err = cache.Set(ctx, text, vec)
	require.NoError(t, err)

	result, found, sim, exact := cache.Get(ctx, text)
	assert.True(t, found)
	assert.Equal(t, vec, result)
	assert.Equal(t, float32(1.0), sim)
	assert.True(t, exact)
}

func TestSemanticCache_SemanticMatch(t *testing.T) {
	ctx := context.Background()
	mockSvc := newMockEmbeddingService()

	cfg := SemanticCacheConfig{
		MaxEntries:          100,
		SimilarityThreshold: 0.95,
		TTL:                 time.Hour,
		EmbeddingService:    mockSvc,
	}

	cache := NewSemanticCache(cfg)

	// Set a known text with high similarity vector
	text1 := "如何创建笔记"
	vec1 := []float32{0.5, 0.5, 0.5, 0.5}
	err := cache.Set(ctx, text1, vec1)
	require.NoError(t, err)

	// Create a similar query with very close embedding
	text2 := "怎么新建笔记"
	vec2 := []float32{0.51, 0.5, 0.5, 0.5} // Very similar - cos similarity ≈ 0.99
	err = cache.Set(ctx, text2, vec2)
	require.NoError(t, err)

	// Query with similar text should find a semantic match if above threshold
	result, found, sim, _ := cache.Get(ctx, "如何创建笔记")
	assert.True(t, found, "should find a match")
	assert.NotNil(t, result)
	assert.GreaterOrEqual(t, sim, float32(0.95), "similarity should be above threshold")
}

func TestSemanticCache_Expiration(t *testing.T) {
	ctx := context.Background()
	mockSvc := newMockEmbeddingService()

	cfg := SemanticCacheConfig{
		MaxEntries:          100,
		SimilarityThreshold: 0.95,
		TTL:                 100 * time.Millisecond,
		EmbeddingService:    mockSvc,
	}

	cache := NewSemanticCache(cfg)

	text := "test query"
	vec, err := mockSvc.Embed(ctx, text)
	require.NoError(t, err)

	err = cache.Set(ctx, text, vec)
	require.NoError(t, err)

	// Should hit immediately
	_, found, _, _ := cache.Get(ctx, text)
	assert.True(t, found)

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	// Should miss after expiration
	_, found, _, _ = cache.Get(ctx, text)
	assert.False(t, found)
}

func TestSemanticCache_Stats(t *testing.T) {
	ctx := context.Background()
	mockSvc := newMockEmbeddingService()

	cfg := SemanticCacheConfig{
		MaxEntries:          100,
		SimilarityThreshold: 0.95,
		TTL:                 time.Hour,
		EmbeddingService:    mockSvc,
	}

	cache := NewSemanticCache(cfg)

	text := "test query"
	vec, err := mockSvc.Embed(ctx, text)
	require.NoError(t, err)

	// Miss
	cache.Get(ctx, text)

	// Set and hit
	cache.Set(ctx, text, vec)
	cache.Get(ctx, text)

	stats := cache.GetStats()
	assert.Equal(t, int64(1), stats.ExactHits)
	assert.Equal(t, int64(1), stats.ExactMisses)
}

func TestSemanticCache_Clear(t *testing.T) {
	ctx := context.Background()
	mockSvc := newMockEmbeddingService()

	cfg := SemanticCacheConfig{
		MaxEntries:          100,
		SimilarityThreshold: 0.95,
		TTL:                 time.Hour,
		EmbeddingService:    mockSvc,
	}

	cache := NewSemanticCache(cfg)

	text := "test query"
	vec, err := mockSvc.Embed(ctx, text)
	require.NoError(t, err)

	cache.Set(ctx, text, vec)

	// Verify cached - records a hit
	_, found, _, _ := cache.Get(ctx, text)
	assert.True(t, found)

	// Clear
	cache.Clear()

	// Stats should be reset immediately after Clear
	stats := cache.GetStats()
	assert.Equal(t, int64(0), stats.ExactHits)
	assert.Equal(t, int64(0), stats.ExactMisses)
	assert.Equal(t, 0, stats.SemanticSize)

	// Should miss after clear
	_, found, _, _ = cache.Get(ctx, text)
	assert.False(t, found)

	// Now there will be 1 miss from the Get after Clear
	stats = cache.GetStats()
	assert.Equal(t, int64(0), stats.ExactHits)
	assert.Equal(t, int64(1), stats.ExactMisses)
}

func TestCosineSimilarity(t *testing.T) {
	// Test identical vectors
	a := []float32{1.0, 2.0, 3.0}
	b := []float32{1.0, 2.0, 3.0}
	sim := cosineSimilarity(a, b)
	assert.InDelta(t, 1.0, sim, 0.001)

	// Test orthogonal vectors
	c := []float32{1.0, 0.0, 0.0}
	d := []float32{0.0, 1.0, 0.0}
	sim = cosineSimilarity(c, d)
	assert.InDelta(t, 0.0, sim, 0.001)

	// Test opposite vectors
	e := []float32{1.0, 0.0, 0.0}
	f := []float32{-1.0, 0.0, 0.0}
	sim = cosineSimilarity(e, f)
	assert.InDelta(t, -1.0, sim, 0.001)

	// Test different lengths - should return 0
	g := []float32{1.0, 2.0}
	h := []float32{1.0, 2.0, 3.0}
	sim = cosineSimilarity(g, h)
	assert.Equal(t, float32(0), sim)
}

func TestCosineSimilarity_ZeroVectors(t *testing.T) {
	zeroVec := []float32{0.0, 0.0, 0.0}
	nonZero := []float32{1.0, 0.0, 0.0}

	sim := cosineSimilarity(zeroVec, nonZero)
	assert.Equal(t, float32(0), sim)

	sim = cosineSimilarity(nonZero, zeroVec)
	assert.Equal(t, float32(0), sim)
}
