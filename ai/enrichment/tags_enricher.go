package enrichment

import (
	"context"
	"time"

	"github.com/hrygo/divinesense/ai/tags"
)

// TagsEnricher 使用现有的 TagSuggester 提供标签建议
type TagsEnricher struct {
	suggester tags.TagSuggester
	timeout   time.Duration
}

// NewTagsEnricher 创建新的标签增强器
func NewTagsEnricher(suggester tags.TagSuggester) *TagsEnricher {
	return &TagsEnricher{
		suggester: suggester,
		timeout:   2 * time.Second,
	}
}

// Type 返回增强器类型
func (e *TagsEnricher) Type() EnrichmentType {
	return EnrichmentTags
}

// Phase 返回该 Enricher 所属阶段
func (e *TagsEnricher) Phase() Phase {
	return PhasePost
}

// Enrich 执行标签增强
func (e *TagsEnricher) Enrich(ctx context.Context, content *MemoContent) *EnrichmentResult {
	start := time.Now()

	if e.suggester == nil {
		return &EnrichmentResult{
			Type:    EnrichmentTags,
			Success: false,
			Error:   nil, // Graceful degradation
			Latency: time.Since(start),
		}
	}

	// Set timeout for tag suggestion
	ctx, cancel := context.WithTimeout(ctx, e.timeout)
	defer cancel()

	// Build request
	req := &tags.SuggestRequest{
		MemoID:  content.MemoID,
		Content: content.Content,
		Title:   content.Title,
		UserID:  content.UserID,
		MaxTags: 5,
		UseLLM:  true, // Enable LLM layer for AI suggestions
	}

	// Call tag suggester
	resp, err := e.suggester.Suggest(ctx, req)
	if err != nil {
		return &EnrichmentResult{
			Type:    EnrichmentTags,
			Success: false,
			Error:   err,
			Latency: time.Since(start),
		}
	}

	// Extract tag names
	tagNames := make([]string, 0, len(resp.Tags))
	for _, t := range resp.Tags {
		if t.Confidence > 0.5 { // Only include high-confidence suggestions
			tagNames = append(tagNames, t.Name)
		}
	}

	return &EnrichmentResult{
		Type:    EnrichmentTags,
		Success: true,
		Data:    tagNames,
		Latency: time.Since(start),
	}
}
