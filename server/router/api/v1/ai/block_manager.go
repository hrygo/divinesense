package ai

import (
	"context"
	"log/slog"
	"time"

	"github.com/hrygo/divinesense/store"
	"github.com/lithammer/shortuuid/v4"
)

// BlockManager manages the lifecycle of conversation blocks.
//
// It handles creating blocks for new chat rounds, appending events during
// streaming, and updating block status upon completion.
type BlockManager struct {
	store *store.Store
}

// NewBlockManager creates a new BlockManager.
func NewBlockManager(store *store.Store) *BlockManager {
	return &BlockManager{store: store}
}

// CreateBlockForChat creates a new block for a chat round.
//
// This should be called when starting a new chat round (user sends message).
func (m *BlockManager) CreateBlockForChat(
	ctx context.Context,
	conversationID int32,
	userMessage string,
	agentType AgentType,
	mode BlockMode,
) (*store.AIBlock, error) {
	now := time.Now().Unix()

	// Determine block type from mode
	var blockType store.AIBlockType
	switch mode {
	case BlockModeEvolution:
		blockType = store.AIBlockTypeMessage
	case BlockModeGeek:
		blockType = store.AIBlockTypeMessage
	default:
		blockType = store.AIBlockTypeMessage
	}

	// Convert mode to store type
	storeMode := convertBlockModeToStore(mode)

	block, err := m.store.CreateAIBlockWithRound(ctx, &store.CreateAIBlock{
		UID:            shortuuid.New(),
		ConversationID: conversationID,
		BlockType:      blockType,
		Mode:           storeMode,
		UserInputs: []store.UserInput{
			{
				Content:   userMessage,
				Timestamp: now,
			},
		},
		Status:    store.AIBlockStatusPending,
		CreatedTs: now,
		UpdatedTs: now,
	})
	if err != nil {
		slog.Error("Failed to create block",
			"conversation_id", conversationID,
			"error", err,
		)
		return nil, err
	}

	slog.Info("Created block for chat",
		"block_id", block.ID,
		"conversation_id", conversationID,
		"round_number", block.RoundNumber,
	)

	return block, nil
}

// AppendEvent appends an event to the block's event stream.
//
// This should be called during streaming to record thinking, tool_use, tool_result, and answer events.
func (m *BlockManager) AppendEvent(
	ctx context.Context,
	blockID int64,
	eventType string,
	content string,
	metadata map[string]any,
) error {
	event := store.BlockEvent{
		Type:      eventType,
		Content:   content,
		Timestamp: time.Now().Unix(),
		Meta:      metadata,
	}

	if err := m.store.AppendEvent(ctx, blockID, event); err != nil {
		slog.Error("Failed to append event",
			"block_id", blockID,
			"event_type", eventType,
			"error", err,
		)
		return err
	}

	slog.Debug("Appended event to block",
		"block_id", blockID,
		"event_type", eventType,
	)

	return nil
}

// AppendUserInput appends an additional user input to an existing block.
//
// This is used when the user provides follow-up input during streaming.
func (m *BlockManager) AppendUserInput(
	ctx context.Context,
	blockID int64,
	userInput string,
) error {
	input := store.UserInput{
		Content:   userInput,
		Timestamp: time.Now().Unix(),
	}

	if err := m.store.AppendUserInput(ctx, blockID, input); err != nil {
		slog.Error("Failed to append user input",
			"block_id", blockID,
			"error", err,
		)
		return err
	}

	slog.Debug("Appended user input to block",
		"block_id", blockID,
	)

	return nil
}

// UpdateBlockStatus updates the status of a block.
//
// This should be called when streaming completes or fails.
func (m *BlockManager) UpdateBlockStatus(
	ctx context.Context,
	blockID int64,
	status store.AIBlockStatus,
	assistantContent string,
	sessionStats *store.SessionStats,
) error {
	now := time.Now().Unix()
	update := &store.UpdateAIBlock{
		ID:               blockID,
		Status:           &status,
		AssistantContent: &assistantContent,
		UpdatedTs:        &now,
	}

	if sessionStats != nil {
		update.SessionStats = sessionStats
	}

	block, err := m.store.UpdateAIBlock(ctx, update)
	if err != nil {
		slog.Error("Failed to update block status",
			"block_id", blockID,
			"status", status,
			"error", err,
		)
		return err
	}

	slog.Info("Updated block status",
		"block_id", blockID,
		"status", status,
		"round_number", block.RoundNumber,
	)

	return nil
}

// CompleteBlock marks a block as completed with the final assistant content.
func (m *BlockManager) CompleteBlock(
	ctx context.Context,
	blockID int64,
	assistantContent string,
	sessionStats *store.SessionStats,
) error {
	return m.UpdateBlockStatus(ctx, blockID, store.AIBlockStatusCompleted, assistantContent, sessionStats)
}

// MarkBlockError marks a block as failed with error status.
func (m *BlockManager) MarkBlockError(
	ctx context.Context,
	blockID int64,
	errorMessage string,
) error {
	return m.UpdateBlockStatus(ctx, blockID, store.AIBlockStatusError, errorMessage, nil)
}

// GetLatestBlock retrieves the most recent block for a conversation.
func (m *BlockManager) GetLatestBlock(
	ctx context.Context,
	conversationID int32,
) (*store.AIBlock, error) {
	return m.store.GetLatestAIBlock(ctx, conversationID)
}

// ============================================================================
// Helper Functions
// ============================================================================

// BlockMode represents the AI mode for a block.
type BlockMode string

const (
	BlockModeNormal    BlockMode = "normal"
	BlockModeGeek      BlockMode = "geek"
	BlockModeEvolution BlockMode = "evolution"
)

func convertBlockModeToStore(mode BlockMode) store.AIBlockMode {
	switch mode {
	case BlockModeNormal:
		return store.AIBlockModeNormal
	case BlockModeGeek:
		return store.AIBlockModeGeek
	case BlockModeEvolution:
		return store.AIBlockModeEvolution
	default:
		return store.AIBlockModeNormal
	}
}
