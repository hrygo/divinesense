package enrichment

import (
	"context"
	"time"
)

// EnrichmentType 标识增强类型
type EnrichmentType string

// Phase 标识执行阶段
type Phase string

const (
	// EnrichmentFormat 格式化（同步，用户触发）
	EnrichmentFormat EnrichmentType = "format"

	// EnrichmentSummary 摘要（异步，自动触发）
	EnrichmentSummary EnrichmentType = "summary"

	// EnrichmentTags 标签（异步，自动触发）
	EnrichmentTags EnrichmentType = "tags"

	// EnrichmentTitle 标题（异步，自动触发）
	EnrichmentTitle EnrichmentType = "title"
)

const (
	// PhasePre 同步，保存前
	PhasePre Phase = "pre_save"

	// PhasePost 异步，保存后
	PhasePost Phase = "post_save"
)

// MemoContent 待增强的 Memo 内容
type MemoContent struct {
	MemoID  string
	Content string
	Title   string
	UserID  int32
}

// EnrichmentResult 增强结果
type EnrichmentResult struct {
	Type    EnrichmentType
	Success bool
	Data    any
	Error   error
	Latency time.Duration
}

// Enricher 内容增强器接口
type Enricher interface {
	// Type 返回增强器类型
	Type() EnrichmentType
	// Phase 返回该 Enricher 所属阶段
	Phase() Phase
	// Enrich 执行增强，返回结果
	Enrich(ctx context.Context, content *MemoContent) *EnrichmentResult
}
