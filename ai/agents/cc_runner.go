package agent

import (
	"log/slog"
	"time"

	"github.com/hrygo/divinesense/ai/agents/runner"
)

// ============================================================================
// BACKWARD COMPATIBILITY BRIDGE
// ============================================================================
//
// This file provides backward compatibility aliases for types and functions
// that have been moved to the runner subpackage.
//
// Deprecated: Use github.com/hrygo/divinesense/ai/agents/runner directly.
// ============================================================================

// Type aliases for backward compatibility
// 类型别名，用于向后兼容

// CCRunner is an alias for runner.CCRunner.
//
// Deprecated: Use runner.CCRunner directly.
type CCRunner = runner.CCRunner

// CCRunnerConfig is an alias for runner.Config.
//
// Deprecated: Use runner.Config directly.
type CCRunnerConfig = runner.Config

// DangerDetector is an alias for runner.Detector.
//
// Deprecated: Use runner.Detector directly.
type DangerDetector = runner.Detector

// CCSessionManager is an alias for runner.CCSessionManager.
//
// Deprecated: Use runner.CCSessionManager directly.
type CCSessionManager = runner.CCSessionManager

// Session is an alias for runner.Session.
//
// Deprecated: Use runner.Session directly.
type Session = runner.Session

// SessionStatus is an alias for runner.SessionStatus.
//
// Deprecated: Use runner.SessionStatus directly.
type SessionStatus = runner.SessionStatus

// SessionManager is an alias for runner.SessionManager.
//
// Deprecated: Use runner.SessionManager directly.
type SessionManager = runner.SessionManager

// StreamMessage is an alias for runner.StreamMessage.
//
// Deprecated: Use runner.StreamMessage directly.
type StreamMessage = runner.StreamMessage

// EventCallback is an alias for runner.EventCallback.
//
// Deprecated: Use runner.EventCallback directly.
type EventCallback = runner.EventCallback

// EventMeta is an alias for runner.EventMeta.
//
// Deprecated: Use runner.EventMeta directly.
type EventMeta = runner.EventMeta

// EventWithMeta is an alias for runner.EventWithMeta.
//
// Deprecated: Use runner.EventWithMeta directly.
type EventWithMeta = runner.EventWithMeta

// SessionStats is an alias for runner.SessionStats.
//
// Deprecated: Use runner.SessionStats directly.
type SessionStats = runner.SessionStats

// SessionStatsData is an alias for runner.SessionStatsData.
//
// Deprecated: Use runner.SessionStatsData directly.
type SessionStatsData = runner.SessionStatsData

// AgentSessionStatsForStorage is an alias for runner.AgentSessionStatsForStorage.
//
// Deprecated: Use runner.AgentSessionStatsForStorage directly.
type AgentSessionStatsForStorage = runner.AgentSessionStatsForStorage

// SessionStatsProvider is the interface for agents that provide session statistics.
// SessionStatsProvider 是提供会话统计信息的代理接口。
//
// This interface is implemented by GeekParrot and EvolutionParrot which have
// direct access to CCRunner's SessionStats.
type SessionStatsProvider interface {
	GetSessionStats() *SessionStats
}

// ParrotStreamAdapter is an adapter that converts event callbacks to the
// format expected by the streaming response handler.
// ParrotStreamAdapter 是一个适配器，将事件回调转换为流响应处理器期望的格式。
type ParrotStreamAdapter struct {
	send func(eventType string, eventData any) error
}

// Send sends an event through the adapter.
func (a *ParrotStreamAdapter) Send(eventType string, eventData any) error {
	return a.send(eventType, eventData)
}

// NewParrotStreamAdapter creates a new stream adapter from a send function.
// NewParrotStreamAdapter 从发送函数创建新的流适配器。
func NewParrotStreamAdapter(send func(eventType string, eventData any) error) *ParrotStreamAdapter {
	return &ParrotStreamAdapter{send: send}
}

// DangerLevel is an alias for runner.DangerLevel.
//
// Deprecated: Use runner.DangerLevel directly.
type DangerLevel = runner.DangerLevel

// DangerBlockEvent is an alias for runner.DangerBlockEvent.
//
// Deprecated: Use runner.DangerBlockEvent directly.
type DangerBlockEvent = runner.DangerBlockEvent

// ProcessingPhase is an alias for runner.ProcessingPhase.
//
// Deprecated: Use runner.ProcessingPhase directly.
type ProcessingPhase = runner.ProcessingPhase

// PhaseChangeEvent is an alias for runner.PhaseChangeEvent.
//
// Deprecated: Use runner.PhaseChangeEvent directly.
type PhaseChangeEvent = runner.PhaseChangeEvent

// ProgressEvent is an alias for runner.ProgressEvent.
//
// Deprecated: Use runner.ProgressEvent directly.
type ProgressEvent = runner.ProgressEvent

// SafeCallbackFunc is an alias for runner.SafeCallbackFunc.
//
// Deprecated: Use runner.SafeCallbackFunc directly.
type SafeCallbackFunc = runner.SafeCallbackFunc

// ContentBlock is an alias for runner.ContentBlock.
//
// Deprecated: Use runner.ContentBlock directly.
type ContentBlock = runner.ContentBlock

// AssistantMessage is an alias for runner.AssistantMessage.
//
// Deprecated: Use runner.AssistantMessage directly.
type AssistantMessage = runner.AssistantMessage

// UsageStats is an alias for runner.UsageStats.
//
// Deprecated: Use runner.UsageStats directly.
type UsageStats = runner.UsageStats

// Constants for backward compatibility
const (
	// SessionStatus constants
	SessionStatusStarting = runner.SessionStatusStarting
	SessionStatusReady    = runner.SessionStatusReady
	SessionStatusBusy     = runner.SessionStatusBusy
	SessionStatusDead     = runner.SessionStatusDead

	// DangerLevel constants
	DangerLevelCritical = runner.DangerLevelCritical
	DangerLevelHigh     = runner.DangerLevelHigh
	DangerLevelModerate = runner.DangerLevelModerate

	// ProcessingPhase constants
	PhaseAnalyzing    = runner.PhaseAnalyzing
	PhasePlanning     = runner.PhasePlanning
	PhaseRetrieving   = runner.PhaseRetrieving
	PhaseSynthesizing = runner.PhaseSynthesizing

	// Event type constants
	EventTypePhaseChange  = runner.EventTypePhaseChange
	EventTypeProgress     = runner.EventTypeProgress
	EventTypeThinking     = runner.EventTypeThinking
	EventTypeToolUse      = runner.EventTypeToolUse
	EventTypeToolResult   = runner.EventTypeToolResult
	EventTypeAnswer       = runner.EventTypeAnswer
	EventTypeError        = runner.EventTypeError
	EventTypeSessionStats = "session_stats" // Session statistics event
)

// Function aliases for backward compatibility
// 函数别名，用于向后兼容

// NewCCRunner creates a new CCRunner instance.
//
// Deprecated: Use runner.NewCCRunner directly.
func NewCCRunner(timeout time.Duration, logger *slog.Logger) (*CCRunner, error) {
	return runner.NewCCRunner(timeout, logger)
}

// NewDangerDetector creates a new danger detector.
//
// Deprecated: Use runner.NewDetector directly.
func NewDangerDetector(logger *slog.Logger) *DangerDetector {
	return runner.NewDetector(logger)
}

// NewCCSessionManager creates a new session manager.
//
// Deprecated: Use runner.NewCCSessionManager directly.
func NewCCSessionManager(logger *slog.Logger, timeout time.Duration) *CCSessionManager {
	return runner.NewCCSessionManager(logger, timeout)
}

// ConversationIDToSessionID converts a database ConversationID to a SessionID.
//
// Deprecated: Use runner.ConversationIDToSessionID directly.
func ConversationIDToSessionID(conversationID int64) string {
	return runner.ConversationIDToSessionID(conversationID)
}

// BuildSystemPrompt provides context for Claude Code CLI.
//
// Deprecated: Use runner.BuildSystemPrompt directly.
func BuildSystemPrompt(workDir, sessionID string, userID int32, deviceContext string) string {
	return runner.BuildSystemPrompt(workDir, sessionID, userID, deviceContext)
}

// SafeCallback wraps an EventCallback to log errors instead of propagating them.
//
// Deprecated: Use runner.SafeCallback directly.
func SafeCallback(callback EventCallback) SafeCallbackFunc {
	return runner.SafeCallback(callback)
}

// NewEventWithMeta creates a new EventWithMeta.
//
// Deprecated: Use runner.NewEventWithMeta directly.
func NewEventWithMeta(eventType, eventData string, meta *EventMeta) *EventWithMeta {
	return runner.NewEventWithMeta(eventType, eventData, meta)
}

// TruncateString truncates a string to a maximum length for logging.
//
// Deprecated: Use runner.TruncateString directly.
func TruncateString(s string, maxLen int) string {
	return runner.TruncateString(s, maxLen)
}
