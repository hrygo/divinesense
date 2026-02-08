// Package universal provides ReAct loop execution strategy.
package universal

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/hrygo/divinesense/ai"
	"github.com/hrygo/divinesense/ai/agent"
)

// ReActExecutor implements the reasoning-acting loop pattern.
// It uses a loop where the LLM generates thoughts and tool calls
// until a final answer is reached.
type ReActExecutor struct {
	maxIterations int
}

// NewReActExecutor creates a new ReActExecutor with streaming enabled.
func NewReActExecutor(maxIterations int) *ReActExecutor {
	if maxIterations <= 0 {
		maxIterations = 10 // Default safety limit
	}
	return &ReActExecutor{
		maxIterations: maxIterations,
	}
}

// Name returns the strategy name.
func (e *ReActExecutor) Name() string {
	return "react"
}

// Execute runs the ReAct loop with streaming support.
func (e *ReActExecutor) Execute(
	ctx context.Context,
	input string,
	history []ai.Message,
	tools []agent.ToolWithSchema,
	llm ai.LLMService,
	callback agent.EventCallback,
) (string, *ExecutionStats, error) {
	stats := &ExecutionStats{Strategy: "react"}
	startTime := time.Now()
	defer func() {
		stats.TotalDurationMs = time.Since(startTime).Milliseconds()
	}()

	// Build messages from history + current input
	messages := BuildMessagesWithInput(history, input)

	// Safe callback for non-critical events
	safeCallback := agent.SafeCallback(callback)

	// ReAct iteration loop
	for iteration := 0; iteration < e.maxIterations; iteration++ {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return "", stats, ctx.Err()
		default:
		}

		// Send thinking event with iteration info
		safeCallback(agent.EventTypeThinking, &agent.EventWithMeta{
			EventType: agent.EventTypeThinking,
			Meta: &agent.EventMeta{
				CurrentStep:     int32(iteration + 1),
				TotalSteps:      int32(e.maxIterations),
				TotalDurationMs: time.Since(startTime).Milliseconds(),
			},
		})

		// Use ChatStream for streaming LLM response
		contentChan, statsChan, errChan := llm.ChatStream(ctx, messages)

		// Collect all streaming data
		streamResult := CollectChatStream(ctx, contentChan, statsChan, errChan, callback)
		if streamResult.Error != nil {
			return "", stats, fmt.Errorf("LLM streaming failed: %w", streamResult.Error)
		}
		if streamResult.Stats != nil {
			stats.AccumulateLLM(streamResult.Stats)
		}
		response := streamResult.Content

		// Parse tool call from response
		toolName, toolInput, cleanText := parseToolCall(response)

		// No tool call = final answer
		if toolName == "" {
			// Stream the final answer
			streamAnswer(response, callback)
			return response, stats, nil
		}

		// Send clean text (pleasantries before tool call)
		if cleanText != "" {
			safeCallback(agent.EventTypeAnswer, cleanText)
		}

		// Execute tool with events
		toolResult, _, err := ExecuteToolWithEvents(ctx, tools, toolName, toolInput, callback, stats, startTime)

		// Check context cancellation after tool execution
		select {
		case <-ctx.Done():
			return "", stats, ctx.Err()
		default:
		}

		if err != nil {
			slog.Warn("tool execution failed", "tool", toolName, "error", err)
		}

		// Append assistant message and tool result to conversation
		messages = append(messages,
			ai.Message{Role: "assistant", Content: response},
			ai.Message{Role: "user", Content: fmt.Sprintf("工具结果: %s", toolResult)},
		)
	}

	return "", stats, fmt.Errorf("max iterations (%d) exceeded", e.maxIterations)
}

// StreamingSupported returns true - ReAct executor supports streaming with step-by-step output.
func (e *ReActExecutor) StreamingSupported() bool {
	return true
}

// parseToolCall extracts tool call information from LLM response.
// It looks for the pattern: "TOOL: tool_name" followed by "INPUT: {json}".
func parseToolCall(response string) (toolName, toolInput, cleanText string) {
	lines := strings.Split(response, "\n")

	var toolNameStr, toolInputStr string
	var cleanTextBuilder strings.Builder
	inToolCall := false

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if strings.HasPrefix(line, "TOOL:") || strings.HasPrefix(line, "Tool:") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				toolNameStr = strings.TrimSpace(parts[1])
				inToolCall = true
			}
			continue
		}

		if strings.HasPrefix(line, "INPUT:") || strings.HasPrefix(line, "Input:") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				toolInputStr = strings.TrimSpace(parts[1])
			}
			continue
		}

		if !inToolCall && line != "" {
			if cleanTextBuilder.Len() > 0 {
				cleanTextBuilder.WriteString(" ")
			}
			cleanTextBuilder.WriteString(line)
		}
	}

	cleanText = strings.TrimSpace(cleanTextBuilder.String())
	// If clean text is too long, truncate it at UTF-8 rune boundary
	if utf8.RuneCountInString(cleanText) > 200 {
		runes := []rune(cleanText)
		cleanText = string(runes[:200]) + "..."
	}

	return toolNameStr, toolInputStr, cleanText
}
