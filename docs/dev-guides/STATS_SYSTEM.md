# 统计与告警系统

> **实现状态**: ✅ 完成 (v0.97.0) | **位置**: `ai/stats/`

## 概述

统计与告警系统提供 AI 代理的实时指标收集、持久化存储和智能告警功能，帮助监控系统健康状况、成本使用和性能瓶颈。

### 核心功能

| 功能 | 描述 |
|:-----|:-----|
| **指标收集** | 实时收集 Token 使用、成本、延迟等指标 |
| **持久化** | 将统计数据持久化到数据库 |
| **告警检测** | 基于阈值的智能告警 |
| **成本追踪** | 用户级别的成本预算管理 |
| **性能分析** | A/B 测试支持 |

---

## 架构设计

```
┌─────────────────────────────────────────────────────────┐
│                    AI 请求                               │
└────────────────────┬────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────┐
│                  MetricsCollector                        │
│  收集: Token 使用、延迟、工具调用、成本                    │
└────────────────────┬────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────┐
│                   实时聚合                                │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  │
│  │ 会话级指标    │  │ 用户级指标    │  │ 系统级指标    │  │
│  └──────────────┘  └──────────────┘  └──────────────┘  │
└────────────────────┬────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────┐
│                   告警检测器                             │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  │
│  │ 成本告警      │  │ 性能告警     │  │ 错误率告警    │  │
│  └──────────────┘  └──────────────┘  └──────────────┘  │
└────────────────────┬────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────┐
│                   Persister                              │
│  批量写入数据库（agent_session_stats）                    │
└─────────────────────────────────────────────────────────┘
```

---

## 核心组件

### MetricsCollector (`collector.go`)

```go
type MetricsCollector struct {
    sessionCache *lru.Cache  // 会话级缓存
    userCache   *lru.Cache  // 用户级缓存
    systemStats SystemStats // 系统级统计
}

type SessionMetrics struct {
    SessionID       string
    UserID          int
    AgentType       string
    PromptTokens    int
    CompletionTokens int
    CacheReadTokens int
    CacheWriteTokens int
    TotalTokens     int
    PromptCost      int64  // 毫美分
    CompletionCost  int64
    TotalCost       int64
    LatencyMs       int64
    ToolCalls       int
    ThinkingTimeMs  int64
    Status          string
}

// Record 记录指标
func (mc *MetricsCollector) Record(ctx context.Context, metrics SessionMetrics) error

// Flush 刷新缓存到数据库
func (mc *MetricsCollector) Flush(ctx context.Context) error
```

### Alerting (`alerting.go`)

```go
type AlertType int
const (
    CostAlert AlertType = iota
    PerformanceAlert
    ErrorRateAlert
    UsageQuotaAlert
)

type Alert struct {
    ID        string
    Type      AlertType
    UserID    int
    Level     AlertLevel  // Warning, Critical
    Message   string
    Metrics   map[string]interface{}
    CreatedAt time.Time
}

type Alerting struct {
    rules       []AlertRule
    notifier    Notifier
    persister   *Persister
}

// Check 检查是否触发告警
func (a *Alerting) Check(ctx context.Context, metrics SessionMetrics) *Alert
```

### Persister (`persister.go`)

```go
type Persister struct {
    db     store.DB
    buffer chan SessionMetrics
    ticker *time.Ticker
}

// Start 启动持久化协程
func (p *Persister) Start(ctx context.Context)

// Persist 持久化单条记录
func (p *Persister) Persist(ctx context.Context, metrics SessionMetrics) error

// BatchPersist 批量持久化
func (p *Persister) BatchPersist(ctx context.Context, metrics []SessionMetrics) error
```

---

## 数据库表

### agent_session_stats

```sql
CREATE TABLE agent_session_stats (
  id                      BIGSERIAL PRIMARY KEY,
  user_id                 INTEGER NOT NULL REFERENCES "user"(id),
  session_id              VARCHAR(64) NOT NULL,
  agent_type              VARCHAR(20) NOT NULL,
  parrot_id               VARCHAR(20),

  -- Token 统计
  prompt_tokens           INTEGER DEFAULT 0,
  completion_tokens       INTEGER DEFAULT 0,
  cache_read_tokens       INTEGER DEFAULT 0,
  cache_write_tokens      INTEGER DEFAULT 0,
  total_tokens            INTEGER DEFAULT 0,

  -- 成本统计（毫美分）
  prompt_cost             BIGINT DEFAULT 0,
  completion_cost         BIGINT DEFAULT 0,
  total_cost              BIGINT DEFAULT 0,

  -- 性能指标
  latency_ms              BIGINT DEFAULT 0,
  tool_calls              INTEGER DEFAULT 0,
  thinking_time_ms        BIGINT DEFAULT 0,

  -- 状态
  status                  VARCHAR(20) DEFAULT 'success',

  created_ts              BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW()) * 1000
);

-- 索引
CREATE INDEX idx_agent_session_stats_user ON agent_session_stats(user_id);
CREATE INDEX idx_agent_session_stats_session ON agent_session_stats(session_id);
CREATE INDEX idx_agent_session_stats_created ON agent_session_stats(created_ts DESC);
```

### user_cost_settings

```sql
CREATE TABLE user_cost_settings (
  id              SERIAL PRIMARY KEY,
  user_id         INTEGER NOT NULL UNIQUE REFERENCES "user"(id),
  daily_budget    BIGINT DEFAULT 100000,  -- 每日预算（毫美分）
  alert_threshold REAL DEFAULT 0.8,       -- 告警阈值（百分比）
  cost_alerts_enabled BOOLEAN DEFAULT true,
  created_ts      BIGINT NOT NULL,
  updated_ts      BIGINT NOT NULL
);
```

---

## 告警规则

### 成本告警

| 条件 | 级别 | 动作 |
|:-----|:-----|:-----|
| 达到日预算的 80% | Warning | 通知用户 |
| 达到日预算的 100% | Critical | 暂停服务 |
| 单次请求成本 > $1 | Warning | 记录日志 |

### 性能告警

| 条件 | 级别 | 动作 |
|:-----|:-----|:-----|
| 延迟 > 5s | Warning | 记录日志 |
| 延迟 > 10s | Critical | 告警 |
| 错误率 > 10% | Warning | 记录日志 |
| 错误率 > 30% | Critical | 告警 |

---

## 使用方式

### 基本使用

```go
import "divinesense/ai/stats"

// 创建收集器
collector := stats.NewMetricsCollector(db)

// 记录指标
metrics := stats.SessionMetrics{
    SessionID:        "session-123",
    UserID:           123,
    AgentType:        "memo",
    PromptTokens:     1000,
    CompletionTokens: 500,
    TotalCost:       50,  // 0.5 美元 = 50 毫美分
    LatencyMs:       2000,
    ToolCalls:       2,
    Status:          "success",
}

if err := collector.Record(ctx, metrics); err != nil {
    log.Error("failed to record metrics", "error", err)
}

// 刷新到数据库
if err := collector.Flush(ctx); err != nil {
    log.Error("failed to flush metrics", "error", err)
}
```

### AI 代理集成

```go
func (p *MemoParrot) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
    startTime := time.Now()

    // 执行聊天
    response, err := p.doChat(ctx, req)

    // 收集指标
    metrics := stats.SessionMetrics{
        SessionID: req.SessionID,
        UserID:    req.UserID,
        AgentType: "memo",
        Status:    "success",
    }

    if llmStats, ok := response.Stats["llm"]; ok {
        metrics.PromptTokens = llmStats.PromptTokens
        metrics.CompletionTokens = llmStats.CompletionTokens
        metrics.TotalCost = llmStats.Cost
    }

    metrics.LatencyMs = time.Since(startTime).Milliseconds()

    // 记录指标（异步）
    go p.collector.Record(context.Background(), metrics)

    return response, err
}
```

### 成本预算管理

```go
// 检查用户预算
func (p *Parrot) checkBudget(ctx context.Context, userID int) error {
    settings, err := p.store.GetUserCostSettings(ctx, userID)
    if err != nil {
        return err
    }

    // 获取今日已使用成本
    used, err := p.store.GetDailyCost(ctx, userID, time.Now())
    if err != nil {
        return err
    }

    // 检查是否超预算
    if used >= settings.DailyBudget {
        return fmt.Errorf("已达到每日成本预算")
    }

    // 检查是否需要告警
    if used >= settings.DailyBudget*int64(settings.AlertThreshold) {
        p.alerting.Send(ctx, stats.Alert{
            Type:    stats.CostAlert,
            UserID:  userID,
            Level:   stats.Warning,
            Message: fmt.Sprintf("已使用 %d%% 的日预算", int(used*100/settings.DailyBudget)),
        })
    }

    return nil
}
```

---

## 配置选项

| 环境变量 | 默认值 | 说明 |
|:---------|:------|:-----|
| `DIVINESENSE_STATS_ENABLED` | `true` | 是否启用统计 |
| `DIVINESENSE_STATS_PERSIST_INTERVAL` | `30s` | 持久化间隔 |
| `DIVINESENSE_STATS_BUFFER_SIZE` | `1000` | 缓冲区大小 |
| `DIVINESENSE_ALERTS_ENABLED` | `true` | 是否启用告警 |
| `DIVINESENSE_DEFAULT_DAILY_BUDGET` | `100000` | 默认日预算（毫美分） |

---

## Prometheus 集成

### 指标导出

```go
var (
    tokensUsed = promauto.NewCounterVec(prometheus.CounterOpts{
        Name: "ai_tokens_used_total",
        Help: "Total tokens used",
    }, []string{"agent_type", "token_type"})

    requestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
        Name:    "ai_request_duration_ms",
        Help:    "Request duration in milliseconds",
        Buckets: prometheus.DefBuckets,
    }, []string{"agent_type"})

    costTotal = promauto.NewCounterVec(prometheus.CounterOpts{
        Name: "ai_cost_total_cents",
        Help: "Total cost in cents",
    }, []string{"agent_type"})
)
```

### 查询示例

```promql
# 每日成本
sum increase(ai_cost_total_cents[1d])

# 平均延迟
rate(ai_request_duration_ms_sum[5m]) / rate(ai_request_duration_ms_count[5m])

# Token 使用量（按代理类型）
sum by (agent_type) (ai_tokens_used_total{token_type="total"})
```

---

## 测试

```bash
# 运行统计系统测试
go test ./ai/stats/ -v

# 性能基准测试
go test ./ai/stats/ -bench=. -benchmem
```

---

## 相关文档

- [成本管理表](BACKEND_DB.md#user_cost_settings-结构v0930)
- [会话统计表](BACKEND_DB.md#agent_session_stats-结构v0930)
- [AI 性能追踪](ARCHITECTURE.md#ai-服务-ai)
