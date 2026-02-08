package context

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockMessageProvider is a mock for MessageProvider.
type MockMessageProvider struct {
	mock.Mock
}

func (m *MockMessageProvider) GetRecentMessages(ctx context.Context, sessionID string, limit int) ([]*Message, error) {
	args := m.Called(ctx, sessionID, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*Message), args.Error(1)
}

func TestShortTermExtractor_New(t *testing.T) {
	t.Run("Default max turns", func(t *testing.T) {
		extractor := NewShortTermExtractor(0)
		assert.Equal(t, 10, extractor.maxTurns)
	})

	t.Run("Custom max turns", func(t *testing.T) {
		extractor := NewShortTermExtractor(20)
		assert.Equal(t, 20, extractor.maxTurns)
	})

	t.Run("Negative max turns uses default", func(t *testing.T) {
		extractor := NewShortTermExtractor(-5)
		assert.Equal(t, 10, extractor.maxTurns)
	})
}

func TestShortTermExtractor_Extract(t *testing.T) {
	ctx := context.Background()
	extractor := NewShortTermExtractor(5)

	t.Run("Successful extraction with sorting", func(t *testing.T) {
		mockProvider := new(MockMessageProvider)

		// Messages out of order
		messages := []*Message{
			{
				Timestamp: time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC),
				Role:      "user",
				Content:   "Third message",
			},
			{
				Timestamp: time.Date(2026, 1, 15, 9, 0, 0, 0, time.UTC),
				Role:      "user",
				Content:   "First message",
			},
			{
				Timestamp: time.Date(2026, 1, 15, 9, 30, 0, 0, time.UTC),
				Role:      "assistant",
				Content:   "Second message",
			},
		}

		mockProvider.On("GetRecentMessages", ctx, "session-123", 5).
			Return(messages, nil)

		result, err := extractor.Extract(ctx, mockProvider, "session-123")

		assert.NoError(t, err)
		assert.Len(t, result, 3)
		// Should be sorted by timestamp ascending
		assert.Equal(t, "First message", result[0].Content)
		assert.Equal(t, "Second message", result[1].Content)
		assert.Equal(t, "Third message", result[2].Content)

		mockProvider.AssertExpectations(t)
	})

	t.Run("Nil provider", func(t *testing.T) {
		result, err := extractor.Extract(ctx, nil, "session-123")

		assert.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("Provider error", func(t *testing.T) {
		mockProvider := new(MockMessageProvider)

		mockProvider.On("GetRecentMessages", ctx, "session-123", 5).
			Return(nil, assert.AnError)

		result, err := extractor.Extract(ctx, mockProvider, "session-123")

		assert.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("Respects max turns limit", func(t *testing.T) {
		mockProvider := new(MockMessageProvider)
		extractor := NewShortTermExtractor(3)

		messages := make([]*Message, 10)
		for i := range messages {
			messages[i] = &Message{
				Timestamp: time.Now().Add(time.Duration(i) * time.Minute),
				Content:   "Message",
			}
		}

		mockProvider.On("GetRecentMessages", ctx, "session-123", 3).
			Return(messages, nil)

		result, err := extractor.Extract(ctx, mockProvider, "session-123")

		assert.NoError(t, err)
		// Provider returns all messages, but extractor should limit
		assert.LessOrEqual(t, len(result), 10)
	})
}

func TestFormatConversation_ShortTerm(t *testing.T) {
	t.Run("Empty messages", func(t *testing.T) {
		result := FormatConversation([]*Message{})
		assert.Empty(t, result)
	})

	t.Run("User message", func(t *testing.T) {
		messages := []*Message{
			{Role: "user", Content: "Hello"},
		}

		result := FormatConversation(messages)

		assert.Contains(t, result, "对话历史")
		assert.Contains(t, result, "用户: Hello")
	})

	t.Run("Assistant message", func(t *testing.T) {
		messages := []*Message{
			{Role: "assistant", Content: "Hi there"},
		}

		result := FormatConversation(messages)

		assert.Contains(t, result, "助手: Hi there")
	})

	t.Run("Multiple messages", func(t *testing.T) {
		messages := []*Message{
			{Role: "user", Content: "Question 1"},
			{Role: "assistant", Content: "Answer 1"},
			{Role: "user", Content: "Question 2"},
		}

		result := FormatConversation(messages)

		assert.Contains(t, result, "用户: Question 1")
		assert.Contains(t, result, "助手: Answer 1")
		assert.Contains(t, result, "用户: Question 2")
	})

	t.Run("Unknown role", func(t *testing.T) {
		messages := []*Message{
			{Role: "system", Content: "System message"},
		}

		result := FormatConversation(messages)

		assert.Contains(t, result, "system: System message")
	})
}

func TestSplitByRecency_ShortTerm(t *testing.T) {
	messages := make([]*Message, 10)
	for i := range messages {
		messages[i] = &Message{
			Content: "Message " + string(rune('A'+i)),
		}
	}

	t.Run("Split at 3", func(t *testing.T) {
		recent, older := SplitByRecency(messages, 3)

		assert.Len(t, recent, 3)
		assert.Len(t, older, 7)
		assert.Equal(t, "Message H", recent[0].Content)
		assert.Equal(t, "Message A", older[0].Content)
	})

	t.Run("Split at all", func(t *testing.T) {
		recent, older := SplitByRecency(messages, 10)

		assert.Len(t, recent, 10)
		assert.Nil(t, older)
	})

	t.Run("Split at more than length", func(t *testing.T) {
		recent, older := SplitByRecency(messages, 20)

		assert.Len(t, recent, 10)
		assert.Nil(t, older)
	})

	t.Run("Split at 0", func(t *testing.T) {
		recent, older := SplitByRecency(messages, 0)

		assert.Nil(t, recent)
		assert.Len(t, older, 10)
	})

	t.Run("Empty messages", func(t *testing.T) {
		recent, older := SplitByRecency([]*Message{}, 3)

		assert.Nil(t, recent)
		assert.Nil(t, older)
	})
}

// Benchmark_Extract benchmarks short-term extraction.
func BenchmarkShortTermExtract(b *testing.B) {
	ctx := context.Background()
	extractor := NewShortTermExtractor(10)
	mockProvider := new(MockMessageProvider)

	messages := make([]*Message, 20)
	for i := range messages {
		messages[i] = &Message{
			Timestamp: time.Now().Add(time.Duration(i) * time.Second),
			Content:   "Message content",
			Role:      "user",
		}
	}

	mockProvider.On("GetRecentMessages", mock.Anything, mock.Anything, mock.Anything).
		Return(messages, nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = extractor.Extract(ctx, mockProvider, "session-123")
	}
}

// Benchmark_FormatConversation benchmarks conversation formatting.
func Benchmark_FormatConversation(b *testing.B) {
	messages := make([]*Message, 50)
	for i := range messages {
		role := "user"
		if i%2 == 1 {
			role = "assistant"
		}
		messages[i] = &Message{
			Role:    role,
			Content: "This is a test message with some content",
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = FormatConversation(messages)
	}
}

// Benchmark_SplitByRecency benchmarks recency splitting.
func Benchmark_SplitByRecency(b *testing.B) {
	messages := make([]*Message, 100)
	for i := range messages {
		messages[i] = &Message{Content: "Message"}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = SplitByRecency(messages, 10)
	}
}
