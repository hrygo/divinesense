// Package stats provides cost alerting for agent sessions.
package stats

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/hrygo/divinesense/ai/agent"
	"github.com/hrygo/divinesense/store"
)

// CostAlertService checks for cost threshold violations and sends alerts.
// CostAlertService 检查成本阈值违规并发送告警。
type CostAlertService struct {
	store    store.AgentStatsStore
	notifier AlertNotifier
	logger   *slog.Logger
}

// AlertNotifier is an interface for sending cost alerts to users.
// AlertNotifier 是向用户发送成本告警的接口。
type AlertNotifier interface {
	SendCostAlert(ctx context.Context, userID int32, alert *CostAlert) error
}

// CostAlert represents a cost threshold violation.
// CostAlert 表示成本阈值违规。
type CostAlert struct {
	Type         string // "session_threshold_exceeded", "daily_budget_exceeded"
	SessionID    string // For session-specific alerts
	CostUSD      float64
	ThresholdUSD float64
	DailyCostUSD float64
	BudgetUSD    float64
	OverByUSD    float64
	Timestamp    time.Time
}

// NewCostAlertService creates a new cost alert service.
func NewCostAlertService(store store.AgentStatsStore, notifier AlertNotifier, logger *slog.Logger) *CostAlertService {
	if logger == nil {
		logger = slog.Default()
	}
	return &CostAlertService{
		store:    store,
		notifier: notifier,
		logger:   logger,
	}
}

// CheckSessionCost checks if a session exceeds cost thresholds and sends alerts.
// CheckSessionCost 检查会话是否超过成本阈值并发送告警。
func (s *CostAlertService) CheckSessionCost(ctx context.Context, stats *agent.SessionStatsData) error {
	if stats == nil {
		return nil
	}

	// Get user's cost settings
	settings, err := s.store.GetUserCostSettings(ctx, stats.UserID)
	if err != nil {
		s.logger.Warn("CostAlert: failed to get user cost settings",
			"user_id", stats.UserID,
			"error", err)
		return nil // Don't fail on settings errors
	}

	// Check if alerts are enabled
	if !settings.AlertEnabled {
		return nil
	}

	// 1. Check per-session threshold
	if stats.TotalCostUSD > settings.PerSessionThresholdUSD {
		s.logger.Info("CostAlert: session threshold exceeded",
			"user_id", stats.UserID,
			"session_id", stats.SessionID,
			"cost_usd", stats.TotalCostUSD,
			"threshold_usd", settings.PerSessionThresholdUSD)

		alert := &CostAlert{
			Type:         "session_threshold_exceeded",
			SessionID:    stats.SessionID,
			CostUSD:      stats.TotalCostUSD,
			ThresholdUSD: settings.PerSessionThresholdUSD,
			Timestamp:    time.Now(),
		}

		if err := s.notifier.SendCostAlert(ctx, stats.UserID, alert); err != nil {
			s.logger.Error("CostAlert: failed to send session threshold alert",
				"user_id", stats.UserID,
				"error", err)
		}
	}

	// 2. Check daily budget
	if settings.DailyBudgetUSD != nil && *settings.DailyBudgetUSD > 0 {
		today := time.Now().Truncate(24 * time.Hour)
		tomorrow := today.Add(24 * time.Hour)

		dailyCost, err := s.store.GetDailyCostUsage(ctx, stats.UserID, today, tomorrow)
		if err != nil {
			s.logger.Warn("CostAlert: failed to get daily cost usage",
				"user_id", stats.UserID,
				"error", err)
			return nil
		}

		// Include current session cost
		totalDailyCost := dailyCost + stats.TotalCostUSD
		remainingBudget := *settings.DailyBudgetUSD - totalDailyCost

		if remainingBudget < 0 {
			s.logger.Info("CostAlert: daily budget exceeded",
				"user_id", stats.UserID,
				"daily_cost_usd", totalDailyCost,
				"budget_usd", *settings.DailyBudgetUSD,
				"over_by_usd", -remainingBudget)

			alert := &CostAlert{
				Type:         "daily_budget_exceeded",
				DailyCostUSD: totalDailyCost,
				BudgetUSD:    *settings.DailyBudgetUSD,
				OverByUSD:    -remainingBudget,
				Timestamp:    time.Now(),
			}

			if err := s.notifier.SendCostAlert(ctx, stats.UserID, alert); err != nil {
				s.logger.Error("CostAlert: failed to send daily budget alert",
					"user_id", stats.UserID,
					"error", err)
			}
		} else if remainingBudget < (*settings.DailyBudgetUSD * 0.1) {
			// Warning: less than 10% budget remaining
			s.logger.Info("CostAlert: daily budget warning",
				"user_id", stats.UserID,
				"daily_cost_usd", totalDailyCost,
				"budget_usd", *settings.DailyBudgetUSD,
				"remaining_usd", remainingBudget)

			alert := &CostAlert{
				Type:         "daily_budget_warning",
				DailyCostUSD: totalDailyCost,
				BudgetUSD:    *settings.DailyBudgetUSD,
				OverByUSD:    remainingBudget,
				Timestamp:    time.Now(),
			}

			if err := s.notifier.SendCostAlert(ctx, stats.UserID, alert); err != nil {
				s.logger.Error("CostAlert: failed to send daily budget warning",
					"user_id", stats.UserID,
					"error", err)
			}
		}
	}

	return nil
}

// GetCostSummary retrieves a cost summary for the user.
// GetCostSummary 检索用户的成本摘要。
func (s *CostAlertService) GetCostSummary(ctx context.Context, userID int32, days int) (*store.CostStats, error) {
	return s.store.GetCostStats(ctx, userID, days)
}

// GetDailyCost retrieves today's cost for a user.
// GetDailyCost 检索用户今天的成本。
func (s *CostAlertService) GetDailyCost(ctx context.Context, userID int32) (float64, error) {
	today := time.Now().Truncate(24 * time.Hour)
	tomorrow := today.Add(24 * time.Hour)
	return s.store.GetDailyCostUsage(ctx, userID, today, tomorrow)
}

// String returns a string representation of the alert.
func (a *CostAlert) String() string {
	switch a.Type {
	case "session_threshold_exceeded":
		return fmt.Sprintf("Session cost $%.4f exceeds threshold $%.4f", a.CostUSD, a.ThresholdUSD)
	case "daily_budget_exceeded":
		return fmt.Sprintf("Daily cost $%.4f exceeds budget $%.4f by $%.4f", a.DailyCostUSD, a.BudgetUSD, a.OverByUSD)
	case "daily_budget_warning":
		return fmt.Sprintf("Daily cost $%.4f, $%.4f budget remaining", a.DailyCostUSD, a.OverByUSD)
	default:
		return fmt.Sprintf("Unknown alert type: %s", a.Type)
	}
}
