package runner

import (
	"fmt"
)

// TruncateString truncates a string to a maximum length for logging.
func TruncateString(s string, maxLen int) string {
	if s == "" {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen]) + "..."
}

// SummarizeInput creates a human-readable summary of tool input.
// Uses rune-level truncation to avoid creating invalid UTF-8.
func SummarizeInput(input map[string]any) string {
	if input == nil {
		return ""
	}
	// Extract common fields for summary
	if command, ok := input["command"].(string); ok && command != "" {
		return TruncateString(command, 50)
	}
	if query, ok := input["query"].(string); ok && query != "" {
		return TruncateString(query, 50)
	}
	if path, ok := input["path"].(string); ok && path != "" {
		return "file: " + path
	}
	// Fallback to truncated string representation
	if len(input) == 0 {
		return ""
	}
	// Simple truncated representation
	str := fmt.Sprintf("%+v", input)
	return TruncateString(str, 100)
}

// StreamMessage represents a single event in the stream-json format.
// StreamMessage 表示 stream-json 格式中的单个事件。
type StreamMessage struct {
	Message      *AssistantMessage `json:"message,omitempty"`
	Input        map[string]any    `json:"input,omitempty"`
	Type         string            `json:"type"`
	Timestamp    string            `json:"timestamp,omitempty"`
	SessionID    string            `json:"session_id,omitempty"`
	Role         string            `json:"role,omitempty"`
	Name         string            `json:"name,omitempty"`
	Output       string            `json:"output,omitempty"`
	Status       string            `json:"status,omitempty"`
	Error        string            `json:"error,omitempty"`
	Content      []ContentBlock    `json:"content,omitempty"`
	Duration     int               `json:"duration_ms,omitempty"`
	Subtype      string            `json:"subtype,omitempty"`        // For "result" message
	IsError      bool              `json:"is_error,omitempty"`       // For "result" message
	TotalCostUSD float64           `json:"total_cost_usd,omitempty"` // For "result" message
	Usage        *UsageStats       `json:"usage,omitempty"`          // For "result" message
	Result       string            `json:"result,omitempty"`         // For "result" message
}

// UsageStats represents token usage from result messages.
// UsageStats 表示 result 消息中的 token 使用情况。
type UsageStats struct {
	InputTokens           int32 `json:"input_tokens"`
	OutputTokens          int32 `json:"output_tokens"`
	CacheWriteInputTokens int32 `json:"cache_creation_input_tokens,omitempty"`
	CacheReadInputTokens  int32 `json:"cache_read_input_tokens,omitempty"`
}

// GetContentBlocks returns the content blocks, checking both direct and nested locations.
// GetContentBlocks 返回内容块，同时检查直接和嵌套位置。
func (m *StreamMessage) GetContentBlocks() []ContentBlock {
	if m.Message != nil && len(m.Message.Content) > 0 {
		return m.Message.Content
	}
	return m.Content
}

// AssistantMessage represents the nested message structure in assistant events.
// AssistantMessage 表示 assistant 事件中的嵌套消息结构。
type AssistantMessage struct {
	ID      string         `json:"id,omitempty"`
	Type    string         `json:"type,omitempty"`
	Role    string         `json:"role,omitempty"`
	Content []ContentBlock `json:"content,omitempty"`
}

// ContentBlock represents a content block in stream-json format.
// ContentBlock 表示 stream-json 格式中的内容块。
type ContentBlock struct {
	Type    string         `json:"type"`
	Text    string         `json:"text,omitempty"`
	Name    string         `json:"name,omitempty"`
	ID      string         `json:"id,omitempty"`
	Input   map[string]any `json:"input,omitempty"`
	Content string         `json:"content,omitempty"`
	IsError bool           `json:"is_error,omitempty"`
}

// Config defines mode-specific configuration for CCRunner execution.
// Config 定义 CCRunner 执行的模式特定配置。
type Config struct {
	Mode           string // "geek" | "evolution"
	WorkDir        string // Working directory for CLI
	ConversationID int64  // Database conversation ID for deterministic UUID v5 mapping
	SessionID      string // Session identifier (derived from ConversationID if empty)
	UserID         int32  // User ID for logging/context
	SystemPrompt   string // Mode-specific system prompt
	DeviceContext  string // Device/browser context JSON

	// Security / Permission Control
	// 安全/权限控制
	PermissionMode string // "default", "bypassPermissions", etc.

	// Evolution Mode specific
	// 进化模式专用
	AllowedPaths   []string // Path whitelist (evolution mode)
	ForbiddenPaths []string // Path blacklist (evolution mode)
}

// CCRunnerConfig is an alias for Config for backward compatibility.
// Deprecated: Use Config directly.
type CCRunnerConfig = Config

// ProcessingPhase represents the current phase of agent processing.
// ProcessingPhase 表示代理处理的当前阶段。
type ProcessingPhase string

const (
	// PhaseAnalyzing is the initial analysis phase.
	PhaseAnalyzing ProcessingPhase = "analyzing"
	// PhasePlanning is the planning phase for multi-step tasks.
	PhasePlanning ProcessingPhase = "planning"
	// PhaseRetrieving is the information retrieval phase.
	PhaseRetrieving ProcessingPhase = "retrieving"
	// PhaseSynthesizing is the final response generation phase.
	PhaseSynthesizing ProcessingPhase = "synthesizing"
)

// PhaseChangeEvent represents a phase change event.
// PhaseChangeEvent 表示阶段变更事件。
type PhaseChangeEvent struct {
	Phase            ProcessingPhase `json:"phase"`
	PhaseNumber      int             `json:"phase_number"`
	TotalPhases      int             `json:"total_phases"`
	EstimatedSeconds int             `json:"estimated_seconds"`
}

// ProgressEvent represents a progress update event.
// ProgressEvent 表示进度更新事件。
type ProgressEvent struct {
	Percent              int `json:"percent"`
	EstimatedSeconds     int `json:"estimated_seconds"`
	EstimatedTimeSeconds int `json:"estimated_time_seconds"`
}

// Event type constants for streaming events.
const (
	// EventTypePhaseChange is the event type for phase changes.
	EventTypePhaseChange = "phase_change"
	// EventTypeProgress is the event type for progress updates.
	EventTypeProgress = "progress"
	// EventTypeThinking is the event type for thinking updates.
	EventTypeThinking = "thinking"
	// EventTypeToolUse is the event type for tool invocations.
	EventTypeToolUse = "tool_use"
	// EventTypeToolResult is the event type for tool results.
	EventTypeToolResult = "tool_result"
	// EventTypeAnswer is the event type for final answers.
	EventTypeAnswer = "answer"
	// EventTypeError is the event type for errors.
	EventTypeError = "error"
)
