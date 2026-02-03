package ai

import (
	"testing"

	"github.com/hrygo/divinesense/internal/profile"
)

// TestNewConfigFromProfile_SiliconFlow tests SiliconFlow configuration.
func TestNewConfigFromProfile_SiliconFlow(t *testing.T) {
	prof := &profile.Profile{
		AIEnabled:            true,
		AIEmbeddingProvider:  "siliconflow",
		AIEmbeddingModel:     "BAAI/bge-m3",
		AISiliconFlowAPIKey:  "test-key",
		AISiliconFlowBaseURL: "https://api.siliconflow.cn/v1",
		AILLMProvider:        "deepseek",
		AILLMModel:           "deepseek-chat",
		AIDeepSeekAPIKey:     "deepseek-key",
		AIDeepSeekBaseURL:    "https://api.deepseek.com",
		AIRerankModel:        "BAAI/bge-reranker-v2-m3",
	}

	cfg := NewConfigFromProfile(prof)

	if !cfg.Enabled {
		t.Errorf("Expected Enabled=true, got false")
	}

	if cfg.Embedding.Provider != "siliconflow" {
		t.Errorf("Expected Embedding.Provider=siliconflow, got %s", cfg.Embedding.Provider)
	}
	if cfg.Embedding.Model != "BAAI/bge-m3" {
		t.Errorf("Expected Embedding.Model=BAAI/bge-m3, got %s", cfg.Embedding.Model)
	}
	if cfg.Embedding.APIKey != "test-key" {
		t.Errorf("Expected Embedding.APIKey=test-key, got %s", cfg.Embedding.APIKey)
	}
	if cfg.Embedding.BaseURL != "https://api.siliconflow.cn/v1" {
		t.Errorf("Expected Embedding.BaseURL=https://api.siliconflow.cn/v1, got %s", cfg.Embedding.BaseURL)
	}
	if cfg.Embedding.Dimensions != 1024 {
		t.Errorf("Expected Embedding.Dimensions=1024, got %d", cfg.Embedding.Dimensions)
	}

	// LLM config
	if cfg.LLM.Provider != "deepseek" {
		t.Errorf("Expected LLM.Provider=deepseek, got %s", cfg.LLM.Provider)
	}
	if cfg.LLM.Model != "deepseek-chat" {
		t.Errorf("Expected LLM.Model=deepseek-chat, got %s", cfg.LLM.Model)
	}
	if cfg.LLM.APIKey != "deepseek-key" {
		t.Errorf("Expected LLM.APIKey=deepseek-key, got %s", cfg.LLM.APIKey)
	}
	if cfg.LLM.MaxTokens != 2048 {
		t.Errorf("Expected LLM.MaxTokens=2048, got %d", cfg.LLM.MaxTokens)
	}
	if cfg.LLM.Temperature != 0.7 {
		t.Errorf("Expected LLM.Temperature=0.7, got %f", cfg.LLM.Temperature)
	}

	// Reranker config
	if !cfg.Reranker.Enabled {
		t.Errorf("Expected Reranker.Enabled=true, got false")
	}
	if cfg.Reranker.Provider != "siliconflow" {
		t.Errorf("Expected Reranker.Provider=siliconflow, got %s", cfg.Reranker.Provider)
	}
	if cfg.Reranker.Model != "BAAI/bge-reranker-v2-m3" {
		t.Errorf("Expected Reranker.Model=BAAI/bge-reranker-v2-m3, got %s", cfg.Reranker.Model)
	}
}

// TestNewConfigFromProfile_OpenAI tests OpenAI configuration.
func TestNewConfigFromProfile_OpenAI(t *testing.T) {
	prof := &profile.Profile{
		AIEnabled:           true,
		AIEmbeddingProvider: "openai",
		AIEmbeddingModel:    "text-embedding-3-small",
		AIOpenAIAPIKey:      "openai-key",
		AIOpenAIBaseURL:     "https://api.openai.com/v1",
		AILLMProvider:       "openai",
		AILLMModel:          "gpt-4",
	}

	cfg := NewConfigFromProfile(prof)

	if cfg.Embedding.Provider != "openai" {
		t.Errorf("Expected Embedding.Provider=openai, got %s", cfg.Embedding.Provider)
	}
	if cfg.Embedding.APIKey != "openai-key" {
		t.Errorf("Expected Embedding.APIKey=openai-key, got %s", cfg.Embedding.APIKey)
	}
	if cfg.Embedding.BaseURL != "https://api.openai.com/v1" {
		t.Errorf("Expected Embedding.BaseURL=https://api.openai.com/v1, got %s", cfg.Embedding.BaseURL)
	}

	if cfg.LLM.Provider != "openai" {
		t.Errorf("Expected LLM.Provider=openai, got %s", cfg.LLM.Provider)
	}
	if cfg.LLM.APIKey != "openai-key" {
		t.Errorf("Expected LLM.APIKey=openai-key, got %s", cfg.LLM.APIKey)
	}
}

// TestNewConfigFromProfile_Ollama tests Ollama configuration.
func TestNewConfigFromProfile_Ollama(t *testing.T) {
	prof := &profile.Profile{
		AIEnabled:           true,
		AIEmbeddingProvider: "ollama",
		AIEmbeddingModel:    "nomic-embed-text",
		AIOllamaBaseURL:     "http://localhost:11434",
		AILLMProvider:       "ollama",
		AILLMModel:          "llama2",
	}

	cfg := NewConfigFromProfile(prof)

	if cfg.Embedding.Provider != "ollama" {
		t.Errorf("Expected Embedding.Provider=ollama, got %s", cfg.Embedding.Provider)
	}
	if cfg.Embedding.BaseURL != "http://localhost:11434" {
		t.Errorf("Expected Embedding.BaseURL=http://localhost:11434, got %s", cfg.Embedding.BaseURL)
	}

	if cfg.LLM.Provider != "ollama" {
		t.Errorf("Expected LLM.Provider=ollama, got %s", cfg.LLM.Provider)
	}
}

// TestNewConfigFromProfile_Disabled tests disabled AI configuration.
func TestNewConfigFromProfile_Disabled(t *testing.T) {
	prof := &profile.Profile{
		AIEnabled: false,
	}

	cfg := NewConfigFromProfile(prof)

	if cfg.Enabled {
		t.Errorf("Expected Enabled=false, got true")
	}
}

// TestValidate tests configuration validation.
func TestValidate(t *testing.T) {
	tests := []struct {
		cfg         *Config
		name        string
		expectError bool
	}{
		{
			name: "Disabled config should pass",
			cfg: &Config{
				Enabled: false,
			},
			expectError: false,
		},
		{
			name: "Valid SiliconFlow config",
			cfg: &Config{
				Enabled: true,
				Embedding: EmbeddingConfig{
					Provider: "siliconflow",
					APIKey:   "test-key",
				},
				LLM: LLMConfig{
					Provider: "deepseek",
					APIKey:   "deepseek-key",
				},
			},
			expectError: false,
		},
		{
			name: "Valid Ollama config (no API key required)",
			cfg: &Config{
				Enabled: true,
				Embedding: EmbeddingConfig{
					Provider: "ollama",
				},
				LLM: LLMConfig{
					Provider: "ollama",
				},
			},
			expectError: false,
		},
		{
			name: "Missing embedding provider",
			cfg: &Config{
				Enabled: true,
				Embedding: EmbeddingConfig{
					Provider: "",
				},
			},
			expectError: true,
		},
		{
			name: "Missing embedding API key for non-Ollama",
			cfg: &Config{
				Enabled: true,
				Embedding: EmbeddingConfig{
					Provider: "openai",
					APIKey:   "",
				},
			},
			expectError: true,
		},
		{
			name: "Missing LLM provider",
			cfg: &Config{
				Enabled: true,
				LLM: LLMConfig{
					Provider: "",
				},
			},
			expectError: true,
		},
		{
			name: "Missing LLM API key for non-Ollama",
			cfg: &Config{
				Enabled: true,
				LLM: LLMConfig{
					Provider: "deepseek",
					APIKey:   "",
				},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if (err != nil) != tt.expectError {
				t.Errorf("Validate() error = %v, expectError %v", err, tt.expectError)
			}
		})
	}
}
