// Package memory provides the memory extension point for AI agents.
//
// Memory engineering is a specialized domain involving:
//   - Importance scoring (relevance + frequency + recency)
//   - Forgetting mechanisms (biologically-inspired decay)
//   - Memory consolidation (merging, deduplication, compression)
//   - Security (memory poisoning detection)
//   - Multi-dimensional retrieval (semantic + temporal + causal)
//
// This package defines the extension interface. Implementations can range from
// simple (for development/testing) to sophisticated (Mem0, Letta integration).
//
// Reference: docs/architecture/context-engineering.md Phase 3
package memory

import "context"

// Generator defines the memory generation extension point.
// Implementations can be:
//   - NoOpGenerator: No-op implementation (default, production-safe)
//   - SimpleGenerator: Basic implementation for dev/test (not production-ready)
//   - ExternalGenerator: Integration with Mem0, Letta, or custom services
type Generator interface {
	// GenerateAsync starts asynchronous memory generation.
	// This method returns immediately and processes the memory in a goroutine.
	// Implementations SHOULD be non-blocking and handle their own error logging.
	GenerateAsync(ctx context.Context, req MemoryRequest)

	// GenerateSync generates memory synchronously (for testing).
	// This method blocks until memory generation completes or fails.
	GenerateSync(ctx context.Context, req MemoryRequest) error

	// Shutdown waits for all pending memory generation tasks to complete.
	// This should be called during graceful service shutdown.
	Shutdown(ctx context.Context) error
}

// MemoryRequest contains the data needed to generate a memory.
type MemoryRequest struct {
	BlockID   int64
	UserID    int32
	AgentType string
	UserInput string
	Outcome   string // Assistant's response
	Metadata  map[string]any
}

// NoOpGenerator is a no-op implementation of Generator.
// It is the default implementation, safe for production use.
// Use this when memory generation is disabled or not yet configured.
type NoOpGenerator struct{}

// NewNoOpGenerator creates a new no-op memory generator.
func NewNoOpGenerator() *NoOpGenerator {
	return &NoOpGenerator{}
}

func (n *NoOpGenerator) GenerateAsync(_ context.Context, _ MemoryRequest) {}
func (n *NoOpGenerator) GenerateSync(_ context.Context, _ MemoryRequest) error {
	return nil
}
func (n *NoOpGenerator) Shutdown(_ context.Context) error {
	return nil
}

// Ensure NoOpGenerator implements Generator.
var _ Generator = (*NoOpGenerator)(nil)
