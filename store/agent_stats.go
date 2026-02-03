package store

import (
	"context"
	"time"
)

// AgentSessionStats represents a session statistics record for storage.
// AgentSessionStats 表示会话统计数据的存储记录。
type AgentSessionStats struct {
	ID                   int64
	SessionID            string
	ConversationID       int64
	UserID               int32
	AgentType            string
	StartedAt            time.Time
	EndedAt              time.Time
	TotalDurationMs      int64
	ThinkingDurationMs   int64
	ToolDurationMs       int64
	GenerationDurationMs int64
	InputTokens          int32
	OutputTokens         int32
	CacheWriteTokens     int32
	CacheReadTokens      int32
	TotalTokens          int32
	TotalCostUSD         float64
	ToolCallCount        int32
	ToolsUsed            []string
	FilesModified        int32
	FilePaths            []string
	ModelUsed            string
	IsError              bool
	ErrorMessage         string
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

// CostStats represents aggregated cost statistics for a user.
// CostStats 表示用户的聚合成本统计。
type CostStats struct {
	TotalCostUSD         float64
	DailyAverageUSD      float64
	SessionCount         int64
	MostExpensiveSession *AgentSessionStats
	DailyBreakdown       []*DailyCostData
}

// DailyCostData represents cost data for a single day.
// DailyCostData 表示单日的成本数据。
type DailyCostData struct {
	Date         string // YYYY-MM-DD
	CostUSD      float64
	SessionCount int64
}

// UserCostSettings represents user-specific cost control settings.
// UserCostSettings 表示用户特定的成本控制设置。
type UserCostSettings struct {
	UserID                 int32
	DailyBudgetUSD         *float64 // NULL = unlimited
	PerSessionThresholdUSD float64
	AlertEnabled           bool
	AlertEmail             bool
	AlertInApp             bool
	BudgetResetAt          *time.Time
}

// AgentStatsStore defines the interface for session statistics persistence.
// AgentStatsStore 定义会话统计持久化的接口。
type AgentStatsStore interface {
	// SaveSessionStats saves a session statistics record.
	SaveSessionStats(ctx context.Context, stats *AgentSessionStats) error

	// GetSessionStats retrieves stats by session ID.
	GetSessionStats(ctx context.Context, sessionID string) (*AgentSessionStats, error)

	// ListSessionStats retrieves stats for a user with pagination.
	ListSessionStats(ctx context.Context, userID int32, limit, offset int) ([]*AgentSessionStats, int64, error)

	// GetDailyCostUsage retrieves total cost for a user in a date range.
	GetDailyCostUsage(ctx context.Context, userID int32, startDate, endDate time.Time) (float64, error)

	// GetCostStats retrieves aggregated cost statistics.
	GetCostStats(ctx context.Context, userID int32, days int) (*CostStats, error)

	// GetUserCostSettings retrieves or creates user cost settings.
	GetUserCostSettings(ctx context.Context, userID int32) (*UserCostSettings, error)

	// SetUserCostSettings updates user cost settings.
	SetUserCostSettings(ctx context.Context, settings *UserCostSettings) error
}

// SecurityAuditEvent represents a security-related event for audit logging.
// SecurityAuditEvent 表示安全相关事件的审计日志。
type SecurityAuditEvent struct {
	ID                    int64
	SessionID             string
	UserID                int32
	AgentType             string
	OperationType         string
	OperationName         string
	RiskLevel             string
	CommandInput          string
	CommandMatchedPattern string
	ActionTaken           string
	Reason                string
	FilePath              string
	ToolID                string
	OccurredAt            time.Time
}

// SecurityAuditStore defines the interface for security audit logging.
// SecurityAuditStore 定义安全审计日志的接口。
type SecurityAuditStore interface {
	// LogSecurityEvent saves a security event.
	LogSecurityEvent(ctx context.Context, event *SecurityAuditEvent) error

	// ListSecurityEvents retrieves security events for a user with pagination.
	ListSecurityEvents(ctx context.Context, userID int32, limit, offset int) ([]*SecurityAuditEvent, int64, error)

	// ListSecurityEventsByRisk retrieves events filtered by risk level.
	ListSecurityEventsByRisk(ctx context.Context, userID int32, riskLevel string, limit, offset int) ([]*SecurityAuditEvent, int64, error)
}
