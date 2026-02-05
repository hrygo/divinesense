# Unified Block Model: 规格与实现差距分析报告

## 1. 执行摘要

**状态:** 当前系统实现显著 **领先于** `docs/specs/block-design/archived` 中的规格文档。

`unified-block-model-index.md` 显示大多数阶段 (Phase 1-6) 仍处于待开发状态 (🔲)。然而，代码分析显示 Phase 1 到 Phase 5 已基本 **完成**。所谓的“差距”主要是文档滞后，未反映当前代码库的成熟状态。

## 2. 详细差距分析 (按阶段)

| 阶段                   | 规格状态 | 实际实现状态      | 差距描述                                                                                                                                                    |
| :--------------------- | :------- | :---------------- | :---------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **P1: 数据库 & Store** | 🔲 待开发 | **✅ 已完成**      | `store/ai_block.go` 和 `store/db/postgres/ai_block.go` 实现了完整的 schema（`ai_block` 表，包含 `user_inputs`, `event_stream` 等 JSONB 字段）及 CRUD 操作。 |
| **P2: Proto & API**    | 🔲 待开发 | **✅ 已完成**      | `proto/api/v1/ai_service.proto` 包含了所有 Block RPC 定义（`ListBlocks`, `CreateBlock` 等），服务端也在 `ai_service_block.go` 中实现了这些接口。            |
| **P3: 前端类型**       | 🔲 待开发 | **✅ 已完成**      | `web/src/types/block.ts` 定义了所有必要的类型、枚举和辅助函数（如 `createBlockWithMetadata`）。                                                             |
| **P4: 前端组件**       | 🔲 待开发 | **✅ 已完成**      | `UnifiedMessageBlock.tsx` 是一个功能完备的组件，支持流式传输、可折叠区域、工具调用显示以及基于模式的主题切换。`ChatMessages.tsx` 包含了适配 UI 的逻辑。     |
| **P5: Chat 集成**      | 🔲 待开发 | **⚠️ 部分/已集成** | `ai_service_chat.go` 初始化了 `BlockManager` 并将其传递给 `ParrotHandler`。集成逻辑已存在，尽管本次静态分析未包含深度运行时行为验证。                       |
| **P6: 测试**           | 🔲 待开发 | **❓ 未知**        | 静态分析未涵盖测试覆盖率，但测试基础设施已存在。                                                                                                            |

## 3. 语义与架构对比

### 3.1 数据模型一致性
*   **规格意图:** 将 "Block" 视为对话的原子单元，封装用户输入（数组）、AI 回复和内部事件（思考/工具调用）。
*   **实现情况:**
    *   **完全匹配:** `AIBlock` 结构体完美复刻了规格设计，包含 `UserInputs []UserInput`, `EventStream []BlockEvent`, `SessionStats`, 和 `Mode`。
    *   **演进:** 实现中包含了 `ParentBlockID` 和 `BranchPath` (在 `store/ai_block.go` 中)，这表明系统已经 **超越** 了原始规格，支持分支/分叉功能（可能用于 "Canvas" 或高级历史记录功能）。

### 3.2 关键偏差 (正向演进)
实现中包含了归档规格中未详细说明的特性，表明开发工作已超出初始设计：
1.  **分支支持:** 数据库和结构体中增加了 `parent_block_id` 和 `branch_path`。
2.  **扩展状态管理:** `ChatMessages.tsx` 包含处理混合状态和回退到旧版 `ChatMessage` 类型的逻辑，确保了向后兼容性。
3.  **复杂 UI 逻辑:** 前端组件处理了丰富的交互（例如区分 "思考中" 与 "工具调用中" 的 `streamingPhase`），比规格的高层级要求更为细致。

## 4. 建议

1.  **更新文档:** `docs/specs/block-design/archived` 中的规格应被标记为 **已实现** 或 **已废弃**。它们是准确的历史参考，但如果被视为“待办事项”则具有误导性。
2.  **验证运行时迁移:** 虽然代码已就位，建议验证旧的 `ai_message` 表是否仍在冗余填充，或者系统是否已完全切换到 `ai_block`。`ChatMessages.tsx` 中使用 `convertAIBlocksToMessageBlocks` 表明 UI 已准备好直接消费 Block 数据。
3.  **整合规格:** 创建一份新的“现状 (As-Built)”架构文档，反映当前的实际实现，包括已添加的分支/分叉能力。

## 5. 结论
在功能缺失方面不存在“实现差距”。系统完全支持 Unified Block Model。真正的差距在于 **文档的时效性**。工程团队已执行了计划，只是未及时更新状态看板。
