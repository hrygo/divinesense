// Package schedule provides schedule domain context types and helpers.
package schedule

import (
	"time"

	"github.com/hrygo/divinesense/ai/services/schedule"
	"github.com/hrygo/divinesense/store"
)

// ContextKey is the key for schedule context in ConversationContext.Extensions.
const ContextKey = "schedule"

// WorkingState tracks the agent's current understanding and work in progress.
type WorkingState struct {
	ProposedSchedule *ScheduleDraft
	LastIntent       string
	LastToolUsed     string
	CurrentStep      WorkflowStep
	Conflicts        []*store.Schedule
}

// ScheduleDraft represents a partially specified schedule.
type ScheduleDraft struct {
	StartTime     *time.Time
	EndTime       *time.Time
	Recurrence    *schedule.RecurrenceRule
	Confidence    map[string]float32
	Title         string
	Description   string
	Location      string
	Timezone      string
	OriginalInput string
	AllDay        bool
}

// WorkflowStep represents the current step in the scheduling workflow.
type WorkflowStep string

const (
	StepIdle            WorkflowStep = "idle"
	StepParsing         WorkflowStep = "parsing"
	StepConflictCheck   WorkflowStep = "conflict_check"
	StepConflictResolve WorkflowStep = "conflict_resolve"
	StepConfirming      WorkflowStep = "confirming"
	StepCompleted       WorkflowStep = "completed"
)

// ContextProvider is an interface for accessing domain extensions.
// This avoids circular dependency with the agent package.
type ContextProvider interface {
	GetExtension(key string) any
	SetExtension(key string, value any)
}

// GetWorkingState retrieves the schedule working state from context.
// Returns nil if no schedule context exists.
func GetWorkingState(ctx ContextProvider) *WorkingState {
	if ctx == nil {
		return nil
	}
	ext := ctx.GetExtension(ContextKey)
	if ext == nil {
		return nil
	}
	if ws, ok := ext.(*WorkingState); ok {
		return ws
	}
	return nil
}

// SetWorkingState stores the schedule working state in context.
func SetWorkingState(ctx ContextProvider, ws *WorkingState) {
	if ctx == nil {
		return
	}
	ctx.SetExtension(ContextKey, ws)
}

// NewWorkingState creates a new working state with idle step.
func NewWorkingState() *WorkingState {
	return &WorkingState{
		CurrentStep: StepIdle,
	}
}
