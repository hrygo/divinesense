package sqlite

import (
	"context"
	"database/sql"

	"github.com/hrygo/divinesense/store"
	"github.com/pkg/errors"
)

func (d *DB) UpsertMemoTag(ctx context.Context, upsert *store.UpsertMemoTag) (*store.MemoTag, error) {
	stmt := `
		INSERT INTO memo_tags (memo_id, tag, confidence, source, created_ts)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT (memo_id, tag) DO UPDATE SET
			confidence = excluded.confidence,
			source = excluded.source
		RETURNING id, memo_id, tag, confidence, source, created_ts
	`
	var tag store.MemoTag
	err := d.db.QueryRowContext(ctx, stmt,
		upsert.MemoID,
		upsert.Tag,
		upsert.Confidence,
		upsert.Source,
		0, // created_ts will be set by default
	).Scan(
		&tag.ID,
		&tag.MemoID,
		&tag.Tag,
		&tag.Confidence,
		&tag.Source,
		&tag.CreatedTs,
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to upsert memo tag")
	}
	return &tag, nil
}

func (d *DB) UpsertMemoTags(ctx context.Context, upserts []*store.UpsertMemoTag) error {
	if len(upserts) == 0 {
		return nil
	}

	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "failed to begin transaction")
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO memo_tags (memo_id, tag, confidence, source, created_ts)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT (memo_id, tag) DO UPDATE SET
			confidence = excluded.confidence,
			source = excluded.source
	`)
	if err != nil {
		return errors.Wrap(err, "failed to prepare statement")
	}
	defer stmt.Close()

	for _, upsert := range upserts {
		_, err := stmt.ExecContext(ctx, upsert.MemoID, upsert.Tag, upsert.Confidence, upsert.Source, 0)
		if err != nil {
			return errors.Wrap(err, "failed to upsert memo tag")
		}
	}

	if err := tx.Commit(); err != nil {
		return errors.Wrap(err, "failed to commit transaction")
	}
	return nil
}

func (d *DB) ListMemoTags(ctx context.Context, find *store.FindMemoTag) ([]*store.MemoTag, error) {
	where, args := []string{"1 = 1"}, []any{}

	if find.MemoID != nil {
		where, args = append(where, "memo_id = ?"), append(args, *find.MemoID)
	}
	if find.Tag != nil {
		where, args = append(where, "tag = ?"), append(args, *find.Tag)
	}

	query := `SELECT id, memo_id, tag, confidence, source, created_ts
		FROM memo_tags
		WHERE ` + where[0]
	if len(where) > 1 {
		query += " AND " + where[1]
	}
	query += " ORDER BY confidence DESC, created_ts DESC"

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
		return nil, errors.Wrap(err, "failed to list memo tags")
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
			return nil, errors.Wrap(err, "failed to scan memo tag")
		}
		tags = append(tags, &tag)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return tags, nil
}

func (d *DB) DeleteMemoTag(ctx context.Context, memoID int32, tag string) error {
	stmt := `DELETE FROM memo_tags WHERE memo_id = ? AND tag = ?`
	result, err := d.db.ExecContext(ctx, stmt, memoID, tag)
	if err != nil {
		return errors.Wrap(err, "failed to delete memo tag")
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (d *DB) DeleteAllMemoTags(ctx context.Context, memoID int32) error {
	stmt := `DELETE FROM memo_tags WHERE memo_id = ?`
	_, err := d.db.ExecContext(ctx, stmt, memoID)
	if err != nil {
		return errors.Wrap(err, "failed to delete memo tags")
	}
	return nil
}
