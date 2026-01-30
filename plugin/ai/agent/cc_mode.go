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
	sourceDir string // Project root directory (for reference only)
}

// NewGeekMode creates a new GeekMode instance.
// NewGeekMode 创建一个新的 GeekMode 实例。
func NewGeekMode(sourceDir string) *GeekMode {
	return &GeekMode{sourceDir: sourceDir}
}

// Name returns the mode identifier.
func (m *GeekMode) Name() string {
	return "geek"
}

// BuildSystemPrompt builds the Geek Mode system prompt.
// Geek Mode is a general-purpose assistant for code-related tasks.
// Adds File Output section (Geek-specific) to base prompt.
func (m *GeekMode) BuildSystemPrompt(cfg *CCRunnerConfig) string {
	basePrompt := buildSystemPrompt(cfg.WorkDir, cfg.SessionID, cfg.UserID, cfg.DeviceContext)
	return basePrompt + `

# File Output

When you create a file, announce the filename so the user knows it was created.
`
}

// GetWorkDir returns the user-specific sandbox directory.
func (m *GeekMode) GetWorkDir(userID int32) string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "/tmp"
	}
	return filepath.Join(homeDir, ".divinesense", "claude", fmt.Sprintf("user_%d", userID))
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
func (m *EvolutionMode) BuildSystemPrompt(cfg *CCRunnerConfig) string {
	return `# Evolution Mode 🧬

You are modifying DivineSense's own source code.

## Rules
- Follow @CLAUDE.md
- All changes via PR

Read @CLAUDE.md first.
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
