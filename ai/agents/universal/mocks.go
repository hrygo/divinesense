// Package universal provides mock types for testing.
package universal

import (
	"context"

	"github.com/hrygo/divinesense/ai"
)

// mockLLM is a test double for LLMService.
type mockLLM struct {
	chatFunc          func(ctx context.Context, messages []ai.Message) (string, *ai.LLMCallStats, error)
	chatStreamFunc    func(ctx context.Context, messages []ai.Message) (<-chan string, <-chan *ai.LLMCallStats, <-chan error)
	chatWithToolsFunc func(ctx context.Context, messages []ai.Message, tools []ai.ToolDescriptor) (*ai.ChatResponse, *ai.LLMCallStats, error)
}

func (m *mockLLM) Chat(ctx context.Context, messages []ai.Message) (string, *ai.LLMCallStats, error) {
	if m.chatFunc != nil {
		return m.chatFunc(ctx, messages)
	}
	return "test response", &ai.LLMCallStats{PromptTokens: 10, CompletionTokens: 5}, nil
}

func (m *mockLLM) ChatStream(ctx context.Context, messages []ai.Message) (<-chan string, <-chan *ai.LLMCallStats, <-chan error) {
	if m.chatStreamFunc != nil {
		return m.chatStreamFunc(ctx, messages)
	}
	contentChan := make(chan string, 1)
	statsChan := make(chan *ai.LLMCallStats, 1)
	errChan := make(chan error, 1)

	contentChan <- "test response"
	statsChan <- &ai.LLMCallStats{PromptTokens: 10, CompletionTokens: 5}
	close(contentChan)
	close(statsChan)
	close(errChan)

	return contentChan, statsChan, errChan
}

func (m *mockLLM) ChatWithTools(ctx context.Context, messages []ai.Message, tools []ai.ToolDescriptor) (*ai.ChatResponse, *ai.LLMCallStats, error) {
	if m.chatWithToolsFunc != nil {
		return m.chatWithToolsFunc(ctx, messages, tools)
	}
	return &ai.ChatResponse{
		Content:   "test response",
		ToolCalls: []ai.ToolCall{},
	}, &ai.LLMCallStats{PromptTokens: 10, CompletionTokens: 5}, nil
}

// Warmup is a no-op for the mock.
func (m *mockLLM) Warmup(ctx context.Context) {
	// No-op for mock
}

// mockTool is a test double for ToolWithSchema.
type mockTool struct {
	name        string
	description string
	parameters  map[string]any
	runFunc     func(ctx context.Context, input string) (string, error)
}

func (m *mockTool) Name() string        { return m.name }
func (m *mockTool) Description() string { return m.description }
func (m *mockTool) Parameters() map[string]any {
	if m.parameters == nil {
		return map[string]any{"type": "object"}
	}
	return m.parameters
}
func (m *mockTool) Run(ctx context.Context, input string) (string, error) {
	if m.runFunc != nil {
		return m.runFunc(ctx, input)
	}
	return "tool result: " + input, nil
}
