// Package context provides context building for LLM prompts.
package context

import (
	"sort"
)

// ContextPriority represents the priority level of a context segment.
type ContextPriority int

const (
	PrioritySystem      ContextPriority = 100 // System prompt - highest
	PriorityUserQuery   ContextPriority = 90  // Current user query
	PriorityRecentTurns ContextPriority = 80  // Most recent 3 turns
	PriorityRetrieval   ContextPriority = 70  // RAG retrieval results
	PriorityEpisodic    ContextPriority = 60  // Episodic memory
	PriorityPreferences ContextPriority = 50  // User preferences
	PriorityOlderTurns  ContextPriority = 40  // Older conversation turns
)

// ContextSegment represents a piece of context with priority.
type ContextSegment struct {
	Content   string
	Source    string
	Priority  ContextPriority
	TokenCost int
}

// PriorityRanker ranks and truncates context segments by priority.
type PriorityRanker struct{}

// NewPriorityRanker creates a new priority ranker.
func NewPriorityRanker() *PriorityRanker {
	return &PriorityRanker{}
}

// RankAndTruncate sorts segments by priority and truncates to fit budget.
func (r *PriorityRanker) RankAndTruncate(segments []*ContextSegment, budget int) []*ContextSegment {
	if len(segments) == 0 {
		return nil
	}

	// Sort by priority descending
	sorted := make([]*ContextSegment, len(segments))
	copy(sorted, segments)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Priority > sorted[j].Priority
	})

	var result []*ContextSegment
	usedTokens := 0

	for _, seg := range sorted {
		if seg.TokenCost <= 0 {
			continue
		}

		if usedTokens+seg.TokenCost <= budget {
			// Segment fits entirely
			result = append(result, seg)
			usedTokens += seg.TokenCost
		} else {
			// Try to fit partial segment
			remaining := budget - usedTokens
			if remaining >= MinSegmentTokens {
				truncated := truncateToTokens(seg.Content, remaining)
				if len(truncated) > 0 {
					result = append(result, &ContextSegment{
						Content:   truncated,
						Priority:  seg.Priority,
						TokenCost: remaining,
						Source:    seg.Source,
					})
				}
			}
			break
		}
	}

	return result
}

// PrioritizeAndTruncate is a convenience function.
func PrioritizeAndTruncate(segments []*ContextSegment, budget int) []*ContextSegment {
	return NewPriorityRanker().RankAndTruncate(segments, budget)
}

// Uses simple heuristic: 1 Chinese char ≈ 2 tokens, 1 English word ≈ 1 token.
func truncateToTokens(content string, maxTokens int) string {
	if maxTokens <= 0 {
		return ""
	}

	runes := []rune(content)

	// Rough estimate: average 1.5 tokens per rune for mixed Chinese/English
	estimatedRunes := int(float64(maxTokens) / 1.5)
	if estimatedRunes >= len(runes) {
		return content
	}

	// Truncate and add ellipsis
	if estimatedRunes > 3 {
		return string(runes[:estimatedRunes-3]) + "..."
	}

	return string(runes[:estimatedRunes])
}

// EstimateTokens estimates the token count for a string.
// Uses heuristic: Chinese chars count as ~2 tokens, ASCII as ~0.3 tokens per char.
func EstimateTokens(content string) int {
	if len(content) == 0 {
		return 0
	}

	chineseCount := 0
	asciiCount := 0

	for _, r := range content {
		if r >= 0x4E00 && r <= 0x9FFF {
			chineseCount++
		} else if r < 128 {
			asciiCount++
		} else {
			chineseCount++ // Other Unicode treated as Chinese
		}
	}

	// Heuristic: Chinese ~2 tokens per char, ASCII ~0.25 tokens per char
	tokens := chineseCount*2 + asciiCount/4
	if tokens == 0 && len(content) > 0 {
		tokens = 1
	}

	return tokens
}
