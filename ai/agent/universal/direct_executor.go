// Package universal provides direct execution strategy using native LLM tool calling.
package universal

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/hrygo/divinesense/ai"
	"github.com/hrygo/divinesense/ai/agent"
)

// DirectExecutor uses native LLM tool calling without ReAct loops.
// This leverages modern LLM function calling capabilities for faster
// and more reliable tool execution.
type DirectExecutor struct {
	maxIterations int
}

// NewDirectExecutor creates a new DirectExecutor.
func NewDirectExecutor(maxIterations int) *DirectExecutor {
	if maxIterations <= 0 {
		maxIterations = 10
	}
	return &DirectExecutor{
		maxIterations: maxIterations,
	}
}

// Name returns the strategy name.
func (e *DirectExecutor) Name() string {
	return "direct"
}

// Execute runs the direct tool calling strategy.
func (e *DirectExecutor) Execute(
	ctx context.Context,
	input string,
	history []ai.Message,
	tools []agent.ToolWithSchema,
	llm ai.LLMService,
	callback agent.EventCallback,
) (string, *ExecutionStats, error) {
	stats := &ExecutionStats{Strategy: "direct"}
	startTime := time.Now()
	defer func() {
		stats.TotalDurationMs = time.Since(startTime).Milliseconds()
	}()

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

	// Build messages
	messages := BuildMessagesWithInput(history, input)

	// Use ChatWithTools for native function calling
	response, llmStats, err := llm.ChatWithTools(ctx, messages, toolDescriptors)
	if err != nil {
		return "", stats, fmt.Errorf("ChatWithTools failed: %w", err)
	}

	// Accumulate stats
	stats.AccumulateLLM(llmStats)

	// Send thinking event with token stats
	safeCallback := agent.SafeCallback(callback)
	safeCallback(agent.EventTypeThinking, &agent.EventWithMeta{
		EventType: agent.EventTypeThinking,
		Meta: &agent.EventMeta{
			InputTokens:     int32(llmStats.PromptTokens),
			OutputTokens:    int32(llmStats.CompletionTokens),
			CacheReadTokens: int32(llmStats.CacheReadTokens),
			TotalDurationMs: time.Since(startTime).Milliseconds(),
		},
	})

	// Main execution loop for multi-turn tool calling
	for stats.LLMCalls < e.maxIterations {
		// Process tool calls if present
		if len(response.ToolCalls) > 0 {
			for _, tc := range response.ToolCalls {
				// Execute tool with events
				toolResult, _, err := ExecuteToolWithEvents(ctx, tools, tc.Function.Name, tc.Function.Arguments, callback, stats, startTime)

				if err != nil {
					slog.Warn("tool execution failed", "tool", tc.Function.Name, "error", err)
				}

				// Add tool result to messages for next iteration
				messages = append(messages,
					ai.Message{Role: "assistant", Content: ""}, // Placeholder for tool call
					ai.Message{Role: "user", Content: fmt.Sprintf("[Result from %s]: %s", tc.Function.Name, toolResult)},
				)
			}

			// Check if there's a final answer in the response
			if response.Content != "" {
				streamAnswer(response.Content, callback)
				return response.Content, stats, nil
			}

			// Make another LLM call with the updated messages
			if stats.LLMCalls < e.maxIterations {
				response, llmStats, err = llm.ChatWithTools(ctx, messages, toolDescriptors)
				if err != nil {
					return "", stats, fmt.Errorf("follow-up ChatWithTools failed: %w", err)
				}

				stats.AccumulateLLM(llmStats)

				// Continue to next iteration to process the new response
				continue
			}
		}

		// No tool calls = final answer
		if response.Content != "" {
			streamAnswer(response.Content, callback)
			return response.Content, stats, nil
		}

		// Break if we have no content and no tool calls
		break
	}

	// No content and no tool calls after exhausting iterations
	if response.Content != "" {
		streamAnswer(response.Content, callback)
		return response.Content, stats, nil
	}
	return "", stats, fmt.Errorf("LLM returned empty response after %d iterations", stats.LLMCalls)
}

// StreamingSupported returns true - direct executor supports streaming.
func (e *DirectExecutor) StreamingSupported() bool {
	return true
}
