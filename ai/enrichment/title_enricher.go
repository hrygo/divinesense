package enrichment

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/hrygo/divinesense/ai/core/llm"
	"github.com/hrygo/divinesense/ai/internal/strutil"
)

// TitleEnricher 为 Memo 内容生成标题
type TitleEnricher struct {
	llmService llm.Service
	timeout    time.Duration
	maxLen     int
	maxRunes   int
}

// NewTitleEnricher 创建新的标题增强器
func NewTitleEnricher(llmService llm.Service) *TitleEnricher {
	return &TitleEnricher{
		llmService: llmService,
		timeout:    10 * time.Second,
		maxLen:     500,
		maxRunes:   50,
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

	if e.llmService == nil {
		return &EnrichmentResult{
			Type:    EnrichmentTitle,
			Success: false,
			Error:   nil, // Graceful degradation
			Latency: time.Since(start),
		}
	}

	// Set timeout for title generation
	ctx, cancel := context.WithTimeout(ctx, e.timeout)
	defer cancel()

	// Truncate content (rune-aware for UTF-8)
	truncatedContent := strutil.Truncate(content.Content, e.maxLen)

	title := content.Title
	if title == "" {
		title = "(无标题)"
	}

	// Build prompt
	prompt := fmt.Sprintf(`请为以下笔记生成一个简短的标题。

笔记标题: %s

笔记内容:
%s

要求：
1. 标题长度：3-15个字符（中文）或 3-8个单词（英文）
2. 标题应该反映笔记的核心主题
3. 使用简洁的语言
4. 直接返回JSON格式：{"title": "生成的标题"}`, title, truncatedContent)

	// Call LLM
	messages := []llm.Message{
		{Role: "user", Content: prompt},
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
	runes := []rune(result.Title)
	if len(runes) > e.maxRunes {
		result.Title = string(runes[:e.maxRunes])
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
