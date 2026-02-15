// Package format provides the formatter interface for AI text formatting.
// This interface is consumed by Team B (Assistant) to format user input into standard Markdown.
package format

import (
	"context"
	"time"
)

// Formatter 将随意输入的文本格式化为标准 Markdown
type Formatter interface {
	Format(ctx context.Context, req *FormatRequest) (*FormatResponse, error)
}

type FormatRequest struct {
	Content string // 用户原始输入
	UserID  int32
}

type FormatResponse struct {
	Formatted string // 格式化后的 Markdown 内容
	Changed   bool   // 内容是否有变化
	Source    string // "llm" | "passthrough"
	Latency   time.Duration
}
