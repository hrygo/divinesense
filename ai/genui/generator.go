package genui

import "time"

// UI text constants for i18n readiness.
const (
	TextSchedulePreviewPrompt = "已为您解析日程，请确认："
	TextTimeSelectionPrompt   = "请选择具体时间："
)

// UIGenerator generates UI components from agent outputs.
type UIGenerator struct{}

// NewUIGenerator creates a new UIGenerator.
func NewUIGenerator() *UIGenerator {
	return &UIGenerator{}
}

// GenerateFromAgentOutput generates UI components from agent output.
func (g *UIGenerator) GenerateFromAgentOutput(output *AgentOutput) *AgentResponse {
	response := &AgentResponse{}

	switch output.Type {
	case OutputTypeSchedulePreview:
		// Schedule preview → Schedule card
		if output.Schedule != nil {
			card := NewScheduleCard(output.Schedule, CardStatusPreview)
			response.Components = append(response.Components, *card)
			response.Text = TextSchedulePreviewPrompt
		}

	case OutputTypeConfirmation:
		// Confirmation needed → Confirm dialog
		dialog := NewConfirmDialog(
			output.Title,
			output.Message,
			output.Payload,
			output.Danger,
		)
		response.Components = append(response.Components, *dialog)

	case OutputTypeTimeAmbiguous:
		// Time ambiguous → Time picker
		picker := NewTimePicker("请选择具体时间", output.SuggestedTime)
		response.Components = append(response.Components, *picker)
		response.Text = TextTimeSelectionPrompt

	case OutputTypeMultipleOptions:
		// Multiple options → Options list
		list := NewOptionsList(output.Title, output.Options, false)
		response.Components = append(response.Components, *list)

	case OutputTypeSuccess:
		// Success → Success banner
		banner := NewSuccessBanner(output.Message)
		response.Components = append(response.Components, *banner)

	case OutputTypeError:
		// Error → Error alert
		alert := NewErrorAlert(output.Message)
		response.Components = append(response.Components, *alert)

	default:
		// Default: plain text
		response.Text = output.Text
	}

	return response
}

// GenerateSchedulePreview generates a schedule preview card.
func (g *UIGenerator) GenerateSchedulePreview(schedule *ScheduleData) *AgentResponse {
	return g.GenerateFromAgentOutput(&AgentOutput{
		Type:     OutputTypeSchedulePreview,
		Schedule: schedule,
	})
}

// GenerateConfirmation generates a confirmation dialog.
func (g *UIGenerator) GenerateConfirmation(title, message string, payload any, danger bool) *AgentResponse {
	return g.GenerateFromAgentOutput(&AgentOutput{
		Type:    OutputTypeConfirmation,
		Title:   title,
		Message: message,
		Payload: payload,
		Danger:  danger,
	})
}

// GenerateTimeSelection generates a time picker.
func (g *UIGenerator) GenerateTimeSelection(suggestedTime any) *AgentResponse {
	output := &AgentOutput{
		Type: OutputTypeTimeAmbiguous,
	}

	// Handle different time input types
	switch t := suggestedTime.(type) {
	case int64:
		output.SuggestedTime = time.Unix(t, 0)
	case time.Time:
		output.SuggestedTime = t
	default:
		output.SuggestedTime = time.Now()
	}

	return g.GenerateFromAgentOutput(output)
}

// GenerateOptionSelection generates an options list.
func (g *UIGenerator) GenerateOptionSelection(title string, options []OptionItem) *AgentResponse {
	return g.GenerateFromAgentOutput(&AgentOutput{
		Type:    OutputTypeMultipleOptions,
		Title:   title,
		Options: options,
	})
}

// GenerateSuccess generates a success banner.
func (g *UIGenerator) GenerateSuccess(message string) *AgentResponse {
	return g.GenerateFromAgentOutput(&AgentOutput{
		Type:    OutputTypeSuccess,
		Message: message,
	})
}

// GenerateError generates an error alert.
func (g *UIGenerator) GenerateError(message string) *AgentResponse {
	return g.GenerateFromAgentOutput(&AgentOutput{
		Type:    OutputTypeError,
		Message: message,
	})
}

// NewSuccessBanner creates a success banner component.
func NewSuccessBanner(message string) *UIComponent {
	return &UIComponent{
		Type: ComponentSuccessBanner,
		ID:   generateID(),
		Data: map[string]string{
			"message": message,
		},
	}
}

// NewErrorAlert creates an error alert component.
func NewErrorAlert(message string) *UIComponent {
	return &UIComponent{
		Type: ComponentErrorAlert,
		ID:   generateID(),
		Data: map[string]string{
			"message": message,
		},
	}
}

// NewProgressBar creates a progress bar component.
func NewProgressBar(label string, progress int, total int) *UIComponent {
	return &UIComponent{
		Type: ComponentProgressBar,
		ID:   generateID(),
		Data: map[string]any{
			"label":    label,
			"progress": progress,
			"total":    total,
			"percent":  float64(progress) / float64(total) * 100,
		},
	}
}

// NewTextComponent creates a simple text component.
func NewTextComponent(text string) *UIComponent {
	return &UIComponent{
		Type: ComponentText,
		ID:   generateID(),
		Data: map[string]string{
			"text": text,
		},
	}
}

// NewMemoCard creates a memo card component.
func NewMemoCard(id, content string, createdTs int64, tags []string) *UIComponent {
	return &UIComponent{
		Type: ComponentMemoCard,
		ID:   generateID(),
		Data: map[string]any{
			"id":         id,
			"content":    content,
			"created_ts": createdTs,
			"tags":       tags,
		},
		Actions: []UIAction{
			{
				ID:    "view",
				Type:  "link",
				Label: "查看详情",
				Style: "secondary",
			},
		},
	}
}
