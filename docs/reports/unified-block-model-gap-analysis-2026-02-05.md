# Unified Block Model - 规格与实现差距分析报告

**报告日期**: 2026-02-05
**分析范围**: Issue #71 - Unified Block Model 实现
**报告类型**: 深度代码审查与规格对比

---

## 执行摘要

### 核心发现

**实现显著领先于规格文档**。经过全面的代码审查，所谓的"差距"主要是文档滞后，而非代码缺失。Phase 1-5 已全部完成，Phase 6（测试）部分完成。

### 关键数据

| 指标 | 数值 |
|:-----|:-----:|
| 已完成阶段 | 5/6 (83%) |
| 规格标记准确的阶段 | 0/6 (0%) |
| 代码实现覆盖率 | ~95% |
| 真实缺失功能 | 仅前端测试 |

### 建议优先级

| 优先级 | 行动 | 时间框架 |
|:-------|:-----|:---------:|
| P0 | 更新规格文档状态标记 | 立即 |
| P1 | 创建"现状架构文档" | 本周 |
| P2 | 补充前端测试 | 本月 |

---

## 详细分析

### Phase 1: 数据库 & Store

**规格状态**: 🔲 待开发
**实际状态**: ✅ 已完成

#### 实现证据

**文件**: `store/ai_block.go` + `store/db/postgres/ai_block.go`

1. **接口定义** (`store/ai_block.go` 行 6-167):
   ```go
   type AIBlock struct {
       ID                 int64
       UID                string
       ConversationID     int32
       RoundNumber        int32
       BlockType          AIBlockType
       Mode               AIBlockMode
       UserInputs         []UserInput
       AssistantContent   string
       EventStream        []BlockEvent
       SessionStats       *SessionStats
       CCSessionID        string
       Status             AIBlockStatus
       ParentBlockID      *int64  // 分支支持
       BranchPath         string  // 分支支持
       // ...
   }
   ```

2. **数据库操作** (`store/db/postgres/ai_block.go`):
   | 方法 | 行号 | 状态 |
   |:-----|:-----|:-----:|
   | `CreateAIBlock` | 68-141 | ✅ |
   | `GetAIBlock` | 143-221 | ✅ |
   | `ListAIBlocks` | 223-279 | ✅ |
   | `UpdateAIBlock` | 281-335 | ✅ |
   | `AppendUserInput` | 337-370 | ✅ |
   | `AppendEvent` | 372-405 | ✅ |
   | `AppendEventsBatch` | 407-447 | ✅ |
   | `DeleteAIBlock` | 479-497 | ✅ |
   | `CompleteBlock` | 568-622 | ✅ |

3. **超越规格的特性**:
   - **分支支持**: `ParentBlockID` 和 `BranchPath` 字段支持树形对话结构
   - **事务完整性**: `CompleteBlock` 使用事务确保原子性
   - **批量优化**: `AppendEventsBatch` 减少数据库往返

---

### Phase 2: Proto & API

**规格状态**: 🔲 待开发
**实际状态**: ✅ 已完成

#### 实现证据

**文件**: `proto/api/v1/ai_service.proto` + `server/router/api/v1/ai_service_block.go`

1. **Proto 定义** (`ai_service.proto` 行 186-234):
   ```protobuf
   // Unified Block Model (Phase 2)
   rpc ListBlocks(ListBlocksRequest) returns (ListBlocksResponse);
   rpc GetBlock(GetBlockRequest) returns (Block);
   rpc CreateBlock(CreateBlockRequest) returns (Block);
   rpc UpdateBlock(UpdateBlockRequest) returns (Block);
   rpc DeleteBlock(DeleteBlockRequest) returns (google.protobuf.Empty);
   rpc AppendUserInput(AppendUserInputRequest) returns (google.protobuf.Empty);
   rpc AppendEvent(AppendEventRequest) returns (google.protobuf.Empty);
   ```

2. **RPC 处理器** (`ai_service_block.go`):
   | RPC | 行号 | 状态 |
   |:-----|:-----|:-----:|
   | `ListBlocks` | 18-68 | ✅ |
   | `GetBlock` | 70-92 | ✅ |
   | `CreateBlock` | 94-162 | ✅ |
   | `UpdateBlock` | 164-231 | ✅ |
   | `DeleteBlock` | 233-259 | ✅ |
   | `AppendUserInput` | 261-298 | ✅ |
   | `AppendEvent` | 300-338 | ✅ |

3. **附加特性**:
   - 完整的所有权验证（每个 RPC 都检查 conversation 所有权）
   - Proto ↔ Store 双向转换器（14 个转换函数）
   - 结构化日志记录

---

### Phase 3: 前端类型

**规格状态**: 🔲 待开发
**实际状态**: ✅ 已完成

#### 实现证据

**文件**: `web/src/types/block.ts` (206 行)

1. **类型重新导出**:
   ```typescript
   export type {
       Block, BlockEvent, BlockMode, BlockStatus, BlockType,
       UserInput, SessionStats,
       // ... 所有 Block 相关类型
   } from "./proto/api/v1/ai_service_pb";
   ```

2. **类型常量**:
   ```typescript
   export const BLOCK_TYPE = {
       MESSAGE: "BLOCK_TYPE_MESSAGE",
       CONTEXT_SEPARATOR: "BLOCK_TYPE_CONTEXT_SEPARATOR",
   } as const;

   export const BLOCK_MODE = {
       NORMAL: "BLOCK_MODE_NORMAL",
       GEEK: "BLOCK_MODE_GEEK",
       EVOLUTION: "BLOCK_MODE_EVOLUTION",
   } as const;

   export const BLOCK_STATUS = {
       PENDING: "BLOCK_STATUS_PENDING",
       STREAMING: "BLOCK_STATUS_STREAMING",
       COMPLETED: "BLOCK_STATUS_COMPLETED",
       ERROR: "BLOCK_STATUS_ERROR",
   } as const;
   ```

3. **工具函数**:
   - `isTerminalStatus()` - 检查是否为终止状态
   - `isActiveStatus()` - 检查是否为活跃状态
   - `getBlockModeName()` - 获取模式显示名称
   - `blockModeToParrotAgentType()` - BlockMode ↔ Parrot 转换

---

### Phase 4: 前端组件

**规格状态**: 🔲 待开发
**实际状态**: ✅ 功能完备

#### 实现证据

**文件**: `web/src/components/AIChat/UnifiedMessageBlock.tsx` (1263 行)

1. **组件结构**:
   ```
   UnifiedMessageBlock
   ├── BlockHeader (固定显示)
   │   ├── 用户头像 + 消息预览
   │   ├── 追加输入计数徽章
   │   ├── Geek/Evolution 成本显示
   │   └── 时间戳 + 模式徽章 + 折叠切换
   ├── BlockBody (可折叠)
   │   ├── UserInputsSection (追加用户输入)
   │   ├── ThinkingSection (思考过程)
   │   ├── ToolCallsSection (工具调用)
   │   ├── AnswerSection (Markdown 渲染)
   │   ├── ErrorSection (错误显示)
   │   └── SummarySection (会话统计)
   └── BlockFooter (固定显示)
       ├── 折叠/展开按钮
       ├── 重新生成按钮
       ├── 遗忘按钮
       └── 复制/删除按钮
   ```

2. **主题适配** (5 种 Parrot 主题):
   ```typescript
   const BLOCK_THEMES: Record<ParrotAgentType | "default", ThemeConfig> = {
     default: { border: "...", headerBg: "...", ... },
     MEMO: { border: "slate", headerBg: "slate", ... },
     SCHEDULE: { border: "cyan", headerBg: "cyan", ... },
     AMAZING: { border: "emerald", headerBg: "emerald", ... },
     GEEK: { border: "violet", headerBg: "violet", ... },
     EVOLUTION: { border: "rose", headerBg: "rose", ... },
   };
   ```

3. **超越规格的特性**:
   - **视觉宽度计算**: 中英文字符混合的精确截取
   - **时序事件流**: 按时间戳排序的事件展示
   - **流式阶段动画**: `streamingPhase` 驱动的呼吸效果
   - **Geek/Evolution 成本显示**: 实时 token/cost 统计

---

### Phase 5: Chat 集成

**规格状态**: 🔲 待开发
**实际状态**: ✅ 已集成

#### 实现证据

1. **后端集成** (`server/router/api/v1/ai/handler.go`):
   ```go
   type ParrotHandler struct {
       factory      *AgentFactory
       llm          ai.LLMService
       chatRouter   *agentpkg.ChatRouter
       persister    *aistats.Persister
       blockManager *BlockManager  // Phase 5: Unified Block Model support
   }

   func NewParrotHandler(..., blockManager *BlockManager) *ParrotHandler {
       return &ParrotHandler{
           blockManager: blockManager,  // Phase 5
       }
   }
   ```

2. **前端集成** (`web/src/hooks/useBlockQueries.ts`, 636 行):
   | Hook | 功能 | 状态 |
   |:-----|:-----|:-----:|
   | `useBlocks()` | 获取会话 Blocks | ✅ |
   | `useBlock()` | 获取单个 Block | ✅ |
   | `useCreateBlock()` | 创建 Block（乐观更新） | ✅ |
   | `useUpdateBlock()` | 更新 Block | ✅ |
   | `useDeleteBlock()` | 删除 Block | ✅ |
   | `useAppendUserInput()` | 追加用户输入 | ✅ |
   | `useAppendEvent()` | 追加流式事件 | ✅ |
   | `useAppendEventsBatch()` | 批量追加事件 | ✅ |
   | `useStreamingBlock()` | 流式状态管理 | ✅ |
   | `useBlocksWithFallback()` | 错误回退支持 | ✅ |

3. **SSE 集成** (`proto/api/v1/ai_service.proto` 行 444):
   ```protobuf
   // Phase 4: Block ID for Unified Block Model
   int64 block_id = 10; // Allows frontend to update Block state during streaming
   ```

---

### Phase 6: 测试

**规格状态**: 🔲 待开发
**实际状态**: ⚠️ 后端完成，前端缺失

#### 后端测试（已存在）

| 文件 | 覆盖内容 | 状态 |
|:-----|:---------|:-----:|
| `store/db/postgres/ai_block_test.go` | CRUD 操作测试 | ✅ |
| `server/router/api/v1/ai/block_manager_test.go` | BlockManager 测试 | ✅ |

#### 前端测试（缺失）

| 组件/Hook | 测试文件 | 状态 |
|:----------|:---------|:-----:|
| `useBlockQueries.ts` | ❌ 不存在 | 缺失 |
| `UnifiedMessageBlock.tsx` | ❌ 不存在 | 缺失 |
| E2E 测试 | ❌ 未配置 | 缺失 |

**这是唯一的真实实现差距。**

---

## 语义与架构对比

### 数据模型一致性

| 规格 | 实现 | 一致性 |
|:-----|:-----|:------:|
| Block 作为对话原子单元 | `AIBlock` 结构体 | ✅ 100% |
| UserInputs 数组 | `[]UserInput` | ✅ 100% |
| EventStream 数组 | `[]BlockEvent` | ✅ 100% |
| SessionStats | `*SessionStats` | ✅ 100% |
| Mode (normal/geek/evolution) | `AIBlockMode` | ✅ 100% |
| Status (pending/streaming/completed/error) | `AIBlockStatus` | ✅ 100% |

### 正向演进（超越规格）

| 特性 | 位置 | 描述 |
|:-----|:-----|:-----|
| **分支支持** | `store/ai_block.go:21-22` | `ParentBlockID`, `BranchPath` 支持树形对话 |
| **事务完整性** | `store/db/postgres/ai_block.go:568-622` | `CompleteBlock` 原子操作 |
| **批量优化** | `store/db/postgres/ai_block.go:407-447` | `AppendEventsBatch` |
| **错误回退** | `web/src/hooks/useBlockQueries.ts:595-635` | `useBlocksWithFallback` |
| **向后兼容** | `web/src/components/AIChat/ChatMessages.tsx` | `convertAIBlocksToMessageBlocks` |

---

## 结论

### 主要发现

1. **实现领先规格**: Phase 1-5 已全部完成，Phase 6 部分完成
2. **文档滞后**: 规格文档中的状态标记已过时，未反映实际代码状态
3. **测试差距**: 前端测试是唯一的真实缺失项
4. **正向演进**: 代码包含多个规格未提及的特性（分支、批量优化、错误回退）

### 建议

| 优先级 | 行动 | 负责方 | 时间框架 |
|:-------|:-----|:-------|:---------:|
| **P0** | 更新 `unified-block-model-index.md` 状态标记 | 文档维护 | 立即 |
| **P0** | 将已完成的 Phase 标记为 ✅ | 文档维护 | 立即 |
| **P1** | 创建"现状架构文档" | 架构师 | 本周 |
| **P1** | 记录分支/分叉功能设计意图 | 架构师 | 本周 |
| **P2** | 补充 `useBlockQueries.test.ts` | 前端开发 | 本月 |
| **P2** | 配置 Playwright E2E 测试 | 前端开发 | 本月 |

---

## 附录

### 审查方法

本次分析采用以下方法：
1. 静态代码审查 - 逐行检查实现文件
2. 规格对比 - 对照 `docs/specs/block-design/` 中的规格
3. 功能验证 - 确认 API/组件是否实际可用

### 审查范围

| 组件 | 文件数 | 代码行数 |
|:-----|:-------|:---------:|
| 后端 Store | 2 | ~800 |
| 后端 API | 1 | ~550 |
| Proto 定义 | 1 | ~140 |
| 前端类型 | 1 | ~200 |
| 前端组件 | 1 | ~1,260 |
| 前端 Hooks | 1 | ~640 |
| **总计** | **7** | **~3,590** |

---

*报告生成: Loki Mode v5.9.0*
*分析日期: 2026-02-05*
*下次审查: 前端测试完成后*
