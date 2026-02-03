package genui

import (
	"fmt"
	"time"
)

// OptionsListData represents data for an options list component.
type OptionsListData struct {
	Title       string       `json:"title"`
	Description string       `json:"description,omitempty"`
	Options     []OptionItem `json:"options"`
	MultiSelect bool         `json:"multi_select"`
}

// OptionItem represents a single option in an options list.
type OptionItem struct {
	Value       any    `json:"value,omitempty"`
	ID          string `json:"id"`
	Label       string `json:"label"`
	Description string `json:"description,omitempty"`
	Icon        string `json:"icon,omitempty"`
	Selected    bool   `json:"selected"`
}

// NewOptionsList creates a new options list component.
func NewOptionsList(title string, options []OptionItem, multiSelect bool) *UIComponent {
	return &UIComponent{
		Type: ComponentOptionsList,
		ID:   generateID(),
		Data: &OptionsListData{
			Title:       title,
			Options:     options,
			MultiSelect: multiSelect,
		},
		Actions: []UIAction{
			{
				ID:    "submit",
				Type:  "submit",
				Label: "确定",
				Style: "primary",
			},
		},
	}
}

// NewOptionsListWithDescription creates an options list with a description.
func NewOptionsListWithDescription(title, description string, options []OptionItem, multiSelect bool) *UIComponent {
	return &UIComponent{
		Type: ComponentOptionsList,
		ID:   generateID(),
		Data: &OptionsListData{
			Title:       title,
			Description: description,
			Options:     options,
			MultiSelect: multiSelect,
		},
		Actions: []UIAction{
			{
				ID:    "submit",
				Type:  "submit",
				Label: "确定",
				Style: "primary",
			},
		},
	}
}

// NewTimeSlotPicker creates an options list for selecting time slots.
func NewTimeSlotPicker(title string, slots []time.Time) *UIComponent {
	options := make([]OptionItem, len(slots))
	for i, slot := range slots {
		options[i] = OptionItem{
			ID:          fmt.Sprintf("slot_%d", i),
			Label:       slot.Format("15:04"),
			Description: slot.Format("01月02日 (周一)"),
			Value:       slot.Unix(),
		}
	}

	return NewOptionsList(title, options, false)
}

// NewDurationPicker creates an options list for selecting duration.
func NewDurationPicker() *UIComponent {
	options := []OptionItem{
		{ID: "15min", Label: "15 分钟", Value: 15},
		{ID: "30min", Label: "30 分钟", Value: 30},
		{ID: "45min", Label: "45 分钟", Value: 45},
		{ID: "1hour", Label: "1 小时", Value: 60, Selected: true},
		{ID: "1.5hour", Label: "1.5 小时", Value: 90},
		{ID: "2hour", Label: "2 小时", Value: 120},
		{ID: "3hour", Label: "3 小时", Value: 180},
		{ID: "4hour", Label: "4 小时", Value: 240},
	}

	return NewOptionsList("选择时长", options, false)
}

// NewReminderPicker creates an options list for selecting reminder time.
func NewReminderPicker() *UIComponent {
	options := []OptionItem{
		{ID: "none", Label: "不提醒", Value: 0},
		{ID: "5min", Label: "提前 5 分钟", Value: 5},
		{ID: "10min", Label: "提前 10 分钟", Value: 10},
		{ID: "15min", Label: "提前 15 分钟", Value: 15, Selected: true},
		{ID: "30min", Label: "提前 30 分钟", Value: 30},
		{ID: "1hour", Label: "提前 1 小时", Value: 60},
		{ID: "1day", Label: "提前 1 天", Value: 1440},
	}

	return NewOptionsList("设置提醒", options, false)
}

// NewQuickActionPicker creates an options list for quick actions.
func NewQuickActionPicker(actions []string) *UIComponent {
	options := make([]OptionItem, len(actions))
	for i, action := range actions {
		options[i] = OptionItem{
			ID:    fmt.Sprintf("action_%d", i),
			Label: action,
			Value: action,
		}
	}

	return NewOptionsList("请选择操作", options, false)
}
