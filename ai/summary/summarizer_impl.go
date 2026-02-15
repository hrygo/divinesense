package summary

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/hrygo/divinesense/ai/core/llm"
)

// llmSummarizer 使用 LLM 生成摘要
type llmSummarizer struct {
	llm     llm.Service
	timeout time.Duration
}

// NewSummarizer 创建摘要生成器
func NewSummarizer(llmSvc llm.Service) Summarizer {
	return &llmSummarizer{
		llm:     llmSvc,
		timeout: 15 * time.Second,
	}
}

func (s *llmSummarizer) Summarize(ctx context.Context, req *SummarizeRequest) (*SummarizeResponse, error) {
	maxLen := req.MaxLen
	if maxLen <= 0 {
		maxLen = 200
	}

	// 1. 短文本无需摘要
	if runeLen(req.Content) <= maxLen {
		return &SummarizeResponse{
			Summary: req.Content,
			Source:  "original",
		}, nil
	}

	// 2. LLM 不可用时走 Fallback
	if s.llm == nil {
		return FallbackSummarize(req)
	}

	// 3. LLM 生成摘要
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	userPrompt := fmt.Sprintf(`请为以下笔记生成不超过 %d 字的摘要：

%s

请直接返回JSON格式：{"summary": "生成的摘要"}`, maxLen, req.Content)

	messages := []llm.Message{
		llm.SystemPrompt(summarySystemPrompt),
		llm.UserMessage(userPrompt),
	}

	content, stats, err := s.llm.Chat(ctx, messages)
	if err != nil {
		return FallbackSummarize(req)
	}

	summary := parseSummary(content)
	summary = truncateRunes(summary, maxLen)

	return &SummarizeResponse{
		Summary: summary,
		Source:  "llm",
		Latency: time.Duration(stats.TotalDurationMs) * time.Millisecond,
	}, nil
}

func parseSummary(content string) string {
	var result struct {
		Summary string `json:"summary"`
	}
	if err := json.Unmarshal([]byte(content), &result); err == nil && result.Summary != "" {
		return strings.TrimSpace(result.Summary)
	}

	if idx := strings.Index(content, `"summary"`); idx >= 0 {
		start := strings.Index(content[idx:], ":") + idx + 1
		end := strings.Index(content[start:], "}")
		if end > 0 {
			return strings.Trim(content[start:start+end], `" `)
		}
	}

	return strings.TrimSpace(content)
}

const summarySystemPrompt = `你是一个专业的笔记摘要助手。你的任务是根据笔记原文，生成一段精炼的摘要。

要求：
1. 摘要长度不超过指定字数
2. 保留笔记的核心观点和关键信息
3. 使用与原文一致的语言
4. 不要添加原文没有的观点
5. 直接输出摘要文本，不要添加"摘要："等前缀
6. 返回JSON格式：{"summary": "生成的摘要"}`
