// Package routing provides unit tests for HistoryMatcher.
package routing

import (
	"testing"
)

// TestNewHistoryMatcher tests HistoryMatcher creation.
func TestNewHistoryMatcher(t *testing.T) {
	matcher := NewHistoryMatcher(nil)

	if matcher == nil {
		t.Fatal("expected non-nil HistoryMatcher")
	}
	if matcher.similarityThreshold != 0.8 {
		t.Errorf("expected similarity threshold 0.8, got %f", matcher.similarityThreshold)
	}
	if matcher.semanticThreshold != 0.75 {
		t.Errorf("expected semantic threshold 0.75, got %f", matcher.semanticThreshold)
	}
}

// TestHistoryMatcher_Match tests matching (currently disabled).
func TestHistoryMatcher_Match(t *testing.T) {
	matcher := NewHistoryMatcher(nil)

	result, err := matcher.Match(nil, 123, "搜索笔记")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.Matched {
		t.Error("expected no match (feature disabled)")
	}
}

// TestHistoryMatcher_CalculateLexicalSimilarity tests lexical similarity calculation.
func TestHistoryMatcher_CalculateLexicalSimilarity(t *testing.T) {
	matcher := NewHistoryMatcher(nil)

	testCases := []struct {
		input1        string
		input2        string
		minSimilarity float32
	}{
		{"搜索笔记", "搜索笔记", 1.0},
		{"搜索笔记", "查找笔记", 0.2},
		{"明天会议", "后天会议", 0.5},
		{"搜索笔记", "明天会议", 0.0},
		{"", "", 0.0},
	}

	for _, tc := range testCases {
		similarity := matcher.calculateLexicalSimilarity(tc.input1, tc.input2)
		if similarity < tc.minSimilarity {
			t.Errorf("calculateLexicalSimilarity(%q, %q) = %f, expected >= %f",
				tc.input1, tc.input2, similarity, tc.minSimilarity)
		}
	}
}

// TestHistoryMatcher_ExtractBigrams tests bigram extraction.
func TestHistoryMatcher_ExtractBigrams(t *testing.T) {
	matcher := NewHistoryMatcher(nil)

	testCases := []struct {
		input      string
		minBigrams int
	}{
		{"搜索", 1},
		{"搜索笔记", 3},
		{"明天有会议", 4},
		{"", 0},
	}

	for _, tc := range testCases {
		bigrams := matcher.extractBigrams(tc.input)
		if len(bigrams) < tc.minBigrams {
			t.Errorf("extractBigrams(%q) = %d bigrams, expected >= %d",
				tc.input, len(bigrams), tc.minBigrams)
		}
	}
}

// TestHistoryMatcher_AgentTypeToIntent tests agent type to intent conversion.
func TestHistoryMatcher_AgentTypeToIntent(t *testing.T) {
	matcher := NewHistoryMatcher(nil)

	testCases := []struct {
		agentType string
		input     string
		expected  Intent
	}{
		{"schedule", "查看明天会议", IntentScheduleQuery},
		{"schedule", "修改会议", IntentScheduleUpdate},
		{"schedule", "创建日程", IntentScheduleCreate},
		{"memo", "搜索笔记", IntentMemoSearch},
		{"memo", "记录内容", IntentMemoCreate},
		{"unknown", "随便说", IntentUnknown},
	}

	for _, tc := range testCases {
		result := matcher.agentTypeToIntent(tc.agentType, tc.input)
		if result != tc.expected {
			t.Errorf("agentTypeToIntent(%q, %q) = %s, expected %s",
				tc.agentType, tc.input, result, tc.expected)
		}
	}
}

// TestHistoryMatcher_IntentToAgentType tests intent to agent type conversion.
func TestHistoryMatcher_IntentToAgentType(t *testing.T) {
	matcher := NewHistoryMatcher(nil)

	testCases := []struct {
		intent   Intent
		expected string
	}{
		{IntentMemoSearch, "memo"},
		{IntentMemoCreate, "memo"},
		{IntentScheduleQuery, "schedule"},
		{IntentScheduleCreate, "schedule"},
		{IntentScheduleUpdate, "schedule"},
		{IntentBatchSchedule, "schedule"},
		{IntentUnknown, "unknown"},
	}

	for _, tc := range testCases {
		result := matcher.intentToAgentType(tc.intent)
		if result != tc.expected {
			t.Errorf("intentToAgentType(%s) = %s, expected %s",
				tc.intent, result, tc.expected)
		}
	}
}

// TestHistoryMatcher_SaveDecision tests saving routing decisions (no-op).
func TestHistoryMatcher_SaveDecision(t *testing.T) {
	matcher := NewHistoryMatcher(nil)

	err := matcher.SaveDecision(nil, 123, "搜索笔记", IntentMemoSearch, true)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

// TestCosineSimilarity tests cosine similarity calculation.
func TestCosineSimilarity(t *testing.T) {
	testCases := []struct {
		a        []float32
		b        []float32
		expected float32
		delta    float32
	}{
		{[]float32{1, 0, 0}, []float32{1, 0, 0}, 1.0, 0.001},
		{[]float32{1, 0, 0}, []float32{0, 1, 0}, 0.0, 0.001},
		{[]float32{1, 0, 0}, []float32{-1, 0, 0}, -1.0, 0.001},
		{[]float32{1, 1}, []float32{1, 1}, 1.0, 0.001},
		{[]float32{1}, []float32{1, 1}, 0.0, 0.001},
		{[]float32{}, []float32{1}, 0.0, 0.001},
	}

	for _, tc := range testCases {
		result := cosineSimilarity(tc.a, tc.b)
		diff := result - tc.expected
		if diff < 0 {
			diff = -diff
		}
		if diff > tc.delta {
			t.Errorf("cosineSimilarity(%v, %v) = %f, expected %f ± %f",
				tc.a, tc.b, result, tc.expected, tc.delta)
		}
	}
}

// BenchmarkHistoryMatcher_LexicalSimilarity benchmarks lexical similarity.
func BenchmarkHistoryMatcher_LexicalSimilarity(b *testing.B) {
	matcher := NewHistoryMatcher(nil)

	input1 := "搜索关于人工智能的笔记"
	input2 := "查找AI相关的备忘录"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		matcher.calculateLexicalSimilarity(input1, input2)
	}
}

// BenchmarkHistoryMatcher_ExtractBigrams benchmarks bigram extraction.
func BenchmarkHistoryMatcher_ExtractBigrams(b *testing.B) {
	matcher := NewHistoryMatcher(nil)

	input := "搜索关于人工智能和机器学习的相关笔记内容"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		matcher.extractBigrams(input)
	}
}
