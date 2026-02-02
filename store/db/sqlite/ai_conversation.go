package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/hrygo/divinesense/store"
)

// AI conversation features - FULL SUPPORT ON SQLITE

func (d *DB) CreateAIConversation(ctx context.Context, create *store.AIConversation) (*store.AIConversation, error) {
	var fields []string
	var args []any

	if create.ID != 0 {
		fields = []string{"id", "uid", "creator_id", "title", "parrot_id", "pinned", "created_ts", "updated_ts"}
		args = []any{create.ID, create.UID, create.CreatorID, create.Title, create.ParrotID, create.Pinned, create.CreatedTs, create.UpdatedTs}
		stmt := `INSERT INTO ai_conversation (` + strings.Join(fields, ", ") + `)
			VALUES (` + strings.Repeat("?,", len(args)-1) + `?)`
		if _, err := d.db.ExecContext(ctx, stmt, args...); err != nil {
			return nil, fmt.Errorf("failed to create ai_conversation with fixed id: %w", err)
		}
	} else {
		fields = []string{"uid", "creator_id", "title", "parrot_id", "pinned", "created_ts", "updated_ts"}
		args = []any{create.UID, create.CreatorID, create.Title, create.ParrotID, create.Pinned, create.CreatedTs, create.UpdatedTs}
		stmt := `INSERT INTO ai_conversation (` + strings.Join(fields, ", ") + `)
			VALUES (` + strings.Repeat("?,", len(args)-1) + `?)
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
		where, args = append(where, "id = ?"), append(args, *find.ID)
	}
	if find.UID != nil {
		where, args = append(where, "uid = ?"), append(args, *find.UID)
	}
	if find.CreatorID != nil {
		where, args = append(where, "creator_id = ?"), append(args, *find.CreatorID)
	}
	if find.Pinned != nil {
		where, args = append(where, "pinned = ?"), append(args, *find.Pinned)
	}

	query := `SELECT id, uid, creator_id, title, parrot_id, pinned, created_ts, updated_ts
		FROM ai_conversation
		WHERE ` + strings.Join(where, " AND ") + `
		ORDER BY updated_ts DESC`

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
		set, args = append(set, "title = ?"), append(args, *update.Title)
	}
	if update.ParrotID != nil {
		set, args = append(set, "parrot_id = ?"), append(args, *update.ParrotID)
	}
	if update.Pinned != nil {
		set, args = append(set, "pinned = ?"), append(args, *update.Pinned)
	}
	if update.UpdatedTs != nil {
		set, args = append(set, "updated_ts = ?"), append(args, *update.UpdatedTs)
	}

	if len(set) == 0 {
		return nil, fmt.Errorf("no fields to update")
	}

	args = append(args, update.ID)
	stmt := `UPDATE ai_conversation SET ` + strings.Join(set, ", ") + ` WHERE id = ?`
	result, err := d.db.ExecContext(ctx, stmt, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to update ai_conversation: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return nil, fmt.Errorf("ai_conversation not found")
	}

	// Fetch updated record
	return d.GetAIConversation(ctx, update.ID)
}

func (d *DB) GetAIConversation(ctx context.Context, id int32) (*store.AIConversation, error) {
	query := `SELECT id, uid, creator_id, title, parrot_id, pinned, created_ts, updated_ts
		FROM ai_conversation WHERE id = ?`

	c := &store.AIConversation{}
	err := d.db.QueryRowContext(ctx, query, id).Scan(
		&c.ID, &c.UID, &c.CreatorID, &c.Title, &c.ParrotID, &c.Pinned, &c.CreatedTs, &c.UpdatedTs,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("ai_conversation not found")
		}
		return nil, fmt.Errorf("failed to get ai_conversation: %w", err)
	}

	return c, nil
}

func (d *DB) DeleteAIConversation(ctx context.Context, delete *store.DeleteAIConversation) error {
	// Delete messages first
	if _, err := d.db.ExecContext(ctx, `DELETE FROM ai_message WHERE conversation_id = ?`, delete.ID); err != nil {
		return fmt.Errorf("failed to delete ai_messages: %w", err)
	}
	// Delete conversation
	result, err := d.db.ExecContext(ctx, `DELETE FROM ai_conversation WHERE id = ?`, delete.ID)
	if err != nil {
		return fmt.Errorf("failed to delete ai_conversation: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("ai_conversation not found")
	}

	return nil
}

func (d *DB) CreateAIMessage(ctx context.Context, create *store.AIMessage) (*store.AIMessage, error) {
	fields := []string{"uid", "conversation_id", "type", "role", "content", "metadata", "created_ts"}
	args := []any{create.UID, create.ConversationID, string(create.Type), string(create.Role), create.Content, create.Metadata, create.CreatedTs}

	stmt := `INSERT INTO ai_message (` + strings.Join(fields, ", ") + `)
		VALUES (` + strings.Repeat("?,", len(args)-1) + `?)
		RETURNING id`

	if err := d.db.QueryRowContext(ctx, stmt, args...).Scan(&create.ID); err != nil {
		return nil, fmt.Errorf("failed to create ai_message: %w", err)
	}

	return create, nil
}

func (d *DB) ListAIMessages(ctx context.Context, find *store.FindAIMessage) ([]*store.AIMessage, error) {
	where, args := []string{"1 = 1"}, []any{}

	if find.ID != nil {
		where, args = append(where, "id = ?"), append(args, *find.ID)
	}
	if find.UID != nil {
		where, args = append(where, "uid = ?"), append(args, *find.UID)
	}
	if find.ConversationID != nil {
		where, args = append(where, "conversation_id = ?"), append(args, *find.ConversationID)
	}

	query := `SELECT id, uid, conversation_id, type, role, content, metadata, created_ts
		FROM ai_message
		WHERE ` + strings.Join(where, " AND ") + `
		ORDER BY created_ts ASC`

	rows, err := d.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list ai_messages: %w", err)
	}
	defer rows.Close()

	list := make([]*store.AIMessage, 0)
	for rows.Next() {
		m := &store.AIMessage{}
		var msgType, role string
		if err := rows.Scan(&m.ID, &m.UID, &m.ConversationID, &msgType, &role, &m.Content, &m.Metadata, &m.CreatedTs); err != nil {
			return nil, fmt.Errorf("failed to scan ai_message: %w", err)
		}
		m.Type = store.AIMessageType(msgType)
		m.Role = store.AIMessageRole(role)
		list = append(list, m)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate ai_messages: %w", err)
	}

	return list, nil
}

func (d *DB) DeleteAIMessage(ctx context.Context, delete *store.DeleteAIMessage) error {
	where, args := []string{}, []any{}

	if delete.ID != nil {
		where, args = append(where, "id = ?"), append(args, *delete.ID)
	}
	if delete.ConversationID != nil {
		where, args = append(where, "conversation_id = ?"), append(args, *delete.ConversationID)
	}

	if len(where) == 0 {
		return fmt.Errorf("no condition to delete")
	}

	stmt := `DELETE FROM ai_message WHERE ` + strings.Join(where, " AND ")
	if _, err := d.db.ExecContext(ctx, stmt, args...); err != nil {
		return fmt.Errorf("failed to delete ai_message: %w", err)
	}

	return nil
}

// ListAIConversationsBasic is a minimal implementation for UI rendering.
// Now returns actual conversations (previously returned empty list).
func (d *DB) ListAIConversationsBasic(ctx context.Context, find *store.FindAIConversation) ([]*store.AIConversation, error) {
	return d.ListAIConversations(ctx, find)
}
