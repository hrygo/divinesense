// Package routing provides the LLM routing service interface for AI agents.
// This interface is consumed by Team B (Assistant+Schedule) and Team C (Memo Enhancement).
package routing

import (
	"context"
	"time"
)

// Consumers: Team B (Assistant+Schedule), Team C (Memo Enhancement).
type RouterService interface {
	// ClassifyIntent classifies user intent from input text.
	// Returns: intent type, confidence (0-1), needsOrchestration, error
	// Implementation: FastRouter (cache -> rule), high confidence routes directly,
	// low confidence/complex requests need orchestration
	ClassifyIntent(ctx context.Context, input string) (Intent, float32, bool, error)

	// SelectModel selects an appropriate model based on task type.
	// Returns: model configuration (local/cloud)
	SelectModel(ctx context.Context, task TaskType) (ModelConfig, error)

	// RecordFeedback records user feedback for a routing decision.
	// This enables dynamic weight adjustment for improved routing accuracy.
	RecordFeedback(ctx context.Context, feedback *RouterFeedback) error

	// GetRouterStats retrieves routing accuracy statistics.
	GetRouterStats(ctx context.Context, userID int32, timeRange time.Duration) (*RouterStats, error)
}

// AgentType represents the agent type for routing.
type AgentType string

const (
	AgentTypeMemo     AgentType = "memo"
	AgentTypeSchedule AgentType = "schedule"
	AgentTypeUnknown  AgentType = "unknown"
	// Note: AgentTypeAmazing removed - Orchestrator handles complex/ambiguous requests
)

// IntentToAgentType converts Intent to AgentType.
// This is the canonical mapping used across the routing system.
func IntentToAgentType(intent Intent) AgentType {
	switch intent {
	case IntentMemoSearch, IntentMemoCreate:
		return AgentTypeMemo
	case IntentScheduleQuery, IntentScheduleCreate, IntentScheduleUpdate, IntentBatchSchedule:
		return AgentTypeSchedule
	default:
		return AgentTypeUnknown
	}
}

// AgentTypeToIntent converts AgentType to default Intent.
// Used when a specific intent subtype cannot be determined.
func AgentTypeToIntent(agentType AgentType) Intent {
	switch agentType {
	case AgentTypeMemo:
		return IntentMemoCreate
	case AgentTypeSchedule:
		return IntentScheduleCreate
	default:
		return IntentUnknown
	}
}

// Intent represents the type of user intent.
type Intent string

const (
	IntentMemoSearch     Intent = "memo_search"
	IntentMemoCreate     Intent = "memo_create"
	IntentScheduleQuery  Intent = "schedule_query"
	IntentScheduleCreate Intent = "schedule_create"
	IntentScheduleUpdate Intent = "schedule_update"
	IntentBatchSchedule  Intent = "batch_schedule"
	// Note: IntentAmazing removed - Orchestrator handles complex/ambiguous requests
	IntentUnknown Intent = "unknown"
)

// TaskType represents the type of task for model selection.
type TaskType string

const (
	TaskIntentClassification TaskType = "intent_classification"
	TaskEntityExtraction     TaskType = "entity_extraction"
	TaskSimpleQA             TaskType = "simple_qa"
	TaskComplexReasoning     TaskType = "complex_reasoning"
	TaskSummarization        TaskType = "summarization"
	TaskTagSuggestion        TaskType = "tag_suggestion"
)

// ModelConfig represents the configuration for a model.
type ModelConfig struct {
	Provider    string  `json:"provider"` // local/cloud
	Model       string  `json:"model"`    // model name
	MaxTokens   int     `json:"max_tokens"`
	Temperature float32 `json:"temperature"`
}
