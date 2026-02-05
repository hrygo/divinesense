# P1-A004: Unified Block Model - Phase 4 Frontend Components

> **状态**: 🔲 待开发
> **优先级**: P1 (重要)
> **投入**: 4人天
> **Sprint**: Sprint 1
> **关联 Issue**: [#71](https://github.com/hrygo/divinesense/issues/71)
> **依赖**: Phase 3 (Frontend Types)

---

## 1. 目标与背景

### 1.1 核心目标

更新前端组件以支持 Block 模型，主要改造 `ChatMessages` 和 `UnifiedMessageBlock` 组件，使其能够正确处理从后端获取的 Block 数据。

### 1.2 用户价值

- **完整的对话历史**：所有 Block 内容（包括事件流、会话统计）都能正确显示
- **追加式输入支持**：用户可以在 AI 响应完成前追加输入

### 1.3 技术价值

- **代码简化**：移除前端配对逻辑，直接使用 Block 数据
- **性能优化**：减少不必要的状态计算

---

## 2. 依赖关系

### 2.1 前置依赖（必须完成）

- [x] **Phase 3**: 前端类型定义已更新

### 2.2 并行依赖（可同步进行）

- [ ] **P1-A005**: Chat Handler 改造

### 2.3 后续依赖（依赖本 Spec）

- [ ] **P1-A006**: 集成测试

---

## 3. 功能设计

### 3.1 架构图

```
┌─────────────────────────────────────────────────────────────────┐
│  AIChat.tsx (主页面)                                            │
│  ┌─────────────────────────────────────────────────────────────┐│
│  │  useBlocks() hook → AIBlock[]                              ││
│  └─────────────────────────────────────────────────────────────┘│
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│  ChatMessages.tsx (消息列表)                                   │
│  ┌─────────────────────────────────────────────────────────────┐│
│  │  AIBlock[] → UnifiedMessageBlock[]                         ││
│  └─────────────────────────────────────────────────────────────┘│
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│  UnifiedMessageBlock.tsx (单个 Block)                          │
│  ┌─────────────────────────────────────────────────────────────┐│
│  │  AIBlock → Block Header/Body/Footer                        ││
│  └─────────────────────────────────────────────────────────────┘│
└─────────────────────────────────────────────────────────────────┘
```

### 3.2 核心流程

1. **数据加载**：`useBlocks()` hook 从后端获取 Blocks
2. **状态同步**：SSE 事件更新 Block 状态
3. **渲染**：`ChatMessages` 渲染 Block 列表
4. **交互**：用户操作（追加输入、折叠等）更新 Block

### 3.3 关键决策

| 决策点       | 方案 A         | 方案 B          | 选择  | 理由       |
| :----------- | :------------- | :-------------- | :---: | :--------- |
| **数据来源** | 保留 ChatItem  | 切换到 AIBlock  | **B** | 统一数据源 |
| **配对逻辑** | 保留前端配对   | 使用 Block 结构 | **B** | 简化代码   |
| **向后兼容** | 保留旧代码路径 | 完全替换        | **A** | 平滑迁移   |

---

## 4. 技术实现

### 4.1 ChatMessages 改造

#### 4.1.1 移除配对逻辑

```typescript
// web/src/components/AIChat/ChatMessages.tsx

import { memo, ReactNode, useCallback, useEffect, useMemo, useRef } from "react";
import TypingCursor from "@/components/AIChat/TypingCursor";
import type { AIBlock } from "@/types/block";
import type { SessionSummary } from "@/types/parrot";
import { UnifiedMessageBlock } from "./UnifiedMessageBlock";

interface ChatMessagesProps {
  blocks: AIBlock[];  // 改用 Block 数据
  isTyping?: boolean;
  currentParrotId?: ParrotAgentType;
  onCopyMessage?: (content: string) => void;
  onRegenerate?: () => void;
  onDeleteMessage?: (blockId: number) => void;
  children?: ReactNode;
  className?: string;
  amazingInsightCard?: ReactNode;
  uiTools?: GenerativeUIContainerProps["tools"];
  onUIAction?: GenerativeUIContainerProps["onAction"];
  onUIDismiss?: GenerativeUIContainerProps["onDismiss"];
  isStreaming?: boolean;
  streamingContent?: string;
  sessionSummary?: SessionSummary;
  onAppendInput?: (blockId: number, content: string) => void; // 新增
}

const ChatMessages = memo(function ChatMessages({
  blocks,
  isTyping = false,
  currentParrotId,
  onCopyMessage,
  onRegenerate,
  onDeleteMessage,
  children,
  className,
  amazingInsightCard,
  uiTools,
  onUIAction,
  onUIDismiss,
  isStreaming = false,
  streamingContent = "",
  sessionSummary,
  onAppendInput,
}: ChatMessagesProps) {
  // ... 滚动逻辑保持不变 ...

  // 计算当前流式阶段（从最后一个 Block 的状态）
  const streamingPhase = useMemo((): "thinking" | "tools" | "answer" | null => {
    const lastBlock = blocks[blocks.length - 1];
    if (!lastBlock || lastBlock.status !== BlockStatus.STREAMING) {
      return null;
    }

    // 从 event_stream 判断当前阶段
    const events = lastBlock.eventStream;
    if (events.length === 0) return "thinking";

    const lastEvent = events[events.length - 1];
    if (lastEvent.type === "tool_use") return "tools";
    if (lastEvent.type === "answer") return "answer";
    if (lastEvent.type === "thinking") return "thinking";

    return null;
  }, [blocks]);

  // 确定 Block 的 Parrot ID（从 mode 字段）
  const getBlockParrotId = (block: AIBlock): ParrotAgentType => {
    return blockModeToParrotAgentType(block.mode);
  };

  return (
    <div
      ref={scrollRef}
      onScroll={handleScrollThrottled}
      className={cn("flex-1 overflow-y-auto px-3 md:px-6 py-4 overscroll-contain", className)}
      style={{ overflowAnchor: "auto", scrollbarGutter: "stable", contain: "layout style paint" }}
    >
      {children}

      {blocks.length > 0 && (
        <div className="max-w-3xl lg:max-w-4xl xl:max-w-5xl 2xl:max-w-6xl mx-auto space-y-3">
          {blocks.map((block, index) => {
            const blockIsLast = index === blocks.length - 1;
            const blockParrotId = getBlockParrotId(block);
            const isLastStreaming = blockIsLast && isStreaming && block.status === BlockStatus.STREAMING;

            // 构建 ConversationMessage 用于向后兼容
            const userMessage: ConversationMessage = {
              id: `${block.uid}-user`,
              role: "user",
              content: block.userInputs[0]?.content || "",
              timestamp: block.userInputs[0]?.timestamp || block.createdTs,
              metadata: {
                mode: block.mode,
                blockId: block.id,
                blockUid: block.uid,
              },
            };

            const assistantMessage: ConversationMessage | undefined = block.assistantContent
              ? {
                  id: `${block.uid}-assistant`,
                  uid: block.uid,
                  role: "assistant",
                  content: block.assistantContent,
                  timestamp: block.assistantTimestamp || block.updatedTs,
                  metadata: {
                    mode: block.mode,
                    blockId: block.id,
                    blockUid: block.uid,
                    // 从 event_stream 构建元数据
                    toolCalls: block.eventStream
                      .filter((e) => e.type === "tool_use")
                      .map((e) => ({
                        name: e.meta?.tool_name || "unknown",
                        toolId: e.meta?.tool_id,
                        inputSummary: e.meta?.input_summary,
                        outputSummary: e.meta?.output_summary,
                        filePath: e.meta?.file_path,
                        duration: e.meta?.duration_ms,
                        isError: e.meta?.is_error,
                      })),
                    toolResults: block.eventStream
                      .filter((e) => e.type === "tool_result")
                      .map((e) => ({
                        name: e.meta?.tool_name || "unknown",
                        toolId: e.meta?.tool_id,
                        inputSummary: e.meta?.input_summary,
                        outputSummary: e.content,
                        duration: e.meta?.duration_ms,
                        isError: e.meta?.is_error,
                      })),
                    thinkingSteps: block.eventStream
                      .filter((e) => e.type === "thinking")
                      .map((e) => ({
                        content: e.content || "",
                        timestamp: e.timestamp,
                        round: 0,
                      })),
                  },
                }
              : undefined;

            return (
              <UnifiedMessageBlock
                key={block.uid}
                userMessage={userMessage}
                assistantMessage={assistantMessage}
                sessionSummary={blockIsLast ? sessionSummary || block.sessionStats : undefined}
                parrotId={blockParrotId}
                isLatest={blockIsLast}
                isStreaming={isLastStreaming}
                streamingPhase={blockIsLast ? streamingPhase : null}
                onCopy={onCopyMessage}
                onRegenerate={blockIsLast ? onRegenerate : undefined}
                onDelete={blockIsLast && onDeleteMessage ? () => onDeleteMessage(block.id) : undefined}
              >
                {/* Typing cursor for streaming messages */}
                {blockIsLast && isTyping && !assistantMessage?.error && (
                  <TypingCursor active={true} parrotId={blockParrotId} variant="dots" />
                )}
              </UnifiedMessageBlock>
            );
          })}
        </div>
      )}

      {/* Amazing Insight Card - rendered separately */}
      {amazingInsightCard && !isTyping && blocks.length > 0 && (
        <div className="max-w-3xl lg:max-w-4xl xl:max-w-5xl 2xl:max-w-6xl mx-auto mt-3">{amazingInsightCard}</div>
      )}

      {/* Generative UI Tools */}
      {uiTools && uiTools.length > 0 && onUIAction && onUIDismiss && (
        <div className="max-w-3xl lg:max-w-4xl xl:max-w-5xl 2xl:max-w-6xl mx-auto mt-3">
          <GenerativeUIContainer tools={uiTools} onAction={onUIAction} onDismiss={onUIDismiss} />
        </div>
      )}

      {/* Typing indicator when no blocks yet */}
      {isTyping && blocks.length === 0 && (
        <div className="flex gap-3 md:gap-4 animate-in fade-in duration-300">
          <div className="w-9 h-9 md:w-10 md:h-10 rounded-full flex items-center justify-center shadow-sm bg-muted">
            <span className="text-lg md:text-xl">🤖</span>
          </div>
          <div className={cn("px-4 py-3 rounded-2xl border shadow-sm", PARROT_THEMES.AMAZING.bubbleBg, PARROT_THEMES.AMAZING.bubbleBorder)}>
            <TypingCursor active={true} parrotId={currentParrotId || ParrotAgentType.AMAZING} variant="dots" />
          </div>
        </div>
      )}

      {/* Scroll anchor */}
      <div ref={endRef} className="h-px" />
    </div>
  );
});

export { ChatMessages };
```

### 4.2 AIChat Context 扩展

```typescript
// web/src/contexts/AIChatContext.tsx

import type { AIBlock, BlockStatus } from "@/types/block";

export const AIChatContext = createContext<AIChatContextValue | undefined>(undefined);

export function AIChatProvider({ children }: { children: ReactNode }) {
  // ... existing state ...

  // Block state (Phase 4)
  const [blocksMap, setBlocksMap] = useState<Map<number, AIBlock>>(new Map());

  // ... existing methods ...

  // Block methods (Phase 4)
  const loadBlocks = useCallback(async (conversationId: string): Promise<AIBlock[]> => {
    const id = parseInt(conversationId);
    if (isNaN(id)) return [];

    const response = await fetch(`/api/v1/ai/conversations/${id}/blocks`);
    if (!response.ok) throw new Error("Failed to load blocks");

    const data = await response.json();
    const blocks: AIBlock[] = data.blocks || [];

    // Update blocks map
    const newMap = new Map<number, AIBlock>();
    blocks.forEach((block: AIBlock) => {
      newMap.set(block.id, block);
    });
    setBlocksMap(newMap);

    return blocks;
  }, []);

  const appendUserInput = useCallback(async (blockId: number, content: string) => {
    const response = await fetch(`/api/v1/ai/blocks/${blockId}/input`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ content }),
    });

    if (!response.ok) throw new Error("Failed to append user input");

    const updatedBlock: AIBlock = await response.json();

    // Update blocks map
    setBlocksMap((prev) => {
      const newMap = new Map(prev);
      newMap.set(blockId, updatedBlock);
      return newMap;
    });

    // Update conversation messages
    updateMessage(conversationId, `${updatedBlock.uid}-user`, {
      content: updatedBlock.userInputs[updatedBlock.userInputs.length - 1].content,
    });
  }, [conversationId, updateMessage]);

  const updateBlockStatus = useCallback(async (blockId: number, status: BlockStatus) => {
    const response = await fetch(`/api/v1/ai/blocks/${blockId}/status`, {
      method: "PATCH",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ status }),
    });

    if (!response.ok) throw new Error("Failed to update block status");

    const updatedBlock: AIBlock = await response.json();

    // Update blocks map
    setBlocksMap((prev) => {
      const newMap = new Map(prev);
      newMap.set(blockId, updatedBlock);
      return newMap;
    });
  }, []);

  // Get blocks for current conversation
  const currentBlocks = useMemo(() => {
    if (!currentConversationId) return [];
    return Array.from(blocksMap.values())
      .filter((b) => b.conversationId === parseInt(currentConversationId))
      .sort((a, b) => a.roundNumber - b.roundNumber);
  }, [blocksMap, currentConversationId]);

  // Update context value
  const value: AIChatContextValue = useMemo(
    () => ({
      // ... existing values ...
      currentBlocks,
      loadBlocks,
      appendUserInput,
      updateBlockStatus,
    }),
    [
      // ... existing dependencies ...
      currentBlocks,
      loadBlocks,
      appendUserInput,
      updateBlockStatus,
    ],
  );

  return <AIChatContext.Provider value={value}>{children}</AIChatContext.Provider>;
}
```

### 4.3 SSE 事件处理扩展

```typescript
// web/src/hooks/grpc/useAIChatStream.ts (扩展)

// 处理 Block 相关事件
if (event.block_id !== undefined) {
  const blockId = event.block_id;
  const blockUid = event.block_uid;

  // 更新 Block 状态
  if (event.block_status !== undefined) {
    updateBlockStatus(blockId, convertProtoBlockStatus(event.block_status));
  }

  // 追加事件到 event_stream
  if (event.event_type !== undefined) {
    // 在前端维护 event_stream（用于显示）
    // 实际数据来自后端
  }
}
```

### 4.4 关键代码路径

| 文件路径                                     | 职责                |
| :------------------------------------------- | :------------------ |
| `web/src/components/AIChat/ChatMessages.tsx` | 改用 Block 数据渲染 |
| `web/src/contexts/AIChatContext.tsx`         | 添加 Block 状态管理 |
| `web/src/hooks/grpc/useAIChatStream.ts`      | 扩展 SSE 事件处理   |
| `web/src/hooks/grpc/useAIBlocks.ts`          | Block API hooks     |

---

## 5. 交付物清单

### 5.1 代码文件

- [ ] `web/src/components/AIChat/ChatMessages.tsx` - 改用 Block 数据
- [ ] `web/src/contexts/AIChatContext.tsx` - 添加 Block 方法
- [ ] `web/src/hooks/grpc/useAIChatStream.ts` - 扩展 SSE 处理

### 5.2 数据库变更

无

### 5.3 配置变更

- [ ] `web/src/locales/en.json` - 添加 Block 相关翻译
- [ ] `web/src/locales/zh-Hans.json` - 添加 Block 相关翻译

### 5.4 文档更新

- [ ] `docs/dev-guides/FRONTEND.md` - 更新组件说明

---

## 6. 测试验收

### 6.1 功能测试

| 场景             | 输入                       | 预期输出              |
| :--------------- | :------------------------- | :-------------------- |
| **加载 Blocks**  | 打开会话                   | 显示完整的 Block 列表 |
| **追加用户输入** | 在 Block 完成前输入        | 追加到当前 Block      |
| **状态更新**     | SSE 事件到达               | Block 状态实时更新    |
| **折叠/展开**    | 点击 Block Header          | Block 内容折叠/展开   |
| **多模式 Block** | 混合 normal/geek/evolution | 每种模式显示正确主题  |

### 6.2 性能验收

| 指标            | 目标值  | 测试方法 |
| :-------------- | :------ | :------- |
| 渲染 100 Blocks | < 100ms | 性能测试 |
| 追加输入延迟    | < 50ms  | 网络测试 |

### 6.3 集成验收

- [ ] 与 Phase 3 类型定义兼容
- [ ] 与后端 Block API 集成成功
- [ ] 现有功能不受影响

---

## 7. ROI 分析

| 维度     | 值                             |
| :------- | :----------------------------- |
| 开发投入 | 4人天                          |
| 预期收益 | 简化前端代码，支持完整对话历史 |
| 风险评估 | 中（涉及核心组件改造）         |
| 回报周期 | 1 Sprint                       |

---

## 8. 风险与缓解

| 风险             | 概率  | 影响 | 缓解措施                   |
| :--------------- | :---: | :--- | :------------------------- |
| **向后兼容破坏** |  中   | 高   | 保留旧代码路径，渐进式迁移 |
| **性能下降**     |  低   | 中   | 使用 useMemo 优化计算      |
| **状态同步问题** |  中   | 中   | 使用 uid 作为稳定 key      |

---

## 9. 实施计划

### 9.1 时间表

| 阶段      | 时间  | 任务               |
| :-------- | :---- | :----------------- |
| **Day 1** | 1人天 | ChatMessages 改造  |
| **Day 2** | 1人天 | AIChatContext 扩展 |
| **Day 3** | 1人天 | SSE 事件处理扩展   |
| **Day 4** | 1人天 | 集成测试，问题修复 |

### 9.2 检查点

- [ ] Checkpoint 1: 单元测试通过
- [ ] Checkpoint 2: 手动测试通过
- [ ] Checkpoint 3: 现有功能回归测试通过

---

## 附录

### A. 参考资料

- [Phase 3 Spec](./unified-block-model-phase3.md)
- [前端开发指南](../../dev-guides/FRONTEND.md)

### B. 变更记录

| 日期       | 版本 | 变更内容 | 作者   |
| :--------- | :--- | :------- | :----- |
| 2026-02-04 | v1.0 | 初始版本 | Claude |
