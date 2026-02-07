package agent

import (
	"strings"
	"sync"
	"time"

	"github.com/hrygo/divinesense/ai"
)

// NormalSessionStats represents the accumulated statistics for a single agent session in normal mode.
// A session may consist of multiple LLM calls (e.g., ReAct loops with tool calls).
// NormalSessionStats 表示普通模式下单个代理会话的累积统计数据。
// 一个会话可能包含多次 LLM 调用（例如，带有工具调用的 ReAct 循环）。
type NormalSessionStats struct {
	mu sync.Mutex

	// Session identification
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
	AgentType string    `json:"agent_type"`
	ModelUsed string    `json:"model_used"`

	// Token usage (accumulated across all LLM calls)
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
	CacheReadTokens  int `json:"cache_read_tokens,omitempty"`
	CacheWriteTokens int `json:"cache_write_tokens,omitempty"`

	// Timing (milliseconds)
	ThinkingDurationMs   int64 `json:"thinking_duration_ms"`   // Time to first token (avg)
	GenerationDurationMs int64 `json:"generation_duration_ms"` // Content generation time (total)
	TotalDurationMs      int64 `json:"total_duration_ms"`      // Wall-clock time

	// Tool usage
	ToolCallCount int      `json:"tool_call_count"`
	ToolsUsed     []string `json:"tools_used,omitempty"`

	// Cost estimation (in milli-cents: 1/1000 of a US cent, or 1/100000 USD)
	// For DeepSeek: $0.14/M input, $0.28/M output = 0.014¢/1K input, 0.028¢/1K output
	TotalCostMilliCents int64 `json:"total_cost_milli_cents"`
}

// BaseParrot provides common statistics accumulation functionality for all normal mode agents.
// Normal mode agents (MemoParrot, ScheduleParrotV2, AmazingParrot) embed BaseParrot
// to track LLM call statistics across their execution sessions.
//
// BaseParrot is thread-safe and uses a mutex to protect the stats state.
//
// BaseParrot 为所有普通模式代理提供通用的统计累积功能。
// 普通模式代理（MemoParrot、ScheduleParrotV2、AmazingParrot）嵌入 BaseParrot
// 以跟踪其执行会话期间的 LLM 调用统计。
//
// BaseParrot 是线程安全的，使用互斥锁保护统计状态。
type BaseParrot struct {
	mu    sync.Mutex
	stats *NormalSessionStats
}

// NewBaseParrot creates a new BaseParrot with initialized stats.
// NewBaseParrot 创建一个具有初始化统计数据的 BaseParrot。
func NewBaseParrot(agentType string) *BaseParrot {
	return &BaseParrot{
		stats: &NormalSessionStats{
			StartTime: time.Now(),
			AgentType: agentType,
			ToolsUsed: make([]string, 0),
		},
	}
}

// GetStatsSnapshot returns a thread-safe snapshot of the current statistics.
// GetStatsSnapshot 返回当前统计数据的线程安全快照。
func (s *NormalSessionStats) GetStatsSnapshot() *NormalSessionStats {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Return a copy without the mutex to avoid copylocks warning
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

// TrackLLMCall records statistics from a single LLM call.
// It accumulates tokens, timing, and cost information into the session stats.
//
// TrackLLMCall 记录单次 LLM 调用的统计信息。
// 它将 tokens、时间和成本信息累积到会话统计中。
func (b *BaseParrot) TrackLLMCall(stats *ai.LLMCallStats, model string) {
	if stats == nil {
		return
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	b.stats.PromptTokens += stats.PromptTokens
	b.stats.CompletionTokens += stats.CompletionTokens
	b.stats.TotalTokens += stats.TotalTokens
	b.stats.CacheReadTokens += stats.CacheReadTokens
	b.stats.CacheWriteTokens += stats.CacheWriteTokens

	b.stats.ThinkingDurationMs += stats.ThinkingDurationMs
	b.stats.GenerationDurationMs += stats.GenerationDurationMs
	b.stats.TotalDurationMs += stats.TotalDurationMs

	if b.stats.ModelUsed == "" {
		b.stats.ModelUsed = model
	}

	// Calculate cost in milli-cents (1/1000 of a cent)
	// DeepSeek pricing (example): $0.14/M input, $0.28/M output
	// NOTE: Pricing is hardcoded per model for common providers.
	// To add a new provider/model, extend calculateCost() method with its pricing.
	costMilliCents := b.calculateCost(stats, model)
	b.stats.TotalCostMilliCents += costMilliCents
}

// calculateCost estimates the cost of an LLM call in milli-cents.
// Currently uses DeepSeek pricing as default:
// - Input: $0.14 per million tokens = 0.014¢ per 1K tokens = 14 milli-cents per 1K tokens
// - Output: $0.28 per million tokens = 0.028¢ per 1K tokens = 28 milli-cents per 1K tokens
//
// calculateCost 估算 LLM 调用的成本（单位：毫分）。
// 当前使用 DeepSeek 定价作为默认值。
func (b *BaseParrot) calculateCost(stats *ai.LLMCallStats, model string) int64 {
	// Default pricing (DeepSeek): input $0.14/M, output $0.28/M
	var inputPricePerMillion, outputPricePerMillion float64

	modelLower := strings.ToLower(model)
	switch {
	case strings.Contains(modelLower, "deepseek"):
		inputPricePerMillion = 0.14  // $0.14 per million
		outputPricePerMillion = 0.28 // $0.28 per million
	case strings.Contains(modelLower, "gpt-4"):
		inputPricePerMillion = 2.50   // $2.50 per million
		outputPricePerMillion = 10.00 // $10.00 per million
	case strings.Contains(modelLower, "gpt-3.5"):
		inputPricePerMillion = 0.15  // $0.15 per million
		outputPricePerMillion = 0.60 // $0.60 per million
	default:
		// Default to DeepSeek pricing
		inputPricePerMillion = 0.14
		outputPricePerMillion = 0.28
	}

	// Calculate cost in USD, then convert to milli-cents
	// 1 USD = 100 cents = 100000 milli-cents
	inputCost := (float64(stats.PromptTokens) / 1_000_000) * inputPricePerMillion
	outputCost := (float64(stats.CompletionTokens) / 1_000_000) * outputPricePerMillion
	totalCostUSD := inputCost + outputCost

	// Convert to milli-cents (1 USD = 100000 milli-cents)
	return int64(totalCostUSD * 100000)
}

// TrackToolCall records a tool invocation.
// TrackToolCall 记录工具调用。
func (b *BaseParrot) TrackToolCall(toolName string) {
	if toolName == "" {
		return
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	b.stats.ToolCallCount++
	// Avoid duplicate tool names
	for _, t := range b.stats.ToolsUsed {
		if t == toolName {
			return
		}
	}
	b.stats.ToolsUsed = append(b.stats.ToolsUsed, toolName)
}

// RecordAgentStats transfers accumulated stats from the Agent framework to BaseParrot.
// This is used by agents (e.g., SchedulerAgentV2) that embed both Agent and BaseParrot
// to ensure stats from the Agent's LLM calls are captured in the session stats.
//
// Note: agentStats should be a snapshot (e.g., from Agent.GetStats()) which is safe to pass.
//
// RecordAgentStats 将 Agent 框架的累积统计数据传输到 BaseParrot。
// 这被同时嵌入 Agent 和 BaseParrot 的代理（例如 SchedulerAgentV2）使用，
// 以确保来自 Agent LLM 调用的统计数据被捕获到会话统计中。
func (b *BaseParrot) RecordAgentStats(agentStats *AgentStats) {
	if agentStats == nil {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()

	b.stats.PromptTokens += agentStats.PromptTokens
	b.stats.CompletionTokens += agentStats.CompletionTokens
	b.stats.TotalTokens += agentStats.PromptTokens + agentStats.CompletionTokens
	b.stats.CacheReadTokens += agentStats.TotalCacheRead
	b.stats.CacheWriteTokens += agentStats.TotalCacheWrite

	// ToolCallCount is tracked separately via TrackToolCall in the wrapped callback.
	// The Agent framework tracks all tool calls internally, but we don't use that
	// count to avoid double-counting with the callback-based tracking.
}

// Finalize marks the session as complete and returns the final stats.
// Finalize 标记会话完成并返回最终统计数据。
func (b *BaseParrot) Finalize() *NormalSessionStats {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.stats.EndTime = time.Now()
	if b.stats.TotalDurationMs == 0 {
		b.stats.TotalDurationMs = b.stats.EndTime.Sub(b.stats.StartTime).Milliseconds()
	}

	return b.stats
}

// GetSessionStats returns a copy of the current session stats.
// GetSessionStats 返回当前会话统计的副本。
func (b *BaseParrot) GetSessionStats() *NormalSessionStats {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Return a copy to avoid external mutations and copying the mutex
	return &NormalSessionStats{
		StartTime:            b.stats.StartTime,
		EndTime:              b.stats.EndTime,
		AgentType:            b.stats.AgentType,
		ModelUsed:            b.stats.ModelUsed,
		PromptTokens:         b.stats.PromptTokens,
		CompletionTokens:     b.stats.CompletionTokens,
		TotalTokens:          b.stats.TotalTokens,
		CacheReadTokens:      b.stats.CacheReadTokens,
		CacheWriteTokens:     b.stats.CacheWriteTokens,
		ThinkingDurationMs:   b.stats.ThinkingDurationMs,
		GenerationDurationMs: b.stats.GenerationDurationMs,
		TotalDurationMs:      b.stats.TotalDurationMs,
		ToolCallCount:        b.stats.ToolCallCount,
		ToolsUsed:            append([]string{}, b.stats.ToolsUsed...),
		TotalCostMilliCents:  b.stats.TotalCostMilliCents,
	}
}

// Reset clears the current stats and starts a new session.
// Reset 清除当前统计数据并开始新会话。
func (b *BaseParrot) Reset(agentType string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.stats = &NormalSessionStats{
		StartTime: time.Now(),
		AgentType: agentType,
		ToolsUsed: make([]string, 0),
	}
}
