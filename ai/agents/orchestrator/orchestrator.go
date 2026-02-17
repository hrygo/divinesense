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
	"fmt"
	"log/slog"
	"time"

	agents "github.com/hrygo/divinesense/ai/agents"
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

	// Create executor with or without handoff support based on config
	var executor *Executor
	if config.EnableHandoff {
		// Create capability map and handoff handler
		capabilityMap := NewCapabilityMap()

		// Populate capability map from registry
		var configs []*agents.ParrotSelfCognition
		for _, name := range registry.GetAvailableExperts() {
			if config := registry.GetExpertConfig(name); config != nil {
				configs = append(configs, config)
			}
		}
		capabilityMap.BuildFromConfigs(configs)

		handoffHandler := NewHandoffHandler(capabilityMap, 2)
		executor = NewExecutorWithHandoff(registry, config, handoffHandler)
	} else {
		executor = NewExecutor(registry, config)
	}

	return &Orchestrator{
		decomposer: NewDecomposer(llmService, config),
		executor:   executor,
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

// WithHandoff enables or disables expert handoff.
// When enabled, if an expert cannot handle a task, the orchestrator will
// attempt to find an alternative expert that can handle it.
func WithHandoff(enabled bool) Option {
	return func(c *OrchestratorConfig) {
		c.EnableHandoff = enabled
	}
}

// Process handles a user request by decomposing, executing, and aggregating.
// This is the main entry point for the orchestrator.
func (o *Orchestrator) Process(ctx context.Context, userInput string, callback EventCallback) (*ExecutionResult, error) {
	startTime := time.Now()

	// Generate trace_id for request tracing
	traceID := GenerateTraceID()

	slog.Info("orchestrator: processing request",
		"trace_id", traceID,
		"input_length", len(userInput))

	// Send decompose_start event for UX feedback
	if callback != nil {
		callback("decompose_start", `{"status":"analyzing"}`)
	}

	// Step 1: Decompose the request into tasks
	plan, err := o.decomposer.Decompose(ctx, userInput, o.executor.registry, traceID)
	if err != nil {
		slog.Error("orchestrator: decomposition failed",
			"trace_id", traceID,
			"error", err)
		return nil, err
	}

	// Send decompose_end event with task count
	if callback != nil {
		taskInfo := fmt.Sprintf(`{"task_count":%d,"analysis":%q}`, len(plan.Tasks), plan.Analysis)
		callback("decompose_end", taskInfo)
	}

	// Step 2: Execute the tasks
	result := o.executor.ExecutePlan(ctx, plan, callback, traceID)

	// Step 3: Aggregate results if needed
	if result.IsAggregated && o.config.EnableAggregation {
		aggregated, err := o.aggregator.Aggregate(ctx, result, callback)
		if err != nil {
			slog.Warn("orchestrator: aggregation failed, using concatenated results",
				"trace_id", traceID,
				"error", err)
			result.IsAggregated = false
		} else {
			result.FinalResponse = aggregated
			result.IsAggregated = true
		}
	}

	duration := time.Since(startTime)
	slog.Info("orchestrator: processing completed",
		"trace_id", traceID,
		"tasks", len(plan.Tasks),
		"aggregated", result.IsAggregated,
		"duration_ms", duration.Milliseconds())

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
