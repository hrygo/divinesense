package enrichment

import (
	"context"
	"testing"
	"time"
)

// mockEnricher 用于测试
type mockEnricher struct {
	enrichmentType EnrichmentType
	phase          Phase
	latency        time.Duration
}

func (m *mockEnricher) Type() EnrichmentType { return m.enrichmentType }
func (m *mockEnricher) Phase() Phase         { return m.phase }
func (m *mockEnricher) Enrich(ctx context.Context, content *MemoContent) *EnrichmentResult {
	time.Sleep(m.latency)
	return &EnrichmentResult{
		Type:    m.enrichmentType,
		Success: true,
		Data:    "mock result",
	}
}

func TestPipeline_EnrichAll(t *testing.T) {
	pipeline := NewPipeline(
		&mockEnricher{enrichmentType: EnrichmentSummary, phase: PhasePost, latency: 10 * time.Millisecond},
		&mockEnricher{enrichmentType: EnrichmentTags, phase: PhasePost, latency: 20 * time.Millisecond},
	)

	content := &MemoContent{
		MemoID:  "test-123",
		Content: "test content",
		Title:   "test title",
		UserID:  1,
	}

	results := pipeline.EnrichAll(context.Background(), content)

	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}
	if !results[EnrichmentSummary].Success {
		t.Error("summary enricher should succeed")
	}
	if !results[EnrichmentTags].Success {
		t.Error("tags enricher should succeed")
	}
}

func TestPipeline_EnrichPostSave(t *testing.T) {
	pipeline := NewPipeline(
		&mockEnricher{enrichmentType: EnrichmentFormat, phase: PhasePre, latency: 10 * time.Millisecond},
		&mockEnricher{enrichmentType: EnrichmentSummary, phase: PhasePost, latency: 10 * time.Millisecond},
	)

	content := &MemoContent{MemoID: "test-123", Content: "test", UserID: 1}
	results := pipeline.EnrichPostSave(context.Background(), content)

	if len(results) != 1 {
		t.Errorf("expected 1 result, got %d", len(results))
	}
	if _, ok := results[EnrichmentFormat]; ok {
		t.Error("should not include pre-save enricher in post-save results")
	}
}
