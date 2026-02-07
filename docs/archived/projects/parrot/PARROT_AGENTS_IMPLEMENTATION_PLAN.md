# 鹦鹉助手家族实施方案 (Master Plan)

> **版本**: v3.0 (根据 RAG 调研报告与现状深度优化)
> **状态**: 待执行
> **核心文档**: 本文档作为总纲，详细技术规范请查阅 `docs/specs/` 目录下的子文档。

本方案整合了《Memos 重构方案》、《终版技术方案 v2.1》及《RAG 系统调研报告》，旨在通过 **"聊天增强"** 的方式，将 Memos 升级为真正的个人智能助理。

## 📚 详细规范文档 (Specs)

请首先阅读以下详细设计文档，它们包含具体的实现细节和验收标准：

- **[SPEC-001: 后端基础设施与 BaseParrot](specs/SPEC-001-INFRA-BASE-PARROT.md)**
    - 涵盖: `BaseParrot` 基类, `ParrotRouter` 路由, 统一类型系统, 错误处理。
- **[SPEC-002: 笔记与创意助手 (Memo & Creative)](specs/SPEC-002-AGENT-MEMO-CREATIVE.md)**
    - 涵盖: `MemoParrot` (重构), `CreativeParrot` (新建), 提示词工程。
- **[SPEC-003: 惊奇助手与 RAG 引擎 (Amazing)](specs/SPEC-003-AGENT-AMAZING.md)**
    - 涵盖: `AmazingParrot` (元 Agent), RRF 混合检索, 并发搜索, Reranker 集成。
- **[SPEC-004: 前端交互与 UI/UX 优化](specs/SPEC-004-FRONTEND-UI-UX.md)**
    - 涵盖: 鹦鹉选择器, 快捷卡片, 结果卡片 (Memo/Schedule/Amazing), 动效与交互优化。

## 🗓️ 实施路线图 (Roadmap)

### 第一阶段：核心基础设施 (P0) - 预计 1.5 天
- [ ] 建立 `docs/specs/` 目录并归档规范文档。
- [ ] 实现 `BaseParrot` 基类 (统一 ReAct 循环、工具解析、重试机制)。
- [ ] 升级 `ParrotRouter` 以支持多 Agent 注册与自动路由。
- [ ] 统一后端类型定义 (`AgentType`, 事件常量)。

### 第二阶段：基础 Agent 实现 (P0) - 预计 2 天
- [ ] 重构 `MemoParrot` 以继承 `BaseParrot` (移除冗余代码)。
- [ ] 实现 `CreativeParrot` (灵灵)，专注于创意发散与头脑风暴。
- [ ] 验证 `ScheduleParrot` (金刚) 的包装器实现 (零代码重写)。

### 第三阶段：惊奇 Agent (RAG 核心) (P1) - 预计 2 天
- [ ] 实现 `AmazingParrot` (惊奇)，作为编排者。
- [ ] 集成并发检索 (Memo + Schedule)。
- [ ] (可选) 初步集成 RRF 融合排序逻辑 (参考 RAG 报告)。

### 第四阶段：前端集成与体验打磨 (P1) - 预计 2.5 天
- [ ] 优化 `ParrotQuickActions` 和 `ParrotSelector` 视觉效果 (动效、状态反馈)。
- [ ] 开发/优化 `MemoQueryResult` 和 `AmazingQueryResult` 组件。
- [ ] 实现流式响应的平滑渲染与 "正在思考" 状态动画。
- [ ] 验收测试与 UI 走查。

## 🎯 核心目标与验收概览

| 模块                | 核心目标                   | 验收关键点                                             |
| :------------------ | :------------------------- | :----------------------------------------------------- |
| **基础设施**        | 消除重复代码，统一错误处理 | 所有 Agent 共享同一套 ReAct 逻辑；Panic 恢复机制生效。 |
| **灰灰 (Memo)**     | 准确检索，去幻觉           | 搜索结果相关度 > 80%；无结果时如实告知。               |
| **金刚 (Schedule)** | 稳定创建，零破坏           | 现有日程管理功能 100% 正常；自然语言解析准确。         |
| **灵灵 (Creative)** | 思维发散，活泼有趣         | 给出的建议具有创意；Tone (语气) 符合人设。             |
| **惊奇 (Amazing)**  | 全能编排，信息综合         | 一次查询同时返回笔记和日程；响应时间 < 2s。            |
| **UI/UX**           | 流畅自然，感知清晰         | 切换 Agent 无卡顿；思考状态可见；移动端适配良好。      |

## ⚠️ 风险与缓解

1.  **RAG 性能**: 混合检索可能增加延迟。
    *   *缓解*: 使用并发 Go 协程 (`errgroup`) 执行检索；第一阶段暂不上 Reranker。
2.  **UI 复杂度**: 多种结果卡片可能导致聊天流杂乱。
    *   *缓解*: 统一卡片设计语言 (Unified Style)，参考 `docs/images/parrot_agents_ui_concept.png`。
3.  **Prompt 调优**: 创意 Agent 可能过于发散。
    *   *缓解*: 使用 Few-Shot Prompting 约束输出格式。
