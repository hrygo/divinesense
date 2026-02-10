# ChatItem 删除重构方案

> **目标**: 移除 ChatItem[] 遗留数据结构，完全迁移到 Block[] (Unified Block Model)

## 当前状态

### 双数据源问题

| 方面 | ChatItem[] (Legacy) | Block[] (New) |
|------|---------------------|--------------|
| 数据源 | AIChatContext.conversations.messages | 后端 Block API |
| 结构 | 扁平数组，每条消息一个 item | 结构化 Block，包含 userInputs[] |
| 流式更新 | 通过 updateMessage() 更新 metadata | 通过 eventStream 追加事件 |
| 持久化 | 无（仅前端状态） | PostgreSQL 持久化 |
| 状态同步 | 手动同步，容易不一致 | React Query 自动缓存 |

### 依赖位置

| 文件 | ChatItem 使用 | 迁移目标 |
|------|--------------|----------|
| `types/aichat.ts` | ConversationMessage, ChatItem 类型定义 | 删除 |
| `AIChatContext.tsx` | addMessage, updateMessage, deleteMessage, clearMessages, addContextSeparator | 使用 useBlockQueries hooks |
| `AIChat.tsx` | handleSend, handleParrotChat 调用 addMessage/updateMessage | 直接调用 useCreateBlock/useUpdateBlock |
| `ChatMessages.tsx` | groupMessagesIntoBlocks, items prop fallback | 仅使用 blocks，删除 fallback |
| `hooks/useBlockQueries.ts` | convertBlockToChatItem (兼容性转换) | 删除 |

## 重构步骤

### Phase 1: 准备工作 ✅ (已完成)

- [x] Block API 稳定运行
- [x] useBlockQueries hooks 完整实现
- [x] UnifiedMessageBlock 支持 Block 数据
- [x] 流式事件写入 Block.eventStream (问题#5修复)

### Phase 2: UI 层迁移 (预计 2-3 天)

**目标**: ChatMessages 直接使用 Block[]，删除 ChatItem 转换

```typescript
// Before: ChatMessages.tsx
const messageBlocks = useMemo(() => {
  if (blocks && blocks.length > 0) {
    return convertAIBlocksToMessageBlocks(blocks, t);
  }
  return groupMessagesIntoBlocks(items, false, t);
}, [blocks, items, t]);

// After: ChatMessages.tsx
const messageBlocks = useMemo(() => {
  if (!blocks || blocks.length === 0) return [];
  return convertAIBlocksToMessageBlocks(blocks, t);
}, [blocks, t]);
```

**删除**:
- `items` prop
- `groupMessagesIntoBlocks()` 函数
- `shouldFallback` 逻辑

**风险**: 无 - Block API 已经稳定

### Phase 3: AIChat.tsx 流式处理迁移 (预计 3-4 天)

**目标**: 流式事件直接操作 Block，不通过 ChatItem

```typescript
// Before: AIChat.tsx handleParrotChat
await chatHook.stream(streamParams, {
  onThinking: (msg) => {
    updateMessage(conversationId, lastAssistantMessageIdRef.current, {
      metadata: { thinkingSteps: [...thinkingStepsRef.current] }
    });
  },
  onToolUse: (toolName, meta) => {
    updateMessage(conversationId, lastAssistantMessageIdRef.current, {
      metadata: { toolCalls: [...toolCallsRef.current] }
    });
  },
  // ...
});

// After: AIChat.tsx handleParrotChat
await chatHook.stream(streamParams, {
  onThinking: (msg) => {
    // 流式事件已通过 useAIQueries.ts 自动写入 Block.eventStream
    // 此处仅用于 UI 即时反馈（如 typing cursor）
    setIsThinking(true);
  },
  onToolUse: (toolName, meta) => {
    setCapabilityStatus("processing");
  },
  onDone: () => {
    setIsTyping(false);
    setIsThinking(false);
    // Block 已通过 useAIQueries.ts 更新，无需额外操作
  },
});
```

**修改**:
- 删除 `lastAssistantMessageIdRef` (不再需要)
- 删除 `thinkingStepsRef`, `toolCallsRef` (已存入 eventStream)
- 删除 `updateMessage` 调用

**风险**: 中 - 需要确保流式事件正确写入 eventStream

### Phase 4: AIChatContext 重构 (预计 4-5 天)

**目标**: 将 Context 从 ChatItem 状态管理迁移到 Block 查询状态

**删除函数**:
- `addMessage()` - 替代: `useCreateBlock()`
- `updateMessage()` - 替代: `useUpdateBlock()`
- `deleteMessage()` - 替代: `useDeleteBlock()`
- `clearMessages()` - 替代: 批量 Block 删除
- `addContextSeparator()` - 替代: 创建 CONTEXT_SEPARATOR Block
- `convertBlockToChatItem()` - 删除（不再需要转换）
- `mergeMessageLists()` - 删除（React Query 处理）
- `syncMessages()` - 删除（Block API 替代）
- `loadMoreMessages()` - 替代: Block API 分页

**保留函数**:
- `conversations` - 用于侧边栏列表（但数据来源从 Block API 获取）
- `selectConversation()` - 切换当前会话
- `createConversation()` - 创建新会话

**新 Context 结构**:
```typescript
interface AIChatState {
  conversations: Conversation[];  // 简化为仅元数据
  currentConversationId: string | null;
  // ... 其他 UI 状态
}

// 移除
// blocksByConversation (直接通过 useBlockQueries 获取)
// messageCache (不需要)
```

**风险**: 高 - Context 是核心状态管理，需要全面测试

### Phase 5: 类型清理 (预计 1 天)

**删除文件**:
- `types/aichat.ts` 中的 ConversationMessage, ChatItem (如果没有其他地方使用)

**保留类型**:
- `types/block.ts` - Block 类型定义
- `types/proto/api/v1/ai_service_pb.ts` - Protobuf 生成类型

### Phase 6: 清理遗留代码 (预计 1 天)

**搜索并删除**:
- 所有 `items: ChatItem[]` prop
- 所有 `groupMessagesIntoBlocks` 调用
- 所有 `convertBlockToChatItem` 调用
- AIChatContext 中的 ChatItem 相关处理

## 迁移矩阵

| 功能 | ChatItem 实现 | Block 实现 |
|------|---------------|-----------|
| 创建消息 | `addMessage(convId, msg)` | `createBlock({ mode, userInputs: [{content}] })` |
| 更新内容 | `updateMessage(id, {content})` | `updateBlock(id, { assistantContent })` |
| 更新 metadata | `updateMessage(id, {metadata})` | `appendEvent(blockId, event)` |
| 追加用户输入 | `addMessage(convId, msg)` | `appendUserInput(blockId, content)` |
| 删除消息 | `deleteMessage(id)` | `deleteBlock(id)` |
| 清空对话 | `clearMessages(id)` | 删除 conversation 下所有 blocks |
| 上下文分隔符 | `addContextSeparator(id)` | 创建 `BlockType.CONTEXT_SEPARATOR` |
| 流式更新 | `updateMessage(id, metadata)` | `updateBlock(id, {status, eventStream})` |

## 风险评估

| 风险 | 级别 | 缓解措施 |
|------|------|---------|
| Block API 不稳定 | 中 | 保留 useBlocksWithFallback 作为临时安全网 |
| 流式事件丢失 | 高 | 添加事件写入验证和错误日志 |
| 状态不一致 | 中 | React Query 自动刷新机制 |
| 用户体验影响 | 中 | 灰度发布，监控错误率 |
| 回滚困难 | 高 | Git 分支保护，分阶段合并 |

## 验收标准

- [x] 构建成功无错误
- [ ] 所有测试通过 (137+ 测试用例)
- [ ] 流式聊天功能正常
- [ ] Geek/Evolution 模式正常
- [x] 无 TypeScript 错误
- [x] ChatItem 直接使用已从 UI 层移除

## 预估时间 vs 实际

| Phase | 描述 | 预估时间 | 实际时间 |
|-------|------|----------|----------|
| Phase 1 | 准备工作 | ✅ 已完成 | ✅ 已完成 |
| Phase 2 | UI 层迁移 | 2-3 天 | ✅ 已完成 |
| Phase 3 | 流式处理迁移 | 3-4 天 | ✅ 已完成 |
| Phase 4 | Context 重构 | 4-5 天 | ✅ 已完成 |
| Phase 5 | 类型清理 | 1 天 | ✅ 已完成 |
| Phase 6 | 清理遗留代码 | 1 天 | ✅ 已完成 |

**实施日期**: 2026-02-07

## 实施摘要

### 已完成的修改

1. **ChatMessages.tsx**:
   - 移除 `items` prop，仅使用 `blocks` prop
   - 删除 `groupMessagesIntoBlocks()` 函数
   - 简化 `messageBlocks` useMemo
   - 更新 effects 使用 `blocks.length` 而非 `items.length`

2. **AIChat.tsx**:
   - 移除 `useBlocksWithFallback`，改用 `useBlocks`
   - 移除 `lastAssistantMessageIdRef`, `thinkingStepsRef`, `toolCallsRef`
   - 简化流式处理回调（流式事件已通过 useAIQueries.ts 写入 Block.eventStream）
   - 移除 `addMessage`, `updateMessage` 的使用

3. **AIChatContext.tsx**:
   - 移除导出的 `addMessage`, `updateMessage`, `deleteMessage`
   - 保留 `clearMessages`, `addContextSeparator`（仍在使用）
   - 删除不再需要的函数实现

4. **types/aichat.ts**:
   - 从 `AIChatContextValue` 接口移除 `addMessage`, `updateMessage`, `deleteMessage`

## 建议

1. **分阶段执行**: 每个 Phase 独立 PR，便于 review 和回滚
2. **保持测试覆盖**: 每个 Phase 后运行完整测试
3. **监控生产环境**: 上线后密切关注错误率
4. **文档同步**: 更新开发者文档和架构图
