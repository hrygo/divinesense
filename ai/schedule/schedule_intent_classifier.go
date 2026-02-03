// Package schedule provides schedule-related AI agent utilities.
package schedule

import (
	"context"
	"regexp"
	"strings"

	"github.com/hrygo/divinesense/ai/router"
)

// ScheduleIntent represents the type of schedule task intent.
type ScheduleIntent int

const (
	// IntentUnknown is for unrecognized intents.
	IntentUnknown ScheduleIntent = iota
	// IntentSimpleCreate is for simple single schedule creation.
	IntentSimpleCreate
	// IntentSimpleQuery is for simple schedule queries.
	IntentSimpleQuery
	// IntentSimpleUpdate is for simple schedule modifications.
	IntentSimpleUpdate
	// IntentBatchCreate is for batch schedule creation (e.g., "每天", "每周").
	IntentBatchCreate
)

// String returns the string representation of ScheduleIntent.
func (i ScheduleIntent) String() string {
	switch i {
	case IntentSimpleCreate:
		return "simple_create"
	case IntentSimpleQuery:
		return "simple_query"
	case IntentSimpleUpdate:
		return "simple_update"
	case IntentBatchCreate:
		return "batch_create"
	default:
		return "unknown"
	}
}

// Pre-compiled regex patterns for intent classification.
var (
	// Create patterns: time + action/event.
	createPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(明天|后天|下周|今天|大后天).*(点|时).*(开会|会议|面试|约|见|做|去)`),
		regexp.MustCompile(`(上午|下午|晚上|早上|中午).*(安排|约|预约|开会|会议)`),
		regexp.MustCompile(`安排.*(会议|面试|约会|活动|事情)`),
		regexp.MustCompile(`(预约|约).*(时间|面试|会议)`),
		regexp.MustCompile(`(帮我|请|给我).*(安排|创建|添加|新建).*(日程|会议|事项)`),
		regexp.MustCompile(`(\d{1,2})[点时:].*(开会|会议|面试|约|见|做)`),
	}

	// Query patterns: question words + time.
	queryPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(今天|明天|这周|下周|本周|后天).*(有什么|什么安排|忙吗|有空|空闲|安排了什么)`),
		regexp.MustCompile(`(查|看|显示|列出).*(日程|安排|计划|会议)`),
		regexp.MustCompile(`(几点|什么时候).*(会|开始|结束|有空)`),
		regexp.MustCompile(`(有没有|有无).*(安排|日程|会议|事情)`),
		regexp.MustCompile(`(今天|明天|这周|下周)的?(日程|安排)$`),
		regexp.MustCompile(`安排了(什么|啥)`),
		regexp.MustCompile(`\?$|？$`), // Ends with question mark
	}

	// Update patterns: modification verbs + target.
	updatePatterns = []*regexp.Regexp{
		regexp.MustCompile(`(改|换|调|推迟|提前|取消|删除|延后|调整).*(会议|日程|安排|时间)`),
		regexp.MustCompile(`把.*([改换调移])到`),
		regexp.MustCompile(`(取消|删除|移除).*(这个|那个|上午|下午)?(会议|日程|安排)?`),
		regexp.MustCompile(`(会议|日程|安排).*(改|换|调|推迟|提前|取消|删除|延后)`),
		regexp.MustCompile(`(会议|日程).*(延后|推迟|提前)`),
	}

	// Batch patterns: repetition keywords.
	batchPatterns = []*regexp.Regexp{
		regexp.MustCompile(`每([天日周月年])`),
		regexp.MustCompile(`工作日`),
		regexp.MustCompile(`周[一二三四五六日](到|至)周[一二三四五六日]`),
		regexp.MustCompile(`(连续|接下来).*\d+.*天`),
		regexp.MustCompile(`([这下本])周(所有|每天|每日)`),
		regexp.MustCompile(`[从自].*[到至].*每`),
	}

	// Pre-compiled patterns for hasTimeAndAction.
	viewPattern         = regexp.MustCompile(`(查|看|显示|列出).*(安排|日程)`)
	arrangeVerbPattern  = regexp.MustCompile(`安排.*(会议|面试|约会|活动|事情|一下)`)
	specificTimePattern = regexp.MustCompile(`\d{1,2}[点时:]`)

	// Keywords for fallback detection.
	timeKeywords   = []string{"今天", "明天", "后天", "下周", "本周", "这周", "上午", "下午", "晚上", "早上", "中午", "点", "时"}
	createKeywords = []string{"开会", "会议", "面试", "约", "安排", "预约", "创建", "添加", "新建"}
	queryKeywords  = []string{"有什么", "什么安排", "忙吗", "有空", "查", "看", "显示", "列出", "有没有"}
	updateKeywords = []string{"改", "换", "调", "推迟", "提前", "取消", "删除", "延后", "调整", "移"}
	batchKeywords  = []string{"每天", "每日", "每周", "每月", "每年", "工作日"}
)

// ScheduleIntentClassifier classifies user input into schedule task intents.
// It uses rule-based matching first (0ms) and falls back to RouterService (~400ms).
type ScheduleIntentClassifier struct {
	routerService router.RouterService
}

// NewScheduleIntentClassifier creates a new ScheduleIntentClassifier.
func NewScheduleIntentClassifier(routerService router.RouterService) *ScheduleIntentClassifier {
	return &ScheduleIntentClassifier{
		routerService: routerService,
	}
}

// ClassifyResult holds the classification result.
type ClassifyResult struct {
	Intent     ScheduleIntent
	Confidence float32
	UsedLLM    bool // Whether LLM was used for classification
}

// Classify determines the schedule intent of the user input.
// Returns the intent, confidence score (0-1), and whether LLM was used.
func (c *ScheduleIntentClassifier) Classify(ctx context.Context, input string) ClassifyResult {
	// Normalize input
	normalizedInput := strings.ToLower(input)

	// Step 1: Try batch patterns first (highest priority for schedule domain)
	if c.matchPatterns(input, normalizedInput, batchPatterns) {
		return ClassifyResult{Intent: IntentBatchCreate, Confidence: 0.95, UsedLLM: false}
	}

	// Step 2: Try update patterns
	if c.matchPatterns(input, normalizedInput, updatePatterns) {
		return ClassifyResult{Intent: IntentSimpleUpdate, Confidence: 0.9, UsedLLM: false}
	}

	// Step 3: Try query patterns
	if c.matchPatterns(input, normalizedInput, queryPatterns) {
		// Make sure it's not a create with time+action
		if !c.hasTimeAndAction(input, normalizedInput) {
			return ClassifyResult{Intent: IntentSimpleQuery, Confidence: 0.9, UsedLLM: false}
		}
	}

	// Step 4: Try create patterns
	if c.matchPatterns(input, normalizedInput, createPatterns) {
		return ClassifyResult{Intent: IntentSimpleCreate, Confidence: 0.9, UsedLLM: false}
	}

	// Step 5: Keyword-based fallback
	if result := c.keywordFallback(input, normalizedInput); result.Intent != IntentUnknown {
		return result
	}

	// Step 6: Use RouterService as final fallback
	if c.routerService != nil {
		return c.routerFallback(ctx, input)
	}

	return ClassifyResult{Intent: IntentUnknown, Confidence: 0, UsedLLM: false}
}

// matchPatterns checks if input matches any of the given patterns.
func (c *ScheduleIntentClassifier) matchPatterns(input, normalizedInput string, patterns []*regexp.Regexp) bool {
	for _, pattern := range patterns {
		if pattern.MatchString(input) || pattern.MatchString(normalizedInput) {
			return true
		}
	}
	return false
}

// hasTimeAndAction checks if input has both time reference and action.
// This distinguishes "今天有什么" (query) from "今天3点开会" (create).
func (c *ScheduleIntentClassifier) hasTimeAndAction(input, normalizedInput string) bool {
	// Check for query indicators first - these override time+action detection
	queryIndicators := []string{"有什么", "什么安排", "安排了什么", "安排了啥", "忙吗", "有空", "空闲"}
	for _, indicator := range queryIndicators {
		if strings.Contains(input, indicator) {
			return false
		}
	}

	// Check for display/view patterns - "显示...安排" is query, not create
	if viewPattern.MatchString(input) {
		return false
	}

	hasTime := false
	hasAction := false

	for _, keyword := range timeKeywords {
		if strings.Contains(input, keyword) || strings.Contains(normalizedInput, keyword) {
			hasTime = true
			break
		}
	}

	// For action, exclude "安排" when it's used as a noun (target of query)
	// Only count it as action when it's a verb (e.g., "安排会议")
	actionKeywordsForCreate := []string{"开会", "会议", "面试", "约", "预约", "创建", "添加", "新建"}
	for _, keyword := range actionKeywordsForCreate {
		if strings.Contains(input, keyword) || strings.Contains(normalizedInput, keyword) {
			hasAction = true
			break
		}
	}

	// Check if "安排" is used as a verb (followed by object)
	if !hasAction && strings.Contains(input, "安排") {
		if arrangeVerbPattern.MatchString(input) {
			hasAction = true
		}
	}

	// Also check for specific time patterns (hour:minute)
	if !hasTime {
		hasTime = specificTimePattern.MatchString(input)
	}

	return hasTime && hasAction
}

// keywordFallback uses keyword counting for uncertain cases.
func (c *ScheduleIntentClassifier) keywordFallback(input, normalizedInput string) ClassifyResult {
	scores := map[ScheduleIntent]int{
		IntentSimpleCreate: 0,
		IntentSimpleQuery:  0,
		IntentSimpleUpdate: 0,
		IntentBatchCreate:  0,
	}

	// Count batch keywords (highest weight)
	for _, keyword := range batchKeywords {
		if strings.Contains(input, keyword) {
			scores[IntentBatchCreate] += 3
		}
	}

	// Count update keywords
	for _, keyword := range updateKeywords {
		if strings.Contains(input, keyword) {
			scores[IntentSimpleUpdate] += 2
		}
	}

	// Count query keywords
	for _, keyword := range queryKeywords {
		if strings.Contains(input, keyword) {
			scores[IntentSimpleQuery] += 2
		}
	}

	// Count create keywords + time keywords
	createScore := 0
	for _, keyword := range createKeywords {
		if strings.Contains(input, keyword) {
			createScore++
		}
	}
	timeScore := 0
	for _, keyword := range timeKeywords {
		if strings.Contains(input, keyword) {
			timeScore++
		}
	}
	if createScore > 0 && timeScore > 0 {
		scores[IntentSimpleCreate] = createScore + timeScore
	}

	// Find highest score
	maxScore := 0
	maxIntent := IntentUnknown
	for intent, score := range scores {
		if score > maxScore {
			maxScore = score
			maxIntent = intent
		}
	}

	// Require minimum score threshold
	if maxScore >= 2 {
		confidence := float32(0.6) + float32(maxScore)*0.05
		if confidence > 0.85 {
			confidence = 0.85
		}
		return ClassifyResult{Intent: maxIntent, Confidence: confidence, UsedLLM: false}
	}

	return ClassifyResult{Intent: IntentUnknown, Confidence: 0, UsedLLM: false}
}

// routerFallback uses RouterService for classification.
func (c *ScheduleIntentClassifier) routerFallback(ctx context.Context, input string) ClassifyResult {
	intent, confidence, err := c.routerService.ClassifyIntent(ctx, input)
	if err != nil {
		return ClassifyResult{Intent: IntentUnknown, Confidence: 0, UsedLLM: true}
	}

	return ClassifyResult{
		Intent:     c.mapRouterIntent(intent),
		Confidence: confidence,
		UsedLLM:    true,
	}
}

// mapRouterIntent maps router.Intent to ScheduleIntent.
func (c *ScheduleIntentClassifier) mapRouterIntent(intent router.Intent) ScheduleIntent {
	switch intent {
	case router.IntentScheduleCreate:
		return IntentSimpleCreate
	case router.IntentScheduleQuery:
		return IntentSimpleQuery
	case router.IntentScheduleUpdate:
		return IntentSimpleUpdate
	case router.IntentBatchSchedule:
		return IntentBatchCreate
	default:
		return IntentUnknown
	}
}

// ShouldUsePlanExecute returns true if the intent should use Plan-Execute mode.
func (c *ScheduleIntentClassifier) ShouldUsePlanExecute(intent ScheduleIntent) bool {
	return intent == IntentBatchCreate
}

// ClassifyAndRoute is a convenience method that classifies and returns execution mode.
func (c *ScheduleIntentClassifier) ClassifyAndRoute(ctx context.Context, input string) (ScheduleIntent, bool, float32) {
	result := c.Classify(ctx, input)
	usePlanExecute := c.ShouldUsePlanExecute(result.Intent)
	return result.Intent, usePlanExecute, result.Confidence
}
