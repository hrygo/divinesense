package context

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/hrygo/divinesense/ai/core/embedding"
	"github.com/hrygo/divinesense/store"
)

// EpisodicConfig configures episodic memory retrieval behavior.
// This is loaded from agent YAML configuration.
type EpisodicConfig struct {
	// Enabled controls whether episodic memory is enabled for this agent
	Enabled bool `yaml:"enabled"`
	// MaxEpisodes is the maximum number of episodes to retrieve
	MaxEpisodes int `yaml:"max_episodes"`
	// MinSimilarity is the minimum similarity threshold for retrieval
	MinSimilarity float32 `yaml:"min_similarity"`
	// EmbeddingModel is the embedding model to use
	EmbeddingModel string `yaml:"embedding_model"`
}

// DefaultEpisodicConfig returns the default episodic configuration.
func DefaultEpisodicConfig() *EpisodicConfig {
	return &EpisodicConfig{
		Enabled:        false, // Disabled by default
		MaxEpisodes:    3,
		MinSimilarity:  0.7,
		EmbeddingModel: "BAAI/bge-m3",
	}
}

// EpisodicProviderImpl implements EpisodicProvider using vector search.
// This enables long-term memory for agents as described in context-engineering.md.
type EpisodicProviderImpl struct {
	store     VectorSearchStore
	embedder  EmbeddingService
	config    *EpisodicConfig
	agentType string
}

// VectorSearchStore defines the interface for vector search operations.
// This follows Interface Segregation Principle (ISP).
type VectorSearchStore interface {
	// VectorSearchEpisodic searches for episodic memories using vector similarity
	VectorSearchEpisodic(ctx context.Context, opts *VectorSearchEpisodicOptions) ([]*store.EpisodicMemory, error)
}

// VectorSearchEpisodicOptions specifies options for episodic vector search.
type VectorSearchEpisodicOptions struct {
	UserID    int32
	AgentType string
	Vector    []float32
	Limit     int
	MinScore  float32
}

// EmbeddingService defines the interface for text embedding generation.
// This interface abstracts the embedding provider, allowing different
// implementations (e.g., SiliconFlow, OpenAI) to be used interchangeably.
// Following Interface Segregation Principle (ISP), this interface only
// exposes the essential embedding operation needed for episodic retrieval.
type EmbeddingService interface {
	// Embed generates a vector embedding for the given text.
	// Returns a float32 slice representing the semantic vector.
	Embed(ctx context.Context, text string) ([]float32, error)
}

// NewEpisodicProvider creates a new episodic memory provider.
func NewEpisodicProvider(
	store VectorSearchStore,
	embedder EmbeddingService,
	config *EpisodicConfig,
	agentType string,
) *EpisodicProviderImpl {
	if config == nil {
		config = DefaultEpisodicConfig()
	}
	return &EpisodicProviderImpl{
		store:     store,
		embedder:  embedder,
		config:    config,
		agentType: agentType,
	}
}

// SearchEpisodes searches for relevant episodic memories.
// Implements context.EpisodicProvider interface.
func (p *EpisodicProviderImpl) SearchEpisodes(
	ctx context.Context,
	userID int32,
	query string,
	limit int,
) ([]*EpisodicMemory, error) {
	// Check if enabled
	if !p.config.Enabled {
		return nil, nil
	}

	// Apply limit from config if not specified
	if limit <= 0 {
		limit = p.config.MaxEpisodes
	}

	// Generate embedding for query
	queryEmb, err := p.embedder.Embed(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to generate query embedding: %w", err)
	}

	// Perform vector search
	opts := &VectorSearchEpisodicOptions{
		UserID:    userID,
		AgentType: p.agentType,
		Vector:    queryEmb,
		Limit:     limit,
		MinScore:  p.config.MinSimilarity,
	}

	results, err := p.store.VectorSearchEpisodic(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("vector search failed: %w", err)
	}

	// Convert store.EpisodicMemory to context.EpisodicMemory
	episodes := make([]*EpisodicMemory, 0, len(results))
	for _, r := range results {
		episodes = append(episodes, &EpisodicMemory{
			ID:        r.ID,
			Timestamp: r.Timestamp,
			Summary:   r.Summary,
			AgentType: r.AgentType,
			Outcome:   r.Outcome,
		})
	}

	slog.Debug("EpisodicProviderImpl.SearchEpisodes",
		"user_id", userID,
		"agent_type", p.agentType,
		"query_length", len(query),
		"results", len(episodes),
		"enabled", p.config.Enabled)

	return episodes, nil
}

// IsEnabled returns whether episodic memory is enabled for this provider.
func (p *EpisodicProviderImpl) IsEnabled() bool {
	return p.config.Enabled
}

// MemoBasedEpisodicProvider adapts the existing memo vector search for episodic memory.
// This bridges the existing memo_embedding infrastructure without requiring a new table.
type MemoBasedEpisodicProvider struct {
	embedder  *embedding.Provider
	memoStore MemoVectorSearchStore
	config    *EpisodicConfig
	userID    int32
}

// MemoVectorSearchStore defines the interface for memo vector search.
// This interface abstracts the memo storage's vector search capability,
// allowing MemoBasedEpisodicProvider to search through historical memos
// for relevant context. Following ISP, it only exposes the vector search
// operation needed for episodic memory retrieval.
type MemoVectorSearchStore interface {
	// VectorSearch performs a similarity search against memo embeddings.
	// Returns memos ordered by similarity score (highest first).
	VectorSearch(ctx context.Context, opts *MemoVectorSearchOptions) ([]*MemoWithScore, error)
}

// MemoVectorSearchOptions specifies options for memo vector search.
type MemoVectorSearchOptions struct {
	Vector []float32
	UserID int32
	Limit  int
}

// MemoWithScore represents a memo with its similarity score.
type MemoWithScore struct {
	ID      int64
	Content string
	Score   float32
}

// NewMemoBasedEpisodicProvider creates a new memo-based episodic provider.
func NewMemoBasedEpisodicProvider(
	embedder *embedding.Provider,
	memoStore MemoVectorSearchStore,
	config *EpisodicConfig,
	userID int32,
) *MemoBasedEpisodicProvider {
	if config == nil {
		config = DefaultEpisodicConfig()
	}
	return &MemoBasedEpisodicProvider{
		embedder:  embedder,
		memoStore: memoStore,
		config:    config,
		userID:    userID,
	}
}

// SearchEpisodes searches for relevant episodic memories using memo vector search.
func (p *MemoBasedEpisodicProvider) SearchEpisodes(
	ctx context.Context,
	userID int32,
	query string,
	limit int,
) ([]*EpisodicMemory, error) {
	if !p.config.Enabled {
		return nil, nil
	}

	if limit <= 0 {
		limit = p.config.MaxEpisodes
	}

	// Generate embedding
	queryEmb, err := p.embedder.Embedding(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to generate embedding: %w", err)
	}

	// Search memos
	results, err := p.memoStore.VectorSearch(ctx, &MemoVectorSearchOptions{
		Vector: queryEmb,
		UserID: userID,
		Limit:  limit,
	})
	if err != nil {
		return nil, fmt.Errorf("memo vector search failed: %w", err)
	}

	// Filter by similarity threshold
	episodes := make([]*EpisodicMemory, 0, len(results))
	for _, r := range results {
		if r.Score < p.config.MinSimilarity {
			continue
		}
		episodes = append(episodes, &EpisodicMemory{
			ID:        r.ID,
			Timestamp: time.Now(), // Would need to extract from memo
			Summary:   truncateSummary(r.Content, 500),
			AgentType: "memo",
			Outcome:   fmt.Sprintf("similarity: %.2f", r.Score),
		})
	}

	return episodes, nil
}

// truncateSummary truncates a summary to maxLen characters.
func truncateSummary(content string, maxLen int) string {
	if len(content) <= maxLen {
		return content
	}
	return content[:maxLen] + "..."
}

// Ensure implementations satisfy interfaces.
var _ EpisodicProvider = (*EpisodicProviderImpl)(nil)
var _ EpisodicProvider = (*MemoBasedEpisodicProvider)(nil)

// EmbeddingProviderAdapter adapts *embedding.Provider to EmbeddingService interface.
// This allows the core embedding.Provider to be used with EpisodicProviderImpl.
type EmbeddingProviderAdapter struct {
	provider *embedding.Provider
}

// NewEmbeddingProviderAdapter creates a new adapter.
func NewEmbeddingProviderAdapter(p *embedding.Provider) *EmbeddingProviderAdapter {
	return &EmbeddingProviderAdapter{provider: p}
}

// Embed implements EmbeddingService interface.
func (a *EmbeddingProviderAdapter) Embed(ctx context.Context, text string) ([]float32, error) {
	return a.provider.Embedding(ctx, text)
}

// Ensure EmbeddingProviderAdapter implements EmbeddingService interface.
var _ EmbeddingService = (*EmbeddingProviderAdapter)(nil)
