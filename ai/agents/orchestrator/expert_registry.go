package orchestrator

import (
	"context"
	"fmt"

	"github.com/hrygo/divinesense/ai/agents/universal"
)

// ParrotExpertRegistry implements ExpertRegistry using ParrotFactory.
// It adapts the ParrotFactory to the ExpertRegistry interface used by Orchestrator.
type ParrotExpertRegistry struct {
	factory *universal.ParrotFactory
	userID  int32
}

// NewParrotExpertRegistry creates a new expert registry backed by ParrotFactory.
func NewParrotExpertRegistry(factory *universal.ParrotFactory, userID int32) *ParrotExpertRegistry {
	return &ParrotExpertRegistry{
		factory: factory,
		userID:  userID,
	}
}

// GetAvailableExperts returns the list of available expert agent names.
// It filters out special agents (geek, evolution) that are not meant for orchestration.
func (r *ParrotExpertRegistry) GetAvailableExperts() []string {
	configs := r.factory.ListConfigs()

	// Filter out special agents
	var experts []string
	for _, name := range configs {
		// Skip external executors (geek, evolution) and amazing (will be removed)
		if name == "geek" || name == "evolution" || name == "amazing" {
			continue
		}
		experts = append(experts, name)
	}

	// Ensure at least memo and schedule are available
	if len(experts) == 0 {
		experts = []string{"memo", "schedule"}
	}

	return experts
}

// GetExpertDescription returns a description of what an expert agent can do.
func (r *ParrotExpertRegistry) GetExpertDescription(name string) string {
	config, ok := r.factory.GetConfig(name)
	if !ok {
		// Return default descriptions for known experts
		switch name {
		case "memo":
			return "搜索和检索笔记内容。适用于查找用户记录的信息、文档、想法等。"
		case "schedule":
			return "创建、查询、更新日程安排。适用于时间管理、会议安排、查找空闲时间等。"
		default:
			return fmt.Sprintf("专家代理: %s", name)
		}
	}

	// Build description from config
	desc := fmt.Sprintf("%s (%s)", config.DisplayName, config.Emoji)
	if len(config.Tools) > 0 {
		desc += fmt.Sprintf(" - 工具: %v", config.Tools)
	}
	return desc
}

// ExecuteExpert executes a task with the specified expert agent.
func (r *ParrotExpertRegistry) ExecuteExpert(ctx context.Context, expertName string, input string, callback EventCallback) error {
	// Create the expert agent
	parrot, err := r.factory.CreateParrot(expertName, r.userID)
	if err != nil {
		return fmt.Errorf("create expert %s: %w", expertName, err)
	}

	// Convert orchestrator.EventCallback to agent.EventCallback
	// agent.EventCallback signature: func(eventType string, eventData any) error
	agentCallback := func(eventType string, eventData any) error {
		// Forward to orchestrator callback (convert eventData to string)
		if callback != nil {
			var eventDataStr string
			switch v := eventData.(type) {
			case string:
				eventDataStr = v
			case []byte:
				eventDataStr = string(v)
			default:
				// For other types, just pass empty string or marshal if needed
				eventDataStr = fmt.Sprintf("%v", v)
			}
			callback(eventType, eventDataStr)
		}
		return nil
	}

	// Execute with callback
	return parrot.ExecuteWithCallback(ctx, input, nil, agentCallback)
}

// Ensure ParrotExpertRegistry implements ExpertRegistry
var _ ExpertRegistry = (*ParrotExpertRegistry)(nil)
