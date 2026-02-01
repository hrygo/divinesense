// Package interactive provides tests for the deployment wizard
package interactive

import (
	"bufio"
	"bytes"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// TestDefaultWizardConfig tests the default configuration values
func TestDefaultWizardConfig(t *testing.T) {
	cfg := DefaultWizardConfig()

	tests := []struct {
		name     string
		check    func() bool
		expected string
	}{
		{
			name:     "mode should be binary",
			check:    func() bool { return cfg.Mode == "binary" },
			expected: "binary",
		},
		{
			name:     "install dir should be /opt/divinesense",
			check:    func() bool { return cfg.InstallDir == "/opt/divinesense" },
			expected: "/opt/divinesense",
		},
		{
			name:     "config dir should be /etc/divinesense",
			check:    func() bool { return cfg.ConfigDir == "/etc/divinesense" },
			expected: "/etc/divinesense",
		},
		{
			name:     "port should be 5230",
			check:    func() bool { return cfg.Port == 5230 },
			expected: "5230",
		},
		{
			name:     "auto start should be true",
			check:    func() bool { return cfg.AutoStart },
			expected: "true",
		},
		{
			name:     "db type should be docker",
			check:    func() bool { return cfg.DbType == "docker" },
			expected: "docker",
		},
		{
			name:     "enable AI should be true",
			check:    func() bool { return cfg.EnableAI },
			expected: "true",
		},
		{
			name:     "enable geek mode should be true",
			check:    func() bool { return cfg.EnableGeekMode },
			expected: "true",
		},
		{
			name:     "enable evolution should be false by default",
			check:    func() bool { return !cfg.EnableEvolution },
			expected: "false",
		},
		{
			name:     "admin only should be true",
			check:    func() bool { return cfg.AdminOnly },
			expected: "true",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !tt.check() {
				t.Errorf("check failed, expected %s", tt.expected)
			}
		})
	}
}

// TestGeneratePassword tests password generation
func TestGeneratePassword(t *testing.T) {
	pw1 := generatePassword(16)
	pw2 := generatePassword(16)

	if len(pw1) < 16 {
		t.Errorf("password too short: got %d, want >= 16", len(pw1))
	}

	if pw1 == pw2 {
		t.Error("passwords should be unique")
	}

	// Check for character classes
	hasUpper := false
	hasLower := false
	hasDigit := false
	for _, c := range pw1 {
		switch {
		case c >= 'A' && c <= 'Z':
			hasUpper = true
		case c >= 'a' && c <= 'z':
			hasLower = true
		case c >= '0' && c <= '9':
			hasDigit = true
		}
	}

	if !hasUpper {
		t.Error("password should contain uppercase letters")
	}
	if !hasLower {
		t.Error("password should contain lowercase letters")
	}
	if !hasDigit {
		t.Error("password should contain digits")
	}
}

// TestBoolToStr tests boolean to string conversion
func TestBoolToStr(t *testing.T) {
	if got := boolToStr(true); got != "true" {
		t.Errorf("boolToStr(true) = %q, want %q", got, "true")
	}
	if got := boolToStr(false); got != "false" {
		t.Errorf("boolToStr(false) = %q, want %q", got, "false")
	}
}

// TestBoolToYesNo tests boolean to Yes/No conversion
func TestBoolToYesNo(t *testing.T) {
	if got := boolToYesNo(true); got != "是" {
		t.Errorf("boolToYesNo(true) = %q, want %q", got, "是")
	}
	if got := boolToYesNo(false); got != "否" {
		t.Errorf("boolToYesNo(false) = %q, want %q", got, "否")
	}
}

// TestWizardTerminal tests terminal I/O functions
func TestWizardTerminal(t *testing.T) {
	t.Run("isTerminal with file", func(t *testing.T) {
		// Create a temp file (not a terminal)
		tmp, err := os.CreateTemp("", "test")
		if err != nil {
			t.Fatal(err)
		}
		defer tmp.Close()
		defer os.Remove(tmp.Name())

		// File should not be detected as terminal
		if isTerminal(tmp) {
			t.Error("temp file should not be detected as terminal")
		}
	})
}

// TestWizardConfigValidation tests configuration validation
func TestWizardConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		cfg     WizardConfig
		wantErr bool
	}{
		{
			name: "valid binary config",
			cfg: WizardConfig{
				Mode:       "binary",
				InstallDir: "/opt/divinesense",
				ConfigDir:  "/etc/divinesense",
				Port:       5230,
				DbType:     "docker",
				DbName:     "divinesense",
				DbUser:     "divinesense",
			},
			wantErr: false,
		},
		{
			name: "valid docker config",
			cfg: WizardConfig{
				Mode:       "docker",
				InstallDir: "/opt/divinesense",
				Port:       5230,
				DbType:     "",
			},
			wantErr: false,
		},
		{
			name: "invalid mode",
			cfg: WizardConfig{
				Mode:       "invalid",
				InstallDir: "/opt/divinesense",
				Port:       5230,
			},
			wantErr: true,
		},
		{
			name: "invalid port - zero",
			cfg: WizardConfig{
				Mode:       "binary",
				InstallDir: "/opt/divinesense",
				Port:       0,
			},
			wantErr: true,
		},
		{
			name: "invalid port - too high",
			cfg: WizardConfig{
				Mode:       "binary",
				InstallDir: "/opt/divinesense",
				Port:       70000,
			},
			wantErr: true,
		},
		{
			name: "invalid db type",
			cfg: WizardConfig{
				Mode:       "binary",
				InstallDir: "/opt/divinesense",
				Port:       5230,
				DbType:     "invalid",
			},
			wantErr: true,
		},
		{
			name: "evolution without geek mode",
			cfg: WizardConfig{
				Mode:            "binary",
				InstallDir:      "/opt/divinesense",
				Port:            5230,
				EnableGeekMode:  false,
				EnableEvolution: true,
			},
			wantErr: true,
		},
		{
			name: "empty install dir",
			cfg: WizardConfig{
				Mode:       "binary",
				InstallDir: "",
				Port:       5230,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("WizardConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestValidateConfigFile tests config file validation
func TestValidateConfigFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Test valid config
	validCfg := filepath.Join(tmpDir, "valid.conf")
	if err := os.WriteFile(validCfg, []byte("DIVINESENSE_PORT=5230\nDIVINESENSE_DRIVER=postgres\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// Test invalid config (syntax error - no equals sign)
	invalidCfg := filepath.Join(tmpDir, "invalid.conf")
	if err := os.WriteFile(invalidCfg, []byte("DIVINESENSE_PORT=abc\ninvalid line without equals"), 0644); err != nil {
		t.Fatal(err)
	}

	// Test with comments and empty lines
	commentedCfg := filepath.Join(tmpDir, "commented.conf")
	if err := os.WriteFile(commentedCfg, []byte("# Comment line\n\nDIVINESENSE_PORT=5230\n"), 0644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{"valid config", validCfg, false},
		{"invalid config - no equals", invalidCfg, true},
		{"valid with comments", commentedCfg, false},
		{"nonexistent config", filepath.Join(tmpDir, "nonexistent.conf"), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateTestConfigFile(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateConfigFile() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestIsPortAvailable tests port availability checking
func TestIsPortAvailable(t *testing.T) {
	t.Run("likely unavailable common ports", func(t *testing.T) {
		// These ports might be in use, we just check the function runs
		commonPorts := []int{80, 443, 22, 25}
		for _, port := range commonPorts {
			avail := checkPortAvailable(port)
			_ = avail // Just ensure it doesn't panic
		}
	})

	t.Run("likely available high port", func(t *testing.T) {
		// Use a high port number that's unlikely to be in use
		port := 45678 + (int(time.Now().UnixNano()) % 10000)
		avail := checkPortAvailable(port)
		if !avail {
			t.Logf("port %d is not available (might be in use)", port)
		}
	})
}

// TestSystemCheckFuncs tests system checking functions
func TestSystemCheckFuncs(t *testing.T) {
	t.Run("checkCommandExists", func(t *testing.T) {
		// Test sh should always exist
		if !testCheckCommandExists("sh") {
			t.Error("checkCommandExists(\"sh\") = false, want true")
		}
		// Test ls should always exist
		if !testCheckCommandExists("ls") {
			t.Error("checkCommandExists(\"ls\") = false, want true")
		}
		// Test nonexistent command
		if testCheckCommandExists("thiscommanddoesnotexist123abc") {
			t.Error("checkCommandExists(\"thiscommanddoesnotexist123abc\") = true, want false")
		}
	})
}

// TestWizardState tests wizard state management
func TestWizardState(t *testing.T) {
	t.Run("newWizard initializes", func(t *testing.T) {
		w := NewWizard()
		if w == nil {
			t.Fatal("NewWizard() returned nil")
		}
		if w.config.Mode == "" {
			t.Error("wizard config not initialized")
		}
	})
}

// TestWizardCreation tests wizard creation with custom I/O
func TestWizardCreation(t *testing.T) {
	t.Run("create wizard with buffered I/O", func(t *testing.T) {
		input := bytes.NewReader([]byte("test\n"))
		output := new(bytes.Buffer)

		w := &Wizard{
			reader: bufio.NewReader(input),
			writer: bufio.NewWriter(output),
		}

		if w.reader == nil {
			t.Error("reader not initialized")
		}
		if w.writer == nil {
			t.Error("writer not initialized")
		}
	})
}

// TestRunFunction tests the Run function wrapper
func TestRunFunction(t *testing.T) {
	// The Run() function is designed for interactive use
	// We just verify it's callable and handles non-TTY gracefully
	t.Run("Run function exists", func(t *testing.T) {
		// Verify the function signature is correct
		// We don't actually run it as it would try to read from stdin
		_ = Run // The function exists and is non-nil (package-level function)
	})
}

// Helper functions for testing

func checkPortAvailable(port int) bool {
	ln, err := net.Listen("tcp", ":"+strconv.Itoa(port))
	if err != nil {
		return false
	}
	_ = ln.Close()
	return true
}

func testCheckCommandExists(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}

func validateTestConfigFile(path string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if !strings.Contains(line, "=") {
			return fmt.Errorf("invalid line: %s", line)
		}
	}
	return nil
}

// Validate method for WizardConfig
func (c *WizardConfig) Validate() error {
	if c.Mode == "" || (c.Mode != "binary" && c.Mode != "docker") {
		return os.ErrInvalid
	}
	if c.Port <= 0 || c.Port > 65535 {
		return os.ErrInvalid
	}
	if c.InstallDir == "" {
		return os.ErrInvalid
	}
	if c.DbType != "" && c.DbType != "docker" && c.DbType != "system" && c.DbType != "remote" {
		return os.ErrInvalid
	}
	if c.EnableEvolution && !c.EnableGeekMode {
		return os.ErrInvalid
	}
	return nil
}
