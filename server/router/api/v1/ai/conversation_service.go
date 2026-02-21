package ai

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/hrygo/divinesense/store"
	"github.com/lithammer/shortuuid/v4"
)

// ChatEvent represents a chat event that can be processed by listeners.
type ChatEvent struct {
	Type               ChatEventType
	AgentType          AgentType
	MessageID          string
	UserMessage        string
	AssistantResponse  string
	SeparatorContent   string
	Timestamp          int64
	UserID             int32
	ConversationID     int32
	IsTempConversation bool
}

// ChatEventType represents the type of chat event.
type ChatEventType string

const (
	// EventConversationStart is fired when a conversation should be created/retrieved.
	EventConversationStart ChatEventType = "conversation_start"
	// EventUserMessage is fired when a user sends a message.
	EventUserMessage ChatEventType = "user_message"
	// EventAssistantResponse is fired when an assistant responds.
	EventAssistantResponse ChatEventType = "assistant_response"
	// EventSeparator is fired when a separator (---) is sent.
	EventSeparator ChatEventType = "separator"
	// EventBlockCompleted is fired when a block is completed.
	EventBlockCompleted ChatEventType = "block_completed"
)

// ChatEventListener is a function that processes chat events.
//
// IMPORTANT: Listeners MUST respect context cancellation.
// The context passed to listeners has a timeout (default 5s).
// Listeners should check ctx.Done() periodically in long-running operations.
// Failure to respect context will result in the listener continuing to run
// in the background after timeout, which is a resource leak.
//
// Example:
//
//	func myListener(ctx context.Context, event *ChatEvent) (interface{}, error) {
//		// Check context before expensive operation
//		select {
//		case <-ctx.Done():
//			return nil, ctx.Err()
//		default:
//		}
//		// Do work...
//		return result, nil
//	}
type ChatEventListener func(ctx context.Context, event *ChatEvent) (interface{}, error)

// EventBus manages chat event listeners.
//
// Listeners are invoked concurrently with a per-listener timeout.
// Results are collected and returned as a map indexed by listener index.
type EventBus struct {
	listeners map[ChatEventType][]ChatEventListener
	mu        sync.RWMutex
	timeout   time.Duration
}

// NewEventBus creates a new event bus with configurable timeout.
func NewEventBus() *EventBus {
	return &EventBus{
		listeners: make(map[ChatEventType][]ChatEventListener),
		timeout:   5 * time.Second, // Default timeout per listener
	}
}

// SetTimeout sets the timeout for event listeners.
func (b *EventBus) SetTimeout(d time.Duration) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.timeout = d
}

// Subscribe registers a listener for a specific event type.
func (b *EventBus) Subscribe(eventType ChatEventType, listener ChatEventListener) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.listeners[eventType] = append(b.listeners[eventType], listener)
}

// Publish emits an event to all registered listeners.
//
// Listeners are executed concurrently, each with its own timeout context.
// Returns a map of listener results indexed by listener index.
// For conversation_start event, it returns the conversation ID.
//
// If any listener returns an error, the first error is returned,
// but all listeners are still executed (fire-and-forget semantics).
func (b *EventBus) Publish(ctx context.Context, event *ChatEvent) (map[int]interface{}, error) {
	// Get listeners for this event type
	b.mu.RLock()
	listeners := make([]ChatEventListener, len(b.listeners[event.Type]))
	copy(listeners, b.listeners[event.Type])
	b.mu.RUnlock()

	if len(listeners) == 0 {
		return nil, nil
	}

	results := make(map[int]interface{})
	var wg sync.WaitGroup
	var resultsMu sync.Mutex
	var firstErr error
	var errOnce sync.Once

	for i, listener := range listeners {
		wg.Add(1)
		go func(index int, l ChatEventListener) {
			defer wg.Done()

			// Panic recovery: prevent one listener's panic from affecting others
			defer func() {
				if r := recover(); r != nil {
					slog.Default().Error("Event listener panic",
						"event_type", event.Type,
						"listener_index", index,
						"panic", r,
					)
					errOnce.Do(func() { firstErr = fmt.Errorf("listener panic: %v", r) })
				}
			}()

			// Create timeout context for this listener
			listenerCtx, cancel := context.WithTimeout(ctx, b.timeout)
			defer cancel()

			// Execute listener directly (no nested goroutine)
			// The listener MUST respect listenerCtx cancellation
			result, err := l(listenerCtx, event)

			// Check if timeout occurred (listener ran too long)
			if listenerCtx.Err() == context.DeadlineExceeded {
				slog.Default().Warn("Event listener timeout, discarding partial result",
					"event_type", event.Type,
					"listener_index", index,
					"timeout", b.timeout,
				)
				errOnce.Do(func() { firstErr = fmt.Errorf("listener timeout") })
				// Do NOT store partial results - timeout means operation did not complete
				return
			}

			// Check for other context errors (cancellation)
			if listenerCtx.Err() != nil {
				slog.Default().Warn("Event listener context error",
					"event_type", event.Type,
					"listener_index", index,
					"error", listenerCtx.Err(),
				)
				errOnce.Do(func() { firstErr = listenerCtx.Err() })
				return
			}

			// Handle listener errors
			if err != nil {
				slog.Default().Warn("Event listener failed",
					"event_type", event.Type,
					"listener_index", index,
					"error", err,
				)
				errOnce.Do(func() { firstErr = err })
				return
			}

			// Store successful result
			if result != nil {
				resultsMu.Lock()
				results[index] = result
				resultsMu.Unlock()
			}
		}(i, listener)
	}

	wg.Wait()
	return results, firstErr
}

// ConversationService handles conversation persistence independently.
// It listens to chat events and saves conversations/messages to the database.
type ConversationService struct {
	store ConversationStore
}

// NewConversationService creates a new conversation service.
func NewConversationService(store ConversationStore) *ConversationService {
	return &ConversationService{
		store: store,
	}
}

// Subscribe registers event listeners for conversation persistence.
//
// NOTE: Message persistence is now handled by BlockManager in the main chat flow.
// This service only handles conversation lifecycle events (start/create).
func (s *ConversationService) Subscribe(bus *EventBus) {
	bus.Subscribe(EventConversationStart, s.handleConversationStart)
	// Message events removed: BlockManager now handles all message persistence
}

// handleConversationStart ensures a conversation exists for the chat.
// Returns the conversation ID.
//
// Note: Fixed conversation mechanism was removed as it was never used.
// The frontend always creates conversations via CreateAIConversation API first,
// then passes the conversation ID to Chat API.
func (s *ConversationService) handleConversationStart(ctx context.Context, event *ChatEvent) (interface{}, error) {
	if event.ConversationID != 0 {
		// Conversation already specified, just update timestamp
		now := time.Now().UnixMilli()
		_, err := s.store.UpdateAIConversation(ctx, &store.UpdateAIConversation{
			ID:        event.ConversationID,
			UpdatedTs: &now,
		})
		if err != nil {
			slog.Default().Warn("Failed to update conversation timestamp",
				"conversation_id", event.ConversationID,
				"error", err,
			)
		}
		return event.ConversationID, nil
	}

	// Skip conversation creation for temp conversations
	if event.IsTempConversation {
		return int32(0), nil
	}

	// Create new conversation
	id, err := s.createConversation(ctx, event)

	if err != nil {
		slog.Default().Error("Failed to create conversation",
			"user_id", event.UserID,
			"agent_type", event.AgentType,
			"error", err,
		)
		return nil, err
	}

	return id, nil
}

// Message persistence removed: BlockManager now handles all message persistence
// in the main chat flow (handler.go).
// The following methods were removed:
// - handleUserMessage
// - handleAssistantResponse
// - handleSeparator

// createConversation creates a new conversation.
func (s *ConversationService) createConversation(ctx context.Context, event *ChatEvent) (int32, error) {
	title := s.generateTitle()
	conversation, err := s.store.CreateAIConversation(ctx, &store.AIConversation{
		UID:         shortuuid.New(),
		CreatorID:   event.UserID,
		Title:       title,
		TitleSource: store.TitleSourceDefault, // Explicitly set default title source
		ParrotID:    event.AgentType.String(),
		CreatedTs:   event.Timestamp,
		UpdatedTs:   event.Timestamp,
		RowStatus:   store.Normal,
	})
	if err != nil {
		return 0, fmt.Errorf("create conversation: %w", err)
	}
	return conversation.ID, nil
}

// generateTitle generates a title for a new conversation.
// Returns a title key that the frontend should localize and handle numbering.
// The numbering is handled by the frontend to avoid expensive database queries.
func (s *ConversationService) generateTitle() string {
	// Return a simple title key; the frontend will handle display numbering
	// based on the actual list of conversations it receives.
	return "chat.new"
}

// ConversationStore is the interface needed for conversation persistence.
//
// NOTE: Message persistence is now handled by BlockManager in the main chat flow.
// This interface only handles conversation lifecycle (create/update/list).
type ConversationStore interface {
	CreateAIConversation(ctx context.Context, create *store.AIConversation) (*store.AIConversation, error)
	ListAIConversations(ctx context.Context, find *store.FindAIConversation) ([]*store.AIConversation, error)
	UpdateAIConversation(ctx context.Context, update *store.UpdateAIConversation) (*store.AIConversation, error)
}
