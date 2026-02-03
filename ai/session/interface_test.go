package session

import (
	"context"
	"testing"
)

// TestSessionServiceContract tests the SessionService contract.
func TestSessionServiceContract(t *testing.T) {
	ctx := context.Background()
	svc := NewMockSessionService()

	t.Run("SaveContext_And_LoadContext", func(t *testing.T) {
		sessionID := "test-session-001"
		context := &ConversationContext{
			UserID:    1,
			AgentType: "memo",
			Messages: []Message{
				{Role: "user", Content: "Test message"},
				{Role: "assistant", Content: "Test response"},
			},
			Metadata: map[string]any{"key": "value"},
		}

		err := svc.SaveContext(ctx, sessionID, context)
		if err != nil {
			t.Fatalf("SaveContext failed: %v", err)
		}

		loaded, err := svc.LoadContext(ctx, sessionID)
		if err != nil {
			t.Fatalf("LoadContext failed: %v", err)
		}
		if loaded == nil {
			t.Fatal("expected context to be loaded")
		}
		if loaded.SessionID != sessionID {
			t.Errorf("expected session ID %s, got %s", sessionID, loaded.SessionID)
		}
		if loaded.AgentType != "memo" {
			t.Errorf("expected agent type memo, got %s", loaded.AgentType)
		}
		if len(loaded.Messages) != 2 {
			t.Errorf("expected 2 messages, got %d", len(loaded.Messages))
		}
	})

	t.Run("LoadContext_NonexistentSession", func(t *testing.T) {
		loaded, err := svc.LoadContext(ctx, "nonexistent-session")
		if err != nil {
			t.Fatalf("LoadContext failed: %v", err)
		}
		if loaded != nil {
			t.Error("expected nil for nonexistent session")
		}
	})

	t.Run("SaveContext_UpdatesExisting", func(t *testing.T) {
		sessionID := "test-update-session"

		// Save initial context
		initial := &ConversationContext{
			UserID:    1,
			AgentType: "memo",
			Messages: []Message{
				{Role: "user", Content: "First message"},
			},
		}
		svc.SaveContext(ctx, sessionID, initial)

		// Update with more messages
		updated := &ConversationContext{
			UserID:    1,
			AgentType: "memo",
			Messages: []Message{
				{Role: "user", Content: "First message"},
				{Role: "assistant", Content: "Response"},
				{Role: "user", Content: "Second message"},
			},
		}
		svc.SaveContext(ctx, sessionID, updated)

		loaded, _ := svc.LoadContext(ctx, sessionID)
		if len(loaded.Messages) != 3 {
			t.Errorf("expected 3 messages after update, got %d", len(loaded.Messages))
		}
	})

	t.Run("SaveContext_SetsTimestamps", func(t *testing.T) {
		sessionID := "test-timestamp-session"
		context := &ConversationContext{
			UserID:    1,
			AgentType: "memo",
			Messages:  []Message{},
		}

		svc.SaveContext(ctx, sessionID, context)
		loaded, _ := svc.LoadContext(ctx, sessionID)

		if loaded.CreatedAt == 0 {
			t.Error("CreatedAt should be set")
		}
		if loaded.UpdatedAt == 0 {
			t.Error("UpdatedAt should be set")
		}
	})

	t.Run("ListSessions_FiltersByUserID", func(t *testing.T) {
		sessions, err := svc.ListSessions(ctx, 1, 10)
		if err != nil {
			t.Fatalf("ListSessions failed: %v", err)
		}

		for _, s := range sessions {
			// Verify all returned sessions are from seed data for user 1
			if s.SessionID == "session-004" {
				t.Error("session-004 belongs to user 2, should not be returned")
			}
		}
	})

	t.Run("ListSessions_RespectsLimit", func(t *testing.T) {
		sessions, err := svc.ListSessions(ctx, 1, 2)
		if err != nil {
			t.Fatalf("ListSessions failed: %v", err)
		}

		if len(sessions) > 2 {
			t.Errorf("expected at most 2 sessions, got %d", len(sessions))
		}
	})

	t.Run("ListSessions_SortedByUpdatedAt", func(t *testing.T) {
		sessions, err := svc.ListSessions(ctx, 1, 10)
		if err != nil {
			t.Fatalf("ListSessions failed: %v", err)
		}

		for i := 1; i < len(sessions); i++ {
			if sessions[i].UpdatedAt > sessions[i-1].UpdatedAt {
				t.Error("sessions should be sorted by UpdatedAt descending")
			}
		}
	})

	t.Run("ListSessions_IncludesLastMessage", func(t *testing.T) {
		sessions, err := svc.ListSessions(ctx, 1, 10)
		if err != nil {
			t.Fatalf("ListSessions failed: %v", err)
		}

		for _, s := range sessions {
			if s.LastMessage == "" {
				t.Logf("Session %s has empty last message (may be expected if no messages)", s.SessionID)
			}
		}
	})

	t.Run("ConversationContext_PreservesMetadata", func(t *testing.T) {
		sessionID := "test-metadata-session"
		context := &ConversationContext{
			UserID:    1,
			AgentType: "memo",
			Messages:  []Message{},
			Metadata: map[string]any{
				"topic":     "test",
				"important": true,
				"count":     42,
			},
		}

		svc.SaveContext(ctx, sessionID, context)
		loaded, _ := svc.LoadContext(ctx, sessionID)

		if loaded.Metadata["topic"] != "test" {
			t.Error("metadata topic not preserved")
		}
		if loaded.Metadata["important"] != true {
			t.Error("metadata important not preserved")
		}
	})

	t.Run("SessionSummary_HasRequiredFields", func(t *testing.T) {
		sessions, err := svc.ListSessions(ctx, 1, 1)
		if err != nil {
			t.Fatalf("ListSessions failed: %v", err)
		}

		if len(sessions) > 0 {
			s := sessions[0]
			if s.SessionID == "" {
				t.Error("SessionID should not be empty")
			}
			if s.AgentType == "" {
				t.Error("AgentType should not be empty")
			}
			if s.UpdatedAt == 0 {
				t.Error("UpdatedAt should not be zero")
			}
		}
	})

	t.Run("LoadContext_ReturnsCopy", func(t *testing.T) {
		loaded1, _ := svc.LoadContext(ctx, "session-001")
		loaded2, _ := svc.LoadContext(ctx, "session-001")

		// Modify loaded1
		loaded1.Messages = append(loaded1.Messages, Message{Role: "user", Content: "New"})

		// loaded2 should not be affected
		if len(loaded2.Messages) == len(loaded1.Messages) {
			t.Error("LoadContext should return independent copies")
		}
	})

	t.Run("DeleteSession_RemovesSession", func(t *testing.T) {
		sessionID := "test-delete-session"
		context := &ConversationContext{
			UserID:    1,
			AgentType: "memo",
			Messages:  []Message{{Role: "user", Content: "Test"}},
		}

		// Create session
		svc.SaveContext(ctx, sessionID, context)

		// Verify it exists
		loaded, _ := svc.LoadContext(ctx, sessionID)
		if loaded == nil {
			t.Fatal("session should exist before delete")
		}

		// Delete session
		err := svc.DeleteSession(ctx, sessionID)
		if err != nil {
			t.Fatalf("DeleteSession failed: %v", err)
		}

		// Verify it's gone
		loaded, _ = svc.LoadContext(ctx, sessionID)
		if loaded != nil {
			t.Error("session should not exist after delete")
		}
	})

	t.Run("CleanupExpired_RemovesOldSessions", func(t *testing.T) {
		// Use SetSessionDirectly to avoid timestamp override
		oldSession := &ConversationContext{
			SessionID: "old-session",
			UserID:    999,
			AgentType: "memo",
			Messages:  []Message{},
			CreatedAt: 1000000,
			UpdatedAt: 1000000, // Very old timestamp
		}
		svc.SetSessionDirectly("old-session", oldSession)

		// Cleanup with 0 retention (all sessions are "expired")
		deleted, err := svc.CleanupExpired(ctx, 0)
		if err != nil {
			t.Fatalf("CleanupExpired failed: %v", err)
		}

		if deleted == 0 {
			t.Error("expected at least one session to be deleted")
		}

		// Verify old session is gone
		loaded, _ := svc.LoadContext(ctx, "old-session")
		if loaded != nil {
			t.Error("old session should be deleted")
		}
	})
}
