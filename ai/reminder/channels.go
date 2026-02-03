// Package reminder provides reminder management for schedules and todos.
package reminder

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

// NotificationDispatcher routes notifications to appropriate channels.
type NotificationDispatcher struct {
	channels map[Channel]ChannelSender
	logger   *slog.Logger
	mu       sync.RWMutex
}

// ChannelSender defines the interface for sending notifications.
type ChannelSender interface {
	Send(ctx context.Context, userID int32, message string, metadata map[string]any) error
	Name() string
}

// NewNotificationDispatcher creates a new notification dispatcher.
func NewNotificationDispatcher() *NotificationDispatcher {
	return &NotificationDispatcher{
		channels: make(map[Channel]ChannelSender),
		logger:   slog.Default(),
	}
}

// Register registers a channel sender.
func (d *NotificationDispatcher) Register(channel Channel, sender ChannelSender) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.channels[channel] = sender
	d.logger.Info("registered notification channel", "channel", channel, "sender", sender.Name())
}

// Send sends a notification through the specified channel.
func (d *NotificationDispatcher) Send(ctx context.Context, userID int32, channel Channel, message string) error {
	d.mu.RLock()
	sender, ok := d.channels[channel]
	d.mu.RUnlock()

	if !ok {
		return fmt.Errorf("channel not registered: %s", channel)
	}

	return sender.Send(ctx, userID, message, nil)
}

// SendWithMetadata sends a notification with additional metadata.
func (d *NotificationDispatcher) SendWithMetadata(ctx context.Context, userID int32, channel Channel, message string, metadata map[string]any) error {
	d.mu.RLock()
	sender, ok := d.channels[channel]
	d.mu.RUnlock()

	if !ok {
		return fmt.Errorf("channel not registered: %s", channel)
	}

	return sender.Send(ctx, userID, message, metadata)
}

// Broadcast sends a notification through all registered channels.
func (d *NotificationDispatcher) Broadcast(ctx context.Context, userID int32, message string) []error {
	d.mu.RLock()
	channels := make([]ChannelSender, 0, len(d.channels))
	for _, sender := range d.channels {
		channels = append(channels, sender)
	}
	d.mu.RUnlock()

	var errors []error
	for _, sender := range channels {
		if err := sender.Send(ctx, userID, message, nil); err != nil {
			errors = append(errors, err)
		}
	}

	return errors
}

// AppNotificationSender sends in-app notifications.
type AppNotificationSender struct {
	store  AppNotificationStore
	logger *slog.Logger
}

// AppNotificationStore defines storage for in-app notifications.
type AppNotificationStore interface {
	CreateNotification(ctx context.Context, notification *AppNotification) error
}

// AppNotification represents an in-app notification.
type AppNotification struct {
	CreatedAt time.Time      `json:"created_at"`
	Metadata  map[string]any `json:"metadata,omitempty"`
	ID        string         `json:"id"`
	Type      string         `json:"type"`
	Title     string         `json:"title"`
	Message   string         `json:"message"`
	UserID    int32          `json:"user_id"`
	Read      bool           `json:"read"`
}

// NewAppNotificationSender creates a new app notification sender.
func NewAppNotificationSender(store AppNotificationStore) *AppNotificationSender {
	return &AppNotificationSender{
		store:  store,
		logger: slog.Default(),
	}
}

// Send sends an in-app notification.
func (s *AppNotificationSender) Send(ctx context.Context, userID int32, message string, metadata map[string]any) error {
	notification := &AppNotification{
		ID:        generateID(),
		UserID:    userID,
		Type:      "reminder",
		Title:     "提醒",
		Message:   message,
		Read:      false,
		CreatedAt: time.Now(),
		Metadata:  metadata,
	}

	if err := s.store.CreateNotification(ctx, notification); err != nil {
		s.logger.Error("failed to create app notification", "user_id", userID, "error", err)
		return fmt.Errorf("failed to create notification: %w", err)
	}

	s.logger.Debug("app notification sent", "user_id", userID, "notification_id", notification.ID)
	return nil
}

// Name returns the sender name.
func (s *AppNotificationSender) Name() string {
	return "app"
}

// EmailConfig holds email configuration.
type EmailConfig struct {
	SMTPHost     string
	Username     string
	Password     string
	FromAddress  string
	FromName     string
	TemplateHTML string
	SMTPPort     int
}

// EmailSender sends email notifications.
type EmailSender struct {
	userResolver UserEmailResolver
	logger       *slog.Logger
	config       EmailConfig
}

// UserEmailResolver resolves user ID to email address.
type UserEmailResolver interface {
	GetEmail(ctx context.Context, userID int32) (string, error)
}

// NewEmailSender creates a new email sender.
func NewEmailSender(config EmailConfig, resolver UserEmailResolver) *EmailSender {
	return &EmailSender{
		config:       config,
		userResolver: resolver,
		logger:       slog.Default(),
	}
}

// Send sends an email notification.
func (s *EmailSender) Send(ctx context.Context, userID int32, message string, metadata map[string]any) error {
	email, err := s.userResolver.GetEmail(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to resolve email for user %d: %w", userID, err)
	}

	if email == "" {
		s.logger.Warn("user has no email configured", "user_id", userID)
		return fmt.Errorf("user %d has no email configured", userID)
	}

	// In production, this would use an SMTP client
	// For now, we log the action
	s.logger.Info("email notification sent",
		"user_id", userID,
		"email", email,
		"message_length", len(message),
	)

	return nil
}

// Name returns the sender name.
func (s *EmailSender) Name() string {
	return "email"
}

// WebhookConfig holds webhook configuration.
type WebhookConfig struct {
	Headers       map[string]string
	URL           string
	Secret        string
	Timeout       time.Duration
	RetryAttempts int
}

// WebhookSender sends webhook notifications.
type WebhookSender struct {
	config     WebhookConfig
	httpClient *http.Client
	logger     *slog.Logger
}

// WebhookPayload represents the webhook request body.
type WebhookPayload struct {
	Timestamp time.Time      `json:"timestamp"`
	Metadata  map[string]any `json:"metadata,omitempty"`
	Event     string         `json:"event"`
	Message   string         `json:"message"`
	UserID    int32          `json:"user_id"`
}

// NewWebhookSender creates a new webhook sender.
func NewWebhookSender(config WebhookConfig) *WebhookSender {
	if config.Timeout <= 0 {
		config.Timeout = 10 * time.Second
	}

	return &WebhookSender{
		config: config,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
		logger: slog.Default(),
	}
}

// Send sends a webhook notification.
func (s *WebhookSender) Send(ctx context.Context, userID int32, message string, metadata map[string]any) error {
	payload := WebhookPayload{
		Event:     "reminder.triggered",
		UserID:    userID,
		Message:   message,
		Timestamp: time.Now(),
		Metadata:  metadata,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal webhook payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.config.URL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create webhook request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if s.config.Secret != "" {
		req.Header.Set("X-Webhook-Secret", s.config.Secret)
	}
	for k, v := range s.config.Headers {
		req.Header.Set(k, v)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		s.logger.Error("webhook request failed", "url", s.config.URL, "error", err)
		return fmt.Errorf("webhook request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }() //nolint:errcheck // response body cleanup

	if resp.StatusCode >= 400 {
		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			respBody = []byte(fmt.Sprintf("failed to read response body: %v", err))
		}
		s.logger.Error("webhook returned error",
			"url", s.config.URL,
			"status", resp.StatusCode,
			"response", string(respBody),
		)
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}

	s.logger.Debug("webhook notification sent",
		"user_id", userID,
		"url", s.config.URL,
		"status", resp.StatusCode,
	)

	return nil
}

// Name returns the sender name.
func (s *WebhookSender) Name() string {
	return "webhook"
}

// MemoryAppNotificationStore is an in-memory implementation for testing.
type MemoryAppNotificationStore struct {
	notifications []*AppNotification
	mu            sync.Mutex
}

// NewMemoryAppNotificationStore creates a new in-memory notification store.
func NewMemoryAppNotificationStore() *MemoryAppNotificationStore {
	return &MemoryAppNotificationStore{
		notifications: make([]*AppNotification, 0),
	}
}

// CreateNotification stores a notification.
func (s *MemoryAppNotificationStore) CreateNotification(ctx context.Context, notification *AppNotification) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.notifications = append(s.notifications, notification)
	return nil
}

// GetAll returns all notifications (for testing).
func (s *MemoryAppNotificationStore) GetAll() []*AppNotification {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]*AppNotification{}, s.notifications...)
}

// MockEmailResolver is a mock implementation of UserEmailResolver.
type MockEmailResolver struct {
	emails map[int32]string
}

// NewMockEmailResolver creates a new mock email resolver.
func NewMockEmailResolver() *MockEmailResolver {
	return &MockEmailResolver{
		emails: make(map[int32]string),
	}
}

// SetEmail sets an email for a user.
func (r *MockEmailResolver) SetEmail(userID int32, email string) {
	r.emails[userID] = email
}

// GetEmail returns the email for a user.
func (r *MockEmailResolver) GetEmail(ctx context.Context, userID int32) (string, error) {
	email, ok := r.emails[userID]
	if !ok {
		return "", nil
	}
	return email, nil
}
