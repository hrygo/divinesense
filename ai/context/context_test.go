package context

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTokenBudgetAllocation(t *testing.T) {
	t.Run("With retrieval", func(t *testing.T) {
		budget := AllocateBudget(4096, true)

		total := budget.SystemPrompt + budget.ShortTermMemory +
			budget.LongTermMemory + budget.Retrieval + budget.UserPrefs

		assert.LessOrEqual(t, total, 4096)
		assert.Equal(t, 500, budget.SystemPrompt)
		assert.Greater(t, budget.Retrieval, 0)
	})

	t.Run("Without retrieval", func(t *testing.T) {
		budget := AllocateBudget(4096, false)

		assert.Equal(t, 0, budget.Retrieval)
		assert.Greater(t, budget.ShortTermMemory, 0)
		assert.Greater(t, budget.LongTermMemory, 0)
	})

	t.Run("Default total", func(t *testing.T) {
		budget := AllocateBudget(0, true)
		assert.Equal(t, DefaultMaxTokens, budget.Total)
	})
}

func TestPriorityRanking(t *testing.T) {
	ranker := NewPriorityRanker()

	t.Run("Sort by priority", func(t *testing.T) {
		segments := []*ContextSegment{
			{Content: "low", Priority: PriorityOlderTurns, TokenCost: 100},
			{Content: "high", Priority: PrioritySystem, TokenCost: 100},
			{Content: "mid", Priority: PriorityRetrieval, TokenCost: 100},
		}

		result := ranker.RankAndTruncate(segments, 300)

		require.Len(t, result, 3)
		assert.Equal(t, "high", result[0].Content)
		assert.Equal(t, "mid", result[1].Content)
		assert.Equal(t, "low", result[2].Content)
	})

	t.Run("Truncate to budget", func(t *testing.T) {
		segments := []*ContextSegment{
			{Content: "system", Priority: PrioritySystem, TokenCost: 500},
			{Content: "recent", Priority: PriorityRecentTurns, TokenCost: 2000},
			{Content: "retrieval", Priority: PriorityRetrieval, TokenCost: 2000},
		}

		result := ranker.RankAndTruncate(segments, 3000)

		// Should include System(500) + RecentTurns(2000) + truncated Retrieval(500) = 3000
		assert.Equal(t, 3, len(result))
		assert.Equal(t, "system", result[0].Content)
		assert.Equal(t, "recent", result[1].Content)
		assert.Equal(t, 500, result[2].TokenCost) // Retrieval truncated to remaining budget
	})
}

func TestEstimateTokens(t *testing.T) {
	tests := []struct {
		input     string
		minTokens int
		maxTokens int
	}{
		{"hello world", 1, 10},
		{"你好世界", 4, 12},
		{"Hello 世界", 3, 10},
		{"", 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			tokens := EstimateTokens(tt.input)
			assert.GreaterOrEqual(t, tokens, tt.minTokens)
			assert.LessOrEqual(t, tokens, tt.maxTokens)
		})
	}
}

func TestFormatConversation(t *testing.T) {
	messages := []*Message{
		{Role: "user", Content: "Hello"},
		{Role: "assistant", Content: "Hi there"},
		{Role: "user", Content: "How are you?"},
	}

	result := FormatConversation(messages)

	assert.Contains(t, result, "用户: Hello")
	assert.Contains(t, result, "助手: Hi there")
	assert.Contains(t, result, "用户: How are you?")
}

func TestFormatEpisodes(t *testing.T) {
	episodes := []*EpisodicMemory{
		{
			Timestamp: time.Date(2026, 1, 15, 10, 30, 0, 0, time.Local),
			Summary:   "Created meeting notes",
		},
		{
			Timestamp: time.Date(2026, 1, 20, 14, 0, 0, 0, time.Local),
			Summary:   "Searched for Go documentation",
		},
	}

	result := FormatEpisodes(episodes)

	assert.Contains(t, result, "相关历史")
	assert.Contains(t, result, "Created meeting notes")
	assert.Contains(t, result, "Searched for Go documentation")
}

func TestFormatPreferences(t *testing.T) {
	t.Run("With preferences", func(t *testing.T) {
		prefs := &UserPreferences{
			Timezone:        "Asia/Shanghai",
			DefaultDuration: 60,
			PreferredTimes:  []string{"09:00", "14:00"},
		}

		result := FormatPreferences(prefs)

		assert.Contains(t, result, "时区: Asia/Shanghai")
		assert.Contains(t, result, "默认会议时长: 60分钟")
	})

	t.Run("Empty preferences", func(t *testing.T) {
		result := FormatPreferences(&UserPreferences{})
		assert.Empty(t, result)
	})

	t.Run("Nil preferences", func(t *testing.T) {
		result := FormatPreferences(nil)
		assert.Empty(t, result)
	})
}

func TestSplitByRecency(t *testing.T) {
	messages := make([]*Message, 10)
	for i := range messages {
		messages[i] = &Message{Content: string(rune('A' + i))}
	}

	recent, older := SplitByRecency(messages, 3)

	assert.Len(t, recent, 3)
	assert.Len(t, older, 7)
	assert.Equal(t, "H", recent[0].Content) // 8th message
	assert.Equal(t, "A", older[0].Content)  // 1st message
}

func TestServiceBuild(t *testing.T) {
	svc := NewService(DefaultConfig())
	ctx := context.Background()

	t.Run("Basic build without providers", func(t *testing.T) {
		req := &ContextRequest{
			UserID:       1,
			SessionID:    "session-1",
			CurrentQuery: "Hello",
			AgentType:    "memo",
			MaxTokens:    4096,
		}

		result, err := svc.Build(ctx, req)

		require.NoError(t, err)
		assert.NotEmpty(t, result.SystemPrompt)
		assert.Greater(t, result.TotalTokens, 0)
		assert.NotNil(t, result.TokenBreakdown)
	})

	t.Run("Build with retrieval", func(t *testing.T) {
		req := &ContextRequest{
			UserID:       1,
			CurrentQuery: "Search notes",
			AgentType:    "memo",
			RetrievalResults: []*RetrievalItem{
				{ID: "1", Content: "Note about Go"},
				{ID: "2", Content: "Note about Python"},
			},
		}

		result, err := svc.Build(ctx, req)

		require.NoError(t, err)
		assert.NotEmpty(t, result.RetrievalContext)
		assert.Greater(t, result.TokenBreakdown.Retrieval, 0)
	})
}

func TestServiceStats(t *testing.T) {
	svc := NewService(DefaultConfig())
	ctx := context.Background()

	// Build a few times
	for i := 0; i < 3; i++ {
		svc.Build(ctx, &ContextRequest{CurrentQuery: "test"})
	}

	stats := svc.GetStats()

	assert.Equal(t, int64(3), stats.TotalBuilds)
	assert.Greater(t, stats.AverageTokens, float64(0))
}

// MockMessageProvider for testing BuildHistory
type mockMessageProvider struct {
	messages []*Message
	err      error
}

func (m *mockMessageProvider) GetRecentMessages(ctx context.Context, sessionID string, limit int) ([]*Message, error) {
	return m.messages, m.err
}

func TestServiceBuildHistory(t *testing.T) {
	ctx := context.Background()

	t.Run("Returns alternating user/assistant pairs", func(t *testing.T) {
		messages := []*Message{
			{Role: "user", Content: "Hello", Timestamp: time.Now()},
			{Role: "assistant", Content: "Hi there", Timestamp: time.Now().Add(time.Second)},
			{Role: "user", Content: "How are you?", Timestamp: time.Now().Add(2 * time.Second)},
			{Role: "assistant", Content: "I'm doing well", Timestamp: time.Now().Add(3 * time.Second)},
		}
		mockProvider := &mockMessageProvider{messages: messages}
		svc := NewService(DefaultConfig()).WithMessageProvider(mockProvider)

		history, err := svc.BuildHistory(ctx, &ContextRequest{SessionID: "test"})

		require.NoError(t, err)
		assert.Equal(t, []string{"Hello", "Hi there", "How are you?", "I'm doing well"}, history)
	})

	t.Run("Returns nil when no provider", func(t *testing.T) {
		svc := NewService(DefaultConfig())

		history, err := svc.BuildHistory(ctx, &ContextRequest{SessionID: "test"})

		require.NoError(t, err)
		assert.Nil(t, history)
	})

	t.Run("Returns nil when no messages", func(t *testing.T) {
		mockProvider := &mockMessageProvider{messages: []*Message{}}
		svc := NewService(DefaultConfig()).WithMessageProvider(mockProvider)

		history, err := svc.BuildHistory(ctx, &ContextRequest{SessionID: "test"})

		require.NoError(t, err)
		assert.Nil(t, history)
	})

	t.Run("Handles provider error", func(t *testing.T) {
		mockProvider := &mockMessageProvider{err: assert.AnError}
		svc := NewService(DefaultConfig()).WithMessageProvider(mockProvider)

		history, err := svc.BuildHistory(ctx, &ContextRequest{SessionID: "test"})

		require.Error(t, err)
		assert.Nil(t, history)
	})

	t.Run("Ensures even number of messages", func(t *testing.T) {
		// Odd number of messages (trailing user without assistant response)
		messages := []*Message{
			{Role: "user", Content: "Hello", Timestamp: time.Now()},
			{Role: "assistant", Content: "Hi there", Timestamp: time.Now().Add(time.Second)},
			{Role: "user", Content: "How are you?", Timestamp: time.Now().Add(2 * time.Second)},
		}
		mockProvider := &mockMessageProvider{messages: messages}
		svc := NewService(DefaultConfig()).WithMessageProvider(mockProvider)

		history, err := svc.BuildHistory(ctx, &ContextRequest{SessionID: "test"})

		require.NoError(t, err)
		assert.Equal(t, []string{"Hello", "Hi there"}, history) // Trailing user removed
	})
}

// Benchmark tests.
func BenchmarkBuild(b *testing.B) {
	svc := NewService(DefaultConfig())
	ctx := context.Background()
	req := &ContextRequest{
		UserID:       1,
		SessionID:    "bench-session",
		CurrentQuery: "Search for notes about Go programming",
		AgentType:    "memo",
		RetrievalResults: []*RetrievalItem{
			{ID: "1", Content: "Go is a statically typed language"},
			{ID: "2", Content: "Go has goroutines for concurrency"},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		svc.Build(ctx, req)
	}
}

func BenchmarkEstimateTokens(b *testing.B) {
	content := "这是一段混合中英文的测试文本 This is a test with mixed Chinese and English content"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		EstimateTokens(content)
	}
}
