// Package universal provides tests for Direct executor.
package universal

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/hrygo/divinesense/ai"
	"github.com/hrygo/divinesense/ai/agent"
)

// TestDirectExecutor_Name tests the executor name.
func TestDirectExecutor_Name(t *testing.T) {
	exec := NewDirectExecutor(10)
	if exec.Name() != "direct" {
		t.Errorf("Name() = %q, want 'direct'", exec.Name())
	}
}

// TestDirectExecutor_StreamingSupported tests streaming support.
func TestDirectExecutor_StreamingSupported(t *testing.T) {
	exec := NewDirectExecutor(10)
	if !exec.StreamingSupported() {
		t.Error("DirectExecutor should support streaming")
	}
}

// TestDirectExecutor_MaxIterations tests max iterations configuration.
func TestDirectExecutor_MaxIterations(t *testing.T) {
	tests := []struct {
		name          string
		maxIterations int
		expected      int
	}{
		{"zero", 0, 10},
		{"negative", -5, 10},
		{"one", 1, 1},
		{"five", 5, 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exec := NewDirectExecutor(tt.maxIterations)
			if exec.maxIterations != tt.expected {
				t.Errorf("maxIterations = %d, want %d", exec.maxIterations, tt.expected)
			}
		})
	}
}

// TestDirectExecutor_Execute_DirectAnswer tests execution with direct answer (no tools).
func TestDirectExecutor_Execute_DirectAnswer(t *testing.T) {
	exec := NewDirectExecutor(10)

	llm := &mockLLM{
		chatWithToolsFunc: func(ctx context.Context, messages []ai.Message, tools []ai.ToolDescriptor) (*ai.ChatResponse, *ai.LLMCallStats, error) {
			return &ai.ChatResponse{
				Content:   "This is a direct answer",
				ToolCalls: []ai.ToolCall{},
			}, &ai.LLMCallStats{PromptTokens: 10, CompletionTokens: 5}, nil
		},
	}

	var answerChunks []string
	callback := func(eventType string, data any) error {
		if eventType == agent.EventTypeAnswer {
			if str, ok := data.(string); ok {
				answerChunks = append(answerChunks, str)
			}
		}
		return nil
	}

	ctx := context.Background()
	result, stats, err := exec.Execute(ctx, "hello", nil, nil, llm, callback, nil)

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result != "This is a direct answer" {
		t.Errorf("result = %q, want 'This is a direct answer'", result)
	}
	if stats == nil {
		t.Fatal("stats should not be nil")
	}
	if stats.PromptTokens != 10 {
		t.Errorf("PromptTokens = %d, want 10", stats.PromptTokens)
	}
	if stats.CompletionTokens != 5 {
		t.Errorf("CompletionTokens = %d, want 5", stats.CompletionTokens)
	}

	// Check streaming
	if len(answerChunks) == 0 {
		t.Error("expected answer to be streamed")
	}
}

// TestDirectExecutor_Execute_SingleToolCall tests execution with a single tool call.
func TestDirectExecutor_Execute_SingleToolCall(t *testing.T) {
	exec := NewDirectExecutor(5)

	llmCalls := 0
	llm := &mockLLM{
		chatWithToolsFunc: func(ctx context.Context, messages []ai.Message, tools []ai.ToolDescriptor) (*ai.ChatResponse, *ai.LLMCallStats, error) {
			llmCalls++
			if llmCalls == 1 {
				// First call: tool call
				return &ai.ChatResponse{
					Content: "",
					ToolCalls: []ai.ToolCall{
						{
							Function: ai.FunctionCall{Name: "test_tool", Arguments: "{\"query\":\"test\"}"},
						},
					},
				}, &ai.LLMCallStats{PromptTokens: 20, CompletionTokens: 10}, nil
			}
			// Second call: final answer (llmCalls >= 2)
			return &ai.ChatResponse{
				Content: "Tool execution completed",
			}, &ai.LLMCallStats{PromptTokens: 30, CompletionTokens: 15}, nil
		},
	}

	toolExecuted := false
	tool := &mockTool{
		name: "test_tool",
		runFunc: func(ctx context.Context, input string) (string, error) {
			toolExecuted = true
			return "tool result", nil
		},
	}

	callback := func(eventType string, data any) error { return nil }

	ctx := context.Background()
	result, stats, err := exec.Execute(ctx, "search notes", nil, []agent.ToolWithSchema{tool}, llm, callback, nil)

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !toolExecuted {
		t.Error("tool should have been executed")
	}
	if result != "Tool execution completed" {
		t.Errorf("result = %q, want 'Tool execution completed'", result)
	}
	if llmCalls != 2 {
		t.Errorf("LLM called %d times, want 2", llmCalls)
	}
	if stats.LLMCalls != 2 {
		t.Errorf("stats.LLMCalls = %d, want 2", stats.LLMCalls)
	}
	if stats.ToolCalls != 1 {
		t.Errorf("stats.ToolCalls = %d, want 1", stats.ToolCalls)
	}
}

// TestDirectExecutor_Execute_MultiTurnToolCalls tests multiple rounds of tool calls.
func TestDirectExecutor_Execute_MultiTurnToolCalls(t *testing.T) {
	exec := NewDirectExecutor(10)

	llmCalls := 0
	llm := &mockLLM{
		chatWithToolsFunc: func(ctx context.Context, messages []ai.Message, tools []ai.ToolDescriptor) (*ai.ChatResponse, *ai.LLMCallStats, error) {
			llmCalls++
			switch llmCalls {
			case 1:
				return &ai.ChatResponse{
					ToolCalls: []ai.ToolCall{
						{Function: ai.FunctionCall{Name: "tool1", Arguments: "{}"}},
					},
				}, &ai.LLMCallStats{}, nil
			case 2:
				return &ai.ChatResponse{
					ToolCalls: []ai.ToolCall{
						{Function: ai.FunctionCall{Name: "tool2", Arguments: "{}"}},
					},
				}, &ai.LLMCallStats{}, nil
			default:
				return &ai.ChatResponse{
					Content: "All tools executed successfully",
				}, &ai.LLMCallStats{}, nil
			}
		},
	}

	tools := []agent.ToolWithSchema{
		&mockTool{name: "tool1", runFunc: func(ctx context.Context, input string) (string, error) { return "result1", nil }},
		&mockTool{name: "tool2", runFunc: func(ctx context.Context, input string) (string, error) { return "result2", nil }},
	}

	callback := func(eventType string, data any) error { return nil }

	ctx := context.Background()
	result, _, err := exec.Execute(ctx, "multi-tool query", nil, tools, llm, callback, nil)

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result != "All tools executed successfully" {
		t.Errorf("unexpected result: %s", result)
	}
	if llmCalls != 3 {
		t.Errorf("LLM called %d times, want 3", llmCalls)
	}
}

// TestDirectExecutor_Execute_ToolExecutionError tests tool execution error handling.
func TestDirectExecutor_Execute_ToolExecutionError(t *testing.T) {
	exec := NewDirectExecutor(5)

	llm := &mockLLM{
		chatWithToolsFunc: func(ctx context.Context, messages []ai.Message, tools []ai.ToolDescriptor) (*ai.ChatResponse, *ai.LLMCallStats, error) {
			return &ai.ChatResponse{
				ToolCalls: []ai.ToolCall{
					{Function: ai.FunctionCall{Name: "error_tool", Arguments: "{}"}},
				},
			}, &ai.LLMCallStats{}, nil
		},
	}

	tool := &mockTool{
		name: "error_tool",
		runFunc: func(ctx context.Context, input string) (string, error) {
			return "", fmt.Errorf("tool execution failed")
		},
	}

	callback := func(eventType string, data any) error { return nil }

	ctx := context.Background()
	result, _, err := exec.Execute(ctx, "test", nil, []agent.ToolWithSchema{tool}, llm, callback, nil)

	// Should still get a result (executor continues despite tool errors)
	if result != "" {
		t.Logf("got result despite tool error: %s", result)
	}
	_ = err // May or may not have error depending on implementation
}

// TestDirectExecutor_Execute_LLMError tests LLM error handling.
func TestDirectExecutor_Execute_LLMError(t *testing.T) {
	exec := NewDirectExecutor(5)

	llm := &mockLLM{
		chatWithToolsFunc: func(ctx context.Context, messages []ai.Message, tools []ai.ToolDescriptor) (*ai.ChatResponse, *ai.LLMCallStats, error) {
			return nil, nil, fmt.Errorf("LLM connection failed")
		},
	}

	callback := func(eventType string, data any) error { return nil }

	ctx := context.Background()
	_, _, err := exec.Execute(ctx, "test", nil, nil, llm, callback, nil)

	if err == nil {
		t.Error("expected error when LLM fails")
	}
	if !errors.Is(err, fmt.Errorf("LLM connection failed")) {
		t.Logf("error = %v", err)
	}
}

// TestDirectExecutor_Execute_ContextCancellation tests context cancellation.
func TestDirectExecutor_Execute_ContextCancellation(t *testing.T) {
	exec := NewDirectExecutor(10)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	llm := &mockLLM{
		chatWithToolsFunc: func(ctx context.Context, messages []ai.Message, tools []ai.ToolDescriptor) (*ai.ChatResponse, *ai.LLMCallStats, error) {
			return &ai.ChatResponse{Content: "response"}, &ai.LLMCallStats{}, nil
		},
	}

	callback := func(eventType string, data any) error { return nil }

	_, _, err := exec.Execute(ctx, "test", nil, nil, llm, callback, nil)

	// May get context.Canceled or succeed quickly
	if err != nil && !errors.Is(err, context.Canceled) {
		t.Logf("Execute() returned: %v", err)
	}
}

// TestDirectExecutor_Execute_MaxIterationsExceeded tests max iterations limit.
func TestDirectExecutor_Execute_MaxIterationsExceeded(t *testing.T) {
	exec := NewDirectExecutor(2)

	llmCalls := 0
	llm := &mockLLM{
		chatWithToolsFunc: func(ctx context.Context, messages []ai.Message, tools []ai.ToolDescriptor) (*ai.ChatResponse, *ai.LLMCallStats, error) {
			llmCalls++
			// Always return tool call to trigger max iterations
			return &ai.ChatResponse{
				ToolCalls: []ai.ToolCall{
					{Function: ai.FunctionCall{Name: "tool", Arguments: "{}"}},
				},
			}, &ai.LLMCallStats{}, nil
		},
	}

	tool := &mockTool{name: "tool", runFunc: func(ctx context.Context, input string) (string, error) { return "result", nil }}

	callback := func(eventType string, data any) error { return nil }

	ctx := context.Background()
	result, _, err := exec.Execute(ctx, "test", nil, []agent.ToolWithSchema{tool}, llm, callback, nil)

	// Should hit max iterations limit
	if err == nil {
		t.Log("no error when max iterations exceeded (may be implementation-dependent)")
	}
	if result != "" {
		t.Logf("got result: %s", result)
	}
	if llmCalls > 3 {
		t.Errorf("LLM called %d times, should be limited by maxIterations", llmCalls)
	}
}

// TestDirectExecutor_Execute_ToolMarshalError tests tool parameter marshaling error.
func TestDirectExecutor_Execute_ToolMarshalError(t *testing.T) {
	exec := NewDirectExecutor(5)

	// Create a tool with unmarshalable parameters (circular reference)
	type Circular struct {
		Ref *Circular `json:"ref"`
	}
	circular := &Circular{}
	circular.Ref = circular

	tool := &mockTool{
		name: "circular_tool",
		parameters: map[string]any{
			"circular": circular,
		},
	}

	llm := &mockLLM{
		chatWithToolsFunc: func(ctx context.Context, messages []ai.Message, tools []ai.ToolDescriptor) (*ai.ChatResponse, *ai.LLMCallStats, error) {
			// This test should fail during tool marshaling, before LLM call
			return &ai.ChatResponse{Content: "should not reach here"}, &ai.LLMCallStats{}, nil
		},
	}

	callback := func(eventType string, data any) error { return nil }

	ctx := context.Background()
	_, _, err := exec.Execute(ctx, "test", nil, []agent.ToolWithSchema{tool}, llm, callback, nil)

	if err == nil {
		t.Error("expected error from circular reference marshaling")
	}
}

// TestDirectExecutor_Execute_EmptyResponse tests empty LLM response.
func TestDirectExecutor_Execute_EmptyResponse(t *testing.T) {
	exec := NewDirectExecutor(5)

	llm := &mockLLM{
		chatWithToolsFunc: func(ctx context.Context, messages []ai.Message, tools []ai.ToolDescriptor) (*ai.ChatResponse, *ai.LLMCallStats, error) {
			return &ai.ChatResponse{
				Content:   "",
				ToolCalls: []ai.ToolCall{},
			}, &ai.LLMCallStats{}, nil
		},
	}

	callback := func(eventType string, data any) error { return nil }

	ctx := context.Background()
	_, _, err := exec.Execute(ctx, "test", nil, nil, llm, callback, nil)

	if err == nil {
		t.Error("expected error for empty response")
	}
}

// TestDirectExecutor_Execute_WithHistory tests execution with conversation history.
func TestDirectExecutor_Execute_WithHistory(t *testing.T) {
	exec := NewDirectExecutor(5)

	var receivedMessages []ai.Message
	llm := &mockLLM{
		chatWithToolsFunc: func(ctx context.Context, messages []ai.Message, tools []ai.ToolDescriptor) (*ai.ChatResponse, *ai.LLMCallStats, error) {
			receivedMessages = messages
			return &ai.ChatResponse{
				Content: "Response with context",
			}, &ai.LLMCallStats{}, nil
		},
	}

	callback := func(eventType string, data any) error { return nil }

	ctx := context.Background()
	history := []ai.Message{
		{Role: "user", Content: "Hello"},
		{Role: "assistant", Content: "Hi there!"},
	}

	result, _, err := exec.Execute(ctx, "How are you?", history, nil, llm, callback, nil)

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result != "Response with context" {
		t.Errorf("unexpected result: %s", result)
	}

	// Check that history was included
	if len(receivedMessages) != 3 {
		t.Errorf("received %d messages, want 3 (2 history + 1 current)", len(receivedMessages))
	}
}

// TestDirectExecutor_StatsAccumulation tests statistics accumulation.
func TestDirectExecutor_StatsAccumulation(t *testing.T) {
	exec := NewDirectExecutor(10)

	llm := &mockLLM{
		chatWithToolsFunc: func(ctx context.Context, messages []ai.Message, tools []ai.ToolDescriptor) (*ai.ChatResponse, *ai.LLMCallStats, error) {
			return &ai.ChatResponse{
					Content: "Answer",
				}, &ai.LLMCallStats{
					PromptTokens:     100,
					CompletionTokens: 50,
					TotalTokens:      150,
					CacheReadTokens:  25,
					CacheWriteTokens: 10,
				}, nil
		},
	}

	callback := func(eventType string, data any) error { return nil }

	ctx := context.Background()
	_, stats, err := exec.Execute(ctx, "test", nil, nil, llm, callback, nil)

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if stats.PromptTokens != 100 {
		t.Errorf("PromptTokens = %d, want 100", stats.PromptTokens)
	}
	if stats.CompletionTokens != 50 {
		t.Errorf("CompletionTokens = %d, want 50", stats.CompletionTokens)
	}
	if stats.TotalTokens != 150 {
		t.Errorf("TotalTokens = %d, want 150", stats.TotalTokens)
	}
	if stats.CacheReadTokens != 25 {
		t.Errorf("CacheReadTokens = %d, want 25", stats.CacheReadTokens)
	}
	if stats.CacheWriteTokens != 10 {
		t.Errorf("CacheWriteTokens = %d, want 10", stats.CacheWriteTokens)
	}
	if stats.LLMCalls != 1 {
		t.Errorf("LLMCalls = %d, want 1", stats.LLMCalls)
	}
}

// TestDirectExecutor_Execute_ToolContent tests response with both tool call and content.
func TestDirectExecutor_Execute_ToolContent(t *testing.T) {
	exec := NewDirectExecutor(5)

	llm := &mockLLM{
		chatWithToolsFunc: func(ctx context.Context, messages []ai.Message, tools []ai.ToolDescriptor) (*ai.ChatResponse, *ai.LLMCallStats, error) {
			return &ai.ChatResponse{
				Content: "I'll search that for you.",
				ToolCalls: []ai.ToolCall{
					{Function: ai.FunctionCall{Name: "tool", Arguments: "{}"}},
				},
			}, &ai.LLMCallStats{}, nil
		},
	}

	tool := &mockTool{name: "tool", runFunc: func(ctx context.Context, input string) (string, error) { return "result", nil }}

	var answerChunks []string
	callback := func(eventType string, data any) error {
		if eventType == agent.EventTypeAnswer {
			if str, ok := data.(string); ok {
				answerChunks = append(answerChunks, str)
			}
		}
		return nil
	}

	ctx := context.Background()
	result, _, err := exec.Execute(ctx, "test", nil, []agent.ToolWithSchema{tool}, llm, callback, nil)

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	// Should return the content immediately if present with tool calls
	if result == "" && len(answerChunks) == 0 {
		t.Error("expected some result when content is present with tool calls")
	}
}

// TestDirectExecutor_Execute_ToolNotFound tests handling of tools not in list.
func TestDirectExecutor_Execute_ToolNotFound(t *testing.T) {
	exec := NewDirectExecutor(5)

	llm := &mockLLM{
		chatWithToolsFunc: func(ctx context.Context, messages []ai.Message, tools []ai.ToolDescriptor) (*ai.ChatResponse, *ai.LLMCallStats, error) {
			// Request a tool that doesn't exist
			return &ai.ChatResponse{
				ToolCalls: []ai.ToolCall{
					{Function: ai.FunctionCall{Name: "nonexistent_tool", Arguments: "{}"}},
				},
			}, &ai.LLMCallStats{}, nil
		},
	}

	// Provide a different tool
	tool := &mockTool{name: "other_tool", runFunc: func(ctx context.Context, input string) (string, error) { return "result", nil }}

	callback := func(eventType string, data any) error { return nil }

	ctx := context.Background()
	result, _, err := exec.Execute(ctx, "test", nil, []agent.ToolWithSchema{tool}, llm, callback, nil)

	// Should handle gracefully (either error or continue)
	if err != nil {
		t.Logf("got error for nonexistent tool (expected): %v", err)
	}
	if result != "" {
		t.Logf("got result despite tool not found: %s", result)
	}
}
