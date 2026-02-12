// Package agent provides conversation context management for multi-turn dialogues.
// This module maintains state across conversation turns to enable handling
// of refinements like "change it to 3pm" without re-specifying the full context.
package agent

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"log/slog"

	"github.com/hrygo/divinesense/ai/services/schedule"
	"github.com/hrygo/divinesense/store"
)

// ConversationContext maintains state across conversation turns.
type ConversationContext struct {
	CreatedAt    time.Time
	UpdatedAt    time.Time
	WorkingState *WorkingState
	SessionID    string
	Timezone     string
	Turns        []ConversationTurn
	mu           sync.RWMutex
	UserID       int32
	// RouteSticky: Intent stickiness for short confirmations (Issue #163)
	LastRouteType ChatRouteType // Last successful route type
	LastRouteTime time.Time     // When the last route was made
}

// ConversationTurn represents a single turn in the conversation.
type ConversationTurn struct {
	Timestamp   time.Time
	UserInput   string
	AgentOutput string
	ToolCalls   []ToolCallRecord
}

// ToolCallRecord records a tool invocation.
type ToolCallRecord struct {
	Timestamp time.Time
	Tool      string
	Input     string
	Output    string
	Duration  time.Duration
	Success   bool
}

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

// NewConversationContext creates a new conversation context.
func NewConversationContext(sessionID string, userID int32, timezone string) *ConversationContext {
	return &ConversationContext{
		SessionID:    sessionID,
		UserID:       userID,
		Timezone:     timezone,
		Turns:        make([]ConversationTurn, 0),
		WorkingState: &WorkingState{CurrentStep: StepIdle},
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
}

// AddTurn adds a new turn to the conversation history.
func (c *ConversationContext) AddTurn(userInput, agentOutput string, toolCalls []ToolCallRecord) {
	c.mu.Lock()
	defer c.mu.Unlock()

	turn := ConversationTurn{
		UserInput:   userInput,
		AgentOutput: agentOutput,
		ToolCalls:   toolCalls,
		Timestamp:   time.Now(),
	}

	c.Turns = append(c.Turns, turn)
	c.UpdatedAt = time.Now()

	// Keep only last 10 turns to manage memory
	if len(c.Turns) > 10 {
		c.Turns = c.Turns[len(c.Turns)-10:]
	}
}

// SetLastRoute sets the last successful route for intent stickiness (Issue #163).
func (c *ConversationContext) SetLastRoute(routeType ChatRouteType) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.LastRouteType = routeType
	c.LastRouteTime = time.Now()
}

// GetLastRoute returns the last route type and whether it's within the sticky window.
func (c *ConversationContext) GetLastRoute() (ChatRouteType, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	// Sticky window: 5 minutes
	if c.LastRouteType != "" && time.Since(c.LastRouteTime) < 5*time.Minute {
		return c.LastRouteType, true
	}
	return "", false
}

// UpdateWorkingState updates the working state with new information.
func (c *ConversationContext) UpdateWorkingState(state *WorkingState) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.WorkingState = state
	c.UpdatedAt = time.Now()
}

// GetWorkingState returns a deep copy of the current working state.
func (c *ConversationContext) GetWorkingState() *WorkingState {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.WorkingState == nil {
		return nil
	}

	// Return a deep copy to avoid race conditions
	result := &WorkingState{
		LastIntent:   c.WorkingState.LastIntent,
		LastToolUsed: c.WorkingState.LastToolUsed,
		CurrentStep:  c.WorkingState.CurrentStep,
	}

	// Deep copy ProposedSchedule
	if c.WorkingState.ProposedSchedule != nil {
		result.ProposedSchedule = &ScheduleDraft{
			Title:         c.WorkingState.ProposedSchedule.Title,
			Description:   c.WorkingState.ProposedSchedule.Description,
			Location:      c.WorkingState.ProposedSchedule.Location,
			AllDay:        c.WorkingState.ProposedSchedule.AllDay,
			Timezone:      c.WorkingState.ProposedSchedule.Timezone,
			OriginalInput: c.WorkingState.ProposedSchedule.OriginalInput,
		}
		if c.WorkingState.ProposedSchedule.StartTime != nil {
			t := *c.WorkingState.ProposedSchedule.StartTime
			result.ProposedSchedule.StartTime = &t
		}
		if c.WorkingState.ProposedSchedule.EndTime != nil {
			t := *c.WorkingState.ProposedSchedule.EndTime
			result.ProposedSchedule.EndTime = &t
		}
		if c.WorkingState.ProposedSchedule.Recurrence != nil {
			// RecurrenceRule contains simple types, shallow copy is sufficient
			result.ProposedSchedule.Recurrence = c.WorkingState.ProposedSchedule.Recurrence
		}
		if c.WorkingState.ProposedSchedule.Confidence != nil {
			result.ProposedSchedule.Confidence = make(map[string]float32, len(c.WorkingState.ProposedSchedule.Confidence))
			for k, v := range c.WorkingState.ProposedSchedule.Confidence {
				result.ProposedSchedule.Confidence[k] = v
			}
		}
	}

	// Deep copy Conflicts slice
	if len(c.WorkingState.Conflicts) > 0 {
		result.Conflicts = make([]*store.Schedule, len(c.WorkingState.Conflicts))
		copy(result.Conflicts, c.WorkingState.Conflicts)
	}

	return result
}

// GetLastTurn returns a copy of the most recent conversation turn.
func (c *ConversationContext) GetLastTurn() *ConversationTurn {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if len(c.Turns) == 0 {
		return nil
	}

	// Return a copy, not a pointer to the slice element
	last := c.Turns[len(c.Turns)-1]
	return &last
}

// GetLastNTurns returns the last N conversation turns.
func (c *ConversationContext) GetLastNTurns(n int) []ConversationTurn {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if len(c.Turns) == 0 {
		return nil
	}

	start := 0
	if len(c.Turns) > n {
		start = len(c.Turns) - n
	}

	result := make([]ConversationTurn, len(c.Turns)-start)
	copy(result, c.Turns[start:])
	return result
}

// ExtractRefinement attempts to extract a refinement from user input
// based on the current working state.
// For example: "change to 3pm" when there's a proposed schedule.
func (c *ConversationContext) ExtractRefinement(userInput string) *ScheduleDraft {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Check if we have a working state with a proposed schedule
	if c.WorkingState == nil || c.WorkingState.ProposedSchedule == nil {
		return nil
	}

	// Check if the input looks like a refinement
	// Refinement patterns:
	// - Time modifications: "change to 3pm", "move to tomorrow", etc.
	// - Title modifications: "call it meeting", "change title to..."
	// - Location modifications: "at the office", "change location to..."

	refinement := &ScheduleDraft{}
	updated := false

	// Copy existing draft
	existing := c.WorkingState.ProposedSchedule

	// Check for time modification patterns
	lowerInput := lower(userInput)
	if contains(lowerInput, []string{"change to", "move to", "reschedule to", "set for"}) {
		// This looks like a time refinement - let parser handle it
		// Just indicate that this is a refinement
		refinement.OriginalInput = userInput
		updated = true
	}

	// Check for simple time patterns like "3pm", "tomorrow", etc.
	if containsAny(lowerInput, []string{"am", "pm", "today", "tomorrow", "monday", "tuesday", "wednesday", "thursday", "friday", "saturday", "sunday"}) {
		refinement.OriginalInput = userInput
		updated = true
	}

	// If we detected a refinement, copy over existing fields
	if updated && existing != nil {
		if refinement.Title == "" && existing.Title != "" {
			refinement.Title = existing.Title
		}
		if refinement.Description == "" && existing.Description != "" {
			refinement.Description = existing.Description
		}
		if refinement.Location == "" && existing.Location != "" {
			refinement.Location = existing.Location
		}
		if refinement.StartTime == nil && existing.StartTime != nil {
			t := *existing.StartTime
			refinement.StartTime = &t
		}
		if refinement.EndTime == nil && existing.EndTime != nil {
			t := *existing.EndTime
			refinement.EndTime = &t
		}
		refinement.Timezone = existing.Timezone
		refinement.AllDay = existing.AllDay
		refinement.Recurrence = existing.Recurrence

		slog.Debug("context: extracted refinement",
			"session_id", c.SessionID,
			"user_input", userInput,
			"existing_title", existing.Title,
			"refinement_title", refinement.Title)

		return refinement
	}

	return nil
}

// Clear resets the conversation context.
func (c *ConversationContext) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.Turns = make([]ConversationTurn, 0)
	c.WorkingState = &WorkingState{CurrentStep: StepIdle}
	c.UpdatedAt = time.Now()
}

// GetSummary returns a summary of the conversation context.
func (c *ConversationContext) GetSummary() ContextSummary {
	c.mu.RLock()
	defer c.mu.RUnlock()

	summary := ContextSummary{
		SessionID:   c.SessionID,
		UserID:      c.UserID,
		TurnCount:   len(c.Turns),
		CurrentStep: StepIdle,
		CreatedAt:   c.CreatedAt,
		UpdatedAt:   c.UpdatedAt,
	}

	if c.WorkingState != nil {
		summary.CurrentStep = c.WorkingState.CurrentStep
		summary.LastIntent = c.WorkingState.LastIntent
		summary.HasProposedSchedule = c.WorkingState.ProposedSchedule != nil
		summary.ConflictCount = len(c.WorkingState.Conflicts)
	}

	return summary
}

// ToJSON exports the conversation context to JSON for persistence.
func (c *ConversationContext) ToJSON() (string, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	data, err := json.Marshal(c)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// ToHistoryPrompt converts the conversation history to a string format suitable for LLM context.
// It formats turns as "User: ...\nAssistant: ..." and optionally includes tool usage summaries.
func (c *ConversationContext) ToHistoryPrompt() string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if len(c.Turns) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("Conversation History:\n")

	for _, turn := range c.Turns {
		sb.WriteString(fmt.Sprintf("User: %s\n", turn.UserInput))

		// Add tool summary if available, to provide context on what the agent did
		if len(turn.ToolCalls) > 0 {
			// Simplified tool summary: "Assistant (Action: tool_name): [Success/Fail]"
			// We avoid dumping full JSON output to save tokens, but give a hint of action.
			toolsUsed := make([]string, 0, len(turn.ToolCalls))
			for _, tc := range turn.ToolCalls {
				status := "OK"
				if !tc.Success {
					status = "Failed"
				}
				toolsUsed = append(toolsUsed, fmt.Sprintf("%s (%s)", tc.Tool, status))
			}
			sb.WriteString(fmt.Sprintf("System: Agent used tools: %s\n", strings.Join(toolsUsed, ", ")))
		}

		sb.WriteString(fmt.Sprintf("Assistant: %s\n", turn.AgentOutput))
	}

	result := sb.String()
	slog.Debug("ToHistoryPrompt generated",
		"session_id", c.SessionID,
		"turn_count", len(c.Turns),
		"length", len(result))

	return result
}

// ContextSummary provides a quick overview of the context state.
type ContextSummary struct {
	CreatedAt           time.Time
	UpdatedAt           time.Time
	SessionID           string
	CurrentStep         WorkflowStep
	LastIntent          string
	TurnCount           int
	ConflictCount       int
	UserID              int32
	HasProposedSchedule bool
}

// ContextStore manages conversation contexts for multiple sessions.
type ContextStore struct {
	contexts map[string]*ConversationContext
	mu       sync.RWMutex
}

// NewContextStore creates a new context store.
func NewContextStore() *ContextStore {
	return &ContextStore{
		contexts: make(map[string]*ConversationContext),
	}
}

// GetOrCreate retrieves or creates a conversation context.
func (s *ContextStore) GetOrCreate(sessionID string, userID int32, timezone string) *ConversationContext {
	s.mu.Lock()
	defer s.mu.Unlock()

	if ctx, exists := s.contexts[sessionID]; exists {
		return ctx
	}

	ctx := NewConversationContext(sessionID, userID, timezone)
	s.contexts[sessionID] = ctx
	return ctx
}

// Get retrieves a conversation context if it exists.
func (s *ContextStore) Get(sessionID string) *ConversationContext {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.contexts[sessionID]
}

// Delete removes a conversation context.
func (s *ContextStore) Delete(sessionID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.contexts, sessionID)
}

// CleanupOld removes contexts older than the specified duration.
func (s *ContextStore) CleanupOld(maxAge time.Duration) int {
	s.mu.Lock()
	defer s.mu.Unlock()

	cutoff := time.Now().Add(-maxAge)
	deleted := 0

	for sessionID, ctx := range s.contexts {
		if ctx.UpdatedAt.Before(cutoff) {
			delete(s.contexts, sessionID)
			deleted++
		}
	}

	return deleted
}

// Helper functions

// lower converts a string to lowercase using the standard library for proper Unicode support.
func lower(s string) string {
	return strings.ToLower(s)
}

func contains(s string, substrings []string) bool {
	for _, sub := range substrings {
		if len(s) >= len(sub) {
			for i := 0; i <= len(s)-len(sub); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
		}
	}
	return false
}

func containsAny(s string, substrings []string) bool {
	return contains(s, substrings)
}
