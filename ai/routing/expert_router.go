// Package routing provides expert router for config-driven expert selection.
package routing

import (
	"strings"
	"sync"
)

// ExpertRouter defines the interface for mapping AgentType to specific expert names.
// This is the final layer in config-driven routing: GenericAction → AgentType → ExpertName.
type ExpertRouter interface {
	// Route returns the expert name for a given agent type.
	// Returns empty string if no expert found.
	Route(agentType AgentType) string

	// RouteWithKeywords returns the expert name considering both agent type and matched keywords.
	// This provides more precise routing when multiple experts handle the same agent type.
	RouteWithKeywords(agentType AgentType, keywords []string) string
}

// DefaultExpertRouter is the default implementation of ExpertRouter.
// It builds mappings from expert configurations.
type DefaultExpertRouter struct {
	mu             sync.RWMutex
	agentToExpert  map[AgentType]string // AgentType -> Expert name
	keywordExperts map[string][]string  // Keyword -> Expert names (for disambiguation)
}

// NewExpertRouter creates a new expert router.
func NewExpertRouter() *DefaultExpertRouter {
	return &DefaultExpertRouter{
		agentToExpert:  make(map[AgentType]string),
		keywordExperts: make(map[string][]string),
	}
}

// BuildFromConfigs builds the router from expert configurations.
func (r *DefaultExpertRouter) BuildFromConfigs(configs []ExpertConfig) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Clear existing mappings
	r.agentToExpert = make(map[AgentType]string)
	r.keywordExperts = make(map[string][]string)

	for _, config := range configs {
		if config == nil {
			continue
		}

		name := config.GetName()
		caps := config.GetCapabilities()

		// Determine agent type
		agentType := r.inferAgentType(name, caps)

		// Register primary mapping
		if _, exists := r.agentToExpert[agentType]; !exists {
			r.agentToExpert[agentType] = name
		}

		// Register keyword mappings for disambiguation
		for _, cap := range caps {
			// Add capability keywords
			capLower := strings.ToLower(cap)
			r.keywordExperts[capLower] = append(r.keywordExperts[capLower], name)

			// Also add common trigger words
			for _, trigger := range r.getTriggersForCapability(cap) {
				triggerLower := strings.ToLower(trigger)
				r.keywordExperts[triggerLower] = append(r.keywordExperts[triggerLower], name)
			}
		}
	}
}

// getTriggersForCapability returns common trigger words for a capability.
func (r *DefaultExpertRouter) getTriggersForCapability(capability string) []string {
	capLower := strings.ToLower(capability)

	// Map capabilities to their trigger words
	triggers := map[string][]string{
		"笔记":   {"笔记", "搜索", "memo", "note"},
		"搜索":   {"搜索", "查找", "找", "查"},
		"日程":   {"日程", "安排", "会议", "schedule"},
		"日程查询": {"日程", "安排", "查看", "查询"},
		"日程创建": {"创建", "添加", "新建"},
		"会议":   {"会议", "开会", "预约"},
	}

	for key, value := range triggers {
		if strings.Contains(capLower, strings.ToLower(key)) {
			return value
		}
	}

	return nil
}

// inferAgentType infers the agent type from expert name and capabilities.
func (r *DefaultExpertRouter) inferAgentType(name string, capabilities []string) AgentType {
	nameLower := strings.ToLower(name)

	// Check name first
	if strings.Contains(nameLower, "memo") || strings.Contains(nameLower, "笔记") {
		return AgentTypeMemo
	}
	if strings.Contains(nameLower, "schedule") || strings.Contains(nameLower, "日程") {
		return AgentTypeSchedule
	}

	// Check capabilities
	for _, cap := range capabilities {
		capLower := strings.ToLower(cap)
		if strings.Contains(capLower, "笔记") || strings.Contains(capLower, "memo") ||
			strings.Contains(capLower, "搜索") {
			return AgentTypeMemo
		}
		if strings.Contains(capLower, "日程") || strings.Contains(capLower, "schedule") ||
			strings.Contains(capLower, "会议") {
			return AgentTypeSchedule
		}
	}

	return AgentTypeUnknown
}

// Route returns the expert name for a given agent type.
func (r *DefaultExpertRouter) Route(agentType AgentType) string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if expert, ok := r.agentToExpert[agentType]; ok {
		return expert
	}
	return ""
}

// RouteWithKeywords returns the expert name considering both agent type and matched keywords.
func (r *DefaultExpertRouter) RouteWithKeywords(agentType AgentType, keywords []string) string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// First try keyword-based routing
	if len(keywords) > 0 {
		// Find experts that match the keywords
		keywordMatchCount := make(map[string]int)
		for _, kw := range keywords {
			kwLower := strings.ToLower(kw)
			if experts, ok := r.keywordExperts[kwLower]; ok {
				for _, exp := range experts {
					keywordMatchCount[exp]++
				}
			}
		}

		// Find best matching expert
		var bestExpert string
		var bestCount int
		for expert, count := range keywordMatchCount {
			if count > bestCount {
				bestCount = count
				bestExpert = expert
			}
		}

		if bestExpert != "" {
			return bestExpert
		}
	}

	// Fallback to agent type mapping
	if expert, ok := r.agentToExpert[agentType]; ok {
		return expert
	}

	return ""
}

// Ensure DefaultExpertRouter implements ExpertRouter
var _ ExpertRouter = (*DefaultExpertRouter)(nil)
