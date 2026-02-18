package geek

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/google/uuid"

	agentpkg "github.com/hrygo/divinesense/ai/agents"
	"github.com/hrygo/divinesense/store"
)

// evolutionManagers tracks all CCSessionManagers created by EvolutionParrot instances.
// This enables graceful shutdown of all CLI processes during server termination.
// evolutionManagers 跟踪 EvolutionParrot 实例创建的所有 CCSessionManager。
// 这使得在服务器终止期间可以优雅关闭所有 CLI 进程。
var (
	evolutionManagers   []*agentpkg.CCSessionManager
	evolutionManagersMu sync.Mutex
)

// EvolutionParrot implements the Evolution Mode agent for self-evolution.
// EvolutionParrot 实现进化模式代理用于自我进化。
//
// Evolution Mode allows DivineSense to modify its own source code under
// strict safety constraints. All git operations and PR creation are handled
// by Claude Code CLI itself - this parrot only provides configuration.
type EvolutionParrot struct {
	runner      *agentpkg.CCRunner
	mode        *EvolutionMode
	workDir     string
	sessionID   string
	userID      int32
	deviceCtx   string
	taskID      string
	initialized bool
}

// NewEvolutionParrot creates a new EvolutionParrot instance.
// NewEvolutionParrot 创建一个新的 EvolutionParrot 实例。
//
// Parameters:
//   - sourceDir: DivineSense source code directory
//   - userID: User ID requesting evolution mode
//   - sessionID: Session identifier for persistence
//   - st: Store for user role checking (required for admin verification)
//   - adminOnly: Whether only admins can use evolution mode (default: from env or true)
func NewEvolutionParrot(sourceDir string, userID int32, sessionID string, st *store.Store, adminOnly ...bool) (*EvolutionParrot, error) {
	// Generate task ID if not provided
	taskID := uuid.New().String()[:8]
	if sessionID == "" {
		sessionID = taskID
	}

	// Determine adminOnly setting
	// Priority: explicit parameter > environment variable > default true
	adminOnlySetting := true
	if len(adminOnly) > 0 {
		adminOnlySetting = adminOnly[0]
	} else if env := os.Getenv("DIVINESENSE_EVOLUTION_ADMIN_ONLY"); env != "" {
		adminOnlySetting = env == "true" || env == "1"
	}

	// Create dedicated session manager for Evolution Mode (process isolation)
	// 为进化模式创建专用会话管理器（进程隔离）
	// Evolution mode sessions might be long-running, use 30m idle timeout as standard
	manager := agentpkg.NewCCSessionManager(slog.Default(), 30*time.Minute)

	// Track manager for graceful shutdown
	// 跟踪管理器以便优雅关闭
	evolutionManagersMu.Lock()
	evolutionManagers = append(evolutionManagers, manager)
	evolutionManagersMu.Unlock()

	// Create CCRunner with manager
	runner, err := agentpkg.NewCCRunnerWithManager(manager, 10*time.Minute, slog.Default())
	if err != nil {
		return nil, fmt.Errorf("failed to create CCRunner: %w", err)
	}

	// Create EvolutionMode
	mode := NewEvolutionMode(&EvolutionModeConfig{
		SourceDir: sourceDir,
		AdminOnly: adminOnlySetting,
		Store:     st,
	})

	return &EvolutionParrot{
		runner:      runner,
		mode:        mode,
		workDir:     sourceDir,
		sessionID:   sessionID,
		userID:      userID,
		taskID:      taskID,
		initialized: false,
	}, nil
}

// Name returns the name of the parrot.
// Name 返回鹦鹉名称。
func (p *EvolutionParrot) Name() string {
	return "evolution"
}

// SetDeviceContext sets the device context for the parrot.
// SetDeviceContext 设置鹦鹉的设备上下文。
func (p *EvolutionParrot) SetDeviceContext(contextJson string) {
	p.deviceCtx = contextJson
}

// Execute implements agentpkg.ParrotAgent.
// history is ignored - Evolution mode manages its own state.
func (p *EvolutionParrot) Execute(
	ctx context.Context,
	userInput string,
	history []string, // Ignored - Evolution mode manages its own state
	callback agentpkg.EventCallback,
) error {
	// Check permissions first
	// 首先检查权限
	if err := p.mode.CheckPermission(ctx, p.userID); err != nil {
		p.sendError(callback, fmt.Sprintf("Permission denied: %s", err.Error()))
		return agentpkg.NewParrotError(p.Name(), "CheckPermission", err)
	}

	// Build config for CCRunner
	// 为 CCRunner 构建配置
	cfg := &agentpkg.CCRunnerConfig{
		Mode:          p.mode.Name(),
		WorkDir:       p.workDir,
		SessionID:     p.sessionID,
		UserID:        p.userID,
		DeviceContext: p.deviceCtx,
	}
	cfg.SystemPrompt = p.mode.BuildSystemPrompt(cfg)

	// Execute via shared persistent session logic
	// 通过共享持久化会话逻辑执行
	// Input format: {"type":"user","message":{"role":"user","content":"text"}}
	inputMsg := map[string]any{
		"type": "user",
		"message": map[string]any{
			"role":    "user",
			"content": userInput,
		},
	}

	if err := ExecutePersistentSession(ctx, p.runner, cfg, inputMsg, callback); err != nil {
		return agentpkg.NewParrotError(p.Name(), "ExecutePersistentSession", err)
	}

	// Mark as initialized after first successful execution
	// 首次成功执行后标记为已初始化
	if !p.initialized {
		p.initialized = true
		slog.Info("EvolutionParrot: Session initialized",
			"user_id", p.userID,
			"task_id", p.taskID)
	}

	return nil
}

// sendError sends an error event via callback.
// sendError 通过回调发送错误事件。
func (p *EvolutionParrot) sendError(callback agentpkg.EventCallback, message string) {
	if callback != nil {
		if err := callback(agentpkg.EventTypeError, message); err != nil {
			slog.Warn("Failed to send error notification to client", "error", err)
		}
	}
}

// ResetSession resets the evolution session.
// ResetSession 重置进化会话。
func (p *EvolutionParrot) ResetSession() {
	p.initialized = false
	p.sessionID = uuid.New().String()[:8]
	slog.Info("EvolutionParrot: Session reset",
		"user_id", p.userID)
}

// GetSessionID returns the current session ID.
// GetSessionID 返回当前会话 ID。
func (p *EvolutionParrot) GetSessionID() string {
	return p.sessionID
}

// GetTaskID returns the evolution task ID.
// GetTaskID 返回进化任务 ID。
func (p *EvolutionParrot) GetTaskID() string {
	return p.taskID
}

// SelfDescribe returns the EvolutionParrot's metacognitive information.
// SelfDescribe 返回进化鹦鹉的元认知信息。
func (p *EvolutionParrot) SelfDescribe() *agentpkg.ParrotSelfCognition {
	return &agentpkg.ParrotSelfCognition{
		Name:  "evolution",
		Emoji: "🧬",
		Title: "Evolution Mode - Self-Evolving Agent",
		Personality: []string{
			"谨慎 (Cautious)",
			"结构化 (Structured)",
			"协作 (Collaborative)",
		},
		Capabilities: []string{
			"通过 Claude Code CLI 修改 DivineSense 源代码",
			"遵循 CLAUDE.md 规范",
			"通过 PR 审查进行代码变更",
		},
		Limitations: []string{
			"仅限管理员访问",
			"强制 PR 审查流程",
			"路径白名单限制",
		},
		WorkingStyle: "Go backend → CCRunner → Claude Code CLI → Source Code → GitHub PR",
	}
}

// IsSessionActive returns whether a session has been started.
// IsSessionActive 返回是否已启动会话。
func (p *EvolutionParrot) IsSessionActive() bool {
	return p.initialized
}

// GetWorkDir returns the working directory.
// GetWorkDir 返回工作目录。
func (p *EvolutionParrot) GetWorkDir() string {
	return p.workDir
}

// GetUserID returns the user ID.
// GetUserID 返回用户 ID。
func (p *EvolutionParrot) GetUserID() int32 {
	return p.userID
}

// Cancel cancels the current evolution session.
// Cancel 取消当前进化会话。
func (p *EvolutionParrot) Cancel() {
	p.ResetSession()
}

// GetSessionStats returns the session statistics from the last execution.
// GetSessionStats 返回上次执行的会话统计数据。
// Implements agentpkg.ParrotAgent interface.
func (p *EvolutionParrot) GetSessionStats() *agentpkg.NormalSessionStats {
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
		AgentType:            "evolution",
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
var _ agentpkg.ParrotAgent = (*EvolutionParrot)(nil)

// shutdownEvolution terminates all active CLI sessions managed by EvolutionParrot instances.
// This is called internally by the package-level Shutdown function.
// shutdownEvolution 终止 EvolutionParrot 实例管理的所有活动 CLI 会话。
// 由包级别的 Shutdown 函数内部调用。
func shutdownEvolution() {
	evolutionManagersMu.Lock()
	defer evolutionManagersMu.Unlock()

	if len(evolutionManagers) > 0 {
		slog.Info("EvolutionParrot: shutting down evolution session managers", "count", len(evolutionManagers))
		for _, mgr := range evolutionManagers {
			if mgr != nil {
				mgr.Shutdown()
			}
		}
		evolutionManagers = nil
	}
}
