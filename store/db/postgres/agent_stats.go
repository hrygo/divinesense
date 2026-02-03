package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"database/sql"

	"github.com/hrygo/divinesense/store"
	"github.com/lib/pq"
)

// SaveSessionStats saves a session statistics record.
func (d *DB) SaveSessionStats(ctx context.Context, stats *store.AgentSessionStats) error {
	query := `
		INSERT INTO agent_session_stats (
			session_id, conversation_id, user_id, agent_type,
			started_at, ended_at, total_duration_ms,
			thinking_duration_ms, tool_duration_ms, generation_duration_ms,
			input_tokens, output_tokens, cache_write_tokens, cache_read_tokens, total_tokens,
			total_cost_usd, tool_call_count, tools_used, files_modified, file_paths,
			model_used, is_error, error_message
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23)
		ON CONFLICT (session_id) DO UPDATE SET
			ended_at = EXCLUDED.ended_at,
			total_duration_ms = EXCLUDED.total_duration_ms,
			thinking_duration_ms = EXCLUDED.thinking_duration_ms,
			tool_duration_ms = EXCLUDED.tool_duration_ms,
			generation_duration_ms = EXCLUDED.generation_duration_ms,
			input_tokens = EXCLUDED.input_tokens,
			output_tokens = EXCLUDED.output_tokens,
			cache_write_tokens = EXCLUDED.cache_write_tokens,
			cache_read_tokens = EXCLUDED.cache_read_tokens,
			total_tokens = EXCLUDED.total_tokens,
			total_cost_usd = EXCLUDED.total_cost_usd,
			tool_call_count = EXCLUDED.tool_call_count,
			tools_used = EXCLUDED.tools_used,
			files_modified = EXCLUDED.files_modified,
			file_paths = EXCLUDED.file_paths,
			model_used = EXCLUDED.model_used,
			is_error = EXCLUDED.is_error,
			error_message = EXCLUDED.error_message,
			updated_at = NOW()
		RETURNING id, created_at, updated_at
	`

	// tools_used is JSONB - encode as JSON
	// Always provide valid JSON (empty array [] when no tools used)
	// 始终提供有效的 JSON（没有工具时使用空数组 []）
	var toolsUsedJSON []byte
	if len(stats.ToolsUsed) > 0 {
		var err error
		toolsUsedJSON, err = json.Marshal(stats.ToolsUsed)
		if err != nil {
			return fmt.Errorf("failed to marshal tools_used: %w", err)
		}
	} else {
		toolsUsedJSON = []byte("[]") // Empty JSON array for no tools
	}

	// file_paths is TEXT[] - use pq.Array
	// Always provide a valid array (empty array when no files)
	// 始终提供有效的数组（没有文件时使用空数组）
	var filePathsArray interface{}
	if len(stats.FilePaths) > 0 {
		filePathsArray = pq.Array(stats.FilePaths)
	} else {
		filePathsArray = pq.Array([]string{}) // Empty PostgreSQL array
	}

	err := d.db.QueryRowContext(ctx, query,
		stats.SessionID,
		stats.ConversationID,
		stats.UserID,
		stats.AgentType,
		stats.StartedAt,
		stats.EndedAt,
		stats.TotalDurationMs,
		stats.ThinkingDurationMs,
		stats.ToolDurationMs,
		stats.GenerationDurationMs,
		stats.InputTokens,
		stats.OutputTokens,
		stats.CacheWriteTokens,
		stats.CacheReadTokens,
		stats.TotalTokens,
		stats.TotalCostUSD,
		stats.ToolCallCount,
		toolsUsedJSON,
		stats.FilesModified,
		filePathsArray,
		stats.ModelUsed,
		stats.IsError,
		stats.ErrorMessage,
	).Scan(&stats.ID, &stats.CreatedAt, &stats.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to save session stats: %w", err)
	}

	return nil
}

// GetSessionStats retrieves stats by session ID.
func (d *DB) GetSessionStats(ctx context.Context, sessionID string) (*store.AgentSessionStats, error) {
	query := `
		SELECT id, session_id, conversation_id, user_id, agent_type,
			   started_at, ended_at, total_duration_ms,
			   thinking_duration_ms, tool_duration_ms, generation_duration_ms,
			   input_tokens, output_tokens, cache_write_tokens, cache_read_tokens, total_tokens,
			   total_cost_usd, tool_call_count, tools_used, files_modified, file_paths,
			   model_used, is_error, error_message, created_at, updated_at
		FROM agent_session_stats
		WHERE session_id = $1
	`

	var stats store.AgentSessionStats
	var toolsUsedJSONB []byte
	var filePathsArray []string

	err := d.db.QueryRowContext(ctx, query, sessionID).Scan(
		&stats.ID,
		&stats.SessionID,
		&stats.ConversationID,
		&stats.UserID,
		&stats.AgentType,
		&stats.StartedAt,
		&stats.EndedAt,
		&stats.TotalDurationMs,
		&stats.ThinkingDurationMs,
		&stats.ToolDurationMs,
		&stats.GenerationDurationMs,
		&stats.InputTokens,
		&stats.OutputTokens,
		&stats.CacheWriteTokens,
		&stats.CacheReadTokens,
		&stats.TotalTokens,
		&stats.TotalCostUSD,
		&stats.ToolCallCount,
		&toolsUsedJSONB,
		&stats.FilesModified,
		&filePathsArray,
		&stats.ModelUsed,
		&stats.IsError,
		&stats.ErrorMessage,
		&stats.CreatedAt,
		&stats.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get session stats: %w", err)
	}

	// Parse JSONB tools_used
	if toolsUsedJSONB != nil {
		stats.ToolsUsed = parseStringArray(toolsUsedJSONB)
	}

	stats.FilePaths = filePathsArray

	return &stats, nil
}

// ListSessionStats retrieves stats for a user with pagination.
func (d *DB) ListSessionStats(ctx context.Context, userID int32, limit, offset int) ([]*store.AgentSessionStats, int64, error) {
	// Get total count
	var total int64
	err := d.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM agent_session_stats WHERE user_id = $1", userID).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count session stats: %w", err)
	}

	query := `
		SELECT id, session_id, conversation_id, user_id, agent_type,
			   started_at, ended_at, total_duration_ms,
			   thinking_duration_ms, tool_duration_ms, generation_duration_ms,
			   input_tokens, output_tokens, cache_write_tokens, cache_read_tokens, total_tokens,
			   total_cost_usd, tool_call_count, tools_used, files_modified, file_paths,
			   model_used, is_error, error_message, created_at, updated_at
		FROM agent_session_stats
		WHERE user_id = $1
		ORDER BY started_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := d.db.QueryContext(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list session stats: %w", err)
	}
	defer rows.Close()

	var statsList []*store.AgentSessionStats
	for rows.Next() {
		var stats store.AgentSessionStats
		var toolsUsedJSONB []byte
		var filePathsArray []string

		err := rows.Scan(
			&stats.ID,
			&stats.SessionID,
			&stats.ConversationID,
			&stats.UserID,
			&stats.AgentType,
			&stats.StartedAt,
			&stats.EndedAt,
			&stats.TotalDurationMs,
			&stats.ThinkingDurationMs,
			&stats.ToolDurationMs,
			&stats.GenerationDurationMs,
			&stats.InputTokens,
			&stats.OutputTokens,
			&stats.CacheWriteTokens,
			&stats.CacheReadTokens,
			&stats.TotalTokens,
			&stats.TotalCostUSD,
			&stats.ToolCallCount,
			&toolsUsedJSONB,
			&stats.FilesModified,
			&filePathsArray,
			&stats.ModelUsed,
			&stats.IsError,
			&stats.ErrorMessage,
			&stats.CreatedAt,
			&stats.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan session stats: %w", err)
		}

		if toolsUsedJSONB != nil {
			stats.ToolsUsed = parseStringArray(toolsUsedJSONB)
		}
		stats.FilePaths = filePathsArray

		statsList = append(statsList, &stats)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating session stats: %w", err)
	}

	return statsList, total, nil
}

// GetDailyCostUsage retrieves total cost for a user in a date range.
func (d *DB) GetDailyCostUsage(ctx context.Context, userID int32, startDate, endDate time.Time) (float64, error) {
	query := `
		SELECT COALESCE(SUM(total_cost_usd), 0) as total_cost
		FROM agent_session_stats
		WHERE user_id = $1
		  AND started_at >= $2
		  AND started_at < $3
		  AND is_error = false
	`

	var totalCost float64
	err := d.db.QueryRowContext(ctx, query, userID, startDate, endDate).Scan(&totalCost)
	if err != nil {
		return 0, fmt.Errorf("failed to get daily cost usage: %w", err)
	}

	return totalCost, nil
}

// GetCostStats retrieves aggregated cost statistics.
func (d *DB) GetCostStats(ctx context.Context, userID int32, days int) (*store.CostStats, error) {
	startDate := time.Now().AddDate(0, 0, -days).Truncate(24 * time.Hour)

	query := `
		SELECT
			COALESCE(SUM(total_cost_usd), 0) as total_cost,
			COUNT(*) as session_count,
			COALESCE(MAX(total_cost_usd), 0) as max_cost
		FROM agent_session_stats
		WHERE user_id = $1
		  AND started_at >= $2
		  AND is_error = false
	`

	var totalCost float64
	var sessionCount int64
	var maxCost float64

	err := d.db.QueryRowContext(ctx, query, userID, startDate).Scan(&totalCost, &sessionCount, &maxCost)
	if err != nil {
		return nil, fmt.Errorf("failed to get cost stats: %w", err)
	}

	dailyAverage := totalCost / float64(days)

	// Get most expensive session
	var mostExpensive *store.AgentSessionStats
	if maxCost > 0 {
		var err error
		mostExpensive, err = d.GetMostExpensiveSession(ctx, userID, startDate)
		if err != nil {
			// Log but don't fail - mostExpensive will be nil
			slog.Warn("Failed to get most expensive session for cost stats",
				"user_id", userID,
				"error", err)
		}
	}

	// Get daily breakdown
	dailyBreakdown, err := d.getDailyCostBreakdown(ctx, userID, days)
	if err != nil {
		dailyBreakdown = []*store.DailyCostData{}
	}

	return &store.CostStats{
		TotalCostUSD:         totalCost,
		DailyAverageUSD:      dailyAverage,
		SessionCount:         sessionCount,
		MostExpensiveSession: mostExpensive,
		DailyBreakdown:       dailyBreakdown,
	}, nil
}

// GetMostExpensiveSession retrieves the most expensive session for a user.
func (d *DB) GetMostExpensiveSession(ctx context.Context, userID int32, since time.Time) (*store.AgentSessionStats, error) {
	query := `
		SELECT id, session_id, conversation_id, user_id, agent_type,
			   started_at, ended_at, total_duration_ms,
			   thinking_duration_ms, tool_duration_ms, generation_duration_ms,
			   input_tokens, output_tokens, cache_write_tokens, cache_read_tokens, total_tokens,
			   total_cost_usd, tool_call_count, tools_used, files_modified, file_paths,
			   model_used, is_error, error_message, created_at, updated_at
		FROM agent_session_stats
		WHERE user_id = $1 AND started_at >= $2 AND is_error = false
		ORDER BY total_cost_usd DESC
		LIMIT 1
	`

	var stats store.AgentSessionStats
	var toolsUsedJSONB []byte
	var filePathsArray []string

	err := d.db.QueryRowContext(ctx, query, userID, since).Scan(
		&stats.ID,
		&stats.SessionID,
		&stats.ConversationID,
		&stats.UserID,
		&stats.AgentType,
		&stats.StartedAt,
		&stats.EndedAt,
		&stats.TotalDurationMs,
		&stats.ThinkingDurationMs,
		&stats.ToolDurationMs,
		&stats.GenerationDurationMs,
		&stats.InputTokens,
		&stats.OutputTokens,
		&stats.CacheWriteTokens,
		&stats.CacheReadTokens,
		&stats.TotalTokens,
		&stats.TotalCostUSD,
		&stats.ToolCallCount,
		&toolsUsedJSONB,
		&stats.FilesModified,
		&filePathsArray,
		&stats.ModelUsed,
		&stats.IsError,
		&stats.ErrorMessage,
		&stats.CreatedAt,
		&stats.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // No sessions found
		}
		return nil, fmt.Errorf("failed to get most expensive session: %w", err)
	}

	if toolsUsedJSONB != nil {
		stats.ToolsUsed = parseStringArray(toolsUsedJSONB)
	}
	stats.FilePaths = filePathsArray

	return &stats, nil
}

// getDailyCostBreakdown retrieves daily cost data for the last N days.
func (d *DB) getDailyCostBreakdown(ctx context.Context, userID int32, days int) ([]*store.DailyCostData, error) {
	query := `
		SELECT
			DATE(started_at) as date,
			COALESCE(SUM(total_cost_usd), 0) as cost,
			COUNT(*) as session_count
		FROM agent_session_stats
		WHERE user_id = $1
		  AND started_at >= NOW() - INTERVAL '1 day' * $2
		  AND is_error = false
		GROUP BY DATE(started_at)
		ORDER BY date DESC
	`

	rows, err := d.db.QueryContext(ctx, query, userID, days)
	if err != nil {
		return nil, fmt.Errorf("failed to get daily cost breakdown: %w", err)
	}
	defer rows.Close()

	var breakdown []*store.DailyCostData
	for rows.Next() {
		var data store.DailyCostData
		err := rows.Scan(&data.Date, &data.CostUSD, &data.SessionCount)
		if err != nil {
			return nil, fmt.Errorf("failed to scan daily cost: %w", err)
		}
		breakdown = append(breakdown, &data)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating daily cost breakdown: %w", err)
	}

	return breakdown, nil
}

// GetUserCostSettings retrieves or creates user cost settings.
func (d *DB) GetUserCostSettings(ctx context.Context, userID int32) (*store.UserCostSettings, error) {
	query := `
		SELECT user_id, daily_budget_usd, per_session_threshold_usd,
			   alert_enabled, alert_email, alert_in_app, budget_reset_at
		FROM user_cost_settings
		WHERE user_id = $1
	`

	var settings store.UserCostSettings
	err := d.db.QueryRowContext(ctx, query, userID).Scan(
		&settings.UserID,
		&settings.DailyBudgetUSD,
		&settings.PerSessionThresholdUSD,
		&settings.AlertEnabled,
		&settings.AlertEmail,
		&settings.AlertInApp,
		&settings.BudgetResetAt,
	)

	if err != nil {
		// Create default settings if not found
		settings = store.UserCostSettings{
			UserID:                 userID,
			DailyBudgetUSD:         nil,
			PerSessionThresholdUSD: 5.0,
			AlertEnabled:           true,
			AlertEmail:             false,
			AlertInApp:             true,
			BudgetResetAt:          nil,
		}

		insertErr := d.SetUserCostSettings(ctx, &settings)
		if insertErr != nil {
			return nil, fmt.Errorf("failed to create default settings: %w", insertErr)
		}

		return &settings, nil
	}

	return &settings, nil
}

// SetUserCostSettings updates user cost settings.
func (d *DB) SetUserCostSettings(ctx context.Context, settings *store.UserCostSettings) error {
	query := `
		INSERT INTO user_cost_settings (
			user_id, daily_budget_usd, per_session_threshold_usd,
			alert_enabled, alert_email, alert_in_app, budget_reset_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (user_id) DO UPDATE SET
			daily_budget_usd = EXCLUDED.daily_budget_usd,
			per_session_threshold_usd = EXCLUDED.per_session_threshold_usd,
			alert_enabled = EXCLUDED.alert_enabled,
			alert_email = EXCLUDED.alert_email,
			alert_in_app = EXCLUDED.alert_in_app,
			budget_reset_at = EXCLUDED.budget_reset_at,
			updated_at = NOW()
	`

	_, err := d.db.ExecContext(ctx, query,
		settings.UserID,
		settings.DailyBudgetUSD,
		settings.PerSessionThresholdUSD,
		settings.AlertEnabled,
		settings.AlertEmail,
		settings.AlertInApp,
		settings.BudgetResetAt,
	)

	if err != nil {
		return fmt.Errorf("failed to set user cost settings: %w", err)
	}

	return nil
}

// LogSecurityEvent saves a security event.
func (d *DB) LogSecurityEvent(ctx context.Context, event *store.SecurityAuditEvent) error {
	query := `
		INSERT INTO agent_security_audit (
			session_id, user_id, agent_type, operation_type, operation_name,
			risk_level, command_input, command_matched_pattern,
			action_taken, reason, file_path, tool_id
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id, occurred_at
	`

	err := d.db.QueryRowContext(ctx, query,
		event.SessionID,
		event.UserID,
		event.AgentType,
		event.OperationType,
		event.OperationName,
		event.RiskLevel,
		event.CommandInput,
		event.CommandMatchedPattern,
		event.ActionTaken,
		event.Reason,
		event.FilePath,
		event.ToolID,
	).Scan(&event.ID, &event.OccurredAt)

	if err != nil {
		return fmt.Errorf("failed to log security event: %w", err)
	}

	return nil
}

// ListSecurityEvents retrieves security events for a user with pagination.
func (d *DB) ListSecurityEvents(ctx context.Context, userID int32, limit, offset int) ([]*store.SecurityAuditEvent, int64, error) {
	// Get total count
	var total int64
	err := d.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM agent_security_audit WHERE user_id = $1", userID).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count security events: %w", err)
	}

	query := `
		SELECT id, session_id, user_id, agent_type, operation_type, operation_name,
			   risk_level, command_input, command_matched_pattern,
			   action_taken, reason, file_path, tool_id, occurred_at
		FROM agent_security_audit
		WHERE user_id = $1
		ORDER BY occurred_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := d.db.QueryContext(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list security events: %w", err)
	}
	defer rows.Close()

	var events []*store.SecurityAuditEvent
	for rows.Next() {
		var event store.SecurityAuditEvent
		err := rows.Scan(
			&event.ID,
			&event.SessionID,
			&event.UserID,
			&event.AgentType,
			&event.OperationType,
			&event.OperationName,
			&event.RiskLevel,
			&event.CommandInput,
			&event.CommandMatchedPattern,
			&event.ActionTaken,
			&event.Reason,
			&event.FilePath,
			&event.ToolID,
			&event.OccurredAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan security event: %w", err)
		}
		events = append(events, &event)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating security events: %w", err)
	}

	return events, total, nil
}

// ListSecurityEventsByRisk retrieves events filtered by risk level.
func (d *DB) ListSecurityEventsByRisk(ctx context.Context, userID int32, riskLevel string, limit, offset int) ([]*store.SecurityAuditEvent, int64, error) {
	// Get total count
	var total int64
	err := d.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM agent_security_audit WHERE user_id = $1 AND risk_level = $2", userID, riskLevel).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count security events by risk: %w", err)
	}

	query := `
		SELECT id, session_id, user_id, agent_type, operation_type, operation_name,
			   risk_level, command_input, command_matched_pattern,
			   action_taken, reason, file_path, tool_id, occurred_at
		FROM agent_security_audit
		WHERE user_id = $1 AND risk_level = $2
		ORDER BY occurred_at DESC
		LIMIT $3 OFFSET $4
	`

	rows, err := d.db.QueryContext(ctx, query, userID, riskLevel, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list security events by risk: %w", err)
	}
	defer rows.Close()

	var events []*store.SecurityAuditEvent
	for rows.Next() {
		var event store.SecurityAuditEvent
		err := rows.Scan(
			&event.ID,
			&event.SessionID,
			&event.UserID,
			&event.AgentType,
			&event.OperationType,
			&event.OperationName,
			&event.RiskLevel,
			&event.CommandInput,
			&event.CommandMatchedPattern,
			&event.ActionTaken,
			&event.Reason,
			&event.FilePath,
			&event.ToolID,
			&event.OccurredAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan security event: %w", err)
		}
		events = append(events, &event)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating security events by risk: %w", err)
	}

	return events, total, nil
}

// parseStringArray parses a JSONB array of strings from PostgreSQL.
// Uses strings.Builder for O(n) performance instead of O(n²) concatenation.
// Always returns a non-nil slice for consistency.
func parseStringArray(data []byte) []string {
	if len(data) == 0 || string(data) == "null" {
		return []string{} // Return empty slice, not nil
	}

	// Remove surrounding brackets and quotes, split by comma
	// This is a simple parser for JSON arrays like ["a","b","c"]
	str := string(data)
	if str == "[]" {
		return []string{}
	}

	// Simple JSON array parsing
	str = str[1 : len(str)-1] // Remove [ ]
	var result []string
	var builder strings.Builder
	inQuotes := false

	for _, c := range str {
		switch c {
		case '"':
			inQuotes = !inQuotes
		case ',':
			if inQuotes {
				builder.WriteRune(c)
			} else {
				result = append(result, builder.String())
				builder.Reset()
			}
		case ' ', '\t', '\n':
			if inQuotes {
				builder.WriteRune(c)
			}
		default:
			builder.WriteRune(c)
		}
	}
	if builder.Len() > 0 {
		result = append(result, builder.String())
	}

	return result
}
