# AI Module

The `ai` package provides the core AI capabilities for DivineSense, including embedding services, language model integrations, intent routing, and agent implementations.

## Overview

This module is organized into several key subsystems:

- **`core/`** - Foundational AI services (embedding, LLM, retrieval, reranking)
- **`agents/`** - AI agent implementations (Parrots)
- **`routing/`** - Intent classification and routing system
- **`services/`** - High-level service abstractions
- **`observability/`** - Metrics, tracing, and monitoring

## Directory Structure

```
ai/
├── core/               # Core AI services
│   ├── embedding/      # Vector embedding service
│   ├── llm/           # Language Model client
│   ├── reranker/      # Result reranking service
│   └── retrieval/     # Hybrid retrieval (BM25 + vector)
├── agents/            # AI agent implementations
│   ├── universal/     # Configuration-driven parrot system
│   ├── tools/         # Agent tools (memo_search, schedule_add, etc.)
│   ├── registry/      # Tool registration and discovery
│   ├── runner/        # Agent execution runners
│   ├── base_parrot.go # Base parrot interface
│   ├── chat_router.go # Chat-to-agent routing
│   ├── geek_parrot.go # Claude Code CLI integration
│   └── evolution_parrot.go # Self-evolution agent
├── routing/           # Intent classification and routing
│   ├── cache.go       # LRU routing cache
│   ├── rule_matcher.go # Rule-based classification (0ms)
│   ├── history_matcher.go # History-aware matching (~10ms)
│   ├── llm_intent_classifier.go # LLM fallback (~400ms)
│   └── service.go     # Unified routing service
├── services/          # High-level services
│   ├── schedule/      # Schedule AI services
│   └── ...
├── observability/     # Metrics and monitoring
├── cache/            # Semantic caching layer
├── context/          # LLM context construction
├── memory/           # Episodic memory storage
├── session/          # Conversation persistence
└── metrics/          # Performance metrics

# Root-level files
├── config.go         # AI configuration
├── embedding.go      # Legacy embedding (use core/embedding)
├── llm.go           # Legacy LLM (use core/llm)
└── reranker.go      # Legacy reranker (use core/reranker)
```

## Core Services

### EmbeddingService

```go
type EmbeddingService interface {
    Embed(ctx context.Context, text string) ([]float32, error)
    EmbedBatch(ctx context.Context, texts []string) ([][]float32, error)
    Dimensions() int
}
```

### LLMService

```go
type LLMService interface {
    Chat(ctx context.Context, messages []Message) (string, *LLMCallStats, error)
    ChatStream(ctx context.Context, messages []Message) (<-chan string, <-chan *LLMCallStats, <-chan error)
    ChatWithTools(ctx context.Context, messages []Message, tools []ToolDescriptor) (*ChatResponse, *LLMCallStats, error)
    Warmup(ctx context.Context)
}
```

### RerankerService

```go
type RerankerService interface {
    Rerank(ctx context.Context, query string, results []string) ([]int, error)
    RerankWithScores(ctx context.Context, query string, results []string) ([]RerankResult, error)
}
```

## Agents (Parrots)

DivineSense uses a "parrot" metaphor for its AI agents:

| Parrot          | ID        | Description                 | Config                         |
| :-------------- | :-------- | :-------------------------- | :----------------------------- |
| MemoParrot      | MEMO      | Note search and retrieval   | `config/parrots/memo.yaml`     |
| ScheduleParrot  | SCHEDULE  | Schedule management         | `config/parrots/schedule.yaml` |
| AmazingParrot   | AMAZING   | Comprehensive assistant     | `config/parrots/amazing.yaml`  |
| GeekParrot      | GEEK      | Claude Code CLI integration | Code implementation            |
| EvolutionParrot | EVOLUTION | Self-evolution              | Code implementation            |

### Using Agents

```go
import "github.com/hrygo/divinesense/ai/agents"

// Create a UniversalParrot from config
parrot, err := universal.NewUniversalParrot(config, llm, tools, userID)

// Execute with callback
err = parrot.ExecuteWithCallback(ctx, userInput, history, func(eventType string, data interface{}) error {
    switch eventType {
    case agents.EventTypeThinking:
        // Agent is thinking
    case agents.EventTypeToolUse:
        // Agent is using a tool
    case agents.EventTypeAnswer:
        // Final answer
    }
    return nil
})
```

## Routing System

The routing system classifies user intent through four layers:

```
Layer 0: Cache (LRU) → ~0ms
Layer 1: RuleMatcher → ~0ms
Layer 2: HistoryMatcher → ~10ms
Layer 3: LLM Classifier → ~400ms
```

```go
import "github.com/hrygo/divinesense/ai/routing"

router := routing.NewService(store, llm, config)
intent, confidence, err := router.ClassifyIntent(ctx, userInput)
```

## Configuration

AI configuration is loaded from environment variables:

```bash
# Enable AI
DIVINESENSE_AI_ENABLED=true

# Unified LLM Configuration (Main Chat)
# Supports: zai, deepseek, openai, siliconflow, dashscope, openrouter, ollama
DIVINESENSE_AI_LLM_PROVIDER=zai
DIVINESENSE_AI_LLM_API_KEY=your_unified_key
DIVINESENSE_AI_LLM_BASE_URL=https://open.bigmodel.cn/api/paas/v4
DIVINESENSE_AI_LLM_MODEL=glm-4.7

# Embedding Service
DIVINESENSE_AI_EMBEDDING_PROVIDER=siliconflow
DIVINESENSE_AI_EMBEDDING_API_KEY=your_embedding_key
DIVINESENSE_AI_EMBEDDING_BASE_URL=https://api.siliconflow.cn/v1
DIVINESENSE_AI_EMBEDDING_MODEL=BAAI/bge-m3

# Reranker Service
DIVINESENSE_AI_RERANK_PROVIDER=siliconflow
DIVINESENSE_AI_RERANK_API_KEY=your_rerank_key
DIVINESENSE_AI_RERANK_BASE_URL=https://api.siliconflow.cn/v1
DIVINESENSE_AI_RERANK_MODEL=BAAI/bge-reranker-v2-m3

# Intent Classification
DIVINESENSE_AI_INTENT_PROVIDER=siliconflow
DIVINESENSE_AI_INTENT_API_KEY=your_intent_key
DIVINESENSE_AI_INTENT_BASE_URL=https://api.siliconflow.cn/v1
DIVINESENSE_AI_INTENT_MODEL=Qwen/Qwen2.5-7B-Instruct
```



## Testing

Run tests for the AI module:

```bash
# All AI tests
go test ./ai/... -v

# Specific subsystem
go test ./ai/core/... -v
go test ./ai/agents/... -v
go test ./ai/routing/... -v

# With coverage
go test ./ai/... -cover
```

## Event Types

Agents emit structured events during execution:

| Event           | Description              | Data Type        |
| :-------------- | :----------------------- | :--------------- |
| `thinking`      | Agent is thinking        | string           |
| `tool_use`      | Tool invocation          | ToolCallData     |
| `tool_result`   | Tool result              | ToolResultData   |
| `answer`        | Final answer             | string           |
| `error`         | Error occurred           | error            |
| `phase_change`  | Processing phase changed | PhaseChangeEvent |
| `progress`      | Progress update          | ProgressEvent    |
| `session_stats` | Session statistics       | SessionStatsData |

## See Also

- [core/README.md](./core/README.md) - Core services documentation
- [agents/README.md](./agents/README.md) - Agent system documentation
- [routing/README.md](./routing/README.md) - Routing system documentation
