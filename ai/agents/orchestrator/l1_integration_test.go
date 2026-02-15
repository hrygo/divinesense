//go:build e2e_manual
// +build e2e_manual

package orchestrator

import (
	"context"
	"sync"
	"testing"

	"github.com/hrygo/divinesense/ai/e2e/mocks"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockExpertRegistryForTest implements ExpertRegistry for testing.
type MockExpertRegistryForTest struct {
	mock.Mock
	results   map[string]string
	mu        sync.Mutex
	executeFn func(ctx context.Context, expertName string, input string, callback EventCallback) error
}

// NewMockExpertRegistryForTest creates a new MockExpertRegistryForTest.
func NewMockExpertRegistryForTest() *MockExpertRegistryForTest {
	return &MockExpertRegistryForTest{
		results:   make(map[string]string),
		executeFn: nil, // Use default implementation
	}
}

// WithExpertResult sets the result for a specific expert and input.
func (m *MockExpertRegistryForTest) WithExpertResult(expert, input, result string) *MockExpertRegistryForTest {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.results[expert+":"+input] = result
	return m
}

// GetAvailableExperts returns the list of available expert agent names.
func (m *MockExpertRegistryForTest) GetAvailableExperts() []string {
	return []string{"memo", "schedule"}
}

// GetExpertDescription returns a description of what an expert agent can do.
func (m *MockExpertRegistryForTest) GetExpertDescription(name string) string {
	switch name {
	case "memo":
		return "笔记搜索专家。搜索用户记录的笔记、文档、想法。"
	case "schedule":
		return "日程管理专家。创建、查询、更新日程。"
	default:
		return "专家代理: " + name
	}
}

// ExecuteExpert executes a task with the specified expert agent.
func (m *MockExpertRegistryForTest) ExecuteExpert(ctx context.Context, expertName string, input string, callback EventCallback) error {
	// Use custom execute function if set
	if m.executeFn != nil {
		return m.executeFn(ctx, expertName, input, callback)
	}

	// Default implementation
	m.mu.Lock()
	result, ok := m.results[expertName+":"+input]
	m.mu.Unlock()

	if !ok {
		result = "Mock result for " + expertName
	}

	if callback != nil {
		callback("content", result)
	}

	return nil
}

// Ensure MockExpertRegistryForTest implements ExpertRegistry.
var _ ExpertRegistry = (*MockExpertRegistryForTest)(nil)

// TestL1_SimpleTaskDecomposition tests simple task decomposition.
// Corresponds to TC-ORCH-001: Single Agent Task.
func TestL1_SimpleTaskDecomposition(t *testing.T) {
	// 1. Create Mock LLM - use default response for any input
	mockLLM := mocks.NewMockLLM().
		WithDefaultResponse(`{"analysis":"用户想要搜索笔记","tasks":[{"id":"1","agent":"memo","input":"Go 学习笔记","purpose":"搜索用户的笔记","dependencies":[]}],"parallel":false}`)

	// 2. Create Mock Expert Registry
	registry := NewMockExpertRegistryForTest().
		WithExpertResult("memo", "Go 学习笔记", `{"id":1,"content":"Go 语言学习笔记","tags":["go","programming"]}`)

	// 3. Create Orchestrator
	orch := NewOrchestrator(mockLLM, registry)

	// 4. Execute test
	result, err := orch.Process(context.Background(), "搜索我记录的 Go 学习笔记", nil)

	// 5. Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotNil(t, result.Plan)
	assert.Len(t, result.Plan.Tasks, 1)
	assert.Equal(t, "memo", result.Plan.Tasks[0].Agent)
	assert.Equal(t, "Go 学习笔记", result.Plan.Tasks[0].Input)
	assert.Empty(t, result.Plan.Tasks[0].Dependencies)
}

// TestL1_ComplexTaskDecomposition tests complex task decomposition (multi-task DAG).
// Corresponds to TC-ORCH-002: Multi-Agent Task with dependencies.
func TestL1_ComplexTaskDecomposition(t *testing.T) {
	// 1. Create Mock LLM - use default response
	mockLLM := mocks.NewMockLLM().
		WithDefaultResponse(`{"analysis":"用户想要搜索会议纪要并安排跟进会议","tasks":[{"id":"1","agent":"memo","input":"会议纪要","purpose":"搜索上次会议的纪要","dependencies":[]},{"id":"2","agent":"schedule","input":"下周的跟进会议","purpose":"安排跟进会议","dependencies":["1"]}],"parallel":false}`)

	// 2. Create Mock Expert Registry
	registry := NewMockExpertRegistryForTest().
		WithExpertResult("memo", "会议纪要", `{"id":1,"content":"2024年Q4评审会议纪要"}`).
		WithExpertResult("schedule", "下周的跟进会议", `{"id":100,"title":"跟进会议","start_time":"2024-01-15T10:00:00Z","duration":60}`)

	// 3. Create Orchestrator
	orch := NewOrchestrator(mockLLM, registry)

	// 4. Execute test
	result, err := orch.Process(context.Background(), "搜索上次会议的纪要，并帮我安排下周的跟进会议", nil)

	// 5. Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotNil(t, result.Plan)
	assert.Len(t, result.Plan.Tasks, 2)

	// Verify task order and dependencies
	task1 := result.Plan.Tasks[0]
	task2 := result.Plan.Tasks[1]

	assert.Equal(t, "memo", task1.Agent)
	assert.Equal(t, "schedule", task2.Agent)
	assert.Contains(t, task2.Dependencies, task1.ID)
}

// TestL1_ParallelTasks tests parallel task execution.
// Corresponds to TC-ORCH-003: Parallel Tasks.
func TestL1_ParallelTasks(t *testing.T) {
	// 1. Create Mock LLM - use default response
	mockLLM := mocks.NewMockLLM().
		WithDefaultResponse(`{"analysis":"用户想要同时搜索笔记和查看日程","tasks":[{"id":"1","agent":"memo","input":"Go 笔记","purpose":"搜索Go笔记","dependencies":[]},{"id":"2","agent":"schedule","input":"这周日程","purpose":"查看这周日程","dependencies":[]}],"parallel":true}`)

	// 2. Create Mock Expert Registry
	registry := NewMockExpertRegistryForTest().
		WithExpertResult("memo", "Go 笔记", `{"id":1,"content":"Go 语言学习笔记"}`).
		WithExpertResult("schedule", "这周日程", `[{"id":1,"title":"团队周会","start_time":"2024-01-01T10:00:00Z"}]`)

	// 3. Create Orchestrator
	orch := NewOrchestrator(mockLLM, registry)

	// 4. Execute test
	result, err := orch.Process(context.Background(), "帮我搜索 Go 笔记，同时看看这周有什么日程", nil)

	// 5. Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotNil(t, result.Plan)
	assert.Len(t, result.Plan.Tasks, 2)

	// Verify both tasks can be executed independently
	assert.Equal(t, "memo", result.Plan.Tasks[0].Agent)
	assert.Equal(t, "schedule", result.Plan.Tasks[1].Agent)
	assert.Empty(t, result.Plan.Tasks[0].Dependencies)
	assert.Empty(t, result.Plan.Tasks[1].Dependencies)
	assert.True(t, result.Plan.Parallel, "Tasks should be marked as parallel")
}

// TestL1_SimpleTaskExecution tests simple task execution flow.
// Corresponds to TC-ORCH-004: Single Task Execution.
func TestL1_SimpleTaskExecution(t *testing.T) {
	// 1. Create Mock LLM
	mockLLM := mocks.NewMockLLM().
		WithDefaultResponse(`{"analysis":"用户想要查找React相关笔记","tasks":[{"id":"1","agent":"memo","input":"React","purpose":"搜索React笔记","dependencies":[]}],"parallel":false}`)

	// 2. Create Mock Expert Registry
	registry := NewMockExpertRegistryForTest().
		WithExpertResult("memo", "React", `{"id":2,"content":"React 入门指南","tags":["react","frontend"]}`)

	// 3. Create Orchestrator
	orch := NewOrchestrator(mockLLM, registry)

	// 4. Execute test
	result, err := orch.Process(context.Background(), "查找关于 React 的笔记", nil)

	// 5. Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotNil(t, result.Plan)
	assert.Len(t, result.Plan.Tasks, 1)

	// Verify task execution result
	task := result.Plan.Tasks[0]
	assert.Equal(t, TaskStatusCompleted, task.Status)
	assert.NotEmpty(t, task.Result)
}

// TestL1_TaskFailureHandling tests task failure handling.
// Corresponds to TC-ORCH-005: Task Failure.
func TestL1_TaskFailureHandling(t *testing.T) {
	// 1. Create Mock LLM
	mockLLM := mocks.NewMockLLM().
		WithDefaultResponse(`{"analysis":"用户想要搜索笔记","tasks":[{"id":"1","agent":"memo","input":"不存在的笔记","purpose":"搜索笔记","dependencies":[]}],"parallel":false}`)

	// 2. Create Mock Expert Registry that returns error on execution
	failRegistry := NewMockExpertRegistryForTest()
	// Override to return error
	failRegistry.executeFn = func(ctx context.Context, expertName string, input string, callback EventCallback) error {
		return assert.AnError
	}

	// 3. Create Orchestrator
	orch := NewOrchestrator(mockLLM, failRegistry)

	// 4. Execute test
	result, err := orch.Process(context.Background(), "搜索不存在的笔记", nil)

	// 5. Assert - error should be logged but flow continues
	assert.NoError(t, err, "Process should not return error even when task fails")
	assert.NotNil(t, result)
	assert.NotNil(t, result.Plan)
	assert.Len(t, result.Plan.Tasks, 1)

	// Verify task failure status
	task := result.Plan.Tasks[0]
	assert.Equal(t, TaskStatusFailed, task.Status)
	assert.NotEmpty(t, task.Error)
}
