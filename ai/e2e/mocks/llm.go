//go:build e2e_manual
// +build e2e_manual

package mocks

import (
	"context"

	"github.com/hrygo/divinesense/ai/core/llm"
)

// MockLLM is a configurable mock LLM service for E2E testing.
// MockLLM 是一个可配置的 Mock LLM 服务，用于 E2E 测试。
type MockLLM struct {
	responses       map[string]string
	callStats      *llm.LLMCallStats
	defaultResponse string
}

// NewMockLLM creates a new MockLLM instance.
// NewMockLLM 创建一个新的 MockLLM 实例。
func NewMockLLM() *MockLLM {
	return &MockLLM{
		responses: make(map[string]string),
		callStats: &llm.LLMCallStats{
			PromptTokens:     100,
			CompletionTokens: 50,
			TotalTokens:      150,
		},
		defaultResponse: "Mock response",
	}
}

// WithResponse adds a preset response for a given input.
// WithResponse 为给定的输入添加预设响应。
func (m *MockLLM) WithResponse(input, output string) *MockLLM {
	m.responses[input] = output
	return m
}

// WithDefaultResponse sets the default response when no preset matches.
// WithDefaultResponse 设置没有预设匹配时的默认响应。
func (m *MockLLM) WithDefaultResponse(output string) *MockLLM {
	m.defaultResponse = output
	return m
}

// WithCallStats sets custom call statistics.
// WithCallStats 设置自定义调用统计信息。
func (m *MockLLM) WithCallStats(stats *llm.LLMCallStats) *MockLLM {
	m.callStats = stats
	return m
}

// Chat implements the llm.Service interface.
// Chat 实现 llm.Service 接口。
func (m *MockLLM) Chat(ctx context.Context, msgs []llm.Message) (string, *llm.LLMCallStats, error) {
	// Get the last user message as the key
	key := ""
	for i := len(msgs) - 1; i >= 0; i-- {
		if msgs[i].Role == "user" {
			key = msgs[i].Content
			break
		}
	}

	if response, ok := m.responses[key]; ok {
		return response, m.callStats, nil
	}

	// Default response
	return m.defaultResponse, m.callStats, nil
}

// ChatStream implements the llm.Service interface (stub).
// ChatStream 实现 llm.Service 接口（存根）。
func (m *MockLLM) ChatStream(ctx context.Context, msgs []llm.Message) (<-chan string, <-chan *llm.LLMCallStats, <-chan error) {
	contentChan := make(chan string, 1)
	statsChan := make(chan *llm.LLMCallStats, 1)
	errChan := make(chan error, 1)

	go func() {
		defer close(errChan)

		// Send content first, then close the channel
		contentChan <- m.defaultResponse
		close(contentChan)

		// Send stats, then close the channel
		stats := &llm.LLMCallStats{
			PromptTokens:       100,
			CompletionTokens:   25,
			TotalTokens:        125,
			ThinkingDurationMs: 10,
			TotalDurationMs:    50,
		}
		statsChan <- stats
		close(statsChan)
	}()

	return contentChan, statsChan, errChan
}

// ChatWithTools implements the llm.Service interface (stub).
// ChatWithTools 实现 llm.Service 接口（存根）。
func (m *MockLLM) ChatWithTools(ctx context.Context, msgs []llm.Message, tools []llm.ToolDescriptor) (*llm.ChatResponse, *llm.LLMCallStats, error) {
	// For testing, return a simple response without tool calls
	response := &llm.ChatResponse{
		Content:   "Mock tool call response",
		ToolCalls: nil,
	}
	return response, m.callStats, nil
}

// Warmup implements the llm.Service interface (no-op).
// Warmup 实现 llm.Service 接口（空操作）。
func (m *MockLLM) Warmup(ctx context.Context) {
	// No-op for mock
}

// Ensure MockLLM implements llm.Service interface.
// 确保 MockLLM 实现了 llm.Service 接口。
var _ llm.Service = (*MockLLM)(nil)
