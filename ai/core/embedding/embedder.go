package embedding

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/hrygo/divinesense/store"
)

// Embedder handles embedding generation and storage for memos.
type Embedder struct {
	provider *Provider
	store    *store.Store
}

// NewEmbedder creates a new embedder.
func NewEmbedder(provider *Provider, store *store.Store) *Embedder {
	return &Embedder{
		provider: provider,
		store:    store,
	}
}

// EmbedMemo generates and stores embedding for a single memo.
func (e *Embedder) EmbedMemo(ctx context.Context, memo *store.Memo) error {
	if memo == nil {
		return fmt.Errorf("memo is nil")
	}
	if memo.Content == "" {
		return fmt.Errorf("memo content is empty")
	}

	// Chunk document
	chunks := ChunkDocument(memo.Content)

	// Generate embeddings for all chunks
	embeddings := make([][]float32, len(chunks))
	for i, chunk := range chunks {
		emb, err := e.provider.Embedding(ctx, chunk)
		if err != nil {
			return fmt.Errorf("failed to embed chunk %d: %w", i, err)
		}
		embeddings[i] = emb
	}

	// Average pool embeddings (multiple chunks -> single vector)
	avgEmbedding := averageEmbeddings(embeddings)

	// Store in database
	driver := e.store.GetDriver()
	if err := driver.UpdateMemoEmbedding(ctx, memo.ID, avgEmbedding); err != nil {
		return fmt.Errorf("failed to update memo embedding: %w", err)
	}

	slog.Debug("Memo embedded successfully",
		"memo_id", memo.ID,
		"chunks", len(chunks),
		"embedding_dim", len(avgEmbedding))

	return nil
}

// EmbedMemoBatch generates and stores embeddings for multiple memos concurrently.
// The concurrency is limited to avoid overwhelming the API.
func (e *Embedder) EmbedMemoBatch(ctx context.Context, memos []*store.Memo) <-chan error {
	results := make(chan error, len(memos))

	// Limit concurrency to 3 to avoid overwhelming the API
	sem := make(chan struct{}, 3)

	go func() {
		for _, memo := range memos {
			sem <- struct{}{}
			go func(m *store.Memo) {
				defer func() { <-sem }()
				results <- e.EmbedMemo(ctx, m)
			}(memo)
		}
	}()

	// Close results channel when all goroutines complete
	go func() {
		// Wait for all goroutines to finish
		for i := 0; i < len(memos); i++ {
			<-sem
		}
		close(results)
	}()

	return results
}

// averageEmbeddings computes the element-wise average of multiple embeddings.
func averageEmbeddings(embeddings [][]float32) []float32 {
	if len(embeddings) == 0 {
		return nil
	}

	// All embeddings should have the same dimension
	n := len(embeddings[0])
	if n == 0 {
		return nil
	}

	result := make([]float32, n)

	// Sum all embeddings
	for _, emb := range embeddings {
		for i := 0; i < n; i++ {
			result[i] += emb[i]
		}
	}

	// Divide by count
	count := float32(len(embeddings))
	for i := 0; i < n; i++ {
		result[i] /= count
	}

	return result
}
