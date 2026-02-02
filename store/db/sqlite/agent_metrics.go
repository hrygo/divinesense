package sqlite

import (
	"context"

	"github.com/hrygo/divinesense/store"
)

// Agent metrics - STUB (TODO: implement full support)
// For now, return success without storing metrics
// Full implementation requires aligning type definitions

func (d *DB) UpsertAgentMetrics(_ context.Context, _ *store.UpsertAgentMetrics) (*store.AgentMetrics, error) {
	// Return success without error - metrics are optional for core AI functionality
	return &store.AgentMetrics{}, nil
}

func (d *DB) ListAgentMetrics(_ context.Context, _ *store.FindAgentMetrics) ([]*store.AgentMetrics, error) {
	return []*store.AgentMetrics{}, nil
}

func (d *DB) DeleteAgentMetrics(_ context.Context, _ *store.DeleteAgentMetrics) error {
	return nil
}

func (d *DB) UpsertToolMetrics(_ context.Context, _ *store.UpsertToolMetrics) (*store.ToolMetrics, error) {
	return &store.ToolMetrics{}, nil
}

func (d *DB) ListToolMetrics(_ context.Context, _ *store.FindToolMetrics) ([]*store.ToolMetrics, error) {
	return []*store.ToolMetrics{}, nil
}

func (d *DB) DeleteToolMetrics(_ context.Context, _ *store.DeleteToolMetrics) error {
	return nil
}
