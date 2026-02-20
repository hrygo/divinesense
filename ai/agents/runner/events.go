package runner

import (
	"time"
)

// EventMeta contains detailed metadata for streaming events.
type EventMeta struct {
	// Timing
	DurationMs      int64 `json:"duration_ms"`       // Event duration in milliseconds
	TotalDurationMs int64 `json:"total_duration_ms"` // Total elapsed time since start

	// Tool call info
	ToolName string `json:"tool_name"` // Tool name (e.g., "bash", "editor_write", "memo_search")
	ToolID   string `json:"tool_id"`   // Unique tool call ID
	Status   string `json:"status"`    // "running", "success", "error"
	ErrorMsg string `json:"error_msg"` // Error message if status=error

	// Token usage (when available)
	InputTokens      int32 `json:"input_tokens"`       // Input tokens
	OutputTokens     int32 `json:"output_tokens"`      // Output tokens
	CacheWriteTokens int32 `json:"cache_write_tokens"` // Cache write tokens
	CacheReadTokens  int32 `json:"cache_read_tokens"`  // Cache read tokens

	// Summaries for UI
	InputSummary  string `json:"input_summary"`  // Human-readable input summary
	OutputSummary string `json:"output_summary"` // Truncated output preview

	// File operations
	FilePath  string `json:"file_path"`  // Affected file path
	LineCount int32  `json:"line_count"` // Number of lines affected

	// Progress (for long-running operations)
	Progress    int32 `json:"progress"`     // Progress percentage (0-100)
	TotalSteps  int32 `json:"total_steps"`  // Total number of steps (for multi-stage operations)
	CurrentStep int32 `json:"current_step"` // Current step number
}

// EventWithMeta extends the basic event with metadata for observability.
// This type is used by executors (DirectExecutor, ReActExecutor, PlanningExecutor)
// and CCRunner to send detailed event metadata to the frontend.
type EventWithMeta struct {
	EventType string     // Event type (thinking, tool_use, tool_result, etc.)
	EventData string     // Event data content
	Meta      *EventMeta // Enhanced metadata (never nil when created via NewEventWithMeta)
}

// NewEventWithMeta creates a new EventWithMeta with guaranteed non-nil Meta.
// If meta is nil, an empty EventMeta{} is used instead.
// This prevents nil pointer dereferences when accessing Meta fields.
func NewEventWithMeta(eventType, eventData string, meta *EventMeta) *EventWithMeta {
	if meta == nil {
		meta = &EventMeta{}
	}
	return &EventWithMeta{
		EventType: eventType,
		EventData: eventData,
		Meta:      meta,
	}
}

// SessionStatsData represents the final session statistics sent to the frontend and stored in database.
type SessionStatsData struct {
	SessionID            string   `json:"session_id"`
	ConversationID       int64    `json:"conversation_id"` // Database conversation ID
	UserID               int32    `json:"user_id"`
	AgentType            string   `json:"agent_type"` // "geek", "evolution", etc.
	StartTime            int64    `json:"start_time"` // Unix timestamp
	EndTime              int64    `json:"end_time"`   // Unix timestamp
	TotalDurationMs      int64    `json:"total_duration_ms"`
	ThinkingDurationMs   int64    `json:"thinking_duration_ms"`
	ToolDurationMs       int64    `json:"tool_duration_ms"`
	GenerationDurationMs int64    `json:"generation_duration_ms"`
	InputTokens          int32    `json:"input_tokens"`
	OutputTokens         int32    `json:"output_tokens"`
	CacheWriteTokens     int32    `json:"cache_write_tokens"`
	CacheReadTokens      int32    `json:"cache_read_tokens"`
	TotalTokens          int32    `json:"total_tokens"`
	ToolCallCount        int32    `json:"tool_call_count"`
	ToolsUsed            []string `json:"tools_used"`
	FilesModified        int32    `json:"files_modified"`
	FilePaths            []string `json:"file_paths"`
	TotalCostUSD         float64  `json:"total_cost_usd"`
	ModelUsed            string   `json:"model_used"`
	IsError              bool     `json:"is_error"`
	ErrorMessage         string   `json:"error_message,omitempty"`
}

// AgentSessionStatsForStorage represents session statistics for database persistence.
// This is a bridge type that converts SessionStatsData (with Unix timestamps)
// to the format expected by the store layer (with time.Time).
type AgentSessionStatsForStorage struct {
	SessionID            string
	ConversationID       int64
	UserID               int32
	AgentType            string
	StartTime            time.Time
	EndedAt              time.Time
	TotalDurationMs      int64
	ThinkingDurationMs   int64
	ToolDurationMs       int64
	GenerationDurationMs int64
	InputTokens          int32
	OutputTokens         int32
	CacheWriteTokens     int32
	CacheReadTokens      int32
	TotalTokens          int32
	TotalCostUSD         float64
	ToolCallCount        int32
	ToolsUsed            []string
	FilesModified        int32
	FilePaths            []string
	ModelUsed            string
	IsError              bool
	ErrorMessage         string
}

// ToAgentSessionStats converts SessionStatsData to AgentSessionStatsForStorage.
// It converts Unix timestamps to time.Time for database storage.
func (d *SessionStatsData) ToAgentSessionStats() *AgentSessionStatsForStorage {
	return &AgentSessionStatsForStorage{
		SessionID:            d.SessionID,
		ConversationID:       d.ConversationID,
		UserID:               d.UserID,
		AgentType:            d.AgentType,
		StartTime:            time.Unix(d.StartTime, 0),
		EndedAt:              time.Unix(d.EndTime, 0),
		TotalDurationMs:      d.TotalDurationMs,
		ThinkingDurationMs:   d.ThinkingDurationMs,
		ToolDurationMs:       d.ToolDurationMs,
		GenerationDurationMs: d.GenerationDurationMs,
		InputTokens:          d.InputTokens,
		OutputTokens:         d.OutputTokens,
		CacheWriteTokens:     d.CacheWriteTokens,
		CacheReadTokens:      d.CacheReadTokens,
		TotalTokens:          d.TotalTokens,
		TotalCostUSD:         d.TotalCostUSD,
		ToolCallCount:        d.ToolCallCount,
		ToolsUsed:            d.ToolsUsed,
		FilesModified:        d.FilesModified,
		FilePaths:            d.FilePaths,
		ModelUsed:            d.ModelUsed,
		IsError:              d.IsError,
		ErrorMessage:         d.ErrorMessage,
	}
}
