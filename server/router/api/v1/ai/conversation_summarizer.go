package ai

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hrygo/divinesense/ai"
	"github.com/hrygo/divinesense/store"
	"github.com/lithammer/shortuuid/v4"
	"log/slog"
)

// MessageReader defines the interface for reading messages from storage.
type MessageReader interface {
	ListAIMessages(ctx context.Context, find *store.FindAIMessage) ([]*store.AIMessage, error)
}

// MessageWriter defines the interface for writing messages to storage.
type MessageWriter interface {
	CreateAIMessage(ctx context.Context, create *store.AIMessage) (*store.AIMessage, error)
}

// ConversationSummarizer handles automatic conversation summarization.
// When a conversation exceeds the message threshold, it generates a summary
// and stores SEPARATOR + SUMMARY messages to optimize context for future LLM calls.
type ConversationSummarizer struct {
	reader           MessageReader
	writer           MessageWriter
	llm              ai.LLMService
	messageThreshold int // Trigger summarization after this many MESSAGE types
}

// NewConversationSummarizer creates a new conversation summarizer.
func NewConversationSummarizer(reader MessageReader, writer MessageWriter, llm ai.LLMService, threshold int) *ConversationSummarizer {
	if threshold <= 0 {
		threshold = 11 // Default threshold
	}
	return &ConversationSummarizer{
		reader:           reader,
		writer:           writer,
		llm:              llm,
		messageThreshold: threshold,
	}
}

// NewConversationSummarizerWithStore creates a summarizer with a single store for both read and write.
// The store must implement both MessageReader and MessageWriter.
func NewConversationSummarizerWithStore(store interface {
	MessageReader
	MessageWriter
}, llm ai.LLMService, threshold int) *ConversationSummarizer {
	if threshold <= 0 {
		threshold = 11
	}
	return &ConversationSummarizer{
		reader:           store,
		writer:           store,
		llm:              llm,
		messageThreshold: threshold,
	}
}

// ShouldSummarize checks if a conversation needs summarization.
// Returns (shouldSummarize, messageCountAfterLastSeparator).
func (s *ConversationSummarizer) ShouldSummarize(ctx context.Context, conversationID int32) (bool, int) {
	messages, err := s.reader.ListAIMessages(ctx, &store.FindAIMessage{
		ConversationID: &conversationID,
	})
	if err != nil {
		return false, 0
	}

	// Count MESSAGE types after the last SEPARATOR
	messageCount := 0
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Type == store.AIMessageTypeSeparator {
			break
		}
		if messages[i].Type == store.AIMessageTypeMessage {
			messageCount++
		}
	}

	return messageCount >= s.messageThreshold, messageCount
}

// Summarize generates a summary and stores SEPARATOR + SUMMARY messages.
// The SEPARATOR marks the context cutoff point, and SUMMARY stores the summary content
// that will be used as a prefix in future LLM context building.
func (s *ConversationSummarizer) Summarize(ctx context.Context, conversationID int32) error {
	// 1. Load all messages from the conversation
	messages, err := s.reader.ListAIMessages(ctx, &store.FindAIMessage{
		ConversationID: &conversationID,
	})
	if err != nil {
		return fmt.Errorf("failed to load messages: %w", err)
	}

	// 2. Get MESSAGE types after the last SEPARATOR
	messagesToSummarize := s.getMessagesAfterLastSeparator(messages)
	if len(messagesToSummarize) == 0 {
		return nil
	}

	slog.Default().Info("Triggering conversation summarization",
		"conversation_id", conversationID,
		"message_count", len(messagesToSummarize),
	)

	// 3. Generate summary content using LLM
	summary, err := s.generateSummary(ctx, messagesToSummarize)
	if err != nil {
		return fmt.Errorf("failed to generate summary: %w", err)
	}

	// 4. Insert SEPARATOR message (marks context cutoff point)
	now := time.Now().Unix()
	_, err = s.writer.CreateAIMessage(ctx, &store.AIMessage{
		UID:            shortuuid.New(),
		ConversationID: conversationID,
		Type:           store.AIMessageTypeSeparator,
		Role:           store.AIMessageRoleSystem,
		Content:        "Context summarized",
		Metadata:       "{}",
		CreatedTs:      now,
	})
	if err != nil {
		return fmt.Errorf("failed to create separator: %w", err)
	}

	// 5. Insert SUMMARY message (stores summary for future context building)
	_, err = s.writer.CreateAIMessage(ctx, &store.AIMessage{
		UID:            shortuuid.New(),
		ConversationID: conversationID,
		Type:           store.AIMessageTypeSummary,
		Role:           store.AIMessageRoleSystem,
		Content:        summary,
		Metadata:       "{}",
		CreatedTs:      now + 1, // Ensure it comes after SEPARATOR
	})
	if err != nil {
		return fmt.Errorf("failed to create summary message: %w", err)
	}

	slog.Default().Info("Conversation summarization completed",
		"conversation_id", conversationID,
		"summary_length", len(summary),
	)

	return nil
}

// getMessagesAfterLastSeparator returns MESSAGE type messages after the last SEPARATOR.
func (s *ConversationSummarizer) getMessagesAfterLastSeparator(messages []*store.AIMessage) []*store.AIMessage {
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Type == store.AIMessageTypeSeparator {
			// Filter only MESSAGE types (exclude SUMMARY which may be after SEPARATOR)
			var result []*store.AIMessage
			for _, msg := range messages[i+1:] {
				if msg.Type == store.AIMessageTypeMessage {
					result = append(result, msg)
				}
			}
			return result
		}
	}
	// No SEPARATOR found, return all MESSAGE types
	var result []*store.AIMessage
	for _, msg := range messages {
		if msg.Type == store.AIMessageTypeMessage {
			result = append(result, msg)
		}
	}
	return result
}

// generateSummary uses LLM to generate a summary of the messages.
func (s *ConversationSummarizer) generateSummary(ctx context.Context, messages []*store.AIMessage) (string, error) {
	var sb strings.Builder
	sb.WriteString("请总结以下对话内容，提取关键信息和结论：\n\n")

	for _, msg := range messages {
		role := "用户"
		if msg.Role == store.AIMessageRoleAssistant {
			role = "助手"
		}
		// Limit each message length to avoid excessive summary input
		content := msg.Content
		if len(content) > 500 {
			content = content[:500] + "..."
		}
		sb.WriteString(fmt.Sprintf("[%s]: %s\n\n", role, content))
	}

	prompt := sb.String()

	llmMessages := []ai.Message{
		{Role: "system", Content: "你是一个专业的对话总结助手，擅长提取对话关键信息。请用简洁的语言总结对话要点。"},
		{Role: "user", Content: prompt},
	}

	summary, err := s.llm.Chat(ctx, llmMessages)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(summary), nil
}
