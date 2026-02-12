// Package orchestrator implements the Orchestrator-Workers pattern for multi-agent coordination.
// It uses LLM to dynamically decompose tasks, dispatch to expert agents, and aggregate results.
//
// Architecture:
//
//	User Input
//	    ↓
//	┌─────────────────┐
//	│  Orchestrator   │ ← LLM-driven task decomposition
//	└────────┬────────┘
//	         │
//	    ┌────┴────┐
//	    ↓         ↓
//	┌───────┐ ┌───────┐
//	│ Memo  │ │ Sched │  ← Expert Agents (config-driven)
//	└───────┘ └───────┘
//	    │         │
//	    └────┬────┘
//	         ↓
//	┌─────────────────┐
//	│  Aggregator     │ ← Combine results (if needed)
//	└─────────────────┘
//
// Key features:
//   - LLM-driven task decomposition (not hard-coded rules)
//   - Automatic adaptation to new expert agents
//   - Parallel execution for independent tasks
//   - Transparent planning (shows steps to user)
//   - Result aggregation for multi-agent responses
package orchestrator

import (
	"context"
	"log/slog"

	"github.com/hrygo/divinesense/ai/core/llm"
)

// Orchestrator coordinates the decomposition, execution, and aggregation of tasks.
// It implements the Orchestrator-Workers pattern as recommended by Anthropic.
type Orchestrator struct {
	decomposer *Decomposer
	executor   *Executor
	aggregator *Aggregator
	config     *OrchestratorConfig
}

// NewOrchestrator creates a new orchestrator with the given LLM service and expert registry.
func NewOrchestrator(llmService llm.Service, registry ExpertRegistry, opts ...Option) *Orchestrator {
	config := DefaultOrchestratorConfig()
	for _, opt := range opts {
		opt(config)
	}

	return &Orchestrator{
		decomposer: NewDecomposer(llmService, config),
		executor:   NewExecutor(registry, config),
		aggregator: NewAggregator(llmService, config),
		config:     config,
	}
}

// Option configures the orchestrator.
type Option func(*OrchestratorConfig)

// WithMaxParallelTasks sets the maximum number of parallel tasks.
func WithMaxParallelTasks(n int) Option {
	return func(c *OrchestratorConfig) {
		if n > 0 {
			c.MaxParallelTasks = n
		}
	}
}

// WithAggregation enables or disables result aggregation.
func WithAggregation(enabled bool) Option {
	return func(c *OrchestratorConfig) {
		c.EnableAggregation = enabled
	}
}

// Process handles a user request by decomposing, executing, and aggregating.
// This is the main entry point for the orchestrator.
func (o *Orchestrator) Process(ctx context.Context, userInput string, callback EventCallback) (*ExecutionResult, error) {
	slog.Info("orchestrator: processing request",
		"input_length", len(userInput))

	// Step 1: Decompose the request into tasks
	plan, err := o.decomposer.Decompose(ctx, userInput, o.executor.registry)
	if err != nil {
		slog.Error("orchestrator: decomposition failed", "error", err)
		return nil, err
	}

	// Step 2: Execute the tasks
	result := o.executor.ExecutePlan(ctx, plan, callback)

	// Step 3: Aggregate results if needed
	if result.IsAggregated && o.config.EnableAggregation {
		aggregated, err := o.aggregator.Aggregate(ctx, result, callback)
		if err != nil {
			slog.Warn("orchestrator: aggregation failed, using concatenated results", "error", err)
			result.IsAggregated = false
		} else {
			result.FinalResponse = aggregated
			result.IsAggregated = true
		}
	}

	slog.Info("orchestrator: processing completed",
		"tasks", len(plan.Tasks),
		"aggregated", result.IsAggregated)

	return result, nil
}

// ProcessSimple is a convenience method that returns just the final response string.
// Use this when you don't need the full execution result.
func (o *Orchestrator) ProcessSimple(ctx context.Context, userInput string, callback EventCallback) (string, error) {
	result, err := o.Process(ctx, userInput, callback)
	if err != nil {
		return "", err
	}
	return result.FinalResponse, nil
}
