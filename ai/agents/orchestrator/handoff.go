package orchestrator

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"
)

// Handoff configuration constants
const (
	// HandoffTimeout defines the maximum time allowed for a handoff operation.
	HandoffTimeout = 30 * time.Second

	// MaxHandoffDepth defines the maximum depth of nested handoffs.
	MaxHandoffDepth = 3
)

// Handoff event types
const (
	// EventTypeCannotComplete indicates an expert cannot complete the task.
	EventTypeCannotComplete = "cannot_complete"
	// EventTypeHandoffStart indicates a handoff has started.
	EventTypeHandoffStart = "handoff_start"
	// EventTypeHandoffEnd indicates a handoff has completed.
	EventTypeHandoffEnd = "handoff_end"
	// EventTypeHandoffFail indicates a handoff has failed.
	EventTypeHandoffFail = "handoff_fail"
)

// HandoffFailReason defines reasons why a handoff might fail.
type HandoffFailReason string

const (
	// FailNoMatchingExpert indicates no expert found with required capabilities.
	FailNoMatchingExpert HandoffFailReason = "no_matching_expert"
	// FailTargetUnavailable indicates the target expert is unavailable.
	FailTargetUnavailable HandoffFailReason = "target_unavailable"
	// FailTargetExecution indicates the target expert failed to execute.
	FailTargetExecution HandoffFailReason = "target_execution"
	// FailMaxDepthExceeded indicates maximum handoff depth was exceeded.
	FailMaxDepthExceeded HandoffFailReason = "max_depth_exceeded"
	// FailTimeout indicates the handoff operation timed out.
	FailTimeout HandoffFailReason = "timeout"
	// FailContextLost indicates the context was lost during handoff.
	FailContextLost HandoffFailReason = "context_lost"
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
	// Reason is the failure reason if handoff failed.
	Reason HandoffFailReason `json:"reason,omitempty"`
	// FallbackMessage is a user-friendly message when handoff fails.
	FallbackMessage string `json:"fallback_message,omitempty"`
	// Attempts records the number of handoff attempts.
	Attempts int `json:"attempts"`
	// Depth records the current handoff depth.
	Depth int `json:"depth"`
}

// HandoffContext holds the context for a handoff operation.
type HandoffContext struct {
	// Depth is the current handoff depth.
	Depth int
	// StartTime is when the handoff chain started.
	StartTime time.Time
	// ParentTaskID is the ID of the parent task that initiated this handoff.
	ParentTaskID string
}

// NewHandoffContext creates a new HandoffContext.
func NewHandoffContext() *HandoffContext {
	return &HandoffContext{
		Depth:     0,
		StartTime: time.Now(),
	}
}

// NewHandoffContextWithDepth creates a new HandoffContext with a given depth.
func NewHandoffContextWithDepth(depth int, parentTaskID string) *HandoffContext {
	return &HandoffContext{
		Depth:        depth,
		StartTime:    time.Now(),
		ParentTaskID: parentTaskID,
	}
}

// IsMaxDepthExceeded checks if the maximum handoff depth has been exceeded.
func (c *HandoffContext) IsMaxDepthExceeded() bool {
	return c.Depth >= MaxHandoffDepth
}

// IsTimeoutExceeded checks if the handoff operation has exceeded the timeout.
func (c *HandoffContext) IsTimeoutExceeded() bool {
	return time.Since(c.StartTime) > HandoffTimeout
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
	handOffContext *HandoffContext,
) *HandoffResult {

	slog.Info("handoff: processing cannot_complete",
		"task_id", task.ID,
		"agent", task.Agent,
		"missing", reason.MissingCapabilities,
		"depth", handOffContext.Depth)

	result := &HandoffResult{
		Attempts: 1,
		Depth:    handOffContext.Depth,
	}

	// Check timeout first
	if handOffContext.IsTimeoutExceeded() {
		slog.Warn("handoff: timeout exceeded",
			"task_id", task.ID,
			"elapsed", time.Since(handOffContext.StartTime))
		result.Success = false
		result.Error = "handoff operation timed out"
		result.Reason = FailTimeout
		result.FallbackMessage = h.buildFallbackResponse(task, FailTimeout)
		// Send handoff_fail event
		if callback != nil {
			h.sendHandoffFailEvent(task, result, callback)
		}
		return result
	}

	// Check max depth
	if handOffContext.IsMaxDepthExceeded() {
		slog.Warn("handoff: max depth exceeded",
			"task_id", task.ID,
			"depth", handOffContext.Depth,
			"max_depth", MaxHandoffDepth)
		result.Success = false
		result.Error = fmt.Sprintf("maximum handoff depth (%d) exceeded", MaxHandoffDepth)
		result.Reason = FailMaxDepthExceeded
		result.FallbackMessage = h.buildFallbackResponse(task, FailMaxDepthExceeded)
		// Send handoff_fail event
		if callback != nil {
			h.sendHandoffFailEvent(task, result, callback)
		}
		return result
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
			result.Reason = FailTargetExecution
			result.FallbackMessage = h.buildFallbackResponse(task, FailTargetExecution)
			// Send handoff_fail event
			if callback != nil {
				h.sendHandoffFailEvent(task, result, callback)
			}
			return result
		}

		result.Success = true
		result.NewExpert = selectedExpert.Name
		result.NewTask = newTask
		result.Depth = handOffContext.Depth + 1

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

	result.Success = false
	result.Error = "no suitable expert found for missing capabilities"
	result.Reason = FailNoMatchingExpert
	result.FallbackMessage = h.buildFallbackResponse(task, FailNoMatchingExpert)
	// Send handoff_fail event
	if callback != nil {
		h.sendHandoffFailEvent(task, result, callback)
	}
	return result
}

// buildFallbackResponse generates a user-friendly fallback message when handoff fails.
// Note: This message is shown to users, so it should NOT contain any user input
// to prevent sensitive information leakage.
func (h *HandoffHandler) buildFallbackResponse(_ *Task, reason HandoffFailReason) string {
	var message string

	switch reason {
	case FailNoMatchingExpert:
		message = "抱歉，当前任务需要的能力超出了我可以协调的范围。建议您重新描述任务，或明确指定需要的帮助类型（如「搜索笔记」或「创建日程」）。"
	case FailTargetUnavailable:
		message = "抱歉，目标的专家代理暂时不可用。请稍后重试，或尝试修改任务描述。"
	case FailTargetExecution:
		message = "抱歉，在将任务转交给合适的专家时遇到问题。请尝试重新描述您的需求。"
	case FailMaxDepthExceeded:
		message = "抱歉，您的请求经过多次转接仍未找到合适的处理方式。请尝试将任务拆分成更简单的步骤，或直接说明您需要什么帮助。"
	case FailTimeout:
		message = "抱歉，处理您的请求超时了。请尝试简化任务或稍后重试。"
	case FailContextLost:
		message = "抱歉，处理您的请求时出现了连接问题。请重新提交您的请求。"
	default:
		message = "抱歉，无法完成您的请求。请尝试重新描述您的需求。"
	}

	return message
}

// HandleTaskFailure handles task failure and determines if handoff is appropriate.
func (h *HandoffHandler) HandleTaskFailure(
	ctx context.Context,
	task *Task,
	err error,
	callback EventCallback,
	handOffContext *HandoffContext,
) *HandoffResult {

	// Analyze the error to determine if handoff is appropriate
	reason := h.analyzeFailureReason(task.Agent, err)

	// If no clear missing capabilities, return failure
	// Note: OriginalError is logged internally but not exposed to users for security
	if len(reason.MissingCapabilities) == 0 {
		return &HandoffResult{
			Success:         false,
			Error:           "任务执行失败，请重试", // Sanitized error message for user
			FallbackMessage: h.buildFallbackResponse(task, FailTargetExecution),
			Reason:          FailTargetExecution,
		}
	}

	return h.HandleCannotComplete(ctx, task, reason, callback, handOffContext)
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

// sendHandoffFailEvent sends a handoff_fail event to the frontend when handoff fails.
func (h *HandoffHandler) sendHandoffFailEvent(
	originalTask *Task,
	result *HandoffResult,
	callback EventCallback,
) {
	event := map[string]interface{}{
		"original_task":    originalTask.ID,
		"original_agent":   originalTask.Agent,
		"success":          result.Success,
		"error":            result.Error,
		"reason":           string(result.Reason),
		"fallback_message": result.FallbackMessage,
	}
	h.sendEvent(EventTypeHandoffFail, event, callback)
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

// SimpleHandoffRequest is a simplified request for ChatRouter integration.
// This avoids circular imports between agent and orchestrator packages.
type SimpleHandoffRequest struct {
	TaskID   string
	Agent    string
	Input    string
	Capacity string
	Reason   string
}

// SimpleHandoffResult is a simplified result for ChatRouter integration.
type SimpleHandoffResult struct {
	Success         bool
	FromExpert      string
	ToExpert        string
	NewTaskInput    string
	Error           string
	FallbackMessage string
}

// HandleSimpleHandoff handles a simplified handoff request from ChatRouter.
// This is a wrapper around the full HandleCannotComplete method.
func (h *HandoffHandler) HandleSimpleHandoff(req SimpleHandoffRequest) SimpleHandoffResult {
	// Convert to internal types
	task := &Task{
		ID:    req.TaskID,
		Agent: req.Agent,
		Input: req.Input,
	}

	reason := CannotCompleteReason{
		MissingCapabilities: []string{req.Capacity},
		OriginalError:       req.Reason,
	}

	// Use a context with timeout to allow cancellation
	ctx, cancel := context.WithTimeout(context.Background(), HandoffTimeout)
	defer cancel()

	callback := func(eventType string, eventData string) {}
	handOffContext := NewHandoffContext()

	result := h.HandleCannotComplete(ctx, task, reason, callback, handOffContext)

	return SimpleHandoffResult{
		Success:         result.Success,
		FromExpert:      req.Agent,
		ToExpert:        result.NewExpert,
		NewTaskInput:    "",
		Error:           result.Error,
		FallbackMessage: result.FallbackMessage,
	}
}

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
