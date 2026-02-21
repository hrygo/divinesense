package geek

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	agentpkg "github.com/hrygo/divinesense/ai/agents"
)

// GeekParrot is the Geek Mode specialist parrot (🦜 极客).
// GeekParrot 是极客模式专用鹦鹉（🦜 极客）.
//
// It provides DIRECT access to Claude Code CLI without any LLM processing,
// using the unified CCRunner + GeekMode architecture.
// 它提供 Claude Code CLI 的直接访问，不经过任何 LLM 处理，使用统一的 CCRunner + GeekMode 架构。
type GeekParrot struct {
	runner      *agentpkg.CCRunner
	mode        *GeekMode
	sessionID   string
	userID      int32
	workDir     string
	deviceCtx   string
	initialized bool // Tracks if user context has been injected
}

// NewGeekParrot creates a new GeekParrot instance.
// NewGeekParrot 创建一个新的 GeekParrot 实例。
func NewGeekParrot(runner *agentpkg.CCRunner, sourceDir string, userID int32, sessionID string) (*GeekParrot, error) {

	// Create GeekMode
	// Geek Mode uses its own sandbox directory, so we pass empty string to let it
	// use DIVINESENSE_CLAUDE_CODE_WORKDIR or default path.
	mode := NewGeekMode("")

	// Generate session ID if not provided
	if sessionID == "" {
		sessionID = uuid.New().String()
	}

	// Get working directory from mode
	workDir := mode.GetWorkDir(userID)

	return &GeekParrot{
		runner:    runner,
		mode:      mode,
		sessionID: sessionID,
		userID:    userID,
		workDir:   workDir,
	}, nil
}

// SetDeviceContext sets the full device and browser context for the parrot.
// SetDeviceContext 为鹦鹉设置完整的设备和浏览器上下文。
func (p *GeekParrot) SetDeviceContext(contextJson string) {
	p.deviceCtx = contextJson
}

// Name returns the name of the parrot.
// Name 返回鹦鹉名称。
func (p *GeekParrot) Name() string {
	return p.mode.Name()
}

// Execute implements agentpkg.ParrotAgent.
// history is ignored - Claude Code manages its own history.
func (p *GeekParrot) Execute(
	ctx context.Context,
	userInput string,
	history []string, // Ignored - Claude Code manages its own history
	callback agentpkg.EventCallback,
) error {
	slog.Info("GeekParrot: Executing Claude Code CLI",
		"user_id", p.userID,
		"session_id", p.sessionID,
		"input_length", len(userInput))

	// Check permissions
	if err := p.mode.CheckPermission(ctx, p.userID); err != nil {
		p.sendError(callback, fmt.Sprintf("Permission denied: %s", err.Error()))
		return agentpkg.NewParrotError(p.Name(), "CheckPermission", err)
	}

	// Build config for CCRunner
	cfg := &agentpkg.CCRunnerConfig{
		Mode:           p.mode.Name(),
		WorkDir:        p.workDir,
		SessionID:      p.sessionID,
		UserID:         p.userID,
		DeviceContext:  p.deviceCtx,
		PermissionMode: "bypassPermissions",
	}

	// TaskSystemPrompt: only inject user context on first session creation
	// BaseSystemPrompt (DivineSense context + Output Behavior) is set at engine creation
	if !p.initialized {
		cfg.SystemPrompt = p.mode.BuildContextPrompt(cfg)
		p.initialized = true
		slog.Info("GeekParrot: User context injected for first session",
			"user_id", p.userID,
			"session_id", p.sessionID)
	} else {
		cfg.SystemPrompt = "" // Subsequent requests: only user prompt, no system context
	}

	// Execute via CCRunner
	if err := p.runner.Execute(ctx, cfg, userInput, callback); err != nil {
		return agentpkg.NewParrotError(p.Name(), "Execute", err)
	}

	return nil
}

// sendError sends an error event via callback.
// sendError 通过回调发送错误事件。
func (p *GeekParrot) sendError(callback agentpkg.EventCallback, message string) {
	if callback != nil {
		if err := callback(agentpkg.EventTypeError, message); err != nil {
			slog.Warn("Failed to send error notification to client", "error", err)
		}
	}
}

// ResetSession resets the session state (e.g., on error or user request).
// ResetSession 重置会话状态（例如出错或用户请求时）。
func (p *GeekParrot) ResetSession() {
	p.sessionID = uuid.New().String()
	slog.Info("GeekParrot: Session reset",
		"user_id", p.userID,
		"new_session_id", p.sessionID)
}

// GetSessionID returns the current session ID.
// GetSessionID 返回当前会话 ID。
func (p *GeekParrot) GetSessionID() string {
	return p.sessionID
}

// SelfDescribe returns the GeekParrot's metacognitive information.
// SelfDescribe 返回极客鹦鹉的元认知信息。
func (p *GeekParrot) SelfDescribe() *agentpkg.ParrotSelfCognition {
	return &agentpkg.ParrotSelfCognition{
		Name:  "geek",
		Emoji: "🦜",
		Title: "Claude Code CLI Runner",
		Personality: []string{
			"直接 (Direct)",
			"高效 (Efficient)",
			"技术专家 (Technical Expert)",
		},
		Capabilities: []string{
			"调用 Claude Code CLI",
			"通过 CCRunner 执行",
			"服务 Web 界面用户",
			"实时流式响应",
			"会话持久化",
		},
		Limitations: []string{
			"需要安装 Claude Code CLI",
			"Headless 模式运行",
		},
		WorkingStyle: "Go backend → CCRunner → Claude Code CLI → Web 用户",
	}
}

// IsSessionActive returns whether a session has been started.
// IsSessionActive 返回是否已启动会话。
func (p *GeekParrot) IsSessionActive() bool {
	return p.sessionID != ""
}

// GetWorkDir returns the working directory for Claude Code CLI.
// GetWorkDir 返回 Claude Code CLI 的工作目录。
func (p *GeekParrot) GetWorkDir() string {
	return p.workDir
}

// GetUserID returns the user ID for this parrot.
// GetUserID 返回此鹦鹉的用户 ID。
func (p *GeekParrot) GetUserID() int32 {
	return p.userID
}

// Cancel is a no-op for Geek Mode (session continues unless explicitly reset).
// Cancel 对极客模式是空操作（会话继续，除非显式重置）。
func (p *GeekParrot) Cancel() {
	// No-op - session continues
	// Use ResetSession() to explicitly clear the session
}

// GetSessionStats returns the session statistics from the last execution.
// GetSessionStats 返回上次执行的会话统计数据。
// Implements agentpkg.ParrotAgent interface.
func (p *GeekParrot) GetSessionStats() *agentpkg.NormalSessionStats {
	stats := p.runner.GetSessionStats()
	if stats == nil {
		return nil
	}
	// Convert runner.SessionStats to agent.NormalSessionStats
	toolsUsed := make([]string, 0, len(stats.ToolsUsed))
	for tool := range stats.ToolsUsed {
		toolsUsed = append(toolsUsed, tool)
	}
	return &agentpkg.NormalSessionStats{
		StartTime:            stats.StartTime,
		EndTime:              time.Now(),
		AgentType:            "geek",
		ModelUsed:            "",
		PromptTokens:         int(stats.InputTokens),
		CompletionTokens:     int(stats.OutputTokens),
		TotalTokens:          int(stats.InputTokens + stats.OutputTokens),
		CacheReadTokens:      int(stats.CacheReadTokens),
		CacheWriteTokens:     int(stats.CacheWriteTokens),
		ThinkingDurationMs:   stats.ThinkingDurationMs,
		GenerationDurationMs: stats.GenerationDurationMs,
		TotalDurationMs:      stats.TotalDurationMs,
		ToolCallCount:        int(stats.ToolCallCount),
		ToolDurationMs:       stats.ToolDurationMs,
		FilesModified:        stats.FilesModified,
		FilePaths:            stats.FilePaths,
		ToolsUsed:            toolsUsed,
	}
}

// Compile-time interface compliance check.
// 编译时接口合规性检查。
var _ agentpkg.ParrotAgent = (*GeekParrot)(nil)
