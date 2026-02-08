package router

import "strings"

// truncate truncates a string to maxLen characters.
// This is a utility function used across the router package.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// containsAny checks if s contains any of the patterns.
func containsAny(s string, patterns []string) bool {
	for _, p := range patterns {
		if strings.Contains(s, p) {
			return true
		}
	}
	return false
}
