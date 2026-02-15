package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/hrygo/divinesense/store"
)

func (db *DB) UpsertMemoSummary(ctx context.Context, upsert *store.UpsertMemoSummary) (*store.MemoSummary, error) {
	query := `
		INSERT INTO memo_summary (memo_id, summary, status, error_message)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (memo_id) DO UPDATE SET
			summary = EXCLUDED.summary,
			status = EXCLUDED.status,
			error_message = EXCLUDED.error_message,
			updated_ts = EXTRACT(EPOCH FROM NOW())::BIGINT
		RETURNING id, memo_id, summary, status, error_message, created_ts, updated_ts
	`
	var summary store.MemoSummary
	var errorMessage sql.NullString
	err := db.db.QueryRowContext(ctx, query,
		upsert.MemoID,
		upsert.Summary,
		upsert.Status,
		upsert.ErrorMessage,
	).Scan(
		&summary.ID,
		&summary.MemoID,
		&summary.Summary,
		&summary.Status,
		&errorMessage,
		&summary.CreatedTs,
		&summary.UpdatedTs,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to upsert memo summary: %w", err)
	}
	if errorMessage.Valid {
		summary.ErrorMessage = &errorMessage.String
	}
	return &summary, nil
}

func (db *DB) ListMemoSummarys(ctx context.Context, find *store.FindMemoSummary) ([]*store.MemoSummary, error) {
	query := `
		SELECT id, memo_id, summary, status, error_message, created_ts, updated_ts
		FROM memo_summary
		WHERE 1=1
	`
	var args []interface{}
	argIndex := 1

	if find.MemoID != nil {
		query += fmt.Sprintf(" AND memo_id = $%d", argIndex)
		args = append(args, *find.MemoID)
		argIndex++
	}
	if find.ID != nil {
		query += fmt.Sprintf(" AND id = $%d", argIndex)
		args = append(args, *find.ID)
		argIndex++
	}
	if find.Status != nil {
		query += fmt.Sprintf(" AND status = $%d", argIndex)
		args = append(args, *find.Status)
		argIndex++
	}

	if find.Limit != nil {
		query += fmt.Sprintf(" LIMIT $%d", argIndex)
		args = append(args, *find.Limit)
		argIndex++
	}
	if find.Offset != nil {
		query += fmt.Sprintf(" OFFSET $%d", argIndex)
		args = append(args, *find.Offset)
	}

	rows, err := db.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list memo summaries: %w", err)
	}
	defer rows.Close()

	var summaries []*store.MemoSummary
	for rows.Next() {
		var summary store.MemoSummary
		var errorMessage sql.NullString
		err := rows.Scan(
			&summary.ID,
			&summary.MemoID,
			&summary.Summary,
			&summary.Status,
			&errorMessage,
			&summary.CreatedTs,
			&summary.UpdatedTs,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan memo summary: %w", err)
		}
		if errorMessage.Valid {
			summary.ErrorMessage = &errorMessage.String
		}
		summaries = append(summaries, &summary)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to list memo summaries: %w", err)
	}
	return summaries, nil
}

func (db *DB) DeleteMemoSummary(ctx context.Context, memoID int32) error {
	query := `DELETE FROM memo_summary WHERE memo_id = $1`
	_, err := db.db.ExecContext(ctx, query, memoID)
	if err != nil {
		return fmt.Errorf("failed to delete memo summary: %w", err)
	}
	return nil
}

func (db *DB) FindMemosWithoutSummary(ctx context.Context, limit int) ([]*store.Memo, error) {
	query := `
		SELECT m.id, m.uid, m.creator_id, m.created_ts, m.updated_ts, m.row_status,
		       m.content, m.visibility, m.pinned, m.payload
		FROM memo m
		LEFT JOIN memo_summary ms ON m.id = ms.memo_id
		WHERE ms.id IS NULL AND m.row_status = 'NORMAL'
		ORDER BY m.created_ts DESC
		LIMIT $1
	`
	rows, err := db.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to find memos without summary: %w", err)
	}
	defer rows.Close()

	var memos []*store.Memo
	for rows.Next() {
		memo := &store.Memo{}
		err := rows.Scan(
			&memo.ID,
			&memo.UID,
			&memo.CreatorID,
			&memo.CreatedTs,
			&memo.UpdatedTs,
			&memo.RowStatus,
			&memo.Content,
			&memo.Visibility,
			&memo.Pinned,
			&memo.Payload,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan memo: %w", err)
		}
		memos = append(memos, memo)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to find memos without summary: %w", err)
	}
	return memos, nil
}
