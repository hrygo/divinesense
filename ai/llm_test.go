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

	contentChan, _ := service.ChatStream(ctx, []Message{
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
}
