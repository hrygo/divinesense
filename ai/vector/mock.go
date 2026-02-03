package vector

import (
	"context"
	"math"
	"sort"
	"strings"
	"sync"
)

// MockVectorService is a mock implementation of VectorService for testing.
type MockVectorService struct {
	embeddings map[string]*storedEmbedding
	mu         sync.RWMutex
}

type storedEmbedding struct {
	Metadata map[string]any
	Vector   []float32
}

// NewMockVectorService creates a new MockVectorService with sample data.
func NewMockVectorService() *MockVectorService {
	mock := &MockVectorService{
		embeddings: make(map[string]*storedEmbedding),
	}
	mock.seedData()
	return mock
}

// seedData populates the mock with sample data for testing.
func (m *MockVectorService) seedData() {
	// Sample embeddings (simplified 8-dimensional vectors for testing)
	sampleData := []struct {
		metadata map[string]any
		docID    string
		vector   []float32
	}{
		{
			docID:  "memo-001",
			vector: []float32{0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8},
			metadata: map[string]any{
				"user_id":    int32(1),
				"content":    "今天完成了项目评审",
				"created_at": "2026-01-27T10:00:00Z",
				"tags":       []string{"工作", "项目"},
			},
		},
		{
			docID:  "memo-002",
			vector: []float32{0.15, 0.25, 0.35, 0.45, 0.55, 0.65, 0.75, 0.85},
			metadata: map[string]any{
				"user_id":    int32(1),
				"content":    "Sprint 0 接口定义完成",
				"created_at": "2026-01-27T11:00:00Z",
				"tags":       []string{"工作", "开发"},
			},
		},
		{
			docID:  "memo-003",
			vector: []float32{0.8, 0.7, 0.6, 0.5, 0.4, 0.3, 0.2, 0.1},
			metadata: map[string]any{
				"user_id":    int32(1),
				"content":    "学习了Go语言的接口设计",
				"created_at": "2026-01-26T15:00:00Z",
				"tags":       []string{"学习", "技术"},
			},
		},
		{
			docID:  "memo-004",
			vector: []float32{0.5, 0.5, 0.5, 0.5, 0.5, 0.5, 0.5, 0.5},
			metadata: map[string]any{
				"user_id":    int32(1),
				"content":    "周末计划：去公园跑步",
				"created_at": "2026-01-25T20:00:00Z",
				"tags":       []string{"生活", "运动"},
			},
		},
		{
			docID:  "memo-005",
			vector: []float32{0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 0.9},
			metadata: map[string]any{
				"user_id":    int32(2),
				"content":    "另一个用户的笔记",
				"created_at": "2026-01-27T09:00:00Z",
				"tags":       []string{"测试"},
			},
		},
	}

	for _, data := range sampleData {
		m.embeddings[data.docID] = &storedEmbedding{
			Vector:   data.vector,
			Metadata: data.metadata,
		}
	}
}

// StoreEmbedding stores a vector embedding with metadata.
func (m *MockVectorService) StoreEmbedding(ctx context.Context, docID string, vector []float32, metadata map[string]any) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.embeddings[docID] = &storedEmbedding{
		Vector:   vector,
		Metadata: metadata,
	}
	return nil
}

// SearchSimilar performs similarity search on vectors.
func (m *MockVectorService) SearchSimilar(ctx context.Context, vector []float32, limit int, filter map[string]any) ([]VectorResult, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	type scoredResult struct {
		metadata map[string]any
		docID    string
		score    float32
	}

	var results []scoredResult

	for docID, stored := range m.embeddings {
		// Apply filters
		if !matchesFilter(stored.Metadata, filter) {
			continue
		}

		// Calculate cosine similarity
		score := cosineSimilarity(vector, stored.Vector)
		results = append(results, scoredResult{
			docID:    docID,
			score:    score,
			metadata: stored.Metadata,
		})
	}

	// Sort by score descending
	sort.Slice(results, func(i, j int) bool {
		return results[i].score > results[j].score
	})

	// Apply limit
	if limit > 0 && limit < len(results) {
		results = results[:limit]
	}

	// Convert to VectorResult
	vectorResults := make([]VectorResult, len(results))
	for i, r := range results {
		vectorResults[i] = VectorResult{
			DocID:    r.docID,
			Score:    r.score,
			Metadata: r.metadata,
		}
	}

	return vectorResults, nil
}

// HybridSearch performs hybrid search combining vector and keyword search.
// Match types:
// - "keyword": exact keyword match found in content
// - "vector": no keyword match, but included via vector similarity
// In this mock, we don't do real vector embedding, so "hybrid" would require both.
func (m *MockVectorService) HybridSearch(ctx context.Context, query string, limit int) ([]SearchResult, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	type scoredResult struct {
		docID     string
		content   string
		matchType string
		score     float32
	}

	var results []scoredResult
	queryLower := strings.ToLower(query)

	for docID, stored := range m.embeddings {
		content, _ := stored.Metadata["content"].(string) //nolint:errcheck // type assertion, fallback to empty string
		contentLower := strings.ToLower(content)

		var score float32
		var matchType string

		hasKeywordMatch := strings.Contains(contentLower, queryLower)
		// Mock: assume all stored docs have some vector similarity (0.3 base score)
		hasVectorMatch := true

		if hasKeywordMatch && hasVectorMatch {
			// Both keyword and vector match = hybrid
			score = 0.9
			matchType = "hybrid"
		} else if hasKeywordMatch {
			// Only keyword match
			score = 0.8
			matchType = "keyword"
		} else if hasVectorMatch {
			// Only vector match (mock: all docs have base similarity)
			score = 0.3
			matchType = "vector"
		} else {
			continue
		}

		results = append(results, scoredResult{
			docID:     docID,
			content:   content,
			score:     score,
			matchType: matchType,
		})
	}

	// Sort by score descending
	sort.Slice(results, func(i, j int) bool {
		return results[i].score > results[j].score
	})

	// Apply limit
	if limit > 0 && limit < len(results) {
		results = results[:limit]
	}

	// Convert to SearchResult
	searchResults := make([]SearchResult, len(results))
	for i, r := range results {
		searchResults[i] = SearchResult{
			Name:      r.docID,
			Content:   r.content,
			Score:     r.score,
			MatchType: r.matchType,
		}
	}

	return searchResults, nil
}

// matchesFilter checks if metadata matches the filter conditions.
// Strict matching: if a filter key is missing from metadata, it's a non-match.
// This ensures multi-tenant isolation (e.g., missing user_id won't match any filter).
func matchesFilter(metadata, filter map[string]any) bool {
	if filter == nil {
		return true
	}

	for key, value := range filter {
		metaValue, ok := metadata[key]
		if !ok {
			// Missing filter key = non-match (strict multi-tenant isolation)
			return false
		}
		if metaValue != value {
			return false
		}
	}
	return true
}

// cosineSimilarity calculates the cosine similarity between two vectors.
// Returns a value clamped to [0, 1] to match interface contract.
func cosineSimilarity(a, b []float32) float32 {
	if len(a) != len(b) {
		return 0
	}

	var dotProduct, normA, normB float64
	for i := range a {
		dotProduct += float64(a[i]) * float64(b[i])
		normA += float64(a[i]) * float64(a[i])
		normB += float64(b[i]) * float64(b[i])
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	raw := dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))

	// Clamp to [0, 1] to match interface contract (Score: similarity score 0-1)
	if raw < 0 {
		raw = 0
	}
	if raw > 1 {
		raw = 1
	}
	return float32(raw)
}

// Ensure MockVectorService implements VectorService.
var _ VectorService = (*MockVectorService)(nil)
