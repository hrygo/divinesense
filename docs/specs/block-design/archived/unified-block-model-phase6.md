# P1-A006: Unified Block Model - Phase 6 Integration & Testing

> **状态**: 🔲 待开发
> **优先级**: P1 (重要)
> **投入**: 3人天
> **Sprint**: Sprint 2
> **关联 Issue**: [#71](https://github.com/hrygo/divinesense/issues/71)
> **依赖**: Phase 1-5 全部完成

---

## 1. 目标与背景

### 1.1 核心目标

完成端到端集成测试，确保 Unified Block Model 在所有模式下（Normal/Geek/Evolution）都能正确工作。

### 1.2 用户价值

- **功能完整性**：所有对话功能正常工作
- **数据完整性**：对话历史完整保存，可随时恢复

### 1.3 技术价值

- **质量保证**：通过全面的测试确保代码质量
- **回归测试**：为后续开发提供测试基准

---

## 2. 依赖关系

### 2.1 前置依赖（必须完成）

- [x] **Phase 1**: 数据库和 Store 层
- [x] **Phase 2**: Proto 和 API
- [x] **Phase 3**: 前端类型定义
- [x] **Phase 4**: 前端组件改造
- [x] **Phase 5**: Chat Handler 改造

### 2.2 并行依赖

无

### 2.3 后续依赖

- [ ] **Issue #69**: Warp Block UI 完成（前端已完成）

---

## 3. 功能设计

### 3.1 测试场景

```
┌─────────────────────────────────────────────────────────────────┐
│  测试场景覆盖                                                     │
├─────────────────────────────────────────────────────────────────┤
│  ┌─────────────────────────────────────────────────────────────┐ │
│  │  单元测试 (Unit Tests)                                     │ │
│  │  ├── Store CRUD 操作                                      │ │
│  │  ├── Block 状态转换                                        │ │
│  │  └── 事件流写入                                            │ │
│  ├─────────────────────────────────────────────────────────────┤ │
│  │  集成测试 (Integration Tests)                              │ │
│  │  ├── Chat Handler → Store                                 │ │
│  │  ├── SSE 事件流                                            │ │
│  │  └── CC Runner 集成                                        │ │
│  ├─────────────────────────────────────────────────────────────┤ │
│  │  端到端测试 (E2E Tests)                                    │ │
│  │  ├── Normal 模式完整流程                                   │ │
│  │  ├── Geek 模式完整流程                                    │ │
│  │  ├── Evolution 模式完整流程                               │ │
│  │  └── 追加输入流程                                          │ │
│  └─────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────┘
```

### 3.2 测试用例

#### 3.2.1 单元测试

| 测试用例            | 描述           | 验收条件                       |
| :------------------ | :------------- | :----------------------------- |
| **CreateBlock**     | 创建新 Block   | ID 分配成功，status=pending    |
| **AppendUserInput** | 追加用户输入   | UserInputs 数组长度增加        |
| **AppendEvent**     | 追加事件       | EventStream 数组长度增加       |
| **UpdateStatus**    | 更新状态       | Status 字段正确更新            |
| **GetLatestBlock**  | 获取最新 Block | 返回 round_number 最大的 Block |

#### 3.2.2 集成测试

| 测试用例         | 描述                     | 验收条件              |
| :--------------- | :----------------------- | :-------------------- |
| **Chat → Store** | Chat Handler 调用 Store  | Block 正确保存        |
| **SSE → Block**  | SSE 事件更新 Block       | Block 状态实时更新    |
| **CC → Block**   | CC Runner 会话写入 Block | SessionStats 正确保存 |

#### 3.2.3 端到端测试

| 测试用例           | 描述                      | 验收条件                        |
| :----------------- | :------------------------ | :------------------------------ |
| **Normal 模式**    | 完整的普通对话流程        | Block 创建→流式更新→完成        |
| **Geek 模式**      | 完整的 Geek 对话流程      | CC 会话映射正确，事件完整记录   |
| **Evolution 模式** | 完整的 Evolution 对话流程 | CC 会话映射正确，PR 创建成功    |
| **追加输入**       | 在 AI 回复前追加输入      | 追加到当前 Block，而非创建新的  |
| **多 Block**       | 连续多轮对话              | 所有 Block 按 round_number 排序 |

### 3.3 关键决策

| 决策点       | 方案 A             | 方案 B        | 选择  | 理由                     |
| :----------- | :----------------- | :------------ | :---: | :----------------------- |
| **测试框架** | table-driven tests | testify suite | **A** | 更易维护                 |
| **E2E 工具** | Playwright         | 自定义        | **A** | 项目已有 Playwright 配置 |
| **Mock**     | SQLite 内存库      | Mock 接口     | **A** | 更接近真实场景           |

---

## 4. 技术实现

### 4.1 单元测试

```go
// store/db/postgres/ai_block_test.go

package postgres

import (
    "context"
    "testing"
    "time"

    "github.com/hrygo/divinesense/store"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestAIBlockStore_CreateBlock(t *testing.T) {
    ctx := context.Background()
    db := setupTestDB(t)
    defer db.Close()

    // Create test conversation
    conv := &store.AIConversation{
        CreatorID: 1,
        Title:     "Test Conversation",
        ParrotID:  "AMAZING",
        CreatedTs: time.Now().Unix(),
        UpdatedTs: time.Now().Unix(),
    }
    conv, err := db.CreateAIConversation(ctx, conv)
    require.NoError(t, err)

    // Create block
    create := &store.CreateAIBlock{
        ConversationID: conv.ID,
        BlockType:      store.AIBlockTypeMessage,
        Mode:           store.AIBlockModeNormal,
        UserInputs: []store.UserInput{
            {
                Content:   "Hello, AI",
                Timestamp: time.Now().Unix(),
            },
        },
        Status:    store.AIBlockStatusPending,
        CreatedTs: time.Now().Unix(),
        UpdatedTs: time.Now().Unix(),
    }

    block, err := db.CreateAIBlock(ctx, create)

    assert.NoError(t, err)
    assert.NotZero(t, block.ID)
    assert.NotEmpty(t, block.UID)
    assert.Equal(t, store.AIBlockStatusPending, block.Status)
    assert.Len(t, block.UserInputs, 1)
}

func TestAIBlockStore_AppendUserInput(t *testing.T) {
    // Setup...
    ctx := context.Background()
    db := setupTestDB(t)
    defer db.Close()

    // Create block...
    block, _ := createTestBlock(ctx, t, db)

    // Append user input
    input := store.UserInput{
        Content:   "Additional input",
        Timestamp: time.Now().Unix(),
    }

    err := db.AppendUserInput(ctx, block.ID, input)

    assert.NoError(t, err)

    // Verify
    updated, _ := db.GetAIBlock(ctx, block.ID)
    assert.Len(t, updated.UserInputs, 2)
    assert.Equal(t, "Additional input", updated.UserInputs[1].Content)
}

func TestAIBlockStore_UpdateStatus(t *testing.T) {
    // Setup...
    ctx := context.Background()
    db := setupTestDB(t)
    defer db.Close()

    // Create block with pending status...
    block, _ := createTestBlock(ctx, t, db)

    // Update to streaming
    err := db.UpdateStatus(ctx, block.ID, store.AIBlockStatusStreaming)

    assert.NoError(t, err)

    // Verify
    updated, _ := db.GetAIBlock(ctx, block.ID)
    assert.Equal(t, store.AIBlockStatusStreaming, updated.Status)
}

func TestAIBlockStore_GetLatestBlock(t *testing.T) {
    // Setup...
    ctx := context.Background()
    db := setupTestDB(t)
    defer db.Close()

    conv, _ := createTestConversation(ctx, t, db)

    // Create multiple blocks
    for i := 0; i < 3; i++ {
        create := &store.CreateAIBlock{
            ConversationID: conv.ID,
            BlockType:      store.AIBlockTypeMessage,
            Mode:           store.AIBlockModeNormal,
            UserInputs: []store.UserInput{
                {
                    Content:   fmt.Sprintf("Message %d", i),
                    Timestamp: time.Now().Unix(),
                },
            },
            Status:    store.AIBlockStatusCompleted,
            CreatedTs: time.Now().Unix(),
            UpdatedTs: time.Now().Unix(),
        }
        db.CreateAIBlock(ctx, create)
    }

    // Get latest
    latest, err := db.GetLatestBlock(ctx, conv.ID)

    assert.NoError(t, err)
    assert.NotNil(t, latest)
    assert.Equal(t, int32(2), latest.RoundNumber)
}
```

### 4.2 集成测试

```go
// server/router/api/v1/ai/integration_test.go

package ai

import (
    "context"
    "testing"
    "time"

    "connectrpc.com/connect"
    "github.com/hrygo/divinesense/gen/api/v1/aiv1"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestChatHandler_BlockLifecycle(t *testing.T) {
    // Setup...
    ctx := context.Background()
    handler := setupTestHandler(t)
    stream := newMockStream()

    // Create conversation
    conv, _ := handler.db.CreateAIConversation(ctx, &store.AIConversation{
        CreatorID: 1,
        Title:     "Test",
    })

    // Request
    req := &aiv1.ChatRequest{
        Message:         "Hello",
        ConversationId: conv.ID,
    }

    // Execute
    err := handler.Chat(ctx, connect.NewRequest(req), stream)

    // Verify
    assert.NoError(t, err)

    // Check block was created
    blocks, _ := handler.blockStore.ListBlocks(ctx, &store.FindAIBlock{
        ConversationID: &conv.ID,
    })
    assert.Len(t, blocks, 1)

    block := blocks[0]
    assert.Equal(t, store.AIBlockStatusCompleted, block.Status)
    assert.Len(t, block.EventStream, 0)
}

func TestChatHandler_AppendInput(t *testing.T) {
    // Setup...
    ctx := context.Background()
    handler := setupTestHandler(t)

    // Create conversation and block
    conv, _ := handler.db.CreateAIConversation(ctx, &store.AIConversation{
        CreatorID: 1,
        Title:     "Test",
    })

    block, _ := handler.blockStore.CreateBlock(ctx, &store.CreateAIBlock{
        ConversationID: conv.ID,
        UserInputs: []store.UserInput{
            {Content: "First input", Timestamp: time.Now().Unix()},
        },
        Status:    store.AIBlockStatusStreaming,
    })

    // Send append request
    req := &aiv1.ChatRequest{
        Message:         "Additional input",
        ConversationId: conv.ID,
    }

    stream := newMockStream()
    err := handler.Chat(ctx, connect.NewRequest(req), stream)

    // Verify
    assert.NoError(t, err)

    // Check input was appended
    updated, _ := handler.blockStore.GetBlock(ctx, block.ID)
    assert.Len(t, updated.UserInputs, 2)
    assert.Equal(t, "Additional input", updated.UserInputs[1].Content)
}
```

### 4.3 E2E 测试

```typescript
// web/e2e/block-model.spec.ts

import { test, expect } from '@playwright/test';

test.describe('Unified Block Model - Normal Mode', () => {
  test('should create and complete a block', async ({ page }) => {
    // Navigate to chat
    await page.goto('/chat');

    // Select parrot
    await page.click('[data-testid="parrot-AMAZING"]');

    // Send message
    await page.fill('[data-testid="chat-input"]', 'Hello, AI');
    await page.click('[data-testid="send-button"]');

    // Wait for block to appear
    await expect(page.locator('[data-testid="block"]')).toBeVisible();

    // Verify block status
    const block = page.locator('[data-testid="block"]').first();
    await expect(block).toHaveAttribute('data-status', 'completed');

    // Verify block content
    await expect(block.locator('[data-testid="block-user-content"]')).toHaveText('Hello, AI');
    await expect(block.locator('[data-testid="block-ai-content"]')).toBeVisible();
  });

  test('should append input to streaming block', async ({ page }) => {
    // This test requires mocking the AI response to be slow
    await page.goto('/chat');

    // Send first message
    await page.fill('[data-testid="chat-input"]', 'First question');
    await page.click('[data-testid="send-button"]');

    // Wait for streaming to start
    await page.waitForSelector('[data-status="streaming"]');

    // Send second message while streaming
    await page.fill('[data-testid="chat-input"]', 'Additional context');
    await page.click('[data-testid="send-button"]');

    // Verify both inputs are in the same block
    const block = page.locator('[data-testid="block"]').first();
    await expect(block.locator('[data-testid="block-user-inputs"]')).toHaveCount(2);
  });
});

test.describe('Unified Block Model - Geek Mode', () => {
  test('should create geek block with CC session mapping', async ({ page }) => {
    await page.goto('/chat');

    // Enable Geek Mode
    await page.click('[data-testid="mode-toggle"]');
    await page.click('[data-testid="mode-geek"]');

    // Send code request
    await page.fill('[data-testid="chat-input"]', 'Write a hello world function');
    await page.click('[data-testid="send-button"]');

    // Wait for geek block
    await expect(page.locator('[data-testid="block"][data-mode="geek"]')).toBeVisible();

    // Verify session stats are displayed
    await expect(page.locator('[data-testid="session-summary"]')).toBeVisible();
    await expect(page.locator('[data-testid="session-cost"]')).toBeVisible();
  });
});

test.describe('Unified Block Model - Block History', () => {
  test('should load all blocks for conversation', async ({ page }) => {
    // Setup: Create conversation with multiple blocks
    await page.goto('/chat');

    // Send multiple messages
    for (let i = 0; i < 3; i++) {
      await page.fill('[data-testid="chat-input"]', `Message ${i}`);
      await page.click('[data-testid="send-button"]');
      await page.waitForSelector('[data-status="completed"]');
    }

    // Reload page
    await page.reload();

    // Verify all blocks are restored
    await expect(page.locator('[data-testid="block"]')).toHaveCount(3);
  });

  test('should maintain block expansion state', async ({ page }) => {
    await page.goto('/chat');

    // Send message
    await page.fill('[data-testid="chat-input"]', 'Test message');
    await page.click('[data-testid="send-button"]');
    await page.waitForSelector('[data-status="completed"]');

    // Collapse block
    await page.click('[data-testid="block-header"]');

    // Reload page
    await page.reload();

    // Verify block remains collapsed
    const block = page.locator('[data-testid="block"]').first();
    await expect(block).toHaveAttribute('data-collapsed', 'true');
  });
});
```

### 4.4 关键代码路径

| 文件路径                                      | 职责             |
| :-------------------------------------------- | :--------------- |
| `store/db/postgres/ai_block_test.go`          | 单元测试（新增） |
| `server/router/api/v1/ai/integration_test.go` | 集成测试（新增） |
| `web/e2e/block-model.spec.ts`                 | E2E 测试（新增） |

---

## 5. 交付物清单

### 5.1 代码文件

- [ ] `store/db/postgres/ai_block_test.go` - 单元测试
- [ ] `server/router/api/v1/ai/integration_test.go` - 集成测试
- [ ] `web/e2e/block-model.spec.ts` - E2E 测试

### 5.2 数据库变更

无

### 5.3 配置变更

无

### 5.4 文档更新

- [ ] `docs/specs/unified-block-model.md` - 更新完成状态

---

## 6. 测试验收

### 6.1 功能验收

| 测试类型 | 数量 | 通过率目标 |
| :------- | :--- | :--------- |
| 单元测试 | > 20 | 100%       |
| 集成测试 | > 10 | 100%       |
| E2E 测试 | > 15 | 100%       |

### 6.2 性能验收

| 指标       | 目标值  | 测试方法 |
| :--------- | :------ | :------- |
| 创建 Block | < 20ms  | 单元测试 |
| 追加事件   | < 10ms  | 单元测试 |
| 端到端延迟 | < 500ms | E2E 测试 |

### 6.3 质量验收

- [ ] 代码覆盖率 > 80%
- [ ] 所有 lint 检查通过
- [ ] 构建成功

---

## 7. ROI 分析

| 维度     | 值                             |
| :------- | :----------------------------- |
| 开发投入 | 3人天                          |
| 预期收益 | 全面测试保证质量，减少线上问题 |
| 风险评估 | 低（纯测试）                   |
| 回报周期 | 1 Sprint                       |

---

## 8. 风险与缓解

| 风险               | 概率  | 影响 | 缓解措施             |
| :----------------- | :---: | :--- | :------------------- |
| **测试环境不稳定** |  低   | 中   | 使用 Docker 容器隔离 |
| **Mock 数据不足**  |  低   | 低   | 覆盖边界场景         |

---

## 9. 实施计划

### 9.1 时间表

| 阶段      | 时间  | 任务                   |
| :-------- | :---- | :--------------------- |
| **Day 1** | 1人天 | 单元测试编写           |
| **Day 2** | 1人天 | 集成测试编写           |
| **Day 3** | 1人天 | E2E 测试编写，全量测试 |

### 9.2 检查点

- [ ] Checkpoint 1: 单元测试全部通过
- [ ] Checkpoint 2: 集成测试全部通过
- [ ] Checkpoint 3: E2E 测试全部通过

---

## 附录

### A. 参考资料

- [Phase 1-5 Specs](./)
- [CC Runner 异步架构](../cc_runner_async_arch.md)

### B. 变更记录

| 日期       | 版本 | 变更内容 | 作者   |
| :--------- | :--- | :------- | :----- |
| 2026-02-04 | v1.0 | 初始版本 | Claude |
