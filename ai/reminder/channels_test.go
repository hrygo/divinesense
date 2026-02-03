package reminder

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNotificationDispatcher_Register(t *testing.T) {
	dispatcher := NewNotificationDispatcher()

	store := NewMemoryAppNotificationStore()
	appSender := NewAppNotificationSender(store)

	dispatcher.Register(ChannelApp, appSender)

	// Should be able to send now
	err := dispatcher.Send(context.Background(), 1, ChannelApp, "Test message")
	require.NoError(t, err)
}

func TestNotificationDispatcher_Send_UnregisteredChannel(t *testing.T) {
	dispatcher := NewNotificationDispatcher()

	err := dispatcher.Send(context.Background(), 1, ChannelEmail, "Test message")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not registered")
}

func TestNotificationDispatcher_SendWithMetadata(t *testing.T) {
	dispatcher := NewNotificationDispatcher()
	store := NewMemoryAppNotificationStore()
	appSender := NewAppNotificationSender(store)
	dispatcher.Register(ChannelApp, appSender)

	metadata := map[string]any{
		"schedule_id": "sched-001",
		"priority":    "high",
	}

	err := dispatcher.SendWithMetadata(context.Background(), 1, ChannelApp, "Test", metadata)
	require.NoError(t, err)

	notifications := store.GetAll()
	require.Len(t, notifications, 1)
	assert.Equal(t, "sched-001", notifications[0].Metadata["schedule_id"])
}

func TestNotificationDispatcher_Broadcast(t *testing.T) {
	dispatcher := NewNotificationDispatcher()

	store := NewMemoryAppNotificationStore()
	appSender := NewAppNotificationSender(store)
	dispatcher.Register(ChannelApp, appSender)

	resolver := NewMockEmailResolver()
	resolver.SetEmail(1, "user@example.com")
	emailSender := NewEmailSender(EmailConfig{}, resolver)
	dispatcher.Register(ChannelEmail, emailSender)

	errors := dispatcher.Broadcast(context.Background(), 1, "Broadcast message")
	assert.Empty(t, errors)
}

func TestAppNotificationSender(t *testing.T) {
	store := NewMemoryAppNotificationStore()
	sender := NewAppNotificationSender(store)

	assert.Equal(t, "app", sender.Name())

	err := sender.Send(context.Background(), 1, "Test notification", nil)
	require.NoError(t, err)

	notifications := store.GetAll()
	require.Len(t, notifications, 1)
	assert.Equal(t, int32(1), notifications[0].UserID)
	assert.Equal(t, "Test notification", notifications[0].Message)
	assert.Equal(t, "reminder", notifications[0].Type)
	assert.False(t, notifications[0].Read)
}

func TestAppNotificationSender_WithMetadata(t *testing.T) {
	store := NewMemoryAppNotificationStore()
	sender := NewAppNotificationSender(store)

	metadata := map[string]any{
		"action_url": "/schedule/123",
	}

	err := sender.Send(context.Background(), 1, "Click to view", metadata)
	require.NoError(t, err)

	notifications := store.GetAll()
	require.Len(t, notifications, 1)
	assert.Equal(t, "/schedule/123", notifications[0].Metadata["action_url"])
}

func TestEmailSender(t *testing.T) {
	resolver := NewMockEmailResolver()
	resolver.SetEmail(1, "user@example.com")

	sender := NewEmailSender(EmailConfig{
		SMTPHost:    "smtp.example.com",
		SMTPPort:    587,
		FromAddress: "noreply@example.com",
	}, resolver)

	assert.Equal(t, "email", sender.Name())

	err := sender.Send(context.Background(), 1, "Email notification", nil)
	require.NoError(t, err)
}

func TestEmailSender_NoEmail(t *testing.T) {
	resolver := NewMockEmailResolver()
	// User 1 has no email configured

	sender := NewEmailSender(EmailConfig{}, resolver)

	err := sender.Send(context.Background(), 1, "Email notification", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no email configured")
}

func TestWebhookSender(t *testing.T) {
	// Create test server
	var receivedPayload WebhookPayload
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "test-secret", r.Header.Get("X-Webhook-Secret"))

		err := json.NewDecoder(r.Body).Decode(&receivedPayload)
		assert.NoError(t, err)

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	sender := NewWebhookSender(WebhookConfig{
		URL:     server.URL,
		Secret:  "test-secret",
		Timeout: 5 * time.Second,
	})

	assert.Equal(t, "webhook", sender.Name())

	metadata := map[string]any{"key": "value"}
	err := sender.Send(context.Background(), 1, "Webhook test", metadata)
	require.NoError(t, err)

	assert.Equal(t, "reminder.triggered", receivedPayload.Event)
	assert.Equal(t, int32(1), receivedPayload.UserID)
	assert.Equal(t, "Webhook test", receivedPayload.Message)
	assert.Equal(t, "value", receivedPayload.Metadata["key"])
}

func TestWebhookSender_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("Internal error"))
	}))
	defer server.Close()

	sender := NewWebhookSender(WebhookConfig{
		URL:     server.URL,
		Timeout: 5 * time.Second,
	})

	err := sender.Send(context.Background(), 1, "Test", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "status 500")
}

func TestWebhookSender_CustomHeaders(t *testing.T) {
	var receivedHeaders http.Header
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHeaders = r.Header
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	sender := NewWebhookSender(WebhookConfig{
		URL: server.URL,
		Headers: map[string]string{
			"X-Custom-Header": "custom-value",
			"Authorization":   "Bearer token123",
		},
	})

	err := sender.Send(context.Background(), 1, "Test", nil)
	require.NoError(t, err)

	assert.Equal(t, "custom-value", receivedHeaders.Get("X-Custom-Header"))
	assert.Equal(t, "Bearer token123", receivedHeaders.Get("Authorization"))
}

func TestWebhookSender_Timeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	sender := NewWebhookSender(WebhookConfig{
		URL:     server.URL,
		Timeout: 50 * time.Millisecond,
	})

	err := sender.Send(context.Background(), 1, "Test", nil)
	assert.Error(t, err)
}

func TestMemoryAppNotificationStore(t *testing.T) {
	store := NewMemoryAppNotificationStore()

	notification := &AppNotification{
		ID:        "notif-001",
		UserID:    1,
		Type:      "reminder",
		Message:   "Test",
		CreatedAt: time.Now(),
	}

	err := store.CreateNotification(context.Background(), notification)
	require.NoError(t, err)

	all := store.GetAll()
	require.Len(t, all, 1)
	assert.Equal(t, "notif-001", all[0].ID)
}

func TestMockEmailResolver(t *testing.T) {
	resolver := NewMockEmailResolver()

	// No email initially
	email, err := resolver.GetEmail(context.Background(), 1)
	require.NoError(t, err)
	assert.Empty(t, email)

	// Set email
	resolver.SetEmail(1, "user@example.com")

	email, err = resolver.GetEmail(context.Background(), 1)
	require.NoError(t, err)
	assert.Equal(t, "user@example.com", email)
}

func TestDispatcherAsNotifier(t *testing.T) {
	// Verify dispatcher implements Notifier interface
	dispatcher := NewNotificationDispatcher()

	store := NewMemoryAppNotificationStore()
	appSender := NewAppNotificationSender(store)
	dispatcher.Register(ChannelApp, appSender)

	// Use as Notifier
	var notifier Notifier = dispatcher

	err := notifier.Send(context.Background(), 1, ChannelApp, "Test")
	require.NoError(t, err)
}

func BenchmarkAppNotificationSender(b *testing.B) {
	store := NewMemoryAppNotificationStore()
	sender := NewAppNotificationSender(store)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = sender.Send(ctx, 1, "Benchmark notification", nil)
	}
}

func BenchmarkWebhookSender(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	sender := NewWebhookSender(WebhookConfig{
		URL:     server.URL,
		Timeout: 5 * time.Second,
	})
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = sender.Send(ctx, 1, "Benchmark", nil)
	}
}
