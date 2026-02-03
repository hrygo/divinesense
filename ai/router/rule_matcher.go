// Package router provides the LLM routing service.
package router

import (
	"regexp"
	"strings"
)

// Pre-defined core keywords for each category (avoid map creation on every call).
var coreKeywordsByCategory = map[string][]string{
	"schedule": {"日程", "安排", "会议", "提醒", "预约", "开会"},
	"memo":     {"笔记", "搜索", "查找", "记录", "memo"},
	"amazing":  {"综合", "总结", "分析", "周报"},
}

// Pre-defined intent patterns (avoid slice creation on every call).
var (
	updatePatterns = []string{"修改", "更新", "取消", "改", "删除"}
	queryPatterns  = []string{"查看", "有什么", "哪些", "看看", "什么安排", "有没有"}
	batchPatterns  = []string{"批量", "多个", "一系列", "每天", "每周"}
	searchPatterns = []string{"搜索", "查找", "找", "查", "有什么", "哪些"}
	createPatterns = []string{"记录", "记一下", "写", "保存", "创建"}
)

// RuleMatcher implements Layer 1 rule-based intent matching.
// Target: 0ms latency, handle 60%+ of requests.
type RuleMatcher struct {
	scheduleKeywords map[string]int
	memoKeywords     map[string]int
	amazingKeywords  map[string]int
	timePatterns     []*regexp.Regexp
}

// NewRuleMatcher creates a new rule matcher with predefined keyword weights.
func NewRuleMatcher() *RuleMatcher {
	return &RuleMatcher{
		// Schedule keywords: weight +2 for core, +1 for supporting
		scheduleKeywords: map[string]int{
			// Core keywords (+2)
			"日程": 2, "安排": 2, "会议": 2, "提醒": 2, "预约": 2,
			"开会": 2, "约会": 2, "设置提醒": 3, "创建日程": 3,
			// Supporting keywords (+1)
			"今天": 1, "明天": 1, "后天": 1, "下周": 1, "本周": 1,
			"上午": 1, "下午": 1, "晚上": 1, "点": 1,
		},
		// Memo keywords: weight +2 for core, +1 for supporting
		memoKeywords: map[string]int{
			// Core keywords (+2)
			"笔记": 2, "搜索": 2, "查找": 2, "记录": 2, "写过": 2,
			"找": 2, "memo": 2, "查": 2,
			// Supporting keywords (+1)
			"关于": 1, "提到": 1, "之前": 1, "有关": 1, "记": 1,
		},
		// Amazing (general assistant) keywords
		amazingKeywords: map[string]int{
			// Core keywords (+2)
			"综合": 2, "总结": 2, "分析": 2, "周报": 2, "帮我": 2,
			"怎么": 2, "什么": 2, "为什么": 2,
			// Supporting keywords (+1)
			"本周": 1, "工作": 1, "解释": 1, "说说": 1,
		},
		// Time patterns for schedule detection
		timePatterns: []*regexp.Regexp{
			regexp.MustCompile(`\d{1,2}[:\s时点]\d{0,2}`),       // 10:30, 10点, 10时30
			regexp.MustCompile(`(上午|下午|晚上|早上|中午)\d{1,2}[点时]`), // 下午3点
			regexp.MustCompile(`(明天|后天|今天|下周|本周)`),            // Relative dates
			regexp.MustCompile(`\d{1,2}月\d{1,2}[日号]`),         // 1月15日
		},
	}
}

// Returns: intent, confidence, matched (true if rule matched).
func (m *RuleMatcher) Match(input string) (Intent, float32, bool) {
	lower := strings.ToLower(input)

	// Calculate scores for each intent category
	scheduleScore := m.calculateScore(lower, m.scheduleKeywords)
	memoScore := m.calculateScore(lower, m.memoKeywords)
	amazingScore := m.calculateScore(lower, m.amazingKeywords)

	// Time pattern adds score to schedule only if it has core schedule keywords
	hasTimePattern := m.hasTimePattern(input)
	hasCoreScheduleKeyword := m.hasCoreKeyword(lower, "schedule")
	if hasTimePattern && hasCoreScheduleKeyword {
		scheduleScore += 2
	}

	// Memo takes priority if it has explicit memo keywords
	if memoScore >= 3 || (memoScore >= 2 && m.hasCoreKeyword(lower, "memo")) {
		intent := m.determineMemoIntent(lower)
		confidence := m.normalizeConfidence(memoScore, 5)
		return intent, confidence, true
	}

	// Schedule needs both high score AND core schedule keyword
	if scheduleScore >= 3 && hasCoreScheduleKeyword {
		intent := m.determineScheduleIntent(lower, scheduleScore)
		confidence := m.normalizeConfidence(scheduleScore, 6)
		return intent, confidence, true
	}

	// Amazing needs high score AND core amazing keyword (not just "帮我")
	if amazingScore >= 3 && m.hasCoreKeyword(lower, "amazing") {
		confidence := m.normalizeConfidence(amazingScore, 5)
		return IntentAmazing, confidence, true
	}

	// No match - needs higher layer processing
	return IntentUnknown, 0, false
}

// hasCoreKeyword checks if input contains a core keyword for the given category.
func (m *RuleMatcher) hasCoreKeyword(input, category string) bool {
	keywords, ok := coreKeywordsByCategory[category]
	if !ok {
		return false
	}
	for _, kw := range keywords {
		if strings.Contains(input, kw) {
			return true
		}
	}
	return false
}

// calculateScore calculates the weighted score for a keyword set.
func (m *RuleMatcher) calculateScore(input string, keywords map[string]int) int {
	score := 0
	for keyword, weight := range keywords {
		if strings.Contains(input, keyword) {
			score += weight
		}
	}
	return score
}

// hasTimePattern checks if input contains time patterns.
func (m *RuleMatcher) hasTimePattern(input string) bool {
	for _, pattern := range m.timePatterns {
		if pattern.MatchString(input) {
			return true
		}
	}
	return false
}

// determineScheduleIntent determines if it's create, query, or update.
func (m *RuleMatcher) determineScheduleIntent(input string, _ int) Intent {
	// Update patterns
	for _, p := range updatePatterns {
		if strings.Contains(input, p) {
			return IntentScheduleUpdate
		}
	}

	// Query patterns
	for _, p := range queryPatterns {
		if strings.Contains(input, p) {
			return IntentScheduleQuery
		}
	}

	// Batch schedule patterns
	for _, p := range batchPatterns {
		if strings.Contains(input, p) {
			return IntentBatchSchedule
		}
	}

	// Default to create if time pattern present
	return IntentScheduleCreate
}

// determineMemoIntent determines if it's search or create.
func (m *RuleMatcher) determineMemoIntent(input string) Intent {
	// Search patterns
	for _, p := range searchPatterns {
		if strings.Contains(input, p) {
			return IntentMemoSearch
		}
	}

	// Create patterns
	for _, p := range createPatterns {
		if strings.Contains(input, p) {
			return IntentMemoCreate
		}
	}

	// Default to search
	return IntentMemoSearch
}

// normalizeConfidence normalizes score to 0-1 confidence range.
func (m *RuleMatcher) normalizeConfidence(score, maxScore int) float32 {
	if score >= maxScore {
		return 0.95
	}
	return float32(score) / float32(maxScore)
}
