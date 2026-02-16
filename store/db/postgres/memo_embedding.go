package postgres

import (
	"context"
	"fmt"
	"strings"

	"github.com/pgvector/pgvector-go"
	"github.com/pkg/errors"

	storepb "github.com/hrygo/divinesense/proto/gen/store"
	"github.com/hrygo/divinesense/store"
)

// UpsertMemoEmbedding inserts or updates a memo embedding.
func (d *DB) UpsertMemoEmbedding(ctx context.Context, embedding *store.MemoEmbedding) (*store.MemoEmbedding, error) {

	stmt := `
		INSERT INTO memo_embedding (memo_id, embedding, model, created_ts, updated_ts)
		VALUES (` + placeholders(5) + `)
		ON CONFLICT (memo_id, model)
		DO UPDATE SET
			embedding = EXCLUDED.embedding,
			updated_ts = EXCLUDED.updated_ts
		RETURNING id, created_ts, updated_ts
	`

	vector := pgvector.NewVector(embedding.Embedding)
	err := d.db.QueryRowContext(ctx, stmt,
		embedding.MemoID,
		vector,
		embedding.Model,
		embedding.CreatedTs,
		embedding.UpdatedTs,
	).Scan(&embedding.ID, &embedding.CreatedTs, &embedding.UpdatedTs)

	if err != nil {
		return nil, errors.Wrap(err, "failed to upsert memo embedding")
	}

	return embedding, nil
}

// ListMemoEmbeddings lists memo embeddings.
func (d *DB) ListMemoEmbeddings(ctx context.Context, find *store.FindMemoEmbedding) ([]*store.MemoEmbedding, error) {
	where, args := []string{"1 = 1"}, []any{}

	if find.MemoID != nil {
		where, args = append(where, "memo_id = "+placeholder(len(args)+1)), append(args, *find.MemoID)
	}
	if find.Model != nil {
		where, args = append(where, "model = "+placeholder(len(args)+1)), append(args, *find.Model)
	}

	query := `
		SELECT id, memo_id, embedding, model, created_ts, updated_ts
		FROM memo_embedding
		WHERE ` + strings.Join(where, " AND ") + `
		ORDER BY created_ts DESC
	`

	rows, err := d.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list memo embeddings")
	}
	defer rows.Close()

	list := []*store.MemoEmbedding{}
	for rows.Next() {
		var embedding store.MemoEmbedding
		var vector pgvector.Vector
		err := rows.Scan(
			&embedding.ID,
			&embedding.MemoID,
			&vector,
			&embedding.Model,
			&embedding.CreatedTs,
			&embedding.UpdatedTs,
		)
		if err != nil {
			return nil, errors.Wrap(err, "failed to scan memo embedding")
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

// DeleteMemoEmbedding deletes a memo embedding.
func (d *DB) DeleteMemoEmbedding(ctx context.Context, memoID int32) error {
	stmt := `DELETE FROM memo_embedding WHERE memo_id = ` + placeholder(1)
	result, err := d.db.ExecContext(ctx, stmt, memoID)
	if err != nil {
		return errors.Wrap(err, "failed to delete memo embedding")
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("memo embedding with memo_id %d not found", memoID)
	}
	return nil
}

// VectorSearch performs vector similarity search using pgvector.
func (d *DB) VectorSearch(ctx context.Context, opts *store.VectorSearchOptions) ([]*store.MemoWithScore, error) {

	limit := opts.Limit
	if limit <= 0 {
		limit = 10
	}

	// Use cosine similarity with pgvector
	// The <=> operator computes cosine distance (1 - cosine_similarity)
	// So we order by distance ASC to get most similar first
	query := `
		SELECT
			m.id, m.uid, m.creator_id, m.created_ts, m.updated_ts, m.row_status,
			m.visibility, m.pinned, m.content, m.payload,
			1 - (e.embedding <=> ` + placeholder(1) + `) AS score
		FROM memo m
		INNER JOIN memo_embedding e ON m.id = e.memo_id
		WHERE m.creator_id = ` + placeholder(2) + `
			AND m.row_status = 'NORMAL'
			AND e.model = ` + placeholder(3) + `
		ORDER BY e.embedding <=> ` + placeholder(4) + `
		LIMIT ` + placeholder(5)

	// Use default model if not specified
	model := "BAAI/bge-m3"

	vector := pgvector.NewVector(opts.Vector)
	rows, err := d.db.QueryContext(ctx, query,
		vector,
		opts.UserID,
		model,
		vector,
		limit,
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to vector search")
	}
	defer rows.Close()

	results := []*store.MemoWithScore{}
	for rows.Next() {
		var result store.MemoWithScore
		var memo store.Memo
		var payloadBytes []byte

		err := rows.Scan(
			&memo.ID,
			&memo.UID,
			&memo.CreatorID,
			&memo.CreatedTs,
			&memo.UpdatedTs,
			&memo.RowStatus,
			&memo.Visibility,
			&memo.Pinned,
			&memo.Content,
			&payloadBytes,
			&result.Score,
		)
		if err != nil {
			return nil, errors.Wrap(err, "failed to scan vector search result")
		}

		// Parse payload
		if len(payloadBytes) > 0 {
			payload := &storepb.MemoPayload{}
			if err := protojsonUnmarshaler.Unmarshal(payloadBytes, payload); err != nil {
				return nil, errors.Wrap(err, "failed to unmarshal payload")
			}
			memo.Payload = payload
		}

		result.Memo = &memo
		results = append(results, &result)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return results, nil
}

// FindMemosWithoutEmbedding finds memos that don't have embeddings for the specified model.
func (d *DB) FindMemosWithoutEmbedding(ctx context.Context, find *store.FindMemosWithoutEmbedding) ([]*store.Memo, error) {
	limit := find.Limit
	if limit <= 0 {
		limit = 100
	}

	query := `
		SELECT
			m.id, m.uid, m.creator_id, m.created_ts, m.updated_ts, m.row_status,
			m.visibility, m.pinned, m.content, m.payload
		FROM memo m
		LEFT JOIN memo_embedding e ON m.id = e.memo_id AND e.model = ` + placeholder(1) + `
		WHERE e.id IS NULL
			AND m.row_status = 'NORMAL'
			AND LENGTH(m.content) > 0
		ORDER BY m.created_ts DESC
		LIMIT ` + placeholder(2)

	rows, err := d.db.QueryContext(ctx, query, find.Model, limit)
	if err != nil {
		return nil, errors.Wrap(err, "failed to find memos without embedding")
	}
	defer rows.Close()

	list := []*store.Memo{}
	for rows.Next() {
		var memo store.Memo
		var payloadBytes []byte

		err := rows.Scan(
			&memo.ID,
			&memo.UID,
			&memo.CreatorID,
			&memo.CreatedTs,
			&memo.UpdatedTs,
			&memo.RowStatus,
			&memo.Visibility,
			&memo.Pinned,
			&memo.Content,
			&payloadBytes,
		)
		if err != nil {
			return nil, errors.Wrap(err, "failed to scan memo")
		}

		// Parse payload
		if len(payloadBytes) > 0 {
			payload := &storepb.MemoPayload{}
			if err := protojsonUnmarshaler.Unmarshal(payloadBytes, payload); err != nil {
				return nil, errors.Wrap(err, "failed to unmarshal payload")
			}
			memo.Payload = payload
		}

		list = append(list, &memo)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return list, nil
}

// BM25Search performs full-text search using PostgreSQL's ts_vector with BM25 ranking.
// Uses bilingual text search configuration for better Chinese and English support.
func (d *DB) BM25Search(ctx context.Context, opts *store.BM25SearchOptions) ([]*store.BM25Result, error) {
	limit := opts.Limit
	if limit <= 0 {
		limit = 10
	}

	// Use PostgreSQL's full-text search with bilingual support
	// The to_tsvector_chinese/to_tsquery_chinese functions auto-detect Chinese vs English
	// Note: ts_rank with normalization (32) divides by document length for fairer scoring
	query := `
		SELECT
			m.id, m.uid, m.creator_id, m.created_ts, m.updated_ts, m.row_status,
			m.visibility, m.pinned, m.content, m.payload,
			ts_rank(to_tsvector_chinese(m.content), to_tsquery_chinese(` + placeholder(1) + `), 32) AS score
		FROM memo m
		WHERE m.creator_id = ` + placeholder(2) + `
			AND row_status = 'NORMAL'
			AND to_tsvector_chinese(m.content) @@ to_tsquery_chinese(` + placeholder(3) + `)
		ORDER BY score DESC, m.updated_ts DESC
		LIMIT ` + placeholder(4)

	rows, err := d.db.QueryContext(ctx, query, opts.Query, opts.UserID, opts.Query, limit)
	if err != nil {
		return nil, errors.Wrap(err, "failed to BM25 search")
	}
	defer rows.Close()

	results := []*store.BM25Result{}
	for rows.Next() {
		var result store.BM25Result
		var memo store.Memo
		var payloadBytes []byte

		err := rows.Scan(
			&memo.ID,
			&memo.UID,
			&memo.CreatorID,
			&memo.CreatedTs,
			&memo.UpdatedTs,
			&memo.RowStatus,
			&memo.Visibility,
			&memo.Pinned,
			&memo.Content,
			&payloadBytes,
			&result.Score,
		)
		if err != nil {
			return nil, errors.Wrap(err, "failed to scan BM25 result")
		}

		// Parse payload
		if len(payloadBytes) > 0 {
			payload := &storepb.MemoPayload{}
			if err := protojsonUnmarshaler.Unmarshal(payloadBytes, payload); err != nil {
				return nil, errors.Wrap(err, "failed to unmarshal payload")
			}
			memo.Payload = payload
		}

		result.Memo = &memo

		// Best practice: Don't apply MinScore filter after DB has already sorted and limited
		// The database query already ensures we get the most relevant results
		// Only filter if MinScore is explicitly set to a meaningful value (> 0)
		if opts.MinScore <= 0 || result.Score >= opts.MinScore {
			results = append(results, &result)
		}
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return results, nil
}
