package context

import (
	"context"
)

// OrchestratorContextKey is the key type for orchestrator context values.
// Using a custom type prevents collisions with other packages.
type OrchestratorContextKey string

const (
	// KeyUserID stores the user ID for orchestrator execution.
	KeyUserID OrchestratorContextKey = "orchestrator_user_id"

	// KeyConversationID stores the conversation ID for orchestrator execution.
	KeyConversationID OrchestratorContextKey = "orchestrator_conversation_id"

	// KeyBlockID stores the current block ID for orchestrator execution.
	KeyBlockID OrchestratorContextKey = "orchestrator_block_id"

	// KeyHistory stores the conversation history for orchestrator execution.
	KeyHistory OrchestratorContextKey = "orchestrator_history"

	// KeySessionID stores the session ID for orchestrator execution.
	KeySessionID OrchestratorContextKey = "orchestrator_session_id"

	// KeyAgentType stores the agent type for orchestrator execution.
	KeyAgentType OrchestratorContextKey = "orchestrator_agent_type"
)

// OrchestratorContext holds context data for orchestrator execution.
// This is used to pass request-level data through the call chain
// without modifying function signatures.
type OrchestratorContext struct {
	UserID         int32
	ConversationID int32
	BlockID        int64
	History        []string
	AgentType      string
	SessionID      string
}

// WithOrchestratorContext returns a new context with orchestrator data.
// This follows the standard Go context pattern for request-scoped values.
func WithOrchestratorContext(ctx context.Context, oc *OrchestratorContext) context.Context {
	if oc == nil {
		return ctx
	}
	ctx = context.WithValue(ctx, KeyUserID, oc.UserID)
	ctx = context.WithValue(ctx, KeyConversationID, oc.ConversationID)
	ctx = context.WithValue(ctx, KeyBlockID, oc.BlockID)
	ctx = context.WithValue(ctx, KeyHistory, oc.History)
	ctx = context.WithValue(ctx, KeyAgentType, oc.AgentType)
	ctx = context.WithValue(ctx, KeySessionID, oc.SessionID)
	return ctx
}

// GetUserID extracts user ID from orchestrator context.
// Returns 0 and false if not set.
func GetUserID(ctx context.Context) (int32, bool) {
	v := ctx.Value(KeyUserID)
	if v == nil {
		return 0, false
	}
	id, ok := v.(int32)
	return id, ok
}

// GetConversationID extracts conversation ID from orchestrator context.
// Returns 0 and false if not set.
func GetConversationID(ctx context.Context) (int32, bool) {
	v := ctx.Value(KeyConversationID)
	if v == nil {
		return 0, false
	}
	id, ok := v.(int32)
	return id, ok
}

// GetBlockID extracts block ID from orchestrator context.
// Returns 0 and false if not set.
func GetBlockID(ctx context.Context) (int64, bool) {
	v := ctx.Value(KeyBlockID)
	if v == nil {
		return 0, false
	}
	id, ok := v.(int64)
	return id, ok
}

// GetHistory extracts conversation history from orchestrator context.
// Returns an empty slice if not set.
func GetHistory(ctx context.Context) []string {
	v := ctx.Value(KeyHistory)
	if v == nil {
		return []string{}
	}
	history, ok := v.([]string)
	if !ok {
		return []string{}
	}
	return history
}

// GetSessionID extracts session ID from orchestrator context.
// Returns empty string if not set.
func GetSessionID(ctx context.Context) string {
	v := ctx.Value(KeySessionID)
	if v == nil {
		return ""
	}
	id, ok := v.(string)
	if !ok {
		return ""
	}
	return id
}

// GetAgentType extracts agent type from orchestrator context.
// Returns empty string if not set.
func GetAgentType(ctx context.Context) string {
	v := ctx.Value(KeyAgentType)
	if v == nil {
		return ""
	}
	agentType, ok := v.(string)
	if !ok {
		return ""
	}
	return agentType
}
