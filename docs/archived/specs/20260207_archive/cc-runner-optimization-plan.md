# CC Runner 系统优化规划

**规划日期**: 2026-02-03
**版本**: 1.0
**基于调研**: [CCRunner 消息处理机制调研](../research/cc-runner-message-handling-research.md)
**当前状态**: ✅ 基础功能已实现，测试覆盖完整

---

## 1. 现状分析

### 1.1 已完成 ✅

| 功能 | 状态 | 说明 |
|:-----|:-----|:-----|
| **消息类型处理** | ✅ 完成 | 11 种 CLI 消息类型全部正确处理 |
| **系统消息** | ✅ 完成 | system 消息静默处理，无日志噪音 |
| **结果消息** | ✅ 完成 | result 消息统计提取并发送 session_stats 事件 |
| **成本追踪** | ✅ 完成 | TotalCostUSD 通过 SessionSummary 发送到前端 |
| **测试覆盖** | ✅ 完成 | 69 个测试用例，全部通过 |
| **会话隔离** | ✅ 完成 | 基于 UUID v5 的确定性映射，进程级隔离 |

### 1.2 待优化 ⚠️

| 领域 | 当前状态 | 问题/机会 |
|:-----|:--------|:----------|
| **数据持久化** | ❌ 未实现 | 会话统计数据仅存在于内存，服务重启丢失 |
| **前端展示** | ⚠️ 部分 | 成本数据已到达但未展示 |
| **成本告警** | ❌ 未实现 | 无单会话成本超阈值提醒 |
| **历史查询** | ❌ 未实现 | 无法查询历史会话统计 |
| **性能监控** | ⚠️ 基础 | 仅有基础日志，无聚合指标 |
| **安全审计** | ❌ 未实现 | 危险操作拦截未记录 |

---

## 2. 优化目标

### 2.1 核心目标

1. **数据持久化**：会话统计数据存储到数据库，支持历史查询
2. **成本可见性**：前端展示实时成本和累计统计
3. **成本控制**：超阈值告警，用户预算管理
4. **性能洞察**：会话性能分析，优化建议
5. **安全审计**：危险操作日志，合规性追踪

### 2.2 非功能性目标

- **性能**：统计提取开销 < 1ms，数据库写入异步化
- **可靠性**：数据不丢失，服务重启不影响统计
- **可维护性**：清晰的模块边界，易于扩展
- **安全性**：敏感数据脱敏，访问控制

---

## 3. 分阶段实施计划

### 阶段一：数据持久化（高优先级）

#### 3.1.1 数据库表设计

```sql
-- 会话统计主表
CREATE TABLE agent_session_stats (
    id                      BIGSERIAL PRIMARY KEY,
    session_id              VARCHAR(64)  NOT NULL UNIQUE,
    conversation_id        BIGINT       NOT NULL,
    user_id                 INTEGER     NOT NULL,
    agent_type              VARCHAR(20)  NOT NULL, -- 'geek', 'evolution'

    -- 时间维度
    started_at              TIMESTAMPTZ NOT NULL,
    ended_at                TIMESTAMPTZ NOT NULL,
    total_duration_ms       BIGINT       NOT NULL,
    thinking_duration_ms    BIGINT       NOT NULL DEFAULT 0,
    tool_duration_ms        BIGINT       NOT NULL DEFAULT 0,
    generation_duration_ms   BIGINT       NOT NULL DEFAULT 0,

    -- Token 使用
    input_tokens            INTEGER     NOT NULL DEFAULT 0,
    output_tokens           INTEGER     NOT NULL DEFAULT 0,
    cache_write_tokens      INTEGER     NOT NULL DEFAULT 0,
    cache_read_tokens       INTEGER     NOT NULL DEFAULT 0,
    total_tokens            INTEGER     NOT NULL DEFAULT 0,

    -- 成本
    total_cost_usd          NUMERIC(10,4) NOT NULL DEFAULT 0,

    -- 工具使用
    tool_call_count         INTEGER     NOT NULL DEFAULT 0,
    tools_used               JSONB,               -- ["Bash", "editor_write", ...]

    -- 文件操作
    files_modified          INTEGER     NOT NULL DEFAULT 0,
    file_paths              TEXT[],             -- ["path1", "path2", ...]

    -- 模型信息
    model_used               VARCHAR(100),

    -- 状态
    is_error                BOOLEAN     NOT NULL DEFAULT FALSE,
    error_message           TEXT,

    -- 时间戳
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_session_stats_user FOREIGN KEY (user_id) REFERENCES "user"(id) ON DELETE CASCADE,
    CONSTRAINT fk_session_stats_conv FOREIGN KEY (conversation_id) REFERENCES conversation(id) ON DELETE CASCADE
);

-- 索引
CREATE INDEX idx_session_stats_user_date ON agent_session_stats(user_id, started_at DESC);
CREATE INDEX idx_session_stats_conv ON agent_session_stats(conversation_id);
CREATE INDEX idx_session_stats_agent ON agent_session_stats(agent_type, started_at DESC);
CREATE INDEX idx_session_stats_cost ON agent_session_stats(total_cost_usd) WHERE total_cost_usd > 0;

-- 成本告警配置表
CREATE TABLE user_cost_settings (
    id                      BIGSERIAL PRIMARY KEY,
    user_id                 INTEGER     NOT NULL UNIQUE,

    -- 告警阈值
    daily_budget_usd         NUMERIC(10,4),
    per_session_threshold_usd NUMERIC(10,4) DEFAULT 5.0,

    -- 通知设置
    alert_enabled           BOOLEAN     DEFAULT TRUE,
    alert_email             BOOLEAN     DEFAULT FALSE,
    alert_in_app            BOOLEAN     DEFAULT TRUE,

    -- 时间范围重置
    budget_reset_at         DATE,

    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_cost_settings_user FOREIGN KEY (user_id) REFERENCES "user"(id) ON DELETE CASCADE
);

-- 安全审计日志表
CREATE TABLE agent_security_audit (
    id                      BIGSERIAL PRIMARY KEY,
    session_id              VARCHAR(64),

    -- 用户信息
    user_id                 INTEGER     NOT NULL,
    agent_type              VARCHAR(20) NOT NULL,

    -- 操作信息
    operation_type          VARCHAR(50) NOT NULL, -- 'danger_block', 'tool_use', etc.
    operation_name          VARCHAR(100),        -- 'rm -rf', 'format', etc.

    -- 危险等级
    risk_level              VARCHAR(20) NOT NULL, -- 'low', 'medium', 'high', 'critical'

    -- 命令详情
    command_input           TEXT,
    command_matched_pattern  TEXT,

    -- 拦截结果
    action_taken            VARCHAR(50),         -- 'blocked', 'allowed', 'logged_only'
    reason                  TEXT,

    -- 额外上下文
    file_path               TEXT,
    tool_id                 VARCHAR(100),

    -- 时间戳
    occurred_at             TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_audit_user FOREIGN KEY (user_id) REFERENCES "user"(id) ON DELETE CASCADE
);

CREATE INDEX idx_audit_user_date ON agent_security_audit(user_id, occurred_at DESC);
CREATE INDEX idx_audit_risk ON agent_security_audit(risk_level, occurred_at DESC);
CREATE INDEX idx_audit_operation ON agent_security_audit(operation_type, occurred_at DESC);
```

#### 3.1.2 数据访问层

**文件**: `store/agent_stats.go`

```go
package store

import (
    "context"
    "time"
)

// AgentSessionStats represents a session statistics record.
type AgentSessionStats struct {
    SessionID            string
    ConversationID       int64
    UserID                int32
    AgentType             string
    StartedAt             time.Time
    EndedAt               time.Time
    TotalDurationMs       int64
    ThinkingDurationMs    int64
    ToolDurationMs        int64
    GenerationDurationMs   int64
    InputTokens           int32
    OutputTokens          int32
    CacheWriteTokens      int32
    CacheReadTokens       int32
    TotalTokens           int32
    TotalCostUSD          float64
    ToolCallCount         int32
    ToolsUsed             []string
    FilesModified         int32
    FilePaths             []string
    ModelUsed             string
    IsError               bool
    ErrorMessage          string
}

// AgentStatsStore defines the interface for session statistics persistence.
type AgentStatsStore interface {
    // SaveSessionStats saves a session statistics record.
    SaveSessionStats(ctx context.Context, stats *AgentSessionStats) error

    // GetSessionStats retrieves stats by session ID.
    GetSessionStats(ctx context.Context, sessionID string) (*AgentSessionStats, error)

    // ListSessionStats retrieves stats for a user with pagination.
    ListSessionStats(ctx context.Context, userID int32, limit, offset int) ([]*AgentSessionStats, int64, error)

    // GetDailyCostUsage retrieves total cost for a user in a date range.
    GetDailyCostUsage(ctx context.Context, userID int32, startDate, endDate time.Time) (float64, error)

    // GetCostStats retrieves aggregated cost statistics.
    GetCostStats(ctx context.Context, userID int32, days int) (*CostStats, error)
}

// CostStats represents aggregated cost statistics.
type CostStats struct {
    TotalCostUSD      float64
    DailyAverageUSD   float64
    SessionCount      int64
    MostExpensiveSession *AgentSessionStats
}
```

#### 3.1.3 持久化服务

**文件**: `ai/stats/persister.go`

```go
package stats

import (
    "context"
    "log/slog"
    "sync"
)

// Persister handles async persistence of session statistics.
type Persister struct {
    store   AgentStatsStore
    queue   chan *AgentSessionStats
    wg      sync.WaitGroup
    logger  *slog.Logger
}

// NewPersister creates a new async persister.
func NewPersister(store AgentStatsStore, queueSize int, logger *slog.Logger) *Persister {
    p := &Persister{
        store:  store,
        queue:  make(chan *AgentSessionStats, queueSize),
        logger: logger,
    }
    p.wg.Add(1)
    go p.processQueue()
    return p
}

// Enqueue queues a stats record for persistence.
func (p *Persister) Enqueue(stats *AgentSessionStats) bool {
    select {
    case p.queue <- stats:
        return true
    default:
        p.logger.Warn("Persister queue full, dropping stats record",
            "session_id", stats.SessionID)
        return false
    }
}

// processQueue processes stats records in the background.
func (p *Persister) processQueue() {
    defer p.wg.Done()

    for stats := range p.queue {
        if err := p.store.SaveSessionStats(context.Background(), stats); err != nil {
            p.logger.Error("Failed to save session stats",
                "session_id", stats.SessionID,
                "error", err)
        } else {
            p.logger.Debug("Saved session stats",
                "session_id", stats.SessionID,
                "cost_usd", stats.TotalCostUSD)
        }
    }
}

// Close waits for the queue to drain and shuts down the persister.
func (p *Persister) Close(timeout time.Duration) error {
    close(p.queue)
    done := make(chan struct{})
    go func() {
        p.wg.Wait()
        close(done)
    }()
    select {
    case <-done:
        return nil
    case <-time.After(timeout):
        return fmt.Errorf("persister shutdown timeout")
    }
}
```

### 阶段二：前端展示（中优先级）

#### 3.2.1 会话统计组件

**文件**: `web/src/components/SessionStatsPanel.tsx`

```tsx
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { useTranslation } from "react-i18n";

interface SessionStatsData {
  total_duration_ms: number;
  thinking_duration_ms: number;
  tool_duration_ms: number;
  generation_duration_ms: number;
  input_tokens: number;
  output_tokens: number;
  total_cost_usd: number;
  tool_call_count: number;
  tools_used: string[];
}

interface SessionStatsPanelProps {
  stats: SessionStatsData;
}

export function SessionStatsPanel({ stats }: SessionStatsPanelProps) {
  const { t } = useTranslation();

  const formatDuration = (ms: number) => {
    if (ms < 1000) return `${ms}ms`;
    const seconds = Math.floor(ms / 1000);
    if (seconds < 60) return `${seconds}s`;
    const minutes = Math.floor(seconds / 60);
    return `${minutes}m ${seconds % 60}s`;
  };

  const formatCost = (cost: number) => {
    return `$${cost.toFixed(4)}`;
  };

  const formatTokens = (tokens: number) => {
    if (tokens >= 1000000) return `${(tokens / 1000000).toFixed(1)}M`;
    if (tokens >= 1000) return `${(tokens / 1000).toFixed(1)}K`;
    return tokens.toString();
  };

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-sm font-medium">
          {t("chat.session_stats")}
        </CardTitle>
      </CardHeader>
      <CardContent className="space-y-4">
        {/* Cost */}
        <div className="flex items-center justify-between">
          <span className="text-sm text-muted-foreground">
            {t("chat.total_cost")}
          </span>
          <Badge variant={stats.total_cost_usd > 1.0 ? "destructive" : "secondary"}>
            {formatCost(stats.total_cost_usd)}
          </Badge>
        </div>

        {/* Tokens */}
        <div className="space-y-2">
          <div className="flex items-center justify-between text-sm">
            <span className="text-muted-foreground">
              {t("chat.input_tokens")}
            </span>
            <span className="font-mono">{formatTokens(stats.input_tokens)}</span>
          </div>
          <div className="flex items-center justify-between text-sm">
            <span className="text-muted-foreground">
              {t("chat.output_tokens")}
            </span>
            <span className="font-mono">{formatTokens(stats.output_tokens)}</span>
          </div>
          <div className="flex items-center justify-between text-sm">
            <span className="text-muted-foreground">
              {t("chat.total_tokens")}
            </span>
            <span className="font-mono">{formatTokens(stats.input_tokens + stats.output_tokens)}</span>
          </div>
        </div>

        {/* Duration Breakdown */}
        <div className="space-y-2">
          <div className="flex items-center justify-between text-sm">
            <span className="text-muted-foreground">{t("chat.thinking_time")}</span>
            <span>{formatDuration(stats.thinking_duration_ms)}</span>
          </div>
          <div className="flex items-center justify-between text-sm">
            <span className="text-muted-foreground">{t("chat.tool_time")}</span>
            <span>{formatDuration(stats.tool_duration_ms)}</span>
          </div>
          <div className="flex items-center justify-between text-sm">
            <span className="text-muted-foreground">{t("chat.generation_time")}</span>
            <span>{formatDuration(stats.generation_duration_ms)}</span>
          </div>
        </div>

        {/* Tools */}
        {stats.tool_call_count > 0 && (
          <div>
            <div className="text-sm text-muted-foreground mb-2">
              {t("chat.tools_used")} ({stats.tool_call_count})
            </div>
            <div className="flex flex-wrap gap-1">
              {stats.tools_used.map((tool) => (
                <Badge key={tool} variant="outline" className="text-xs">
                  {tool}
                </Badge>
              ))}
            </div>
          </div>
        )}
      </CardContent>
    </Card>
  );
}
```

#### 3.2.2 成本趋势组件

**文件**: `web/src/components/CostTrendChart.tsx`

```tsx
import { useEffect, useState } from "react";
import { Line, LineChart } from "recharts";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { api } from "@/hooks/grpc";
import { useTranslation } from "react-i18n";

interface CostDataPoint {
  date: string;
  cost: number;
  sessions: number;
}

export function CostTrendChart({ days = 7 }: { days?: number }) {
  const { t } = useTranslation();
  const [data, setData] = useState<CostDataPoint[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const fetchData = async () => {
      try {
        const response = await api.ai.getCostStats({ days });
        setData(response.data || []);
      } catch (error) {
        console.error("Failed to fetch cost stats:", error);
      } finally {
        setLoading(false);
      }
    };
    fetchData();
  }, [days]);

  if (loading) {
    return <div>{t("common.loading")}</div>;
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>{t("chat.cost_trend", { days })}</CardTitle>
      </CardHeader>
      <CardContent>
        {data.length > 0 ? (
          <LineChart width={300} height={200} data={data}>
            <Line
              type="monotone"
              dataKey="cost"
              stroke="var(--primary)"
              strokeWidth={2}
              dot={false}
            />
          </LineChart>
        ) : (
          <p className="text-sm text-muted-foreground text-center py-8">
            {t("chat.no_cost_data")}
          </p>
        )}
      </CardContent>
    </Card>
  );
}
```

### 阶段三：成本控制（高优先级）

#### 3.3.1 成本告警服务

**文件**: `ai/stats/alerting.go`

```go
package stats

import (
    "context"
    "log/slog"
    "time"
)

type CostAlertService struct {
    store      AgentStatsStore
    notifier  NotificationService
    threshold  float64 // USD
    logger     *slog.Logger
}

func (s *CostAlertService) CheckSessionCost(ctx context.Context, stats *SessionStatsData) error {
    // Get daily usage
    today := time.Now().Truncate(24 * time.Hour)
    tomorrow := today.Add(24 * time.Hour)
    dailyCost, err := s.store.GetDailyCostUsage(ctx, stats.UserID, today, tomorrow)
    if err != nil {
        return err
    }

    // Check session threshold
    if stats.TotalCostUSD > s.threshold {
        s.notifier.SendCostAlert(stats.UserID, &CostAlert{
            Type:           "session_threshold_exceeded",
            SessionID:      stats.SessionID,
            CostUSD:        stats.TotalCostUSD,
            ThresholdUSD:   s.threshold,
            Timestamp:      time.Now(),
        })
    }

    // Check daily budget
    userSettings, err := s.store.GetUserCostSettings(ctx, stats.UserID)
    if err == nil && userSettings.DailyBudgetUsd > 0 {
        remainingBudget := userSettings.DailyBudgetUsd - dailyCost - stats.TotalCostUSD
        if remainingBudget < 0 {
            s.notifier.SendCostAlert(stats.UserID, &CostAlert{
                Type:           "daily_budget_exceeded",
                DailyCostUSD:   dailyCost + stats.TotalCostUSD,
                BudgetUSD:      userSettings.DailyBudgetUsd,
                OverByUSD:       -remainingBudget,
                Timestamp:      time.Now(),
            })
        }
    }

    return nil
}

type CostAlert struct {
    Type           string  // "session_threshold_exceeded", "daily_budget_exceeded"
    SessionID      string  // For session-specific alerts
    CostUSD        float64
    ThresholdUSD   float64
    DailyCostUSD   float64
    BudgetUSD      float64
    OverByUSD      float64
    Timestamp      time.Time
}
```

#### 3.3.2 用户设置 API

**Proto**: `proto/api/v1/ai_service.proto`

```protobuf
service AIService {
  // ... existing methods ...

  // Cost management
  rpc GetCostStats(GetCostStatsRequest) returns (CostStatsResponse);
  rpc SetCostAlert(SetCostAlertRequest) returns (SetCostAlertResponse);
  rpc GetUserCostSettings(GetUserCostSettingsRequest) returns (UserCostSettingsResponse);
}

message GetCostStatsRequest {
  int32 days = 1;
}

message CostStatsResponse {
  double total_cost_usd = 1;
  double daily_average_usd = 2;
  int64 session_count = 3;
  repeated DailyCostData daily_breakdown = 4;
}

message DailyCostData {
  string date = 1;  // YYYY-MM-DD
  double cost_usd = 2;
  int64 session_count = 3;
}

message SetCostAlertRequest {
  double daily_budget_usd = 1;
  double per_session_threshold_usd = 2;
  bool alert_enabled = 3;
}
```

### 阶段四：性能监控（中优先级）

#### 3.4.1 性能指标收集

**文件**: `ai/agent/metrics_collector.go`

```go
package agent

import (
    "time"
    "sync/atomic"
)

// MetricsCollector collects performance metrics.
type MetricsCollector struct {
    // Request metrics
    TotalRequests      atomic.Int64
    ActiveRequests     atomic.Int64
    FailedRequests     atomic.Int64

    // Latency metrics (in milliseconds)
    TotalLatencyMs     atomic.Int64
    MinLatencyMs       atomic.Int64
    MaxLatencyMs       atomic.Int64

    // Token metrics
    TotalInputTokens   atomic.Int64
    TotalOutputTokens  atomic.Int64

    // Cost metrics
    TotalCostUSD       atomic.Float64

    // Tool metrics
    TotalToolCalls     atomic.Int64
    ToolErrors         atomic.Int64

    startTime time.Time
}

func (m *MetricsCollector) RecordRequest(latencyMs int64, inputTokens, outputTokens int32, costUSD float64) {
    m.TotalRequests.Add(1)
    m.ActiveRequests.Add(1)
    m.TotalLatencyMs.Add(latencyMs)
    m.TotalInputTokens.Add(int64(inputTokens))
    m.TotalOutputTokens.Add(int64(outputTokens))
    m.TotalCostUSD.Add(costUSD)

    // Update min/max latency
    for {
        current := m.MinLatencyMs.Load()
        if latencyMs >= current || current == 0 {
            if m.MinLatencyMs.CompareAndSwap(current, int64(latencyMs)) {
                break
            }
        }
    }
    for {
        current := m.MaxLatencyMs.Load()
        if latencyMs <= current {
            if m.MaxLatencyMs.CompareAndSwap(current, int64(latencyMs)) {
                break
            }
        }
    }
}

func (m *MetricsCollector) EndRequest(latencyMs int64, success bool) {
    m.ActiveRequests.Add(-1)
    if !success {
        m.FailedRequests.Add(1)
    }
}

func (m *MetricsCollector) GetMetrics() *MetricsSnapshot {
    uptime := time.Since(m.startTime).Seconds()

    return &MetricsSnapshot{
        UptimeSeconds:       uptime,
        TotalRequests:       m.TotalRequests.Load(),
        ActiveRequests:      m.ActiveRequests.Load(),
        FailedRequests:      m.FailedRequests.Load(),
        AvgLatencyMs:        m.TotalLatencyMs.Load() / m.TotalRequests.Load(),
        MinLatencyMs:        m.MinLatencyMs.Load(),
        MaxLatencyMs:        m.MaxLatencyMs.Load(),
        TotalInputTokens:    m.TotalInputTokens.Load(),
        TotalOutputTokens:   m.TotalOutputTokens.Load(),
        TotalCostUSD:        m.TotalCostUSD.Load(),
        TotalToolCalls:      m.TotalToolCalls.Load(),
        ToolErrors:          m.ToolErrors.Load(),
        RequestErrorRate:    float64(m.FailedRequests.Load()) / float64(m.TotalRequests.Load()),
    }
}
```

### 阶段五：安全审计（低优先级）

#### 3.5.1 审计日志服务

**文件**: `ai/security/auditor.go`

```go
package security

import (
    "context"
    "log/slog"
    "time"
)

type SecurityEvent struct {
    SessionID    string
    UserID      int32
    AgentType   string
    Timestamp   time.Time

    EventType   string  // "danger_block", "tool_use", "file_operation"
    RiskLevel   string  // "low", "medium", "high", "critical"

    // Operation details
    OperationType   string
    OperationName   string
    CommandInput    string
    MatchedPattern  string

    // Action taken
    ActionTaken  string  // "blocked", "allowed", "logged_only"
    Reason        string

    // Additional context
    FilePath    string
    ToolID      string
}

type Auditor struct {
    store  SecurityAuditStore
    logger *slog.Logger
}

func (a *Auditor) LogSecurityEvent(ctx context.Context, event *SecurityEvent) error {
    // Always log security events
    a.logger.Info("Security event",
        "user_id", event.UserID,
        "agent_type", event.AgentType,
        "event_type", event.EventType,
        "risk_level", event.RiskLevel,
        "operation", event.OperationName,
        "action", event.ActionTaken,
    )

    // Persist to database
    return a.store.SaveSecurityEvent(ctx, event)
}
```

---

## 4. API 设计

### 4.1 新增 RPC 方法

| 方法 | 请求 | 响应 | 描述 |
|:-----|:-----|:-----|:-----|
| `GetCostStats` | `GetCostStatsRequest` | `CostStatsResponse` | 获取成本统计 |
| `SetCostAlert` | `SetCostAlertRequest` | `SetCostAlertResponse` | 设置成本告警 |
| `GetUserCostSettings` | `GetUserCostSettingsRequest` | `UserCostSettingsResponse` | 获取用户成本设置 |
| `ListSessionStats` | `ListSessionStatsRequest` | `ListSessionStatsResponse` | 列出会话统计 |
| `GetSessionStats` | `GetSessionStatsRequest` | `GetSessionStatsResponse` | 获取单会话统计 |
| `ListSecurityEvents` | `ListSecurityEventsRequest` | `ListSecurityEventsResponse` | 列出安全事件 |

### 4.2 WebSocket 事件

| 事件类型 | 数据结构 | 触发时机 |
|:--------|:--------|:---------|
| `session_stats` | `SessionStatsData` | 会话完成时 |
| `cost_alert` | `CostAlert` | 成本超阈值时 |
| `daily_budget_warning` | `DailyBudgetWarning` | 每日预算接近上限时 |

---

## 5. 优先级排序

| 阶段 | 优先级 | 预估工作量 | 依赖 |
|:-----|:------|:----------|:-----|
| **阶段一：数据持久化** | P0 | 2-3 天 | 数据库迁移 |
| **阶段二：前端展示** | P1 | 1-2 天 | 阶段一完成 |
| **阶段三：成本控制** | P0 | 1-2 天 | 阶段一完成 |
| **阶段四：性能监控** | P2 | 1 天 | 无 |
| **阶段五：安全审计** | P3 | 2 天 | 阶段一完成 |

---

## 6. 风险与缓解

| 风险 | 影响 | 缓解措施 |
|:-----|:-----|:---------|
| 数据库性能 | 写入延迟可能影响用户体验 | 异步持久化，队列缓冲 |
| 数据隐私 | 成本数据敏感 | 加密存储，访问控制 |
| 存储成本 | 历史数据持续增长 | 定期清理，数据归档 |
| 告警疲劳 | 过多告警导致用户忽略 | 智能去重，重要性分级 |

---

## 7. 成功指标

| 指标 | 目标 | 测量方式 |
|:-----|:-----|:--------|
| 数据持久化成功率 | >99.9% | 监控写入失败率 |
| 告警及时性 | <5秒 | 端到端延迟监控 |
| 前端加载性能 | <200ms | 页面加载时间 |
| 查询响应时间 | <500ms | API 延迟监控 |

---

## 8. 后续迭代方向

1. **成本优化建议**：基于历史数据给出节省成本的建议
2. **会话质量评分**：AI 回答质量与成本的关系分析
3. **A/B 测试框架**：不同提示词的效果对比
4. **多维度报表**：按时间、代理类型、工具类型的聚合分析
5. **导出功能**：CSV/Excel 导出统计数据

---

**相关文档**:
- [CCRunner 异步架构说明书](../specs/cc_runner_async_arch.md)
- [CCRunner 消息处理机制调研](../research/cc-runner-message-handling-research.md)
