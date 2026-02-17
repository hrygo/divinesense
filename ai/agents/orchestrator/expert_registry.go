package orchestrator

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	agents "github.com/hrygo/divinesense/ai/agents"
	agentpkg "github.com/hrygo/divinesense/ai/agents/runner"
	"github.com/hrygo/divinesense/ai/agents/universal"
	ctxpkg "github.com/hrygo/divinesense/ai/context"
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

	// Filter out special agents (external executors)
	var experts []string
	for _, name := range configs {
		// Skip external executors that run outside Orchestrator
		if name == "geek" || name == "evolution" {
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
// It prioritizes SelfDescription from config, falls back to default descriptions.
func (r *ParrotExpertRegistry) GetExpertDescription(name string) string {
	config, ok := r.factory.GetConfig(name)
	if !ok {
		// Return default descriptions for known experts
		switch name {
		case "memo":
			return "笔记搜索专家。搜索用户记录的笔记、文档、想法。适用：查找之前记录的信息。"
		case "schedule":
			return "日程管理专家。创建、查询、更新日程。适用：时间管理、会议安排、查找空闲时间。"
		case "general":
			return "通用智能代理。处理文本总结、翻译、改写、问答等不需要特定工具的任务。适用：总结内容、翻译文本、解释说明。"
		default:
			return fmt.Sprintf("专家代理: %s", name)
		}
	}

	// Use SelfDescription if available (preferred for Orchestrator routing)
	if config.SelfDescription != nil {
		return buildDescriptionFromSelfCognition(config.SelfDescription)
	}

	// Fallback: build description from config fields
	desc := fmt.Sprintf("%s (%s)", config.DisplayName, config.Emoji)
	if len(config.Tools) > 0 {
		desc += fmt.Sprintf(" - 工具: %v", config.Tools)
	}
	return desc
}

// buildDescriptionFromSelfCognition creates a concise description for Orchestrator routing.
func buildDescriptionFromSelfCognition(cog *agents.ParrotSelfCognition) string {
	var parts []string

	// Title as main description
	if cog.Title != "" {
		parts = append(parts, cog.Title)
	}

	// Capabilities
	if len(cog.Capabilities) > 0 {
		parts = append(parts, fmt.Sprintf("能力: %s", strings.Join(cog.Capabilities, "、")))
	}

	// Working style
	if cog.WorkingStyle != "" {
		parts = append(parts, fmt.Sprintf("风格: %s", cog.WorkingStyle))
	}

	if len(parts) == 0 {
		return fmt.Sprintf("%s (%s)", cog.Name, cog.Emoji)
	}

	return strings.Join(parts, "。")
}

// GetExpertConfig returns the self-cognition configuration of an expert agent.
func (r *ParrotExpertRegistry) GetExpertConfig(name string) *agents.ParrotSelfCognition {
	config, ok := r.factory.GetConfig(name)
	if !ok {
		return nil
	}
	// Use SelfDescription if available
	if config.SelfDescription != nil {
		return config.SelfDescription
	}

	// Fallback: build minimal self-cognition from config fields
	return &agents.ParrotSelfCognition{
		Title:        config.DisplayName,
		Name:         config.Name,
		Emoji:        config.Emoji,
		Capabilities: config.Tools,
	}
}

// ExecuteExpert executes a task with the specified expert agent.
// history is automatically extracted from context via GetHistory.
func (r *ParrotExpertRegistry) ExecuteExpert(ctx context.Context, expertName string, input string, callback EventCallback) error {
	// Extract history from context for context-aware sub-agent execution
	history := ctxpkg.GetHistory(ctx)
	if len(history) > 0 {
		slog.Debug("expert: using conversation history for sub-agent",
			"expert", expertName,
			"history_len", len(history))
	}
	// Extract userID from context first, fallback to registry's default userID
	// This enables per-request userID override without changing registry initialization
	userID := r.userID
	if ctxUserID, ok := ctxpkg.GetUserID(ctx); ok {
		userID = ctxUserID
	}

	// Create the expert agent
	parrot, err := r.factory.CreateParrot(expertName, userID)
	if err != nil {
		return fmt.Errorf("create expert %s: %w", expertName, err)
	}

	// Convert orchestrator.EventCallback to agent.EventCallback
	// agent.EventCallback signature: func(eventType string, eventData any) error
	//
	// IMPORTANT: Include both EventData and Meta in JSON format.
	// The handler (server/router/api/v1/ai/handler.go) already parses this format
	// for tool_use/tool_result events to extract tool_name from meta.
	// Format: {"data": "...", "meta": {"tool_name": "xxx", ...}}
	agentCallback := func(eventType string, eventData any) error {
		if callback != nil {
			var eventDataStr string
			switch v := eventData.(type) {
			case string:
				eventDataStr = v
			case []byte:
				eventDataStr = string(v)
			case *agentpkg.EventWithMeta:
				// If Meta exists, include it as JSON for handler to parse tool_name etc.
				// This ensures handler can extract Meta fields like tool_name, status, etc.
				if v.Meta != nil {
					metaJSON, err := json.Marshal(v.Meta)
					if err == nil {
						// Use consistent format: {"data": "...", "meta": {...}}
						combined := fmt.Sprintf(`{"data":%q,"meta":%s}`, v.EventData, string(metaJSON))
						eventDataStr = combined
					} else {
						eventDataStr = v.EventData
					}
				} else {
					eventDataStr = v.EventData
				}
			default:
				eventDataStr = fmt.Sprintf("%v", v)
			}
			callback(eventType, eventDataStr)
		}
		return nil
	}

	// Execute with callback and history
	return parrot.Execute(ctx, input, history, agentCallback)
}

// GetIntentKeywords returns a map of intent names to their related keywords from expert configurations.
// This enables sticky routing to dynamically load keywords instead of hardcoding them.
func (r *ParrotExpertRegistry) GetIntentKeywords() map[string][]string {
	keywords := make(map[string][]string)

	// Get all expert configs and extract routing keywords
	for _, expertName := range r.GetAvailableExperts() {
		config := r.GetExpertConfig(expertName)
		if config == nil || config.Routing == nil {
			continue
		}

		// Map expert name to intent name
		intent := r.mapExpertToIntent(expertName)
		if intent != "" && len(config.Routing.Keywords) > 0 {
			keywords[intent] = config.Routing.Keywords
		}
	}

	// Ensure at least some defaults if empty
	if len(keywords) == 0 {
		keywords = getDefaultIntentKeywords()
	}

	return keywords
}

// mapExpertToIntent maps expert name to intent name for sticky routing.
func (r *ParrotExpertRegistry) mapExpertToIntent(expertName string) string {
	switch expertName {
	case "memo":
		return "memo_search"
	case "schedule":
		return "schedule_manage"
	case "general":
		return "general_task"
	default:
		return ""
	}
}

// getDefaultIntentKeywords returns default intent keywords as fallback.
func getDefaultIntentKeywords() map[string][]string {
	return map[string][]string{
		"memo_search":     {"笔记", "搜索", "查找", "找", "记录", "note", "search", "find", "look"},
		"schedule_manage": {"日程", "会议", "安排", "schedule", "meeting", "event", "calendar"},
		"general_task":    {"总结", "翻译", "改写", "解释", "summarize", "translate", "rewrite", "explain"},
	}
}

// Ensure ParrotExpertRegistry implements ExpertRegistry
var _ ExpertRegistry = (*ParrotExpertRegistry)(nil)
