# 架构决策记录

> DivineSense 项目的关键架构决策及其理由

---

## AI 模块组织

### 决策：AI 作为一级模块

```
ai/               # ✅ 一级模块
├── agent/        # 鹦鹉代理
├── router/       # 意图路由
├── core/         # AI 基础设施
└── ...

server/ai/        # ❌ 不使用
plugin/ai/        # ❌ 不使用（插件为非AI）
```

**理由**：
- AI 是核心功能，不是附加组件
- 简化导入路径：`ai/agent/` vs `server/router/api/v1/ai/agent/`
- 清晰的架构边界：AI 模块独立演进

**相关**：
- `plugin/` 目录包含非 AI 插件（Markdown、OCR、Webhook等）
- `server/` 专注于 HTTP/gRPC 服务层

---

## 数据库策略

### 决策：PostgreSQL 生产级，SQLite 仅开发

| 环境 | 数据库 | AI 支持 |
|:-----|:-------|:-------|
| 生产 | PostgreSQL | ✅ 完整支持 |
| 开发 | PostgreSQL | ✅ 完整支持 |
| 开发（可选）| SQLite | ❌ 无 AI |

**理由**：
- `pgvector` 扩展是向量搜索的基础
- SQLite 不支持 pgvector，限制 AI 功能
- 开发环境可用 SQLite 进行非 AI 功能开发

**相关 Issue**: #9 - SQLite AI 支持研究

---

## 路由机制

### 决策：四层意图路由 + AUTO 标记

```
用户输入 → EvolutionMode? ─Yes→ EvolutionParrot
                  │
                  No
                  ↓
           GeekMode? ─Yes→ GeekParrot
                  │
                  No
                  ↓
           AgentType == AUTO?
                  │
           Yes ─────┴── No (直接使用指定鹦鹉)
                  ↓
           ChatRouter.Route()
                  ↓
    Cache → Rule → History → LLM
```

**关键理解**：
- `AUTO` 不是一只鹦鹉，是"请后端路由"的标记
- 五只鹦鹉：MEMO, SCHEDULE, AMAZING, GEEK, EVOLUTION
- 路由层有优先级：Evolution > Geek > 具体鹦鹉 > AUTO

---

## 数据库迁移

### 决策：双文件同步

每次数据库变更必须同步更新：
1. `store/migration/postgres/migrate/YYYYMMDDHHMMSS_description.up.sql`
2. `store/migration/postgres/schema/LATEST.sql`

**理由**：
- `migrate/` 支持增量升级（已部署数据库）
- `LATEST.sql` 支持全新安装
- 不一致会导致：新安装缺少表 / 升级后结构不同

**验证命令**：
```bash
# 检查表数量一致性
grep -c '^CREATE TABLE' migrate/*.up.sql
grep -c '^CREATE TABLE' schema/LATEST.sql
```

---

## Unified Block Model

### 决策：AI 聊天使用统一块模型

```
AIBlock {
    id, uid, conversation_id, round_number
    mode: normal | geek | evolution
    user_inputs[]
    assistant_content
    event_stream[]
    session_stats
    status: pending | streaming | completed | error
}
```

**理由**：
- 统一的数据结构支持多种模式
- 流式事件持久化（thinking, tool_use, answer）
- 会话统计（tokens, cost）可追溯

**相关**：
- 前端：`UnifiedMessageBlock` 组件
- 数据库表：`ai_block`

---

## 前端状态管理

### 决策：TanStack Query 为主，Context API 辅助

| 用途 | 方案 |
|:-----|:-----|
| 服务器状态 | TanStack Query |
| 全局 UI 状态 | Context API |
| 组件本地状态 | useState |
| 表单状态 | useState + useImmer |

**理由**：
- TanStack Query 自动处理缓存、重试、失效
- Context API 适合全局 UI（主题、语言、用户）
- 避免过度抽象，保持简单
