package agent

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

	// OnComplete is called after successful execution (for PR creation, etc.).
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
func (m *GeekMode) BuildSystemPrompt(cfg *CCRunnerConfig) string {
	// Use the existing buildSystemPrompt function from geek_parrot.go
	return buildSystemPrompt(cfg.WorkDir, cfg.SessionID, cfg.UserID, cfg.DeviceContext)
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

// OnComplete is a no-op for Geek Mode (no PR needed).
func (m *GeekMode) OnComplete(ctx context.Context) error {
	return nil
}

// EvolutionMode implements CCMode for Evolution Mode (self-evolution).
// EvolutionMode 为进化模式（自我进化）实现 CCMode。
type EvolutionMode struct {
	sourceDir     string
	gitService    *GitService
	githubService *GitHubService
	adminOnly     bool
	envEnabled    bool
}

// EvolutionModeConfig holds configuration for EvolutionMode.
// EvolutionModeConfig 保存 EvolutionMode 的配置。
type EvolutionModeConfig struct {
	SourceDir       string
	GitHubToken     string // Optional: for PR creation
	RepositoryOwner string // For PR creation
	RepositoryName  string // For PR creation
	AdminOnly       bool
}

// NewEvolutionMode creates a new EvolutionMode instance.
// NewEvolutionMode 创建一个新的 EvolutionMode 实例。
func NewEvolutionMode(cfg *EvolutionModeConfig) *EvolutionMode {
	mode := &EvolutionMode{
		sourceDir:  cfg.SourceDir,
		adminOnly:  cfg.AdminOnly,
		envEnabled: os.Getenv("DIVINESENSE_EVOLUTION_ENABLED") == "true",
	}

	// Initialize Git service
	if cfg.SourceDir != "" {
		mode.gitService = NewGitService(cfg.SourceDir)
	}

	// Initialize GitHub service if token provided
	if cfg.GitHubToken != "" {
		mode.githubService = NewGitHubService(cfg.GitHubToken, cfg.RepositoryOwner, cfg.RepositoryName)
	}

	return mode
}

// Name returns the mode identifier.
func (m *EvolutionMode) Name() string {
	return "evolution"
}

// BuildSystemPrompt builds the Evolution Mode system prompt.
// Evolution Mode emphasizes following CLAUDE.md and making careful changes.
func (m *EvolutionMode) BuildSystemPrompt(cfg *CCRunnerConfig) string {
	basePrompt := buildSystemPrompt(cfg.WorkDir, cfg.SessionID, cfg.UserID, cfg.DeviceContext)

	evolutionPrompt := `

# EVOLUTION MODE 🧬

You are operating in **Evolution Mode** inside DivineSense.

**CRITICAL**: You are now modifying DivineSense's OWN source code.
This is a self-evolution scenario where you improve the system you are part of.

## Working Directory
- **Source Root**: %s
- **Current Branch**: evolution/%s
- **Session**: %s

## Evolution Guidelines
1. **Safety First**: Never modify .env, secrets, or deployment configs
2. **Atomic Changes**: Make small, focused commits
3. **Test Before Commit**: Run tests before committing
4. **Update Docs**: If you change behavior, update CLAUDE.md
5. **Git Hygiene**: Use conventional commits (feat/fix/refactor)

## Path Constraints
- **Allowed**: plugin/, server/, web/src/, docs/, CLAUDE.md
- **Forbidden**: .env*, *.secret*, deploy/, .git/, go.mod, go.sum

Begin by analyzing the relevant code, then propose a plan before making changes.
`

	return basePrompt + fmt.Sprintf(evolutionPrompt, cfg.WorkDir, cfg.SessionID, cfg.SessionID)
}

// GetWorkDir returns the source code directory.
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

	// Check admin status (implementation depends on auth system)
	if m.adminOnly && !m.isAdmin(ctx, userID) {
		return fmt.Errorf("evolution mode requires admin privileges")
	}

	return nil
}

// isAdmin checks if the user is an administrator.
// This is a placeholder - actual implementation depends on auth system.
func (m *EvolutionMode) isAdmin(ctx context.Context, userID int32) bool {
	// TODO: Implement actual admin check using auth service
	// For now, return false to require explicit implementation
	return false
}

// OnComplete creates a GitHub PR after evolution execution.
func (m *EvolutionMode) OnComplete(ctx context.Context) error {
	if m.githubService == nil {
		// No GitHub service configured, skip PR creation
		return nil
	}

	// Get current branch
	currentBranch, err := m.gitService.GetCurrentBranch()
	if err != nil {
		return fmt.Errorf("failed to get current branch: %w", err)
	}

	// Create PR
	prURL, err := m.githubService.CreatePR(ctx, &CreatePRRequest{
		HeadBranch: currentBranch,
		BaseBranch: "main",
		Title:      fmt.Sprintf("Evolution: %s", currentBranch),
		Body:       "Automated evolution changes. Please review carefully.",
	})
	if err != nil {
		return fmt.Errorf("failed to create PR: %w", err)
	}

	// Log PR creation
	logger := ctx.Value("logger").(*struct{ URL string })
	if logger != nil {
		logger.URL = prURL
	}

	return nil
}

// ValidatePath checks if a path is allowed for modification in Evolution Mode.
// Evolution Mode 的路径白名单/黑名单验证。
func (m *EvolutionMode) ValidatePath(path string) error {
	// Normalize path
	path = filepath.Clean(path)

	// Check forbidden paths (blacklist)
	// 检查禁止路径（黑名单）
	forbiddenPatterns := []string{
		".env",
		".secret",
		"deploy/",
		".git/",
		"go.mod",
		"go.sum",
	}

	for _, pattern := range forbiddenPatterns {
		if strings.Contains(path, pattern) {
			return fmt.Errorf("path is forbidden: %s (matches %s)", path, pattern)
		}
	}

	// Check allowed paths (whitelist) - if specified, only allow these
	// 检查允许路径（白名单）- 如果指定，只允许这些
	allowedPatterns := []string{
		"plugin/",
		"server/",
		"web/src/",
		"docs/",
		"CLAUDE.md",
	}

	// If path is in project root, check if it's explicitly allowed
	if !strings.Contains(filepath.Dir(path), "/") {
		found := false
		for _, pattern := range allowedPatterns {
			if strings.HasPrefix(path, strings.TrimSuffix(pattern, "/")) {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("path not in allowed list: %s", path)
		}
	}

	return nil
}
