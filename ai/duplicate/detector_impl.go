// Package duplicate - detector implementation for P2-C002.
package duplicate

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"

	"github.com/hrygo/divinesense/ai"
	storepb "github.com/hrygo/divinesense/proto/gen/store"
	"github.com/hrygo/divinesense/store"
)

// duplicateDetector implements DuplicateDetector.
type duplicateDetector struct {
	embedding ai.EmbeddingService
	store     *store.Store
	model     string
	weights   Weights
}

// NewDuplicateDetector creates a new DuplicateDetector.
func NewDuplicateDetector(s *store.Store, embedding ai.EmbeddingService, model string) DuplicateDetector {
	return &duplicateDetector{
		store:     s,
		embedding: embedding,
		weights:   DefaultWeights,
		model:     model,
	}
}

// NewDuplicateDetectorWithWeights creates a detector with custom weights.
func NewDuplicateDetectorWithWeights(s *store.Store, embedding ai.EmbeddingService, model string, weights Weights) DuplicateDetector {
	return &duplicateDetector{
		store:     s,
		embedding: embedding,
		weights:   weights,
		model:     model,
	}
}

func (d *duplicateDetector) Detect(ctx context.Context, req *DetectRequest) (*DetectResponse, error) {
	start := time.Now()
	response := &DetectResponse{}

	if req.TopK <= 0 {
		req.TopK = DefaultTopK
	}

	// Step 1: Generate embedding for new content
	contentForEmbed := req.Title + "\n" + req.Content
	queryVector, err := d.embedding.Embed(ctx, contentForEmbed)
	if err != nil {
		slog.Warn("failed to generate embedding for duplicate detection", "error", err)
		// Return empty response on embedding failure (graceful degradation)
		response.LatencyMs = time.Since(start).Milliseconds()
		return response, nil
	}

	// Step 2: Vector search for candidates
	candidates, err := d.store.VectorSearch(ctx, &store.VectorSearchOptions{
		UserID: req.UserID,
		Vector: queryVector,
		Limit:  req.TopK * 2, // Get more candidates for filtering
	})
	if err != nil {
		slog.Warn("vector search failed for duplicate detection", "user_id", req.UserID, "error", err)
		response.LatencyMs = time.Since(start).Milliseconds()
		return response, nil
	}

	if len(candidates) == 0 {
		response.LatencyMs = time.Since(start).Milliseconds()
		return response, nil
	}

	// Step 3: Calculate 3D similarity for each candidate
	var similarities []SimilarMemo
	now := time.Now()

	for _, candidate := range candidates {
		if candidate.Memo == nil {
			continue
		}

		// Get candidate's embedding for precise cosine similarity
		candidateEmbed, err := d.store.GetMemoEmbedding(ctx, candidate.Memo.ID, d.model)
		if err != nil || candidateEmbed == nil {
			// Use vector search score as fallback
			candidateEmbed = &store.MemoEmbedding{Embedding: nil}
		}

		// Calculate breakdown
		breakdown := &Breakdown{}

		// Vector similarity (use search score or compute cosine)
		if len(candidateEmbed.Embedding) > 0 {
			breakdown.Vector = CosineSimilarity(queryVector, candidateEmbed.Embedding)
		} else {
			breakdown.Vector = float64(candidate.Score) // Use search score
		}

		// Tag co-occurrence
		candidateTags := extractTagsFromMemo(candidate.Memo)
		breakdown.TagCoOccur = TagCoOccurrence(req.Tags, candidateTags)

		// Time proximity
		candidateTime := time.Unix(candidate.Memo.CreatedTs, 0)
		breakdown.TimeProx = TimeProximity(now, candidateTime)

		// Calculate weighted score
		score := CalculateWeightedSimilarity(breakdown, d.weights)

		if score >= RelatedThreshold {
			level := "related"
			if score >= DuplicateThreshold {
				level = "duplicate"
			}

			similarities = append(similarities, SimilarMemo{
				ID:         fmt.Sprintf("%d", candidate.Memo.ID),
				Name:       candidate.Memo.UID,
				Title:      ExtractTitle(candidate.Memo.Content),
				Snippet:    Truncate(candidate.Memo.Content, 100),
				Similarity: score,
				SharedTags: FindSharedTags(req.Tags, candidateTags),
				Level:      level,
				Breakdown:  breakdown,
			})
		}
	}

	// Step 4: Sort by similarity (descending)
	sort.Slice(similarities, func(i, j int) bool {
		return similarities[i].Similarity > similarities[j].Similarity
	})

	// Step 5: Categorize
	for _, sim := range similarities {
		if sim.Level == "duplicate" {
			response.Duplicates = append(response.Duplicates, sim)
			response.HasDuplicate = true
		} else {
			response.Related = append(response.Related, sim)
			response.HasRelated = true
		}
	}

	// Limit results
	if len(response.Duplicates) > req.TopK {
		response.Duplicates = response.Duplicates[:req.TopK]
	}
	if len(response.Related) > req.TopK {
		response.Related = response.Related[:req.TopK]
	}

	response.LatencyMs = time.Since(start).Milliseconds()
	return response, nil
}

func (d *duplicateDetector) Merge(ctx context.Context, userID int32, sourceID, targetID string) error {
	// Get source memo
	source, err := d.getMemoByUID(ctx, userID, sourceID)
	if err != nil {
		return fmt.Errorf("get source memo: %w", err)
	}

	// Get target memo
	target, err := d.getMemoByUID(ctx, userID, targetID)
	if err != nil {
		return fmt.Errorf("get target memo: %w", err)
	}

	// Merge content
	mergedContent := target.Content + "\n\n---\n\n" + source.Content

	// Merge tags
	sourceTags := extractTagsFromMemo(source)
	targetTags := extractTagsFromMemo(target)
	mergedTags := mergeTags(sourceTags, targetTags)

	// Build updated payload with merged tags
	updatedPayload := &storepb.MemoPayload{Tags: mergedTags}
	if target.Payload != nil {
		updatedPayload.Property = target.Payload.Property
		updatedPayload.Location = target.Payload.Location
	}

	// Update target memo with merged content and tags
	err = d.store.UpdateMemo(ctx, &store.UpdateMemo{
		ID:      target.ID,
		Content: &mergedContent,
		Payload: updatedPayload,
	})
	if err != nil {
		return fmt.Errorf("update target memo: %w", err)
	}

	// Archive source memo (set to ARCHIVED status)
	archived := store.Archived
	err = d.store.UpdateMemo(ctx, &store.UpdateMemo{
		ID:        source.ID,
		RowStatus: &archived,
	})
	if err != nil {
		return fmt.Errorf("archive source memo: %w", err)
	}

	slog.Info("merged memos",
		"source_id", sourceID,
		"target_id", targetID,
		"merged_tags", mergedTags)

	return nil
}

func (d *duplicateDetector) Link(ctx context.Context, userID int32, memoID1, memoID2 string) error {
	// Get both memos to verify ownership
	memo1, err := d.getMemoByUID(ctx, userID, memoID1)
	if err != nil {
		return fmt.Errorf("get memo1: %w", err)
	}

	memo2, err := d.getMemoByUID(ctx, userID, memoID2)
	if err != nil {
		return fmt.Errorf("get memo2: %w", err)
	}

	// Create bidirectional relation
	_, err = d.store.UpsertMemoRelation(ctx, &store.MemoRelation{
		MemoID:        memo1.ID,
		RelatedMemoID: memo2.ID,
		Type:          store.MemoRelationReference,
	})
	if err != nil {
		return fmt.Errorf("create relation 1->2: %w", err)
	}

	_, err = d.store.UpsertMemoRelation(ctx, &store.MemoRelation{
		MemoID:        memo2.ID,
		RelatedMemoID: memo1.ID,
		Type:          store.MemoRelationReference,
	})
	if err != nil {
		return fmt.Errorf("create relation 2->1: %w", err)
	}

	slog.Info("linked memos", "memo1", memoID1, "memo2", memoID2)
	return nil
}

// getMemoByUID retrieves a memo by UID and verifies ownership.
func (d *duplicateDetector) getMemoByUID(ctx context.Context, userID int32, uid string) (*store.Memo, error) {
	memos, err := d.store.ListMemos(ctx, &store.FindMemo{
		UID:       &uid,
		CreatorID: &userID,
	})
	if err != nil {
		return nil, err
	}
	if len(memos) == 0 {
		return nil, fmt.Errorf("memo not found: %s", uid)
	}
	return memos[0], nil
}

// extractTagsFromMemo extracts tags from memo content using payload.
func extractTagsFromMemo(memo *store.Memo) []string {
	if memo == nil || memo.Payload == nil {
		return nil
	}
	return memo.Payload.Tags
}

// mergeTags combines two tag slices, removing duplicates (case-insensitive).
func mergeTags(tags1, tags2 []string) []string {
	seen := make(map[string]bool)
	var result []string

	for _, tag := range tags1 {
		lower := strings.ToLower(tag)
		if !seen[lower] {
			seen[lower] = true
			result = append(result, tag)
		}
	}

	for _, tag := range tags2 {
		lower := strings.ToLower(tag)
		if !seen[lower] {
			seen[lower] = true
			result = append(result, tag)
		}
	}

	return result
}
