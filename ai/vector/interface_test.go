package vector

import (
	"context"
	"testing"
)

// TestVectorServiceContract tests the VectorService contract.
func TestVectorServiceContract(t *testing.T) {
	ctx := context.Background()
	svc := NewMockVectorService()

	t.Run("StoreEmbedding_StoresData", func(t *testing.T) {
		docID := "test-doc-001"
		vector := []float32{0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8}
		metadata := map[string]any{
			"user_id": int32(1),
			"content": "Test content",
		}

		err := svc.StoreEmbedding(ctx, docID, vector, metadata)
		if err != nil {
			t.Fatalf("StoreEmbedding failed: %v", err)
		}

		// Verify by searching
		results, err := svc.SearchSimilar(ctx, vector, 1, nil)
		if err != nil {
			t.Fatalf("SearchSimilar failed: %v", err)
		}
		if len(results) == 0 {
			t.Error("expected at least one result after storing")
		}
	})

	t.Run("SearchSimilar_ReturnsSortedResults", func(t *testing.T) {
		// Use a vector similar to memo-001
		queryVector := []float32{0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8}

		results, err := svc.SearchSimilar(ctx, queryVector, 3, nil)
		if err != nil {
			t.Fatalf("SearchSimilar failed: %v", err)
		}
		if len(results) == 0 {
			t.Error("expected results from search")
		}

		// Check results are sorted by score descending
		for i := 1; i < len(results); i++ {
			if results[i].Score > results[i-1].Score {
				t.Error("results should be sorted by score descending")
			}
		}
	})

	t.Run("SearchSimilar_RespectsLimit", func(t *testing.T) {
		queryVector := []float32{0.5, 0.5, 0.5, 0.5, 0.5, 0.5, 0.5, 0.5}

		results, err := svc.SearchSimilar(ctx, queryVector, 2, nil)
		if err != nil {
			t.Fatalf("SearchSimilar failed: %v", err)
		}
		if len(results) > 2 {
			t.Errorf("expected at most 2 results, got %d", len(results))
		}
	})

	t.Run("SearchSimilar_AppliesFilter", func(t *testing.T) {
		queryVector := []float32{0.5, 0.5, 0.5, 0.5, 0.5, 0.5, 0.5, 0.5}
		filter := map[string]any{"user_id": int32(1)}

		results, err := svc.SearchSimilar(ctx, queryVector, 10, filter)
		if err != nil {
			t.Fatalf("SearchSimilar failed: %v", err)
		}

		// All results should have user_id = 1
		for _, r := range results {
			if userID, ok := r.Metadata["user_id"].(int32); ok {
				if userID != 1 {
					t.Errorf("filter not applied, got user_id %d", userID)
				}
			}
		}
	})

	t.Run("SearchSimilar_ScoresInRange", func(t *testing.T) {
		queryVector := []float32{0.5, 0.5, 0.5, 0.5, 0.5, 0.5, 0.5, 0.5}

		results, err := svc.SearchSimilar(ctx, queryVector, 10, nil)
		if err != nil {
			t.Fatalf("SearchSimilar failed: %v", err)
		}

		for _, r := range results {
			if r.Score < 0 || r.Score > 1 {
				t.Errorf("score out of range [0,1]: %f", r.Score)
			}
		}
	})

	t.Run("HybridSearch_ReturnsResults", func(t *testing.T) {
		results, err := svc.HybridSearch(ctx, "项目", 5)
		if err != nil {
			t.Fatalf("HybridSearch failed: %v", err)
		}
		if len(results) == 0 {
			t.Error("expected results from hybrid search")
		}
	})

	t.Run("HybridSearch_KeywordMatchHigherScore", func(t *testing.T) {
		results, err := svc.HybridSearch(ctx, "Sprint", 10)
		if err != nil {
			t.Fatalf("HybridSearch failed: %v", err)
		}

		// Results with keyword match should have higher scores
		var keywordMatchFound bool
		for _, r := range results {
			if r.MatchType == "keyword" || r.MatchType == "hybrid" {
				keywordMatchFound = true
				if r.Score < 0.7 {
					t.Errorf("keyword match should have high score, got %f", r.Score)
				}
			}
		}
		if !keywordMatchFound && len(results) > 0 {
			t.Log("no exact keyword match found, which is acceptable for partial matches")
		}
	})

	t.Run("HybridSearch_RespectsLimit", func(t *testing.T) {
		results, err := svc.HybridSearch(ctx, "笔记", 2)
		if err != nil {
			t.Fatalf("HybridSearch failed: %v", err)
		}
		if len(results) > 2 {
			t.Errorf("expected at most 2 results, got %d", len(results))
		}
	})

	t.Run("HybridSearch_ValidMatchTypes", func(t *testing.T) {
		validMatchTypes := map[string]bool{"vector": true, "keyword": true, "hybrid": true}

		results, err := svc.HybridSearch(ctx, "test", 10)
		if err != nil {
			t.Fatalf("HybridSearch failed: %v", err)
		}

		for _, r := range results {
			if !validMatchTypes[r.MatchType] {
				t.Errorf("invalid match_type: %s", r.MatchType)
			}
		}
	})

	t.Run("VectorResult_HasRequiredFields", func(t *testing.T) {
		queryVector := []float32{0.5, 0.5, 0.5, 0.5, 0.5, 0.5, 0.5, 0.5}
		results, err := svc.SearchSimilar(ctx, queryVector, 1, nil)
		if err != nil {
			t.Fatalf("SearchSimilar failed: %v", err)
		}
		if len(results) > 0 {
			r := results[0]
			if r.DocID == "" {
				t.Error("DocID should not be empty")
			}
			if r.Metadata == nil {
				t.Error("Metadata should not be nil")
			}
		}
	})

	t.Run("SearchResult_HasRequiredFields", func(t *testing.T) {
		results, err := svc.HybridSearch(ctx, "test", 1)
		if err != nil {
			t.Fatalf("HybridSearch failed: %v", err)
		}
		if len(results) > 0 {
			r := results[0]
			if r.Name == "" {
				t.Error("Name should not be empty")
			}
			if r.MatchType == "" {
				t.Error("MatchType should not be empty")
			}
		}
	})
}
