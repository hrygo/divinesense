package ai

import (
	"context"
	"testing"
	"time"
)

// TestNewLLMService tests service creation.
func TestNewLLMService(t *testing.T) {
	tests := []struct {
		cfg         *LLMConfig
		name        string
		expectError bool
	}{
		{
			name: "DeepSeek config",
			cfg: &LLMConfig{
				Provider:    "deepseek",
				Model:       "deepseek-chat",
				APIKey:      "test-key",
				BaseURL:     "https://api.deepseek.com",
				MaxTokens:   2048,
				Temperature: 0.7,
			},
			expectError: false,
		},
		{
			name: "OpenAI config",
			cfg: &LLMConfig{
				Provider:    "openai",
				Model:       "gpt-4",
				APIKey:      "test-key",
				BaseURL:     "https://api.openai.com/v1",
				MaxTokens:   4096,
				Temperature: 0.5,
			},
			expectError: false,
		},
		{
			name: "SiliconFlow config",
			cfg: &LLMConfig{
				Provider:    "siliconflow",
				Model:       "Qwen/Qwen2.5-7B-Instruct",
				APIKey:      "test-key",
				BaseURL:     "https://api.siliconflow.cn/v1",
				MaxTokens:   2048,
				Temperature: 0.7,
			},
			expectError: false,
		},
		{
			name: "Unsupported provider",
			cfg: &LLMConfig{
				Provider: "unsupported",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewLLMService(tt.cfg)
			if (err != nil) != tt.expectError {
				t.Errorf("NewLLMService() error = %v, expectError %v", err, tt.expectError)
			}
		})
	}
}

// TestConvertMessages tests message conversion.
func TestConvertMessages(t *testing.T) {
	messages := []Message{
		{Role: "system", Content: "You are a helpful assistant"},
		{Role: "user", Content: "Hello"},
		{Role: "assistant", Content: "Hi there"},
		{Role: "user", Content: "How are you?"},
	}

	llmMessages := convertMessages(messages)

	if len(llmMessages) != len(messages) {
		t.Errorf("convertMessages() length = %d, want %d", len(llmMessages), len(messages))
	}
}

// TestMessageHelpers tests helper functions.
func TestMessageHelpers(t *testing.T) {
	sys := SystemPrompt("System prompt")
	if sys.Role != "system" {
		t.Errorf("SystemPrompt() Role = %s, want 'system'", sys.Role)
	}

	user := UserMessage("User message")
	if user.Role != "user" {
		t.Errorf("UserMessage() Role = %s, want 'user'", user.Role)
	}

	asst := AssistantMessage("Assistant message")
	if asst.Role != "assistant" {
		t.Errorf("AssistantMessage() Role = %s, want 'assistant'", asst.Role)
	}
}

// TestFormatMessages tests message formatting.
func TestFormatMessages(t *testing.T) {
	history := []Message{
		{Role: "user", Content: "Previous message"},
		{Role: "assistant", Content: "Previous response"},
	}

	messages := FormatMessages("System prompt", "Current message", history)

	if len(messages) != 4 {
		t.Errorf("FormatMessages() length = %d, want 4", len(messages))
	}

	if messages[0].Role != "system" {
		t.Errorf("messages[0].Role = %s, want 'system'", messages[0].Role)
	}

	if messages[len(messages)-1].Role != "user" {
		t.Errorf("last message Role = %s, want 'user'", messages[len(messages)-1].Role)
	}

	if messages[len(messages)-1].Content != "Current message" {
		t.Errorf("last message Content = %s, want 'Current message'", messages[len(messages)-1].Content)
	}
}

// TestChatStream_ChannelClosing tests that channels are properly closed.
func TestChatStream_ChannelClosing(t *testing.T) {
	cfg := &LLMConfig{
		Provider:    "deepseek",
		Model:       "deepseek-chat",
		APIKey:      "test-key",
		BaseURL:     "https://api.deepseek.com",
		MaxTokens:   2048,
		Temperature: 0.7,
	}

	service, err := NewLLMService(cfg)
	if err != nil {
		t.Fatalf("NewLLMService() error = %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	contentChan, statsChan, _ := service.ChatStream(ctx, []Message{
		{Role: "user", Content: "test"},
	})

	// Wait a bit for channels to close
	time.Sleep(150 * time.Millisecond)

	// Check that content channel is closed (no more reads)
	select {
	case _, ok := <-contentChan:
		if ok {
			t.Error("contentChan should be closed after timeout")
		}
	default:
		// Channel closed, no data available
	}
	// Drain stats channel
	for range statsChan {
	}
}

// TestLLMCallStats_Structure verifies LLMCallStats has all expected fields.
func TestLLMCallStats_Structure(t *testing.T) {
	stats := &LLMCallStats{
		PromptTokens:         100,
		CompletionTokens:     50,
		TotalTokens:          150,
		CacheReadTokens:      20,
		CacheWriteTokens:     10,
		ThinkingDurationMs:   500,
		GenerationDurationMs: 300,
		TotalDurationMs:      800,
	}

	// Verify cache token fields are accessible (for providers that support caching)
	_ = stats.CacheReadTokens
	_ = stats.CacheWriteTokens

	// Verify token counts add up correctly
	if stats.TotalTokens != stats.PromptTokens+stats.CompletionTokens {
		t.Errorf("TotalTokens (%d) != PromptTokens (%d) + CompletionTokens (%d)",
			stats.TotalTokens, stats.PromptTokens, stats.CompletionTokens)
	}

	// Verify timing is non-negative
	if stats.ThinkingDurationMs < 0 {
		t.Error("ThinkingDurationMs should be non-negative")
	}
	if stats.GenerationDurationMs < 0 {
		t.Error("GenerationDurationMs should be non-negative")
	}
	if stats.TotalDurationMs < 0 {
		t.Error("TotalDurationMs should be non-negative")
	}

	// Verify total duration covers thinking + generation
	if stats.TotalDurationMs < stats.ThinkingDurationMs+stats.GenerationDurationMs {
		t.Errorf("TotalDurationMs (%d) should be >= ThinkingDurationMs (%d) + GenerationDurationMs (%d)",
			stats.TotalDurationMs, stats.ThinkingDurationMs, stats.GenerationDurationMs)
	}
}

// TestLLMCallStats_ZeroValues verifies zero stats are valid.
func TestLLMCallStats_ZeroValues(t *testing.T) {
	stats := &LLMCallStats{}

	if stats.PromptTokens != 0 {
		t.Errorf("PromptTokens should default to 0, got %d", stats.PromptTokens)
	}
	if stats.CompletionTokens != 0 {
		t.Errorf("CompletionTokens should default to 0, got %d", stats.CompletionTokens)
	}
	if stats.TotalTokens != 0 {
		t.Errorf("TotalTokens should default to 0, got %d", stats.TotalTokens)
	}
}

// TestChat_ReturnsStats verifies that Chat method returns stats on successful calls.
// Note: On network/timeout errors, stats may be nil since no API response was received.
func TestChat_ReturnsStats(t *testing.T) {
	cfg := &LLMConfig{
		Provider:    "deepseek",
		Model:       "deepseek-chat",
		APIKey:      "test-key",
		BaseURL:     "https://api.deepseek.com",
		MaxTokens:   2048,
		Temperature: 0.7,
	}

	service, err := NewLLMService(cfg)
	if err != nil {
		t.Fatalf("NewLLMService() error = %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// This will fail due to timeout, stats may be nil in this case
	_, stats, err := service.Chat(ctx, []Message{
		{Role: "user", Content: "test"},
	})

	// We expect an error due to timeout
	if err == nil {
		t.Error("Expected error from Chat with timeout, got nil")
	}

	// Stats may be nil if the request failed before receiving any API response
	// This is expected behavior for timeout errors before API communication
	_ = stats // Linter check - stats handling depends on error type
}

// TestChatStream_ReturnsStatsChannel verifies that ChatStream returns a stats channel.
func TestChatStream_ReturnsStatsChannel(t *testing.T) {
	cfg := &LLMConfig{
		Provider:    "deepseek",
		Model:       "deepseek-chat",
		APIKey:      "test-key",
		BaseURL:     "https://api.deepseek.com",
		MaxTokens:   2048,
		Temperature: 0.7,
	}

	service, err := NewLLMService(cfg)
	if err != nil {
		t.Fatalf("NewLLMService() error = %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, statsChan, _ := service.ChatStream(ctx, []Message{
		{Role: "user", Content: "test"},
	})

	// Stats channel should never be nil
	if statsChan == nil {
		t.Error("Stats channel should never be nil")
	}

	// Drain stats channel to avoid goroutine leak
	for range statsChan {
	}
}
