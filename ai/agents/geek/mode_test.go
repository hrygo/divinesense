package geek

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestGeekModeName tests GeekMode name.
func TestGeekModeName(t *testing.T) {
	mode := NewGeekMode("")
	if got := mode.Name(); got != "geek" {
		t.Errorf("GeekMode.Name() = %v, want %v", got, "geek")
	}
}

// TestGeekModeGetWorkDir tests GeekMode work directory.
func TestGeekModeGetWorkDir(t *testing.T) {
	t.Run("default behavior", func(t *testing.T) {
		mode := NewGeekMode("")
		homeDir, err := os.UserHomeDir()
		if err != nil {
			t.Skipf("cannot get home directory: %v", err)
		}
		expectedDir := filepath.Join(homeDir, ".divinesense", "claude", "user_123")
		if got := mode.GetWorkDir(123); got != expectedDir {
			t.Errorf("GeekMode.GetWorkDir() (default) = %v, want %v", got, expectedDir)
		}
	})

	t.Run("custom base dir", func(t *testing.T) {
		customDir := "/custom/path"
		mode := NewGeekMode(customDir)
		expectedDir := filepath.Join(customDir, "user_123")
		if got := mode.GetWorkDir(123); got != expectedDir {
			t.Errorf("GeekMode.GetWorkDir() (custom) = %v, want %v", got, expectedDir)
		}
	})
}

// TestGeekModeCheckPermission tests GeekMode permission check.
func TestGeekModeCheckPermission(t *testing.T) {
	mode := NewGeekMode("")

	tests := []struct {
		name    string
		userID  int32
		wantErr bool
	}{
		{
			name:    "valid user ID",
			userID:  1,
			wantErr: false,
		},
		{
			name:    "zero user ID",
			userID:  0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := mode.CheckPermission(context.Background(), tt.userID)
			if (err != nil) != tt.wantErr {
				t.Errorf("GeekMode.CheckPermission() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestGeekModeBuildSystemPrompt tests GeekMode system prompt includes Output Behavior.
func TestGeekModeBuildSystemPrompt(t *testing.T) {
	mode := NewGeekMode("")

	// Import agentpkg to use CCRunnerConfig
	// We need to reference the type, so we use a struct literal
	type TestConfig struct {
		WorkDir   string
		SessionID string
		UserID    int32
	}

	cfg := &TestConfig{
		WorkDir:   "/workspace",
		SessionID: "session-123",
		UserID:    42,
	}

	// Call BuildSystemPrompt - we can't use the agentpkg.CCRunnerConfig directly
	// due to import cycle, but the method works with any struct that has
	// the required fields. We'll test the actual method indirectly by
	// creating a real CCRunnerConfig in a separate integration test.
	_ = cfg // TODO: create proper test without import cycle
	_ = mode
}

// TestEvolutionModeName tests EvolutionMode name.
func TestEvolutionModeName(t *testing.T) {
	mode := NewEvolutionMode(&EvolutionModeConfig{
		SourceDir: "/src",
		AdminOnly: true,
	})
	if got := mode.Name(); got != "evolution" {
		t.Errorf("EvolutionMode.Name() = %v, want %v", got, "evolution")
	}
}

// TestEvolutionModeGetWorkDir tests EvolutionMode work directory.
func TestEvolutionModeGetWorkDir(t *testing.T) {
	sourceDir := "/project/src"
	mode := NewEvolutionMode(&EvolutionModeConfig{
		SourceDir: sourceDir,
		AdminOnly: true,
	})

	if got := mode.GetWorkDir(123); got != sourceDir {
		t.Errorf("EvolutionMode.GetWorkDir() = %v, want %v", got, sourceDir)
	}
}

// TestEvolutionModeCheckPermission tests EvolutionMode permission check.
func TestEvolutionModeCheckPermission(t *testing.T) {
	// Enable evolution mode for testing
	t.Setenv("DIVINESENSE_EVOLUTION_ENABLED", "true")

	t.Run("without store - deny by default", func(t *testing.T) {
		mode := NewEvolutionMode(&EvolutionModeConfig{
			SourceDir: "/src",
			AdminOnly: true,
			Store:     nil, // No store configured
		})

		// Should fail because no store means no admin verification
		err := mode.CheckPermission(context.Background(), 1)
		if err == nil {
			t.Error("EvolutionMode.CheckPermission() should return error when store is nil and AdminOnly=true")
		}
	})

	t.Run("without store when admin check disabled", func(t *testing.T) {
		mode := NewEvolutionMode(&EvolutionModeConfig{
			SourceDir: "/src",
			AdminOnly: false, // Admin check disabled
			Store:     nil,
		})

		// Should succeed because AdminOnly=false skips admin check
		err := mode.CheckPermission(context.Background(), 1)
		if err != nil {
			t.Errorf("EvolutionMode.CheckPermission() should succeed when AdminOnly=false, got: %v", err)
		}
	})
}

// TestEvolutionModeIsAdmin tests the isAdmin method with store integration.
// This test is skipped in normal test runs as it requires a valid store.
func TestEvolutionModeIsAdmin(t *testing.T) {
	t.Skip("requires valid store connection - integration test only")
}

// TestEvolutionModeCheckPermissionDisabled tests EvolutionMode when disabled.
func TestEvolutionModeCheckPermissionDisabled(t *testing.T) {
	// Ensure evolution mode is disabled
	t.Setenv("DIVINESENSE_EVOLUTION_ENABLED", "false")

	mode := NewEvolutionMode(&EvolutionModeConfig{
		SourceDir: "/src",
		AdminOnly: true,
	})

	err := mode.CheckPermission(context.Background(), 1)
	if err == nil {
		t.Error("EvolutionMode.CheckPermission() should return error when disabled")
	}

	if !strings.Contains(err.Error(), "disabled") {
		t.Errorf("EvolutionMode.CheckPermission() error should mention 'disabled', got: %v", err)
	}
}

// TestEvolutionModeBuildSystemPrompt tests EvolutionMode system prompt.
func TestEvolutionModeBuildSystemPrompt(t *testing.T) {
	mode := NewEvolutionMode(&EvolutionModeConfig{
		SourceDir: "/src",
		AdminOnly: true,
	})

	// Test with a minimal config that has the fields we need
	type MinimalConfig struct {
		WorkDir string
	}

	prompt := mode.BuildSystemPrompt(nil) // mode doesn't actually use the config for EvolutionMode

	// Should have Evolution Mode specific content
	if !strings.Contains(prompt, "Evolution Mode") {
		t.Error("EvolutionMode.BuildSystemPrompt() should mention Evolution Mode")
	}

	if !strings.Contains(prompt, "CLAUDE.md") {
		t.Error("EvolutionMode.BuildSystemPrompt() should mention CLAUDE.md")
	}

	if !strings.Contains(prompt, "PR") {
		t.Error("EvolutionMode.BuildSystemPrompt() should mention PR")
	}

	// Should NOT have File Output section (not relevant for evolution)
	if strings.Contains(prompt, "File Output") {
		t.Error("EvolutionMode.BuildSystemPrompt() should not contain File Output section")
	}
}
