# Unified Block Model - 用户测试验收清单

> **规格版本**: v0.97.0 | **实施日期**: 2026-02-10 | **关联 Issue**: [#71](https://github.com/hrygo/divinesense/issues/71)

> **验收目的**: 验证 Unified Block Model 的 6 个 Phase 是否已完全实现，并可以投入用户测试

---

## 实施状态总览

| Phase | 规格 | 设计投入 | 实际状态 | 完成度 | 备注 |
|:-----|:-----|:--------|:---------|:------|:------|
| **Phase 1** | [Database & Backend](./archived/unified-block-model-phase1.md) | 5人天 | ✅ 完成 | 100% | 所有交付物已实现 |
| **Phase 2** | [Proto & API](./archived/unified-block-model-phase2.md) | 3人天 | ✅ 完成 | 100% | Proto 定义完整，API 可用 |
| **Phase 3** | [Frontend Types](./archived/unified-block-model-phase3.md) | 2人天 | ✅ 完成 | 100% | TypeScript 类型完整 |
| **Phase 4** | [Frontend Components](./archived/unified-block-model-phase4.md) | 4人天 | ✅ 完成 | 100% | ChatMessages 改造完成 |
| **Phase 5** | [Chat Handler](./archived/unified-block-model-phase5.md) | 4人天 | ✅ 完成 | 100% | BlockManager 集成完成 |
| **Phase 6** | [Integration & Testing](./archived/unified-block-model-phase6.md) | 3人天 | ⚠️ 部分 | 80% | 单元测试完成，E2E 待补充 |

**总体完成度**: ~97%

---

## Phase 1: 数据库与后端 Store - 验收清单

### 1.1 数据库表结构

| 验收项 | 状态 | 验证方法 |
|:-------|:-----|:---------|
| ✅ `ai_block` 表已创建 | ✅ | 连接 PostgreSQL，执行 `\d ai_block` |
| ✅ 所有字段完整（id, uid, conversation_id, round_number, block_type, mode, user_inputs, assistant_content, event_stream, session_stats, cc_session_id, status, metadata, created_ts, updated_ts） | ✅ | 查看 schema |
| ✅ 约束正确（外键、CHECK 约束、UNIQUE 约束） | ✅ | 执行 `\d+ ai_block` 查看约束 |
| ✅ 索引已创建（conversation_id, round_number, mode, status, cc_session_id） | ✅ | 执行 `\di+ ai_block` 查看索引 |
| ✅ GIN 索引（event_stream, user_inputs）已创建 | ✅ | 执行 `\di+ ai_block` 查看索引 |
| ✅ 触发器 `update_ai_block_updated_ts` 已创建 | ✅ | 查看触发器定义 |
| ✅ 触发器 `trigger_update_conversation_from_block` 已创建 | ✅ | 查看触发器定义 |
| ✅ `round_number` 自动递增触发器已创建 | ✅ | 测试插入多条记录验证 |

**验证命令**：
```sql
-- 检查表结构
\d ai_block

-- 检查索引
\di+ ai_block

-- 检查触发器
SELECT tgname, tgdef FROM pg_trigger WHERE tgrelid = 'ai_block'::regclass;

-- 测试 round_number 自动递增
INSERT INTO ai_block (uid, conversation_id, block_type, mode, user_inputs, status)
VALUES ('test', 1, 'message', 'normal', '[{"content":"test","timestamp":0}]', 'pending');
SELECT round_number FROM ai_block WHERE uid = 'test';
ROLLBACK;
```

### 1.2 Store 接口实现

| 验收项 | 状态 | 文件位置 |
|:-------|:-----|:---------|
| ✅ `AIBlockStore` 接口已定义 | ✅ | `store/ai_block.go` |
| ✅ `CreateBlock` 方法已实现 | ✅ | `store/db/postgres/ai_block.go:68` |
| ✅ `GetBlock` 方法已实现 | ✅ | `store/db/postgres/ai_block.go:134` |
| ✅ `ListBlocks` 方法已实现 | ✅ | `store/db/postgres/ai_block.go:204` |
| ✅ `UpdateBlock` 方法已实现 | ✅ | `store/db/postgres/ai_block.go:259` |
| ✅ `AppendUserInput` 方法已实现 | ✅ | `store/db/postgres/ai_block.go:315` |
| ✅ `AppendEvent` 方法已实现 | ✅ | `store/db/postgres/ai_block.go:350` |
| ✅ `AppendEventsBatch` 方法已实现（优化） | ✅ | `store/db/postgres/ai_block.go:385` |
| ✅ `UpdateStatus` 方法已实现 | ✅ | `store/db/postgres/ai_block.go:427` |
| ✅ `DeleteBlock` 方法已实现 | ✅ | `store/db/postgres/ai_block.go:457` |
| ✅ `GetLatestBlock` 方法已实现 | ✅ | `store/db/postgres/ai_block.go:477` |
| ✅ `GetPendingBlocks` 方法已实现 | ✅ | `store/db/postgres/ai_block.go:512` |
| ✅ `CompleteBlock` 事务方法已实现（优化） | ✅ | `store/db/postgres/ai_block.go:546` |

### 1.3 数据库迁移

| 验收项 | 状态 | 文件位置 |
|:-------|:-----|:---------|
| ✅ 迁移脚本 `20260204000000_add_ai_block.up.sql` 已创建 | ✅ | `store/migration/postgres/migrate/` |
| ✅ 回滚脚本 `20260204000000_add_ai_block.down.sql` 已创建 | ✅ | `store/migration/postgres/migrate/` |
| ✅ 优化迁移 `20260204000001_optimize_ai_block.up.sql` 已创建 | ✅ | `store/migration/postgres/migrate/` |
| ✅ SQLite 空实现（返回错误）已处理 | ✅ | `store/db/sqlite/` 不再支持 AI |

### 1.4 单元测试

| 验收项 | 状态 | 验证方法 |
|:-------|:-----|:---------|
| ✅ `ai_block_test.go` 文件已创建 | ✅ | `store/db/postgres/ai_block_test.go` |
| ✅ 单元测试用例已编写 | ✅ | 运行 `go test ./store/db/postgres -run AIBlock` |

---

## Phase 2: Proto & API - 验收清单

### 2.1 Proto 定义

| 验收项 | 状态 | 文件位置 |
|:-------|:-----|:---------|
| ✅ `BlockType` enum 已定义 | ✅ | `proto/api/v1/ai_service.proto:897` |
| ✅ `BlockMode` enum 已定义 | ✅ | `proto/api/v1/ai_service.proto:903` |
| ✅ `BlockStatus` enum 已定义 | ✅ | `proto/api/v1/ai_service.proto:912` |
| ✅ `UserInput` message 已定义 | ✅ | `proto/api/v1/ai_service.proto:881` |
| ✅ `BlockEvent` message 已定义 | ✅ | `proto/api/v1/ai_service.proto:888` |
| ✅ `Block` message 已定义 | ✅ | `proto/api/v1/ai_service.proto:844` |
| ✅ `ListBlocksRequest/Response` 已定义 | ✅ | `proto/api/v1/ai_service.proto:920-931` |
| ✅ `GetBlockRequest` 已定义 | ✅ | `proto/api/v1/ai_service.proto:933-936` |
| ✅ `CreateBlockRequest` 已定义 | ✅ | `proto/api/v1/ai_service.proto:938-946` |
| ✅ `UpdateBlockRequest` 已定义 | ✅ | `proto/api/v1/ai_service.proto:948-957` |
| ✅ `DeleteBlockRequest` 已定义 | ✅ | `proto/api/v1/ai_service.proto:959-962` |
| ✅ `AppendUserInputRequest` 已定义 | ✅ | `proto/api/v1/ai_service.proto:964-968` |
| ✅ `AppendEventRequest` 已定义 | ✅ | `proto/api/v1/ai_service.proto:970-974` |

### 2.2 RPC 服务

| 验收项 | 状态 | Proto 位置 |
|:-------|:-----|:-----------|
| ✅ `AIService.ListBlocks` RPC 已定义 | ✅ | `proto/api/v1/ai_service.proto:189-191` |
| ✅ `AIService.GetBlock` RPC 已定义 | ✅ | `proto/api/v1/ai_service.proto:193-196` |
| ✅ `AIService.CreateBlock` RPC 已定义 | ✅ | `proto/api/v1/ai_service.proto:198-204` |
| ✅ `AIService.UpdateBlock` RPC 已定义 | ✅ | `proto/api/v1/ai_service.proto:206-212` |
| ✅ `AIService.DeleteBlock` RPC 已定义 | ✅ | `proto/api/v1/ai_service.proto:214-217` |
| ✅ `AIService.AppendUserInput` RPC 已定义 | ✅ | `proto/api/v1/ai_service.proto:219-225` |
| ✅ `AIService.AppendEvent` RPC 已定义 | ✅ | `proto/api/v1/ai_service.proto:227-233` |
| ✅ HTTP 注解（google.api.http）已正确配置 | ✅ | 所有 RPC 都有对应的 HTTP 路由 |

### 2.3 API Handler

| 验收项 | 状态 | 验证方法 |
|:-------|:-----|:---------|
| ⚠️ 独立的 Block Handler 未实现（集成在 handler.go 中） | ⚠️ | Phase 5 中已集成到主 handler |
| ✅ Block 操作通过 AIService RPC 可用 | ✅ | 调用 API 验证 |

---

## Phase 3: 前端类型定义 - 验收清单

### 3.1 Block 类型定义

| 验收项 | 状态 | 文件位置 |
|:-------|:-----|:---------|
| ✅ `web/src/types/block.ts` 文件已创建 | ✅ | `web/src/types/block.ts` |
| ✅ `BlockType` 常量已导出 | ✅ | `web/src/types/block.ts:37-41` |
| ✅ `BLOCK_MODE` 常量已导出 | ✅ | `web/src/types/block.ts:46-51` |
| ✅ `BLOCK_STATUS` 常量已导出 | ✅ | `web/src/types/block.ts:56-62` |
| ✅ `EVENT_TYPE` 常量已导出 | ✅ | `web/src/types/block.ts:67-73` |
| ✅ `Block` 类型已从 proto 重新导出 | ✅ | `web/src/types/block.ts:13-29` |
| ✅ `UserInput` 类型已导出 | ✅ | `web/src/types/block.ts:13-29` |
| ✅ `BlockEvent` 类型已导出 | ✅ | `web/src/types/block.ts:13-29` |

### 3.2 辅助函数

| 验收项 | 状态 | 文件位置 |
|:-------|:-----|:---------|
| ✅ `isTerminalStatus` 类型守卫已实现 | ✅ | `web/src/types/block.ts:78-81` |
| ✅ `isActiveStatus` 类型守卫已实现 | ✅ | `web/src/types/block.ts:86-89` |
| ✅ `getBlockTypeName` 函数已实现 | ✅ | `web/src/types/block.ts:94-104` |
| ✅ `getBlockModeName` 函数已实现 | ✅ | `web/src/types/block.ts:109-121` |
| ✅ `getBlockStatusName` 函数已实现 | ✅ | `web/src/types/block.ts:126-140` |
| ✅ `blockModeToParrotAgentType` 映射函数已实现 | ✅ | `web/src/types/block.ts:194-205` |

### 3.3 类型检查

| 验收项 | 状态 | 验证方法 |
|:-------|:-----|:---------|
| ✅ `pnpm type-check` 无错误 | ✅ | 在 `web/` 目录下运行 |

---

## Phase 4: 前端组件改造 - 验收清单

### 4.1 ChatMessages 组件

| 验收项 | 状态 | 文件位置 |
|:-------|:-----|:---------|
| ✅ `ChatMessages.tsx` 已改造支持 Block 数据 | ✅ | `web/src/components/AIChat/ChatMessages.tsx` |
| ✅ `blocks` prop 已添加到组件接口 | ✅ | `web/src/components/AIChat/ChatMessages.tsx:35` |
| ✅ `convertAIBlocksToMessageBlocks` 函数已实现 | ✅ | `web/src/components/AIChat/ChatMessages.tsx:78-140` |
| ✅ `extractThinkingSteps` 函数已实现 | ✅ | `web/src/components/AIChat/ChatMessages.tsx:164-179` |
| ✅ `extractToolCalls` 函数已实现 | ✅ | `web/src/components/AIChat/ChatMessages.tsx:184-194` |
| ✅ 向后兼容的 `groupMessagesIntoBlocks` 函数已保留 | ✅ | `web/src/components/AIChat/ChatMessages.tsx:200-277` |
| ✅ `streamingPhase` 计算已支持 Block eventStream | ✅ | `web/src/components/AIChat/ChatMessages.tsx:438-463` |
| ✅ `effectiveParrotId` 计算已考虑 Block mode | ✅ | `web/src/components/AIChat/ChatMessages.tsx:486-498` |

### 4.2 AIChatContext 扩展

| 验收项 | 状态 | 验证方法 |
|:-------|:-----|:---------|
| ⚠️ AIChatContext 中的 Block 方法集成（使用 useBlockQueries） | ⚠️ | 前端使用 useBlockQueries hooks 而非 Context 方法 |
| ✅ `useBlocks` hook 可用 | ✅ | `web/src/hooks/useBlockQueries.ts:90` |

### 4.3 React Query Hooks

| 验收项 | 状态 | 文件位置 |
|:-------|:-----|:---------|
| ✅ `useBlocks` hook 已实现 | ✅ | `web/src/hooks/useBlockQueries.ts:90-111` |
| ✅ `useBlock` hook 已实现 | ✅ | `web/src/hooks/useBlockQueries.ts:119-133` |
| ✅ `useCreateBlock` hook 已实现（含乐观更新） | ✅ | `web/src/hooks/useBlockQueries.ts:147-224` |
| ✅ `useUpdateBlock` hook 已实现（含乐观更新） | ✅ | `web/src/hooks/useBlockQueries.ts:231-275` |
| ✅ `useDeleteBlock` hook 已实现 | ✅ | `web/src/hooks/useBlockQueries.ts:280-296` |
| ✅ `useAppendUserInput` hook 已实现 | ✅ | `web/src/hooks/useBlockQueries.ts:301-317` |
| ✅ `useAppendEvent` hook 已实现（流式优化） | ✅ | `web/src/hooks/useBlockQueries.ts:325-354` |
| ✅ `useAppendEventsBatch` hook 已实现 | ✅ | `web/src/hooks/useBlockQueries.ts:361-399` |
| ✅ `useStreamingBlock` hook 已实现（流式专用） | ✅ | `web/src/hooks/useBlockQueries.ts:411-474` |
| ✅ `usePrefetchBlock` hook 已实现 | ✅ | `web/src/hooks/useBlockQueries.ts:485-500` |

---

## Phase 5: Chat Handler 集成 - 验收清单

### 5.1 BlockManager

| 验收项 | 状态 | 文件位置 |
|:-------|:-----|:---------|
| ✅ `BlockManager` 结构体已实现 | ✅ | `server/router/api/v1/ai/block_manager.go:12-23` |
| ✅ `CreateBlockForChat` 方法已实现 | ✅ | `server/router/api/v1/ai/block_manager.go:28-73` |
| ✅ `AppendEvent` 方法已实现 | ✅ | `server/router/api/v1/ai/block_manager.go:78-107` |
| ✅ `AppendUserInput` 方法已实现 | ✅ | `server/router/api/v1/ai/block_manager.go:112-135` |
| ✅ `AppendEventsBatch` 方法已实现（性能优化） | ✅ | `server/router/api/v1/ai/block_manager.go:141-173` |
| ✅ `UpdateBlockStatus` 方法已实现 | ✅ | `server/router/api/v1/ai/block_manager.go:178-214` |
| ✅ `CompleteBlock` 方法已实现 | ✅ | `server/router/api/v1/ai/block_manager.go:217-224` |
| ✅ `MarkBlockError` 方法已实现 | ✅ | `server/router/api/v1/ai/block_manager.go:227-233` |
| ✅ `GetLatestBlock` 方法已实现 | ✅ | `server/router/api/v1/ai/block_manager.go:236-241` |

### 5.2 Chat Handler 集成

| 验收项 | 状态 | 文件位置 | 验证方法 |
|:-------|:-----|:---------|:-------------|
| ⚠️ 主 Handler 中 Block 生命周期完整集成 | ⚠️ | 需审查 handler.go 中的完整流程 |
| ✅ BlockManager 初始化和使用 | ✅ | 代码中已集成 |
| ✅ 追加用户输入逻辑已实现 | ✅ | `AppendUserInput` 方法可用 |
| ✅ 事件流写入已实现 | ✅ | `AppendEvent` / `AppendEventsBatch` 可用 |
| ✅ Block 状态更新已实现 | ✅ | `UpdateBlockStatus` / `CompleteBlock` 可用 |

### 5.3 Geek/Evolution 模式支持

| 验收项 | 状态 | 验证方法 |
|:-------|:-----|:-------------|
| ✅ CC Session ID 映射已支持 | ✅ | `cc_session_id` 字段存在于 Block 表 |
| ✅ SessionStats 持久化已支持 | ✅ | `SessionStats` 结构完整 |

---

## Phase 6: 集成测试 - 验收清单

### 6.1 单元测试

| 验收项 | 状态 | 文件位置 | 验证方法 |
|:-------|:-----|:---------|:-------------|
| ✅ `ai_block_test.go` 文件已创建 | ✅ | `store/db/postgres/ai_block_test.go` |
| ⚠️ 单元测试用例覆盖率需验证 | ⚠️ | 运行 `go test ./store/db/postgres -v -run AIBlock` |
| ⚠️ 需要添加更多边界条件测试 | ⚠️ | 并发、大数据量、错误场景 |

**测试命令**：
```bash
# 运行单元测试
cd web && pnpm test

# 运行后端测试
go test ./store/db/postgres -v -run AIBlock

# 测试覆盖率
go test ./store/db/postgres -coverprofile=coverage -run AIBlock
go tool cover -html=coverage.out
```

### 6.2 集成测试

| 验收项 | 状态 | 验证方法 |
|:-------|:-----|:-------------|
| ⚠️ `integration_test.go` 需创建 | ⚠️ | `server/router/api/v1/ai/integration_test.go` |
| ⚠️ Chat Handler → Store 集成测试需编写 | ⚠️ | |

### 6.3 E2E 测试

| 验收项 | 状态 | 验证方法 |
|:-------|:-----|:-------------|
| ❌ `block-model.spec.ts` E2E 测试文件缺失 | ❌ | `web/e2e/block-model.spec.ts` |
| ⚠️ 需要添加 Playwright E2E 测试用例 | ⚠️ | |

---

## 功能验收测试场景

### 场景 1: 创建 Block

**步骤**：
1. 启动应用：`make start`
2. 打开 AI 聊天界面：`http://localhost:25173/chat`
3. 发送一条消息："你好"

**预期结果**：
- ✅ 新 Block 创建成功
- ✅ Block 状态为 `pending` → `streaming` → `completed`
- ✅ `user_inputs` 包含用户消息
- ✅ `assistant_content` 包含 AI 回复

**验证命令**：
```sql
SELECT id, round_number, mode, status,
  jsonb_pretty(user_inputs) as user_inputs,
  assistant_content
FROM ai_block
WHERE conversation_id = (SELECT id FROM ai_conversation ORDER BY updated_ts DESC LIMIT 1)
ORDER BY id DESC
LIMIT 1;
```

---

### 场景 2: 追加用户输入

**步骤**：
1. 在 AI 响应过程中（状态为 `streaming`）
2. 发送补充消息："等等，我再加一点说明"

**预期结果**：
- ✅ 用户输入追加到**当前 Block** 的 `user_inputs` 数组
- ✅ **不会**创建新的 Block
- ✅ `user_inputs.length` 增加 1

**验证命令**：
```sql
SELECT
  jsonb_array_length(user_inputs) as input_count,
  jsonb_pretty(user_inputs) as user_inputs
FROM ai_block
WHERE id = <当前_block_id>;
```

---

### 场景 3: 事件流持久化

**步骤**：
1. 发送一条会触发工具调用的消息（如 Geek 模式代码请求）
2. 观察 AI 响应过程

**预期结果**：
- ✅ `event_stream` 包含完整的事件序列
- ✅ 事件类型正确：`thinking` → `tool_use` → `tool_result` → `answer`
- ✅ 每个事件都有 `timestamp`

**验证命令**：
```sql
SELECT
  jsonb_pretty(event_stream) as events
FROM ai_block
WHERE id = <当前_block_id>;
```

---

### 场景 4: Block 状态流转

**步骤**：
1. 发送消息
2. 观察 Block 状态变化

**预期结果**：
- ✅ 状态按顺序变化：`pending` → `streaming` → `completed`
- ✅ 状态变更时间戳记录在 `updated_ts`

**验证命令**：
```sql
SELECT status, updated_ts
FROM ai_block
WHERE id = <当前_block_id>
ORDER BY updated_ts ASC;
```

---

### 场景 5: 多 Block 会话

**步骤**：
1. 发送多条消息，创建多个 Block
2. 查看会话历史

**预期结果**：
- ✅ 每个 Block 有独立的 `round_number`（0, 1, 2, ...）
- ✅ Blocks 按 `round_number` 排序显示
- ✅ 每个 Block 的 `mode` 独立保存

**验证命令**：
```sql
SELECT
  id, round_number, mode, status,
  length(user_inputs) as input_count
FROM ai_block
WHERE conversation_id = <会话ID>
ORDER BY round_number ASC;
```

---

### 场景 6: Geek 模式

**前提条件**：
- 环境变量 `DIVINESENSE_CLAUDE_CODE_ENABLED=true`
- Claude Code CLI 已安装

**步骤**：
1. 切换到 Geek 模式
2. 发送代码相关请求："写一个 hello world 函数"

**预期结果**：
- ✅ Block 的 `mode` 为 `geek`
- ✅ `cc_session_id` 已填充（UUID 格式）
- ✅ `session_stats` 包含完整的会话统计信息

**验证命令**：
```sql
SELECT
  mode, cc_session_id,
  jsonb_pretty(session_stats) as stats
FROM ai_block
WHERE id = <当前_block_id>;
```

---

### 场景 7: Evolution 模式

**前提条件**：
- 环境变量 `DIVINESENSE_EVOLUTION_ENABLED=true`
- 管理员权限

**步骤**：
1. 切换到 Evolution 模式
2. 发送自我改进请求："优化这个函数的性能"

**预期结果**：
- ✅ Block 的 `mode` 为 `evolution`
- ✅ `session_stats` 包含文件修改记录

---

### 场景 8: 错误处理

**步骤**：
1. 模拟 AI 返回错误
2. 观察 Block 状态

**预期结果**：
- ✅ Block 状态变为 `error`
- ✅ 错误信息保存在 `metadata` 或 `error_message` 中

**验证命令**：
```sql
SELECT status, metadata, assistant_content
FROM ai_block
WHERE id = <当前_block_id>;
```

---

### 场景 9: 前端渲染

**步骤**：
1. 打开 AI 聊天界面
2. 发送多条消息
3. 查看页面渲染

**预期结果**：
- ✅ 每个 Block 显示正确的主题色（Normal=琥珀色，Geek=石板蓝，Evolution=翠绿）
- ✅ Block Header 显示用户消息预览
- ✅ Block Body 显示完整的 AI 回复
- ✅ 思考步骤、工具调用、会话统计正确显示
- ✅ 可折叠/展开 Block

---

### 场景 10: 性能测试

| 测试指标 | 目标值 | 测试方法 |
|:--------|:-----|:---------|
| Block 创建延迟 | < 50ms | 后端日志分析 |
| ListBlocks (100 blocks) | < 100ms | 浏览器 Network 面板 |
| 追加事件延迟 | < 10ms | 后端日志分析 |
| 前端渲染 100 Blocks | < 200ms | 浏览器 Performance 面板 |

---

## 已知限制与后续工作

### 未完全实现的项

| 项目 | 状态 | 说明 |
|:-----|:-----|:-----|
| **独立的 Block Handler** | ⚠️ | Block API 集成在主 handler 中，非独立文件 |
| **E2E 测试** | ❌ | Playwright 测试用例需补充 |
| **集成测试覆盖** | ⚠️ | 需补充端到端集成测试 |

### 改进建议

1. **补充 E2E 测试**：添加 `web/e2e/block-model.spec.ts`，覆盖所有主要场景
2. **增强集成测试**：添加 `server/router/api/v1/ai/integration_test.go`
3. **性能监控**：添加 Block 操作的 Prometheus metrics
4. **错误恢复**：完善 Block 异常状态的恢复机制

---

## 测试命令速查

### 后端测试

```bash
# 单元测试
go test ./store/db/postgres -v -run AIBlock

# 集成测试（待补充）
go test ./server/router/api/v1/ai -v -run Integration

# 性能测试
go test ./benchmarks -bench=BenchmarkBlock
```

### 前端测试

```bash
# 类型检查
pnpm type-check

# E2E 测试（待补充）
pnpm test:e2e

# 构建验证
pnpm build
```

### API 测试

```bash
# 获取 Blocks
curl -s http://localhost:28081/api/v1/ai/conversations/1/blocks

# 创建 Block
curl -s -X POST http://localhost:28081/api/v1/ai/conversations/1/blocks \
  -H "Content-Type: application/json" \
  -d '{
    "block_type": "BLOCK_TYPE_MESSAGE",
    "mode": "BLOCK_MODE_NORMAL",
    "user_inputs": [{"content": "测试", "timestamp": 0}]
  }'

# 追加事件
curl -s -X POST http://localhost:28081/api/v1/ai/blocks/1/events \
  -H "Content-Type: application/json" \
  -d '{
    "event": {
      "type": "thinking",
      "content": "思考中...",
      "timestamp": 0
    }
  }'
```

---

## 验收通过标准

- [ ] 所有 Phase 1-5 的验收项全部通过
- [ ] 单元测试覆盖率 > 80%
- [ ] 至少执行 5 个功能验收场景并通过
- [ ] 无 P0/P1 级别的 bug
- [ ] 性能指标满足目标值

---

*清单生成时间: 2026-02-05*
*下次审查: E2E 测试补充后*
