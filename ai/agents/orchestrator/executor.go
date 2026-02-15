package orchestrator

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"
)

// Event types for orchestrator events
const (
	// EventTypePlan is sent when a task plan is created
	EventTypePlan = "plan"
	// EventTypeTaskStart is sent when a task starts executing
	EventTypeTaskStart = "task_start"
	// EventTypeTaskEnd is sent when a task finishes executing
	EventTypeTaskEnd = "task_end"
)

// Executor executes tasks by dispatching them to expert agents.
type Executor struct {
	registry ExpertRegistry
	config   *OrchestratorConfig
}

// NewExecutor creates a new task executor.
func NewExecutor(registry ExpertRegistry, config *OrchestratorConfig) *Executor {
	if config == nil {
		config = DefaultOrchestratorConfig()
	}
	return &Executor{
		registry: registry,
		config:   config,
	}
}

// ExecutePlan executes all tasks in the plan using DAG scheduling and returns results.
func (e *Executor) ExecutePlan(ctx context.Context, plan *TaskPlan, callback EventCallback) *ExecutionResult {
	result := &ExecutionResult{
		Plan:         plan,
		IsAggregated: false,
	}

	startTime := time.Now()
	slog.Info("executor: starting DAG plan execution",
		"tasks", len(plan.Tasks),
		"parallel", plan.Parallel)

	// Send plan event to frontend
	if callback != nil {
		e.sendPlanEvent(plan, callback)
	}

	// Initialize DAG Scheduler
	scheduler, err := NewDAGScheduler(e, plan.Tasks)
	if err != nil {
		slog.Error("executor: failed to initialize DAG scheduler", "error", err)
		result.Errors = append(result.Errors, fmt.Sprintf("DAG Init Error: %v", err))
		return result
	}

	// Inject context injector dependency if needed (or just use global/util)
	// For now, DAGScheduler uses Executor methods which use ContextInjector.

	// Run Scheduler
	err = scheduler.Run(ctx)
	if err != nil {
		slog.Error("executor: DAG execution failed", "error", err)
		result.Errors = append(result.Errors, fmt.Sprintf("Execution Error: %v", err))
	}

	// Collect results and errors
	var results []string
	for _, task := range plan.Tasks {
		if task.Status == TaskStatusCompleted && task.Result != "" {
			results = append(results, task.Result)
		}
		if task.Status == TaskStatusFailed && task.Error != "" {
			result.Errors = append(result.Errors, fmt.Sprintf("Task %s: %s", task.ID, task.Error))
		}
	}

	// Set final response based on result count
	if len(results) == 1 {
		result.FinalResponse = results[0]
	} else if len(results) > 1 && plan.Aggregate {
		// Multiple results need aggregation - will be handled by Aggregator
		result.IsAggregated = true
	} else if len(results) > 1 {
		// Multiple results without aggregation - join them
		result.FinalResponse = strings.Join(results, "\n\n---\n\n")
	}

	slog.Info("executor: plan execution completed",
		"duration_ms", time.Since(startTime).Milliseconds(),
		"success_count", len(results),
		"error_count", len(result.Errors))

	return result
}

// executeSingleTask executes a single task. Used by DAGScheduler.
func (e *Executor) executeSingleTask(ctx context.Context, task *Task, callback EventCallback) error {
	startTime := time.Now()
	task.Status = TaskStatusRunning

	// Send task_start event
	if callback != nil {
		// We need index for event? TaskPlan has slice index, but Task struct might not know it.
		// For backward compatibility, we might pass -1 or find index if strictly needed.
		// The original code passed index. Let's try to assume index is not critical or 0.
		e.sendTaskStartEvent(task, -1, callback)
	}

	slog.Debug("executor: executing task",
		"id", task.ID,
		"agent", task.Agent,
		"purpose", task.Purpose)

	// 1. Context Injection
	// We need access to all tasks to resolve variables.
	// But executeSingleTask receives *Task.
	// The DAGScheduler should have already resolved inputs OR we pass the map here.
	// BETTER DESIGN: DAGScheduler calls ContextInjector BEFORE calling executeSingleTask.
	// So here we assume task.Input is already resolved.

	// Create result collector
	resultCollector := &resultCollector{callback: callback}

	// Execute via expert registry
	err := e.registry.ExecuteExpert(ctx, task.Agent, task.Input, resultCollector.onEvent)

	if err != nil {
		// Task status updated by caller (DAGScheduler) or here?
		// Let's update here for consistency
		task.Status = TaskStatusFailed
		task.Error = err.Error()
		slog.Error("executor: task failed",
			"id", task.ID,
			"agent", task.Agent,
			"error", err)

		// Send task_end event with error
		if callback != nil {
			e.sendTaskEndEvent(task, -1, callback)
		}
		return err
	}

	// Success
	task.Status = TaskStatusCompleted
	task.Result = resultCollector.getResult()
	slog.Debug("executor: task completed",
		"id", task.ID,
		"agent", task.Agent,
		"duration_ms", time.Since(startTime).Milliseconds())

	// Send task_end event
	if callback != nil {
		e.sendTaskEndEvent(task, -1, callback)
	}
	return nil
}

// sendPlanEvent sends the task plan to the frontend.
func (e *Executor) sendPlanEvent(plan *TaskPlan, callback EventCallback) {
	event := map[string]interface{}{
		"analysis": plan.Analysis,
		"tasks":    plan.Tasks,
		"parallel": plan.Parallel,
	}
	eventJSON, err := json.Marshal(event)
	if err != nil {
		slog.Error("executor: failed to marshal plan event", "error", err)
		return
	}
	callback(EventTypePlan, string(eventJSON))
}

// sendTaskStartEvent sends a task start event to the frontend.
func (e *Executor) sendTaskStartEvent(task *Task, index int, callback EventCallback) {
	event := map[string]interface{}{
		"id":      task.ID,
		"index":   index, // -1 if unknown
		"agent":   task.Agent,
		"purpose": task.Purpose,
		"status":  string(task.Status),
	}
	eventJSON, err := json.Marshal(event)
	if err != nil {
		slog.Error("executor: failed to marshal task_start event", "error", err, "id", task.ID)
		return
	}
	callback(EventTypeTaskStart, string(eventJSON))
}

// sendTaskEndEvent sends a task end event to the frontend.
func (e *Executor) sendTaskEndEvent(task *Task, index int, callback EventCallback) {
	event := map[string]interface{}{
		"id":     task.ID,
		"index":  index,
		"agent":  task.Agent,
		"status": string(task.Status),
	}
	if task.Error != "" {
		event["error"] = task.Error
	}
	eventJSON, err := json.Marshal(event)
	if err != nil {
		slog.Error("executor: failed to marshal task_end event", "error", err, "id", task.ID)
		return
	}
	callback(EventTypeTaskEnd, string(eventJSON))
}

// Max result size to prevent OOM (10MB)
const maxResultSize = 10 * 1024 * 1024

// resultCollector collects results from event callbacks.
type resultCollector struct {
	mu        sync.Mutex
	callback  EventCallback
	result    strings.Builder
	truncated bool
}

func (rc *resultCollector) onEvent(eventType string, eventData string) {
	// Forward event to original callback
	if rc.callback != nil {
		rc.callback(eventType, eventData)
	}

	// Collect text/content events as results with size limit
	if eventType == "content" || eventType == "text" || eventType == "response" {
		rc.mu.Lock()
		if rc.result.Len()+len(eventData) <= maxResultSize {
			rc.result.WriteString(eventData)
		} else if !rc.truncated {
			// Log once when we hit the limit
			rc.truncated = true
			slog.Warn("executor: result truncated due to size limit", "limit_bytes", maxResultSize)
		}
		rc.mu.Unlock()
	}
}

func (rc *resultCollector) getResult() string {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	return rc.result.String()
}
