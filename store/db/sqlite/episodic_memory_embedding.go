package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"sort"
	"time"

	"github.com/pkg/errors"

	"github.com/hrygo/divinesense/store"
)

// UpsertEpisodicMemoryEmbedding inserts or updates an episodic memory embedding.
// It stores vector as BLOB in vec0 format for sqlite-vec.
func (d *DB) UpsertEpisodicMemoryEmbedding(ctx context.Context, embedding *store.EpisodicMemoryEmbedding) (*store.EpisodicMemoryEmbedding, error) {
	// Convert vector to BLOB for sqlite-vec
	vectorBLOB, err := float32ArrayToBLOB(embedding.Embedding)
	if err != nil {
		return nil, errors.Wrap(err, "failed to convert embedding vector to BLOB")
	}

	now := time.Now().Unix()
	if embedding.CreatedTs == 0 {
		embedding.CreatedTs = now
	}
	embedding.UpdatedTs = now

	stmt := `INSERT INTO episodic_memory_embedding (episodic_memory_id, embedding, model, created_ts, updated_ts)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT (episodic_memory_id, model) DO UPDATE SET
			embedding = excluded.embedding,
			updated_ts = excluded.updated_ts
		RETURNING id, created_ts, updated_ts`

	err = d.db.QueryRowContext(ctx, stmt,
		embedding.EpisodicMemoryID,
		vectorBLOB,
		embedding.Model,
		embedding.CreatedTs,
		embedding.UpdatedTs,
	).Scan(&embedding.ID, &embedding.CreatedTs, &embedding.UpdatedTs)

	if err != nil {
		return nil, errors.Wrap(err, "failed to upsert episodic memory embedding")
	}

	return embedding, nil
}

// ListEpisodicMemoryEmbeddings lists episodic memory embeddings.
func (d *DB) ListEpisodicMemoryEmbeddings(ctx context.Context, find *store.FindEpisodicMemoryEmbedding) ([]*store.EpisodicMemoryEmbedding, error) {
	where, args := []string{"1 = 1"}, []any{}

	if find.EpisodicMemoryID != nil {
		where, args = append(where, "episodic_memory_id = ?"), append(args, *find.EpisodicMemoryID)
	}
	if find.Model != nil {
		where, args = append(where, "model = ?"), append(args, *find.Model)
	}

	query := `SELECT id, episodic_memory_id, embedding, model, created_ts, updated_ts
		FROM episodic_memory_embedding
		WHERE ` + where[0]
	if len(where) > 1 {
		query += " AND " + where[1]
	}
	query += " ORDER BY created_ts DESC"

	rows, err := d.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list episodic memory embeddings")
	}
	defer rows.Close()

	list := []*store.EpisodicMemoryEmbedding{}
	for rows.Next() {
		var embedding store.EpisodicMemoryEmbedding
		var vectorBLOB []byte

		err := rows.Scan(
			&embedding.ID,
			&embedding.EpisodicMemoryID,
			&vectorBLOB,
			&embedding.Model,
			&embedding.CreatedTs,
			&embedding.UpdatedTs,
		)
		if err != nil {
			return nil, errors.Wrap(err, "failed to scan episodic memory embedding")
		}

		// Deserialize vector from BLOB
		embedding.Embedding, err = blobToFloat32Array(vectorBLOB)
		if err != nil {
			return nil, errors.Wrap(err, "failed to convert embedding BLOB to array")
		}

		list = append(list, &embedding)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return list, nil
}

// DeleteEpisodicMemoryEmbedding deletes an episodic memory embedding.
func (d *DB) DeleteEpisodicMemoryEmbedding(ctx context.Context, episodicMemoryID int32) error {
	stmt := `DELETE FROM episodic_memory_embedding WHERE episodic_memory_id = ?`
	result, err := d.db.ExecContext(ctx, stmt, episodicMemoryID)
	if err != nil {
		return errors.Wrap(err, "failed to delete episodic memory embedding")
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// FindEpisodicMemoriesWithoutEmbedding finds episodic memories that don't have embeddings for the specified model.
func (d *DB) FindEpisodicMemoriesWithoutEmbedding(ctx context.Context, find *store.FindEpisodicMemoriesWithoutEmbedding) ([]*store.EpisodicMemory, error) {
	limit := find.Limit
	if limit <= 0 {
		limit = 100
	}

	query := `
		SELECT
			em.id, em.user_id, em.timestamp, em.agent_type, em.user_input, em.outcome, em.summary, em.importance, em.created_ts
		FROM episodic_memory em
		LEFT JOIN episodic_memory_embedding e ON em.id = e.episodic_memory_id AND e.model = ?
		WHERE e.id IS NULL
			AND LENGTH(em.user_input) > 0
		ORDER BY em.created_ts DESC
		LIMIT ?`

	rows, err := d.db.QueryContext(ctx, query, find.Model, limit)
	if err != nil {
		return nil, errors.Wrap(err, "failed to find episodic memories without embedding")
	}
	defer rows.Close()

	list := []*store.EpisodicMemory{}
	for rows.Next() {
		var memory store.EpisodicMemory
		err := rows.Scan(
			&memory.ID,
			&memory.UserID,
			&memory.Timestamp,
			&memory.AgentType,
			&memory.UserInput,
			&memory.Outcome,
			&memory.Summary,
			&memory.Importance,
			&memory.CreatedTs,
		)
		if err != nil {
			return nil, errors.Wrap(err, "failed to scan episodic memory")
		}

		list = append(list, &memory)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return list, nil
}

// EpisodicVectorSearch performs vector similarity search on episodic memories.
// Uses Go-based cosine similarity computation (application-layer).
func (d *DB) EpisodicVectorSearch(ctx context.Context, opts *store.EpisodicVectorSearchOptions) ([]*store.EpisodicMemoryWithScore, error) {
	limit := opts.Limit
	if limit <= 0 {
		limit = 10
	}

	model := DefaultEmbeddingModel

	// Build query with optional filters
	whereClause := "em.user_id = ? AND e.model = ?"
	args := []any{opts.UserID, model}

	if opts.AgentType != nil {
		whereClause += " AND em.agent_type = ?"
		args = append(args, *opts.AgentType)
	}
	if opts.CreatedAfter > 0 {
		whereClause += " AND em.created_ts >= ?"
		args = append(args, opts.CreatedAfter)
	}

	query := fmt.Sprintf(`
		SELECT
			em.id, em.user_id, em.timestamp, em.agent_type, em.user_input, em.outcome, em.summary, em.importance, em.created_ts,
			e.embedding
		FROM episodic_memory em
		INNER JOIN episodic_memory_embedding e ON em.id = e.episodic_memory_id
		WHERE %s
		ORDER BY em.created_ts DESC
		LIMIT ?
	`, whereClause)

	// Limit candidates for memory-efficient similarity computation
	candidateLimit := limit * 5
	if candidateLimit > 500 {
		candidateLimit = 500
	}
	args = append(args, candidateLimit)

	rows, err := d.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.Wrap(err, "failed to episodic vector search")
	}
	defer rows.Close()

	// Collect candidates
	type candidate struct {
		memory    *store.EpisodicMemory
		embedding []float32
	}
	candidates := []candidate{}

	for rows.Next() {
		var memory store.EpisodicMemory
		var vectorBLOB []byte

		err := rows.Scan(
			&memory.ID,
			&memory.UserID,
			&memory.Timestamp,
			&memory.AgentType,
			&memory.UserInput,
			&memory.Outcome,
			&memory.Summary,
			&memory.Importance,
			&memory.CreatedTs,
			&vectorBLOB,
		)
		if err != nil {
			return nil, errors.Wrap(err, "failed to scan episodic vector search result")
		}

		// Deserialize embedding from BLOB
		embedding, err := blobToFloat32Array(vectorBLOB)
		if err != nil {
			slog.Warn("failed to convert embedding BLOB to array", "memory_id", memory.ID, "error", err)
			continue
		}

		candidates = append(candidates, candidate{
			memory:    &memory,
			embedding: embedding,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Compute cosine similarity and rank
	type scoredResult struct {
		memory *store.EpisodicMemory
		score  float32
	}
	results := []scoredResult{}

	for _, cand := range candidates {
		similarity := cosineSimilarity(opts.Vector, cand.embedding)
		results = append(results, scoredResult{
			memory: cand.memory,
			score:  similarity,
		})
	}

	// Sort by similarity (descending)
	sort.Slice(results, func(i, j int) bool {
		return results[i].score > results[j].score
	})

	// Return top-k
	finalResults := []*store.EpisodicMemoryWithScore{}
	for i := 0; i < len(results) && i < limit; i++ {
		finalResults = append(finalResults, &store.EpisodicMemoryWithScore{
			EpisodicMemory: results[i].memory,
			Score:          results[i].score,
		})
	}

	return finalResults, nil
}
