package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/hrygo/divinesense/store"
)

func (d *DB) CreateAIConversation(ctx context.Context, create *store.AIConversation) (*store.AIConversation, error) {
	// If ID is specified, use it (for fixed conversations)
	// Otherwise, let the database generate it
	var fields []string
	var args []any

	if create.ID != 0 {
		fields = []string{"id", "uid", "creator_id", "title", "parrot_id", "pinned", "created_ts", "updated_ts"}
		args = []any{create.ID, create.UID, create.CreatorID, create.Title, create.ParrotID, create.Pinned, create.CreatedTs, create.UpdatedTs}
		stmt := `INSERT INTO ai_conversation (` + strings.Join(fields, ", ") + `)
			VALUES (` + placeholders(len(args)) + `)`
		if _, err := d.db.ExecContext(ctx, stmt, args...); err != nil {
			return nil, fmt.Errorf("failed to create ai_conversation with fixed id: %w", err)
		}
	} else {
		fields = []string{"uid", "creator_id", "title", "parrot_id", "pinned", "created_ts", "updated_ts"}
		args = []any{create.UID, create.CreatorID, create.Title, create.ParrotID, create.Pinned, create.CreatedTs, create.UpdatedTs}
		stmt := `INSERT INTO ai_conversation (` + strings.Join(fields, ", ") + `)
			VALUES (` + placeholders(len(args)) + `)
			RETURNING id`
		if err := d.db.QueryRowContext(ctx, stmt, args...).Scan(&create.ID); err != nil {
			return nil, fmt.Errorf("failed to create ai_conversation: %w", err)
		}
	}

	return create, nil
}

func (d *DB) ListAIConversations(ctx context.Context, find *store.FindAIConversation) ([]*store.AIConversation, error) {
	where, args := []string{"1 = 1"}, []any{}

	if find.ID != nil {
		where, args = append(where, "id = "+placeholder(len(args)+1)), append(args, *find.ID)
	}
	if find.UID != nil {
		where, args = append(where, "uid = "+placeholder(len(args)+1)), append(args, *find.UID)
	}
	if find.CreatorID != nil {
		where, args = append(where, "creator_id = "+placeholder(len(args)+1)), append(args, *find.CreatorID)
	}
	if find.Pinned != nil {
		where, args = append(where, "pinned = "+placeholder(len(args)+1)), append(args, *find.Pinned)
	}

	query := `SELECT id, uid, creator_id, title, parrot_id, pinned, created_ts, updated_ts FROM ai_conversation WHERE ` + strings.Join(where, " AND ") + ` ORDER BY updated_ts DESC`
	rows, err := d.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list ai_conversations: %w", err)
	}
	defer rows.Close()

	list := make([]*store.AIConversation, 0)
	for rows.Next() {
		c := &store.AIConversation{}
		if err := rows.Scan(&c.ID, &c.UID, &c.CreatorID, &c.Title, &c.ParrotID, &c.Pinned, &c.CreatedTs, &c.UpdatedTs); err != nil {
			return nil, fmt.Errorf("failed to scan ai_conversation: %w", err)
		}
		list = append(list, c)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate ai_conversations: %w", err)
	}

	return list, nil
}

func (d *DB) UpdateAIConversation(ctx context.Context, update *store.UpdateAIConversation) (*store.AIConversation, error) {
	set, args := []string{}, []any{}

	if update.Title != nil {
		set, args = append(set, "title = "+placeholder(len(args)+1)), append(args, *update.Title)
	}
	if update.ParrotID != nil {
		set, args = append(set, "parrot_id = "+placeholder(len(args)+1)), append(args, *update.ParrotID)
	}
	if update.Pinned != nil {
		set, args = append(set, "pinned = "+placeholder(len(args)+1)), append(args, *update.Pinned)
	}
	if update.UpdatedTs != nil {
		set, args = append(set, "updated_ts = "+placeholder(len(args)+1)), append(args, *update.UpdatedTs)
	}

	if len(set) == 0 {
		return nil, fmt.Errorf("no fields to update")
	}

	args = append(args, update.ID)
	// RETURNING all fields to avoid N+1 query
	stmt := `UPDATE ai_conversation SET ` + strings.Join(set, ", ") + ` WHERE id = ` + placeholder(len(args)) + ` RETURNING id, uid, creator_id, title, parrot_id, pinned, created_ts, updated_ts`
	result := &store.AIConversation{}
	err := d.db.QueryRowContext(ctx, stmt, args...).Scan(
		&result.ID, &result.UID, &result.CreatorID, &result.Title, &result.ParrotID, &result.Pinned, &result.CreatedTs, &result.UpdatedTs,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("ai_conversation not found")
		}
		return nil, fmt.Errorf("failed to update ai_conversation: %w", err)
	}

	return result, nil
}

func (d *DB) DeleteAIConversation(ctx context.Context, delete *store.DeleteAIConversation) error {
	// Note: ai_block has CASCADE delete automatically
	result, err := d.db.ExecContext(ctx, `DELETE FROM ai_conversation WHERE id = `+placeholder(1), delete.ID)
	if err != nil {
		return fmt.Errorf("failed to delete ai_conversation: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("ai_conversation not found")
	}

	return nil
}

// ai_message functions removed: ALL IN Block!
// Message persistence is now handled by BlockManager in the main chat flow.
// - CreateAIMessage (removed)
// - ListAIMessages (removed)
// - DeleteAIMessage (removed)
