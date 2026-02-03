package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	// Default timeout for Claude Code CLI execution.
	defaultCodeTimeout = 5 * time.Minute

	// Maximum file changes allowed per session.
	maxFileChanges = 100
)

// ClaudeCodeSession manages a Claude Code CLI session for a user.
// Maintains session state and resource limits.
// Uses atomic operations for FileChanges to ensure thread safety.
type ClaudeCodeSession struct {
	CreatedAt   time.Time
	SessionName string
	WorkDir     string
	UserID      int32
	FileChanges atomic.Int32
}

// ClaudeCodeTool integrates Claude Code CLI for code-related tasks.
// Only available when Geek Mode is enabled by the user.
// Uses sync.Map for concurrent-safe session storage.
type ClaudeCodeTool struct {
	userIDGetter func(ctx context.Context) int32
	sessions     sync.Map
	workDir      string
	timeout      time.Duration
	enabled      bool
}

// NewClaudeCodeTool creates a new Claude Code CLI integration tool.
// NewClaudeCodeTool 创建一个新的 Claude Code CLI 集成工具。
//
// The tool requires Claude Code CLI to be installed on the system.
// When disabled, Run() returns a friendly error message.
func NewClaudeCodeTool(
	enabled bool,
	workDir string,
	userIDGetter func(ctx context.Context) int32,
) (*ClaudeCodeTool, error) {
	if userIDGetter == nil {
		return nil, errors.New("userIDGetter cannot be nil")
	}

	// Resolve work directory
	if workDir == "" || workDir == "." {
		workDir = getWorkingDirectory()
	}

	// Ensure work directory exists
	if err := os.MkdirAll(workDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create work directory: %w", err)
	}

	timeout := defaultCodeTimeout

	return &ClaudeCodeTool{
		enabled:      enabled,
		workDir:      workDir,
		timeout:      timeout,
		userIDGetter: userIDGetter,
	}, nil
}

// Name returns the name of the tool.
// Name 返回工具名称。
func (t *ClaudeCodeTool) Name() string {
	return "claude_code"
}

// Description returns a description of what the tool does.
// Description 返回工具描述。
func (t *ClaudeCodeTool) Description() string {
	if !t.enabled {
		return `Claude Code CLI integration (DISABLED).

This tool is currently disabled. Enable Geek Mode to use Claude Code CLI for code-related tasks.

Note: Claude Code CLI must be installed on the system to use this tool.`
	}

	return `Executes Claude Code CLI for code-related tasks in headless mode.

INPUT FORMAT:
{"prompt": "your coding task or question"}

OUTPUT FORMAT (text):
[Claude Code CLI Output]
result...

[End of Output]

NOTES:
- This tool runs Claude Code CLI in headless mode with --print flag
- All file operations are scoped to the configured work directory
- Each user has an isolated session
- Maximum execution time: 5 minutes
- File changes are tracked and limited for safety
`
}

// ClaudeCodeInput represents the input for Claude Code CLI.
type ClaudeCodeInput struct {
	Prompt string `json:"prompt"` // Required: The coding task or question
}

// Run executes the Claude Code CLI tool.
// Run 执行 Claude Code CLI 工具。
func (t *ClaudeCodeTool) Run(ctx context.Context, input string) (string, error) {
	// Add timeout protection
	ctx, cancel := context.WithTimeout(ctx, t.timeout)
	defer cancel()

	// Check if enabled
	if !t.enabled {
		return "", errors.New("Geek Mode is disabled. Enable Geek Mode to use Claude Code CLI")
	}

	// Parse input
	var codeInput ClaudeCodeInput
	if err := json.Unmarshal([]byte(input), &codeInput); err != nil {
		return "", fmt.Errorf("invalid JSON input: %w", err)
	}

	// Validate prompt
	if strings.TrimSpace(codeInput.Prompt) == "" {
		return "", errors.New("prompt cannot be empty")
	}

	// Get user ID and session
	userID := t.userIDGetter(ctx)
	session := t.getOrCreateSession(ctx, userID)

	// Check file change limit using atomic operation
	if session.FileChanges.Load() >= int32(maxFileChanges) {
		return "", fmt.Errorf("file change limit reached for this session (%d changes)", maxFileChanges)
	}

	// Build command
	cmd := t.buildCommand(session, codeInput.Prompt)

	// Execute with timeout
	output, err := t.executeCommand(ctx, cmd)
	if err != nil {
		return "", fmt.Errorf("execution failed: %w", err)
	}

	// Track file changes using atomic increment (simplified - just count executions as changes)
	session.FileChanges.Add(1)

	return output, nil
}

// buildCommand constructs the Claude Code CLI command.
func (t *ClaudeCodeTool) buildCommand(session *ClaudeCodeSession, prompt string) *exec.Cmd {
	// Claude Code CLI headless mode command:
	// claude code --print --output-format stream-json --prompt "..."

	// Note: The actual CLI invocation depends on the installed package
	// Common paths: "claude" or full path to installed binary

	args := []string{
		"code",
		"--print",
		"--output-format", "stream-json",
		"--",
		prompt,
	}

	cmd := exec.Command("claude", args...)

	// Set working directory for file operations
	cmd.Dir = session.WorkDir

	// Set environment to ensure non-interactive mode
	cmd.Env = append(os.Environ(),
		"CLAUDE_DISABLE_TELEMETRY=1",
		"CLAUDE_HEADLESS=1",
	)

	return cmd
}

// executeCommand runs the command with timeout and returns output.
// Includes proper error handling and timeout protection.
func (t *ClaudeCodeTool) executeCommand(ctx context.Context, cmd *exec.Cmd) (string, error) {
	// Create a channel to capture the result
	type result struct {
		err    error
		output string
	}
	resultCh := make(chan result, 1)

	go func() {
		output, err := cmd.CombinedOutput()
		resultCh <- result{err: err, output: string(output)}
	}()

	select {
	case <-ctx.Done():
		// Attempt to kill the process, ignoring errors if process already done
		if cmd.Process != nil {
			if err := cmd.Process.Kill(); err != nil && !errors.Is(err, os.ErrProcessDone) {
				slog.Warn("Failed to kill timed out process", "error", err)
			}
		}
		return "", fmt.Errorf("execution timeout after %v", t.timeout)
	case res := <-resultCh:
		if res.err != nil {
			// Include output in error for debugging
			return "", fmt.Errorf("execution failed: %w, output: %s", res.err, res.output)
		}
		return res.output, nil
	}
}

// getOrCreateSession gets an existing session or creates a new one.
// Uses sync.Map.LoadOrStore for concurrent-safe access.
func (t *ClaudeCodeTool) getOrCreateSession(_ context.Context, userID int32) *ClaudeCodeSession {
	// Use LoadOrStore for atomic check-and-create operation
	actual, _ := t.sessions.LoadOrStore(userID, &ClaudeCodeSession{
		UserID:      userID,
		SessionName: fmt.Sprintf("user_%d_%d", userID, time.Now().Unix()),
		WorkDir:     t.workDir,
		CreatedAt:   time.Now(),
	})

	session, ok := actual.(*ClaudeCodeSession)
	if !ok {
		// This should never happen since we control the map contents
		slog.Error("Unexpected type in sessions map", "user_id", userID)
		return &ClaudeCodeSession{
			UserID:      userID,
			SessionName: "fallback",
			WorkDir:     t.workDir,
			CreatedAt:   time.Now(),
		}
	}

	// Log if this was a newly created session (FileChanges is 0)
	if session.FileChanges.Load() == 0 {
		slog.Info("Created Claude Code session",
			"user_id", userID,
			"session", session.SessionName,
			"work_dir", session.WorkDir)
	}

	return session
}

// getWorkingDirectory returns the appropriate working directory.
func getWorkingDirectory() string {
	// In production, use a dedicated work directory
	// For now, use current directory
	if dir, err := os.Getwd(); err == nil {
		return dir
	}
	return "."
}

// IsEnabled returns whether the tool is enabled.
func (t *ClaudeCodeTool) IsEnabled() bool {
	return t.enabled
}

// CleanupOldSessions removes sessions older than the specified duration.
// Uses sync.Map.Range for concurrent-safe iteration.
func (t *ClaudeCodeTool) CleanupOldSessions(olderThan time.Duration) {
	now := time.Now()

	// Collect keys to delete first, then delete them
	var keysToDelete []int32

	t.sessions.Range(func(key, value any) bool {
		userID, ok := key.(int32)
		if !ok {
			return true
		}
		session, ok := value.(*ClaudeCodeSession)
		if !ok {
			return true
		}

		if now.Sub(session.CreatedAt) > olderThan {
			keysToDelete = append(keysToDelete, userID)
		}
		return true
	})

	// Delete collected keys
	for _, userID := range keysToDelete {
		if actual, loaded := t.sessions.LoadAndDelete(userID); loaded {
			session, ok := actual.(*ClaudeCodeSession)
			if ok {
				slog.Info("Cleaned up old Claude Code session",
					"user_id", userID,
					"session", session.SessionName,
					"age_minutes", now.Sub(session.CreatedAt).Minutes())
			}
		}
	}
}
