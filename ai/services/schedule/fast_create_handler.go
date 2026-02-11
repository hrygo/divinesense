package schedule

import (
	"context"
	"fmt"
)

// ResponseType represents the type of agent response.
type ResponseType string

const (
	ResponseTypeFastCreate ResponseType = "fast_create"
	ResponseTypeFallback   ResponseType = "fallback"
	ResponseTypeError      ResponseType = "error"
)

// Action represents a user action button.
type Action struct {
	Data  any    `json:"data,omitempty"`
	Type  string `json:"type"`
	Label string `json:"label"`
}

// AgentResponse represents the response from the fast create handler.
type AgentResponse struct {
	Type    ResponseType   `json:"type"`
	Message string         `json:"message"`
	Data    map[string]any `json:"data,omitempty"`
	Actions []Action       `json:"actions,omitempty"`
}

// FastCreateHandler handles fast schedule creation.
type FastCreateHandler struct {
	parser *FastCreateParser
}

// NewFastCreateHandler creates a new FastCreateHandler.
func NewFastCreateHandler(parser *FastCreateParser) *FastCreateHandler {
	return &FastCreateHandler{
		parser: parser,
	}
}

// Handle processes user input for fast schedule creation.
func (h *FastCreateHandler) Handle(ctx context.Context, userID int32, input string) (*AgentResponse, error) {
	// Try fast parsing
	result, err := h.parser.Parse(ctx, userID, input)
	if err != nil {
		// Return nil response on error to ensure caller checks error first
		return nil, err
	}

	if !result.CanFastCreate {
		// Fallback to normal flow
		return &AgentResponse{
			Type:    ResponseTypeFallback,
			Message: "需要更多信息，请确认以下内容：",
			Data: map[string]any{
				"missing_fields": result.MissingFields,
				"partial":        result.Schedule,
			},
		}, nil
	}

	// Generate preview
	preview := generatePreview(result.Schedule)

	return &AgentResponse{
		Type:    ResponseTypeFastCreate,
		Message: "已识别日程，请确认：",
		Data: map[string]any{
			"preview":    preview,
			"schedule":   result.Schedule,
			"confidence": result.Confidence,
		},
		Actions: []Action{
			{Type: "confirm", Label: "确认创建", Data: result.Schedule},
			{Type: "edit", Label: "修改", Data: result.Schedule},
			{Type: "cancel", Label: "取消"},
		},
	}, nil
}

// TryFastCreate attempts fast creation and returns whether it succeeded.
// This can be used by the scheduler agent to decide whether to use fast path.
func (h *FastCreateHandler) TryFastCreate(ctx context.Context, userID int32, input string) (*FastCreateResult, bool) {
	result, err := h.parser.Parse(ctx, userID, input)
	if err != nil {
		return nil, false
	}
	return result, result.CanFastCreate
}

// generatePreview generates a human-readable preview of the schedule.
func generatePreview(schedule *ScheduleRequest) string {
	if schedule == nil {
		return ""
	}

	startStr := schedule.StartTime.Format("01月02日 15:04")
	endStr := schedule.EndTime.Format("15:04")

	preview := fmt.Sprintf(
		"📅 %s\n⏰ %s - %s\n⏱️ %d 分钟",
		schedule.Title,
		startStr,
		endStr,
		schedule.Duration,
	)

	if schedule.Location != "" {
		preview += fmt.Sprintf("\n📍 %s", schedule.Location)
	}

	if schedule.ReminderMinutes > 0 {
		preview += fmt.Sprintf("\n🔔 提前 %d 分钟提醒", schedule.ReminderMinutes)
	}

	return preview
}

// FormatConfirmationMessage formats the confirmation message for display.
func FormatConfirmationMessage(schedule *ScheduleRequest, confidence float64) string {
	confidenceLabel := ""
	if confidence >= 0.95 {
		confidenceLabel = " [高置信度]"
	} else if confidence >= 0.9 {
		confidenceLabel = " [置信度良好]"
	}

	return fmt.Sprintf("✓ 已创建: %s (%s)%s",
		schedule.Title,
		schedule.StartTime.Format("2006-01-02 15:04"),
		confidenceLabel,
	)
}
