// Package routing provides intent registry for OCP-compliant intent management.
package routing

import (
	"regexp"
	"slices"
	"strings"
	"sync"
)

// ExpertConfig represents the minimal expert configuration needed for routing.
// This avoids import cycles between routing and agent packages.
type ExpertConfig interface {
	GetName() string
	GetCapabilities() []string
}

// ActionToCategoryMapping defines how GenericAction maps to Category (AgentType).
// This is the bridge between RuleMatcher (outputs GenericAction) and ExpertRouter (maps to Expert).
type ActionToCategoryMapping struct {
	Action    GenericAction // Generic action from RuleMatcher
	AgentType AgentType     // Target agent type
	Priority  int           // Higher = checked first
	// Optional: additional keywords for disambiguation
	Keywords []string
}

// IntentConfig holds configuration for a single intent type.
type IntentConfig struct {
	Intent    Intent           // Intent identifier
	AgentType AgentType        // Associated agent type
	Keywords  []string         // Keywords for rule-based matching
	Patterns  []*regexp.Regexp // Regex patterns for matching
	Priority  int              // Higher = checked first
	RouteType string           // Route type for chat router
}

// IntentRegistry manages intent configurations with OCP-compliant registration.
type IntentRegistry struct {
	mu          sync.RWMutex
	configs     map[Intent]IntentConfig
	agentMap    map[Intent]AgentType
	intentMap   map[AgentType]Intent
	sortedByPri []IntentConfig // Cache for matching order
}

// NewIntentRegistry creates a new empty registry.
func NewIntentRegistry() *IntentRegistry {
	return &IntentRegistry{
		configs:   make(map[Intent]IntentConfig),
		agentMap:  make(map[Intent]AgentType),
		intentMap: make(map[AgentType]Intent),
	}
}

// BuildFromExpertConfigs builds the intent registry from ParrotSelfCognition configurations.
// This enables config-driven routing without hardcoded expert types.
// BuildFromExpertConfigs builds the intent registry from expert configurations.
// This enables config-driven routing without hardcoded expert types.
func (r *IntentRegistry) BuildFromExpertConfigs(configs []ExpertConfig) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Clear existing configs
	r.configs = make(map[Intent]IntentConfig)
	r.agentMap = make(map[Intent]AgentType)
	r.intentMap = make(map[AgentType]Intent)

	// Build action to category mappings from expert capabilities
	// Each expert's capabilities define what actions they can handle
	actionToCategory := make(map[GenericAction][]ActionToCategoryMapping)

	for _, config := range configs {
		if config == nil {
			continue
		}

		// Determine agent type from expert name
		agentType := r.inferAgentType(config.GetName(), config.GetCapabilities())

		// Map each capability to a generic action
		for _, cap := range config.GetCapabilities() {
			action := r.capabilityToAction(cap)
			if action == ActionNone {
				continue
			}

			mapping := ActionToCategoryMapping{
				Action:    action,
				AgentType: agentType,
				Priority:  100,
			}

			actionToCategory[action] = append(actionToCategory[action], mapping)
		}
	}

	// Register intent configs based on action mappings
	// This is the bridge: GenericAction -> Intent -> AgentType
	for action, mappings := range actionToCategory {
		for _, m := range mappings {
			intent := r.actionToIntent(action, m.AgentType)
			r.configs[intent] = IntentConfig{
				Intent:    intent,
				AgentType: m.AgentType,
				Priority:  m.Priority,
				RouteType: string(m.AgentType),
			}
			r.agentMap[intent] = m.AgentType
			r.intentMap[m.AgentType] = intent
		}
	}

	r.rebuildSortedCache()
}

// inferAgentType infers the agent type from expert name and capabilities.
func (r *IntentRegistry) inferAgentType(name string, capabilities []string) AgentType {
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

// capabilityToAction maps a capability name to a generic action.
func (r *IntentRegistry) capabilityToAction(capability string) GenericAction {
	capLower := strings.ToLower(capability)

	// Search actions
	if strings.Contains(capLower, "搜索") || strings.Contains(capLower, "查询") ||
		strings.Contains(capLower, "search") || strings.Contains(capLower, "query") ||
		strings.Contains(capLower, "查找") {
		return ActionSearch
	}

	// Create actions
	if strings.Contains(capLower, "创建") || strings.Contains(capLower, "新建") ||
		strings.Contains(capLower, "create") || strings.Contains(capLower, "记录") {
		return ActionCreate
	}

	// Update actions
	if strings.Contains(capLower, "更新") || strings.Contains(capLower, "修改") ||
		strings.Contains(capLower, "delete") || strings.Contains(capLower, "删除") {
		return ActionUpdate
	}

	// Batch actions
	if strings.Contains(capLower, "批量") || strings.Contains(capLower, "batch") ||
		strings.Contains(capLower, "重复") {
		return ActionBatch
	}

	// Query actions (default for schedule)
	if strings.Contains(capLower, "查询") || strings.Contains(capLower, "查看") ||
		strings.Contains(capLower, "query") {
		return ActionQuery
	}

	return ActionNone
}

// actionToIntent converts a GenericAction + AgentType to a specific Intent.
func (r *IntentRegistry) actionToIntent(action GenericAction, agentType AgentType) Intent {
	switch agentType {
	case AgentTypeMemo:
		switch action {
		case ActionSearch:
			return IntentMemoSearch
		case ActionCreate:
			return IntentMemoCreate
		}
	case AgentTypeSchedule:
		switch action {
		case ActionQuery:
			return IntentScheduleQuery
		case ActionCreate:
			return IntentScheduleCreate
		case ActionUpdate:
			return IntentScheduleUpdate
		case ActionBatch:
			return IntentBatchSchedule
		}
	case AgentTypeGeneral:
		// General agent handles pure LLM tasks without action distinction
		return IntentGeneralTask
	}
	// Default
	return IntentUnknown
}

// ClassifyAction classifies a GenericAction to an AgentType using registered mappings.
func (r *IntentRegistry) ClassifyAction(action GenericAction, keywords []string) (AgentType, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Find the first matching mapping based on priority
	// This allows multiple experts to handle the same action
	keywordsLower := make(map[string]bool)
	for _, kw := range keywords {
		keywordsLower[strings.ToLower(kw)] = true
	}

	for _, cfg := range r.sortedByPri {
		if cfg.Intent == IntentUnknown {
			continue
		}
		// Match by action
		intentAction := r.intentToAction(cfg.Intent)
		if intentAction == action {
			return cfg.AgentType, true
		}
	}

	return AgentTypeUnknown, false
}

// intentToAction converts an Intent back to GenericAction.
func (r *IntentRegistry) intentToAction(intent Intent) GenericAction {
	switch intent {
	case IntentMemoSearch, IntentMemoCreate:
		return ActionSearch
	case IntentScheduleQuery, IntentScheduleCreate:
		return ActionQuery
	case IntentScheduleUpdate:
		return ActionUpdate
	case IntentBatchSchedule:
		return ActionBatch
	}
	return ActionNone
}

// Register adds or updates an intent configuration.
func (r *IntentRegistry) Register(cfg IntentConfig) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.configs[cfg.Intent] = cfg
	r.agentMap[cfg.Intent] = cfg.AgentType
	r.intentMap[cfg.AgentType] = cfg.Intent

	// Rebuild sorted cache
	r.rebuildSortedCache()
}

// rebuildSortedCache rebuilds the priority-sorted config slice.
// Must be called with lock held.
func (r *IntentRegistry) rebuildSortedCache() {
	configs := make([]IntentConfig, 0, len(r.configs))
	for _, cfg := range r.configs {
		configs = append(configs, cfg)
	}
	// Sort by priority descending (higher priority first)
	slices.SortFunc(configs, func(a, b IntentConfig) int {
		return b.Priority - a.Priority
	})
	r.sortedByPri = configs
}

// Match performs rule-based intent matching.
// Returns: matched intent, confidence (0-1), whether match was found.
func (r *IntentRegistry) Match(input string) (Intent, float32, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	lowerInput := strings.ToLower(input)

	for _, cfg := range r.sortedByPri {
		// Check regex patterns first (higher precision)
		for _, pattern := range cfg.Patterns {
			if pattern.MatchString(input) {
				return cfg.Intent, 0.9, true
			}
		}

		// Check keywords
		for _, kw := range cfg.Keywords {
			if strings.Contains(lowerInput, strings.ToLower(kw)) {
				return cfg.Intent, 0.7, true
			}
		}
	}

	return IntentUnknown, 0, false
}

// GetAgentType returns the agent type for an intent.
func (r *IntentRegistry) GetAgentType(intent Intent) (AgentType, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	at, ok := r.agentMap[intent]
	return at, ok
}

// GetRouteType returns the route type for an intent.
func (r *IntentRegistry) GetRouteType(intent Intent) (string, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	cfg, ok := r.configs[intent]
	if !ok {
		return "", false
	}
	return cfg.RouteType, true
}

// GetIntent returns the default intent for an agent type.
func (r *IntentRegistry) GetIntent(agentType AgentType) (Intent, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	intent, ok := r.intentMap[agentType]
	return intent, ok
}

// RegisterDefaults registers built-in intent configurations.
func (r *IntentRegistry) RegisterDefaults() {
	// Schedule intents
	r.Register(IntentConfig{
		Intent:    IntentScheduleCreate,
		AgentType: AgentTypeSchedule,
		Keywords:  []string{"创建", "添加", "安排", "预约", "提醒", "设置"},
		Priority:  100,
		RouteType: "schedule",
	})
	r.Register(IntentConfig{
		Intent:    IntentScheduleQuery,
		AgentType: AgentTypeSchedule,
		Keywords:  []string{"查询", "查看", "有什么", "列出", "显示", "日程", "安排"},
		Priority:  100,
		RouteType: "schedule",
	})
	r.Register(IntentConfig{
		Intent:    IntentScheduleUpdate,
		AgentType: AgentTypeSchedule,
		Keywords:  []string{"修改", "更新", "删除", "取消", "改到", "调整"},
		Priority:  100,
		RouteType: "schedule",
	})
	r.Register(IntentConfig{
		Intent:    IntentBatchSchedule,
		AgentType: AgentTypeSchedule,
		Keywords:  []string{"每天", "每周", "每月", "批量", "重复", "工作日"},
		Priority:  110, // Higher priority to catch batch patterns first
		RouteType: "schedule",
	})

	// Memo intents
	r.Register(IntentConfig{
		Intent:    IntentMemoSearch,
		AgentType: AgentTypeMemo,
		Keywords:  []string{"笔记", "搜索", "查找", "memo", "note", "记录"},
		Priority:  100,
		RouteType: "memo",
	})
	r.Register(IntentConfig{
		Intent:    IntentMemoCreate,
		AgentType: AgentTypeMemo,
		Keywords:  []string{"记录", "写笔记", "添加笔记", "新建笔记"},
		Priority:  90,
		RouteType: "memo",
	})

	// General intents - pure LLM tasks without tools
	r.Register(IntentConfig{
		Intent:    IntentGeneralTask,
		AgentType: AgentTypeGeneral,
		Keywords:  []string{"总结", "摘要", "翻译", "改写", "润色", "解释", "说明", "什么意思", "重写", "summarize", "summary", "translate", "rewrite", "polish", "explain"},
		Priority:  50, // Lower priority than specialists
		RouteType: "general",
	})
}

// Global default registry instance
var defaultRegistry = NewIntentRegistry()

func init() {
	defaultRegistry.RegisterDefaults()
}

// DefaultRegistry returns the global default registry.
func DefaultRegistry() *IntentRegistry {
	return defaultRegistry
}
