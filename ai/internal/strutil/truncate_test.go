package strutil

import "testing"

func TestTruncate(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxLen   int
		expected string
	}{
		// Basic cases
		{"empty string", "", 10, ""},
		{"short string", "hello", 10, "hello"},
		{"exact length", "hello", 5, "hello"},
		{"needs truncation", "hello world", 5, "hello..."},
		{"single char", "a", 1, "a"},
		{"single char truncated", "ab", 1, "a..."},

		// Edge cases - negative/zero maxLen
		{"negative maxLen", "hello", -1, ""},
		{"zero maxLen", "hello", 0, ""},
		{"negative maxLen empty", "", -5, ""},

		// Unicode safety - multi-byte characters
		{"chinese exact", "中文测试", 4, "中文测试"},
		{"chinese truncated", "中文测试abc", 4, "中文测试..."},
		{"emoji", "hello 🎉 world", 8, "hello 🎉 ..."},
		{"mixed unicode", "a中b文c", 3, "a中b..."},

		// Edge cases
		{"maxLen 1", "abc", 1, "a..."},
		{"maxLen 1 unicode", "中文", 1, "中..."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Truncate(tt.input, tt.maxLen)
			if result != tt.expected {
				t.Errorf("Truncate(%q, %d) = %q, want %q", tt.input, tt.maxLen, result, tt.expected)
			}
		})
	}
}

func TestTruncateNoPanic(t *testing.T) {
	// Ensure Truncate never panics on edge cases
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Truncate panicked: %v", r)
		}
	}()

	// These should all return empty string without panicking
	_ = Truncate("test", -100)
	_ = Truncate("test", 0)
	_ = Truncate("", -1)
	_ = Truncate("", 0)
}
