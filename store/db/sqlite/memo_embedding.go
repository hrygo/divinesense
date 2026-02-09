package sqlite

import (
	"context"
	"crypto/sha1"
	"database/sql"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"regexp"
	"sort"
	"strings"

	"github.com/pkg/errors"

	storepb "github.com/hrygo/divinesense/proto/gen/store"
	"github.com/hrygo/divinesense/store"
)

// Constants for vector embedding configuration
const (
	DefaultEmbeddingDim   = 1024 // BAAI/bge-m3 dimension
	DefaultEmbeddingModel = "BAAI/bge-m3"
)

// isValidTableName validates that a table name contains only safe characters.
// This prevents SQL injection when using dynamic table names.
func isValidTableName(name string) bool {
	// Only allow alphanumeric characters and underscores
	// Table names must start with a letter or underscore
	matched, _ := regexp.MatchString(`^[a-zA-Z_][a-zA-Z0-9_]*$`, name)
	return matched && len(name) <= 64 // SQLite limit
}

// generateTempTableName generates a safe, unique temporary table name for vector search.
// It uses SHA-1 hash of userID to ensure:
// 1. Uniqueness across different users
// 2. Consistent naming for the same user
// 3. Safe table name (alphanumeric only, limited length)
func generateTempTableName(userID int32) string {
	// Hash userID to ensure safety and consistency
	h := sha1.New()
	binary.Write(h, binary.LittleEndian, userID)
	hashBytes := h.Sum(nil)

	// Use first 16 hex characters (64 bits) for table name
	// This provides 2^64 possible values, sufficient for uniqueness
	hashStr := hex.EncodeToString(hashBytes)[:16]

	return fmt.Sprintf("temp_vec_%s", hashStr)
}

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

// float32ArrayToBLOB converts a []float32 to a BLOB for sqlite-vec.
// It validates that the vector has the expected dimension.
func float32ArrayToBLOB(vec []float32) ([]byte, error) {
	if len(vec) != DefaultEmbeddingDim {
		return nil, fmt.Errorf("invalid vector dimension: got %d, want %d",
			len(vec), DefaultEmbeddingDim)
	}

	buf := make([]byte, len(vec)*4)
	for i, v := range vec {
		binary.LittleEndian.PutUint32(buf[i*4:i*4+4], math.Float32bits(v))
	}
	return buf, nil
}

// blobToFloat32Array converts a vec0 BLOB back to a float32 array.
// This is the inverse of float32ArrayToBLOB.
func blobToFloat32Array(blob []byte) ([]float32, error) {
	expectedLen := DefaultEmbeddingDim * 4
	if len(blob) != expectedLen {
		return nil, fmt.Errorf("invalid BLOB length: got %d, want %d",
			len(blob), expectedLen)
	}

	vec := make([]float32, DefaultEmbeddingDim)
	for i := 0; i < DefaultEmbeddingDim; i++ {
		bits := binary.LittleEndian.Uint32(blob[i*4 : i*4+4])
		vec[i] = math.Float32frombits(bits)
	}
	return vec, nil
}

// TestFloat32ArrayToBLOB is a test helper that exports float32ArrayToBLOB for testing.
// This is only used in test packages.
func TestFloat32ArrayToBLOB(vec []float32) ([]byte, error) {
	return float32ArrayToBLOB(vec)
}

// UpsertMemoEmbedding inserts or updates a memo embedding.
// It stores vector as BLOB in vec0 format for sqlite-vec.
func (d *DB) UpsertMemoEmbedding(ctx context.Context, embedding *store.MemoEmbedding) (*store.MemoEmbedding, error) {
	// Convert vector to BLOB for sqlite-vec
	vectorBLOB, err := float32ArrayToBLOB(embedding.Embedding)
	if err != nil {
		return nil, errors.Wrap(err, "failed to convert embedding vector to BLOB")
	}

	// SQLite stores vector as BLOB in 'embedding' column
	// PRIMARY KEY is (memo_id, model) - composite key
	stmt := `INSERT INTO memo_embedding (memo_id, embedding, model, created_ts, updated_ts)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT (memo_id, model) DO UPDATE SET
			embedding = excluded.embedding,
			updated_ts = excluded.updated_ts
		RETURNING memo_id, created_ts, updated_ts`

	err = d.db.QueryRowContext(ctx, stmt,
		embedding.MemoID,
		vectorBLOB,
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

// VectorSearch performs vector similarity search using sqlite-vec when available,
// otherwise falls back to Go-based cosine similarity computation.
//
// Performance comparison:
// - sqlite-vec: O(log n) with indexed vec0 virtual table (fast, memory-efficient)
// - Go fallback: O(n) with application-layer computation (slower, loads all vectors into memory)
func (d *DB) VectorSearch(ctx context.Context, opts *store.VectorSearchOptions) ([]*store.MemoWithScore, error) {
	// Use sqlite-vec if extension is loaded (database-layer computation)
	if d.vecExtensionLoaded {
		slog.Debug("Using sqlite-vec for vector search", "user_id", opts.UserID, "limit", opts.Limit)
		results, err := d.vectorSearchVec0(ctx, opts)
		if err == nil {
			slog.Info("Vector search completed using sqlite-vec", "user_id", opts.UserID, "result_count", len(results))
		}
		return results, err
	}
	// Fall back to Go-based computation (application-layer)
	slog.Debug("Using Go fallback for vector search", "user_id", opts.UserID)
	return d.vectorSearchGo(ctx, opts)
}

// vectorSearchVec0 performs vector similarity search using sqlite-vec's vec0 virtual table.
// This is the optimized path that uses database-layer computation (O(log n) with indexing).
// It uses the embedding_vec BLOB column and vec0 MATCH syntax.
func (d *DB) vectorSearchVec0(ctx context.Context, opts *store.VectorSearchOptions) ([]*store.MemoWithScore, error) {
	limit := opts.Limit
	if limit <= 0 {
		limit = 10
	}

	model := "BAAI/bge-m3"

	// Convert query vector to BLOB format (float32 array)
	queryVectorBLOB, err := float32ArrayToBLOB(opts.Vector)
	if err != nil {
		return nil, errors.Wrap(err, "failed to convert query vector to BLOB")
	}

	// Create a temporary vec0 virtual table for this search
	// Note: We use a session-specific temp table with a safe, unique name
	tempTableName := generateTempTableName(opts.UserID)

	// Validate table name to prevent SQL injection
	if !isValidTableName(tempTableName) {
		return nil, fmt.Errorf("invalid temporary table name: %s", tempTableName)
	}

	// Drop temp table if exists
	d.db.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s;", tempTableName))

	// Create vec0 virtual table
	//
	// FAQ: Why create a temporary virtual table for each search?
	//
	// sqlite-vec's vec0 MATCH syntax requires a virtual table as the query target.
	// The workflow is:
	// 1. Create a temp vec0 table
	// 2. Insert the query vector into it
	// 3. Use MATCH to find similar vectors in memo_embedding
	//
	// Alternative approaches considered:
	// - Permanent vec0 table: Requires maintaining sync with memo_embedding (complex)
	// - Global temp table: Would conflict with concurrent searches
	// - Current approach (per-user temp table): Simple and safe
	//
	// Performance: CREATE VIRTUAL TABLE is fast (~1ms) and the table is session-scoped
	_, err = d.db.ExecContext(ctx, fmt.Sprintf(`
		CREATE VIRTUAL TABLE %s USING vec0(
			embedding float32[%d]
		)
	`, tempTableName, DefaultEmbeddingDim))
	if err != nil {
		// If vec0 table creation fails, fall back to Go-based search
		slog.Warn("failed to create vec0 table, using Go fallback", "error", err)
		return d.vectorSearchGo(ctx, opts)
	}

	slog.Debug("vec0 temporary table created", "table", tempTableName, "user_id", opts.UserID)

	// Ensure cleanup on both success and error paths
	defer func() {
		if _, cleanupErr := d.db.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s;", tempTableName)); cleanupErr != nil {
			slog.Warn("failed to drop temporary vec0 table", "table", tempTableName, "error", cleanupErr)
		}
	}()

	// Insert query vector into temp table
	_, err = d.db.ExecContext(ctx, fmt.Sprintf(`
		INSERT INTO %s(rowid, embedding) VALUES(1, ?)
	`, tempTableName), queryVectorBLOB)
	if err != nil {
		slog.Warn("failed to insert query vector into vec0 table, using Go fallback", "error", err)
		return d.vectorSearchGo(ctx, opts)
	}

	slog.Debug("query vector inserted", "table", tempTableName, "size_bytes", len(queryVectorBLOB))

	// Build query using vec0 MATCH syntax
	// vec0 MATCH returns distance (not similarity). Convert: similarity = 1 - distance
	// We use embedding BLOB column (memo_id is the primary key)
	baseQuery := `
		SELECT
			m.id, m.uid, m.creator_id, m.created_ts, m.updated_ts, m.row_status,
			m.visibility, m.pinned, m.content, m.payload,
			(1.0 - search_results.distance) AS similarity
		FROM memo m
		INNER JOIN memo_embedding e ON m.id = e.memo_id
		INNER JOIN (
			SELECT rowid, distance
			FROM %s
			WHERE embedding MATCH ?
			ORDER BY distance
			LIMIT ?
		) search_results ON search_results.rowid = e.memo_id
		WHERE m.creator_id = ?
			AND m.row_status = 'NORMAL'
			AND e.model = ?
			AND e.embedding IS NOT NULL
	`

	// Add time-based filtering if specified
	if opts.CreatedAfter > 0 {
		baseQuery += " AND m.created_ts >= ?"
	}

	baseQuery += " ORDER BY similarity DESC, m.created_ts DESC"

	query := fmt.Sprintf(baseQuery, tempTableName)

	// Prepare args: query vector BLOB, limit, user ID, model, optional created_after
	args := []any{queryVectorBLOB, limit, opts.UserID, model}
	if opts.CreatedAfter > 0 {
		args = append(args, opts.CreatedAfter)
	}

	rows, err := d.db.QueryContext(ctx, query, args...)
	if err != nil {
		slog.Warn("vec0 search failed, using Go fallback", "error", err)
		return d.vectorSearchGo(ctx, opts)
	}
	defer rows.Close() // ✅ Ensure rows is always closed

	results := []*store.MemoWithScore{}
	for rows.Next() {
		var memo store.Memo
		var payloadBytes []byte
		var similarity float32

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
			&similarity,
		)
		if err != nil {
			return nil, errors.Wrap(err, "failed to scan vec0 search result")
		}

		// Parse payload
		if len(payloadBytes) > 0 {
			payload := &storepb.MemoPayload{}
			if err := protojsonUnmarshaler.Unmarshal(payloadBytes, payload); err != nil {
				return nil, errors.Wrap(err, "failed to unmarshal payload")
			}
			memo.Payload = payload
		}

		results = append(results, &store.MemoWithScore{
			Memo:  &memo,
			Score: similarity,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Note: Temp table cleanup is handled by defer statement (line ~240)
	return results, nil
}

// vectorSearchGo performs vector similarity search using application-layer cosine similarity.
// This is the fallback path when sqlite-vec is not available (O(n) complexity).
func (d *DB) vectorSearchGo(ctx context.Context, opts *store.VectorSearchOptions) ([]*store.MemoWithScore, error) {
	limit := opts.Limit
	if limit <= 0 {
		limit = 10
	}

	model := "BAAI/bge-m3"

	// Build optimized query with time-based filtering
	// Strategy: Filter by creation time first, then limit candidates for similarity computation
	query := `
		SELECT
			m.id, m.uid, m.creator_id, m.created_ts, m.updated_ts, m.row_status,
			m.visibility, m.pinned, m.content, m.payload,
			e.embedding
		FROM memo m
		INNER JOIN memo_embedding e ON m.id = e.memo_id
		WHERE m.creator_id = ?
			AND m.row_status = 'NORMAL'
			AND e.model = ?`

	// Add time-based filtering if specified
	args := []any{opts.UserID, model}
	if opts.CreatedAfter > 0 {
		query += " AND m.created_ts >= ?"
		args = append(args, opts.CreatedAfter)
	}

	// Order by most recent first to prioritize fresh content
	query += " ORDER BY m.created_ts DESC"

	// Limit candidates for memory-efficient similarity computation
	// Use smaller limit for better performance with large datasets
	candidateLimit := opts.MaxCandidates
	if candidateLimit <= 0 {
		candidateLimit = limit * 5 // Reduced from 10x to 5x for better performance
	}
	if candidateLimit > 500 { // Reduced from 1000 to 500 for memory efficiency
		candidateLimit = 500
	}

	query += " LIMIT ?"
	args = append(args, candidateLimit)

	rows, err := d.db.QueryContext(ctx, query, args...)
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
		var vectorBLOB []byte

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
			&vectorBLOB,
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

		// Deserialize embedding from BLOB (vec0 format)
		embedding, err := blobToFloat32Array(vectorBLOB)
		if err != nil {
			return nil, errors.Wrap(err, "failed to convert embedding BLOB to array")
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
		WHERE e.memo_id IS NULL
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
