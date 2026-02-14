package store

import (
	"context"

	"github.com/pkg/errors"
)

// EpisodicMemoryEmbedding represents the vector embedding of an episodic memory.
type EpisodicMemoryEmbedding struct {
	ID               int32
	EpisodicMemoryID int32
	Model            string
	Embedding        []float32
	CreatedTs        int64
	UpdatedTs        int64
}

// FindEpisodicMemoryEmbedding is the find condition for episodic memory embeddings.
type FindEpisodicMemoryEmbedding struct {
	EpisodicMemoryID *int32
	Model            *string
}

// FindEpisodicMemoriesWithoutEmbedding is the find condition for episodic memories without embeddings.
type FindEpisodicMemoriesWithoutEmbedding struct {
	Model string // Embedding model to check
	Limit int    // Maximum number of memories to return
}

// EpisodicMemoryWithScore represents a vector search result with similarity score.
type EpisodicMemoryWithScore struct {
	EpisodicMemory *EpisodicMemory
	Score          float32 // Similarity score (0-1, higher is more similar)
}

// EpisodicVectorSearchOptions represents the options for episodic memory vector search.
type EpisodicVectorSearchOptions struct {
	Vector       []float32
	Limit        int
	UserID       int32
	AgentType    *string // Optional: filter by agent type
	CreatedAfter int64   // Optional: only search memories created after this timestamp
}

// Validate validates the EpisodicVectorSearchOptions.
func (o *EpisodicVectorSearchOptions) Validate() error {
	if o.UserID <= 0 {
		return errors.Errorf("invalid UserID: %d", o.UserID)
	}
	if len(o.Vector) == 0 {
		return errors.Errorf("vector cannot be empty")
	}
	if o.Limit < 0 {
		return errors.Errorf("limit cannot be negative: %d", o.Limit)
	}
	if o.Limit == 0 {
		o.Limit = 10 // Default limit
	}
	if o.Limit > 1000 {
		return errors.Errorf("limit too large (max 1000): %d", o.Limit)
	}
	return nil
}

// UpsertEpisodicMemoryEmbedding inserts or updates an episodic memory embedding.
func (s *Store) UpsertEpisodicMemoryEmbedding(ctx context.Context, embedding *EpisodicMemoryEmbedding) (*EpisodicMemoryEmbedding, error) {
	return s.driver.UpsertEpisodicMemoryEmbedding(ctx, embedding)
}

// GetEpisodicMemoryEmbedding gets the embedding of a specific episodic memory.
func (s *Store) GetEpisodicMemoryEmbedding(ctx context.Context, episodicMemoryID int32, model string) (*EpisodicMemoryEmbedding, error) {
	list, err := s.driver.ListEpisodicMemoryEmbeddings(ctx, &FindEpisodicMemoryEmbedding{
		EpisodicMemoryID: &episodicMemoryID,
		Model:            &model,
	})
	if err != nil {
		return nil, err
	}
	if len(list) == 0 {
		return nil, nil
	}
	return list[0], nil
}

// ListEpisodicMemoryEmbeddings lists episodic memory embeddings.
func (s *Store) ListEpisodicMemoryEmbeddings(ctx context.Context, find *FindEpisodicMemoryEmbedding) ([]*EpisodicMemoryEmbedding, error) {
	return s.driver.ListEpisodicMemoryEmbeddings(ctx, find)
}

// DeleteEpisodicMemoryEmbedding deletes an episodic memory embedding.
func (s *Store) DeleteEpisodicMemoryEmbedding(ctx context.Context, episodicMemoryID int32) error {
	return s.driver.DeleteEpisodicMemoryEmbedding(ctx, episodicMemoryID)
}

// FindEpisodicMemoriesWithoutEmbedding finds episodic memories that don't have embeddings for the specified model.
func (s *Store) FindEpisodicMemoriesWithoutEmbedding(ctx context.Context, find *FindEpisodicMemoriesWithoutEmbedding) ([]*EpisodicMemory, error) {
	return s.driver.FindEpisodicMemoriesWithoutEmbedding(ctx, find)
}

// EpisodicVectorSearch performs vector similarity search on episodic memories.
func (s *Store) EpisodicVectorSearch(ctx context.Context, opts *EpisodicVectorSearchOptions) ([]*EpisodicMemoryWithScore, error) {
	if err := opts.Validate(); err != nil {
		return nil, err
	}
	return s.driver.EpisodicVectorSearch(ctx, opts)
}
