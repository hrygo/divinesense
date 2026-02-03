package genui

// ScheduleCardData represents data for a schedule card component.
type ScheduleCardData struct {
	ID          string `json:"id,omitempty"`
	Title       string `json:"title"`
	Location    string `json:"location,omitempty"`
	Description string `json:"description,omitempty"`
	Status      string `json:"status"`
	StartTime   int64  `json:"start_time"`
	EndTime     int64  `json:"end_time"`
	Duration    int    `json:"duration"`
}

// CardStatus defines the status of a schedule card.
const (
	CardStatusPreview   = "preview"
	CardStatusConfirmed = "confirmed"
	CardStatusConflict  = "conflict"
)

// NewScheduleCard creates a new schedule card component.
func NewScheduleCard(schedule *ScheduleData, status string) *UIComponent {
	cardData := &ScheduleCardData{
		ID:          schedule.ID,
		Title:       schedule.Title,
		StartTime:   schedule.StartTime.Unix(),
		EndTime:     schedule.EndTime.Unix(),
		Duration:    schedule.Duration,
		Location:    schedule.Location,
		Description: schedule.Description,
		Status:      status,
	}

	actions := []UIAction{}

	if status == CardStatusPreview {
		actions = append(actions,
			UIAction{
				ID:    "confirm",
				Type:  "button",
				Label: "确认创建",
				Style: "primary",
				Payload: map[string]any{
					"action":   "create_schedule",
					"schedule": schedule,
				},
			},
			UIAction{
				ID:    "edit",
				Type:  "button",
				Label: "修改",
				Style: "secondary",
			},
			UIAction{
				ID:    "cancel",
				Type:  "button",
				Label: "取消",
				Style: "ghost",
			},
		)
	} else if status == CardStatusConflict {
		actions = append(actions,
			UIAction{
				ID:    "reschedule",
				Type:  "button",
				Label: "重新安排",
				Style: "primary",
			},
			UIAction{
				ID:    "force_create",
				Type:  "button",
				Label: "仍然创建",
				Style: "secondary",
				Payload: map[string]any{
					"action":   "force_create",
					"schedule": schedule,
				},
			},
			UIAction{
				ID:    "cancel",
				Type:  "button",
				Label: "取消",
				Style: "ghost",
			},
		)
	}

	return &UIComponent{
		Type:    ComponentScheduleCard,
		ID:      generateID(),
		Data:    cardData,
		Actions: actions,
	}
}

// NewScheduleCardFromRaw creates a schedule card from raw data.
func NewScheduleCardFromRaw(id, title string, startTs, endTs int64, duration int, location, status string) *UIComponent {
	cardData := &ScheduleCardData{
		ID:        id,
		Title:     title,
		StartTime: startTs,
		EndTime:   endTs,
		Duration:  duration,
		Location:  location,
		Status:    status,
	}

	actions := []UIAction{}

	if status == CardStatusPreview {
		actions = append(actions,
			UIAction{
				ID:    "confirm",
				Type:  "button",
				Label: "确认创建",
				Style: "primary",
			},
			UIAction{
				ID:    "edit",
				Type:  "button",
				Label: "修改",
				Style: "secondary",
			},
			UIAction{
				ID:    "cancel",
				Type:  "button",
				Label: "取消",
				Style: "ghost",
			},
		)
	}

	return &UIComponent{
		Type:    ComponentScheduleCard,
		ID:      generateID(),
		Data:    cardData,
		Actions: actions,
	}
}
