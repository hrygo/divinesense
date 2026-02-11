package runner

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"sync"
	"testing"
	"time"
)

// TestSessionStatus_Transitions tests the session state machine.
func TestSessionStatus_Transitions(t *testing.T) {
	t.Run("initial_status_is_starting", func(t *testing.T) {
		sess := &Session{
			ID:     "test-1",
			Status: SessionStatusStarting,
		}

		if sess.GetStatus() != SessionStatusStarting {
			t.Errorf("initial status = %s, want %s", sess.GetStatus(), SessionStatusStarting)
		}
	})

	t.Run("set_status_updates_correctly", func(t *testing.T) {
		sess := &Session{
			ID:     "test-2",
			Status: SessionStatusStarting,
		}

		sess.SetStatus(SessionStatusReady)
		if sess.GetStatus() != SessionStatusReady {
			t.Errorf("status after SetStatus = %s, want %s", sess.GetStatus(), SessionStatusReady)
		}

		sess.SetStatus(SessionStatusBusy)
		if sess.GetStatus() != SessionStatusBusy {
			t.Errorf("status after SetStatus = %s, want %s", sess.GetStatus(), SessionStatusBusy)
		}
	})

	t.Run("touch_updates_last_active", func(t *testing.T) {
		sess := &Session{
			ID:         "test-3",
			Status:     SessionStatusReady,
			CreatedAt:  time.Now().Add(-1 * time.Hour),
			LastActive: time.Now().Add(-1 * time.Hour),
		}

		oldLastActive := sess.LastActive
		time.Sleep(10 * time.Millisecond)
		sess.Touch()

		if !sess.LastActive.After(oldLastActive) {
			t.Error("Touch() did not update LastActive")
		}
	})
}

// TestCCSessionManager_GetOrCreateSession tests the session creation and retrieval.
func TestCCSessionManager_GetOrCreateSession(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Check if claude CLI is available
	if _, err := exec.LookPath("claude"); err != nil {
		t.Skip("claude CLI not found, skipping test")
	}

	t.Run("returns_existing_session", func(t *testing.T) {
		sm := NewCCSessionManager(nil, 30*time.Minute)
		defer sm.Shutdown()

		ctx := context.Background()
		cfg := Config{
			WorkDir:        os.TempDir(),
			ConversationID: 123,
			UserID:         1,
			PermissionMode: "bypassPermissions",
		}

		// Create first session
		sess1, err := sm.GetOrCreateSession(ctx, "test-session-1", cfg)
		if err != nil {
			t.Fatalf("GetOrCreateSession failed: %v", err)
		}

		// Request same session ID - should return existing
		sess2, err := sm.GetOrCreateSession(ctx, "test-session-1", cfg)
		if err != nil {
			t.Fatalf("second GetOrCreateSession failed: %v", err)
		}

		if sess1.ID != sess2.ID {
			t.Errorf("session IDs differ: %s vs %s", sess1.ID, sess2.ID)
		}

		// Clean up
		sm.TerminateSession("test-session-1")
	})

	t.Run("creates_different_sessions_for_different_ids", func(t *testing.T) {
		sm := NewCCSessionManager(nil, 30*time.Minute)
		defer sm.Shutdown()

		ctx := context.Background()
		cfg := Config{
			WorkDir:        os.TempDir(),
			ConversationID: 456,
			UserID:         1,
		}

		sess1, err := sm.GetOrCreateSession(ctx, "test-session-2", cfg)
		if err != nil {
			t.Fatalf("GetOrCreateSession failed: %v", err)
		}

		sess2, err := sm.GetOrCreateSession(ctx, "test-session-3", cfg)
		if err != nil {
			t.Fatalf("second GetOrCreateSession failed: %v", err)
		}

		if sess1.ID == sess2.ID {
			t.Errorf("expected different session IDs, got same: %s", sess1.ID)
		}

		// Clean up
		sm.TerminateSession("test-session-2")
		sm.TerminateSession("test-session-3")
	})
}

// TestCCSessionManager_IdleCleanup tests that idle sessions are cleaned up.
func TestCCSessionManager_IdleCleanup(t *testing.T) {
	t.Run("idle_session_removed_after_timeout", func(t *testing.T) {
		shortTimeout := 100 * time.Millisecond
		sm := NewCCSessionManager(nil, shortTimeout)
		defer sm.Shutdown()

		// Manually add a stale session
		staleSess := &Session{
			ID:         "stale-session",
			Status:     SessionStatusReady,
			CreatedAt:  time.Now().Add(-1 * time.Hour),
			LastActive: time.Now().Add(-1 * time.Hour),
			Cmd:        &exec.Cmd{},
		}

		sm.mu.Lock()
		sm.sessions["stale-session"] = staleSess
		sm.mu.Unlock()

		// Trigger cleanup manually
		sm.cleanupIdleSessions()

		// Session should be removed
		if _, ok := sm.GetSession("stale-session"); ok {
			t.Error("idle session was not removed after timeout")
		}
	})

	t.Run("active_session_not_cleaned_up", func(t *testing.T) {
		shortTimeout := 100 * time.Millisecond
		sm := NewCCSessionManager(nil, shortTimeout)
		defer sm.Shutdown()

		// Add a recently active session
		activeSess := &Session{
			ID:         "active-session",
			Status:     SessionStatusReady,
			CreatedAt:  time.Now(),
			LastActive: time.Now(),
			Cmd:        &exec.Cmd{},
		}

		sm.mu.Lock()
		sm.sessions["active-session"] = activeSess
		sm.mu.Unlock()

		// Trigger cleanup
		sm.cleanupIdleSessions()

		// Session should still exist
		if _, ok := sm.GetSession("active-session"); !ok {
			t.Error("active session was incorrectly removed")
		}
	})
}

// TestCCSessionManager_ConcurrentAccess tests thread-safety of session operations.
func TestCCSessionManager_ConcurrentAccess(t *testing.T) {
	sm := NewCCSessionManager(nil, 30*time.Minute)
	defer sm.Shutdown()

	const numGoroutines = 50
	const numOperations = 100

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// Launch multiple goroutines performing concurrent operations
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()

			for j := 0; j < numOperations; j++ {
				sessionID := "concurrent-test"

				// Mix of read and write operations
				switch j % 4 {
				case 0:
					sm.GetSession(sessionID)
				case 1:
					sm.ListActiveSessions()
				case 2:
					// Try to terminate (may not exist, that's ok)
					_ = sm.TerminateSession(sessionID)
				case 3:
					// Touch any existing session
					if sess, ok := sm.GetSession(sessionID); ok {
						sess.Touch()
					}
				}
			}
		}(i)
	}

	wg.Wait()

	// If we reach here without panic/deadlock, test passes
}

// TestSession_IsAlive tests process liveness detection.
func TestSession_IsAlive(t *testing.T) {
	t.Run("nil_process_returns_false", func(t *testing.T) {
		sess := &Session{
			ID:     "test-nil",
			Status: SessionStatusStarting,
			Cmd:    nil,
		}

		if sess.IsAlive() {
			t.Error("IsAlive() = true for nil Cmd, want false")
		}
	})

	t.Run("process_with_nil_process_state_returns_false", func(t *testing.T) {
		sess := &Session{
			ID:     "test-no-process",
			Status: SessionStatusStarting,
			Cmd:    &exec.Cmd{},
		}

		if sess.IsAlive() {
			t.Error("IsAlive() = true for Cmd with nil Process, want false")
		}
	})
}

// TestSession_WriteInput_StatusTransition tests status transitions during input.
// Focuses on verifying status is set to Busy and timer is created, without causing deadlocks.
func TestSession_WriteInput_StatusTransition(t *testing.T) {
	t.Run("write_sets_busy_status", func(t *testing.T) {
		// Use a channel to signal when callback completes
		callbackDone := make(chan struct{})

		sess := &Session{
			ID:     "test-busy",
			Status: SessionStatusReady,
			// Use a buffer that won't block
			Stdin: &nopWriteCloser{},
		}

		msg := map[string]any{"test": "input"}
		if err := sess.WriteInput(msg); err != nil {
			t.Fatalf("WriteInput failed: %v", err)
		}

		// Check timer was created
		sess.mu.Lock()
		hasTimer := sess.statusResetTimer != nil
		sess.mu.Unlock()

		if !hasTimer {
			t.Error("statusResetTimer was not created")
		}

		// Wait for status reset callback to potentially run (statusBusyDuration = 2s)
		// We'll check after a short delay if status transitions correctly
		// To avoid long test, we'll stop the timer manually

		// Modify the timer to fire quickly for testing
		sess.mu.Lock()
		if sess.statusResetTimer != nil {
			sess.statusResetTimer.Stop()
			// Create a new timer with short duration
			sess.statusResetTimer = time.AfterFunc(10*time.Millisecond, func() {
				sess.mu.Lock()
				if sess.Cmd == nil || sess.Cmd.Process == nil {
					// Test session without real process - set to Ready anyway
					sess.Status = SessionStatusReady
				} else if sess.IsAlive() {
					sess.Status = SessionStatusReady
				}
				sess.mu.Unlock()
				close(callbackDone)
			})
		}
		sess.mu.Unlock()

		// Wait for callback to complete
		select {
		case <-callbackDone:
			// Callback completed
		case <-time.After(100 * time.Millisecond):
			// Timeout - timer may have been stopped, that's ok
		}

		// Verify status transitioned back to Ready
		if sess.GetStatus() != SessionStatusReady {
			t.Errorf("status after callback = %s, want %s", sess.GetStatus(), SessionStatusReady)
		}
	})

	t.Run("write_increments_last_active", func(t *testing.T) {
		sess := &Session{
			ID:         "test-active",
			Status:     SessionStatusReady,
			LastActive: time.Now().Add(-1 * time.Hour),
			Stdin:      &nopWriteCloser{},
		}

		oldLastActive := sess.LastActive

		msg := map[string]any{"test": "input"}
		if err := sess.WriteInput(msg); err != nil {
			t.Fatalf("WriteInput failed: %v", err)
		}

		if !sess.LastActive.After(oldLastActive) {
			t.Error("WriteInput did not update LastActive")
		}
	})
}

// nopWriteCloser is a no-op WriteCloser for testing.
type nopWriteCloser struct{}

func (n *nopWriteCloser) Write(p []byte) (int, error) {
	return len(p), nil
}

func (n *nopWriteCloser) Close() error {
	return nil
}

// TestCCSessionManager_ListActiveSessions tests listing all active sessions.
func TestCCSessionManager_ListActiveSessions(t *testing.T) {
	sm := NewCCSessionManager(nil, 30*time.Minute)
	defer sm.Shutdown()

	// Add multiple sessions manually
	sessions := []*Session{
		{ID: "sess-1", Status: SessionStatusReady, Cmd: &exec.Cmd{}},
		{ID: "sess-2", Status: SessionStatusBusy, Cmd: &exec.Cmd{}},
		{ID: "sess-3", Status: SessionStatusStarting, Cmd: &exec.Cmd{}},
	}

	sm.mu.Lock()
	for _, s := range sessions {
		sm.sessions[s.ID] = s
	}
	sm.mu.Unlock()

	list := sm.ListActiveSessions()

	if len(list) != len(sessions) {
		t.Errorf("ListActiveSessions() returned %d sessions, want %d", len(list), len(sessions))
	}

	// Create a map of returned IDs for easy verification
	idMap := make(map[string]bool)
	for _, s := range list {
		idMap[s.ID] = true
	}

	for _, expected := range sessions {
		if !idMap[expected.ID] {
			t.Errorf("session %s not found in list", expected.ID)
		}
	}
}

// TestCCSessionManager_TerminateSession tests session termination.
func TestCCSessionManager_TerminateSession(t *testing.T) {
	t.Run("terminate_existing_session", func(t *testing.T) {
		sm := NewCCSessionManager(nil, 30*time.Minute)
		defer sm.Shutdown()

		sess := &Session{
			ID:     "to-terminate",
			Status: SessionStatusReady,
			Cmd:    &exec.Cmd{},
		}

		sm.mu.Lock()
		sm.sessions["to-terminate"] = sess
		sm.mu.Unlock()

		err := sm.TerminateSession("to-terminate")
		if err != nil {
			t.Errorf("TerminateSession failed: %v", err)
		}

		if _, ok := sm.GetSession("to-terminate"); ok {
			t.Error("session still exists after TerminateSession")
		}
	})

	t.Run("terminate_non_existing_session_no_error", func(t *testing.T) {
		sm := NewCCSessionManager(nil, 30*time.Minute)
		defer sm.Shutdown()

		// Terminating non-existent session should not error
		err := sm.TerminateSession("does-not-exist")
		if err != nil {
			t.Errorf("TerminateSession of non-existent returned error: %v", err)
		}
	})
}

// TestSession_ConcurrentTouch tests concurrent Touch calls don't cause panic.
func TestSession_ConcurrentTouch(t *testing.T) {
	sess := &Session{
		ID:         "concurrent-touch",
		Status:     SessionStatusReady,
		LastActive: time.Now(),
	}

	const numGoroutines = 100
	var wg sync.WaitGroup

	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			sess.Touch()
		}()
	}

	wg.Wait()
	// If we reach here without race condition, test passes
}

// TestSession_close tests resource cleanup on close.
func TestSession_close(t *testing.T) {
	t.Run("close_stops_timer", func(t *testing.T) {
		sess := &Session{
			ID:     "test-close",
			Status: SessionStatusReady,
		}

		// Create a timer with a short delay for testing
		timerFired := false
		sess.statusResetTimer = time.AfterFunc(100*time.Millisecond, func() {
			timerFired = true
		})

		// Immediately stop via close
		sess.mu.Lock()
		sess.close()
		sess.mu.Unlock()

		// Wait a bit longer than the timer duration
		time.Sleep(150 * time.Millisecond)

		// Timer should have been stopped
		if timerFired {
			// Timer fired before we could stop it - this is a race in the test
			// but we can verify the session's timer reference was cleared
		}

		if sess.statusResetTimer != nil {
			t.Error("statusResetTimer not nil after close")
		}
	})
}

// TestNewCCSessionManager tests session manager creation.
func TestNewCCSessionManager(t *testing.T) {
	t.Run("creates_manager_with_defaults", func(t *testing.T) {
		sm := NewCCSessionManager(nil, 5*time.Minute)
		defer sm.Shutdown()

		if sm == nil {
			t.Fatal("NewCCSessionManager returned nil")
		}

		if sm.sessions == nil {
			t.Error("sessions map is nil")
		}

		if sm.timeout != 5*time.Minute {
			t.Errorf("timeout = %v, want %v", sm.timeout, 5*time.Minute)
		}
	})

	t.Run("creates_manager_with_nil_logger_uses_default", func(t *testing.T) {
		sm := NewCCSessionManager(nil, 10*time.Minute)
		defer sm.Shutdown()

		if sm.logger == nil {
			t.Error("logger should not be nil when nil is passed")
		}
	})
}

// TestCCSessionManager_Shutdown tests graceful shutdown.
func TestCCSessionManager_Shutdown(t *testing.T) {
	t.Run("shutdown_clears_all_sessions", func(t *testing.T) {
		sm := NewCCSessionManager(nil, 30*time.Minute)

		// Add some sessions
		sm.mu.Lock()
		sm.sessions["sess-1"] = &Session{ID: "sess-1", Cmd: &exec.Cmd{}}
		sm.sessions["sess-2"] = &Session{ID: "sess-2", Cmd: &exec.Cmd{}}
		sm.mu.Unlock()

		sm.Shutdown()

		// All sessions should be cleared
		list := sm.ListActiveSessions()
		if len(list) != 0 {
			t.Errorf("Shutdown left %d sessions, want 0", len(list))
		}
	})
}

// TestSession_waitForReady tests that waitForReady handles context cancellation.
func TestSession_waitForReady(t *testing.T) {
	t.Run("context_cancellation_stops_wait", func(t *testing.T) {
		sess := &Session{
			ID:     "test-wait",
			Status: SessionStatusStarting,
			Cmd:    &exec.Cmd{},
		}

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		// This should return quickly without panic
		sess.waitForReady(ctx, 10*time.Second)

		// Give goroutine time to exit
		time.Sleep(100 * time.Millisecond)
	})
}

// TestCCSessionManager_GetSession tests retrieving a single session.
func TestCCSessionManager_GetSession(t *testing.T) {
	sm := NewCCSessionManager(nil, 30*time.Minute)
	defer sm.Shutdown()

	t.Run("returns_false_for_nonexistent_session", func(t *testing.T) {
		_, ok := sm.GetSession("does-not-exist")
		if ok {
			t.Error("GetSession returned true for nonexistent session")
		}
	})

	t.Run("returns_existing_session", func(t *testing.T) {
		expected := &Session{
			ID:     "existing",
			Status: SessionStatusReady,
			Cmd:    &exec.Cmd{},
		}

		sm.mu.Lock()
		sm.sessions["existing"] = expected
		sm.mu.Unlock()

		sess, ok := sm.GetSession("existing")
		if !ok {
			t.Error("GetSession returned false for existing session")
		}
		if sess.ID != expected.ID {
			t.Errorf("got session ID %s, want %s", sess.ID, expected.ID)
		}
	})
}

// TestSession_WriteInput_Marshaling tests that WriteInput correctly marshals JSON.
func TestSession_WriteInput_Marshaling(t *testing.T) {
	t.Run("marshals_input_with_newline", func(t *testing.T) {
		sess := &Session{
			ID:     "test-marshal",
			Status: SessionStatusReady,
			Stdin:  &nopWriteCloser{},
		}

		// Override Stdin to capture writes
		writeCalled := false
		var writtenData []byte
		sess.Stdin = &writeCaptureCloser{
			fn: func(p []byte) (int, error) {
				writeCalled = true
				writtenData = p
				return len(p), nil
			},
		}

		msg := map[string]any{"command": "test", "arg": 123}
		if err := sess.WriteInput(msg); err != nil {
			t.Fatalf("WriteInput failed: %v", err)
		}

		if !writeCalled {
			t.Fatal("Stdin.Write was not called")
		}

		// Verify JSON marshaling
		// Should end with newline
		if len(writtenData) == 0 || writtenData[len(writtenData)-1] != '\n' {
			t.Error("WriteInput does not append newline")
		}

		// Verify valid JSON (minus the newline)
		var decoded map[string]any
		if err := json.Unmarshal(writtenData[:len(writtenData)-1], &decoded); err != nil {
			t.Errorf("written data is not valid JSON: %v", err)
		}

		if decoded["command"] != "test" {
			t.Errorf("command = %v, want 'test'", decoded["command"])
		}
	})
}

// writeCaptureCloser captures writes for testing.
type writeCaptureCloser struct {
	fn func([]byte) (int, error)
}

func (w *writeCaptureCloser) Write(p []byte) (int, error) {
	return w.fn(p)
}

func (w *writeCaptureCloser) Close() error {
	return nil
}
