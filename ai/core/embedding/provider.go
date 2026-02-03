package embedding

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"os"
	"time"

	"github.com/sashabaranov/go-openai"
)

// Config holds the AI provider configuration.
type Config struct {
	BaseURL        string
	APIKey         string
	EmbeddingModel string
	ChatModel      string
	MaxRetries     int
	Timeout        time.Duration
}

// DefaultConfig returns the default configuration.
func DefaultConfig() *Config {
	return &Config{
		BaseURL:        "https://api.openai.com/v1",
		APIKey:         "",
		EmbeddingModel: "text-embedding-3-small",
		ChatModel:      "gpt-4o-mini",
		MaxRetries:     3,
		Timeout:        30 * time.Second,
	}
}

// Provider provides AI capabilities including LLM and Embedding.
type Provider struct {
	client *openai.Client
	config *Config
}

// NewProvider creates a new AI provider.
func NewProvider(cfg *Config) (*Provider, error) {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	// Apply defaults for unset values
	if cfg.MaxRetries == 0 {
		cfg.MaxRetries = 3
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 30 * time.Second
	}
	if cfg.EmbeddingModel == "" {
		cfg.EmbeddingModel = "text-embedding-3-small"
	}
	if cfg.ChatModel == "" {
		cfg.ChatModel = "gpt-4o-mini"
	}

	clientConfig := openai.DefaultConfig(cfg.APIKey)
	if cfg.BaseURL != "" {
		clientConfig.BaseURL = cfg.BaseURL
	}

	client := openai.NewClientWithConfig(clientConfig)

	return &Provider{
		client: client,
		config: cfg,
	}, nil
}

// Embedding generates an embedding vector for the given text.
func (p *Provider) Embedding(ctx context.Context, text string) ([]float32, error) {
	var result []float32
	err := p.doWithRetry(ctx, func() error {
		req := openai.EmbeddingRequest{
			Input: []string{text},
			Model: openai.EmbeddingModel(p.config.EmbeddingModel),
		}

		resp, err := p.client.CreateEmbeddings(ctx, req)
		if err != nil {
			return err
		}
		if len(resp.Data) == 0 {
			return fmt.Errorf("empty embedding response")
		}
		result = resp.Data[0].Embedding
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to generate embedding: %w", err)
	}

	return result, nil
}

// Chat performs a chat completion.
func (p *Provider) Chat(ctx context.Context, messages []Message) (string, error) {
	var result string
	err := p.doWithRetry(ctx, func() error {
		// Convert messages to openai format
		llmMessages := make([]openai.ChatCompletionMessage, len(messages))
		for i, msg := range messages {
			llmMessages[i] = openai.ChatCompletionMessage{
				Role:    msg.Role,
				Content: msg.Content,
			}
		}

		req := openai.ChatCompletionRequest{
			Model:    p.config.ChatModel,
			Messages: llmMessages,
		}

		resp, err := p.client.CreateChatCompletion(ctx, req)
		if err != nil {
			return err
		}
		if len(resp.Choices) == 0 {
			return fmt.Errorf("empty chat response")
		}
		result = resp.Choices[0].Message.Content
		return nil
	})

	if err != nil {
		return "", fmt.Errorf("failed to complete chat: %w", err)
	}

	return result, nil
}

// Message represents a chat message.
type Message struct {
	Role    string
	Content string
}

// ChatStream performs a streaming chat completion.
func (p *Provider) ChatStream(ctx context.Context, messages []Message) (string, error) {
	// For simplicity, fall back to non-streaming for now
	return p.Chat(ctx, messages)
}

// ListModels lists available models from the provider.
func (p *Provider) ListModels(ctx context.Context) ([]string, error) {
	return []string{
		p.config.EmbeddingModel,
		p.config.ChatModel,
	}, nil
}

// Validate validates the provider configuration by testing API connectivity.
func (p *Provider) Validate(ctx context.Context) error {
	if p.config.APIKey == "" {
		return fmt.Errorf("API key is required, set MEMOS_AI_API_KEY environment variable")
	}

	// Test embedding generation with a simple request
	_, err := p.Embedding(ctx, "test")
	if err != nil {
		return fmt.Errorf("embedding validation failed: %w", err)
	}

	slog.Info("AI provider validated successfully",
		"embedding_model", p.config.EmbeddingModel,
		"chat_model", p.config.ChatModel)

	return nil
}

// doWithRetry executes a function with exponential backoff retry.
func (p *Provider) doWithRetry(ctx context.Context, fn func() error) error {
	var lastErr error
	for attempt := 0; attempt < p.config.MaxRetries; attempt++ {
		if err := fn(); err == nil {
			return nil
		} else {
			lastErr = err
			if attempt < p.config.MaxRetries-1 {
				waitTime := time.Duration(math.Pow(2, float64(attempt))) * time.Second
				slog.Debug("AI request failed, retrying",
					"attempt", attempt+1,
					"wait_time", waitTime,
					"error", err)
				select {
				case <-time.After(waitTime):
				case <-ctx.Done():
					return ctx.Err()
				}
			}
		}
	}
	return lastErr
}

// NewProviderFromEnv creates a provider from environment variables.
func NewProviderFromEnv() (*Provider, error) {
	return NewProvider(&Config{
		BaseURL:        getEnv("MEMOS_AI_BASE_URL", "https://api.openai.com/v1"),
		APIKey:         getEnv("MEMOS_AI_API_KEY", ""),
		EmbeddingModel: getEnv("MEMOS_AI_EMBEDDING_MODEL", "text-embedding-3-small"),
		ChatModel:      getEnv("MEMOS_AI_CHAT_MODEL", "gpt-4o-mini"),
		MaxRetries:     3,
		Timeout:        30 * time.Second,
	})
}

// getEnv gets an environment variable with a fallback.
func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
