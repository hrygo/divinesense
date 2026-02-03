package ai

import (
	"errors"

	"github.com/hrygo/divinesense/internal/profile"
)

// Config represents AI configuration.
type Config struct {
	Embedding        EmbeddingConfig
	Reranker         RerankerConfig
	IntentClassifier IntentClassifierConfig
	LLM              LLMConfig
	Enabled          bool
}

// EmbeddingConfig represents vector embedding configuration.
type EmbeddingConfig struct {
	Provider   string
	Model      string
	APIKey     string
	BaseURL    string
	Dimensions int
}

// RerankerConfig represents reranker configuration.
type RerankerConfig struct {
	Provider string
	Model    string
	APIKey   string
	BaseURL  string
	Enabled  bool
}

// LLMConfig represents LLM configuration.
type LLMConfig struct {
	Provider    string // deepseek, openai, ollama
	Model       string // deepseek-chat
	APIKey      string
	BaseURL     string
	MaxTokens   int     // default: 2048
	Temperature float32 // default: 0.7
}

// IntentClassifierConfig represents intent classification LLM configuration.
// Uses a lightweight model for fast, cost-effective classification.
type IntentClassifierConfig struct {
	Model   string
	APIKey  string
	BaseURL string
	Enabled bool
}

// NewConfigFromProfile creates AI config from profile.
func NewConfigFromProfile(p *profile.Profile) *Config {
	cfg := &Config{
		Enabled: p.AIEnabled,
	}

	if !cfg.Enabled {
		return cfg
	}

	// Embedding configuration
	cfg.Embedding = EmbeddingConfig{
		Provider:   p.AIEmbeddingProvider,
		Model:      p.AIEmbeddingModel,
		Dimensions: 1024,
	}

	switch p.AIEmbeddingProvider {
	case "siliconflow":
		cfg.Embedding.APIKey = p.AISiliconFlowAPIKey
		cfg.Embedding.BaseURL = p.AISiliconFlowBaseURL
	case "openai":
		cfg.Embedding.APIKey = p.AIOpenAIAPIKey
		cfg.Embedding.BaseURL = p.AIOpenAIBaseURL
	case "ollama":
		cfg.Embedding.BaseURL = p.AIOllamaBaseURL
	}

	// Reranker configuration
	cfg.Reranker = RerankerConfig{
		Enabled:  p.AISiliconFlowAPIKey != "",
		Provider: "siliconflow",
		Model:    p.AIRerankModel,
		APIKey:   p.AISiliconFlowAPIKey,
		BaseURL:  p.AISiliconFlowBaseURL,
	}

	// LLM configuration
	cfg.LLM = LLMConfig{
		Provider:    p.AILLMProvider,
		Model:       p.AILLMModel,
		MaxTokens:   2048,
		Temperature: 0.7,
	}

	switch p.AILLMProvider {
	case "deepseek":
		cfg.LLM.APIKey = p.AIDeepSeekAPIKey
		cfg.LLM.BaseURL = p.AIDeepSeekBaseURL
	case "openai":
		cfg.LLM.APIKey = p.AIOpenAIAPIKey
		cfg.LLM.BaseURL = p.AIOpenAIBaseURL
	case "ollama":
		cfg.LLM.BaseURL = p.AIOllamaBaseURL
	}

	// Intent Classifier configuration
	// Uses SiliconFlow with a lightweight model for fast classification
	cfg.IntentClassifier = IntentClassifierConfig{
		Enabled: p.AISiliconFlowAPIKey != "",
		Model:   "Qwen/Qwen2.5-7B-Instruct", // Fast, cost-effective
		APIKey:  p.AISiliconFlowAPIKey,
		BaseURL: p.AISiliconFlowBaseURL,
	}

	return cfg
}

// Validate validates the configuration.
func (c *Config) Validate() error {
	if !c.Enabled {
		return nil
	}

	if c.Embedding.Provider == "" {
		return errors.New("embedding provider is required")
	}

	if c.Embedding.Provider != "ollama" && c.Embedding.APIKey == "" {
		return errors.New("embedding API key is required")
	}

	if c.LLM.Provider == "" {
		return errors.New("LLM provider is required")
	}

	if c.LLM.Provider != "ollama" && c.LLM.APIKey == "" {
		return errors.New("LLM API key is required")
	}

	return nil
}
