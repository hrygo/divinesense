package runner

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
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

	// DeepSeek V3 pricing (USD per million tokens).
	// DeepSeek V3 定价（每百万 token 美元）。
	// Source: https://api.deepseek.com/
	deepSeekInputCostPerMillion  = 0.27
	deepSeekOutputCostPerMillion = 2.25
)

// UUID v5 namespace for DivineSense session mapping.
// Using a custom v4 namespace ensures uniqueness across projects.
// DivineSense 专用的 UUID v5 命名空间，用于会话映射。
// Generated with: uuid.NewRandom() to avoid conflicts with other projects.
var divineSenseNamespace = uuid.Must(uuid.FromBytes([]byte{
	0xd1, 0x7e, 0xc3, 0x9b, 0x1a, 0x5f, 0x4e, 0x8a,
	0x9b, 0x2c, 0x4d, 0x6e, 0x8f, 0x1a, 0x3b, 0x7c,
}))

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

// EventCallback is the callback function type for agent events.
// EventCallback 是代理事件的回调函数类型。
type EventCallback func(eventType string, eventData any) error

// SafeCallbackFunc is a callback that logs errors instead of returning them.
// Use SafeCallback to wrap an EventCallback for non-critical events.
// SafeCallbackFunc 是一个记录错误而不是返回错误的回调函数。
// 使用 SafeCallback 包装 EventCallback 用于非关键事件。
type SafeCallbackFunc func(eventType string, eventData any)

// SafeCallback wraps an EventCallback to log errors instead of propagating them.
// Use this for non-critical callbacks where errors should not interrupt execution.
// SafeCallback 包装 EventCallback 以记录错误而不是传播它们。
// 用于错误不应中断执行的非关键回调。
func SafeCallback(callback EventCallback) SafeCallbackFunc {
	if callback == nil {
		return nil
	}
	return func(eventType string, eventData any) {
		// Execute callback and log errors instead of returning them
		if err := callback(eventType, eventData); err != nil {
			// Log the callback error but don't propagate it
			// This prevents callback failures from interrupting agent execution
			// Use Background context as this is independent logging with no deadline
			slog.Default().LogAttrs(context.Background(), slog.LevelWarn,
				"callback failed (non-critical)",
				slog.String("event_type", eventType),
				slog.Any("error", err),
			)
		}
	}
}

// CCRunner is the unified Claude Code CLI integration layer.
// CCRunner 是统一的 Claude Code CLI 集成层。
//
// It provides a shared implementation for all modes that need to interact
// with Claude Code CLI (Geek Mode, Evolution Mode, etc.).
// 它为所有需要与 Claude Code CLI 交互的模式提供共享实现（极客模式、进化模式等）。
type CCRunner struct {
	cliPath        string
	timeout        time.Duration
	logger         *slog.Logger
	mu             sync.Mutex
	manager        SessionManager
	dangerDetector *Detector
	// Session stats for the last execution (thread-safe)
	statsMu      sync.RWMutex
	currentStats *SessionStats
}

// NewCCRunner creates a new CCRunner instance.
// NewCCRunner 创建一个新的 CCRunner 实例。
func NewCCRunner(timeout time.Duration, logger *slog.Logger) (*CCRunner, error) {
	cliPath, err := exec.LookPath("claude")
	if err != nil {
		return nil, fmt.Errorf("claude Code CLI not found: %w", err)
	}

	if logger == nil {
		logger = slog.Default()
	}

	// Initialize danger detector for security
	dangerDetector := NewDetector(logger)

	return &CCRunner{
		cliPath:        cliPath,
		timeout:        timeout,
		logger:         logger,
		manager:        NewCCSessionManager(logger, 30*time.Minute), // Default 30m idle timeout
		dangerDetector: dangerDetector,
	}, nil
}

// Execute runs Claude Code CLI with the given configuration and streams events.
// Execute 使用给定配置运行 Claude Code CLI 并流式传输事件。
func (r *CCRunner) Execute(ctx context.Context, cfg *Config, prompt string, callback EventCallback) error {
	// Security check: Detect dangerous operations before execution
	// Skip danger check for Evolution mode (admin only, self-modification)
	if cfg.Mode != "evolution" {
		if dangerEvent := r.dangerDetector.CheckInput(prompt); dangerEvent != nil {
			r.logger.Warn("Dangerous operation blocked",
				"operation", dangerEvent.Operation,
				"reason", dangerEvent.Reason,
				"level", dangerEvent.Level,
			)
			// Send danger block event to client (non-critical - error already being returned)
			callbackSafe := SafeCallback(callback)
			if callbackSafe != nil {
				callbackSafe("danger_block", dangerEvent)
			}
			return fmt.Errorf("dangerous operation blocked: %s", dangerEvent.Reason)
		}
	}

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
	if err := r.ValidateConfig(cfg); err != nil {
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
	firstCall := r.IsFirstCall(sessionDir)

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

	// Initialize session stats for observability
	stats := &SessionStats{
		SessionID: cfg.SessionID,
		StartTime: time.Now(),
	}

	// Send thinking event
	// 发送思考事件
	callbackSafe := SafeCallback(callback)
	if callbackSafe != nil {
		meta := &EventMeta{
			Status:          "running",
			TotalDurationMs: 0,
		}
		callbackSafe("thinking", &EventWithMeta{EventType: "thinking", EventData: fmt.Sprintf("ai.%s_mode.thinking", cfg.Mode), Meta: meta})
	}

	// Execute CLI with session management
	// 执行 CLI 并管理会话
	if err := r.executeWithSession(ctx, cfg, prompt, firstCall, callback, stats); err != nil {
		r.logger.Error("CCRunner: execution failed",
			"user_id", cfg.UserID,
			"mode", cfg.Mode,
			"error", err)
		return err
	}

	// Finalize and save session stats
	// 完成并保存会话统计数据
	// Use CLI-reported duration if available and reasonable (> 1ms to filter out zeros/errors).
	// Otherwise fallback to server-measured duration.
	// 使用 CLI 报告的持续时间（如果合理），否则回退到服务器测量值。
	if stats.TotalDurationMs <= 1 {
		measuredDuration := time.Since(stats.StartTime).Milliseconds()
		if measuredDuration > stats.TotalDurationMs {
			stats.TotalDurationMs = measuredDuration
		}
	}
	r.statsMu.Lock()
	r.currentStats = stats
	r.statsMu.Unlock()

	r.logger.Info("CCRunner: Session completed",
		"session_id", stats.SessionID,
		"total_duration_ms", stats.TotalDurationMs,
		"tool_duration_ms", stats.ToolDurationMs,
		"tool_calls", stats.ToolCallCount,
		"tools_used", len(stats.ToolsUsed))

	return nil
}

// StartAsyncSession starts a persistent session and returns the session object.
func (r *CCRunner) StartAsyncSession(ctx context.Context, cfg *Config) (*Session, error) {
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

	if err := r.ValidateConfig(cfg); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	// Ensure working directory exists
	if err := os.MkdirAll(cfg.WorkDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create work directory: %w", err)
	}

	// Create session via manager
	return r.manager.GetOrCreateSession(ctx, cfg.SessionID, *cfg)
}

// GetSessionManager returns the session manager.
func (r *CCRunner) GetSessionManager() SessionManager {
	return r.manager
}

// GetSessionStats returns a copy of the current session stats.
// GetSessionStats 返回当前会话统计数据的副本。
func (r *CCRunner) GetSessionStats() *SessionStats {
	r.statsMu.Lock()
	defer r.statsMu.Unlock()

	if r.currentStats == nil {
		return nil
	}

	// Finalize any ongoing phases before copying
	// 完成任何正在进行的阶段，然后再复制
	return r.currentStats.FinalizeDuration()
}

// ValidateConfig validates the Config.
// ValidateConfig 验证 Config。
func (r *CCRunner) ValidateConfig(cfg *Config) error {
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

// IsFirstCall checks if this is the first call for a session.
// IsFirstCall 检查是否是会话的首次调用。
func (r *CCRunner) IsFirstCall(sessionDir string) bool {
	_, err := os.Stat(sessionDir)
	return os.IsNotExist(err)
}

// executeWithSession executes Claude Code CLI with appropriate session flags.
// executeWithSession 使用适当的会话标志执行 Claude Code CLI。
func (r *CCRunner) executeWithSession(
	ctx context.Context,
	cfg *Config,
	prompt string,
	firstCall bool,
	callback EventCallback,
	stats *SessionStats,
) error {
	// Build system prompt
	// 构建系统提示词
	systemPrompt := cfg.SystemPrompt
	if systemPrompt == "" {
		systemPrompt = BuildSystemPrompt(cfg.WorkDir, cfg.SessionID, cfg.UserID, cfg.DeviceContext)
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
	// Note: We do NOT set CLAUDE_CONFIG_DIR here, so CLI uses the main
	// config which already has authentication credentials.
	// 注意：这里不设置 CLAUDE_CONFIG_DIR，让 CLI 使用已认证的主配置
	cmd.Env = append(os.Environ(),
		"CLAUDE_DISABLE_TELEMETRY=1",
	)

	// Get pipes
	// 获取管道
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("stdout pipe: %w", err)
	}
	defer stdout.Close() //nolint:errcheck // cleanup on error path

	stderr, err := cmd.StderrPipe()
	if err != nil {
		// Close stdout immediately since we won't reach the normal defer
		_ = stdout.Close() //nolint:errcheck // cleanup on error path
		return fmt.Errorf("stderr pipe: %w", err)
	}
	defer stderr.Close() //nolint:errcheck // cleanup on error path

	// Start command
	// 启动命令
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start command: %w", err)
	}

	// Create stderr buffer to capture output for error context
	// 创建 stderr 缓冲区以捕获输出用于错误上下文
	stderrBuf := newStderrBuffer(100)

	// Stream output with timeout
	// 带超时流式输出
	if err := r.streamOutput(ctx, cfg, stdout, stderr, callback, stats, stderrBuf); err != nil {
		r.logger.Error("CCRunner: streamOutput failed", "mode", cfg.Mode, "session_id", cfg.SessionID, "error", err)
		if cmd.Process != nil {
			_ = cmd.Process.Kill() //nolint:errcheck // process already terminating
		}
		// Include stderr context in error
		if stderrLines := stderrBuf.getLastN(10); len(stderrLines) > 0 {
			return fmt.Errorf("stream failed: %w (last %d stderr lines: %s)", err, len(stderrLines), joinStrings(stderrLines, "; "))
		}
		return err
	}

	// Wait for command completion
	// 等待命令完成
	waitErr := cmd.Wait()
	if waitErr != nil {
		r.logger.Error("CCRunner: CLI process exited with error",
			"mode", cfg.Mode,
			"session_id", cfg.SessionID,
			"error", waitErr)

		// Get exit code if available
		exitCode := 0
		if cmd.ProcessState != nil {
			exitCode = cmd.ProcessState.ExitCode()
		}

		// Include stderr context in error if available
		if stderrLines := stderrBuf.getLastN(10); len(stderrLines) > 0 {
			return fmt.Errorf("command exited with code %d: %w (stderr: %s)",
				exitCode, waitErr, joinStrings(stderrLines, "; "))
		}
		return fmt.Errorf("command exited with code %d: %w", exitCode, waitErr)
	}

	return nil
}

// streamOutput reads and parses stream-json output from CLI.
// streamOutput 读取并解析 CLI 的 stream-json 输出。
func (r *CCRunner) streamOutput(
	ctx context.Context,
	cfg *Config,
	stdout, stderr io.ReadCloser,
	callback EventCallback,
	stats *SessionStats,
	stderrBuf *stderrBuffer,
) error {
	var wg sync.WaitGroup
	errCh := make(chan error, 2)
	done := make(chan struct{})
	// Create a cancel context to signal goroutines to stop
	// Derive from parent ctx so cancellation propagates (fixes goroutine leak)
	streamCtx, stopStreams := context.WithCancel(ctx)
	defer stopStreams()

	// Create safe callback once for all goroutines to reuse
	// This avoids redundant wrapping in each goroutine
	callbackSafe := SafeCallback(callback)

	// Stream stdout
	// 流式处理 stdout
	wg.Add(1)
	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(stdout)
		buf := make([]byte, 0, scannerInitialBufSize)
		scanner.Buffer(buf, scannerMaxBufSize)

		scanDone := make(chan bool)
		go func() {
			lineCount := 0
			lastValidDataTime := time.Now() // Track last time we received valid data

			// Add panic recovery to ensure scanDone is always closed even on panic
			// 添加 panic recovery 以确保即使 panic 也关闭 scanDone
			defer func() {
				if panicVal := recover(); panicVal != nil {
					r.logger.Error("CCRunner: scanner goroutine panic recovered",
						"mode", cfg.Mode,
						"session_id", cfg.SessionID,
						"panic", panicVal)
					scanDone <- true // Signal completion even on panic
				} else {
					close(scanDone) // Normal exit: close channel for proper cleanup
				}
			}()

			for scanner.Scan() {
				lineCount++

				// Check for inactivity before processing the line
				// This detects when CLI stops sending data while scanner is blocked
				if time.Since(lastValidDataTime) > 60*time.Second {
					r.logger.Warn("CCRunner: no valid data from CLI for 60+ seconds",
						"mode", cfg.Mode,
						"session_id", cfg.SessionID,
						"last_line_count", lineCount)
					// Reset timer to avoid spamming
					lastValidDataTime = time.Now()
				}

				line := scanner.Text()
				if line == "" {
					continue
				}

				// Update last activity time when we receive non-empty line
				lastValidDataTime = time.Now()

				var msg StreamMessage
				if err := json.Unmarshal([]byte(line), &msg); err != nil {
					// Not JSON, treat as plain text
					if len(line) > maxNonJSONOutputLength {
						line = line[:maxNonJSONOutputLength]
					}
					r.logger.Debug("CCRunner: non-JSON output",
						"mode", cfg.Mode,
						"line", line)
					if callbackSafe != nil {
						callbackSafe("answer", line)
					}
					continue
				}

				// Handle result message - extract and send session statistics
				if msg.Type == "result" {
					r.handleResultMessage(msg, stats, cfg, callback)
					break // break loop instead of return - let scanDone be sent
				}

				// Handle system message - silently consume
				if msg.Type == "system" {
					r.logger.Debug("CCRunner: system message received (control data, no callback needed)",
						"subtype", msg.Subtype,
						"session_id", msg.SessionID)
					continue
				}

				// Dispatch event to callback
				if callback != nil {
					if err := r.dispatchCallback(msg, callback, stats); err != nil {
						select {
						case errCh <- err:
						case <-streamCtx.Done():
						}
						break // break loop on error
					}
				}

				// Check for error completion
				if msg.Type == "error" {
					break // break loop instead of return - let scanDone be sent
				}
			}
			scanDone <- true
		}()

		// Wait for scan to complete or context to be cancelled
		select {
		case <-scanDone:
			if scanErr := scanner.Err(); scanErr != nil {
				r.logger.Error("CCRunner: scanner error",
					"mode", cfg.Mode,
					"session_id", cfg.SessionID,
					"error", scanErr)
				select {
				case errCh <- scanErr:
				case <-streamCtx.Done():
				}
			}
			// stdout scan completed - signal stderr goroutine to stop
			// This prevents deadlock where stderr goroutine waits forever
			stopStreams()
		case <-streamCtx.Done():
			// Force close pipes to interrupt any blocked scanner
			// This prevents goroutine leak when scanner is blocked on I/O
			_ = stdout.Close() //nolint:errcheck // force close to unblock scanner
			_ = stderr.Close() //nolint:errcheck // force close to unblock scanner
			// Wait for scanner to exit (with timeout to prevent indefinite blocking)
			select {
			case <-scanDone:
			case <-time.After(1 * time.Second):
				r.logger.Warn("CCRunner: scanner did not exit after pipe close",
					"mode", cfg.Mode,
					"session_id", cfg.SessionID)
			}
		}
	}()

	// Stream stderr with sampling for logs and capture last N lines for error context.
	// 对 stderr 进行采样以防止日志泛滥，同时保留调试信息。
	wg.Add(1)
	go func() {
		defer wg.Done()

		// Sample stderr output (10% rate) for logs, capture all for error context.
		// 对 stderr 进行采样记录到日志，同时捕获所有内容用于错误上下文。
		scanner := bufio.NewScanner(stderr)
		sampleRate := 10 // Sample 10% of stderr lines
		for scanner.Scan() {
			line := scanner.Text()
			stderrBuf.addLine(line)

			//nolint:gosec // Sampling for logging, not security-critical
			if rand.Intn(100) < sampleRate {
				r.logger.Warn("CCRunner: stderr sample",
					"user_id", cfg.UserID,
					"mode", cfg.Mode,
					"session_id", cfg.SessionID,
					"line", line)
			}
		}
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
		stopStreams() // Signal goroutines to stop
		// Drain errCh to prevent goroutines from blocking
		for i := 0; i < 2; i++ {
			select {
			case <-errCh:
			default:
			}
		}
		return ctx.Err()
	case <-timer.C:
		stopStreams() // Signal goroutines to stop
		// Drain errCh to prevent goroutines from blocking
		for i := 0; i < 2; i++ {
			select {
			case <-errCh:
			default:
			}
		}
		return fmt.Errorf("execution timeout after %v", r.timeout)
	}
}

// handleResultMessage processes the result message from CLI, extracts statistics,
// and sends session_stats event to frontend.
// handleResultMessage 处理 CLI 的 result 消息，提取统计数据，并发送 session_stats 事件到前端。
func (r *CCRunner) handleResultMessage(msg StreamMessage, stats *SessionStats, cfg *Config, callback EventCallback) {
	stats.mu.Lock()
	defer stats.mu.Unlock()

	// Update final duration from CLI report
	if msg.Duration > 0 {
		stats.TotalDurationMs = int64(msg.Duration)
	}

	// Update token usage from CLI report
	if msg.Usage != nil {
		stats.InputTokens = msg.Usage.InputTokens
		stats.OutputTokens = msg.Usage.OutputTokens
		stats.CacheWriteTokens = msg.Usage.CacheWriteInputTokens
		stats.CacheReadTokens = msg.Usage.CacheReadInputTokens
	}

	// Collect tools used (convert map to slice)
	toolsUsed := make([]string, 0, len(stats.ToolsUsed))
	for tool := range stats.ToolsUsed {
		toolsUsed = append(toolsUsed, tool)
	}

	// Collect file paths (with deduplication)
	filePathsSet := make(map[string]bool, len(stats.FilePaths))
	for _, path := range stats.FilePaths {
		if path != "" {
			filePathsSet[path] = true
		}
	}
	filePaths := make([]string, 0, len(filePathsSet))
	for path := range filePathsSet {
		filePaths = append(filePaths, path)
	}

	// Calculate total cost with fallback if CLI doesn't report it
	totalCostUSD := msg.TotalCostUSD
	if totalCostUSD == 0 && stats.InputTokens+stats.OutputTokens > 0 {
		// Use DeepSeek V3 pricing (defined as package-level constants)
		inputCost := float64(stats.InputTokens) * deepSeekInputCostPerMillion / 1_000_000
		outputCost := float64(stats.OutputTokens) * deepSeekOutputCostPerMillion / 1_000_000
		totalCostUSD = inputCost + outputCost
	}

	// Log session completion stats
	r.logger.Info("CCRunner: session completed",
		"mode", cfg.Mode,
		"session_id", cfg.SessionID,
		"duration_ms", stats.TotalDurationMs,
		"input_tokens", stats.InputTokens,
		"output_tokens", stats.OutputTokens,
		"total_cost_usd", msg.TotalCostUSD,
		"tool_calls", stats.ToolCallCount,
		"files_modified", stats.FilesModified)

	// Send session_stats event to frontend (non-critical)
	if callback != nil {
		callbackSafe := SafeCallback(callback)
		callbackSafe("session_stats", &SessionStatsData{
			SessionID:            cfg.SessionID,
			ConversationID:       cfg.ConversationID,
			UserID:               cfg.UserID,
			AgentType:            cfg.Mode,
			StartTime:            stats.StartTime.Unix(),
			EndTime:              time.Now().Unix(),
			TotalDurationMs:      stats.TotalDurationMs,
			ThinkingDurationMs:   stats.ThinkingDurationMs,
			ToolDurationMs:       stats.ToolDurationMs,
			GenerationDurationMs: stats.GenerationDurationMs,
			InputTokens:          stats.InputTokens,
			OutputTokens:         stats.OutputTokens,
			CacheWriteTokens:     stats.CacheWriteTokens,
			CacheReadTokens:      stats.CacheReadTokens,
			TotalTokens:          stats.InputTokens + stats.OutputTokens,
			ToolCallCount:        stats.ToolCallCount,
			ToolsUsed:            toolsUsed,
			FilesModified:        stats.FilesModified,
			FilePaths:            filePaths,
			ModelUsed:            "claude-code",
			TotalCostUSD:         totalCostUSD,
			IsError:              msg.IsError,
			ErrorMessage:         msg.Error,
		})
	}
}

// dispatchCallback dispatches stream events to the callback with metadata.
// IMPORTANT: This function is called from stream goroutines. The callback MUST:
// 1. Return quickly (< 5 seconds) to avoid blocking stream processing
// 2. NOT call back into Session/CCRunner methods (risk of deadlock)
// 3. Be safe for concurrent invocation from multiple goroutines
// dispatchCallback 将流事件分发给回调，附带元数据。
func (r *CCRunner) dispatchCallback(msg StreamMessage, callback EventCallback, stats *SessionStats) error {
	// Calculate total duration
	totalDuration := time.Since(stats.StartTime).Milliseconds()

	switch msg.Type {
	case "error":
		if msg.Error != "" {
			return callback("error", msg.Error)
		}
	case "thinking", "status":
		// Start thinking phase tracking (ended in other cases or by defer)
		stats.StartThinking()
		// Ensure thinking is ended even if we return early from this case
		// Note: if control flows to another case (tool_use, assistant), they will end thinking explicitly
		defer func() {
			stats.EndThinking()
		}()

		for _, block := range msg.GetContentBlocks() {
			if block.Type == "text" && block.Text != "" {
				meta := &EventMeta{
					Status:          "running",
					TotalDurationMs: totalDuration,
				}
				if err := callback("thinking", &EventWithMeta{EventType: "thinking", EventData: block.Text, Meta: meta}); err != nil {
					return err
				}
			}
		}
	case "tool_use":
		// Tool use ends thinking, starts tool execution
		stats.EndThinking()

		if msg.Name != "" {
			// Extract tool ID and input from content blocks
			var toolID string
			var inputSummary string
			var filePath string
			for _, block := range msg.GetContentBlocks() {
				if block.Type == "tool_use" {
					toolID = block.ID
					if block.Input != nil {
						// Create a human-readable summary of the input
						inputSummary = SummarizeInput(block.Input)

						// Extract file path for Write/Edit operations
						if msg.Name == "Write" || msg.Name == "Edit" || msg.Name == "WriteFile" || msg.Name == "EditFile" {
							if path, ok := block.Input["path"].(string); ok {
								filePath = path
							}
						}
					}
				}
			}
			stats.RecordToolUse(msg.Name, toolID)

			// Record file modification for Write/Edit tools
			if filePath != "" {
				stats.RecordFileModification(filePath)
			}

			meta := &EventMeta{
				ToolName:        msg.Name,
				ToolID:          toolID,
				Status:          "running",
				TotalDurationMs: totalDuration,
				InputSummary:    inputSummary,
			}
			r.logger.Debug("CCRunner: sending tool_use event", "tool_name", msg.Name, "tool_id", toolID)
			if err := callback("tool_use", &EventWithMeta{EventType: "tool_use", EventData: msg.Name, Meta: meta}); err != nil {
				return err
			}
		}
	case "tool_result":
		if msg.Output != "" {
			durationMs := stats.RecordToolResult()

			meta := &EventMeta{
				Status:          "success",
				DurationMs:      durationMs,
				TotalDurationMs: totalDuration,
				OutputSummary:   TruncateString(msg.Output, 500),
			}
			r.logger.Debug("CCRunner: sending tool_result event", "output_length", len(msg.Output), "duration_ms", durationMs)
			if err := callback("tool_result", &EventWithMeta{EventType: "tool_result", EventData: msg.Output, Meta: meta}); err != nil {
				return err
			}
		}
	case "message", "content", "text", "delta", "assistant":
		// Assistant message starts generation phase
		stats.EndThinking()
		stats.StartGeneration()

		for _, block := range msg.GetContentBlocks() {
			if block.Type == "text" && block.Text != "" {
				if err := callback("answer", &EventWithMeta{EventType: "answer", EventData: block.Text, Meta: &EventMeta{TotalDurationMs: totalDuration}}); err != nil {
					return err
				}
			} else if block.Type == "tool_use" && block.Name != "" {
				// Tool use is nested inside assistant message content
				// End generation when tool is about to be used
				stats.EndGeneration()

				stats.RecordToolUse(block.Name, block.ID)

				// Record file modification for Write/Edit tools
				if block.Name == "Write" || block.Name == "Edit" || block.Name == "WriteFile" || block.Name == "EditFile" {
					if block.Input != nil {
						if path, ok := block.Input["path"].(string); ok {
							stats.RecordFileModification(path)
						}
					}
				}

				meta := &EventMeta{
					ToolName:        block.Name,
					ToolID:          block.ID,
					Status:          "running",
					TotalDurationMs: totalDuration,
					InputSummary:    SummarizeInput(block.Input),
				}
				if err := callback("tool_use", &EventWithMeta{EventType: "tool_use", EventData: block.Name, Meta: meta}); err != nil {
					return err
				}
			}
		}
	case "user":
		// Tool results come as type:"user" with nested tool_result blocks
		for _, block := range msg.GetContentBlocks() {
			if block.Type == "tool_result" {
				durationMs := stats.RecordToolResult()

				meta := &EventMeta{
					Status:          "success",
					DurationMs:      durationMs,
					TotalDurationMs: totalDuration,
					OutputSummary:   TruncateString(block.Content, 500),
				}
				if err := callback("tool_result", &EventWithMeta{EventType: "tool_result", EventData: block.Content, Meta: meta}); err != nil {
					return err
				}
			}
		}
	default:
		// Log unknown message type for debugging
		r.logger.Warn("CCRunner: unknown message type",
			"type", msg.Type,
			"role", msg.Role,
			"name", msg.Name,
			"has_content", len(msg.Content) > 0,
			"has_message", msg.Message != nil,
			"has_error", msg.Error != "",
			"has_output", msg.Output != "")

		// Try to extract any text content (non-critical - use safe callback)
		callbackSafe := SafeCallback(callback)
		for _, block := range msg.GetContentBlocks() {
			if block.Type == "text" && block.Text != "" {
				if callbackSafe != nil {
					callbackSafe("answer", &EventWithMeta{EventType: "answer", EventData: block.Text, Meta: &EventMeta{TotalDurationMs: totalDuration}})
				}
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

// StopSession terminates a running session by session ID.
// This is the implementation for session.stop from the spec.
// StopSession 通过 session ID 终止正在运行的会话。
// 这是规范中 session.stop 的实现。
func (r *CCRunner) StopSession(sessionID string, reason string) error {
	r.logger.Info("CCRunner: stopping session",
		"session_id", sessionID,
		"reason", reason)

	return r.manager.TerminateSession(sessionID)
}

// StopSessionByConversationID terminates a session by its conversation ID.
// StopSessionByConversationID 通过对话 ID 终止会话。
func (r *CCRunner) StopSessionByConversationID(conversationID int64, reason string) error {
	sessionID := ConversationIDToSessionID(conversationID)
	return r.StopSession(sessionID, reason)
}

// SetDangerAllowPaths sets the allowed safe paths for the danger detector.
// SetDangerAllowPaths 设置危险检测器的允许安全路径。
func (r *CCRunner) SetDangerAllowPaths(paths []string) {
	r.dangerDetector.SetAllowPaths(paths)
}

// SetDangerBypassEnabled enables or disables danger detection bypass.
// WARNING: Only use for Evolution mode (admin only).
// SetDangerBypassEnabled 启用或禁用危险检测绕过。
// 警告：仅用于进化模式（仅管理员）。
func (r *CCRunner) SetDangerBypassEnabled(enabled bool) {
	r.dangerDetector.SetBypassEnabled(enabled)
}

// GetDangerDetector returns the danger detector instance.
// GetDangerDetector 返回危险检测器实例。
func (r *CCRunner) GetDangerDetector() *Detector {
	return r.dangerDetector
}

// BuildSystemPrompt provides minimal, high-signal context for Claude Code CLI.
// BuildSystemPrompt 为 Claude Code CLI 提供最小化、高信噪比的上下文。
func BuildSystemPrompt(workDir, sessionID string, userID int32, deviceContext string) string {
	return BuildSystemPromptWithRuntime(workDir, sessionID, userID, deviceContext, getRuntimeInfo())
}

// BuildSystemPromptWithRuntime is the implementation that allows runtime info injection.
func BuildSystemPromptWithRuntime(workDir, sessionID string, userID int32, deviceContext string, runtimeInfo RuntimeInfo) string {
	osName := runtimeInfo.OS
	arch := runtimeInfo.Arch
	if osName == "darwin" {
		osName = "macOS"
	}

	timestamp := runtimeInfo.Timestamp.Format("2006-01-02 15:04:05")

	// Try to parse device context for better formatting
	// 尝试解析设备上下文以便更好地格式化
	var contextMap map[string]any
	userAgent := "Unknown"
	deviceInfo := "Unknown"
	if deviceContext != "" {
		// Optimization: only attempt JSON parse if it looks like JSON
		// 优化：只在看起来像 JSON 时才尝试解析
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
				// Fallback: use raw string if JSON parse failed
				userAgent = deviceContext
			}
		} else {
			// Not JSON - use raw string
			userAgent = deviceContext
		}
	}

	return fmt.Sprintf(`# Context

You are running inside DivineSense, an intelligent assistant system.

**User Interaction**: Users type questions in their web browser, which invokes you via a Go backend. Your response streams back to their browser in real-time. **Always respond in Chinese (Simplified).**

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

// RuntimeInfo contains runtime information for system prompt generation.
type RuntimeInfo struct {
	OS        string
	Arch      string
	Timestamp time.Time
}

// getRuntimeInfo returns the current runtime information.
func getRuntimeInfo() RuntimeInfo {
	return RuntimeInfo{
		OS:        runtime.GOOS,
		Arch:      runtime.GOARCH,
		Timestamp: time.Now(),
	}
}

// joinStrings joins a slice of strings with a separator.
func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	var sb strings.Builder
	for i, s := range strs {
		if i > 0 {
			sb.WriteString(sep)
		}
		sb.WriteString(s)
	}
	return sb.String()
}
