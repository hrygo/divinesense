// Package routing provides the LLM routing service.
package routing

import (
	"context"
	"regexp"
	"strings"
	"sync"
	"unicode"
)

// KeywordCapabilitySource defines an interface for dynamic keyword loading.
// This avoids import cycles between routing and orchestrator packages.
type KeywordCapabilitySource interface {
	IdentifyCapabilities(text string) []string
}

// RoutingMatcher defines an interface for configuration-driven routing.
// This enables Layer 2 rule-based routing using keywords from config.
type RoutingMatcher interface {
	// MatchInput matches input against the keyword index.
	// Returns sorted expert names and match confidence.
	MatchInput(input string) ([]string, float64)
}

// SemanticMatcher defines an interface for semantic routing using embeddings.
// This enables Layer 3 semantic routing when Layer 2 rule matching fails.
type SemanticMatcher interface {
	// MatchSemantic matches input using semantic similarity.
	// Returns expert name and confidence.
	// Note: The implementation should handle embedding generation internally.
	MatchSemantic(ctx context.Context, input string) ([]string, float64)
}

// EmbeddingProvider defines an interface for generating embeddings.
type EmbeddingProvider interface {
	// Embed generates an embedding vector for the given text.
	Embed(ctx context.Context, text string) ([]float32, error)
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
	capabilityMap   KeywordCapabilitySource // Dynamic capability map for keyword loading
	routingMatcher  RoutingMatcher          // Configuration-driven routing (Layer 2)
	semanticMatcher SemanticMatcher         // Semantic routing (Layer 3)
	timePatterns    []*regexp.Regexp
	// User-specific custom weights (optional, for dynamic adjustment)
	customWeights   map[int32]map[string]map[string]int // userID -> category -> keyword -> weight
	customWeightsMu sync.RWMutex
}

// NewRuleMatcher creates a new rule matcher.
// Requires CapabilityMap to be set via SetCapabilityMap for keyword matching.
func NewRuleMatcher() *RuleMatcher {
	return &RuleMatcher{
		customWeights: make(map[int32]map[string]map[string]int),
		// Time patterns for schedule detection
		timePatterns: []*regexp.Regexp{
			regexp.MustCompile(`\d{1,2}[:\s时点]\d{0,2}`),       // 10:30, 10点, 10时30
			regexp.MustCompile(`(上午|下午|晚上|早上|中午)\d{1,2}[点时]`), // 下午3点
			regexp.MustCompile(`(明天|后天|今天|下周|本周)`),            // Relative dates
			regexp.MustCompile(`\d{1,2}月\d{1,2}[日号]`),         // 1月15日
		},
	}
}

// SetCapabilityMap sets the capability map for dynamic keyword loading.
// This enables the RuleMatcher to load keywords from configured capabilities instead of hardcoded values.
func (m *RuleMatcher) SetCapabilityMap(capMap KeywordCapabilitySource) {
	m.capabilityMap = capMap
}

// SetRoutingMatcher sets the routing matcher for configuration-driven routing.
// This enables Layer 2 rule-based routing using keywords from config.
func (m *RuleMatcher) SetRoutingMatcher(router RoutingMatcher) {
	m.routingMatcher = router
}

// SetSemanticMatcher sets the semantic matcher for Layer 3 semantic routing.
// This enables embedding-based routing when Layer 2 rule matching fails.
func (m *RuleMatcher) SetSemanticMatcher(matcher SemanticMatcher) {
	m.semanticMatcher = matcher
}

// Match performs rule-based pattern matching and returns generic action + matched keywords.
// This is SOLID compliant: RuleMatcher only recognizes patterns, not expert types.
// The mapping from GenericAction + Keywords to Expert is handled by IntentRegistry/ExpertRouter.
func (m *RuleMatcher) Match(input string) *MatchResult {
	// If no capabilityMap, still can do generic pattern matching
	lower := m.normalizeInput(input)

	// 1. Detect generic action using pre-compiled regex patterns (domain-agnostic)
	action := m.detectGenericAction(lower)

	// 2. Get matched keywords from CapabilityMap (dynamic, config-driven)
	keywords := m.getMatchedKeywords(input)

	// 3. Calculate confidence based on matches
	confidence := m.calculateMatchConfidence(action, keywords)

	// 4. Return match result
	// If no action detected and no keywords matched, return no match
	if action == ActionNone && len(keywords) == 0 {
		return &MatchResult{
			Action:     ActionNone,
			Keywords:   nil,
			Confidence: 0,
			Matched:    false,
		}
	}

	return &MatchResult{
		Action:     action,
		Keywords:   keywords,
		Confidence: confidence,
		Matched:    true,
	}
}

// Legacy Match method for backward compatibility.
// Returns: intent, confidence, matched (true if rule matched).
//
// Deprecated: Use Match() which returns *MatchResult instead.
func (m *RuleMatcher) MatchLegacy(input string) (Intent, float32, bool) {
	result := m.Match(input)
	if !result.Matched {
		return IntentUnknown, 0, false
	}

	// Convert GenericAction to Intent using registry
	// This is a fallback for backward compatibility
	intent := m.GenericActionToIntent(result.Action, result.Keywords, input)
	return intent, result.Confidence, true
}

// detectGenericAction detects the generic action type from input using regex patterns.
// This is completely domain-agnostic - no hardcoded expert types.
func (m *RuleMatcher) detectGenericAction(input string) GenericAction {
	// Check patterns in order of specificity
	if updatePatternRegex.MatchString(input) {
		return ActionUpdate
	}
	if batchPatternRegex.MatchString(input) {
		return ActionBatch
	}
	if searchPatternRegex.MatchString(input) {
		return ActionSearch
	}
	if queryPatternRegex.MatchString(input) {
		return ActionQuery
	}
	if createPatternRegex.MatchString(input) {
		return ActionCreate
	}

	// If has time pattern but no action, default to query (common for schedule queries)
	if m.hasTimePattern(input) {
		return ActionQuery
	}

	return ActionNone
}

// getMatchedKeywords returns all matched trigger keywords from CapabilityMap.
func (m *RuleMatcher) getMatchedKeywords(input string) []string {
	if m.capabilityMap == nil {
		return nil
	}

	capabilities := m.capabilityMap.IdentifyCapabilities(input)
	// Capabilities are the matched keywords/categories from config
	// Return them as-is for downstream routing
	return capabilities
}

// calculateMatchConfidence calculates confidence based on matched patterns and keywords.
func (m *RuleMatcher) calculateMatchConfidence(action GenericAction, keywords []string) float32 {
	var confidence float32 = 0.5 // Base confidence

	// Higher confidence if action detected
	if action != ActionNone {
		confidence += 0.3
	}

	// Higher confidence if keywords matched
	if len(keywords) > 0 {
		confidence += float32(len(keywords)) * 0.1
	}

	// Cap at 0.95
	if confidence > 0.95 {
		confidence = 0.95
	}

	return confidence
}

// GenericActionToIntent converts GenericAction to Intent for backward compatibility.
// This is a temporary bridge - in the new architecture, routing is handled by IntentRegistry.
// Input is provided to detect implicit schedule intent from time patterns.
func (m *RuleMatcher) GenericActionToIntent(action GenericAction, keywords []string, input string) Intent {
	// Check keywords for domain hints (this is the last remaining hardcoded part)
	// In the new architecture, this should be handled by IntentRegistry
	hasScheduleHint := false
	hasMemoHint := false
	for _, kw := range keywords {
		kwLower := strings.ToLower(kw)
		if containsScheduleKeyword(kwLower) {
			hasScheduleHint = true
		}
		if containsMemoKeyword(kwLower) {
			hasMemoHint = true
		}
	}

	// Check for implicit schedule intent from time patterns
	// If input has time pattern but no explicit action, assume schedule create
	hasTimePattern := m.hasTimePattern(input)

	// Determine intent based on action and hints
	// Priority: explicit hints (schedule/memo keywords) > default action mapping
	switch action {
	case ActionSearch:
		// If has schedule hint, it's likely a schedule query/search
		if hasScheduleHint {
			return IntentScheduleQuery
		}
		if hasMemoHint {
			return IntentMemoSearch
		}
		// Default to memo search for search action
		return IntentMemoSearch
	case ActionCreate:
		if hasMemoHint {
			return IntentMemoCreate
		}
		if hasScheduleHint || hasTimePattern {
			return IntentScheduleCreate
		}
		return IntentScheduleCreate
	case ActionQuery:
		// If has time pattern but action is query, it's likely a schedule query
		// But if no explicit schedule keywords, could be schedule query
		if hasScheduleHint || hasTimePattern {
			return IntentScheduleQuery
		}
		return IntentScheduleQuery
	case ActionUpdate:
		return IntentScheduleUpdate
	case ActionBatch:
		return IntentBatchSchedule
	default:
		// No explicit action, but has time pattern → schedule query
		if hasTimePattern {
			return IntentScheduleQuery
		}
		return IntentUnknown
	}
}

// calculateDynamicScore calculates scores by matching input capabilities.
// This is truly dynamic - RuleMatcher doesn't know about specific expert types.
// Each expert defines its capabilities via configuration.
func (m *RuleMatcher) calculateDynamicScore(input string) (scheduleScore, memoScore int) {
	if m.capabilityMap == nil {
		return 0, 0
	}

	// Get all capabilities from input
	capabilities := m.capabilityMap.IdentifyCapabilities(input)

	// Score based on capabilities matched - check if capability contains schedule/memo related terms
	// This is still a hint but the capability names come from config, not hardcoded
	for _, cap := range capabilities {
		capLower := strings.ToLower(cap)
		// Check if this capability is schedule-related (name from config)
		if containsScheduleKeyword(capLower) {
			scheduleScore += 2
		}
		// Check if this capability is memo-related (name from config)
		if containsMemoKeyword(capLower) {
			memoScore += 2
		}
	}
	return scheduleScore, memoScore
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
// Uses dynamic capabilityMap to determine keywords.
func (m *RuleMatcher) hasCoreKeyword(input, category string) bool {
	if m.capabilityMap == nil {
		return false
	}
	capabilities := m.capabilityMap.IdentifyCapabilities(input)
	for _, cap := range capabilities {
		if m.capabilityMatchesCategory(cap, category) {
			return true
		}
	}
	return false
}

// capabilityMatchesCategory checks if a capability matches the given category.
// This maps capability names to rule matcher categories.
func (m *RuleMatcher) capabilityMatchesCategory(capability, category string) bool {
	capLower := strings.ToLower(capability)

	switch category {
	case "schedule":
		return containsScheduleKeyword(capLower)
	case "memo":
		return containsMemoKeyword(capLower)
	}
	return false
}

// containsScheduleKeyword checks if text contains any schedule-related keywords.
// Extracted to eliminate DRY violation.
func containsScheduleKeyword(text string) bool {
	return strings.Contains(text, "日程") ||
		strings.Contains(text, "schedule") ||
		strings.Contains(text, "会议") ||
		strings.Contains(text, "提醒") ||
		strings.Contains(text, "批量") ||
		strings.Contains(text, "创建")
}

// containsMemoKeyword checks if text contains any memo-related keywords.
// Extracted to eliminate DRY violation.
func containsMemoKeyword(text string) bool {
	return strings.Contains(text, "笔记") ||
		strings.Contains(text, "memo") ||
		strings.Contains(text, "搜索") ||
		strings.Contains(text, "记录")
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

// SetCustomWeights sets custom weights for a specific user.
// This allows dynamic weight adjustment based on user feedback.
func (m *RuleMatcher) SetCustomWeights(userID int32, weights map[string]map[string]int) {
	m.customWeightsMu.Lock()
	defer m.customWeightsMu.Unlock()
	m.customWeights[userID] = weights
}

// GetCustomWeights retrieves custom weights for a specific user.
func (m *RuleMatcher) GetCustomWeights(userID int32) map[string]map[string]int {
	m.customWeightsMu.RLock()
	defer m.customWeightsMu.RUnlock()
	if w, ok := m.customWeights[userID]; ok {
		// Return a copy to avoid concurrent modification
		result := make(map[string]map[string]int, len(w))
		for cat, kw := range w {
			result[cat] = make(map[string]int, len(kw))
			for k, v := range kw {
				result[cat][k] = v
			}
		}
		return result
	}
	return nil
}

// KeywordExpertSource defines an interface for getting keywords associated with an expert.
// This enables HILT feedback to adjust weights for specific keywords.
// 避免循环导入：在 routing 包中定义接口，由 orchestrator.CapabilityMap 实现。
type KeywordExpertSource interface {
	GetKeywordsForExpert(expertName string) []string
}

// getKeywordsForCategory returns the list of keywords for a given category.
// This is used by the feedback collector to identify which keywords to adjust.
// Returns keywords from capabilityMap if available, otherwise returns nil.
func (m *RuleMatcher) getKeywordsForCategory(category string) []string {
	// Try to get keywords from capabilityMap if it implements KeywordExpertSource
	if kes, ok := m.capabilityMap.(KeywordExpertSource); ok {
		return kes.GetKeywordsForExpert(category)
	}
	return nil
}

// GetKeywordWeight returns the weight for a keyword, using custom weights if available.
// Returns 0 if no custom weight is set and no capabilityMap is available.
func (m *RuleMatcher) GetKeywordWeight(userID int32, category, keyword string) int {
	m.customWeightsMu.RLock()
	defer m.customWeightsMu.RUnlock()

	// Check for custom weight first
	if custom, ok := m.customWeights[userID]; ok {
		if catWeights, ok := custom[category]; ok {
			if weight, ok := catWeights[keyword]; ok {
				return weight
			}
		}
	}

	// No default weight without capabilityMap
	return 0
}

// MatchWithUser matches input with user-specific custom weights.
// This is the enhanced version of Match that uses dynamic weights.
func (m *RuleMatcher) MatchWithUser(input string, userID int32) (Intent, float32, bool) {
	return m.MatchWithContext(context.Background(), input, userID)
}

// MatchWithContext matches input with context support for semantic routing.
// This enables Layer 3 semantic routing when Layer 2 rule matching fails.
func (m *RuleMatcher) MatchWithContext(ctx context.Context, input string, userID int32) (Intent, float32, bool) {
	// Require capabilityMap for matching
	if m.capabilityMap == nil {
		return IntentUnknown, 0, false
	}

	// Layer 2: CONFIGURATION-DRIVEN PATH: Use routing matcher
	// This is the preferred path - it uses keywords from YAML config
	if m.routingMatcher != nil {
		matchedExperts, confidence := m.routingMatcher.MatchInput(input)
		if len(matchedExperts) > 0 && confidence > 0.3 {
			intent := m.expertToIntent(matchedExperts[0], input)
			return intent, float32(confidence), true
		}
	}

	// Layer 3: SEMANTIC PATH: Use semantic matcher if Layer 2 failed
	// This uses embeddings for semantic similarity
	if m.semanticMatcher != nil {
		matchedExperts, confidence := m.semanticMatcher.MatchSemantic(ctx, input)
		if len(matchedExperts) > 0 && confidence > 0.3 {
			intent := m.expertToIntent(matchedExperts[0], input)
			return intent, float32(confidence), true
		}
	}

	// Fallback: legacy dynamic scoring (for backward compatibility)
	return m.matchWithLegacyScoring(input)
}

// matchWithLegacyScoring provides backward compatibility for dynamic scoring.
// This is the fallback when no routing matcher is configured.
func (m *RuleMatcher) matchWithLegacyScoring(input string) (Intent, float32, bool) {
	// Fast path: normalize once
	lower := m.normalizeInput(input)

	// Calculate scores dynamically from capabilityMap
	scheduleScore, memoScore := m.calculateDynamicScore(lower)

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

	// No match - needs higher layer processing
	return IntentUnknown, 0, false
}

// expertToIntent converts an expert name to an Intent.
// This uses heuristics based on the expert name and input content.
func (m *RuleMatcher) expertToIntent(expertName, input string) Intent {
	expertLower := strings.ToLower(expertName)
	lower := m.normalizeInput(input)

	// Determine base intent from action patterns
	action := m.detectGenericAction(lower)

	switch expertLower {
	case "memo":
		// Memo expert
		if action == ActionSearch || action == ActionNone {
			return IntentMemoSearch
		}
		return IntentMemoCreate
	case "schedule":
		// Schedule expert
		switch action {
		case ActionUpdate:
			return IntentScheduleUpdate
		case ActionBatch:
			return IntentBatchSchedule
		case ActionQuery:
			return IntentScheduleQuery
		default:
			// Default: if has time pattern, it's a schedule intent
			if m.hasTimePattern(input) {
				return IntentScheduleCreate
			}
			return IntentScheduleQuery
		}
	default:
		// Unknown expert - determine based on action
		if action == ActionSearch || action == ActionCreate {
			return IntentMemoSearch // Default to memo for unknown
		}
		return IntentScheduleQuery // Default to schedule for unknown
	}
}
