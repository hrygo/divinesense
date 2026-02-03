// Package reminder provides reminder management for schedules and todos.
package reminder

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// ReminderType defines the type of reminder.
type ReminderType string

const (
	ReminderTypeSchedule ReminderType = "schedule"
	ReminderTypeTodo     ReminderType = "todo"
	ReminderTypeSmart    ReminderType = "smart"
)

// ReminderStatus defines the status of a reminder.
type ReminderStatus string

const (
	StatusPending   ReminderStatus = "pending"
	StatusSent      ReminderStatus = "sent"
	StatusCancelled ReminderStatus = "cancelled"
	StatusFailed    ReminderStatus = "failed"
)

// Channel defines notification channel types.
type Channel string

const (
	ChannelApp     Channel = "app"
	ChannelEmail   Channel = "email"
	ChannelWebhook Channel = "webhook"
)

// Reminder represents a reminder entity.
type Reminder struct {
	TriggerAt time.Time      `json:"trigger_at"`
	CreatedAt time.Time      `json:"created_at"`
	SentAt    *time.Time     `json:"sent_at,omitempty"`
	Metadata  map[string]any `json:"metadata,omitempty"`
	ID        string         `json:"id"`
	Type      ReminderType   `json:"type"`
	TargetID  string         `json:"target_id"`
	Message   string         `json:"message"`
	Status    ReminderStatus `json:"status"`
	Channels  []Channel      `json:"channels"`
	UserID    int32          `json:"user_id"`
}

// CreateReminderRequest represents a request to create a reminder.
type CreateReminderRequest struct {
	TriggerAt   time.Time
	Type        ReminderType
	TargetID    string
	Message     string
	Channels    []Channel
	LeadMinutes int
	UserID      int32
}

// ScheduleInfo contains schedule information for reminder creation.
type ScheduleInfo struct {
	ID        string
	Title     string
	StartTime time.Time
	Location  string
}

// ReminderStore defines the storage interface for reminders.
type ReminderStore interface {
	Create(ctx context.Context, reminder *Reminder) error
	Get(ctx context.Context, id string) (*Reminder, error)
	GetByTarget(ctx context.Context, targetID string) ([]*Reminder, error)
	GetDueReminders(ctx context.Context, before time.Time) ([]*Reminder, error)
	GetByUser(ctx context.Context, userID int32, status ReminderStatus) ([]*Reminder, error)
	Update(ctx context.Context, reminder *Reminder) error
	Delete(ctx context.Context, id string) error
	MarkSent(ctx context.Context, id string) error
	MarkFailed(ctx context.Context, id string, reason string) error
}

// Notifier defines the notification interface.
type Notifier interface {
	Send(ctx context.Context, userID int32, channel Channel, message string) error
}

// Service provides reminder management functionality.
type Service struct {
	store           ReminderStore
	notifier        Notifier
	defaultChannels []Channel
	defaultLead     int
	mu              sync.RWMutex
}

// NewService creates a new reminder service.
func NewService(store ReminderStore, notifier Notifier) *Service {
	return &Service{
		store:           store,
		notifier:        notifier,
		defaultLead:     15,
		defaultChannels: []Channel{ChannelApp},
	}
}

// CreateForSchedule creates a reminder for a schedule.
func (s *Service) CreateForSchedule(ctx context.Context, userID int32, schedule *ScheduleInfo, leadMinutes int) (*Reminder, error) {
	if leadMinutes <= 0 {
		leadMinutes = s.defaultLead
	}

	triggerAt := schedule.StartTime.Add(-time.Duration(leadMinutes) * time.Minute)

	// Don't create reminder if trigger time is in the past
	if triggerAt.Before(time.Now()) {
		return nil, fmt.Errorf("reminder trigger time is in the past")
	}

	message := fmt.Sprintf("您有一个日程「%s」将在 %d 分钟后开始", schedule.Title, leadMinutes)
	if schedule.Location != "" {
		message += fmt.Sprintf("，地点：%s", schedule.Location)
	}

	reminder := &Reminder{
		ID:        generateID(),
		UserID:    userID,
		Type:      ReminderTypeSchedule,
		TargetID:  schedule.ID,
		TriggerAt: triggerAt,
		Message:   message,
		Channels:  s.defaultChannels,
		Status:    StatusPending,
		CreatedAt: time.Now(),
		Metadata: map[string]any{
			"schedule_title": schedule.Title,
			"lead_minutes":   leadMinutes,
		},
	}

	if err := s.store.Create(ctx, reminder); err != nil {
		return nil, fmt.Errorf("failed to create reminder: %w", err)
	}

	return reminder, nil
}

// CreateCustom creates a custom reminder.
func (s *Service) CreateCustom(ctx context.Context, req *CreateReminderRequest) (*Reminder, error) {
	if req.TriggerAt.Before(time.Now()) {
		return nil, fmt.Errorf("trigger time cannot be in the past")
	}

	channels := req.Channels
	if len(channels) == 0 {
		channels = s.defaultChannels
	}

	reminder := &Reminder{
		ID:        generateID(),
		UserID:    req.UserID,
		Type:      req.Type,
		TargetID:  req.TargetID,
		TriggerAt: req.TriggerAt,
		Message:   req.Message,
		Channels:  channels,
		Status:    StatusPending,
		CreatedAt: time.Now(),
	}

	if err := s.store.Create(ctx, reminder); err != nil {
		return nil, fmt.Errorf("failed to create reminder: %w", err)
	}

	return reminder, nil
}

// Cancel cancels a pending reminder.
func (s *Service) Cancel(ctx context.Context, id string) error {
	reminder, err := s.store.Get(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get reminder: %w", err)
	}

	if reminder.Status != StatusPending {
		return fmt.Errorf("cannot cancel reminder with status: %s", reminder.Status)
	}

	reminder.Status = StatusCancelled
	return s.store.Update(ctx, reminder)
}

// CancelByTarget cancels all pending reminders for a target.
func (s *Service) CancelByTarget(ctx context.Context, targetID string) error {
	reminders, err := s.store.GetByTarget(ctx, targetID)
	if err != nil {
		return fmt.Errorf("failed to get reminders: %w", err)
	}

	for _, r := range reminders {
		if r.Status == StatusPending {
			r.Status = StatusCancelled
			if err := s.store.Update(ctx, r); err != nil {
				return fmt.Errorf("failed to cancel reminder %s: %w", r.ID, err)
			}
		}
	}

	return nil
}

// ProcessDueReminders processes all reminders that are due.
func (s *Service) ProcessDueReminders(ctx context.Context) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	reminders, err := s.store.GetDueReminders(ctx, time.Now())
	if err != nil {
		return 0, fmt.Errorf("failed to get due reminders: %w", err)
	}

	processed := 0
	for _, r := range reminders {
		if err := s.sendReminder(ctx, r); err != nil {
			_ = s.store.MarkFailed(ctx, r.ID, err.Error())
			continue
		}

		if err := s.store.MarkSent(ctx, r.ID); err != nil {
			continue
		}
		processed++
	}

	return processed, nil
}

// sendReminder sends a reminder through all configured channels.
func (s *Service) sendReminder(ctx context.Context, r *Reminder) error {
	if s.notifier == nil {
		return fmt.Errorf("notifier not configured")
	}

	var lastErr error
	for _, channel := range r.Channels {
		if err := s.notifier.Send(ctx, r.UserID, channel, r.Message); err != nil {
			lastErr = err
			continue
		}
	}

	return lastErr
}

// GetUserReminders gets all reminders for a user.
func (s *Service) GetUserReminders(ctx context.Context, userID int32, status ReminderStatus) ([]*Reminder, error) {
	return s.store.GetByUser(ctx, userID, status)
}

// GetReminder gets a specific reminder.
func (s *Service) GetReminder(ctx context.Context, id string) (*Reminder, error) {
	return s.store.Get(ctx, id)
}

// SetDefaultLeadTime sets the default lead time for reminders.
func (s *Service) SetDefaultLeadTime(minutes int) {
	if minutes > 0 {
		s.mu.Lock()
		s.defaultLead = minutes
		s.mu.Unlock()
	}
}

// SetDefaultChannels sets the default notification channels.
func (s *Service) SetDefaultChannels(channels []Channel) {
	if len(channels) > 0 {
		s.mu.Lock()
		s.defaultChannels = channels
		s.mu.Unlock()
	}
}

// generateID creates a unique reminder ID.
func generateID() string {
	return uuid.New().String()[:12]
}
