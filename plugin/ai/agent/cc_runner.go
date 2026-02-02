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
		// Optimization: only attempt JSON parse if it looks like JSON
		// 优化：只在看起来像 JSON 时才尝试解析
		if strings.HasPrefix(strings.TrimSpace(deviceContext), "{") {
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
	cliPath        string
	timeout        time.Duration
	logger         *slog.Logger
	mu             sync.Mutex
	manager        *CCSessionManager
	dangerDetector *DangerDetector
	// Session stats for the last execution (thread-safe)
	statsMu      sync.RWMutex
	currentStats *SessionStats
}

// EventWithMeta extends the basic event with metadata for observability.
// EventWithMeta 扩展基本事件，添加元数据以增强可观测性。
type EventWithMeta struct {
	EventType string     // Event type (thinking, tool_use, tool_result, etc.)
	EventData string     // Event data content
	Meta      *EventMeta // Enhanced metadata
}

// EventMeta contains detailed metadata for streaming events.
// EventMeta 包含流式事件的详细元数据。
type EventMeta struct {
	// Timing
	DurationMs      int64 // Event duration in milliseconds
	TotalDurationMs int64 // Total elapsed time since start

	// Tool call info
	ToolName string // Tool name (e.g., "bash", "editor_write")
	ToolID   string // Unique tool call ID
	Status   string // "running", "success", "error"
	ErrorMsg string // Error message if status=error

	// Token usage (when available)
	InputTokens      int32 // Input tokens
	OutputTokens     int32 // Output tokens
	CacheWriteTokens int32 // Cache write tokens
	CacheReadTokens  int32 // Cache read tokens

	// Summaries for UI
	InputSummary  string // Human-readable input summary
	OutputSummary string // Truncated output preview

	// File operations
	FilePath  string // Affected file path
	LineCount int32  // Number of lines affected
}

// SessionStats collects session-level statistics for Geek/Evolution modes.
// SessionStats 收集极客/进化模式的会话级别统计数据。
type SessionStats struct {
	mu                   sync.Mutex
	SessionID            string
	StartTime            time.Time
	TotalDurationMs      int64
	ThinkingDurationMs   int64
	ToolDurationMs       int64
	GenerationDurationMs int64
	InputTokens          int32
	OutputTokens         int32
	CacheWriteTokens     int32
	CacheReadTokens      int32
	ToolCallCount        int32
	ToolsUsed            map[string]bool
	FilesModified        int32
	FilePaths            []string

	// Current tool tracking
	currentToolStart time.Time
	currentToolName  string
	currentToolID    string

	// Phase tracking for duration breakdown
	thinkingStart   time.Time
	generationStart time.Time
	hasGeneration   bool // Tracks if any content was generated
}

// RecordToolUse records the start of a tool call.
func (s *SessionStats) RecordToolUse(toolName, toolID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.currentToolStart = time.Now()
	s.currentToolName = toolName
	s.currentToolID = toolID
	// Ensure ToolsUsed map is initialized (concurrency safety)
	if s.ToolsUsed == nil {
		s.ToolsUsed = make(map[string]bool)
	}
}

// RecordToolResult records the end of a tool call.
func (s *SessionStats) RecordToolResult() (durationMs int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.currentToolStart.IsZero() {
		duration := time.Since(s.currentToolStart)
		durationMs = duration.Milliseconds()
		s.ToolDurationMs += durationMs
		s.ToolCallCount++
		if s.currentToolName != "" {
			if s.ToolsUsed == nil {
				s.ToolsUsed = make(map[string]bool)
			}
			s.ToolsUsed[s.currentToolName] = true
		}
		s.currentToolStart = time.Time{}
		s.currentToolName = ""
		s.currentToolID = ""
	}
	return
}

// RecordTokens records token usage.
func (s *SessionStats) RecordTokens(input, output, cacheWrite, cacheRead int32) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.InputTokens += input
	s.OutputTokens += output
	s.CacheWriteTokens += cacheWrite
	s.CacheReadTokens += cacheRead
}

// StartThinking marks the start of the thinking phase.
// StartThinking 标记思考阶段的开始。
func (s *SessionStats) StartThinking() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.thinkingStart.IsZero() {
		s.thinkingStart = time.Now()
	}
}

// EndThinking marks the end of the thinking phase and records its duration.
// EndThinking 标记思考阶段的结束并记录其持续时间。
func (s *SessionStats) EndThinking() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.thinkingStart.IsZero() {
		s.ThinkingDurationMs += time.Since(s.thinkingStart).Milliseconds()
		s.thinkingStart = time.Time{} // Reset for next thinking phase
	}
}

// StartGeneration marks the start of the generation phase.
// StartGeneration 标记生成阶段的开始。
func (s *SessionStats) StartGeneration() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.generationStart.IsZero() {
		s.generationStart = time.Now()
		s.hasGeneration = true
	}
}

// EndGeneration marks the end of the generation phase and records its duration.
// EndGeneration 标记生成阶段的结束并记录其持续时间。
func (s *SessionStats) EndGeneration() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.generationStart.IsZero() {
		s.GenerationDurationMs += time.Since(s.generationStart).Milliseconds()
		s.generationStart = time.Time{} // Reset for next generation phase
	}
}

// ToSummary converts stats to a summary map for JSON serialization.
func (s *SessionStats) ToSummary() map[string]interface{} {
	s.mu.Lock()
	defer s.mu.Unlock()

	tools := make([]string, 0, len(s.ToolsUsed))
	for tool := range s.ToolsUsed {
		tools = append(tools, tool)
	}

	return map[string]interface{}{
		"session_id":               s.SessionID,
		"total_duration_ms":        s.TotalDurationMs,
		"thinking_duration_ms":     s.ThinkingDurationMs,
		"tool_duration_ms":         s.ToolDurationMs,
		"generation_duration_ms":   s.GenerationDurationMs,
		"total_input_tokens":       s.InputTokens,
		"total_output_tokens":      s.OutputTokens,
		"total_cache_write_tokens": s.CacheWriteTokens,
		"total_cache_read_tokens":  s.CacheReadTokens,
		"tool_call_count":          s.ToolCallCount,
		"tools_used":               tools,
		"files_modified":           s.FilesModified,
		"file_paths":               s.FilePaths,
		"status":                   "success",
	}
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

	// Initialize danger detector for security
	dangerDetector := NewDangerDetector(logger)

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
func (r *CCRunner) Execute(ctx context.Context, cfg *CCRunnerConfig, prompt string, callback EventCallback) error {
	// Security check: Detect dangerous operations before execution
	// Skip danger check for Evolution mode (admin only, self-modification)
	if cfg.Mode != "evolution" {
		if dangerEvent := r.dangerDetector.CheckInput(prompt); dangerEvent != nil {
			r.logger.Warn("Dangerous operation blocked",
				"operation", dangerEvent.Operation,
				"reason", dangerEvent.Reason,
				"level", dangerEvent.Level,
			)
			// Send danger block event to client
			if callback != nil {
				_ = callback(EventTypeDangerBlock, dangerEvent)
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

	// Initialize session stats for observability
	stats := &SessionStats{
		SessionID: cfg.SessionID,
		StartTime: time.Now(),
	}

	// Send thinking event
	// 发送思考事件
	if callback != nil {
		meta := &EventMeta{
			Status:          "running",
			TotalDurationMs: 0,
		}
		if err := callback(EventTypeThinking, &EventWithMeta{EventType: EventTypeThinking, EventData: fmt.Sprintf("ai.%s_mode.thinking", cfg.Mode), Meta: meta}); err != nil {
			return err
		}
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
	stats.TotalDurationMs = time.Since(stats.StartTime).Milliseconds()
	r.statsMu.Lock()
	r.currentStats = stats
	r.statsMu.Unlock()

	r.logger.Debug("CCRunner: Session completed",
		"session_id", stats.SessionID,
		"total_duration_ms", stats.TotalDurationMs,
		"tool_duration_ms", stats.ToolDurationMs,
		"tool_calls", stats.ToolCallCount,
		"tools_used", len(stats.ToolsUsed))

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
	stats := r.currentStats

	// Close any open phase tracking (calculate remaining duration)
	// 关闭任何打开的阶段追踪（计算剩余时长）
	totalThinking := stats.ThinkingDurationMs
	totalGeneration := stats.GenerationDurationMs

	if !stats.thinkingStart.IsZero() {
		totalThinking += time.Since(stats.thinkingStart).Milliseconds()
	}
	if !stats.generationStart.IsZero() {
		totalGeneration += time.Since(stats.generationStart).Milliseconds()
	}

	// Return a copy with finalized durations
	// 返回包含已完成时长的副本
	return &SessionStats{
		SessionID:            stats.SessionID,
		StartTime:            stats.StartTime,
		TotalDurationMs:      stats.TotalDurationMs,
		ThinkingDurationMs:   totalThinking,
		ToolDurationMs:       stats.ToolDurationMs,
		GenerationDurationMs: totalGeneration,
		InputTokens:          stats.InputTokens,
		OutputTokens:         stats.OutputTokens,
		CacheWriteTokens:     stats.CacheWriteTokens,
		CacheReadTokens:      stats.CacheReadTokens,
		ToolCallCount:        stats.ToolCallCount,
		ToolsUsed:            stats.ToolsUsed,
		FilesModified:        stats.FilesModified,
		FilePaths:            stats.FilePaths,
	}
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
	stats *SessionStats,
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
	if err := r.streamOutput(ctx, cfg, stdout, stderr, callback, stats); err != nil {
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
	stats *SessionStats,
) error {
	var wg sync.WaitGroup
	errCh := make(chan error, 2)
	done := make(chan struct{})
	// Create a cancel context to signal goroutines to stop
	streamCtx, stopStreams := context.WithCancel(context.Background())
	defer stopStreams()

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
			r.logger.Info("CCRunner: scanner loop started",
				"mode", cfg.Mode,
				"session_id", cfg.SessionID)

			lineCount := 0
			lastValidDataTime := time.Now() // Track last time we received valid data

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

				// Log raw line for debugging (truncate if too long)
				logLine := line
				if len(logLine) > 200 {
					logLine = logLine[:200] + "..."
				}
				// Use Info level for visibility in production
				r.logger.Info("CCRunner: raw line",
					"mode", cfg.Mode,
					"line_number", lineCount,
					"line", logLine)

				var msg StreamMessage
				if err := json.Unmarshal([]byte(line), &msg); err != nil {
					// Not JSON, treat as plain text
					if len(line) > maxNonJSONOutputLength {
						line = line[:maxNonJSONOutputLength]
					}
					r.logger.Debug("CCRunner: non-JSON output",
						"mode", cfg.Mode,
						"line", line)
					if callback != nil {
						callback(EventTypeAnswer, line)
					}
					continue
				}

				// Log message type for debugging (Info level for production visibility)
				r.logger.Info("CCRunner: received message",
					"mode", cfg.Mode,
					"line_number", lineCount,
					"type", msg.Type,
					"name", msg.Name,
					"has_output", msg.Output != "",
					"has_error", msg.Error != "")

				// Dispatch event to callback
				if callback != nil {
					if err := r.dispatchCallback(msg, callback, stats); err != nil {
						select {
						case errCh <- err:
						case <-streamCtx.Done():
						}
						return
					}
				}

				// Check for completion
				if msg.Type == "result" || msg.Type == "error" {
					r.logger.Info("CCRunner: completion message received, ending scanner loop",
						"mode", cfg.Mode,
						"type", msg.Type,
						"total_lines", lineCount)
					return
				}
			}
			scanDone <- true
			r.logger.Info("CCRunner: scanner loop ended",
				"mode", cfg.Mode,
				"total_lines", lineCount)
		}()

		// Wait for scan to complete or context to be cancelled
		select {
		case <-scanDone:
			if scanErr := scanner.Err(); scanErr != nil {
				select {
				case errCh <- scanErr:
				case <-streamCtx.Done():
				}
			}
		case <-streamCtx.Done():
			// Scanner will be interrupted when stdout is closed externally
		}
	}()

	// Stream stderr to log
	// 流式处理 stderr 到日志
	wg.Add(1)
	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(stderr)

		scanDone := make(chan bool)
		go func() {
			for scanner.Scan() {
				r.logger.Warn("CCRunner: stderr from Claude Code CLI",
					"user_id", cfg.UserID,
					"mode", cfg.Mode,
					"line", scanner.Text())
			}
			scanDone <- true
		}()

		select {
		case <-scanDone:
			if scanErr := scanner.Err(); scanErr != nil {
				select {
				case errCh <- scanErr:
				case <-streamCtx.Done():
				}
			}
		case <-streamCtx.Done():
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

// dispatchCallback dispatches stream events to the callback with metadata.
// IMPORTANT: This function is called from stream goroutines. The callback MUST:
// 1. Return quickly (< 5 seconds) to avoid blocking stream processing
// 2. NOT call back into Session/CCRunner methods (risk of deadlock)
// 3. Be safe for concurrent invocation from multiple goroutines
// dispatchCallback 将流事件分发给回调，附带元数据。
// 重要：此函数从 stream goroutine 中调用。回调必须：
// 1. 快速返回（< 5 秒）以避免阻塞流处理
// 2. 不回调 Session/CCRunner 方法（死锁风险）
// 3. 支持多 goroutine 并发调用
func (r *CCRunner) dispatchCallback(msg StreamMessage, callback EventCallback, stats *SessionStats) error {
	// Calculate total duration
	totalDuration := time.Since(stats.StartTime).Milliseconds()

	switch msg.Type {
	case "error":
		if msg.Error != "" {
			return callback(EventTypeError, msg.Error)
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
				if err := callback(EventTypeThinking, &EventWithMeta{EventType: EventTypeThinking, EventData: block.Text, Meta: meta}); err != nil {
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
			for _, block := range msg.GetContentBlocks() {
				if block.Type == "tool_use" {
					toolID = block.ID
					if block.Input != nil {
						// Create a human-readable summary of the input
						inputSummary = summarizeInput(block.Input)
					}
				}
			}
			stats.RecordToolUse(msg.Name, toolID)

			meta := &EventMeta{
				ToolName:        msg.Name,
				ToolID:          toolID,
				Status:          "running",
				TotalDurationMs: totalDuration,
				InputSummary:    inputSummary,
			}
			r.logger.Debug("CCRunner: sending tool_use event", "tool_name", msg.Name, "tool_id", toolID)
			if err := callback(EventTypeToolUse, &EventWithMeta{EventType: EventTypeToolUse, EventData: msg.Name, Meta: meta}); err != nil {
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
				OutputSummary:   truncateString(msg.Output, 500),
			}
			r.logger.Debug("CCRunner: sending tool_result event", "output_length", len(msg.Output), "duration_ms", durationMs)
			if err := callback(EventTypeToolResult, &EventWithMeta{EventType: EventTypeToolResult, EventData: msg.Output, Meta: meta}); err != nil {
				return err
			}
		}
	case "message", "content", "text", "delta", "assistant":
		// Assistant message starts generation phase
		stats.EndThinking()
		stats.StartGeneration()

		for _, block := range msg.GetContentBlocks() {
			if block.Type == "text" && block.Text != "" {
				if err := callback(EventTypeAnswer, &EventWithMeta{EventType: EventTypeAnswer, EventData: block.Text, Meta: &EventMeta{TotalDurationMs: totalDuration}}); err != nil {
					return err
				}
			} else if block.Type == "tool_use" && block.Name != "" {
				// Tool use is nested inside assistant message content
				// End generation when tool is about to be used
				stats.EndGeneration()

				stats.RecordToolUse(block.Name, block.ID)

				meta := &EventMeta{
					ToolName:        block.Name,
					ToolID:          block.ID,
					Status:          "running",
					TotalDurationMs: totalDuration,
					InputSummary:    summarizeInput(block.Input),
				}
				r.logger.Info("CCRunner: found nested tool_use", "tool_name", block.Name, "id", block.ID)
				if err := callback(EventTypeToolUse, &EventWithMeta{EventType: EventTypeToolUse, EventData: block.Name, Meta: meta}); err != nil {
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
					OutputSummary:   truncateString(block.Content, 500),
				}
				r.logger.Info("CCRunner: found nested tool_result", "content_length", len(block.Content), "duration_ms", durationMs)
				if err := callback(EventTypeToolResult, &EventWithMeta{EventType: EventTypeToolResult, EventData: block.Content, Meta: meta}); err != nil {
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

		// Try to extract any text content
		for _, block := range msg.GetContentBlocks() {
			if block.Type == "text" && block.Text != "" {
				callback(EventTypeAnswer, &EventWithMeta{EventType: EventTypeAnswer, EventData: block.Text, Meta: &EventMeta{TotalDurationMs: totalDuration}})
			}
		}
	}
	return nil
}

// sanitizeUTF8 ensures a string contains only valid UTF-8 characters.
// Invalid UTF-8 sequences are replaced with the Unicode replacement character.
func sanitizeUTF8(s string) string {
	// Go's string type already handles UTF-8, but when data comes from
	// external sources (like file content or CLI output), it may contain
	// invalid sequences. We use utf8.ValidString to check and strings.ToValidUTF8 to fix.
	if s == "" {
		return ""
	}
	// Convert to valid UTF-8, replacing invalid sequences with �
	// Note: strings.ToValidUTF8 was added in Go 1.15
	return strings.ToValidUTF8(s, "�")
}

// summarizeInput creates a human-readable summary of tool input.
// Uses rune-level truncation to avoid creating invalid UTF-8.
func summarizeInput(input map[string]any) string {
	if input == nil {
		return ""
	}
	// Extract common fields for summary (sanitize first)
	if command, ok := input["command"].(string); ok && command != "" {
		return truncateString(sanitizeUTF8(command), 50)
	}
	if query, ok := input["query"].(string); ok && query != "" {
		return truncateString(sanitizeUTF8(query), 50)
	}
	if path, ok := input["path"].(string); ok && path != "" {
		return "file: " + sanitizeUTF8(path)
	}
	// Fallback to JSON representation (with sanitization)
	jsonBytes, err := json.Marshal(input)
	if err != nil {
		return "(invalid input)"
	}
	str := sanitizeUTF8(string(jsonBytes))
	return truncateString(str, 100)
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
func (r *CCRunner) GetDangerDetector() *DangerDetector {
	return r.dangerDetector
}
