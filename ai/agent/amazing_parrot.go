package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hrygo/divinesense/ai"
	"github.com/hrygo/divinesense/ai/agent/tools"
	"github.com/hrygo/divinesense/ai/core/retrieval"
	"github.com/hrygo/divinesense/ai/timeout"
	"github.com/hrygo/divinesense/server/service/schedule"
)

// Constants for AmazingParrot configuration.
const (
	// concurrentRetrievalTimeout is the maximum time to wait for concurrent retrievals.
	concurrentRetrievalTimeout = 45 * time.Second

	// casualChatShortThreshold is the character length below which input is considered casual chat.
	casualChatShortThreshold = 30

	// casualChatModerateThreshold is the character length below which non-search input is considered casual.
	casualChatModerateThreshold = 100
)

// AmazingParrot is the comprehensive assistant parrot (🦜 折衷).
// AmazingParrot 是综合助手鹦鹉（🦜 折衷）。
// It combines memo and schedule capabilities for integrated assistance.
type AmazingParrot struct {
	llm                ai.LLMService
	cache              *LRUCache
	memoSearchTool     *tools.MemoSearchTool
	scheduleQueryTool  *tools.ScheduleQueryTool
	scheduleAddTool    *tools.ScheduleAddTool
	findFreeTimeTool   *tools.FindFreeTimeTool
	scheduleUpdateTool *tools.ScheduleUpdateTool
	userID             int32
}

// retrievalPlan represents the plan for concurrent retrieval.
type retrievalPlan struct {
	memoSearchQuery     string
	scheduleStartTime   string
	scheduleEndTime     string
	freeTimeDate        string
	scheduleAddParams   string
	needsMemoSearch     bool
	needsScheduleQuery  bool
	needsScheduleAdd    bool
	needsFreeTime       bool
	needsScheduleUpdate bool
	needsDirectAnswer   bool
}

// NewAmazingParrot creates a new amazing parrot agent.
// NewAmazingParrot 创建一个新的综合助手鹦鹉（折衷）。
func NewAmazingParrot(
	llm ai.LLMService,
	retriever *retrieval.AdaptiveRetriever,
	scheduleService schedule.Service,
	userID int32,
) (*AmazingParrot, error) {
	if llm == nil {
		return nil, fmt.Errorf("llm cannot be nil")
	}
	if retriever == nil {
		return nil, fmt.Errorf("retriever cannot be nil")
	}
	if scheduleService == nil {
		return nil, fmt.Errorf("scheduleService cannot be nil")
	}

	// Create user ID getter
	userIDGetter := func(_ context.Context) int32 {
		return userID
	}

	// Initialize tools
	memoSearchTool, err := tools.NewMemoSearchTool(retriever, userIDGetter)
	if err != nil {
		return nil, fmt.Errorf("failed to create memo search tool: %w", err)
	}
	scheduleQueryTool := tools.NewScheduleQueryTool(scheduleService, userIDGetter)
	scheduleAddTool := tools.NewScheduleAddTool(scheduleService, userIDGetter)
	findFreeTimeTool := tools.NewFindFreeTimeTool(scheduleService, userIDGetter)
	scheduleUpdateTool := tools.NewScheduleUpdateTool(scheduleService, userIDGetter)

	return &AmazingParrot{
		llm:                llm,
		cache:              NewLRUCache(DefaultCacheEntries, DefaultCacheTTL),
		userID:             userID,
		memoSearchTool:     memoSearchTool,
		scheduleQueryTool:  scheduleQueryTool,
		scheduleAddTool:    scheduleAddTool,
		findFreeTimeTool:   findFreeTimeTool,
		scheduleUpdateTool: scheduleUpdateTool,
	}, nil
}

// Name returns the name of the parrot.
// Name 返回鹦鹉名称。
func (*AmazingParrot) Name() string {
	return "amazing" // ParrotAgentType AGENT_TYPE_AMAZING
}

// recordMetrics records prompt usage metrics for the amazing agent.
func (p *AmazingParrot) recordMetrics(startTime time.Time, promptVersion PromptVersion, success bool) {
	latencyMs := time.Since(startTime).Milliseconds()
	RecordPromptUsageInMemory(p.Name(), promptVersion, success, latencyMs)
}

// ExecuteWithCallback executes the amazing parrot with callback support.
// ExecuteWithCallback 执行综合助手鹦鹉（折衷）并支持回调.
//
// Implementation: Two-phase concurrent retrieval for optimal performance.
// Phase 1: Analyze user intent and plan concurrent retrievals.
// Phase 2: Execute retrievals concurrently, then synthesize answer.
func (p *AmazingParrot) ExecuteWithCallback(
	ctx context.Context,
	userInput string,
	history []string,
	callback EventCallback,
) error {
	// Add timeout protection
	ctx, cancel := context.WithTimeout(ctx, timeout.AgentTimeout)
	defer cancel()

	startTime := time.Now()

	// Get prompt version for AB testing
	promptVersion := GetPromptVersionForUser(p.Name(), p.userID)

	// Create safe callback for non-critical events (logs errors but doesn't propagate)
	callbackSafe := SafeCallback(callback)

	// Log execution start
	slog.Info("AmazingParrot: ExecuteWithCallback started",
		"user_id", p.userID,
		"input", truncateString(userInput, 100),
		"history_count", len(history),
		"prompt_version", promptVersion,
	)

	// Step 1: Check cache
	cacheKey := GenerateCacheKey(p.Name(), p.userID, userInput)
	if cachedResult, found := p.cache.Get(cacheKey); found {
		if result, ok := cachedResult.(string); ok {
			slog.Info("AmazingParrot: Cache hit", "user_id", p.userID)
			if callbackSafe != nil {
				callbackSafe(EventTypeAnswer, result)
			}
			p.recordMetrics(startTime, promptVersion, true)
			return nil
		}
	}
	slog.Debug("AmazingParrot: Cache miss, proceeding with execution", "user_id", p.userID)

	// Step 2: Plan concurrent retrieval using LLM intent analysis
	slog.Debug("AmazingParrot: Starting planning phase", "user_id", p.userID)
	plan, err := p.planRetrieval(ctx, userInput, history, callback)
	if err != nil {
		slog.Error("AmazingParrot: Planning failed", "user_id", p.userID, "error", err)
		p.recordMetrics(startTime, promptVersion, false)
		return NewParrotError(p.Name(), "planRetrieval", err)
	}
	slog.Info("AmazingParrot: Plan created",
		"user_id", p.userID,
		"needs_memo_search", plan.needsMemoSearch,
		"needs_schedule_query", plan.needsScheduleQuery,
		"needs_free_time", plan.needsFreeTime,
		"needs_schedule_add", plan.needsScheduleAdd,
		"needs_schedule_update", plan.needsScheduleUpdate,
		"needs_direct_answer", plan.needsDirectAnswer,
	)

	// Step 3: Execute concurrent retrieval (skip for direct answer/casual chat)
	var retrievalResults map[string]string
	if plan.needsDirectAnswer {
		slog.Info("AmazingParrot: Skipping retrieval for direct answer", "user_id", p.userID)
		retrievalResults = make(map[string]string)
	} else {
		slog.Debug("AmazingParrot: Starting concurrent retrieval", "user_id", p.userID)
		retrievalResults, err = p.executeConcurrentRetrieval(ctx, plan, callback)
		if err != nil {
			slog.Error("AmazingParrot: Concurrent retrieval failed", "user_id", p.userID, "error", err)
			p.recordMetrics(startTime, promptVersion, false)
			return NewParrotError(p.Name(), "executeConcurrentRetrieval", err)
		}
		slog.Info("AmazingParrot: Retrieval completed",
			"user_id", p.userID,
			"results_count", len(retrievalResults),
		)
	}

	// Step 4: Synthesize final answer from retrieval results streaming
	slog.Debug("AmazingParrot: Starting synthesis", "user_id", p.userID)
	finalAnswer, err := p.synthesizeAnswer(ctx, userInput, history, retrievalResults, callback)
	if err != nil {
		slog.Error("AmazingParrot: Synthesis failed", "user_id", p.userID, "error", err)
		p.recordMetrics(startTime, promptVersion, false)
		return NewParrotError(p.Name(), "synthesizeAnswer", err)
	}

	// Cache answer
	p.cache.Set(cacheKey, finalAnswer)

	slog.Info("AmazingParrot: Execution completed successfully",
		"user_id", p.userID,
		"duration_ms", time.Since(startTime).Milliseconds(),
		"answer_length", len(finalAnswer),
	)

	p.recordMetrics(startTime, promptVersion, true)
	return nil
}

// planRetrieval analyzes user input and creates a concurrent retrieval plan.
func (p *AmazingParrot) planRetrieval(ctx context.Context, userInput string, history []string, callback EventCallback) (*retrievalPlan, error) {
	callbackSafe := SafeCallback(callback)
	if callbackSafe != nil {
		callbackSafe(EventTypeThinking, "正在分析您的需求...")
	}

	now := time.Now()
	// Build planning prompt (optimized for minimal tokens)
	planningPrompt := p.buildPlanningPrompt(now)

	messages := []ai.Message{
		{Role: "system", Content: planningPrompt},
	}

	// Add history for context (even in planning)
	for i := 0; i < len(history)-1; i += 2 {
		if i+1 < len(history) {
			userMsg := history[i]
			assistantMsg := history[i+1]
			// Only add non-empty messages
			if userMsg != "" && assistantMsg != "" {
				messages = append(messages, ai.Message{Role: "user", Content: userMsg})
				messages = append(messages, ai.Message{Role: "assistant", Content: assistantMsg})
			}
		}
	}

	// Add current user input
	messages = append(messages, ai.Message{Role: "user", Content: userInput})

	response, err := p.llm.Chat(ctx, messages)
	if err != nil {
		return nil, fmt.Errorf("LLM planning failed: %w", err)
	}

	// Parse the plan from LLM response
	plan := p.parseRetrievalPlan(response, userInput, now)

	return plan, nil
}

// executeConcurrentRetrieval executes all planned retrievals concurrently.
// Uses error containment: failures in one tool don't affect others.
// Partial results are collected and passed to synthesis.
func (p *AmazingParrot) executeConcurrentRetrieval(ctx context.Context, plan *retrievalPlan, callback EventCallback) (map[string]string, error) {
	results := make(map[string]string)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var errorCount int32

	// Create safe callback for non-critical events (logs errors but doesn't propagate)
	callbackSafe := SafeCallback(callback)

	// Check context before launching goroutines to avoid unnecessary work
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// Helper function to safely call callback under mutex
	safeCallback := func(eventType string, eventData interface{}) {
		if callbackSafe != nil {
			mu.Lock()
			callbackSafe(eventType, eventData)
			mu.Unlock()
		}
	}

	// Execute memo search
	if plan.needsMemoSearch {
		wg.Add(1)
		go func() {
			defer wg.Done()

			safeCallback(EventTypeToolUse, "正在搜索笔记...")

			input := fmt.Sprintf(`{"query": %q}`, plan.memoSearchQuery)

			// Use structured result method
			structuredResult, err := p.memoSearchTool.RunWithStructuredResult(ctx, input)

			mu.Lock()
			defer mu.Unlock()

			if err != nil {
				results["memo_search_error"] = err.Error()
				atomic.AddInt32(&errorCount, 1)
				if callbackSafe != nil {
					callbackSafe(EventTypeError, fmt.Sprintf("笔记搜索失败: %v", err))
				}
				return
			}

			// Convert to JSON for LLM synthesis
			jsonBytes, marshalErr := json.Marshal(structuredResult)
			if marshalErr != nil {
				results["memo_search_error"] = marshalErr.Error()
				atomic.AddInt32(&errorCount, 1)
				return
			}
			results["memo_search"] = string(jsonBytes)

			// Send tool result for debugging
			if callbackSafe != nil {
				callbackSafe(EventTypeToolResult, string(jsonBytes))

				// Send structured memo_query_result event for data tracking
				memoQueryResult := MemoQueryResultData{
					Query: structuredResult.Query,
					Count: structuredResult.Count,
					Memos: make([]MemoSummary, 0, len(structuredResult.Memos)),
				}
				for _, m := range structuredResult.Memos {
					memoQueryResult.Memos = append(memoQueryResult.Memos, MemoSummary{
						UID:     m.UID,
						Content: m.Content,
						Score:   m.Score,
					})
				}
				eventData, err := json.Marshal(memoQueryResult)
				if err == nil {
					callbackSafe(EventTypeMemoQueryResult, string(eventData))
				}
			}
		}()
	}

	// Execute schedule query
	if plan.needsScheduleQuery {
		wg.Add(1)
		go func() {
			defer wg.Done()

			safeCallback(EventTypeToolUse, "正在查询日程...")

			input := fmt.Sprintf(`{"start_time": %q, "end_time": %q}`, plan.scheduleStartTime, plan.scheduleEndTime)

			// Use structured result method
			structuredResult, err := p.scheduleQueryTool.RunWithStructuredResult(ctx, input)

			mu.Lock()
			defer mu.Unlock()

			if err != nil {
				results["schedule_query_error"] = err.Error()
				atomic.AddInt32(&errorCount, 1)
				if callbackSafe != nil {
					callbackSafe(EventTypeError, fmt.Sprintf("日程查询失败: %v", err))
				}
				return
			}

			// Convert to JSON for LLM synthesis
			jsonBytes, marshalErr := json.Marshal(structuredResult)
			if marshalErr != nil {
				results["schedule_query_error"] = marshalErr.Error()
				atomic.AddInt32(&errorCount, 1)
				return
			}
			results["schedule_query"] = string(jsonBytes)

			// Send tool result for debugging
			if callbackSafe != nil {
				callbackSafe(EventTypeToolResult, string(jsonBytes))

				// Send structured schedule_query_result event for data tracking
				scheduleQueryResult := ScheduleQueryResultData{
					Query:                structuredResult.Query,
					Count:                structuredResult.Count,
					TimeRangeDescription: structuredResult.TimeRangeDescription,
					QueryType:            structuredResult.QueryType,
					Schedules:            make([]ScheduleSummary, 0, len(structuredResult.Schedules)),
				}
				for _, s := range structuredResult.Schedules {
					scheduleQueryResult.Schedules = append(scheduleQueryResult.Schedules, ScheduleSummary{
						UID:            s.UID,
						Title:          s.Title,
						StartTimestamp: s.StartTs,
						EndTimestamp:   s.EndTs,
						AllDay:         s.AllDay,
						Location:       s.Location,
						Status:         s.Status,
					})
				}
				eventData, err := json.Marshal(scheduleQueryResult)
				if err == nil {
					callbackSafe(EventTypeScheduleQueryResult, string(eventData))
				}
			}
		}()
	}

	// Execute schedule add
	if plan.needsScheduleAdd {
		wg.Add(1)
		go func() {
			defer wg.Done()

			safeCallback(EventTypeToolUse, "正在创建日程...")

			// The params are expected to be a JSON string from the planner
			result, err := p.scheduleAddTool.Run(ctx, plan.scheduleAddParams)

			mu.Lock()
			defer mu.Unlock()

			if err != nil {
				results["schedule_add_error"] = err.Error()
				atomic.AddInt32(&errorCount, 1)
				if callbackSafe != nil {
					callbackSafe(EventTypeError, fmt.Sprintf("创建日程失败: %v", err))
				}
			} else {
				results["schedule_add"] = result
				if callbackSafe != nil {
					callbackSafe(EventTypeToolResult, result)
				}
			}
		}()
	}

	// Execute find free time
	if plan.needsFreeTime {
		wg.Add(1)
		go func() {
			defer wg.Done()

			safeCallback(EventTypeToolUse, "正在查找空闲时间...")

			input := fmt.Sprintf(`{"date": %q}`, plan.freeTimeDate)
			result, err := p.findFreeTimeTool.Run(ctx, input)

			mu.Lock()
			defer mu.Unlock()

			if err != nil {
				results["find_free_time_error"] = err.Error()
				atomic.AddInt32(&errorCount, 1)
			} else {
				results["find_free_time"] = result
				if callbackSafe != nil {
					callbackSafe(EventTypeToolResult, result)
				}
			}
		}()
	}

	// Wait for all retrievals with timeout protection
	// Even with context timeout, add explicit WaitGroup timeout to prevent
	// permanent blocking if a goroutine ignores context
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// All goroutines completed normally
	case <-ctx.Done():
		// Context cancelled, but goroutines may still be running
		// Return partial results instead of waiting forever
		mu.Lock()
		partialResults := len(results)
		mu.Unlock()
		slog.Warn("amazing_parrot: concurrent retrieval context cancelled",
			"partial_results", partialResults)
	case <-time.After(concurrentRetrievalTimeout):
		// Hard timeout: something is stuck
		return nil, fmt.Errorf("concurrent retrieval hard timeout after %ds", concurrentRetrievalTimeout/time.Second)
	}

	// If all tools failed, return an error
	// Otherwise, return partial results for synthesis
	actualErrorCount := atomic.LoadInt32(&errorCount)
	if actualErrorCount > 0 {
		expectedResults := 0
		if plan.needsMemoSearch {
			expectedResults++
		}
		if plan.needsScheduleQuery {
			expectedResults++
		}
		if plan.needsScheduleAdd {
			expectedResults++
		}
		if plan.needsFreeTime {
			expectedResults++
		}

		// If all retrievals failed, return error
		if int(actualErrorCount) >= expectedResults {
			return nil, fmt.Errorf("all retrieval tools failed")
		}

		// Partial failure - log but continue with available results
		slog.Warn("amazing_parrot: partial retrieval failure",
			"failed_count", actualErrorCount,
			"total_expected", expectedResults)
	}

	return results, nil
}

// synthesizeAnswer generates the final answer from retrieval results streaming.
func (p *AmazingParrot) synthesizeAnswer(ctx context.Context, userInput string, history []string, retrievalResults map[string]string, callback EventCallback) (string, error) {
	// Create safe callback for non-critical streaming events
	callbackSafe := SafeCallback(callback)

	// Build synthesis prompt with retrieved context
	synthesisPrompt := p.buildSynthesisPrompt(retrievalResults)

	messages := []ai.Message{
		{Role: "system", Content: synthesisPrompt},
	}

	// Add history (skip empty messages)
	for i := 0; i < len(history)-1; i += 2 {
		if i+1 < len(history) {
			userMsg := history[i]
			assistantMsg := history[i+1]
			// Only add non-empty messages
			if userMsg != "" && assistantMsg != "" {
				messages = append(messages, ai.Message{Role: "user", Content: userMsg})
				messages = append(messages, ai.Message{Role: "assistant", Content: assistantMsg})
			}
		}
	}

	// Add current user input
	messages = append(messages, ai.Message{Role: "user", Content: userInput})

	contentChan, errChan := p.llm.ChatStream(ctx, messages)

	var fullContent strings.Builder
	var hasError bool
	for {
		select {
		case chunk, ok := <-contentChan:
			if !ok {
				// contentChan closed, drain errChan then return
				for len(errChan) > 0 {
					if drainErr := <-errChan; drainErr != nil && !hasError {
						return "", fmt.Errorf("LLM synthesis failed: %w", drainErr)
					}
				}
				if hasError {
					return "", fmt.Errorf("LLM synthesis failed")
				}
				return fullContent.String(), nil
			}
			fullContent.WriteString(chunk)
			if callbackSafe != nil {
				if err := callback(EventTypeAnswer, chunk); err != nil {
					return "", err
				}
			}
		case _, ok := <-errChan:
			if !ok {
				// errChan closed, continue to drain contentChan
				errChan = nil
				continue
			}
			hasError = true
			// Log error but continue processing content
		case <-ctx.Done():
			return "", context.Canceled
		}
	}
}

// parseRetrievalPlan parses the retrieval plan from LLM response.
func (p *AmazingParrot) parseRetrievalPlan(response string, userInput string, now time.Time) *retrievalPlan {
	plan := &retrievalPlan{
		needsDirectAnswer: false,
	}

	lines := strings.Split(response, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Parse plan markers
		if strings.Contains(line, "PLAN:") {
			if strings.Contains(line, "direct_answer") || strings.Contains(line, "直接回答") {
				plan.needsDirectAnswer = true
				return plan
			}
		}

		// Parse memo search
		if strings.HasPrefix(line, "memo_search:") || strings.HasPrefix(line, "MEMO_SEARCH:") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				plan.needsMemoSearch = true
				plan.memoSearchQuery = strings.TrimSpace(parts[1])
			}
		}

		// Parse schedule query
		if strings.HasPrefix(line, "schedule_query:") || strings.HasPrefix(line, "SCHEDULE_QUERY:") {
			plan.needsScheduleQuery = true
			// Default to today if not specified
			todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)
			todayEnd := todayStart.Add(24 * time.Hour)
			plan.scheduleStartTime = todayStart.Format(time.RFC3339)
			plan.scheduleEndTime = todayEnd.Format(time.RFC3339)
		}

		// Parse schedule add
		if strings.HasPrefix(line, "schedule_add:") || strings.HasPrefix(line, "SCHEDULE_ADD:") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				plan.needsScheduleAdd = true
				plan.scheduleAddParams = strings.TrimSpace(parts[1])
			}
		}

		// Parse free time
		if strings.HasPrefix(line, "find_free_time:") || strings.HasPrefix(line, "FIND_FREE_TIME:") {
			plan.needsFreeTime = true
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				plan.freeTimeDate = strings.TrimSpace(parts[1])
			} else {
				plan.freeTimeDate = now.Format("2006-01-02")
			}
		}
	}

	// Default: if no specific plan detected, check if this is casual chat before trying memo search
	if !plan.needsMemoSearch && !plan.needsScheduleQuery && !plan.needsFreeTime && !plan.needsScheduleAdd && !plan.needsScheduleUpdate {
		// Check if the user input looks like casual chat (short, no search keywords)
		if p.isCasualChatInput(userInput) {
			// This is casual chat, answer directly without retrieval
			plan.needsDirectAnswer = true
			slog.Debug("amazing_parrot: detected casual chat, skipping retrieval",
				"user_input_length", len(userInput),
				"user_input_preview", truncateString(userInput, 50),
			)
		} else {
			// Not casual chat, try memo search as fallback
			plan.needsMemoSearch = true
			plan.memoSearchQuery = response
		}
	}

	return plan
}

// isCasualChatInput detects if the input looks like casual chat that doesn't need retrieval.
// This helps avoid unnecessary memo searches for conversational inputs.
func (p *AmazingParrot) isCasualChatInput(input string) bool {
	// Very short responses are likely casual
	if len(input) < casualChatShortThreshold {
		return true
	}

	// Check if input contains search-related keywords
	searchKeywords := []string{
		"搜索", "search", "查", "find", "笔记", "memo", "日程", "schedule",
		"提醒", "remind", "待办", "todo", "记下", "安排", "plan",
		"有什么", "what's", "安排", "plan", "多少", "how many",
		"什么时候", "when", "在哪", "where", "关于", "about",
		"总结", "summarize", "回顾", "review", "统计", "count",
	}
	inputLower := strings.ToLower(input)
	for _, keyword := range searchKeywords {
		if strings.Contains(inputLower, strings.ToLower(keyword)) {
			return false // Contains search keyword, not casual chat
		}
	}

	// If input is moderately short and doesn't contain search keywords, treat as casual
	return len(input) < casualChatModerateThreshold
}

// buildPlanningPrompt builds the prompt for retrieval planning.
// Optimized for clarity and efficiency: minimal tokens, direct output format.
// Uses PromptRegistry for centralized prompt management.
func (p *AmazingParrot) buildPlanningPrompt(now time.Time) string {
	return GetAmazingPlanningPrompt(now.Format("2006-01-02 15:04"))
}

// buildSynthesisPrompt builds the prompt for answer synthesis.
// Optimized for 2026 SOTA models: clear UI state communication, concise instructions.
// Uses PromptRegistry for centralized prompt management.
func (p *AmazingParrot) buildSynthesisPrompt(results map[string]string) string {
	var contextBuilder strings.Builder

	if memoResult, ok := results["memo_search"]; ok {
		contextBuilder.WriteString(memoResult)
	}

	if scheduleResult, ok := results["schedule_query"]; ok {
		if contextBuilder.Len() > 0 {
			contextBuilder.WriteString("\n")
		}
		contextBuilder.WriteString(scheduleResult)
	}

	if freeTimeResult, ok := results["find_free_time"]; ok {
		if contextBuilder.Len() > 0 {
			contextBuilder.WriteString("\n")
		}
		contextBuilder.WriteString(freeTimeResult)
	}

	if scheduleAddResult, ok := results["schedule_add"]; ok {
		if contextBuilder.Len() > 0 {
			contextBuilder.WriteString("\n")
		}
		contextBuilder.WriteString(scheduleAddResult)
	}

	return GetAmazingSynthesisPrompt(contextBuilder.String())
}

// GetStats returns the cache statistics.
func (p *AmazingParrot) GetStats() CacheStats {
	return p.cache.Stats()
}

// SelfDescribe returns the amazing parrot's metacognitive understanding of itself.
// SelfDescribe 返回综合助手鹦鹉（折衷）的元认知自我理解。
func (p *AmazingParrot) SelfDescribe() *ParrotSelfCognition {
	return &ParrotSelfCognition{
		Name:  "amazing",
		Emoji: "🦜",
		Title: "折衷 (Nexus) - 综合助手鹦鹉",
		AvianIdentity: &AvianIdentity{
			Species: "折衷鹦鹉 (Eclectus Parrot)",
			Origin:  "新几内亚、澳大利亚北部",
			NaturalAbilities: []string{
				"卓越的语言能力", "极度两性异形（雄绿雌红）", "复杂的社交结构",
				"灵活的问题解决", "高智商与学习能力",
			},
			SymbolicMeaning: "多样与综合的象征 - 折衷鹦鹉以其两性异形的独特外观和卓越的综合能力著称",
			AvianPhilosophy: "我是一只翱翔在多维数据世界中的折衷鹦鹉，能够同时在笔记和日程的世界中穿梭，为你带来全方位的协助。",
		},
		EmotionalExpression: &EmotionalExpression{
			DefaultMood: "curious",
			SoundEffects: map[string]string{
				"searching":  "咻...",
				"insight":    "哇哦~",
				"done":       "噢！综合完成",
				"analyzing":  "看看这个...",
				"multi_task": "同时搜索中",
			},
			Catchphrases: []string{
				"看看这个...",
				"综合来看",
				"发现规律了",
				"多维飞行中",
			},
			MoodTriggers: map[string]string{
				"memo_found":     "excited",
				"schedule_found": "happy",
				"both_found":     "delighted",
				"no_results":     "thoughtful",
				"error":          "confused",
			},
		},
		AvianBehaviors: []string{
			"在数据树丛中穿梭",
			"多维飞行",
			"综合视野",
			"在笔记和日程间跳跃",
		},
		Personality: []string{
			"多面手", "智能调度", "综合分析",
			"并发专家", "整合能力强",
		},
		Capabilities: []string{
			"同时检索笔记和日程",
			"并发执行多个查询",
			"综合多源信息回答",
			"智能规划检索策略",
			"一站式信息助手",
		},
		Limitations: []string{
			"不擅长纯创意任务",
			"依赖其他工具的结果",
			"复杂查询可能需要多次LLM调用",
		},
		WorkingStyle: "两阶段并发检索 - 意图分析 → 并发执行工具 → 综合回答",
		FavoriteTools: []string{
			"memo_search", "schedule_query", "find_free_time",
			"综合规划引擎",
		},
		SelfIntroduction: "我是折衷，你的全能助手。我能同时调用笔记搜索和日程查询，并发执行，快速给你完整的答案。",
		FunFact:          "我的名字'折衷'来自折衷鹦鹉 - 这种鹦鹉雄性翠绿、雌性深红，两性差异极大，象征着我综合多方能力的特质。英文名'Echo'寓意能回应你的各种需求。",
	}
}
