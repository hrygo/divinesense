package agent

import (
	"testing"
)

// TestRouteByRules tests the rule-based routing logic.
func TestRouteByRules(t *testing.T) {
	router := &ChatRouter{}

	testCases := []struct {
		name           string
		input          string
		expectedRoute  ChatRouteType
		expectedMethod string
		shouldMatch    bool // true if rules should match, false if LLM needed
	}{
		// === 明确意图 (Clear Intent) ===
		{
			name:           "clear_schedule_chinese",
			input:          "明天下午3点有个会议",
			expectedRoute:  RouteTypeSchedule,
			expectedMethod: "rule",
			shouldMatch:    true,
		},
		{
			name:           "clear_memo_chinese",
			input:          "搜索关于项目计划的笔记",
			expectedRoute:  RouteTypeMemo,
			expectedMethod: "rule",
			shouldMatch:    true,
		},
		{
			name:           "clear_amazing_chinese",
			input:          "总结一下本周工作",
			expectedRoute:  RouteTypeAmazing,
			expectedMethod: "rule",
			shouldMatch:    true,
		},

		// === 中英文混合 (Mixed Chinese/English) ===
		{
			name:           "mixed_schedule_en_time",
			input:          "schedule a meeting 明天上午",
			expectedRoute:  RouteTypeSchedule,
			expectedMethod: "rule",
			shouldMatch:    true,
		},
		{
			name:           "mixed_memo_search",
			input:          "find my notes 关于 AI",
			expectedRoute:  RouteTypeMemo,
			expectedMethod: "rule",
			shouldMatch:    true,
		},
		{
			name:           "mixed_summary",
			input:          "给我一个 summary 本周",
			expectedRoute:  RouteTypeAmazing,
			expectedMethod: "rule",
			shouldMatch:    true,
		},

		// === 模糊输入 (Ambiguous Input) - 需要 LLM ===
		{
			name:        "ambiguous_help",
			input:       "帮我处理一下",
			shouldMatch: false, // No clear keywords
		},
		{
			name:        "ambiguous_greeting",
			input:       "你好",
			shouldMatch: false,
		},
		{
			name:        "ambiguous_question",
			input:       "这个怎么弄",
			shouldMatch: false,
		},
		{
			name:        "ambiguous_english_only",
			input:       "can you help me with something",
			shouldMatch: false,
		},

		// === 多意图组合 (Multi-Intent) ===
		// These cases have mixed signals - properly fall through to LLM
		{
			name:        "multi_intent_schedule_memo_mix",
			input:       "查一下我明天有没有时间开会",
			shouldMatch: false, // schedule score=1 (明天), not enough for threshold
		},
		{
			name:        "multi_intent_memo_with_schedule_word",
			input:       "我写过关于会议纪要的笔记吗",
			shouldMatch: false, // memo=6, schedule=2 (会议), condition memoScore>=2 && scheduleScore<2 fails
		},

		// === 边界情况 (Edge Cases) ===
		{
			name:        "empty_input",
			input:       "",
			shouldMatch: false,
		},
		{
			name:        "whitespace_only",
			input:       "   ",
			shouldMatch: false,
		},
		{
			name:        "single_keyword_schedule",
			input:       "日程",
			shouldMatch: false, // score=2, threshold=3
		},
		{
			name:           "single_keyword_memo",
			input:          "笔记",
			expectedRoute:  RouteTypeMemo,
			expectedMethod: "rule",
			shouldMatch:    true, // score = 2, threshold is 2
		},

		// === 时间表达式 (Time Expressions) ===
		{
			name:           "time_expression_tomorrow",
			input:          "明天上午10点",
			expectedRoute:  RouteTypeSchedule,
			expectedMethod: "rule",
			shouldMatch:    true, // 明天(1) + 上午(1) + 点(1) = 3
		},
		{
			name:           "time_expression_week",
			input:          "下周三有空吗",
			expectedRoute:  RouteTypeSchedule,
			expectedMethod: "rule",
			shouldMatch:    true, // 下周(1) + 周三(1) + 有空(2) = 4
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := router.routeByRules(tc.input)

			if tc.shouldMatch {
				if result == nil {
					t.Errorf("Expected rule match for input %q, but got nil (LLM needed)", tc.input)
					return
				}
				if result.Route != tc.expectedRoute {
					t.Errorf("Expected route %q, got %q for input %q", tc.expectedRoute, result.Route, tc.input)
				}
				if result.Method != tc.expectedMethod {
					t.Errorf("Expected method %q, got %q for input %q", tc.expectedMethod, result.Method, tc.input)
				}
			} else {
				if result != nil {
					t.Errorf("Expected no rule match (LLM needed) for input %q, but got route %q", tc.input, result.Route)
				}
			}
		})
	}
}

// TestRouteByRulesScoring tests the scoring mechanism.
func TestRouteByRulesScoring(t *testing.T) {
	router := &ChatRouter{}

	// Test cases that verify the scoring thresholds
	testCases := []struct {
		name          string
		input         string
		expectedRoute ChatRouteType
		minConfidence float64
	}{
		{
			name:          "high_confidence_schedule",
			input:         "帮我安排明天下午3点的会议",
			expectedRoute: RouteTypeSchedule,
			minConfidence: 0.80,
		},
		{
			name:          "high_confidence_memo",
			input:         "搜索我的笔记记录",
			expectedRoute: RouteTypeMemo,
			minConfidence: 0.80,
		},
		{
			name:          "high_confidence_amazing",
			input:         "给我总结一下本周工作",
			expectedRoute: RouteTypeAmazing,
			minConfidence: 0.80,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := router.routeByRules(tc.input)
			if result == nil {
				t.Fatalf("Expected rule match for input %q", tc.input)
			}
			if result.Route != tc.expectedRoute {
				t.Errorf("Expected route %q, got %q", tc.expectedRoute, result.Route)
			}
			if result.Confidence < tc.minConfidence {
				t.Errorf("Expected confidence >= %.2f, got %.2f", tc.minConfidence, result.Confidence)
			}
		})
	}
}

// TestMapRoute tests the route string mapping.
func TestMapRoute(t *testing.T) {
	router := &ChatRouter{}

	testCases := []struct {
		input    string
		expected ChatRouteType
	}{
		{"memo", RouteTypeMemo},
		{"MEMO", RouteTypeMemo},
		{"笔记", RouteTypeMemo},
		{"schedule", RouteTypeSchedule},
		{"SCHEDULE", RouteTypeSchedule},
		{"日程", RouteTypeSchedule},
		{"amazing", RouteTypeAmazing},
		{"AMAZING", RouteTypeAmazing},
		{"unknown", RouteTypeAmazing},
		{"", RouteTypeAmazing},
		{"  memo  ", RouteTypeMemo},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			result := router.mapRoute(tc.input)
			if result != tc.expected {
				t.Errorf("mapRoute(%q) = %q, want %q", tc.input, result, tc.expected)
			}
		})
	}
}

// BenchmarkRouteByRules benchmarks the rule-based routing performance.
func BenchmarkRouteByRules(b *testing.B) {
	router := &ChatRouter{}
	inputs := []string{
		"明天下午3点有个会议",
		"搜索关于项目的笔记",
		"总结一下本周工作",
		"帮我处理一下",
		"schedule a meeting tomorrow",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, input := range inputs {
			router.routeByRules(input)
		}
	}
}
