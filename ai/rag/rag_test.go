package rag

import (
	"testing"

	"github.com/hrygo/divinesense/ai/router"
	"github.com/stretchr/testify/assert"
)

func TestRetrievalDecision(t *testing.T) {
	decider := NewRetrievalDecider()

	tests := []struct {
		name     string
		query    string
		reason   string
		expected bool
	}{
		// Chitchat - no retrieval
		{"Greeting", "你好", ReasonChitchat, false},
		{"Thanks", "谢谢", ReasonChitchat, false},
		{"Short", "嗯", ReasonChitchat, false},
		{"OK", "好的", ReasonChitchat, false},

		// System commands - no retrieval (need longer input to avoid chitchat match)
		{"Help command", "帮助命令", ReasonSystemCommand, false},
		{"Exit system", "退出系统", ReasonSystemCommand, false},

		// Retrieval triggers - should retrieve
		{"Search memo", "搜索我的笔记", ReasonRetrievalTrigger, true},
		{"Find notes", "查找关于 Go 的记录", ReasonRetrievalTrigger, true},
		{"Previous", "之前写过什么", ReasonRetrievalTrigger, true},

		// Schedule triggers - should retrieve
		{"Schedule query", "明天的日程", ReasonScheduleQuery, true},
		{"Meeting", "下周的会议安排", ReasonScheduleQuery, true},
		{"Today", "今天有什么安排", ReasonScheduleQuery, true},

		// Default - longer queries retrieve
		{"Long query", "帮我分析一下这个问题的解决方案", ReasonDefault, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decision := decider.Decide(tt.query)
			assert.Equal(t, tt.expected, decision.ShouldRetrieve, "query: %s", tt.query)
			if decision.ShouldRetrieve == tt.expected {
				assert.Equal(t, tt.reason, decision.Reason)
			}
		})
	}
}

func TestResultEvaluator(t *testing.T) {
	evaluator := NewResultEvaluator()

	tests := []struct {
		name           string
		expectedAction SuggestedAction
		results        []*SearchResult
		expectedUseful bool
	}{
		{
			name:           "Empty results",
			results:        []*SearchResult{},
			expectedUseful: false,
			expectedAction: ActionDirect,
		},
		{
			name: "High relevance",
			results: []*SearchResult{
				{ID: "1", Score: 0.85},
				{ID: "2", Score: 0.7},
			},
			expectedUseful: true,
			expectedAction: ActionUse,
		},
		{
			name: "Medium relevance",
			results: []*SearchResult{
				{ID: "1", Score: 0.5},
				{ID: "2", Score: 0.4},
			},
			expectedUseful: true,
			expectedAction: ActionUse,
		},
		{
			name: "Low relevance",
			results: []*SearchResult{
				{ID: "1", Score: 0.25},
				{ID: "2", Score: 0.2},
			},
			expectedUseful: false,
			expectedAction: ActionExpand,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evaluator.Evaluate(tt.results)
			assert.Equal(t, tt.expectedUseful, result.IsUseful)
			assert.Equal(t, tt.expectedAction, result.SuggestedAction)
		})
	}
}

func TestShouldRerank(t *testing.T) {
	decider := NewRerankDecider()

	tests := []struct {
		name     string
		query    string
		results  []*SearchResult
		expected bool
	}{
		{
			name:     "Too few results",
			query:    "搜索笔记",
			results:  make([]*SearchResult, 3),
			expected: false,
		},
		{
			name:  "Simple keyword query",
			query: "笔记",
			results: func() []*SearchResult {
				r := make([]*SearchResult, 5)
				for i := range r {
					r[i] = &SearchResult{Score: 0.5}
				}
				return r
			}(),
			expected: false,
		},
		{
			name:  "Large score gap",
			query: "搜索关于 Go 的笔记",
			results: []*SearchResult{
				{Score: 0.9},
				{Score: 0.5}, // Gap > 0.15
				{Score: 0.4},
				{Score: 0.3},
				{Score: 0.2},
			},
			expected: false,
		},
		{
			name:  "Close scores - should rerank",
			query: "搜索关于 Go 的笔记",
			results: []*SearchResult{
				{Score: 0.6},
				{Score: 0.55}, // Gap < 0.15
				{Score: 0.5},
				{Score: 0.45},
				{Score: 0.4},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := decider.ShouldRerank(tt.query, tt.results)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestStrategySelection(t *testing.T) {
	selector := NewStrategySelector()

	tests := []struct {
		intent   router.Intent
		expected SearchStrategy
	}{
		{router.IntentScheduleQuery, StrategyBM25Only},
		{router.IntentScheduleCreate, StrategyBM25Only},
		{router.IntentMemoSearch, StrategySemanticOnly},
		{router.IntentAmazing, StrategyFullPipeline},
		{router.IntentUnknown, StrategyHybridStandard},
	}

	for _, tt := range tests {
		t.Run(string(tt.intent), func(t *testing.T) {
			strategy := selector.Select(tt.intent)
			assert.Equal(t, tt.expected, strategy)
		})
	}
}

func TestRRFFusion(t *testing.T) {
	bm25Results := []*SearchResult{
		{ID: "doc1", Score: 0.9, Content: "Document 1"},
		{ID: "doc2", Score: 0.8, Content: "Document 2"},
		{ID: "doc3", Score: 0.7, Content: "Document 3"},
	}

	vectorResults := []*SearchResult{
		{ID: "doc2", Score: 0.95, Content: "Document 2"},
		{ID: "doc4", Score: 0.85, Content: "Document 4"},
		{ID: "doc1", Score: 0.75, Content: "Document 1"},
	}

	config := StrategyConfig{
		BM25Weight:   0.5,
		VectorWeight: 0.5,
	}

	results := FuseWithRRF(bm25Results, vectorResults, config)

	// Should have 4 unique documents
	assert.Equal(t, 4, len(results))

	// doc2 should be first (appears in both with good ranks)
	assert.Equal(t, "doc2", results[0].ID)

	// All results should have "hybrid" source
	for _, r := range results {
		assert.Equal(t, "hybrid", r.Source)
	}
}

func TestRRFEmptyInputs(t *testing.T) {
	config := StrategyConfig{BM25Weight: 0.5, VectorWeight: 0.5}

	// Empty BM25
	results := FuseWithRRF(nil, []*SearchResult{{ID: "1"}}, config)
	assert.Equal(t, 1, len(results))

	// Empty vector
	results = FuseWithRRF([]*SearchResult{{ID: "1"}}, nil, config)
	assert.Equal(t, 1, len(results))

	// Both empty
	results = FuseWithRRF(nil, nil, config)
	assert.Equal(t, 0, len(results))
}

func TestHybridSearcher(t *testing.T) {
	t.Run("BM25 only", func(t *testing.T) {
		searcher := NewHybridSearcher(StrategyBM25Only)
		bm25 := []*SearchResult{{ID: "1"}, {ID: "2"}}
		vector := []*SearchResult{{ID: "3"}, {ID: "4"}}

		results := searcher.MergeResults(bm25, vector)
		assert.Equal(t, bm25, results)
	})

	t.Run("Semantic only", func(t *testing.T) {
		searcher := NewHybridSearcher(StrategySemanticOnly)
		bm25 := []*SearchResult{{ID: "1"}, {ID: "2"}}
		vector := []*SearchResult{{ID: "3"}, {ID: "4"}}

		results := searcher.MergeResults(bm25, vector)
		assert.Equal(t, vector, results)
	})

	t.Run("Hybrid merges both", func(t *testing.T) {
		searcher := NewHybridSearcher(StrategyHybridStandard)
		bm25 := []*SearchResult{{ID: "1"}, {ID: "2"}}
		vector := []*SearchResult{{ID: "2"}, {ID: "3"}}

		results := searcher.MergeResults(bm25, vector)
		// Should have 3 unique documents
		assert.Equal(t, 3, len(results))
	})
}

// Benchmark tests.
func BenchmarkRetrievalDecision(b *testing.B) {
	decider := NewRetrievalDecider()
	query := "搜索我之前写的关于 Go 编程的笔记"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		decider.Decide(query)
	}
}

func BenchmarkRRFFusion(b *testing.B) {
	bm25 := make([]*SearchResult, 100)
	vector := make([]*SearchResult, 100)
	for i := 0; i < 100; i++ {
		bm25[i] = &SearchResult{ID: string(rune('a' + i%26)), Score: float32(100-i) / 100}
		vector[i] = &SearchResult{ID: string(rune('a' + (i+5)%26)), Score: float32(100-i) / 100}
	}
	config := StrategyConfig{BM25Weight: 0.5, VectorWeight: 0.5}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		FuseWithRRF(bm25, vector, config)
	}
}
