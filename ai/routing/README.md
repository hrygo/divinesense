# Routing (Intent Classification)

The `routing` package provides intelligent intent classification and routing for user queries.

## Overview

The routing system uses a four-layer approach to classify user intent:

```
Layer 0: Cache (LRU) → ~0ms, 60-70% hit rate
Layer 1: RuleMatcher → ~0ms, 20-30% hit rate
Layer 2: HistoryMatcher → ~10ms, 5-10% hit rate
Layer 3: LLM Classifier → ~400ms, <5% hit rate
```

## Directory Structure

```
routing/
├── service.go           # Main routing service (orchestrates all layers)
├── cache.go             # LRU cache for routing results
├── rule_matcher.go      # Rule-based classification (keywords)
├── history_matcher.go   # History-aware matching (vector similarity)
├── llm_intent_classifier.go # LLM fallback classification
├── feedback.go          # User feedback collection
├── interface.go         # Service interfaces
├── postgres_storage.go  # Feedback persistence
└── *test.go             # Test files
```

## Intent Types

```go
type Intent string

const (
    IntentMemoCreate      Intent = "memo_create"
    IntentMemoQuery       Intent = "memo_query"
    IntentScheduleCreate  Intent = "schedule_create"
    IntentScheduleQuery   Intent = "schedule_query"
    IntentScheduleUpdate  Intent = "schedule_update"
    IntentBatchSchedule   Intent = "batch_schedule"
    IntentUnknown         Intent = "unknown"
)

type AgentType string

const (
    AgentTypeMemo     AgentType = "memo"
    AgentTypeSchedule AgentType = "schedule"
    AgentTypeAmazing  AgentType = "amazing"
)
```

## Usage

```go
import "github.com/hrygo/divinesense/ai/routing"

// Create service
svc := routing.NewService(store, llmClient, routing.Config{
    CacheEnabled:   true,
    CacheSize:      500,
    RuleEnabled:    true,
    HistoryEnabled: true,
    LLMEnabled:     true,
})

// Classify intent
intent, confidence, err := svc.ClassifyIntent(ctx, "明天有什么安排")

// Convert to agent type
agentType := routing.IntentToAgentType(intent)
// Returns: AgentTypeSchedule
```

## Layer Details

### Layer 0: Cache (LRU)

Fast in-memory cache for previously classified queries.

```go
type Cache struct {
    capacity int
    ttl      time.Duration
}

// TTL Settings:
// - Rule matches: 5 minutes
// - LLM matches: 30 minutes
```

### Layer 1: RuleMatcher

Keyword-based matching with weighted scoring:

```go
// Time keywords (weight: 2)
"今天", "明天", "后天", "下周", "上午", "下午", "晚上"

// Action keywords (weight: 2)
"日程", "安排", "会议", "提醒", "预约"

// Query keywords (weight: 1)
"有什么", "查询", "显示"
```

**Scoring:**
```
score = Σ(keyword_weight × count)

Threshold: score >= 2 → rule match
```

### Layer 2: HistoryMatcher

Vector similarity matching against conversation history.

```go
// Uses conversation_context table
// Embeds query and compares with historical embeddings
// Returns intent from most similar historical query
```

### Layer 3: LLM Classifier

Lightweight LLM classification for ambiguous queries.

```go
// Uses Qwen/Qwen2.5-7B-Instruct (via SiliconFlow)
// Token limit: 50
// Temperature: 0 (deterministic)
// Output: JSON Schema {intent, confidence}
```

## Feedback Collection

The routing system collects feedback for continuous improvement:

```go
// Record feedback
svc.RecordFeedback(ctx, &routing.Feedback{
    SessionID:       sessionID,
    QueryText:       query,
    PredictedIntent: predicted,
    ActualIntent:    actual,
    Confidence:      confidence,
    FeedbackType:    "manual", // "auto", "manual", "correction"
})
```

**Feedback Schema:**
```sql
CREATE TABLE router_feedback (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL,
    session_id VARCHAR(64),
    query_text TEXT NOT NULL,
    predicted_intent VARCHAR(20) NOT NULL,
    actual_intent VARCHAR(20),
    confidence REAL,
    feedback_type VARCHAR(20),
    created_ts BIGINT NOT NULL
);
```

## Configuration

```go
type Config struct {
    // Cache settings
    CacheEnabled bool
    CacheSize    int  // Default: 500
    CacheTTL     time.Duration

    // Layer enable/disable
    RuleEnabled    bool
    HistoryEnabled bool
    LLMEnabled     bool

    // LLM settings
    LLMModel    string // Default: Qwen/Qwen2.5-7B-Instruct
    LLMMaxTokens int    // Default: 50
}
```

## Performance

| Layer | Latency | Hit Rate | Cumulative Hit |
|:------|:--------|:---------|:---------------|
| Cache | ~0ms | - | 60-70% |
| Rule | ~0ms | - | 80-90% |
| History | ~10ms | - | 90-95% |
| LLM | ~400ms | - | 100% |

## Testing

```bash
# Test routing system
go test ./ai/routing/... -v

# Run integration tests
go test ./ai/routing/... -tags=integration -v
```

## Intent to Agent Mapping

```go
func IntentToAgentType(intent Intent) AgentType {
    switch intent {
    case IntentMemoCreate, IntentMemoQuery:
        return AgentTypeMemo
    case IntentScheduleCreate, IntentScheduleQuery, IntentScheduleUpdate, IntentBatchSchedule:
        return AgentTypeSchedule
    default:
        return AgentTypeAmazing
    }
}
```

## Advanced Usage

### Custom Rule Weights

```go
ruleMatcher := routing.NewRuleMatcher(routing.RuleConfig{
    TimeKeywords:  []string{"今天", "明天", ...},
    TimeWeight:    2.0,
    ActionWeight:  2.0,
    QueryWeight:   1.0,
    Threshold:     2.0,
})
```

### Custom LLM Prompt

```go
llmClassifier := routing.NewLLMClassifier(routing.LLMConfig{
    Model:       "custom-model",
    MaxTokens:   100,
    Temperature: 0.1,
    SystemPrompt: "You are a specialized classifier...",
})
```

### History Configuration

```go
historyMatcher := routing.NewHistoryMatcher(routing.HistoryConfig{
    Store:         store,
    EmbeddingSvc:  embeddingSvc,
    SimilarityThreshold: 0.85,
    MaxResults:    5,
})
```

## Best Practices

1. **Enable all layers** for best accuracy/latency balance
2. **Monitor cache hit rate** - aim for >60%
3. **Collect feedback** to improve rule weights
4. **Use low temperature** for LLM (0-0.1) for consistency
5. **Limit token count** for LLM (50 tokens is sufficient)

## Troubleshooting

**High LLM usage (>10%):**
- Check rule weights - may need adjustment
- Review failed classifications for patterns
- Add missing keywords to rule matcher

**Low confidence scores:**
- May indicate ambiguous input
- Consider using follow-up questions
- Check if training data is representative

**Cache miss rate:**
- Normal for diverse queries
- Consider increasing cache size if memory allows
- Check TTL settings
