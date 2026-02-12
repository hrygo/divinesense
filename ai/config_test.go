package ai

import (
	"testing"

	"github.com/hrygo/divinesense/internal/profile"
)

// TestNewConfigFromProfile_SiliconFlow tests SiliconFlow configuration.
func TestNewConfigFromProfile_SiliconFlow(t *testing.T) {
	prof := &profile.Profile{
		AIEnabled:           true,
		AIEmbeddingProvider: "siliconflow",
		AIEmbeddingModel:    "BAAI/bge-m3",
		AIEmbeddingAPIKey:   "test-key",
		AIEmbeddingBaseURL:  "https://api.siliconflow.cn/v1",
		ALLMProvider:        "deepseek",
		ALLMAPIKey:          "deepseek-key",
		ALLMBaseURL:         "https://api.deepseek.com",
		ALLMModel:           "deepseek-chat",
		AIRerankModel:       "BAAI/bge-reranker-v2-m3",
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
	if cfg.LLM.BaseURL != "https://api.deepseek.com" {
		t.Errorf("Expected LLM.BaseURL=https://api.deepseek.com, got %s", cfg.LLM.BaseURL)
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

// TestNewConfigFromProfile_UnifiedLLM tests unified LLM configuration.
func TestNewConfigFromProfile_UnifiedLLM(t *testing.T) {
	tests := []struct {
		name        string
		prof        *profile.Profile
		expectAPI   string
		expectBase  string
		expectModel string
	}{
		{
			name: "Z.AI configuration",
			prof: &profile.Profile{
				AIEnabled:    true,
				ALLMProvider: "zai",
				ALLMAPIKey:   "zai-key",
				ALLMBaseURL:  "https://open.bigmodel.cn/api/paas/v4",
				ALLMModel:    "glm-4.7",
			},
			expectAPI:   "zai-key",
			expectBase:  "https://open.bigmodel.cn/api/paas/v4",
			expectModel: "glm-4.7",
		},
		{
			name: "DeepSeek configuration",
			prof: &profile.Profile{
				AIEnabled:    true,
				ALLMProvider: "deepseek",
				ALLMAPIKey:   "deepseek-key",
				ALLMBaseURL:  "https://api.deepseek.com",
				ALLMModel:    "deepseek-chat",
			},
			expectAPI:   "deepseek-key",
			expectBase:  "https://api.deepseek.com",
			expectModel: "deepseek-chat",
		},
		{
			name: "OpenAI configuration",
			prof: &profile.Profile{
				AIEnabled:    true,
				ALLMProvider: "openai",
				ALLMAPIKey:   "openai-key",
				ALLMBaseURL:  "https://api.openai.com/v1",
				ALLMModel:    "gpt-4o",
			},
			expectAPI:   "openai-key",
			expectBase:  "https://api.openai.com/v1",
			expectModel: "gpt-4o",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := NewConfigFromProfile(tt.prof)

			if cfg.LLM.Provider != tt.prof.ALLMProvider {
				t.Errorf("Expected LLM.Provider=%s, got %s", tt.prof.ALLMProvider, cfg.LLM.Provider)
			}
			if cfg.LLM.APIKey != tt.expectAPI {
				t.Errorf("Expected LLM.APIKey=%s, got %s", tt.expectAPI, cfg.LLM.APIKey)
			}
			if cfg.LLM.BaseURL != tt.expectBase {
				t.Errorf("Expected LLM.BaseURL=%s, got %s", tt.expectBase, cfg.LLM.BaseURL)
			}
			if cfg.LLM.Model != tt.expectModel {
				t.Errorf("Expected LLM.Model=%s, got %s", tt.expectModel, cfg.LLM.Model)
			}
		})
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
			name: "Valid unified LLM config",
			cfg: &Config{
				Enabled: true,
				Embedding: EmbeddingConfig{
					Provider: "siliconflow",
					APIKey:   "test-key",
				},
				LLM: LLMConfig{
					Provider: "zai",
					APIKey:   "zai-key",
				},
			},
			expectError: false,
		},
		{
			name: "Valid config with siliconflow embedding and ollama LLM",
			cfg: &Config{
				Enabled: true,
				Embedding: EmbeddingConfig{
					Provider: "siliconflow",
					APIKey:   "test-key",
				},
				LLM: LLMConfig{
					Provider: "ollama",
					APIKey:   "dummy-key",
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
