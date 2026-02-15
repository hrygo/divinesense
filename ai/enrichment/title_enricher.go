package enrichment

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/hrygo/divinesense/ai"
	"github.com/hrygo/divinesense/ai/core/llm"
	"github.com/hrygo/divinesense/ai/internal/strutil"
)

// TitleEnricher 为 Memo 内容生成标题
// Uses configuration from config/prompts/title.yaml.
type TitleEnricher struct {
	llmService llm.Service
	config     *ai.TitlePromptConfig
}

// NewTitleEnricher 创建新的标题增强器
func NewTitleEnricher(llmService llm.Service) *TitleEnricher {
	return &TitleEnricher{
		llmService: llmService,
		config:     ai.GetTitlePromptConfig(),
	}
}

// Type 返回增强器类型
func (e *TitleEnricher) Type() EnrichmentType {
	return EnrichmentTitle
}

// Phase 返回该 Enricher 所属阶段
func (e *TitleEnricher) Phase() Phase {
	return PhasePost
}

// Enrich 执行标题增强
func (e *TitleEnricher) Enrich(ctx context.Context, content *MemoContent) *EnrichmentResult {
	start := time.Now()
	cfg := e.config

	if e.llmService == nil {
		return &EnrichmentResult{
			Type:    EnrichmentTitle,
			Success: false,
			Error:   nil, // Graceful degradation
			Latency: time.Since(start),
		}
	}

	// Set timeout from config
	timeout := time.Duration(cfg.Params.TimeoutSeconds) * time.Second
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Truncate content (rune-aware for UTF-8)
	truncateLen := cfg.Params.InputTruncateChars
	truncatedContent := strutil.Truncate(content.Content, truncateLen)

	title := content.Title
	if title == "" {
		title = "(无标题)"
	}

	// Build prompt from template
	prompt, err := cfg.BuildMemoPrompt(&ai.MemoPromptData{
		Content: truncatedContent,
		Title:   title,
	})
	if err != nil {
		return &EnrichmentResult{
			Type:    EnrichmentTitle,
			Success: false,
			Error:   fmt.Errorf("build prompt: %w", err),
			Latency: time.Since(start),
		}
	}

	// Call LLM
	messages := []llm.Message{
		llm.SystemPrompt(cfg.SystemPrompt),
		llm.UserMessage(prompt),
	}

	response, stats, err := e.llmService.Chat(ctx, messages)
	latency := time.Since(start)

	if err != nil {
		slog.Warn("title_enrichment_failed",
			"error", err,
			"memo_id", content.MemoID,
			"latency_ms", latency.Milliseconds())
		return &EnrichmentResult{
			Type:    EnrichmentTitle,
			Success: false,
			Error:   err,
			Latency: latency,
		}
	}

	// Parse response
	var result struct {
		Title string `json:"title"`
	}
	if err := json.Unmarshal([]byte(response), &result); err != nil {
		slog.Warn("title_enrichment_parse_failed",
			"response", strutil.Truncate(response, 100),
			"error", err)
		return &EnrichmentResult{
			Type:    EnrichmentTitle,
			Success: false,
			Error:   fmt.Errorf("parse response failed: %w", err),
			Latency: latency,
		}
	}

	if result.Title == "" {
		return &EnrichmentResult{
			Type:    EnrichmentTitle,
			Success: false,
			Error:   fmt.Errorf("empty title in response"),
			Latency: latency,
		}
	}

	// Truncate to max length (rune-aware for UTF-8)
	maxRunes := cfg.Params.MaxRunes
	runes := []rune(result.Title)
	if len(runes) > maxRunes {
		result.Title = string(runes[:maxRunes])
	}

	slog.Debug("title_enrichment_success",
		"memo_id", content.MemoID,
		"title", result.Title,
		"latency_ms", latency.Milliseconds(),
		"tokens_total", stats.TotalTokens)

	return &EnrichmentResult{
		Type:    EnrichmentTitle,
		Success: true,
		Data:    result.Title,
		Latency: latency,
	}
}
