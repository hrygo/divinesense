# P1-A001: 轻量记忆系统

> **状态**: ✅ 已完成  
> **优先级**: P0 (核心)  
> **投入**: 3 人天  
> **负责团队**: 团队 A  
> **Sprint**: Sprint 1

---

## 1. 目标与背景

### 1.1 核心目标

实现两层记忆架构（短期+长期），支持跨会话记忆和用户偏好持久化。

### 1.2 用户价值

- 跨会话记忆：减少重复询问 50%+
- 个性化体验："懂你"的私人助理

### 1.3 技术价值

- 统一记忆服务供团队 B/C 调用
- 为习惯学习 (P2-B001) 奠定基础

---

## 2. 依赖关系

### 2.1 前置依赖

- [x] S0-interface-contract: 接口定义

### 2.2 并行依赖

- P1-A002: 指标框架（可并行）

### 2.3 后续依赖

- P1-B001: 工具可靠性增强
- P2-B001: 用户习惯学习
- P2-A002: 上下文增强构建器

---

## 3. 功能设计

### 3.1 架构图

```
┌─────────────────────────────────────────────────────────────────┐
│                    轻量两层记忆架构                               │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌───────────────────────────────────────────────────────────┐ │
│  │                   短期记忆 (会话内)                         │ │
│  │  • 滑动窗口上下文 (最近 10 轮)                              │ │
│  │  • 实现: 内存数组                                          │ │
│  │  • 开销: ~10KB/会话                                        │ │
│  └───────────────────────────────────────────────────────────┘ │
│                            │                                    │
│                            ▼                                    │
│  ┌───────────────────────────────────────────────────────────┐ │
│  │                   长期记忆 (跨会话)                         │ │
│  │                                                            │ │
│  │  ┌─────────────────────┐  ┌─────────────────────────────┐ │ │
│  │  │  情景记忆 (Episodic) │  │  用户偏好 (Preferences)     │ │ │
│  │  │  • 重要对话摘要      │  │  • 时区/语言                 │ │ │
│  │  │  • 任务执行历史      │  │  • 默认日程时长              │ │ │
│  │  │  • 常见问题模式      │  │  • 常用关键词                │ │ │
│  │  │  实现: PostgreSQL    │  │  实现: JSONB                 │ │ │
│  │  └─────────────────────┘  └─────────────────────────────┘ │ │
│  │                                                            │ │
│  │  存储开销: ~1MB (10万条记录)                               │ │
│  └───────────────────────────────────────────────────────────┘ │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### 3.2 核心流程

1. **短期记忆**: 会话消息存入内存滑动窗口
2. **长期记忆**: 重要交互摘要写入 PostgreSQL
3. **偏好管理**: 用户偏好以 JSONB 存储

### 3.3 关键决策

| 决策点 | 方案 A | 方案 B | 选择 | 理由 |
|:---|:---|:---|:---:|:---|
| 短期存储 | 内存 | Redis | A | 私人部署，无需 Redis |
| 长期存储 | SQLite | PostgreSQL | B | 已有 PostgreSQL，复用 |
| 偏好格式 | 独立列 | JSONB | B | 灵活扩展 |

---

## 4. 技术实现

### 4.1 接口定义

见 [S0-interface-contract](../sprint-0/S0-interface-contract.md)

### 4.2 数据模型

```sql
-- store/db/postgres/migration/xxx_add_episodic_memory.sql

CREATE TABLE episodic_memory (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES "user"(id),
    timestamp TIMESTAMP NOT NULL DEFAULT NOW(),
    agent_type VARCHAR(20) NOT NULL,
    user_input TEXT NOT NULL,
    outcome VARCHAR(20) NOT NULL DEFAULT 'success',
    summary TEXT,
    importance REAL DEFAULT 0.5,
    created_at TIMESTAMP DEFAULT NOW(),
    
    INDEX idx_episodic_user_time (user_id, timestamp DESC),
    INDEX idx_episodic_agent (agent_type)
);

CREATE TABLE user_preferences (
    user_id INTEGER PRIMARY KEY REFERENCES "user"(id),
    preferences JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);
```

### 4.3 关键代码路径

| 文件路径 | 职责 |
|:---|:---|
| `plugin/ai/memory/service.go` | 记忆服务实现 |
| `plugin/ai/memory/short_term.go` | 短期记忆（内存） |
| `plugin/ai/memory/long_term.go` | 长期记忆（DB） |
| `store/episodic_memory.go` | 数据访问层 |
| `store/user_preferences.go` | 偏好数据访问层 |

---

## 5. 交付物清单

### 5.1 代码文件

- [ ] `plugin/ai/memory/service.go` - 记忆服务主实现
- [ ] `plugin/ai/memory/short_term.go` - 短期记忆实现
- [ ] `plugin/ai/memory/long_term.go` - 长期记忆实现
- [ ] `store/episodic_memory.go` - 情景记忆 Store
- [ ] `store/user_preferences.go` - 用户偏好 Store
- [ ] `store/db/postgres/episodic_memory.go` - PostgreSQL 实现

### 5.2 数据库变更

- [ ] `store/db/postgres/migration/xxx_add_episodic_memory.sql`
- [ ] `store/db/postgres/migration/xxx_add_user_preferences.sql`

### 5.3 测试文件

- [ ] `plugin/ai/memory/service_test.go`
- [ ] `store/episodic_memory_test.go`

---

## 6. 测试验收

### 6.1 功能测试

| 场景 | 输入 | 预期输出 |
|:---|:---|:---|
| 添加消息 | sessionID + Message | 成功存储 |
| 获取最近消息 | sessionID + limit=10 | 返回最多10条 |
| 保存情景记忆 | EpisodicMemory | 持久化到DB |
| 搜索情景记忆 | query="会议" | 返回相关记录 |
| 获取用户偏好 | userID | 返回偏好配置 |

### 6.2 性能验收

| 指标 | 目标值 | 测试方法 |
|:---|:---|:---|
| 短期记忆读写 | < 1ms | 单元测试 |
| 长期记忆写入 | < 50ms | 集成测试 |
| 偏好读取 | < 10ms | 集成测试 |

### 6.3 集成验收

- [ ] 团队 B 调用成功
- [ ] 团队 C 调用成功（偏好接口）

---

## 7. ROI 分析

| 维度 | 值 |
|:---|:---|
| 开发投入 | 3 人天 |
| 预期收益 | 跨会话记忆，重复询问 -50% |
| 风险评估 | 低 |
| 回报周期 | Phase 1 结束 |

---

## 8. 风险与缓解

| 风险 | 概率 | 影响 | 缓解措施 |
|:---|:---:|:---:|:---|
| 情景记忆数据膨胀 | 中 | 中 | 定期清理 + 重要性评分 |
| 短期记忆内存占用 | 低 | 低 | 限制窗口大小 |

---

## 9. 实施计划

### 9.1 时间表

| 阶段 | 时间 | 任务 |
|:---|:---|:---|
| Day 1 | 1人天 | 数据库 Schema + Store 实现 |
| Day 2 | 1人天 | 短期/长期记忆服务实现 |
| Day 3 | 1人天 | 测试 + 集成验证 |

### 9.2 检查点

- [ ] Day 1: DB 迁移完成
- [ ] Day 2: 服务实现完成
- [ ] Day 3: 团队 B/C 调用验证

---

## 附录

### A. 参考资料

- [主路线图 - 记忆系统架构](../../../archived/research_20250202/00-master-roadmap.md)
- [智能助理路线图](../../../archived/research_20250202/assistant-roadmap.md)

### B. 变更记录

| 日期 | 版本 | 变更内容 | 作者 |
|:---|:---|:---|:---|
| 2026-01-27 | v1.0 | 初始版本 | - |
