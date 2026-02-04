package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/hrygo/divinesense/internal/util"
	"github.com/hrygo/divinesense/store"
)

// CreateAIBlock creates a new block
func (d *DB) CreateAIBlock(ctx context.Context, create *store.CreateAIBlock) (*store.AIBlock, error) {
	// Generate UID if not provided
	uid := create.UID
	if uid == "" {
		uid = util.GenUUID()
	}

	// Marshal JSONB fields
	userInputsJSON, err := json.Marshal(create.UserInputs)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal user_inputs: %w", err)
	}
	metadataJSON, err := json.Marshal(create.Metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal metadata: %w", err)
	}

	query := `
		INSERT INTO ai_block (
			uid, conversation_id, round_number, block_type, mode,
			user_inputs, assistant_content, assistant_timestamp,
			event_stream, session_stats, cc_session_id, status, metadata,
			created_ts, updated_ts
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		RETURNING id, created_ts, updated_ts
	`

	var block store.AIBlock
	err = d.db.QueryRowContext(ctx, query,
		uid,
		create.ConversationID,
		0, // round_number starts at 0
		string(create.BlockType),
		string(create.Mode),
		userInputsJSON,
		nil,          // assistant_content
		0,            // assistant_timestamp
		[]byte("[]"), // event_stream
		nil,          // session_stats
		create.CCSessionID,
		string(create.Status),
		metadataJSON,
		create.CreatedTs,
		create.UpdatedTs,
	).Scan(&block.ID, &block.CreatedTs, &block.UpdatedTs)

	if err != nil {
		return nil, fmt.Errorf("failed to create ai_block: %w", err)
	}

	// Set remaining fields
	block.UID = uid
	block.ConversationID = create.ConversationID
	block.RoundNumber = 0
	block.BlockType = create.BlockType
	block.Mode = create.Mode
	block.UserInputs = create.UserInputs
	block.EventStream = []store.BlockEvent{}
	block.Status = create.Status
	block.Metadata = create.Metadata

	return &block, nil
}

// GetAIBlock retrieves a block by ID
func (d *DB) GetAIBlock(ctx context.Context, id int64) (*store.AIBlock, error) {
	query := `
		SELECT id, uid, conversation_id, round_number, block_type, mode,
		       user_inputs, assistant_content, assistant_timestamp,
		       event_stream, session_stats, cc_session_id, status, metadata,
		       created_ts, updated_ts
		FROM ai_block
		WHERE id = $1
	`

	var block store.AIBlock
	var userInputsJSON, eventStreamJSON, sessionStatsJSON, metadataJSON []byte
	var assistantContent sql.NullString
	var assistantTimestamp sql.NullInt64
	var ccSessionID sql.NullString

	err := d.db.QueryRowContext(ctx, query, id).Scan(
		&block.ID,
		&block.UID,
		&block.ConversationID,
		&block.RoundNumber,
		&block.BlockType,
		&block.Mode,
		&userInputsJSON,
		&assistantContent,
		&assistantTimestamp,
		&eventStreamJSON,
		&sessionStatsJSON,
		&ccSessionID,
		&block.Status,
		&metadataJSON,
		&block.CreatedTs,
		&block.UpdatedTs,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get ai_block: %w", err)
	}

	// Unmarshal JSONB fields
	if err := json.Unmarshal(userInputsJSON, &block.UserInputs); err != nil {
		return nil, fmt.Errorf("failed to unmarshal user_inputs: %w", err)
	}
	if err := json.Unmarshal(eventStreamJSON, &block.EventStream); err != nil {
		return nil, fmt.Errorf("failed to unmarshal event_stream: %w", err)
	}
	if err := json.Unmarshal(metadataJSON, &block.Metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}
	if assistantContent.Valid {
		block.AssistantContent = assistantContent.String
	}
	if assistantTimestamp.Valid {
		block.AssistantTimestamp = assistantTimestamp.Int64
	}
	if ccSessionID.Valid {
		block.CCSessionID = ccSessionID.String
	}
	// Parse nullable session_stats JSONB
	if sessionStatsJSON != nil {
		var stats store.SessionStats
		if err := json.Unmarshal(sessionStatsJSON, &stats); err == nil {
			block.SessionStats = &stats
		}
	}

	return &block, nil
}

// ListAIBlocks retrieves blocks for a conversation
func (d *DB) ListAIBlocks(ctx context.Context, find *store.FindAIBlock) ([]*store.AIBlock, error) {
	where, args := []string{"1 = 1"}, []any{1}

	if find.ID != nil {
		where, args = append(where, "id = "+placeholder(len(args)+1)), append(args, *find.ID)
	}
	if find.UID != nil {
		where, args = append(where, "uid = "+placeholder(len(args)+1)), append(args, *find.UID)
	}
	if find.ConversationID != nil {
		where, args = append(where, "conversation_id = "+placeholder(len(args)+1)), append(args, *find.ConversationID)
	}
	if find.Status != nil {
		where, args = append(where, "status = "+placeholder(len(args)+1)), append(args, string(*find.Status))
	}
	if find.Mode != nil {
		where, args = append(where, "mode = "+placeholder(len(args)+1)), append(args, string(*find.Mode))
	}
	if find.CCSessionID != nil {
		where, args = append(where, "cc_session_id = "+placeholder(len(args)+1)), append(args, *find.CCSessionID)
	}

	query := `
		SELECT id, uid, conversation_id, round_number, block_type, mode,
		       user_inputs, assistant_content, assistant_timestamp,
		       event_stream, session_stats, cc_session_id, status, metadata,
		       created_ts, updated_ts
		FROM ai_block
		WHERE ` + strings.Join(where, " AND ") + `
		ORDER BY round_number ASC
	`

	rows, err := d.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list ai_blocks: %w", err)
	}
	defer rows.Close()

	list := make([]*store.AIBlock, 0)
	for rows.Next() {
		block, err := scanAIBlock(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan ai_block: %w", err)
		}
		list = append(list, block)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate ai_blocks: %w", err)
	}

	return list, nil
}

// UpdateAIBlock updates a block
func (d *DB) UpdateAIBlock(ctx context.Context, update *store.UpdateAIBlock) (*store.AIBlock, error) {
	set, args := []string{}, []any{}

	if update.UserInputs != nil {
		userInputsJSON, err := json.Marshal(*update.UserInputs)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal user_inputs: %w", err)
		}
		set, args = append(set, "user_inputs = "+placeholder(len(args)+1)), append(args, userInputsJSON)
	}
	if update.AssistantContent != nil {
		set, args = append(set, "assistant_content = "+placeholder(len(args)+1)), append(args, *update.AssistantContent)
	}
	if update.EventStream != nil {
		eventStreamJSON, err := json.Marshal(*update.EventStream)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal event_stream: %w", err)
		}
		set, args = append(set, "event_stream = "+placeholder(len(args)+1)), append(args, eventStreamJSON)
	}
	if update.SessionStats != nil {
		sessionStatsJSON, err := json.Marshal(update.SessionStats)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal session_stats: %w", err)
		}
		set, args = append(set, "session_stats = "+placeholder(len(args)+1)), append(args, sessionStatsJSON)
	}
	if update.CCSessionID != nil {
		set, args = append(set, "cc_session_id = "+placeholder(len(args)+1)), append(args, *update.CCSessionID)
	}
	if update.Status != nil {
		set, args = append(set, "status = "+placeholder(len(args)+1)), append(args, string(*update.Status))
	}
	if update.UpdatedTs != nil {
		set, args = append(set, "updated_ts = "+placeholder(len(args)+1)), append(args, *update.UpdatedTs)
	}

	if len(set) == 0 {
		return d.GetAIBlock(ctx, update.ID)
	}

	// Merge metadata
	set, args = append(set, "metadata = metadata || "+placeholder(len(args)+1)), append(args, update.Metadata)

	args = append(args, update.ID)

	query := `UPDATE ai_block SET ` + strings.Join(set, ", ") + ` WHERE id = ` + placeholder(len(args))

	if _, err := d.db.ExecContext(ctx, query, args...); err != nil {
		return nil, fmt.Errorf("failed to update ai_block: %w", err)
	}

	return d.GetAIBlock(ctx, update.ID)
}

// AppendUserInput appends a user input to an existing block
func (d *DB) AppendUserInput(ctx context.Context, blockID int64, input store.UserInput) error {
	query := `
		UPDATE ai_block
		SET user_inputs = user_inputs || $1::jsonb,
		    updated_ts = EXTRACT(EPOCH FROM NOW())::BIGINT
		WHERE id = $2
	`

	inputJSON, err := json.Marshal([]store.UserInput{input})
	if err != nil {
		return fmt.Errorf("failed to marshal user input: %w", err)
	}

	result, err := d.db.ExecContext(ctx, query, inputJSON, blockID)
	if err != nil {
		return fmt.Errorf("failed to append user input: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("block not found: %d", blockID)
	}

	slog.Debug("Appended user input to block",
		"block_id", blockID,
		"content_length", len(input.Content),
	)

	return nil
}

// AppendEvent appends an event to the event stream
func (d *DB) AppendEvent(ctx context.Context, blockID int64, event store.BlockEvent) error {
	query := `
		UPDATE ai_block
		SET event_stream = event_stream || $1::jsonb,
		    updated_ts = EXTRACT(EPOCH FROM NOW())::BIGINT
		WHERE id = $2
	`

	eventJSON, err := json.Marshal([]store.BlockEvent{event})
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	result, err := d.db.ExecContext(ctx, query, eventJSON, blockID)
	if err != nil {
		return fmt.Errorf("failed to append event: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("block not found: %d", blockID)
	}

	slog.Debug("Appended event to block",
		"block_id", blockID,
		"event_type", event.Type,
	)

	return nil
}

// UpdateAIBlockStatus updates the block status
func (d *DB) UpdateAIBlockStatus(ctx context.Context, blockID int64, status store.AIBlockStatus) error {
	query := `
		UPDATE ai_block
		SET status = $1,
		    updated_ts = EXTRACT(EPOCH FROM NOW())::BIGINT
		WHERE id = $2
	`

	result, err := d.db.ExecContext(ctx, query, string(status), blockID)
	if err != nil {
		return fmt.Errorf("failed to update block status: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("block not found: %d", blockID)
	}

	slog.Debug("Updated block status",
		"block_id", blockID,
		"status", status,
	)

	return nil
}

// DeleteAIBlock deletes a block
func (d *DB) DeleteAIBlock(ctx context.Context, id int64) error {
	query := `DELETE FROM ai_block WHERE id = $1`

	result, err := d.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete ai_block: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("block not found: %d", id)
	}

	return nil
}

// GetLatestAIBlock retrieves the latest block for a conversation
func (d *DB) GetLatestAIBlock(ctx context.Context, conversationID int32) (*store.AIBlock, error) {
	query := `
		SELECT id, uid, conversation_id, round_number, block_type, mode,
		       user_inputs, assistant_content, assistant_timestamp,
		       event_stream, session_stats, cc_session_id, status, metadata,
		       created_ts, updated_ts
		FROM ai_block
		WHERE conversation_id = $1
		ORDER BY round_number DESC
		LIMIT 1
	`

	rows, err := d.db.QueryContext(ctx, query, conversationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get latest ai_block: %w", err)
	}
	defer rows.Close()

	if !rows.Next() {
		return nil, nil // No block found
	}

	block, err := scanAIBlock(rows)
	if err != nil {
		return nil, fmt.Errorf("failed to scan ai_block: %w", err)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate ai_block: %w", err)
	}

	return block, nil
}

// GetPendingAIBlocks retrieves all pending/streaming blocks for cleanup
func (d *DB) GetPendingAIBlocks(ctx context.Context) ([]*store.AIBlock, error) {
	query := `
		SELECT id, uid, conversation_id, round_number, block_type, mode,
		       user_inputs, assistant_content, assistant_timestamp,
		       event_stream, session_stats, cc_session_id, status, metadata,
		       created_ts, updated_ts
		FROM ai_block
		WHERE status IN ('pending', 'streaming')
		ORDER BY created_ts ASC
	`

	rows, err := d.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get pending ai_blocks: %w", err)
	}
	defer rows.Close()

	list := make([]*store.AIBlock, 0)
	for rows.Next() {
		block, err := scanAIBlock(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan ai_block: %w", err)
		}
		list = append(list, block)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate ai_blocks: %w", err)
	}

	return list, nil
}

// scanAIBlock scans a row into an AIBlock
func scanAIBlock(rows *sql.Rows) (*store.AIBlock, error) {
	var block store.AIBlock
	var userInputsJSON, eventStreamJSON, sessionStatsJSON, metadataJSON []byte
	var assistantContent sql.NullString
	var assistantTimestamp sql.NullInt64
	var ccSessionID sql.NullString

	err := rows.Scan(
		&block.ID,
		&block.UID,
		&block.ConversationID,
		&block.RoundNumber,
		&block.BlockType,
		&block.Mode,
		&userInputsJSON,
		&assistantContent,
		&assistantTimestamp,
		&eventStreamJSON,
		&sessionStatsJSON,
		&ccSessionID,
		&block.Status,
		&metadataJSON,
		&block.CreatedTs,
		&block.UpdatedTs,
	)

	if err != nil {
		return nil, err
	}

	// Unmarshal JSONB fields
	if err := json.Unmarshal(userInputsJSON, &block.UserInputs); err != nil {
		return nil, fmt.Errorf("failed to unmarshal user_inputs: %w", err)
	}
	if err := json.Unmarshal(eventStreamJSON, &block.EventStream); err != nil {
		return nil, fmt.Errorf("failed to unmarshal event_stream: %w", err)
	}
	if err := json.Unmarshal(metadataJSON, &block.Metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}
	if assistantContent.Valid {
		block.AssistantContent = assistantContent.String
	}
	if assistantTimestamp.Valid {
		block.AssistantTimestamp = assistantTimestamp.Int64
	}
	if ccSessionID.Valid {
		block.CCSessionID = ccSessionID.String
	}
	// Parse nullable session_stats JSONB
	if sessionStatsJSON != nil {
		var stats store.SessionStats
		if err := json.Unmarshal(sessionStatsJSON, &stats); err == nil {
			block.SessionStats = &stats
		}
	}

	return &block, nil
}

// Round number calculation helper
func (d *DB) getNextRoundNumber(ctx context.Context, conversationID int32) (int32, error) {
	var roundNumber int32
	query := `SELECT COALESCE(MAX(round_number), -1) + 1 FROM ai_block WHERE conversation_id = $1`
	err := d.db.QueryRowContext(ctx, query, conversationID).Scan(&roundNumber)
	if err != nil {
		return 0, fmt.Errorf("failed to get next round number: %w", err)
	}
	return roundNumber, nil
}

// CreateAIBlockWithRound creates a block with auto-incremented round number
func (d *DB) CreateAIBlockWithRound(ctx context.Context, create *store.CreateAIBlock) (*store.AIBlock, error) {
	// Get next round number
	roundNumber, err := d.getNextRoundNumber(ctx, create.ConversationID)
	if err != nil {
		return nil, err
	}

	// Generate UID if not provided
	uid := create.UID
	if uid == "" {
		uid = util.GenUUID()
	}

	// Marshal JSONB fields
	userInputsJSON, err := json.Marshal(create.UserInputs)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal user_inputs: %w", err)
	}
	metadataJSON, err := json.Marshal(create.Metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal metadata: %w", err)
	}

	query := `
		INSERT INTO ai_block (
			uid, conversation_id, round_number, block_type, mode,
			user_inputs, assistant_content, assistant_timestamp,
			event_stream, session_stats, cc_session_id, status, metadata,
			created_ts, updated_ts
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		RETURNING id, created_ts, updated_ts
	`

	var block store.AIBlock
	err = d.db.QueryRowContext(ctx, query,
		uid,
		create.ConversationID,
		roundNumber,
		string(create.BlockType),
		string(create.Mode),
		userInputsJSON,
		nil,          // assistant_content
		0,            // assistant_timestamp
		[]byte("[]"), // event_stream
		nil,          // session_stats
		create.CCSessionID,
		string(create.Status),
		metadataJSON,
		create.CreatedTs,
		create.UpdatedTs,
	).Scan(&block.ID, &block.CreatedTs, &block.UpdatedTs)

	if err != nil {
		return nil, fmt.Errorf("failed to create ai_block: %w", err)
	}

	// Set remaining fields
	block.UID = uid
	block.ConversationID = create.ConversationID
	block.RoundNumber = roundNumber
	block.BlockType = create.BlockType
	block.Mode = create.Mode
	block.UserInputs = create.UserInputs
	block.EventStream = []store.BlockEvent{}
	block.Status = create.Status
	block.Metadata = create.Metadata

	return &block, nil
}
