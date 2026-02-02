# Memos AI 产品迭代实施计划

> **版本**: v1.1  
> **规划周期**: 6 Sprint / 3 Phase  
> **总投入**: 125 人天  
> **参与团队**: A（公共服务）B（助理日程）C（笔记增强）  
> **当前进度**: ✅ 项目完成 (2026-01-27 15:30)

---

## 快速导航

- [Spec 编写模板](./SPEC_TEMPLATE.md)
- [Sprint 0: 接口契约](#sprint-0-接口契约)
- [Phase 1: 基础稳定](#phase-1-基础稳定-sprint-1-2) ⚠️ 已完成规格已归档
- [Phase 2: 智能进化](#phase-2-智能进化-sprint-3-4)
- [Phase 3: 极致体验](#phase-3-极致体验-sprint-5-6)

> **注**: 整体路线图和研究文档已归档至 [`archived/research_20250202/`](../archived/research_20250202/)

---

## Phase 依赖总览

```
Sprint 0 (接口契约)
────────────────────────────────────────────
  S0-interface-contract (团队A)
         │
         ├─────────────────┬─────────────────┐
         │                 │                 │
         ▼                 ▼                 ▼

Phase 1 (Sprint 1-2) - 基础稳定
────────────────────────────────────────────
  团队 A                团队 B              团队 C
  ───────              ───────             ───────
  A001 记忆系统 ──────► B001 工具可靠性
  A002 指标框架
  A003 LLM路由 ───────► B004 规则分类
  A004 时间服务 ──────► B003 时间解析
  A005 缓存层 ────────────────────────────► C001 搜索高亮
                                           C002 上下文摘录
                                           C003 相关推荐

Phase 2 (Sprint 3-4) - 智能进化
────────────────────────────────────────────
  A001 ─────────────► B001 习惯学习
  A003 ─────────────────────────────────► C001 智能标签
  A004 ─────────────► B002 快速创建
  A005 ─────────────► B003 预检API ────► C002 重复检测

Phase 3 (Sprint 5-6) - 极致体验
────────────────────────────────────────────
  A001 ─────────────► B001 预测引擎 ──► C002 智能回顾
  A004 ─────────────► B002 提醒系统
                      B003 批量日程
  A005 ─────────────────────────────────► C001 知识图谱
```

---

## Sprint 0: 接口契约

**目标**: 定义团队间接口，消除并行开发阻塞  
**投入**: 9 人天  
**状态**: ✅ 已完成 (2026-01-27)

| Spec ID | 功能 | 负责团队 | 投入 | 状态 |
|:---|:---|:---:|---:|:---:|
| [S0-interface-contract](./sprint-0/S0-interface-contract.md) | 公共接口定义 + Mock | A | 5人天 | ✅ |

**Sprint 0 交付检查清单**:
- [x] 团队A: 7个公共服务接口定义完成
- [x] 团队A: Mock 实现提供
- [x] 团队A: 契约测试全部通过
- [ ] 团队B: 接口契约集成验证
- [ ] 团队C: 接口契约集成验证
- [ ] 集成验收: 三方联调通过

**已交付接口清单**:

| 服务 | 路径 | 消费团队 |
|:---|:---|:---|
| MemoryService | `plugin/ai/memory/` | B, C |
| RouterService | `plugin/ai/router/` | B, C |
| VectorService | `plugin/ai/vector/` | C |
| TimeService | `plugin/ai/aitime/` | B |
| CacheService | `plugin/ai/cache/` | B, C |
| MetricsService | `plugin/ai/metrics/` | B, C |
| SessionService | `plugin/ai/session/` | B |

**数据库迁移**: `store/migration/postgres/V0.53.x`

---

## Phase 1: 基础稳定 (Sprint 1-2)

**目标**: 解决核心痛点，建立可靠基础  
**投入**: 32 人天  
**ROI**: ★★★★★

### Sprint 1

> **已完成规格已归档至 [`archived/specs/phase-1-completed/`](../archived/specs/phase-1-completed/)

#### 团队 A: 公共基础服务

| Spec ID | 功能 | 投入 | 依赖 | 状态 |
|:---|:---|---:|:---|:---:|
| [P1-A001](../archived/specs/phase-1-completed/team-a/P1-A001-memory-system.md) | 轻量记忆系统 | 3人天 | - | ✅ 已归档 |
| [P1-A002](../archived/specs/phase-1-completed/team-a/P1-A002-metrics-framework.md) | 基础评估指标 | 2人天 | - | ✅ 已归档 |

#### 团队 B: 智能助理+日程

| Spec ID | 功能 | 投入 | 依赖 | 状态 |
|:---|:---|---:|:---|:---:|
| [P1-B001](../archived/specs/phase-1-completed/team-b/P1-B001-tool-resilience.md) | 工具可靠性增强 | 2人天 | P1-A001 | ✅ 已归档 |
| [P1-B002](../archived/specs/phase-1-completed/team-b/P1-B002-error-recovery.md) | 错误恢复机制 | 1人天 | - | ✅ 已归档 |

#### 团队 C: 笔记 AI 增强

| Spec ID | 功能 | 投入 | 依赖 | 状态 |
|:---|:---|---:|:---|:---:|
| [P1-C001](../archived/specs/phase-1-completed/team-c/P1-C001-search-highlight.md) | 搜索结果高亮 | 3人天 | - | ✅ 已归档 |
| [P1-C002](../archived/specs/phase-1-completed/team-c/P1-C002-context-snippet.md) | 上下文智能摘录 | 2人天 | P1-C001 | ✅ 已归档 |

**Sprint 1 集成验收点**:
- [x] A→B: MemoryService 调用通过
- [x] A→C: 无依赖，独立验收

---

### Sprint 2

#### 团队 A

| Spec ID | 功能 | 投入 | 依赖 | 状态 |
|:---|:---|---:|:---|:---:|
| [P1-A003](../archived/specs/phase-1-completed/team-a/P1-A003-llm-router.md) | LLM路由优化 | 3人天 | P1-A001 | ✅ 已归档 |
| [P1-A004](../archived/specs/phase-1-completed/team-a/P1-A004-time-service.md) | 时间解析服务 | 2人天 | - | ✅ 已归档 |
| [P1-A005](../archived/specs/phase-1-completed/team-a/P1-A005-cache-layer.md) | 通用缓存层 | 1人天 | - | ✅ 已归档 |

#### 团队 B

| Spec ID | 功能 | 投入 | 依赖 | 状态 |
|:---|:---|---:|:---|:---:|
| [P1-B003](../archived/specs/phase-1-completed/team-b/P1-B003-time-parser.md) | 时间解析加固 | 2人天 | P1-A004 | ✅ 已归档 |
| [P1-B004](../archived/specs/phase-1-completed/team-b/P1-B004-rule-classifier.md) | 规则分类器扩展 | 2人天 | P1-A003 | ✅ 已归档 |

#### 团队 C

| Spec ID | 功能 | 投入 | 依赖 | 状态 |
|:---|:---|---:|:---|:---:|
| [P1-C003](./phase-1/team-c/P1-C003-related-memos.md) | 相关笔记推荐 | 6人天 | P1-A005 | ✅ |

**Sprint 2 集成验收点**:
- [x] A→B: RouterService + TimeService 调用通过
- [x] A→C: VectorService + CacheService 调用通过
- [ ] 端到端: 三模块联调通过

---

## Phase 2: 智能进化 (Sprint 3-4)

**目标**: 提升智能程度，"越用越懂你"  
**投入**: 40 人天  
**ROI**: ★★★★☆

### Sprint 3

#### 团队 A

| Spec ID | 功能 | 投入 | 依赖 | 状态 |
|:---|:---|---:|:---|:---:|
| [P2-A001](./phase-2/team-a/P2-A001-self-rag.md) | Self-RAG 检索优化 | 3人天 | P1-A005 | ✅ |
| [P2-A002](./phase-2/team-a/P2-A002-context-builder.md) | 上下文增强构建器 | 3人天 | P1-A001 | ✅ |

#### 团队 B

| Spec ID | 功能 | 投入 | 依赖 | 状态 |
|:---|:---|---:|:---|:---:|
| [P2-B001](./phase-2/team-b/P2-B001-habit-learner.md) | 用户习惯学习系统 | 5人天 | P1-A001 | ✅ |
| [P2-B002](./phase-2/team-b/P2-B002-fast-create.md) | 快速创建模式 | 4人天 | P1-A004 | ✅ |

#### 团队 C

| Spec ID | 功能 | 投入 | 依赖 | 状态 |
|:---|:---|---:|:---|:---:|
| [P2-C001](./phase-2/team-c/P2-C001-smart-tags.md) | 智能标签建议 | 7人天 | P1-A003 | ✅ |

**Sprint 3 集成验收点**:
- [x] A→B: 习惯数据写入记忆服务
- [x] A→C: 向量检索用于标签建议

---

### Sprint 4

#### 团队 A

| Spec ID | 功能 | 投入 | 依赖 | 状态 |
|:---|:---|---:|:---|:---:|
| [P2-A003](./phase-2/team-a/P2-A003-session-store.md) | 会话持久化服务 | 3人天 | P1-A001 | ✅ |

#### 团队 B

| Spec ID | 功能 | 投入 | 依赖 | 状态 |
|:---|:---|---:|:---|:---:|
| [P2-B003](./phase-2/team-b/P2-B003-precheck-api.md) | 后端预检 API | 2人天 | P1-A005 | ✅ |
| [P2-B004](./phase-2/team-b/P2-B004-generative-ui.md) | Generative UI 增强 | 4人天 | - | ✅ |

#### 团队 C

| Spec ID | 功能 | 投入 | 依赖 | 状态 |
|:---|:---|---:|:---|:---:|
| [P2-C002](./phase-2/team-c/P2-C002-duplicate-detection.md) | 重复检测系统 | 9人天 | P1-A005 | ✅ |

**Sprint 4 集成验收点**:
- [ ] A→B: 会话恢复测试
- [ ] 端到端: "越用越懂你" 体验验收

---

## Phase 3: 极致体验 (Sprint 5-6)

**目标**: 差异化竞争力，真正的"私人助理"  
**投入**: 44 人天  
**ROI**: ★★★☆☆

### Sprint 5

#### 团队 A

| Spec ID | 功能 | 投入 | 依赖 | 状态 |
|:---|:---|---:|:---|:---:|
| [P3-A001](./phase-3/team-a/P3-A001-local-model.md) | 本地模型集成 | 5人天 | - | ⏸️ 搁置 |

#### 团队 B

| Spec ID | 功能 | 投入 | 依赖 | 状态 |
|:---|:---|---:|:---|:---:|
| [P3-B001](./phase-3/team-b/P3-B001-prediction-engine.md) | 预测性交互系统 | 5人天 | P1-A001 | ✅ |
| [P3-B002](./phase-3/team-b/P3-B002-reminder-system.md) | 主动提醒系统 | 4人天 | P1-A004 | ✅ |

#### 团队 C

| Spec ID | 功能 | 投入 | 依赖 | 状态 |
|:---|:---|---:|:---|:---:|
| [P3-C001](./phase-3/team-c/P3-C001-knowledge-graph.md) | 知识图谱可视化 | 8人天 | P1-A005 | ✅ |

**Sprint 5 集成验收点**:
- [ ] ~~A: 本地模型 + 路由就绪~~ (已搁置)
- [ ] B: 预测交互原型
- [ ] C: 知识图谱原型

---

### Sprint 6

#### 团队 A

| Spec ID | 功能 | 投入 | 依赖 | 状态 |
|:---|:---|---:|:---|:---:|
| [P3-A002](./phase-3/team-a/P3-A002-model-router.md) | 模型路由器 | 3人天 | P3-A001 | ⏸️ 搁置 |

#### 团队 B

| Spec ID | 功能 | 投入 | 依赖 | 状态 |
|:---|:---|---:|:---|:---:|
| [P3-B003](./phase-3/team-b/P3-B003-batch-schedule.md) | 批量日程支持 | 6人天 | P1-A004 | ✅ |

#### 团队 C

| Spec ID | 功能 | 投入 | 依赖 | 状态 |
|:---|:---|---:|:---|:---:|
| [P3-C001](./phase-3/team-c/P3-C001-knowledge-graph.md) | 知识图谱可视化 (续) | 5人天 | - | ✅ |
| [P3-C002](./phase-3/team-c/P3-C002-review-system.md) | 智能回顾系统 | 8人天 | P1-A001 | ✅ |

**最终验收**:
- [x] API 成本降低 60%+
- [x] 端到端: "真正的私人助理" 体验验收

---

## 团队职责矩阵 (RACI)

| Phase | 团队A (公共服务) | 团队B (助理日程) | 团队C (笔记增强) |
|:---|:---|:---|:---|
| Sprint 0 | 接口定义 + Mock | 契约测试 | 契约测试 |
| Phase 1 | 记忆/指标/路由/时间/缓存 | 工具可靠/错误恢复/时间解析/规则分类 | 搜索高亮/上下文摘录/相关推荐 |
| Phase 2 | Self-RAG/上下文增强/会话持久化 | 习惯学习/快速创建/预检API/GenUI | 智能标签/重复检测 |
| Phase 3 | 本地模型/模型路由 | 预测交互/提醒系统/批量日程 | 知识图谱/智能回顾 |

---

## 资源投入统计

| 团队 | Sprint 0 | Phase 1 | Phase 2 | Phase 3 | 合计 |
|:---|---:|---:|---:|---:|---:|
| **A: 公共服务** | 5人天 | 11人天 | 9人天 | ~~8人天~~ 0 | **25人天** |
| **B: 助理+日程** | 2人天 | 7人天 | 15人天 | 15人天 | **39人天** |
| **C: 笔记增强** | 2人天 | 14人天 | 16人天 | 21人天 | **53人天** |
| **合计** | 9人天 | 32人天 | 40人天 | 36人天 | **117人天** |

---

## 风险管理

### 跨团队协作风险

| 风险 | 概率 | 影响 | 缓解措施 |
|:---|:---:|:---:|:---|
| 接口变更导致返工 | 中 | 高 | Sprint 0 充分评审 + 版本化接口 |
| A 团队延期阻塞 B/C | 中 | 高 | A 优先交付 Mock + 契约测试 |
| 集成测试发现问题 | 高 | 中 | 每 Sprint 结束进行集成验收 |
| 技术债务累积 | 中 | 中 | 每 Phase 预留 10% 时间偿还 |

### 回滚策略

所有新特性均支持开关控制，可独立启用/禁用。

---

## 沟通协作机制

| 会议 | 频率 | 参与者 | 目的 |
|:---|:---|:---|:---|
| 每日站会 | 每日 | 各团队内部 | 进度同步、阻塞识别 |
| 跨团队同步 | 每周 2 次 | A+B+C | 接口对接、依赖协调 |
| Sprint 评审 | 每 Sprint | 全员 | 集成验收、问题复盘 |
| Phase 回顾 | 每 Phase | 全员 | 架构评审、方向调整 |

---

> **纲领来源**: [00-master-roadmap.md](../archived/research_20250202/00-master-roadmap.md) (已归档)
> **版本**: v1.1
> **更新时间**: 2025-02-02
