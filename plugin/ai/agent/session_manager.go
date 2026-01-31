package agent

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

// Session represents a persistent process of Claude Code CLI.
type Session struct {
	ID         string
	Config     CCRunnerConfig
	Cmd        *exec.Cmd
	Stdin      io.WriteCloser
	Stdout     io.ReadCloser
	Stderr     io.ReadCloser
	Cancel     context.CancelFunc
	CreatedAt  time.Time
	LastActive time.Time
	Status     SessionStatus

	mu sync.RWMutex
}

// SessionManager defines the interface for managing persistent sessions.
type SessionManager interface {
	GetOrCreateSession(ctx context.Context, sessionID string, cfg CCRunnerConfig) (*Session, error)
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
}

// NewCCSessionManager creates a new session manager.
func NewCCSessionManager(logger *slog.Logger, timeout time.Duration) *CCSessionManager {
	if logger == nil {
		logger = slog.Default()
	}
	return &CCSessionManager{
		sessions: make(map[string]*Session),
		logger:   logger,
		timeout:  timeout,
	}
}

// GetOrCreateSession returns an existing session or starts a new one.
func (sm *CCSessionManager) GetOrCreateSession(ctx context.Context, sessionID string, cfg CCRunnerConfig) (*Session, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Check if session exists and is alive
	if sess, ok := sm.sessions[sessionID]; ok {
		if sess.IsAlive() {
			sess.Touch()
			return sess, nil
		}
		// If dead, cleanup and recreate
		sm.cleanupSessionLocked(sessionID)
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
	var list []*Session
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

	// Cancel context to kill process if using CommandContext
	if sess.Cancel != nil {
		sess.Cancel()
	}

	// Force kill if needed
	if sess.Cmd != nil && sess.Cmd.Process != nil {
		// Use specific signal or Kill
		_ = sess.Cmd.Process.Kill()
	}

	return nil
}

// startSession initializes the process. Caller must hold lock.
func (sm *CCSessionManager) startSession(ctx context.Context, sessionID string, cfg CCRunnerConfig) (*Session, error) {
	cliPath, err := exec.LookPath("claude")
	if err != nil {
		return nil, fmt.Errorf("Claude Code CLI not found: %w", err)
	}

	// Prepare context with cancellation
	// We detach from the incoming ctx because the session should outlive the request
	// But we need a cancel function to stop it.
	sessCtx, cancel := context.WithCancel(context.Background())

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

	stdin, err := cmd.StdinPipe()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		cancel()
		return nil, fmt.Errorf("cmd start: %w", err)
	}

	sm.logger.Info("Session started", "session_id", sessionID, "pid", cmd.Process.Pid)

	return &Session{
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
	}, nil
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

// WriteInput injects a JSON message to Stdin.
func (s *Session) WriteInput(msg map[string]any) error {
	s.mu.Lock()
	defer s.mu.Unlock()

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
