// Package universal provides the UniversalParrot - a configuration-driven
// parrot that can mimic any existing parrot through configuration.
package universal

import (
	"container/list"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/hrygo/divinesense/ai"
	"github.com/hrygo/divinesense/ai/agents"
)

const (
	// DefaultTimezone is the default timezone for time context calculations.
	DefaultTimezone = "Asia/Shanghai"
)

// UniversalParrot is a configuration-driven parrot that can
// mimic any existing parrot (Memo, Schedule, Amazing) through config.
type UniversalParrot struct {
	// Configuration
	config *ParrotConfig

	// Execution strategy
	strategy ExecutionStrategy

	// Dependencies
	llm   ai.LLMService
	tools map[string]agent.ToolWithSchema

	// Optional dependencies for specific tools
	retriever       interface{} // *retrieval.AdaptiveRetriever
	scheduleService interface{} // schedule.Service

	// Cache
	cache *LRUCache

	// Statistics
	stats *NormalSessionStats

	// User context
	userID   int32
	timezone string

	mu sync.RWMutex
}

// NewUniversalParrot creates a parrot from configuration.
func NewUniversalParrot(
	config *ParrotConfig,
	llm ai.LLMService,
	tools map[string]agent.ToolWithSchema,
	userID int32,
) (*UniversalParrot, error) {
	// Validate config
	if err := validateConfig(config); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	// Resolve strategy (streaming enabled by default)
	resolver := NewDefaultResolver(config.MaxIterations)
	strategy, err := resolver.Resolve(config.Strategy)
	if err != nil {
		return nil, fmt.Errorf("strategy resolution: %w", err)
	}

	// Initialize cache if enabled
	var cache *LRUCache
	if config.EnableCache {
		cacheSize := config.CacheSize
		if cacheSize <= 0 {
			cacheSize = 100 // Default
		}
		cache = NewLRUCache(cacheSize, config.CacheTTL)
	}

	return &UniversalParrot{
		config:   config,
		strategy: strategy,
		llm:      llm,
		tools:    tools,
		cache:    cache,
		stats:    NewNormalSessionStats(config.Name),
		userID:   userID,
		timezone: "Local",
	}, nil
}

// Execute implements agent.ParrotAgent.
// Execute is a wrapper around ExecuteWithCallback with empty history.
func (p *UniversalParrot) Execute(
	ctx context.Context,
	userInput string,
	callback agent.EventCallback,
) error {
	return p.ExecuteWithCallback(ctx, userInput, nil, callback)
}

// ExecuteWithCallback provides an extended execution method with history support.
func (p *UniversalParrot) ExecuteWithCallback(
	ctx context.Context,
	userInput string,
	history []string,
	callback agent.EventCallback,
) error {
	p.mu.Lock()
	startTime := time.Now()
	p.mu.Unlock()

	// Check cache
	if p.cache != nil {
		cacheKey := p.generateCacheKey(p.config.Name, p.userID, userInput)
		if cached, found := p.cache.Get(cacheKey); found {
			slog.Info("UniversalParrot cache hit", "parrot", p.config.Name)
			if callback != nil {
				if err := callback(agent.EventTypeAnswer, cached); err != nil {
					slog.Warn("cache hit callback failed", "error", err)
				}
			}
			return nil
		}
	}

	// Build messages from history
	messages := p.buildMessages(history)

	// Build time context for time-aware reasoning
	timeContext := p.buildTimeContext()

	// Execute strategy with time context
	result, execStats, err := p.strategy.Execute(
		ctx,
		userInput,
		messages,
		p.resolveTools(),
		p.llm,
		callback,
		timeContext,
	)

	if err != nil {
		return agent.NewParrotError(p.config.Name, "ExecuteWithCallback", err)
	}

	// Cache result
	if p.cache != nil && result != "" {
		cacheKey := p.generateCacheKey(p.config.Name, p.userID, userInput)
		p.cache.Set(cacheKey, result)
	}

	// Accumulate stats
	p.accumulateStats(execStats, startTime)

	return nil
}

// Name implements agent.ParrotAgent.
func (p *UniversalParrot) Name() string {
	return p.config.Name
}

// SelfDescribe implements agent.ParrotAgent.
func (p *UniversalParrot) SelfDescribe() *agent.ParrotSelfCognition {
	if p.config.SelfDescription != nil {
		return p.config.SelfDescription
	}
	// Fallback to basic description
	return &agent.ParrotSelfCognition{
		Title:        p.config.DisplayName,
		Name:         p.config.Name,
		Emoji:        p.config.Emoji,
		Capabilities: p.config.Tools,
	}
}

// GetSessionStats returns the accumulated session statistics.
func (p *UniversalParrot) GetSessionStats() *NormalSessionStats {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.stats.GetStatsSnapshot()
}

// SetRetriever sets the retriever dependency.
func (p *UniversalParrot) SetRetriever(retriever interface{}) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.retriever = retriever
}

// SetScheduleService sets the schedule service dependency.
func (p *UniversalParrot) SetScheduleService(scheduleService interface{}) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.scheduleService = scheduleService
}

// validateConfig validates the parrot configuration.
func validateConfig(config *ParrotConfig) error {
	if config.Name == "" {
		return fmt.Errorf("name is required")
	}
	if config.Strategy == "" {
		return fmt.Errorf("strategy is required")
	}
	return nil
}

// buildMessages converts string history to AI messages.
// Enhances system prompt with current date context for time-aware reasoning.
func (p *UniversalParrot) buildMessages(history []string) []ai.Message {
	messages := make([]ai.Message, 0, len(history)+1)

	// Add system prompt first, enhanced with current date context
	if p.config.SystemPrompt != "" {
		timeContext := p.buildTimeContext()
		systemContent := p.enhanceSystemPromptWithDate(p.config.SystemPrompt, timeContext)
		messages = append(messages, ai.Message{
			Role:    "system",
			Content: systemContent,
		})
	}

	// Add conversation history
	for i := 0; i < len(history); i += 2 {
		if i+1 < len(history) {
			messages = append(messages,
				ai.Message{Role: "user", Content: history[i]},
				ai.Message{Role: "assistant", Content: history[i+1]},
			)
		}
	}

	return messages
}

// buildTimeContext creates a structured time context for time-aware reasoning.
// This is used by ExecutionStrategy implementations (especially PlanningExecutor)
// and by buildMessages for system prompt enhancement.
func (p *UniversalParrot) buildTimeContext() *TimeContext {
	// Get timezone location
	loc := p.timezone
	if loc == "Local" || loc == "" {
		loc = DefaultTimezone
	}
	timezoneLoc, err := time.LoadLocation(loc)
	if err != nil {
		slog.Error("failed to load timezone for time context, using UTC as safe fallback",
			"timezone", loc,
			"user_id", p.userID,
			"error", err,
		)
		// Use UTC as safe fallback rather than time.Local (server timezone)
		// This ensures consistent behavior regardless of server location
		timezoneLoc = time.UTC
	}

	// Build structured time context
	return BuildTimeContext(timezoneLoc)
}

// enhanceSystemPromptWithDate injects structured current date context into the system prompt.
// The JSON format is more reliably parsed by LLMs than free-form text (20%+ accuracy boost).
func (p *UniversalParrot) enhanceSystemPromptWithDate(basePrompt string, tc *TimeContext) string {
	// If no timeContext provided, build one
	if tc == nil {
		tc = p.buildTimeContext()
	}

	// Minimal instruction: JSON context + one-line pattern guide
	// Based on 2025 research:简洁优于复杂，LLM可从结构化数据推断规则
	dateContext := fmt.Sprintf(`

<time_context>
%s
</time_context>

Use the JSON above for time calculations. Output format: ISO8601 (YYYY-MM-DDTHH:mm:ss+08:00)
`,
		tc.FormatAsJSONBlock(),
	)

	return basePrompt + dateContext
}

// resolveTools resolves tool names to ToolWithSchema instances.
func (p *UniversalParrot) resolveTools() []agent.ToolWithSchema {
	p.mu.RLock()
	defer p.mu.RUnlock()

	tools := make([]agent.ToolWithSchema, 0, len(p.config.Tools))
	for _, toolName := range p.config.Tools {
		if tool, ok := p.tools[toolName]; ok && tool != nil {
			tools = append(tools, tool)
		} else {
			slog.Warn("tool not found", "tool", toolName)
		}
	}
	return tools
}

// accumulateStats accumulates execution statistics.
func (p *UniversalParrot) accumulateStats(execStats *ExecutionStats, startTime time.Time) {
	p.mu.Lock()
	defer p.mu.Unlock()

	endTime := time.Now()
	duration := endTime.Sub(startTime)

	p.stats.PromptTokens += execStats.PromptTokens
	p.stats.CompletionTokens += execStats.CompletionTokens
	p.stats.TotalTokens += execStats.TotalTokens
	p.stats.CacheReadTokens += execStats.CacheReadTokens
	p.stats.CacheWriteTokens += execStats.CacheWriteTokens
	p.stats.ToolCallCount += execStats.ToolCalls
	p.stats.TotalDurationMs += duration.Milliseconds()
}

// generateCacheKey generates a cache key for the input.
func (p *UniversalParrot) generateCacheKey(name string, userID int32, input string) string {
	return fmt.Sprintf("%s:%d:%x", name, userID, hashString(input))
}

// hashString creates a SHA-256 hash of a string for cache keys.
// Using SHA-256 provides better distribution and collision resistance
// compared to the simple FNV-1a hash.
func hashString(s string) string {
	hash := sha256.Sum256([]byte(s))
	return hex.EncodeToString(hash[:]) // Use full hash for better distribution
}

// LRUCache is a simple LRU cache implementation using container/list for O(1) operations.
type LRUCache struct {
	mu    sync.Mutex
	size  int
	ttl   time.Duration
	items map[string]*list.Element
	lru   *list.List // front=least recent, back=most recent
}

type cacheItem struct {
	key        string
	value      string
	expiration time.Time
}

// NewLRUCache creates a new LRU cache.
func NewLRUCache(size int, ttl time.Duration) *LRUCache {
	return &LRUCache{
		size:  size,
		ttl:   ttl,
		items: make(map[string]*list.Element),
		lru:   list.New(),
	}
}

// Get retrieves a value from the cache.
func (c *LRUCache) Get(key string) (string, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	elem, ok := c.items[key]
	if !ok {
		return "", false
	}

	item, ok := elem.Value.(*cacheItem)
	if !ok {
		return "", false
	}
	if time.Now().After(item.expiration) {
		c.lru.Remove(elem)
		delete(c.items, key)
		return "", false
	}

	// Move to back (most recently used) - O(1)
	c.lru.MoveToBack(elem)
	return item.value, true
}

// Set stores a value in the cache.
func (c *LRUCache) Set(key, value string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	item := &cacheItem{
		key:        key,
		value:      value,
		expiration: time.Now().Add(c.ttl),
	}

	if elem, ok := c.items[key]; ok {
		// Update existing - O(1)
		elem.Value = item
		c.lru.MoveToBack(elem)
		return
	}

	// Evict if at capacity - O(1)
	if c.lru.Len() >= c.size {
		oldest := c.lru.Front()
		if oldest != nil {
			c.lru.Remove(oldest)
			if item, ok := oldest.Value.(*cacheItem); ok {
				delete(c.items, item.key)
			}
		}
	}

	// Add new - O(1)
	elem := c.lru.PushBack(item)
	c.items[key] = elem
}

// NormalSessionStats represents accumulated session statistics.
type NormalSessionStats struct {
	mu sync.Mutex

	// Session identification
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
	AgentType string    `json:"agent_type"`
	ModelUsed string    `json:"model_used"`

	// Token usage
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
	CacheReadTokens  int `json:"cache_read_tokens,omitempty"`
	CacheWriteTokens int `json:"cache_write_tokens,omitempty"`

	// Timing (milliseconds)
	ThinkingDurationMs   int64 `json:"thinking_duration_ms"`
	GenerationDurationMs int64 `json:"generation_duration_ms"`
	TotalDurationMs      int64 `json:"total_duration_ms"`

	// Tool usage
	ToolCallCount int      `json:"tool_call_count"`
	ToolsUsed     []string `json:"tools_used,omitempty"`

	// Cost estimation
	TotalCostMilliCents int64 `json:"total_cost_milli_cents"`
}

// NewNormalSessionStats creates a new session stats.
func NewNormalSessionStats(agentType string) *NormalSessionStats {
	return &NormalSessionStats{
		StartTime: time.Now(),
		AgentType: agentType,
		ToolsUsed: make([]string, 0),
	}
}

// GetStatsSnapshot returns a thread-safe snapshot.
func (s *NormalSessionStats) GetStatsSnapshot() *NormalSessionStats {
	s.mu.Lock()
	defer s.mu.Unlock()

	return &NormalSessionStats{
		StartTime:            s.StartTime,
		EndTime:              s.EndTime,
		AgentType:            s.AgentType,
		ModelUsed:            s.ModelUsed,
		PromptTokens:         s.PromptTokens,
		CompletionTokens:     s.CompletionTokens,
		TotalTokens:          s.TotalTokens,
		CacheReadTokens:      s.CacheReadTokens,
		CacheWriteTokens:     s.CacheWriteTokens,
		ThinkingDurationMs:   s.ThinkingDurationMs,
		GenerationDurationMs: s.GenerationDurationMs,
		TotalDurationMs:      s.TotalDurationMs,
		ToolCallCount:        s.ToolCallCount,
		ToolsUsed:            s.ToolsUsed,
		TotalCostMilliCents:  s.TotalCostMilliCents,
	}
}
