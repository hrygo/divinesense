# LLM 统计收集规格

> **状态**: 🔲 待开发 | **优先级**: P1 (重要)
> **投入**: 3人天 | **负责团队**: 团队 A (AI Core)
> **关联 Issue**: [#79](https://github.com/hrygo/divinesense/issues/79)
> **版本**: v0.97.0

---

## 1. 目标与背景

### 1.1 核心目标

将 AI 会话统计收集逻辑下沉到 LLM 层，为普通模式（MemoParrot/ScheduleParrot/AmazingParrot）提供完整的 Session Summary，包括 Token 使用量、时间分解等统计数据。

### 1.2 当前问题

| 模式 | Session Summary 完整度 | 问题 |
|:-----|:-----------------------|:-----|
| **Geek/Evolution** | ✅ 完整 | 通过 CC Runner 获取详细统计 |
| **Normal** | ❌ 不完整 | 仅显示基础 duration，缺少 token/tool 统计 |

**根本原因**：
- LLM 调用层已产生 `resp.Usage` 数据（Token 统计），但未返回给 Agent
- Agent 层无法获取 LLM 统计，导致 `SessionStatsProvider` 无法实现

### 1.3 用户价值

- 普通模式用户可查看完整的 AI 调用统计（Token 使用、工具调用、时间分解）
- 与 Geek/Evolution 模式体验一致
- 帮助用户理解 AI 资源消耗（成本追踪）

---

## 2. Token 统计数据结构

### 2.1 LLMCallStats

单次 LLM 调用的统计数据（Immutable Data）：

```go
// ai/llm.go

type LLMCallStats struct {
    // Token 计数
    PromptTokens     int
    CompletionTokens int
    TotalTokens      int

    // 缓存统计
    CacheReadTokens  int  // 缓存读取 Token 数
    CacheWriteTokens int  // 缓存写入 Token 数

    // 时间统计 (毫秒)
    ThinkingDurationMs   int64  // 首字延迟 (Time to First Token)
    GenerationDurationMs int64  // 生成时长
    TotalDurationMs      int64  // 总时长
}
```

### 2.2 NormalSessionStats

Agent 级别聚合的会话统计：

```go
// ai/agent/universal/universal_parrot.go

type NormalSessionStats struct {
    mu sync.Mutex

    // 会话标识
    StartTime time.Time `json:"start_time"`
    EndTime   time.Time `json:"end_time"`
    AgentType string    `json:"agent_type"`
    ModelUsed string    `json:"model_used"`

    // Token 使用
    PromptTokens     int `json:"prompt_tokens"`
    CompletionTokens int `json:"completion_tokens"`
    TotalTokens      int `json:"total_tokens"`
    CacheReadTokens  int `json:"cache_read_tokens,omitempty"`
    CacheWriteTokens int `json:"cache_write_tokens,omitempty"`

    // 时间统计 (毫秒)
    ThinkingDurationMs   int64 `json:"thinking_duration_ms"`
    GenerationDurationMs int64 `json:"generation_duration_ms"`
    TotalDurationMs      int64 `json:"total_duration_ms"`

    // 工具使用
    ToolCallCount int      `json:"tool_call_count"`
    ToolsUsed     []string `json:"tools_used,omitempty"`

    // 成本估算
    TotalCostMilliCents int64 `json:"total_cost_milli_cents"`
}
```

---

## 3. 成本计算方法

### 3.1 定价模型

| LLM Provider | Input (¥/1M tokens) | Output (¥/1M tokens) |
|:-------------|:-------------------|:--------------------|
| DeepSeek | 1.0 | 2.0 |
| SiliconFlow (Embedding) | 0.1 | - |
| SiliconFlow (Reranker) | 0.1 | - |

### 3.2 计算公式

```go
// 计算单次 LLM 调用成本 (毫美分: 1/1000 美分)
func CalculateCost(stats *LLMCallStats, provider string) int64 {
    var inputCost, outputCost float64

    switch provider {
    case "deepseek":
        // ¥1/M input, ¥2/M output
        // 汇率假设: 1 USD ≈ 7.2 CNY
        inputCost = float64(stats.PromptTokens) * 1.0 / 1_000_000 / 7.2 * 100_000  // 转为毫美分
        outputCost = float64(stats.CompletionTokens) * 2.0 / 1_000_000 / 7.2 * 100_000
    case "siliconflow":
        // Embedding/Reranker: ¥0.1/M
        inputCost = float64(stats.TotalTokens) * 0.1 / 1_000_000 / 7.2 * 100_000
    }

    return int64(inputCost + outputCost)
}
```

### 3.3 缓存优化效益

缓存命中可大幅降低成本：

| 轮次 | Prompt Tokens | Cache Hit | 缓存率 | Input Cost |
|:-----|:--------------|:---------|:-------|:-----------|
| 第1轮 | 5000 | 0 | 0% | 100% |
| 第2轮 | 6000 | 5000 | 83% | ~17% |
| 第3轮 | 8000 | 5760 | 72% | ~28% |

---

## 4. 存储表设计

### 4.1 agent_session_stats 表

```sql
CREATE TABLE agent_session_stats (
    id BIGSERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES "user"(id),
    session_id VARCHAR(64) NOT NULL,
    agent_type VARCHAR(20) NOT NULL,
    parrot_id VARCHAR(20),

    -- Token 统计
    prompt_tokens INTEGER DEFAULT 0,
    completion_tokens INTEGER DEFAULT 0,
    cache_read_tokens INTEGER DEFAULT 0,
    cache_write_tokens INTEGER DEFAULT 0,
    total_tokens INTEGER DEFAULT 0,

    -- 成本统计 (毫美分)
    prompt_cost BIGINT DEFAULT 0,
    completion_cost BIGINT DEFAULT 0,
    total_cost BIGINT DEFAULT 0,

    -- 性能指标
    latency_ms BIGINT DEFAULT 0,
    tool_calls INTEGER DEFAULT 0,
    thinking_time_ms BIGINT DEFAULT 0,

    -- 状态
    status VARCHAR(20) DEFAULT 'success',

    created_ts BIGINT NOT NULL,
    updated_ts BIGINT NOT NULL
);

-- 索引
CREATE INDEX idx_session_stats_user ON agent_session_stats(user_id);
CREATE INDEX idx_session_stats_session ON agent_session_stats(session_id);
CREATE INDEX idx_session_stats_created ON agent_session_stats(created_ts DESC);
```

### 4.2 数据映射

| Go 字段 | 数据库字段 | 类型 |
|:--------|:----------|:-----|
| `PromptTokens` | `prompt_tokens` | INTEGER |
| `CompletionTokens` | `completion_tokens` | INTEGER |
| `CacheReadTokens` | `cache_read_tokens` | INTEGER |
| `CacheWriteTokens` | `cache_write_tokens` | INTEGER |
| `TotalCostMilliCents` | `total_cost` | BIGINT |
| `TotalDurationMs` | `latency_ms` | BIGINT |
| `ToolCallCount` | `tool_calls` | INTEGER |

---

## 5. API 端点

### 5.1 获取会话统计

**请求**:
```http
GET /api/v1/ai/sessions/{session_id}
```

**响应**:
```json
{
  "id": 1,
  "session_id": "uuid-v5-123",
  "agent_type": "MEMO",
  "prompt_tokens": 1500,
  "completion_tokens": 500,
  "cache_read_tokens": 1000,
  "cache_write_tokens": 500,
  "total_tokens": 2000,
  "total_cost": 1234,
  "latency_ms": 2500,
  "tool_calls": 2,
  "tools_used": ["memo_search"],
  "status": "success",
  "created_at": 1707520800000
}
```

### 5.2 列出会话统计

**请求**:
```http
GET /api/v1/ai/sessions?limit=20&days=30
```

**响应**:
```json
{
  "sessions": [...],
  "total_count": 150,
  "total_cost_usd": 12.34
}
```

### 5.3 获取成本统计

**请求**:
```http
GET /api/v1/ai/cost-stats?days=7
```

**响应**:
```json
{
  "total_cost_usd": 5.67,
  "daily_average_usd": 0.81,
  "session_count": 42,
  "daily_breakdown": [
    {"date": "2026-02-01", "cost_usd": 1.20, "session_count": 8},
    {"date": "2026-02-02", "cost_usd": 0.95, "session_count": 7}
  ]
}
```

---

## 6. 实现参考

### 6.1 接口重构

**修改前** (有状态):
```go
type LLMService interface {
    Chat(ctx context.Context, messages []Message) (string, error)
    ChatStream(ctx context.Context, messages []Message) <-chan string
}
```

**修改后** (无状态):
```go
type LLMService interface {
    Chat(ctx context.Context, messages []Message) (string, *LLMCallStats, error)
    ChatStream(ctx context.Context, messages []Message) (<-chan string, <-chan *LLMCallStats, <-chan error)
}
```

### 6.2 Agent 聚合

```go
// ai/agent/base_parrot.go

func (p *BaseParrot) trackLLMCall(stats *ai.LLMCallStats) {
    p.lock.Lock()
    defer p.lock.Unlock()

    if p.accumulatedStats == nil {
        p.accumulatedStats = &ai.LLMCallStats{}
    }

    p.accumulatedStats.PromptTokens += stats.PromptTokens
    p.accumulatedStats.CompletionTokens += stats.CompletionTokens
    p.accumulatedStats.TotalTokens += stats.TotalTokens
    p.accumulatedStats.CacheReadTokens += stats.CacheReadTokens
    p.accumulatedStats.CacheWriteTokens += stats.CacheWriteTokens
    p.accumulatedStats.TotalDurationMs += stats.TotalDurationMs
}
```

### 6.3 关键代码路径

| 文件路径 | 职责 | 修改类型 |
|:---------|:-----|:---------|
| `ai/llm.go` | 重构接口，返回 `LLMCallStats` | 🔧 重构 |
| `ai/agent/base_parrot.go` | 实现统计聚合逻辑 | ➕ 新建 |
| `ai/agent/memo_parrot.go` | 适配新接口 | 🔧 修改 |
| `ai/agent/schedule_parrot_v2.go` | 适配新接口 | 🔧 修改 |

---

## 7. 实施计划

### 7.1 阶段划分

| 阶段 | 任务 | 投入 |
|:-----|:-----|:-----|
| **Phase 1** | 接口重构 | 1人天 |
| **Phase 2** | Agent 适配 | 1.5人天 |
| **Phase 3** | 测试验收 | 0.5人天 |

### 7.2 验收标准

- [ ] 普通模式显示 Token 使用量
- [ ] 普通模式显示工具调用次数
- [ ] 普通模式显示时间分解
- [ ] 成本估算准确
- [ ] 并发安全测试通过

---

## 8. 相关文档

| 文档 | 描述 |
|:-----|:-----|
| [Unified Block Model](./unified-block-model.md) | Block 数据模型 |
| [架构文档](../../dev-guides/ARCHITECTURE.md) | AI 系统架构 |
| [DeepSeek 上下文缓存](../../dev-guides/ARCHITECTURE.md#deepseek-上下文缓存) | 缓存优化说明 |

---

*维护者*: DivineSense 开发团队
*反馈渠道*: [GitHub Issues](https://github.com/hrygo/divinesense/issues/79)
