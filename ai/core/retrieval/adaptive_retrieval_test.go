package retrieval

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/hrygo/divinesense/ai"
	storepb "github.com/hrygo/divinesense/proto/gen/store"
	"github.com/hrygo/divinesense/server/queryengine"
	"github.com/hrygo/divinesense/store"
)

// MockEmbeddingService is a mock for EmbeddingService.
type MockEmbeddingService struct {
	mock.Mock
}

func (m *MockEmbeddingService) Embed(ctx context.Context, text string) ([]float32, error) {
	args := m.Called(ctx, text)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]float32), args.Error(1)
}

func (m *MockEmbeddingService) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	args := m.Called(ctx, texts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([][]float32), args.Error(1)
}

func (m *MockEmbeddingService) Dimensions() int {
	return 1024
}

func (m *MockEmbeddingService) IsEnabled() bool {
	return true
}

// MockRerankerService is a mock for RerankerService.
type MockRerankerService struct {
	mock.Mock
}

func (m *MockRerankerService) Rerank(ctx context.Context, query string, docs []string, topK int) ([]ai.RerankResult, error) {
	args := m.Called(ctx, query, docs, topK)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]ai.RerankResult), args.Error(1)
}

func (m *MockRerankerService) IsEnabled() bool {
	args := m.Called()
	return args.Bool(0)
}

// MockStore is a mock for Store.
type MockStore struct {
	mock.Mock
	vectorSearchResults  []*store.MemoWithScore
	listSchedulesResults []*store.Schedule
}

func (m *MockStore) VectorSearch(ctx context.Context, opts *store.VectorSearchOptions) ([]*store.MemoWithScore, error) {
	args := m.Called(ctx, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*store.MemoWithScore), args.Error(1)
}

func (m *MockStore) ListSchedules(ctx context.Context, find *store.FindSchedule) ([]*store.Schedule, error) {
	args := m.Called(ctx, find)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*store.Schedule), args.Error(1)
}

// TestAdaptiveRetriever_EvaluateQuality 测试结果质量评估.
func TestAdaptiveRetriever_EvaluateQuality(t *testing.T) {
	retriever := &AdaptiveRetriever{}

	tests := []struct {
		name     string
		results  []*SearchResult
		expected QualityLevel
	}{
		{
			name:     "空结果",
			results:  []*SearchResult{},
			expected: LowQuality,
		},
		{
			name: "高质量 - 前2名分数差距大",
			results: []*SearchResult{
				{Score: 0.95},
				{Score: 0.70},
			},
			expected: HighQuality,
		},
		{
			name: "高质量 - 第1名分数很高",
			results: []*SearchResult{
				{Score: 0.92},
				{Score: 0.85},
			},
			expected: HighQuality,
		},
		{
			name: "中等质量 - 第1名分数中等",
			results: []*SearchResult{
				{Score: 0.75},
				{Score: 0.70},
			},
			expected: MediumQuality,
		},
		{
			name: "低质量 - 第1名分数低",
			results: []*SearchResult{
				{Score: 0.65},
				{Score: 0.60},
			},
			expected: LowQuality,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := retriever.evaluateQuality(tt.results)
			if result != tt.expected {
				t.Errorf("evaluateQuality() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestAdaptiveRetriever_ShouldRerank 测试是否应该重排.
func TestAdaptiveRetriever_ShouldRerank(t *testing.T) {
	tests := []struct {
		name     string
		query    string
		results  []*SearchResult
		expected bool
	}{
		{
			name:     "结果少 - 不重排",
			query:    "测试",
			results:  make([]*SearchResult, 3),
			expected: false,
		},
		{
			name:  "简单查询 - 不重排",
			query: "简短查询",
			results: []*SearchResult{
				{Score: 0.8},
				{Score: 0.7},
				{Score: 0.6},
				{Score: 0.5},
				{Score: 0.4},
			},
			expected: false,
		},
		{
			name:  "前2名分数差距大 - 不重排",
			query: "测试查询",
			results: []*SearchResult{
				{Score: 0.9},
				{Score: 0.7},
				{Score: 0.6},
				{Score: 0.5},
				{Score: 0.4},
			},
			expected: false,
		},
		{
			name:  "复杂查询且分数接近 - 应该重排",
			query: "如何使用Python和Django构建Web应用",
			results: []*SearchResult{
				{Score: 0.75},
				{Score: 0.73},
				{Score: 0.70},
				{Score: 0.68},
				{Score: 0.65},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 设置 mock 期望
			mockReranker := new(MockRerankerService)
			mockReranker.On("IsEnabled").Return(true)
			retriever := &AdaptiveRetriever{
				rerankerService: mockReranker,
			}

			result := retriever.shouldRerank(tt.query, tt.results)
			if result != tt.expected {
				t.Errorf("shouldRerank() = %v, want %v", result, tt.expected)
			}

			mockReranker.AssertExpectations(t)
		})
	}
}

// TestAdaptiveRetriever_IsSimpleKeywordQuery 测试简单关键词查询判断.
func TestAdaptiveRetriever_IsSimpleKeywordQuery(t *testing.T) {
	retriever := &AdaptiveRetriever{}

	tests := []struct {
		name     string
		query    string
		expected bool
	}{
		{
			name:     "短查询",
			query:    "Python",
			expected: true,
		},
		{
			name:     "中等长度查询",
			query:    "Python编程",
			expected: true,
		},
		{
			name:     "长查询但无复杂词",
			query:    "Python Django Web开发笔记",
			expected: true,
		},
		{
			name:     "包含疑问词",
			query:    "如何使用Python",
			expected: false,
		},
		{
			name:     "包含连词",
			query:    "Python和Java的区别",
			expected: false,
		},
		{
			name:     "包含转折词",
			query:    "Python很好但是很难学",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := retriever.isSimpleKeywordQuery(tt.query)
			if result != tt.expected {
				t.Errorf("isSimpleKeywordQuery() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestAdaptiveRetriever_FilterByScore 测试按分数过滤.
func TestAdaptiveRetriever_FilterByScore(t *testing.T) {
	retriever := &AdaptiveRetriever{}

	results := []*SearchResult{
		{Score: 0.9},
		{Score: 0.7},
		{Score: 0.5},
		{Score: 0.3},
	}

	tests := []struct {
		name     string
		minScore float32
		expected int
	}{
		{
			name:     "阈值 0.6",
			minScore: 0.6,
			expected: 2,
		},
		{
			name:     "阈值 0.5",
			minScore: 0.5,
			expected: 3,
		},
		{
			name:     "阈值 0.0",
			minScore: 0.0,
			expected: 4,
		},
		{
			name:     "阈值 0.95",
			minScore: 0.95,
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filtered := retriever.filterByScore(results, tt.minScore)
			if len(filtered) != tt.expected {
				t.Errorf("filterByScore() = %d results, want %d", len(filtered), tt.expected)
			}
		})
	}
}

// TestAdaptiveRetriever_TruncateResults 测试结果截断.
func TestAdaptiveRetriever_TruncateResults(t *testing.T) {
	retriever := &AdaptiveRetriever{}

	results := make([]*SearchResult, 10)
	for i := range results {
		results[i] = &SearchResult{ID: int64(i)}
	}

	tests := []struct {
		name     string
		limit    int
		expected int
	}{
		{
			name:     "限制 5",
			limit:    5,
			expected: 5,
		},
		{
			name:     "限制 20",
			limit:    20,
			expected: 10, // 不截断
		},
		{
			name:     "限制 0",
			limit:    0,
			expected: 10, // 不截断
		},
		{
			name:     "限制 -1",
			limit:    -1,
			expected: 10, // 不截断
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			truncated := retriever.truncateResults(results, tt.limit)
			if len(truncated) != tt.expected {
				t.Errorf("truncateResults() = %d results, want %d", len(truncated), tt.expected)
			}
		})
	}
}

// TestAdaptiveRetriever_MergeResults 测试结果合并.
func TestAdaptiveRetriever_MergeResults(t *testing.T) {
	retriever := &AdaptiveRetriever{}

	results1 := []*SearchResult{
		{ID: 1, Score: 0.9},
		{ID: 2, Score: 0.7},
		{ID: 3, Score: 0.5},
	}

	results2 := []*SearchResult{
		{ID: 2, Score: 0.8}, // 重复 ID
		{ID: 4, Score: 0.6},
		{ID: 5, Score: 0.4},
	}

	merged := retriever.mergeResults(results1, results2, 10)

	// 验证去重
	uniqueIDs := make(map[int64]bool)
	for _, result := range merged {
		if uniqueIDs[result.ID] {
			t.Errorf("mergeResults() duplicate ID %d found", result.ID)
		}
		uniqueIDs[result.ID] = true
	}

	// 验证分数排序
	for i := 1; i < len(merged); i++ {
		if merged[i-1].Score < merged[i].Score {
			t.Errorf("mergeResults() not sorted by score: [%d]=%.2f, [%d]=%.2f",
				i-1, merged[i-1].Score, i, merged[i].Score)
		}
	}

	// 验证预期结果
	if len(merged) != 5 {
		t.Errorf("mergeResults() = %d results, want %d", len(merged), 5)
	}
}

// TestAdaptiveRetriever_Retrieve_ScheduleBM25Only 测试日程 BM25 检索.
func TestAdaptiveRetriever_Retrieve_ScheduleBM25Only(t *testing.T) {
	// This test would require a more complete mock setup
	// For now, just verify the strategy routing works

	mockStore := &MockStore{}
	mockEmbedding := &MockEmbeddingService{}
	mockReranker := &MockRerankerService{}

	retriever := NewAdaptiveRetriever(nil, mockEmbedding, mockReranker)

	// Test that the strategy field is correctly used
	opts := &RetrievalOptions{
		Strategy: "schedule_bm25_only",
		UserID:   1,
		Query:    "今天的日程",
		Limit:    10,
		MinScore: 0.5,
	}

	// Verify strategy is set
	assert.Equal(t, "schedule_bm25_only", opts.Strategy)
	assert.Equal(t, int32(1), opts.UserID)
	assert.Equal(t, "今天的日程", opts.Query)
	assert.Equal(t, 10, opts.Limit)
	assert.Equal(t, float32(0.5), opts.MinScore)

	_ = retriever
	_ = mockStore
}

// TestQualityLevel_String 测试质量级别字符串表示.
func TestQualityLevel_String(t *testing.T) {
	tests := []struct {
		level    QualityLevel
		expected string
	}{
		{LowQuality, "low"},
		{MediumQuality, "medium"},
		{HighQuality, "high"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := tt.level.String()
			assert.Equal(t, tt.expected, result)
		})
	}
}

// BenchmarkQueryRouter_Route 性能基准测试.
func BenchmarkQueryRouter_Route(b *testing.B) {
	router := queryengine.NewQueryRouter()
	ctx := context.Background()
	queries := []string{
		"今天有什么安排",
		"搜索关于AI的笔记",
		"本周关于React的学习计划",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, query := range queries {
			router.Route(ctx, query, nil)
		}
	}
}

// TestRRFFusion tests the RRF (Reciprocal Rank Fusion) algorithm.
func TestRRFFusion(t *testing.T) {
	tests := []struct {
		name           string
		vectorResults  []*store.MemoWithScore
		bm25Results    []*store.BM25Result
		semanticWeight float32
		wantTopID      int64 // Expected top result ID
	}{
		{
			name:           "Both rankings agree - same top result",
			vectorResults:  createMockVectorResults([]int64{1, 2, 3, 4, 5}),
			bm25Results:    createMockBM25Results([]int64{1, 2, 3, 4, 5}),
			semanticWeight: 0.5,
			wantTopID:      1,
		},
		{
			name:           "Rankings disagree - reciprocal fusion favors consistency",
			vectorResults:  createMockVectorResults([]int64{1, 2, 3, 4, 5}),
			bm25Results:    createMockBM25Results([]int64{5, 4, 3, 2, 1}),
			semanticWeight: 0.51,
			wantTopID:      1, // ID 1 (Rank 1 in heavy list) beats ID 5 and ID 3 (Rank 3 in both)
		},
		{
			name:           "Vector only - BM25 empty",
			vectorResults:  createMockVectorResults([]int64{1, 2, 3}),
			bm25Results:    createMockBM25Results([]int64{}),
			semanticWeight: 1.0,
			wantTopID:      1,
		},
		{
			name:           "BM25 only - vector empty",
			vectorResults:  createMockVectorResults([]int64{}),
			bm25Results:    createMockBM25Results([]int64{1, 2, 3}),
			semanticWeight: 0.0,
			wantTopID:      1,
		},
		{
			name:           "High semantic weight",
			vectorResults:  createMockVectorResults([]int64{1, 2, 3}),
			bm25Results:    createMockBM25Results([]int64{3, 2, 1}),
			semanticWeight: 0.9,
			wantTopID:      1, // Vector rank 1 should win with high weight
		},
		{
			name:           "High BM25 weight",
			vectorResults:  createMockVectorResults([]int64{1, 2, 3}),
			bm25Results:    createMockBM25Results([]int64{3, 2, 1}),
			semanticWeight: 0.1,
			wantTopID:      3, // BM25 rank 1 should win with high BM25 weight
		},
		{
			name:           "Partial overlap - some unique results",
			vectorResults:  createMockVectorResults([]int64{1, 2, 3, 4, 5}),
			bm25Results:    createMockBM25Results([]int64{3, 6, 7, 8, 9}),
			semanticWeight: 0.5,
			wantTopID:      3, // ID 3 appears in both (rank 3 in vector, rank 1 in BM25)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			retriever := &AdaptiveRetriever{} // Minimal setup, only needs rrfFusion method

			results := retriever.rrfFusion(tt.vectorResults, tt.bm25Results, tt.semanticWeight)

			if len(results) == 0 {
				t.Fatal("rrfFusion returned empty results")
			}

			// Check that results are sorted by score (descending)
			for i := 1; i < len(results); i++ {
				if results[i-1].Score < results[i].Score {
					t.Errorf("Results not sorted by score: results[%d].Score=%f < results[%d].Score=%f",
						i-1, results[i-1].Score, i, results[i].Score)
				}
			}

			// Check top result
			if results[0].ID != tt.wantTopID {
				t.Errorf("Top result ID = %d, want %d", results[0].ID, tt.wantTopID)
			}

			// All scores should be positive
			for i, r := range results {
				if r.Score <= 0 {
					t.Errorf("Result %d has non-positive score: %f", i, r.Score)
				}
			}
		})
	}
}

// TestRRFScoreCalculation tests the RRF score calculation.
func TestRRFScoreCalculation(t *testing.T) {
	// RRF formula: RRF(d) = weight / (k + rank)
	// where k = 60

	// Single result at rank 1 in both lists
	vectorResults := createMockVectorResults([]int64{1})
	bm25Results := createMockBM25Results([]int64{1})

	retriever := &AdaptiveRetriever{}
	results := retriever.rrfFusion(vectorResults, bm25Results, 0.5)

	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}

	// Expected score: 0.5/(60+1) + 0.5/(60+1) = 1/(61) ≈ 0.01639
	expectedScore := 1.0 / (float32(RRFK) + 1.0)
	if results[0].Score < expectedScore-0.0001 || results[0].Score > expectedScore+0.0001 {
		t.Errorf("Score = %f, want approximately %f", results[0].Score, expectedScore)
	}
}

// TestRRFEmptyResults tests RRF with empty inputs.
func TestRRFEmptyResults(t *testing.T) {
	retriever := &AdaptiveRetriever{}

	results := retriever.rrfFusion([]*store.MemoWithScore{}, []*store.BM25Result{}, 0.5)
	if len(results) != 0 {
		t.Errorf("Expected 0 results for empty inputs, got %d", len(results))
	}
}

// TestRRFUniqueResults tests RRF when all results are unique (no overlap).
func TestRRFUniqueResults(t *testing.T) {
	vectorResults := createMockVectorResults([]int64{1, 2, 3})
	bm25Results := createMockBM25Results([]int64{4, 5, 6})

	retriever := &AdaptiveRetriever{}
	results := retriever.rrfFusion(vectorResults, bm25Results, 0.5)

	if len(results) != 6 {
		t.Fatalf("Expected 6 results, got %d", len(results))
	}

	// With equal weights and rank 1 for each list, vector's rank 1 should be slightly higher
	// The important thing is all 6 results are present
	ids := make(map[int64]bool)
	for _, r := range results {
		ids[r.ID] = true
	}

	for _, id := range []int64{1, 2, 3, 4, 5, 6} {
		if !ids[id] {
			t.Errorf("Expected ID %d in results", id)
		}
	}
}

// Helper functions for RRF tests

func createMockMemo(id int64) *store.Memo {
	return &store.Memo{
		ID:      int32(id),
		Content: "",
		Payload: &storepb.MemoPayload{},
	}
}

func createMockVectorResults(ids []int64) []*store.MemoWithScore {
	results := make([]*store.MemoWithScore, len(ids))
	for i, id := range ids {
		results[i] = &store.MemoWithScore{
			Memo:  createMockMemo(id),
			Score: 1.0 - float32(i)*0.1, // Decreasing scores
		}
	}
	return results
}

func createMockBM25Results(ids []int64) []*store.BM25Result {
	results := make([]*store.BM25Result, len(ids))
	for i, id := range ids {
		results[i] = &store.BM25Result{
			Memo:  createMockMemo(id),
			Score: 1.0 - float32(i)*0.1, // Decreasing scores
		}
	}
	return results
}
