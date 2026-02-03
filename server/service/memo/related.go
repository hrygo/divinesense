package memo

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/hrygo/divinesense/ai/cache"
	"github.com/hrygo/divinesense/store"
)

// RelatedMemo represents a related memo with similarity score.
type RelatedMemo struct {
	Name       string   `json:"name"`
	Title      string   `json:"title"`
	SharedTags []string `json:"shared_tags"`
	CreatedTs  int64    `json:"created_ts"`
	Similarity float32  `json:"similarity"`
}

// RelatedService provides related memo recommendations.
// P1-C003: Related memo recommendations based on semantic similarity and tag co-occurrence.
type RelatedService struct {
	store *store.Store
	cache cache.CacheService

	// Weights for score calculation
	vectorWeight float32
	tagWeight    float32
	timeWeight   float32
}

// NewRelatedService creates a new RelatedService instance.
func NewRelatedService(s *store.Store, c cache.CacheService) *RelatedService {
	return &RelatedService{
		store:        s,
		cache:        c,
		vectorWeight: 0.6,
		tagWeight:    0.3,
		timeWeight:   0.1,
	}
}

// GetRelatedMemosOptions contains options for related memo queries.
type GetRelatedMemosOptions struct {
	MemoUID string
	UserID  int32
	Limit   int
}

// GetRelatedMemos returns related memos for the given memo.
func (s *RelatedService) GetRelatedMemos(
	ctx context.Context,
	opts *GetRelatedMemosOptions,
) ([]RelatedMemo, error) {
	if opts.Limit <= 0 {
		opts.Limit = 5
	}
	if opts.Limit > 20 {
		opts.Limit = 20
	}

	// Check cache first
	cacheKey := fmt.Sprintf("related:%d:%s", opts.UserID, opts.MemoUID)
	if s.cache != nil {
		if cached, ok := s.cache.Get(ctx, cacheKey); ok {
			var result []RelatedMemo
			if err := json.Unmarshal(cached, &result); err == nil {
				return result, nil
			}
		}
	}

	// Get current memo
	currentMemo, err := s.store.GetMemo(ctx, &store.FindMemo{UID: &opts.MemoUID})
	if err != nil {
		return nil, fmt.Errorf("failed to get current memo: %w", err)
	}
	if currentMemo == nil {
		return nil, fmt.Errorf("memo not found: %s", opts.MemoUID)
	}

	// Get current memo's tags
	currentTags := extractTagsFromContent(currentMemo.Content)

	// Get candidate memos (same user, exclude self)
	candidates, err := s.getCandidateMemos(ctx, opts.UserID, opts.MemoUID, opts.Limit*3)
	if err != nil {
		return nil, err
	}

	// Calculate scores for each candidate
	var results []RelatedMemo
	for _, candidate := range candidates {
		candidateTags := extractTagsFromContent(candidate.Content)

		// Calculate tag co-occurrence score
		sharedTags := intersectTags(currentTags, candidateTags)
		tagScore := float32(0)
		if len(currentTags) > 0 {
			tagScore = float32(len(sharedTags)) / float32(len(currentTags))
		}

		// Calculate time proximity score (higher for memos within 7 days)
		timeDiff := abs64(currentMemo.CreatedTs - candidate.CreatedTs)
		sevenDays := int64(7 * 24 * 3600)
		timeScore := float32(0)
		if timeDiff < sevenDays {
			timeScore = 1.0 - float32(timeDiff)/float32(sevenDays)
		}

		// Calculate content similarity (simple keyword overlap for now)
		// TODO: Use vector similarity when embedding store is available
		contentScore := calculateContentSimilarity(currentMemo.Content, candidate.Content)

		// Weighted combination
		finalScore := s.vectorWeight*contentScore + s.tagWeight*tagScore + s.timeWeight*timeScore

		// Only include if score is meaningful
		if finalScore > 0.1 {
			results = append(results, RelatedMemo{
				Name:       candidate.UID,
				Title:      extractTitleFromContent(candidate.Content),
				Similarity: finalScore,
				SharedTags: sharedTags,
				CreatedTs:  candidate.CreatedTs,
			})
		}
	}

	// Sort by similarity descending
	sort.Slice(results, func(i, j int) bool {
		return results[i].Similarity > results[j].Similarity
	})

	// Limit results
	if len(results) > opts.Limit {
		results = results[:opts.Limit]
	}

	// Cache results
	if s.cache != nil && len(results) > 0 {
		if data, err := json.Marshal(results); err == nil {
			s.cache.Set(ctx, cacheKey, data, 5*time.Minute)
		}
	}

	return results, nil
}

// getCandidateMemos retrieves candidate memos for comparison.
func (s *RelatedService) getCandidateMemos(
	ctx context.Context,
	userID int32,
	excludeUID string,
	limit int,
) ([]*store.Memo, error) {
	// Get recent memos from the same user
	memos, err := s.store.ListMemos(ctx, &store.FindMemo{
		CreatorID: &userID,
		Limit:     &limit,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list memos: %w", err)
	}

	// Filter out the current memo
	var candidates []*store.Memo
	for _, m := range memos {
		if m.UID != excludeUID {
			candidates = append(candidates, m)
		}
	}

	return candidates, nil
}

// extractTagsFromContent extracts hashtags from memo content.
func extractTagsFromContent(content string) []string {
	var tags []string
	seen := make(map[string]bool)

	words := strings.Fields(content)
	for _, word := range words {
		if strings.HasPrefix(word, "#") && len(word) > 1 {
			// Clean the tag
			tag := strings.TrimPrefix(word, "#")
			tag = strings.TrimRight(tag, ".,!?;:")
			if tag != "" && !seen[tag] {
				tags = append(tags, tag)
				seen[tag] = true
			}
		}
	}

	return tags
}

// intersectTags returns tags present in both slices.
func intersectTags(a, b []string) []string {
	if len(a) == 0 || len(b) == 0 {
		return nil
	}

	bSet := make(map[string]bool)
	for _, t := range b {
		bSet[t] = true
	}

	var result []string
	for _, t := range a {
		if bSet[t] {
			result = append(result, t)
		}
	}

	return result
}

// extractTitleFromContent extracts the first line as title.
func extractTitleFromContent(content string) string {
	lines := strings.SplitN(content, "\n", 2)
	if len(lines) > 0 {
		title := strings.TrimSpace(lines[0])
		// Truncate if too long
		if len(title) > 50 {
			title = title[:47] + "..."
		}
		return title
	}
	return ""
}

// calculateContentSimilarity calculates simple keyword-based similarity.
func calculateContentSimilarity(content1, content2 string) float32 {
	words1 := tokenizeForSimilarity(content1)
	words2 := tokenizeForSimilarity(content2)

	if len(words1) == 0 || len(words2) == 0 {
		return 0
	}

	// Create word sets
	set1 := make(map[string]bool)
	for _, w := range words1 {
		set1[w] = true
	}

	set2 := make(map[string]bool)
	for _, w := range words2 {
		set2[w] = true
	}

	// Calculate Jaccard similarity
	intersection := 0
	for w := range set1 {
		if set2[w] {
			intersection++
		}
	}

	union := len(set1) + len(set2) - intersection
	if union == 0 {
		return 0
	}

	return float32(intersection) / float32(union)
}

// tokenizeForSimilarity tokenizes content for similarity calculation.
func tokenizeForSimilarity(content string) []string {
	content = strings.ToLower(content)

	// Simple tokenization
	var words []string
	var current strings.Builder

	for _, r := range content {
		if isWordChar(r) {
			current.WriteRune(r)
		} else if current.Len() > 0 {
			word := current.String()
			if len(word) >= 2 { // Ignore single chars
				words = append(words, word)
			}
			current.Reset()
		}
	}

	if current.Len() > 0 {
		word := current.String()
		if len(word) >= 2 {
			words = append(words, word)
		}
	}

	return words
}

// isWordChar returns true if the rune is part of a word.
func isWordChar(r rune) bool {
	return (r >= 'a' && r <= 'z') ||
		(r >= 'A' && r <= 'Z') ||
		(r >= '0' && r <= '9') ||
		(r >= 0x4E00 && r <= 0x9FFF) // CJK characters
}

// abs64 returns absolute value of int64.
func abs64(x int64) int64 {
	if x < 0 {
		return -x
	}
	return x
}
