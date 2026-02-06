// Package router provides the LLM routing service.
package router

import (
	"regexp"
	"strings"
	"unicode"
)

// Pre-defined core keywords for each category (avoid map creation on every call).
var coreKeywordsByCategory = map[string][]string{
	"schedule": {"日程", "安排", "会议", "提醒", "预约", "开会"},
	"memo":     {"笔记", "搜索", "查找", "记录", "memo"},
	"amazing":  {"综合", "总结", "分析", "周报"},
}

// Pre-compiled regex patterns for intent sub-classification.
var (
	updatePatternRegex = regexp.MustCompile(`修改|更新|取消|改|删除`)
	queryPatternRegex  = regexp.MustCompile(`查看|有什么|哪些|看看|什么安排|有没有`)
	batchPatternRegex  = regexp.MustCompile(`批量|多个|一系列|每天|每周`)
	searchPatternRegex = regexp.MustCompile(`搜索|查找|找|查`)
	createPatternRegex = regexp.MustCompile(`记录|记一下|写|保存|创建`)
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
			"今天": 2, "明天": 2, "后天": 2, "下周": 2, "本周": 2,
			"上午": 2, "下午": 2, "晚上": 2, "点": 2,
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
	// Fast path: normalize once
	lower := m.normalizeInput(input)

	// FAST PATH: Time pattern + query pattern → schedule query (e.g., "明天有什么事情要做")
	// This handles common schedule queries without requiring core keywords like "日程" or "安排"
	if m.hasTimePattern(input) && queryPatternRegex.MatchString(lower) {
		return IntentScheduleQuery, 0.85, true
	}

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
	if scheduleScore >= 2 && hasCoreScheduleKeyword {
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

// normalizeInput normalizes input for faster matching.
// Removes punctuation and converts to lowercase once.
func (m *RuleMatcher) normalizeInput(input string) string {
	// Quick ASCII-only path (most common for English/mixed input)
	isASCII := true
	for _, r := range input {
		if r > unicode.MaxASCII {
			isASCII = false
			break
		}
	}

	if isASCII {
		return strings.ToLower(input)
	}

	// Chinese path: normalize spaces and punctuation
	result := strings.Builder{}
	result.Grow(len(input))

	for _, r := range input {
		// Skip common punctuation
		if r == ' ' || r == ',' || r == '。' || r == '，' ||
			r == '？' || r == '?' || r == '！' || r == '!' ||
			r == '、' || r == '\t' || r == '\n' {
			continue
		}
		// Convert to lowercase if ASCII
		if r <= 'Z' && r >= 'A' {
			r += 32
		}
		result.WriteRune(r)
	}

	return result.String()
}

// hasCoreKeyword checks if input contains a core keyword for the given category.
// Optimized: uses strings.Contains which is highly optimized in Go.
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
// Optimized: single pass over keywords, early exit on max score.
func (m *RuleMatcher) calculateScore(input string, keywords map[string]int) int {
	score := 0
	for keyword, weight := range keywords {
		if strings.Contains(input, keyword) {
			score += weight
			// Early exit: max reasonable score is 6-7
			if score >= 7 {
				return score
			}
		}
	}
	return score
}

// hasTimePattern checks if input contains time patterns.
// Optimized: returns early on first match.
func (m *RuleMatcher) hasTimePattern(input string) bool {
	for _, pattern := range m.timePatterns {
		if pattern.MatchString(input) {
			return true
		}
	}
	return false
}

// determineScheduleIntent determines if it's create, query, or update.
// Optimized: uses pre-compiled regex patterns.
func (m *RuleMatcher) determineScheduleIntent(input string, _ int) Intent {
	if updatePatternRegex.MatchString(input) {
		return IntentScheduleUpdate
	}
	if queryPatternRegex.MatchString(input) {
		return IntentScheduleQuery
	}
	if batchPatternRegex.MatchString(input) {
		return IntentBatchSchedule
	}
	// Default to create if time pattern present
	return IntentScheduleCreate
}

// determineMemoIntent determines if it's search or create.
// Optimized: uses pre-compiled regex patterns.
func (m *RuleMatcher) determineMemoIntent(input string) Intent {
	if searchPatternRegex.MatchString(input) {
		return IntentMemoSearch
	}
	if createPatternRegex.MatchString(input) {
		return IntentMemoCreate
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
