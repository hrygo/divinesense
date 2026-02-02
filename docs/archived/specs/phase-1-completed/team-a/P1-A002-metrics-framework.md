# P1-A002: 基础评估指标

> **状态**: ✅ 已完成  
> **优先级**: P0 (核心)  
> **投入**: 2 人天  
> **负责团队**: 团队 A  
> **Sprint**: Sprint 1

---

## 1. 目标与背景

### 1.1 核心目标

实现轻量级 Agent 指标采集框架，支持问题定位和优化决策。

### 1.2 用户价值

- 管理员可查看 AI 助理运行状态
- 问题可快速定位

### 1.3 技术价值

- 量化 Agent 表现
- 为优化提供数据依据
- 统一团队 B/C 的指标上报

---

## 2. 依赖关系

### 2.1 前置依赖

- [x] S0-interface-contract: 接口定义

### 2.2 并行依赖

- P1-A001: 记忆系统（可并行）

### 2.3 后续依赖

- P1-B001: 工具可靠性增强（上报工具调用指标）

---

## 3. 功能设计

### 3.1 架构图

```
┌─────────────────────────────────────────────────────────────────┐
│                      轻量指标框架                                 │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │                    指标采集层                             │   │
│  │                                                          │   │
│  │  Agent 执行 ──► RecordRequest(type, latency, success)   │   │
│  │  工具调用 ────► RecordToolCall(tool, latency, success)   │   │
│  │                                                          │   │
│  └──────────────────────────┬──────────────────────────────┘   │
│                             │                                   │
│                             ▼                                   │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │                    指标聚合层                             │   │
│  │                                                          │   │
│  │  • 计数器 (Counter)                                      │   │
│  │  • 延迟百分位 (Histogram)                                │   │
│  │  • 成功率 (Gauge)                                        │   │
│  │                                                          │   │
│  │  聚合粒度: 小时                                          │   │
│  └──────────────────────────┬──────────────────────────────┘   │
│                             │                                   │
│                             ▼                                   │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │                    持久化层                               │   │
│  │                                                          │   │
│  │  每小时写入 SQLite/PostgreSQL                            │   │
│  │  保留 30 天                                              │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### 3.2 核心流程

1. **采集**: Agent/工具执行时调用 Record 方法
2. **聚合**: 内存中按小时聚合
3. **持久化**: 每小时写入数据库
4. **查询**: 管理后台调用 GetStats 展示

### 3.3 关键决策

| 决策点 | 方案 A | 方案 B | 选择 | 理由 |
|:---|:---|:---|:---:|:---|
| 存储方式 | Prometheus | 嵌入式 | B | 私人部署，无需外部依赖 |
| 聚合粒度 | 分钟 | 小时 | B | 减少存储开销 |

---

## 4. 技术实现

### 4.1 接口定义

见 [S0-interface-contract](../sprint-0/S0-interface-contract.md)

### 4.2 数据模型

```sql
-- store/db/postgres/migration/xxx_add_agent_metrics.sql

CREATE TABLE agent_metrics (
    id SERIAL PRIMARY KEY,
    hour_bucket TIMESTAMP NOT NULL,
    agent_type VARCHAR(20) NOT NULL,
    request_count INTEGER DEFAULT 0,
    success_count INTEGER DEFAULT 0,
    latency_sum_ms BIGINT DEFAULT 0,
    latency_p50_ms INTEGER,
    latency_p95_ms INTEGER,
    errors JSONB DEFAULT '{}',
    
    UNIQUE(hour_bucket, agent_type)
);

CREATE TABLE tool_metrics (
    id SERIAL PRIMARY KEY,
    hour_bucket TIMESTAMP NOT NULL,
    tool_name VARCHAR(50) NOT NULL,
    call_count INTEGER DEFAULT 0,
    success_count INTEGER DEFAULT 0,
    latency_sum_ms BIGINT DEFAULT 0,
    
    UNIQUE(hour_bucket, tool_name)
);
```

### 4.3 关键代码路径

| 文件路径 | 职责 |
|:---|:---|
| `plugin/ai/metrics/service.go` | 指标服务实现 |
| `plugin/ai/metrics/aggregator.go` | 内存聚合器 |
| `plugin/ai/metrics/persister.go` | 持久化任务 |
| `store/agent_metrics.go` | 数据访问层 |

---

## 5. 交付物清单

### 5.1 代码文件

- [ ] `plugin/ai/metrics/service.go` - 指标服务主实现
- [ ] `plugin/ai/metrics/aggregator.go` - 内存聚合器
- [ ] `plugin/ai/metrics/persister.go` - 持久化后台任务
- [ ] `store/agent_metrics.go` - 指标 Store

### 5.2 数据库变更

- [ ] `store/db/postgres/migration/xxx_add_agent_metrics.sql`

### 5.3 测试文件

- [ ] `plugin/ai/metrics/service_test.go`

---

## 6. 测试验收

### 6.1 功能测试

| 场景 | 输入 | 预期输出 |
|:---|:---|:---|
| 记录请求 | agentType + latency + success | 计数器增加 |
| 记录工具调用 | toolName + latency + success | 计数器增加 |
| 获取统计 | timeRange | 返回聚合数据 |

### 6.2 性能验收

| 指标 | 目标值 | 测试方法 |
|:---|:---|:---|
| Record 延迟 | < 1ms | 单元测试 |
| GetStats 延迟 | < 100ms | 集成测试 |

---

## 7. ROI 分析

| 维度 | 值 |
|:---|:---|
| 开发投入 | 2 人天 |
| 预期收益 | 问题可定位，优化有据可依 |
| 风险评估 | 低 |
| 回报周期 | 立即 |

---

## 8. 实施计划

### 8.1 时间表

| 阶段 | 时间 | 任务 |
|:---|:---|:---|
| Day 1 | 1人天 | 聚合器 + 服务实现 |
| Day 2 | 1人天 | 持久化 + 测试 |

### 8.2 检查点

- [ ] Day 1: Record 方法可用
- [ ] Day 2: GetStats 返回正确数据

---

## 附录

### A. 参考资料

- [智能助理路线图 - 评估指标](../../research/assistant-roadmap.md)

### B. 变更记录

| 日期 | 版本 | 变更内容 | 作者 |
|:---|:---|:---|:---|
| 2026-01-27 | v1.0 | 初始版本 | - |
