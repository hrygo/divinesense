// Package rag provides Self-RAG optimization.
package rag

import (
	"sort"
)

// RRF (Reciprocal Rank Fusion) constants.
const (
	RRFDampingFactor = 60 // k = 60 is a common default
)

// RRFConfig contains configuration for RRF fusion.
type RRFConfig struct {
	DampingFactor int
}

// DefaultRRFConfig returns the default RRF configuration.
func DefaultRRFConfig() RRFConfig {
	return RRFConfig{
		DampingFactor: RRFDampingFactor,
	}
}

// RRF(d) = Σ weight_i / (k + rank_i(d)).
func FuseWithRRF(bm25Results, vectorResults []*SearchResult, config StrategyConfig) []*SearchResult {
	return FuseWithRRFConfig(bm25Results, vectorResults, config, DefaultRRFConfig())
}

// FuseWithRRFConfig fuses results with custom RRF configuration.
func FuseWithRRFConfig(bm25Results, vectorResults []*SearchResult, config StrategyConfig, rrfConfig RRFConfig) []*SearchResult {
	k := rrfConfig.DampingFactor
	scoreMap := make(map[string]float64)
	resultMap := make(map[string]*SearchResult)

	// BM25 score contribution
	for rank, result := range bm25Results {
		score := config.BM25Weight / float64(k+rank+1)
		scoreMap[result.ID] += score
		if _, exists := resultMap[result.ID]; !exists {
			resultMap[result.ID] = result
		}
	}

	// Vector score contribution
	for rank, result := range vectorResults {
		score := config.VectorWeight / float64(k+rank+1)
		scoreMap[result.ID] += score
		if _, exists := resultMap[result.ID]; !exists {
			resultMap[result.ID] = result
		}
	}

	// Convert to sorted slice
	type scoredResult struct {
		result *SearchResult
		score  float64
	}

	var scored []scoredResult
	for id, score := range scoreMap {
		if result, ok := resultMap[id]; ok {
			scored = append(scored, scoredResult{
				result: result,
				score:  score,
			})
		}
	}

	// Sort by RRF score descending
	sort.Slice(scored, func(i, j int) bool {
		return scored[i].score > scored[j].score
	})

	// Build result list with updated scores
	results := make([]*SearchResult, len(scored))
	for i, sr := range scored {
		results[i] = &SearchResult{
			ID:      sr.result.ID,
			Content: sr.result.Content,
			Score:   float32(sr.score),
			Source:  "hybrid",
		}
	}

	return results
}

// FuseMultiple fuses multiple result lists using RRF.
// weights should have the same length as resultLists.
func FuseMultiple(resultLists [][]*SearchResult, weights []float64) []*SearchResult {
	if len(resultLists) == 0 {
		return nil
	}
	if len(resultLists) != len(weights) {
		// Fallback to equal weights
		weights = make([]float64, len(resultLists))
		equalWeight := 1.0 / float64(len(resultLists))
		for i := range weights {
			weights[i] = equalWeight
		}
	}

	k := RRFDampingFactor
	scoreMap := make(map[string]float64)
	resultMap := make(map[string]*SearchResult)

	for listIdx, results := range resultLists {
		weight := weights[listIdx]
		for rank, result := range results {
			score := weight / float64(k+rank+1)
			scoreMap[result.ID] += score
			if _, exists := resultMap[result.ID]; !exists {
				resultMap[result.ID] = result
			}
		}
	}

	// Convert and sort
	type scoredResult struct {
		result *SearchResult
		score  float64
	}

	var scored []scoredResult
	for id, score := range scoreMap {
		if result, ok := resultMap[id]; ok {
			scored = append(scored, scoredResult{
				result: result,
				score:  score,
			})
		}
	}

	sort.Slice(scored, func(i, j int) bool {
		return scored[i].score > scored[j].score
	})

	results := make([]*SearchResult, len(scored))
	for i, sr := range scored {
		results[i] = &SearchResult{
			ID:      sr.result.ID,
			Content: sr.result.Content,
			Score:   float32(sr.score),
			Source:  "fused",
		}
	}

	return results
}
