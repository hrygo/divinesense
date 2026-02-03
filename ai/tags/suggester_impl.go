package tags

import (
	"context"
	"sort"
	"strings"
	"time"

	"github.com/hrygo/divinesense/ai"
	"github.com/hrygo/divinesense/ai/cache"
	"github.com/hrygo/divinesense/store"
)

// suggesterImpl implements TagSuggester with three-layer progressive strategy.
type suggesterImpl struct {
	layer1 *StatisticsLayer
	layer2 *RulesLayer
	layer3 *LLMLayer
}

// NewTagSuggester creates a new TagSuggester with all three layers.
func NewTagSuggester(s *store.Store, llmService ai.LLMService, c cache.CacheService) TagSuggester {
	return &suggesterImpl{
		layer1: NewStatisticsLayer(s, c),
		layer2: NewRulesLayer(),
		layer3: NewLLMLayer(llmService),
	}
}

// Suggest returns tag suggestions using three-layer progressive strategy.
func (s *suggesterImpl) Suggest(ctx context.Context, req *SuggestRequest) (*SuggestResponse, error) {
	start := time.Now()
	var allSuggestions []Suggestion
	var sources []string

	// Set defaults
	if req.MaxTags <= 0 {
		req.MaxTags = 5
	}
	if req.MaxTags > 10 {
		req.MaxTags = 10
	}

	// Layer 1: Statistics (sync, always run)
	l1Suggestions := s.layer1.Suggest(ctx, req)
	allSuggestions = append(allSuggestions, l1Suggestions...)
	if len(l1Suggestions) > 0 {
		sources = append(sources, "statistics")
	}

	// Layer 2: Rules (sync, always run)
	l2Suggestions := s.layer2.Suggest(ctx, req)
	allSuggestions = append(allSuggestions, l2Suggestions...)
	if len(l2Suggestions) > 0 {
		sources = append(sources, "rules")
	}

	// Layer 3: LLM (optional, only if needed and enabled)
	if req.UseLLM && len(allSuggestions) < req.MaxTags {
		l3Suggestions := s.layer3.Suggest(ctx, req)
		allSuggestions = append(allSuggestions, l3Suggestions...)
		if len(l3Suggestions) > 0 {
			sources = append(sources, "llm")
		}
	}

	// Merge, dedupe, and rank
	finalTags := s.mergeAndRank(allSuggestions, req.MaxTags)

	return &SuggestResponse{
		Tags:    finalTags,
		Latency: time.Since(start),
		Sources: sources,
	}, nil
}

// mergeAndRank deduplicates and ranks suggestions.
func (s *suggesterImpl) mergeAndRank(suggestions []Suggestion, limit int) []Suggestion {
	// Dedupe: keep highest confidence for each tag
	tagMap := make(map[string]Suggestion)
	for _, sug := range suggestions {
		key := strings.ToLower(sug.Name)
		existing, ok := tagMap[key]
		if !ok || sug.Confidence > existing.Confidence {
			tagMap[key] = sug
		}
	}

	// Convert to slice
	var result []Suggestion
	for _, sug := range tagMap {
		result = append(result, sug)
	}

	// Sort by confidence descending
	sort.Slice(result, func(i, j int) bool {
		return result[i].Confidence > result[j].Confidence
	})

	// Limit results
	if len(result) > limit {
		result = result[:limit]
	}

	return result
}
