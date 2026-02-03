// Package rag provides Self-RAG optimization.
package rag

// Constants for result evaluation.
const (
	UsefulScoreThreshold = 0.6  // Top1 score threshold for "useful"
	MinResultsForRerank  = 5    // Minimum results to consider reranking
	ScoreDiffThreshold   = 0.15 // Score difference threshold for reranking
)

// SuggestedAction represents the recommended action after evaluation.
type SuggestedAction string

const (
	ActionUse    SuggestedAction = "use"    // Use the results as-is
	ActionExpand SuggestedAction = "expand" // Expand/refine the query
	ActionDirect SuggestedAction = "direct" // Skip retrieval, answer directly
)

// EvaluationResult contains the evaluation of retrieval results.
type EvaluationResult struct {
	Reason          string
	SuggestedAction SuggestedAction
	TopScore        float32
	ScoreSpread     float32
	IsUseful        bool
}

// SearchResult represents a single search result.
type SearchResult struct {
	ID      string
	Content string
	Source  string
	Score   float32
}

// ResultEvaluator evaluates the usefulness of retrieval results.
type ResultEvaluator struct {
	usefulThreshold    float32
	scoreDiffThreshold float32
}

// NewResultEvaluator creates a new result evaluator.
func NewResultEvaluator() *ResultEvaluator {
	return &ResultEvaluator{
		usefulThreshold:    UsefulScoreThreshold,
		scoreDiffThreshold: ScoreDiffThreshold,
	}
}

// Evaluate evaluates the retrieval results.
func (e *ResultEvaluator) Evaluate(results []*SearchResult) *EvaluationResult {
	// Empty results - go direct
	if len(results) == 0 {
		return &EvaluationResult{
			IsUseful:        false,
			Reason:          "empty_results",
			SuggestedAction: ActionDirect,
			TopScore:        0,
			ScoreSpread:     0,
		}
	}

	topScore := results[0].Score

	// Calculate score spread if multiple results
	var scoreSpread float32
	if len(results) >= 2 {
		scoreSpread = topScore - results[1].Score
	}

	// High relevance - use results
	if topScore >= e.usefulThreshold {
		return &EvaluationResult{
			IsUseful:        true,
			Reason:          "high_relevance",
			SuggestedAction: ActionUse,
			TopScore:        topScore,
			ScoreSpread:     scoreSpread,
		}
	}

	// Medium relevance - depends on context
	if topScore >= 0.4 {
		return &EvaluationResult{
			IsUseful:        true,
			Reason:          "medium_relevance",
			SuggestedAction: ActionUse,
			TopScore:        topScore,
			ScoreSpread:     scoreSpread,
		}
	}

	// Low relevance - try expanding query
	return &EvaluationResult{
		IsUseful:        false,
		Reason:          "low_relevance",
		SuggestedAction: ActionExpand,
		TopScore:        topScore,
		ScoreSpread:     scoreSpread,
	}
}

// EvaluateResults is a convenience function for evaluation.
func EvaluateResults(results []*SearchResult) *EvaluationResult {
	return NewResultEvaluator().Evaluate(results)
}

// RerankDecider decides whether to apply reranking.
type RerankDecider struct {
	minResults         int
	scoreDiffThreshold float32
}

// NewRerankDecider creates a new rerank decider.
func NewRerankDecider() *RerankDecider {
	return &RerankDecider{
		minResults:         MinResultsForRerank,
		scoreDiffThreshold: ScoreDiffThreshold,
	}
}

// ShouldRerank determines if reranking should be applied.
func (d *RerankDecider) ShouldRerank(query string, results []*SearchResult) bool {
	// Too few results - don't rerank
	if len(results) < d.minResults {
		return false
	}

	// Simple keyword query - don't rerank
	if isSimpleKeywordQuery(query) {
		return false
	}

	// Large score gap between top results - top1 already wins
	if len(results) >= 2 {
		scoreDiff := results[0].Score - results[1].Score
		if scoreDiff > d.scoreDiffThreshold {
			return false
		}
	}

	// Close scores - reranking may help
	return true
}

// isSimpleKeywordQuery checks if query is a simple keyword search.
// Simple queries have 3 or fewer meaningful tokens.
func isSimpleKeywordQuery(query string) bool {
	// Count runes as a proxy for semantic complexity
	// Chinese: each character is roughly one token
	// English: count word-like segments
	runes := []rune(query)
	if len(runes) <= 3 {
		return true
	}

	// Count word boundaries (spaces and Chinese punctuation)
	wordCount := 1
	for _, r := range runes {
		if r == ' ' || r == '，' || r == ',' || r == '、' {
			wordCount++
		}
	}

	// For Chinese text without spaces, estimate by character count
	// Queries with <= 5 chars and no spaces are likely simple keywords
	hasSpaces := wordCount > 1
	if !hasSpaces && len(runes) <= 5 {
		return true
	}

	return wordCount <= 2
}

// ShouldRerank is a convenience function.
func ShouldRerank(query string, results []*SearchResult) bool {
	return NewRerankDecider().ShouldRerank(query, results)
}
