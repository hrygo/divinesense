// Package universal provides utility functions for the universal parrot system.
package universal

import (
	"context"
	"fmt"
	"time"

	"github.com/hrygo/divinesense/ai"
	"github.com/hrygo/divinesense/ai/agents"
)

const (
	// DefaultStreamChunkSize is the default chunk size for streaming answers (in runes).
	DefaultStreamChunkSize = 80
)

// streamAnswer streams the final answer by chunks.
// This simulates streaming by chunking the response.
// Uses rune-aware chunking to avoid UTF-8 truncation.
func streamAnswer(answer string, callback agent.EventCallback) {
	if callback == nil {
		return
	}

	// Split into chunks for "streaming" effect
	// Use rune count for safer UTF-8 handling
	runes := []rune(answer)
	chunkSize := DefaultStreamChunkSize // runes, not bytes
	for i := 0; i < len(runes); i += chunkSize {
		end := i + chunkSize
		if end > len(runes) {
			end = len(runes)
		}
		chunk := string(runes[i:end])
		// nolint:errcheck // streaming callback is best-effort
		callback(agent.EventTypeAnswer, chunk)
	}
}

// FindAndExecuteTool finds a tool by name in the provided list and executes it.
// This is a shared utility used by multiple executors to avoid code duplication.
// Returns the tool result or an error if the tool is not found or execution fails.
func FindAndExecuteTool(
	ctx context.Context,
	tools []agent.ToolWithSchema,
	toolName string,
	toolInput string,
) (string, error) {
	// Find tool - check for nil before calling Name()
	for _, t := range tools {
		if t != nil && t.Name() == toolName {
			return t.Run(ctx, toolInput)
		}
	}
	return "", fmt.Errorf("tool not found: %s", toolName)
}

// BuildMessagesWithInput creates a message slice from history and user input.
// This is the single source of truth for building message arrays.
func BuildMessagesWithInput(history []ai.Message, input string) []ai.Message {
	messages := make([]ai.Message, 0, len(history)+1)
	messages = append(messages, history...)
	messages = append(messages, ai.Message{
		Role:    "user",
		Content: input,
	})
	return messages
}

// ExecuteToolWithEvents executes a tool with full event streaming.
// This is the single source of truth for tool execution with events.
// Returns (result, durationMs, error).
func ExecuteToolWithEvents(
	ctx context.Context,
	tools []agent.ToolWithSchema,
	toolName string,
	toolInput string,
	callback agent.EventCallback,
	stats *ExecutionStats,
	startTime time.Time,
) (string, int64, error) {
	safeCallback := agent.SafeCallback(callback)
	toolStartTime := time.Now()

	// Send tool use event (if callback exists)
	if safeCallback != nil {
		safeCallback(agent.EventTypeToolUse, &agent.EventWithMeta{
			EventType: agent.EventTypeToolUse,
			EventData: toolInput,
			Meta: &agent.EventMeta{
				ToolName:        toolName,
				Status:          "running",
				TotalDurationMs: time.Since(startTime).Milliseconds(),
			},
		})
	}

	// Execute tool
	result, err := FindAndExecuteTool(ctx, tools, toolName, toolInput)
	toolDuration := time.Since(toolStartTime).Milliseconds()

	if err != nil {
		if safeCallback != nil {
			safeCallback(agent.EventTypeToolResult, &agent.EventWithMeta{
				EventType: agent.EventTypeToolResult,
				EventData: fmt.Sprintf("Error: %v", err),
				Meta: &agent.EventMeta{
					ToolName:        toolName,
					Status:          "error",
					ErrorMsg:        err.Error(),
					DurationMs:      toolDuration,
					TotalDurationMs: time.Since(startTime).Milliseconds(),
				},
			})
		}
		return "", toolDuration, err
	}

	if safeCallback != nil {
		safeCallback(agent.EventTypeToolResult, &agent.EventWithMeta{
			EventType: agent.EventTypeToolResult,
			EventData: result,
			Meta: &agent.EventMeta{
				ToolName:        toolName,
				Status:          "success",
				DurationMs:      toolDuration,
				TotalDurationMs: time.Since(startTime).Milliseconds(),
			},
		})
	}

	// Update stats
	stats.ToolCalls++
	stats.ToolDurationMs += toolDuration

	// Track unique tool names
	hasTool := false
	for _, t := range stats.ToolsUsed {
		if t == toolName {
			hasTool = true
			break
		}
	}
	if !hasTool {
		stats.ToolsUsed = append(stats.ToolsUsed, toolName)
	}

	return result, toolDuration, nil
}

// StreamResult contains the complete streaming response and stats.
type StreamResult struct {
	Content string
	Stats   *ai.LLMCallStats
	Error   error
}

// CollectChatStream collects all channels from ChatStream into a single result.
// This consolidates the repetitive channel-select pattern.
func CollectChatStream(
	ctx context.Context,
	contentChan <-chan string,
	statsChan <-chan *ai.LLMCallStats,
	errChan <-chan error,
	callback agent.EventCallback,
) *StreamResult {
	result := &StreamResult{}
	safeCallback := agent.SafeCallback(callback)

	for {
		select {
		case content, ok := <-contentChan:
			if ok {
				if safeCallback != nil {
					safeCallback(agent.EventTypeThinking, content)
				}
				result.Content += content
			} else {
				contentChan = nil
			}

		case llmStats, ok := <-statsChan:
			if ok {
				result.Stats = llmStats
			} else {
				statsChan = nil
			}

		case err, ok := <-errChan:
			if ok && err != nil {
				result.Error = err
			} else {
				errChan = nil
			}

		case <-ctx.Done():
			result.Error = ctx.Err()
			return result
		}

		if contentChan == nil && statsChan == nil && errChan == nil {
			return result
		}
	}
}
