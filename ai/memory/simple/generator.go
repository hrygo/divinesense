// Package simple provides a basic memory generator implementation.
//
// WARNING: This implementation is NOT production-ready for the following reasons:
//   - Fixed importance scoring (0.5) without dynamic evaluation
//   - No forgetting/decay mechanism
//   - No memory consolidation or deduplication
//   - No security protection against memory poisoning
//   - Single-dimension (vector) retrieval only
//
// For production use, consider:
//   - Mem0 (https://mem0.ai) - Hybrid datastore with graph/vector/KV
//   - Letta (https://letta.com) - Hierarchical memory with sleep-time compute
//   - Custom implementation based on your domain requirements
//
// This package is suitable for:
//   - Development and testing environments
//   - Prototyping memory-aware features
//   - Learning about memory engineering concepts
package simple

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/hrygo/divinesense/ai/core/llm"
	"github.com/hrygo/divinesense/ai/memory"
	"github.com/hrygo/divinesense/store"
)

// DefaultEmbeddingModel is the default embedding model for episodic memories.
const DefaultEmbeddingModel = "BAAI/bge-m3"

// Config holds configuration for the memory generator.
type Config struct {
	// Enabled controls whether memory generation is active.
	Enabled bool
	// SummaryMaxTokens is the max tokens for LLM summary generation.
	SummaryMaxTokens int
	// MaxConcurrency limits concurrent memory generation tasks.
	MaxConcurrency int
	// Timeout is the maximum time for memory generation.
	Timeout time.Duration
}

// DefaultConfig returns the default configuration.
func DefaultConfig() *Config {
	return &Config{
		Enabled:          true,
		SummaryMaxTokens: 256,
		MaxConcurrency:   5,
		Timeout:          30 * time.Second,
	}
}

// Generator generates episodic memories from completed interactions.
// See package documentation for limitations and production recommendations.
type Generator struct {
	store    MemoryStore
	llm      LLMService
	embedder EmbeddingService
	config   *Config
	sem      chan struct{} // Concurrency limiter
	wg       sync.WaitGroup
}

// MemoryStore defines the interface for memory persistence.
type MemoryStore interface {
	CreateEpisodicMemory(ctx context.Context, create *store.EpisodicMemory) (*store.EpisodicMemory, error)
	UpsertEpisodicMemoryEmbedding(ctx context.Context, embedding *store.EpisodicMemoryEmbedding) (*store.EpisodicMemoryEmbedding, error)
}

// LLMService defines the interface for LLM-based summary generation.
type LLMService interface {
	Chat(ctx context.Context, messages []llm.Message) (string, *llm.LLMCallStats, error)
}

// EmbeddingService defines the interface for embedding generation.
type EmbeddingService interface {
	Embedding(ctx context.Context, text string) ([]float32, error)
}

// NewGenerator creates a new memory generator.
func NewGenerator(
	store MemoryStore,
	llm LLMService,
	embedder EmbeddingService,
	config *Config,
) *Generator {
	if config == nil {
		config = DefaultConfig()
	}
	if config.MaxConcurrency <= 0 {
		config.MaxConcurrency = 5
	}

	return &Generator{
		store:    store,
		llm:      llm,
		embedder: embedder,
		config:   config,
		sem:      make(chan struct{}, config.MaxConcurrency),
	}
}

// GenerateAsync starts asynchronous memory generation.
func (g *Generator) GenerateAsync(ctx context.Context, req memory.MemoryRequest) {
	if !g.config.Enabled {
		return
	}

	g.wg.Add(1)
	go func() {
		defer g.wg.Done()

		// Acquire semaphore for concurrency control
		select {
		case g.sem <- struct{}{}:
			defer func() { <-g.sem }()
		case <-ctx.Done():
			slog.Debug("Memory generation cancelled (semaphore wait)",
				"block_id", req.BlockID,
			)
			return
		}

		// Create dedicated context with timeout
		genCtx, cancel := context.WithTimeout(context.Background(), g.config.Timeout)
		defer cancel()

		if err := g.generate(genCtx, req); err != nil {
			slog.Error("Failed to generate memory",
				"block_id", req.BlockID,
				"user_id", req.UserID,
				"error", err,
			)
		}
	}()
}

// generate performs the actual memory generation.
func (g *Generator) generate(ctx context.Context, req memory.MemoryRequest) error {
	startTime := time.Now()

	// Step 1: Generate summary
	summary, err := g.generateSummary(ctx, req.UserInput, req.Outcome)
	if err != nil {
		// Fallback to truncated outcome if LLM fails
		summary = truncateText(req.Outcome, 500)
		slog.Warn("LLM summary failed, using truncated outcome",
			"block_id", req.BlockID,
			"error", err,
		)
	}

	// Step 2: Generate embedding for the summary
	embeddingVector, err := g.embedder.Embedding(ctx, summary)
	if err != nil {
		return fmt.Errorf("embedding generation failed: %w", err)
	}

	// Step 3: Create episodic memory record
	// NOTE: Importance is fixed at 0.5 - this is a limitation of the simple implementation.
	// Production systems should use dynamic scoring based on relevance, frequency, and recency.
	memoryRecord := &store.EpisodicMemory{
		UserID:     req.UserID,
		AgentType:  req.AgentType,
		UserInput:  req.UserInput,
		Outcome:    "success", // Completed blocks are successful
		Summary:    summary,
		Timestamp:  time.Now(),
		CreatedTs:  time.Now().Unix(),
		Importance: 0.5, // Fixed importance - see note above
	}

	created, err := g.store.CreateEpisodicMemory(ctx, memoryRecord)
	if err != nil {
		return fmt.Errorf("memory creation failed: %w", err)
	}

	// Step 4: Store embedding vector
	embeddingRecord := &store.EpisodicMemoryEmbedding{
		EpisodicMemoryID: int32(created.ID),
		Model:            DefaultEmbeddingModel,
		Embedding:        embeddingVector,
		CreatedTs:        time.Now().Unix(),
		UpdatedTs:        time.Now().Unix(),
	}
	if _, err := g.store.UpsertEpisodicMemoryEmbedding(ctx, embeddingRecord); err != nil {
		return fmt.Errorf("embedding storage failed: %w", err)
	}

	slog.Info("Memory generated successfully",
		"block_id", req.BlockID,
		"memory_id", created.ID,
		"user_id", req.UserID,
		"agent_type", req.AgentType,
		"summary_length", len(summary),
		"embedding_dim", len(embeddingVector),
		"duration_ms", time.Since(startTime).Milliseconds(),
	)

	return nil
}

// generateSummary creates a concise summary of the interaction.
func (g *Generator) generateSummary(ctx context.Context, userInput, outcome string) (string, error) {
	// Truncate inputs to avoid excessive token usage
	truncatedInput := truncateText(userInput, 1000)
	truncatedOutcome := truncateText(outcome, 2000)

	systemPrompt := `You are a memory summarizer. Create a concise summary (max 100 words) of this AI interaction that captures:
1. The user's intent or question
2. The key information or action from the response
Focus on information that would be useful for future reference.`

	userPrompt := fmt.Sprintf(`User: %s

Assistant: %s

Summary:`, truncatedInput, truncatedOutcome)

	messages := []llm.Message{
		llm.SystemPrompt(systemPrompt),
		llm.UserMessage(userPrompt),
	}

	response, _, err := g.llm.Chat(ctx, messages)
	if err != nil {
		return "", err
	}

	return truncateText(response, 500), nil
}

// GenerateSync generates memory synchronously (for testing).
func (g *Generator) GenerateSync(ctx context.Context, req memory.MemoryRequest) error {
	return g.generate(ctx, req)
}

// Shutdown waits for all pending memory generation tasks to complete.
func (g *Generator) Shutdown(ctx context.Context) error {
	done := make(chan struct{})
	go func() {
		g.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// truncateText truncates text to maxLen characters.
func truncateText(text string, maxLen int) string {
	if len(text) <= maxLen {
		return text
	}
	return text[:maxLen] + "..."
}

// Ensure Generator implements memory.Generator.
var _ memory.Generator = (*Generator)(nil)
