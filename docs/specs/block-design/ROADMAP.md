# Block Design Specs - 实施路线图

> **最后更新**: 2026-02-05 | **总投入**: 34-40 人天

## 快速导航

| 类别 | 文档 |
|:-----|:-----|
| **索引** | [INDEX.md](./INDEX.md) |
| **核心规格** | [unified-block-model.md](./unified-block-model.md) |
| **改进建议** | [unified-block-model_improvement.md](./unified-block-model_improvement.md) |
| **联合审计** | [joint-audit-report.md](./joint-audit-report.md) |

---

## 实施状态总览

| 模块 | 状态 | 版本 | 投入 |
|:-----|:-----|:-----|:-----|
| **UBM Phase 1-6** | ✅ 已实现 | v0.93.0 | 21人天 |
| **UBM 改进建议 (P0)** | 🔲 待开发 | - | 2-3人天 |
| **LLM 统计收集 (P1-A006)** | 🔲 待开发 | - | 3人天 |
| **树状会话分支** | 🔲 待开发 | - | 6-8人天 |

## 推荐后续实施顺序

根据 `joint-audit-report.md` 的审计结论，建议按以下顺序实施后续功能：

```
✅ 已完成: UBM Phase 1-6 (v0.93.0)
   └─ 数据库、API、前端、Handler、测试 全部完成

1️⃣ unified-block-model_improvement.md (P0)
   └─ 修复时间戳、乐观更新等基础问题

2️⃣ P1-A006-llm-stats-collection.md (P1)
   └─ LLM 统计收集，普通模式 Session Summary

3️⃣ tree-conversation-branching.md (P1)
   └─ 树状会话分支，编辑重生成
```

---

## Phase 1: 数据库 & 后端 Store (5人天) ✅

**文件**: [archived/unified-block-model-phase1.md](./archived/unified-block-model-phase1.md)
**状态**: ✅ 已实现 (v0.93.0)

| 项目 | 内容 |
|:-----|:-----|
| **目标** | 创建 `ai_block` 表和 PostgreSQL Store 实现 |
| **交付物** | 数据库迁移脚本、AIBlockStore 接口、PostgreSQL 实现 |
| **关键决策** | JSONB 存储用户输入和事件流、保留兼容视图 |

### 数据库表结构

```sql
CREATE TABLE ai_block (
  id BIGSERIAL PRIMARY KEY,
  conversation_id INTEGER NOT NULL,
  round_number INTEGER NOT NULL DEFAULT 0,
  block_type TEXT NOT NULL DEFAULT 'MESSAGE',
  mode TEXT NOT NULL DEFAULT 'normal',
  user_inputs JSONB NOT NULL DEFAULT '[]',
  assistant_content TEXT,
  event_stream JSONB NOT NULL DEFAULT '[]',
  session_stats JSONB,
  cc_session_id TEXT,
  status TEXT NOT NULL DEFAULT 'pending',
  metadata JSONB NOT NULL DEFAULT '{}',
  created_ts BIGINT NOT NULL,
  updated_ts BIGINT NOT NULL
);
```

---

## Phase 2: Proto & API (3人天) ✅

**文件**: [archived/unified-block-model-phase2.md](./archived/unified-block-model-phase2.md)
**状态**: ✅ 已实现 (v0.93.0)

| 项目 | 内容 |
|:-----|:-----|
| **目标** | 定义 Protobuf 消息类型和 BlockService API |
| **依赖** | Phase 1 |
| **交付物** | Proto 定义、Block Handler |

### Proto 消息类型

```protobuf
enum BlockType { MESSAGE = 1; CONTEXT_SEPARATOR = 2; }
enum BlockMode { NORMAL = 1; GEEK = 2; EVOLUTION = 3; }
enum BlockStatus { PENDING = 1; STREAMING = 2; COMPLETED = 3; ERROR = 4; }

message AIBlock {
  int64 id = 1;
  string uid = 2;
  int32 conversation_id = 3;
  int32 round_number = 4;
  BlockType block_type = 5;
  BlockMode mode = 6;
  repeated UserInput user_inputs = 7;
  string assistant_content = 8;
  repeated BlockEvent event_stream = 10;
  string cc_session_id = 12;
  BlockStatus status = 13;
  // ...
}
```

---

## Phase 3: 前端类型定义 (2人天) ✅

**文件**: [archived/unified-block-model-phase3.md](./archived/unified-block-model-phase3.md)
**状态**: ✅ 已实现 (v0.93.0)

| 项目 | 内容 |
|:-----|:-----|
| **目标** | 更新前端类型定义，支持 Block 模型 |
| **依赖** | Phase 2 |
| **交付物** | TypeScript 类型、React Query Hooks |

### 核心类型

```typescript
export interface AIBlock {
  id: number;
  uid: string;
  conversationId: number;
  roundNumber: number;
  blockType: BlockType;
  mode: BlockMode;
  userInputs: BlockUserInput[];
  assistantContent?: string;
  eventStream: BlockEvent[];
  sessionStats?: SessionSummary;
  status: BlockStatus;
  // ...
}
```

---

## Phase 4: 前端组件改造 (4人天) ✅

**文件**: [archived/unified-block-model-phase4.md](./archived/unified-block-model-phase4.md)
**状态**: ✅ 已实现 (v0.93.0)

| 项目 | 内容 |
|:-----|:-----|
| **目标** | 更新 ChatMessages 和 UnifiedMessageBlock 组件 |
| **依赖** | Phase 3 |
| **交付物** | ChatMessages 改造、AIChatContext 扩展 |

### 组件改造

```typescript
// 改造前：配对逻辑
const { userMessage, assistantMessage } = pairMessages(messages);

// 改造后：直接使用 Block
const { blocks } = useBlocks(conversationId);
blocks.map(block => <UnifiedMessageBlock block={block} />);
```

---

## Phase 5: Chat Handler 集成 (4人天) ✅

**文件**: [archived/unified-block-model-phase5.md](./archived/unified-block-model-phase5.md)
**状态**: ✅ 已实现 (v0.93.0)

| 项目 | 内容 |
|:-----|:-----|
| **目标** | 改造后端 Chat Handler，管理 Block 生命周期 |
| **依赖** | Phase 1, 2 |
| **交付物** | Handler 改造、EventWriter |

### Block 生命周期

```
Pending → Streaming → Completed
   │           │           │
   │           │           └── Error (异常)
   │           └── 事件流式写入
   └── 追加输入
```

---

## Phase 6: 集成测试 (3人天) ✅

**文件**: [archived/unified-block-model-phase6.md](./archived/unified-block-model-phase6.md)
**状态**: ✅ 已实现 (v0.93.0)

| 项目 | 内容 |
|:-----|:-----|
| **目标** | 端到端测试覆盖 |
| **依赖** | Phase 1-5 |
| **交付物** | 单元测试、集成测试、E2E 测试 |

---

## P0: 改进建议 (必须优先完成)

**文件**: [unified-block-model_improvement.md](./unified-block-model_improvement.md)

| Bug/改进 | 描述 | 影响 |
|:---------|:-----|:-----|
| **时间戳不一致** | 后端用秒、前端用毫秒 | 前端显示 1970 年 |
| **乐观更新失效** | onMutate 未插入缓存 | 用户体验卡顿 |
| **缺乏分支支持** | 无 parent_block_id | 无法支持编辑重生成 |

### 修复方案

```go
// 统一使用毫秒
time.Now().UnixMilli()  // 而非 Unix()
```

---

## P1-A006: LLM 统计收集 (3人天)

**文件**: [P1-A006-llm-stats-collection.md](./P1-A006-llm-stats-collection.md)

| 项目 | 内容 |
|:-----|:-----|
| **目标** | 普通模式 Session Summary 增强 |
| **依赖** | UBM Improvement (P0) |
| **交付物** | LLMService 重构、BaseParrot |

### 接口变更

```go
// 旧接口
Chat(ctx, messages) (string, error)

// 新接口
Chat(ctx, messages) (string, *LLMCallStats, error)
ChatStream(ctx, messages) (<-chan string, <-chan *LLMCallStats, <-chan error)
```

---

## 树状会话分支 (6-8人天)

**文件**: [tree-conversation-branching.md](./tree-conversation-branching.md)

| 项目 | 内容 |
|:-----|:-----|
| **目标** | 支持编辑历史消息并创建新分支 |
| **依赖** | UBM Phase 1-4, P1-A006 |
| **交付物** | Schema 变更、分支 API、前端组件 |

### 数据库变更

```sql
ALTER TABLE ai_block ADD COLUMN parent_block_id BIGINT;
ALTER TABLE ai_block ADD COLUMN branch_path TEXT;
```

### 分支结构

```
Block #0 (root)
  ├─ Block #1 (branch A)
  │   └─ Block #3
  └─ Block #2 (branch B - 用户编辑后重新生成)
```

---

## 验收标准汇总

| 类别 | 验收条件 |
|:-----|:---------|
| **数据库** | `ai_block` 表创建，索引生效 |
| **后端 API** | BlockService CRUD 完整，流式更新正常 |
| **前端** | 可渲染三种模式 Block，追加输入正常 |
| **集成** | 端到端流程通过，向后兼容 |

---

*维护者*: DivineSense 开发团队
