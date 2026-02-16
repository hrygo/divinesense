package ai

import (
	"errors"
	"os"

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
	Provider    string // Provider identifier for logging/future extension: zai, deepseek, openai, ollama
	Model       string // Model name: glm-4.7, deepseek-chat, gpt-4o, etc.
	APIKey      string
	BaseURL     string
	MaxTokens   int     // default: 2048
	Temperature float32 // default: 0.7
	Timeout     int     // Request timeout in seconds (default: 120)
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
	BaseURL      string // Frontend base URL for generating links in prompts
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
		APIKey:     p.AIEmbeddingAPIKey,
		BaseURL:    p.AIEmbeddingBaseURL,
		Dimensions: 1024,
	}

	// Reranker configuration
	cfg.Reranker = RerankerConfig{
		Enabled:  p.AIRerankAPIKey != "",
		Provider: p.AIRerankProvider,
		Model:    p.AIRerankModel,
		APIKey:   p.AIRerankAPIKey,
		BaseURL:  p.AIRerankBaseURL,
	}

	// LLM configuration - use unified config from profile
	cfg.LLM = LLMConfig{
		Provider:    p.ALLMProvider,
		Model:       p.ALLMModel,
		APIKey:      p.ALLMAPIKey,
		BaseURL:     p.ALLMBaseURL,
		MaxTokens:   2048,
		Temperature: 0.7,
		Timeout:     p.ALLMTimeout,
	}

	// Intent Classifier configuration
	// Default uses SiliconFlow with Qwen2.5-7B-Instruct for fast, cost-effective classification
	cfg.IntentClassifier = IntentClassifierConfig{
		Enabled: p.AIIntentAPIKey != "",
		Model:   p.AIIntentModel,
		APIKey:  p.AIIntentAPIKey,
		BaseURL: p.AIIntentBaseURL,
	}

	// UniversalParrot configuration
	// BaseURL can be set via DIVINESENSE_FRONTEND_URL env var, defaults to localhost:25173
	baseURL := os.Getenv("DIVINESENSE_FRONTEND_URL")
	if baseURL == "" {
		baseURL = "http://localhost:25173"
	}
	cfg.UniversalParrot = UniversalParrotConfig{
		Enabled:      true,
		ConfigDir:    "./config/parrots",
		FallbackMode: "legacy",
		BaseURL:      baseURL,
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
