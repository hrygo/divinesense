//go:build e2e_manual
// +build e2e_manual

package mocks

import (
	"context"
	"fmt"

	agent "github.com/hrygo/divinesense/ai/agents"
)

// MemoSearchStub creates a stub tool for memo search functionality.
// MemoSearchStub 创建一个用于笔记搜索功能的存根工具。
func MemoSearchStub() agent.Tool {
	return agent.NewBaseTool(
		"memo_search",
		"搜索用户笔记",
		func(ctx context.Context, input string) (string, error) {
			// Return preset test data
			return `[{"id":1,"content":"Go 语言学习笔记","tags":["go","programming"]},{"id":2,"content":"Python 入门指南","tags":["python","programming"]}]`, nil
		},
	)
}

// ScheduleCreateStub creates a stub tool for schedule creation.
// ScheduleCreateStub 创建一个用于日程创建的存根工具。
func ScheduleCreateStub() agent.Tool {
	return agent.NewBaseTool(
		"schedule_create",
		"创建新日程",
		func(ctx context.Context, input string) (string, error) {
			// Return preset test data
			return `{"id":100,"title":"团队会议","start_time":"2024-01-01T10:00:00Z","duration":60,"created":true}`, nil
		},
	)
}

// ScheduleQueryStub creates a stub tool for schedule query.
// ScheduleQueryStub 创建一个用于日程查询的存根工具。
func ScheduleQueryStub() agent.Tool {
	return agent.NewBaseTool(
		"schedule_query",
		"查询日程",
		func(ctx context.Context, input string) (string, error) {
			// Return preset test data
			return `[{"id":1,"title":"团队周会","start_time":"2024-01-01T10:00:00Z","duration":60}]`, nil
		},
	)
}

// RegisterTestTools registers all test stub tools to the registry.
// RegisterTestTools 将所有测试存根工具注册到注册表。
func RegisterTestTools(registry *agent.ToolRegistry) error {
	if err := registry.Register(MemoSearchStub()); err != nil {
		return fmt.Errorf("failed to register memo_search: %w", err)
	}
	if err := registry.Register(ScheduleCreateStub()); err != nil {
		return fmt.Errorf("failed to register schedule_create: %w", err)
	}
	if err := registry.Register(ScheduleQueryStub()); err != nil {
		return fmt.Errorf("failed to register schedule_query: %w", err)
	}
	return nil
}
