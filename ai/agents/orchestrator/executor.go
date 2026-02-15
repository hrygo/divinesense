package orchestrator

import (
	"context"
	"encoding/json"
	"errors"
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
	registry       ExpertRegistry
	config         *OrchestratorConfig
	handoffHandler HandoffHandlerInterface // Use interface for dependency injection
}

// NewExecutor creates a new task executor.
func NewExecutor(registry ExpertRegistry, config *OrchestratorConfig) *Executor {
	if config == nil {
		config = DefaultOrchestratorConfig()
	}
	return &Executor{
		registry:       registry,
		config:         config,
		handoffHandler: nil,
	}
}

// NewExecutorWithHandoff creates a new task executor with handoff support.
func NewExecutorWithHandoff(registry ExpertRegistry, config *OrchestratorConfig, handoffHandler HandoffHandlerInterface) *Executor {
	if config == nil {
		config = DefaultOrchestratorConfig()
	}
	return &Executor{
		registry:       registry,
		config:         config,
		handoffHandler: handoffHandler,
	}
}

// ExecutePlan executes all tasks in the plan using DAG scheduling and returns results.
func (e *Executor) ExecutePlan(ctx context.Context, plan *TaskPlan, callback EventCallback, traceID string) *ExecutionResult {
	result := &ExecutionResult{
		Plan:         plan,
		IsAggregated: false,
	}

	startTime := time.Now()

	// Initialize EventDispatcher
	dispatcher := NewEventDispatcher(traceID, callback)
	defer dispatcher.Close()

	slog.Info("executor: starting DAG plan execution",
		"trace_id", traceID,
		"tasks", len(plan.Tasks),
		"parallel", plan.Parallel)

	// Send plan event to frontend
	// Send plan event to frontend
	if callback != nil {
		e.sendPlanEvent(plan, dispatcher)
	}

	// Initialize DAG Scheduler
	scheduler, err := NewDAGScheduler(e, plan.Tasks, traceID, dispatcher)
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
		// Use safe accessors
		status := task.GetStatus()
		resultVal := task.GetResult()
		errorVal := task.GetError()

		if status == TaskStatusCompleted && resultVal != "" {
			results = append(results, resultVal)
		}
		if status == TaskStatusFailed && errorVal != "" {
			result.Errors = append(result.Errors, fmt.Sprintf("Task %s: %s", task.ID, errorVal))
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

// executeTask executes a single task.
func (e *Executor) executeTask(ctx context.Context, task *Task, index int, dispatcher *EventDispatcher, traceID string) error {
	return e.executeTaskWithHandoff(ctx, task, index, dispatcher, 0, traceID)
}

// executeTaskWithHandoff executes a single task with handoff depth tracking.
func (e *Executor) executeTaskWithHandoff(ctx context.Context, task *Task, index int, dispatcher *EventDispatcher, depth int, traceID string) error {
	startTime := time.Now()
	// Use safe accessor to mark running
	if err := task.MarkRunning(); err != nil {
		// Should not happen if logic is correct
		slog.Error("executor: failed to mark task running", "error", err)
	}

	// Log task start
	slog.Info("executor: task start",
		"trace_id", traceID,
		"task_id", task.ID,
		"agent", task.Agent,
		"dependencies", task.Dependencies,
	)

	// Send task_start event
	e.sendTaskStartEvent(task, -1, dispatcher)

	slog.Debug("executor: executing task",
		"trace_id", traceID,
		"id", task.ID,
		"agent", task.Agent,
		"purpose", task.Purpose)

	// 1. Context Injection
	// We need access to all tasks to resolve variables.
	// But executeSingleTask receives *Task.
	// The DAGScheduler should have already resolved inputs OR we pass the map here.
	// BETTER DESIGN: DAGScheduler calls ContextInjector BEFORE calling executeSingleTask.
	// So here we assume task.Input is already resolved.

	// Create result collector with thread-safe event forwarding
	resultCollector := newResultCollector(dispatcher)

	// Execute via expert registry with retry logic
	var err error
	maxRetries := e.config.MaxRetries
	backoff := e.config.RetryBackoff

	for i := 0; i <= maxRetries; i++ {
		// Use resultCollector.onEvent as callback
		err = e.registry.ExecuteExpert(ctx, task.Agent, task.Input, resultCollector.onEvent)
		if err == nil {
			break
		}

		// Only retry transient errors
		if !isTransientError(err) {
			slog.Warn("executor: task execution failed, non-transient error, not retrying",
				"trace_id", traceID,
				"task_id", task.ID,
				"error", err,
			)
			break
		}

		if i < maxRetries {
			slog.Warn("executor: task execution failed, retrying transient error",
				"trace_id", traceID,
				"task_id", task.ID,
				"attempt", i+1,
				"error", err,
			)
			time.Sleep(backoff)
			backoff *= 2 // Exponential backoff
		}
	}

	if err != nil {
		// Task status updated by caller (DAGScheduler) or here?
		// Let's update here for consistency using safe accessor
		task.SetError(err.Error())

		slog.Warn("executor: task failed",
			"trace_id", traceID,
			"task_id", task.ID,
			"error", err.Error(),
			"retry_count", depth,
		)

		// Try handoff if handler is available
		if e.handoffHandler != nil {
			// Create handoff context with depth tracking
			handOffCtx := NewHandoffContextWithDepth(depth, task.ID)
			// Pass raw callback for now? Or adapt HandoffHandler?
			// HandoffHandler expects EventCallback. We need to adapter dispatcher back to callback?
			// Or update HandoffHandler signature.
			// Ideally update HandoffHandler signature, but that affects interface.
			// Let's create an adapter for now to keep interface stable, or update interface.
			// Adapt:
			cb := func(t, d string) { dispatcher.Send(t, d) }
			handoffResult := e.handoffHandler.HandleTaskFailure(ctx, task, err, cb, handOffCtx)
			if handoffResult.Success && handoffResult.NewTask != nil {
				slog.Info("executor: attempting handoff",
					"trace_id", traceID,
					"task_id", task.ID,
					"from_agent", task.Agent,
					"to_agent", handoffResult.NewExpert,
					"depth", handoffResult.Depth)

				// Execute with new expert
				task.Agent = handoffResult.NewExpert
				task.Input = handoffResult.NewTask.Input
				// Reset status for retry/handoff
				task.SetStatus(TaskStatusPending)

				// Re-execute the task with new expert and updated context
				return e.executeTaskWithHandoff(ctx, task, index, dispatcher, handoffResult.Depth, traceID)
			}
		}

		// Send task_end event with error
		e.sendTaskEndEvent(task, -1, dispatcher)
		return err
	}

	// Success
	// Use safe accessor
	task.SetResult(resultCollector.getResult())

	duration := time.Since(startTime)
	slog.Info("executor: task complete",
		"trace_id", traceID,
		"task_id", task.ID,
		"status", task.GetStatus(),
		"duration_ms", duration.Milliseconds(),
	)

	// Send task_end event
	e.sendTaskEndEvent(task, -1, dispatcher)
	return nil
}

// sendPlanEvent sends the task plan to the frontend.
func (e *Executor) sendPlanEvent(plan *TaskPlan, dispatcher *EventDispatcher) {
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
	dispatcher.Send(EventTypePlan, string(eventJSON))
}

// sendTaskStartEvent sends a task start event to the frontend.
func (e *Executor) sendTaskStartEvent(task *Task, index int, dispatcher *EventDispatcher) {
	event := map[string]interface{}{
		"id":      task.ID,
		"index":   index, // -1 if unknown
		"agent":   task.Agent,
		"purpose": task.Purpose,
		"status":  string(task.GetStatus()),
	}
	eventJSON, err := json.Marshal(event)
	if err != nil {
		slog.Error("executor: failed to marshal task_start event", "error", err, "id", task.ID)
		return
	}
	dispatcher.Send(EventTypeTaskStart, string(eventJSON))
}

// sendTaskEndEvent sends a task end event to the frontend.
func (e *Executor) sendTaskEndEvent(task *Task, index int, dispatcher *EventDispatcher) {
	event := map[string]interface{}{
		"id":     task.ID,
		"index":  index,
		"agent":  task.Agent,
		"status": string(task.GetStatus()),
	}
	if errVal := task.GetError(); errVal != "" {
		event["error"] = errVal
	}
	eventJSON, err := json.Marshal(event)
	if err != nil {
		slog.Error("executor: failed to marshal task_end event", "error", err, "id", task.ID)
		return
	}
	dispatcher.Send(EventTypeTaskEnd, string(eventJSON))
}

// Max result size to prevent OOM (10MB)
const maxResultSize = 10 * 1024 * 1024

// resultCollector collects results from event callbacks.
type resultCollector struct {
	mu         sync.Mutex
	dispatcher *EventDispatcher
	result     strings.Builder
	truncated  bool
}

func newResultCollector(dispatcher *EventDispatcher) *resultCollector {
	return &resultCollector{
		dispatcher: dispatcher,
	}
}

func (rc *resultCollector) onEvent(eventType string, eventData string) {
	// Forward event via dispatcher (thread-safe, sequential)
	if rc.dispatcher != nil {
		rc.dispatcher.Send(eventType, eventData)
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

// getResult returns the collected result.
func (rc *resultCollector) getResult() string {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	return rc.result.String()
}

// transientErrorKeywords contains keywords that indicate a transient error.
var transientErrorKeywords = []string{
	"timeout",
	"timed out",
	"connection refused",
	"connection reset",
	"temporary failure",
	"service unavailable",
	"too many requests",
	"rate limit",
	"429",
	"503",
	"502",
	"504",
	"context deadline exceeded",
	"i/o timeout",
	"network unreachable",
	"no route to host",
	"transient",
	"retry",
	"temporary",
}

// isTransientError checks if an error is transient and worth retrying.
// It returns true if the error is likely temporary and retrying might help.
func isTransientError(err error) bool {
	if err == nil {
		return false
	}

	errMsg := err.Error()
	errMsgLower := strings.ToLower(errMsg)

	// Check for context cancellation/deadline - don't retry if explicitly cancelled
	if errors.Is(err, context.Canceled) {
		return false
	}

	// Check for deadline exceeded - these are often retryable
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}

	// Check for transient error keywords
	for _, keyword := range transientErrorKeywords {
		if strings.Contains(errMsgLower, strings.ToLower(keyword)) {
			return true
		}
	}

	return false
}
