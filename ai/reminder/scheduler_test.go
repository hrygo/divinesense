package reminder

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScheduler_StartStop(t *testing.T) {
	store := NewMemoryStore()
	notifier := NewMockNotifier()
	svc := NewService(store, notifier)

	config := SchedulerConfig{
		Interval: 100 * time.Millisecond,
	}
	scheduler := NewScheduler(svc, config)

	// Start
	ctx := context.Background()
	err := scheduler.Start(ctx)
	require.NoError(t, err)
	assert.True(t, scheduler.IsRunning())

	// Double start should be no-op
	err = scheduler.Start(ctx)
	require.NoError(t, err)

	// Stop
	scheduler.Stop()
	assert.False(t, scheduler.IsRunning())

	// Double stop should be no-op
	scheduler.Stop()
}

func TestScheduler_ProcessesDueReminders(t *testing.T) {
	store := NewMemoryStore()
	notifier := NewMockNotifier()
	svc := NewService(store, notifier)

	// Create due reminders
	for i := 0; i < 3; i++ {
		_ = store.Create(context.Background(), &Reminder{
			ID:        "sched-" + string(rune('a'+i)),
			UserID:    1,
			TriggerAt: time.Now().Add(-time.Minute),
			Status:    StatusPending,
			Message:   "Test",
			Channels:  []Channel{ChannelApp},
		})
	}

	config := SchedulerConfig{
		Interval: 50 * time.Millisecond,
	}
	scheduler := NewScheduler(svc, config)
	processedChan := scheduler.EnableTestMode()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	err := scheduler.Start(ctx)
	require.NoError(t, err)

	// Wait for at least one cycle
	select {
	case processed := <-processedChan:
		assert.Equal(t, 3, processed)
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timeout waiting for reminders to be processed")
	}

	scheduler.Stop()
	assert.Equal(t, 3, notifier.GetSentCount())
}

func TestScheduler_RunOnce(t *testing.T) {
	store := NewMemoryStore()
	notifier := NewMockNotifier()
	svc := NewService(store, notifier)

	_ = store.Create(context.Background(), &Reminder{
		ID:        "once-1",
		UserID:    1,
		TriggerAt: time.Now().Add(-time.Minute),
		Status:    StatusPending,
		Message:   "Test",
		Channels:  []Channel{ChannelApp},
	})

	scheduler := NewScheduler(svc, DefaultSchedulerConfig())

	processed, err := scheduler.RunOnce(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 1, processed)
}

func TestScheduler_ContextCancellation(t *testing.T) {
	store := NewMemoryStore()
	notifier := NewMockNotifier()
	svc := NewService(store, notifier)

	config := SchedulerConfig{
		Interval: 100 * time.Millisecond,
	}
	scheduler := NewScheduler(svc, config)

	ctx, cancel := context.WithCancel(context.Background())
	_ = scheduler.Start(ctx)

	time.Sleep(50 * time.Millisecond)
	cancel()

	// Give time for graceful shutdown
	time.Sleep(100 * time.Millisecond)
}

func TestDefaultSchedulerConfig(t *testing.T) {
	config := DefaultSchedulerConfig()
	assert.Equal(t, time.Minute, config.Interval)
	assert.Equal(t, 3, config.MaxRetries)
	assert.Equal(t, 5*time.Second, config.RetryDelay)
	assert.Equal(t, 100, config.BatchSize)
}

func TestWorker_ProcessReminder(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()
	notifier := NewMockNotifier()
	worker := NewWorker(store, notifier, 3)

	reminder := &Reminder{
		ID:       "worker-1",
		UserID:   1,
		Message:  "Test",
		Channels: []Channel{ChannelApp, ChannelEmail},
		Status:   StatusPending,
	}
	_ = store.Create(ctx, reminder)

	err := worker.ProcessReminder(ctx, reminder)
	require.NoError(t, err)
	assert.Equal(t, 2, notifier.GetSentCount())

	r, _ := store.Get(ctx, reminder.ID)
	assert.Equal(t, StatusSent, r.Status)
}

func TestWorker_ProcessReminder_WithRetry(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()
	notifier := NewMockNotifier()
	notifier.ShouldFail = true

	worker := NewWorker(store, notifier, 2)
	worker.retryDelay = 10 * time.Millisecond // Speed up test

	reminder := &Reminder{
		ID:       "retry-1",
		UserID:   1,
		Message:  "Test",
		Channels: []Channel{ChannelApp},
		Status:   StatusPending,
	}
	_ = store.Create(ctx, reminder)

	err := worker.ProcessReminder(ctx, reminder)
	assert.Error(t, err) // Should fail after retries

	r, _ := store.Get(ctx, reminder.ID)
	assert.Equal(t, StatusFailed, r.Status)
}

func TestWorker_ProcessBatch(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()
	notifier := NewMockNotifier()
	worker := NewWorker(store, notifier, 1)

	var reminders []*Reminder
	for i := 0; i < 5; i++ {
		r := &Reminder{
			ID:       "batch-" + string(rune('a'+i)),
			UserID:   1,
			Message:  "Test",
			Channels: []Channel{ChannelApp},
			Status:   StatusPending,
		}
		_ = store.Create(ctx, r)
		reminders = append(reminders, r)
	}

	processed, failed := worker.ProcessBatch(ctx, reminders)
	assert.Equal(t, 5, processed)
	assert.Equal(t, 0, failed)
}

func TestWorker_ProcessBatch_ContextCancel(t *testing.T) {
	store := NewMemoryStore()
	notifier := NewMockNotifier()
	worker := NewWorker(store, notifier, 1)

	var reminders []*Reminder
	for i := 0; i < 10; i++ {
		r := &Reminder{
			ID:       "cancel-" + string(rune('a'+i)),
			UserID:   1,
			Message:  "Test",
			Channels: []Channel{ChannelApp},
			Status:   StatusPending,
		}
		_ = store.Create(context.Background(), r)
		reminders = append(reminders, r)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	processed, _ := worker.ProcessBatch(ctx, reminders)
	assert.Less(t, processed, 10) // Should not process all
}

func TestHealthCheck(t *testing.T) {
	store := NewMemoryStore()
	svc := NewService(store, nil)
	scheduler := NewScheduler(svc, DefaultSchedulerConfig())
	healthCheck := NewHealthCheck(scheduler)

	// Not running
	status := healthCheck.Check()
	assert.False(t, status.Healthy)
	assert.Equal(t, int64(1), status.CheckCount)

	// Start scheduler
	ctx := context.Background()
	_ = scheduler.Start(ctx)

	status = healthCheck.Check()
	assert.True(t, status.Healthy)
	assert.Equal(t, int64(2), status.CheckCount)

	scheduler.Stop()
}

func TestMetricsCollector(t *testing.T) {
	collector := NewMetricsCollector()

	collector.RecordProcessed(10)
	collector.RecordProcessed(5)
	collector.RecordFailed(2)

	stats := collector.GetStats()
	assert.Equal(t, int64(15), stats.TotalProcessed)
	assert.Equal(t, int64(2), stats.TotalFailed)
	assert.False(t, stats.LastRunAt.IsZero())
}

func BenchmarkScheduler_ProcessCycle(b *testing.B) {
	store := NewMemoryStore()
	notifier := NewMockNotifier()
	svc := NewService(store, notifier)

	// Pre-populate with reminders
	for i := 0; i < 100; i++ {
		_ = store.Create(context.Background(), &Reminder{
			ID:        "bench-" + string(rune(i)),
			UserID:    1,
			TriggerAt: time.Now().Add(-time.Minute),
			Status:    StatusPending,
			Message:   "Benchmark",
			Channels:  []Channel{ChannelApp},
		})
	}

	scheduler := NewScheduler(svc, DefaultSchedulerConfig())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = scheduler.RunOnce(context.Background())

		// Reset reminders for next iteration
		for j := 0; j < 100; j++ {
			r, _ := store.Get(context.Background(), "bench-"+string(rune(j)))
			if r != nil {
				r.Status = StatusPending
			}
		}
	}
}
