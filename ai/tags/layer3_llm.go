package tags

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/hrygo/divinesense/ai"
)

// LLMLayer provides tag suggestions using LLM.
// Layer 3: ~300ms latency, optional, graceful degradation.
type LLMLayer struct {
	llmService ai.LLMService
	timeout    time.Duration
}

// NewLLMLayer creates a new LLM layer.
func NewLLMLayer(llmService ai.LLMService) *LLMLayer {
	return &LLMLayer{
		llmService: llmService,
		timeout:    500 * time.Millisecond,
	}
}

// Name returns the layer name.
func (l *LLMLayer) Name() string {
	return "llm"
}

const tagSuggestPrompt = `请为以下笔记内容建议 3-5 个合适的标签。

## 笔记标题
%s

## 笔记内容
%s

## 要求
1. 标签应该简洁，1-4 个字
2. 优先使用常见分类词（技术、生活、工作、学习等）
3. 可以包含主题词（如具体技术名称）
4. 只返回 JSON 数组格式，如: ["标签1", "标签2", "标签3"]
5. 不要返回其他内容，只返回 JSON 数组`

// Suggest returns tag suggestions using LLM.
func (l *LLMLayer) Suggest(ctx context.Context, req *SuggestRequest) []Suggestion {
	if l.llmService == nil {
		return nil
	}

	// Set timeout for LLM call
	ctx, cancel := context.WithTimeout(ctx, l.timeout)
	defer cancel()

	// Prepare content (truncate if too long)
	content := req.Content
	if len(content) > 500 {
		content = content[:500] + "..."
	}

	title := req.Title
	if title == "" {
		title = "(无标题)"
	}

	// Build prompt
	prompt := formatPrompt(tagSuggestPrompt, title, content)

	// Call LLM
	messages := []ai.Message{
		{Role: "user", Content: prompt},
	}

	response, stats, err := l.llmService.Chat(ctx, messages)
	if err != nil {
		slog.Warn("LLM tag suggestion failed",
			"error", err,
			"timeout", l.timeout,
		)
		return nil // Graceful degradation: return empty, don't affect L1/L2
	}
	_ = stats // Stats not needed for tag suggestion

	// Parse response
	tags := parseTagsFromJSON(response)
	if len(tags) == 0 {
		slog.Warn("LLM returned no parseable tags",
			"response", truncateLog(response, 100),
		)
		return nil
	}

	var suggestions []Suggestion
	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		tag = strings.TrimPrefix(tag, "#")
		if tag != "" && len(tag) <= 20 {
			suggestions = append(suggestions, Suggestion{
				Name:       tag,
				Confidence: 0.75,
				Source:     "llm",
				Reason:     "AI suggested",
			})
		}
	}

	return suggestions
}

// formatPrompt formats the prompt with title and content.
func formatPrompt(template, title, content string) string {
	return fmt.Sprintf(template, title, content)
}

// parseTagsFromJSON parses JSON array from LLM response.
func parseTagsFromJSON(response string) []string {
	// Clean response (remove markdown code blocks if present)
	response = strings.TrimSpace(response)
	response = strings.TrimPrefix(response, "```json")
	response = strings.TrimPrefix(response, "```")
	response = strings.TrimSuffix(response, "```")
	response = strings.TrimSpace(response)

	// Try to extract JSON array
	start := strings.Index(response, "[")
	end := strings.LastIndex(response, "]")
	if start >= 0 && end > start {
		response = response[start : end+1]
	}

	var tags []string
	if err := json.Unmarshal([]byte(response), &tags); err != nil {
		// Try line-by-line parsing as fallback
		lines := strings.Split(response, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			line = strings.TrimPrefix(line, "-")
			line = strings.TrimPrefix(line, "#")
			line = strings.TrimSpace(line)
			if line != "" && len(line) <= 20 {
				tags = append(tags, line)
			}
		}
	}

	return tags
}

// truncateLog truncates string for logging.
func truncateLog(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
