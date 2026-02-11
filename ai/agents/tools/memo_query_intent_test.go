package tools

import (
	"strings"
	"testing"
	"time"
)

// TestMemoQueryClassifier_Classify tests the query intent classification logic.
func TestMemoQueryClassifier_Classify(t *testing.T) {
	classifier := NewMemoQueryClassifier()

	tests := []struct {
		name     string
		query    string
		expected MemoQueryIntent
	}{
		// List intent queries
		{"list all - wildcard", "*", IntentList},
		{"list all - chinese", "有什么笔记", IntentList},
		{"list all - full", "列出所有笔记", IntentList},
		{"list all - my memos", "我的笔记", IntentList},
		{"list all - show memo", "show all memos", IntentList},
		{"list all - empty", "", IntentList},
		{"list all - whitespace", "   ", IntentList},

		// Filter intent queries
		{"time filter - today", "今天的笔记", IntentFilter},
		{"time filter - yesterday", "昨天笔记", IntentFilter},
		{"time filter - this week", "本周笔记", IntentFilter},
		{"time filter - recent", "最近笔记", IntentFilter},

		// Keyword intent queries (short, specific terms)
		{"keyword - short", "Python", IntentKeyword},
		{"keyword - medium", "React", IntentKeyword},
		{"keyword - two chars", "Go", IntentKeyword},
		{"keyword - chinese", "数据库", IntentKeyword},

		// Semantic intent queries (complex phrases, questions)
		// Note: Short semantic queries may still be classified as keyword
		{"semantic - how to chinese", "如何部署服务", IntentSemantic},
		{"semantic - about chinese", "关于数据库设计", IntentSemantic},
		{"semantic - mention chinese", "笔记中提到的时间管理", IntentSemantic},
		{"semantic - english", "how to deploy database", IntentSemantic},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := classifier.Classify(tt.query)
			if result != tt.expected {
				t.Errorf("Classify(%q) = %v, want %v", tt.query, result, tt.expected)
			}
		})
	}
}

// TestMemoQueryClassifier_ExtractTags tests tag extraction from queries.
func TestMemoQueryClassifier_ExtractTags(t *testing.T) {
	classifier := NewMemoQueryClassifier()

	tests := []struct {
		name     string
		query    string
		expected []string
	}{
		{"no tags", "find notes about Python", []string{}},
		{"single tag", "#work notes", []string{"work"}},
		{"multiple tags", "#work and #personal tags", []string{"work", "personal"}},
		{"chinese tag", "#工作 标签", []string{"工作"}},
		{"mixed tags", "#dev #frontend #后端", []string{"dev", "frontend", "后端"}},
		{"tag with special chars", "#my-tag notes", []string{"my-tag"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := classifier.ExtractTags(tt.query)
			if len(result) != len(tt.expected) {
				t.Errorf("ExtractTags(%q) length = %d, want %d", tt.query, len(result), len(tt.expected))
				return
			}
			for i, tag := range result {
				if i >= len(tt.expected) || tag != tt.expected[i] {
					t.Errorf("ExtractTags(%q)[%d] = %q, want %q", tt.query, i, tag, tt.expected[i])
				}
			}
		})
	}
}

// TestMemoQueryClassifier_ExtractTimeFilter tests time range extraction.
func TestMemoQueryClassifier_ExtractTimeFilter(t *testing.T) {
	classifier := NewMemoQueryClassifier()
	now := time.Now().Truncate(time.Hour) // Truncate to hour for consistent testing

	// Set a fixed reference time for testing
	// Note: This test may be flaky if run at certain times, but provides basic coverage

	tests := []struct {
		name            string
		query           string
		shouldHaveRange bool
	}{
		{"no time filter", "search notes", false},
		{"today filter", "今天的笔记", true},
		{"yesterday filter", "昨天笔记", true},
		{"this week filter", "本周笔记", true},
		{"last week filter", "上周笔记", true},
		{"recent days", "最近3天", true},
		{"recent days - chinese", "最近7日", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start, end, ok := classifier.ExtractTimeFilter(tt.query)
			if ok != tt.shouldHaveRange {
				t.Errorf("ExtractTimeFilter(%q) ok = %v, want %v", tt.query, ok, tt.shouldHaveRange)
				return
			}
			if ok {
				if start.After(end) {
					t.Errorf("ExtractTimeFilter(%q) start %v after end %v", tt.query, start, end)
				}
				// Verify range is reasonable (not too far in the past or future)
				maxPast := 365 * 24 * time.Hour
				if now.Sub(start) > maxPast {
					t.Errorf("ExtractTimeFilter(%q) start %v is more than a year ago", tt.query, start)
				}
			}
		})
	}
}

// TestMemoQueryClassifier_EdgeCases tests edge cases and boundary conditions.
func TestMemoQueryClassifier_EdgeCases(t *testing.T) {
	classifier := NewMemoQueryClassifier()

	tests := []struct {
		name     string
		query    string
		expected MemoQueryIntent
	}{
		{"empty query", "", IntentList}, // Empty treated as list all
		{"whitespace only", "   ", IntentList},
		{"very long query", strings.Repeat("a", 200), IntentSemantic},
		{"special chars only", "!@#$%", IntentKeyword},
		{"mixed case", "PYTHON", IntentKeyword},
		{"mixed case semantic", "HOW TO DEPLOY", IntentSemantic},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := classifier.Classify(tt.query)
			if result != tt.expected {
				t.Errorf("Classify(%q) = %v, want %v", tt.query, result, tt.expected)
			}
		})
	}
}

// TestMemoQueryIntent_ToStrategy tests strategy mapping.
func TestMemoQueryIntent_ToStrategy(t *testing.T) {
	tests := []struct {
		intent   MemoQueryIntent
		expected string
	}{
		{IntentList, "memo_list_only"},
		{IntentFilter, "memo_filter_only"},
		{IntentKeyword, "memo_bm25_only"},
		{IntentSemantic, "memo_semantic_only"},
		{MemoQueryIntent(99), "memo_semantic_only"}, // Unknown defaults to semantic
	}

	for _, tt := range tests {
		t.Run(tt.intent.String(), func(t *testing.T) {
			result := tt.intent.ToStrategy()
			if result != tt.expected {
				t.Errorf("%v.ToStrategy() = %q, want %q", tt.intent, result, tt.expected)
			}
		})
	}
}
