package simple

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/hrygo/divinesense/ai/core/llm"
	"github.com/hrygo/divinesense/ai/memory"
	"github.com/hrygo/divinesense/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockMemoryStore implements MemoryStore for testing.
type MockMemoryStore struct {
	memories   []*store.EpisodicMemory
	embeddings []*store.EpisodicMemoryEmbedding
	err        error
}

func (m *MockMemoryStore) CreateEpisodicMemory(ctx context.Context, create *store.EpisodicMemory) (*store.EpisodicMemory, error) {
	if m.err != nil {
		return nil, m.err
	}
	create.ID = int64(len(m.memories) + 1)
	m.memories = append(m.memories, create)
	return create, nil
}

func (m *MockMemoryStore) UpsertEpisodicMemoryEmbedding(ctx context.Context, embedding *store.EpisodicMemoryEmbedding) (*store.EpisodicMemoryEmbedding, error) {
	if m.err != nil {
		return nil, m.err
	}
	embedding.ID = int32(len(m.embeddings) + 1)
	m.embeddings = append(m.embeddings, embedding)
	return embedding, nil
}

// MockLLMService implements LLMService for testing.
type MockLLMService struct {
	response string
	err      error
}

func (m *MockLLMService) Chat(ctx context.Context, messages []llm.Message) (string, *llm.LLMCallStats, error) {
	if m.err != nil {
		return "", nil, m.err
	}
	return m.response, &llm.LLMCallStats{TotalTokens: 50}, nil
}

// MockEmbeddingService implements EmbeddingService for testing.
type MockEmbeddingService struct {
	embedding []float32
	err       error
}

func (m *MockEmbeddingService) Embedding(ctx context.Context, text string) ([]float32, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.embedding, nil
}

func TestNewGenerator(t *testing.T) {
	mockStore := &MockMemoryStore{}
	mockLLM := &MockLLMService{}
	mockEmbedder := &MockEmbeddingService{embedding: make([]float32, 1024)}

	t.Run("with default config", func(t *testing.T) {
		g := NewGenerator(mockStore, mockLLM, mockEmbedder, nil)
		assert.NotNil(t, g)
		assert.True(t, g.config.Enabled)
		assert.Equal(t, 5, g.config.MaxConcurrency)
	})

	t.Run("with custom config", func(t *testing.T) {
		config := &Config{
			Enabled:          false,
			SummaryMaxTokens: 128,
			MaxConcurrency:   3,
		}
		g := NewGenerator(mockStore, mockLLM, mockEmbedder, config)
		assert.NotNil(t, g)
		assert.False(t, g.config.Enabled)
		assert.Equal(t, 128, g.config.SummaryMaxTokens)
		assert.Equal(t, 3, g.config.MaxConcurrency)
	})
}

func TestGenerator_GenerateSync(t *testing.T) {
	mockStore := &MockMemoryStore{}
	mockLLM := &MockLLMService{response: "Test summary of the interaction"}
	mockEmbedder := &MockEmbeddingService{embedding: make([]float32, 1024)}

	g := NewGenerator(mockStore, mockLLM, mockEmbedder, &Config{
		Enabled:          true,
		SummaryMaxTokens: 256,
		MaxConcurrency:   5,
		Timeout:          10 * time.Second,
	})

	req := memory.MemoryRequest{
		BlockID:   1,
		UserID:    100,
		AgentType: "memo",
		UserInput: "Find my notes about project X",
		Outcome:   "I found 3 notes related to project X in your workspace",
	}

	err := g.GenerateSync(context.Background(), req)
	require.NoError(t, err)

	// Verify memory was created
	require.Len(t, mockStore.memories, 1)
	mem := mockStore.memories[0]

	assert.Equal(t, int32(100), mem.UserID)
	assert.Equal(t, "memo", mem.AgentType)
	assert.Equal(t, "success", mem.Outcome)
	assert.Equal(t, "Test summary of the interaction", mem.Summary)
}

func TestGenerator_GenerateSync_LLMFailure(t *testing.T) {
	mockStore := &MockMemoryStore{}
	mockLLM := &MockLLMService{err: errors.New("LLM error")}
	mockEmbedder := &MockEmbeddingService{embedding: make([]float32, 1024)}

	g := NewGenerator(mockStore, mockLLM, mockEmbedder, DefaultConfig())

	req := memory.MemoryRequest{
		BlockID:   1,
		UserID:    100,
		AgentType: "memo",
		UserInput: "Test input",
		Outcome:   "Test outcome that should be truncated if it's long enough to exceed the limit",
	}

	err := g.GenerateSync(context.Background(), req)
	require.NoError(t, err)

	// Should fallback to truncated outcome
	require.Len(t, mockStore.memories, 1)
	assert.Contains(t, mockStore.memories[0].Summary, "Test outcome")
}

func TestGenerator_GenerateSync_EmbeddingFailure(t *testing.T) {
	mockStore := &MockMemoryStore{}
	mockLLM := &MockLLMService{response: "Summary"}
	mockEmbedder := &MockEmbeddingService{err: errors.New("embedding error")}

	g := NewGenerator(mockStore, mockLLM, mockEmbedder, DefaultConfig())

	req := memory.MemoryRequest{
		BlockID:   1,
		UserID:    100,
		AgentType: "memo",
		UserInput: "Test",
		Outcome:   "Response",
	}

	err := g.GenerateSync(context.Background(), req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "embedding generation failed")
}

func TestGenerator_GenerateAsync(t *testing.T) {
	mockStore := &MockMemoryStore{}
	mockLLM := &MockLLMService{response: "Summary"}
	mockEmbedder := &MockEmbeddingService{embedding: make([]float32, 1024)}

	g := NewGenerator(mockStore, mockLLM, mockEmbedder, DefaultConfig())

	req := memory.MemoryRequest{
		BlockID:   1,
		UserID:    100,
		AgentType: "memo",
		UserInput: "Test",
		Outcome:   "Response",
	}

	g.GenerateAsync(context.Background(), req)

	// Wait for async generation to complete
	err := g.Shutdown(context.Background())
	require.NoError(t, err)

	// Verify memory was created
	require.Len(t, mockStore.memories, 1)
}

func TestGenerator_GenerateAsync_Disabled(t *testing.T) {
	mockStore := &MockMemoryStore{}
	mockLLM := &MockLLMService{}
	mockEmbedder := &MockEmbeddingService{}

	g := NewGenerator(mockStore, mockLLM, mockEmbedder, &Config{
		Enabled: false,
	})

	req := memory.MemoryRequest{
		BlockID:   1,
		UserID:    100,
		AgentType: "memo",
		UserInput: "Test",
		Outcome:   "Response",
	}

	g.GenerateAsync(context.Background(), req)

	// Should not generate any memory
	err := g.Shutdown(context.Background())
	require.NoError(t, err)
	assert.Len(t, mockStore.memories, 0)
}

func TestGenerator_Shutdown(t *testing.T) {
	mockStore := &MockMemoryStore{}
	mockLLM := &MockLLMService{response: "Summary"}
	mockEmbedder := &MockEmbeddingService{embedding: make([]float32, 1024)}

	g := NewGenerator(mockStore, mockLLM, mockEmbedder, DefaultConfig())

	// Start multiple async generations
	for i := 0; i < 3; i++ {
		req := memory.MemoryRequest{
			BlockID:   int64(i + 1),
			UserID:    100,
			AgentType: "memo",
			UserInput: "Test",
			Outcome:   "Response",
		}
		g.GenerateAsync(context.Background(), req)
	}

	// Shutdown should wait for all
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := g.Shutdown(ctx)
	require.NoError(t, err)

	// All memories should be created
	assert.Len(t, mockStore.memories, 3)
}

func TestTruncateText(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		maxLen   int
		expected string
	}{
		{
			name:     "short text unchanged",
			text:     "short",
			maxLen:   100,
			expected: "short",
		},
		{
			name:     "long text truncated",
			text:     "this is a very long text that needs to be truncated",
			maxLen:   20,
			expected: "this is a very long ...",
		},
		{
			name:     "exact length unchanged",
			text:     "exact",
			maxLen:   5,
			expected: "exact",
		},
		{
			name:     "empty text",
			text:     "",
			maxLen:   10,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncateText(tt.text, tt.maxLen)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGenerator_ImplementsInterface(t *testing.T) {
	// Ensure Generator implements memory.Generator interface
	var _ memory.Generator = (*Generator)(nil)
}
