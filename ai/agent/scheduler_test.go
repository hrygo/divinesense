package agent

import (
	"context"
	"testing"
	"time"

	"github.com/hrygo/divinesense/ai"
	"github.com/hrygo/divinesense/server/service/schedule"
	"github.com/hrygo/divinesense/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockLLM implements ai.LLMService for testing.
type MockLLM struct {
	mock.Mock
}

func (m *MockLLM) Chat(ctx context.Context, messages []ai.Message) (string, *ai.LLMCallStats, error) {
	args := m.Called(ctx, messages)
	return args.String(0), nil, args.Error(1)
}

func (m *MockLLM) ChatStream(ctx context.Context, messages []ai.Message) (<-chan string, <-chan *ai.LLMCallStats, <-chan error) {
	args := m.Called(ctx, messages)
	statsChan := make(chan *ai.LLMCallStats, 1)
	close(statsChan)
	return args.Get(0).(<-chan string), statsChan, args.Get(1).(<-chan error)
}

func (m *MockLLM) ChatWithTools(ctx context.Context, messages []ai.Message, tools []ai.ToolDescriptor) (*ai.ChatResponse, *ai.LLMCallStats, error) {
	args := m.Called(ctx, messages, tools)
	if args.Get(0) == nil {
		return nil, nil, args.Error(1)
	}
	return args.Get(0).(*ai.ChatResponse), nil, args.Error(1)
}

// MockScheduleService implements schedule.Service for testing.
type MockScheduleService struct {
	mock.Mock
}

func (m *MockScheduleService) FindSchedules(ctx context.Context, userID int32, start, end time.Time) ([]*schedule.ScheduleInstance, error) {
	args := m.Called(ctx, userID, start, end)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*schedule.ScheduleInstance), args.Error(1)
}

func (m *MockScheduleService) CreateSchedule(ctx context.Context, userID int32, create *schedule.CreateScheduleRequest) (*store.Schedule, error) {
	args := m.Called(ctx, userID, create)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*store.Schedule), args.Error(1)
}

func (m *MockScheduleService) UpdateSchedule(ctx context.Context, userID int32, id int32, update *schedule.UpdateScheduleRequest) (*store.Schedule, error) {
	args := m.Called(ctx, userID, id, update)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*store.Schedule), args.Error(1)
}

func (m *MockScheduleService) DeleteSchedule(ctx context.Context, userID int32, id int32) error {
	args := m.Called(ctx, userID, id)
	return args.Error(0)
}

func (m *MockScheduleService) CheckConflicts(ctx context.Context, userID int32, startTs int64, endTs *int64, excludeIDs []int32) ([]*store.Schedule, error) {
	args := m.Called(ctx, userID, startTs, endTs, excludeIDs)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*store.Schedule), args.Error(1)
}

// Helper to create a tool call response.
func mockToolCallResponse(toolName, args string, content string) *ai.ChatResponse {
	return &ai.ChatResponse{
		Content: content,
		ToolCalls: []ai.ToolCall{
			{
				ID:   "call_123",
				Type: "function",
				Function: ai.FunctionCall{
					Name:      toolName,
					Arguments: args,
				},
			},
		},
	}
}

// Helper to create a final answer response.
func mockFinalAnswer(answer string) *ai.ChatResponse {
	return &ai.ChatResponse{
		Content:   answer,
		ToolCalls: []ai.ToolCall{},
	}
}

// TestSchedulerAgentV2_Execute tests the framework-less scheduler agent.
func TestSchedulerAgentV2_Execute(t *testing.T) {
	mockLLM := new(MockLLM)
	mockSvc := new(MockScheduleService)

	// Setup V2 Agent
	agentSvc, err := NewSchedulerAgentV2(mockLLM, mockSvc, 1, "Asia/Shanghai")
	assert.NoError(t, err)

	// Step 1: LLM calls schedule_query
	queryArgs := `{"start_time": "2026-01-26T10:00:00+08:00", "end_time": "2026-01-26T11:00:00+08:00"}`
	mockLLM.On("ChatWithTools", mock.Anything, mock.Anything, mock.Anything).
		Return(mockToolCallResponse("schedule_query", queryArgs, "Checking..."), nil).
		Once()

	// Step 2: Tool execution
	mockSvc.On("FindSchedules", mock.Anything, int32(1), mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).
		Return([]*schedule.ScheduleInstance{}, nil).
		Times(1)

	// Step 3: LLM calls schedule_add
	addArgs := `{"title": "Meeting", "start_time": "2026-01-26T10:00:00+08:00"}`
	mockLLM.On("ChatWithTools", mock.Anything, mock.Anything, mock.Anything).
		Return(mockToolCallResponse("schedule_add", addArgs, "Creating..."), nil).
		Once()

	// Step 4: Tool execution
	mockSvc.On("CreateSchedule", mock.Anything, int32(1), mock.AnythingOfType("*schedule.CreateScheduleRequest")).
		Return(&store.Schedule{ID: 1, Title: "Meeting", StartTs: 1769421600}, nil).
		Once()

	// Note: No Step 5 needed - agent now returns formatted Chinese message directly after schedule creation

	// Execute
	resp, err := agentSvc.Execute(context.TODO(), "Create a meeting tomorrow at 10am")

	assert.NoError(t, err)
	assert.Contains(t, resp, "已创建") // Chinese response format: "✓ 已创建: Meeting..."

	mockLLM.AssertExpectations(t)
	mockSvc.AssertExpectations(t)
}

// TestSchedulerAgentV2_StateInjection verifies state handling.
func TestSchedulerAgentV2_StateInjection(t *testing.T) {
	mockLLM := new(MockLLM)
	mockSvc := new(MockScheduleService)

	// Setup V2 Agent
	agentSvc, err := NewSchedulerAgentV2(mockLLM, mockSvc, 1, "Asia/Shanghai")
	assert.NoError(t, err)

	// Create Context with State
	ctx := context.Background()
	convoCtx := NewConversationContext("test-session", 1, "Asia/Shanghai")
	convoCtx.WorkingState.CurrentStep = StepConflictResolve
	convoCtx.WorkingState.Conflicts = []*store.Schedule{
		{Title: "Existing Meeting"},
	}

	// Mock LLM response - verify agent handles context state correctly
	mockLLM.On("ChatWithTools", mock.Anything, mock.Anything, mock.Anything).
		Return(mockFinalAnswer("I see the conflict."), nil).
		Once()

	// Execute - should not panic with context state
	_, err = agentSvc.ExecuteWithCallback(ctx, "Resolve conflict", convoCtx, nil)
	assert.NoError(t, err)

	mockLLM.AssertExpectations(t)
}

// TestScheduleParrotV2_StreamChat tests the V2 parrot's streaming interface.
func TestScheduleParrotV2_StreamChat(t *testing.T) {
	mockLLM := new(MockLLM)
	mockSvc := new(MockScheduleService)

	// Setup V2 Agent and Parrot
	agentSvc, err := NewSchedulerAgentV2(mockLLM, mockSvc, 1, "Asia/Shanghai")
	assert.NoError(t, err)

	parrot, err := NewScheduleParrotV2(agentSvc)
	assert.NoError(t, err)

	// Mock direct final answer (no tool calls for simplicity)
	mockLLM.On("ChatWithTools", mock.Anything, mock.Anything, mock.Anything).
		Return(mockFinalAnswer("Hello! I can help you manage your schedule."), nil).
		Once()

	// Execute streaming
	ctx := context.Background()
	stream, err := parrot.StreamChat(ctx, "Hello", []string{})
	assert.NoError(t, err)

	// Collect response
	var response string
	for chunk := range stream {
		response += chunk
	}

	assert.Contains(t, response, "help you manage your schedule")
	mockLLM.AssertExpectations(t)
}

// TestConversationContext_ToJSON tests JSON serialization of context.
func TestConversationContext_ToJSON(t *testing.T) {
	ctx := NewConversationContext("test-session", 1, "Asia/Shanghai")
	ctx.AddTurn("Hello", "Hi there", []ToolCallRecord{
		{Tool: "schedule_query", Success: true},
	})

	data, err := ctx.ToJSON()
	assert.NoError(t, err)

	// Verify JSON can be parsed and contains key data
	assert.Contains(t, data, "test-session")
	assert.Contains(t, data, "Hello")

	// Note: Full unmarshal may fail due to unexported fields like mu (RWMutex)
	// We just verify the JSON string contains expected data
}

// TestNativeTool tests the NativeTool implementation.
func TestNativeTool(t *testing.T) {
	called := false
	tool := NewNativeTool(
		"test_tool",
		"A test tool",
		func(ctx context.Context, input string) (string, error) {
			called = true
			return "result: " + input, nil
		},
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"input": map[string]interface{}{
					"type":        "string",
					"description": "Input value",
				},
			},
		},
	)

	assert.Equal(t, "test_tool", tool.Name())
	assert.Equal(t, "A test tool", tool.Description())

	result, err := tool.Run(context.Background(), "test input")
	assert.NoError(t, err)
	assert.Equal(t, "result: test input", result)
	assert.True(t, called)

	params := tool.Parameters()
	assert.Equal(t, "object", params["type"])
}

// TestToolFromLegacy tests the legacy tool adapter.
func TestToolFromLegacy(t *testing.T) {
	// Simulate a legacy tool with InputType() method
	inputTypeFunc := func() map[string]interface{} {
		return map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"query": map[string]interface{}{
					"type": "string",
				},
			},
		}
	}

	tool := ToolFromLegacy(
		"legacy_tool",
		"A legacy tool",
		func(ctx context.Context, input string) (string, error) {
			return "legacy: " + input, nil
		},
		inputTypeFunc,
	)

	assert.Equal(t, "legacy_tool", tool.Name())
	assert.Equal(t, "A legacy tool", tool.Description())

	result, err := tool.Run(context.Background(), "test")
	assert.NoError(t, err)
	assert.Equal(t, "legacy: test", result)
}

// TestSchedulerAgentV2_Callback tests callback support in V2 agent.
func TestSchedulerAgentV2_Callback(t *testing.T) {
	mockLLM := new(MockLLM)
	mockSvc := new(MockScheduleService)

	agentSvc, err := NewSchedulerAgentV2(mockLLM, mockSvc, 1, "Asia/Shanghai")
	assert.NoError(t, err)

	var events []string
	var eventData []string
	callback := func(event, data string) {
		events = append(events, event)
		eventData = append(eventData, data)
	}

	// Mock: LLM calls tool
	queryArgs := `{"start_time": "2026-01-26T10:00:00+08:00", "end_time": "2026-01-26T11:00:00+08:00"}`
	mockLLM.On("ChatWithTools", mock.Anything, mock.Anything, mock.Anything).
		Return(mockToolCallResponse("schedule_query", queryArgs, "Checking..."), nil).
		Once()

	// Mock: FindSchedules is called twice during ExecuteWithCallback:
	// 1. Once by the tool.Run() during agent execution
	// 2. Once by the handleScheduleQuery callback that generates the tool_result event
	// This verifies the callback mechanism correctly invokes the service for event generation.
	mockSvc.On("FindSchedules", mock.Anything, int32(1), mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).
		Return([]*schedule.ScheduleInstance{}, nil).
		Times(2)

	// Mock: LLM provides final answer
	mockLLM.On("ChatWithTools", mock.Anything, mock.Anything, mock.Anything).
		Return(mockFinalAnswer("No conflicts found."), nil).
		Once()

	_, err = agentSvc.ExecuteWithCallback(context.Background(), "Check my schedule", nil, callback)
	assert.NoError(t, err)

	// Verify callback was called
	assert.Contains(t, events, "tool_use")
	assert.Contains(t, events, "tool_result")
	assert.Contains(t, events, "answer")
}

// TestSchedulerAgentV2_SelfDescription tests parrot self-description.
func TestSchedulerAgentV2_SelfDescription(t *testing.T) {
	mockLLM := new(MockLLM)
	mockSvc := new(MockScheduleService)

	agentSvc, err := NewSchedulerAgentV2(mockLLM, mockSvc, 1, "Asia/Shanghai")
	assert.NoError(t, err)

	parrot, err := NewScheduleParrotV2(agentSvc)
	assert.NoError(t, err)

	desc := parrot.SelfDescribe()
	assert.NotNil(t, desc)
	assert.Equal(t, "schedule", desc.Name)
	assert.Equal(t, "🦜", desc.Emoji)
	assert.Equal(t, "时巧 (Tick) - 日程助手鹦鹉", desc.Title)
	assert.Contains(t, desc.Capabilities, "创建日程事件")
	assert.Contains(t, desc.Capabilities, "查询时间安排")
}
