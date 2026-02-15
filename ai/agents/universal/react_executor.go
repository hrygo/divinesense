// Package universal provides ReAct loop execution strategy.
package universal

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/hrygo/divinesense/ai"
	"github.com/hrygo/divinesense/ai/agents"
)

/*
ReActExecutor - ReAct Loop with Tool Calling

POSITIONING:

	ReActExecutor combines streaming output with tool calling for scenarios that
	require reasoning before/after tool use. The LLM can both stream content and
	call tools in the same response.

ALGORITHM:
 1. LLM streams content while optionally calling tools (via ChatWithTools)
 2. If tool call detected in response:
    a. Execute tool, stream result to user
    b. Add result to messages and continue to next iteration
 3. If no tool call = final answer, return

DIFFERENCE FROM DirectExecutor:
  - Direct: Optimized for simple one-shot tool calls
  - ReAct: Designed for multi-turn reasoning with tool use
*/
type ReActExecutor struct {
	maxIterations int
}

// NewReActExecutor creates a new ReActExecutor.
func NewReActExecutor(maxIterations int) *ReActExecutor {
	if maxIterations <= 0 {
		maxIterations = 10
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
	timeContext *TimeContext,
) (string, *ExecutionStats, error) {
	stats := &ExecutionStats{Strategy: "react"}
	startTime := time.Now()
	defer func() {
		stats.TotalDurationMs = time.Since(startTime).Milliseconds()
	}()

	// Build messages from history + current input
	messages := BuildMessagesWithInput(history, input)

	// Build tool descriptors for LLM
	toolDescriptors := make([]ai.ToolDescriptor, len(tools))
	for i, tool := range tools {
		paramsJSON := tool.Parameters()
		paramsBytes, err := json.Marshal(paramsJSON)
		if err != nil {
			slog.Error("Failed to marshal tool parameters", "tool", tool.Name(), "error", err)
			return "", stats, fmt.Errorf("marshal tool parameters for tool %s: %w", tool.Name(), err)
		}
		toolDescriptors[i] = ai.ToolDescriptor{
			Name:        tool.Name(),
			Description: tool.Description(),
			Parameters:  string(paramsBytes),
		}
	}

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

		// Send thinking event with iteration info (metadata-only, no content)
		safeCallback(agent.EventTypeThinking, &agent.EventWithMeta{
			EventType: agent.EventTypeThinking,
			Meta: &agent.EventMeta{
				CurrentStep:     int32(iteration + 1),
				TotalSteps:      int32(e.maxIterations),
				TotalDurationMs: time.Since(startTime).Milliseconds(),
			},
		})

		// Log LLM call start for observability
		llmStart := time.Now()
		slog.Debug("react: LLM chat started",
			"iteration", iteration+1,
			"message_count", len(messages))

		// Use ChatWithTools for streaming LLM response with tool support
		response, llmStats, err := llm.ChatWithTools(ctx, messages, toolDescriptors)
		if err != nil {
			return "", stats, fmt.Errorf("LLM chat with tools failed: %w", err)
		}
		stats.AccumulateLLM(llmStats)

		slog.Info("react: LLM response",
			"iteration", iteration+1,
			"tool_calls", len(response.ToolCalls),
			"content_length", len(response.Content),
			"duration_ms", time.Since(llmStart).Milliseconds())

		// Check if LLM wants to call tools
		hasStructuredToolCalls := len(response.ToolCalls) > 0

		// No tool calls = final answer (stream and return)
		if !hasStructuredToolCalls {
			if response.Content != "" {
				streamAnswer(response.Content, callback)
			}
			slog.Info("react: no tool calls, returning final answer",
				"content_length", len(response.Content))
			return response.Content, stats, nil
		}

		// Has structured tool calls - stream thinking content first, then execute tools
		if response.Content != "" {
			streamAnswer(response.Content, callback)
		}

		// Execute tools
		for _, tc := range response.ToolCalls {
			toolName := tc.Function.Name
			toolInput := tc.Function.Arguments

			// Notify callback with structured EventWithMeta
			if callback != nil {
				meta := &agent.EventMeta{
					ToolName:     toolName,
					Status:       "running",
					InputSummary: toolInput,
				}
				safeCallback(agent.EventTypeToolUse, &agent.EventWithMeta{
					EventType: agent.EventTypeToolUse,
					EventData: toolInput,
					Meta:      meta,
				})
			}

			toolStart := time.Now()

			// Execute the tool
			toolResult, toolErr := executeTool(ctx, tools, toolName, toolInput)
			status := "success"
			if toolErr != nil {
				status = "error"
				toolResult = fmt.Sprintf("Error: %v", toolErr)
			}

			slog.Info("react: tool execution completed",
				"tool", toolName,
				"status", status,
				"duration_ms", time.Since(toolStart).Milliseconds(),
			)

			// Notify callback of result with structured EventWithMeta
			if callback != nil {
				meta := &agent.EventMeta{
					ToolName:      toolName,
					Status:        status,
					OutputSummary: toolResult,
					DurationMs:    time.Since(toolStart).Milliseconds(),
				}
				safeCallback(agent.EventTypeToolResult, &agent.EventWithMeta{
					EventType: agent.EventTypeToolResult,
					EventData: toolResult,
					Meta:      meta,
				})
			}

			// Add tool result as a user message
			// Note: Only add non-empty assistant message to avoid API rejection
			messages = append(messages,
				ai.Message{Role: "user", Content: fmt.Sprintf("[Result from %s]: %s", toolName, toolResult)},
			)

			// Check for early stopping (for Handoff mechanism)
			// When report_inability is called, we should stop and return the result to trigger handoff
			if shouldEarlyStop(toolResult) {
				slog.Info("react: early stopping due to tool result",
					"tool", toolName,
					"reason", "shouldEarlyStop returned true")
				return toolResult, stats, nil
			}
			if response.Content != "" {
				// Insert assistant message before user message if there was thinking content
				messages = append(messages[:len(messages)-1],
					ai.Message{Role: "assistant", Content: response.Content},
					messages[len(messages)-1],
				)
			}
		}
	}

	return "", stats, fmt.Errorf("max iterations (%d) exceeded", e.maxIterations)
}

// StreamingSupported returns true - ReAct executor supports streaming.
func (e *ReActExecutor) StreamingSupported() bool {
	return true
}

// executeTool finds and executes a tool by name.
func executeTool(ctx context.Context, tools []agent.ToolWithSchema, name, input string) (string, error) {
	for _, tool := range tools {
		if tool.Name() == name {
			return tool.Run(ctx, input)
		}
	}
	return "", fmt.Errorf("unknown tool: %s", name)
}

// shouldEarlyStop checks if the agent should stop early based on tool results.
// Returns true if a tool executed successfully, or inability was reported.
// Success is auto-detected by checking JSON response "success" or "error" fields.
func shouldEarlyStop(toolResult string) bool {
	if toolResult == "" {
		return false
	}

	// Auto-detect success from JSON response
	if isSuccessFromJSON(toolResult) {
		return true
	}

	// Check for inability report (for Handoff mechanism)
	// When an expert reports inability, the agent should stop and let Orchestrator handle handoff
	if strings.Contains(toolResult, "INABILITY_REPORTED:") {
		return true
	}

	return false
}

// isSuccessFromJSON checks if the tool result indicates success by parsing JSON.
// Returns true if:
// - "success" field is true
// - "error" field is null or empty
func isSuccessFromJSON(toolResult string) bool {
	// Try to parse as JSON
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(toolResult), &result); err != nil {
		return false
	}

	// Check for explicit success field
	if success, ok := result["success"].(bool); ok && success {
		return true
	}

	// Check for error field being null/empty
	if err, ok := result["error"]; ok {
		if err == nil {
			return true
		}
		switch v := err.(type) {
		case string:
			if v == "" || v == "null" {
				return true
			}
		case nil:
			return true
		}
	}

	return false
}
