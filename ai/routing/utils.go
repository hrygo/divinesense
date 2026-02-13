package routing

import (
	"strings"

	"github.com/hrygo/divinesense/ai/internal/strutil"
)

// truncate truncates a string to maxLen characters (Unicode-safe).
// This is a utility function used across the router package.
func truncate(s string, maxLen int) string {
	return strutil.Truncate(s, maxLen)
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
