package agent

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/google/uuid"
)

const (
	// Scanner buffer sizes for CLI output parsing.
	// 扫描器缓冲区大小，用于 CLI 输出解析。
	scannerInitialBufSize = 256 * 1024  // 256 KB
	scannerMaxBufSize     = 1024 * 1024 // 1 MB

	// Maximum length of non-JSON output to log.
	// 非 JSON 输出的最大日志长度。
	maxNonJSONOutputLength = 100
)

// UUID v5 namespace for DivineSense session mapping.
// Using a custom namespace ensures deterministic UUID generation from ConversationID.
// DivineSense 专用的 UUID v5 命名空间，用于会话映射。
var divineSenseNamespace = uuid.MustParse("6ba7b811-9dad-11d1-80b4-00c04fd430c8") // TODO: register proper namespace

// ConversationIDToSessionID converts a database ConversationID to a deterministic UUID v5.
// This ensures the same ConversationID always maps to the same SessionID,
// enabling reliable session resume across backend restarts.
// 将数据库 ConversationID 转换为确定性的 UUID v5。
// 确保相同的 ConversationID 始终映射到相同的 SessionID，实现跨重启的可靠会话恢复。
func ConversationIDToSessionID(conversationID int64) string {
	// UUID v5 uses SHA-1 hash of namespace + name
	// Use conversation ID as string bytes for deterministic mapping
	name := fmt.Sprintf("divinesense:conversation:%d", conversationID)
	return uuid.NewSHA1(divineSenseNamespace, []byte(name)).String()
}

// buildSystemPrompt provides minimal, high-signal context for Claude Code CLI.
// buildSystemPrompt 为 Claude Code CLI 提供最小化、高信噪比的上下文。
func buildSystemPrompt(workDir, sessionID string, userID int32, deviceContext string) string {
	osName := runtime.GOOS
	arch := runtime.GOARCH
	if osName == "darwin" {
		osName = "macOS"
	}

	timestamp := time.Now().Format(time.RFC3339)

	// Try to parse device context for better formatting
	// 尝试解析设备上下文以便更好地格式化
	var contextMap map[string]any
	userAgent := "Unknown"
	deviceInfo := "Unknown"
	if deviceContext != "" {
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
			// Add more fields if available (screen, language, etc.)
			// 如果有更多字段则添加（屏幕、语言等）
			if w, ok := contextMap["screenWidth"].(float64); ok {
				if h, ok := contextMap["screenHeight"].(float64); ok {
					deviceInfo = fmt.Sprintf("%s (%dx%d)", deviceInfo, int(w), int(h))
				}
			}
			if lang, ok := contextMap["language"].(string); ok {
				deviceInfo = fmt.Sprintf("%s, Language: %s", deviceInfo, lang)
			}
		} else {
			// Fallback: use raw string if not JSON
			userAgent = deviceContext
		}
	}

	return fmt.Sprintf(`# Context

You are running inside DivineSense, an intelligent assistant system.

**User Interaction**: Users type questions in their web browser, which invokes you via a Go backend. Your response streams back to their browser in real-time.

- **User ID**: %d
- **Client Device**: %s
- **User Agent**: %s
- **Server OS**: %s (%s)
- **Time**: %s
- **Workspace**: %s
- **Mode**: Non-interactive headless (--print)
- **Session**: %s (persists via --session-id/--resume)
`, userID, deviceInfo, userAgent, osName, arch, timestamp, workDir, sessionID)
}

// StreamMessage represents a single event in the stream-json format.
// StreamMessage 表示 stream-json 格式中的单个事件。
type StreamMessage struct {
	Message   *AssistantMessage `json:"message,omitempty"`
	Input     map[string]any    `json:"input,omitempty"`
	Type      string            `json:"type"`
	Timestamp string            `json:"timestamp,omitempty"`
	SessionID string            `json:"session_id,omitempty"`
	Role      string            `json:"role,omitempty"`
	Name      string            `json:"name,omitempty"`
	Output    string            `json:"output,omitempty"`
	Status    string            `json:"status,omitempty"`
	Error     string            `json:"error,omitempty"`
	Content   []ContentBlock    `json:"content,omitempty"`
	Duration  int               `json:"duration_ms,omitempty"`
}

// GetContentBlocks returns the content blocks, checking both direct and nested locations.
// GetContentBlocks 返回内容块，同时检查直接和嵌套位置。
func (m *StreamMessage) GetContentBlocks() []ContentBlock {
	if m.Message != nil && len(m.Message.Content) > 0 {
		return m.Message.Content
	}
	return m.Content
}

// AssistantMessage represents the nested message structure in assistant events.
// AssistantMessage 表示 assistant 事件中的嵌套消息结构。
type AssistantMessage struct {
	ID      string         `json:"id,omitempty"`
	Type    string         `json:"type,omitempty"`
	Role    string         `json:"role,omitempty"`
	Content []ContentBlock `json:"content,omitempty"`
}

// ContentBlock represents a content block in stream-json format.
// ContentBlock 表示 stream-json 格式中的内容块。
type ContentBlock struct {
	Type    string         `json:"type"`
	Text    string         `json:"text,omitempty"`
	Name    string         `json:"name,omitempty"`
	ID      string         `json:"id,omitempty"`
	Input   map[string]any `json:"input,omitempty"`
	Content string         `json:"content,omitempty"`
	IsError bool           `json:"is_error,omitempty"`
}

// CCRunner is the unified Claude Code CLI integration layer.
// CCRunner 是统一的 Claude Code CLI 集成层。
//
// It provides a shared implementation for all modes that need to interact
// with Claude Code CLI (Geek Mode, Evolution Mode, etc.).
// 它为所有需要与 Claude Code CLI 交互的模式提供共享实现（极客模式、进化模式等）。
type CCRunner struct {
	cliPath string
	timeout time.Duration
	logger  *slog.Logger
	mu      sync.Mutex
	manager *CCSessionManager
}

// CCRunnerConfig defines mode-specific configuration for CCRunner execution.
// CCRunnerConfig 定义 CCRunner 执行的模式特定配置。
type CCRunnerConfig struct {
	Mode           string // "geek" | "evolution"
	WorkDir        string // Working directory for CLI
	ConversationID int64  // Database conversation ID for deterministic UUID v5 mapping
	SessionID      string // Session identifier (derived from ConversationID if empty)
	UserID         int32  // User ID for logging/context
	SystemPrompt   string // Mode-specific system prompt
	DeviceContext  string // Device/browser context JSON

	// Security / Permission Control
	// 安全/权限控制
	PermissionMode string // "default", "bypassPermissions", etc.

	// Evolution Mode specific
	// 进化模式专用
	AllowedPaths   []string // Path whitelist (evolution mode)
	ForbiddenPaths []string // Path blacklist (evolution mode)
}

// NewCCRunner creates a new CCRunner instance.
// NewCCRunner 创建一个新的 CCRunner 实例。
func NewCCRunner(timeout time.Duration, logger *slog.Logger) (*CCRunner, error) {
	cliPath, err := exec.LookPath("claude")
	if err != nil {
		return nil, fmt.Errorf("Claude Code CLI not found: %w", err)
	}

	if logger == nil {
		logger = slog.Default()
	}

	return &CCRunner{
		cliPath: cliPath,
		timeout: timeout,
		logger:  logger,
		manager: NewCCSessionManager(logger, 30*time.Minute), // Default 30m idle timeout
	}, nil
}

// Execute runs Claude Code CLI with the given configuration and streams events.
// Execute 使用给定配置运行 Claude Code CLI 并流式传输事件。
func (r *CCRunner) Execute(ctx context.Context, cfg *CCRunnerConfig, prompt string, callback EventCallback) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Derive SessionID from ConversationID using UUID v5 for deterministic mapping.
	// This ensures the same conversation always maps to the same session,
	// enabling reliable resume across backend restarts (per spec 2.2).
	// 使用 UUID v5 从 ConversationID 派生 SessionID，实现确定性映射。
	// 确保同一对话始终映射到同一会话，实现跨重启的可靠恢复（规格 2.2）。
	if cfg.SessionID == "" && cfg.ConversationID > 0 {
		cfg.SessionID = ConversationIDToSessionID(cfg.ConversationID)
		r.logger.Debug("CCRunner: derived SessionID from ConversationID",
			"conversation_id", cfg.ConversationID,
			"session_id", cfg.SessionID)
	}

	// Validate configuration
	// 验证配置
	if err := r.validateConfig(cfg); err != nil {
		return fmt.Errorf("config validation failed: %w", err)
	}

	// Ensure working directory exists
	// 确保工作目录存在
	if err := os.MkdirAll(cfg.WorkDir, 0755); err != nil {
		return fmt.Errorf("failed to create work directory: %w", err)
	}

	// Determine if this is a first call or resume
	// 确定是首次调用还是恢复
	sessionDir := filepath.Join(cfg.WorkDir, ".claude", "sessions", cfg.SessionID)
	firstCall := r.isFirstCall(sessionDir)

	if firstCall {
		if err := os.MkdirAll(sessionDir, 0755); err != nil {
			r.logger.Warn("Failed to create session directory",
				"user_id", cfg.UserID,
				"session_id", cfg.SessionID,
				"error", err)
		}
		r.logger.Info("CCRunner: Starting NEW session",
			"user_id", cfg.UserID,
			"mode", cfg.Mode,
			"session_id", cfg.SessionID)
	} else {
		r.logger.Info("CCRunner: Resuming EXISTING session",
			"user_id", cfg.UserID,
			"mode", cfg.Mode,
			"session_id", cfg.SessionID)
	}

	// Send thinking event
	// 发送思考事件
	if callback != nil {
		if err := callback(EventTypeThinking, fmt.Sprintf("ai.%s_mode.thinking", cfg.Mode)); err != nil {
			return err
		}
	}

	// Execute CLI with session management
	// 执行 CLI 并管理会话
	if err := r.executeWithSession(ctx, cfg, prompt, firstCall, callback); err != nil {
		r.logger.Error("CCRunner: execution failed",
			"user_id", cfg.UserID,
			"mode", cfg.Mode,
			"error", err)
		return err
	}

	return nil
}

// StartAsyncSession starts a persistent session and returns the session object.
func (r *CCRunner) StartAsyncSession(ctx context.Context, cfg *CCRunnerConfig) (*Session, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Derive SessionID from ConversationID using UUID v5 for deterministic mapping.
	// 使用 UUID v5 从 ConversationID 派生 SessionID，实现确定性映射。
	if cfg.SessionID == "" && cfg.ConversationID > 0 {
		cfg.SessionID = ConversationIDToSessionID(cfg.ConversationID)
		r.logger.Debug("CCRunner: derived SessionID from ConversationID",
			"conversation_id", cfg.ConversationID,
			"session_id", cfg.SessionID)
	}

	if err := r.validateConfig(cfg); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	// Ensure working directory exists
	if err := os.MkdirAll(cfg.WorkDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create work directory: %w", err)
	}

	// Create session via manager
	return r.manager.GetOrCreateSession(ctx, cfg.SessionID, *cfg)
}

// GetSessionManager returns the improved session manager.
func (r *CCRunner) GetSessionManager() *CCSessionManager {
	return r.manager
}

// validateConfig validates the CCRunnerConfig.
// validateConfig 验证 CCRunnerConfig。
func (r *CCRunner) validateConfig(cfg *CCRunnerConfig) error {
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

// isFirstCall checks if this is the first call for a session.
// isFirstCall 检查是否是会话的首次调用。
func (r *CCRunner) isFirstCall(sessionDir string) bool {
	_, err := os.Stat(sessionDir)
	return os.IsNotExist(err)
}

// executeWithSession executes Claude Code CLI with appropriate session flags.
// executeWithSession 使用适当的会话标志执行 Claude Code CLI。
func (r *CCRunner) executeWithSession(
	ctx context.Context,
	cfg *CCRunnerConfig,
	prompt string,
	firstCall bool,
	callback EventCallback,
) error {
	// Build system prompt
	// 构建系统提示词
	systemPrompt := cfg.SystemPrompt
	if systemPrompt == "" {
		systemPrompt = buildSystemPrompt(cfg.WorkDir, cfg.SessionID, cfg.UserID, cfg.DeviceContext)
	}

	// Build command arguments
	// 构建命令参数
	var args []string
	if firstCall {
		args = []string{
			"--print",
			"--verbose",
			"--append-system-prompt", systemPrompt,
			"--session-id", cfg.SessionID,
			"--output-format", "stream-json",
		}

		if cfg.PermissionMode != "" {
			args = append(args, "--permission-mode", cfg.PermissionMode)
		}

		args = append(args, prompt)
	} else {
		args = []string{
			"--print",
			"--verbose",
			"--append-system-prompt", systemPrompt,
			"--resume", cfg.SessionID,
			"--output-format", "stream-json",
		}

		if cfg.PermissionMode != "" {
			args = append(args, "--permission-mode", cfg.PermissionMode)
		}

		args = append(args, prompt)
	}

	cmd := exec.CommandContext(ctx, r.cliPath, args...)
	cmd.Dir = cfg.WorkDir

	// Set environment for programmatic usage
	// 设置程序化使用环境变量
	cmd.Env = append(os.Environ(),
		"CLAUDE_DISABLE_TELEMETRY=1",
	)

	// Get pipes
	// 获取管道
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("stdout pipe: %w", err)
	}
	defer stdout.Close()

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("stderr pipe: %w", err)
	}
	defer stderr.Close()

	// Start command
	// 启动命令
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start command: %w", err)
	}

	// Stream output with timeout
	// 带超时流式输出
	if err := r.streamOutput(ctx, cfg, stdout, stderr, callback); err != nil {
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
		return err
	}

	// Wait for command completion
	// 等待命令完成
	waitErr := cmd.Wait()
	if waitErr != nil {
		exitCode := 0
		if cmd.ProcessState != nil {
			exitCode = cmd.ProcessState.ExitCode()
		}
		return fmt.Errorf("command exited with code %d: %w", exitCode, waitErr)
	}

	return nil
}

// streamOutput reads and parses stream-json output from CLI.
// streamOutput 读取并解析 CLI 的 stream-json 输出。
func (r *CCRunner) streamOutput(
	ctx context.Context,
	cfg *CCRunnerConfig,
	stdout, stderr io.ReadCloser,
	callback EventCallback,
) error {
	var wg sync.WaitGroup
	errCh := make(chan error, 2)
	done := make(chan struct{})

	// Stream stdout
	// 流式处理 stdout
	wg.Add(1)
	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(stdout)
		buf := make([]byte, 0, scannerInitialBufSize)
		scanner.Buffer(buf, scannerMaxBufSize)

		for scanner.Scan() {
			line := scanner.Text()
			if line == "" {
				continue
			}

			var msg StreamMessage
			if err := json.Unmarshal([]byte(line), &msg); err != nil {
				// Not JSON, treat as plain text
				if len(line) > maxNonJSONOutputLength {
					line = line[:maxNonJSONOutputLength]
				}
				r.logger.Debug("CCRunner: non-JSON output",
					"user_id", cfg.UserID,
					"mode", cfg.Mode,
					"line", line)
				if callback != nil {
					callback(EventTypeAnswer, line)
				}
				continue
			}

			// Log message type for debugging
			r.logger.Debug("CCRunner: received message",
				"user_id", cfg.UserID,
				"mode", cfg.Mode,
				"type", msg.Type,
				"has_name", msg.Name != "",
				"has_output", msg.Output != "",
				"has_error", msg.Error != "")

			// Dispatch event to callback
			if callback != nil {
				if err := r.dispatchCallback(msg, callback); err != nil {
					errCh <- err
					return
				}
			}

			// Check for completion
			if msg.Type == "result" || msg.Type == "error" {
				return
			}
		}
		errCh <- scanner.Err()
	}()

	// Stream stderr to log
	// 流式处理 stderr 到日志
	wg.Add(1)
	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			r.logger.Warn("CCRunner: stderr from Claude Code CLI",
				"user_id", cfg.UserID,
				"mode", cfg.Mode,
				"line", scanner.Text())
		}
		errCh <- scanner.Err()
	}()

	// Wait for completion or timeout
	// 等待完成或超时
	go func() {
		wg.Wait()
		close(done)
	}()

	timer := time.NewTimer(r.timeout)
	defer timer.Stop()

	select {
	case <-done:
		// Collect any errors
		var errors []string
		for i := 0; i < 2; i++ {
			select {
			case err := <-errCh:
				if err != nil {
					errors = append(errors, err.Error())
				}
			default:
			}
		}
		if len(errors) > 0 {
			return fmt.Errorf("stream errors: %s", errors[0])
		}
		return nil
	case <-ctx.Done():
		timer.Stop()
		return ctx.Err()
	case <-timer.C:
		return fmt.Errorf("execution timeout after %v", r.timeout)
	}
}

// dispatchCallback dispatches stream events to the callback.
// dispatchCallback 将流事件分发给回调。
func (r *CCRunner) dispatchCallback(msg StreamMessage, callback EventCallback) error {
	switch msg.Type {
	case "error":
		if msg.Error != "" {
			return callback(EventTypeError, msg.Error)
		}
	case "thinking", "status":
		for _, block := range msg.GetContentBlocks() {
			if block.Type == "text" && block.Text != "" {
				if err := callback(EventTypeThinking, block.Text); err != nil {
					return err
				}
			}
		}
	case "tool_use":
		if msg.Name != "" {
			if err := callback(EventTypeToolUse, msg.Name); err != nil {
				return err
			}
		}
	case "tool_result":
		if msg.Output != "" {
			if err := callback(EventTypeToolResult, msg.Output); err != nil {
				return err
			}
		}
	case "message", "content", "text", "delta", "assistant":
		for _, block := range msg.GetContentBlocks() {
			if block.Type == "text" && block.Text != "" {
				if err := callback(EventTypeAnswer, block.Text); err != nil {
					return err
				}
			}
		}
	default:
		// Try to extract any text content
		for _, block := range msg.GetContentBlocks() {
			if block.Type == "text" && block.Text != "" {
				callback(EventTypeAnswer, block.Text)
			}
		}
	}
	return nil
}

// GetCLIVersion returns the Claude Code CLI version.
// GetCLIVersion 返回 Claude Code CLI 版本。
func (r *CCRunner) GetCLIVersion() (string, error) {
	cmd := exec.Command(r.cliPath, "--version")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get CLI version: %w", err)
	}
	return string(output), nil
}
