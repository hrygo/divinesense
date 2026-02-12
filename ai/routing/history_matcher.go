// Package routing provides the LLM routing service.
package routing

import (
	"context"
	"math"
	"strings"
	"sync"

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
		semanticThreshold:   0.75,
		bigramCache:         make(map[string]map[string]bool),
		maxCacheSize:        100,
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

// SaveDecision saves a routing decision for future matching.
// Currently disabled - no-op.
// TODO: Implement using alternative storage.
func (m *HistoryMatcher) SaveDecision(_ context.Context, _ int32, _ string, _ Intent, _ bool) error {
	return nil
}

// cosineSimilarity calculates cosine similarity between two vectors.
func cosineSimilarity(a, b []float32) float32 {
	if len(a) != len(b) || len(a) == 0 {
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
func (m *HistoryMatcher) calculateLexicalSimilarity(a, b string) float32 {
	if a == b {
		return 1.0
	}

	bigramsA := m.getBigrams(a)
	bigramsB := m.getBigrams(b)

	if len(bigramsA) == 0 || len(bigramsB) == 0 {
		return 0
	}

	maxLen := len(bigramsA)
	minLen := len(bigramsB)
	if minLen > maxLen {
		maxLen, minLen = minLen, maxLen
	}

	maxPossibleSim := float32(minLen) / float32(maxLen)
	if maxPossibleSim < m.similarityThreshold {
		return maxPossibleSim
	}

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
	m.bigramCacheMu.Lock()
	defer m.bigramCacheMu.Unlock()

	if bigrams, ok := m.bigramCache[input]; ok {
		return bigrams
	}

	bigrams := m.extractBigrams(input)

	if len(m.bigramCache) >= m.maxCacheSize {
		for key := range m.bigramCache {
			delete(m.bigramCache, key)
			break
		}
	}

	m.bigramCache[input] = bigrams
	return bigrams
}

// extractBigrams extracts character-level bigrams from input.
func (m *HistoryMatcher) extractBigrams(input string) map[string]bool {
	input = strings.TrimSpace(input)
	if len(input) == 0 {
		return nil
	}

	input = strings.ToLower(input)

	var runes []rune
	for _, r := range input {
		switch r {
		case ' ', ',', '。', '，', '？', '?', '！', '!', '、', '\t', '\n':
		default:
			runes = append(runes, r)
		}
	}

	if len(runes) == 0 {
		return nil
	}

	estimatedSize := len(runes) - 1
	if len(runes) <= 4 {
		estimatedSize = len(runes) + len(runes) - 1
	}
	bigrams := make(map[string]bool, estimatedSize)

	for i := 0; i < len(runes)-1; i++ {
		bigram := string(runes[i : i+2])
		bigrams[bigram] = true
	}

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
	default:
		return IntentUnknown
	}
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
