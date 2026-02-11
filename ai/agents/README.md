# Agents (Parrots)

The `agents` package implements the AI agent system for DivineSense, known as the "Parrot" system.

## Overview

DivineSense uses AI agents metaphorically called "parrots" - each with a distinct personality, capabilities, and purpose. The system supports both configuration-driven parrots (UniversalParrot) and code-implemented parrots (GeekParrot, EvolutionParrot).

## Directory Structure

```
agents/
├── universal/              # Configuration-driven parrot system
│   ├── universal_parrot.go # Main UniversalParrot implementation
│   ├── config_loader.go    # YAML config loader
│   ├── parrot_factory.go   # Factory for creating parrots from config
│   ├── strategies.go       # Execution strategy interfaces
│   ├── direct_executor.go  # Direct tool calling strategy
│   ├── react_executor.go   # ReAct (thinking+acting) strategy
│   ├── planning_executor.go # Two-phase planning strategy
│   ├── reflexion_executor.go # Self-reflection strategy
│   ├── time_context.go     # Time-aware context handling
│   ├── utils.go            # Utility functions
│   └── *test.go            # Test files
├── tools/                  # Agent tool implementations
│   ├── memo_search.go      # Note search tool
│   ├── scheduler.go        # Schedule CRUD tools
│   ├── find_free_time.go   # Free time finder
│   ├── registry.go         # Tool registry
│   └── *test.go            # Tool tests
├── registry/               # Tool registration system
│   └── tool_registry.go    # Dynamic tool discovery
├── runner/                 # Agent execution runners
├── geek/                   # GeekParrot (Claude Code CLI)
├── base_parrot.go          # ParrotAgent interface
├── base_tool.go            # Tool interface
├── chat_router.go          # Chat-to-agent routing
├── geek_parrot.go          # GeekParrot implementation
├── evolution_parrot.go     # EvolutionParrot implementation
├── cc_runner.go            # Claude Code CLI runner
├── types.go                # Common type definitions
└── *test.go                # Test files
```

## Parrot Types

### UniversalParrot (Configuration-Driven)

The UniversalParrot can mimic any parrot through YAML configuration files:

```
config/parrots/
├── memo.yaml       # MemoParrot (灰灰) - Note search
├── schedule.yaml   # ScheduleParrot (时巧) - Schedule management
└── amazing.yaml    # AmazingParrot (折衷) - Comprehensive assistant
```

**Config Structure:**
```yaml
id: "MEMO"
name: "MemoParrot"
chinese_name: "灰灰"
emoji: "🦜"

# Personality
personality:
  - "curious"
  - "helpful"
  - "precise"

# Capabilities
capabilities:
  - "semantic_search"
  - "note_retrieval"

# System prompt
system_prompt: |
  You are 灰灰, a note search expert...

# Execution strategy
strategy: "react"  # direct | react | planning | reflexion
max_iterations: 10

# Available tools
tools:
  - memo_search

# Caching
enable_cache: true
cache_size: 100
cache_ttl: 5m
```

### GeekParrot (Code Execution)

Integrates with Claude Code CLI for code-related tasks:

```go
import "github.com/hrygo/divinesense/ai/agents"

geek := agents.NewGeekParrot(store, llm, userID, workdir)
err := geek.ExecuteWithCallback(ctx, userInput, history, callback)
```

**Features:**
- Executes code in isolated environment
- Dangerous operation detection
- Session management with timeout
- Real-time output streaming

### EvolutionParrot (Self-Evolution)

Advanced agent capable of modifying its own codebase:

```go
evolution := agents.NewEvolutionParrot(store, llm, repoPath)
err := evolution.ExecuteWithCallback(ctx, task, history, callback)
```

**Features:**
- Source code analysis
- Automated pull request creation
- Code review capabilities
- Sandbox execution

## Tools

Tools are the building blocks that agents use to interact with the system:

```go
type ToolWithSchema interface {
    Name() string
    Description() string
    Parameters() string  // JSON Schema
    Execute(ctx context.Context, args string) (string, error)
}
```

### Available Tools

| Tool | Description | Parameters |
|:-----|:------------|:-----------|
| `memo_search` | Semantic note search | query, limit, time_range |
| `schedule_add` | Create schedule | title, start_time, end_time |
| `schedule_query` | Query schedules | time_range, status |
| `schedule_update` | Update schedule | uid, updates |
| `find_free_time` | Find free slots | start, end, duration |

### Tool Registration

```go
// Tools are registered in the registry
registry := tools.NewRegistry()
registry.Register(memo_search.NewTool(store))
registry.Register(schedule_tools.NewAddTool(store))
registry.Register(schedule_tools.NewQueryTool(store))
```

## Execution Strategies

UniversalParrot supports multiple execution strategies:

### Direct (Native Function Calling)

```yaml
strategy: "direct"
```

- LLM calls tools directly
- Fastest for simple tasks
- Requires LLM with native tool support

### ReAct (Reasoning + Acting)

```yaml
strategy: "react"
max_iterations: 10
```

- Loop: Think → Act → Observe
- Better for multi-step reasoning
- Shows thinking process

### Planning (Two-Phase)

```yaml
strategy: "planning"
max_iterations: 5
```

- Phase 1: Create detailed plan
- Phase 2: Execute plan steps
- Best for complex multi-tool tasks

### Reflexion (Self-Reflection)

```yaml
strategy: "reflexion"
max_reflections: 3
```

- Execute → Reflect → Retry
- Learns from failures
- Higher quality but slower

## Event Callbacks

Agents emit events during execution:

```go
callback := func(eventType string, data interface{}) error {
    switch eventType {
    case agents.EventTypeThinking:
        // Agent is thinking
    case agents.EventTypeToolUse:
        // Tool invocation
        toolData := data.(*agents.ToolCallData)
    case agents.EventTypeToolResult:
        // Tool result
    case agents.EventTypeAnswer:
        // Final answer
        answer := data.(string)
    case agents.EventTypeError:
        // Error occurred
    case agents.EventTypeSessionStats:
        // Session statistics
        stats := data.(*agents.SessionStatsData)
    }
    return nil
}
```

## Chat Router

Routes user input to the appropriate parrot:

```go
import "github.com/hrygo/divinesense/ai/agents"

router := agents.NewChatRouter(routingSvc)
result, err := router.Route(ctx, userInput)

// Result.Route can be:
// - RouteTypeMemo → MemoParrot
// - RouteTypeSchedule → ScheduleParrot
// - RouteTypeAmazing → AmazingParrot
```

## Configuration

Parrot configs are stored in `config/parrots/`:

```bash
config/parrots/
├── memo.yaml
├── schedule.yaml
└── amazing.yaml
```

## Testing

```bash
# Test all agents
go test ./ai/agents/... -v

# Test specific parrot
go test ./ai/agents/universal/... -v

# Test with coverage
go test ./ai/agents/... -cover
```

## Extending

### Creating a New Tool

```go
// 1. Implement ToolWithSchema
type MyTool struct {
    store *store.Store
}

func (t *MyTool) Name() string { return "my_tool" }
func (t *MyTool) Description() string { return "Does something" }
func (t *MyTool) Parameters() string {
    return `{"type":"object","properties":{...}}`
}
func (t *MyTool) Execute(ctx context.Context, args string) (string, error) {
    // Implementation
}

// 2. Register in tools registry
registry.Register(&MyTool{store: store})

// 3. Add to parrot config
tools:
  - my_tool
```

### Creating a New Strategy

```go
// 1. Implement ExecutionStrategy
type MyStrategy struct {
    config *strategy.Config
}

func (s *MyStrategy) Execute(ctx context.Context, input string, tools ...ToolWithSchema) (string, *NormalSessionStats, error) {
    // Implementation
}

// 2. Register in resolver
resolver.Register("my_strategy", func(cfg *strategy.Config) (ExecutionStrategy, error) {
    return &MyStrategy{config: cfg}, nil
})
```

## Parrot Personalities

| Parrot | Chinese | Role | Personality |
|:-------|:--------|:-----|:------------|
| MemoParrot | 灰灰 | Note search | Curious, precise |
| ScheduleParrot | 时巧 | Schedule management | Organized, efficient |
| AmazingParrot | 折衷 | All-around | Balanced, helpful |
| GeekParrot | 极客 | Code execution | Technical, precise |
| EvolutionParrot | 进化 | Self-improvement | Analytical, careful |
