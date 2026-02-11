package embedding

import (
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.BaseURL != "https://api.openai.com/v1" {
		t.Errorf("BaseURL = %v, want https://api.openai.com/v1", cfg.BaseURL)
	}
	if cfg.EmbeddingModel != "text-embedding-3-small" {
		t.Errorf("EmbeddingModel = %v, want text-embedding-3-small", cfg.EmbeddingModel)
	}
	if cfg.ChatModel != "gpt-4o-mini" {
		t.Errorf("ChatModel = %v, want gpt-4o-mini", cfg.ChatModel)
	}
	if cfg.MaxRetries != 3 {
		t.Errorf("MaxRetries = %v, want 3", cfg.MaxRetries)
	}
	if cfg.Timeout != 30*time.Second {
		t.Errorf("Timeout = %v, want 30s", cfg.Timeout)
	}
}

func TestNewProvider(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *Config
		wantErr bool
	}{
		{
			name: "valid config",
			cfg: &Config{
				BaseURL:        "https://api.openai.com/v1",
				APIKey:         "test-key",
				EmbeddingModel: "text-embedding-3-small",
				ChatModel:      "gpt-4o-mini",
				MaxRetries:     3,
				Timeout:        30 * time.Second,
			},
			wantErr: false,
		},
		{
			name:    "nil config uses defaults",
			cfg:     nil,
			wantErr: false,
		},
		{
			name: "zero values are filled with defaults",
			cfg: &Config{
				BaseURL: "https://api.test.com",
				APIKey:  "test-key",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := NewProvider(tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewProvider() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if p == nil {
					t.Error("NewProvider() returned nil provider")
				}
				if p.config == nil {
					t.Error("NewProvider() returned provider with nil config")
				}
			}
		})
	}
}

func TestNewProvider_DefaultValueFilling(t *testing.T) {
	cfg := &Config{
		BaseURL: "https://api.test.com",
		APIKey:  "test-key",
		// Leave other fields as zero
	}

	p, err := NewProvider(cfg)
	if err != nil {
		t.Fatalf("NewProvider() error = %v", err)
	}

	// Check that defaults were applied
	if p.config.MaxRetries != 3 {
		t.Errorf("MaxRetries = %v, want 3", p.config.MaxRetries)
	}
	if p.config.Timeout != 30*time.Second {
		t.Errorf("Timeout = %v, want 30s", p.config.Timeout)
	}
	if p.config.EmbeddingModel != "text-embedding-3-small" {
		t.Errorf("EmbeddingModel = %v, want text-embedding-3-small", p.config.EmbeddingModel)
	}
	if p.config.ChatModel != "gpt-4o-mini" {
		t.Errorf("ChatModel = %v, want gpt-4o-mini", p.config.ChatModel)
	}
}

func TestProvider_Validate(t *testing.T) {
	tests := []struct {
		name        string
		apiKey      string
		wantErr     bool
		errContains string
	}{
		{
			name:        "missing api key",
			apiKey:      "",
			wantErr:     true,
			errContains: "API key is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, _ := NewProvider(&Config{
				BaseURL: "https://api.test.com",
				APIKey:  tt.apiKey,
			})

			err := p.Validate(nil)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && tt.errContains != "" {
				if !contains(err.Error(), tt.errContains) {
					t.Errorf("Validate() error = %v, should contain %q", err, tt.errContains)
				}
			}
		})
	}
}

func TestNewProviderFromEnv(t *testing.T) {
	// This test verifies that NewProviderFromEnv doesn't panic
	// and returns a provider with defaults when env vars are not set
	p, err := NewProviderFromEnv()
	if err != nil {
		t.Fatalf("NewProviderFromEnv() error = %v", err)
	}
	if p == nil {
		t.Fatal("NewProviderFromEnv() returned nil")
	}
	if p.config == nil {
		t.Fatal("NewProviderFromEnv() returned provider with nil config")
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

func TestListModels(t *testing.T) {
	cfg := &Config{
		BaseURL:        "https://api.test.com",
		APIKey:         "test-key",
		EmbeddingModel: "custom-embedding-model",
		ChatModel:      "custom-chat-model",
	}

	p, _ := NewProvider(cfg)
	models, err := p.ListModels(nil)

	if err != nil {
		t.Fatalf("ListModels() error = %v", err)
	}

	if len(models) != 2 {
		t.Errorf("ListModels() returned %d models, want 2", len(models))
	}

	expectedModels := []string{"custom-embedding-model", "custom-chat-model"}
	for i, model := range models {
		if model != expectedModels[i] {
			t.Errorf("models[%d] = %v, want %v", i, model, expectedModels[i])
		}
	}
}

func TestGetEnv(t *testing.T) {
	// Test with non-existent env var (should return fallback)
	result := getEnv("NON_EXISTENT_ENV_VAR_12345", "fallback")
	if result != "fallback" {
		t.Errorf("getEnv() = %v, want fallback", result)
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsInString(s, substr))
}

func containsInString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
