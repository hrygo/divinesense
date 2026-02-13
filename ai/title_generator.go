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

// LLM parameters for title generation
const (
	titleTimeout      = 15 * time.Second
	titleMaxTokens    = 20
	titleTemperature  = 0.1
	titleTopP         = 0.5
	titleMaxLen       = 500
	titleMaxRuneCount = 50
)

// TitleGenerator generates meaningful titles for AI conversations.
type TitleGenerator struct {
	llm LLMService
}

// TitleGeneratorConfig holds configuration for the title generator.
//
// Deprecated: Use NewTitleGenerator(llm LLMService) directly.
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
	llmCfg := &LLMConfig{
		Provider:    "generic",
		APIKey:      cfg.APIKey,
		BaseURL:     cfg.BaseURL,
		Model:       cfg.Model,
		MaxTokens:   titleMaxTokens,
		Temperature: titleTemperature,
	}
	llmService, err := NewLLMService(llmCfg)
	if err != nil {
		slog.Error("failed to create LLM service for title generator", "error", err)
		return nil
	}
	return &TitleGenerator{llm: llmService}
}

// NewTitleGeneratorWithLLM creates a new title generator with an existing LLMService.
// This is the preferred constructor for dependency injection.
// Panics if llmService is nil.
func NewTitleGeneratorWithLLM(llmService LLMService) *TitleGenerator {
	if llmService == nil {
		panic("ai: NewTitleGeneratorWithLLM: llmService cannot be nil")
	}
	return &TitleGenerator{llm: llmService}
}

// Generate generates a title based on the conversation content.
func (tg *TitleGenerator) Generate(ctx context.Context, userMessage, aiResponse string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, titleTimeout)
	defer cancel()

	// Truncate inputs using rune-aware truncation (Unicode-safe)
	userMessage = strutil.Truncate(userMessage, titleMaxLen)
	aiResponse = strutil.Truncate(aiResponse, titleMaxLen)
	prompt := fmt.Sprintf("用户消息: %s\n\nAI 回复: %s\n\n请为这段对话生成一个简短的标题。", userMessage, aiResponse)

	messages := []llm.Message{
		llm.SystemPrompt(titleSystemPrompt),
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
	runes := []rune(result.Title)
	if len(runes) > titleMaxRuneCount {
		result.Title = string(runes[:titleMaxRuneCount])
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

// titleSystemPrompt is the system prompt for title generation.
const titleSystemPrompt = `你是一个专业的对话标题生成助手。你的任务是根据用户和AI的对话内容，生成一个简洁、准确的标题。

要求：
1. 标题长度：3-15个字符（中文）或 3-8个单词（英文）
2. 标题应该反映对话的核心主题
3. 使用简洁的语言，避免使用"关于..."、"讨论了..."等冗余表述
4. 如果是问题，可以直接用问题本身作为标题
5. 如果是任务，可以用任务描述作为标题
6. 保持中立客观的语气

示例：
- 输入: "如何用Go连接PostgreSQL数据库？" -> 输出: "Go连接PostgreSQL"
- 输入: "帮我写一个二分查找算法" -> 输出: "二分查找算法实现"
- 输入: "今天天气怎么样？" -> 输出: "天气查询"
- 输入: "我的日程安排" -> 输出: "日程管理"

请直接返回JSON格式：{"title": "生成的标题"}`
