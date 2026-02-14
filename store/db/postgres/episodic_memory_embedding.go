package postgres

import (
	"context"
	"fmt"
	"strings"

	"github.com/pgvector/pgvector-go"
	"github.com/pkg/errors"

	"github.com/hrygo/divinesense/store"
)

// UpsertEpisodicMemoryEmbedding inserts or updates an episodic memory embedding.
func (d *DB) UpsertEpisodicMemoryEmbedding(ctx context.Context, embedding *store.EpisodicMemoryEmbedding) (*store.EpisodicMemoryEmbedding, error) {
	stmt := `
		INSERT INTO episodic_memory_embedding (episodic_memory_id, embedding, model, created_ts, updated_ts)
		VALUES (` + placeholders(5) + `)
		ON CONFLICT (episodic_memory_id, model)
		DO UPDATE SET
			embedding = EXCLUDED.embedding,
			updated_ts = EXCLUDED.updated_ts
		RETURNING id, created_ts, updated_ts
	`

	vector := pgvector.NewVector(embedding.Embedding)
	err := d.db.QueryRowContext(ctx, stmt,
		embedding.EpisodicMemoryID,
		vector,
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
		where, args = append(where, "episodic_memory_id = "+placeholder(len(args)+1)), append(args, *find.EpisodicMemoryID)
	}
	if find.Model != nil {
		where, args = append(where, "model = "+placeholder(len(args)+1)), append(args, *find.Model)
	}

	query := `
		SELECT id, episodic_memory_id, embedding, model, created_ts, updated_ts
		FROM episodic_memory_embedding
		WHERE ` + strings.Join(where, " AND ") + `
		ORDER BY created_ts DESC
	`

	rows, err := d.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list episodic memory embeddings")
	}
	defer rows.Close()

	list := []*store.EpisodicMemoryEmbedding{}
	for rows.Next() {
		var embedding store.EpisodicMemoryEmbedding
		var vector pgvector.Vector
		err := rows.Scan(
			&embedding.ID,
			&embedding.EpisodicMemoryID,
			&vector,
			&embedding.Model,
			&embedding.CreatedTs,
			&embedding.UpdatedTs,
		)
		if err != nil {
			return nil, errors.Wrap(err, "failed to scan episodic memory embedding")
		}

		// Convert vector
		embedding.Embedding = vector.Slice()

		list = append(list, &embedding)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return list, nil
}

// DeleteEpisodicMemoryEmbedding deletes an episodic memory embedding.
func (d *DB) DeleteEpisodicMemoryEmbedding(ctx context.Context, episodicMemoryID int32) error {
	stmt := `DELETE FROM episodic_memory_embedding WHERE episodic_memory_id = ` + placeholder(1)
	result, err := d.db.ExecContext(ctx, stmt, episodicMemoryID)
	if err != nil {
		return errors.Wrap(err, "failed to delete episodic memory embedding")
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("episodic memory embedding with episodic_memory_id %d not found", episodicMemoryID)
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
		LEFT JOIN episodic_memory_embedding e ON em.id = e.episodic_memory_id AND e.model = ` + placeholder(1) + `
		WHERE e.id IS NULL
			AND LENGTH(em.user_input) > 0
		ORDER BY em.created_ts DESC
		LIMIT ` + placeholder(2)

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

// EpisodicVectorSearch performs vector similarity search using pgvector.
func (d *DB) EpisodicVectorSearch(ctx context.Context, opts *store.EpisodicVectorSearchOptions) ([]*store.EpisodicMemoryWithScore, error) {
	limit := opts.Limit
	if limit <= 0 {
		limit = 10
	}

	where, args := []string{"em.user_id = " + placeholder(1)}, []any{opts.UserID}
	argIdx := 2

	if opts.AgentType != nil {
		where = append(where, "em.agent_type = "+placeholder(argIdx))
		args = append(args, *opts.AgentType)
		argIdx++
	}
	if opts.CreatedAfter > 0 {
		where = append(where, "em.created_ts >= "+placeholder(argIdx))
		args = append(args, opts.CreatedAfter)
		argIdx++
	}

	// Use cosine similarity with pgvector
	// The <=> operator computes cosine distance (1 - cosine_similarity)
	// So we order by distance ASC to get most similar first
	query := `
		SELECT
			em.id, em.user_id, em.timestamp, em.agent_type, em.user_input, em.outcome, em.summary, em.importance, em.created_ts,
			1 - (e.embedding <=> ` + placeholder(argIdx) + `) AS score
		FROM episodic_memory em
		INNER JOIN episodic_memory_embedding e ON em.id = e.episodic_memory_id
		WHERE ` + strings.Join(where, " AND ") + `
			AND e.model = ` + placeholder(argIdx+1) + `
		ORDER BY e.embedding <=> ` + placeholder(argIdx+2) + `
		LIMIT ` + placeholder(argIdx+3)

	// Use default model if not specified
	model := "BAAI/bge-m3"

	vector := pgvector.NewVector(opts.Vector)
	args = append(args, vector, model, vector, limit)

	rows, err := d.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.Wrap(err, "failed to episodic vector search")
	}
	defer rows.Close()

	results := []*store.EpisodicMemoryWithScore{}
	for rows.Next() {
		var result store.EpisodicMemoryWithScore
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
			&result.Score,
		)
		if err != nil {
			return nil, errors.Wrap(err, "failed to scan episodic vector search result")
		}

		result.EpisodicMemory = &memory
		results = append(results, &result)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return results, nil
}
