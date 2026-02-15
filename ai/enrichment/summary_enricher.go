package enrichment

import (
	"context"
	"time"

	"github.com/hrygo/divinesense/ai/core/llm"
	"github.com/hrygo/divinesense/ai/summary"
)

// SummaryEnricher generates summaries for Memo content
type SummaryEnricher struct {
	summarizer summary.Summarizer
	timeout    time.Duration
}

// NewSummaryEnricher creates a new summary enricher
func NewSummaryEnricher(llmService llm.Service) *SummaryEnricher {
	return &SummaryEnricher{
		summarizer: summary.NewSummarizer(llmService),
		timeout:    10 * time.Second,
	}
}

// Type returns the enrichment type
func (e *SummaryEnricher) Type() EnrichmentType {
	return EnrichmentSummary
}

// Phase returns the enrichment phase
func (e *SummaryEnricher) Phase() Phase {
	return PhasePost
}

// Enrich executes summary enrichment
func (e *SummaryEnricher) Enrich(ctx context.Context, content *MemoContent) *EnrichmentResult {
	start := time.Now()

	if e.summarizer == nil {
		return &EnrichmentResult{
			Type:    EnrichmentSummary,
			Success: false,
			Error:   nil, // Graceful degradation
			Latency: time.Since(start),
		}
	}

	// Set timeout for summary generation
	ctx, cancel := context.WithTimeout(ctx, e.timeout)
	defer cancel()

	// Build request
	req := &summary.SummarizeRequest{
		MemoID:  content.MemoID,
		Content: content.Content,
		Title:   content.Title,
		MaxLen:  200,
	}

	// Call summarizer
	resp, err := e.summarizer.Summarize(ctx, req)
	latency := time.Since(start)

	if err != nil {
		return &EnrichmentResult{
			Type:    EnrichmentSummary,
			Success: false,
			Error:   err,
			Latency: latency,
		}
	}

	return &EnrichmentResult{
		Type:    EnrichmentSummary,
		Success: true,
		Data:    resp.Summary,
		Latency: latency,
	}
}
