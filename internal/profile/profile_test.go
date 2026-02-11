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
		{"AILLMProvider default", "AILLMProvider", "zai", profile.AILLMProvider},
		{"AISiliconFlowBaseURL default", "AISiliconFlowBaseURL", "https://api.siliconflow.cn/v1", profile.AISiliconFlowBaseURL},
		{"AIZAI_BaseURL default", "AIZAI_BaseURL", "https://open.bigmodel.cn/api/paas/v4", profile.AIZAIBaseURL},
		{"AIZAI_APIKey default", "AIZAI_APIKey", "", profile.AIZAI_APIKey},
		{"AIRerankModel default", "AIRerankModel", "BAAI/bge-reranker-v2-m3", profile.AIRerankModel},
		{"AIEmbeddingModel default", "AIEmbeddingModel", "BAAI/bge-m3", profile.AIEmbeddingModel},
		{"AILLMModel default", "AILLMModel", "glm-4.7", profile.AILLMModel},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.actual != tt.expected {
				t.Errorf("%s: expected %q, got %q", tt.name, tt.expected, tt.actual)
			}
		})
	}
}

// TestAIProfileFromEnv 测试从环境变量读取 AI 配置。
func TestAIProfileFromEnv(t *testing.T) {
	tests := []struct {
		name     string
		envVar   string
		envValue string
		field    func(*Profile) string
		expected string
	}{
		{
			name:     "Z.AI enabled with API key",
			envVar:   "DIVINESENSE_AI_ZAI_API_KEY",
			envValue: "test-zai-key",
			field:    func(p *Profile) string { return p.AIZAI_APIKey },
			expected: "test-zai-key",
		},
		{
			name:     "Z.AI Base URL",
			envVar:   "DIVINESENSE_AI_ZAI_BASE_URL",
			envValue: "https://open.bigmodel.cn/api/paas/v4",
			field:    func(p *Profile) string { return p.AIZAIBaseURL },
			expected: "https://open.bigmodel.cn/api/paas/v4",
		},
		{
			name:     "SiliconFlow API key",
			envVar:   "DIVINESENSE_AI_SILICONFLOW_API_KEY",
			envValue: "test-siliconflow-key",
			field:    func(p *Profile) string { return p.AISiliconFlowAPIKey },
			expected: "test-siliconflow-key",
		},
		{
			name:     "LLM provider is Z.AI",
			envVar:   "DIVINESENSE_AI_LLM_PROVIDER",
			envValue: "zai",
			field:    func(p *Profile) string { return p.AILLMProvider },
			expected: "zai",
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

// TestIsAIEnabled 测试 IsAIEnabled 逻辑。
func TestIsAIEnabled(t *testing.T) {
	tests := []struct {
		name           string
		setupProfile   func(*Profile)
		expectedResult bool
	}{
		{
			name: "no API keys returns false",
			setupProfile: func(p *Profile) {
				p.AIZAI_APIKey = ""
				p.AISiliconFlowAPIKey = ""
				p.AIDeepSeekAPIKey = ""
				p.AIOpenAIAPIKey = ""
			},
			expectedResult: false,
		},
		{
			name: "Z.AI API key returns true (with AIEnabled=true)",
			setupProfile: func(p *Profile) {
				p.AIEnabled = true
				p.AIZAI_APIKey = "test-zai-key"
			},
			expectedResult: true,
		},
		{
			name: "SiliconFlow API key returns true (with AIEnabled=true)",
			setupProfile: func(p *Profile) {
				p.AIEnabled = true
				p.AISiliconFlowAPIKey = "test-siliconflow-key"
			},
			expectedResult: true,
		},
		{
			name: "DeepSeek API key returns true (with AIEnabled=true)",
			setupProfile: func(p *Profile) {
				p.AIEnabled = true
				p.AIDeepSeekAPIKey = "test-deepseek-key"
			},
			expectedResult: true,
		},
		{
			name: "OpenAI API key returns true (with AIEnabled=true)",
			setupProfile: func(p *Profile) {
				p.AIEnabled = true
				p.AIOpenAIAPIKey = "test-openai-key"
			},
			expectedResult: true,
		},
		{
			name: "API key without AIEnabled=false returns false",
			setupProfile: func(p *Profile) {
				p.AIZAI_APIKey = "test-zai-key"
			},
			expectedResult: false,
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
		"LLM_MODEL",
		"ZAI_API_KEY",
		"ZAI_BASE_URL",
		"SILICONFLOW_API_KEY",
		"SILICONFLOW_BASE_URL",
		"DEEPSEEK_API_KEY",
		"DEEPSEEK_BASE_URL",
		"OPENAI_API_KEY",
		"OPENAI_BASE_URL",
		"OLLAMA_BASE_URL",
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
