package tools

import (
	"regexp"
	"strings"
	"time"
)

// MemoQueryIntent represents the classification of a memo search query.
// MemoQueryIntent 表示笔记搜索查询的分类意图。
type MemoQueryIntent int

const (
	// IntentList: List all memos without search criteria.
	// Examples: "有什么笔记", "列出所有笔记", "我的笔记", "全部笔记"
	IntentList MemoQueryIntent = iota

	// IntentFilter: Filter memos by time, tags, or other attributes.
	// Examples: "今天的笔记", "最近一周", "标签 #work", "本周笔记"
	IntentFilter

	// IntentKeyword: Simple keyword matching (BM25).
	// Examples: "Python", "包含 React 的笔记"
	IntentKeyword

	// IntentSemantic: Semantic vector search.
	// Examples: "如何部署服务", "关于数据库设计的笔记", "笔记中提到的时间管理方法"
	IntentSemantic
)

// Query length thresholds for classification.
const (
	// maxSimpleQueryLength is the maximum length for a simple (keyword) query.
	maxSimpleQueryLength = 15 // Used in isSimpleKeyword heuristic
)

// String returns the string representation of the intent.
func (i MemoQueryIntent) String() string {
	switch i {
	case IntentList:
		return "list"
	case IntentFilter:
		return "filter"
	case IntentKeyword:
		return "keyword"
	case IntentSemantic:
		return "semantic"
	default:
		return "unknown"
	}
}

// ToStrategy converts intent to retrieval strategy name.
func (i MemoQueryIntent) ToStrategy() string {
	switch i {
	case IntentList:
		return "memo_list_only"
	case IntentFilter:
		return "memo_filter_only"
	case IntentKeyword:
		return "memo_bm25_only"
	case IntentSemantic:
		return "memo_semantic_only"
	default:
		return "memo_semantic_only"
	}
}

// MemoQueryClassifier classifies memo search queries into intents.
// MemoQueryClassifier 将笔记搜索查询分类为意图。
type MemoQueryClassifier struct {
	// Pre-compiled regex patterns for efficiency
	listPatterns    []*regexp.Regexp
	timePatterns    []*regexp.Regexp
	tagPatterns     []*regexp.Regexp
	keywordPatterns []*regexp.Regexp
}

// NewMemoQueryClassifier creates a new query classifier.
// NewMemoQueryClassifier 创建一个新的查询分类器。
func NewMemoQueryClassifier() *MemoQueryClassifier {
	return &MemoQueryClassifier{
		listPatterns: compilePatterns([]string{
			`^有什么笔记`,
			`^列出.*笔记`,
			`^我的笔记$`,
			`^全部笔记`,
			`^所有笔记`,
			`^\*`,         // Wildcard for list all
			`^show.*memo`, // English support
			`^list.*memo`,
		}),
		timePatterns: compilePatterns([]string{
			`今天`,
			`昨天`,
			`本周`,
			`上周`,
			`最近`,
			`今天.*笔记`,
			`昨天.*笔记`,
			`本周.*笔记`,
			`recent`,
			`today`,
			`yesterday`,
			`this week`,
		}),
		tagPatterns: compilePatterns([]string{
			`标签.*#`,
			`#[\w-]+`,
			`tag.*:`,
			`tags?:`,
		}),
		keywordPatterns: compilePatterns([]string{
			`^..{1,10}$`, // Short queries (1-10 chars) are keywords
		}),
		// Note: Empty string handling is done in Classify() before keywordPatterns check
	}
}

// compilePatterns compiles regex patterns for efficiency.
func compilePatterns(patterns []string) []*regexp.Regexp {
	compiled := make([]*regexp.Regexp, len(patterns))
	for i, p := range patterns {
		compiled[i] = regexp.MustCompile(`(?i)` + p) // Case-insensitive
	}
	return compiled
}

// Classify determines the intent of a memo search query.
// Uses a layered approach: rules → heuristics → fallback.
// Classify 确定笔记搜索查询的意图。
// 使用分层方法：规则 → 启发式 → 降级。
func (c *MemoQueryClassifier) Classify(query string) MemoQueryIntent {
	normalized := strings.TrimSpace(query)

	// Special case: empty or whitespace-only query is treated as "list all"
	if normalized == "" {
		return IntentList
	}

	// Layer 1: List intent (highest priority, 0ms)
	// 检查明确的列表意图模式
	for _, pattern := range c.listPatterns {
		if pattern.MatchString(normalized) {
			return IntentList
		}
	}

	// Layer 2: Filter intent (time, tag, etc., 0ms)
	// 检查时间过滤模式
	for _, pattern := range c.timePatterns {
		if pattern.MatchString(normalized) {
			return IntentFilter
		}
	}
	// 检查标签过滤模式
	for _, pattern := range c.tagPatterns {
		if pattern.MatchString(normalized) {
			return IntentFilter
		}
	}

	// Layer 3: Semantic intent check (must happen before keyword patterns)
	// 先检查语义意图，避免被短查询模式误判
	if !c.isSimpleKeyword(normalized) {
		return IntentSemantic
	}

	// Layer 4: Keyword intent (simple terms, 0ms)
	// 短查询（1-10字符）且无语义特征 = 关键词搜索
	for _, pattern := range c.keywordPatterns {
		if pattern.MatchString(normalized) {
			return IntentKeyword
		}
	}

	// Layer 5: Default to semantic search
	// 默认语义搜索
	return IntentSemantic
}

// isSimpleKeyword checks if the query is a simple keyword without complex syntax.
func (c *MemoQueryClassifier) isSimpleKeyword(query string) bool {
	// Complex word patterns that indicate semantic search intent (direct match)
	complexWords := []string{
		"如何", "怎么", "为什么", "怎样", "是什么", // Chinese question words
		"how", "why", "what", "where", "when", // English question words
		"关于", "related to", "about", // About/relation words
		"笔记中提到", "note mention", // Mention patterns (direct match)
	}

	queryLower := strings.ToLower(query)
	for _, pattern := range complexWords {
		if strings.Contains(queryLower, pattern) {
			return false // Not a simple keyword if it matches complex pattern
		}
	}

	// Single word or short phrase = keyword
	// Only return true if no complex patterns matched AND query is short
	return len(query) <= maxSimpleQueryLength
}

// ExtractTimeFilter extracts time range from a filter query.
// Returns nil if no time filter is found.
// ExtractTimeFilter 从过滤查询中提取时间范围。
// 如果未找到时间过滤，则返回 nil。
func (c *MemoQueryClassifier) ExtractTimeFilter(query string) (start, end time.Time, ok bool) {
	now := time.Now()
	normalized := strings.ToLower(query)

	// Today
	if strings.Contains(normalized, "今天") || strings.Contains(normalized, "today") {
		start = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		end = start.Add(24 * time.Hour)
		return start, end, true
	}

	// Yesterday
	if strings.Contains(normalized, "昨天") || strings.Contains(normalized, "yesterday") {
		yesterday := now.AddDate(0, 0, -1)
		start = time.Date(yesterday.Year(), yesterday.Month(), yesterday.Day(), 0, 0, 0, 0, now.Location())
		end = start.Add(24 * time.Hour)
		return start, end, true
	}

	// This week
	if strings.Contains(normalized, "本周") || strings.Contains(normalized, "this week") {
		// Start of week (Monday)
		weekday := int(now.Weekday())
		if weekday == 0 {
			weekday = 7 // Sunday = 7 in ISO
		}
		start = now.AddDate(0, 0, -(weekday - 1))
		start = time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, now.Location())
		end = start.Add(7 * 24 * time.Hour)
		return start, end, true
	}

	// Last week
	if strings.Contains(normalized, "上周") || strings.Contains(normalized, "last week") {
		weekday := int(now.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		start = now.AddDate(0, 0, -(weekday - 1 + 7))
		start = time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, now.Location())
		end = start.Add(7 * 24 * time.Hour)
		return start, end, true
	}

	// "最近X天" pattern
	recentPattern := regexp.MustCompile(`最近\s*(\d+)\s*[天日]`)
	matches := recentPattern.FindStringSubmatch(normalized)
	if len(matches) > 1 {
		days := parseIntSafe(matches[1])
		if days > 0 {
			start = now.AddDate(0, 0, -days)
			start = time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, now.Location())
			end = now
			return start, end, true
		}
	}

	return time.Time{}, time.Time{}, false
}

// ExtractTags extracts tag filters from a query.
// ExtractTags 从查询中提取标签过滤器。
func (c *MemoQueryClassifier) ExtractTags(query string) []string {
	tags := []string{}

	// Match #tag pattern (including Unicode Chinese characters)
	tagPattern := regexp.MustCompile(`#([\w\p{Han}-]+)`)
	matches := tagPattern.FindAllStringSubmatch(query, -1)

	for _, match := range matches {
		if len(match) > 1 {
			tags = append(tags, match[1])
		}
	}

	return tags
}

// parseIntSafe parses integer safely.
func parseIntSafe(s string) int {
	var result int
	for _, ch := range s {
		if ch >= '0' && ch <= '9' {
			result = result*10 + int(ch-'0')
		} else {
			break
		}
	}
	return result
}
