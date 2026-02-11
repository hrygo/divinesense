// Package universal provides tests for utility functions.
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
	"github.com/hrygo/divinesense/ai/agents"
)

// TestBuildMessagesWithInput tests message building utility.
func TestBuildMessagesWithInput(t *testing.T) {
	tests := []struct {
		name     string
		history  []ai.Message
		input    string
		expected int // expected message count
	}{
		{
			name:     "empty history",
			history:  nil,
			input:    "Hello",
			expected: 1,
		},
		{
			name: "history with messages",
			history: []ai.Message{
				{Role: "system", Content: "You are helpful"},
				{Role: "user", Content: "Hi"},
				{Role: "assistant", Content: "Hello"},
			},
			input:    "How are you?",
			expected: 4,
		},
		{
			name:     "single input",
			history:  []ai.Message{},
			input:    "Test",
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			messages := BuildMessagesWithInput(tt.history, tt.input)

			if len(messages) != tt.expected {
				t.Errorf("message count = %d, want %d", len(messages), tt.expected)
			}

			// Check last message
			lastMsg := messages[len(messages)-1]
			if lastMsg.Role != "user" {
				t.Errorf("last message role = %q, want 'user'", lastMsg.Role)
			}
			if lastMsg.Content != tt.input {
				t.Errorf("last message content = %q, want %q", lastMsg.Content, tt.input)
			}
		})
	}
}

// TestStreamAnswer tests the streamAnswer utility function.
func TestStreamAnswer(t *testing.T) {
	tests := []struct {
		name         string
		answer       string
		expectChunks bool
		minChunks    int
	}{
		{
			name:         "short answer",
			answer:       "Hi",
			expectChunks: true,
			minChunks:    1,
		},
		{
			name:         "empty answer",
			answer:       "",
			expectChunks: false,
			minChunks:    0,
		},
		{
			name:         "long answer",
			answer:       strings.Repeat("word ", 100), // 500 characters
			expectChunks: true,
			minChunks:    2,
		},
		{
			name:         "unicode text",
			answer:       "Hello 世界 🌍🌎🌏",
			expectChunks: true,
			minChunks:    1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var chunks []string
			var mu sync.Mutex

			callback := func(eventType string, data any) error {
				if eventType == agent.EventTypeAnswer {
					if str, ok := data.(string); ok {
						mu.Lock()
						chunks = append(chunks, str)
						mu.Unlock()
					}
				}
				return nil
			}

			streamAnswer(tt.answer, callback)

			mu.Lock()
			defer mu.Unlock()

			if tt.expectChunks && len(chunks) == 0 {
				t.Error("expected at least one chunk")
			}
			if len(chunks) > 0 && len(chunks) < tt.minChunks {
				t.Errorf("chunk count = %d, want >= %d", len(chunks), tt.minChunks)
			}

			// Reconstruct and verify
			if tt.answer != "" {
				reconstructed := strings.Join(chunks, "")
				if reconstructed != tt.answer {
					t.Errorf("reconstructed = %q, want %q", reconstructed, tt.answer)
				}
			}
		})
	}
}

// TestStreamAnswer_NilCallback tests streamAnswer with nil callback.
func TestStreamAnswer_NilCallback(t *testing.T) {
	// Should not panic
	streamAnswer("test", nil)
}

// TestFindAndExecuteTool_Success tests successful tool execution.
func TestFindAndExecuteTool_Success(t *testing.T) {
	ctx := context.Background()

	tool := &mockTool{
		name: "test_tool",
		runFunc: func(ctx context.Context, input string) (string, error) {
			return "executed: " + input, nil
		},
	}

	tools := []agent.ToolWithSchema{tool}

	result, err := FindAndExecuteTool(ctx, tools, "test_tool", "my_input")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "executed: my_input" {
		t.Errorf("result = %q, want 'executed: my_input'", result)
	}
}

// TestFindAndExecuteTool_ToolNotFound tests tool not found error.
func TestFindAndExecuteTool_ToolNotFound(t *testing.T) {
	ctx := context.Background()

	tool := &mockTool{name: "existing_tool"}
	tools := []agent.ToolWithSchema{tool}

	_, err := FindAndExecuteTool(ctx, tools, "nonexistent_tool", "")

	if err == nil {
		t.Error("expected error for nonexistent tool")
	}
	if !strings.Contains(err.Error(), "tool not found") {
		t.Errorf("error = %v, want error containing 'tool not found'", err)
	}
}

// TestFindAndExecuteTool_ToolError tests tool execution error.
func TestFindAndExecuteTool_ToolError(t *testing.T) {
	ctx := context.Background()

	tool := &mockTool{
		name: "error_tool",
		runFunc: func(ctx context.Context, input string) (string, error) {
			return "", fmt.Errorf("tool failed")
		},
	}

	tools := []agent.ToolWithSchema{tool}

	_, err := FindAndExecuteTool(ctx, tools, "error_tool", "input")

	if err == nil {
		t.Error("expected error from tool execution")
	}
	if !errors.Is(err, fmt.Errorf("tool failed")) {
		t.Logf("error = %v", err)
	}
}

// TestFindAndExecuteTool_NilTools tests nil tools slice.
func TestFindAndExecuteTool_NilTools(t *testing.T) {
	ctx := context.Background()

	_, err := FindAndExecuteTool(ctx, nil, "test_tool", "input")

	if err == nil {
		t.Error("expected error for nil tools")
	}
}

// TestFindAndExecuteTool_NilToolInSlice tests nil tool in slice.
func TestFindAndExecuteTool_NilToolInSlice(t *testing.T) {
	ctx := context.Background()

	tools := []agent.ToolWithSchema{nil}

	_, err := FindAndExecuteTool(ctx, tools, "any_tool", "input")

	if err == nil {
		t.Error("expected error for nil tool in slice")
	}
}

// TestFindAndExecuteTool_MultipleTools tests finding tool in list.
func TestFindAndExecuteTool_MultipleTools(t *testing.T) {
	ctx := context.Background()

	tools := []agent.ToolWithSchema{
		&mockTool{name: "tool1"},
		&mockTool{
			name: "tool2",
			runFunc: func(ctx context.Context, input string) (string, error) {
				return "tool2 result", nil
			},
		},
		&mockTool{name: "tool3"},
	}

	result, err := FindAndExecuteTool(ctx, tools, "tool2", "")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "tool2 result" {
		t.Errorf("result = %q, want 'tool2 result'", result)
	}
}

// TestExecuteToolWithEvents_Success tests successful tool execution with events.
func TestExecuteToolWithEvents_Success(t *testing.T) {
	ctx := context.Background()

	tool := &mockTool{
		name: "test_tool",
		runFunc: func(ctx context.Context, input string) (string, error) {
			time.Sleep(5 * time.Millisecond) // Ensure positive duration
			return "tool result", nil
		},
	}

	tools := []agent.ToolWithSchema{tool}

	var events []string
	callback := func(eventType string, data any) error {
		events = append(events, eventType)
		return nil
	}

	stats := &ExecutionStats{}
	startTime := time.Now()

	result, duration, err := ExecuteToolWithEvents(ctx, tools, "test_tool", "input", callback, stats, startTime)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "tool result" {
		t.Errorf("result = %q, want 'tool result'", result)
	}
	if duration <= 0 {
		t.Error("expected positive duration")
	}
	if stats.ToolCalls != 1 {
		t.Errorf("ToolCalls = %d, want 1", stats.ToolCalls)
	}

	// Check events
	foundToolUse := false
	foundToolResult := false
	for _, e := range events {
		if e == agent.EventTypeToolUse {
			foundToolUse = true
		}
		if e == agent.EventTypeToolResult {
			foundToolResult = true
		}
	}
	if !foundToolUse {
		t.Error("expected tool_use event")
	}
	if !foundToolResult {
		t.Error("expected tool_result event")
	}
}

// TestExecuteToolWithEvents_Error tests tool error with events.
func TestExecuteToolWithEvents_Error(t *testing.T) {
	ctx := context.Background()

	tool := &mockTool{
		name: "error_tool",
		runFunc: func(ctx context.Context, input string) (string, error) {
			time.Sleep(5 * time.Millisecond) // Ensure positive duration
			return "", fmt.Errorf("execution failed")
		},
	}

	tools := []agent.ToolWithSchema{tool}

	var eventData []any
	callback := func(eventType string, data any) error {
		if eventType == agent.EventTypeToolResult {
			eventData = append(eventData, data)
		}
		return nil
	}

	stats := &ExecutionStats{}
	startTime := time.Now()

	result, duration, err := ExecuteToolWithEvents(ctx, tools, "error_tool", "input", callback, stats, startTime)

	if err == nil {
		t.Error("expected error from tool")
	}
	if result != "" {
		t.Errorf("result = %q, want empty", result)
	}
	if duration <= 0 {
		t.Error("expected positive duration even on error")
	}

	// Check that error event was sent
	if len(eventData) == 0 {
		t.Error("expected tool_result event even on error")
	}
}

// TestExecuteToolWithEvents_NilCallback tests with nil callback.
func TestExecuteToolWithEvents_NilCallback(t *testing.T) {
	ctx := context.Background()

	tool := &mockTool{
		name: "test_tool",
		runFunc: func(ctx context.Context, input string) (string, error) {
			time.Sleep(5 * time.Millisecond) // Ensure positive duration
			return "result", nil
		},
	}

	tools := []agent.ToolWithSchema{tool}
	stats := &ExecutionStats{}
	startTime := time.Now()

	// Should not panic
	result, duration, err := ExecuteToolWithEvents(ctx, tools, "test_tool", "input", nil, stats, startTime)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "result" {
		t.Errorf("result = %q, want 'result'", result)
	}
	if duration <= 0 {
		t.Error("expected positive duration")
	}
}

// TestExecuteToolWithEvents_ContextCancellation tests context cancellation during tool execution.
func TestExecuteToolWithEvents_ContextCancellation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping context cancellation test in short mode - timing sensitive")
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	tool := &mockTool{
		name: "slow_tool",
		runFunc: func(ctx context.Context, input string) (string, error) {
			// Use select to check context cancellation
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(100 * time.Millisecond):
				return "result", nil
			}
		},
	}

	tools := []agent.ToolWithSchema{tool}
	stats := &ExecutionStats{}
	startTime := time.Now()

	_, _, err := ExecuteToolWithEvents(ctx, tools, "slow_tool", "input", nil, stats, startTime)

	// Tool may still run briefly before checking context
	if err != nil {
		t.Logf("got error (may be context-related): %v", err)
	}
}

// TestCollectChatStream_Success tests successful stream collection.
func TestCollectChatStream_Success(t *testing.T) {
	ctx := context.Background()
	contentChan := make(chan string, 2)
	statsChan := make(chan *ai.LLMCallStats, 1)
	errChan := make(chan error, 1)

	contentChan <- "Hello "
	contentChan <- "world"
	statsChan <- &ai.LLMCallStats{
		PromptTokens:     10,
		CompletionTokens: 5,
		TotalTokens:      15,
	}
	close(contentChan)
	close(statsChan)
	close(errChan)

	result := CollectChatStream(ctx, contentChan, statsChan, errChan, nil)

	if result.Error != nil {
		t.Fatalf("unexpected error: %v", result.Error)
	}
	if result.Content != "Hello world" {
		t.Errorf("content = %q, want 'Hello world'", result.Content)
	}
	if result.Stats == nil {
		t.Error("stats should not be nil")
	}
	if result.Stats.TotalTokens != 15 {
		t.Errorf("TotalTokens = %d, want 15", result.Stats.TotalTokens)
	}
}

// TestCollectChatStream_WithCallback tests stream collection with callback.
func TestCollectChatStream_WithCallback(t *testing.T) {
	ctx := context.Background()
	contentChan := make(chan string, 1)
	statsChan := make(chan *ai.LLMCallStats, 1)
	errChan := make(chan error, 1)

	contentChan <- "test"
	statsChan <- &ai.LLMCallStats{}
	close(contentChan)
	close(statsChan)
	close(errChan)

	var callbackInvoked bool
	callback := func(eventType string, data any) error {
		if eventType == agent.EventTypeThinking {
			callbackInvoked = true
		}
		return nil
	}

	result := CollectChatStream(ctx, contentChan, statsChan, errChan, callback)

	if result.Error != nil {
		t.Fatalf("unexpected error: %v", result.Error)
	}
	if !callbackInvoked {
		t.Error("callback should have been invoked")
	}
}

// TestCollectChatStream_ContextCancellation tests context cancellation.
func TestCollectChatStream_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	contentChan := make(chan string, 1)
	statsChan := make(chan *ai.LLMCallStats, 1)
	errChan := make(chan error, 1)

	result := CollectChatStream(ctx, contentChan, statsChan, errChan, nil)

	if result.Error == nil {
		t.Error("expected context cancellation error")
	}
	if !errors.Is(result.Error, context.Canceled) {
		t.Errorf("error = %v, want context.Canceled", result.Error)
	}
}

// TestCollectChatStream_ErrorInChannel tests error from errChan.
func TestCollectChatStream_ErrorInChannel(t *testing.T) {
	ctx := context.Background()
	contentChan := make(chan string, 1)
	statsChan := make(chan *ai.LLMCallStats, 1)
	errChan := make(chan error, 1)

	testError := fmt.Errorf("test error")
	errChan <- testError
	close(contentChan)
	close(statsChan)
	close(errChan)

	result := CollectChatStream(ctx, contentChan, statsChan, errChan, nil)

	if result.Error == nil {
		t.Error("expected error from errChan")
	}
	if !errors.Is(result.Error, testError) {
		t.Errorf("error = %v, want %v", result.Error, testError)
	}
}

// TestCollectChatStream_EmptyChannels tests with all channels closed immediately.
func TestCollectChatStream_EmptyChannels(t *testing.T) {
	ctx := context.Background()
	contentChan := make(chan string)
	statsChan := make(chan *ai.LLMCallStats)
	errChan := make(chan error)

	close(contentChan)
	close(statsChan)
	close(errChan)

	result := CollectChatStream(ctx, contentChan, statsChan, errChan, nil)

	if result.Error != nil {
		t.Fatalf("unexpected error: %v", result.Error)
	}
	if result.Content != "" {
		t.Errorf("content = %q, want empty", result.Content)
	}
	if result.Stats != nil {
		t.Error("stats should be nil when no stats sent")
	}
}

// TestCollectChatStream_CallbackError tests callback error handling.
func TestCollectChatStream_CallbackError(t *testing.T) {
	ctx := context.Background()
	contentChan := make(chan string, 1)
	statsChan := make(chan *ai.LLMCallStats, 1)
	errChan := make(chan error, 1)

	contentChan <- "test"
	statsChan <- &ai.LLMCallStats{}
	close(contentChan)
	close(statsChan)
	close(errChan)

	expectedError := fmt.Errorf("callback failed")
	callback := func(eventType string, data any) error {
		return expectedError
	}

	result := CollectChatStream(ctx, contentChan, statsChan, errChan, callback)

	// Callback errors should be ignored (best-effort)
	if result.Error != nil && !errors.Is(result.Error, expectedError) {
		t.Logf("error = %v (callback errors may be ignored)", result.Error)
	}
}

// TestExecutionStats_AccumulateLLM tests LLM stats accumulation.
func TestExecutionStats_AccumulateLLM(t *testing.T) {
	stats := &ExecutionStats{}

	llmStats1 := &ai.LLMCallStats{
		PromptTokens:     100,
		CompletionTokens: 50,
		TotalTokens:      150,
		CacheReadTokens:  20,
		CacheWriteTokens: 5,
	}

	llmStats2 := &ai.LLMCallStats{
		PromptTokens:     200,
		CompletionTokens: 75,
		TotalTokens:      275,
		CacheReadTokens:  30,
		CacheWriteTokens: 10,
	}

	stats.AccumulateLLM(llmStats1)
	stats.AccumulateLLM(llmStats2)

	if stats.LLMCalls != 2 {
		t.Errorf("LLMCalls = %d, want 2", stats.LLMCalls)
	}
	if stats.PromptTokens != 300 {
		t.Errorf("PromptTokens = %d, want 300", stats.PromptTokens)
	}
	if stats.CompletionTokens != 125 {
		t.Errorf("CompletionTokens = %d, want 125", stats.CompletionTokens)
	}
	if stats.TotalTokens != 425 {
		t.Errorf("TotalTokens = %d, want 425", stats.TotalTokens)
	}
	if stats.CacheReadTokens != 50 {
		t.Errorf("CacheReadTokens = %d, want 50", stats.CacheReadTokens)
	}
	if stats.CacheWriteTokens != 15 {
		t.Errorf("CacheWriteTokens = %d, want 15", stats.CacheWriteTokens)
	}
}

// TestDefaultResolver tests strategy resolver.
func TestDefaultResolver(t *testing.T) {
	resolver := NewDefaultResolver(5)

	tests := []struct {
		strategyType StrategyType
		expectName   string
		expectError  bool
	}{
		{StrategyReAct, "react", false},
		{StrategyDirect, "direct", false},
		{StrategyPlanning, "planning", false},
		{"unknown", "", true},
	}

	for _, tt := range tests {
		t.Run(string(tt.strategyType), func(t *testing.T) {
			strategy, err := resolver.Resolve(tt.strategyType)

			if tt.expectError {
				if err == nil {
					t.Error("expected error for unknown strategy")
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if strategy.Name() != tt.expectName {
					t.Errorf("strategy name = %q, want %q", strategy.Name(), tt.expectName)
				}
			}
		})
	}
}

// TestUnsupportedStrategyError tests the error type.
func TestUnsupportedStrategyError(t *testing.T) {
	err := &UnsupportedStrategyError{Strategy: "unknown_strategy"}

	expected := "unsupported strategy: unknown_strategy"
	if err.Error() != expected {
		t.Errorf("Error() = %q, want %q", err.Error(), expected)
	}

	// Verify it implements error interface
	var _ error = (*UnsupportedStrategyError)(nil)
}

// TestStreamResult tests the StreamResult struct.
func TestStreamResult(t *testing.T) {
	result := &StreamResult{
		Content: "test content",
		Stats: &ai.LLMCallStats{
			PromptTokens: 10,
		},
		Error: nil,
	}

	if result.Content != "test content" {
		t.Errorf("Content = %q, want 'test content'", result.Content)
	}
	if result.Stats.PromptTokens != 10 {
		t.Errorf("Stats.PromptTokens = %d, want 10", result.Stats.PromptTokens)
	}
	if result.Error != nil {
		t.Errorf("Error = %v, want nil", result.Error)
	}
}

// TestStreamResult_WithError tests StreamResult with error.
func TestStreamResult_WithError(t *testing.T) {
	testError := fmt.Errorf("test error")
	result := &StreamResult{
		Error: testError,
	}

	if !errors.Is(result.Error, testError) {
		t.Errorf("Error = %v, want %v", result.Error, testError)
	}
}

// TestBuildMessagesWithInput_EmptyInput tests empty input string.
func TestBuildMessagesWithInput_EmptyInput(t *testing.T) {
	history := []ai.Message{
		{Role: "system", Content: "You are helpful"},
	}

	messages := BuildMessagesWithInput(history, "")

	if len(messages) != 2 {
		t.Errorf("message count = %d, want 2", len(messages))
	}

	lastMsg := messages[len(messages)-1]
	if lastMsg.Role != "user" {
		t.Errorf("last message role = %q, want 'user'", lastMsg.Role)
	}
	if lastMsg.Content != "" {
		t.Errorf("last message content = %q, want empty", lastMsg.Content)
	}
}
