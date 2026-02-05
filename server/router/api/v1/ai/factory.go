package ai

import (
	"context"
	"fmt"

	"github.com/hrygo/divinesense/ai"
	agentpkg "github.com/hrygo/divinesense/ai/agent"
	"github.com/hrygo/divinesense/ai/core/retrieval"
	v1pb "github.com/hrygo/divinesense/proto/gen/api/v1"
	"github.com/hrygo/divinesense/server/service/schedule"
	"github.com/hrygo/divinesense/store"
)

// AgentType represents the type of agent to create.
type AgentType string

const (
	AgentTypeMemo     AgentType = "MEMO"
	AgentTypeSchedule AgentType = "SCHEDULE"
	AgentTypeAmazing  AgentType = "AMAZING"
	AgentTypeAuto     AgentType = "AUTO" // Auto-route based on intent
)

// String returns the string representation of the agent type.
func (t AgentType) String() string {
	return string(t)
}

// AgentTypeFromProto converts proto AgentType to internal AgentType.
// DEFAULT triggers auto-routing based on user intent.
func AgentTypeFromProto(protoType v1pb.AgentType) AgentType {
	switch protoType {
	case v1pb.AgentType_AGENT_TYPE_MEMO:
		return AgentTypeMemo
	case v1pb.AgentType_AGENT_TYPE_SCHEDULE:
		return AgentTypeSchedule
	case v1pb.AgentType_AGENT_TYPE_AMAZING:
		return AgentTypeAmazing
	default:
		// DEFAULT and unknown types trigger auto-routing
		return AgentTypeAuto
	}
}

// ToProto converts internal AgentType to proto AgentType.
// AUTO maps to DEFAULT to let backend ChatRouter decide the agent.
func (t AgentType) ToProto() v1pb.AgentType {
	switch t {
	case AgentTypeMemo:
		return v1pb.AgentType_AGENT_TYPE_MEMO
	case AgentTypeSchedule:
		return v1pb.AgentType_AGENT_TYPE_SCHEDULE
	case AgentTypeAuto:
		return v1pb.AgentType_AGENT_TYPE_DEFAULT
	default:
		return v1pb.AgentType_AGENT_TYPE_AMAZING
	}
}

// CreateConfig contains configuration for creating an agent.
type CreateConfig struct {
	Type     AgentType
	Timezone string
	UserID   int32
}

// AgentFactory creates parrot agents based on type.
type AgentFactory struct {
	llm       ai.LLMService
	retriever *retrieval.AdaptiveRetriever
	store     *store.Store
}

// NewAgentFactory creates a new agent factory.
func NewAgentFactory(
	llm ai.LLMService,
	retriever *retrieval.AdaptiveRetriever,
	st *store.Store,
) *AgentFactory {
	return &AgentFactory{
		llm:       llm,
		retriever: retriever,
		store:     st,
	}
}

// Create creates an agent based on the configuration.
func (f *AgentFactory) Create(ctx context.Context, cfg *CreateConfig) (agentpkg.ParrotAgent, error) {
	if f.llm == nil {
		return nil, fmt.Errorf("llm service is required")
	}

	switch cfg.Type {
	case AgentTypeMemo:
		return f.createMemoParrot(cfg)
	case AgentTypeSchedule:
		return f.createScheduleParrot(ctx, cfg)
	case AgentTypeAmazing:
		return f.createAmazingParrot(ctx, cfg)
	default:
		// Fallback to AMAZING for comprehensive assistance
		return f.createAmazingParrot(ctx, cfg)
	}
}

// createMemoParrot creates a memo parrot agent.
func (f *AgentFactory) createMemoParrot(cfg *CreateConfig) (agentpkg.ParrotAgent, error) {
	if f.retriever == nil {
		return nil, fmt.Errorf("retriever is required for memo parrot")
	}

	agent, err := agentpkg.NewMemoParrot(
		f.retriever,
		f.llm,
		cfg.UserID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create memo parrot: %w", err)
	}

	return agent, nil
}

// createScheduleParrot creates a schedule parrot agent.
// Uses the new framework-less SchedulerAgentV2 (no LangChainGo dependency).
func (f *AgentFactory) createScheduleParrot(_ context.Context, cfg *CreateConfig) (agentpkg.ParrotAgent, error) {
	if f.store == nil {
		return nil, fmt.Errorf("store is required for schedule parrot")
	}

	// Normalize timezone: use provided timezone or default
	timezone := NormalizeTimezone(cfg.Timezone)

	// Create schedule service
	scheduleSvc := schedule.NewService(f.store)

	// Create scheduler agent V2 (framework-less, uses native LLM tool calling)
	schedulerAgent, err := agentpkg.NewSchedulerAgentV2(
		f.llm,
		scheduleSvc,
		cfg.UserID,
		timezone,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create scheduler agent v2: %w", err)
	}

	// Wrap in schedule parrot V2
	parrot, err := agentpkg.NewScheduleParrotV2(schedulerAgent)
	if err != nil {
		return nil, fmt.Errorf("failed to create schedule parrot v2: %w", err)
	}

	return parrot, nil
}

// createAmazingParrot creates an amazing parrot agent.
func (f *AgentFactory) createAmazingParrot(_ context.Context, cfg *CreateConfig) (agentpkg.ParrotAgent, error) {
	if f.retriever == nil {
		return nil, fmt.Errorf("retriever is required for amazing parrot")
	}
	if f.store == nil {
		return nil, fmt.Errorf("store is required for amazing parrot")
	}

	// Create schedule service
	scheduleSvc := schedule.NewService(f.store)

	agent, err := agentpkg.NewAmazingParrot(
		f.llm,
		f.retriever,
		scheduleSvc,
		cfg.UserID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create amazing parrot: %w", err)
	}

	return agent, nil
}
