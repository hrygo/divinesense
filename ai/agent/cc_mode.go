package agent

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/hrygo/divinesense/store"
)

// CCMode defines the interface for mode-specific behavior in CCRunner.
// CCMode 定义 CCRunner 中模式特定行为的接口。
//
// Each mode (Geek, Evolution, etc.) implements this interface to provide
// mode-specific configuration, permissions, and post-execution behavior.
// 每个模式（极客、进化等）实现此接口以提供模式特定的配置、权限和执行后行为。
type CCMode interface {
	// Name returns the mode identifier.
	Name() string

	// BuildSystemPrompt constructs the mode-specific system prompt.
	BuildSystemPrompt(cfg *CCRunnerConfig) string

	// GetWorkDir returns the working directory for the mode.
	GetWorkDir(userID int32) string

	// CheckPermission validates if the user can use this mode.
	CheckPermission(ctx context.Context, userID int32) error

	// OnComplete is called after successful execution.
	OnComplete(ctx context.Context) error
}

// GeekMode implements CCMode for the Geek Mode (user sandbox).
// GeekMode 为极客模式（用户沙箱）实现 CCMode。
type GeekMode struct {
	baseWorkDir string // Base directory for user sandboxes
}

// NewGeekMode creates a new GeekMode instance.
// NewGeekMode 创建一个新的 GeekMode 实例。
func NewGeekMode(baseWorkDir string) *GeekMode {
	// If no base dir provided, try environment variable
	if baseWorkDir == "" {
		baseWorkDir = os.Getenv("DIVINESENSE_CLAUDE_CODE_WORKDIR")
	}
	return &GeekMode{baseWorkDir: baseWorkDir}
}

// Name returns the mode identifier.
func (m *GeekMode) Name() string {
	return "geek"
}

// BuildSystemPrompt builds the Geek Mode system prompt.
// Geek Mode is a general-purpose assistant for code-related tasks.
// Adds Output Behavior section (Geek-specific) to base prompt.
func (m *GeekMode) BuildSystemPrompt(cfg *CCRunnerConfig) string {
	basePrompt := buildSystemPrompt(cfg.WorkDir, cfg.SessionID, cfg.UserID, cfg.DeviceContext)
	return basePrompt + fmt.Sprintf(`

# Output Behavior

You are running in an **embedded web service**, NOT a terminal.

Users can access created files directly via HTTP at:
  /file/geek/%d/<filename>

## Rules
- DO NOT Read files after creating them
- Chinese verbs "展示/显示/查看" mean: create file + announce path
- For files >500 lines: Write only, never Read
`, cfg.UserID)
}

// GetWorkDir returns the user-specific sandbox directory.
func (m *GeekMode) GetWorkDir(userID int32) string {
	base := m.baseWorkDir
	if base == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			base = "/tmp"
		} else {
			base = filepath.Join(homeDir, ".divinesense", "claude")
		}
	}
	return filepath.Join(base, fmt.Sprintf("user_%d", userID))
}

// CheckPermission validates that the user can use Geek Mode.
// All authenticated users can use Geek Mode.
func (m *GeekMode) CheckPermission(ctx context.Context, userID int32) error {
	if userID == 0 {
		return fmt.Errorf("user ID is required")
	}
	return nil
}

// OnComplete is a no-op for Geek Mode.
func (m *GeekMode) OnComplete(ctx context.Context) error {
	return nil
}

// EvolutionMode implements CCMode for Evolution Mode (self-evolution).
// EvolutionMode 为进化模式（自我进化）实现 CCMode。
//
// Evolution Mode allows DivineSense to modify its own source code using Claude Code CLI.
// The actual git operations and PR creation are handled by CC itself - this mode
// only provides configuration and permission checking.
type EvolutionMode struct {
	sourceDir  string
	adminOnly  bool
	envEnabled bool
	store      *store.Store // For user role checking
}

// EvolutionModeConfig holds configuration for EvolutionMode.
// EvolutionModeConfig 保存 EvolutionMode 的配置。
type EvolutionModeConfig struct {
	SourceDir string       // Project root directory for evolution
	AdminOnly bool         // Whether only admins can use evolution mode
	Store     *store.Store // Store for user role checking (optional, skips admin check if nil)
}

// NewEvolutionMode creates a new EvolutionMode instance.
// NewEvolutionMode 创建一个新的 EvolutionMode 实例。
func NewEvolutionMode(cfg *EvolutionModeConfig) *EvolutionMode {
	return &EvolutionMode{
		sourceDir:  cfg.SourceDir,
		adminOnly:  cfg.AdminOnly,
		envEnabled: os.Getenv("DIVINESENSE_EVOLUTION_ENABLED") == "true",
		store:      cfg.Store,
	}
}

// Name returns the mode identifier.
func (m *EvolutionMode) Name() string {
	return "evolution"
}

// BuildSystemPrompt builds the Evolution Mode system prompt.
// Implements a research-first workflow: idea-researcher → Issue → Planning → PR
func (m *EvolutionMode) BuildSystemPrompt(cfg *CCRunnerConfig) string {
	return `# Evolution Mode 🧬

You are evolving DivineSense's source code through a structured, interactive process.

## 决策树 (Decision Tree)

当用户提出请求时，首先判断请求的明确程度：

### 路径 A：模糊请求 / 新 Idea → 交互式调研优先
**触发条件**：用户描述一个想法、概念、或开放性问题
**示例**："能不能加个 XXX 功能"、"我想优化 YYY"、"有个点子..."

1. **启动 idea-researcher**
   Use /idea-researcher skill to conduct interactive research:
   - 阶段1: 理解与扩展用户 idea
   - 阶段2: 深度调研（技术可行性、用户价值、复杂度）
   - 阶段3: 方案设计
   - 阶段4: 迭代修订（与用户确认）
   - 阶段5: 创建 GitHub Issue
   - 阶段6: 保存调研报告

2. **调研完成后**
   - 询问用户：是否现在执行？还是留待后续？
   - 如果用户选择执行 → 进入路径 B

### 路径 B：明确命令 → 详细规划 + 确认执行
**触发条件**：用户给出具体、可操作的命令
**示例**："执行 Issue #123"、"修复 XXX bug，错误信息是 YYY"、"按照 spec XXX 实现"

1. **详细规划**
   - 分析任务范围和影响
   - 列出具体实现步骤
   - 识别风险和依赖

2. **等待用户确认**
   Output planning summary, then ask:
   "请确认规划是否正确。回复 '执行' 开始实现，或提出修改意见。"

3. **执行（仅在用户确认后）**
   - 创建 feature/evolution 分支
   - 实现变更
   - 运行 make check-all
   - 通过 PR 提交（禁止直接 push main）

## 核心规则

1. **Read @CLAUDE.md first** — 理解项目架构和规范
2. **Follow @.claude/rules/git-workflow.md** — 严格遵循 Git 工作流
3. **All changes via PR** — 禁止直接修改 main 分支
4. **Always confirm before execution** — 重大变更必须用户确认

## 快捷指令

| 用户输入 | 行为 |
|:---------|:-----|
| "调研 X" / "分析 X" | 启动 idea-researcher |
| "执行 #N" | 按 Issue #N 规划并确认后执行 |
| "继续" | 从上次中断处继续 |
| "确认" / "执行" | 开始实现已确认的规划 |
`
}

// GetWorkDir returns the source code directory for evolution.
func (m *EvolutionMode) GetWorkDir(userID int32) string {
	return m.sourceDir
}

// CheckPermission validates that the user can use Evolution Mode.
// Only admins can use Evolution Mode when enabled via environment variable.
func (m *EvolutionMode) CheckPermission(ctx context.Context, userID int32) error {
	// Check environment variable
	if !m.envEnabled {
		return fmt.Errorf("evolution mode is disabled (set DIVINESENSE_EVOLUTION_ENABLED=true)")
	}

	// Check admin status
	if m.adminOnly && !m.isAdmin(ctx, userID) {
		return fmt.Errorf("evolution mode requires admin privileges")
	}

	return nil
}

// isAdmin checks if the user is an administrator.
// isAdmin 检查用户是否为管理员。
//
// If no Store is configured, returns false (deny by default).
// 如果没有配置 Store，返回 false（默认拒绝）。
func (m *EvolutionMode) isAdmin(ctx context.Context, userID int32) bool {
	if m.store == nil {
		return false
	}

	user, err := m.store.GetUser(ctx, &store.FindUser{ID: &userID})
	if err != nil {
		return false
	}

	return user.Role == store.RoleAdmin || user.Role == store.RoleHost
}

// OnComplete is a no-op for Evolution Mode (CC handles PR creation).
func (m *EvolutionMode) OnComplete(ctx context.Context) error {
	// CC handles git operations and PR creation automatically
	return nil
}
