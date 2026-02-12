package llm

import (
	"context"
	"testing"
)

func TestNewService_UnsupportedProvider(t *testing.T) {
	cfg := &Config{
		Provider: "unsupported",
		Model:    "test-model",
	}

	_, err := NewService(cfg)
	if err == nil {
		t.Error("NewService() with unsupported provider should return error")
	}
}

func TestNewService_DeepSeekDefaults(t *testing.T) {
	cfg := &Config{
		Provider: "deepseek",
		APIKey:   "test-key",
	}

	svc, err := NewService(cfg)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	if svc == nil {
		t.Fatal("NewService() returned nil service")
	}
}

func TestNewService_OpenAI(t *testing.T) {
	cfg := &Config{
		Provider:    "openai",
		Model:       "gpt-4",
		APIKey:      "test-key",
		BaseURL:     "https://api.openai.com/v1",
		MaxTokens:   4096,
		Temperature: 0.5,
	}

	svc, err := NewService(cfg)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	if svc == nil {
		t.Fatal("NewService() returned nil service")
	}
}

func TestNewService_SiliconFlowDefaults(t *testing.T) {
	cfg := &Config{
		Provider: "siliconflow",
		APIKey:   "test-key",
	}

	svc, err := NewService(cfg)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	if svc == nil {
		t.Fatal("NewService() returned nil service")
	}
}

func TestConfig_DefaultValues(t *testing.T) {
	cfg := &Config{
		Provider:    "deepseek",
		APIKey:      "test-key",
		MaxTokens:   2048,
		Temperature: 0.7,
	}

	svc, err := NewService(cfg)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	s, ok := svc.(*service)
	if !ok {
		t.Fatal("NewService() did not return *service type")
	}

	if s.maxTokens != 2048 {
		t.Errorf("maxTokens = %v, want 2048", s.maxTokens)
	}
	if s.temperature != 0.7 {
		t.Errorf("temperature = %v, want 0.7", s.temperature)
	}
}

func TestMessage(t *testing.T) {
	msg := Message{
		Role:    "user",
		Content: "test content",
	}

	if msg.Role != "user" {
		t.Errorf("Role = %v, want user", msg.Role)
	}
	if msg.Content != "test content" {
		t.Errorf("Content = %v, want test content", msg.Content)
	}
}

func TestLLMCallStats(t *testing.T) {
	stats := &LLMCallStats{
		PromptTokens:         100,
		CompletionTokens:     50,
		TotalTokens:          150, // Set explicitly (not auto-calculated)
		CacheReadTokens:      30,
		CacheWriteTokens:     70,
		ThinkingDurationMs:   500,
		GenerationDurationMs: 300,
		TotalDurationMs:      800,
	}

	if stats.PromptTokens != 100 {
		t.Errorf("PromptTokens = %v, want 100", stats.PromptTokens)
	}
	if stats.CompletionTokens != 50 {
		t.Errorf("CompletionTokens = %v, want 50", stats.CompletionTokens)
	}
	if stats.TotalTokens != 150 {
		t.Errorf("TotalTokens = %v, want 150", stats.TotalTokens)
	}
	if stats.CacheReadTokens != 30 {
		t.Errorf("CacheReadTokens = %v, want 30", stats.CacheReadTokens)
	}
	if stats.CacheWriteTokens != 70 {
		t.Errorf("CacheWriteTokens = %v, want 70", stats.CacheWriteTokens)
	}
	if stats.ThinkingDurationMs != 500 {
		t.Errorf("ThinkingDurationMs = %v, want 500", stats.ThinkingDurationMs)
	}
	if stats.GenerationDurationMs != 300 {
		t.Errorf("GenerationDurationMs = %v, want 300", stats.GenerationDurationMs)
	}
	if stats.TotalDurationMs != 800 {
		t.Errorf("TotalDurationMs = %v, want 800", stats.TotalDurationMs)
	}
}

func TestToolDescriptor(t *testing.T) {
	tool := ToolDescriptor{
		Name:        "search",
		Description: "Search the database",
		Parameters:  `{"type":"object"}`,
	}

	if tool.Name != "search" {
		t.Errorf("Name = %v, want search", tool.Name)
	}
	if tool.Description != "Search the database" {
		t.Errorf("Description = %v, want Search the database", tool.Description)
	}
	if tool.Parameters != `{"type":"object"}` {
		t.Errorf("Parameters = %v, want {\"type\":\"object\"}", tool.Parameters)
	}
}

func TestChatResponse(t *testing.T) {
	resp := &ChatResponse{
		Content: "Test response",
		ToolCalls: []ToolCall{
			{
				ID:   "call_123",
				Type: "function",
				Function: FunctionCall{
					Name:      "search",
					Arguments: `{"query":"test"}`,
				},
			},
		},
	}

	if resp.Content != "Test response" {
		t.Errorf("Content = %v, want Test response", resp.Content)
	}
	if len(resp.ToolCalls) != 1 {
		t.Fatalf("ToolCalls length = %v, want 1", len(resp.ToolCalls))
	}
	if resp.ToolCalls[0].ID != "call_123" {
		t.Errorf("ToolCalls[0].ID = %v, want call_123", resp.ToolCalls[0].ID)
	}
	if resp.ToolCalls[0].Function.Name != "search" {
		t.Errorf("Function.Name = %v, want search", resp.ToolCalls[0].Function.Name)
	}
}

func TestToolCall(t *testing.T) {
	tc := ToolCall{
		ID:   "call_abc",
		Type: "function",
		Function: FunctionCall{
			Name:      "get_weather",
			Arguments: `{"location":"NYC"}`,
		},
	}

	if tc.ID != "call_abc" {
		t.Errorf("ID = %v, want call_abc", tc.ID)
	}
	if tc.Type != "function" {
		t.Errorf("Type = %v, want function", tc.Type)
	}
	if tc.Function.Name != "get_weather" {
		t.Errorf("Function.Name = %v, want get_weather", tc.Function.Name)
	}
	if tc.Function.Arguments != `{"location":"NYC"}` {
		t.Errorf("Function.Arguments = %v, want {\"location\":\"NYC\"}", tc.Function.Arguments)
	}
}

func TestService_Warmup_NoPanic(t *testing.T) {
	cfg := &Config{
		Provider: "deepseek",
		APIKey:   "test-key",
	}

	svc, err := NewService(cfg)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	// Warmup should not panic (will fail with network error but that's OK)
	svc.Warmup(context.Background())
}

func TestService_Chat_NoPanic(t *testing.T) {
	cfg := &Config{
		Provider: "deepseek",
		APIKey:   "test-key",
	}

	svc, err := NewService(cfg)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	messages := []Message{
		{Role: "user", Content: "test"},
	}

	// Chat should not panic (will fail with network error but that's OK for unit test)
	_, _, _ = svc.Chat(context.Background(), messages)
}

func TestService_ChatStream_NoPanic(t *testing.T) {
	cfg := &Config{
		Provider: "deepseek",
		APIKey:   "test-key",
	}

	svc, err := NewService(cfg)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	messages := []Message{
		{Role: "user", Content: "test"},
	}

	// ChatStream should not panic
	streamCh, statsCh, errCh := svc.ChatStream(context.Background(), messages)

	if streamCh == nil {
		t.Error("ChatStream() returned nil stream channel")
	}
	if statsCh == nil {
		t.Error("ChatStream() returned nil stats channel")
	}
	if errCh == nil {
		t.Error("ChatStream() returned nil error channel")
	}
}

func TestService_ChatWithTools_NoPanic(t *testing.T) {
	cfg := &Config{
		Provider: "deepseek",
		APIKey:   "test-key",
	}

	svc, err := NewService(cfg)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	messages := []Message{
		{Role: "user", Content: "test"},
	}
	tools := []ToolDescriptor{
		{
			Name:        "search",
			Description: "Search",
			Parameters:  `{}`,
		},
	}

	// ChatWithTools should not panic
	_, _, _ = svc.ChatWithTools(context.Background(), messages, tools)
}
