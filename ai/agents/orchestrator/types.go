// Package orchestrator implements the Orchestrator-Workers pattern for multi-agent coordination.
// It uses LLM to dynamically decompose tasks, dispatch to expert agents, and aggregate results.
package orchestrator

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"math/big"
	"sync"
	"time"

	agents "github.com/hrygo/divinesense/ai/agents"
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

	// mu protects concurrent access to Status, Result, and Error
	mu sync.RWMutex
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

// Thread-safe accessors

// SetStatus updates the task status thread-safely.
func (t *Task) SetStatus(status TaskStatus) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.Status = status
}

// GetStatus returns the current status thread-safely.
func (t *Task) GetStatus() TaskStatus {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.Status
}

// SetResult updates the task result and status thread-safely.
func (t *Task) SetResult(result string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.Result = result
	t.Status = TaskStatusCompleted
}

// GetResult returns the task result thread-safely.
func (t *Task) GetResult() string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.Result
}

// SetError updates the task error and status thread-safely.
func (t *Task) SetError(err string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.Error = err
	t.Status = TaskStatusFailed
}

// GetError returns the task error thread-safely.
func (t *Task) GetError() string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.Error
}

// SetSkipped marks the task as skipped with a reason.
func (t *Task) SetSkipped(reason string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.Error = reason
	t.Status = TaskStatusSkipped
}

// MarkRunning transitions the task to running state.
// Returns an error if the transition is invalid.
func (t *Task) MarkRunning() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.Status != TaskStatusPending {
		return errors.New("can only mark pending task as running")
	}
	t.Status = TaskStatusRunning
	return nil
}

// Complete transitions the task to completed state with a result.
// Returns an error if the transition is invalid.
func (t *Task) Complete(result string) error {
	t.mu.Lock()
	defer t.mu.Unlock()
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
	t.mu.Lock()
	defer t.mu.Unlock()
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

	// DirectResponse indicates that the task can be handled directly by LLM
	// without needing to call any expert agents. This is used for simple tasks
	// like summarization, translation, simple Q&A, etc.
	DirectResponse bool `json:"direct_response"`

	// Response is the LLM-generated response when DirectResponse is true.
	// This field is populated by the Decomposer when it determines the task
	// can be handled directly without expert agents.
	Response string `json:"response,omitempty"`
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

	// DefaultLanguage is the default language for aggregation
	DefaultLanguage string `json:"default_language"`

	// MaxRetries is the maximum number of retries for transient errors
	MaxRetries int `json:"max_retries"`

	// RetryBackoff is the initial backoff duration for retries
	RetryBackoff time.Duration `json:"retry_backoff"`
}

// DefaultOrchestratorConfig returns the default configuration.
func DefaultOrchestratorConfig() *OrchestratorConfig {
	return &OrchestratorConfig{
		MaxParallelTasks:   3,
		EnableAggregation:  true,
		EnableHandoff:      true, // Enable handoff by default for better expert coordination
		DecompositionModel: "default",
		AggregationModel:   "default",
		DefaultLanguage:    "zh",
		MaxRetries:         3,
		RetryBackoff:       time.Second,
	}
}

// ExpertRegistry defines the interface for accessing expert agents.
// It allows the orchestrator to discover and invoke expert agents dynamically.
type ExpertRegistry interface {
	// GetAvailableExperts returns the list of available expert agent names
	GetAvailableExperts() []string

	// GetExpertDescription returns a description of what an expert agent can do
	GetExpertDescription(name string) string

	// GetExpertConfig returns the self-cognition configuration of an expert agent
	GetExpertConfig(name string) *agents.ParrotSelfCognition

	// ExecuteExpert executes a task with the specified expert agent
	ExecuteExpert(ctx context.Context, expertName string, input string, callback EventCallback) error

	// GetIntentKeywords returns a map of intent names to their related keywords.
	// This enables sticky routing to dynamically load keywords from expert configurations.
	GetIntentKeywords() map[string][]string
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

// TaskContext holds the execution context for a task, including trace_id for observability.
type TaskContext struct {
	// TraceID is the unique identifier for tracing the entire request flow
	TraceID string
	// UserID is the user who initiated the request
	UserID int32
	// BlockID is the block ID associated with this task
	BlockID int64
	// ParentTaskID is the ID of the parent task (for subtasks)
	ParentTaskID string
}

func randomString(n int) string {
	const letters = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	b := make([]byte, n)
	letterLen := big.NewInt(int64(len(letters)))
	for i := range b {
		// Use crypto/rand for cryptographically secure random number
		num, err := rand.Int(rand.Reader, letterLen)
		if err != nil {
			// Fallback to time-based if crypto/rand fails (should never happen)
			slog.Warn("randomString: crypto rand failed, using fallback", "error", err)
			b[i] = letters[time.Now().UnixNano()%int64(len(letters))]
			continue
		}
		b[i] = letters[num.Int64()]
	}
	return string(b)
}

// GenerateTraceID generates a new trace ID using crypto/rand for secure random bytes.
func GenerateTraceID() string {
	// Generate 16 bytes of random data for secure trace ID
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback if crypto/rand fails
		slog.Warn("GenerateTraceID: crypto rand failed, using fallback", "error", err)
		return fmt.Sprintf("trace-%d-%s", time.Now().UnixMilli(), randomString(12))
	}
	return fmt.Sprintf("trace-%d-%s", time.Now().UnixMilli(), hex.EncodeToString(bytes)[:12])
}
