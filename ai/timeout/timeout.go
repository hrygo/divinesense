// Package timeout defines centralized timeout constants for AI operations.
// Package timeout 定义 AI 操作的集中式超时常量。
package timeout

import "time"

// AI operation timeout constants.
// AI 操作超时常量。
const (
	// StreamTimeout is the timeout for streaming responses from LLM.
	// StreamTimeout 是 LLM 流式响应的超时时间。
	StreamTimeout = 5 * time.Minute

	// AgentTimeout is the timeout for agent execution.
	// AgentTimeout 是 Agent 执行的超时时间。
	AgentTimeout = 2 * time.Minute

	// AgentExecutionTimeout is an alias for AgentTimeout for backward compatibility.
	// AgentExecutionTimeout 是 AgentTimeout 的别名，用于向后兼容。
	AgentExecutionTimeout = AgentTimeout

	// ToolExecutionTimeout is the timeout for individual tool execution.
	// ToolExecutionTimeout 是单个工具执行的超时时间。
	ToolExecutionTimeout = 30 * time.Second

	// EmbeddingTimeout is the timeout for embedding generation.
	// EmbeddingTimeout 是向量生成的超时时间。
	EmbeddingTimeout = 30 * time.Second

	// MaxIterations is the maximum number of ReAct loop iterations.
	// MaxIterations 是 ReAct 循环的最大迭代次数。
	MaxIterations = 5

	// MaxToolIterations is an alias for MaxIterations.
	MaxToolIterations = MaxIterations

	// MaxRecentToolCalls is the number of recent tool calls to track for loop detection.
	// MaxRecentToolCalls 是用于循环检测的最近工具调用记录数量。
	MaxRecentToolCalls = 10

	// MaxToolFailures is the maximum number of consecutive failures before aborting.
	// MaxToolFailures 是工具连续失败的最大次数，超过后中止执行。
	MaxToolFailures = 3

	// MaxTruncateLength is the maximum length for truncating strings in logs.
	// MaxTruncateLength 是日志中字符串截断的最大长度。
	MaxTruncateLength = 200
)
