// Package strutil provides string utility functions for the ai package.
package strutil

// Truncate truncates a string to a maximum length.
// Uses rune-level truncation to ensure Unicode safety (correct handling of multi-byte characters like Chinese).
func Truncate(s string, maxLen int) string {
	if s == "" {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen]) + "..."
}
