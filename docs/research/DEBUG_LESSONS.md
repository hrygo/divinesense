# 调试经验教训

> 记录 DivineSense 开发过程中遇到的典型问题和解决方案，避免重复踩坑。
>
> **保鲜状态**: ✅ 2026-02-17

---

## 快速查找

| 类型 | 问题 | 状态 |
|:-----|:-----|:-----|
| **[前端](#前端问题)** | [布局宽度不统一](#前端布局宽度不统一) | ✅ 已解决 |
| **[前端](#前端问题)** | [空白页面滚动条溢出](#空白页面滚动条溢出) | ✅ 已解决 |
| **[前端](#前端问题)** | [快捷指令硬编码未 i18n](#快捷指令硬编码未-i18n) | ✅ 已解决 |
| **[前端](#前端问题)** | [流式渲染事件缺失](#流式渲染事件缺失) | ✅ 已解决 |
| **[前端](#前端问题)** | [工具调用元数据错误](#工具调用元数据错误) | ✅ 已解决 |
| **[前端](#前端问题)** | [过度工程：FIFO 队列](#过度工程fifo-队列) | ✅ 已解决 |
| **[后端](#后端问题)** | [Go embed 忽略下划线文件](#go-embed-忽略以下划线开头的文件) | ✅ 已解决 |
| **[后端](#后端问题)** | [调试日志管理规范](#调试日志管理规范) | ⚠️ 规范 |
| **[AI](#ai-问题)** | [Evolution Mode 路由失败](#evolution-mode-路由失败) | ✅ 已解决 |
| **[AI](#ai-问题)** | [AI Token 统计与缓存指标](#ai-token-统计与缓存指标) | ✅ 已解决 |
| **[AI](#ai-问题)** | [Orchestrator 模式多轮对话问题](#orchestrator-模式多轮对话问题) | ✅ 已解决 |
| **[部署](#部署问题)** | [二进制部署运维权限](#二进制部署运维权限问题) | ✅ 已解决 |
| **[开发流程](#开发流程问题)** | [环境意识不足](#环境意识不足导致的重复错误) | ✅ 已解决 |

---

## 前端问题

### 前端布局宽度不统一

**问题**：不同页面在大屏幕上的最大宽度不一致。

**根本原因**：
1. 布局层级混乱：Layout 层和 Page 层都设置了 `max-w-*`
2. 组件内部限制：`MasonryColumn` 组件内部有 `max-w-2xl` 限制
3. Tailwind v4 的 `max-w-md/lg/xl` 解析为 ~16px

**解决方案**：
```tsx
max-w-[100rem]  // 1600px
mx-auto
px-4 sm:px-6
```

---

### 空白页面滚动条溢出

**问题**：AI 聊天页面在空白状态（无消息）时仍显示滚动条。

**根本原因**：双重 padding + h-full 组合导致高度溢出

**解决方案**：
```tsx
// ChatMessages.tsx
style={{ scrollbarGutter: "auto" }}

// PartnerGreeting.tsx
className="... min-h-0 w-full px-6 py-8"
```

---

### 快捷指令硬编码未 i18n

**问题**：AI 聊天快捷指令点击后填充到输入框的内容是英文硬编码。

**根本原因**：`quickReplyAnalyzer.ts` 中的 `payload` 字段使用英文硬编码字符串。

**解决方案**：
1. `quickReplyAnalyzer.ts` - 所有 `payload` 改为 i18n key 格式
2. `QuickReplies.tsx` - `hint` 支持 i18n 翻译
3. `AIChat.tsx` - `handleQuickReply` 翻译 i18n key 类型的 payload

```typescript
// ✅ 正确
const handleQuickReply = useCallback((text: string) => {
  const translatedText = text.startsWith("ai.quick_replies.payload_") || text.startsWith("ai.")
    ? t(text)
    : text;
  setInput(translatedText);
}, [setInput, t]);
```

---

### 流式渲染事件缺失

**问题**：AI 聊天过程中，用户发送问题后 block 创建，但只展示用户输入，长时间卡顿后直接展示结果，缺少 `ToolCallsSection` 和 `ThinkingSection` 等过程展示。

**根本原因**：前端 `useAIQueries.ts` 中的事件处理条件判断错误。

```typescript
// ❌ 错误：要求 eventData 非空才处理事件
if (response.eventType && response.eventData) {
```

问题：`thinking` 事件发送 `""`（空字符串），`tool_use` 的 `EventData` 可能为空，但关键信息在 `eventMeta` 中。

**解决方案**：
```typescript
// ✅ 正确：只检查 eventType
if (response.eventType) {
```

---

### 工具调用元数据错误

**问题**：工具调用状态显示的工具名称是 JSON 参数字符串，而非实际工具名称。

**现象**：
- 预期：`toolName: "schedule_query"`
- 实际：`toolName: '{"start_time": "2026-02-09T00:00:00+08:00", "end_time": "2026-02-10T00:00:00+08:00"}'`

**根本原因**：`useAIQueries.ts` 中 `tool_use` 事件处理使用了错误的数据源：
```typescript
// ❌ 错误：response.eventData 是工具的 JSON 参数
updateBlockEventStream({
  toolName: response.eventData,
  // ...
});
```

**解决方案**：使用从 `eventMeta` 提取的工具名称：
```typescript
// ✅ 正确：toolMeta.toolName 是实际的工具名称
const toolMeta = response.eventMeta ? { toolName: response.eventMeta.toolName, ... } : undefined;
updateBlockEventStream({
  toolName: toolMeta?.toolName,
  // ...
});
```

---

### 过度工程：FIFO 队列

**问题**：引入 ~250 行 FIFO 队列代码来"防止竞态条件"，但实际上队列始终为空。

**根本原因**：
1. **SSE 是串行的** —— Server-Sent Events 本身是单线程流式传输
2. **JavaScript 单线程** —— 在 `queueMicrotask` 调度前，队列已被处理
3. **React Query 同步更新** —— `setQueryData` 是同步的，不需要排队

**证据**：日志显示每次 `queueLength: 0`
```
[BlockUpdateQueue] Enqueue: {queueLength: 0, processing: false}
```

**教训**：
- **先诊断后治疗** —— 添加日志确认队列确实有积压再引入复杂方案
- **SSE 不需要排队** —— 流式事件天然串行，不存在并发竞态
- **简化优于复杂** —— React Query 的 `setQueryData` 已经有内部批处理优化

**解决方案**：移除 FIFO 队列，直接使用 `queryClient.setQueryData()`。

**真正需要的是**：
- `blockAlreadyExists` 检查 —— 防止每次事件都重新创建 block

---

## 后端问题

### Go embed 忽略以下划线开头的文件

**问题**：部署到生产环境后，部分 JavaScript 文件无法加载。

**根本原因**：Go 的 `//go:embed` 指令会忽略**以下划线 `_` 开头的文件**

**解决方案**：修改 Vite 配置，将 lodash-es 模块打包到单个 chunk
```typescript
// vite.config.mts
manualChunks(id) {
  if (id.includes("lodash-es") || id.includes("/_base")) {
    return "lodash-vendor";
  }
}
```

---

### 调试日志管理规范

**前端日志**：
```typescript
// ✅ 正确
if (import.meta.env.DEV) {
  console.debug("[Component] Debug info", data);
}
console.error("[Component] Error:", error);

// ❌ 错误
console.log("[Component] Some info");
```

**后端日志**：
```go
// ✅ 正确
slog.Info("AI chat started", "agent_type", req.AgentType)

// ❌ 错误
slog.Debug("Every single step", ...)
```

---

## AI 问题

### Evolution Mode 路由失败

**问题**：进化模式 (`evolutionMode: true`) 无法正确路由到后端。

**根本原因**：Protobuf JSON 序列化中，`false` 值会被省略

**解决方案**：
```typescript
if (params.evolutionMode && request.evolutionMode === undefined) {
  (request as any).evolutionMode = true;
}
```

---

### AI Token 统计与缓存指标

**问题**：日志显示 `content_length=451`，数据库 `LENGTH(content)=163`

**根本原因**：UTF-8 编码差异，中文字符占用 3 字节

---

### Orchestrator 模式多轮对话问题

**问题**：多轮对话（"最近记了什么笔记" → "总结这些笔记"）出现 5 个关联 bug：

1. **Block 未创建** - 第二轮无 block_id
2. **UserID 丢失** - `user_id=0` 导致 BM25 搜索失败
3. **粘性路由失效** - 非确认词输入未复用粘性
4. **标题重复生成** - 非首轮仍触发标题生成
5. **上下文未复用** - 第二轮重新搜索，结果为 0

**根本原因**：

| 问题 | 根因 |
|------|------|
| Block 未创建 | `executeWithOrchestrator` 绕过了 `executeAgent` 中的 block 创建逻辑 |
| UserID 丢失 | `ParrotExpertRegistry` 创建时硬编码 `user_id=0`，未从 context 获取 |
| 粘性路由失效 | 粘性检查仅对简短确认词（好/ok/是的）生效 |
| 标题重复生成 | 依赖问题1：未创建 block 导致 `len(blocks)==1` 判断错误 |
| 上下文未复用 | `executeWithOrchestrator` 未构建历史，`history=nil` |

**代码定位**：

```
Block 创建: handler.go:628-643 (executeAgent) vs handler.go:526-577 (executeWithOrchestrator 缺少)
UserID 传递: ai_service_chat.go:286 (硬编码0) → expert_registry.go:128 (使用固定值)
粘性检查: chat_router_metadata.go:46-63 (仅对确认词)
上下文构建: handler.go:553 (Orchestrator 缺少 BuildHistory 调用)
```

**修复方案**：

```go
// 1. executeWithOrchestrator 中添加 block 创建
var currentBlock *store.AIBlock
if h.blockManager != nil && req.ConversationID > 0 {
    currentBlock, _ = h.blockManager.CreateBlockForChat(ctx, req.ConversationID, req.Message,
        AgentTypeOrchestrator, BlockModeNormal)
}

// 2. ExpertRegistry 从 context 获取 user_id
func (r *ParrotExpertRegistry) getUserIDFromContext(ctx context.Context) int32 {
    if userID, ok := ctx.Value("user_id").(int32); ok {
        return userID
    }
    return 0
}

// 3. 放宽粘性路由条件
if isSticky, meta := r.metadataMgr.IsStickyValid(ctx, conversationID); isSticky {
    if isShortConfirmation(input) || isRelatedToLastIntent(input, meta.LastIntent) {
        return stickyRoute
    }
}

// 4. executeWithOrchestrator 中构建历史
history, _ = h.contextBuilder.BuildHistory(ctx, &ctxpkg.ContextRequest{...})
```

**依赖关系**：问题1(Block未创建) → 问题4(标题重复)；问题1 → 问题5(上下文持久化)

**修复状态**（2026-02-17）：

采用面向未来的架构：通过 `OrchestratorContext` 在 context 中传递 userID、history 等参数。

| 问题 | 修复方式 | 关键文件 |
|------|----------|----------|
| Block 未创建 | 在 `executeWithOrchestrator` 中添加 block 创建逻辑 | `handler.go:executeWithOrchestrator` |
| UserID 丢失 | 通过 `ctxpkg.WithOrchestratorContext` 注入 userID，`ExecuteExpert` 从 ctx 提取 | `handler.go`, `expert_registry.go` |
| 粘性路由失效 | 添加 `isRelatedToLastIntent` 函数，检查输入是否与上一意图相关 | `chat_router.go`, `chat_router_metadata.go` |
| 标题重复生成 | 随问题1修复自动解决 | - |
| 上下文未复用 | 在 `executeWithOrchestrator` 中构建 history 并注入 context | `handler.go` |

**新增文件**：
- `ai/context/orchestrator.go` - OrchestratorContext 定义和 helper 函数

**架构优势**：
- 未来添加新参数只需修改 context，无需改接口签名
- 解耦各层之间的参数传递

---

## 部署问题

### 二进制部署运维权限问题

**问题**：divine 用户执行运维操作遇到权限问题。

**解决方案**：
```bash
usermod -aG docker divine
cat > /etc/sudoers.d/divinesense << 'EOF'
divine ALL=(ALL) NOPASSWD: /bin/systemctl restart divinesense.service
EOF
```

---

## 开发流程问题

### 环境意识不足导致的重复错误

**问题**：AI Agent 执行命令时频繁犯错。

**解决方案**：优先使用 Makefile wrapper，添加容器检测

---

## 贡献指南

1. 记录问题：添加新章节，标题格式：`## 问题名称`
2. 描述现象：用户可见的故障表现
3. 分析原因：深入分析，不要停留在表面
4. 记录方案：最终采用的解决方案
5. 提炼教训：可复用的经验，避免重复踩坑
6. 更新索引：在"快速查找"章节添加对应条目
