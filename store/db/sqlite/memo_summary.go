package sqlite

import (
	"context"
	"database/sql"

	"github.com/hrygo/divinesense/store"
	"github.com/pkg/errors"
)

// UpsertMemoSummary inserts or updates a memo summary.
func (d *DB) UpsertMemoSummary(ctx context.Context, upsert *store.UpsertMemoSummary) (*store.MemoSummary, error) {
	stmt := `
		INSERT INTO memo_summary (memo_id, summary, status, error_message, created_ts, updated_ts)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT (memo_id) DO UPDATE SET
			summary = excluded.summary,
			status = excluded.status,
			error_message = excluded.error_message,
			updated_ts = excluded.updated_ts
		RETURNING id, memo_id, summary, status, error_message, created_ts, updated_ts
	`
	var summary store.MemoSummary
	var errorMessage sql.NullString
	err := d.db.QueryRowContext(ctx, stmt,
		upsert.MemoID,
		upsert.Summary,
		upsert.Status,
		upsert.ErrorMessage,
		0, // created_ts will be set by default
		0, // updated_ts will be set by default
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
		return nil, errors.Wrap(err, "failed to upsert memo summary")
	}
	if errorMessage.Valid {
		summary.ErrorMessage = &errorMessage.String
	}
	return &summary, nil
}

// ListMemoSummarys lists memo summaries.
func (d *DB) ListMemoSummarys(ctx context.Context, find *store.FindMemoSummary) ([]*store.MemoSummary, error) {
	where, args := []string{"1 = 1"}, []any{}

	if find.MemoID != nil {
		where, args = append(where, "memo_id = ?"), append(args, *find.MemoID)
	}
	if find.ID != nil {
		where, args = append(where, "id = ?"), append(args, *find.ID)
	}
	if find.Status != nil {
		where, args = append(where, "status = ?"), append(args, *find.Status)
	}

	query := `SELECT id, memo_id, summary, status, error_message, created_ts, updated_ts
		FROM memo_summary
		WHERE ` + where[0]
	if len(where) > 1 {
		query += " AND " + where[1]
	}
	query += " ORDER BY created_ts DESC"

	if find.Limit != nil {
		query += " LIMIT ?"
		args = append(args, *find.Limit)
	}
	if find.Offset != nil {
		query += " OFFSET ?"
		args = append(args, *find.Offset)
	}

	rows, err := d.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list memo summaries")
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
			return nil, errors.Wrap(err, "failed to scan memo summary")
		}
		if errorMessage.Valid {
			summary.ErrorMessage = &errorMessage.String
		}
		summaries = append(summaries, &summary)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return summaries, nil
}

// DeleteMemoSummary deletes a memo summary.
func (d *DB) DeleteMemoSummary(ctx context.Context, memoID int32) error {
	stmt := `DELETE FROM memo_summary WHERE memo_id = ?`
	result, err := d.db.ExecContext(ctx, stmt, memoID)
	if err != nil {
		return errors.Wrap(err, "failed to delete memo summary")
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// FindMemosWithoutSummary finds memos that don't have summaries.
func (d *DB) FindMemosWithoutSummary(ctx context.Context, limit int) ([]*store.Memo, error) {
	if limit <= 0 {
		limit = 100
	}
	query := `
		SELECT m.id, m.uid, m.creator_id, m.created_ts, m.updated_ts, m.row_status,
		       m.content, m.visibility, m.pinned, m.payload
		FROM memo m
		LEFT JOIN memo_summary ms ON m.id = ms.memo_id
		WHERE ms.id IS NULL AND m.row_status = 'NORMAL'
		ORDER BY m.created_ts DESC
		LIMIT ?
	`
	rows, err := d.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, errors.Wrap(err, "failed to find memos without summary")
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
			return nil, errors.Wrap(err, "failed to scan memo")
		}
		memos = append(memos, memo)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return memos, nil
}
