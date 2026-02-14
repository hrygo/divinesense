package context

import (
	"context"

	"github.com/hrygo/divinesense/store"
)

// StoreAdapter adapts store.AIBlockStore to BlockStore interface.
// This follows the Adapter pattern, allowing the context package to
// remain decoupled from the full store package.
type StoreAdapter struct {
	store store.AIBlockStore
}

// NewStoreAdapter creates a new store adapter.
func NewStoreAdapter(s store.AIBlockStore) *StoreAdapter {
	return &StoreAdapter{store: s}
}

// ListBlocks retrieves blocks for a conversation.
func (a *StoreAdapter) ListBlocks(ctx context.Context, conversationID int32) ([]*Block, error) {
	blocks, err := a.store.ListAIBlocks(ctx, &store.FindAIBlock{
		ConversationID: &conversationID,
	})
	if err != nil {
		return nil, err
	}

	// Convert store.AIBlock to context.Block
	result := make([]*Block, 0, len(blocks))
	for _, b := range blocks {
		result = append(result, adaptBlock(b))
	}

	return result, nil
}

// GetLatestBlock retrieves the most recent block for a conversation.
func (a *StoreAdapter) GetLatestBlock(ctx context.Context, conversationID int32) (*Block, error) {
	block, err := a.store.GetLatestAIBlock(ctx, conversationID)
	if err != nil {
		return nil, err
	}
	if block == nil {
		return nil, nil
	}
	return adaptBlock(block), nil
}

// adaptBlock converts store.AIBlock to context.Block.
func adaptBlock(b *store.AIBlock) *Block {
	// Convert UserInputs
	inputs := make([]UserInputView, 0, len(b.UserInputs))
	for _, ui := range b.UserInputs {
		inputs = append(inputs, UserInputView{
			Content:   ui.Content,
			Timestamp: ui.Timestamp,
		})
	}

	return &Block{
		ID:               b.ID,
		ConversationID:   b.ConversationID,
		RoundNumber:      b.RoundNumber,
		UserInputs:       inputs,
		AssistantContent: b.AssistantContent,
		AssistantTs:      b.AssistantTimestamp,
		Metadata:         b.Metadata,
		CreatedTs:        b.CreatedTs,
	}
}

// Ensure StoreAdapter implements BlockStore interface.
var _ BlockStore = (*StoreAdapter)(nil)
