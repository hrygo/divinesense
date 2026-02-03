// Package tags provides intelligent tag suggestion for memos.
// P2-C001: Three-layer progressive tag suggestion system.
package tags

import (
	"context"
	"time"
)

// TagSuggester provides tag suggestions for memo content.
type TagSuggester interface {
	// Suggest returns tag suggestions based on content, user history, and rules.
	Suggest(ctx context.Context, req *SuggestRequest) (*SuggestResponse, error)
}

// SuggestRequest contains parameters for tag suggestion.
type SuggestRequest struct {
	MemoID  string
	Content string
	Title   string
	MaxTags int
	UserID  int32
	UseLLM  bool
}

// SuggestResponse contains tag suggestions and metadata.
type SuggestResponse struct {
	Tags    []Suggestion  `json:"tags"`
	Sources []string      `json:"sources"`
	Latency time.Duration `json:"latency"`
}

// Suggestion represents a single tag suggestion.
type Suggestion struct {
	Name       string  `json:"name"`
	Source     string  `json:"source"`
	Reason     string  `json:"reason,omitempty"`
	Confidence float64 `json:"confidence"`
}

// TagFrequency represents tag usage frequency.
type TagFrequency struct {
	Name  string
	Count int
}

// TagWithSimilarity represents a tag from similar memo.
type TagWithSimilarity struct {
	Name       string
	Similarity float64
}

// Layer represents a single layer in the suggestion pipeline.
type Layer interface {
	// Name returns the layer name for logging/metrics.
	Name() string
	// Suggest returns suggestions from this layer.
	Suggest(ctx context.Context, req *SuggestRequest) []Suggestion
}
