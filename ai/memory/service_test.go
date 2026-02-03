package memory

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestService_ShortTermMemory(t *testing.T) {
	ctx := context.Background()
	svc := NewService(nil, 5) // nil store = short-term only mode
	defer svc.Close()

	t.Run("AddAndGetMessages", func(t *testing.T) {
		sessionID := "test-session-1"

		// Add messages
		err := svc.AddMessage(ctx, sessionID, Message{Role: "user", Content: "Hello"})
		require.NoError(t, err)
		err = svc.AddMessage(ctx, sessionID, Message{Role: "assistant", Content: "Hi there!"})
		require.NoError(t, err)

		// Get messages
		msgs, err := svc.GetRecentMessages(ctx, sessionID, 10)
		require.NoError(t, err)
		assert.Len(t, msgs, 2)
		assert.Equal(t, "Hello", msgs[0].Content)
		assert.Equal(t, "Hi there!", msgs[1].Content)
	})

	t.Run("SlidingWindow", func(t *testing.T) {
		sessionID := "test-session-sliding"

		// Add 7 messages to a service with max 5
		for i := 0; i < 7; i++ {
			err := svc.AddMessage(ctx, sessionID, Message{Role: "user", Content: string(rune('A' + i))})
			require.NoError(t, err)
		}

		// Should only keep last 5
		msgs, err := svc.GetRecentMessages(ctx, sessionID, 10)
		require.NoError(t, err)
		assert.Len(t, msgs, 5)
		assert.Equal(t, "C", msgs[0].Content) // Oldest remaining
		assert.Equal(t, "G", msgs[4].Content) // Newest
	})

	t.Run("LimitMessages", func(t *testing.T) {
		sessionID := "test-session-limit"

		for i := 0; i < 5; i++ {
			err := svc.AddMessage(ctx, sessionID, Message{Role: "user", Content: string(rune('A' + i))})
			require.NoError(t, err)
		}

		// Request only 2 messages
		msgs, err := svc.GetRecentMessages(ctx, sessionID, 2)
		require.NoError(t, err)
		assert.Len(t, msgs, 2)
		assert.Equal(t, "D", msgs[0].Content)
		assert.Equal(t, "E", msgs[1].Content)
	})

	t.Run("ClearSession", func(t *testing.T) {
		sessionID := "test-session-clear"

		err := svc.AddMessage(ctx, sessionID, Message{Role: "user", Content: "Hello"})
		require.NoError(t, err)

		svc.ClearSession(sessionID)

		msgs, err := svc.GetRecentMessages(ctx, sessionID, 10)
		require.NoError(t, err)
		assert.Len(t, msgs, 0)
	})

	t.Run("EmptySession", func(t *testing.T) {
		msgs, err := svc.GetRecentMessages(ctx, "nonexistent-session", 10)
		require.NoError(t, err)
		assert.Len(t, msgs, 0)
	})

	t.Run("MessageTimestamp", func(t *testing.T) {
		sessionID := "test-session-timestamp"

		before := time.Now()
		err := svc.AddMessage(ctx, sessionID, Message{Role: "user", Content: "Test"})
		require.NoError(t, err)
		after := time.Now()

		msgs, err := svc.GetRecentMessages(ctx, sessionID, 1)
		require.NoError(t, err)
		assert.Len(t, msgs, 1)
		assert.True(t, msgs[0].Timestamp.After(before) || msgs[0].Timestamp.Equal(before))
		assert.True(t, msgs[0].Timestamp.Before(after) || msgs[0].Timestamp.Equal(after))
	})
}

func TestService_LongTermMemory_NoStore(t *testing.T) {
	ctx := context.Background()
	svc := NewService(nil, 10) // nil store
	defer svc.Close()

	t.Run("SaveEpisode_ReturnsError", func(t *testing.T) {
		err := svc.SaveEpisode(ctx, EpisodicMemory{
			UserID:    1,
			AgentType: "memo",
			UserInput: "test",
		})
		// Should return ErrLongTermNotConfigured (Issue #3 fix)
		assert.ErrorIs(t, err, ErrLongTermNotConfigured)
	})

	t.Run("SearchEpisodes_ReturnsError", func(t *testing.T) {
		episodes, err := svc.SearchEpisodes(ctx, 1, "test", 10)
		// Should return ErrLongTermNotConfigured (Issue #3 fix)
		assert.ErrorIs(t, err, ErrLongTermNotConfigured)
		assert.Nil(t, episodes)
	})

	t.Run("GetPreferences_ReturnsDefaults", func(t *testing.T) {
		// GetPreferences returns defaults for better UX even without long-term store
		prefs, err := svc.GetPreferences(ctx, 1)
		assert.NoError(t, err)
		assert.NotNil(t, prefs)
		assert.Equal(t, "Asia/Shanghai", prefs.Timezone)
		assert.Equal(t, 60, prefs.DefaultDuration)
	})

	t.Run("UpdatePreferences_ReturnsError", func(t *testing.T) {
		err := svc.UpdatePreferences(ctx, 1, &UserPreferences{Timezone: "UTC"})
		// Should return ErrLongTermNotConfigured (Issue #3 fix)
		assert.ErrorIs(t, err, ErrLongTermNotConfigured)
	})

	t.Run("HasLongTermMemory_ReturnsFalse", func(t *testing.T) {
		assert.False(t, svc.HasLongTermMemory())
	})
}

func TestService_SessionManagement(t *testing.T) {
	svc := NewService(nil, 10)
	defer svc.Close()
	ctx := context.Background()

	t.Run("ActiveSessionCount", func(t *testing.T) {
		initialCount := svc.ActiveSessionCount()

		// Add messages to new sessions
		_ = svc.AddMessage(ctx, "session-a", Message{Role: "user", Content: "A"})
		_ = svc.AddMessage(ctx, "session-b", Message{Role: "user", Content: "B"})

		assert.Equal(t, initialCount+2, svc.ActiveSessionCount())

		// Clear one session
		svc.ClearSession("session-a")
		assert.Equal(t, initialCount+1, svc.ActiveSessionCount())
	})
}

func TestShortTermMemory_ConcurrentAccess(t *testing.T) {
	stm := NewShortTermMemory(100)
	defer stm.Close()

	done := make(chan bool)
	sessionID := "concurrent-session"

	// Writer goroutine
	go func() {
		for i := 0; i < 100; i++ {
			stm.AddMessage(sessionID, Message{Role: "user", Content: "msg"})
		}
		done <- true
	}()

	// Reader goroutine
	go func() {
		for i := 0; i < 100; i++ {
			_ = stm.GetMessages(sessionID, 10)
		}
		done <- true
	}()

	// Wait for both
	<-done
	<-done

	// Should not panic or have race conditions
	msgs := stm.GetMessages(sessionID, 200)
	assert.Equal(t, 100, len(msgs))
}

func TestShortTermMemory_Close(t *testing.T) {
	stm := NewShortTermMemory(10)

	// Add some data
	stm.AddMessage("test", Message{Role: "user", Content: "hello"})

	// Close should not panic and should stop cleanup goroutine
	stm.Close()

	// Operations after close should still work (just no cleanup)
	msgs := stm.GetMessages("test", 10)
	assert.Len(t, msgs, 1)
}

func TestDefaultPreferences(t *testing.T) {
	// Test that DefaultPreferences returns consistent values
	prefs1 := DefaultPreferences()
	prefs2 := DefaultPreferences()

	assert.Equal(t, prefs1.Timezone, prefs2.Timezone)
	assert.Equal(t, prefs1.DefaultDuration, prefs2.DefaultDuration)
	assert.Equal(t, prefs1.CommunicationStyle, prefs2.CommunicationStyle)
}
