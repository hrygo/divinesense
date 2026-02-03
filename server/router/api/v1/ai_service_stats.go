package v1

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	v1pb "github.com/hrygo/divinesense/proto/gen/api/v1"
	"github.com/hrygo/divinesense/server/auth"
	"github.com/hrygo/divinesense/store"
)

// protoTime safely converts time.Time to Unix timestamp, returning 0 for zero times.
func protoTime(t time.Time) int64 {
	if t.IsZero() {
		return 0
	}
	return t.Unix()
}

// GetSessionStats retrieves statistics for a specific session.
func (s *AIService) GetSessionStats(ctx context.Context, req *v1pb.GetSessionStatsRequest) (*v1pb.SessionStats, error) {
	if s.Store == nil {
		return nil, status.Error(codes.Unavailable, "store not available")
	}

	userID := auth.GetUserID(ctx)
	if userID == 0 {
		return nil, status.Error(codes.Unauthenticated, "user not authenticated")
	}

	// Get session stats from store
	stats, err := s.Store.AgentStatsStore.GetSessionStats(ctx, req.SessionId)
	if err != nil {
		slog.Warn("Failed to get session stats",
			"user_id", userID,
			"session_id", req.SessionId,
			"error", err)
		return nil, status.Error(codes.NotFound, fmt.Sprintf("session %s not found", req.SessionId))
	}

	// Verify user owns this session
	if stats.UserID != userID {
		return nil, status.Error(codes.PermissionDenied, "access denied to this session")
	}

	// Convert store stats to proto
	return &v1pb.SessionStats{
		Id:                   stats.ID,
		SessionId:            stats.SessionID,
		ConversationId:       stats.ConversationID,
		UserId:               stats.UserID,
		AgentType:            stats.AgentType,
		StartedAt:            protoTime(stats.StartedAt),
		EndedAt:              protoTime(stats.EndedAt),
		TotalDurationMs:      stats.TotalDurationMs,
		ThinkingDurationMs:   stats.ThinkingDurationMs,
		ToolDurationMs:       stats.ToolDurationMs,
		GenerationDurationMs: stats.GenerationDurationMs,
		InputTokens:          stats.InputTokens,
		OutputTokens:         stats.OutputTokens,
		CacheWriteTokens:     stats.CacheWriteTokens,
		CacheReadTokens:      stats.CacheReadTokens,
		TotalTokens:          stats.TotalTokens,
		TotalCostUsd:         stats.TotalCostUSD,
		ToolCallCount:        stats.ToolCallCount,
		ToolsUsed:            stats.ToolsUsed,
		FilesModified:        stats.FilesModified,
		FilePaths:            stats.FilePaths,
		ModelUsed:            stats.ModelUsed,
		IsError:              stats.IsError,
		ErrorMessage:         stats.ErrorMessage,
		CreatedAt:            protoTime(stats.CreatedAt),
		UpdatedAt:            protoTime(stats.UpdatedAt),
	}, nil
}

// ListSessionStats retrieves session statistics with pagination.
func (s *AIService) ListSessionStats(ctx context.Context, req *v1pb.ListSessionStatsRequest) (*v1pb.ListSessionStatsResponse, error) {
	if s.Store == nil {
		return nil, status.Error(codes.Unavailable, "store not available")
	}

	userID := auth.GetUserID(ctx)
	if userID == 0 {
		return nil, status.Error(codes.Unauthenticated, "user not authenticated")
	}

	// Set defaults
	limit := req.Limit
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	offset := req.Offset
	if offset < 0 {
		offset = 0
	}
	// Prevent unbounded offset queries
	const maxOffset = 10000
	if offset > maxOffset {
		offset = maxOffset
	}

	// Get session stats from store
	sessions, total, err := s.Store.AgentStatsStore.ListSessionStats(ctx, userID, int(limit), int(offset))
	if err != nil {
		slog.Warn("Failed to list session stats",
			"user_id", userID,
			"error", err)
		return nil, status.Error(codes.Internal, "failed to list session stats")
	}

	// Convert store sessions to proto
	pbSessions := make([]*v1pb.SessionStats, len(sessions))
	totalCost := 0.0
	for i, sess := range sessions {
		pbSessions[i] = &v1pb.SessionStats{
			Id:                   sess.ID,
			SessionId:            sess.SessionID,
			ConversationId:       sess.ConversationID,
			UserId:               sess.UserID,
			AgentType:            sess.AgentType,
			StartedAt:            protoTime(sess.StartedAt),
			EndedAt:              protoTime(sess.EndedAt),
			TotalDurationMs:      sess.TotalDurationMs,
			ThinkingDurationMs:   sess.ThinkingDurationMs,
			ToolDurationMs:       sess.ToolDurationMs,
			GenerationDurationMs: sess.GenerationDurationMs,
			InputTokens:          sess.InputTokens,
			OutputTokens:         sess.OutputTokens,
			CacheWriteTokens:     sess.CacheWriteTokens,
			CacheReadTokens:      sess.CacheReadTokens,
			TotalTokens:          sess.TotalTokens,
			TotalCostUsd:         sess.TotalCostUSD,
			ToolCallCount:        sess.ToolCallCount,
			ToolsUsed:            sess.ToolsUsed,
			FilesModified:        sess.FilesModified,
			FilePaths:            sess.FilePaths,
			ModelUsed:            sess.ModelUsed,
			IsError:              sess.IsError,
			ErrorMessage:         sess.ErrorMessage,
			CreatedAt:            protoTime(sess.CreatedAt),
			UpdatedAt:            protoTime(sess.UpdatedAt),
		}
		totalCost += sess.TotalCostUSD
	}

	return &v1pb.ListSessionStatsResponse{
		Sessions:     pbSessions,
		TotalCount:   total,
		TotalCostUsd: totalCost,
	}, nil
}

// GetCostStats retrieves aggregated cost statistics for the user.
func (s *AIService) GetCostStats(ctx context.Context, req *v1pb.GetCostStatsRequest) (*v1pb.CostStats, error) {
	if s.Store == nil {
		return nil, status.Error(codes.Unavailable, "store not available")
	}

	userID := auth.GetUserID(ctx)
	if userID == 0 {
		return nil, status.Error(codes.Unauthenticated, "user not authenticated")
	}

	// Set default days
	days := req.Days
	if days <= 0 || days > 365 {
		days = 7
	}

	// Get cost stats from store
	costStats, err := s.Store.AgentStatsStore.GetCostStats(ctx, userID, int(days))
	if err != nil {
		slog.Warn("Failed to get cost stats",
			"user_id", userID,
			"days", days,
			"error", err)
		return nil, status.Error(codes.Internal, "failed to get cost stats")
	}

	// Convert daily breakdown
	dailyBreakdown := make([]*v1pb.DailyCostData, len(costStats.DailyBreakdown))
	for i, day := range costStats.DailyBreakdown {
		dailyBreakdown[i] = &v1pb.DailyCostData{
			Date:         day.Date,
			CostUsd:      day.CostUSD,
			SessionCount: day.SessionCount,
		}
	}

	// Convert most expensive session
	var mostExpensive *v1pb.SessionStats
	if costStats.MostExpensiveSession != nil {
		sess := costStats.MostExpensiveSession
		mostExpensive = &v1pb.SessionStats{
			Id:                   sess.ID,
			SessionId:            sess.SessionID,
			ConversationId:       sess.ConversationID,
			UserId:               sess.UserID,
			AgentType:            sess.AgentType,
			StartedAt:            protoTime(sess.StartedAt),
			EndedAt:              protoTime(sess.EndedAt),
			TotalDurationMs:      sess.TotalDurationMs,
			ThinkingDurationMs:   sess.ThinkingDurationMs,
			ToolDurationMs:       sess.ToolDurationMs,
			GenerationDurationMs: sess.GenerationDurationMs,
			InputTokens:          sess.InputTokens,
			OutputTokens:         sess.OutputTokens,
			CacheWriteTokens:     sess.CacheWriteTokens,
			CacheReadTokens:      sess.CacheReadTokens,
			TotalTokens:          sess.TotalTokens,
			TotalCostUsd:         sess.TotalCostUSD,
			ToolCallCount:        sess.ToolCallCount,
			ToolsUsed:            sess.ToolsUsed,
			FilesModified:        sess.FilesModified,
			FilePaths:            sess.FilePaths,
			ModelUsed:            sess.ModelUsed,
			IsError:              sess.IsError,
			ErrorMessage:         sess.ErrorMessage,
			CreatedAt:            protoTime(sess.CreatedAt),
			UpdatedAt:            protoTime(sess.UpdatedAt),
		}
	}

	return &v1pb.CostStats{
		TotalCostUsd:         costStats.TotalCostUSD,
		DailyAverageUsd:      costStats.DailyAverageUSD,
		SessionCount:         costStats.SessionCount,
		MostExpensiveSession: mostExpensive,
		DailyBreakdown:       dailyBreakdown,
	}, nil
}

// GetUserCostSettings retrieves user-specific cost control settings.
func (s *AIService) GetUserCostSettings(ctx context.Context, _ *emptypb.Empty) (*v1pb.UserCostSettings, error) {
	if s.Store == nil {
		return nil, status.Error(codes.Unavailable, "store not available")
	}

	userID := auth.GetUserID(ctx)
	if userID == 0 {
		return nil, status.Error(codes.Unauthenticated, "user not authenticated")
	}

	// Get or create user cost settings
	settings, err := s.Store.AgentStatsStore.GetUserCostSettings(ctx, userID)
	if err != nil {
		slog.Warn("Failed to get user cost settings",
			"user_id", userID,
			"error", err)
		return nil, status.Error(codes.Internal, "failed to get cost settings")
	}

	// Handle NULL daily_budget (0 in proto means unlimited, but NULL in DB means unlimited)
	dailyBudget := settings.DailyBudgetUSD
	if dailyBudget == nil {
		zero := 0.0
		dailyBudget = &zero
	}

	var budgetResetAt int64
	if settings.BudgetResetAt != nil {
		budgetResetAt = settings.BudgetResetAt.Unix()
	}

	return &v1pb.UserCostSettings{
		DailyBudgetUsd:         *dailyBudget,
		PerSessionThresholdUsd: settings.PerSessionThresholdUSD,
		AlertEnabled:           settings.AlertEnabled,
		AlertEmail:             settings.AlertEmail,
		AlertInApp:             settings.AlertInApp,
		BudgetResetAt:          budgetResetAt,
	}, nil
}

// SetUserCostSettings updates user-specific cost control settings.
func (s *AIService) SetUserCostSettings(ctx context.Context, req *v1pb.SetUserCostSettingsRequest) (*v1pb.UserCostSettings, error) {
	if s.Store == nil {
		return nil, status.Error(codes.Unavailable, "store not available")
	}

	userID := auth.GetUserID(ctx)
	if userID == 0 {
		return nil, status.Error(codes.Unauthenticated, "user not authenticated")
	}

	// Get current settings first
	currentSettings, err := s.Store.AgentStatsStore.GetUserCostSettings(ctx, userID)
	if err != nil {
		slog.Warn("Failed to get current cost settings",
			"user_id", userID,
			"error", err)
		return nil, status.Error(codes.Internal, "failed to get current settings")
	}

	// Update only provided fields
	updateSettings := &store.UserCostSettings{
		UserID:                 userID,
		DailyBudgetUSD:         currentSettings.DailyBudgetUSD,
		PerSessionThresholdUSD: currentSettings.PerSessionThresholdUSD,
		AlertEnabled:           currentSettings.AlertEnabled,
		AlertEmail:             currentSettings.AlertEmail,
		AlertInApp:             currentSettings.AlertInApp,
		BudgetResetAt:          currentSettings.BudgetResetAt,
	}

	if req.DailyBudgetUsd != nil {
		updateSettings.DailyBudgetUSD = req.DailyBudgetUsd
	}
	if req.PerSessionThresholdUsd != nil {
		updateSettings.PerSessionThresholdUSD = *req.PerSessionThresholdUsd
	}
	if req.AlertEnabled != nil {
		updateSettings.AlertEnabled = *req.AlertEnabled
	}
	if req.AlertEmail != nil {
		updateSettings.AlertEmail = *req.AlertEmail
	}
	if req.AlertInApp != nil {
		updateSettings.AlertInApp = *req.AlertInApp
	}

	// Save settings
	if err := s.Store.AgentStatsStore.SetUserCostSettings(ctx, updateSettings); err != nil {
		slog.Error("Failed to save user cost settings",
			"user_id", userID,
			"error", err)
		return nil, status.Error(codes.Internal, "failed to save cost settings")
	}

	slog.Info("User cost settings updated",
		"user_id", userID,
		"daily_budget", updateSettings.DailyBudgetUSD,
		"per_session_threshold", updateSettings.PerSessionThresholdUSD,
		"alert_enabled", updateSettings.AlertEnabled)

	// Return updated settings
	var budgetResetAt int64
	if updateSettings.BudgetResetAt != nil {
		budgetResetAt = updateSettings.BudgetResetAt.Unix()
	}

	return &v1pb.UserCostSettings{
		DailyBudgetUsd:         *updateSettings.DailyBudgetUSD,
		PerSessionThresholdUsd: updateSettings.PerSessionThresholdUSD,
		AlertEnabled:           updateSettings.AlertEnabled,
		AlertEmail:             updateSettings.AlertEmail,
		AlertInApp:             updateSettings.AlertInApp,
		BudgetResetAt:          budgetResetAt,
	}, nil
}
