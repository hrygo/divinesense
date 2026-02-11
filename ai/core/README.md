# Core AI Services

The `core` package provides foundational AI services used throughout DivineSense.

## Overview

These are low-level, well-tested services that form the building blocks for higher-level AI functionality.

## Subpackages

### embedding/ - Vector Embedding Service

Converts text to dense vector representations for semantic search.

```go
import "github.com/hrygo/divinesense/ai/core/embedding"

svc := embedding.NewService(embedding.Config{
    Provider: "siliconflow",
    Model:    "BAAI/bge-m3",
    APIKey:   key,
    BaseURL:  "https://api.siliconflow.cn/v1",
})

vector, err := svc.Embed(ctx, "search query")
// Returns: []float32 (1024 dimensions for bge-m3)
```

**Key Features:**
- Multi-provider support (SiliconFlow, OpenAI, Ollama)
- Batch embedding for efficiency
- Configurable dimensions (default: 1024)

### llm/ - Language Model Service

Unified client for LLM providers with streaming and tool calling support.

```go
import "github.com/hrygo/divinesense/ai/core/llm"

svc := llm.NewService(llm.Config{
    Provider: "deepseek",
    Model:    "deepseek-chat",
    APIKey:   key,
})

// Synchronous chat
response, stats, err := svc.Chat(ctx, []llm.Message{
    {Role: "user", Content: "Hello"},
})

// Streaming chat
streamCh, statsCh, errCh := svc.ChatStream(ctx, messages)
for chunk := range streamCh {
    // Process streaming chunks
}

// Tool calling
response, stats, err := svc.ChatWithTools(ctx, messages, []llm.ToolDescriptor{
    {
        Name:        "search",
        Description: "Search the database",
        Parameters:  `{"type":"object","properties":{...}}`,
    },
})
```

**Key Features:**
- Synchronous and streaming modes
- Function calling support
- Token usage tracking with cache metrics
- Connection warmup for reduced latency

**LLMCallStats:**
```go
type LLMCallStats struct {
    PromptTokens       int    // Input tokens
    CompletionTokens   int    // Generated tokens
    TotalTokens        int    // Sum of both
    CacheReadTokens    int    // Tokens from cache (DeepSeek)
    CacheWriteTokens   int    // Tokens written to cache
    ThinkingDurationMs int64  // Time to first token
    GenerationDurationMs int64 // Streaming generation time
    TotalDurationMs    int64  // Total request time
}
```

### reranker/ - Result Reranking Service

Reorders search results using cross-attention models for improved relevance.

```go
import "github.com/hrygo/divinesense/ai/core/reranker"

svc := reranker.NewService(reranker.Config{
    Provider: "siliconflow",
    Model:    "BAAI/bge-reranker-v2-m3",
    APIKey:   key,
})

// Get reordered indices
indices, err := svc.Rerank(ctx, query, resultTexts)

// Get results with scores
results, err := svc.RerankWithScores(ctx, query, resultTexts)
```

**Key Features:**
- Improves retrieval quality by 15-30%
- Optional feature (disable for speed)
- Returns relevance scores

### retrieval/ - Adaptive Retrieval System

Hybrid search combining BM25 keyword matching with vector semantic search.

```go
import "github.com/hrygo/divinesense/ai/core/retrieval"

r := retrieval.NewAdaptiveRetriever(store, embeddingSvc, rerankerSvc)

results, err := r.Retrieve(ctx, &retrieval.RetrievalOptions{
    Query:    "user query",
    Strategy: "hybrid_standard", // BM25 + vector + RRF
    Limit:    10,
    MinScore: 0.5,
})
```

**Strategies:**

| Strategy | Description | Latency | Quality |
|:---------|:------------|:--------|:--------|
| `BM25Only` | Keyword search only | Fast | Low |
| `SemanticOnly` | Vector search only | Slow | Medium |
| `HybridStandard` | BM25 + vector + RRF | Medium | High |
| `FullPipeline` | Hybrid + reranker | Slow | Highest |

**RRF (Reciprocal Rank Fusion):**
```
score = Σ weight_i / (60 + rank_i)
```

## Configuration

Services are configured through `Config` structs:

```go
// Embedding
type Config struct {
    Provider   string  // siliconflow, openai, ollama
    Model      string  // Model name
    APIKey     string  // API key (not for ollama)
    BaseURL    string  // Base URL
    Dimensions int     // Vector dimensions (default: from model)
}

// LLM
type Config struct {
    Provider    string  // deepseek, openai, ollama
    Model       string  // Model name
    APIKey      string  // API key (not for ollama)
    BaseURL     string  // Base URL
    MaxTokens   int     // Default: 2048
    Temperature float32 // Default: 0.7
}

// Reranker
type Config struct {
    Provider string // siliconflow, openai
    Model    string // Model name
    APIKey   string // API key
    BaseURL  string // Base URL
    Enabled  bool   // Enable/disable
}
```

## Testing

```bash
# Test all core services
go test ./ai/core/... -v

# With coverage
go test ./ai/core/... -cover
```

## Integration with Store

Services integrate with the `store` package for data persistence:

```go
// For retrieval
store := &store.Store{
    MemoStore:     memostore.New(db),
    ScheduleStore: schedulestore.New(db),
}

// Embeddings are stored in memo_embedding table
// 1024-dimensional vectors using pgvector
```

## Performance Characteristics

| Service | Typical Latency | Notes |
|:--------|:----------------|:------|
| Embedding | 200-500ms | Per batch |
| LLM Chat | 500-2000ms | Depends on output length |
| LLM Stream | 50-200ms TTFT | Time to first token |
| Reranker | 100-300ms | Per 10 results |
| Retrieval | 50-200ms | Depends on strategy |
