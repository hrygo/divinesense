package orchestrator

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	agents "github.com/hrygo/divinesense/ai/agents"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mock Registry for testing
type MockRegistry struct {
	mock.Mock
}

func (m *MockRegistry) GetAvailableExperts() []string {
	args := m.Called()
	return args.Get(0).([]string)
}

func (m *MockRegistry) GetExpertDescription(name string) string {
	args := m.Called(name)
	return args.String(0)
}

func (m *MockRegistry) ExecuteExpert(ctx context.Context, expertName string, input string, callback EventCallback) error {
	args := m.Called(ctx, expertName, input, callback)
	return args.Error(0)
}

func (m *MockRegistry) GetExpertConfig(name string) *agents.ParrotSelfCognition {
	args := m.Called(name)
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(*agents.ParrotSelfCognition)
}

// Helper to create a task
func createTask(id, agent, input string, deps []string) *Task {
	return &Task{
		ID:           id,
		Agent:        agent,
		Input:        input,
		Dependencies: deps,
		Status:       TaskStatusPending,
	}
}

// Case 1: 线性依赖 (A -> B -> C)
func TestDAG_LinearExecution(t *testing.T) {
	registry := new(MockRegistry)
	config := DefaultOrchestratorConfig()
	config.MaxParallelTasks = 3
	executor := NewExecutor(registry, config)

	// Setup tasks
	t1 := createTask("t1", "memo", "task 1", nil)
	t2 := createTask("t2", "memo", "task 2 {{t1.result}}", []string{"t1"})
	t3 := createTask("t3", "memo", "task 3 {{t2.result}}", []string{"t2"})

	plan := &TaskPlan{
		Tasks: []*Task{t1, t2, t3},
	}

	// Execution order tracking
	var order []string
	var mu sync.Mutex

	// Helper to simulate execution and return result
	mockExec := func(res string, task *Task) func(mock.Arguments) {
		return func(args mock.Arguments) {
			mu.Lock()
			order = append(order, task.ID)
			mu.Unlock()
			cb := args.Get(3).(EventCallback)
			if cb != nil {
				// Simulate streaming result
				cb("content", res)
			}
			time.Sleep(10 * time.Millisecond)
		}
	}

	registry.On("ExecuteExpert", mock.Anything, "memo", "task 1", mock.Anything).Return(nil).Run(mockExec("result1", t1))
	registry.On("ExecuteExpert", mock.Anything, "memo", "task 2 \"result1\"", mock.Anything).Return(nil).Run(mockExec("result2", t2))
	registry.On("ExecuteExpert", mock.Anything, "memo", "task 3 \"result2\"", mock.Anything).Return(nil).Run(mockExec("result3", t3))

	// Execute
	result := executor.ExecutePlan(context.Background(), plan, nil, "test-trace-id")

	// Assertions
	assert.Empty(t, result.Errors)
	assert.Equal(t, 3, len(result.Plan.Tasks)) // Or check results count if captured in result object
	assert.Equal(t, []string{"t1", "t2", "t3"}, order, "Execution order must be sequential due to dependencies")
	// Task result is updated by executor from callback
	assert.Equal(t, "result1", t1.Result)
	assert.Equal(t, "result2", t2.Result)
}

// Case 2: 菱形依赖 (A -> [B, C] -> D)
func TestDAG_DiamondExecution(t *testing.T) {
	registry := new(MockRegistry)
	config := DefaultOrchestratorConfig()
	config.MaxParallelTasks = 3
	executor := NewExecutor(registry, config)

	tA := createTask("A", "memo", "Root", nil)
	tB := createTask("B", "memo", "Branch B", []string{"A"})
	tC := createTask("C", "memo", "Branch C", []string{"A"})
	tD := createTask("D", "memo", "Join {{B.result}} {{C.result}}", []string{"B", "C"})

	plan := &TaskPlan{Tasks: []*Task{tA, tB, tC, tD}}

	var executed sync.Map

	mockExec := func(res string, delay time.Duration) func(mock.Arguments) {
		return func(args mock.Arguments) {
			cb := args.Get(3).(EventCallback)
			if cb != nil {
				cb("content", res)
			}
			time.Sleep(delay)
		}
	}

	// Mock A
	registry.On("ExecuteExpert", mock.Anything, "memo", "Root", mock.Anything).Return(nil).Run(func(args mock.Arguments) {
		mockExec("ResA", 0)(args)
		executed.Store("A", time.Now())
	})

	// Mock B & C (Parallel)
	registry.On("ExecuteExpert", mock.Anything, "memo", "Branch B", mock.Anything).Return(nil).Run(func(args mock.Arguments) {
		// Assert A finished
		_, aDone := executed.Load("A")
		assert.True(t, aDone, "A must be done before B")
		mockExec("ResB", 50*time.Millisecond)(args)
		executed.Store("B", time.Now())
	})
	registry.On("ExecuteExpert", mock.Anything, "memo", "Branch C", mock.Anything).Return(nil).Run(func(args mock.Arguments) {
		_, aDone := executed.Load("A")
		assert.True(t, aDone, "A must be done before C")
		mockExec("ResC", 50*time.Millisecond)(args)
		executed.Store("C", time.Now())
	})

	// Mock D
	registry.On("ExecuteExpert", mock.Anything, "memo", "Join \"ResB\" \"ResC\"", mock.Anything).Return(nil).Run(func(args mock.Arguments) {
		_, bDone := executed.Load("B")
		_, cDone := executed.Load("C")
		assert.True(t, bDone, "B must be done before D")
		assert.True(t, cDone, "C must be done before D")
		mockExec("ResD", 0)(args)
		executed.Store("D", time.Now())
	})

	// Reverse check for C.B case order if variable injection happens
	// Note: The input string match might fail if map iteration order varies,
	// so we might need loose matching in strict implementations.
	// specific "Join ResB ResC" assumes replacement order.
	// For now we assume implementation does consistent replacement.

	result := executor.ExecutePlan(context.Background(), plan, nil, "test-trace-id")
	assert.Empty(t, result.Errors)
}

// Case 3: 错误传播 (Cascade Skip)
func TestDAG_CascadeSkip(t *testing.T) {
	registry := new(MockRegistry)
	config := DefaultOrchestratorConfig()
	config.MaxParallelTasks = 3
	executor := NewExecutor(registry, config)

	t1 := createTask("t1", "memo", "FailTask", nil)
	t2 := createTask("t2", "memo", "Dependent", []string{"t1"})

	plan := &TaskPlan{Tasks: []*Task{t1, t2}}

	// T1 fails
	registry.On("ExecuteExpert", mock.Anything, "memo", "FailTask", mock.Anything).Return(fmt.Errorf("random error"))

	result := executor.ExecutePlan(context.Background(), plan, nil, "test-trace-id")

	// In the new design, partial failure doesn't necessarily return error for ExecutePlan if handled gracefully,
	// but here we expect the function to return the results map.

	// Ensure at least one error was reported (for t1)
	assert.NotEmpty(t, result.Errors)

	assert.Equal(t, TaskStatusFailed, t1.Status)
	assert.Equal(t, TaskStatusSkipped, t2.Status, "Dependent task should be skipped")
}

// Case 4: 循环依赖检测
func TestDAG_CircularDependency(t *testing.T) {
	registry := new(MockRegistry)
	config := DefaultOrchestratorConfig()
	config.MaxParallelTasks = 3
	executor := NewExecutor(registry, config)

	t1 := createTask("t1", "memo", "Task 1", []string{"t2"})
	t2 := createTask("t2", "memo", "Task 2", []string{"t1"})

	plan := &TaskPlan{Tasks: []*Task{t1, t2}}

	result := executor.ExecutePlan(context.Background(), plan, nil, "test-trace-id")

	assert.NotEmpty(t, result.Errors, "Should report cycle error")
	// assert.Contains(t, result.Errors[0], "cycle detected") // Exact message might vary
}

// Case 5: 变量替换 - 引用不存在的任务
func TestContextInjector_InvalidReference(t *testing.T) {
	registry := new(MockRegistry)
	config := DefaultOrchestratorConfig()
	config.MaxParallelTasks = 3
	executor := NewExecutor(registry, config)

	t1 := createTask("t1", "memo", "Task 1 {{ghost.result}}", nil)
	plan := &TaskPlan{Tasks: []*Task{t1}}

	// Should fail immediately before execution or during execution
	// Current plan says: "If referenced task not found/failed -> Error"

	// Mock execution attempts if it gets that far, but likely fails at injection
	// registry.On("ExecuteExpert", ...).Return(nil)

	result := executor.ExecutePlan(context.Background(), plan, nil, "test-trace-id")
	assert.NotEmpty(t, result.Errors)
	// assert.Contains(t, result.Errors[0], "reference not found")
}

// Case 6: 菱形依赖失败 (A fails -> B,C skipped -> D skipped)
func TestDAG_DiamondFailure(t *testing.T) {
	registry := new(MockRegistry)
	config := DefaultOrchestratorConfig()
	config.MaxParallelTasks = 3
	executor := NewExecutor(registry, config)

	tA := createTask("A", "memo", "Root", nil)
	tB := createTask("B", "memo", "Branch B", []string{"A"})
	tC := createTask("C", "memo", "Branch C", []string{"A"})
	tD := createTask("D", "memo", "Join", []string{"B", "C"})

	plan := &TaskPlan{Tasks: []*Task{tA, tB, tC, tD}}

	// Mock A failing
	registry.On("ExecuteExpert", mock.Anything, "memo", "Root", mock.Anything).Return(fmt.Errorf("root error"))

	result := executor.ExecutePlan(context.Background(), plan, nil, "test-diamond-fail")

	assert.NotEmpty(t, result.Errors)
	assert.Equal(t, TaskStatusFailed, tA.Status)
	assert.Equal(t, TaskStatusSkipped, tB.Status)
	assert.Equal(t, TaskStatusSkipped, tC.Status)
	assert.Equal(t, TaskStatusSkipped, tD.Status)
}

// Case 7: 重试后成功 (Retry Success)
func TestDAG_RetrySuccess(t *testing.T) {
	registry := new(MockRegistry)
	config := DefaultOrchestratorConfig()
	config.MaxParallelTasks = 3
	executor := NewExecutor(registry, config)

	t1 := createTask("t1", "memo", "FlakyTask", nil)
	plan := &TaskPlan{Tasks: []*Task{t1}}

	// Mock flaky behavior: fail once, then succeed
	// We use .Once() to enforce order if using strict mocks, or a counter closure
	attempts := 0
	mockExec := func(args mock.Arguments) {
		attempts++
	}

	registry.On("ExecuteExpert", mock.Anything, "memo", "FlakyTask", mock.Anything).
		Return(fmt.Errorf("transient error")).
		Run(mockExec).
		Once()

	registry.On("ExecuteExpert", mock.Anything, "memo", "FlakyTask", mock.Anything).
		Return(nil).
		Run(func(args mock.Arguments) {
			attempts++
			cb := args.Get(3).(EventCallback)
			if cb != nil {
				cb("content", "success_result")
			}
		}).
		Once()

	result := executor.ExecutePlan(context.Background(), plan, nil, "test-retry-success")

	assert.Empty(t, result.Errors)
	assert.Equal(t, TaskStatusCompleted, t1.Status)
	assert.Equal(t, "success_result", t1.Result)
	assert.Equal(t, 2, attempts, "Should have executed twice")
}
