package geek

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	agentpkg "github.com/hrygo/divinesense/ai/agents"
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

	// BaseSystemPrompt returns the fixed part of the system prompt.
	// This should be passed to hotplex.EngineOptions.BaseSystemPrompt.
	BaseSystemPrompt() string

	// BuildSystemPrompt constructs the mode-specific system prompt.
	//
	// Deprecated: Use BaseSystemPrompt() and BuildContextPrompt() instead.
	BuildSystemPrompt(cfg *agentpkg.CCRunnerConfig) string

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
//
// Deprecated: Use BaseSystemPrompt() and BuildContextPrompt() instead.
func (m *GeekMode) BuildSystemPrompt(cfg *agentpkg.CCRunnerConfig) string {
	return m.BaseSystemPrompt() + "\n" + m.BuildContextPrompt(cfg)
}

// BaseSystemPrompt returns the fixed part of the Geek Mode system prompt.
// This should be passed to hotplex.EngineOptions.BaseSystemPrompt.
func (m *GeekMode) BaseSystemPrompt() string {
	return agentpkg.DivineSenseBaseContext + `

# Output Behavior

You are running in an **embedded web service**, NOT a terminal.

Users can access created files directly via HTTP at:
  /file/geek/{user_id}/<filename>

## Rules
- DO NOT Read files after creating them
- Chinese verbs "展示/显示/查看" mean: create file + announce path
- For files >500 lines: Write only, never Read
`
}

// BuildContextPrompt builds the user-specific context prompt.
// This should be passed to hotplex.Config.TaskSystemPrompt on first session creation.
// Subsequent requests should NOT include this (user context is already established).
func (m *GeekMode) BuildContextPrompt(cfg *agentpkg.CCRunnerConfig) string {
	return agentpkg.BuildUserContextPrompt(cfg.WorkDir, cfg.SessionID, cfg.UserID, cfg.DeviceContext)
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
//
// Deprecated: Use BaseSystemPrompt() instead (EvolutionMode has no dynamic context).
func (m *EvolutionMode) BuildSystemPrompt(cfg *agentpkg.CCRunnerConfig) string {
	return m.BaseSystemPrompt()
}

// BaseSystemPrompt returns the Evolution Mode system prompt.
// This should be passed to hotplex.EngineOptions.BaseSystemPrompt.
func (m *EvolutionMode) BaseSystemPrompt() string {
	return agentpkg.DivineSenseBaseContext + `

# Evolution Mode 🧬

You are evolving DivineSense's source code through a structured, interactive process.

## Decision Tree

### Path A: Vague Request → Research First
**Trigger**: "Can we add XXX", "I want to optimize YYY", "I have an idea..."

→ Use /idea-researcher skill (interactive research → create Issue)
→ After research: Ask user whether to execute now?

### Path B: Clear Command → Plan & Confirm
**Trigger**: "Execute #123", "Fix XXX bug", "Implement per spec"

1. Output detailed plan (scope, steps, risks)
2. Wait for confirmation: "Please confirm the plan. Reply 'execute' to proceed, or suggest changes."
3. After confirmation: Follow @.claude/rules/git-workflow.md

## Core Rules

- Read @CLAUDE.md first
- Follow @.claude/rules/git-workflow.md
- All changes via PR
- Always confirm before execution
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
