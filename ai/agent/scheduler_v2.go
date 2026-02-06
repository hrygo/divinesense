package agent

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/hrygo/divinesense/ai"
	localtools "github.com/hrygo/divinesense/ai/agent/tools"
	"github.com/hrygo/divinesense/server/service/schedule"
)

// SchedulerAgentV2 is the new framework-less schedule agent.
// It uses native LLM tool calling without LangChainGo dependency.
type SchedulerAgentV2 struct {
	agent            *Agent
	llm              ai.LLMService
	scheduleSvc      schedule.Service
	userID           int32
	timezone         string
	timezoneLoc      *time.Location
	intentClassifier *LLMIntentClassifier // LLM-based intent classification
	queryTool        interface{}          // Stored for structured result access
	*BaseParrot                           // Embedded for stats accumulation (P1-A006)
}

// NewSchedulerAgentV2 creates a new framework-less schedule agent.
func NewSchedulerAgentV2(llm ai.LLMService, scheduleSvc schedule.Service, userID int32, userTimezone string) (*SchedulerAgentV2, error) {
	if llm == nil {
		return nil, fmt.Errorf("LLM service is required")
	}
	if scheduleSvc == nil {
		return nil, fmt.Errorf("schedule service is required")
	}
	if userID <= 0 {
		return nil, fmt.Errorf("invalid user ID: %d", userID)
	}

	if userTimezone == "" {
		userTimezone = "Asia/Shanghai"
	}

	timezoneLoc, err := time.LoadLocation(userTimezone)
	if err != nil {
		slog.Warn("invalid timezone, using UTC",
			"timezone", userTimezone,
			"user_id", userID,
			"error", err)
		userTimezone = "UTC"
		timezoneLoc = time.UTC
	}

	// Create user ID getter
	userIDGetter := func(ctx context.Context) int32 {
		return userID
	}

	// Create actual tool instances
	queryTool := localtools.NewScheduleQueryTool(scheduleSvc, userIDGetter)
	addTool := localtools.NewScheduleAddTool(scheduleSvc, userIDGetter)
	updateTool := localtools.NewScheduleUpdateTool(scheduleSvc, userIDGetter)
	findFreeTimeTool := localtools.NewFindFreeTimeTool(scheduleSvc, userIDGetter)
	findFreeTimeTool.SetTimezone(userTimezone)

	// Convert to ToolWithSchema using adapter
	tools := []ToolWithSchema{
		wrapToolWithName("schedule_query", queryTool),
		wrapToolWithName("schedule_add", addTool),
		wrapToolWithName("find_free_time", findFreeTimeTool),
		wrapToolWithName("schedule_update", updateTool),
	}

	// Build system prompt
	systemPrompt := buildSystemPromptV2(timezoneLoc)

	// Create the agent
	agent := NewAgent(llm, AgentConfig{
		Name:          "schedule",
		SystemPrompt:  systemPrompt,
		MaxIterations: 10,
	}, tools)

	return &SchedulerAgentV2{
		agent:       agent,
		llm:         llm,
		scheduleSvc: scheduleSvc,
		userID:      userID,
		timezone:    userTimezone,
		timezoneLoc: timezoneLoc,
		queryTool:   queryTool,
		BaseParrot:  NewBaseParrot("schedule"),
	}, nil
}

// SetIntentClassifier configures the LLM-based intent classifier.
// When set, the agent will classify user input before execution to optimize
// routing and provide better responses.
func (a *SchedulerAgentV2) SetIntentClassifier(classifier *LLMIntentClassifier) {
	a.intentClassifier = classifier
}

// recordMetrics records prompt usage metrics for the schedule agent.
func (a *SchedulerAgentV2) recordMetrics(startTime time.Time, promptVersion PromptVersion, success bool) {
	latencyMs := time.Since(startTime).Milliseconds()
	RecordPromptUsageInMemory("schedule", promptVersion, success, latencyMs)
}

// wrapTool converts a tool with Run() and Description() methods to ToolWithSchema.
// It handles tools that also have InputType() for JSON Schema.
func wrapTool(tool interface{}) ToolWithSchema {
	// Try to get Run method
	var runFunc func(ctx context.Context, input string) (string, error)
	var description string
	var params map[string]interface{}

	switch t := tool.(type) {
	case interface {
		Run(ctx context.Context, input string) (string, error)
	}:
		runFunc = t.Run
	case interface {
		Call(ctx context.Context, input string) (string, error)
	}:
		runFunc = t.Call
	}

	// Get description
	if d, ok := tool.(interface{ Description() string }); ok {
		description = d.Description()
	}

	// Get input type/schema
	if i, ok := tool.(interface{ InputType() map[string]interface{} }); ok {
		params = i.InputType()
	}
	if i, ok := tool.(interface{ Parameters() map[string]interface{} }); ok {
		params = i.Parameters()
	}

	// Fallback params
	if params == nil {
		params = map[string]interface{}{
			"type":       "object",
			"properties": map[string]interface{}{},
		}
	}

	// Create tool with zero name; will be set by wrapToolWithName
	return &NativeTool{
		name:        "", // Set by wrapToolWithName
		description: description,
		execute:     runFunc,
		params:      params,
	}
}

// wrapToolWithName is a helper that also sets the tool name.
func wrapToolWithName(name string, tool interface{}) ToolWithSchema {
	wrapped := wrapTool(tool)
	// Set the tool name to the provided value
	if nt, ok := wrapped.(*NativeTool); ok {
		nt.name = name
	}
	return wrapped
}

// Execute runs the agent with the given user input.
func (a *SchedulerAgentV2) Execute(ctx context.Context, userInput string) (string, error) {
	return a.ExecuteWithCallback(ctx, userInput, nil, nil)
}

// ExecuteWithCallback runs the agent with state-aware context and callback support.
func (a *SchedulerAgentV2) ExecuteWithCallback(ctx context.Context, userInput string, conversationCtx *ConversationContext, callback func(event string, data string)) (string, error) {
	startTime := time.Now()

	// Get prompt version for AB testing
	promptVersion := GetPromptVersionForUser("schedule", a.userID)

	// Intent classification (if classifier is configured)
	//nolint:staticcheck // explicit type for clarity
	var intent TaskIntent
	intent = IntentSimpleCreate // default
	if a.intentClassifier != nil {
		classifiedIntent, err := a.intentClassifier.Classify(ctx, userInput)
		if err != nil {
			slog.Warn("intent classification failed, using default",
				"error", err,
				"input", truncateForLog(userInput, 30))
		} else {
			intent = classifiedIntent
			slog.Debug("intent classified",
				"intent", intent,
				"input", truncateForLog(userInput, 30),
				"prompt_version", promptVersion)

			// Notify frontend about classified intent
			if callback != nil {
				callback("intent_classified", string(intent))
			}
		}
	}

	// If there's conversation context, prepend it to the input
	fullInput := userInput
	if conversationCtx != nil {
		historyPrompt := conversationCtx.ToHistoryPrompt()
		if historyPrompt != "" {
			fullInput = historyPrompt + "\nCurrent Request: " + userInput
			slog.Debug("Conversation context applied",
				"user_id", a.userID,
				"history_len", len(historyPrompt),
				"full_input_len", len(fullInput))
		} else {
			slog.Warn("Conversation context exists but ToHistoryPrompt returned empty",
				"user_id", a.userID,
				"session_id", conversationCtx.SessionID)
		}
	}

	// Add intent hint to help the agent
	if intent != IntentSimpleCreate {
		fullInput = fmt.Sprintf("[意图: %s]\n%s", a.intentToHint(intent), fullInput)
	}

	// Run the agent
	// TODO: For IntentBatchCreate, use Plan-Execute mode instead of ReAct
	result, err := a.agent.RunWithCallback(ctx, fullInput, callback)

	// Record metrics
	a.recordMetrics(startTime, promptVersion, err == nil)

	return result, err
}

// intentToHint converts intent to a hint string for the LLM.
func (a *SchedulerAgentV2) intentToHint(intent TaskIntent) string {
	switch intent {
	case IntentSimpleCreate:
		return "创建单个日程"
	case IntentSimpleQuery:
		return "查询日程或空闲时间"
	case IntentSimpleUpdate:
		return "修改或删除日程"
	case IntentBatchCreate:
		return "批量创建重复日程"
	case IntentConflictResolve:
		return "处理日程冲突"
	case IntentMultiQuery:
		return "综合查询"
	default:
		return "通用日程操作"
	}
}

// buildSystemPromptV2 builds the system prompt for the schedule agent.
// Uses PromptRegistry for centralized prompt management.
func buildSystemPromptV2(timezoneLoc *time.Location) string {
	nowLocal := time.Now().In(timezoneLoc)
	_, tzOffset := nowLocal.Zone()
	tzOffsetStr := FormatTZOffset(tzOffset)
	return GetScheduleSystemPrompt(
		nowLocal.Format("2006-01-02 15:04"),
		timezoneLoc.String(),
		tzOffsetStr,
	)
}

// GetSessionStats returns the accumulated session statistics.
// GetSessionStats 返回累积的会话统计信息。
func (a *SchedulerAgentV2) GetSessionStats() *NormalSessionStats {
	if a.BaseParrot == nil {
		return nil
	}
	return a.BaseParrot.GetSessionStats()
}
