// Package v1 - duplicate detection handlers for P2-C002.
package v1

import (
	"context"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/hrygo/divinesense/ai/duplicate"
	v1pb "github.com/hrygo/divinesense/proto/gen/api/v1"
)

const (
	maxContentLength = 5000
	minContentLength = 3
)

// DetectDuplicates checks for duplicate or related memos.
func (s *AIService) DetectDuplicates(ctx context.Context, req *v1pb.DetectDuplicatesRequest) (*v1pb.DetectDuplicatesResponse, error) {
	if !s.IsEnabled() {
		return nil, status.Errorf(codes.Unavailable, "AI features are disabled")
	}

	// Get current user
	user, err := getCurrentUser(ctx, s.Store)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "unauthorized")
	}

	// Validate content
	content := strings.TrimSpace(req.Content)
	contentLen := len([]rune(content))
	if contentLen < minContentLength {
		return nil, status.Errorf(codes.InvalidArgument, "content too short (min %d characters)", minContentLength)
	}
	if contentLen > maxContentLength {
		return nil, status.Errorf(codes.InvalidArgument, "content too long (max %d characters)", maxContentLength)
	}

	topK := int(req.TopK)
	if topK <= 0 {
		topK = duplicate.DefaultTopK
	}

	// Create detector
	detector := duplicate.NewDuplicateDetector(s.Store, s.EmbeddingService, s.EmbeddingModel)

	// Detect duplicates
	result, err := detector.Detect(ctx, &duplicate.DetectRequest{
		UserID:  user.ID,
		Title:   req.Title,
		Content: content,
		Tags:    req.Tags,
		TopK:    topK,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "detection failed: %v", err)
	}

	// Convert to response
	response := &v1pb.DetectDuplicatesResponse{
		HasDuplicate: result.HasDuplicate,
		HasRelated:   result.HasRelated,
		LatencyMs:    result.LatencyMs,
	}

	for _, d := range result.Duplicates {
		response.Duplicates = append(response.Duplicates, convertSimilarMemo(&d))
	}
	for _, r := range result.Related {
		response.Related = append(response.Related, convertSimilarMemo(&r))
	}

	return response, nil
}

// MergeMemos merges source memo into target memo.
func (s *AIService) MergeMemos(ctx context.Context, req *v1pb.MergeMemosRequest) (*v1pb.MergeMemosResponse, error) {
	if !s.IsEnabled() {
		return nil, status.Errorf(codes.Unavailable, "AI features are disabled")
	}

	// Get current user
	user, err := getCurrentUser(ctx, s.Store)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "unauthorized")
	}

	// Validate inputs
	sourceUID := extractMemoUID(req.SourceName)
	if sourceUID == "" {
		return nil, status.Errorf(codes.InvalidArgument, "invalid source_name format")
	}
	targetUID := extractMemoUID(req.TargetName)
	if targetUID == "" {
		return nil, status.Errorf(codes.InvalidArgument, "invalid target_name format")
	}

	// Create detector and merge
	detector := duplicate.NewDuplicateDetector(s.Store, s.EmbeddingService, s.EmbeddingModel)
	err = detector.Merge(ctx, user.ID, sourceUID, targetUID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "merge failed: %v", err)
	}

	return &v1pb.MergeMemosResponse{
		MergedName: req.TargetName,
	}, nil
}

// LinkMemos creates a bidirectional relation between two memos.
func (s *AIService) LinkMemos(ctx context.Context, req *v1pb.LinkMemosRequest) (*v1pb.LinkMemosResponse, error) {
	if !s.IsEnabled() {
		return nil, status.Errorf(codes.Unavailable, "AI features are disabled")
	}

	// Get current user
	user, err := getCurrentUser(ctx, s.Store)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "unauthorized")
	}

	// Validate inputs
	uid1 := extractMemoUID(req.MemoName_1)
	if uid1 == "" {
		return nil, status.Errorf(codes.InvalidArgument, "invalid memo_name_1 format")
	}
	uid2 := extractMemoUID(req.MemoName_2)
	if uid2 == "" {
		return nil, status.Errorf(codes.InvalidArgument, "invalid memo_name_2 format")
	}

	// Create detector and link
	detector := duplicate.NewDuplicateDetector(s.Store, s.EmbeddingService, s.EmbeddingModel)
	err = detector.Link(ctx, user.ID, uid1, uid2)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "link failed: %v", err)
	}

	return &v1pb.LinkMemosResponse{
		Success: true,
	}, nil
}

// extractMemoUID extracts UID from resource name "memos/{uid}".
func extractMemoUID(name string) string {
	if !strings.HasPrefix(name, "memos/") {
		return ""
	}
	return strings.TrimPrefix(name, "memos/")
}

// convertSimilarMemo converts duplicate.SimilarMemo to proto.
func convertSimilarMemo(m *duplicate.SimilarMemo) *v1pb.SimilarMemo {
	result := &v1pb.SimilarMemo{
		Id:         m.ID,
		Name:       "memos/" + m.Name,
		Title:      m.Title,
		Snippet:    m.Snippet,
		Similarity: m.Similarity,
		SharedTags: m.SharedTags,
		Level:      m.Level,
	}

	if m.Breakdown != nil {
		result.Breakdown = &v1pb.SimilarityBreakdown{
			Vector:     m.Breakdown.Vector,
			TagCoOccur: m.Breakdown.TagCoOccur,
			TimeProx:   m.Breakdown.TimeProx,
		}
	}

	return result
}
