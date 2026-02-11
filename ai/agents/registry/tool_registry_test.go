// Package registry provides tests for tool registration.
package registry

import (
	"context"
	"strings"
	"testing"

	"github.com/hrygo/divinesense/ai/agents"
)

// mockTool is a simple tool implementation for testing.
type mockTool struct {
	name        string
	description string
	parameters  map[string]interface{}
}

func newMockTool(name, description string) agent.ToolWithSchema {
	return &mockTool{
		name:        name,
		description: description,
		parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"input": map[string]interface{}{
					"type":        "string",
					"description": "test input",
				},
			},
		},
	}
}

func (t *mockTool) Name() string        { return t.name }
func (t *mockTool) Description() string { return t.description }
func (t *mockTool) Parameters() map[string]interface{} {
	return t.parameters
}
func (t *mockTool) Run(_ context.Context, input string) (string, error) {
	return "mock result: " + input, nil
}

func TestToolRegistry_Register(t *testing.T) {
	// Clear registry before test
	Clear()

	tool := newMockTool("test_tool", "A test tool")
	err := Register("test_tool", tool)
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	// Verify tool is registered
	retrieved, ok := Get("test_tool")
	if !ok {
		t.Fatal("Tool not found after registration")
	}
	if retrieved.Name() != "test_tool" {
		t.Errorf("Expected tool name 'test_tool', got '%s'", retrieved.Name())
	}

	// Test duplicate registration
	err = Register("test_tool", tool)
	if err == nil {
		t.Error("Expected error when registering duplicate tool")
	}
}

func TestToolRegistry_RegisterWithMetadata(t *testing.T) {
	Clear()

	tool := newMockTool("memo_search", "Search memos")
	metadata := ToolMetadata{
		Category: CategoryMemo,
		Tags:     []string{"semantic", "search"},
		Version:  "1.0.0",
	}

	err := RegisterWithMetadata("memo_search", tool, metadata)
	if err != nil {
		t.Fatalf("RegisterWithMetadata failed: %v", err)
	}

	// Verify metadata
	entry, ok := GetWithMetadata("memo_search")
	if !ok {
		t.Fatal("Tool entry not found")
	}
	if entry.Metadata.Category != CategoryMemo {
		t.Errorf("Expected category %s, got %s", CategoryMemo, entry.Metadata.Category)
	}
	if len(entry.Metadata.Tags) != 2 {
		t.Errorf("Expected 2 tags, got %d", len(entry.Metadata.Tags))
	}
}

func TestToolRegistry_List(t *testing.T) {
	Clear()

	// Register multiple tools
	tools := []struct {
		name string
		cat  ToolCategory
	}{
		{"memo_search", CategoryMemo},
		{"schedule_add", CategorySchedule},
		{"semantic_search", CategorySearch},
	}

	for _, tt := range tools {
		tool := newMockTool(tt.name, tt.name+" description")
		if err := RegisterInCategory(tt.cat, tt.name, tool); err != nil {
			t.Fatalf("Failed to register %s: %v", tt.name, err)
		}
	}

	// Test List
	allTools := List()
	if len(allTools) != 3 {
		t.Errorf("Expected 3 tools, got %d", len(allTools))
	}

	// Test ListByCategory
	memoTools := ListByCategory(CategoryMemo)
	if len(memoTools) != 1 {
		t.Errorf("Expected 1 memo tool, got %d", len(memoTools))
	}
	if memoTools[0] != "memo_search" {
		t.Errorf("Expected 'memo_search', got '%s'", memoTools[0])
	}

	scheduleTools := ListByCategory(CategorySchedule)
	if len(scheduleTools) != 1 {
		t.Errorf("Expected 1 schedule tool, got %d", len(scheduleTools))
	}
}

func TestToolRegistry_ListWithTags(t *testing.T) {
	Clear()

	// Register tools with different tags
	tool1 := newMockTool("tool1", "Tool 1")
	RegisterWithMetadata("tool1", tool1, ToolMetadata{
		Tags: []string{"semantic", "vector"},
	})

	tool2 := newMockTool("tool2", "Tool 2")
	RegisterWithMetadata("tool2", tool2, ToolMetadata{
		Tags: []string{"keyword", "search"},
	})

	tool3 := newMockTool("tool3", "Tool 3")
	RegisterWithMetadata("tool3", tool3, ToolMetadata{
		Tags: []string{"semantic", "rerank"},
	})

	// Find tools with "semantic" tag
	semanticTools := ListWithTags("semantic")
	if len(semanticTools) != 2 {
		t.Errorf("Expected 2 tools with 'semantic' tag, got %d", len(semanticTools))
	}

	// Find tools with "keyword" tag
	keywordTools := ListWithTags("keyword")
	if len(keywordTools) != 1 {
		t.Errorf("Expected 1 tool with 'keyword' tag, got %d", len(keywordTools))
	}
}

func TestToolRegistry_Describe(t *testing.T) {
	Clear()

	// Register tools in different categories
	RegisterInCategory(CategoryMemo, "memo_search", newMockTool("memo_search", "Search memos"))
	RegisterInCategory(CategorySchedule, "schedule_add", newMockTool("schedule_add", "Add schedule"))

	// Get description
	desc := Describe()
	if desc == "" {
		t.Error("Describe returned empty string")
	}
	if !strings.Contains(desc, "MEMO") {
		t.Error("Description should contain MEMO category")
	}
	if !strings.Contains(desc, "SCHEDULE") {
		t.Error("Description should contain SCHEDULE category")
	}
}

func TestToolRegistry_DescribeTool(t *testing.T) {
	Clear()

	tool := newMockTool("test_tool", "A test tool")
	metadata := ToolMetadata{
		Category: CategoryMemo,
		Tags:     []string{"test"},
		Version:  "1.0.0",
	}
	RegisterWithMetadata("test_tool", tool, metadata)

	desc := DescribeTool("test_tool")
	if !strings.Contains(desc, "Name: test_tool") {
		t.Error("Description should contain tool name")
	}
	if !strings.Contains(desc, "Category: memo") {
		t.Error("Description should contain category")
	}
	if !strings.Contains(desc, "Version: 1.0.0") {
		t.Error("Description should contain version")
	}
}

func TestToolRegistry_Deprecated(t *testing.T) {
	Clear()

	// Register a deprecated tool
	oldTool := newMockTool("old_tool", "Old deprecated tool")
	RegisterWithMetadata("old_tool", oldTool, ToolMetadata{
		Category:     CategoryMemo,
		IsDeprecated: true,
		ReplacedBy:   "new_tool",
	})

	// Check deprecation info
	entry, _ := GetWithMetadata("old_tool")
	if !entry.Metadata.IsDeprecated {
		t.Error("Tool should be marked as deprecated")
	}
	if entry.Metadata.ReplacedBy != "new_tool" {
		t.Errorf("Expected ReplacedBy 'new_tool', got '%s'", entry.Metadata.ReplacedBy)
	}
}

func TestToolRegistry_Count(t *testing.T) {
	Clear()

	if Count() != 0 {
		t.Errorf("Expected empty registry, got %d tools", Count())
	}

	Register("tool1", newMockTool("tool1", "Tool 1"))
	RegisterInCategory(CategorySchedule, "tool2", newMockTool("tool2", "Tool 2"))

	if Count() != 2 {
		t.Errorf("Expected 2 tools, got %d", Count())
	}

	if CountByCategory(CategorySchedule) != 1 {
		t.Errorf("Expected 1 schedule tool, got %d", CountByCategory(CategorySchedule))
	}
}

func TestToolRegistry_Unregister(t *testing.T) {
	Clear()

	Register("tool1", newMockTool("tool1", "Tool 1"))
	if Count() != 1 {
		t.Fatal("Tool not registered")
	}

	Unregister("tool1")
	if Count() != 0 {
		t.Errorf("Expected empty registry after unregister, got %d tools", Count())
	}
}

func TestToolRegistry_MustGet(t *testing.T) {
	Clear()

	defer func() {
		if r := recover(); r == nil {
			t.Error("MustGet should panic when tool not found")
		}
	}()

	MustGet("nonexistent")
}

func TestToolRegistry_GetAll(t *testing.T) {
	Clear()

	Register("tool1", newMockTool("tool1", "Tool 1"))
	Register("tool2", newMockTool("tool2", "Tool 2"))

	allTools := GetAllTools()
	if len(allTools) != 2 {
		t.Errorf("Expected 2 tools, got %d", len(allTools))
	}

	allEntries := GetAllEntries()
	if len(allEntries) != 2 {
		t.Errorf("Expected 2 entries, got %d", len(allEntries))
	}
}

func TestToolFactoryRegistry(t *testing.T) {
	ClearFactories()

	// Create a simple factory
	factory := func(userID int32) (agent.ToolWithSchema, error) {
		return newMockTool("factory_tool", "Created by factory"), nil
	}

	// Register factory
	err := RegisterFactory("test_factory", factory)
	if err != nil {
		t.Fatalf("RegisterFactory failed: %v", err)
	}

	// Test GetFactory
	retrieved, ok := GetFactory("test_factory")
	if !ok {
		t.Fatal("Factory not found after registration")
	}

	// Test CreateTool
	tool, err := CreateTool("test_factory", 123)
	if err != nil {
		t.Fatalf("CreateTool failed: %v", err)
	}
	if tool.Name() != "factory_tool" {
		t.Errorf("Expected tool name 'factory_tool', got '%s'", tool.Name())
	}

	// Test factory creation
	createdTool, err := retrieved(456)
	if err != nil {
		t.Fatalf("Factory call failed: %v", err)
	}
	if createdTool.Name() != "factory_tool" {
		t.Errorf("Expected tool name 'factory_tool', got '%s'", createdTool.Name())
	}
}

func TestToolFactoryRegistry_WithMetadata(t *testing.T) {
	ClearFactories()

	factory := func(userID int32) (agent.ToolWithSchema, error) {
		return newMockTool("memo_tool", "Memo tool"), nil
	}

	metadata := ToolMetadata{
		Category: CategoryMemo,
		Tags:     []string{"test"},
		Version:  "2.0.0",
	}

	err := RegisterFactoryWithMetadata("memo_factory", factory, metadata)
	if err != nil {
		t.Fatalf("RegisterFactoryWithMetadata failed: %v", err)
	}

	// Get metadata
	retrievedMeta, ok := GetFactoryMetadata("memo_factory")
	if !ok {
		t.Fatal("Metadata not found")
	}
	if retrievedMeta.Category != CategoryMemo {
		t.Errorf("Expected category %s, got %s", CategoryMemo, retrievedMeta.Category)
	}
	if retrievedMeta.Version != "2.0.0" {
		t.Errorf("Expected version '2.0.0', got '%s'", retrievedMeta.Version)
	}
}

func TestToolFactoryRegistry_ListByCategory(t *testing.T) {
	ClearFactories()

	RegisterFactoryInCategory(CategoryMemo, "memo_tool", func(userID int32) (agent.ToolWithSchema, error) {
		return newMockTool("memo_tool", "Memo"), nil
	})
	RegisterFactoryInCategory(CategorySchedule, "schedule_tool", func(userID int32) (agent.ToolWithSchema, error) {
		return newMockTool("schedule_tool", "Schedule"), nil
	})
	RegisterFactoryInCategory(CategoryMemo, "memo_search", func(userID int32) (agent.ToolWithSchema, error) {
		return newMockTool("memo_search", "Search"), nil
	})

	memoFactories := ListFactoriesByCategory(CategoryMemo)
	if len(memoFactories) != 2 {
		t.Errorf("Expected 2 memo factories, got %d", len(memoFactories))
	}

	scheduleFactories := ListFactoriesByCategory(CategorySchedule)
	if len(scheduleFactories) != 1 {
		t.Errorf("Expected 1 schedule factory, got %d", len(scheduleFactories))
	}
}

func TestBuildToolFactoriesMap(t *testing.T) {
	ClearFactories()

	RegisterFactory("factory1", func(userID int32) (agent.ToolWithSchema, error) {
		return newMockTool("f1", "F1"), nil
	})
	RegisterFactory("factory2", func(userID int32) (agent.ToolWithSchema, error) {
		return newMockTool("f2", "F2"), nil
	})

	factoriesMap := BuildToolFactoriesMap()
	if len(factoriesMap) != 2 {
		t.Errorf("Expected 2 factories in map, got %d", len(factoriesMap))
	}

	if _, ok := factoriesMap["factory1"]; !ok {
		t.Error("factory1 not found in map")
	}
	if _, ok := factoriesMap["factory2"]; !ok {
		t.Error("factory2 not found in map")
	}
}

func TestInferCategory(t *testing.T) {
	tests := []struct {
		name     string
		expected ToolCategory
	}{
		{"memo_search", CategoryMemo},
		{"memo_create", CategoryMemo},
		{"schedule_add", CategorySchedule},
		{"schedule_query", CategorySchedule},
		{"semantic_search", CategorySearch},
		{"query_tool", CategorySearch},
		{"claude_code", CategoryAI},
		{"llm_tool", CategoryAI},
		{"system_tool", CategorySystem},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := inferCategory(tt.name)
			if got != tt.expected {
				t.Errorf("inferCategory(%q) = %v, want %v", tt.name, got, tt.expected)
			}
		})
	}
}
