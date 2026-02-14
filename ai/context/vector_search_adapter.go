package context

import (
	"context"

	"github.com/hrygo/divinesense/store"
)

// VectorSearchStoreAdapter adapts store.Driver to VectorSearchStore interface.
// This enables EpisodicProviderImpl to use the store's vector search capabilities.
type VectorSearchStoreAdapter struct {
	store EpisodicVectorSearcher
}

// EpisodicVectorSearcher defines the interface for episodic vector search.
// This follows Interface Segregation Principle (ISP).
type EpisodicVectorSearcher interface {
	EpisodicVectorSearch(ctx context.Context, opts *store.EpisodicVectorSearchOptions) ([]*store.EpisodicMemoryWithScore, error)
}

// NewVectorSearchStoreAdapter creates a new adapter.
func NewVectorSearchStoreAdapter(s EpisodicVectorSearcher) *VectorSearchStoreAdapter {
	return &VectorSearchStoreAdapter{store: s}
}

// VectorSearchEpisodic implements VectorSearchStore interface.
// It adapts context.VectorSearchEpisodicOptions to store.EpisodicVectorSearchOptions
// and filters results by minimum similarity score.
func (a *VectorSearchStoreAdapter) VectorSearchEpisodic(
	ctx context.Context,
	opts *VectorSearchEpisodicOptions,
) ([]*store.EpisodicMemory, error) {
	// Convert options
	storeOpts := &store.EpisodicVectorSearchOptions{
		Vector: opts.Vector,
		Limit:  opts.Limit,
		UserID: opts.UserID,
	}
	if opts.AgentType != "" {
		storeOpts.AgentType = &opts.AgentType
	}

	// Call store method
	results, err := a.store.EpisodicVectorSearch(ctx, storeOpts)
	if err != nil {
		return nil, err
	}

	// Filter by MinScore and extract EpisodicMemory
	memories := make([]*store.EpisodicMemory, 0, len(results))
	for _, r := range results {
		if r.Score < opts.MinScore {
			continue
		}
		memories = append(memories, r.EpisodicMemory)
	}

	return memories, nil
}

// Ensure VectorSearchStoreAdapter implements VectorSearchStore interface.
var _ VectorSearchStore = (*VectorSearchStoreAdapter)(nil)
