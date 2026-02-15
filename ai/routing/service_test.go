package routing

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRuleMatcher_ScheduleIntent(t *testing.T) {
	matcher := newTestMatcher()

	// Note: New architecture uses generic actions. Time pattern without explicit action
	// keyword is now treated as query (schedule_query), not create.
	tests := []struct {
		name           string
		input          string
		expectedIntent Intent
		shouldMatch    bool
		minConfidence  float32
	}{
		{
			name:           "Schedule create with time",
			input:          "明天下午3点开会",
			expectedIntent: IntentScheduleQuery, // Changed: time pattern -> query
			shouldMatch:    true,
			minConfidence:  0.8,
		},
		{
			name:           "Schedule create reminder",
			input:          "设置提醒明天早上9点",
			expectedIntent: IntentScheduleQuery, // Changed: no explicit create keyword
			shouldMatch:    true,
			minConfidence:  0.8,
		},
		{
			name:           "Schedule query explicit",
			input:          "查看今天有什么日程",
			expectedIntent: IntentScheduleQuery,
			shouldMatch:    true,
			minConfidence:  0.6,
		},
		{
			name:           "Schedule update",
			input:          "修改明天的日程",
			expectedIntent: IntentScheduleUpdate,
			shouldMatch:    true,
			minConfidence:  0.6,
		},
		{
			name:           "Batch schedule",
			input:          "批量创建本周的会议日程",
			expectedIntent: IntentBatchSchedule,
			shouldMatch:    true,
			minConfidence:  0.6,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			intent, confidence, matched := matcher.MatchLegacy(tt.input)
			assert.Equal(t, tt.shouldMatch, matched, "match status")
			if matched {
				assert.Equal(t, tt.expectedIntent, intent, "intent")
				assert.GreaterOrEqual(t, confidence, tt.minConfidence, "confidence")
			}
		})
	}
}

func TestRuleMatcher_MemoIntent(t *testing.T) {
	matcher := newTestMatcher()

	tests := []struct {
		name           string
		input          string
		expectedIntent Intent
		shouldMatch    bool
	}{
		{
			name:           "Memo search",
			input:          "搜索关于 Go 的笔记",
			expectedIntent: IntentMemoSearch,
			shouldMatch:    true,
		},
		{
			name:           "Memo find",
			input:          "查找之前写过的关于架构的记录",
			expectedIntent: IntentMemoSearch,
			shouldMatch:    true,
		},
		{
			name:           "Memo create explicit",
			input:          "记录一下这个想法到笔记",
			expectedIntent: IntentMemoCreate,
			shouldMatch:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			intent, _, matched := matcher.MatchLegacy(tt.input)
			assert.Equal(t, tt.shouldMatch, matched, "match status")
			if matched {
				assert.Equal(t, tt.expectedIntent, intent, "intent")
			}
		})
	}
}

func TestRuleMatcher_NoMatch(t *testing.T) {
	matcher := newTestMatcher()

	tests := []string{
		"hi", // Too short
		"你好", // Simple greeting
		"ok", // Simple response
	}

	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			_, _, matched := matcher.MatchLegacy(input)
			assert.False(t, matched, "should not match: %s", input)
		})
	}
}

func TestService_ClassifyIntent_Layer1Only(t *testing.T) {
	// Create service with no memory or LLM
	svc := newTestService(Config{})
	ctx := context.Background()

	// Note: New architecture treats time pattern without explicit action as query
	tests := []struct {
		name           string
		input          string
		expectedIntent Intent
	}{
		{
			name:           "Clear schedule create",
			input:          "明天下午3点开会",
			expectedIntent: IntentScheduleQuery, // Changed: time pattern -> query
		},
		{
			name:           "Clear memo search",
			input:          "搜索关于 Go 的笔记",
			expectedIntent: IntentMemoSearch,
		},
		{
			name:           "Simple greeting - unknown",
			input:          "你好",
			expectedIntent: IntentUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			intent, _, _, err := svc.ClassifyIntent(ctx, tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedIntent, intent)
		})
	}
}

func TestService_SelectModel(t *testing.T) {
	svc := NewService(Config{})
	ctx := context.Background()

	tests := []struct {
		task             TaskType
		expectedProvider string
		expectedModel    string
	}{
		{
			task:             TaskIntentClassification,
			expectedProvider: "local",
			expectedModel:    "qwen2.5-0.5b",
		},
		{
			task:             TaskEntityExtraction,
			expectedProvider: "local",
			expectedModel:    "qwen2.5-1.5b",
		},
		{
			task:             TaskComplexReasoning,
			expectedProvider: "cloud",
			expectedModel:    "deepseek-chat",
		},
	}

	for _, tt := range tests {
		t.Run(string(tt.task), func(t *testing.T) {
			config, err := svc.SelectModel(ctx, tt.task)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedProvider, config.Provider)
			assert.Equal(t, tt.expectedModel, config.Model)
		})
	}
}

func TestHistoryMatcher_Similarity(t *testing.T) {
	matcher := NewHistoryMatcher(nil)

	tests := []struct {
		name       string
		a          string
		b          string
		minSimilar float32
		maxSimilar float32
	}{
		{
			name:       "Identical",
			a:          "明天下午开会",
			b:          "明天下午开会",
			minSimilar: 0.99,
			maxSimilar: 1.0,
		},
		{
			name:       "Similar Chinese text",
			a:          "明天下午开会讨论",
			b:          "明天上午开会讨论",
			minSimilar: 0.5,
			maxSimilar: 0.9,
		},
		{
			name:       "Different",
			a:          "搜索笔记",
			b:          "明天开会",
			minSimilar: 0.0,
			maxSimilar: 0.2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			similarity := matcher.calculateLexicalSimilarity(tt.a, tt.b)
			assert.GreaterOrEqual(t, similarity, tt.minSimilar)
			assert.LessOrEqual(t, similarity, tt.maxSimilar)
		})
	}
}

func TestRuleMatcher_TimePatterns(t *testing.T) {
	matcher := newTestMatcher()

	tests := []struct {
		input   string
		hasTime bool
	}{
		{"下午3点开会", true},
		{"10:30分钟后", true},
		{"1月15日的安排", true},
		{"明天下午", true},
		{"帮我看看", false},
		{"搜索笔记", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			hasTime := matcher.hasTimePattern(tt.input)
			assert.Equal(t, tt.hasTime, hasTime)
		})
	}
}

// Benchmark tests.
func BenchmarkRuleMatcher_Match(b *testing.B) {
	matcher := newTestMatcher()
	input := "明天下午3点开会"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = matcher.Match(input)
	}
}

func BenchmarkHistoryMatcher_Similarity(b *testing.B) {
	matcher := NewHistoryMatcher(nil)
	a := "明天下午开会讨论项目进展"
	c := "明天上午开会讨论工作安排"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		matcher.calculateLexicalSimilarity(a, c)
	}
}
