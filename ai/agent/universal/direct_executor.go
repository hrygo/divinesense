// Package universal provides execution strategy implementations for UniversalParrot.
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

/*
DirectExecutor - Direct Native Tool Calling Strategy

POSITIONING:
  DirectExecutor is designed for SIMPLE, ONE-SHOT tool calls where the LLM
  can determine the final answer without multi-step reasoning.

IDEAL USE CASES:
  - Single tool call with immediate answer (e.g., "创建日程: 明天下午3点开会")
  - Simple CRUD operations (create, update, delete)
  - Actions that don't require data synthesis or explanation

NOT SUITABLE FOR:
  - Queries requiring reasoning (e.g., "下午有空闲时间吗？" - needs schedule analysis)
  - Multi-step planning
  - Scenarios where the LLM needs to "think" before/after tool calls

FOR COMPLEX SCENARIOS, USE:
  - ReActExecutor: For reasoning before/after tool calls
  - PlanningExecutor: For multi-tool coordination with planning phase

ALGORITHM:
  1. Call LLM with tools
  2. If tool_calls returned:
     a. Execute tools
     b. Call LLM again with tool results
     c. Repeat until max_iterations or no more tool_calls
  3. Return final content when no tool_calls present

EXAMPLE FLOW (Simple "Create Schedule"):
  User: "帮我安排明天下午3点开会"
  LLM:  content="好的，我来创建日程" + tool_calls=[schedule_add]
  → Execute schedule_add
  → LLM:  content="✓ 已成功创建日程..." (final answer)
  → Done (1 iteration)

EXAMPLE FLOW (Complex Query - UNSUITABLE):
  User: "下午有空闲时间吗？"
  LLM:  content="让我查询一下" + tool_calls=[schedule_query]
  → Execute schedule_query
  → LLM:  content="让我再确认一下" + tool_calls=[schedule_query]  ← PROBLEM!
  → LLM keeps calling tools instead of generating answer
  → Should use ReActExecutor instead

*/

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
	timeContext *TimeContext,
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

	// Accumulate stats (AccumulateLLM increments LLMCalls)
	stats.AccumulateLLM(llmStats)

	slog.Info("direct: initial LLM response",
		"tool_calls", len(response.ToolCalls),
		"content_length", len(response.Content),
		"content_preview", func() string {
			if len(response.Content) > 100 {
				return response.Content[:100] + "..."
			}
			return response.Content
		}())

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
		slog.Info("direct: iteration check",
			"iteration", stats.LLMCalls,
			"max_iterations", e.maxIterations,
			"tool_calls", len(response.ToolCalls),
			"content_length", len(response.Content),
			"content_preview", func() string {
				if len(response.Content) > 100 {
					return response.Content[:100] + "..."
				}
				return response.Content
			}())

		// Process tool calls if present
		if len(response.ToolCalls) > 0 {
			slog.Info("direct: processing tool calls", "count", len(response.ToolCalls))

			for _, tc := range response.ToolCalls {
				// Execute tool with events
				toolResult, _, err := ExecuteToolWithEvents(ctx, tools, tc.Function.Name, tc.Function.Arguments, callback, stats, startTime)

				if err != nil {
					slog.Warn("tool execution failed", "tool", tc.Function.Name, "error", err)
				}

				// Add tool result as a user message (simulating user giving feedback)
				// Note: We don't add an empty assistant message because DeepSeek API rejects it:
				// "Invalid assistant message: content or tool_calls must be set"
				messages = append(messages,
					ai.Message{Role: "user", Content: fmt.Sprintf("[Result from %s]: %s", tc.Function.Name, toolResult)},
				)
			}

			// Make another LLM call with the updated messages to get final answer
			if stats.LLMCalls < e.maxIterations {
				slog.Info("direct: calling LLM again for final answer")
				response, llmStats, err = llm.ChatWithTools(ctx, messages, toolDescriptors)
				if err != nil {
					return "", stats, fmt.Errorf("follow-up ChatWithTools failed: %w", err)
				}

				// Accumulate stats (AccumulateLLM increments LLMCalls)
				stats.AccumulateLLM(llmStats)

				slog.Info("direct: got LLM response",
					"tool_calls", len(response.ToolCalls),
					"content_length", len(response.Content))

				// Continue to next iteration to process the new response
				continue
			}
		}

		// No tool calls = final answer
		if response.Content != "" {
			slog.Info("direct: sending final answer (no tools)",
				"content_length", len(response.Content))
			streamAnswer(response.Content, callback)
			return response.Content, stats, nil
		}

		// Break if we have no content and no tool calls
		break
	}

	// No content and no tool calls after exhausting iterations
	if response.Content != "" {
		slog.Info("direct: sending final answer after iterations exhausted",
			"content_length", len(response.Content))
		streamAnswer(response.Content, callback)
		return response.Content, stats, nil
	}
	slog.Error("direct: no content after iterations", "llm_calls", stats.LLMCalls)
	return "", stats, fmt.Errorf("LLM returned empty response after %d iterations", stats.LLMCalls)
}

// StreamingSupported returns true - direct executor supports streaming.
func (e *DirectExecutor) StreamingSupported() bool {
	return true
}
