package routing

import (
	"strings"
	"testing"
)

// TestRuleMatcher_MemoSearch tests memo search intent matching.
func TestRuleMatcher_MemoSearch(t *testing.T) {
	matcher := NewRuleMatcher()

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
		intent, confidence, matched := matcher.Match(tc.input)
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
func TestRuleMatcher_ScheduleCreate(t *testing.T) {
	matcher := NewRuleMatcher()

	testCases := []struct {
		input         string
		expected      Intent
		minConfidence float32
	}{
		{"明天下午3点开会", IntentScheduleCreate, 0.6},
		{"提醒我明天开会", IntentScheduleCreate, 0.7},
		{"今天下午2点会议", IntentScheduleCreate, 0.6},
		{"创建日程明天", IntentScheduleCreate, 0.7},
	}

	for _, tc := range testCases {
		intent, confidence, matched := matcher.Match(tc.input)
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
	matcher := NewRuleMatcher()

	testCases := []struct {
		input         string
		expected      Intent
		minConfidence float32
	}{
		{"修改明天的会议", IntentScheduleUpdate, 0.6},
		{"取消明天会议", IntentScheduleUpdate, 0.6},
	}

	for _, tc := range testCases {
		intent, confidence, matched := matcher.Match(tc.input)
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
	matcher := NewRuleMatcher()

	testCases := []struct {
		input         string
		expected      Intent
		minConfidence float32
	}{
		{"批量创建日程", IntentBatchSchedule, 0.6},
		{"设置每周会议提醒", IntentBatchSchedule, 0.6},
	}

	for _, tc := range testCases {
		intent, confidence, matched := matcher.Match(tc.input)
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

// TestRuleMatcher_Amazing tests amazing intent matching.
func TestRuleMatcher_Amazing(t *testing.T) {
	matcher := NewRuleMatcher()

	testCases := []struct {
		input         string
		expected      Intent
		minConfidence float32
	}{
		{"总结本周工作", IntentAmazing, 0.6},
		{"综合分析一下", IntentAmazing, 0.6},
		{"帮我分析总结", IntentAmazing, 0.5},
	}

	for _, tc := range testCases {
		intent, confidence, matched := matcher.Match(tc.input)
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
	matcher := NewRuleMatcher()

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
	matcher := NewRuleMatcher()
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
