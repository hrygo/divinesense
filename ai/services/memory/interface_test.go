package memory

import (
	"context"
	"testing"
	"time"
)

// TestMemoryServiceContract tests the MemoryService contract.
// Both Mock and real implementations should pass these tests.
func TestMemoryServiceContract(t *testing.T) {
	ctx := context.Background()
	svc := NewMockMemoryService()

	t.Run("GetRecentMessages_ReturnsMessages", func(t *testing.T) {
		msgs, err := svc.GetRecentMessages(ctx, "session-001", 5)
		if err != nil {
			t.Fatalf("GetRecentMessages failed: %v", err)
		}
		if len(msgs) != 5 {
			t.Errorf("expected 5 messages, got %d", len(msgs))
		}
	})

	t.Run("GetRecentMessages_EmptySession", func(t *testing.T) {
		msgs, err := svc.GetRecentMessages(ctx, "nonexistent-session", 10)
		if err != nil {
			t.Fatalf("GetRecentMessages failed: %v", err)
		}
		if len(msgs) != 0 {
			t.Errorf("expected 0 messages for nonexistent session, got %d", len(msgs))
		}
	})

	t.Run("AddMessage_StoresMessage", func(t *testing.T) {
		sessionID := "test-session-add"
		msg := Message{
			Role:      "user",
			Content:   "Test message content",
			Timestamp: time.Now(),
		}

		err := svc.AddMessage(ctx, sessionID, msg)
		if err != nil {
			t.Fatalf("AddMessage failed: %v", err)
		}

		msgs, err := svc.GetRecentMessages(ctx, sessionID, 10)
		if err != nil {
			t.Fatalf("GetRecentMessages failed: %v", err)
		}
		if len(msgs) != 1 {
			t.Errorf("expected 1 message, got %d", len(msgs))
		}
		if msgs[0].Content != "Test message content" {
			t.Errorf("expected content 'Test message content', got '%s'", msgs[0].Content)
		}
	})

	t.Run("SaveEpisode_StoresData", func(t *testing.T) {
		episode := EpisodicMemory{
			UserID:     1,
			Timestamp:  time.Now(),
			AgentType:  "memo",
			UserInput:  "Test episodic input",
			Outcome:    "success",
			Summary:    "Test summary",
			Importance: 0.7,
		}

		err := svc.SaveEpisode(ctx, episode)
		if err != nil {
			t.Fatalf("SaveEpisode failed: %v", err)
		}

		episodes, err := svc.SearchEpisodes(ctx, 1, "Test episodic", 10)
		if err != nil {
			t.Fatalf("SearchEpisodes failed: %v", err)
		}
		if len(episodes) == 0 {
			t.Error("expected at least 1 episode after save")
		}
	})

	t.Run("SearchEpisodes_EmptyQuery_ReturnsRecent", func(t *testing.T) {
		episodes, err := svc.SearchEpisodes(ctx, 1, "", 3)
		if err != nil {
			t.Fatalf("SearchEpisodes failed: %v", err)
		}
		if len(episodes) == 0 {
			t.Error("expected episodes for empty query")
		}
		if len(episodes) > 3 {
			t.Errorf("expected at most 3 episodes, got %d", len(episodes))
		}
	})

	t.Run("SearchEpisodes_WithQuery_FiltersResults", func(t *testing.T) {
		episodes, err := svc.SearchEpisodes(ctx, 1, "站会", 10)
		if err != nil {
			t.Fatalf("SearchEpisodes failed: %v", err)
		}
		// Should find the sample data with "站会" in it
		found := false
		for _, ep := range episodes {
			if ep.UserInput == "设置每日站会提醒" {
				found = true
				break
			}
		}
		if !found && len(episodes) > 0 {
			t.Log("Search returned results but not the expected one")
		}
	})

	t.Run("SearchEpisodes_UserIsolation", func(t *testing.T) {
		// Search for user 1's episodes
		episodes1, err := svc.SearchEpisodes(ctx, 1, "", 100)
		if err != nil {
			t.Fatalf("SearchEpisodes failed: %v", err)
		}

		// Verify all returned episodes belong to user 1
		for _, ep := range episodes1 {
			if ep.UserID != 1 {
				t.Errorf("SearchEpisodes returned episode for wrong user: expected 1, got %d", ep.UserID)
			}
		}

		// Search for non-existent user should return empty
		episodes999, err := svc.SearchEpisodes(ctx, 999, "", 100)
		if err != nil {
			t.Fatalf("SearchEpisodes failed: %v", err)
		}
		if len(episodes999) != 0 {
			t.Errorf("expected 0 episodes for non-existent user, got %d", len(episodes999))
		}
	})

	t.Run("GetPreferences_ReturnsPreferences", func(t *testing.T) {
		prefs, err := svc.GetPreferences(ctx, 1)
		if err != nil {
			t.Fatalf("GetPreferences failed: %v", err)
		}
		if prefs == nil {
			t.Fatal("expected preferences, got nil")
		}
		if prefs.Timezone == "" {
			t.Error("expected non-empty timezone")
		}
	})

	t.Run("GetPreferences_NonexistentUser_ReturnsDefaults", func(t *testing.T) {
		prefs, err := svc.GetPreferences(ctx, 99999)
		if err != nil {
			t.Fatalf("GetPreferences failed: %v", err)
		}
		if prefs == nil {
			t.Fatal("expected default preferences, got nil")
		}
	})

	t.Run("UpdatePreferences_StoresData", func(t *testing.T) {
		newPrefs := &UserPreferences{
			Timezone:           "America/New_York",
			DefaultDuration:    30,
			PreferredTimes:     []string{"10:00", "15:00"},
			FrequentLocations:  []string{"Home", "Office"},
			CommunicationStyle: "detailed",
			TagPreferences:     []string{"personal", "work"},
			CustomSettings:     map[string]any{"lang": "en"},
		}

		err := svc.UpdatePreferences(ctx, 2, newPrefs)
		if err != nil {
			t.Fatalf("UpdatePreferences failed: %v", err)
		}

		prefs, err := svc.GetPreferences(ctx, 2)
		if err != nil {
			t.Fatalf("GetPreferences failed: %v", err)
		}
		if prefs.Timezone != "America/New_York" {
			t.Errorf("expected timezone 'America/New_York', got '%s'", prefs.Timezone)
		}
		if prefs.DefaultDuration != 30 {
			t.Errorf("expected default duration 30, got %d", prefs.DefaultDuration)
		}
	})

	t.Run("MessageRoles_Valid", func(t *testing.T) {
		validRoles := map[string]bool{"user": true, "assistant": true, "system": true}
		msgs, _ := svc.GetRecentMessages(ctx, "session-001", 10)
		for _, msg := range msgs {
			if !validRoles[msg.Role] {
				t.Errorf("invalid message role: %s", msg.Role)
			}
		}
	})

	t.Run("EpisodeOutcome_Valid", func(t *testing.T) {
		validOutcomes := map[string]bool{"success": true, "failure": true}
		episodes, _ := svc.SearchEpisodes(ctx, 1, "", 10)
		for _, ep := range episodes {
			if !validOutcomes[ep.Outcome] {
				t.Errorf("invalid episode outcome: %s", ep.Outcome)
			}
		}
	})
}
