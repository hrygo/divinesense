package reminder

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// MemoryStore is an in-memory implementation of ReminderStore for testing.
type MemoryStore struct {
	reminders map[string]*Reminder
	mu        sync.RWMutex
}

// NewMemoryStore creates a new in-memory reminder store.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		reminders: make(map[string]*Reminder),
	}
}

// Create stores a new reminder.
func (s *MemoryStore) Create(ctx context.Context, reminder *Reminder) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.reminders[reminder.ID]; exists {
		return fmt.Errorf("reminder already exists: %s", reminder.ID)
	}

	s.reminders[reminder.ID] = reminder
	return nil
}

// Get retrieves a reminder by ID.
func (s *MemoryStore) Get(ctx context.Context, id string) (*Reminder, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	reminder, ok := s.reminders[id]
	if !ok {
		return nil, fmt.Errorf("reminder not found: %s", id)
	}

	return reminder, nil
}

// GetByTarget retrieves all reminders for a target.
func (s *MemoryStore) GetByTarget(ctx context.Context, targetID string) ([]*Reminder, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*Reminder
	for _, r := range s.reminders {
		if r.TargetID == targetID {
			result = append(result, r)
		}
	}

	return result, nil
}

// GetDueReminders retrieves all pending reminders due before the given time.
func (s *MemoryStore) GetDueReminders(ctx context.Context, before time.Time) ([]*Reminder, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*Reminder
	for _, r := range s.reminders {
		if r.Status == StatusPending && !r.TriggerAt.After(before) {
			result = append(result, r)
		}
	}

	return result, nil
}

// GetByUser retrieves all reminders for a user with optional status filter.
func (s *MemoryStore) GetByUser(ctx context.Context, userID int32, status ReminderStatus) ([]*Reminder, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*Reminder
	for _, r := range s.reminders {
		if r.UserID == userID {
			if status == "" || r.Status == status {
				result = append(result, r)
			}
		}
	}

	return result, nil
}

// Update updates an existing reminder.
func (s *MemoryStore) Update(ctx context.Context, reminder *Reminder) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.reminders[reminder.ID]; !exists {
		return fmt.Errorf("reminder not found: %s", reminder.ID)
	}

	s.reminders[reminder.ID] = reminder
	return nil
}

// Delete removes a reminder.
func (s *MemoryStore) Delete(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.reminders[id]; !exists {
		return fmt.Errorf("reminder not found: %s", id)
	}

	delete(s.reminders, id)
	return nil
}

// MarkSent marks a reminder as sent.
func (s *MemoryStore) MarkSent(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	reminder, ok := s.reminders[id]
	if !ok {
		return fmt.Errorf("reminder not found: %s", id)
	}

	now := time.Now()
	reminder.Status = StatusSent
	reminder.SentAt = &now
	return nil
}

// MarkFailed marks a reminder as failed.
func (s *MemoryStore) MarkFailed(ctx context.Context, id string, reason string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	reminder, ok := s.reminders[id]
	if !ok {
		return fmt.Errorf("reminder not found: %s", id)
	}

	reminder.Status = StatusFailed
	if reminder.Metadata == nil {
		reminder.Metadata = make(map[string]any)
	}
	reminder.Metadata["failure_reason"] = reason
	return nil
}

// MockNotifier is a mock implementation of Notifier for testing.
type MockNotifier struct {
	SentMessages []SentMessage
	ShouldFail   bool
	mu           sync.Mutex
}

// SentMessage represents a message that was sent.
type SentMessage struct {
	SentAt  time.Time
	Channel Channel
	Message string
	UserID  int32
}

// NewMockNotifier creates a new mock notifier.
func NewMockNotifier() *MockNotifier {
	return &MockNotifier{
		SentMessages: make([]SentMessage, 0),
	}
}

// Send records a sent message.
func (n *MockNotifier) Send(ctx context.Context, userID int32, channel Channel, message string) error {
	if n.ShouldFail {
		return fmt.Errorf("mock notifier failure")
	}

	n.mu.Lock()
	defer n.mu.Unlock()

	n.SentMessages = append(n.SentMessages, SentMessage{
		UserID:  userID,
		Channel: channel,
		Message: message,
		SentAt:  time.Now(),
	})

	return nil
}

// GetSentCount returns the number of messages sent.
func (n *MockNotifier) GetSentCount() int {
	n.mu.Lock()
	defer n.mu.Unlock()
	return len(n.SentMessages)
}

// Clear clears all sent messages.
func (n *MockNotifier) Clear() {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.SentMessages = make([]SentMessage, 0)
}
