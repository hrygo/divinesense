package agent

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// Tool is the interface for agent tools.
// Tool 是代理工具的接口。
type Tool interface {
	// Name returns the name of the tool.
	Name() string

	// Description returns a description of what the tool does.
	Description() string

	// Run executes the tool with the given input.
	// Run 使用给定的输入执行工具。
	Run(ctx context.Context, input string) (string, error)
}

// BaseTool provides a reusable base implementation for tools.
// BaseTool 为工具提供可复用的基础实现。
type BaseTool struct {
	execute     func(ctx context.Context, input string) (string, error)
	validate    func(input string) error
	name        string
	description string
	timeout     time.Duration
}

// ToolOption is a function that configures a BaseTool.
// ToolOption 是配置 BaseTool 的函数。
type ToolOption func(*BaseTool)

// WithTimeout sets a timeout for tool execution.
// WithTimeout 设置工具执行的超时时间。
func WithTimeout(timeout time.Duration) ToolOption {
	return func(t *BaseTool) {
		t.timeout = timeout
	}
}

// WithValidator sets a custom input validator.
// WithValidator 设置自定义输入验证器。
func WithValidator(validator func(input string) error) ToolOption {
	return func(t *BaseTool) {
		t.validate = validator
	}
}

// NewBaseTool creates a new BaseTool.
// NewBaseTool 创建一个新的 BaseTool。
//
// Parameters:
//   - name: The name of the tool
//   - description: A description of what the tool does
//   - execute: The function to execute when the tool is run
//   - opts: Optional configuration functions
//
// Example:
//
//	tool := NewBaseTool(
//	    "my_tool",
//	    "Does something useful",
//	    func(ctx context.Context, input string) (string, error) {
//	        return "result", nil
//	    },
//	    WithTimeout(30*time.Second),
//	)
func NewBaseTool(
	name string,
	description string,
	execute func(ctx context.Context, input string) (string, error),
	opts ...ToolOption,
) *BaseTool {
	tool := &BaseTool{
		name:        name,
		description: description,
		execute:     execute,
		timeout:     30 * time.Second, // Default timeout
		validate:    defaultValidator,
	}

	// Apply options
	for _, opt := range opts {
		opt(tool)
	}

	return tool
}

// Name returns the name of the tool.
// Name 返回工具名称。
func (t *BaseTool) Name() string {
	return t.name
}

// Description returns the description of the tool.
// Description 返回工具描述。
func (t *BaseTool) Description() string {
	return t.description
}

// Run executes the tool with validation and error handling.
// Run 执行工具，包含验证和错误处理。
func (t *BaseTool) Run(ctx context.Context, input string) (string, error) {
	// 1. Input validation
	if err := t.validate(input); err != nil {
		return "", fmt.Errorf("input validation failed: %w", err)
	}

	// 2. Apply timeout if set
	execCtx := ctx
	if t.timeout > 0 {
		var cancel context.CancelFunc
		execCtx, cancel = context.WithTimeout(ctx, t.timeout)
		defer cancel()
	}

	// 3. Execute the tool
	result, err := t.execute(execCtx, input)
	if err != nil {
		return "", fmt.Errorf("tool execution failed: %w", err)
	}

	// 4. Validate result
	if strings.TrimSpace(result) == "" {
		return "", fmt.Errorf("tool returned empty result")
	}

	return result, nil
}

// defaultValidator provides basic input validation.
// defaultValidator 提供基本的输入验证。
func defaultValidator(input string) error {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return fmt.Errorf("input cannot be empty")
	}
	return nil
}

// ToolRegistry manages a collection of tools.
// ToolRegistry 管理工具集合。
type ToolRegistry struct {
	tools map[string]Tool
}

// NewToolRegistry creates a new ToolRegistry.
// NewToolRegistry 创建一个新的 ToolRegistry。
func NewToolRegistry() *ToolRegistry {
	return &ToolRegistry{
		tools: make(map[string]Tool),
	}
}

// Register adds a tool to the registry.
// Register 向注册表添加工具。
func (r *ToolRegistry) Register(tool Tool) error {
	if tool == nil {
		return fmt.Errorf("tool cannot be nil")
	}
	name := tool.Name()
	if name == "" {
		return fmt.Errorf("tool name cannot be empty")
	}
	if _, exists := r.tools[name]; exists {
		return fmt.Errorf("tool %s already registered", name)
	}
	r.tools[name] = tool
	return nil
}

// Get retrieves a tool by name.
// Get 按名称获取工具。
func (r *ToolRegistry) Get(name string) (Tool, bool) {
	tool, exists := r.tools[name]
	return tool, exists
}

// List returns all registered tool names.
// List 返回所有已注册的工具名称。
func (r *ToolRegistry) List() []string {
	names := make([]string, 0, len(r.tools))
	for name := range r.tools {
		names = append(names, name)
	}
	return names
}

// Describe returns a description string for all tools.
// Describe 返回所有工具的描述字符串。
func (r *ToolRegistry) Describe() string {
	if len(r.tools) == 0 {
		return "No tools available"
	}

	var desc strings.Builder
	desc.Grow(256) // Pre-allocate buffer

	for _, name := range r.List() {
		tool, _ := r.Get(name)
		desc.WriteString("- ")
		desc.WriteString(name)
		desc.WriteString(": ")
		desc.WriteString(tool.Description())
		desc.WriteString("\n")
	}

	return desc.String()
}

// Count returns the number of registered tools.
// Count 返回已注册工具的数量。
func (r *ToolRegistry) Count() int {
	return len(r.tools)
}

// ToolResult represents the result of a tool execution.
// ToolResult 表示工具执行的结果。
type ToolResult struct {
	Name      string        `json:"name"`
	Input     string        `json:"input"`
	Output    string        `json:"output"`
	Error     string        `json:"error"`
	Duration  time.Duration `json:"duration"`
	Timestamp int64         `json:"timestamp"`
	Success   bool          `json:"success"`
}

// NewToolResult creates a new ToolResult.
// NewToolResult 创建一个新的 ToolResult。
func NewToolResult(name, input, output string, duration time.Duration, err error) *ToolResult {
	result := &ToolResult{
		Name:      name,
		Input:     input,
		Output:    output,
		Duration:  duration,
		Success:   err == nil,
		Timestamp: time.Now().Unix(),
	}
	if err != nil {
		result.Error = err.Error()
	}
	return result
}
