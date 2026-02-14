package ai

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/hrygo/divinesense/store"
)

// Message represents a conversation message for context building.
type Message struct {
	Content string
	Role    string // "user" or "assistant"
	Type    string // "MESSAGE" or "SEPARATOR"
}

// BlockStore defines the interface for loading blocks from storage.
type BlockStore interface {
	ListAIBlocks(ctx context.Context, find *store.FindAIBlock) ([]*store.AIBlock, error)
}

// TokenCounter estimates token count for a string.
type TokenCounter interface {
	CountTokens(text string) int
}

// SimpleTokenCounter provides a rough token estimation.
// Approximately 4 characters per token for English text.
type SimpleTokenCounter struct{}

func (s *SimpleTokenCounter) CountTokens(text string) int {
	// Rough estimation: ~4 characters per token
	// More accurate would be to use tiktoken, but this is sufficient for estimation
	return len(text) / 4
}

// ContextControl specifies how to build the conversation context.
type ContextControl struct {
	PendingMessages []Message
	MaxMessages     int
	MaxTokens       int
	IgnoreSeparator bool
}

// BuiltContext represents the result of context building.
type BuiltContext struct {
	Messages     []string
	MessageCount int
	TokenCount   int
	SeparatorPos int
	WasTruncated bool
	HasPending   bool
}

// ContextBuilder builds conversation context from stored blocks.
// It enforces SEPARATOR filtering and applies token limits.
//
// Deprecated: Use ai/context.Service with BlockStoreMessageProvider instead.
// The new architecture provides better separation of concerns and supports
// the full context-engineering.md design including episodic memory.
// This implementation will be removed in a future version.
//
// Architecture Note:
// This component supports a hybrid data source:
// 1. Persisted blocks from the database (authoritative)
// 2. Pending messages from EventBus (not yet persisted)
//
// This is necessary because EventBus uses async persistence, which creates
// a race condition: the next message may be sent before the previous
// message is written to the database.
type ContextBuilder struct {
	store        BlockStore
	tokenCounter TokenCounter
	maxTokens    int // Default max tokens (approx 8000 for most models)
	mu           sync.RWMutex
}

// NewContextBuilder creates a new ContextBuilder.
func NewContextBuilder(store BlockStore) *ContextBuilder {
	return &ContextBuilder{
		store:        store,
		tokenCounter: &SimpleTokenCounter{},
		maxTokens:    8000, // Default context window
	}
}

// SetMaxTokens sets the default maximum token count.
func (b *ContextBuilder) SetMaxTokens(max int) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.maxTokens = max
}

// BuildContext loads and filters conversation blocks.
// Returns messages after the last SEPARATOR, respecting token limits.
//
// The function merges persisted blocks (from DB) with pending messages
// (from EventBus) to ensure context is complete even when persistence is delayed.
func (b *ContextBuilder) BuildContext(
	ctx context.Context,
	conversationID int32,
	control *ContextControl,
) (*BuiltContext, error) {
	b.mu.RLock()
	maxTokens := b.maxTokens
	b.mu.RUnlock()

	// Apply control overrides
	if control != nil && control.MaxTokens > 0 {
		maxTokens = control.MaxTokens
	}

	// Extract pending messages from control
	var pendingMessages []Message
	if control != nil {
		pendingMessages = control.PendingMessages
	}

	// 1. Load all blocks from database
	blocks, err := b.store.ListAIBlocks(ctx, &store.FindAIBlock{
		ConversationID: &conversationID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to load blocks: %w", err)
	}

	// 2. Convert store blocks to Message slice
	allMessages := b.convertFromBlocks(blocks)

	// 3. Append pending messages (not yet in DB)
	// These are added because EventBus persistence is async
	for _, pending := range pendingMessages {
		allMessages = append(allMessages, pending)
		if len(pendingMessages) > 0 {
			slog.Default().Debug("BuildContext: included pending messages",
				"conversation_id", conversationID,
				"pending_count", len(pendingMessages),
			)
		}
	}

	// 4. Find last SEPARATOR position
	lastSeparatorIdx := b.findLastSeparator(allMessages)

	// 5. Filter messages: only MESSAGE type after SEPARATOR
	var contextMessages []Message
	if control != nil && control.IgnoreSeparator {
		// Include all messages when ignoring separator
		contextMessages = allMessages
	} else {
		// Only include messages after last separator
		if lastSeparatorIdx == -1 {
			contextMessages = allMessages
		} else if lastSeparatorIdx+1 < len(allMessages) {
			// Normal case: there are messages after the separator
			contextMessages = allMessages[lastSeparatorIdx+1:]
		} else {
			// Separator is the last message, no context after it
			contextMessages = []Message{}
		}
	}

	// 6. Convert to string array
	contents := make([]string, 0, len(contextMessages))
	for _, msg := range contextMessages {
		if msg.Type == "MESSAGE" {
			contents = append(contents, msg.Content)
		}
	}

	// 7. Apply token limit (truncate from oldest)
	truncated := b.truncateByTokens(contents, maxTokens)
	wasTruncated := len(truncated) < len(contents)

	// 8. Apply message count limit
	if control != nil && control.MaxMessages > 0 {
		truncated = b.truncateByCount(truncated, control.MaxMessages)
		wasTruncated = wasTruncated || len(truncated) < len(contents)
	}

	// 9. Calculate token count
	tokenCount := 0
	for _, msg := range truncated {
		tokenCount += b.tokenCounter.CountTokens(msg)
	}

	result := &BuiltContext{
		Messages:     truncated,
		MessageCount: len(truncated),
		TokenCount:   tokenCount,
		WasTruncated: wasTruncated,
		SeparatorPos: lastSeparatorIdx,
		HasPending:   len(pendingMessages) > 0,
	}

	// Log for debugging
	if wasTruncated {
		slog.Default().Debug("Context truncated",
			"conversation_id", conversationID,
			"original_count", len(contents),
			"truncated_count", len(truncated),
			"token_count", tokenCount,
			"separator_pos", lastSeparatorIdx,
		)
	}

	return result, nil
}

// convertFromBlocks converts store.AIBlock slices to Message slices.
// Each block contains user inputs and assistant content as separate messages.
func (b *ContextBuilder) convertFromBlocks(blocks []*store.AIBlock) []Message {
	result := make([]Message, 0, len(blocks)*2) // Estimate capacity

	for _, block := range blocks {
		// Add user inputs as user messages
		for _, input := range block.UserInputs {
			result = append(result, Message{
				Content: input.Content,
				Role:    "user",
				Type:    "MESSAGE",
			})
		}

		// Add assistant content as assistant message
		if block.AssistantContent != "" {
			result = append(result, Message{
				Content: block.AssistantContent,
				Role:    "assistant",
				Type:    "MESSAGE",
			})
		}
	}

	return result
}

// findLastSeparator finds the index of the last SEPARATOR message.
func (b *ContextBuilder) findLastSeparator(messages []Message) int {
	lastSeparatorIdx := -1
	for i, msg := range messages {
		if msg.Type == "SEPARATOR" {
			lastSeparatorIdx = i
		}
	}
	return lastSeparatorIdx
}

// truncateByTokens truncates messages to fit within maxTokens.
// Removes oldest messages first (from the beginning).
func (b *ContextBuilder) truncateByTokens(messages []string, maxTokens int) []string {
	if maxTokens <= 0 {
		return messages
	}

	// Count from newest (end) to oldest (beginning)
	totalTokens := 0
	for i := len(messages) - 1; i >= 0; i-- {
		totalTokens += b.tokenCounter.CountTokens(messages[i])
		if totalTokens > maxTokens {
			// Return messages after this point
			if i+1 < len(messages) {
				return messages[i+1:]
			}
			return []string{}
		}
	}

	return messages
}

// truncateByCount limits the number of messages.
// Keeps newest messages (from the end).
func (b *ContextBuilder) truncateByCount(messages []string, maxCount int) []string {
	if maxCount <= 0 || len(messages) <= maxCount {
		return messages
	}

	// Return the last maxCount messages
	start := len(messages) - maxCount
	return messages[start:]
}
