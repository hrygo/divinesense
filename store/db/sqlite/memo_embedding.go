package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"math"
	"sort"
	"strings"

	"github.com/pkg/errors"

	storepb "github.com/hrygo/divinesense/proto/gen/store"
	"github.com/hrygo/divinesense/store"
)

// ============================================================================
// SQLITE AI FEATURES - FULL SUPPORT
// ============================================================================
// SQLite now supports vector storage and similarity search.
//
// Implementation details:
// - Vectors are stored as BLOB (JSON-encoded float32 arrays)
// - Similarity search is computed in Go application layer
// - BM25 search uses SQLite FTS5 (if available)
// ============================================================================

// UpsertMemoEmbedding inserts or updates a memo embedding.
func (d *DB) UpsertMemoEmbedding(ctx context.Context, embedding *store.MemoEmbedding) (*store.MemoEmbedding, error) {
	// Serialize vector to JSON
	vectorJSON, err := json.Marshal(embedding.Embedding)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal embedding vector")
	}

	stmt := `INSERT INTO memo_embedding (memo_id, embedding, model, created_ts, updated_ts)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT (memo_id, model) DO UPDATE SET
			embedding = excluded.embedding,
			updated_ts = excluded.updated_ts
		RETURNING id, created_ts, updated_ts`

	err = d.db.QueryRowContext(ctx, stmt,
		embedding.MemoID,
		vectorJSON,
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
		where, args = append(where, "memo_id = ?"), append(args, *find.MemoID)
	}
	if find.Model != nil {
		where, args = append(where, "model = ?"), append(args, *find.Model)
	}

	query := `SELECT id, memo_id, embedding, model, created_ts, updated_ts
		FROM memo_embedding
		WHERE ` + where[0]
	if len(where) > 1 {
		query += " AND " + where[1]
	}
	query += " ORDER BY created_ts DESC"

	rows, err := d.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list memo embeddings")
	}
	defer rows.Close()

	list := []*store.MemoEmbedding{}
	for rows.Next() {
		var embedding store.MemoEmbedding
		var vectorJSON []byte

		err := rows.Scan(
			&embedding.ID,
			&embedding.MemoID,
			&vectorJSON,
			&embedding.Model,
			&embedding.CreatedTs,
			&embedding.UpdatedTs,
		)
		if err != nil {
			return nil, errors.Wrap(err, "failed to scan memo embedding")
		}

		// Deserialize vector
		if err := json.Unmarshal(vectorJSON, &embedding.Embedding); err != nil {
			return nil, errors.Wrap(err, "failed to unmarshal embedding vector")
		}

		list = append(list, &embedding)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return list, nil
}

// DeleteMemoEmbedding deletes a memo embedding.
func (d *DB) DeleteMemoEmbedding(ctx context.Context, memoID int32) error {
	stmt := `DELETE FROM memo_embedding WHERE memo_id = ?`
	result, err := d.db.ExecContext(ctx, stmt, memoID)
	if err != nil {
		return errors.Wrap(err, "failed to delete memo embedding")
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// VectorSearch performs vector similarity search.
// Since SQLite doesn't have native vector operations, we compute cosine similarity in Go.
func (d *DB) VectorSearch(ctx context.Context, opts *store.VectorSearchOptions) ([]*store.MemoWithScore, error) {
	limit := opts.Limit
	if limit <= 0 {
		limit = 10
	}

	model := "BAAI/bge-m3"

	// Fetch all embeddings for the user and model
	query := `
		SELECT
			m.id, m.uid, m.creator_id, m.created_ts, m.updated_ts, m.row_status,
			m.visibility, m.pinned, m.content, m.payload,
			e.embedding
		FROM memo m
		INNER JOIN memo_embedding e ON m.id = e.memo_id
		WHERE m.creator_id = ?
			AND m.row_status = 'NORMAL'
			AND e.model = ?
		LIMIT ?`

	// Set a high limit to fetch candidates, then we'll rank them in Go
	// For better performance with large datasets, consider adding a WHERE clause
	// for initial filtering (e.g., by tags, time range, etc.)
	candidateLimit := limit * 10
	if candidateLimit > 1000 {
		candidateLimit = 1000 // Cap to prevent excessive memory usage
	}

	rows, err := d.db.QueryContext(ctx, query, opts.UserID, model, candidateLimit)
	if err != nil {
		return nil, errors.Wrap(err, "failed to vector search")
	}
	defer rows.Close()

	// Collect candidates
	type candidate struct {
		memo      *store.Memo
		embedding []float32
	}
	candidates := []candidate{}

	for rows.Next() {
		var memo store.Memo
		var payloadBytes []byte
		var vectorJSON []byte

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
			&vectorJSON,
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

		// Deserialize embedding
		var embedding []float32
		if err := json.Unmarshal(vectorJSON, &embedding); err != nil {
			return nil, errors.Wrap(err, "failed to unmarshal embedding vector")
		}

		candidates = append(candidates, candidate{
			memo:      &memo,
			embedding: embedding,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Compute cosine similarity and rank
	type scoredResult struct {
		memo  *store.Memo
		score float32
	}
	results := []scoredResult{}

	for _, cand := range candidates {
		similarity := cosineSimilarity(opts.Vector, cand.embedding)
		results = append(results, scoredResult{
			memo:  cand.memo,
			score: similarity,
		})
	}

	// Sort by similarity (descending)
	sort.Slice(results, func(i, j int) bool {
		return results[i].score > results[j].score
	})

	// Return top-k
	finalResults := []*store.MemoWithScore{}
	for i := 0; i < len(results) && i < limit; i++ {
		finalResults = append(finalResults, &store.MemoWithScore{
			Memo:  results[i].memo,
			Score: results[i].score,
		})
	}

	return finalResults, nil
}

// cosineSimilarity computes cosine similarity between two vectors.
func cosineSimilarity(a, b []float32) float32 {
	if len(a) != len(b) {
		return 0
	}

	var dotProduct float32
	var normA float32
	var normB float32

	for i := 0; i < len(a); i++ {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dotProduct / (float32(math.Sqrt(float64(normA))) * float32(math.Sqrt(float64(normB))))
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
		LEFT JOIN memo_embedding e ON m.id = e.memo_id AND e.model = ?
		WHERE e.id IS NULL
			AND m.row_status = 'NORMAL'
			AND LENGTH(m.content) > 0
		ORDER BY m.created_ts DESC
		LIMIT ?`

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

// BM25Search performs full-text search using SQLite FTS5 if available.
func (d *DB) BM25Search(ctx context.Context, opts *store.BM25SearchOptions) ([]*store.BM25Result, error) {
	query := `
		SELECT
			m.id, m.uid, m.creator_id, m.created_ts, m.updated_ts, m.row_status,
			m.visibility, m.pinned, m.content, m.payload,
			bm25(memo_fts) AS score
		FROM memo m
		LEFT JOIN memo_fts ON m.id = memo_fts.rowid
		WHERE m.creator_id = ?
			AND m.row_status = 'NORMAL'
			AND memo_fts MATCH ?
		ORDER BY score DESC, m.updated_ts DESC
		LIMIT ?
	`

	rows, err := d.db.QueryContext(ctx, query, opts.UserID, opts.Query, opts.Limit)
	if err != nil {
		return d.bm25SearchFallback(ctx, opts)
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

		if len(payloadBytes) > 0 {
			payload := &storepb.MemoPayload{}
			if err := protojsonUnmarshaler.Unmarshal(payloadBytes, payload); err != nil {
				return nil, errors.Wrap(err, "failed to unmarshal payload")
			}
			memo.Payload = payload
		}

		result.Memo = &memo
		if result.Score >= opts.MinScore {
			results = append(results, &result)
		}
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return results, nil
}

func (d *DB) bm25SearchFallback(ctx context.Context, opts *store.BM25SearchOptions) ([]*store.BM25Result, error) {
	words := []string{}
	fields := strings.Fields(opts.Query)
	for _, word := range fields {
		if len(word) > 0 {
			escaped := strings.ReplaceAll(strings.ReplaceAll(word, "%", "\\%"), "_", "\\_")
			words = append(words, "%"+escaped+"%")
		}
	}

	if len(words) == 0 {
		return []*store.BM25Result{}, nil
	}

	whereClause := strings.Repeat("AND m.content LIKE ? ", len(words))
	args := make([]any, 0, len(words)+1)
	args = append(args, opts.UserID)
	for _, word := range words {
		args = append(args, word)
	}
	args = append(args, opts.Limit)

	query := `
		SELECT
			m.id, m.uid, m.creator_id, m.created_ts, m.updated_ts, m.row_status,
			m.visibility, m.pinned, m.content, m.payload,
			COUNT(*) AS score
		FROM memo m
		WHERE m.creator_id = ?
			AND m.row_status = 'NORMAL'
			` + whereClause + `
		GROUP BY m.id, m.uid, m.creator_id, m.created_ts, m.updated_ts, m.row_status,
			m.visibility, m.pinned, m.content, m.payload
		ORDER BY score DESC, m.updated_ts DESC
		LIMIT ?
	`

	rows, err := d.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.Wrap(err, "failed to BM25 search fallback")
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

		if len(payloadBytes) > 0 {
			payload := &storepb.MemoPayload{}
			if err := protojsonUnmarshaler.Unmarshal(payloadBytes, payload); err != nil {
				return nil, errors.Wrap(err, "failed to unmarshal payload")
			}
			memo.Payload = payload
		}

		result.Memo = &memo
		if result.Score >= opts.MinScore {
			results = append(results, &result)
		}
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return results, nil
}
