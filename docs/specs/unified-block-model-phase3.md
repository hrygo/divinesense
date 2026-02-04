# P1-A003: Unified Block Model - Phase 3 Frontend Types

> **状态**: 🔲 待开发
> **优先级**: P1 (重要)
> **投入**: 2人天
> **Sprint**: Sprint 1
> **关联 Issue**: [#71](https://github.com/hrygo/divinesense/issues/71)
> **依赖**: Phase 2 (Proto & API)

---

## 1. 目标与背景

### 1.1 核心目标

更新前端类型定义，使其与 Block 模型保持一致，支持完整的数据结构。

### 1.2 用户价值

- **类型安全**：前端代码有完整的类型检查
- **开发体验**：IDE 自动补全，减少错误

### 1.3 技术价值

- **前后端一致**：类型定义与 Proto 保持同步
- **重构安全**：类型检查保证重构安全

---

## 2. 依赖关系

### 2.1 前置依赖（必须完成）

- [x] **Phase 2**: Proto 定义已完成

### 2.2 并行依赖（可同步进行）

- [ ] **P1-A004**: 前端组件改造

### 2.3 后续依赖（依赖本 Spec）

- [ ] **P1-A005**: Chat Handler 改造

---

## 3. 功能设计

### 3.1 类型映射

```
Proto (Go)              →  TypeScript
----------------------------------------
AIBlock                 →  Block
BlockType               →  BlockType
BlockMode               →  BlockMode
BlockStatus             →  BlockStatus
UserInput               →  BlockUserInput
BlockEvent              →  BlockEvent
SessionSummary          →  (已存在，扩展)
```

### 3.2 核心流程

1. **生成 TypeScript 类型**：从 Proto 生成基础类型
2. **扩展类型定义**：添加前端特定字段
3. **更新 Context 类型**：扩展 AIChatContextValue

### 3.3 关键决策

| 决策点 | 方案 A | 方案 B | 选择 | 理由 |
|:---|:---|:---|:---:|:---|
| **类型来源** | 手写 | 从 Proto 生成 | **B** | 与后端保持一致 |
| **扩展方式** | 继承 | 交叉类型 | **B** | TypeScript 交叉类型更灵活 |

---

## 4. 技术实现

### 4.1 类型定义

#### 4.1.1 Block 类型

```typescript
// web/src/types/block.ts

/**
 * Block type enumeration
 */
export enum BlockType {
  MESSAGE = "MESSAGE",
  CONTEXT_SEPARATOR = "CONTEXT_SEPARATOR",
}

/**
 * Block mode enumeration
 */
export enum BlockMode {
  NORMAL = "normal",
  GEEK = "geek",
  EVOLUTION = "evolution",
}

/**
 * Block status enumeration
 */
export enum BlockStatus {
  PENDING = "pending",
  STREAMING = "streaming",
  COMPLETED = "completed",
  ERROR = "error",
}

/**
 * User input in a block
 */
export interface BlockUserInput {
  content: string;
  timestamp: number;
  metadata?: Record<string, string>;
}

/**
 * Event in the event stream
 */
export interface BlockEvent {
  type: "thinking" | "tool_use" | "tool_result" | "answer" | "error";
  content?: string;
  timestamp: number;
  meta?: {
    // Tool call metadata
    tool_name?: string;
    tool_id?: string;
    input_summary?: string;
    output_summary?: string;
    file_path?: string;
    duration_ms?: number;
    is_error?: boolean;
    // Token usage
    input_tokens?: number;
    output_tokens?: number;
    cache_write_tokens?: number;
    cache_read_tokens?: number;
  };
}

/**
 * AIBlock - Conversation block (round)
 * This is the frontend representation of the backend AIBlock
 */
export interface AIBlock {
  id: number;
  uid: string;
  conversationId: number;
  roundNumber: number;

  blockType: BlockType;
  mode: BlockMode;

  userInputs: BlockUserInput[];
  assistantContent?: string;
  assistantTimestamp?: number;

  eventStream: BlockEvent[];
  sessionStats?: SessionSummary;

  ccSessionId?: string;
  status: BlockStatus;

  metadata?: Record<string, string>;

  createdTs: number;
  updatedTs: number;

  // Frontend-specific fields (not from backend)
  isLatest?: boolean;       // Whether this is the latest block in conversation
  isStreaming?: boolean;    // Whether this block is currently streaming
  streamingPhase?: "thinking" | "tools" | "answer" | null; // Current streaming phase
}

/**
 * Block summary for sidebar/list view
 */
export interface BlockSummary {
  id: number;
  uid: string;
  roundNumber: number;
  mode: BlockMode;
  userPreview: string;      // First user input preview
  status: BlockStatus;
  updatedTs: number;
}
```

#### 4.1.2 扩展现有类型

```typescript
// web/src/types/aichat.ts (扩展)

/**
 * Extend ConversationMessage metadata to support Block mode
 */
export interface ConversationMessage {
  // ... existing fields ...
  metadata?: {
    // ... existing fields ...
    mode?: AIMode; // 消息生成时的 AI 模式
    blockId?: number; // 关联的 Block ID (Phase 3)
    blockUid?: string; // Block UID for sync (Phase 3)
  };
}

/**
 * Extend Conversation to support Blocks
 */
export interface Conversation {
  // ... existing fields ...
  blocks?: AIBlock[]; // Block 视图的数据（Phase 3+）
  blockCount?: number; // Block 总数
  latestBlockId?: number;
  latestBlockUid?: string;
}

/**
 * Extend AIChatContextValue to support Block operations
 */
export interface AIChatContextValue {
  // ... existing methods ...

  // Block actions (Phase 3)
  appendUserInput: (blockId: number, content: string) => Promise<void>;
  updateBlockStatus: (blockId: number, status: BlockStatus) => Promise<void>;
  loadBlocks: (conversationId: string) => Promise<AIBlock[]>;
}
```

#### 4.1.3 更新 Parrot 类型

```typescript
// web/src/types/parrot.ts (扩展)

/**
 * Map BlockMode to ParrotAgentType
 */
export function blockModeToParrotAgentType(mode: BlockMode): ParrotAgentType {
  switch (mode) {
    case BlockMode.NORMAL:
      return ParrotAgentType.AMAZING;
    case BlockMode.GEEK:
      return ParrotAgentType.GEEK;
    case BlockMode.EVOLUTION:
      return ParrotAgentType.EVOLUTION;
    default:
      return ParrotAgentType.AMAZING;
  }
}

/**
 * Map ParrotAgentType to BlockMode
 */
export function parrotAgentTypeToBlockMode(agentType: ParrotAgentType): BlockMode {
  switch (agentType) {
    case ParrotAgentType.GEEK:
      return BlockMode.GEEK;
    case ParrotAgentType.EVOLUTION:
      return BlockMode.EVOLUTION;
    default:
      return BlockMode.NORMAL;
  }
}
```

### 4.2 API 客户端更新

```typescript
// web/src/hooks/grpc/useAIBlocks.ts (新文件)

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { aiService } from "@/gen/grpc/v1/ai_service_connect";
import type {
  AIBlock,
  BlockStatus,
  BlockUserInput,
} from "@/types/block";

/**
 * Fetch blocks for a conversation
 */
export function useBlocks(conversationId: number, status?: BlockStatus) {
  return useQuery({
    queryKey: ["blocks", conversationId, status],
    queryFn: async () => {
      const res = await aiService.listBlocks({
        conversationId,
        status: status ?? undefined,
      });
      return res.blocks;
    },
    enabled: conversationId > 0,
  });
}

/**
 * Fetch a single block
 */
export function useBlock(blockId: number) {
  return useQuery({
    queryKey: ["block", blockId],
    queryFn: async () => {
      const res = await aiService.getBlock({ id: blockId });
      return res;
    },
    enabled: blockId > 0,
  });
}

/**
 * Append user input to a block
 */
export function useAppendUserInput() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async ({ blockId, content }: { blockId: number; content: string }) => {
      const res = await aiService.appendUserInput({
        id: blockId,
        content,
      });
      return res;
    },
    onSuccess: (data, variables) => {
      // Invalidate related queries
      queryClient.invalidateQueries({ queryKey: ["blocks"] });
      queryClient.invalidateQueries({ queryKey: ["block", variables.blockId] });
    },
  });
}

/**
 * Update block status
 */
export function useUpdateBlockStatus() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async ({ blockId, status }: { blockId: number; status: BlockStatus }) => {
      const res = await aiService.updateBlockStatus({
        id: blockId,
        status,
      });
      return res;
    },
    onSuccess: (data, variables) => {
      // Invalidate related queries
      queryClient.invalidateQueries({ queryKey: ["blocks"] });
      queryClient.invalidateQueries({ queryKey: ["block", variables.blockId] });
    },
  });
}
```

### 4.3 关键代码路径

| 文件路径 | 职责 |
|:---|:---|
| `web/src/types/block.ts` | Block 类型定义（新文件） |
| `web/src/types/aichat.ts` | 扩展现有类型 |
| `web/src/types/parrot.ts` | 添加 Block-Parrot 映射函数 |
| `web/src/hooks/grpc/useAIBlocks.ts` | Block API hooks（新文件） |
| `web/src/hooks/grpc/index.ts` | 导出新增 hooks |

---

## 5. 交付物清单

### 5.1 代码文件

- [ ] `web/src/types/block.ts` - Block 类型定义（新文件）
- [ ] `web/src/types/aichat.ts` - 扩展现有类型
- [ ] `web/src/types/parrot.ts` - 添加映射函数
- [ ] `web/src/hooks/grpc/useAIBlocks.ts` - Block API hooks（新文件）
- [ ] `web/src/hooks/grpc/index.ts` - 导出新增 hooks

### 5.2 数据库变更

无

### 5.3 配置变更

无

### 5.4 文档更新

- [ ] `docs/dev-guides/FRONTEND.md` - 更新类型定义说明

---

## 6. 测试验收

### 6.1 功能测试

| 场景 | 输入 | 预期输出 |
|:---|:---|:---|
| **类型检查** | pnpm type-check | 无类型错误 |
| **Block 构造** | new AIBlock({...}) | 类型正确 |
| **枚举转换** | blockModeToParrotAgentType(BlockMode.GEEK) | 返回 ParrotAgentType.GEEK |
| **Hook 返回值** | useBlocks(1) | 返回类型为 UseQueryResult<AIBlock[]> |

### 6.2 性能验收

| 指标 | 目标值 | 测试方法 |
|:---|:---|:---|
| 类型检查时间 | < 10s | pnpm type-check |
| 构建时间增加 | < 5% | pnpm build |

### 6.3 集成验收

- [ ] 与 Proto 生成类型兼容
- [ ] 现有组件类型检查通过
- [ ] 新增 Hooks 功能测试通过

---

## 7. ROI 分析

| 维度 | 值 |
|:---|:---|
| 开发投入 | 2人天 |
| 预期收益 | 前端类型安全，减少运行时错误 |
| 风险评估 | 低（纯新增，不破坏现有） |
| 回报周期 | 1 Sprint |

---

## 8. 风险与缓解

| 风险 | 概率 | 影响 | 缓解措施 |
|:---|:---:|:---|:---|
| 类型冲突 | 低 | 中 | 使用交叉类型避免冲突 |
| Proto 变更 | 低 | 低 | 锁定 Proto 版本 |

---

## 9. 实施计划

### 9.1 时间表

| 阶段 | 时间 | 任务 |
|:---|:---|:---|
| **Day 1** | 1人天 | 创建 Block 类型定义 |
| **Day 2** | 1人天 | 创建 Hooks，类型检查 |

### 9.2 检查点

- [ ] Checkpoint 1: pnpm type-check 通过
- [ ] Checkpoint 2: 现有组件类型检查通过

---

## 附录

### A. 参考资料

- [Phase 2 Spec](./unified-block-model-phase2.md)
- [前端开发指南](../dev-guides/FRONTEND.md)

### B. 变更记录

| 日期 | 版本 | 变更内容 | 作者 |
|:---|:---|:---|:---|
| 2026-02-04 | v1.0 | 初始版本 | Claude |
