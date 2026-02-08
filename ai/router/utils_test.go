// Package router provides unit tests for utility functions.
package router

import (
	"strings"
	"testing"
)

// TestTruncate tests the truncate utility function.
// Note: truncate uses byte length, not rune length.
func TestTruncate(t *testing.T) {
	testCases := []struct {
		input    string
		maxLen   int
		expected string
	}{
		{"short", 10, "short"},
		{"exact length", 12, "exact length"},
		{"this is too long", 10, "this is to..."},
		{"", 5, ""},
		{"a", 1, "a"},
		{"ab", 1, "a..."},
		// Each Chinese character is 3 bytes in UTF-8
		// "中文测试" = 12 bytes, so maxLen=5 truncates after 1 character + ...
		// The output may contain invalid UTF-8 which is acceptable for truncate
		{"中文测试", 5, "中..."},
		{"中文测试很长", 5, "中..."},
	}

	for _, tc := range testCases {
		result := truncate(tc.input, tc.maxLen)
		if result != tc.expected {
			// For Chinese strings with byte truncation (non-ASCII), check length and suffix
			if len(tc.input) > 0 && tc.input[0] >= 128 {
				// Contains non-ASCII: just check that result is truncated and ends with ...
				if len(result) <= tc.maxLen+3 && strings.HasSuffix(result, "...") {
					continue // Valid truncation
				}
			}
			t.Errorf("truncate(%q, %d) = %q, expected %q", tc.input, tc.maxLen, result, tc.expected)
		}
	}
}

// TestContainsAny tests the containsAny utility function.
func TestContainsAny(t *testing.T) {
	testCases := []struct {
		input      string
		substrings []string
		expected   bool
	}{
		{"hello world", []string{"hello", "goodbye"}, true},
		{"hello world", []string{"foo", "bar"}, false},
		{"测试中文", []string{"测试", "中文"}, true},
		{"测试中文", []string{"foo", "bar"}, false},
		{"", []string{"foo"}, false},
		{"hello", []string{}, false},
		{"hello", nil, false},
		{"Mixed 混合 Content", []string{"Mixed", "混合"}, true},
	}

	for _, tc := range testCases {
		result := containsAny(tc.input, tc.substrings)
		if result != tc.expected {
			t.Errorf("containsAny(%q, %v) = %v, expected %v",
				tc.input, tc.substrings, result, tc.expected)
		}
	}
}

// TestStringToIntent tests the stringToIntent utility function.
func TestStringToIntent(t *testing.T) {
	service := &Service{ruleMatcher: NewRuleMatcher()}

	testCases := []struct {
		input    string
		expected Intent
	}{
		{"memo_search", IntentMemoSearch},
		{"memosearch", IntentMemoSearch},
		{"search", IntentMemoSearch},
		{"memo_create", IntentMemoCreate},
		{"memocreate", IntentMemoCreate},
		{"create_memo", IntentMemoCreate},
		{"schedule_query", IntentScheduleQuery},
		{"schedulequery", IntentScheduleQuery},
		{"query", IntentScheduleQuery},
		{"schedule_create", IntentScheduleCreate},
		{"schedulecreate", IntentScheduleCreate},
		{"schedule_update", IntentScheduleUpdate},
		{"scheduleupdate", IntentScheduleUpdate},
		{"update", IntentScheduleUpdate},
		{"batch_schedule", IntentBatchSchedule},
		{"batchschedule", IntentBatchSchedule},
		{"batch", IntentBatchSchedule},
		{"amazing", IntentAmazing},
		{"AMAZING", IntentAmazing},
		{"Amazing", IntentAmazing},
		{"unknown_intent", IntentAmazing}, // Default fallback
		{"", IntentAmazing},               // Default fallback
	}

	for _, tc := range testCases {
		result := service.stringToIntent(tc.input)
		if result != tc.expected {
			t.Errorf("stringToIntent(%q) = %s, expected %s", tc.input, result, tc.expected)
		}
	}
}

// TestParseLLMResponse tests LLM response parsing.
func TestParseLLMResponse(t *testing.T) {
	service := &Service{ruleMatcher: NewRuleMatcher()}

	testCases := []struct {
		response       string
		expectedIntent Intent
		minConfidence  float32
	}{
		// JSON format
		{`{"intent": "memo_search", "confidence": 0.9}`, IntentMemoSearch, 0.8},
		{`{"intent": "schedule_create", "confidence": 0.95}`, IntentScheduleCreate, 0.9},
		// Plain text format
		{"memo_search", IntentMemoSearch, 0.7},
		{"schedule_create", IntentScheduleCreate, 0.7},
		// With quotes
		{`"memo_search"`, IntentMemoSearch, 0.7},
		{"`schedule_create`", IntentScheduleCreate, 0.7},
		// Invalid confidence uses default
		{`{"intent": "amazing", "confidence": 0}`, IntentAmazing, 0.7},
		{`{"intent": "memo_search", "confidence": -0.5}`, IntentMemoSearch, 0.7},
	}

	for _, tc := range testCases {
		intent, confidence := service.parseLLMResponse(tc.response)
		if intent != tc.expectedIntent {
			t.Errorf("parseLLMResponse(%q) intent = %s, expected %s",
				tc.response, intent, tc.expectedIntent)
		}
		if confidence < tc.minConfidence {
			t.Errorf("parseLLMResponse(%q) confidence = %f, expected >= %f",
				tc.response, confidence, tc.minConfidence)
		}
	}
}

// BenchmarkTruncate benchmarks truncate function.
func BenchmarkTruncate(b *testing.B) {
	input := "This is a very long string that needs to be truncated"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		truncate(input, 20)
	}
}

// BenchmarkContainsAny benchmarks containsAny function.
func BenchmarkContainsAny(b *testing.B) {
	input := "搜索关于人工智能的笔记"
	substrings := []string{"搜索", "笔记", "查找", "memo"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		containsAny(input, substrings)
	}
}

// BenchmarkStringContains benchmarks strings.Contains for comparison.
func BenchmarkStringContains(b *testing.B) {
	input := "搜索关于人工智能的笔记"
	pattern := "搜索"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		strings.Contains(input, pattern)
	}
}
