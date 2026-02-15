// Package orchestrator implements the Orchestrator-Workers pattern for multi-agent coordination.
// It uses LLM to dynamically decompose tasks, dispatch to expert agents, and aggregate results.
package orchestrator

import (
	"context"
	"errors"
)

// Task represents a single task to be executed by an expert agent.
type Task struct {
	// ID is a unique identifier for the task (e.g., "task_1", "task_2")
	ID string `json:"id,omitempty"`

	// Agent is the name of the expert agent to handle this task (e.g., "memo", "schedule")
	Agent string `json:"agent"`

	// Input is the specific input for this task
	Input string `json:"input"`

	// Purpose describes why this task is needed (for transparency)
	Purpose string `json:"purpose"`

	// Dependencies is a list of task IDs that must complete before this task can start
	// Example: ["task_1"] means this task depends on task_1's result
	Dependencies []string `json:"dependencies,omitempty"`

	// Result contains the execution result (populated after execution)
	Result string `json:"result,omitempty"`

	// Error contains any error that occurred during execution
	Error string `json:"error,omitempty"`

	// Status indicates the current status of the task
	Status TaskStatus `json:"status"`
}

// NewTask creates a new task with validated fields and default status.
func NewTask(agent, input, purpose string) (*Task, error) {
	if agent == "" {
		return nil, errors.New("agent cannot be empty")
	}
	if input == "" {
		return nil, errors.New("input cannot be empty")
	}
	return &Task{
		Agent:   agent,
		Input:   input,
		Purpose: purpose,
		Status:  TaskStatusPending,
	}, nil
}

// MarkRunning transitions the task to running state.
// Returns an error if the transition is invalid.
func (t *Task) MarkRunning() error {
	if t.Status != TaskStatusPending {
		return errors.New("can only mark pending task as running")
	}
	t.Status = TaskStatusRunning
	return nil
}

// Complete transitions the task to completed state with a result.
// Returns an error if the transition is invalid.
func (t *Task) Complete(result string) error {
	if t.Status != TaskStatusRunning {
		return errors.New("can only complete running task")
	}
	t.Status = TaskStatusCompleted
	t.Result = result
	return nil
}

// Fail transitions the task to failed state with an error message.
// Returns an error if the transition is invalid.
func (t *Task) Fail(errMsg string) error {
	if t.Status != TaskStatusRunning {
		return errors.New("can only fail running task")
	}
	t.Status = TaskStatusFailed
	t.Error = errMsg
	return nil
}

// TaskStatus represents the status of a task.
type TaskStatus string

const (
	TaskStatusPending   TaskStatus = "pending"
	TaskStatusRunning   TaskStatus = "running"
	TaskStatusCompleted TaskStatus = "completed"
	// TaskStatusFailed indicates the task failed
	TaskStatusFailed TaskStatus = "failed"
	// TaskStatusSkipped indicates the task was skipped due to upstream failure
	TaskStatusSkipped TaskStatus = "skipped"
)

// IsTerminal returns true if the status is a final state (Completed, Failed, Skipped).
func (ts TaskStatus) IsTerminal() bool {
	return ts == TaskStatusCompleted || ts == TaskStatusFailed || ts == TaskStatusSkipped
}

// TaskPlan represents the overall plan for handling a user request.
type TaskPlan struct {
	// Analysis is the LLM's analysis of the user request
	Analysis string `json:"analysis"`

	// Tasks are the decomposed tasks to execute
	Tasks []*Task `json:"tasks"`

	// Parallel indicates whether tasks can be executed in parallel
	Parallel bool `json:"parallel"`

	// Aggregate indicates whether results need to be aggregated
	Aggregate bool `json:"aggregate"`
}

// ExecutionResult represents the result of executing a task plan.
type ExecutionResult struct {
	// Plan is the original task plan
	Plan *TaskPlan `json:"plan"`

	// FinalResponse is the aggregated response (if aggregate=true)
	FinalResponse string `json:"final_response"`

	// IsAggregated indicates whether the response was aggregated from multiple results
	IsAggregated bool `json:"is_aggregated"`

	// TokenUsage tracks token consumption
	TokenUsage TokenUsage `json:"token_usage"`

	// Errors contains any errors that occurred during execution
	Errors []string `json:"errors,omitempty"`
}

// TokenUsage tracks token consumption for the orchestration.
type TokenUsage struct {
	InputTokens      int32 `json:"input_tokens"`
	OutputTokens     int32 `json:"output_tokens"`
	CacheWriteTokens int32 `json:"cache_write_tokens"`
	CacheReadTokens  int32 `json:"cache_read_tokens"`
}

// OrchestratorConfig contains configuration for the orchestrator.
type OrchestratorConfig struct {
	// MaxParallelTasks is the maximum number of tasks to execute in parallel
	MaxParallelTasks int `json:"max_parallel_tasks"`

	// EnableAggregation determines whether to aggregate multi-agent results
	EnableAggregation bool `json:"enable_aggregation"`

	// EnableHandoff determines whether to enable expert handoff on capability mismatch
	EnableHandoff bool `json:"enable_handoff"`

	// DecompositionModel is the model to use for task decomposition
	DecompositionModel string `json:"decomposition_model"`

	// AggregationModel is the model to use for result aggregation
	AggregationModel string `json:"aggregation_model"`
}

// DefaultOrchestratorConfig returns the default configuration.
func DefaultOrchestratorConfig() *OrchestratorConfig {
	return &OrchestratorConfig{
		MaxParallelTasks:   3,
		EnableAggregation:  true,
		EnableHandoff:      true, // Enable handoff by default for better expert coordination
		DecompositionModel: "default",
		AggregationModel:   "default",
	}
}

// ExpertRegistry defines the interface for accessing expert agents.
// It allows the orchestrator to discover and invoke expert agents dynamically.
type ExpertRegistry interface {
	// GetAvailableExperts returns the list of available expert agent names
	GetAvailableExperts() []string

	// GetExpertDescription returns a description of what an expert agent can do
	GetExpertDescription(name string) string

	// ExecuteExpert executes a task with the specified expert agent
	ExecuteExpert(ctx context.Context, expertName string, input string, callback EventCallback) error
}

// EventCallback is the callback function for streaming events to the frontend.
type EventCallback func(eventType string, eventData string)

// HandoffHandlerInterface defines the interface for handling task handoff between experts.
// This enables dependency injection for better testability.
type HandoffHandlerInterface interface {
	// HandleTaskFailure handles task failure and determines if handoff is appropriate.
	HandleTaskFailure(ctx context.Context, task *Task, err error, callback EventCallback, handOffContext *HandoffContext) *HandoffResult

	// HandleCannotComplete processes a cannot_complete event and determines next action.
	HandleCannotComplete(ctx context.Context, task *Task, reason CannotCompleteReason, callback EventCallback, handOffContext *HandoffContext) *HandoffResult
}
