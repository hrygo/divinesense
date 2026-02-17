// Package context provides context building for LLM prompts.
package context

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"time"
)

// BlockStoreMessageProvider implements MessageProvider using AIBlockStore.
// This decouples frontend from history management, following the
// "Backend as Source of Truth" principle from context-engineering.md.
type BlockStoreMessageProvider struct {
	blockStore BlockStore
	userID     int32
}

// BlockStore defines the minimal block storage operations needed by context builder.
// This follows Interface Segregation Principle (ISP) - only the methods we need.
type BlockStore interface {
	// ListBlocks retrieves blocks for a conversation
	ListBlocks(ctx context.Context, conversationID int32) ([]*Block, error)

	// GetLatestBlock retrieves the most recent block for a conversation
	GetLatestBlock(ctx context.Context, conversationID int32) (*Block, error)
}

// Block represents a minimal view of AIBlock for context building.
// This decouples the context package from the full store package.
type Block struct {
	ID               int64
	ConversationID   int32
	RoundNumber      int32
	UserInputs       []UserInputView
	AssistantContent string
	AssistantTs      int64
	Metadata         map[string]any
	CreatedTs        int64
}

// UserInputView represents a user input for context building.
type UserInputView struct {
	Content   string
	Timestamp int64
}

// NewBlockStoreMessageProvider creates a new block-based message provider.
func NewBlockStoreMessageProvider(store BlockStore, userID int32) *BlockStoreMessageProvider {
	return &BlockStoreMessageProvider{
		blockStore: store,
		userID:     userID,
	}
}

// GetRecentMessages retrieves recent messages from block store.
// Implements MessageProvider interface.
func (p *BlockStoreMessageProvider) GetRecentMessages(
	ctx context.Context,
	sessionID string,
	limit int,
) ([]*Message, error) {
	// Parse sessionID to conversationID
	conversationID, err := ParseSessionID(sessionID)
	if err != nil {
		slog.Debug("failed to parse sessionID", "session_id", sessionID, "error", err)
		return nil, nil // Return empty instead of error for graceful degradation
	}

	// Query blocks from store
	blocks, err := p.blockStore.ListBlocks(ctx, conversationID)
	if err != nil {
		return nil, fmt.Errorf("failed to list blocks: %w", err)
	}

	// Convert blocks to messages
	messages := p.blocksToMessages(blocks)

	// Log for debugging context building
	slog.Debug("context.GetRecentMessages",
		"conversation_id", conversationID,
		"blocks_count", len(blocks),
		"messages_count", len(messages),
		"limit", limit)

	// Log block details for debugging
	for i, b := range blocks {
		hasAssistantContent := b.AssistantContent != ""
		slog.Debug("context.GetRecentMessages.block",
			"index", i,
			"block_id", b.ID,
			"round", b.RoundNumber,
			"has_assistant_content", hasAssistantContent,
			"assistant_content_len", len(b.AssistantContent),
			"user_inputs_count", len(b.UserInputs))
	}

	// Apply limit (most recent)
	if len(messages) > limit {
		messages = messages[len(messages)-limit:]
	}

	return messages, nil
}

// GetBlockCount returns the number of blocks for a conversation.
// Used by GetHistoryLength for accurate conversation turn counting.
func (p *BlockStoreMessageProvider) GetBlockCount(ctx context.Context, conversationID int32) (int, error) {
	blocks, err := p.blockStore.ListBlocks(ctx, conversationID)
	if err != nil {
		return 0, fmt.Errorf("failed to list blocks: %w", err)
	}
	return len(blocks), nil
}

// blocksToMessages converts Block slice to Message slice.
func (p *BlockStoreMessageProvider) blocksToMessages(blocks []*Block) []*Message {
	messages := make([]*Message, 0, len(blocks)*2) // user + assistant per block

	for _, block := range blocks {
		// Add user inputs
		for _, input := range block.UserInputs {
			messages = append(messages, &Message{
				Role:      "user",
				Content:   input.Content,
				Timestamp: time.Unix(input.Timestamp/1000, 0),
			})
		}

		// Add assistant response
		if block.AssistantContent != "" {
			messages = append(messages, &Message{
				Role:      "assistant",
				Content:   block.AssistantContent,
				Timestamp: time.Unix(block.AssistantTs/1000, 0),
			})
		}
	}

	return messages
}

// ParseSessionID extracts conversationID from sessionID.
// Supported formats:
//   - "conv_<conversationID>" (e.g., "conv_123")
//   - Direct conversationID as string (e.g., "123")
func ParseSessionID(sessionID string) (int32, error) {
	if sessionID == "" {
		return 0, fmt.Errorf("empty sessionID")
	}

	// Try "conv_<id>" format
	if len(sessionID) > 5 && sessionID[:5] == "conv_" {
		id, err := strconv.ParseInt(sessionID[5:], 10, 32)
		if err != nil {
			return 0, fmt.Errorf("invalid conv format: %s", sessionID)
		}
		return int32(id), nil
	}

	// Try direct number format
	id, err := strconv.ParseInt(sessionID, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("invalid sessionID format: %s", sessionID)
	}

	return int32(id), nil
}
