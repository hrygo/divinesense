package agent

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestCCRunnerValidateConfig tests config validation.
func TestCCRunnerValidateConfig(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *CCRunnerConfig
		wantErr bool
	}{
		{
			name: "valid config",
			cfg: &CCRunnerConfig{
				Mode:      "geek",
				WorkDir:   "/tmp/test",
				SessionID: "test-session",
				UserID:    1,
			},
			wantErr: false,
		},
		{
			name: "missing mode",
			cfg: &CCRunnerConfig{
				WorkDir:   "/tmp/test",
				SessionID: "test-session",
				UserID:    1,
			},
			wantErr: true,
		},
		{
			name: "missing work dir",
			cfg: &CCRunnerConfig{
				Mode:      "geek",
				SessionID: "test-session",
				UserID:    1,
			},
			wantErr: true,
		},
		{
			name: "missing session id",
			cfg: &CCRunnerConfig{
				Mode:    "geek",
				WorkDir: "/tmp/test",
				UserID:  1,
			},
			wantErr: true,
		},
		{
			name: "missing user id",
			cfg: &CCRunnerConfig{
				Mode:      "geek",
				WorkDir:   "/tmp/test",
				SessionID: "test-session",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &CCRunner{}
			err := r.validateConfig(tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestCCRunnerIsFirstCall tests session first call detection.
func TestCCRunnerIsFirstCall(t *testing.T) {
	tmpDir := t.TempDir()
	sessionDir := filepath.Join(tmpDir, ".claude", "sessions", "test-session")

	r := &CCRunner{}

	// First call should return true
	if !r.isFirstCall(sessionDir) {
		t.Error("isFirstCall() should return true for non-existent session directory")
	}

	// Create session directory
	if err := os.MkdirAll(sessionDir, 0755); err != nil {
		t.Fatalf("failed to create session directory: %v", err)
	}

	// Second call should return false
	if r.isFirstCall(sessionDir) {
		t.Error("isFirstCall() should return false for existing session directory")
	}
}

// TestBuildSystemPrompt tests system prompt generation.
func TestBuildSystemPrompt(t *testing.T) {
	deviceContext := `{"userAgent":"Mozilla/5.0","isMobile":false,"screenWidth":1920,"screenHeight":1080,"language":"zh-CN"}`

	prompt := buildSystemPrompt("/workspace", "session-123", 42, deviceContext)

	// Check for key elements (using bold format)
	requiredStrings := []string{
		"**User ID**: 42",
		"Desktop (1920x1080)",
		"Language: zh-CN",
		"**Workspace**: /workspace",
		"**Session**: session-123",
		"Non-interactive headless",
	}

	for _, s := range requiredStrings {
		if !strings.Contains(prompt, s) {
			t.Errorf("buildSystemPrompt() missing required string: %s\nGot:\n%s", s, prompt)
		}
	}

	// Check that File Output section is NOT in base prompt
	if strings.Contains(prompt, "File Output") {
		t.Error("buildSystemPrompt() should not contain File Output section (Geek-specific)")
	}
}

// TestGeekModeName tests GeekMode name.
func TestGeekModeName(t *testing.T) {
	mode := NewGeekMode("/src")
	if got := mode.Name(); got != "geek" {
		t.Errorf("GeekMode.Name() = %v, want %v", got, "geek")
	}
}

// TestGeekModeGetWorkDir tests GeekMode work directory.
func TestGeekModeGetWorkDir(t *testing.T) {
	mode := NewGeekMode("/src")

	// Mock home directory for testing
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Skipf("cannot get home directory: %v", err)
	}

	expectedDir := filepath.Join(homeDir, ".divinesense", "claude", "user_123")
	if got := mode.GetWorkDir(123); got != expectedDir {
		t.Errorf("GeekMode.GetWorkDir() = %v, want %v", got, expectedDir)
	}
}

// TestGeekModeCheckPermission tests GeekMode permission check.
func TestGeekModeCheckPermission(t *testing.T) {
	mode := NewGeekMode("/src")

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

// TestGeekModeBuildSystemPrompt tests GeekMode system prompt includes File Output.
func TestGeekModeBuildSystemPrompt(t *testing.T) {
	mode := NewGeekMode("/src")
	cfg := &CCRunnerConfig{
		WorkDir:   "/workspace",
		SessionID: "session-123",
		UserID:    42,
	}

	prompt := mode.BuildSystemPrompt(cfg)

	// Should have base prompt elements
	if !strings.Contains(prompt, "**User ID**: 42") {
		t.Error("GeekMode.BuildSystemPrompt() missing base prompt")
	}

	// Should have File Output section (Geek-specific)
	if !strings.Contains(prompt, "File Output") {
		t.Error("GeekMode.BuildSystemPrompt() should contain File Output section")
	}

	if !strings.Contains(prompt, "announce the filename") {
		t.Error("GeekMode.BuildSystemPrompt() should contain file announcement instruction")
	}
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

	// Example for integration testing with real store:
	// mode := NewEvolutionMode(&EvolutionModeConfig{
	// 	SourceDir: "/src",
	// 	AdminOnly: true,
	// 	Store:     testStore,
	// })
	//
	// Test admin user access
	// err := mode.CheckPermission(context.Background(), adminUserID)
	// if err != nil {
	// 	t.Errorf("admin should have access: %v", err)
	// }
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
	cfg := &CCRunnerConfig{
		WorkDir: "/src",
	}

	prompt := mode.BuildSystemPrompt(cfg)

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

// TestStreamMessageGetContentBlocks tests content block extraction.
func TestStreamMessageGetContentBlocks(t *testing.T) {
	tests := []struct {
		name     string
		msg      StreamMessage
		wantLen  int
		wantText string
	}{
		{
			name: "direct content",
			msg: StreamMessage{
				Content: []ContentBlock{
					{Type: "text", Text: "hello"},
				},
			},
			wantLen:  1,
			wantText: "hello",
		},
		{
			name: "nested message content",
			msg: StreamMessage{
				Message: &AssistantMessage{
					Content: []ContentBlock{
						{Type: "text", Text: "world"},
					},
				},
			},
			wantLen:  1,
			wantText: "world",
		},
		{
			name: "nested takes priority",
			msg: StreamMessage{
				Content: []ContentBlock{
					{Type: "text", Text: "direct"},
				},
				Message: &AssistantMessage{
					Content: []ContentBlock{
						{Type: "text", Text: "nested"},
					},
				},
			},
			wantLen:  1,
			wantText: "nested",
		},
		{
			name:    "empty",
			msg:     StreamMessage{},
			wantLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			blocks := tt.msg.GetContentBlocks()
			if len(blocks) != tt.wantLen {
				t.Errorf("GetContentBlocks() len = %v, want %v", len(blocks), tt.wantLen)
			}
			if tt.wantText != "" && len(blocks) > 0 {
				if blocks[0].Text != tt.wantText {
					t.Errorf("GetContentBlocks()[0].Text = %v, want %v", blocks[0].Text, tt.wantText)
				}
			}
		})
	}
}

// TestNewCCRunnerWithoutCLI tests CCRunner creation when CLI is not found.
func TestNewCCRunnerWithoutCLI(t *testing.T) {
	// Save original PATH
	origPath := os.Getenv("PATH")
	defer os.Setenv("PATH", origPath)

	// Set empty PATH to simulate CLI not found
	os.Setenv("PATH", "")

	_, err := NewCCRunner(10*time.Second, nil)
	if err == nil {
		t.Error("NewCCRunner() should return error when CLI is not found")
	}

	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("NewCCRunner() error should mention 'not found', got: %v", err)
	}
}
