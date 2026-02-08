package retrieval

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	storepb "github.com/hrygo/divinesense/proto/gen/store"
	"github.com/hrygo/divinesense/server/queryengine"
	"github.com/hrygo/divinesense/store"
)

// TestAdaptiveRetriever_Retrieve_InvalidInput tests input validation.
func TestAdaptiveRetriever_Retrieve_InvalidInput(t *testing.T) {
	mockEmbedding := &MockEmbeddingService{}
	mockReranker := &MockRerankerService{}

	retriever := NewAdaptiveRetriever(nil, mockEmbedding, mockReranker)

	// Create a query longer than 1000 characters
	longQuery := string(make([]byte, 1001))
	_ = longQuery // Use the variable

	opts := &RetrievalOptions{
		Query: longQuery,
	}

	ctx := context.Background()
	_, err := retriever.Retrieve(ctx, opts)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "query too long")
}

// TestAdaptiveRetriever_ConvertResults tests result conversion functions.
func TestAdaptiveRetriever_ConvertResults(t *testing.T) {
	retriever := &AdaptiveRetriever{}

	t.Run("ConvertVectorResults", func(t *testing.T) {
		memos := []*store.MemoWithScore{
			{
				Memo: &store.Memo{
					ID:      1,
					Content: "test content",
					Payload: &storepb.MemoPayload{},
				},
				Score: 0.9,
			},
		}

		results := retriever.convertVectorResults(memos)

		require.Len(t, results, 1)
		assert.Equal(t, int64(1), results[0].ID)
		assert.Equal(t, "memo", results[0].Type)
		assert.Equal(t, float32(0.9), results[0].Score)
		assert.Equal(t, "test content", results[0].Content)
	})

	t.Run("ConvertBM25Results", func(t *testing.T) {
		memos := []*store.BM25Result{
			{
				Memo: &store.Memo{
					ID:      2,
					Content: "bm25 content",
					Payload: &storepb.MemoPayload{},
				},
				Score: 0.8,
			},
		}

		results := retriever.convertBM25Results(memos)

		require.Len(t, results, 1)
		assert.Equal(t, int64(2), results[0].ID)
		assert.Equal(t, "memo", results[0].Type)
		assert.Equal(t, float32(0.8), results[0].Score)
	})
}

// TestSearchResult_Creation tests SearchResult struct initialization.
func TestSearchResult_Creation(t *testing.T) {
	t.Run("Memo result", func(t *testing.T) {
		memo := &store.Memo{
			ID: 123,
		}
		result := &SearchResult{
			ID:       123,
			Type:     "memo",
			Score:    0.85,
			Memo:     memo,
			Schedule: nil,
		}

		assert.Equal(t, int64(123), result.ID)
		assert.Equal(t, "memo", result.Type)
		assert.Equal(t, float32(0.85), result.Score)
		assert.NotNil(t, result.Memo)
		assert.Nil(t, result.Schedule)
	})

	t.Run("Schedule result", func(t *testing.T) {
		schedule := &store.Schedule{
			ID: 456,
		}

		result := &SearchResult{
			ID:       456,
			Type:     "schedule",
			Score:    1.0,
			Memo:     nil,
			Schedule: schedule,
		}

		assert.Equal(t, int64(456), result.ID)
		assert.Equal(t, "schedule", result.Type)
		assert.Equal(t, float32(1.0), result.Score)
		assert.Nil(t, result.Memo)
		assert.NotNil(t, result.Schedule)
	})
}

// TestGenerateRequestID tests request ID generation.
func TestGenerateRequestID(t *testing.T) {
	ids := make(map[string]bool)

	// Generate 100 IDs, check for uniqueness
	for i := 0; i < 100; i++ {
		id := generateRequestID()
		if ids[id] {
			t.Errorf("duplicate request ID generated: %s", id)
		}
		ids[id] = true
		assert.NotEmpty(t, id)
	}
}

// TestQualityLevels tests QualityLevel String method.
func TestQualityLevels(t *testing.T) {
	tests := []struct {
		level    QualityLevel
		expected string
	}{
		{LowQuality, "low"},
		{MediumQuality, "medium"},
		{HighQuality, "high"},
		{QualityLevel(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.level.String())
		})
	}
}

// TestRetrievalOptions_TimeRangeValidation tests time range validation.
func TestRetrievalOptions_TimeRangeValidation(t *testing.T) {
	mockEmbedding := &MockEmbeddingService{}
	mockReranker := &MockRerankerService{}

	retriever := NewAdaptiveRetriever(nil, mockEmbedding, mockReranker)

	ctx := context.Background()

	t.Run("Invalid time range - start after end", func(t *testing.T) {
		opts := &RetrievalOptions{
			Strategy: "schedule_bm25_only",
			UserID:   1,
			Query:    "今天的日程",
			TimeRange: &queryengine.TimeRange{
				Start: time.Now(),
				End:   time.Now().Add(-24 * time.Hour),
			},
		}

		_, err := retriever.Retrieve(ctx, opts)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid time range")
	})
}

// Benchmark_RRF_Fusion benchmarks RRF fusion performance.
func Benchmark_RRF_Fusion(b *testing.B) {
	retriever := &AdaptiveRetriever{}
	vectorResults := createMockVectorResults([]int64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10})
	bm25Results := createMockBM25Results([]int64{10, 9, 8, 7, 6, 5, 4, 3, 2, 1})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = retriever.rrfFusion(vectorResults, bm25Results, 0.5)
	}
}

// Benchmark_EvaluateQuality benchmarks quality evaluation.
func Benchmark_EvaluateQuality(b *testing.B) {
	retriever := &AdaptiveRetriever{}
	results := make([]*SearchResult, 100)
	for i := range results {
		results[i] = &SearchResult{Score: float32(i) / 100}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = retriever.evaluateQuality(results)
	}
}
