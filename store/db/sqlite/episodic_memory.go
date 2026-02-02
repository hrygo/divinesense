package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/hrygo/divinesense/store"
)

// Episodic memory - FULL SUPPORT ON SQLITE

func (d *DB) CreateEpisodicMemory(ctx context.Context, create *store.EpisodicMemory) (*store.EpisodicMemory, error) {
	if create.Timestamp.IsZero() {
		create.Timestamp = time.Now()
	}
	if create.CreatedTs == 0 {
		create.CreatedTs = time.Now().Unix()
	}

	fields := []string{"user_id", "timestamp", "agent_type", "user_input", "outcome", "summary", "importance", "created_ts"}
	args := []any{
		create.UserID,
		create.Timestamp.Unix(),
		create.AgentType,
		create.UserInput,
		create.Outcome,
		create.Summary,
		create.Importance,
		create.CreatedTs,
	}

	stmt := `INSERT INTO episodic_memory (` + strings.Join(fields, ", ") + `)
		VALUES (` + strings.Repeat("?,", len(args)-1) + `?)
		RETURNING id`

	if err := d.db.QueryRowContext(ctx, stmt, args...).Scan(&create.ID); err != nil {
		return nil, fmt.Errorf("failed to create episodic_memory: %w", err)
	}

	return create, nil
}

func (d *DB) ListEpisodicMemories(ctx context.Context, find *store.FindEpisodicMemory) ([]*store.EpisodicMemory, error) {
	if find == nil {
		return nil, fmt.Errorf("find parameter cannot be nil")
	}

	where, args := []string{"1 = 1"}, []any{}

	if find.ID != nil {
		where, args = append(where, "id = ?"), append(args, *find.ID)
	}
	if find.UserID != nil {
		where, args = append(where, "user_id = ?"), append(args, *find.UserID)
	}
	if find.AgentType != nil {
		where, args = append(where, "agent_type = ?"), append(args, *find.AgentType)
	}
	if find.Query != nil && *find.Query != "" {
		// Simple text search in user_input and summary
		searchPattern := "%" + *find.Query + "%"
		where = append(where, "(user_input LIKE ? OR summary LIKE ?)")
		args = append(args, searchPattern, searchPattern)
	}

	query := `SELECT id, user_id, timestamp, agent_type, user_input, outcome, summary, importance, created_ts
		FROM episodic_memory
		WHERE ` + strings.Join(where, " AND ") + `
		ORDER BY timestamp DESC`

	// Validate and apply pagination (Issue #7 fix)
	limit := find.Limit
	if limit > 0 {
		if limit > 1000 {
			limit = 1000 // Cap to prevent excessive data retrieval
		}
		query += fmt.Sprintf(" LIMIT %d", limit)
	}
	if find.Offset > 0 {
		query += fmt.Sprintf(" OFFSET %d", find.Offset)
	}

	rows, err := d.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list episodic_memories: %w", err)
	}
	defer rows.Close()

	list := make([]*store.EpisodicMemory, 0)
	for rows.Next() {
		m := &store.EpisodicMemory{}
		var timestamp int64
		if err := rows.Scan(
			&m.ID,
			&m.UserID,
			&timestamp,
			&m.AgentType,
			&m.UserInput,
			&m.Outcome,
			&m.Summary,
			&m.Importance,
			&m.CreatedTs,
		); err != nil {
			return nil, fmt.Errorf("failed to scan episodic_memory: %w", err)
		}
		m.Timestamp = time.Unix(timestamp, 0)
		list = append(list, m)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate episodic_memories: %w", err)
	}

	return list, nil
}

func (d *DB) ListActiveUserIDs(ctx context.Context, cutoff time.Time) ([]int32, error) {
	query := `SELECT DISTINCT user_id FROM episodic_memory WHERE timestamp > ?`

	rows, err := d.db.QueryContext(ctx, query, cutoff.Unix())
	if err != nil {
		return nil, fmt.Errorf("failed to list active user IDs: %w", err)
	}
	defer rows.Close()

	var userIDs []int32
	for rows.Next() {
		var userID int32
		if err := rows.Scan(&userID); err != nil {
			return nil, fmt.Errorf("failed to scan user ID: %w", err)
		}
		userIDs = append(userIDs, userID)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate user IDs: %w", err)
	}

	return userIDs, nil
}

func (d *DB) DeleteEpisodicMemory(ctx context.Context, delete *store.DeleteEpisodicMemory) error {
	if delete == nil {
		return fmt.Errorf("delete parameter cannot be nil")
	}

	where, args := []string{}, []any{}

	if delete.ID != nil {
		where, args = append(where, "id = ?"), append(args, *delete.ID)
	}
	if delete.UserID != nil {
		where, args = append(where, "user_id = ?"), append(args, *delete.UserID)
	}

	if len(where) == 0 {
		return fmt.Errorf("no condition to delete episodic_memory")
	}

	stmt := `DELETE FROM episodic_memory WHERE ` + strings.Join(where, " AND ")
	if _, err := d.db.ExecContext(ctx, stmt, args...); err != nil {
		return fmt.Errorf("failed to delete episodic_memory: %w", err)
	}

	return nil
}

// UserPreferences - FULL SUPPORT ON SQLITE (TEXT instead of JSONB)

func (d *DB) UpsertUserPreferences(ctx context.Context, upsert *store.UpsertUserPreferences) (*store.UserPreferences, error) {
	now := time.Now().Unix()

	stmt := `INSERT INTO user_preferences (user_id, preferences, created_ts, updated_ts)
		VALUES (?, ?, ?, ?)
		ON CONFLICT (user_id) DO UPDATE SET
			preferences = excluded.preferences,
			updated_ts = excluded.updated_ts
		RETURNING user_id, preferences, created_ts, updated_ts`

	result := &store.UserPreferences{}
	err := d.db.QueryRowContext(ctx, stmt, upsert.UserID, upsert.Preferences, now, now).Scan(
		&result.UserID,
		&result.Preferences,
		&result.CreatedTs,
		&result.UpdatedTs,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to upsert user_preferences: %w", err)
	}

	return result, nil
}

func (d *DB) GetUserPreferences(ctx context.Context, find *store.FindUserPreferences) (*store.UserPreferences, error) {
	if find.UserID == nil {
		return nil, fmt.Errorf("user_id is required")
	}

	query := `SELECT user_id, preferences, created_ts, updated_ts
		FROM user_preferences
		WHERE user_id = ?`

	result := &store.UserPreferences{}
	err := d.db.QueryRowContext(ctx, query, *find.UserID).Scan(
		&result.UserID,
		&result.Preferences,
		&result.CreatedTs,
		&result.UpdatedTs,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Not found, return nil without error
		}
		return nil, fmt.Errorf("failed to get user_preferences: %w", err)
	}

	return result, nil
}
