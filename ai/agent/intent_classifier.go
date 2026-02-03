package agent

import (
	"regexp"
	"strings"
)

// TaskIntent represents the type of task across all agents.
// This unified intent system covers memo, schedule, and amazing agents.
type TaskIntent string

const (
	// Schedule-related intents.
	IntentSimpleCreate    TaskIntent = "schedule_create"   // 创建单个日程
	IntentSimpleQuery     TaskIntent = "schedule_query"    // 查询日程/空闲
	IntentSimpleUpdate    TaskIntent = "schedule_update"   // 修改/删除日程
	IntentBatchCreate     TaskIntent = "schedule_batch"    // 重复日程
	IntentConflictResolve TaskIntent = "schedule_conflict" // 处理冲突

	// Memo-related intents.
	IntentMemoSearch TaskIntent = "memo_search" // 搜索笔记
	IntentMemoCreate TaskIntent = "memo_create" // 创建笔记

	// Amazing agent intent.
	IntentAmazing TaskIntent = "amazing" // 综合分析

	// Legacy aliases for backward compatibility.
	IntentScheduleQuery  TaskIntent = "schedule_query"
	IntentScheduleCreate TaskIntent = "schedule_create"
	IntentScheduleUpdate TaskIntent = "schedule_update"
	IntentBatchSchedule  TaskIntent = "schedule_batch"
	IntentMultiQuery     TaskIntent = "amazing" // Alias to amazing
)

// - Batch tasks (IntentBatchCreate) → Plan-Execute mode.
type IntentClassifier struct {
	// Batch keywords that trigger Plan-Execute mode
	batchKeywords []string

	// Query keywords that indicate a search/list operation
	queryKeywords []string

	// Update keywords that indicate a modification
	updateKeywords []string

	// Compiled regex patterns for batch detection
	batchPatterns []*regexp.Regexp
}

// NewIntentClassifier creates a new IntentClassifier with default patterns.
func NewIntentClassifier() *IntentClassifier {
	ic := &IntentClassifier{
		// Keywords that suggest batch operations
		batchKeywords: []string{
			// Chinese
			"每天", "每日", "每周", "每月", "每年",
			"这周每天", "下周每天", "本周每天",
			"连续", "批量", "所有", "全部",
			"周一到周五", "工作日",
			// English
			"every day", "daily", "every week", "weekly",
			"every month", "monthly", "batch",
			"all days", "weekdays",
		},

		// Keywords that suggest queries
		queryKeywords: []string{
			// Chinese
			"有什么", "有哪些", "什么安排", "什么日程",
			"查看", "查询", "列出", "显示",
			"今天", "明天", "后天", "这周", "下周", "本周",
			"有空吗", "空闲", "忙吗",
			// English
			"what", "list", "show", "display", "query",
			"schedule", "schedules", "free", "busy", "available",
		},

		// Keywords that suggest updates
		updateKeywords: []string{
			// Chinese
			"改到", "改成", "修改", "调整", "更新",
			"推迟", "提前", "延后", "取消", "删除",
			"移到", "换到",
			// English
			"change", "modify", "update", "reschedule",
			"postpone", "cancel", "delete", "move",
		},
	}

	// Compile batch patterns
	ic.batchPatterns = []*regexp.Regexp{
		// "每[天|周|月]...做..." pattern
		regexp.MustCompile(`每[天日周月年]`),
		// "从...到..." range pattern
		regexp.MustCompile(`从.+到.+[每|都]`),
		// "周一至周五" pattern
		regexp.MustCompile(`周[一二三四五六日]至?到?周[一二三四五六日]`),
		// "下周所有" pattern
		regexp.MustCompile(`[这下本]周(所有|每[天日])`),
	}

	return ic
}

// Classify determines the intent of the user input.
// Handles negation patterns (e.g., "不要创建", "不是今天").
func (ic *IntentClassifier) Classify(input string) TaskIntent {
	lowerInput := strings.ToLower(input)

	// Check for explicit negation first - usually means query or clarification
	if ic.hasNegation(input, lowerInput) {
		// "不要/别/不用" + action → likely a query or clarification request
		// e.g., "不要创建会议" → user might be asking what they have instead
		return IntentSimpleQuery
	}

	// Check for memo-related intents first
	if ic.isMemoSearchIntent(input, lowerInput) {
		return IntentMemoSearch
	}

	// Check for batch patterns (highest priority for schedule)
	if ic.isBatchIntent(input, lowerInput) {
		return IntentBatchCreate
	}

	// Check for update intent
	if ic.isUpdateIntent(input, lowerInput) {
		return IntentSimpleUpdate
	}

	// Check for query intent
	if ic.isQueryIntent(input, lowerInput) {
		return IntentSimpleQuery
	}

	// Default to simple create
	return IntentSimpleCreate
}

// isBatchIntent checks if the input suggests a batch operation.
func (ic *IntentClassifier) isBatchIntent(input, lowerInput string) bool {
	// Check regex patterns
	for _, pattern := range ic.batchPatterns {
		if pattern.MatchString(input) {
			return true
		}
	}

	// Check keywords
	for _, keyword := range ic.batchKeywords {
		if strings.Contains(lowerInput, strings.ToLower(keyword)) {
			return true
		}
	}

	return false
}

// isQueryIntent checks if the input suggests a query operation.
func (ic *IntentClassifier) isQueryIntent(input, lowerInput string) bool {
	for _, keyword := range ic.queryKeywords {
		if strings.Contains(lowerInput, strings.ToLower(keyword)) {
			// Make sure it's not also a create request
			// e.g., "今天下午3点开会" is create, not query
			if !ic.hasTimeAndAction(input) {
				return true
			}
		}
	}

	// Questions are usually queries
	if strings.Contains(input, "?") || strings.Contains(input, "？") {
		return true
	}

	return false
}

// isUpdateIntent checks if the input suggests an update operation.
func (ic *IntentClassifier) isUpdateIntent(_, lowerInput string) bool {
	for _, keyword := range ic.updateKeywords {
		if strings.Contains(lowerInput, strings.ToLower(keyword)) {
			return true
		}
	}
	return false
}

// hasTimeAndAction checks if the input has both a time reference and an action.
// This helps distinguish "今天有什么" (query) from "今天3点开会" (create).
func (ic *IntentClassifier) hasTimeAndAction(input string) bool {
	// Simple heuristic: if it has a specific time (hour/minute), it's likely a create
	timePatterns := []*regexp.Regexp{
		regexp.MustCompile(`\d{1,2}[点时:]`),
		regexp.MustCompile(`上午|下午|早上|晚上|中午`),
		regexp.MustCompile(`\d{1,2}:\d{2}`),
	}

	for _, pattern := range timePatterns {
		if pattern.MatchString(input) {
			return true
		}
	}

	return false
}

// Examples: "不要创建", "别安排", "不是今天", "不用开会".
func (ic *IntentClassifier) hasNegation(input, lowerInput string) bool {
	negationWords := []string{
		// Chinese
		"不要", "别", "不用", "没有", "无", "非", "不是",
		"取消", "删除", "移除",
		// English
		"don't", "dont", "no", "not", "never", "cancel", "delete", "remove",
	}

	for _, word := range negationWords {
		if strings.Contains(lowerInput, strings.ToLower(word)) {
			return true
		}
	}

	// Check for "不是" pattern specifically - often means clarification
	if strings.Contains(input, "不是") || strings.Contains(lowerInput, "is not") || strings.Contains(lowerInput, "isnt") {
		return true
	}

	return false
}

// isMemoSearchIntent checks if the input suggests a memo search operation.
func (ic *IntentClassifier) isMemoSearchIntent(input, lowerInput string) bool {
	memoSearchKeywords := []string{
		// Chinese
		"笔记", "memo", "note", "记录", "搜索", "search", "查找", "find",
		"写过", "记过", "提到", "关于",
		// English
		"memo", "note", "find", "search", "look for",
	}

	for _, keyword := range memoSearchKeywords {
		if strings.Contains(lowerInput, strings.ToLower(keyword)) {
			// Make sure it's not also a schedule create request
			// e.g., "明天3点开会" is schedule, not memo
			if !ic.hasTimeAndAction(input) {
				return true
			}
		}
	}

	return false
}

// ShouldUsePlanExecute returns true if the intent should use Plan-Execute mode.
func (ic *IntentClassifier) ShouldUsePlanExecute(intent TaskIntent) bool {
	switch intent {
	case IntentBatchCreate, IntentAmazing:
		return true
	default:
		return false
	}
}

// ClassifyAndRoute is a convenience method that classifies and returns the execution mode.
func (ic *IntentClassifier) ClassifyAndRoute(input string) (TaskIntent, bool) {
	intent := ic.Classify(input)
	usePlanExecute := ic.ShouldUsePlanExecute(intent)
	return intent, usePlanExecute
}
