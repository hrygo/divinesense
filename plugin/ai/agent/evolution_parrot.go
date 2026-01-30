package agent

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
)

// EvolutionParrot implements the Evolution Mode agent for self-evolution.
// EvolutionParrot 实现进化模式代理用于自我进化。
//
// Evolution Mode allows DivineSense to modify its own source code under
// strict safety constraints. All git operations and PR creation are handled
// by Claude Code CLI itself - this parrot only provides configuration.
type EvolutionParrot struct {
	runner      *CCRunner
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
func NewEvolutionParrot(sourceDir string, userID int32, sessionID string) (*EvolutionParrot, error) {
	// Generate task ID if not provided
	taskID := uuid.New().String()[:8]
	if sessionID == "" {
		sessionID = taskID
	}

	// Create CCRunner
	runner, err := NewCCRunner(10*time.Minute, slog.Default())
	if err != nil {
		return nil, fmt.Errorf("failed to create CCRunner: %w", err)
	}

	// Create EvolutionMode
	mode := NewEvolutionMode(&EvolutionModeConfig{
		SourceDir: sourceDir,
		AdminOnly: true,
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

// ExecuteWithCallback runs the Evolution Mode agent with streaming.
// ExecuteWithCallback 运行进化模式代理并流式传输。
func (p *EvolutionParrot) ExecuteWithCallback(
	ctx context.Context,
	userInput string,
	history []string,
	callback EventCallback,
) error {
	// Check permissions first
	// 首先检查权限
	if err := p.mode.CheckPermission(ctx, p.userID); err != nil {
		p.sendError(callback, fmt.Sprintf("Permission denied: %s", err.Error()))
		return NewParrotError(p.Name(), "CheckPermission", err)
	}

	// Build config for CCRunner
	// 为 CCRunner 构建配置
	cfg := &CCRunnerConfig{
		Mode:          p.mode.Name(),
		WorkDir:       p.workDir,
		SessionID:     p.sessionID,
		UserID:        p.userID,
		DeviceContext: p.deviceCtx,
	}
	cfg.SystemPrompt = p.mode.BuildSystemPrompt(cfg)

	// Execute via CCRunner
	// 通过 CCRunner 执行
	if err := p.runner.Execute(ctx, cfg, userInput, callback); err != nil {
		return NewParrotError(p.Name(), "Execute", err)
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
func (p *EvolutionParrot) sendError(callback EventCallback, message string) {
	if callback != nil {
		callback(EventTypeError, message)
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
func (p *EvolutionParrot) SelfDescribe() *ParrotSelfCognition {
	return &ParrotSelfCognition{
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
