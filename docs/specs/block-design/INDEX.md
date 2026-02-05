# Block Design Specs - 规格索引

> **最后更新**: 2026-02-05 | **状态**: ✅ UBM 已实现 (v0.93.0) | 扩展功能待开发

## 概述

本目录包含 DivineSense AI 聊天系统的 Block Design 规格，涵盖 Unified Block Model 及其扩展功能的完整技术规范。

```
┌─────────────────────────────────────────────────────────────────────────┐
│                        DivineSense AI 聊天系统架构                      │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │  Unified Block Model (UBM) - 核心数据模型                        │   │
│  │  - ai_block 表：统一存储对话回合                                 │   │
│  │  - 支持 Normal/Geek/Evolution 三种模式                            │   │
│  │  - 完整事件流持久化 (thinking/tool_use/answer)                    │   │
│  └─────────────────────────────────────────────────────────────────┘   │
│                                                                         │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │  扩展功能                                                        │   │
│  │  ├── LLM Stats Collection (P1-A006) - 普通 Token 统计           │   │
│  │  └── Tree Conversation Branching - 对话分支/编辑重生成           │   │
│  └─────────────────────────────────────────────────────────────────┘   │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## 文档导航

### 1. 核心规格 (Core Specs)

| 文档 | 描述 | 状态 | 优先级 |
|:-----|:-----|:-----|:-------|
| [unified-block-model.md](./unified-block-model.md) | UBM 主规格文档 - 完整架构设计 | ✅ 已实现 (v0.93.0) | P0 |
| [archived/unified-block-model-index.md](./archived/unified-block-model-index.md) | UBM Phase 实施索引 (已归档) | ✅ 已实现 | P0 |

### 2. Phase 规格 (Implementation Phases)

| Phase | 文档 | 投入 | 状态 | 依赖 |
|:------|:-----|:-----|:-----|:-----|
| **Phase 1** | [archived/unified-block-model-phase1.md](./archived/unified-block-model-phase1.md) | 5人天 | ✅ 已实现 | - |
| **Phase 2** | [archived/unified-block-model-phase2.md](./archived/unified-block-model-phase2.md) | 3人天 | ✅ 已实现 | - |
| **Phase 3** | [archived/unified-block-model-phase3.md](./archived/unified-block-model-phase3.md) | 2人天 | ✅ 已实现 | Phase 2 |
| **Phase 4** | [archived/unified-block-model-phase4.md](./archived/unified-block-model-phase4.md) | 4人天 | ✅ 已实现 | Phase 3 |
| **Phase 5** | [archived/unified-block-model-phase5.md](./archived/unified-block-model-phase5.md) | 4人天 | ✅ 已实现 | Phase 2 |
| **Phase 6** | [archived/unified-block-model-phase6.md](./archived/unified-block-model-phase6.md) | 3人天 | ✅ 已实现 | Phase 1-5 |

**总计**: 21 人天 | **状态**: ✅ 全部完成 (v0.93.0) | 详情已归档至 `archived/`

### 3. 改进与修复 (Improvements & Fixes)

| 文档 | 描述 | 状态 | 优先级 |
|:-----|:-----|:-----|:-------|
| [unified-block-model_improvement.md](./unified-block-model_improvement.md) | UBM 改进建议 - Bug 修复和标准统一 | 🔲 待开发 | **P0** |

**关键修复**:
- 时间戳单位统一 (秒 → 毫秒)
- 乐观更新逻辑修复
- 为树状分支预留 `parent_block_id`

### 4. 扩展功能 (Extended Features)

| 文档 | 描述 | 状态 | 优先级 |
|:-----|:-----|:-----|:-------|
| [P1-A006-llm-stats-collection.md](./P1-A006-llm-stats-collection.md) | LLM 层统计收集 - 普通 Token 统计 | 🔲 待开发 | P1 |
| [tree-conversation-branching.md](./tree-conversation-branching.md) | 树状会话分支 - 编辑重生成功能 | 🔲 待开发 | P1 |
| [ai-block-fields-extension.md](./ai-block-fields-extension.md) | ai_block 字段扩展 - Token/成本/反馈/软删除 | 📝 已提议 | P1 |

### 5. 审计与协调 (Audit & Coordination)

| 文档 | 描述 | 状态 |
|:-----|:-----|:-----|
| [joint-audit-report.md](./joint-audit-report.md) | 三方规格联合审计报告 | ✅ 已完成 |

---

## 实施状态总结

| 模块 | 状态 | 版本 |
|:-----|:-----|:-----|
| **Unified Block Model (Phase 1-6)** | ✅ 已实现 | v0.93.0 |
| **UBM 改进建议 (P0)** | 🔲 待开发 | - |
| **LLM 统计收集 (P1-A006)** | 🔲 待开发 | - |
| **树状会话分支** | 🔲 待开发 | - |

## 推荐后续实施路径

```
┌─────────────────────────────────────────────────────────────────────────┐
│  ✅ 已完成: Unified Block Model (Phase 1-6) - v0.93.0                   │
│     ├─ Phase 1: 数据库 & 后端 Store                                    │
│     ├─ Phase 2: Proto & API                                            │
│     ├─ Phase 3: 前端类型定义                                           │
│     ├─ Phase 4: 前端组件改造                                           │
│     ├─ Phase 5: Chat Handler 集成                                      │
│     └─ Phase 6: 集成测试                                               │
├─────────────────────────────────────────────────────────────────────────┤
│  🔲 下一步: 地基修复 (Foundation) - P0                                  │
│  └─ unified-block-model_improvement.md                                 │
│     ├─ 修复时间戳 Bug (统一为毫秒)                                     │
│     ├─ 优化前端乐观更新逻辑                                           │
│     └─ 为后续功能提供稳定、标准一致的数据底座                          │
├─────────────────────────────────────────────────────────────────────────┤
│  🔲 后续: 核心重构 (Core Refactor) - P1                                │
│  └─ P1-A006-llm-stats-collection.md                                    │
│     ├─ 重构 LLMService 接口 (Stateless)                               │
│     ├─ 实现 Token/Duration 统计流                                      │
│     └─ 确立新的后端服务接口标准                                       │
├─────────────────────────────────────────────────────────────────────────┤
│  🔲 最终: 功能扩展 (Feature Expansion) - P1                            │
│  └─ tree-conversation-branching.md                                     │
│     ├─ 实施 parent_block_id 和 branch_path Schema 变更                │
│     ├─ 基于新版 LLMService 实现 ForkBlock                              │
│     └─ 对话功能从线性升级为树状                                       │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## 依赖关系图

```
           ✅ Unified Block Model (Phase 1-6) - 已实现
                          │
            ┌─────────────┼─────────────┐
            │             │             │
            ▼             ▼             ▼
    🔲 P1-A006     🔲 tree-conversation   🔲 UBM Improvement
    (LLM 统计)     (树状分支)           (Bug 修复)
            │             │             │
            └─────────────┴─────────────┘
                          │
                          ▼
                    完整 Block 设计
```

---

## 数据模型速查

### ai_block 表结构

| 字段 | 类型 | 描述 |
|:-----|:-----|:-----|
| `id` | BIGSERIAL | 主键 |
| `conversation_id` | INTEGER | 所属会话 |
| `round_number` | INTEGER | 会话内轮次 (0-based) |
| `block_type` | TEXT | 'message' | 'context_separator' |
| `mode` | TEXT | 'normal' | 'geek' | 'evolution' |
| `user_inputs` | JSONB | 用户输入数组 [{content, timestamp}] |
| `assistant_content` | TEXT | AI 回复内容 |
| `event_stream` | JSONB | 事件流数组 [{type, content, timestamp, meta}] |
| `session_stats` | JSONB | 会话统计 (CC 模式) |
| `cc_session_id` | TEXT | CC 会话 UUID v5 映射 |
| `status` | TEXT | 'pending' | 'streaming' | 'completed' | 'error' |
| `metadata` | JSONB | 元数据 |
| `created_ts` | BIGINT | 创建时间戳 |
| `updated_ts` | BIGINT | 更新时间戳 |

### 扩展字段 (Tree Branching)

| 字段 | 类型 | 描述 |
|:-----|:-----|:-----|
| `parent_block_id` | BIGINT | 父 Block ID (支持树状分支) |
| `branch_path` | TEXT | 分支路径 (如 "0/1/2") |

### 扩展字段 (Fields Extension - P1)

| 字段 | 类型 | 描述 |
|:-----|:-----|:-----|
| `token_usage` | JSONB | Token 使用明细 (prompt/completion/cache) |
| `cost_estimate` | BIGINT | 成本估算（毫厘，1/1000 美分） |
| `model_version` | TEXT | LLM 模型版本 (如 deepseek/deepseek-chat) |
| `user_feedback` | INTEGER | 用户评分 (1-5, NULL 表示未评分) |
| `error_message` | TEXT | 错误详情（当 status=error 时填充） |
| `regeneration_count` | INTEGER | 重新生成次数 |
| `archived_at` | BIGINT | 软删除时间戳（NULL 表示正常） |

---

## 版本历史

| 日期 | 版本 | 变更内容 |
|:-----|:-----|:---------|
| 2026-02-05 | v1.0 | 创建规格索引，整理文档结构 |

---

*维护者*: DivineSense 开发团队
*反馈渠道*: [GitHub Issues](https://github.com/hrygo/divinesense/issues)
