// Package genui provides Generative UI components for AI agents.
package genui

import (
	"time"

	"github.com/google/uuid"
)

// ComponentType defines the type of UI component.
type ComponentType string

const (
	ComponentText          ComponentType = "text"
	ComponentScheduleCard  ComponentType = "schedule_card"
	ComponentMemoCard      ComponentType = "memo_card"
	ComponentConfirmDialog ComponentType = "confirm_dialog"
	ComponentOptionsList   ComponentType = "options_list"
	ComponentTimePicker    ComponentType = "time_picker"
	ComponentProgressBar   ComponentType = "progress_bar"
	ComponentErrorAlert    ComponentType = "error_alert"
	ComponentSuccessBanner ComponentType = "success_banner"
)

// UIComponent represents a generic UI component.
type UIComponent struct {
	Type    ComponentType `json:"type"`
	ID      string        `json:"id"`
	Data    any           `json:"data"`
	Actions []UIAction    `json:"actions,omitempty"`
}

// UIAction represents an action button on a component.
type UIAction struct {
	Payload any    `json:"payload,omitempty"`
	ID      string `json:"id"`
	Type    string `json:"type"`
	Label   string `json:"label"`
	Style   string `json:"style"`
}

// AgentResponse represents an enhanced agent response with UI components.
type AgentResponse struct {
	Text       string        `json:"text,omitempty"`
	Components []UIComponent `json:"components,omitempty"`
	Streaming  bool          `json:"streaming,omitempty"`
}

// OutputType defines the type of agent output.
type OutputType string

const (
	OutputTypeText            OutputType = "text"
	OutputTypeSchedulePreview OutputType = "schedule_preview"
	OutputTypeConfirmation    OutputType = "confirmation"
	OutputTypeTimeAmbiguous   OutputType = "time_ambiguous"
	OutputTypeMultipleOptions OutputType = "multiple_options"
	OutputTypeSuccess         OutputType = "success"
	OutputTypeError           OutputType = "error"
)

// AgentOutput represents the raw output from an agent.
type AgentOutput struct {
	SuggestedTime time.Time
	Payload       any
	Schedule      *ScheduleData
	Type          OutputType
	Text          string
	Title         string
	Message       string
	Options       []OptionItem
	Danger        bool
}

// ScheduleData represents schedule information for card display.
type ScheduleData struct {
	StartTime   time.Time `json:"start_time"`
	EndTime     time.Time `json:"end_time"`
	ID          string    `json:"id,omitempty"`
	Title       string    `json:"title"`
	Location    string    `json:"location,omitempty"`
	Description string    `json:"description,omitempty"`
	Duration    int       `json:"duration"`
}

// generateID creates a unique component ID.
func generateID() string {
	return uuid.New().String()[:8]
}

// ternary is a helper function for conditional string selection.
func ternary(condition bool, trueVal, falseVal string) string {
	if condition {
		return trueVal
	}
	return falseVal
}
