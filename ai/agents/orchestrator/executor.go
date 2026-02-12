package orchestrator

import (
	"context"
	"encoding/json"
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

// ExecutePlan executes all tasks in the plan and returns results.
func (e *Executor) ExecutePlan(ctx context.Context, plan *TaskPlan, callback EventCallback) *ExecutionResult {
	result := &ExecutionResult{
		Plan:         plan,
		IsAggregated: false,
	}

	startTime := time.Now()
	slog.Info("executor: starting plan execution",
		"tasks", len(plan.Tasks),
		"parallel", plan.Parallel)

	// Send plan event to frontend
	if callback != nil {
		e.sendPlanEvent(plan, callback)
	}

	if plan.Parallel && len(plan.Tasks) > 1 {
		e.executeParallel(ctx, plan.Tasks, callback)
	} else {
		e.executeSequential(ctx, plan.Tasks, callback)
	}

	// Collect results and errors
	var results []string
	for _, task := range plan.Tasks {
		if task.Status == TaskStatusCompleted && task.Result != "" {
			results = append(results, task.Result)
		}
		if task.Status == TaskStatusFailed && task.Error != "" {
			result.Errors = append(result.Errors, task.Error)
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

// executeParallel executes tasks in parallel using goroutines.
func (e *Executor) executeParallel(ctx context.Context, tasks []*Task, callback EventCallback) {
	var wg sync.WaitGroup
	sem := make(chan struct{}, e.config.MaxParallelTasks)

	for i, task := range tasks {
		wg.Add(1)
		go func(idx int, t *Task) {
			defer wg.Done()

			// Acquire semaphore
			sem <- struct{}{}
			defer func() { <-sem }()

			e.executeTask(ctx, t, idx, callback)
		}(i, task)
	}

	wg.Wait()
}

// executeSequential executes tasks one after another.
func (e *Executor) executeSequential(ctx context.Context, tasks []*Task, callback EventCallback) {
	for i, task := range tasks {
		e.executeTask(ctx, task, i, callback)
	}
}

// executeTask executes a single task.
func (e *Executor) executeTask(ctx context.Context, task *Task, index int, callback EventCallback) {
	startTime := time.Now()
	task.Status = TaskStatusRunning

	// Send task_start event
	if callback != nil {
		e.sendTaskStartEvent(task, index, callback)
	}

	slog.Debug("executor: executing task",
		"index", index,
		"agent", task.Agent,
		"purpose", task.Purpose)

	// Create result collector
	resultCollector := &resultCollector{callback: callback}

	// Execute via expert registry
	err := e.registry.ExecuteExpert(ctx, task.Agent, task.Input, resultCollector.onEvent)

	if err != nil {
		task.Status = TaskStatusFailed
		task.Error = err.Error()
		slog.Error("executor: task failed",
			"index", index,
			"agent", task.Agent,
			"error", err)
	} else {
		task.Status = TaskStatusCompleted
		task.Result = resultCollector.getResult()
		slog.Debug("executor: task completed",
			"index", index,
			"agent", task.Agent,
			"duration_ms", time.Since(startTime).Milliseconds())
	}

	// Send task_end event
	if callback != nil {
		e.sendTaskEndEvent(task, index, callback)
	}
}

// sendPlanEvent sends the task plan to the frontend.
func (e *Executor) sendPlanEvent(plan *TaskPlan, callback EventCallback) {
	event := map[string]interface{}{
		"analysis": plan.Analysis,
		"tasks":    plan.Tasks,
		"parallel": plan.Parallel,
	}
	eventJSON, _ := json.Marshal(event)
	callback(EventTypePlan, string(eventJSON))
}

// sendTaskStartEvent sends a task start event to the frontend.
func (e *Executor) sendTaskStartEvent(task *Task, index int, callback EventCallback) {
	event := map[string]interface{}{
		"index":   index,
		"agent":   task.Agent,
		"purpose": task.Purpose,
		"status":  string(task.Status),
	}
	eventJSON, _ := json.Marshal(event)
	callback(EventTypeTaskStart, string(eventJSON))
}

// sendTaskEndEvent sends a task end event to the frontend.
func (e *Executor) sendTaskEndEvent(task *Task, index int, callback EventCallback) {
	event := map[string]interface{}{
		"index":  index,
		"agent":  task.Agent,
		"status": string(task.Status),
	}
	if task.Error != "" {
		event["error"] = task.Error
	}
	eventJSON, _ := json.Marshal(event)
	callback(EventTypeTaskEnd, string(eventJSON))
}

// resultCollector collects results from event callbacks.
type resultCollector struct {
	mu       sync.Mutex
	callback EventCallback
	result   strings.Builder
}

func (rc *resultCollector) onEvent(eventType string, eventData string) {
	// Forward event to original callback
	if rc.callback != nil {
		rc.callback(eventType, eventData)
	}

	// Collect text/content events as results
	if eventType == "content" || eventType == "text" || eventType == "response" {
		rc.mu.Lock()
		rc.result.WriteString(eventData)
		rc.mu.Unlock()
	}
}

func (rc *resultCollector) getResult() string {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	return rc.result.String()
}
