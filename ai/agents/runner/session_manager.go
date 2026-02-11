package runner

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"
)

// SessionStatus defines the current state of a session.
type SessionStatus string

const (
	SessionStatusStarting SessionStatus = "starting"
	SessionStatusReady    SessionStatus = "ready"
	SessionStatusBusy     SessionStatus = "busy"
	SessionStatusDead     SessionStatus = "dead"
)

// Session lifecycle constants.
// 会话生命周期常量。
const (
	defaultReadyTimeout  = 10 * time.Second // Maximum time to wait for session to be ready
	statusBusyDuration   = 2 * time.Second  // Duration to keep session in Busy state after input
	cleanupCheckInterval = 1 * time.Minute  // Interval between idle session cleanup checks
)

// Session represents a persistent process of Claude Code CLI.
type Session struct {
	ID         string
	Config     Config
	Cmd        *exec.Cmd
	Stdin      io.WriteCloser
	Stdout     io.ReadCloser
	Stderr     io.ReadCloser
	Cancel     context.CancelFunc
	CreatedAt  time.Time
	LastActive time.Time
	Status     SessionStatus

	mu               sync.RWMutex
	statusResetTimer *time.Timer // Timer for resetting status from Busy to Ready
}

// SessionManager defines the interface for managing persistent sessions.
type SessionManager interface {
	GetOrCreateSession(ctx context.Context, sessionID string, cfg Config) (*Session, error)
	GetSession(sessionID string) (*Session, bool)
	TerminateSession(sessionID string) error
	ListActiveSessions() []*Session
}

// CCSessionManager implements SessionManager.
type CCSessionManager struct {
	sessions map[string]*Session
	mu       sync.RWMutex
	logger   *slog.Logger
	timeout  time.Duration // Idle timeout
	done     chan struct{} // Shutdown signal
}

// NewCCSessionManager creates a new session manager.
func NewCCSessionManager(logger *slog.Logger, timeout time.Duration) *CCSessionManager {
	if logger == nil {
		logger = slog.Default()
	}
	sm := &CCSessionManager{
		sessions: make(map[string]*Session),
		logger:   logger,
		timeout:  timeout,
		done:     make(chan struct{}),
	}

	// Start idle session cleanup goroutine (per spec 6: 30m idle timeout)
	// 启动空闲会话清理 goroutine（规格 6：30分钟空闲超时）
	go sm.cleanupLoop()

	return sm
}

// GetOrCreateSession returns an existing session or starts a new one.
func (sm *CCSessionManager) GetOrCreateSession(ctx context.Context, sessionID string, cfg Config) (*Session, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Check if session exists and is alive
	if sess, ok := sm.sessions[sessionID]; ok {
		if sess.IsAlive() {
			sess.Touch()
			return sess, nil
		}
		// If dead, cleanup and recreate
		_ = sm.cleanupSessionLocked(sessionID) //nolint:errcheck // cleanup on dead session
	}

	// Create new session
	sess, err := sm.startSession(ctx, sessionID, cfg)
	if err != nil {
		return nil, err
	}

	sm.sessions[sessionID] = sess
	return sess, nil
}

// GetSession retrieves an active session.
func (sm *CCSessionManager) GetSession(sessionID string) (*Session, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	sess, ok := sm.sessions[sessionID]
	return sess, ok
}

// TerminateSession stops and removes a session.
func (sm *CCSessionManager) TerminateSession(sessionID string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	return sm.cleanupSessionLocked(sessionID)
}

// ListActiveSessions returns all active sessions.
func (sm *CCSessionManager) ListActiveSessions() []*Session {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	list := make([]*Session, 0, len(sm.sessions))
	for _, s := range sm.sessions {
		list = append(list, s)
	}
	return list
}

// cleanupSessionLocked stops the process and removes from map. Caller must hold lock.
func (sm *CCSessionManager) cleanupSessionLocked(sessionID string) error {
	sess, ok := sm.sessions[sessionID]
	if !ok {
		return nil
	}

	delete(sm.sessions, sessionID)

	sm.logger.Info("Terminating session", "session_id", sessionID)

	// Stop the status reset timer and clean up session resources
	// Hold session lock to prevent race with WriteInput
	sess.mu.Lock()
	sess.close()
	sess.mu.Unlock()

	// Cancel context to kill process if using CommandContext
	if sess.Cancel != nil {
		sess.Cancel()
	}

	// Force kill if needed
	if sess.Cmd != nil && sess.Cmd.Process != nil {
		// Use specific signal or Kill
		_ = sess.Cmd.Process.Kill() //nolint:errcheck // force terminate
	}

	return nil
}

// startSession initializes the process. Caller must hold lock.
func (sm *CCSessionManager) startSession(ctx context.Context, sessionID string, cfg Config) (*Session, error) {
	// Early exit if request context is already cancelled
	// 尽早退出如果请求上下文已取消
	if ctx.Err() != nil {
		return nil, fmt.Errorf("request context cancelled: %w", ctx.Err())
	}

	cliPath, err := exec.LookPath("claude")
	if err != nil {
		return nil, fmt.Errorf("claude Code CLI not found: %w", err)
	}

	// Prepare context with cancellation.
	// We intentionally use context.Background() instead of the request ctx
	// because the session should outlive the HTTP request that created it.
	// 使用 context.Background() 而非请求 ctx，因为会话的生命周期应超出创建它的 HTTP 请求。
	sessCtx, cancel := context.WithCancel(context.Background())
	// Ensure cancel is always called, even on error paths
	// 确保在所有路径（包括错误路径）上都调用 cancel
	defer cancel()

	// Use a startup timeout to prevent indefinite hangs during process start
	// We monitor startup in a goroutine and cancel if it takes too long
	// 使用启动超时来防止进程启动期间的无限挂起
	startupCtx, startupCancel := context.WithTimeout(ctx, 30*time.Second)
	defer startupCancel()

	// Channel to signal successful startup or failure
	startedCh := make(chan error, 1)

	// Ensure we signal completion even on early return
	// This prevents goroutine leak if function returns before startup completes
	defer close(startedCh)

	// Goroutine to monitor startup timeout
	// If startup takes longer than the timeout, cancel the session
	go func() {
		select {
		case <-startupCtx.Done():
			// Startup timeout or request cancelled - kill the session
			cancel()
			// Channel will be closed by defer, no need to send
		case err, ok := <-startedCh:
			// Startup completed (success or failure)
			if ok && err != nil {
				// Startup failed - cancel the session context
				cancel()
			}
		}
	}()

	// Build arguments
	// NOTE: Logic duplicate from CCRunner.executeWithSession slightly, refactor later if needed.
	// We always force --output-format stream-json and --print

	// Check if first call logic is needed?
	// The session manager just starts the process.
	// Persistence: --session-id is key.

	// We will use "Resume" logic if we trust the session ID persistence on disk,
	// OR we always treat it as "maybe resume".
	// The CLI handles "resume" vs "new" based on session ID existence?
	// Actually CLI has --resume <id> vs --session-id <id>.
	// Let's stick to --session-id for creation and --resume for re-connection?
	// Wait, spec says: Args: --print --verbose --output-format stream-json --session-id <sid>

	args := []string{
		"--print",
		"--verbose",
		"--output-format", "stream-json",
		"--input-format", "stream-json",
		"--session-id", sessionID,
	}

	if cfg.PermissionMode != "" {
		args = append(args, "--permission-mode", cfg.PermissionMode)
	}

	// Note: We don't pass the initial prompt here. The prompt will be injected via stdin later
	// OR passed as argument. BUT we want a persistent session.
	// If we pass a prompt arg, it runs and might exit?
	// CC Runner usually waits for input if interactive?
	// Spec says: "Process starts... hangs waiting for stdin".
	// Depending on CC CLI behavior, if no prompt provided, does it start REPL?
	// We assume passing no prompt starts REPL mode or waits.

	// However, if we need to set System Prompt, we should do it at start.
	if cfg.SystemPrompt != "" {
		args = append(args, "--append-system-prompt", cfg.SystemPrompt)
	}

	cmd := exec.CommandContext(sessCtx, cliPath, args...)
	cmd.Dir = cfg.WorkDir
	cmd.Env = append(os.Environ(), "CLAUDE_DISABLE_TELEMETRY=1")

	// Create pipes with proper cleanup on error paths
	// 创建管道并在错误路径上正确清理
	var stdin io.WriteCloser
	var stdout, stderr io.ReadCloser

	stdin, err = cmd.StdinPipe()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("stdin pipe: %w", err)
	}

	stdout, err = cmd.StdoutPipe()
	if err != nil {
		_ = stdin.Close() //nolint:errcheck // cleanup on error path
		cancel()
		return nil, fmt.Errorf("stdout pipe: %w", err)
	}
	defer func() { _ = stdout.Close() }() //nolint:errcheck // cleanup on defer stack

	stderr, err = cmd.StderrPipe()
	if err != nil {
		_ = stdout.Close() //nolint:errcheck // cleanup on error path
		_ = stdin.Close()  //nolint:errcheck // cleanup on error path
		cancel()
		return nil, fmt.Errorf("stderr pipe: %w", err)
	}
	defer func() { _ = stderr.Close() }() //nolint:errcheck // cleanup on defer stack

	if err := cmd.Start(); err != nil {
		startedCh <- err // Signal startup failed
		return nil, fmt.Errorf("cmd start: %w", err)
	}

	// Signal that startup succeeded
	startedCh <- nil

	sm.logger.Info("Session started", "session_id", sessionID, "pid", cmd.Process.Pid)

	sess := &Session{
		ID:         sessionID,
		Config:     cfg,
		Cmd:        cmd,
		Stdin:      stdin,
		Stdout:     stdout,
		Stderr:     stderr,
		Cancel:     cancel,
		CreatedAt:  time.Now(),
		LastActive: time.Now(),
		Status:     SessionStatusStarting,
	}

	// Start status transition monitor: Starting -> Ready
	// 启动状态转换监控：Starting -> Ready
	// Pass the startup context so waitForReady can be cancelled if startup times out
	sess.waitForReady(startupCtx, defaultReadyTimeout)

	return sess, nil
}

// IsAlive checks if the process is still running.
func (s *Session) IsAlive() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.Cmd == nil || s.Cmd.Process == nil {
		return false
	}

	// Non-blocking wait to check status?
	// Since we use CommandContext, ProcessState is set only after Wait() returns.
	// But Wait() closes pipes.
	// A simple way is relying on the fact that if it crashed, writing to Stdin or Reading Stdout might fail.
	// Or we can check process existence (signal 0).

	if err := s.Cmd.Process.Signal(syscall.Signal(0)); err != nil {
		return false
	}
	return true
}

// Touch updates LastActive time.
func (s *Session) Touch() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.LastActive = time.Now()
}

// SetStatus updates the session status with proper locking.
// SetStatus 使用适当的锁更新会话状态。
func (s *Session) SetStatus(status SessionStatus) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Status = status
}

// GetStatus returns the current session status.
// GetStatus 返回当前会话状态。
func (s *Session) GetStatus() SessionStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Status
}

// waitForReady monitors the session and transitions from Starting to Ready
// when the process is confirmed alive and responsive.
// waitForReady 监控会话，当进程确认存活且响应时从 Starting 转换为 Ready。
// The context parameter allows cancellation if the session is terminated early.
func (s *Session) waitForReady(ctx context.Context, timeout time.Duration) {
	go func() {
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()

		deadline := time.Now().Add(timeout)
		for time.Now().Before(deadline) {
			select {
			case <-ctx.Done():
				// Context cancelled - session terminated or request cancelled
				return
			case <-ticker.C:
				s.mu.Lock()
				if s.Status == SessionStatusDead {
					s.mu.Unlock()
					return
				}
				if s.IsAlive() {
					s.Status = SessionStatusReady
					s.mu.Unlock()
					return
				}
				s.mu.Unlock()
			}
		}
		// Timeout - mark as dead if still not alive
		s.mu.Lock()
		if s.Status == SessionStatusStarting {
			s.Status = SessionStatusDead
		}
		s.mu.Unlock()
	}()
}

// WriteInput injects a JSON message to Stdin.
// Transitions session to Busy during write, back to Ready after completion.
// 注入 JSON 消息到 Stdin。写入时转换为 Busy，完成后恢复为 Ready。
func (s *Session) WriteInput(msg map[string]any) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Set status to Busy while processing input
	// Must be done under lock to prevent race with cleanup
	s.Status = SessionStatusBusy

	// Reset existing timer if any (prevents goroutine accumulation)
	// 重置现有定时器（防止 goroutine 累积）
	if s.statusResetTimer != nil {
		// Stop the timer and check if it was already fired
		if !s.statusResetTimer.Stop() {
			// Timer already fired - callback may be running or about to run
			// Release lock briefly to allow callback to complete if it's holding lock
			s.mu.Unlock()
			time.Sleep(50 * time.Millisecond) // Give callback time to complete
			s.mu.Lock()
		}
	}

	// Schedule status recovery to Ready after a short delay
	// This allows the session to be marked busy while the CLI processes the input
	// 调度状态恢复到 Ready（允许 CLI 处理输入时保持 Busy 状态）
	// Callback acquires lock to prevent race with WriteInput
	s.statusResetTimer = time.AfterFunc(statusBusyDuration, func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		if s.IsAlive() {
			s.Status = SessionStatusReady
		}
	})

	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	// Append newline as protocol often requires it (JSONL)
	data = append(data, '\n')

	_, err = s.Stdin.Write(data)
	if err != nil {
		return err
	}

	s.LastActive = time.Now()
	return nil
}

// close releases resources held by the session.
// Must be called with session lock held.
// close 释放会话持有的资源。必须在持有会话锁时调用。
func (s *Session) close() {
	// Stop the status reset timer if exists
	// Use a local copy to avoid holding lock during Stop()
	if s.statusResetTimer != nil {
		timer := s.statusResetTimer
		s.statusResetTimer = nil
		// Timer.Stop is safe to call multiple times and from different goroutines
		timer.Stop()
	}
}

// cleanupLoop runs periodic cleanup of idle sessions.
// Runs every minute and terminates sessions that have been idle longer than timeout.
// 运行定期清理空闲会话。每分钟检查一次，终止空闲超过超时时间的会话。
func (sm *CCSessionManager) cleanupLoop() {
	ticker := time.NewTicker(cleanupCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			sm.cleanupIdleSessions()
		case <-sm.done:
			return
		}
	}
}

// cleanupIdleSessions removes sessions that have exceeded the idle timeout.
// 移除超过空闲超时的会话。
func (sm *CCSessionManager) cleanupIdleSessions() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	now := time.Now()
	for sessionID, sess := range sm.sessions {
		idleTime := now.Sub(sess.LastActive)
		if idleTime > sm.timeout {
			sm.logger.Info("Session idle timeout, terminating",
				"session_id", sessionID,
				"idle_duration", idleTime,
				"timeout", sm.timeout)
			_ = sm.cleanupSessionLocked(sessionID) //nolint:errcheck // cleanup on idle timeout
		}
	}
}

// Shutdown gracefully stops the session manager and all active sessions.
// Shutdown 优雅停止会话管理器和所有活动会话。
func (sm *CCSessionManager) Shutdown() {
	close(sm.done)

	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Terminate all sessions
	for sessionID := range sm.sessions {
		_ = sm.cleanupSessionLocked(sessionID) //nolint:errcheck // cleanup on shutdown
	}
}
