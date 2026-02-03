// Package rag provides Self-RAG optimization.
package rag

import (
	"github.com/hrygo/divinesense/ai/router"
)

// SearchStrategy represents the retrieval strategy to use.
type SearchStrategy string

const (
	StrategyBM25Only        SearchStrategy = "bm25_only"
	StrategySemanticOnly    SearchStrategy = "semantic_only"
	StrategyHybridStandard  SearchStrategy = "hybrid_standard"
	StrategyHybridBM25Heavy SearchStrategy = "hybrid_bm25_weighted"
	StrategyFullPipeline    SearchStrategy = "full_pipeline"
)

// StrategyConfig contains weights for hybrid search.
type StrategyConfig struct {
	BM25Weight   float64
	VectorWeight float64
	UseReranker  bool
}

// strategyConfigs maps strategies to their configurations.
var strategyConfigs = map[SearchStrategy]StrategyConfig{
	StrategyBM25Only: {
		BM25Weight:   1.0,
		VectorWeight: 0.0,
		UseReranker:  false,
	},
	StrategySemanticOnly: {
		BM25Weight:   0.0,
		VectorWeight: 1.0,
		UseReranker:  false,
	},
	StrategyHybridStandard: {
		BM25Weight:   0.5,
		VectorWeight: 0.5,
		UseReranker:  false,
	},
	StrategyHybridBM25Heavy: {
		BM25Weight:   0.7,
		VectorWeight: 0.3,
		UseReranker:  false,
	},
	StrategyFullPipeline: {
		BM25Weight:   0.5,
		VectorWeight: 0.5,
		UseReranker:  true,
	},
}

// GetStrategyConfig returns the configuration for a strategy.
func GetStrategyConfig(strategy SearchStrategy) StrategyConfig {
	if config, ok := strategyConfigs[strategy]; ok {
		return config
	}
	return strategyConfigs[StrategyHybridStandard]
}

// StrategySelector selects the appropriate search strategy.
type StrategySelector struct{}

// NewStrategySelector creates a new strategy selector.
func NewStrategySelector() *StrategySelector {
	return &StrategySelector{}
}

// Select chooses a search strategy based on intent.
func (s *StrategySelector) Select(intent router.Intent) SearchStrategy {
	switch intent {
	case router.IntentScheduleQuery, router.IntentScheduleCreate, router.IntentScheduleUpdate:
		// Schedule queries work better with BM25 (keyword matching)
		return StrategyBM25Only
	case router.IntentMemoSearch:
		// Memo search benefits from semantic understanding
		return StrategySemanticOnly
	case router.IntentAmazing:
		// Complex questions need full pipeline
		return StrategyFullPipeline
	default:
		// Default to hybrid
		return StrategyHybridStandard
	}
}

// SelectStrategy is a convenience function.
func SelectStrategy(intent router.Intent) SearchStrategy {
	return NewStrategySelector().Select(intent)
}

// HybridSearcher performs hybrid search combining BM25 and vector search.
type HybridSearcher struct {
	config StrategyConfig
}

// NewHybridSearcher creates a new hybrid searcher.
func NewHybridSearcher(strategy SearchStrategy) *HybridSearcher {
	return &HybridSearcher{
		config: GetStrategyConfig(strategy),
	}
}

// MergeResults merges BM25 and vector results using the configured weights.
func (h *HybridSearcher) MergeResults(bm25Results, vectorResults []*SearchResult) []*SearchResult {
	// If one source is empty, return the other
	if len(bm25Results) == 0 {
		return vectorResults
	}
	if len(vectorResults) == 0 {
		return bm25Results
	}

	// If only using one source
	if h.config.BM25Weight == 0 {
		return vectorResults
	}
	if h.config.VectorWeight == 0 {
		return bm25Results
	}

	// Use RRF for fusion
	return FuseWithRRF(bm25Results, vectorResults, h.config)
}

// GetConfig returns the current strategy configuration.
func (h *HybridSearcher) GetConfig() StrategyConfig {
	return h.config
}
