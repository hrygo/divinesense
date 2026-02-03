// Package metrics provides the evaluation metrics service interface for AI agents.
// This interface is consumed by Team B (Assistant+Schedule) and Team C (Memo Enhancement).
package metrics

import (
	"context"
	"time"
)

// Consumers: Team B (Assistant+Schedule), Team C (Memo Enhancement).
type MetricsService interface {
	// RecordRequest records request metrics.
	RecordRequest(ctx context.Context, agentType string, latency time.Duration, success bool)

	// RecordToolCall records tool call metrics.
	RecordToolCall(ctx context.Context, toolName string, latency time.Duration, success bool)

	// GetStats retrieves statistics data.
	GetStats(ctx context.Context, timeRange TimeRange) (*AgentMetrics, error)
}

// TimeRange represents a time range for querying metrics.
type TimeRange struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// AgentMetrics represents aggregated agent metrics.
type AgentMetrics struct {
	AgentStats   map[string]*AgentStat `json:"agent_stats"`
	ErrorsByType map[string]int64      `json:"errors_by_type"`
	RequestCount int64                 `json:"request_count"`
	SuccessCount int64                 `json:"success_count"`
	LatencyP50   time.Duration         `json:"latency_p50"`
	LatencyP95   time.Duration         `json:"latency_p95"`
}

// AgentStat represents statistics for a single agent.
type AgentStat struct {
	Count       int64         `json:"count"`
	SuccessRate float32       `json:"success_rate"`
	AvgLatency  time.Duration `json:"avg_latency"`
}
