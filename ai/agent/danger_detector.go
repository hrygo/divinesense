package agent

import (
	"bufio"
	"fmt"
	"log/slog"
	"os"
	"regexp"
	"strings"
	"sync"
)

// Constants for danger detector logging and display limits.
const (
	// Maximum input length to log (prevents log flooding)
	MaxInputLogLength = 50
	// Maximum pattern match length to log
	MaxPatternLogLength = 100
	// Maximum command display length for UI
	MaxDisplayLength = 100
)

// DangerLevel represents the severity level of a dangerous operation.
type DangerLevel int

const (
	// DangerLevelCritical can cause irreversible data loss or system damage.
	DangerLevelCritical DangerLevel = iota
	// DangerLevelHigh can cause significant data loss or system impact.
	DangerLevelHigh
	// DangerLevelModerate may cause unintended changes.
	DangerLevelModerate
)

// DangerPattern defines a dangerous command pattern to detect.
type DangerPattern struct {
	Pattern     *regexp.Regexp
	Description string
	Level       DangerLevel
	Category    string // "file_delete", "system", "network", "permission"
}

// DangerBlockEvent represents a detected dangerous operation that was blocked.
type DangerBlockEvent struct {
	Operation      string      `json:"operation"`
	Reason         string      `json:"reason"`
	PatternMatched string      `json:"pattern_matched"`
	Level          DangerLevel `json:"level"`
	Category       string      `json:"category"`
	BypassAllowed  bool        `json:"bypass_allowed"`
	Suggestions    []string    `json:"suggestions,omitempty"`
}

// DangerDetector detects potentially dangerous commands before execution.
// Implements security layer per spec 6: "Frontend Confirmation + Git Recovery".
type DangerDetector struct {
	patterns []*DangerPattern
	logger   *slog.Logger
	mu       sync.RWMutex

	// Allowlist of paths that are safe to operate on (e.g., project directory)
	allowPaths []string

	// Whether bypass mode is enabled (admin only, Evolution mode)
	bypassEnabled bool
}

// NewDangerDetector creates a new danger detector with default patterns.
func NewDangerDetector(logger *slog.Logger) *DangerDetector {
	if logger == nil {
		logger = slog.Default()
	}

	dd := &DangerDetector{
		logger:   logger,
		patterns: make([]*DangerPattern, 0),
	}

	// Load default dangerous patterns
	dd.loadDefaultPatterns()

	return dd
}

// loadDefaultPatterns initializes the built-in dangerous command patterns.
func (dd *DangerDetector) loadDefaultPatterns() {
	patterns := []struct {
		pattern     string
		description string
		level       DangerLevel
		category    string
	}{
		// Critical: File deletion
		{`rm\s+-rf\s+?/`, "Delete root directory", DangerLevelCritical, "file_delete"},
		{`rm\s+-rf\s+?\*/\*`, "Delete all files recursively", DangerLevelCritical, "file_delete"},
		{`rm\s+-rf\s+?[~\w/]+/\*`, "Delete directory contents", DangerLevelHigh, "file_delete"},
		{`rm\s+-[rf]+\s+?/`, "Force delete from root", DangerLevelCritical, "file_delete"},
		{`rmdir\s+?/`, "Remove root directory", DangerLevelCritical, "file_delete"},
		{`del\s+?/[^/\s]*`, "Windows delete root", DangerLevelCritical, "file_delete"},

		// Critical: Filesystem operations
		{`mkfs\.\w+`, "Format filesystem", DangerLevelCritical, "system"},
		{`dd\s+if=/dev/zero`, "Wipe disk with zeros", DangerLevelCritical, "system"},
		{`dd\s+if=/dev/random`, "Wipe disk with random data", DangerLevelCritical, "system"},
		{`dd\s+of=/dev/`, "Write directly to device", DangerLevelCritical, "system"},
		{`wipefs\s+`, "Wipe filesystem signature", DangerLevelCritical, "system"},

		// Critical: Kernel/Process manipulation
		{`:\(.*\)\{.*\}\|`, "Fork bomb pattern", DangerLevelCritical, "system"},
		{`kill\s+-9\s+-1`, "Kill all processes", DangerLevelCritical, "system"},
		{`pkill\s+-9`, "Kill processes by name", DangerLevelHigh, "system"},
		{`killall\s+-9`, "Kill all processes by name", DangerLevelHigh, "system"},

		// High: Destructive overwrites
		{`>\s+/`, "Overwrite root file", DangerLevelCritical, "file_delete"},
		{`>\s+/dev/`, "Write to device file", DangerLevelHigh, "system"},
		{`echo\s+.*>\s+/`, "Echo to root path", DangerLevelHigh, "file_delete"},
		{`truncate\s+-s\s+0\s+/`, "Truncate root file", DangerLevelHigh, "file_delete"},

		// High: System configuration
		{`chmod\s+000\s+/`, "Remove all permissions from root", DangerLevelCritical, "permission"},
		{`chmod\s+-R\s+000\s+`, "Recursively remove all permissions", DangerLevelHigh, "permission"},
		{`chown\s+-R\s+root:root\s+/`, "Change ownership of root recursively", DangerLevelHigh, "permission"},
		{`userdel\s+-r\s+`, "Delete user with home directory", DangerLevelHigh, "system"},

		// High: Network/Dangerous downloads
		{`curl\s+.*\|.*sh`, "Download and execute script via pipe", DangerLevelHigh, "network"},
		{`wget\s+.*\|.*sh`, "Download and execute script via pipe", DangerLevelHigh, "network"},
		{`curl\s+.*\|.*bash`, "Download and execute via bash", DangerLevelHigh, "network"},
		{`wget\s+.*\|.*bash`, "Download and execute via bash", DangerLevelHigh, "network"},
		{`sh\s+-c\s+.*\$\(.*curl`, "Execute downloaded content", DangerLevelHigh, "network"},
		{`sh\s+-c\s+.*\$\(.*wget`, "Execute downloaded content", DangerLevelHigh, "network"},

		// High: Package manipulation
		{`apt-get\s+remove\s+--purge\s+.*essential`, "Remove essential packages", DangerLevelHigh, "system"},
		{`apt\s+remove\s+.*essential`, "Remove essential packages", DangerLevelHigh, "system"},
		{`yum\s+remove\s+.*kernel`, "Remove kernel packages", DangerLevelHigh, "system"},
		{`dpkg\s+--remove\s+--force`, "Force remove packages", DangerLevelHigh, "system"},

		// High: Database operations
		{`DROP\s+DATABASE`, "SQL: Drop database", DangerLevelHigh, "database"},
		{`DELETE\s+FROM.*\bWHERE\b`, "SQL: Delete without proper WHERE", DangerLevelModerate, "database"},
		{`TRUNCATE\s+(TABLE|SCHEMA)`, "SQL: Truncate table/schema", DangerLevelHigh, "database"},
		{`rm\s+.*\.db`, "Delete database file", DangerLevelHigh, "database"},
		{`rm\s+.*\.sqlite`, "Delete SQLite database", DangerLevelHigh, "database"},

		// Moderate: Git operations (data loss potential)
		{`git\s+reset\s+--hard\s+HEAD`, "Reset to HEAD (loses uncommitted changes)", DangerLevelModerate, "git"},
		{`git\s+clean\s+-fd`, "Remove untracked files", DangerLevelModerate, "git"},
		{`git\s+branch\s+-D`, "Force delete branch", DangerLevelModerate, "git"},
		{`rm\s+-rf\s+.*\.git`, "Delete git repository", DangerLevelHigh, "git"},

		// Moderate: SSH/Remote execution
		{`ssh\s+.*\|.*rm`, "Remote delete command", DangerLevelHigh, "network"},
		{`scp\s+.*\s+/\s*$`, "Copy to root directory", DangerLevelHigh, "network"},
	}

	for _, p := range patterns {
		// Compile pattern with case-insensitive flag for flexibility
		// Add error handling to prevent panic from malformed regex (ReDoS prevention)
		re, err := regexp.Compile(`(?i)` + p.pattern)
		if err != nil {
			dd.logger.Warn("Failed to compile danger pattern - skipping",
				"pattern", p.pattern,
				"description", p.description,
				"error", err,
			)
			continue // Skip invalid patterns instead of panicking
		}
		dd.patterns = append(dd.patterns, &DangerPattern{
			Pattern:     re,
			Description: p.description,
			Level:       p.level,
			Category:    p.category,
		})
	}
}

// CheckInput checks if the input contains any dangerous operations.
// Returns a DangerBlockEvent if a dangerous operation is detected, nil otherwise.
func (dd *DangerDetector) CheckInput(input string) *DangerBlockEvent {
	dd.mu.RLock()
	defer dd.mu.RUnlock()

	// If bypass is enabled (admin/Evolution mode), skip checks
	if dd.bypassEnabled {
		dd.logger.Warn("Danger detection bypassed", "input", truncateString(input, MaxInputLogLength))
		return nil
	}

	// Check each pattern
	for _, pat := range dd.patterns {
		if pat.Pattern.MatchString(input) {
			dd.logger.Warn("Dangerous operation detected",
				"pattern", pat.Pattern.String(),
				"description", pat.Description,
				"level", pat.Level,
				"category", pat.Category,
				"input", truncateString(input, MaxPatternLogLength),
			)

			return &DangerBlockEvent{
				Operation:      extractCommand(input, pat.Pattern),
				Reason:         pat.Description,
				PatternMatched: pat.Pattern.String(),
				Level:          pat.Level,
				Category:       pat.Category,
				BypassAllowed:  false, // Default to no bypass
				Suggestions:    dd.getSuggestions(pat),
			}
		}
	}

	return nil
}

// extractCommand extracts the relevant command portion from the input using pre-compiled regex.
func extractCommand(input string, pattern *regexp.Regexp) string {
	// Find the command line containing the dangerous pattern
	scanner := bufio.NewScanner(strings.NewReader(input))
	for scanner.Scan() {
		line := scanner.Text()
		if pattern.MatchString(line) {
			// Truncate to reasonable length (use rune-aware truncateString for UTF-8 safety)
			return truncateString(line, MaxDisplayLength)
		}
	}
	// Fallback to truncated input (use existing truncateString from util.go)
	return truncateString(input, MaxDisplayLength)
}

// getSuggestions returns safe alternatives for the dangerous operation.
func (dd *DangerDetector) getSuggestions(pat *DangerPattern) []string {
	switch pat.Category {
	case "file_delete":
		return []string{
			"Use 'rm -i' for interactive deletion with confirmation",
			"Consider moving files to a temporary backup directory first",
			"Use 'git status' to check what would be affected",
		}
	case "git":
		return []string{
			"Use 'git status' to check current changes",
			"Use 'git stash' to temporarily save changes",
			"Consider 'git checkout -- <file>' for single file recovery",
		}
	case "network":
		return []string{
			"Download scripts to a temp directory first for review",
			"Use 'curl -sL <url> | less' to inspect before executing",
			"Verify the source and checksum before execution",
		}
	case "system":
		return []string{
			"Ensure you have a recent backup before proceeding",
			"Test commands in a container or VM first",
			"Review the command documentation carefully",
		}
	case "database":
		return []string{
			"Use 'BEGIN; <query>; ROLLBACK;' to test first",
			"Create a database backup before running DDL/DML",
			"Use WHERE clause carefully to limit scope",
		}
	default:
		return []string{
			"Consider if there's a safer alternative",
			"Ensure you have a backup before proceeding",
		}
	}
}

// SetAllowPaths sets the list of allowed safe paths.
func (dd *DangerDetector) SetAllowPaths(paths []string) {
	dd.mu.Lock()
	defer dd.mu.Unlock()
	dd.allowPaths = paths
	dd.logger.Debug("Danger detector allow paths updated", "paths", paths)
}

// SetBypassEnabled enables or disables bypass mode.
// When enabled, dangerous operations are NOT blocked (admin/Evolution mode only).
func (dd *DangerDetector) SetBypassEnabled(enabled bool) {
	dd.mu.Lock()
	defer dd.mu.Unlock()
	dd.bypassEnabled = enabled
	if enabled {
		dd.logger.Warn("Danger detector bypass ENABLED - use with caution!")
	} else {
		dd.logger.Info("Danger detector bypass disabled")
	}
}

// IsPathAllowed checks if a path is in the allowlist.
func (dd *DangerDetector) IsPathAllowed(path string) bool {
	dd.mu.RLock()
	defer dd.mu.RUnlock()

	for _, allowed := range dd.allowPaths {
		if strings.HasPrefix(path, allowed) {
			return true
		}
	}
	return false
}

// CheckFileAccess checks if file access is within allowed paths.
// Returns true if the access is safe (within allowed paths), false otherwise.
func (dd *DangerDetector) CheckFileAccess(filePath string) bool {
	// Clean the path
	filePath = os.ExpandEnv(filePath)
	if !strings.HasPrefix(filePath, "/") {
		// Relative path - check if it resolves to safe location
		cwd, err := os.Getwd()
		if err == nil {
			filePath = cwd + "/" + filePath
		}
	}

	return dd.IsPathAllowed(filePath)
}

// LoadCustomPatterns loads custom danger patterns from a file.
// File format: one pattern per line: "regex|description|level|category"
func (dd *DangerDetector) LoadCustomPatterns(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("failed to open custom patterns file: %w", err)
	}
	defer func() { _ = file.Close() }() //nolint:errcheck // file cleanup

	dd.mu.Lock()
	defer dd.mu.Unlock()

	scanner := bufio.NewScanner(file)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "|", 4)
		if len(parts) < 4 {
			dd.logger.Warn("Invalid custom pattern format", "line", lineNum, "content", line)
			continue
		}

		re, err := regexp.Compile(`(?i)` + parts[0])
		if err != nil {
			dd.logger.Warn("Failed to compile pattern", "line", lineNum, "pattern", parts[0], "error", err)
			continue
		}

		var level DangerLevel
		switch parts[2] {
		case "critical":
			level = DangerLevelCritical
		case "high":
			level = DangerLevelHigh
		case "moderate":
			level = DangerLevelModerate
		default:
			level = DangerLevelModerate
		}

		dd.patterns = append(dd.patterns, &DangerPattern{
			Pattern:     re,
			Description: parts[1],
			Level:       level,
			Category:    parts[3],
		})
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading patterns file: %w", err)
	}

	dd.logger.Info("Loaded custom danger patterns", "file", filename)
	return nil
}

// String returns a string representation of the danger level.
func (d DangerLevel) String() string {
	switch d {
	case DangerLevelCritical:
		return "critical"
	case DangerLevelHigh:
		return "high"
	case DangerLevelModerate:
		return "moderate"
	default:
		return "unknown"
	}
}
