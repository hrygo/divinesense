// Package reminder provides reminder management for schedules and todos.
package reminder

import (
	"context"
	"fmt"
	"time"

	"github.com/hrygo/divinesense/ai/habit"
)

// TodoInfo contains todo information for reminder creation.
type TodoInfo struct {
	ID       string
	Title    string
	DueTime  *time.Time
	Priority string // high, medium, low
}

// HabitAnalyzer provides user habit analysis.
type HabitAnalyzer interface {
	GetUserHabits(ctx context.Context, userID int32) (*habit.UserHabits, error)
}

// Integrator provides integration between reminders and other entities.
type Integrator struct {
	service       *Service
	habitAnalyzer HabitAnalyzer
}

// NewIntegrator creates a new reminder integrator.
func NewIntegrator(service *Service, habitAnalyzer HabitAnalyzer) *Integrator {
	return &Integrator{
		service:       service,
		habitAnalyzer: habitAnalyzer,
	}
}

// OnScheduleCreated handles schedule creation event.
func (i *Integrator) OnScheduleCreated(ctx context.Context, userID int32, schedule *ScheduleInfo) (*Reminder, error) {
	leadMinutes := i.calculateScheduleLeadTime(ctx, userID, schedule)

	reminder, err := i.service.CreateForSchedule(ctx, userID, schedule, leadMinutes)
	if err != nil {
		// If reminder creation fails (e.g., trigger in past), don't block schedule creation
		return nil, nil //nolint:nilerr // Intentional - non-blocking reminder
	}

	return reminder, nil
}

// OnScheduleUpdated handles schedule update event.
func (i *Integrator) OnScheduleUpdated(ctx context.Context, userID int32, schedule *ScheduleInfo) (*Reminder, error) {
	// Cancel existing reminders for this schedule
	if err := i.service.CancelByTarget(ctx, schedule.ID); err != nil {
		return nil, fmt.Errorf("failed to cancel existing reminders: %w", err)
	}

	// Create new reminder with updated time
	return i.OnScheduleCreated(ctx, userID, schedule)
}

// OnScheduleDeleted handles schedule deletion event.
func (i *Integrator) OnScheduleDeleted(ctx context.Context, scheduleID string) error {
	return i.service.CancelByTarget(ctx, scheduleID)
}

// OnTodoCreated handles todo creation event.
func (i *Integrator) OnTodoCreated(ctx context.Context, userID int32, todo *TodoInfo) (*Reminder, error) {
	if todo.DueTime == nil {
		return nil, nil // No due time, no reminder needed
	}

	leadMinutes := i.calculateTodoLeadTime(ctx, userID, todo)

	req := &CreateReminderRequest{
		UserID:    userID,
		Type:      ReminderTypeTodo,
		TargetID:  todo.ID,
		TriggerAt: todo.DueTime.Add(-time.Duration(leadMinutes) * time.Minute),
		Message:   i.buildTodoMessage(todo, leadMinutes),
	}

	reminder, err := i.service.CreateCustom(ctx, req)
	if err != nil {
		return nil, nil //nolint:nilerr // Don't block todo creation
	}

	return reminder, nil
}

// OnTodoUpdated handles todo update event.
func (i *Integrator) OnTodoUpdated(ctx context.Context, userID int32, todo *TodoInfo) (*Reminder, error) {
	if err := i.service.CancelByTarget(ctx, todo.ID); err != nil {
		return nil, fmt.Errorf("failed to cancel existing reminders: %w", err)
	}

	return i.OnTodoCreated(ctx, userID, todo)
}

// OnTodoCompleted handles todo completion event.
func (i *Integrator) OnTodoCompleted(ctx context.Context, todoID string) error {
	return i.service.CancelByTarget(ctx, todoID)
}

// OnTodoDeleted handles todo deletion event.
func (i *Integrator) OnTodoDeleted(ctx context.Context, todoID string) error {
	return i.service.CancelByTarget(ctx, todoID)
}

// calculateScheduleLeadTime determines optimal lead time based on user habits.
func (i *Integrator) calculateScheduleLeadTime(ctx context.Context, userID int32, schedule *ScheduleInfo) int {
	const defaultLeadMinutes = 15

	if i.habitAnalyzer == nil {
		return defaultLeadMinutes
	}

	habits, err := i.habitAnalyzer.GetUserHabits(ctx, userID)
	if err != nil || habits == nil {
		return defaultLeadMinutes
	}

	// Adjust based on schedule time
	hour := schedule.StartTime.Hour()

	// Morning meetings: longer lead time (people may need to prepare)
	if hour >= 8 && hour < 10 {
		return 30
	}

	// Lunch meetings: shorter lead time
	if hour >= 11 && hour < 14 {
		return 10
	}

	// Evening events: longer lead time
	if hour >= 18 {
		return 30
	}

	// Location-based adjustment
	if schedule.Location != "" {
		// If there's a location, add extra time for travel
		return 20
	}

	return defaultLeadMinutes
}

// calculateTodoLeadTime determines optimal lead time for todos.
func (i *Integrator) calculateTodoLeadTime(ctx context.Context, userID int32, todo *TodoInfo) int {
	// Priority-based lead times
	switch todo.Priority {
	case "high":
		return 60 // 1 hour for high priority
	case "medium":
		return 30 // 30 minutes for medium priority
	case "low":
		return 15 // 15 minutes for low priority
	default:
		return 30
	}
}

// buildTodoMessage creates the reminder message for a todo.
func (i *Integrator) buildTodoMessage(todo *TodoInfo, leadMinutes int) string {
	base := fmt.Sprintf("您有一个待办「%s」将在 %d 分钟后到期", todo.Title, leadMinutes)

	switch todo.Priority {
	case "high":
		return "⚠️ 高优先级: " + base
	case "medium":
		return base
	default:
		return base
	}
}

// CreateSmartReminder creates an AI-suggested reminder based on context.
func (i *Integrator) CreateSmartReminder(ctx context.Context, userID int32, suggestion SmartSuggestion) (*Reminder, error) {
	req := &CreateReminderRequest{
		UserID:    userID,
		Type:      ReminderTypeSmart,
		TargetID:  suggestion.RelatedEntityID,
		TriggerAt: suggestion.SuggestedTime,
		Message:   suggestion.Message,
	}

	return i.service.CreateCustom(ctx, req)
}

// SmartSuggestion represents an AI-generated reminder suggestion.
type SmartSuggestion struct {
	SuggestedTime   time.Time
	RelatedEntityID string
	Message         string
	Reason          string
	Confidence      float64
}

// BatchCreateForSchedules creates reminders for multiple schedules.
func (i *Integrator) BatchCreateForSchedules(ctx context.Context, userID int32, schedules []*ScheduleInfo) ([]*Reminder, error) {
	var reminders []*Reminder

	for _, schedule := range schedules {
		reminder, err := i.OnScheduleCreated(ctx, userID, schedule)
		if err != nil {
			continue // Skip failed ones
		}
		if reminder != nil {
			reminders = append(reminders, reminder)
		}
	}

	return reminders, nil
}

// GetUpcomingReminders gets all upcoming reminders for a user.
func (i *Integrator) GetUpcomingReminders(ctx context.Context, userID int32) ([]*Reminder, error) {
	return i.service.GetUserReminders(ctx, userID, StatusPending)
}

// SyncScheduleReminders syncs reminders with schedule changes.
func (i *Integrator) SyncScheduleReminders(ctx context.Context, userID int32, schedules []*ScheduleInfo) error {
	// Get existing reminders
	existing, err := i.service.GetUserReminders(ctx, userID, StatusPending)
	if err != nil {
		return fmt.Errorf("failed to get existing reminders: %w", err)
	}

	// Build schedule ID set
	scheduleIDs := make(map[string]bool)
	for _, s := range schedules {
		scheduleIDs[s.ID] = true
	}

	// Cancel reminders for deleted schedules
	for _, r := range existing {
		if r.Type == ReminderTypeSchedule && !scheduleIDs[r.TargetID] {
			_ = i.service.Cancel(ctx, r.ID) //nolint:errcheck // cleanup on sync
		}
	}

	// Create/update reminders for current schedules
	for _, schedule := range schedules {
		_, _ = i.OnScheduleUpdated(ctx, userID, schedule) //nolint:errcheck // sync operation, non-critical
	}

	return nil
}
