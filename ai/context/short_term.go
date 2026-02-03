// Package context provides context building for LLM prompts.
package context

import (
	"context"
	"fmt"
	"sort"
	"strings"
)

// ShortTermExtractor extracts recent conversation turns.
type ShortTermExtractor struct {
	maxTurns int
}

// NewShortTermExtractor creates a new short-term memory extractor.
func NewShortTermExtractor(maxTurns int) *ShortTermExtractor {
	if maxTurns <= 0 {
		maxTurns = 10
	}
	return &ShortTermExtractor{
		maxTurns: maxTurns,
	}
}

// MessageProvider provides recent messages for a session.
type MessageProvider interface {
	GetRecentMessages(ctx context.Context, sessionID string, limit int) ([]*Message, error)
}

// Extract extracts recent messages from the session.
func (e *ShortTermExtractor) Extract(ctx context.Context, provider MessageProvider, sessionID string) ([]*Message, error) {
	if provider == nil {
		return nil, nil
	}

	messages, err := provider.GetRecentMessages(ctx, sessionID, e.maxTurns)
	if err != nil {
		return nil, err
	}

	// Sort by timestamp ascending (oldest first)
	sort.Slice(messages, func(i, j int) bool {
		return messages[i].Timestamp.Before(messages[j].Timestamp)
	})

	return messages, nil
}

// FormatConversation formats messages into conversation context.
func FormatConversation(messages []*Message) string {
	if len(messages) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("### 对话历史\n")

	for _, msg := range messages {
		switch msg.Role {
		case "user":
			sb.WriteString(fmt.Sprintf("用户: %s\n", msg.Content))
		case "assistant":
			sb.WriteString(fmt.Sprintf("助手: %s\n", msg.Content))
		default:
			sb.WriteString(fmt.Sprintf("%s: %s\n", msg.Role, msg.Content))
		}
	}

	return sb.String()
}

// SplitByRecency splits messages into recent (high priority) and older (lower priority).
func SplitByRecency(messages []*Message, recentCount int) (recent, older []*Message) {
	if len(messages) <= recentCount {
		return messages, nil
	}

	splitIdx := len(messages) - recentCount
	return messages[splitIdx:], messages[:splitIdx]
}
