package enrichment

import (
	"context"
	"sync"
	"time"
)

// Pipeline 编排多个 Enricher
type Pipeline struct {
	enrichers []Enricher
	timeout   time.Duration
}

// NewPipeline 创建增强管线
func NewPipeline(enrichers ...Enricher) *Pipeline {
	return &Pipeline{
		enrichers: enrichers,
		timeout:   30 * time.Second,
	}
}

// EnrichAll 并行执行所有增强器，返回结果集合
func (p *Pipeline) EnrichAll(ctx context.Context, content *MemoContent) map[EnrichmentType]*EnrichmentResult {
	ctx, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()

	results := make(map[EnrichmentType]*EnrichmentResult)
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, e := range p.enrichers {
		wg.Add(1)
		go func(enricher Enricher) {
			defer wg.Done()
			result := enricher.Enrich(ctx, content)
			mu.Lock()
			results[enricher.Type()] = result
			mu.Unlock()
		}(e)
	}

	wg.Wait()
	return results
}

// EnrichPostSave 执行 Post-save 阶段的增强（异步并行）
func (p *Pipeline) EnrichPostSave(ctx context.Context, content *MemoContent) map[EnrichmentType]*EnrichmentResult {
	var postEnrichers []Enricher
	for _, e := range p.enrichers {
		if e.Phase() == PhasePost {
			postEnrichers = append(postEnrichers, e)
		}
	}
	if len(postEnrichers) == 0 {
		return nil
	}
	tmpPipeline := NewPipeline(postEnrichers...)
	return tmpPipeline.EnrichAll(ctx, content)
}

// EnrichOne 执行单个类型的增强
func (p *Pipeline) EnrichOne(ctx context.Context, t EnrichmentType, content *MemoContent) *EnrichmentResult {
	for _, e := range p.enrichers {
		if e.Type() == t {
			return e.Enrich(ctx, content)
		}
	}
	return &EnrichmentResult{Type: t, Success: false, Error: ErrEnricherNotFound}
}

// Errors
var ErrEnricherNotFound = &EnricherNotFoundError{}

type EnricherNotFoundError struct{}

func (e *EnricherNotFoundError) Error() string {
	return "enricher not found"
}
