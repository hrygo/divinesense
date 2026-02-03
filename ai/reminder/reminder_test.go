package reminder

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemoryStore_CRUD(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()

	// Create
	reminder := &Reminder{
		ID:        "test-001",
		UserID:    1,
		Type:      ReminderTypeSchedule,
		TargetID:  "schedule-001",
		TriggerAt: time.Now().Add(time.Hour),
		Message:   "Test reminder",
		Channels:  []Channel{ChannelApp},
		Status:    StatusPending,
		CreatedAt: time.Now(),
	}

	err := store.Create(ctx, reminder)
	require.NoError(t, err)

	// Get
	got, err := store.Get(ctx, "test-001")
	require.NoError(t, err)
	assert.Equal(t, reminder.ID, got.ID)
	assert.Equal(t, reminder.Message, got.Message)

	// Update
	reminder.Message = "Updated message"
	err = store.Update(ctx, reminder)
	require.NoError(t, err)

	got, _ = store.Get(ctx, "test-001")
	assert.Equal(t, "Updated message", got.Message)

	// Delete
	err = store.Delete(ctx, "test-001")
	require.NoError(t, err)

	_, err = store.Get(ctx, "test-001")
	assert.Error(t, err)
}

func TestMemoryStore_GetByTarget(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()

	// Create multiple reminders for same target
	for i := 0; i < 3; i++ {
		_ = store.Create(ctx, &Reminder{
			ID:        "rem-" + string(rune('a'+i)),
			UserID:    1,
			TargetID:  "schedule-001",
			Status:    StatusPending,
			TriggerAt: time.Now().Add(time.Hour),
		})
	}
	_ = store.Create(ctx, &Reminder{
		ID:        "rem-other",
		UserID:    1,
		TargetID:  "schedule-002",
		Status:    StatusPending,
		TriggerAt: time.Now().Add(time.Hour),
	})

	reminders, err := store.GetByTarget(ctx, "schedule-001")
	require.NoError(t, err)
	assert.Len(t, reminders, 3)
}

func TestMemoryStore_GetDueReminders(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()

	now := time.Now()

	// Past due (should be returned)
	_ = store.Create(ctx, &Reminder{
		ID:        "due-1",
		TriggerAt: now.Add(-time.Hour),
		Status:    StatusPending,
	})

	// Due now (should be returned)
	_ = store.Create(ctx, &Reminder{
		ID:        "due-2",
		TriggerAt: now,
		Status:    StatusPending,
	})

	// Future (should NOT be returned)
	_ = store.Create(ctx, &Reminder{
		ID:        "future",
		TriggerAt: now.Add(time.Hour),
		Status:    StatusPending,
	})

	// Already sent (should NOT be returned)
	_ = store.Create(ctx, &Reminder{
		ID:        "sent",
		TriggerAt: now.Add(-time.Hour),
		Status:    StatusSent,
	})

	due, err := store.GetDueReminders(ctx, now)
	require.NoError(t, err)
	assert.Len(t, due, 2)
}

func TestMemoryStore_GetByUser(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()

	_ = store.Create(ctx, &Reminder{ID: "r1", UserID: 1, Status: StatusPending, TriggerAt: time.Now()})
	_ = store.Create(ctx, &Reminder{ID: "r2", UserID: 1, Status: StatusSent, TriggerAt: time.Now()})
	_ = store.Create(ctx, &Reminder{ID: "r3", UserID: 2, Status: StatusPending, TriggerAt: time.Now()})

	// All for user 1
	reminders, err := store.GetByUser(ctx, 1, "")
	require.NoError(t, err)
	assert.Len(t, reminders, 2)

	// Only pending for user 1
	reminders, err = store.GetByUser(ctx, 1, StatusPending)
	require.NoError(t, err)
	assert.Len(t, reminders, 1)
}

func TestMemoryStore_MarkSent(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()

	_ = store.Create(ctx, &Reminder{
		ID:     "test",
		Status: StatusPending,
	})

	err := store.MarkSent(ctx, "test")
	require.NoError(t, err)

	r, _ := store.Get(ctx, "test")
	assert.Equal(t, StatusSent, r.Status)
	assert.NotNil(t, r.SentAt)
}

func TestMemoryStore_MarkFailed(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()

	_ = store.Create(ctx, &Reminder{
		ID:     "test",
		Status: StatusPending,
	})

	err := store.MarkFailed(ctx, "test", "network error")
	require.NoError(t, err)

	r, _ := store.Get(ctx, "test")
	assert.Equal(t, StatusFailed, r.Status)
	assert.Equal(t, "network error", r.Metadata["failure_reason"])
}

func TestService_CreateForSchedule(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()
	notifier := NewMockNotifier()
	svc := NewService(store, notifier)

	schedule := &ScheduleInfo{
		ID:        "sched-001",
		Title:     "Team Meeting",
		StartTime: time.Now().Add(time.Hour),
		Location:  "Room 101",
	}

	reminder, err := svc.CreateForSchedule(ctx, 1, schedule, 30)
	require.NoError(t, err)
	assert.NotEmpty(t, reminder.ID)
	assert.Equal(t, ReminderTypeSchedule, reminder.Type)
	assert.Contains(t, reminder.Message, "Team Meeting")
	assert.Contains(t, reminder.Message, "30 分钟")
	assert.Contains(t, reminder.Message, "Room 101")
}

func TestService_CreateForSchedule_DefaultLead(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()
	svc := NewService(store, nil)

	schedule := &ScheduleInfo{
		ID:        "sched-002",
		Title:     "Standup",
		StartTime: time.Now().Add(time.Hour),
	}

	reminder, err := svc.CreateForSchedule(ctx, 1, schedule, 0)
	require.NoError(t, err)
	assert.Contains(t, reminder.Message, "15 分钟") // default lead time
}

func TestService_CreateForSchedule_PastTrigger(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()
	svc := NewService(store, nil)

	schedule := &ScheduleInfo{
		ID:        "sched-003",
		Title:     "Past Event",
		StartTime: time.Now().Add(5 * time.Minute),
	}

	_, err := svc.CreateForSchedule(ctx, 1, schedule, 30) // 30 min lead = past
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "past")
}

func TestService_CreateCustom(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()
	svc := NewService(store, nil)

	req := &CreateReminderRequest{
		UserID:    1,
		Type:      ReminderTypeTodo,
		TargetID:  "todo-001",
		TriggerAt: time.Now().Add(time.Hour),
		Message:   "Don't forget to submit report",
		Channels:  []Channel{ChannelApp, ChannelEmail},
	}

	reminder, err := svc.CreateCustom(ctx, req)
	require.NoError(t, err)
	assert.Equal(t, ReminderTypeTodo, reminder.Type)
	assert.Len(t, reminder.Channels, 2)
}

func TestService_CreateCustom_PastTrigger(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()
	svc := NewService(store, nil)

	req := &CreateReminderRequest{
		UserID:    1,
		TriggerAt: time.Now().Add(-time.Hour),
		Message:   "Past reminder",
	}

	_, err := svc.CreateCustom(ctx, req)
	assert.Error(t, err)
}

func TestService_Cancel(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()
	svc := NewService(store, nil)

	reminder, _ := svc.CreateCustom(ctx, &CreateReminderRequest{
		UserID:    1,
		TriggerAt: time.Now().Add(time.Hour),
		Message:   "Cancel me",
	})

	err := svc.Cancel(ctx, reminder.ID)
	require.NoError(t, err)

	got, _ := svc.GetReminder(ctx, reminder.ID)
	assert.Equal(t, StatusCancelled, got.Status)
}

func TestService_Cancel_NotPending(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()
	svc := NewService(store, nil)

	_ = store.Create(ctx, &Reminder{
		ID:     "sent-reminder",
		Status: StatusSent,
	})

	err := svc.Cancel(ctx, "sent-reminder")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot cancel")
}

func TestService_CancelByTarget(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()
	svc := NewService(store, nil)

	// Create multiple reminders for same target
	for i := 0; i < 3; i++ {
		_ = store.Create(ctx, &Reminder{
			ID:        "rem-" + string(rune('a'+i)),
			TargetID:  "schedule-001",
			Status:    StatusPending,
			TriggerAt: time.Now().Add(time.Hour),
		})
	}

	err := svc.CancelByTarget(ctx, "schedule-001")
	require.NoError(t, err)

	reminders, _ := store.GetByTarget(ctx, "schedule-001")
	for _, r := range reminders {
		assert.Equal(t, StatusCancelled, r.Status)
	}
}

func TestService_ProcessDueReminders(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()
	notifier := NewMockNotifier()
	svc := NewService(store, notifier)

	// Create due reminders
	for i := 0; i < 3; i++ {
		_ = store.Create(ctx, &Reminder{
			ID:        "due-" + string(rune('a'+i)),
			UserID:    1,
			TriggerAt: time.Now().Add(-time.Minute),
			Status:    StatusPending,
			Message:   "Due reminder",
			Channels:  []Channel{ChannelApp},
		})
	}

	processed, err := svc.ProcessDueReminders(ctx)
	require.NoError(t, err)
	assert.Equal(t, 3, processed)
	assert.Equal(t, 3, notifier.GetSentCount())
}

func TestService_ProcessDueReminders_NotifierFailure(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()
	notifier := NewMockNotifier()
	notifier.ShouldFail = true
	svc := NewService(store, notifier)

	_ = store.Create(ctx, &Reminder{
		ID:        "fail-rem",
		UserID:    1,
		TriggerAt: time.Now().Add(-time.Minute),
		Status:    StatusPending,
		Message:   "Will fail",
		Channels:  []Channel{ChannelApp},
	})

	processed, err := svc.ProcessDueReminders(ctx)
	require.NoError(t, err)
	assert.Equal(t, 0, processed)

	r, _ := store.Get(ctx, "fail-rem")
	assert.Equal(t, StatusFailed, r.Status)
}

func TestService_SetDefaultLeadTime(t *testing.T) {
	store := NewMemoryStore()
	svc := NewService(store, nil)

	svc.SetDefaultLeadTime(30)
	assert.Equal(t, 30, svc.defaultLead)

	// Invalid value should be ignored
	svc.SetDefaultLeadTime(-10)
	assert.Equal(t, 30, svc.defaultLead)
}

func TestService_SetDefaultChannels(t *testing.T) {
	store := NewMemoryStore()
	svc := NewService(store, nil)

	svc.SetDefaultChannels([]Channel{ChannelEmail, ChannelWebhook})
	assert.Len(t, svc.defaultChannels, 2)

	// Empty should be ignored
	svc.SetDefaultChannels([]Channel{})
	assert.Len(t, svc.defaultChannels, 2)
}

func TestMockNotifier(t *testing.T) {
	ctx := context.Background()
	notifier := NewMockNotifier()

	err := notifier.Send(ctx, 1, ChannelApp, "Hello")
	require.NoError(t, err)
	assert.Equal(t, 1, notifier.GetSentCount())

	err = notifier.Send(ctx, 1, ChannelEmail, "World")
	require.NoError(t, err)
	assert.Equal(t, 2, notifier.GetSentCount())

	notifier.Clear()
	assert.Equal(t, 0, notifier.GetSentCount())
}

func TestMemoryStore_Concurrent(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			_ = store.Create(ctx, &Reminder{
				ID:        "concurrent-" + string(rune(id)),
				UserID:    int32(id % 10),
				Status:    StatusPending,
				TriggerAt: time.Now().Add(time.Hour),
			})
		}(i)
	}
	wg.Wait()
}

func BenchmarkService_CreateForSchedule(b *testing.B) {
	ctx := context.Background()
	store := NewMemoryStore()
	svc := NewService(store, nil)

	schedule := &ScheduleInfo{
		ID:        "bench-sched",
		Title:     "Benchmark Meeting",
		StartTime: time.Now().Add(time.Hour),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = svc.CreateForSchedule(ctx, 1, schedule, 15)
	}
}

func BenchmarkMemoryStore_GetDueReminders(b *testing.B) {
	ctx := context.Background()
	store := NewMemoryStore()

	// Pre-populate with reminders
	for i := 0; i < 1000; i++ {
		_ = store.Create(ctx, &Reminder{
			ID:        "bench-" + string(rune(i)),
			TriggerAt: time.Now().Add(time.Duration(i-500) * time.Minute),
			Status:    StatusPending,
		})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = store.GetDueReminders(ctx, time.Now())
	}
}
