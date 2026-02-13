// Package agent provides conversation context management for multi-turn dialogues.
// This module maintains state across conversation turns to enable handling
// of refinements like "change it to 3pm" without re-specifying the full context.
package agent

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"log/slog"
)

// MaxTurnsPerSession is the maximum number of conversation turns to keep in memory.
// This prevents unbounded memory growth in long conversations.
const MaxTurnsPerSession = 10

// ConversationContext maintains state across conversation turns.
type ConversationContext struct {
	CreatedAt time.Time
	UpdatedAt time.Time
	SessionID string
	Timezone  string
	Turns     []ConversationTurn
	mu        sync.RWMutex
	UserID    int32
	// RouteSticky: Intent stickiness for short confirmations (Issue #163)
	LastRouteType ChatRouteType // Last successful route type
	LastRouteTime time.Time     // When the last route was made
	// Extensions stores domain-specific state (e.g., schedule context).
	// Use type-safe getters/setters from domain packages.
	Extensions map[string]any
}

// ConversationTurn represents a single turn in the conversation.
type ConversationTurn struct {
	Timestamp   time.Time
	UserInput   string
	AgentOutput string
	ToolCalls   []ToolCallRecord
}

// ToolCallRecord records a tool invocation.
type ToolCallRecord struct {
	Timestamp time.Time
	Tool      string
	Input     string
	Output    string
	Duration  time.Duration
	Success   bool
}

// NewConversationContext creates a new conversation context.
func NewConversationContext(sessionID string, userID int32, timezone string) *ConversationContext {
	return &ConversationContext{
		SessionID:  sessionID,
		UserID:     userID,
		Timezone:   timezone,
		Turns:      make([]ConversationTurn, 0),
		Extensions: make(map[string]any),
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
}

// GetExtension retrieves a domain-specific extension by key.
func (c *ConversationContext) GetExtension(key string) any {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.Extensions[key]
}

// SetExtension stores a domain-specific extension by key.
func (c *ConversationContext) SetExtension(key string, value any) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.Extensions == nil {
		c.Extensions = make(map[string]any)
	}
	c.Extensions[key] = value
	c.UpdatedAt = time.Now()
}

// AddTurn adds a new turn to the conversation history.
func (c *ConversationContext) AddTurn(userInput, agentOutput string, toolCalls []ToolCallRecord) {
	c.mu.Lock()
	defer c.mu.Unlock()

	turn := ConversationTurn{
		UserInput:   userInput,
		AgentOutput: agentOutput,
		ToolCalls:   toolCalls,
		Timestamp:   time.Now(),
	}

	c.Turns = append(c.Turns, turn)
	c.UpdatedAt = time.Now()

	// Keep only last MaxTurnsPerSession turns to manage memory
	if len(c.Turns) > MaxTurnsPerSession {
		c.Turns = c.Turns[len(c.Turns)-MaxTurnsPerSession:]
	}
}

// SetLastRoute sets the last successful route for intent stickiness (Issue #163).
func (c *ConversationContext) SetLastRoute(routeType ChatRouteType) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.LastRouteType = routeType
	c.LastRouteTime = time.Now()
}

// GetLastRoute returns the last route type and whether it's within the sticky window.
func (c *ConversationContext) GetLastRoute() (ChatRouteType, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	// Sticky window: 5 minutes
	if c.LastRouteType != "" && time.Since(c.LastRouteTime) < 5*time.Minute {
		return c.LastRouteType, true
	}
	return "", false
}

// GetLastTurn returns a copy of the most recent conversation turn.
func (c *ConversationContext) GetLastTurn() *ConversationTurn {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if len(c.Turns) == 0 {
		return nil
	}

	// Return a copy, not a pointer to the slice element
	last := c.Turns[len(c.Turns)-1]
	return &last
}

// GetLastNTurns returns the last N conversation turns.
func (c *ConversationContext) GetLastNTurns(n int) []ConversationTurn {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if len(c.Turns) == 0 {
		return nil
	}

	start := 0
	if len(c.Turns) > n {
		start = len(c.Turns) - n
	}

	result := make([]ConversationTurn, len(c.Turns)-start)
	copy(result, c.Turns[start:])
	return result
}

// Clear resets the conversation context.
func (c *ConversationContext) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.Turns = make([]ConversationTurn, 0)
	c.Extensions = make(map[string]any)
	c.UpdatedAt = time.Now()
}

// GetSummary returns a summary of the conversation context.
func (c *ConversationContext) GetSummary() ContextSummary {
	c.mu.RLock()
	defer c.mu.RUnlock()

	summary := ContextSummary{
		SessionID: c.SessionID,
		UserID:    c.UserID,
		TurnCount: len(c.Turns),
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
	}

	return summary
}

// ToJSON exports the conversation context to JSON for persistence.
func (c *ConversationContext) ToJSON() (string, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	data, err := json.Marshal(c)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// ToHistoryPrompt converts the conversation history to a string format suitable for LLM context.
// It formats turns as "User: ...\nAssistant: ..." and optionally includes tool usage summaries.
func (c *ConversationContext) ToHistoryPrompt() string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if len(c.Turns) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("Conversation History:\n")

	for _, turn := range c.Turns {
		sb.WriteString(fmt.Sprintf("User: %s\n", turn.UserInput))

		// Add tool summary if available, to provide context on what the agent did
		if len(turn.ToolCalls) > 0 {
			// Simplified tool summary: "Assistant (Action: tool_name): [Success/Fail]"
			// We avoid dumping full JSON output to save tokens, but give a hint of action.
			toolsUsed := make([]string, 0, len(turn.ToolCalls))
			for _, tc := range turn.ToolCalls {
				status := "OK"
				if !tc.Success {
					status = "Failed"
				}
				toolsUsed = append(toolsUsed, fmt.Sprintf("%s (%s)", tc.Tool, status))
			}
			sb.WriteString(fmt.Sprintf("System: Agent used tools: %s\n", strings.Join(toolsUsed, ", ")))
		}

		sb.WriteString(fmt.Sprintf("Assistant: %s\n", turn.AgentOutput))
	}

	result := sb.String()
	slog.Debug("ToHistoryPrompt generated",
		"session_id", c.SessionID,
		"turn_count", len(c.Turns),
		"length", len(result))

	return result
}

// ContextSummary provides a quick overview of the context state.
type ContextSummary struct {
	CreatedAt time.Time
	UpdatedAt time.Time
	SessionID string
	TurnCount int
	UserID    int32
}

// ContextStore manages conversation contexts for multiple sessions.
type ContextStore struct {
	contexts map[string]*ConversationContext
	mu       sync.RWMutex
}

// NewContextStore creates a new context store.
func NewContextStore() *ContextStore {
	return &ContextStore{
		contexts: make(map[string]*ConversationContext),
	}
}

// GetOrCreate retrieves or creates a conversation context.
func (s *ContextStore) GetOrCreate(sessionID string, userID int32, timezone string) *ConversationContext {
	s.mu.Lock()
	defer s.mu.Unlock()

	if ctx, exists := s.contexts[sessionID]; exists {
		return ctx
	}

	ctx := NewConversationContext(sessionID, userID, timezone)
	s.contexts[sessionID] = ctx
	return ctx
}

// Get retrieves a conversation context if it exists.
func (s *ContextStore) Get(sessionID string) *ConversationContext {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.contexts[sessionID]
}

// Delete removes a conversation context.
func (s *ContextStore) Delete(sessionID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.contexts, sessionID)
}

// CleanupOld removes contexts older than the specified duration.
func (s *ContextStore) CleanupOld(maxAge time.Duration) int {
	s.mu.Lock()
	defer s.mu.Unlock()

	cutoff := time.Now().Add(-maxAge)
	deleted := 0

	for sessionID, ctx := range s.contexts {
		if ctx.UpdatedAt.Before(cutoff) {
			delete(s.contexts, sessionID)
			deleted++
		}
	}

	return deleted
}
