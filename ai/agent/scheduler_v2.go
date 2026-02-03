package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
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

	// Wrap the callback to inject UI events
	uiCallback := a.wrapUICallback(callback)

	// Run the agent
	// TODO: For IntentBatchCreate, use Plan-Execute mode instead of ReAct
	result, err := a.agent.RunWithCallback(ctx, fullInput, uiCallback)

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

// wrapUICallback wraps the original callback to inject UI events based on tool usage.
// This enables generative UI by emitting structured UI events when tools are called.
// Creates a detached context with timeout for async callback operations to prevent
// issues when the original request context is cancelled before the callback fires.
func (a *SchedulerAgentV2) wrapUICallback(originalCallback func(event string, data string)) func(event string, data string) {
	var pendingSchedule *UIScheduleSuggestionData

	return func(event string, data string) {
		if originalCallback != nil {
			originalCallback(event, data)
		}

		// Handle schedule_query tool - emit UI schedule list
		// Use a detached context with timeout since this runs in a callback
		// that may execute after the original request context is cancelled.
		if event == "tool_use" && strings.HasPrefix(data, "schedule_query:") {
			// Create a detached context with 30s timeout for the callback operation
			callbackCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			a.handleScheduleQuery(callbackCtx, data, originalCallback)
		}

		// Handle schedule_add tool - emit schedule suggestion card
		if event == "tool_use" && strings.HasPrefix(data, "schedule_add:") {
			if scheduleData := a.parseScheduleAddInput(data); scheduleData != nil {
				pendingSchedule = scheduleData
				a.emitUIEvent(originalCallback, EventTypeUIScheduleSuggestion, scheduleData)
			}
		}

		// Handle schedule_add tool result - check for conflicts
		if event == "tool_result" && pendingSchedule != nil {
			if a.isConflictResult(data) {
				conflictData := a.buildConflictResolutionData(data, pendingSchedule)
				if conflictData != nil {
					a.emitUIEvent(originalCallback, EventTypeUIConflictResolution, conflictData)
				}
			}
			pendingSchedule = nil
		}
	}
}

// parseScheduleAddInput parses the schedule_add tool input to extract schedule data.
func (a *SchedulerAgentV2) parseScheduleAddInput(toolData string) *UIScheduleSuggestionData {
	// Format: "schedule_add:{JSON}"
	if !strings.HasPrefix(toolData, "schedule_add:") {
		return nil
	}

	jsonPart := strings.TrimPrefix(toolData, "schedule_add:")

	var input struct {
		Title       string `json:"title"`
		StartTime   string `json:"start_time"`
		EndTime     string `json:"end_time,omitempty"`
		Description string `json:"description,omitempty"`
		Location    string `json:"location,omitempty"`
		AllDay      bool   `json:"all_day,omitempty"`
	}

	if err := json.Unmarshal([]byte(jsonPart), &input); err != nil {
		slog.Debug("failed to parse schedule_add input", "error", err)
		return nil
	}

	// Parse times
	startTime, err := time.Parse(time.RFC3339, input.StartTime)
	if err != nil {
		slog.Debug("failed to parse start_time", "error", err)
		return nil
	}

	var endTs int64
	if input.EndTime != "" {
		endTime, err := time.Parse(time.RFC3339, input.EndTime)
		if err != nil {
			// Default to 1 hour
			endTs = startTime.Unix() + 3600
		} else {
			endTs = endTime.Unix()
		}
	} else {
		endTs = startTime.Unix() + 3600
	}

	return &UIScheduleSuggestionData{
		Title:       input.Title,
		StartTs:     startTime.Unix(),
		EndTs:       endTs,
		Location:    input.Location,
		Description: input.Description,
		AllDay:      input.AllDay,
		Confidence:  0.9,
		Reason:      "根据您的输入解析",
	}
}

// isConflictResult checks if a tool result indicates a schedule conflict.
func (a *SchedulerAgentV2) isConflictResult(result string) bool {
	lowerResult := strings.ToLower(result)
	return strings.Contains(lowerResult, "conflict") ||
		strings.Contains(lowerResult, "冲突") ||
		strings.Contains(lowerResult, "occupied") ||
		strings.Contains(lowerResult, "已占用")
}

// buildConflictResolutionData builds conflict resolution UI data from tool result.
// Parses the conflict error message to extract conflicting schedules and suggested alternatives.
func (a *SchedulerAgentV2) buildConflictResolutionData(toolResult string, pending *UIScheduleSuggestionData) *UIConflictResolutionData {
	// Try to parse the embedded conflict error from the tool result
	// Format: "ErrScheduleConflict: {\"conflicts\":[...],\"alternatives\":[...],\"original_start\":...}"
	conflictStr := "ErrScheduleConflict: "
	idx := strings.Index(toolResult, conflictStr)
	if idx == -1 {
		// Fallback: return basic conflict data without suggestions
		return &UIConflictResolutionData{
			NewSchedule:          *pending,
			ConflictingSchedules: []UIConflictSchedule{},
			SuggestedSlots:       []UITimeSlotData{},
			Actions:              []string{"override", "cancel"},
		}
	}

	// Extract the JSON part after "ErrScheduleConflict: "
	jsonPart := strings.TrimSpace(toolResult[idx+len(conflictStr):])

	// Parse the conflict error JSON
	var conflictErr struct {
		Conflicts []struct {
			ID      int64  `json:"id"`
			Title   string `json:"title"`
			StartTs int64  `json:"start_ts"`
			EndTs   int64  `json:"end_ts"`
		} `json:"conflicts"`
		Alternatives []struct {
			Start  *time.Time `json:"start"`
			End    *time.Time `json:"end"`
			Reason string     `json:"reason"`
		} `json:"alternatives"`
		OriginalStart *time.Time `json:"original_start"`
	}

	if err := json.Unmarshal([]byte(jsonPart), &conflictErr); err != nil {
		slog.Debug("failed to parse conflict error JSON", "error", err, "json_part", jsonPart)
		// Fallback: return basic conflict data
		return &UIConflictResolutionData{
			NewSchedule:          *pending,
			ConflictingSchedules: []UIConflictSchedule{},
			SuggestedSlots:       []UITimeSlotData{},
			Actions:              []string{"override", "cancel"},
		}
	}

	// Convert conflicting schedules to UI format
	conflictingSchedules := make([]UIConflictSchedule, 0, len(conflictErr.Conflicts))
	for _, c := range conflictErr.Conflicts {
		conflictingSchedules = append(conflictingSchedules, UIConflictSchedule{
			UID:       fmt.Sprintf("%d", c.ID),
			Title:     c.Title,
			StartTime: c.StartTs,
			EndTime:   c.EndTs,
		})
	}

	// Convert alternatives to UITimeSlotData
	suggestedSlots := make([]UITimeSlotData, 0, len(conflictErr.Alternatives))
	for _, alt := range conflictErr.Alternatives {
		suggestedSlots = append(suggestedSlots, UITimeSlotData{
			Label:   alt.Start.In(a.timezoneLoc).Format("15:04") + "-" + alt.End.In(a.timezoneLoc).Format("15:04"),
			StartTs: alt.Start.Unix(),
			EndTs:   alt.End.Unix(),
			Reason:  alt.Reason,
		})
	}

	// Also set auto_resolved if auto-resolution succeeded
	var autoResolved *UITimeSlotData
	if len(conflictErr.Alternatives) > 0 && conflictErr.OriginalStart != nil {
		// Check if the pending schedule was adjusted to the first alternative
		firstAlt := conflictErr.Alternatives[0]
		if pending.StartTs == firstAlt.Start.Unix() {
			autoResolved = &UITimeSlotData{
				Label:   firstAlt.Start.In(a.timezoneLoc).Format("15:04") + "-" + firstAlt.End.In(a.timezoneLoc).Format("15:04"),
				StartTs: firstAlt.Start.Unix(),
				EndTs:   firstAlt.End.Unix(),
				Reason:  firstAlt.Reason,
			}
		}
	}

	return &UIConflictResolutionData{
		NewSchedule:          *pending,
		ConflictingSchedules: conflictingSchedules,
		SuggestedSlots:       suggestedSlots,
		Actions:              []string{"reschedule", "override", "cancel"},
		AutoResolved:         autoResolved,
	}
}

// emitUIEvent emits a UI event by marshaling the data and calling the callback.
func (a *SchedulerAgentV2) emitUIEvent(callback func(event string, data string), eventType string, data interface{}) {
	if callback == nil {
		return
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		slog.Warn("failed to marshal UI event data",
			"event_type", eventType,
			"error", err)
		return
	}

	// Emit the UI event
	callback(eventType, string(jsonData))

	slog.Debug("emitted UI event",
		"event_type", eventType,
		"data", truncateString(string(jsonData), 200))
}

// handleScheduleQuery processes schedule_query tool calls and emits UI events.
func (a *SchedulerAgentV2) handleScheduleQuery(ctx context.Context, toolData string, callback func(event string, data string)) {
	// Format: "schedule_query:{JSON}"
	if !strings.HasPrefix(toolData, "schedule_query:") {
		return
	}

	// Extract JSON input
	jsonPart := strings.TrimPrefix(toolData, "schedule_query:")

	// Type assert to access RunWithStructuredResult
	queryTool, ok := a.queryTool.(interface {
		RunWithStructuredResult(ctx context.Context, inputJSON string) (*localtools.ScheduleQueryToolResult, error)
	})
	if !ok {
		slog.Warn("queryTool does not support RunWithStructuredResult")
		return
	}

	// Get structured result
	structuredResult, err := queryTool.RunWithStructuredResult(ctx, jsonPart)
	if err != nil {
		slog.Debug("failed to get structured query result", "error", err)
		return
	}

	// Only emit UI event if there are schedules to show
	if len(structuredResult.Schedules) == 0 {
		return
	}

	// Convert to UIScheduleListData
	scheduleItems := make([]UIScheduleItem, 0, len(structuredResult.Schedules))
	for _, s := range structuredResult.Schedules {
		scheduleItems = append(scheduleItems, UIScheduleItem{
			UID:      s.UID,
			Title:    s.Title,
			StartTs:  s.StartTs,
			EndTs:    s.EndTs,
			AllDay:   s.AllDay,
			Location: s.Location,
			Status:   s.Status,
		})
	}

	scheduleListData := UIScheduleListData{
		Title:     "日程列表",
		Query:     structuredResult.Query,
		Count:     structuredResult.Count,
		Schedules: scheduleItems,
		TimeRange: structuredResult.TimeRangeDescription,
		Reason:    "根据查询返回的日程",
	}

	a.emitUIEvent(callback, EventTypeUIScheduleList, scheduleListData)
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
