package summary

import (
	"context"
	"time"
)

// Summarizer 提供笔记摘要能力
type Summarizer interface {
	// Summarize 生成笔记摘要
	Summarize(ctx context.Context, req *SummarizeRequest) (*SummarizeResponse, error)
}

// SummarizeRequest 摘要请求
type SummarizeRequest struct {
	MemoID  string
	Content string
	Title   string
	MaxLen  int // 摘要最大长度（rune），默认 200
}

// SummarizeResponse 摘要响应
type SummarizeResponse struct {
	Summary string
	Source  string // "llm" | "fallback_first_para" | "fallback_first_sentence" | "fallback_truncate"
	Latency time.Duration
}
