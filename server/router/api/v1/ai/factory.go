package ai

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/hrygo/divinesense/ai"
	agents "github.com/hrygo/divinesense/ai/agents"
	"github.com/hrygo/divinesense/ai/agents/tools"
	scheduletools "github.com/hrygo/divinesense/ai/agents/tools/schedule"
	"github.com/hrygo/divinesense/ai/agents/universal"
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
	default:
		// DEFAULT, AMAZING, and unknown types trigger auto-routing
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
	default:
		return v1pb.AgentType_AGENT_TYPE_DEFAULT
	}
}

// CreateConfig contains configuration for creating an agent.
type CreateConfig struct {
	Type     AgentType
	Timezone string
	UserID   int32
}

// AgentFactory creates parrot agents based on type.
// Uses ParrotFactory for configuration-driven parrot creation.
type AgentFactory struct {
	llm           ai.LLMService
	retriever     *retrieval.AdaptiveRetriever
	store         *store.Store
	parrotFactory *universal.ParrotFactory
	mu            sync.RWMutex
	initialized   bool
}

// NewAgentFactory creates a new agent factory.
func NewAgentFactory(
	llm ai.LLMService,
	retriever *retrieval.AdaptiveRetriever,
	st *store.Store,
) *AgentFactory {
	factory := &AgentFactory{
		llm:         llm,
		retriever:   retriever,
		store:       st,
		initialized: false,
	}
	return factory
}

// Initialize initializes the ParrotFactory with the given configuration.
func (f *AgentFactory) Initialize(cfg *ai.UniversalParrotConfig) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	// Prevent duplicate initialization
	if f.initialized && f.parrotFactory != nil {
		slog.Debug("AgentFactory already initialized, skipping")
		return nil
	}

	if f.llm == nil || cfg == nil || !cfg.Enabled {
		return fmt.Errorf("llm service required or config not enabled")
	}

	configDir := cfg.ConfigDir
	if configDir == "" {
		configDir = "./config/parrots"
	}

	toolFactories := f.buildToolFactories()

	pf, err := universal.NewParrotFactory(
		universal.WithLLM(f.llm),
		universal.WithConfigDir(configDir),
		universal.WithToolFactories(toolFactories),
		universal.WithBaseURL(cfg.BaseURL),
	)
	if err != nil {
		return fmt.Errorf("initialize parrot factory: %w", err)
	}

	f.parrotFactory = pf
	f.initialized = true
	slog.Info("AgentFactory initialized successfully",
		"config_dir", configDir)
	return nil
}

// buildToolFactories creates tool factory functions for UniversalParrot.
func (f *AgentFactory) buildToolFactories() map[string]universal.ToolFactoryFunc {
	factories := make(map[string]universal.ToolFactoryFunc)

	// memo_search tool factory
	if f.retriever != nil {
		factories["memo_search"] = func(userID int32) (agents.ToolWithSchema, error) {
			userIDGetter := func(ctx context.Context) int32 {
				return userID
			}
			tool, err := tools.NewMemoSearchTool(f.retriever, userIDGetter)
			if err != nil {
				return nil, err
			}
			return agents.NewNativeTool(
				tool.Name(),
				tool.Description(),
				func(ctx context.Context, input string) (string, error) {
					return tool.Run(ctx, input)
				},
				map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"query": map[string]interface{}{
							"type":        "string",
							"description": "Search keywords. Use \"*\" to search all memos.",
						},
						"limit": map[string]interface{}{
							"type":        "integer",
							"description": "Max results, default 10, max 50",
						},
						"min_score": map[string]interface{}{
							"type":        "number",
							"description": "Min relevance 0-1, default 0.5",
						},
					},
					"required": []string{"query"},
				},
			), nil
		}
	}

	// schedule tools factories
	// Note: Each schedule tool has its own InputType() method
	// We create closures that capture the tool and its InputType method
	if f.store != nil {
		factories["schedule_add"] = func(userID int32) (agents.ToolWithSchema, error) {
			userIDGetter := func(ctx context.Context) int32 {
				return userID
			}
			scheduleSvc := schedule.NewService(f.store)
			tool := scheduletools.NewScheduleAddTool(scheduleSvc, userIDGetter)
			return agents.ToolFromLegacy(
				tool.Name(),
				tool.Description(),
				func(ctx context.Context, input string) (string, error) {
					return tool.Run(ctx, input)
				},
				tool.InputType,
			), nil
		}

		factories["schedule_query"] = func(userID int32) (agents.ToolWithSchema, error) {
			userIDGetter := func(ctx context.Context) int32 {
				return userID
			}
			scheduleSvc := schedule.NewService(f.store)
			tool := scheduletools.NewScheduleQueryTool(scheduleSvc, userIDGetter)
			return agents.ToolFromLegacy(
				tool.Name(),
				tool.Description(),
				func(ctx context.Context, input string) (string, error) {
					return tool.Run(ctx, input)
				},
				tool.InputType,
			), nil
		}

		factories["schedule_update"] = func(userID int32) (agents.ToolWithSchema, error) {
			userIDGetter := func(ctx context.Context) int32 {
				return userID
			}
			scheduleSvc := schedule.NewService(f.store)
			tool := scheduletools.NewScheduleUpdateTool(scheduleSvc, userIDGetter)
			return agents.ToolFromLegacy(
				tool.Name(),
				tool.Description(),
				func(ctx context.Context, input string) (string, error) {
					return tool.Run(ctx, input)
				},
				tool.InputType,
			), nil
		}

		factories["schedule_delete"] = func(userID int32) (agents.ToolWithSchema, error) {
			userIDGetter := func(ctx context.Context) int32 {
				return userID
			}
			scheduleSvc := schedule.NewService(f.store)
			tool := scheduletools.NewScheduleDeleteTool(scheduleSvc, userIDGetter)
			return agents.ToolFromLegacy(
				tool.Name(),
				tool.Description(),
				func(ctx context.Context, input string) (string, error) {
					return tool.Run(ctx, input)
				},
				tool.InputType,
			), nil
		}

		factories["find_free_time"] = func(userID int32) (agents.ToolWithSchema, error) {
			userIDGetter := func(ctx context.Context) int32 {
				return userID
			}
			scheduleSvc := schedule.NewService(f.store)
			tool := scheduletools.NewFindFreeTimeTool(scheduleSvc, userIDGetter)
			return agents.ToolFromLegacy(
				tool.Name(),
				tool.Description(),
				func(ctx context.Context, input string) (string, error) {
					return tool.Run(ctx, input)
				},
				tool.InputType,
			), nil
		}
	}

	// report_inability tool factory
	// Allows experts to report when they cannot handle a task (Handoff mechanism)
	factories["report_inability"] = func(userID int32) (agents.ToolWithSchema, error) {
		tool := tools.NewReportInabilityTool()
		return agents.ToolFromLegacy(
			tool.Name(),
			tool.Description(),
			func(ctx context.Context, input string) (string, error) {
				return tool.Run(ctx, input)
			},
			tool.InputType,
		), nil
	}

	return factories
}

// Create creates an agent based on the configuration.
func (f *AgentFactory) Create(ctx context.Context, cfg *CreateConfig) (agents.ParrotAgent, error) {
	f.mu.RLock()
	initialized := f.initialized
	pf := f.parrotFactory
	f.mu.RUnlock()

	if !initialized || pf == nil {
		return nil, fmt.Errorf("factory not initialized, call Initialize first")
	}

	if f.llm == nil {
		return nil, fmt.Errorf("llm service is required")
	}

	switch cfg.Type {
	case AgentTypeMemo:
		return f.createMemoParrot(cfg)
	case AgentTypeSchedule:
		return f.createScheduleParrot(ctx, cfg)
	default:
		// Auto-route: default to MemoParrot
		return f.createMemoParrot(cfg)
	}
}

// createMemoParrot creates a UniversalParrot configured as memo agent.
func (f *AgentFactory) createMemoParrot(cfg *CreateConfig) (agents.ParrotAgent, error) {
	if f.retriever == nil {
		return nil, fmt.Errorf("retriever is required for memo parrot")
	}
	return f.parrotFactory.CreateMemoParrot(cfg.UserID, f.retriever)
}

// createScheduleParrot creates a UniversalParrot configured as schedule agent.
func (f *AgentFactory) createScheduleParrot(_ context.Context, cfg *CreateConfig) (agents.ParrotAgent, error) {
	if f.store == nil {
		return nil, fmt.Errorf("store is required for schedule parrot")
	}
	scheduleSvc := schedule.NewService(f.store)
	return f.parrotFactory.CreateScheduleParrot(cfg.UserID, scheduleSvc)
}

// GetSelfCognitionConfigs returns all ParrotSelfCognition configurations from the factory.
// This is used by routing to build config-driven intent mappings.
func (f *AgentFactory) GetSelfCognitionConfigs() []*agents.ParrotSelfCognition {
	if f.parrotFactory == nil {
		return nil
	}

	names := f.parrotFactory.ListConfigs()
	var configs []*agents.ParrotSelfCognition
	for _, name := range names {
		if cfg, ok := f.parrotFactory.GetConfig(name); ok && cfg.SelfDescription != nil {
			configs = append(configs, cfg.SelfDescription)
		}
	}
	return configs
}

// GetParrotFactory returns the underlying ParrotFactory.
// This is used by Orchestrator to create expert agents.
func (f *AgentFactory) GetParrotFactory() *universal.ParrotFactory {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.parrotFactory
}
