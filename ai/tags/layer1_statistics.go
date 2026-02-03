package tags

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"

	"github.com/hrygo/divinesense/ai/cache"
	"github.com/hrygo/divinesense/store"
)

// StatisticsLayer provides tag suggestions based on user statistics.
// Layer 1: 0ms latency, uses high-frequency tags, recent tags, and similar memo tags.
type StatisticsLayer struct {
	store *store.Store
	cache cache.CacheService
}

// NewStatisticsLayer creates a new statistics layer.
func NewStatisticsLayer(s *store.Store, c cache.CacheService) *StatisticsLayer {
	return &StatisticsLayer{
		store: s,
		cache: c,
	}
}

// Name returns the layer name.
func (l *StatisticsLayer) Name() string {
	return "statistics"
}

// Suggest returns tag suggestions based on user statistics.
// Optimized to query memos only once per request.
func (l *StatisticsLayer) Suggest(ctx context.Context, req *SuggestRequest) []Suggestion {
	// Try to get cached tag stats first
	tagStats := l.getCachedTagStats(ctx, req.UserID)
	if tagStats == nil {
		// Query memos once and compute all stats
		tagStats = l.computeTagStats(ctx, req.UserID)
		if tagStats != nil {
			l.cacheTagStats(ctx, req.UserID, tagStats)
		}
	}

	if tagStats == nil {
		return nil
	}

	var suggestions []Suggestion
	seen := make(map[string]bool)

	// 1. High-frequency tags (TOP-5)
	for i, tag := range tagStats.Frequent {
		if i >= 5 {
			break
		}
		if !seen[strings.ToLower(tag.Name)] {
			seen[strings.ToLower(tag.Name)] = true
			suggestions = append(suggestions, Suggestion{
				Name:       tag.Name,
				Confidence: normalizeFrequency(tag.Count),
				Source:     "statistics",
				Reason:     fmt.Sprintf("used %d times", tag.Count),
			})
		}
	}

	// 2. Recent tags (last 7 days)
	for _, tag := range tagStats.Recent {
		if !seen[strings.ToLower(tag.Name)] {
			seen[strings.ToLower(tag.Name)] = true
			suggestions = append(suggestions, Suggestion{
				Name:       tag.Name,
				Confidence: 0.7,
				Source:     "statistics",
				Reason:     "recently used",
			})
		}
	}

	// 3. Content-based keyword matching with existing tags
	if req.Content != "" {
		contentTags := l.matchContentWithTags(tagStats.Frequent, req.Content)
		for _, tag := range contentTags {
			if !seen[strings.ToLower(tag.Name)] {
				seen[strings.ToLower(tag.Name)] = true
				suggestions = append(suggestions, Suggestion{
					Name:       tag.Name,
					Confidence: tag.Similarity * 0.8,
					Source:     "statistics",
					Reason:     "matches content",
				})
			}
		}
	}

	return suggestions
}

// tagStats holds precomputed tag statistics for a user.
type tagStats struct {
	Frequent []TagFrequency `json:"frequent"`
	Recent   []TagFrequency `json:"recent"`
}

// getCachedTagStats retrieves cached tag stats.
func (l *StatisticsLayer) getCachedTagStats(ctx context.Context, userID int32) *tagStats {
	if l.cache == nil {
		return nil
	}

	cacheKey := fmt.Sprintf("tags:stats:%d", userID)
	if cached, ok := l.cache.Get(ctx, cacheKey); ok {
		var stats tagStats
		if err := decodeJSON(cached, &stats); err == nil {
			return &stats
		}
	}
	return nil
}

// cacheTagStats stores tag stats in cache.
func (l *StatisticsLayer) cacheTagStats(ctx context.Context, userID int32, stats *tagStats) {
	if l.cache == nil || stats == nil {
		return
	}

	cacheKey := fmt.Sprintf("tags:stats:%d", userID)
	if data, err := encodeJSON(stats); err == nil {
		_ = l.cache.Set(ctx, cacheKey, data, 30*time.Minute) //nolint:errcheck // cache failure is non-critical
	}
}

// computeTagStats queries memos once and computes all tag statistics.
func (l *StatisticsLayer) computeTagStats(ctx context.Context, userID int32) *tagStats {
	memos, err := l.store.ListMemos(ctx, &store.FindMemo{
		CreatorID: &userID,
	})
	if err != nil {
		slog.Warn("failed to list memos for tag statistics",
			"user_id", userID,
			"error", err,
		)
		return nil
	}

	// Calculate cutoff for recent tags (7 days)
	cutoff := time.Now().AddDate(0, 0, -7).Unix()

	// Count all tags and recent tags in single pass
	allCounts := make(map[string]int)
	recentCounts := make(map[string]int)

	for _, memo := range memos {
		if memo.Payload == nil {
			continue
		}
		for _, tag := range memo.Payload.Tags {
			allCounts[tag]++
			if memo.CreatedTs >= cutoff {
				recentCounts[tag]++
			}
		}
	}

	// Convert to sorted slices
	frequent := make([]TagFrequency, 0, len(allCounts))
	for name, count := range allCounts {
		frequent = append(frequent, TagFrequency{Name: name, Count: count})
	}
	sort.Slice(frequent, func(i, j int) bool {
		return frequent[i].Count > frequent[j].Count
	})

	recent := make([]TagFrequency, 0, len(recentCounts))
	for name, count := range recentCounts {
		recent = append(recent, TagFrequency{Name: name, Count: count})
	}
	sort.Slice(recent, func(i, j int) bool {
		return recent[i].Count > recent[j].Count
	})

	return &tagStats{
		Frequent: frequent,
		Recent:   recent,
	}
}

// matchContentWithTags finds tags that match words in the content.
// Uses pre-computed tag list instead of querying again.
func (l *StatisticsLayer) matchContentWithTags(allTags []TagFrequency, content string) []TagWithSimilarity {
	// Normalize content for matching
	contentLower := strings.ToLower(content)
	contentWords := tokenize(contentLower)
	contentSet := make(map[string]bool)
	for _, w := range contentWords {
		contentSet[w] = true
	}

	var matches []TagWithSimilarity
	for _, tag := range allTags {
		tagLower := strings.ToLower(tag.Name)
		tagWords := tokenize(tagLower)

		// Check if any tag word appears in content
		for _, tw := range tagWords {
			if contentSet[tw] || strings.Contains(contentLower, tagLower) {
				matches = append(matches, TagWithSimilarity{
					Name:       tag.Name,
					Similarity: 0.8,
				})
				break
			}
		}
	}

	// Limit to top 3
	if len(matches) > 3 {
		matches = matches[:3]
	}
	return matches
}

// normalizeFrequency converts count to confidence (0.6 - 1.0).
func normalizeFrequency(count int) float64 {
	if count >= 10 {
		return 1.0
	}
	if count >= 5 {
		return 0.9
	}
	if count >= 3 {
		return 0.8
	}
	if count >= 2 {
		return 0.7
	}
	return 0.6
}

// tokenize splits text into lowercase words.
func tokenize(text string) []string {
	var words []string
	var current strings.Builder

	for _, r := range text {
		if isWordRune(r) {
			current.WriteRune(r)
		} else if current.Len() > 0 {
			words = append(words, current.String())
			current.Reset()
		}
	}
	if current.Len() > 0 {
		words = append(words, current.String())
	}

	return words
}

// isWordRune returns true if rune is part of a word.
func isWordRune(r rune) bool {
	return (r >= 'a' && r <= 'z') ||
		(r >= 'A' && r <= 'Z') ||
		(r >= '0' && r <= '9') ||
		(r >= 0x4E00 && r <= 0x9FFF) // CJK
}
