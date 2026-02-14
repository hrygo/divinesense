package orchestrator

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
)

// Handoff event types
const (
	// EventTypeCannotComplete indicates an expert cannot complete the task.
	EventTypeCannotComplete = "cannot_complete"
	// EventTypeHandoffStart indicates a handoff has started.
	EventTypeHandoffStart = "handoff_start"
	// EventTypeHandoffEnd indicates a handoff has completed.
	EventTypeHandoffEnd = "handoff_end"
)

// CannotCompleteReason defines reasons why an expert cannot complete a task.
type CannotCompleteReason struct {
	// MissingCapabilities lists capabilities that the expert lacks.
	MissingCapabilities []string `json:"missing_capabilities"`
	// OriginalError is the original error message from the expert.
	OriginalError string `json:"original_error"`
	// SuggestedExpert is an optional hint about which expert might help.
	SuggestedExpert string `json:"suggested_expert,omitempty"`
}

// HandoffResult contains the result of a handoff operation.
type HandoffResult struct {
	// Success indicates whether the handoff was successful.
	Success bool `json:"success"`
	// NewExpert is the name of the expert that took over (if successful).
	NewExpert string `json:"new_expert,omitempty"`
	// NewTask is the new task created for the alternative expert (if any).
	NewTask *Task `json:"new_task,omitempty"`
	// Error is the error message if handoff failed.
	Error string `json:"error,omitempty"`
	// Attempts records the number of handoff attempts.
	Attempts int `json:"attempts"`
}

// HandoffHandler handles the handoff of tasks between expert agents.
type HandoffHandler struct {
	capabilityMap *CapabilityMap
	maxAttempts   int
}

// NewHandoffHandler creates a new HandoffHandler.
func NewHandoffHandler(capabilityMap *CapabilityMap, maxAttempts int) *HandoffHandler {
	if maxAttempts <= 0 {
		maxAttempts = 2 // Default max handoff attempts
	}
	return &HandoffHandler{
		capabilityMap: capabilityMap,
		maxAttempts:   maxAttempts,
	}
}

// HandleCannotComplete processes a cannot_complete event and determines next action.
func (h *HandoffHandler) HandleCannotComplete(
	ctx context.Context,
	task *Task,
	reason CannotCompleteReason,
	callback EventCallback,
) *HandoffResult {

	slog.Info("handoff: processing cannot_complete",
		"task_id", task.ID,
		"agent", task.Agent,
		"missing", reason.MissingCapabilities)

	result := &HandoffResult{
		Attempts: 1,
	}

	// Try to find an alternative expert for each missing capability
	for _, cap := range reason.MissingCapabilities {
		alternatives := h.capabilityMap.FindAlternativeExperts(cap, task.Agent)

		if len(alternatives) == 0 {
			slog.Warn("handoff: no alternative expert found",
				"capability", cap,
				"current_expert", task.Agent)
			continue
		}

		// Select the first available alternative
		selectedExpert := alternatives[0]

		// Send handoff_start event
		if callback != nil {
			h.sendHandoffStartEvent(task, selectedExpert, cap, callback)
		}

		// Create new task for the alternative expert
		newTask, err := NewTask(
			selectedExpert.Name,
			task.Input,
			task.Purpose,
		)
		if err != nil {
			result.Error = err.Error()
			return result
		}

		result.Success = true
		result.NewExpert = selectedExpert.Name
		result.NewTask = newTask

		slog.Info("handoff: created new task",
			"original_task", task.ID,
			"new_task", newTask.ID,
			"new_expert", selectedExpert.Name)

		// Send handoff_end event
		if callback != nil {
			h.sendHandoffEndEvent(task, result, callback)
		}

		return result
	}

	result.Error = "no suitable expert found for missing capabilities"
	return result
}

// HandleTaskFailure handles task failure and determines if handoff is appropriate.
func (h *HandoffHandler) HandleTaskFailure(
	ctx context.Context,
	task *Task,
	err error,
	callback EventCallback,
) *HandoffResult {

	// Analyze the error to determine if handoff is appropriate
	reason := h.analyzeFailureReason(task.Agent, err)

	// If no clear missing capabilities, return failure
	if len(reason.MissingCapabilities) == 0 {
		return &HandoffResult{
			Success: false,
			Error:   err.Error(),
		}
	}

	return h.HandleCannotComplete(ctx, task, reason, callback)
}

// analyzeFailureReason analyzes an error to determine missing capabilities.
// This is a simple keyword-based implementation that can be extended
// with LLM-based analysis for more sophisticated error understanding.
func (h *HandoffHandler) analyzeFailureReason(_ string, err error) CannotCompleteReason {
	errMsg := err.Error()
	reason := CannotCompleteReason{
		OriginalError: errMsg,
	}

	// Keyword to capability mapping
	// These keywords indicate which capabilities might be needed
	capabilityKeywords := map[string][]string{
		// Schedule-related keywords
		"日程":       {"日程管理", "创建日程", "calendar", "schedule", "event", "会议", "安排"},
		"安排":       {"日程管理", "创建日程", "calendar", "schedule", "event"},
		"calendar": {"日程管理", "calendar", "schedule"},
		"schedule": {"日程管理", "calendar", "schedule"},
		"会议":       {"日程管理", "创建日程", "calendar", "schedule"},

		// Memo-related keywords
		"笔记":   {"笔记搜索", "搜索笔记", "note", "memo", "文档"},
		"搜索":   {"笔记搜索", "搜索笔记", "note", "memo", "文档"},
		"note": {"笔记搜索", "note", "memo"},
		"memo": {"笔记搜索", "note", "memo"},
		"文档":   {"笔记搜索", "note", "memo", "文档"},

		// Generic "cannot handle" patterns
		"无法处理":                {},
		"cannot handle":       {},
		"unable to":           {},
		"不在能力范围内":             {},
		"超出能力":                {},
		"not in capabilities": {},
	}

	for keyword, caps := range capabilityKeywords {
		if strings.Contains(strings.ToLower(errMsg), strings.ToLower(keyword)) {
			reason.MissingCapabilities = append(reason.MissingCapabilities, caps...)
		}
	}

	// Remove duplicates
	if len(reason.MissingCapabilities) > 0 {
		seen := make(map[string]bool)
		var unique []string
		for _, cap := range reason.MissingCapabilities {
			if !seen[cap] {
				seen[cap] = true
				unique = append(unique, cap)
			}
		}
		reason.MissingCapabilities = unique
	}

	return reason
}

// sendHandoffStartEvent sends a handoff_start event to the frontend.
func (h *HandoffHandler) sendHandoffStartEvent(
	originalTask *Task,
	newExpert *ExpertInfo,
	capability string,
	callback EventCallback,
) {
	event := map[string]interface{}{
		"original_task":  originalTask.ID,
		"original_agent": originalTask.Agent,
		"new_agent":      newExpert.Name,
		"capability":     capability,
	}
	h.sendEvent(EventTypeHandoffStart, event, callback)
}

// sendHandoffEndEvent sends a handoff_end event to the frontend.
func (h *HandoffHandler) sendHandoffEndEvent(
	originalTask *Task,
	result *HandoffResult,
	callback EventCallback,
) {
	event := map[string]interface{}{
		"original_task": originalTask.ID,
		"success":       result.Success,
		"new_agent":     result.NewExpert,
		"error":         result.Error,
	}
	h.sendEvent(EventTypeHandoffEnd, event, callback)
}

// sendEvent marshals and sends an event via the callback.
func (h *HandoffHandler) sendEvent(eventType string, data map[string]any, callback EventCallback) {
	eventJSON, err := json.Marshal(data)
	if err != nil {
		slog.Error("handoff: failed to marshal event", "error", err)
		return
	}
	callback(eventType, string(eventJSON))
}

// Ensure HandoffHandler implements error handling
var _ error = (*HandoffError)(nil)

// HandoffError represents an error that occurred during handoff.
type HandoffError struct {
	OriginalError error
	TaskID        string
	Expert        string
}

func (e *HandoffError) Error() string {
	if e.OriginalError != nil {
		return e.OriginalError.Error()
	}
	return "handoff error"
}

func (e *HandoffError) Unwrap() error {
	return e.OriginalError
}

// NewHandoffError creates a new HandoffError.
func NewHandoffError(taskID, expert string, err error) *HandoffError {
	return &HandoffError{
		OriginalError: err,
		TaskID:        taskID,
		Expert:        expert,
	}
}

// IsHandoffError checks if an error is a HandoffError.
func IsHandoffError(err error) bool {
	var handoffErr *HandoffError
	return errors.As(err, &handoffErr)
}
