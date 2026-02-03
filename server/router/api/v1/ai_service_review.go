// Package v1 - review system handlers for P3-C002.
package v1

import (
	"context"
	"log/slog"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/hrygo/divinesense/ai/review"
	v1pb "github.com/hrygo/divinesense/proto/gen/api/v1"
)

// GetDueReviews returns memos that are due for review.
func (s *AIService) GetDueReviews(ctx context.Context, req *v1pb.GetDueReviewsRequest) (*v1pb.GetDueReviewsResponse, error) {
	if !s.IsEnabled() {
		return nil, status.Errorf(codes.Unavailable, "AI features are disabled")
	}

	// Get current user
	user, err := getCurrentUser(ctx, s.Store)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "unauthorized")
	}

	// Create review service
	reviewService := review.NewService(s.Store)

	// Get limit from request (default: 20)
	limit := int(req.GetLimit())
	if limit <= 0 {
		limit = 20
	}

	// Get due reviews
	items, totalDue, err := reviewService.GetDueReviews(ctx, user.ID, limit)
	if err != nil {
		slog.Error("failed to get due reviews", "user_id", user.ID, "error", err)
		return nil, status.Error(codes.Internal, "failed to get due reviews")
	}

	// Convert to response
	response := &v1pb.GetDueReviewsResponse{
		Items:    make([]*v1pb.ReviewItem, 0, len(items)),
		TotalDue: int32(totalDue),
	}

	for _, item := range items {
		response.Items = append(response.Items, &v1pb.ReviewItem{
			MemoUid:      item.MemoID,
			MemoName:     item.MemoName,
			Title:        item.Title,
			Snippet:      item.Snippet,
			Tags:         item.Tags,
			LastReviewTs: item.LastReview.Unix(),
			ReviewCount:  int32(item.ReviewCount),
			NextReviewTs: item.NextReview.Unix(),
			Priority:     item.Priority,
			CreatedTs:    item.CreatedAt.Unix(),
		})
	}

	return response, nil
}

// RecordReview records a review result and updates spaced repetition state.
func (s *AIService) RecordReview(ctx context.Context, req *v1pb.RecordReviewRequest) (*emptypb.Empty, error) {
	if !s.IsEnabled() {
		return nil, status.Errorf(codes.Unavailable, "AI features are disabled")
	}

	// Get current user
	user, err := getCurrentUser(ctx, s.Store)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "unauthorized")
	}

	// Validate memo UID
	memoUID := req.GetMemoUid()
	if memoUID == "" {
		return nil, status.Error(codes.InvalidArgument, "memo_uid is required")
	}

	// Map proto quality to service quality
	quality, err := mapProtoQualityToService(req.GetQuality())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// Create review service and record review
	reviewService := review.NewService(s.Store)
	if err := reviewService.RecordReview(ctx, user.ID, memoUID, quality); err != nil {
		slog.Error("failed to record review", "user_id", user.ID, "memo_uid", memoUID, "error", err)
		return nil, status.Error(codes.Internal, "failed to record review")
	}

	slog.Info("review recorded", "user_id", user.ID, "memo_uid", memoUID, "quality", quality)
	return &emptypb.Empty{}, nil
}

// GetReviewStats returns review statistics for the current user.
func (s *AIService) GetReviewStats(ctx context.Context, req *v1pb.GetReviewStatsRequest) (*v1pb.GetReviewStatsResponse, error) {
	if !s.IsEnabled() {
		return nil, status.Errorf(codes.Unavailable, "AI features are disabled")
	}

	// Get current user
	user, err := getCurrentUser(ctx, s.Store)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "unauthorized")
	}

	// Create review service
	reviewService := review.NewService(s.Store)

	// Get stats
	stats, err := reviewService.GetReviewStats(ctx, user.ID)
	if err != nil {
		slog.Error("failed to get review stats", "user_id", user.ID, "error", err)
		return nil, status.Error(codes.Internal, "failed to get review stats")
	}

	return &v1pb.GetReviewStatsResponse{
		TotalMemos:      int32(stats.TotalMemos),
		DueToday:        int32(stats.DueToday),
		ReviewedToday:   int32(stats.ReviewedToday),
		NewMemos:        int32(stats.NewMemos),
		MasteredMemos:   int32(stats.MasteredMemos),
		StreakDays:      int32(stats.StreakDays),
		TotalReviews:    int32(stats.TotalReviews),
		AverageAccuracy: int32(stats.AverageAccuracy),
	}, nil
}

// Service: Again=0, Hard=1, Good=2, Easy=3.
func mapProtoQualityToService(pq v1pb.ReviewQuality) (review.ReviewQuality, error) {
	switch pq {
	case v1pb.ReviewQuality_REVIEW_QUALITY_AGAIN:
		return review.QualityAgain, nil
	case v1pb.ReviewQuality_REVIEW_QUALITY_HARD:
		return review.QualityHard, nil
	case v1pb.ReviewQuality_REVIEW_QUALITY_GOOD:
		return review.QualityGood, nil
	case v1pb.ReviewQuality_REVIEW_QUALITY_EASY:
		return review.QualityEasy, nil
	default:
		return 0, status.Errorf(codes.InvalidArgument, "invalid review quality: %v", pq)
	}
}
