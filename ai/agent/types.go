package agent

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"time"
)

// ParrotAgent is the interface for all parrot agents.
// ParrotAgent 是所有鹦鹉代理的接口。
type ParrotAgent interface {
	// Name returns the name of the parrot agent.
	Name() string

	// ExecuteWithCallback executes the agent with callback support for real-time feedback.
	// ExecuteWithCallback 执行代理并支持回调以实现实时反馈。
	ExecuteWithCallback(ctx context.Context, userInput string, history []string, callback EventCallback) error

	// SelfDescribe returns the parrot's self-cognition (metacognition) information.
	// SelfDescribe 返回鹦鹉的自我认知（元认知）信息。
	SelfDescribe() *ParrotSelfCognition
}

// SessionStatsProvider is an optional interface for agents that can provide session statistics.
// SessionStatsProvider 是可选接口，用于可以提供会话统计数据的代理。
// Only GeekParrot and EvolutionParrot (which use CCRunner) implement this interface.
// 只有 GeekParrot 和 EvolutionParrot（使用 CCRunner）实现此接口。
type SessionStatsProvider interface {
	// GetSessionStats returns the session statistics from the last execution.
	// GetSessionStats 返回上次执行的会话统计数据。
	GetSessionStats() *SessionStats
}

// ParrotSelfCognition represents a parrot's metacognitive understanding of itself.
// ParrotSelfCognition 表示鹦鹉对自己的元认知理解。
type ParrotSelfCognition struct {
	AvianIdentity       *AvianIdentity       `json:"avian_identity"`
	EmotionalExpression *EmotionalExpression `json:"emotional_expression,omitempty"`
	WorkingStyle        string               `json:"working_style"`
	Title               string               `json:"title"`
	Emoji               string               `json:"emoji"`
	Name                string               `json:"name"`
	SelfIntroduction    string               `json:"self_introduction"`
	FunFact             string               `json:"fun_fact"`
	AvianBehaviors      []string             `json:"avian_behaviors,omitempty"`
	Personality         []string             `json:"personality"`
	Capabilities        []string             `json:"capabilities"`
	Limitations         []string             `json:"limitations"`
	FavoriteTools       []string             `json:"favorite_tools"`
}

// AvianIdentity represents the parrot's cognition of its avian nature.
// AvianIdentity 表示鹦鹉对其鸟类本质的认知。
type AvianIdentity struct {
	Species          string   `json:"species"`
	Origin           string   `json:"origin"`
	SymbolicMeaning  string   `json:"symbolic_meaning"`
	AvianPhilosophy  string   `json:"avian_philosophy"`
	NaturalAbilities []string `json:"natural_abilities"`
}

// EmotionalExpression defines how a parrot expresses emotions.
// EmotionalExpression 定义鹦鹉的情感表达方式。
type EmotionalExpression struct {
	SoundEffects map[string]string `json:"sound_effects"`
	MoodTriggers map[string]string `json:"mood_triggers,omitempty"`
	DefaultMood  string            `json:"default_mood"`
	Catchphrases []string          `json:"catchphrases"`
}

// EventCallback is the callback function type for agent events.
// EventCallback 是代理事件的回调函数类型。
//
// The callback receives:
//   - eventType: The type of event (e.g., "thinking", "tool_use", "tool_result", "answer", "error")
//   - eventData: The event data (can be a struct, string, or nil)
//
// 返回错误将中止代理执行。
type EventCallback func(eventType string, eventData interface{}) error

// SafeCallbackFunc is a callback that logs errors instead of returning them.
// Use SafeCallback to wrap an EventCallback for non-critical events.
// SafeCallbackFunc 是一个记录错误而不是返回错误的回调函数。
// 使用 SafeCallback 包装 EventCallback 用于非关键事件。
type SafeCallbackFunc func(eventType string, eventData interface{})

// SafeCallback wraps an EventCallback to log errors instead of propagating them.
// Use this for non-critical callbacks where errors should not interrupt execution.
// SafeCallback 包装 EventCallback 以记录错误而不是传播它们。
// 用于错误不应中断执行的非关键回调。
//
// Usage:
//
//	callbackSafe := SafeCallback(callback)
//	callbackSafe(EventTypeAnswer, result) // Error is logged, not returned
func SafeCallback(callback EventCallback) SafeCallbackFunc {
	if callback == nil {
		return nil
	}
	return func(eventType string, eventData interface{}) {
		// Execute callback and log errors instead of returning them
		if err := callback(eventType, eventData); err != nil {
			// Log the callback error but don't propagate it
			// This prevents callback failures from interrupting agent execution
			// Use Background context as this is independent logging with no deadline
			slog.Default().LogAttrs(context.Background(), slog.LevelWarn,
				"callback failed (non-critical)",
				slog.String("event_type", eventType),
				slog.Any("error", err),
			)
		}
	}
}

// 常用事件类型.
const (
	EventTypeThinking     = "thinking"      // Agent is thinking
	EventTypeToolUse      = "tool_use"      // Agent is using a tool
	EventTypeToolResult   = "tool_result"   // Tool execution result
	EventTypeAnswer       = "answer"        // Final answer from agent
	EventTypeError        = "error"         // Error occurred
	EventTypeDangerBlock  = "danger_block"  // Dangerous operation was blocked
	EventTypeSessionStats = "session_stats" // Session completion statistics (cost, tokens, duration)

	// Memo-specific events.
	EventTypeMemoQueryResult = "memo_query_result" // Memo search results

	// Schedule-specific events.
	EventTypeScheduleQueryResult = "schedule_query_result" // Schedule query results
	EventTypeScheduleUpdated     = "schedule_updated"      // Schedule created/updated

	// UI 工具事件 - 用于生成式 UI.
	EventTypeUIScheduleSuggestion = "ui_schedule_suggestion" // Suggested schedule for confirmation
	EventTypeUITimeSlotPicker     = "ui_time_slot_picker"    // Time slot selection
	EventTypeUIConflictResolution = "ui_conflict_resolution" // Conflict resolution options
	EventTypeUIQuickActions       = "ui_quick_actions"       // Quick action buttons
	EventTypeUIMemoPreview        = "ui_memo_preview"        // Memo preview cards
	EventTypeUIScheduleList       = "ui_schedule_list"       // Schedule list display

	// Progressive progress events (Issue #97).
	EventTypePhaseChange = "phase_change" // Processing phase changed
	EventTypeProgress    = "progress"     // Progress percentage update
)

// MemoQueryResultData represents the result of a memo search.
// MemoQueryResultData 表示笔记搜索的结果。
type MemoQueryResultData struct {
	Query string        `json:"query"`
	Memos []MemoSummary `json:"memos"`
	Count int           `json:"count"`
}

// MemoSummary represents a simplified memo for query results.
// MemoSummary 表示查询结果中的简化笔记。
type MemoSummary struct {
	UID     string  `json:"uid"`
	Content string  `json:"content"`
	Score   float32 `json:"score"`
}

// ScheduleQueryResultData represents the result of a schedule query.
// ScheduleQueryResultData 表示日程查询的结果。
type ScheduleQueryResultData struct {
	Query                string            `json:"query"`
	TimeRangeDescription string            `json:"time_range_description"`
	QueryType            string            `json:"query_type"`
	Schedules            []ScheduleSummary `json:"schedules"`
	Count                int               `json:"count"`
}

// ScheduleSummary represents a simplified schedule for query results.
// ScheduleSummary 表示查询结果中的简化日程。
type ScheduleSummary struct {
	UID            string `json:"uid"`
	Title          string `json:"title"`
	Location       string `json:"location,omitempty"`
	Status         string `json:"status"`
	StartTimestamp int64  `json:"start_ts"`
	EndTimestamp   int64  `json:"end_ts"`
	AllDay         bool   `json:"all_day"`
}

// SessionStatsData represents the final session statistics sent to the frontend and stored in database.
// SessionStatsData 表示发送给前端并存储到数据库的最终会话统计数据。
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

// ToAgentSessionStats converts SessionStatsData to store.AgentSessionStats for persistence.
func (s *SessionStatsData) ToAgentSessionStats() *AgentSessionStatsForStorage {
	return &AgentSessionStatsForStorage{
		SessionID:            s.SessionID,
		ConversationID:       s.ConversationID,
		UserID:               s.UserID,
		AgentType:            s.AgentType,
		StartTime:            time.Unix(s.StartTime, 0),
		EndedAt:              time.Unix(s.EndTime, 0),
		TotalDurationMs:      s.TotalDurationMs,
		ThinkingDurationMs:   s.ThinkingDurationMs,
		ToolDurationMs:       s.ToolDurationMs,
		GenerationDurationMs: s.GenerationDurationMs,
		InputTokens:          s.InputTokens,
		OutputTokens:         s.OutputTokens,
		CacheWriteTokens:     s.CacheWriteTokens,
		CacheReadTokens:      s.CacheReadTokens,
		TotalTokens:          s.TotalTokens,
		TotalCostUSD:         s.TotalCostUSD,
		ToolCallCount:        s.ToolCallCount,
		ToolsUsed:            s.ToolsUsed,
		FilesModified:        s.FilesModified,
		FilePaths:            s.FilePaths,
		ModelUsed:            s.ModelUsed,
		IsError:              s.IsError,
		ErrorMessage:         s.ErrorMessage,
	}
}

// AgentSessionStatsForStorage is a storage-compatible version of SessionStatsData.
// AgentSessionStatsForStorage 是 SessionStatsData 的存储兼容版本。
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

// ProcessingPhase represents a processing phase for progressive progress feedback.
// ProcessingPhase 表示渐进式进度反馈的处理阶段。
type ProcessingPhase string

const (
	PhaseAnalyzing    ProcessingPhase = "analyzing"    // Understanding the user request
	PhasePlanning     ProcessingPhase = "planning"     // Determining the approach
	PhaseRetrieving   ProcessingPhase = "retrieving"   // Searching for relevant data
	PhaseSynthesizing ProcessingPhase = "synthesizing" // Generating the response
)

// PhaseChangeEvent represents a phase change event.
// PhaseChangeEvent 表示阶段变更事件。
type PhaseChangeEvent struct {
	Phase            ProcessingPhase `json:"phase"`
	PhaseNumber      int             `json:"phase_number"`      // 1-4
	TotalPhases      int             `json:"total_phases"`      // Always 4
	EstimatedSeconds int             `json:"estimated_seconds"` // Estimated time for this phase
}

// ProgressEvent represents a progress update event.
// ProgressEvent 表示进度更新事件。
type ProgressEvent struct {
	Percent              int `json:"percent"`                // 0-100
	EstimatedTimeSeconds int `json:"estimated_time_seconds"` // Remaining time
}

// ParrotStream is the interface for streaming responses to the client.
// ParrotStream 是向客户端流式传输响应的接口。
type ParrotStream interface {
	// Send sends an event to the client.
	// Send 向客户端发送一个事件。
	Send(eventType string, eventData interface{}) error

	// Close closes the stream.
	// Close 关闭流。
	Close() error
}

// ParrotStreamAdapter adapts Connect RPC server stream to ParrotStream interface.
// ParrotStreamAdapter 将 Connect RPC 服务端流适配到 ParrotStream 接口。
type ParrotStreamAdapter struct {
	// The actual stream implementation will be provided by the caller
	// 实际的流实现将由调用者提供
	sendFunc func(eventType string, eventData interface{}) error
}

// NewParrotStreamAdapter creates a new ParrotStreamAdapter.
// NewParrotStreamAdapter 创建一个新的 ParrotStreamAdapter。
func NewParrotStreamAdapter(sendFunc func(eventType string, eventData interface{}) error) *ParrotStreamAdapter {
	return &ParrotStreamAdapter{
		sendFunc: sendFunc,
	}
}

// Send sends an event through the adapter.
// Send 通过适配器发送事件。
func (a *ParrotStreamAdapter) Send(eventType string, eventData interface{}) error {
	if a.sendFunc == nil {
		return fmt.Errorf("send function not set")
	}
	return a.sendFunc(eventType, eventData)
}

// Close is a no-op for the adapter (the caller manages stream lifecycle).
// Close 对适配器来说是无操作（调用者管理流的生命周期）。
func (a *ParrotStreamAdapter) Close() error {
	return nil
}

// ParrotError represents an error from a parrot agent.
// ParrotError 表示来自鹦鹉代理的错误。
type ParrotError struct {
	Err       error
	AgentName string
	Operation string
}

// Error implements the error interface.
// Error 实现错误接口。
func (e *ParrotError) Error() string {
	if e == nil {
		return "<nil>"
	}
	return fmt.Sprintf("parrot %s: %s failed: %v", e.AgentName, e.Operation, e.Err)
}

// Unwrap returns the underlying error.
// Unwrap 返回底层错误。
func (e *ParrotError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

// NewParrotError creates a new ParrotError.
// NewParrotError 创建一个新的 ParrotError。
func NewParrotError(agentName, operation string, err error) *ParrotError {
	return &ParrotError{
		AgentName: agentName,
		Operation: operation,
		Err:       err,
	}
}

// UI Tool event data structures
// UI 工具事件数据结构

// UIScheduleSuggestionData represents a suggested schedule for user confirmation.
// UIScheduleSuggestionData 表示需要用户确认的建议日程。
type UIScheduleSuggestionData struct {
	Title       string  `json:"title"`
	Location    string  `json:"location,omitempty"`
	Description string  `json:"description,omitempty"`
	Reason      string  `json:"reason,omitempty"`
	SessionID   string  `json:"session_id,omitempty"`
	StartTs     int64   `json:"start_ts"`
	EndTs       int64   `json:"end_ts"`
	Confidence  float32 `json:"confidence"`
	AllDay      bool    `json:"all_day"`
}

// UITimeSlotData represents a single time slot option.
// UITimeSlotData 表示单个时间槽选项。
type UITimeSlotData struct {
	Label    string `json:"label"`
	Reason   string `json:"reason"`
	StartTs  int64  `json:"start_ts"`
	EndTs    int64  `json:"end_ts"`
	Duration int    `json:"duration"`
}

// UITimeSlotPickerData represents time slot options for user selection.
// UITimeSlotPickerData 表示供用户选择的时间槽选项。
type UITimeSlotPickerData struct {
	Reason     string           `json:"reason"`
	SessionID  string           `json:"session_id,omitempty"`
	Slots      []UITimeSlotData `json:"slots"`
	DefaultIdx int              `json:"default_idx"`
}

// UIConflictSchedule represents a conflicting schedule.
// UIConflictSchedule 表示冲突的日程。
type UIConflictSchedule struct {
	UID       string `json:"uid"`
	Title     string `json:"title"`
	Location  string `json:"location,omitempty"`
	StartTime int64  `json:"start_time"`
	EndTime   int64  `json:"end_time"`
	AllDay    bool   `json:"all_day"`
}

// UIConflictResolutionData represents conflict resolution options.
// UIConflictResolutionData 表示冲突解决选项。
type UIConflictResolutionData struct {
	AutoResolved         *UITimeSlotData          `json:"auto_resolved,omitempty"`
	NewSchedule          UIScheduleSuggestionData `json:"new_schedule"`
	SessionID            string                   `json:"session_id,omitempty"`
	ConflictingSchedules []UIConflictSchedule     `json:"conflicting_schedules"`
	SuggestedSlots       []UITimeSlotData         `json:"suggested_slots"`
	Actions              []string                 `json:"actions"`
}

// UIQuickActionData represents a quick action button.
// UIQuickActionData 表示快捷操作按钮。
type UIQuickActionData struct {
	ID          string `json:"id"`             // Action ID
	Label       string `json:"label"`          // Button label
	Description string `json:"description"`    // Action description
	Icon        string `json:"icon,omitempty"` // Optional icon name
	Prompt      string `json:"prompt"`         // What to send when clicked
}

// UIQuickActionsData represents quick action buttons for user.
// UIQuickActionsData 表示给用户的快捷操作按钮。
type UIQuickActionsData struct {
	Title       string              `json:"title"`
	Description string              `json:"description"`
	SessionID   string              `json:"session_id,omitempty"`
	Actions     []UIQuickActionData `json:"actions"`
}

// UIMemoPreviewData represents a memo preview card for generative UI.
// UIMemoPreviewData 表示生成式 UI 的笔记预览卡片。
type UIMemoPreviewData struct {
	UID        string   `json:"uid,omitempty"`
	Title      string   `json:"title"`
	Content    string   `json:"content"`
	Reason     string   `json:"reason,omitempty"`
	SessionID  string   `json:"session_id,omitempty"`
	Tags       []string `json:"tags,omitempty"`
	Confidence float32  `json:"confidence"`
}

// UIScheduleItem represents a single schedule item for display.
// UIScheduleItem 表示用于展示的单个日程项。
type UIScheduleItem struct {
	UID      string `json:"uid"`
	Title    string `json:"title"`
	Location string `json:"location,omitempty"`
	Status   string `json:"status,omitempty"`
	StartTs  int64  `json:"start_ts"`
	EndTs    int64  `json:"end_ts"`
	AllDay   bool   `json:"all_day"`
}

// UIScheduleListData represents a list of schedules for display.
// UIScheduleListData 表示用于展示的日程列表。
type UIScheduleListData struct {
	Title     string           `json:"title"`
	Query     string           `json:"query"`
	TimeRange string           `json:"time_range,omitempty"`
	Reason    string           `json:"reason,omitempty"`
	SessionID string           `json:"session_id,omitempty"`
	Schedules []UIScheduleItem `json:"schedules"`
	Count     int              `json:"count"`
}

// GenerateCacheKey creates a cache key from agent name, userID and userInput using SHA256 hash.
// GenerateCacheKey 使用 SHA256 哈希从代理名称、用户ID和用户输入创建缓存键。
// Uses full SHA256 hex to eliminate collision risk.
func GenerateCacheKey(agentName string, userID int32, userInput string) string {
	hash := sha256.Sum256([]byte(userInput))
	hashStr := hex.EncodeToString(hash[:])
	// Use full hash (64 hex chars) for zero collision probability
	return fmt.Sprintf("%s:%d:%s", agentName, userID, hashStr)
}

// Compile-time interface compliance checks.
// 编译时接口合规性检查。
// These ensure that all parrot types correctly implement the ParrotAgent interface.
// 如果任何类型未正确实现接口，编译将失败。
var (
	_ ParrotAgent = (*MemoParrot)(nil)       // 灰灰 (Memo)
	_ ParrotAgent = (*AmazingParrot)(nil)    // 折衷 (Amazing)
	_ ParrotAgent = (*ScheduleParrotV2)(nil) // 时巧 (Schedule V2)
	_ ParrotAgent = (*GeekParrot)(nil)       // 极客 (Geek Mode)
	_ ParrotAgent = (*EvolutionParrot)(nil)  // 进化 (Evolution Mode)
)
