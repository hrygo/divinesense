package profile

import (
	"os"
	"testing"
)

// TestAIProfileDefaults 测试 AI 配置的默认值。
func TestAIProfileDefaults(t *testing.T) {
	// 清除环境变量
	clearAIEnvVars()

	profile := &Profile{}
	profile.FromEnv()

	tests := []struct {
		name     string
		field    string
		expected string
		actual   string
	}{
		{"AIEnabled should be false by default", "AIEnabled", "false", boolToString(profile.AIEnabled)},
		{"AIEmbeddingProvider default", "AIEmbeddingProvider", "siliconflow", profile.AIEmbeddingProvider},
		{"ALLMProvider default", "ALLMProvider", "zai", profile.ALLMProvider},
		{"AIEmbeddingBaseURL default", "AIEmbeddingBaseURL", "https://api.siliconflow.cn/v1", profile.AIEmbeddingBaseURL},
		{"ALLMBaseURL default (Z.AI)", "ALLMBaseURL", "https://open.bigmodel.cn/api/paas/v4", profile.ALLMBaseURL},
		{"ALLMModel default (Z.AI)", "ALLMModel", "glm-4.7", profile.ALLMModel},
		{"AIRerankModel default", "AIRerankModel", "BAAI/bge-reranker-v2-m3", profile.AIRerankModel},
		{"AIEmbeddingModel default", "AIEmbeddingModel", "BAAI/bge-m3", profile.AIEmbeddingModel},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.actual != tt.expected {
				t.Errorf("%s: expected %q, got %q", tt.name, tt.expected, tt.actual)
			}
		})
	}
}

// TestAIProfileFromEnv 测试从环境变量读取统一 LLM 配置。
func TestAIProfileFromEnv(t *testing.T) {
	tests := []struct {
		name     string
		envVar   string
		envValue string
		field    func(*Profile) string
		expected string
	}{
		{
			name:     "Unified LLM API key",
			envVar:   "DIVINESENSE_AI_LLM_API_KEY",
			envValue: "test-unified-key",
			field:    func(p *Profile) string { return p.ALLMAPIKey },
			expected: "test-unified-key",
		},
		{
			name:     "Unified LLM Base URL",
			envVar:   "DIVINESENSE_AI_LLM_BASE_URL",
			envValue: "https://custom.example.com/v1",
			field:    func(p *Profile) string { return p.ALLMBaseURL },
			expected: "https://custom.example.com/v1",
		},
		{
			name:     "Unified LLM Model",
			envVar:   "DIVINESENSE_AI_LLM_MODEL",
			envValue: "custom-model",
			field:    func(p *Profile) string { return p.ALLMModel },
			expected: "custom-model",
		},
		{
			name:     "LLM provider",
			envVar:   "DIVINESENSE_AI_LLM_PROVIDER",
			envValue: "deepseek",
			field:    func(p *Profile) string { return p.ALLMProvider },
			expected: "deepseek",
		},
		{
			name:     "Embedding API key",
			envVar:   "DIVINESENSE_AI_EMBEDDING_API_KEY",
			envValue: "test-embedding-key",
			field:    func(p *Profile) string { return p.AIEmbeddingAPIKey },
			expected: "test-embedding-key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearAIEnvVars()
			os.Setenv(tt.envVar, tt.envValue)

			profile := &Profile{}
			profile.FromEnv()

			actual := tt.field(profile)
			if actual != tt.expected {
				t.Errorf("%s: expected %q, got %q", tt.name, tt.expected, actual)
			}
		})
	}
}

// TestAIProfileProviderDefaults 测试各 Provider 的默认值。
func TestAIProfileProviderDefaults(t *testing.T) {
	tests := []struct {
		name            string
		provider        string
		expectedBaseURL string
		expectedModel   string
	}{
		{
			name:            "Z.AI defaults",
			provider:        "zai",
			expectedBaseURL: "https://open.bigmodel.cn/api/paas/v4",
			expectedModel:   "glm-4.7",
		},
		{
			name:            "DeepSeek defaults",
			provider:        "deepseek",
			expectedBaseURL: "https://api.deepseek.com",
			expectedModel:   "deepseek-chat",
		},
		{
			name:            "OpenAI defaults",
			provider:        "openai",
			expectedBaseURL: "https://api.openai.com/v1",
			expectedModel:   "gpt-5.2", // Updated to match llmProviderDefaults
		},
		{
			name:            "SiliconFlow defaults",
			provider:        "siliconflow",
			expectedBaseURL: "https://api.siliconflow.cn/v1",
			expectedModel:   "Qwen/Qwen2.5-72B-Instruct",
		},
		{
			name:            "Ollama defaults",
			provider:        "ollama",
			expectedBaseURL: "http://localhost:11434",
			expectedModel:   "llama3.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearAIEnvVars()
			os.Setenv("DIVINESENSE_AI_LLM_PROVIDER", tt.provider)

			profile := &Profile{}
			profile.FromEnv()

			if profile.ALLMBaseURL != tt.expectedBaseURL {
				t.Errorf("Provider %s: expected BaseURL %q, got %q", tt.provider, tt.expectedBaseURL, profile.ALLMBaseURL)
			}
			if profile.ALLMModel != tt.expectedModel {
				t.Errorf("Provider %s: expected Model %q, got %q", tt.provider, tt.expectedModel, profile.ALLMModel)
			}
		})
	}
}

// TestIsAIEnabled 测试 IsAIEnabled 逻辑。
// Note: IsAIEnabled() now only checks if LLM API key is configured,
// not the AIEnabled field. This allows dynamic enable/disable based on config.
func TestIsAIEnabled(t *testing.T) {
	tests := []struct {
		name           string
		setupProfile   func(*Profile)
		expectedResult bool
	}{
		{
			name: "no API key returns false",
			setupProfile: func(p *Profile) {
				p.ALLMAPIKey = ""
			},
			expectedResult: false,
		},
		{
			name: "API key returns true",
			setupProfile: func(p *Profile) {
				p.ALLMAPIKey = "test-key"
			},
			expectedResult: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			profile := &Profile{}
			tt.setupProfile(profile)

			result := profile.IsAIEnabled()
			if result != tt.expectedResult {
				t.Errorf("IsAIEnabled(): expected %v, got %v", tt.expectedResult, result)
			}
		})
	}
}

// clearAIEnvVars 清除所有 AI 相关的环境变量
func clearAIEnvVars() {
	prefix := "DIVINESENSE_AI_"
	suffixes := []string{
		"ENABLED",
		"EMBEDDING_PROVIDER",
		"EMBEDDING_MODEL",
		"LLM_PROVIDER",
		"LLM_API_KEY",
		"LLM_BASE_URL",
		"LLM_MODEL",
		"SILICONFLOW_API_KEY",
		"SILICONFLOW_BASE_URL",
		"RERANK_MODEL",
	}

	for _, suffix := range suffixes {
		os.Unsetenv(prefix + suffix)
	}
}

// Helper functions
func boolToString(b bool) string {
	if b {
		return "true"
	}
	return "false"
}
