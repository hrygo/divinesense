package agent

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// skipIfNoCLI skips the test if Claude Code CLI is not available.
func skipIfNoCLI(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("claude"); err != nil {
		t.Skip("Claude Code CLI not found in PATH - skipping test")
	}
}

// TestNewEvolutionParrot tests EvolutionParrot creation.
func TestNewEvolutionParrot(t *testing.T) {
	skipIfNoCLI(t)

	// Enable evolution mode for testing
	t.Setenv("DIVINESENSE_EVOLUTION_ENABLED", "true")

	t.Run("successful creation", func(t *testing.T) {
		parrot, err := NewEvolutionParrot("/src", 1, "test-session", nil, false)
		if err != nil {
			t.Fatalf("NewEvolutionParrot() error = %v", err)
		}

		if parrot.Name() != "evolution" {
			t.Errorf("Name() = %v, want %v", parrot.Name(), "evolution")
		}

		if parrot.GetWorkDir() != "/src" {
			t.Errorf("GetWorkDir() = %v, want %v", parrot.GetWorkDir(), "/src")
		}

		if parrot.GetUserID() != 1 {
			t.Errorf("GetUserID() = %v, want %v", parrot.GetUserID(), 1)
		}

		if parrot.GetSessionID() != "test-session" {
			t.Errorf("GetSessionID() = %v, want %v", parrot.GetSessionID(), "test-session")
		}

		if parrot.GetTaskID() == "" {
			t.Error("GetTaskID() should not be empty")
		}

		if parrot.IsSessionActive() {
			t.Error("IsSessionActive() should be false initially")
		}
	})

	t.Run("generates session ID when not provided", func(t *testing.T) {
		parrot, err := NewEvolutionParrot("/src", 1, "", nil, false)
		if err != nil {
			t.Fatalf("NewEvolutionParrot() error = %v", err)
		}

		if parrot.GetSessionID() == "" {
			t.Error("GetSessionID() should generate ID when not provided")
		}
	})
}

// TestEvolutionParrotPermissionDenied tests permission checks.
func TestEvolutionParrotPermissionDenied(t *testing.T) {
	skipIfNoCLI(t)
	t.Run("evolution mode disabled", func(t *testing.T) {
		t.Setenv("DIVINESENSE_EVOLUTION_ENABLED", "false")

		parrot, err := NewEvolutionParrot("/src", 1, "test-session", nil, false)
		if err != nil {
			t.Fatalf("NewEvolutionParrot() error = %v", err)
		}

		ctx := context.Background()
		err = parrot.ExecuteWithCallback(ctx, "test input", nil, nil)
		if err == nil {
			t.Error("ExecuteWithCallback() should return error when evolution mode is disabled")
		}

		if !strings.Contains(err.Error(), "disabled") {
			t.Errorf("Error should mention disabled, got: %v", err)
		}
	})

	t.Run("admin check with nil store", func(t *testing.T) {
		t.Setenv("DIVINESENSE_EVOLUTION_ENABLED", "true")

		parrot, err := NewEvolutionParrot("/src", 1, "test-session", nil, true)
		if err != nil {
			t.Fatalf("NewEvolutionParrot() error = %v", err)
		}

		ctx := context.Background()
		err = parrot.ExecuteWithCallback(ctx, "test input", nil, nil)
		if err == nil {
			t.Error("ExecuteWithCallback() should return error when store is nil and admin check is enabled")
		}

		if !strings.Contains(err.Error(), "admin privileges") {
			t.Errorf("Error should mention admin privileges, got: %v", err)
		}
	})
}

// TestEvolutionParrotSetDeviceContext tests device context setting.
func TestEvolutionParrotSetDeviceContext(t *testing.T) {
	skipIfNoCLI(t)
	parrot, err := NewEvolutionParrot("/src", 1, "test-session", nil, false)
	if err != nil {
		t.Fatalf("NewEvolutionParrot() error = %v", err)
	}

	deviceCtx := `{"userAgent":"test","isMobile":false}`
	parrot.SetDeviceContext(deviceCtx)

	// Can't directly access deviceCtx, but this ensures no panic
	// The field will be used in ExecuteWithCallback
}

// TestEvolutionParrotSelfDescribe tests metacognitive information.
func TestEvolutionParrotSelfDescribe(t *testing.T) {
	skipIfNoCLI(t)
	parrot, err := NewEvolutionParrot("/src", 1, "test-session", nil, false)
	if err != nil {
		t.Fatalf("NewEvolutionParrot() error = %v", err)
	}

	desc := parrot.SelfDescribe()

	if desc.Name != "evolution" {
		t.Errorf("Name = %v, want %v", desc.Name, "evolution")
	}

	if desc.Emoji != "🧬" {
		t.Errorf("Emoji = %v, want %v", desc.Emoji, "🧬")
	}

	if desc.Title != "Evolution Mode - Self-Evolving Agent" {
		t.Errorf("Title = %v, want %v", desc.Title, "Evolution Mode - Self-Evolving Agent")
	}

	if len(desc.Personality) == 0 {
		t.Error("Personality should not be empty")
	}

	if len(desc.Capabilities) == 0 {
		t.Error("Capabilities should not be empty")
	}

	if len(desc.Limitations) == 0 {
		t.Error("Limitations should not be empty")
	}

	// Check specific capability
	found := false
	for _, cap := range desc.Capabilities {
		if strings.Contains(cap, "CLAUDE.md") {
			found = true
			break
		}
	}
	if !found {
		t.Error("Capabilities should mention CLAUDE.md")
	}
}

// TestEvolutionParrotResetSession tests session reset.
func TestEvolutionParrotResetSession(t *testing.T) {
	skipIfNoCLI(t)
	parrot, err := NewEvolutionParrot("/src", 1, "test-session", nil, false)
	if err != nil {
		t.Fatalf("NewEvolutionParrot() error = %v", err)
	}

	originalSessionID := parrot.GetSessionID()

	parrot.ResetSession()

	if parrot.IsSessionActive() {
		t.Error("IsSessionActive() should be false after reset")
	}

	newSessionID := parrot.GetSessionID()
	if newSessionID == originalSessionID {
		t.Error("ResetSession() should generate new session ID")
	}
}

// TestEvolutionParrotCancel tests cancellation.
func TestEvolutionParrotCancel(t *testing.T) {
	skipIfNoCLI(t)
	parrot, err := NewEvolutionParrot("/src", 1, "test-session", nil, false)
	if err != nil {
		t.Fatalf("NewEvolutionParrot() error = %v", err)
	}

	parrot.Cancel()

	if parrot.IsSessionActive() {
		t.Error("IsSessionActive() should be false after cancel")
	}
}

// TestEvolutionParrotExecuteWithoutCLI tests execution when CLI is not available.
func TestEvolutionParrotExecuteWithoutCLI(t *testing.T) {
	// Save original PATH
	origPath := os.Getenv("PATH")
	defer os.Setenv("PATH", origPath)

	// Set empty PATH to simulate CLI not found
	os.Setenv("PATH", "")
	t.Setenv("DIVINESENSE_EVOLUTION_ENABLED", "true")

	_, err := NewEvolutionParrot("/src", 1, "test-session", nil, false)
	if err == nil {
		t.Error("NewEvolutionParrot() should return error when CLI is not found")
	}

	if !strings.Contains(err.Error(), "CCRunner") {
		t.Errorf("Error should mention CCRunner, got: %v", err)
	}
}

// TestEvolutionParrotAdminOnlyEnvVar tests admin-only environment variable.
func TestEvolutionParrotAdminOnlyEnvVar(t *testing.T) {
	skipIfNoCLI(t)
	t.Setenv("DIVINESENSE_EVOLUTION_ENABLED", "true")

	t.Run("admin only true from env", func(t *testing.T) {
		t.Setenv("DIVINESENSE_EVOLUTION_ADMIN_ONLY", "true")
		parrot, err := NewEvolutionParrot("/src", 1, "test-session", nil, true)
		if err != nil {
			t.Fatalf("NewEvolutionParrot() error = %v", err)
		}
		// Admin check will fail with nil store, tested above
		_ = parrot
	})

	t.Run("admin only false from env", func(t *testing.T) {
		t.Setenv("DIVINESENSE_EVOLUTION_ADMIN_ONLY", "false")
		parrot, err := NewEvolutionParrot("/src", 1, "test-session", nil, true)
		if err != nil {
			t.Fatalf("NewEvolutionParrot() error = %v", err)
		}
		_ = parrot
	})
}

// TestEvolutionParrotCallbackError tests error callback handling.
func TestEvolutionParrotCallbackError(t *testing.T) {
	skipIfNoCLI(t)
	t.Setenv("DIVINESENSE_EVOLUTION_ENABLED", "true")

	// Use a temp directory that's writable
	tmpDir := t.TempDir()

	parrot, err := NewEvolutionParrot(tmpDir, 1, "test-session", nil, true) // adminOnly=true
	if err != nil {
		t.Fatalf("NewEvolutionParrot() error = %v", err)
	}

	var callbackCalled bool

	callback := func(eventType string, eventData any) error {
		callbackCalled = true
		return nil
	}

	// With adminOnly=true and nil store, permission check should fail
	ctx := context.Background()
	err = parrot.ExecuteWithCallback(ctx, "test", nil, callback)

	// Error should be returned from ExecuteWithCallback
	if err == nil {
		t.Error("ExecuteWithCallback() should return error when admin check fails")
	}

	// Error should be about admin privileges (permission denied)
	if !strings.Contains(err.Error(), "admin") && !strings.Contains(err.Error(), "Permission") {
		t.Errorf("Error should mention admin/permission, got: %v", err)
	}

	// Callback should be called to send error to client
	if !callbackCalled {
		t.Error("Callback should be called on error")
	}
}

// TestEvolutionParrotSessionInitialization tests session initialization after execution.
func TestEvolutionParrotSessionInitialization(t *testing.T) {
	// This test would require a valid Claude Code CLI to run
	// Mark as skip for normal test runs
	t.Skip("requires Claude Code CLI - integration test only")

	/*
		t.Setenv("DIVINESENSE_EVOLUTION_ENABLED", "true")

		parrot, err := NewEvolutionParrot("/src", 1, "test-session", nil, false)
		if err != nil {
			t.Fatalf("NewEvolutionParrot() error = %v", err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		err = parrot.ExecuteWithCallback(ctx, "test input", nil, nil)
		if err != nil {
			t.Logf("ExecuteWithCallback failed (expected in test env): %v", err)
		}

		// After successful execution, session should be active
		// This requires actual CC to be available
	*/
}

// BenchmarkEvolutionParrotSelfDescribe benchmarks self-description generation.
func BenchmarkEvolutionParrotSelfDescribe(b *testing.B) {
	parrot, _ := NewEvolutionParrot("/src", 1, "test-session", nil, false)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = parrot.SelfDescribe()
	}
}

// TestNewEvolutionParrotTimeout tests custom timeout configuration.
func TestNewEvolutionParrotTimeout(t *testing.T) {
	t.Setenv("DIVINESENSE_EVOLUTION_ENABLED", "true")

	start := time.Now()
	parrot, err := NewEvolutionParrot("/src", 1, "test-session", nil, false)
	elapsed := time.Since(start)

	if err != nil {
		// CLI not found is acceptable in test environment
		return
	}

	if parrot == nil {
		t.Error("NewEvolutionParrot() should return non-nil parrot when CLI exists")
	}

	// Creation should be fast (doesn't execute CLI yet)
	if elapsed > 100*time.Millisecond {
		t.Errorf("NewEvolutionParrot() took too long: %v", elapsed)
	}
}
