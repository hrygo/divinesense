package session

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSessionCleanupJob(t *testing.T) {
	ctx := context.Background()

	t.Run("NewSessionCleanupJob_DefaultConfig", func(t *testing.T) {
		mock := NewMockSessionService()
		job := NewSessionCleanupJob(mock, CleanupConfig{})

		if job.config.RetentionDays != DefaultRetentionDays {
			t.Errorf("expected default retention days %d, got %d", DefaultRetentionDays, job.config.RetentionDays)
		}
		if job.config.CleanupInterval != DefaultCleanupInterval {
			t.Errorf("expected default cleanup interval %v, got %v", DefaultCleanupInterval, job.config.CleanupInterval)
		}
	})

	t.Run("NewSessionCleanupJob_CustomConfig", func(t *testing.T) {
		mock := NewMockSessionService()
		config := CleanupConfig{
			RetentionDays:   7,
			CleanupInterval: 1 * time.Hour,
		}
		job := NewSessionCleanupJob(mock, config)

		if job.config.RetentionDays != 7 {
			t.Errorf("expected retention days 7, got %d", job.config.RetentionDays)
		}
		if job.config.CleanupInterval != 1*time.Hour {
			t.Errorf("expected cleanup interval 1h, got %v", job.config.CleanupInterval)
		}
	})

	t.Run("RunOnce_CleansExpiredSessions", func(t *testing.T) {
		mock := NewMockSessionService()
		mock.Clear()

		// Use SetSessionDirectly to set old timestamp without override
		oldSession := &ConversationContext{
			UserID:    1,
			AgentType: "memo",
			Messages:  []Message{},
			CreatedAt: 1000000,
			UpdatedAt: 1000000, // Very old
		}
		mock.SetSessionDirectly("old-session", oldSession)

		// Create recent session (use SaveContext which sets current timestamp)
		recentSession := &ConversationContext{
			UserID:    1,
			AgentType: "memo",
			Messages:  []Message{},
		}
		mock.SaveContext(ctx, "recent-session", recentSession)

		// Run cleanup with 0 retention (everything older than today is expired)
		job := NewSessionCleanupJob(mock, CleanupConfig{RetentionDays: 0})
		deleted, err := job.RunOnce(ctx)
		if err != nil {
			t.Fatalf("RunOnce failed: %v", err)
		}

		// Old session should be deleted
		if deleted < 1 {
			t.Errorf("expected at least 1 deleted, got %d", deleted)
		}

		// Verify old session is gone
		loaded, _ := mock.LoadContext(ctx, "old-session")
		if loaded != nil {
			t.Error("old session should be deleted")
		}
	})

	t.Run("StartStop_ManagesRunningState", func(t *testing.T) {
		mock := NewMockSessionService()
		job := NewSessionCleanupJob(mock, CleanupConfig{
			RetentionDays:   30,
			CleanupInterval: 1 * time.Hour,
		})

		if job.IsRunning() {
			t.Error("job should not be running initially")
		}

		// Start
		err := job.Start(ctx)
		if err != nil {
			t.Fatalf("Start failed: %v", err)
		}

		// Give it time to start
		time.Sleep(10 * time.Millisecond)

		if !job.IsRunning() {
			t.Error("job should be running after Start")
		}

		// Start again (should be idempotent)
		err = job.Start(ctx)
		if err != nil {
			t.Fatalf("Second Start failed: %v", err)
		}

		// Stop
		job.Stop()

		// Give it time to stop
		time.Sleep(10 * time.Millisecond)

		if job.IsRunning() {
			t.Error("job should not be running after Stop")
		}

		// Stop again (should be idempotent)
		job.Stop() // Should not panic
	})

	t.Run("DefaultCleanupConfig_ReturnsDefaults", func(t *testing.T) {
		config := DefaultCleanupConfig()

		if config.RetentionDays != DefaultRetentionDays {
			t.Errorf("expected default retention days %d, got %d", DefaultRetentionDays, config.RetentionDays)
		}
		if config.CleanupInterval != DefaultCleanupInterval {
			t.Errorf("expected default cleanup interval %v, got %v", DefaultCleanupInterval, config.CleanupInterval)
		}
	})
}

// Additional tests for cleanup functionality

func TestSessionCleanupJob_NegativeConfigValues(t *testing.T) {
	mock := NewMockSessionService()

	t.Run("Negative retention uses default", func(t *testing.T) {
		job := NewSessionCleanupJob(mock, CleanupConfig{
			RetentionDays:   -5,
			CleanupInterval: time.Hour,
		})

		assert.Equal(t, DefaultRetentionDays, job.config.RetentionDays)
		assert.Equal(t, time.Hour, job.config.CleanupInterval)
	})

	t.Run("Negative interval uses default", func(t *testing.T) {
		job := NewSessionCleanupJob(mock, CleanupConfig{
			RetentionDays:   7,
			CleanupInterval: -1,
		})

		assert.Equal(t, 7, job.config.RetentionDays)
		assert.Equal(t, DefaultCleanupInterval, job.config.CleanupInterval)
	})
}

func TestSessionCleanupJob_RunOnce_MultipleExpiredSessions(t *testing.T) {
	ctx := context.Background()
	mock := NewMockSessionService()
	mock.Clear()

	// Add multiple old sessions
	oldTime := time.Now().Add(-40 * 24 * time.Hour).Unix()
	for i := 0; i < 5; i++ {
		mock.SetSessionDirectly("old-"+string(rune('0'+i)), &ConversationContext{
			SessionID: "old-" + string(rune('0'+i)),
			UserID:    1,
			UpdatedAt: oldTime,
			Messages:  []Message{{Role: "user", Content: "Old message"}},
		})
	}

	// Add recent sessions
	for i := 0; i < 3; i++ {
		mock.SaveContext(ctx, "recent-"+string(rune('0'+i)), &ConversationContext{
			UserID:   1,
			Messages: []Message{{Role: "user", Content: "Recent message"}},
		})
	}

	job := NewSessionCleanupJob(mock, CleanupConfig{RetentionDays: 30})
	deleted, err := job.RunOnce(ctx)

	require.NoError(t, err)
	assert.Equal(t, int64(5), deleted)

	// Verify recent sessions still exist
	for i := 0; i < 3; i++ {
		loaded, _ := mock.LoadContext(ctx, "recent-"+string(rune('0'+i)))
		assert.NotNil(t, loaded, "recent session should still exist")
	}
}

func TestSessionCleanupJob_RunOnce_NoExpiredSessions(t *testing.T) {
	ctx := context.Background()
	mock := NewMockSessionService()
	mock.Clear()

	// Add only recent sessions
	for i := 0; i < 3; i++ {
		mock.SaveContext(ctx, "session-"+string(rune('0'+i)), &ConversationContext{
			UserID:   1,
			Messages: []Message{{Role: "user", Content: "Message"}},
		})
	}

	job := NewSessionCleanupJob(mock, CleanupConfig{RetentionDays: 30})
	deleted, err := job.RunOnce(ctx)

	require.NoError(t, err)
	assert.Equal(t, int64(0), deleted)
}

func TestSessionCleanupJob_RunOnce_EmptyService(t *testing.T) {
	ctx := context.Background()
	mock := NewMockSessionService()
	mock.Clear()

	job := NewSessionCleanupJob(mock, CleanupConfig{RetentionDays: 30})
	deleted, err := job.RunOnce(ctx)

	require.NoError(t, err)
	assert.Equal(t, int64(0), deleted)
}

func TestSessionCleanupJob_ContextCancellation(t *testing.T) {
	mock := NewMockSessionService()
	job := NewSessionCleanupJob(mock, CleanupConfig{
		RetentionDays:   30,
		CleanupInterval: 10 * time.Millisecond,
	})

	ctx, cancel := context.WithCancel(context.Background())

	err := job.Start(ctx)
	require.NoError(t, err)

	// Verify running
	time.Sleep(5 * time.Millisecond)
	assert.True(t, job.IsRunning())

	// Cancel context
	cancel()

	// Wait for goroutine to exit
	time.Sleep(20 * time.Millisecond)

	// Should have stopped
	assert.False(t, job.IsRunning())
}

// Benchmark tests

func BenchmarkRunOnce(b *testing.B) {
	ctx := context.Background()
	mock := NewMockSessionService()

	// Add 100 old sessions
	oldTime := time.Now().Add(-40 * 24 * time.Hour).Unix()
	for i := 0; i < 100; i++ {
		mock.SetSessionDirectly("old-"+string(rune(i)), &ConversationContext{
			UserID:    1,
			UpdatedAt: oldTime,
			Messages:  []Message{{Role: "user", Content: "Old"}},
		})
	}

	job := NewSessionCleanupJob(mock, CleanupConfig{RetentionDays: 30})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = job.RunOnce(ctx)
	}
}
