package ai

import (
	"context"
	"testing"
)

// TestNewEmbeddingService tests service creation.
func TestNewEmbeddingService(t *testing.T) {
	tests := []struct {
		cfg         *EmbeddingConfig
		name        string
		expectError bool
	}{
		{
			name: "SiliconFlow config",
			cfg: &EmbeddingConfig{
				Provider:   "siliconflow",
				Model:      "BAAI/bge-m3",
				Dimensions: 1024,
				APIKey:     "test-key",
				BaseURL:    "https://api.siliconflow.cn/v1",
			},
			expectError: false,
		},
		{
			name: "OpenAI config",
			cfg: &EmbeddingConfig{
				Provider:   "openai",
				Model:      "text-embedding-3-small",
				Dimensions: 1536,
				APIKey:     "test-key",
				BaseURL:    "https://api.openai.com/v1",
			},
			expectError: false,
		},
		{
			name: "Ollama config - uses generic fallback",
			cfg: &EmbeddingConfig{
				Provider:   "ollama",
				Model:      "nomic-embed-text",
				Dimensions: 768,
				BaseURL:    "http://localhost:11434",
			},
			// NewEmbeddingService now uses generic OpenAI-compatible fallback for unknown providers
			expectError: false,
		},
		{
			name: "Unsupported provider - uses generic fallback",
			cfg: &EmbeddingConfig{
				Provider: "unsupported",
				Model:    "test-model",
				APIKey:   "test-key",
			},
			// NewEmbeddingService now uses generic OpenAI-compatible fallback for unknown providers
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewEmbeddingService(tt.cfg)
			if (err != nil) != tt.expectError {
				t.Errorf("NewEmbeddingService() error = %v, expectError %v", err, tt.expectError)
			}
		})
	}
}

// TestEmbeddingService_Dimensions tests Dimensions method.
func TestEmbeddingService_Dimensions(t *testing.T) {
	cfg := &EmbeddingConfig{
		Provider:   "siliconflow",
		Model:      "BAAI/bge-m3",
		Dimensions: 1024,
		APIKey:     "test-key",
		BaseURL:    "https://api.siliconflow.cn/v1",
	}

	service, err := NewEmbeddingService(cfg)
	if err != nil {
		t.Fatalf("NewEmbeddingService() error = %v", err)
	}

	if service.Dimensions() != 1024 {
		t.Errorf("Dimensions() = %d, want 1024", service.Dimensions())
	}
}

// TestEmbeddingService_EmptyString tests empty string input for EmbedBatch.
func TestEmbeddingService_EmptyString(t *testing.T) {
	cfg := &EmbeddingConfig{
		Provider:   "openai",
		Model:      "text-embedding-3-small",
		Dimensions: 1536,
		APIKey:     "test-key",
	}

	service, err := NewEmbeddingService(cfg)
	if err != nil {
		t.Fatalf("NewEmbeddingService() error = %v", err)
	}

	_, err = service.EmbedBatch(context.Background(), []string{})
	if err == nil {
		t.Error("Expected error for empty texts slice, got nil")
	}
}
