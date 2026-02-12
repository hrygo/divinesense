// Package routing provides the LLM routing service.
package routing

import (
	"context"
	"log/slog"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/hrygo/divinesense/ai"
)

// HistoryMatcher implements Layer 2 history-based intent matching.
// Layer 2a: Lexical similarity (~1ms) - Jaccard on character bigrams
// Layer 2b: Semantic similarity (~50ms) - Embedding cosine similarity (optional)
// Target: Handle 30%+ of requests that pass Layer 1.
type HistoryMatcher struct {
	embeddingService    ai.EmbeddingService // Optional: for semantic similarity
	similarityThreshold float32
	semanticThreshold   float32 // Threshold for semantic similarity fallback

	// Performance optimization: cache bigrams for recent inputs
	bigramCache   map[string]map[string]bool
	bigramCacheMu sync.Mutex
	maxCacheSize  int
}

// SetEmbeddingService sets the embedding service for semantic similarity matching.
func (m *HistoryMatcher) SetEmbeddingService(es ai.EmbeddingService) {
	m.embeddingService = es
}

// NewHistoryMatcher creates a new history matcher.
func NewHistoryMatcher(_ any) *HistoryMatcher {
	return &HistoryMatcher{
		similarityThreshold: 0.8,
		semanticThreshold:   0.75, // Lower threshold for semantic matching
		bigramCache:         make(map[string]map[string]bool),
		maxCacheSize:        100, // Cache last 100 unique inputs
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
// Currently disabled - returns no match.
// TODO: Implement history matching using alternative storage.
func (m *HistoryMatcher) Match(_ context.Context, _ int32, _ string) (*HistoryMatchResult, error) {
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
// Optimized with single pass and early normalization.
func cosineSimilarity(a, b []float32) float32 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}

	var dotProduct float32
	var normA float32
	var normB float32

	// Single pass: compute dot product and norms
	for i := range a {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	// Use math.Sqrt on final values only
	return dotProduct / (float32(math.Sqrt(float64(normA))) * float32(math.Sqrt(float64(normB))))
}

// calculateLexicalSimilarity calculates lexical similarity score between two strings.
// Uses character-level bigrams for Chinese text.
// Optimized with early exit and cached bigram extraction.
func (m *HistoryMatcher) calculateLexicalSimilarity(a, b string) float32 {
	// Quick exact match check
	if a == b {
		return 1.0
	}

	bigramsA := m.getBigrams(a)
	bigramsB := m.getBigrams(b)

	if len(bigramsA) == 0 || len(bigramsB) == 0 {
		return 0
	}

	// Early exit: if one set is much larger, max similarity is limited
	maxLen := len(bigramsA)
	minLen := len(bigramsB)
	if minLen > maxLen {
		maxLen, minLen = minLen, maxLen
	}
	// Max possible similarity is minLen/maxLen
	maxPossibleSim := float32(minLen) / float32(maxLen)
	if maxPossibleSim < m.similarityThreshold {
		// Early exit: even perfect match won't meet threshold
		return maxPossibleSim
	}

	// Calculate Jaccard similarity on bigram sets
	// Optimize: iterate over smaller set
	intersection := 0
	if len(bigramsA) < len(bigramsB) {
		for bg := range bigramsA {
			if bigramsB[bg] {
				intersection++
			}
		}
	} else {
		for bg := range bigramsB {
			if bigramsA[bg] {
				intersection++
			}
		}
	}

	union := len(bigramsA) + len(bigramsB) - intersection
	if union == 0 {
		return 0
	}

	return float32(intersection) / float32(union)
}

// getBigrams retrieves bigrams from cache or computes them.
func (m *HistoryMatcher) getBigrams(input string) map[string]bool {
	// Fast path: check cache without lock (optimistic read)
	m.bigramCacheMu.Lock()
	defer m.bigramCacheMu.Unlock()

	if bigrams, ok := m.bigramCache[input]; ok {
		return bigrams
	}

	// Cache miss: compute and store
	bigrams := m.extractBigrams(input)

	// Evict if cache is too large (simple FIFO)
	if len(m.bigramCache) >= m.maxCacheSize {
		// Remove first entry
		for key := range m.bigramCache {
			delete(m.bigramCache, key)
			break
		}
	}

	m.bigramCache[input] = bigrams
	return bigrams
}

// extractBigrams extracts character-level bigrams from input.
// Optimized with single pass and pre-allocation.
func (m *HistoryMatcher) extractBigrams(input string) map[string]bool {
	input = strings.TrimSpace(input)
	if len(input) == 0 {
		return nil
	}

	input = strings.ToLower(input)

	// Remove common punctuation in a single pass
	var runes []rune
	for _, r := range input {
		switch r {
		case ' ', ',', '。', '，', '？', '?', '！', '!', '、', '\t', '\n':
			// Skip punctuation
		default:
			runes = append(runes, r)
		}
	}

	if len(runes) == 0 {
		return nil
	}

	// Pre-allocate map with estimated size
	estimatedSize := len(runes) - 1
	if len(runes) <= 4 {
		estimatedSize = len(runes) + len(runes) - 1 // Include unigrams
	}
	bigrams := make(map[string]bool, estimatedSize)

	// Generate character bigrams
	for i := 0; i < len(runes)-1; i++ {
		bigram := string(runes[i : i+2])
		bigrams[bigram] = true
	}

	// Also add individual characters for short inputs (unigrams)
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
		// Amazing intent removed - Orchestrator handles complex requests
		return IntentUnknown
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
	default:
		return "unknown"
	}
}
