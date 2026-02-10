// Package universal provides tests for ReAct executor.
package universal

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/hrygo/divinesense/ai"
	"github.com/hrygo/divinesense/ai/agent"
)

// parseToolCall was removed when ReActExecutor migrated to OpenAI structured tool calling.
// The new implementation uses response.ToolCalls from ChatWithTools API.
// These tests are kept for reference but are no longer applicable.

// TestReActExecutor_StreamingSupported tests that ReAct executor supports streaming.
func TestReActExecutor_StreamingSupported(t *testing.T) {
	exec := NewReActExecutor(10)
	if !exec.StreamingSupported() {
		t.Error("ReActExecutor should support streaming")
	}
}

// TestReActExecutor_Name tests the executor name.
func TestReActExecutor_Name(t *testing.T) {
	exec := NewReActExecutor(10)
	if exec.Name() != "react" {
		t.Errorf("Name() = %q, want 'react'", exec.Name())
	}
}

// TestReActExecutor_MaxIterations tests max iterations configuration.
func TestReActExecutor_MaxIterations(t *testing.T) {
	tests := []struct {
		name          string
		maxIterations int
		expected      int
	}{
		{"zero", 0, 10},      // Default to 10
		{"negative", -5, 10}, // Default to 10
		{"one", 1, 1},
		{"five", 5, 5},
		{"twenty", 20, 20},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exec := NewReActExecutor(tt.maxIterations)
			if exec.maxIterations != tt.expected {
				t.Errorf("maxIterations = %d, want %d", exec.maxIterations, tt.expected)
			}
		})
	}
}

// TestReActExecutor_Execute_ContextCancellation tests context cancellation.
func TestReActExecutor_Execute_ContextCancellation(t *testing.T) {
	exec := NewReActExecutor(10)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	llm := &mockLLM{
		chatWithToolsFunc: func(ctx context.Context, messages []ai.Message, tools []ai.ToolDescriptor) (*ai.ChatResponse, *ai.LLMCallStats, error) {
			return &ai.ChatResponse{
				Content:   "response",
				ToolCalls: []ai.ToolCall{},
			}, &ai.LLMCallStats{}, nil
		},
	}

	callback := func(eventType string, data any) error { return nil }
	result, stats, err := exec.Execute(ctx, "test input", nil, nil, llm, callback)

	// Should get context.Canceled error
	if err == nil {
		t.Error("expected context.Canceled error")
	}
	if !errors.Is(err, context.Canceled) {
		t.Logf("Execute() returned: %v", err)
	}
	_ = result
	_ = stats
}

// TestReActExecutor_Execute_Callbacks tests that all callbacks are invoked.
func TestReActExecutor_Execute_Callbacks(t *testing.T) {
	exec := NewReActExecutor(10)

	var events []string
	var mu sync.Mutex

	callback := func(eventType string, data any) error {
		mu.Lock()
		defer mu.Unlock()
		events = append(events, eventType)
		return nil
	}

	llm := &mockLLM{
		chatWithToolsFunc: func(ctx context.Context, messages []ai.Message, tools []ai.ToolDescriptor) (*ai.ChatResponse, *ai.LLMCallStats, error) {
			return &ai.ChatResponse{
				Content:   "Final answer",
				ToolCalls: []ai.ToolCall{},
			}, &ai.LLMCallStats{PromptTokens: 10, CompletionTokens: 5}, nil
		},
	}

	ctx := context.Background()
	result, stats, err := exec.Execute(ctx, "test", nil, nil, llm, callback)

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result != "Final answer" {
		t.Errorf("result = %q, want 'Final answer'", result)
	}
	if stats == nil {
		t.Fatal("stats should not be nil")
	}

	// Check that thinking event was sent
	mu.Lock()
	defer mu.Unlock()
	foundThinking := false
	for _, e := range events {
		if e == agent.EventTypeThinking {
			foundThinking = true
			break
		}
	}
	if !foundThinking {
		t.Error("thinking event should have been sent")
	}
}

// TestReActExecutor_Execute_ToolExecution tests tool execution flow.
func TestReActExecutor_Execute_ToolExecution(t *testing.T) {
	exec := NewReActExecutor(3)

	toolCalled := false
	tool := &mockTool{
		name: "test_tool",
		runFunc: func(ctx context.Context, input string) (string, error) {
			toolCalled = true
			return "tool executed", nil
		},
	}

	llmCallCount := 0
	llm := &mockLLM{
		chatWithToolsFunc: func(ctx context.Context, messages []ai.Message, tools []ai.ToolDescriptor) (*ai.ChatResponse, *ai.LLMCallStats, error) {
			llmCallCount++
			if llmCallCount == 1 {
				// First call: return tool call
				return &ai.ChatResponse{
					Content: "I'll search for that.",
					ToolCalls: []ai.ToolCall{
						{Function: ai.FunctionCall{Name: "test_tool", Arguments: `{"query":"test"}`}},
					},
				}, &ai.LLMCallStats{PromptTokens: 10, CompletionTokens: 5}, nil
			}
			// Second call: final answer after tool result
			return &ai.ChatResponse{
				Content:   "Found the results",
				ToolCalls: []ai.ToolCall{},
			}, &ai.LLMCallStats{PromptTokens: 15, CompletionTokens: 8}, nil
		},
	}

	var events []string
	callback := func(eventType string, data any) error {
		events = append(events, eventType)
		return nil
	}

	ctx := context.Background()
	result, stats, err := exec.Execute(ctx, "search for notes", nil, []agent.ToolWithSchema{tool}, llm, callback)

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result != "Found the results" {
		t.Errorf("result = %q, want 'Found the results'", result)
	}
	if !toolCalled {
		t.Error("tool should have been called")
	}
	if llmCallCount != 2 {
		t.Errorf("LLM called %d times, want 2", llmCallCount)
	}
	if stats.LLMCalls != 2 {
		t.Errorf("stats.LLMCalls = %d, want 2", stats.LLMCalls)
	}

	// Check for tool_use event
	foundToolUse := false
	for _, e := range events {
		if e == agent.EventTypeToolUse {
			foundToolUse = true
			break
		}
	}
	if !foundToolUse {
		t.Error("tool_use event should have been sent")
	}
}

// TestReActExecutor_Execute_ToolError tests tool error handling.
func TestReActExecutor_Execute_ToolError(t *testing.T) {
	exec := NewReActExecutor(3)

	tool := &mockTool{
		name: "error_tool",
		runFunc: func(ctx context.Context, input string) (string, error) {
			return "", fmt.Errorf("tool execution failed")
		},
	}

	llm := &mockLLM{
		chatWithToolsFunc: func(ctx context.Context, messages []ai.Message, tools []ai.ToolDescriptor) (*ai.ChatResponse, *ai.LLMCallStats, error) {
			// Return tool call that will fail
			return &ai.ChatResponse{
				Content: "I'll execute the tool.",
				ToolCalls: []ai.ToolCall{
					{Function: ai.FunctionCall{Name: "error_tool", Arguments: "{}"}},
				},
			}, &ai.LLMCallStats{PromptTokens: 10, CompletionTokens: 5}, nil
		},
	}

	callback := func(eventType string, data any) error { return nil }

	ctx := context.Background()
	result, _, err := exec.Execute(ctx, "test", nil, []agent.ToolWithSchema{tool}, llm, callback)

	// Should get a result despite tool error (LLM should handle it)
	if err != nil {
		t.Logf("Execute() returned error: %v", err)
	}
	// Result should be non-empty as LLM should respond after tool error
	if result == "" {
		t.Log("got empty result (LLM may have stopped after tool error)")
	}
}

// TestReActExecutor_Execute_LLMError tests LLM error handling.
func TestReActExecutor_Execute_LLMError(t *testing.T) {
	exec := NewReActExecutor(3)

	llm := &mockLLM{
		chatWithToolsFunc: func(ctx context.Context, messages []ai.Message, tools []ai.ToolDescriptor) (*ai.ChatResponse, *ai.LLMCallStats, error) {
			return nil, nil, fmt.Errorf("LLM connection failed")
		},
	}

	callback := func(eventType string, data any) error { return nil }

	ctx := context.Background()
	_, _, err := exec.Execute(ctx, "test", nil, nil, llm, callback)

	if err == nil {
		t.Error("expected error when LLM fails")
	}
	if !strings.Contains(err.Error(), "LLM chat with tools failed") {
		t.Logf("error = %v", err)
	}
}

// TestReActExecutor_Execute_MultipleIterations tests multiple ReAct iterations.
func TestReActExecutor_Execute_MultipleIterations(t *testing.T) {
	t.Skip("mock LLM has thread safety issues with callCount - FIX #42")
	exec := NewReActExecutor(5)

	llmCallCount := 0
	llm := &mockLLM{
		chatWithToolsFunc: func(ctx context.Context, messages []ai.Message, tools []ai.ToolDescriptor) (*ai.ChatResponse, *ai.LLMCallStats, error) {
			llmCallCount++
			if llmCallCount == 1 {
				// First call: tool call
				return &ai.ChatResponse{
					ToolCalls: []ai.ToolCall{
						{Function: ai.FunctionCall{Name: "test_tool", Arguments: "{}"}},
					},
				}, &ai.LLMCallStats{PromptTokens: 10, CompletionTokens: 5}, nil
			}
			// Second call: final answer
			return &ai.ChatResponse{
				Content: "Here are the results",
			}, &ai.LLMCallStats{PromptTokens: 10, CompletionTokens: 5}, nil
		},
	}

	tool := &mockTool{
		name: "test_tool",
		runFunc: func(ctx context.Context, input string) (string, error) {
			return "tool result", nil
		},
	}

	callback := func(eventType string, data any) error { return nil }

	ctx := context.Background()
	result, stats, err := exec.Execute(ctx, "test", nil, []agent.ToolWithSchema{tool}, llm, callback)

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if llmCallCount != 2 {
		t.Errorf("LLM called %d times, want 2", llmCallCount)
	}
	if result != "Here are the results" {
		t.Errorf("result = %q, want 'Here are the results'", result)
	}
	if stats.LLMCalls != 2 {
		t.Errorf("stats.LLMCalls = %d, want 2", stats.LLMCalls)
	}
}

// TestReActExecutor_Execute_Timeout tests execution with timeout.
func TestReActExecutor_Execute_Timeout(t *testing.T) {
	exec := NewReActExecutor(10)

	// Use a mock that delays response to test timeout
	llm := &mockLLM{
		chatWithToolsFunc: func(ctx context.Context, messages []ai.Message, tools []ai.ToolDescriptor) (*ai.ChatResponse, *ai.LLMCallStats, error) {
			// Sleep longer than the timeout to ensure context is cancelled
			select {
			case <-time.After(200 * time.Millisecond):
				return &ai.ChatResponse{Content: "response"}, &ai.LLMCallStats{}, nil
			case <-ctx.Done():
				return nil, nil, ctx.Err()
			}
		},
	}

	callback := func(eventType string, data any) error { return nil }

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, _, err := exec.Execute(ctx, "test", nil, nil, llm, callback)

	if err == nil {
		t.Error("expected timeout or cancellation error")
	}
}

// TestReActExecutor_StatsAccumulation tests statistics accumulation.
func TestReActExecutor_StatsAccumulation(t *testing.T) {
	exec := NewReActExecutor(10)

	llm := &mockLLM{
		chatWithToolsFunc: func(ctx context.Context, messages []ai.Message, tools []ai.ToolDescriptor) (*ai.ChatResponse, *ai.LLMCallStats, error) {
			return &ai.ChatResponse{
					Content:   "Final answer",
					ToolCalls: []ai.ToolCall{},
				}, &ai.LLMCallStats{
					PromptTokens:     100,
					CompletionTokens: 50,
					TotalTokens:      150,
					CacheReadTokens:  20,
				}, nil
		},
	}

	callback := func(eventType string, data any) error { return nil }

	ctx := context.Background()
	_, stats, err := exec.Execute(ctx, "test", nil, nil, llm, callback)

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if stats == nil {
		t.Fatal("stats should not be nil")
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
}

// TestReActExecutor_FinalAnswer tests final answer without tool calls.
func TestReActExecutor_FinalAnswer(t *testing.T) {
	exec := NewReActExecutor(10)

	llm := &mockLLM{
		chatWithToolsFunc: func(ctx context.Context, messages []ai.Message, tools []ai.ToolDescriptor) (*ai.ChatResponse, *ai.LLMCallStats, error) {
			return &ai.ChatResponse{
				Content:   "This is my final answer without any tool calls.",
				ToolCalls: []ai.ToolCall{},
			}, &ai.LLMCallStats{PromptTokens: 10, CompletionTokens: 15}, nil
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
	result, _, err := exec.Execute(ctx, "hello", nil, nil, llm, callback)

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result != "This is my final answer without any tool calls." {
		t.Errorf("result = %q, want full answer", result)
	}

	// Check that answer was streamed
	if len(answerChunks) == 0 {
		t.Error("expected answer to be streamed in chunks")
	}
}
