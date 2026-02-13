// Package routing provides intent registry for OCP-compliant intent management.
package routing

import (
	"regexp"
	"slices"
	"strings"
	"sync"
)

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
