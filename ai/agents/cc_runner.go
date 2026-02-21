package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"runtime"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hrygo/divinesense/ai/agents/events"
	"github.com/hrygo/hotplex"
)

type CCRunner struct {
	engine     hotplex.HotPlexClient
	adminToken string // Token for SetDangerBypassEnabled calls
}

// CCRunnerConfig defines the configuration for CCRunner execution.
// DeviceContext is used to build TaskInstructions, not passed to hotplex directly.
//
// TaskInstructions has session-level persistence in hotplex: once set, it automatically
// applies to all subsequent Execute calls in the same session unless explicitly overridden.
type CCRunnerConfig struct {
	Mode             string
	WorkDir          string
	ConversationID   int64
	SessionID        string
	UserID           int32
	TaskInstructions string // Session-persistent instructions (mapped to hotplex.TaskInstructions)
	DeviceContext    string // Used to build TaskInstructions via BuildUserContextPrompt()
	PermissionMode   string
}

type StreamMessage = hotplex.StreamMessage

type EventCallback = events.Callback

type EventMeta = hotplex.EventMeta

type EventWithMeta = hotplex.EventWithMeta

type SessionStats = hotplex.SessionStats

type SessionStatsData struct {
	SessionID            string   `json:"session_id"`
	ConversationID       int64    `json:"conversation_id"`
	UserID               int32    `json:"user_id"`
	AgentType            string   `json:"agent_type"`
	StartTime            int64    `json:"start_time"`
	EndTime              int64    `json:"end_time"`
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

type SessionStatsProvider interface {
	GetSessionStats() *SessionStats
}

type ParrotStreamAdapter struct {
	send func(eventType string, eventData any) error
}

func (a *ParrotStreamAdapter) Send(eventType string, eventData any) error {
	return a.send(eventType, eventData)
}

func NewParrotStreamAdapter(send func(eventType string, eventData any) error) *ParrotStreamAdapter {
	return &ParrotStreamAdapter{send: send}
}

type ProcessingPhase string

const (
	PhaseAnalyzing    ProcessingPhase = "analyzing"
	PhasePlanning     ProcessingPhase = "planning"
	PhaseRetrieving   ProcessingPhase = "retrieving"
	PhaseSynthesizing ProcessingPhase = "synthesizing"
)

type PhaseChangeEvent struct {
	Phase            ProcessingPhase `json:"phase"`
	PhaseNumber      int             `json:"phase_number"`
	TotalPhases      int             `json:"total_phases"`
	EstimatedSeconds int             `json:"estimated_seconds"`
}

type ProgressEvent struct {
	Percent              int `json:"percent"`
	EstimatedSeconds     int `json:"estimated_seconds"`
	EstimatedTimeSeconds int `json:"estimated_time_seconds"`
}

type SafeCallbackFunc = events.SafeCallback

type ContentBlock = hotplex.ContentBlock

type AssistantMessage = hotplex.AssistantMessage

type UsageStats = hotplex.UsageStats

const (
	EventTypePhaseChange  = "phase_change"
	EventTypeProgress     = "progress"
	EventTypeThinking     = "thinking"
	EventTypeToolUse      = "tool_use"
	EventTypeToolResult   = "tool_result"
	EventTypeAnswer       = "answer"
	EventTypeError        = "error"
	EventTypeSessionStats = "session_stats"
)

// CCRunnerOption is a functional option for configuring CCRunner.
type CCRunnerOption func(*ccRunnerOptions)

type ccRunnerOptions struct {
	adminToken       string
	baseSystemPrompt string
	namespace        string
}

// WithAdminToken sets the admin token for danger bypass mode.
func WithAdminToken(token string) CCRunnerOption {
	return func(o *ccRunnerOptions) {
		o.adminToken = token
	}
}

// WithBaseSystemPrompt sets the base system prompt for the engine.
// This is injected at process startup as foundational rules for all sessions.
func WithBaseSystemPrompt(prompt string) CCRunnerOption {
	return func(o *ccRunnerOptions) {
		o.baseSystemPrompt = prompt
	}
}

// WithNamespace sets the namespace for UUID v5 session ID generation.
// Different namespaces ensure physical isolation between modes (e.g., Geek vs Evolution).
func WithNamespace(namespace string) CCRunnerOption {
	return func(o *ccRunnerOptions) {
		o.namespace = namespace
	}
}

func NewCCRunner(timeout time.Duration, logger *slog.Logger, opts ...CCRunnerOption) (*CCRunner, error) {
	// Apply options
	opt := &ccRunnerOptions{}
	for _, o := range opts {
		o(opt)
	}

	// Default namespace
	namespace := opt.namespace
	if namespace == "" {
		namespace = "divinesense"
	}

	engineOpts := hotplex.EngineOptions{
		Timeout:          timeout,
		IdleTimeout:      30 * time.Minute,
		Logger:           logger,
		Namespace:        namespace,
		BaseSystemPrompt: opt.baseSystemPrompt,
		AdminToken:       opt.adminToken,
	}

	engine, err := hotplex.NewEngine(engineOpts)
	if err != nil {
		return nil, err
	}

	return &CCRunner{engine: engine, adminToken: opt.adminToken}, nil
}

func (r *CCRunner) Execute(ctx context.Context, cfg *CCRunnerConfig, prompt string, callback EventCallback) error {
	if cfg.SessionID == "" && cfg.ConversationID > 0 {
		cfg.SessionID = ConversationIDToSessionID(cfg.ConversationID)
	}

	hotplexCfg := &hotplex.Config{
		WorkDir:          cfg.WorkDir,
		SessionID:        cfg.SessionID,
		TaskInstructions: cfg.TaskInstructions,
	}

	if cfg.PermissionMode == "bypassPermissions" && r.adminToken != "" {
		if err := r.engine.SetDangerBypassEnabled(r.adminToken, true); err != nil {
			return fmt.Errorf("failed to enable danger bypass: %w", err)
		}
	}

	var cb hotplex.Callback
	if callback != nil {
		cb = hotplex.Callback(callback)
	}

	return r.engine.Execute(ctx, hotplexCfg, prompt, cb)
}

func (r *CCRunner) Close() error {
	return r.engine.Close()
}

func (r *CCRunner) GetSessionStats() *SessionStats {
	return r.engine.GetSessionStats()
}

func (r *CCRunner) StopSession(sessionID string, reason string) error {
	return r.engine.StopSession(sessionID, reason)
}

func (r *CCRunner) StopSessionByConversationID(conversationID int64, reason string) error {
	sessionID := ConversationIDToSessionID(conversationID)
	return r.StopSession(sessionID, reason)
}

func (r *CCRunner) SetDangerAllowPaths(paths []string) {
	r.engine.SetDangerAllowPaths(paths)
}

func (r *CCRunner) SetDangerBypassEnabled(token string, enabled bool) error {
	r.adminToken = token // Store for Execute calls
	return r.engine.SetDangerBypassEnabled(token, enabled)
}

func (r *CCRunner) ValidateConfig(cfg *CCRunnerConfig) error {
	if cfg.Mode == "" {
		return fmt.Errorf("mode is required")
	}
	if cfg.WorkDir == "" {
		return fmt.Errorf("work_dir is required")
	}
	if cfg.SessionID == "" {
		return fmt.Errorf("session_id is required")
	}
	if cfg.UserID == 0 {
		return fmt.Errorf("user_id is required")
	}
	return nil
}

func ConversationIDToSessionID(conversationID int64) string {
	namespace := uuid.MustParse("6ba7b810-9dad-11d1-80b4-00c04fd430c8")
	name := fmt.Sprintf("divinesense:conversation:%d", conversationID)
	return uuid.NewSHA1(namespace, []byte(name)).String()
}

// DivineSenseBaseContext is the fixed context for all DivineSense sessions.
// This should be included in EngineOptions.BaseSystemPrompt.
const DivineSenseBaseContext = `# Context

You are running inside DivineSense, an intelligent assistant system.

**User Interaction**: Users type questions in their web browser, which invokes you via a Go backend. Your response streams back to their browser in real-time. **Always respond in Chinese (Simplified).**
`

// BuildUserContextPrompt builds the user-specific context prompt.
// This should be passed to hotplex.Config.TaskInstructions on first session creation.
// Time is excluded as it changes on every request.
func BuildUserContextPrompt(workDir, sessionID string, userID int32, deviceContext string) string {
	osName := runtime.GOOS
	arch := runtime.GOARCH
	if osName == "darwin" {
		osName = "macOS"
	}

	var contextMap map[string]any
	userAgent := "Unknown"
	deviceInfo := "Unknown"
	if deviceContext != "" {
		trimmed := strings.TrimSpace(deviceContext)
		if strings.HasPrefix(trimmed, "{") {
			if err := json.Unmarshal([]byte(deviceContext), &contextMap); err == nil {
				if ua, ok := contextMap["userAgent"].(string); ok {
					userAgent = ua
				}
				if mobile, ok := contextMap["isMobile"].(bool); ok {
					if mobile {
						deviceInfo = "Mobile"
					} else {
						deviceInfo = "Desktop"
					}
				}
				if w, ok := contextMap["screenWidth"].(float64); ok {
					if h, ok := contextMap["screenHeight"].(float64); ok {
						deviceInfo = fmt.Sprintf("%s (%dx%d)", deviceInfo, int(w), int(h))
					}
				}
				if lang, ok := contextMap["language"].(string); ok {
					deviceInfo = fmt.Sprintf("%s, Language: %s", deviceInfo, lang)
				}
			} else {
				userAgent = deviceContext
			}
		} else {
			userAgent = deviceContext
		}
	}

	return fmt.Sprintf(`- **User ID**: %d
- **Client Device**: %s
- **User Agent**: %s
- **Server OS**: %s (%s)
- **Workspace**: %s
- **Mode**: Non-interactive headless (--print)
- **Session**: %s (persists via --session-id/--resume)
`, userID, deviceInfo, userAgent, osName, arch, workDir, sessionID)
}

// BuildSystemPrompt is deprecated. Use DivineSenseBaseContext + BuildUserContextPrompt instead.
func BuildSystemPrompt(workDir, sessionID string, userID int32, deviceContext string) string {
	return DivineSenseBaseContext + "\n" + BuildUserContextPrompt(workDir, sessionID, userID, deviceContext)
}

func SafeCallback(callback EventCallback) SafeCallbackFunc {
	return events.WrapSafe(callback)
}

func NewEventWithMeta(eventType, eventData string, meta *EventMeta) *EventWithMeta {
	return hotplex.NewEventWithMeta(eventType, eventData, meta)
}

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
