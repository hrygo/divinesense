package reminder

import (
	"context"
	"testing"
	"time"

	"github.com/hrygo/divinesense/ai/habit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockHabitAnalyzer struct {
	habits *habit.UserHabits
	err    error
}

func (m *mockHabitAnalyzer) GetUserHabits(ctx context.Context, userID int32) (*habit.UserHabits, error) {
	return m.habits, m.err
}

func TestIntegrator_OnScheduleCreated(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()
	notifier := NewMockNotifier()
	svc := NewService(store, notifier)
	integrator := NewIntegrator(svc, nil)

	schedule := &ScheduleInfo{
		ID:        "sched-001",
		Title:     "Team Meeting",
		StartTime: time.Now().Add(2 * time.Hour),
		Location:  "Room 101",
	}

	reminder, err := integrator.OnScheduleCreated(ctx, 1, schedule)
	require.NoError(t, err)
	require.NotNil(t, reminder)
	assert.Equal(t, ReminderTypeSchedule, reminder.Type)
	assert.Equal(t, schedule.ID, reminder.TargetID)
}

func TestIntegrator_OnScheduleCreated_PastTime(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()
	svc := NewService(store, nil)
	integrator := NewIntegrator(svc, nil)

	schedule := &ScheduleInfo{
		ID:        "sched-past",
		Title:     "Past Meeting",
		StartTime: time.Now().Add(5 * time.Minute), // Too close, reminder trigger in past
	}

	reminder, err := integrator.OnScheduleCreated(ctx, 1, schedule)
	assert.NoError(t, err)  // Should not error
	assert.Nil(t, reminder) // But no reminder created
}

func TestIntegrator_OnScheduleUpdated(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()
	svc := NewService(store, nil)
	integrator := NewIntegrator(svc, nil)

	schedule := &ScheduleInfo{
		ID:        "sched-002",
		Title:     "Original Meeting",
		StartTime: time.Now().Add(2 * time.Hour),
	}

	// Create initial reminder
	original, _ := integrator.OnScheduleCreated(ctx, 1, schedule)
	require.NotNil(t, original)

	// Update schedule time
	schedule.StartTime = time.Now().Add(4 * time.Hour)
	schedule.Title = "Updated Meeting"

	updated, err := integrator.OnScheduleUpdated(ctx, 1, schedule)
	require.NoError(t, err)
	require.NotNil(t, updated)

	// Original should be cancelled
	oldReminder, _ := svc.GetReminder(ctx, original.ID)
	assert.Equal(t, StatusCancelled, oldReminder.Status)

	// New reminder should be pending
	assert.Equal(t, StatusPending, updated.Status)
	assert.Contains(t, updated.Message, "Updated Meeting")
}

func TestIntegrator_OnScheduleDeleted(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()
	svc := NewService(store, nil)
	integrator := NewIntegrator(svc, nil)

	schedule := &ScheduleInfo{
		ID:        "sched-delete",
		Title:     "To Delete",
		StartTime: time.Now().Add(2 * time.Hour),
	}

	reminder, _ := integrator.OnScheduleCreated(ctx, 1, schedule)
	require.NotNil(t, reminder)

	err := integrator.OnScheduleDeleted(ctx, schedule.ID)
	require.NoError(t, err)

	r, _ := svc.GetReminder(ctx, reminder.ID)
	assert.Equal(t, StatusCancelled, r.Status)
}

func TestIntegrator_OnTodoCreated(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()
	svc := NewService(store, nil)
	integrator := NewIntegrator(svc, nil)

	dueTime := time.Now().Add(2 * time.Hour)
	todo := &TodoInfo{
		ID:       "todo-001",
		Title:    "Complete Report",
		DueTime:  &dueTime,
		Priority: "high",
	}

	reminder, err := integrator.OnTodoCreated(ctx, 1, todo)
	require.NoError(t, err)
	require.NotNil(t, reminder)
	assert.Equal(t, ReminderTypeTodo, reminder.Type)
	assert.Contains(t, reminder.Message, "高优先级")
	assert.Contains(t, reminder.Message, "Complete Report")
}

func TestIntegrator_OnTodoCreated_NoDueTime(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()
	svc := NewService(store, nil)
	integrator := NewIntegrator(svc, nil)

	todo := &TodoInfo{
		ID:      "todo-no-due",
		Title:   "No Due Time",
		DueTime: nil,
	}

	reminder, err := integrator.OnTodoCreated(ctx, 1, todo)
	assert.NoError(t, err)
	assert.Nil(t, reminder) // No reminder for todo without due time
}

func TestIntegrator_OnTodoCompleted(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()
	svc := NewService(store, nil)
	integrator := NewIntegrator(svc, nil)

	dueTime := time.Now().Add(2 * time.Hour)
	todo := &TodoInfo{
		ID:       "todo-complete",
		Title:    "Complete Me",
		DueTime:  &dueTime,
		Priority: "medium",
	}

	reminder, _ := integrator.OnTodoCreated(ctx, 1, todo)
	require.NotNil(t, reminder)

	err := integrator.OnTodoCompleted(ctx, todo.ID)
	require.NoError(t, err)

	r, _ := svc.GetReminder(ctx, reminder.ID)
	assert.Equal(t, StatusCancelled, r.Status)
}

func TestIntegrator_CalculateScheduleLeadTime(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()
	svc := NewService(store, nil)

	// Without habit analyzer - always returns default
	t.Run("no_habit_analyzer", func(t *testing.T) {
		integrator := NewIntegrator(svc, nil)
		schedule := &ScheduleInfo{
			ID:        "test",
			Title:     "Test",
			StartTime: time.Now().Add(time.Hour),
		}
		lead := integrator.calculateScheduleLeadTime(ctx, 1, schedule)
		assert.Equal(t, 15, lead) // default
	})

	// With habit analyzer - time-based logic kicks in
	mockAnalyzer := &mockHabitAnalyzer{
		habits: &habit.UserHabits{
			UserID: 1,
			Time:   &habit.TimeHabits{},
		},
	}

	t.Run("morning_meeting", func(t *testing.T) {
		integrator := NewIntegrator(svc, mockAnalyzer)
		// Create a time explicitly at 9 AM today
		now := time.Now()
		morning := time.Date(now.Year(), now.Month(), now.Day(), 9, 0, 0, 0, now.Location())
		schedule := &ScheduleInfo{ID: "m", Title: "Morning", StartTime: morning}
		lead := integrator.calculateScheduleLeadTime(ctx, 1, schedule)
		assert.Equal(t, 30, lead)
	})

	t.Run("lunch_meeting", func(t *testing.T) {
		integrator := NewIntegrator(svc, mockAnalyzer)
		now := time.Now()
		lunch := time.Date(now.Year(), now.Month(), now.Day(), 12, 0, 0, 0, now.Location())
		schedule := &ScheduleInfo{ID: "l", Title: "Lunch", StartTime: lunch}
		lead := integrator.calculateScheduleLeadTime(ctx, 1, schedule)
		assert.Equal(t, 10, lead)
	})

	t.Run("evening_event", func(t *testing.T) {
		integrator := NewIntegrator(svc, mockAnalyzer)
		now := time.Now()
		evening := time.Date(now.Year(), now.Month(), now.Day(), 19, 0, 0, 0, now.Location())
		schedule := &ScheduleInfo{ID: "e", Title: "Evening", StartTime: evening}
		lead := integrator.calculateScheduleLeadTime(ctx, 1, schedule)
		assert.Equal(t, 30, lead)
	})

	t.Run("with_location", func(t *testing.T) {
		integrator := NewIntegrator(svc, mockAnalyzer)
		now := time.Now()
		afternoon := time.Date(now.Year(), now.Month(), now.Day(), 15, 0, 0, 0, now.Location())
		schedule := &ScheduleInfo{ID: "loc", Title: "Offsite", StartTime: afternoon, Location: "Building B"}
		lead := integrator.calculateScheduleLeadTime(ctx, 1, schedule)
		assert.Equal(t, 20, lead)
	})

	t.Run("default_afternoon", func(t *testing.T) {
		integrator := NewIntegrator(svc, mockAnalyzer)
		now := time.Now()
		afternoon := time.Date(now.Year(), now.Month(), now.Day(), 15, 0, 0, 0, now.Location())
		schedule := &ScheduleInfo{ID: "def", Title: "Default", StartTime: afternoon}
		lead := integrator.calculateScheduleLeadTime(ctx, 1, schedule)
		assert.Equal(t, 15, lead)
	})
}

func TestIntegrator_CalculateTodoLeadTime(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()
	svc := NewService(store, nil)
	integrator := NewIntegrator(svc, nil)

	tests := []struct {
		priority string
		expected int
	}{
		{"high", 60},
		{"medium", 30},
		{"low", 15},
		{"", 30},
	}

	for _, tt := range tests {
		dueTime := time.Now().Add(time.Hour)
		todo := &TodoInfo{
			ID:       "test",
			Priority: tt.priority,
			DueTime:  &dueTime,
		}

		lead := integrator.calculateTodoLeadTime(ctx, 1, todo)
		assert.Equal(t, tt.expected, lead, "lead time for priority %s", tt.priority)
	}
}

func TestIntegrator_CreateSmartReminder(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()
	svc := NewService(store, nil)
	integrator := NewIntegrator(svc, nil)

	suggestion := SmartSuggestion{
		RelatedEntityID: "entity-001",
		SuggestedTime:   time.Now().Add(time.Hour),
		Message:         "AI suggests you prepare for tomorrow's meeting",
		Confidence:      0.85,
		Reason:          "Based on your past behavior",
	}

	reminder, err := integrator.CreateSmartReminder(ctx, 1, suggestion)
	require.NoError(t, err)
	require.NotNil(t, reminder)
	assert.Equal(t, ReminderTypeSmart, reminder.Type)
	assert.Equal(t, suggestion.Message, reminder.Message)
}

func TestIntegrator_BatchCreateForSchedules(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()
	svc := NewService(store, nil)
	integrator := NewIntegrator(svc, nil)

	schedules := []*ScheduleInfo{
		{ID: "batch-1", Title: "Meeting 1", StartTime: time.Now().Add(2 * time.Hour)},
		{ID: "batch-2", Title: "Meeting 2", StartTime: time.Now().Add(3 * time.Hour)},
		{ID: "batch-3", Title: "Meeting 3", StartTime: time.Now().Add(4 * time.Hour)},
	}

	reminders, err := integrator.BatchCreateForSchedules(ctx, 1, schedules)
	require.NoError(t, err)
	assert.Len(t, reminders, 3)
}

func TestIntegrator_GetUpcomingReminders(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()
	svc := NewService(store, nil)
	integrator := NewIntegrator(svc, nil)

	// Create some reminders
	for i := 0; i < 3; i++ {
		schedule := &ScheduleInfo{
			ID:        "upcoming-" + string(rune('a'+i)),
			Title:     "Upcoming Meeting",
			StartTime: time.Now().Add(time.Duration(i+1) * time.Hour),
		}
		_, _ = integrator.OnScheduleCreated(ctx, 1, schedule)
	}

	upcoming, err := integrator.GetUpcomingReminders(ctx, 1)
	require.NoError(t, err)
	assert.Len(t, upcoming, 3)
}

func TestIntegrator_SyncScheduleReminders(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()
	svc := NewService(store, nil)
	integrator := NewIntegrator(svc, nil)

	// Initial schedules
	initialSchedules := []*ScheduleInfo{
		{ID: "sync-1", Title: "Keep", StartTime: time.Now().Add(2 * time.Hour)},
		{ID: "sync-2", Title: "Remove", StartTime: time.Now().Add(3 * time.Hour)},
	}

	_, _ = integrator.BatchCreateForSchedules(ctx, 1, initialSchedules)

	// Updated schedules (removed sync-2)
	updatedSchedules := []*ScheduleInfo{
		{ID: "sync-1", Title: "Keep Updated", StartTime: time.Now().Add(2 * time.Hour)},
		{ID: "sync-3", Title: "New", StartTime: time.Now().Add(4 * time.Hour)},
	}

	err := integrator.SyncScheduleReminders(ctx, 1, updatedSchedules)
	require.NoError(t, err)

	// Check results
	all, _ := integrator.GetUpcomingReminders(ctx, 1)
	assert.Len(t, all, 2) // sync-1 and sync-3
}

func TestIntegrator_WithHabitAnalyzer(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()
	svc := NewService(store, nil)

	mockAnalyzer := &mockHabitAnalyzer{
		habits: &habit.UserHabits{
			UserID: 1,
			Time: &habit.TimeHabits{
				ActiveHours: []int{9, 10, 14, 15},
			},
		},
	}

	integrator := NewIntegrator(svc, mockAnalyzer)

	schedule := &ScheduleInfo{
		ID:        "habit-test",
		Title:     "Habit Test",
		StartTime: time.Now().Add(2 * time.Hour),
	}

	reminder, err := integrator.OnScheduleCreated(ctx, 1, schedule)
	require.NoError(t, err)
	require.NotNil(t, reminder)
}

func BenchmarkIntegrator_OnScheduleCreated(b *testing.B) {
	ctx := context.Background()
	store := NewMemoryStore()
	svc := NewService(store, nil)
	integrator := NewIntegrator(svc, nil)

	schedule := &ScheduleInfo{
		ID:        "bench",
		Title:     "Benchmark",
		StartTime: time.Now().Add(time.Hour),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		schedule.ID = "bench-" + string(rune(i%1000))
		_, _ = integrator.OnScheduleCreated(ctx, 1, schedule)
	}
}
