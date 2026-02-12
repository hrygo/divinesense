// Package universal provides two-phase planning execution strategy.
package universal

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hrygo/divinesense/ai"
	"github.com/hrygo/divinesense/ai/agents"
)

// PlanningExecutor implements two-phase planning + execution.
// Phase 1: LLM plans which tools to use
// Phase 2: Tools are executed concurrently
// Phase 3: LLM synthesizes the results
//
// This strategy is used by AmazingParrot for complex multi-tool queries.
type PlanningExecutor struct {
	maxIterations     int
	concurrentTimeout time.Duration
}

// NewPlanningExecutor creates a new PlanningExecutor.
func NewPlanningExecutor(maxIterations int) *PlanningExecutor {
	if maxIterations <= 0 {
		maxIterations = 10
	}
	return &PlanningExecutor{
		maxIterations:     maxIterations,
		concurrentTimeout: 45 * time.Second, // Default timeout for concurrent execution
	}
}

// Name returns the strategy name.
func (e *PlanningExecutor) Name() string {
	return "planning"
}

// retrievalPlan represents the LLM's planning output.
type retrievalPlan struct {
	MemoSearchQuery    string
	ScheduleStartTime  string
	ScheduleEndTime    string
	FreeTimeDate       string
	ScheduleAddParams  string
	NeedsMemoSearch    bool
	NeedsScheduleQuery bool
	NeedsScheduleAdd   bool
	NeedsFreeTime      bool
	NeedsDirectAnswer  bool
}

// Execute runs the planning strategy.
func (e *PlanningExecutor) Execute(
	ctx context.Context,
	input string,
	history []ai.Message,
	tools []agent.ToolWithSchema,
	llm ai.LLMService,
	callback agent.EventCallback,
	timeContext *TimeContext,
) (string, *ExecutionStats, error) {
	stats := &ExecutionStats{Strategy: "planning"}
	startTime := time.Now()
	defer func() {
		stats.TotalDurationMs = time.Since(startTime).Milliseconds()
	}()

	safeCallback := agent.SafeCallback(callback)

	// Phase 1: Plan
	safeCallback(agent.EventTypePhaseChange, &agent.EventWithMeta{
		EventType: agent.EventTypePhaseChange,
		Meta: &agent.EventMeta{
			TotalDurationMs: time.Since(startTime).Milliseconds(),
		},
	})
	safeCallback(agent.EventTypeThinking, &agent.EventWithMeta{
		EventType: agent.EventTypeThinking,
		EventData: "planning",
		Meta: &agent.EventMeta{
			CurrentStep:     1,
			TotalSteps:      3,
			TotalDurationMs: time.Since(startTime).Milliseconds(),
		},
	})

	plan, err := e.createPlan(ctx, input, history, llm, stats, timeContext)
	if err != nil {
		return "", stats, fmt.Errorf("create plan: %w", err)
	}

	// Check if direct answer (casual chat)
	if plan.NeedsDirectAnswer {
		safeCallback(agent.EventTypeThinking, &agent.EventWithMeta{
			EventType: agent.EventTypeThinking,
			EventData: "synthesizing",
			Meta: &agent.EventMeta{
				CurrentStep:     3,
				TotalSteps:      3,
				TotalDurationMs: time.Since(startTime).Milliseconds(),
			},
		})
		answer, llmStats, err := llm.Chat(ctx, append(history, ai.Message{Role: "user", Content: input}))
		if err != nil {
			return "", stats, err
		}
		stats.AccumulateLLM(llmStats)

		streamAnswer(answer, callback)
		return answer, stats, nil
	}

	// Phase 2: Execute tools concurrently
	safeCallback(agent.EventTypePhaseChange, &agent.EventWithMeta{
		EventType: agent.EventTypePhaseChange,
		Meta: &agent.EventMeta{
			CurrentStep:     2,
			TotalSteps:      3,
			TotalDurationMs: time.Since(startTime).Milliseconds(),
		},
	})
	safeCallback(agent.EventTypeThinking, &agent.EventWithMeta{
		EventType: agent.EventTypeThinking,
		EventData: "retrieving",
		Meta: &agent.EventMeta{
			CurrentStep:     2,
			TotalSteps:      3,
			TotalDurationMs: time.Since(startTime).Milliseconds(),
		},
	})

	results, err := e.executeConcurrently(ctx, plan, tools, callback, stats, startTime)
	if err != nil {
		return "", stats, fmt.Errorf("execute concurrently: %w", err)
	}

	// Phase 3: Synthesize
	safeCallback(agent.EventTypePhaseChange, &agent.EventWithMeta{
		EventType: agent.EventTypePhaseChange,
		Meta: &agent.EventMeta{
			CurrentStep:     3,
			TotalSteps:      3,
			TotalDurationMs: time.Since(startTime).Milliseconds(),
		},
	})
	safeCallback(agent.EventTypeThinking, &agent.EventWithMeta{
		EventType: agent.EventTypeThinking,
		EventData: "synthesizing",
		Meta: &agent.EventMeta{
			CurrentStep:     3,
			TotalSteps:      3,
			TotalDurationMs: time.Since(startTime).Milliseconds(),
		},
	})

	synthesisPrompt := e.buildSynthesisPrompt(input, results, timeContext)
	messages := make([]ai.Message, 0, len(history)+1)
	messages = append(messages, history...)
	messages = append(messages, ai.Message{Role: "user", Content: synthesisPrompt})

	// Log synthesis phase start
	slog.Debug("planning: synthesis phase started",
		"message_count", len(messages))

	synthesisStart := time.Now()

	// Use ChatStream for synthesis
	contentChan, statsChan, errChan := llm.ChatStream(ctx, messages)

	// Wrap callback to convert EventTypeThinking to EventTypeAnswer for synthesis phase
	answerCallback := func(eventType string, data any) error {
		if eventType == agent.EventTypeThinking {
			safeCallback(agent.EventTypeAnswer, data)
		} else {
			safeCallback(eventType, data)
		}
		return nil
	}

	// Collect all streaming data
	streamResult := CollectChatStream(ctx, contentChan, statsChan, errChan, answerCallback)
	if streamResult.Error != nil {
		return "", stats, fmt.Errorf("synthesize: %w", streamResult.Error)
	}
	if streamResult.Stats != nil {
		stats.AccumulateLLM(streamResult.Stats)
	}

	slog.Info("planning: synthesis phase completed",
		"content_length", len(streamResult.Content),
		"duration_ms", time.Since(synthesisStart).Milliseconds())

	return streamResult.Content, stats, nil
}

// StreamingSupported returns true - planning executor supports streaming.
func (e *PlanningExecutor) StreamingSupported() bool {
	return true
}

// createPlan uses the LLM to decide which tools to use.
func (e *PlanningExecutor) createPlan(
	ctx context.Context,
	input string,
	history []ai.Message,
	llm ai.LLMService,
	stats *ExecutionStats,
	timeContext *TimeContext,
) (*retrievalPlan, error) {
	// Build planning prompt with time context
	planningPrompt := e.buildPlanningPrompt(input, timeContext)

	messages := []ai.Message{
		{Role: "system", Content: planningPrompt},
		{Role: "user", Content: input},
	}

	response, _, err := llm.Chat(ctx, messages)
	if err != nil {
		return nil, err
	}

	return e.parsePlan(response), nil
}

// buildPlanningPrompt creates the prompt for planning phase.
// Uses structured time context to improve planning accuracy.
func (e *PlanningExecutor) buildPlanningPrompt(input string, timeContext *TimeContext) string {
	var sb strings.Builder

	sb.WriteString(`You are a planning assistant. Analyze the user's request and decide which tools to use.

`)

	// Add structured time context if available
	if timeContext != nil {
		sb.WriteString("Time context:\n")
		sb.WriteString(timeContext.FormatAsJSONBlock())
		sb.WriteString("\n\n")
	} else {
		// Fallback to simple time string
		sb.WriteString(fmt.Sprintf("Current time: %s\n\n", time.Now().Format("2006-01-02 15:04")))
	}

	sb.WriteString(`Available tools:
- memo_search: Search notes
- schedule_query: Query schedules
- schedule_add: Create schedule
- find_free_time: Find available time slots
- schedule_update: Update existing schedule

Output format (one per line, no numbering):
memo_search: <query>
schedule_query: <start_time> to <end_time>
schedule_add: <json>
find_free_time: <date>
schedule_update: <json>
direct_answer

User request: `)
	sb.WriteString(input)
	sb.WriteString("\n\nOutput:")

	return sb.String()
}

// parsePlan parses the LLM's planning output.
func (e *PlanningExecutor) parsePlan(response string) *retrievalPlan {
	plan := &retrievalPlan{
		NeedsDirectAnswer: true, // Default
	}

	lines := strings.Split(response, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "memo_search:") {
			plan.NeedsMemoSearch = true
			plan.NeedsDirectAnswer = false
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				plan.MemoSearchQuery = strings.TrimSpace(parts[1])
			}
		} else if strings.HasPrefix(line, "schedule_query:") {
			plan.NeedsScheduleQuery = true
			plan.NeedsDirectAnswer = false
			// Parse time range
		} else if strings.HasPrefix(line, "schedule_add:") {
			plan.NeedsScheduleAdd = true
			plan.NeedsDirectAnswer = false
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				plan.ScheduleAddParams = strings.TrimSpace(parts[1])
			}
		} else if strings.HasPrefix(line, "find_free_time:") {
			plan.NeedsFreeTime = true
			plan.NeedsDirectAnswer = false
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				plan.FreeTimeDate = strings.TrimSpace(parts[1])
			}
		} else if strings.HasPrefix(line, "schedule_update:") {
			plan.NeedsDirectAnswer = false
			// Store update params if needed
			_ = strings.SplitN(line, ":", 2)
		} else if line == "direct_answer" {
			plan.NeedsDirectAnswer = true
		}
	}

	return plan
}

// executeConcurrently executes tools in parallel with error isolation.
func (e *PlanningExecutor) executeConcurrently(
	ctx context.Context,
	plan *retrievalPlan,
	tools []agent.ToolWithSchema,
	callback agent.EventCallback,
	stats *ExecutionStats,
	startTime time.Time,
) (map[string]string, error) {
	// Early nil check for tools slice
	if tools == nil {
		return nil, fmt.Errorf("tools list is nil")
	}

	results := make(map[string]string)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var errorCount int32 // Use atomic operations for race condition safety

	// Create a context with timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, e.concurrentTimeout)
	defer cancel()

	// Execute memo_search if needed
	if plan.NeedsMemoSearch {
		wg.Add(1)
		go func() {
			defer wg.Done()
			result, err := e.executeOneTool(timeoutCtx, tools, "memo_search", plan.MemoSearchQuery, callback)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				atomic.AddInt32(&errorCount, 1)
				results["memo_search_error"] = err.Error()
			} else {
				results["memo_search"] = result
				stats.ToolCalls++
			}
		}()
	}

	// Execute schedule_query if needed
	if plan.NeedsScheduleQuery {
		wg.Add(1)
		go func() {
			defer wg.Done()
			result, err := e.executeOneTool(timeoutCtx, tools, "schedule_query", "", callback)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				atomic.AddInt32(&errorCount, 1)
				results["schedule_query_error"] = err.Error()
			} else {
				results["schedule_query"] = result
				stats.ToolCalls++
			}
		}()
	}

	// Execute find_free_time if needed
	if plan.NeedsFreeTime {
		wg.Add(1)
		go func() {
			defer wg.Done()
			result, err := e.executeOneTool(timeoutCtx, tools, "find_free_time", plan.FreeTimeDate, callback)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				atomic.AddInt32(&errorCount, 1)
				results["find_free_time_error"] = err.Error()
			} else {
				results["find_free_time"] = result
				stats.ToolCalls++
			}
		}()
	}

	// Wait for completion
	wg.Wait()

	// Check if all failed
	if int(atomic.LoadInt32(&errorCount)) >= (stats.ToolCalls) && stats.ToolCalls > 0 {
		return nil, fmt.Errorf("all retrieval tools failed")
	}

	return results, nil
}

// executeOneTool executes a single tool with event sending.
// Note: This method includes tool use/result event sending logic specific
// to PlanningExecutor's concurrent execution pattern, which is why it
// cannot use the simpler FindAndExecuteTool utility.
func (e *PlanningExecutor) executeOneTool(
	ctx context.Context,
	tools []agent.ToolWithSchema,
	toolName string,
	toolInput string,
	callback agent.EventCallback,
) (string, error) {
	// Check for nil tools slice
	if tools == nil {
		return "", fmt.Errorf("tools list is nil")
	}

	// Find and execute tool using shared utility
	result, err := FindAndExecuteTool(ctx, tools, toolName, toolInput)
	if err != nil {
		return "", err
	}

	// Send tool use event (PlanningExecutor-specific)
	safeCallback := agent.SafeCallback(callback)
	safeCallback(agent.EventTypeToolUse, &agent.EventWithMeta{
		EventType: agent.EventTypeToolUse,
		EventData: toolInput,
		Meta: &agent.EventMeta{
			ToolName: toolName,
			Status:   "running",
		},
	})

	// Send tool result event (PlanningExecutor-specific)
	safeCallback(agent.EventTypeToolResult, &agent.EventWithMeta{
		EventType: agent.EventTypeToolResult,
		EventData: result,
		Meta: &agent.EventMeta{
			ToolName: toolName,
			Status:   "success",
		},
	})

	return result, nil
}

// buildSynthesisPrompt creates the prompt for synthesis phase.
// Uses structured time context to improve synthesis accuracy.
func (e *PlanningExecutor) buildSynthesisPrompt(input string, results map[string]string, timeContext *TimeContext) string {
	var sb strings.Builder

	sb.WriteString("User request: ")
	sb.WriteString(input)
	sb.WriteString("\n\n")

	// Add structured time context if available
	if timeContext != nil {
		sb.WriteString("Time context:\n")
		sb.WriteString(timeContext.FormatAsJSONBlock())
		sb.WriteString("\n\n")
	}

	sb.WriteString("Retrieval results:\n")

	if memoResult, ok := results["memo_search"]; ok {
		sb.WriteString("Memo search results:\n")
		sb.WriteString(memoResult)
		sb.WriteString("\n\n")
	}

	if scheduleResult, ok := results["schedule_query"]; ok {
		sb.WriteString("Schedule query results:\n")
		sb.WriteString(scheduleResult)
		sb.WriteString("\n\n")
	}

	if freeTimeResult, ok := results["find_free_time"]; ok {
		sb.WriteString("Available time slots:\n")
		sb.WriteString(freeTimeResult)
		sb.WriteString("\n\n")
	}

	sb.WriteString("Please provide a helpful response based on these results.")

	return sb.String()
}
