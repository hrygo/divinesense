package postgres

import (
	"context"
	"fmt"

	"github.com/hrygo/divinesense/store"
)

func (db *DB) UpsertMemoTag(ctx context.Context, upsert *store.UpsertMemoTag) (*store.MemoTag, error) {
	query := `
		INSERT INTO memo_tags (memo_id, tag, confidence, source)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (memo_id, tag) DO UPDATE SET
			confidence = EXCLUDED.confidence,
			source = EXCLUDED.source
		RETURNING id, memo_id, tag, confidence, source, created_ts
	`
	var tag store.MemoTag
	err := db.db.QueryRowContext(ctx, query,
		upsert.MemoID,
		upsert.Tag,
		upsert.Confidence,
		upsert.Source,
	).Scan(
		&tag.ID,
		&tag.MemoID,
		&tag.Tag,
		&tag.Confidence,
		&tag.Source,
		&tag.CreatedTs,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to upsert memo tag: %w", err)
	}
	return &tag, nil
}

func (db *DB) UpsertMemoTags(ctx context.Context, upserts []*store.UpsertMemoTag) error {
	if len(upserts) == 0 {
		return nil
	}

	tx, err := db.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO memo_tags (memo_id, tag, confidence, source)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (memo_id, tag) DO UPDATE SET
			confidence = EXCLUDED.confidence,
			source = EXCLUDED.source
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, upsert := range upserts {
		_, err := stmt.ExecContext(ctx, upsert.MemoID, upsert.Tag, upsert.Confidence, upsert.Source)
		if err != nil {
			return fmt.Errorf("failed to upsert memo tag: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	return nil
}

func (db *DB) ListMemoTags(ctx context.Context, find *store.FindMemoTag) ([]*store.MemoTag, error) {
	query := `
		SELECT id, memo_id, tag, confidence, source, created_ts
		FROM memo_tags
		WHERE 1=1
	`
	var args []interface{}
	argIndex := 1

	if find.MemoID != nil {
		query += fmt.Sprintf(" AND memo_id = $%d", argIndex)
		args = append(args, *find.MemoID)
		argIndex++
	}
	if find.Tag != nil {
		query += fmt.Sprintf(" AND tag = $%d", argIndex)
		args = append(args, *find.Tag)
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
		return nil, fmt.Errorf("failed to list memo tags: %w", err)
	}
	defer rows.Close()

	var tags []*store.MemoTag
	for rows.Next() {
		var tag store.MemoTag
		err := rows.Scan(
			&tag.ID,
			&tag.MemoID,
			&tag.Tag,
			&tag.Confidence,
			&tag.Source,
			&tag.CreatedTs,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan memo tag: %w", err)
		}
		tags = append(tags, &tag)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to list memo tags: %w", err)
	}
	return tags, nil
}

func (db *DB) DeleteMemoTag(ctx context.Context, memoID int32, tag string) error {
	query := `DELETE FROM memo_tags WHERE memo_id = $1 AND tag = $2`
	_, err := db.db.ExecContext(ctx, query, memoID, tag)
	if err != nil {
		return fmt.Errorf("failed to delete memo tag: %w", err)
	}
	return nil
}

func (db *DB) DeleteAllMemoTags(ctx context.Context, memoID int32) error {
	query := `DELETE FROM memo_tags WHERE memo_id = $1`
	_, err := db.db.ExecContext(ctx, query, memoID)
	if err != nil {
		return fmt.Errorf("failed to delete memo tags: %w", err)
	}
	return nil
}
