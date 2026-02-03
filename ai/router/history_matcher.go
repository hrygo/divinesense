// Package router provides the LLM routing service.
package router

import (
	"context"
	"log/slog"
	"math"
	"strings"
	"time"

	"github.com/hrygo/divinesense/ai"
	"github.com/hrygo/divinesense/ai/memory"
)

// HistoryMatcher implements Layer 2 history-based intent matching.
// Layer 2a: Lexical similarity (~1ms) - Jaccard on character bigrams
// Layer 2b: Semantic similarity (~50ms) - Embedding cosine similarity (optional)
// Target: Handle 30%+ of requests that pass Layer 1.
type HistoryMatcher struct {
	memoryService       memory.MemoryService
	embeddingService    ai.EmbeddingService // Optional: for semantic similarity
	similarityThreshold float32
	semanticThreshold   float32 // Threshold for semantic similarity fallback
	maxHistoryLookup    int
}

// SetEmbeddingService sets the embedding service for semantic similarity matching.
func (m *HistoryMatcher) SetEmbeddingService(es ai.EmbeddingService) {
	m.embeddingService = es
}

// NewHistoryMatcher creates a new history matcher.
func NewHistoryMatcher(ms memory.MemoryService) *HistoryMatcher {
	return &HistoryMatcher{
		memoryService:       ms,
		similarityThreshold: 0.8,
		semanticThreshold:   0.75, // Lower threshold for semantic matching
		maxHistoryLookup:    10,
	}
}

// HistoryMatchResult contains the result of history matching.
type HistoryMatchResult struct {
	Intent     Intent
	SourceID   int64
	Confidence float32
	Matched    bool
}

// Match attempts to classify intent by finding similar historical patterns.
// Layer 2a: Lexical matching (~1ms) - high precision
// Layer 2b: Semantic matching (~50ms) - triggered when lexical is ambiguous
// Returns matched=true if a similar pattern was found with confidence >= threshold.
func (m *HistoryMatcher) Match(ctx context.Context, userID int32, input string) (*HistoryMatchResult, error) {
	if m.memoryService == nil {
		return &HistoryMatchResult{Matched: false}, nil
	}

	// Search for similar episodes
	episodes, err := m.memoryService.SearchEpisodes(ctx, userID, input, m.maxHistoryLookup)
	if err != nil {
		return nil, err
	}

	if len(episodes) == 0 {
		return &HistoryMatchResult{Matched: false}, nil
	}

	// Layer 2a: Find best lexical match
	var bestMatch *memory.EpisodicMemory
	var bestLexicalSim float32

	for i := range episodes {
		ep := &episodes[i]
		similarity := m.calculateLexicalSimilarity(input, ep.UserInput)

		// Only consider successful outcomes
		if ep.Outcome == "success" && similarity > bestLexicalSim {
			bestLexicalSim = similarity
			bestMatch = ep
		}
	}

	// Layer 2a Result: Check if lexical similarity meets threshold
	if bestMatch != nil && bestLexicalSim >= m.similarityThreshold {
		intent := m.agentTypeToIntent(bestMatch.AgentType, input)
		return &HistoryMatchResult{
			Intent:     intent,
			Confidence: bestLexicalSim,
			SourceID:   bestMatch.ID,
			Matched:    true,
		}, nil
	}

	// Layer 2b: Semantic similarity fallback
	// Triggered when lexical is moderately close but below threshold
	// This catches cases like "帮我找笔记" vs "搜索备忘"
	if m.embeddingService != nil && bestLexicalSim >= 0.4 && bestLexicalSim < m.similarityThreshold {
		semanticResult := m.matchBySemanticSimilarity(ctx, input, episodes)
		if semanticResult.Matched {
			slog.Debug("history matched by semantic similarity",
				"input", truncate(input, 50),
				"intent", semanticResult.Intent,
				"confidence", semanticResult.Confidence,
				"lexical_score", bestLexicalSim)
			return semanticResult, nil
		}
	}

	return &HistoryMatchResult{Matched: false}, nil
}

// matchBySemanticSimilarity matches episodes using embedding cosine similarity.
func (m *HistoryMatcher) matchBySemanticSimilarity(ctx context.Context, input string, episodes []memory.EpisodicMemory) *HistoryMatchResult {
	// Get input embedding
	inputEmbedding, err := m.embeddingService.Embed(ctx, input)
	if err != nil {
		return &HistoryMatchResult{Matched: false}
	}

	// Find best semantic match by embedding on-the-fly
	var bestMatch *memory.EpisodicMemory
	var bestSemanticSim float32

	for i := range episodes {
		ep := &episodes[i]
		if ep.Outcome != "success" {
			continue
		}

		// Embed episode input on-the-fly (can be cached later)
		epEmbedding, err := m.embeddingService.Embed(ctx, ep.UserInput)
		if err != nil {
			continue
		}

		// Calculate cosine similarity
		similarity := cosineSimilarity(inputEmbedding, epEmbedding)
		if similarity > bestSemanticSim {
			bestSemanticSim = similarity
			bestMatch = ep
		}
	}

	if bestMatch == nil || bestSemanticSim < m.semanticThreshold {
		return &HistoryMatchResult{Matched: false}
	}

	intent := m.agentTypeToIntent(bestMatch.AgentType, input)
	return &HistoryMatchResult{
		Intent:     intent,
		Confidence: bestSemanticSim,
		SourceID:   bestMatch.ID,
		Matched:    true,
	}
}

// cosineSimilarity calculates cosine similarity between two vectors.
func cosineSimilarity(a, b []float32) float32 {
	if len(a) != len(b) {
		return 0
	}

	var dotProduct float32
	var normA float32
	var normB float32

	for i := range a {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dotProduct / (float32(math.Sqrt(float64(normA))) * float32(math.Sqrt(float64(normB))))
}

// calculateLexicalSimilarity calculates lexical similarity score between two strings.
// Uses character-level bigrams for Chinese text.
func (m *HistoryMatcher) calculateLexicalSimilarity(a, b string) float32 {
	bigramsA := m.extractBigrams(a)
	bigramsB := m.extractBigrams(b)

	if len(bigramsA) == 0 || len(bigramsB) == 0 {
		return 0
	}

	// Calculate Jaccard similarity on bigram sets
	intersection := 0
	for bg := range bigramsA {
		if bigramsB[bg] {
			intersection++
		}
	}

	union := len(bigramsA) + len(bigramsB) - intersection
	if union == 0 {
		return 0
	}

	return float32(intersection) / float32(union)
}

// extractBigrams extracts character-level bigrams from input.
func (m *HistoryMatcher) extractBigrams(input string) map[string]bool {
	input = strings.TrimSpace(input)
	input = strings.ToLower(input)

	// Remove common punctuation
	for _, r := range []string{" ", ",", "。", "，", "？", "?", "！", "!", "、"} {
		input = strings.ReplaceAll(input, r, "")
	}

	runes := []rune(input)
	bigrams := make(map[string]bool)

	// Generate character bigrams
	for i := 0; i < len(runes)-1; i++ {
		bigram := string(runes[i : i+2])
		bigrams[bigram] = true
	}

	// Also add individual characters for short inputs
	if len(runes) <= 4 {
		for _, r := range runes {
			bigrams[string(r)] = true
		}
	}

	return bigrams
}

// agentTypeToIntent maps agent type from episode to current intent.
func (m *HistoryMatcher) agentTypeToIntent(agentType, input string) Intent {
	switch agentType {
	case "schedule":
		if containsAny(input, []string{"查看", "有什么", "哪些"}) {
			return IntentScheduleQuery
		}
		if containsAny(input, []string{"修改", "更新", "取消"}) {
			return IntentScheduleUpdate
		}
		return IntentScheduleCreate
	case "memo":
		if containsAny(input, []string{"搜索", "查找", "找"}) {
			return IntentMemoSearch
		}
		return IntentMemoCreate
	case "amazing":
		return IntentAmazing
	default:
		return IntentUnknown
	}
}

// SaveDecision saves a routing decision to memory for future matching.
func (m *HistoryMatcher) SaveDecision(ctx context.Context, userID int32, input string, intent Intent, success bool) error {
	if m.memoryService == nil {
		return nil
	}

	outcome := "failure"
	if success {
		outcome = "success"
	}

	episode := memory.EpisodicMemory{
		UserID:     userID,
		Timestamp:  time.Now(),
		AgentType:  m.intentToAgentType(intent),
		UserInput:  input,
		Outcome:    outcome,
		Summary:    "routing_decision:" + string(intent),
		Importance: 0.5,
	}

	return m.memoryService.SaveEpisode(ctx, episode)
}

// intentToAgentType maps intent to agent type for storage.
func (m *HistoryMatcher) intentToAgentType(intent Intent) string {
	switch intent {
	case IntentScheduleCreate, IntentScheduleQuery, IntentScheduleUpdate, IntentBatchSchedule:
		return "schedule"
	case IntentMemoSearch, IntentMemoCreate:
		return "memo"
	case IntentAmazing:
		return "amazing"
	default:
		return "unknown"
	}
}
