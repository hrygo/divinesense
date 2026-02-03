package genui

import "time"

// TimePickerData represents data for a time picker component.
type TimePickerData struct {
	Label       string `json:"label"`
	Format      string `json:"format,omitempty"`
	DefaultDate int64  `json:"default_date,omitempty"`
	MinDate     int64  `json:"min_date,omitempty"`
	MaxDate     int64  `json:"max_date,omitempty"`
	ShowTime    bool   `json:"show_time"`
}

// NewTimePicker creates a new time picker component.
func NewTimePicker(label string, defaultDate time.Time) *UIComponent {
	now := time.Now()

	return &UIComponent{
		Type: ComponentTimePicker,
		ID:   generateID(),
		Data: &TimePickerData{
			Label:       label,
			DefaultDate: defaultDate.Unix(),
			MinDate:     now.Unix(),
			MaxDate:     now.AddDate(1, 0, 0).Unix(),
			ShowTime:    true,
			Format:      "yyyy-MM-dd HH:mm",
		},
		Actions: []UIAction{
			{
				ID:    "submit",
				Type:  "submit",
				Label: "确定",
				Style: "primary",
			},
			{
				ID:    "cancel",
				Type:  "button",
				Label: "取消",
				Style: "secondary",
			},
		},
	}
}

// NewDatePicker creates a date-only picker (without time selection).
func NewDatePicker(label string, defaultDate time.Time) *UIComponent {
	now := time.Now()

	return &UIComponent{
		Type: ComponentTimePicker,
		ID:   generateID(),
		Data: &TimePickerData{
			Label:       label,
			DefaultDate: defaultDate.Unix(),
			MinDate:     now.Unix(),
			MaxDate:     now.AddDate(1, 0, 0).Unix(),
			ShowTime:    false,
			Format:      "yyyy-MM-dd",
		},
		Actions: []UIAction{
			{
				ID:    "submit",
				Type:  "submit",
				Label: "确定",
				Style: "primary",
			},
			{
				ID:    "cancel",
				Type:  "button",
				Label: "取消",
				Style: "secondary",
			},
		},
	}
}

// NewTimePickerWithRange creates a time picker with custom date range.
func NewTimePickerWithRange(label string, defaultDate, minDate, maxDate time.Time, showTime bool) *UIComponent {
	return &UIComponent{
		Type: ComponentTimePicker,
		ID:   generateID(),
		Data: &TimePickerData{
			Label:       label,
			DefaultDate: defaultDate.Unix(),
			MinDate:     minDate.Unix(),
			MaxDate:     maxDate.Unix(),
			ShowTime:    showTime,
			Format:      ternary(showTime, "yyyy-MM-dd HH:mm", "yyyy-MM-dd"),
		},
		Actions: []UIAction{
			{
				ID:    "submit",
				Type:  "submit",
				Label: "确定",
				Style: "primary",
			},
			{
				ID:    "cancel",
				Type:  "button",
				Label: "取消",
				Style: "secondary",
			},
		},
	}
}

// NewEndTimePicker creates a time picker for selecting end time.
func NewEndTimePicker(startTime time.Time) *UIComponent {
	// Default end time is 1 hour after start
	defaultEnd := startTime.Add(time.Hour)

	return &UIComponent{
		Type: ComponentTimePicker,
		ID:   generateID(),
		Data: &TimePickerData{
			Label:       "选择结束时间",
			DefaultDate: defaultEnd.Unix(),
			MinDate:     startTime.Unix(),
			MaxDate:     startTime.AddDate(0, 0, 7).Unix(), // Max 7 days after start
			ShowTime:    true,
			Format:      "yyyy-MM-dd HH:mm",
		},
		Actions: []UIAction{
			{
				ID:    "submit",
				Type:  "submit",
				Label: "确定",
				Style: "primary",
			},
			{
				ID:    "cancel",
				Type:  "button",
				Label: "取消",
				Style: "secondary",
			},
		},
	}
}
