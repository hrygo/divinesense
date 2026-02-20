package agent

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/hrygo/divinesense/ai/agents/runner"
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
			// Convert CCRunnerConfig to runner.Config
			cfg := &runner.Config{
				Mode:      tt.cfg.Mode,
				WorkDir:   tt.cfg.WorkDir,
				SessionID: tt.cfg.SessionID,
				UserID:    tt.cfg.UserID,
			}
			err := r.ValidateConfig(cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestBuildSystemPrompt tests system prompt generation.
func TestBuildSystemPrompt(t *testing.T) {
	deviceContext := `{"userAgent":"Mozilla/5.0","isMobile":false,"screenWidth":1920,"screenHeight":1080,"language":"zh-CN"}`

	prompt := BuildSystemPrompt("/workspace", "session-123", 42, deviceContext)

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
			t.Errorf("BuildSystemPrompt() missing required string: %s\nGot:\n%s", s, prompt)
		}
	}

	// Check that File Output section is NOT in base prompt
	if strings.Contains(prompt, "File Output") {
		t.Error("BuildSystemPrompt() should not contain File Output section (Geek-specific)")
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
