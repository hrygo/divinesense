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

// BlockReader defines the interface for reading blocks from storage.
type BlockReader interface {
	ListAIBlocks(ctx context.Context, find *store.FindAIBlock) ([]*store.AIBlock, error)
}

// BlockWriter defines the interface for writing blocks to storage.
type BlockWriter interface {
	CreateAIBlock(ctx context.Context, create *store.CreateAIBlock) (*store.AIBlock, error)
}

// ConversationSummarizer handles automatic conversation summarization.
// When a conversation exceeds the message threshold, it generates a summary
// and stores SEPARATOR block to optimize context for future LLM calls.
//
// NOTE: ALL IN Block! - Now uses AIBlock instead of AIMessage.
type ConversationSummarizer struct {
	reader           BlockReader
	writer           BlockWriter
	llm              ai.LLMService
	messageThreshold int // Trigger summarization after this many blocks
}

// NewConversationSummarizer creates a new conversation summarizer.
func NewConversationSummarizer(reader BlockReader, writer BlockWriter, llm ai.LLMService, threshold int) *ConversationSummarizer {
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
// The store must implement both BlockReader and BlockWriter.
func NewConversationSummarizerWithStore(store interface {
	BlockReader
	BlockWriter
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
// Returns (shouldSummarize, blockCountAfterLastSeparator).
func (s *ConversationSummarizer) ShouldSummarize(ctx context.Context, conversationID int32) (bool, int) {
	blocks, err := s.reader.ListAIBlocks(ctx, &store.FindAIBlock{
		ConversationID: &conversationID,
	})
	if err != nil {
		return false, 0
	}

	// Count MESSAGE type blocks after the last SEPARATOR
	blockCount := 0
	for i := len(blocks) - 1; i >= 0; i-- {
		if blocks[i].BlockType == store.AIBlockTypeContextSeparator {
			break
		}
		if blocks[i].BlockType == store.AIBlockTypeMessage {
			blockCount++
		}
	}

	return blockCount >= s.messageThreshold, blockCount
}

// Summarize generates a summary and stores a SEPARATOR block.
// The SEPARATOR marks the context cutoff point.
func (s *ConversationSummarizer) Summarize(ctx context.Context, conversationID int32) error {
	// 1. Load all blocks from the conversation
	blocks, err := s.reader.ListAIBlocks(ctx, &store.FindAIBlock{
		ConversationID: &conversationID,
	})
	if err != nil {
		return fmt.Errorf("failed to load blocks: %w", err)
	}

	// 2. Get MESSAGE type blocks after the last SEPARATOR
	blocksToSummarize := s.getBlocksAfterLastSeparator(blocks)
	if len(blocksToSummarize) == 0 {
		return nil
	}

	slog.Default().Info("Triggering conversation summarization",
		"conversation_id", conversationID,
		"block_count", len(blocksToSummarize),
	)

	// 3. Generate summary content using LLM
	summary, err := s.generateSummary(ctx, blocksToSummarize)
	if err != nil {
		return fmt.Errorf("failed to generate summary: %w", err)
	}

	// 4. Insert SEPARATOR block (marks context cutoff point)
	now := time.Now().UnixMilli()
	_, err = s.writer.CreateAIBlock(ctx, &store.CreateAIBlock{
		UID:            shortuuid.New(),
		ConversationID: conversationID,
		BlockType:      store.AIBlockTypeContextSeparator,
		Mode:           store.AIBlockModeNormal,
		UserInputs:     []store.UserInput{},
		// Store summary in metadata for reference
		Metadata: map[string]any{
			"summary":    summary,
			"summary_at": now,
		},
		CreatedTs: now,
		UpdatedTs: now,
	})
	if err != nil {
		return fmt.Errorf("failed to create separator block: %w", err)
	}

	slog.Default().Info("Conversation summarization completed",
		"conversation_id", conversationID,
		"summary_length", len(summary),
	)

	return nil
}

// getBlocksAfterLastSeparator returns MESSAGE type blocks after the last SEPARATOR.
func (s *ConversationSummarizer) getBlocksAfterLastSeparator(blocks []*store.AIBlock) []*store.AIBlock {
	for i := len(blocks) - 1; i >= 0; i-- {
		if blocks[i].BlockType == store.AIBlockTypeContextSeparator {
			// Filter only MESSAGE types
			var result []*store.AIBlock
			for _, block := range blocks[i+1:] {
				if block.BlockType == store.AIBlockTypeMessage {
					result = append(result, block)
				}
			}
			return result
		}
	}
	// No SEPARATOR found, return all MESSAGE type blocks
	var result []*store.AIBlock
	for _, block := range blocks {
		if block.BlockType == store.AIBlockTypeMessage {
			result = append(result, block)
		}
	}
	return result
}

// generateSummary uses LLM to generate a summary of the blocks.
func (s *ConversationSummarizer) generateSummary(ctx context.Context, blocks []*store.AIBlock) (string, error) {
	var sb strings.Builder
	sb.WriteString("请总结以下对话内容，提取关键信息和结论：\n\n")

	for _, block := range blocks {
		// Add user inputs
		for _, input := range block.UserInputs {
			content := input.Content
			if len(content) > 500 {
				content = content[:500] + "..."
			}
			sb.WriteString(fmt.Sprintf("[用户]: %s\n\n", content))
		}

		// Add assistant content
		if block.AssistantContent != "" {
			content := block.AssistantContent
			if len(content) > 500 {
				content = content[:500] + "..."
			}
			sb.WriteString(fmt.Sprintf("[助手]: %s\n\n", content))
		}
	}

	prompt := sb.String()

	llmMessages := []ai.Message{
		{Role: "system", Content: "你是一个专业的对话总结助手，擅长提取对话关键信息。请用简洁的语言总结对话要点。"},
		{Role: "user", Content: prompt},
	}

	summary, stats, err := s.llm.Chat(ctx, llmMessages)
	if err != nil {
		return "", err
	}
	_ = stats // Stats not needed for conversation summary

	return strings.TrimSpace(summary), nil
}
