package session

import (
	"context"
	"testing"
)

func TestSessionRecovery(t *testing.T) {
	ctx := context.Background()
	mock := NewMockSessionService()
	recovery := NewSessionRecovery(mock)

	t.Run("RecoverSession_NewSession", func(t *testing.T) {
		session, err := recovery.RecoverSession(ctx, "new-session-123", 1)
		if err != nil {
			t.Fatalf("RecoverSession failed: %v", err)
		}

		if session == nil {
			t.Fatal("expected new session to be created")
		}
		if session.SessionID != "new-session-123" {
			t.Errorf("expected session ID 'new-session-123', got '%s'", session.SessionID)
		}
		if session.UserID != 1 {
			t.Errorf("expected user ID 1, got %d", session.UserID)
		}
		if len(session.Messages) != 0 {
			t.Errorf("expected 0 messages, got %d", len(session.Messages))
		}
	})

	t.Run("RecoverSession_ExistingSession", func(t *testing.T) {
		// Pre-create a session
		existingCtx := &ConversationContext{
			UserID:    1,
			AgentType: "memo",
			Messages: []Message{
				{Role: "user", Content: "Hello"},
				{Role: "assistant", Content: "Hi there"},
			},
		}
		if err := mock.SaveContext(ctx, "existing-session", existingCtx); err != nil {
			t.Fatalf("SaveContext failed: %v", err)
		}

		// Recover it
		session, err := recovery.RecoverSession(ctx, "existing-session", 1)
		if err != nil {
			t.Fatalf("RecoverSession failed: %v", err)
		}

		if session == nil {
			t.Fatal("expected session to be recovered")
		}
		if len(session.Messages) != 2 {
			t.Errorf("expected 2 messages, got %d", len(session.Messages))
		}
	})

	t.Run("AppendMessage_AddsMessage", func(t *testing.T) {
		// Create session first
		session := &ConversationContext{
			UserID:    1,
			AgentType: "memo",
			Messages:  []Message{},
		}
		if err := mock.SaveContext(ctx, "append-test", session); err != nil {
			t.Fatalf("SaveContext failed: %v", err)
		}

		// Append message
		msg := &Message{Role: "user", Content: "New message"}
		err := recovery.AppendMessage(ctx, "append-test", msg)
		if err != nil {
			t.Fatalf("AppendMessage failed: %v", err)
		}

		// Verify
		loaded, _ := mock.LoadContext(ctx, "append-test")
		if len(loaded.Messages) != 1 {
			t.Errorf("expected 1 message, got %d", len(loaded.Messages))
		}
		if loaded.Messages[0].Content != "New message" {
			t.Errorf("expected 'New message', got '%s'", loaded.Messages[0].Content)
		}
	})

	t.Run("AppendMessage_SlidingWindow", func(t *testing.T) {
		// Create session with max messages
		messages := make([]Message, MaxMessagesPerSession)
		for i := range messages {
			messages[i] = Message{Role: "user", Content: "Message"}
		}
		session := &ConversationContext{
			UserID:    1,
			AgentType: "memo",
			Messages:  messages,
		}
		if err := mock.SaveContext(ctx, "sliding-window-test", session); err != nil {
			t.Fatalf("SaveContext failed: %v", err)
		}

		// Append one more
		msg := &Message{Role: "user", Content: "Overflow message"}
		err := recovery.AppendMessage(ctx, "sliding-window-test", msg)
		if err != nil {
			t.Fatalf("AppendMessage failed: %v", err)
		}

		// Verify sliding window
		loaded, _ := mock.LoadContext(ctx, "sliding-window-test")
		if len(loaded.Messages) > MaxMessagesPerSession {
			t.Errorf("expected at most %d messages, got %d", MaxMessagesPerSession, len(loaded.Messages))
		}
		// Last message should be the overflow
		if loaded.Messages[len(loaded.Messages)-1].Content != "Overflow message" {
			t.Error("last message should be the newly appended one")
		}
	})

	t.Run("AppendTurn_AddsBothMessages", func(t *testing.T) {
		session := &ConversationContext{
			UserID:    1,
			AgentType: "memo",
			Messages:  []Message{},
		}
		if err := mock.SaveContext(ctx, "turn-test", session); err != nil {
			t.Fatalf("SaveContext failed: %v", err)
		}

		err := recovery.AppendTurn(ctx, "turn-test", "User question", "Assistant answer", "schedule")
		if err != nil {
			t.Fatalf("AppendTurn failed: %v", err)
		}

		loaded, _ := mock.LoadContext(ctx, "turn-test")
		if len(loaded.Messages) != 2 {
			t.Errorf("expected 2 messages, got %d", len(loaded.Messages))
		}
		if loaded.AgentType != "schedule" {
			t.Errorf("expected agent type 'schedule', got '%s'", loaded.AgentType)
		}
	})

	t.Run("UpdateMetadata_SetsValue", func(t *testing.T) {
		session := &ConversationContext{
			UserID:    1,
			AgentType: "memo",
			Messages:  []Message{},
			Metadata:  map[string]any{},
		}
		if err := mock.SaveContext(ctx, "metadata-test", session); err != nil {
			t.Fatalf("SaveContext failed: %v", err)
		}

		err := recovery.UpdateMetadata(ctx, "metadata-test", "topic", "important")
		if err != nil {
			t.Fatalf("UpdateMetadata failed: %v", err)
		}

		loaded, _ := mock.LoadContext(ctx, "metadata-test")
		if loaded.Metadata["topic"] != "important" {
			t.Errorf("expected topic 'important', got '%v'", loaded.Metadata["topic"])
		}
	})

	t.Run("GetRecentMessages_ReturnsLastN", func(t *testing.T) {
		session := &ConversationContext{
			UserID:    1,
			AgentType: "memo",
			Messages: []Message{
				{Role: "user", Content: "1"},
				{Role: "assistant", Content: "2"},
				{Role: "user", Content: "3"},
				{Role: "assistant", Content: "4"},
			},
		}
		if err := mock.SaveContext(ctx, "recent-test", session); err != nil {
			t.Fatalf("SaveContext failed: %v", err)
		}

		messages, err := recovery.GetRecentMessages(ctx, "recent-test", 2)
		if err != nil {
			t.Fatalf("GetRecentMessages failed: %v", err)
		}

		if len(messages) != 2 {
			t.Errorf("expected 2 messages, got %d", len(messages))
		}
		if messages[0].Content != "3" || messages[1].Content != "4" {
			t.Error("expected last 2 messages")
		}
	})

	t.Run("ClearMessages_RemovesAllMessages", func(t *testing.T) {
		session := &ConversationContext{
			UserID:    1,
			AgentType: "memo",
			Messages: []Message{
				{Role: "user", Content: "1"},
				{Role: "assistant", Content: "2"},
			},
			Metadata: map[string]any{"keep": "this"},
		}
		if err := mock.SaveContext(ctx, "clear-test", session); err != nil {
			t.Fatalf("SaveContext failed: %v", err)
		}

		err := recovery.ClearMessages(ctx, "clear-test")
		if err != nil {
			t.Fatalf("ClearMessages failed: %v", err)
		}

		loaded, _ := mock.LoadContext(ctx, "clear-test")
		if len(loaded.Messages) != 0 {
			t.Errorf("expected 0 messages, got %d", len(loaded.Messages))
		}
		// Metadata should be preserved
		if loaded.Metadata["keep"] != "this" {
			t.Error("metadata should be preserved")
		}
	})

	t.Run("AppendMessage_NonexistentSession_ReturnsError", func(t *testing.T) {
		msg := &Message{Role: "user", Content: "Test"}
		err := recovery.AppendMessage(ctx, "nonexistent-session", msg)
		if err == nil {
			t.Error("expected error for nonexistent session")
		}
	})
}
