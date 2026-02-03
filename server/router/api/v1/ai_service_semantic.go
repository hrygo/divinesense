package v1

import (
	"context"
	"fmt"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/hrygo/divinesense/ai/tags"
	v1pb "github.com/hrygo/divinesense/proto/gen/api/v1"
	"github.com/hrygo/divinesense/store"
)

// SemanticSearch performs semantic search on memos.
func (s *AIService) SemanticSearch(ctx context.Context, req *v1pb.SemanticSearchRequest) (*v1pb.SemanticSearchResponse, error) {
	if !s.IsEnabled() {
		return nil, status.Errorf(codes.Unavailable, "AI features are disabled")
	}

	// Get current user
	user, err := getCurrentUser(ctx, s.Store)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "unauthorized")
	}

	// Validate parameters
	if req.Query == "" {
		return nil, status.Errorf(codes.InvalidArgument, "query is required")
	}

	// Add input length validation
	const (
		maxQueryLength = 1000
		minQueryLength = 2
	)

	if len(req.Query) > maxQueryLength {
		return nil, status.Errorf(codes.InvalidArgument,
			"query too long: maximum %d characters, got %d", maxQueryLength, len(req.Query))
	}

	// Trim and check minimum length
	trimmedQuery := strings.TrimSpace(req.Query)
	if len(trimmedQuery) < minQueryLength {
		return nil, status.Errorf(codes.InvalidArgument,
			"query too short: minimum %d characters after trimming", minQueryLength)
	}

	limit := int(req.Limit)
	if limit <= 0 {
		limit = 10
	}
	if limit > 50 {
		limit = 50
	}

	// Vectorize the query
	queryVector, err := s.EmbeddingService.Embed(ctx, req.Query)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to embed query: %v", err)
	}

	// Vector search (Top 10, optimized for 2C2G)
	results, err := s.Store.VectorSearch(ctx, &store.VectorSearchOptions{
		UserID: user.ID,
		Vector: queryVector,
		Limit:  10,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to search: %v", err)
	}

	// Filter low relevance results (Threshold: 0.5)
	var filteredResults []*store.MemoWithScore
	for _, r := range results {
		if r.Score >= 0.5 {
			filteredResults = append(filteredResults, r)
		}
	}
	results = filteredResults

	if len(results) == 0 {
		return &v1pb.SemanticSearchResponse{Results: []*v1pb.SearchResult{}}, nil
	}

	// Re-rank (optional)
	if s.RerankerService != nil && s.RerankerService.IsEnabled() && len(results) > limit {
		documents := make([]string, len(results))
		for i, r := range results {
			documents[i] = r.Memo.Content
		}

		rerankResults, err := s.RerankerService.Rerank(ctx, req.Query, documents, limit)
		if err == nil {
			// Reorder based on rerank results
			reordered := make([]*store.MemoWithScore, len(rerankResults))
			for i, rr := range rerankResults {
				reordered[i] = results[rr.Index]
				reordered[i].Score = rr.Score
			}
			results = reordered
		}
	}

	// Truncate results
	if len(results) > limit {
		results = results[:limit]
	}

	// Build response
	response := &v1pb.SemanticSearchResponse{
		Results: make([]*v1pb.SearchResult, len(results)),
	}

	for i, r := range results {
		snippet := r.Memo.Content
		if len(snippet) > 200 {
			snippet = snippet[:200] + "..."
		}

		response.Results[i] = &v1pb.SearchResult{
			Name:    fmt.Sprintf("memos/%s", r.Memo.UID),
			Snippet: snippet,
			Score:   r.Score,
		}
	}

	return response, nil
}

// SuggestTags suggests tags for memo content.
// P2-C001: Uses three-layer progressive strategy (statistics -> rules -> LLM).
func (s *AIService) SuggestTags(ctx context.Context, req *v1pb.SuggestTagsRequest) (*v1pb.SuggestTagsResponse, error) {
	const (
		maxContentLength = 5000 // Maximum content length in characters
		minContentLength = 3    // Minimum content length
	)

	// Get current user
	user, err := getCurrentUser(ctx, s.Store)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "unauthorized")
	}

	if req.Content == "" {
		return nil, status.Errorf(codes.InvalidArgument, "content is required")
	}

	// Validate content length
	contentLen := len([]rune(req.Content))
	if contentLen < minContentLength {
		return nil, status.Errorf(codes.InvalidArgument, "content too short (min %d characters)", minContentLength)
	}
	if contentLen > maxContentLength {
		return nil, status.Errorf(codes.InvalidArgument, "content too long (max %d characters)", maxContentLength)
	}

	// Validate and set limit
	limit := int(req.Limit)
	if limit <= 0 {
		limit = 5
	}
	if limit > 10 {
		limit = 10
	}

	// Use TagSuggester for three-layer progressive suggestions
	suggester := s.getTagSuggester()
	response, err := suggester.Suggest(ctx, &tags.SuggestRequest{
		UserID:  user.ID,
		Content: req.Content,
		MaxTags: limit,
		UseLLM:  s.LLMService != nil, // Only use LLM if available
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to suggest tags")
	}

	// Extract tag names from suggestions
	tagNames := make([]string, 0, len(response.Tags))
	for _, tag := range response.Tags {
		tagNames = append(tagNames, tag.Name)
	}

	return &v1pb.SuggestTagsResponse{Tags: tagNames}, nil
}

// getTagSuggester returns a TagSuggester instance.
func (s *AIService) getTagSuggester() tags.TagSuggester {
	// Note: CacheService is nil for now; caching is handled gracefully
	return tags.NewTagSuggester(s.Store, s.LLMService, nil)
}

// GetRelatedMemos finds memos related to a specific memo.
func (s *AIService) GetRelatedMemos(ctx context.Context, req *v1pb.GetRelatedMemosRequest) (*v1pb.GetRelatedMemosResponse, error) {
	if !s.IsEnabled() {
		return nil, status.Errorf(codes.Unavailable, "AI features are disabled")
	}

	// Get current user
	user, err := getCurrentUser(ctx, s.Store)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "unauthorized")
	}

	if req.Name == "" {
		return nil, status.Errorf(codes.InvalidArgument, "name is required")
	}

	// Parse memo UID from name (format: "memos/{uid}")
	var memoUID string
	if _, err := fmt.Sscanf(req.Name, "memos/%s", &memoUID); err != nil || memoUID == "" {
		return nil, status.Errorf(codes.InvalidArgument, "invalid name format, expected 'memos/{uid}'")
	}

	// Find the memo
	memo, err := s.Store.GetMemo(ctx, &store.FindMemo{
		UID:       &memoUID,
		CreatorID: &user.ID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get memo: %v", err)
	}
	if memo == nil {
		return nil, status.Errorf(codes.NotFound, "memo not found")
	}

	// Get embedding for the memo
	embedding, err := s.Store.GetMemoEmbedding(ctx, memo.ID, "BAAI/bge-m3")
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get memo embedding: %v", err)
	}

	var vector []float32
	if embedding == nil {
		// Generate embedding on-the-fly if not available
		vector, err = s.EmbeddingService.Embed(ctx, memo.Content)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to generate embedding: %v", err)
		}
		// Save the embedding for future use
		_, _ = s.Store.UpsertMemoEmbedding(ctx, &store.MemoEmbedding{
			MemoID:    memo.ID,
			Embedding: vector,
			Model:     "BAAI/bge-m3",
		})
	} else {
		vector = embedding.Embedding
	}

	limit := int(req.Limit)
	if limit <= 0 {
		limit = 5
	}

	// Vector search
	results, err := s.Store.VectorSearch(ctx, &store.VectorSearchOptions{
		UserID: user.ID,
		Vector: vector,
		Limit:  limit + 1, // +1 to exclude the original memo
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to search: %v", err)
	}

	// Filter out the original memo and build response
	response := &v1pb.GetRelatedMemosResponse{
		Memos: []*v1pb.SearchResult{},
	}
	for _, r := range results {
		if r.Memo.ID == memo.ID {
			continue
		}
		snippet := r.Memo.Content
		if len(snippet) > 200 {
			snippet = snippet[:200] + "..."
		}
		response.Memos = append(response.Memos, &v1pb.SearchResult{
			Name:    fmt.Sprintf("memos/%s", r.Memo.UID),
			Snippet: snippet,
			Score:   r.Score,
		})
		if len(response.Memos) >= limit {
			break
		}
	}

	return response, nil
}
