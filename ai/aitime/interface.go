// Package aitime provides the time parsing service interface for AI agents.
// This interface is consumed by Team B (Assistant+Schedule).
package aitime

import (
	"context"
	"time"
)

// Consumers: Team B (Assistant+Schedule).
type TimeService interface {
	// Normalize standardizes time expressions.
	// Supports: "明天3点", "下午三点", "2026-1-28", "15:00"
	// Returns: standardized time.Time
	Normalize(ctx context.Context, input string, timezone string) (time.Time, error)

	// ParseNaturalTime parses natural language time expressions.
	// reference: reference time point (usually current time)
	// Returns: time range
	ParseNaturalTime(ctx context.Context, input string, reference time.Time) (TimeRange, error)
}

// TimeRange represents a time range.
type TimeRange struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}
