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
	UniversalParrot  UniversalParrotConfig // Phase 2: Configuration-driven parrots
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
	Provider    string // deepseek, openai, ollama, anthropic
	Model       string // deepseek-chat, claude-3-5-sonnet-20241022
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

// UniversalParrotConfig represents configuration for UniversalParrot (configuration-driven parrots).
type UniversalParrotConfig struct {
	Enabled      bool   // Enable UniversalParrot for creating parrots from YAML configs
	ConfigDir    string // Path to parrot YAML configs (default: ./config/parrots)
	FallbackMode string // "legacy" | "error" when config load fails (default: legacy)
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
	// Note: Model alias resolution is handled by the user setting the full model name
	// in DIVINESENSE_AI_LLM_MODEL environment variable.
	// Common Anthropic models:
	// - claude-3-5-sonnet-20241022 (default, recommended)
	// - claude-3-5-opus-20241022 (high complexity)
	// - claude-3-5-haiku-20241022 (fast, low cost)
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
	case "anthropic":
		cfg.LLM.APIKey = p.AIAnthropicAPIKey
		cfg.LLM.BaseURL = p.AIAnthropicBaseURL
	case "ollama":
		cfg.LLM.BaseURL = p.AIOllamaBaseURL
	}

	// Intent Classifier configuration
	// Uses SiliconFlow with Qwen2.5-7B-Instruct for fast, cost-effective classification
	// This config is used by routerIntentLLMClient in ai_service.go
	cfg.IntentClassifier = IntentClassifierConfig{
		Enabled: p.AISiliconFlowAPIKey != "",
		Model:   "Qwen/Qwen2.5-7B-Instruct",
		APIKey:  p.AISiliconFlowAPIKey,
		BaseURL: p.AISiliconFlowBaseURL,
	}

	// UniversalParrot configuration
	// Enable configuration-driven parrot system by default when AI is enabled
	cfg.UniversalParrot = UniversalParrotConfig{
		Enabled:      true,
		ConfigDir:    "./config/parrots",
		FallbackMode: "legacy",
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
