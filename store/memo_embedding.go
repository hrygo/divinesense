package store

import (
	"context"

	"github.com/pkg/errors"
)

// MemoEmbedding represents the vector embedding of a memo.
type MemoEmbedding struct {
	Model     string
	Embedding []float32
	CreatedTs int64
	UpdatedTs int64
	ID        int32
	MemoID    int32
}

// FindMemoEmbedding is the find condition for memo embeddings.
type FindMemoEmbedding struct {
	MemoID *int32
	Model  *string
}

// FindMemosWithoutEmbedding is the find condition for memos without embeddings.
type FindMemosWithoutEmbedding struct {
	Model string // Embedding model to check
	Limit int    // Maximum number of memos to return
}

// MemoWithScore represents a vector search result with similarity score.
type MemoWithScore struct {
	Memo  *Memo
	Score float32 // Similarity score (0-1, higher is more similar)
}

// VectorSearchOptions represents the options for vector search.
type VectorSearchOptions struct {
	Vector          []float32
	Limit           int
	UserID          int32
	CreatedAfter    int64 // Optional: only search memos created after this timestamp
	MaxCandidates   int   // Optional: maximum candidates to fetch (default: limit * 10)
	ExcludeComments bool  // Optional: exclude comments (memos with parent)
}

// Validate validates the VectorSearchOptions.
func (o *VectorSearchOptions) Validate() error {
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

// BM25SearchOptions represents the options for BM25 full-text search.
type BM25SearchOptions struct {
	Query           string
	Limit           int
	UserID          int32
	MinScore        float32
	ExcludeComments bool // Optional: exclude comments (memos with parent)
}

// Validate validates the BM25SearchOptions.
func (o *BM25SearchOptions) Validate() error {
	if o.UserID <= 0 {
		return errors.Errorf("invalid UserID: %d", o.UserID)
	}
	if o.Query == "" {
		return errors.Errorf("query cannot be empty")
	}
	// Limit query length to prevent DoS and performance issues
	// PostgreSQL plainto_tsquery handles special characters safely
	if len(o.Query) > 500 {
		return errors.Errorf("query too long (max 500 characters): %d", len(o.Query))
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
	if o.MinScore < 0 || o.MinScore > 1 {
		return errors.Errorf("MinScore must be between 0 and 1: %f", o.MinScore)
	}
	return nil
}

// BM25Result represents a BM25 search result with relevance score.
type BM25Result struct {
	Memo  *Memo
	Score float32 // BM25 relevance score
}

// UpsertMemoEmbedding inserts or updates a memo embedding.
func (s *Store) UpsertMemoEmbedding(ctx context.Context, embedding *MemoEmbedding) (*MemoEmbedding, error) {
	return s.driver.UpsertMemoEmbedding(ctx, embedding)
}

// GetMemoEmbedding gets the embedding of a specific memo.
func (s *Store) GetMemoEmbedding(ctx context.Context, memoID int32, model string) (*MemoEmbedding, error) {
	list, err := s.driver.ListMemoEmbeddings(ctx, &FindMemoEmbedding{
		MemoID: &memoID,
		Model:  &model,
	})
	if err != nil {
		return nil, err
	}
	if len(list) == 0 {
		return nil, nil
	}
	return list[0], nil
}

// ListMemoEmbeddings lists memo embeddings.
func (s *Store) ListMemoEmbeddings(ctx context.Context, find *FindMemoEmbedding) ([]*MemoEmbedding, error) {
	return s.driver.ListMemoEmbeddings(ctx, find)
}

// DeleteMemoEmbedding deletes a memo embedding.
func (s *Store) DeleteMemoEmbedding(ctx context.Context, memoID int32) error {
	return s.driver.DeleteMemoEmbedding(ctx, memoID)
}

// FindMemosWithoutEmbedding finds memos that don't have embeddings for the specified model.
func (s *Store) FindMemosWithoutEmbedding(ctx context.Context, find *FindMemosWithoutEmbedding) ([]*Memo, error) {
	return s.driver.FindMemosWithoutEmbedding(ctx, find)
}

// VectorSearch performs vector similarity search.
func (s *Store) VectorSearch(ctx context.Context, opts *VectorSearchOptions) ([]*MemoWithScore, error) {
	if err := opts.Validate(); err != nil {
		return nil, err
	}
	return s.driver.VectorSearch(ctx, opts)
}

// BM25Search performs full-text search using BM25 ranking.
func (s *Store) BM25Search(ctx context.Context, opts *BM25SearchOptions) ([]*BM25Result, error) {
	if err := opts.Validate(); err != nil {
		return nil, err
	}
	return s.driver.BM25Search(ctx, opts)
}
