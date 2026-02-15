package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/hrygo/divinesense/ai/core/llm"
	"github.com/hrygo/divinesense/ai/internal/strutil"
)

// TitleGenerator generates meaningful titles for AI conversations.
// Uses configuration from config/prompts/title.yaml.
type TitleGenerator struct {
	llm    LLMService
	config *TitlePromptConfig
}

// TitleGeneratorConfig holds configuration for the title generator.
//
// Deprecated: Use NewTitleGeneratorWithLLM(llm LLMService) directly.
// This config is kept for backward compatibility.
type TitleGeneratorConfig struct {
	APIKey  string
	BaseURL string
	Model   string
}

// NewTitleGenerator creates a new title generator instance.
//
// Deprecated: Use NewTitleGeneratorWithLLM(llm LLMService) instead.
// This constructor is kept for backward compatibility.
func NewTitleGenerator(cfg TitleGeneratorConfig) *TitleGenerator {
	// Create LLM service from config (backward compatibility)
	config := GetTitlePromptConfig()
	llmCfg := &LLMConfig{
		Provider:    "generic",
		APIKey:      cfg.APIKey,
		BaseURL:     cfg.BaseURL,
		Model:       cfg.Model,
		MaxTokens:   config.Params.MaxTokens,
		Temperature: float32(config.Params.Temperature),
	}
	llmService, err := NewLLMService(llmCfg)
	if err != nil {
		slog.Error("failed to create LLM service for title generator", "error", err)
		return nil
	}
	return &TitleGenerator{llm: llmService, config: config}
}

// NewTitleGeneratorWithLLM creates a new title generator with an existing LLMService.
// This is the preferred constructor for dependency injection.
// Panics if llmService is nil.
func NewTitleGeneratorWithLLM(llmService LLMService) *TitleGenerator {
	if llmService == nil {
		panic("ai: NewTitleGeneratorWithLLM: llmService cannot be nil")
	}
	return &TitleGenerator{
		llm:    llmService,
		config: GetTitlePromptConfig(),
	}
}

// Generate generates a title based on the conversation content.
func (tg *TitleGenerator) Generate(ctx context.Context, userMessage, aiResponse string) (string, error) {
	cfg := tg.config
	timeout := time.Duration(cfg.Params.TimeoutSeconds) * time.Second

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Truncate inputs using rune-aware truncation (Unicode-safe)
	truncateLen := cfg.Params.InputTruncateChars
	userMessage = strutil.Truncate(userMessage, truncateLen)
	aiResponse = strutil.Truncate(aiResponse, truncateLen)

	// Build prompt from template
	prompt, err := cfg.BuildConversationPrompt(&ConversationPromptData{
		UserMessage: userMessage,
		AIResponse:  aiResponse,
	})
	if err != nil {
		return "", fmt.Errorf("build prompt: %w", err)
	}

	messages := []llm.Message{
		llm.SystemPrompt(cfg.SystemPrompt),
		llm.UserMessage(prompt),
	}

	start := time.Now()
	content, stats, err := tg.llm.Chat(ctx, messages)
	latency := time.Since(start)

	if err != nil {
		slog.Error("title_generation_failed",
			"error", err,
			"latency_ms", latency.Milliseconds())
		return "", fmt.Errorf("LLM request failed: %w", err)
	}

	if content == "" {
		return "", fmt.Errorf("empty response from LLM")
	}

	var result struct {
		Title string `json:"title"`
	}
	if err := json.Unmarshal([]byte(content), &result); err != nil {
		slog.Warn("title_generation_parse_failed",
			"content", content,
			"error", err)
		return "", fmt.Errorf("parse response failed: %w", err)
	}

	if result.Title == "" {
		return "", fmt.Errorf("empty title in response")
	}

	// Truncate to max length (rune-aware for UTF-8)
	maxRunes := cfg.Params.MaxRunes
	runes := []rune(result.Title)
	if len(runes) > maxRunes {
		result.Title = string(runes[:maxRunes])
	}

	slog.Debug("title_generation_success",
		"title", result.Title,
		"latency_ms", latency.Milliseconds(),
		"tokens_total", stats.TotalTokens)

	return result.Title, nil
}

// GenerateTitleFromBlocks generates a title from a slice of blocks.
func (tg *TitleGenerator) GenerateTitleFromBlocks(ctx context.Context, blocks []BlockContent) (string, error) {
	var userMessage, aiResponse string

	for _, block := range blocks {
		if userMessage == "" {
			userMessage = block.UserInput
		}
		if aiResponse == "" {
			aiResponse = block.AssistantContent
		}
		if userMessage != "" && aiResponse != "" {
			break
		}
	}

	if userMessage == "" {
		return "", fmt.Errorf("no user message found in blocks")
	}

	return tg.Generate(ctx, userMessage, aiResponse)
}

// BlockContent represents a simplified block for title generation.
type BlockContent struct {
	UserInput        string
	AssistantContent string
}
