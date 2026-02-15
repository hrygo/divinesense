package routing

import (
	"strings"
	"testing"
)

// mockCapabilityMap implements KeywordCapabilitySource for testing.
type mockCapabilityMap struct {
	capabilities map[string][]string // input -> capabilities
}

func (m *mockCapabilityMap) IdentifyCapabilities(text string) []string {
	text = strings.ToLower(text)
	var results []string
	for key, caps := range m.capabilities {
		if strings.Contains(text, key) {
			results = append(results, caps...)
		}
	}
	// Remove duplicates
	seen := make(map[string]bool)
	var unique []string
	for _, cap := range results {
		if !seen[cap] {
			seen[cap] = true
			unique = append(unique, cap)
		}
	}
	return unique
}

// newTestMatcher creates a RuleMatcher with mock capabilityMap for testing.
func newTestMatcher() *RuleMatcher {
	matcher := NewRuleMatcher()
	matcher.SetCapabilityMap(&mockCapabilityMap{
		capabilities: map[string][]string{
			// Schedule triggers
			"日程":   {"日程", "创建日程", "查询日程"},
			"安排":   {"日程", "安排"},
			"会议":   {"日程", "会议"},
			"提醒":   {"日程", "提醒"},
			"预约":   {"日程"},
			"开会":   {"日程", "会议"},
			"创建日程": {"日程", "创建日程"},
			"查询日程": {"日程", "查询日程"},
			// Memo triggers - more comprehensive
			"笔记":   {"笔记", "搜索笔记"},
			"搜索":   {"笔记", "搜索笔记"},
			"查找":   {"笔记", "搜索笔记"},
			"记录":   {"笔记", "搜索笔记"},
			"memo": {"笔记", "搜索笔记"},
			"找":    {"笔记", "搜索笔记"},
			"帮我找":  {"笔记", "搜索笔记"},
			// Schedule update triggers
			"修改": {"日程更新"},
			"更新": {"日程更新"},
			"取消": {"日程更新"},
			"删除": {"日程更新"},
			// Batch triggers
			"批量": {"批量日程"},
			"每周": {"批量日程"},
			"每天": {"批量日程"},
		},
	})
	return matcher
}

// TestRuleMatcher_MemoSearch tests memo search intent matching.
func TestRuleMatcher_MemoSearch(t *testing.T) {
	matcher := newTestMatcher()

	testCases := []struct {
		input         string
		expected      Intent
		minConfidence float32
	}{
		{"搜索笔记", IntentMemoSearch, 0.7},
		{"查找笔记", IntentMemoSearch, 0.7},
		{"找笔记", IntentMemoSearch, 0.7},
		{"查看笔记", IntentMemoSearch, 0.7},
		{"帮我找memo", IntentMemoSearch, 0.7},
		{"查记录", IntentMemoSearch, 0.6},
	}

	for _, tc := range testCases {
		intent, confidence, matched := matcher.MatchLegacy(tc.input)
		if !matched {
			t.Errorf("input %q: expected match, got no match", tc.input)
			continue
		}
		if intent != tc.expected {
			t.Errorf("input %q: expected %s, got %s", tc.input, tc.expected, intent)
		}
		if confidence < tc.minConfidence {
			t.Errorf("input %q: confidence %f below minimum %f", tc.input, confidence, tc.minConfidence)
		}
	}
}

// TestRuleMatcher_ScheduleCreate tests schedule create intent matching.
// Note: New architecture separates generic action from expert mapping.
// "明天下午3点开会" is now recognized as ActionQuery (time pattern) -> IntentScheduleQuery
// Only inputs with explicit creation keywords (创建, 记录, etc.) become schedule_create.
func TestRuleMatcher_ScheduleCreate(t *testing.T) {
	matcher := newTestMatcher()

	testCases := []struct {
		input         string
		expected      Intent
		minConfidence float32
	}{
		// Time pattern without explicit action keyword → query (new behavior)
		{"明天下午3点开会", IntentScheduleQuery, 0.6},
		{"提醒我明天开会", IntentScheduleQuery, 0.6},
		{"今天下午2点会议", IntentScheduleQuery, 0.6},
		// Explicit creation keyword → create
		{"创建日程明天", IntentScheduleCreate, 0.7},
	}

	for _, tc := range testCases {
		intent, confidence, matched := matcher.MatchLegacy(tc.input)
		if !matched {
			t.Errorf("input %q: expected match, got no match", tc.input)
			continue
		}
		if intent != tc.expected {
			t.Errorf("input %q: expected %s, got %s", tc.input, tc.expected, intent)
		}
		if confidence < tc.minConfidence {
			t.Errorf("input %q: confidence %f below minimum %f", tc.input, confidence, tc.minConfidence)
		}
	}
}

// TestRuleMatcher_ScheduleUpdate tests schedule update intent matching.
func TestRuleMatcher_ScheduleUpdate(t *testing.T) {
	matcher := newTestMatcher()

	testCases := []struct {
		input         string
		expected      Intent
		minConfidence float32
	}{
		{"修改明天的会议", IntentScheduleUpdate, 0.6},
		{"取消明天会议", IntentScheduleUpdate, 0.6},
	}

	for _, tc := range testCases {
		intent, confidence, matched := matcher.MatchLegacy(tc.input)
		if !matched {
			t.Errorf("input %q: expected match, got no match", tc.input)
			continue
		}
		if intent != tc.expected {
			t.Errorf("input %q: expected %s, got %s", tc.input, tc.expected, intent)
		}
		if confidence < tc.minConfidence {
			t.Errorf("input %q: confidence %f below minimum %f", tc.input, confidence, tc.minConfidence)
		}
	}
}

// TestRuleMatcher_BatchSchedule tests batch schedule intent matching.
func TestRuleMatcher_BatchSchedule(t *testing.T) {
	matcher := newTestMatcher()

	testCases := []struct {
		input         string
		expected      Intent
		minConfidence float32
	}{
		{"批量创建日程", IntentBatchSchedule, 0.6},
		{"设置每周会议提醒", IntentBatchSchedule, 0.6},
	}

	for _, tc := range testCases {
		intent, confidence, matched := matcher.MatchLegacy(tc.input)
		if !matched {
			t.Errorf("input %q: expected match, got no match", tc.input)
			continue
		}
		if intent != tc.expected {
			t.Errorf("input %q: expected %s, got %s", tc.input, tc.expected, intent)
		}
		if confidence < tc.minConfidence {
			t.Errorf("input %q: confidence %f below minimum %f", tc.input, confidence, tc.minConfidence)
		}
	}
}

// TestRuleMatcher_NormalizeInput tests input normalization.
func TestRuleMatcher_NormalizeInput(t *testing.T) {
	matcher := newTestMatcher()

	testCases := []struct {
		input    string
		expected string
	}{
		{"Hello World", "hello world"},
		{"你好，世界", "你好世界"},
		{"测试？输入！", "测试输入"},
		{"Mixed 混合 Content 内容", "mixed混合content内容"},
	}

	for _, tc := range testCases {
		result := matcher.normalizeInput(tc.input)
		if result != tc.expected {
			t.Errorf("normalizeInput(%q): expected %q, got %q", tc.input, tc.expected, result)
		}
	}
}

// BenchmarkRuleMatcher_Match benchmarks the rule matching performance.
func BenchmarkRuleMatcher_MatchMixed(b *testing.B) {
	matcher := newTestMatcher()
	inputs := []string{
		"搜索笔记",
		"明天下午3点开会",
		"总结本周工作",
		"修改日程",
		"随便说点什么",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		matcher.Match(inputs[i%len(inputs)])
	}
}

// BenchmarkRuleMatcher_Contains benchmarks strings.Contains performance.
func BenchmarkRuleMatcher_Contains(b *testing.B) {
	input := "明天下午3点开会，提醒我参加"
	pattern := "会议"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		strings.Contains(input, pattern)
	}
}
