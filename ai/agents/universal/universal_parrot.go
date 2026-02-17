// Package universal provides the UniversalParrot - a configuration-driven
// parrot that can mimic any existing parrot through configuration.
package universal

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/hrygo/divinesense/ai"
	"github.com/hrygo/divinesense/ai/agents"
	"github.com/hrygo/divinesense/ai/cache"
)

const (
	// DefaultTimezone is the default timezone for time context calculations.
	DefaultTimezone = "Asia/Shanghai"

	// Default model pricing (per 1M tokens)
	// Using SiliconFlow/DeepSeek pricing as default
	defaultInputCostPerMillion  = 0.27
	defaultOutputCostPerMillion = 2.25
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
	cache *cache.StringLRUCache

	// Statistics
	stats *agent.NormalSessionStats

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
	var lruCache *cache.StringLRUCache
	if config.EnableCache {
		cacheSize := config.CacheSize
		if cacheSize <= 0 {
			cacheSize = 100 // Default
		}
		lruCache = cache.NewStringLRUCache(cacheSize, config.CacheTTL)
	}

	return &UniversalParrot{
		config:   config,
		strategy: strategy,
		llm:      llm,
		tools:    tools,
		cache:    lruCache,
		stats:    NewNormalSessionStats(config.Name),
		userID:   userID,
		timezone: "Local",
	}, nil
}

// Execute implements agent.ParrotAgent.
// history is optional - pass nil if no conversation history is needed.
func (p *UniversalParrot) Execute(
	ctx context.Context,
	userInput string,
	history []string,
	callback agent.EventCallback,
) error {
	startTime := time.Now() // No lock needed: local variable, thread-safe

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
		return agent.NewParrotError(p.config.Name, "Execute", err)
	}

	// Cache result
	if p.cache != nil && result != "" {
		cacheKey := p.generateCacheKey(p.config.Name, p.userID, userInput)
		p.cache.SetWithDefaultTTL(cacheKey, result)
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
func (p *UniversalParrot) GetSessionStats() *agent.NormalSessionStats {
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
		// Debug: log first 200 chars of system prompt to verify config loading
		previewLen := 200
		if len(systemContent) < previewLen {
			previewLen = len(systemContent)
		}
		slog.Debug("using system prompt", "parrot", p.config.Name, "prompt_preview", systemContent[:previewLen])
		messages = append(messages, ai.Message{
			Role:    "system",
			Content: systemContent,
		})
	} else {
		slog.Warn("no system prompt configured, using fallback", "parrot", p.config.Name)
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
// Also replaces {{.BaseURL}} placeholder with the configured base URL.
func (p *UniversalParrot) enhanceSystemPromptWithDate(basePrompt string, tc *TimeContext) string {
	// If no timeContext provided, build one
	if tc == nil {
		tc = p.buildTimeContext()
	}

	// Build time context
	dateContext := fmt.Sprintf(`

<time_context>
%s
</time_context>

Use the JSON above for time calculations. Output format: ISO8601 (YYYY-MM-DDTHH:mm:ss+08:00)
`,
		tc.FormatAsJSONBlock(),
	)

	result := basePrompt + dateContext

	// Replace {{.BaseURL}} placeholder with configured base URL
	if p.config.BaseURL != "" {
		result = strings.ReplaceAll(result, "{{.BaseURL}}", p.config.BaseURL)
	}

	return result
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
	p.stats.ToolDurationMs += execStats.ToolDurationMs
	p.stats.ThinkingDurationMs += execStats.ThinkingDuration
	p.stats.TotalDurationMs += duration.Milliseconds()

	// Calculate cost (in milli-cents: 1/100000 USD)
	// Cost = (input_tokens * input_price + output_tokens * output_price) / 1M * 100000
	inputCost := float64(execStats.PromptTokens) * defaultInputCostPerMillion
	outputCost := float64(execStats.CompletionTokens) * defaultOutputCostPerMillion
	totalCost := (inputCost + outputCost) / 1_000_000 * 100000
	p.stats.TotalCostMilliCents += int64(totalCost)

	// Merge unique tool names
	for _, tool := range execStats.ToolsUsed {
		hasTool := false
		for _, existing := range p.stats.ToolsUsed {
			if existing == tool {
				hasTool = true
				break
			}
		}
		if !hasTool {
			p.stats.ToolsUsed = append(p.stats.ToolsUsed, tool)
		}
	}
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

// NewNormalSessionStats creates a new session stats.
func NewNormalSessionStats(agentType string) *agent.NormalSessionStats {
	return &agent.NormalSessionStats{
		StartTime: time.Now(),
		AgentType: agentType,
		ToolsUsed: make([]string, 0),
	}
}
