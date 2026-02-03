package memo

import (
	"context"
	"sort"
	"strings"

	"github.com/hrygo/divinesense/ai/core/retrieval"
)

// Highlight represents a highlighted match in the content.
type Highlight struct {
	MatchedText string `json:"matched_text"`
	Start       int    `json:"start"`
	End         int    `json:"end"`
}

// HighlightedMemo represents a memo with highlighted search matches.
type HighlightedMemo struct {
	Name       string      `json:"name"`
	Snippet    string      `json:"snippet"`
	Highlights []Highlight `json:"highlights"`
	CreatedTs  int64       `json:"created_ts"`
	Score      float32     `json:"score"`
}

// HighlightService provides search highlighting functionality.
type HighlightService struct {
	retriever        *retrieval.AdaptiveRetriever
	tokenizer        *Tokenizer
	snippetExtractor *SnippetExtractor
}

// NewHighlightService creates a new HighlightService instance.
func NewHighlightService(retriever *retrieval.AdaptiveRetriever) *HighlightService {
	return &HighlightService{
		retriever:        retriever,
		tokenizer:        NewTokenizer(),
		snippetExtractor: NewSnippetExtractor(),
	}
}

// SearchWithHighlightOptions contains options for highlighted search.
type SearchWithHighlightOptions struct {
	Query        string
	UserID       int32
	Limit        int
	ContextChars int
}

// SearchWithHighlight performs a search and returns results with highlighted matches.
func (s *HighlightService) SearchWithHighlight(
	ctx context.Context,
	opts *SearchWithHighlightOptions,
) ([]HighlightedMemo, error) {
	// Set defaults
	if opts.Limit <= 0 {
		opts.Limit = 20
	}
	if opts.ContextChars <= 0 {
		opts.ContextChars = 50
	}

	// 1. Execute hybrid retrieval
	results, err := s.retriever.Retrieve(ctx, &retrieval.RetrievalOptions{
		Query:    opts.Query,
		UserID:   opts.UserID,
		Strategy: "hybrid_standard",
		Limit:    opts.Limit,
	})
	if err != nil {
		return nil, err
	}

	// 2. Tokenize query
	tokens := s.tokenizer.Tokenize(opts.Query)

	// 3. Build highlighted results
	highlighted := make([]HighlightedMemo, 0, len(results))
	for _, result := range results {
		if result.Memo == nil {
			continue
		}

		h := HighlightedMemo{
			Name:      result.Memo.UID,
			Score:     result.Score,
			CreatedTs: result.Memo.CreatedTs,
		}

		// Find match positions
		matches := s.findMatches(result.Content, tokens)

		// Extract snippet with context using SnippetExtractor
		h.Snippet, h.Highlights = s.snippetExtractor.ExtractSnippet(
			result.Content,
			matches,
			&ExtractOptions{ContextChars: opts.ContextChars, AddEllipsis: true},
		)

		highlighted = append(highlighted, h)
	}

	return highlighted, nil
}

// findMatches finds all occurrences of tokens in the content.
func (s *HighlightService) findMatches(content string, tokens []string) []Highlight {
	if len(tokens) == 0 {
		return nil
	}

	var matches []Highlight
	contentRunes := []rune(content)
	contentLen := len(contentRunes)

	for _, token := range tokens {
		if token == "" {
			continue
		}
		lowerToken := strings.ToLower(token)
		tokenRunes := []rune(lowerToken)
		tokenLen := len(tokenRunes)
		if tokenLen == 0 {
			continue
		}

		// Search for token in content using sliding window on contentRunes
		// This avoids index mismatch between original and lowercased rune slices
		limit := contentLen - tokenLen
		for i := 0; i <= limit; i++ {
			// Compare window with token (case-insensitive)
			window := strings.ToLower(string(contentRunes[i : i+tokenLen]))
			if window == lowerToken {
				matches = append(matches, Highlight{
					Start:       i,
					End:         i + tokenLen,
					MatchedText: string(contentRunes[i : i+tokenLen]),
				})
			}
		}
	}

	// Sort by position and remove overlaps
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].Start < matches[j].Start
	})

	return s.removeOverlaps(matches)
}

// removeOverlaps removes overlapping highlights, keeping the earlier ones.
func (s *HighlightService) removeOverlaps(matches []Highlight) []Highlight {
	if len(matches) <= 1 {
		return matches
	}

	result := make([]Highlight, 0, len(matches))
	result = append(result, matches[0])

	for i := 1; i < len(matches); i++ {
		last := result[len(result)-1]
		if matches[i].Start >= last.End {
			result = append(result, matches[i])
		}
	}

	return result
}
