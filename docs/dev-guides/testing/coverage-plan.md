# 单元测试覆盖率提升计划

> **目标**：核心模块提升到 50% 覆盖率
> **原则**：务实、AI Native、面向 CI 快速执行

## 📊 整体覆盖情况

### 当前状态（2026-02-09）

| 覆盖率区间 | 包数量 | 代表包                                                |
| ---------- | ------ | ----------------------------------------------------- |
| 0-10%      | 3      | store/db/postgres (0.9%), server/router/api/v1 (1.7%) |
| 10-30%     | 8      | server/auth (27.1%), ai/agents (28.2%)                |
| 30-50%     | 12     | ai/core/retrieval (37.2%), ai (36.6%)                 |
| 50%+       | 40+    | ai/reminder (90.9%), ai/prediction (92.8%)            |

### 核心模块优先级

| 模块                    | 当前  | 目标 | 差距   | 优先级 | 业务关键度 |
| ----------------------- | ----- | ---- | ------ | ------ | ---------- |
| store/db/postgres       | 0.9%  | 50%  | -49.1% | 🔴 P0   | 数据持久化 |
| server/router/api/v1    | 1.7%  | 50%  | -48.3% | 🔴 P0   | API 入口   |
| server/router/api/v1/ai | 5.1%  | 50%  | -44.9% | 🔴 P0   | AI API     |
| server/runner/embedding | 23.4% | 50%  | -26.6% | 🟡 P1   | Embedding  |
| server/auth             | 27.1% | 50%  | -22.9% | 🟡 P1   | 认证       |
| ai/agents               | 28.2% | 50%  | -21.8% | 🟡 P1   | 代理核心   |

---

## Phase 1: store/db/postgres (0.9% → 30%)

### 现状分析

**已有测试**：
- `ai_block_test.go` - 覆盖 AI Block 基础功能
- `memo_filter_test.go` - 覆盖 Memo 过滤功能

**完全未测试的核心功能**（按业务重要性排序）：

| 文件                 | 功能          | 优先级 | 测试复杂度         |
| -------------------- | ------------- | ------ | ------------------ |
| `memo.go`            | Memo CRUD     | P0     | 中 - 需要 mock DB  |
| `ai_conversation.go` | AI 对话持久化 | P0     | 中                 |
| `schedule.go`        | 日程 CRUD     | P0     | 中                 |
| `user.go`            | 用户管理      | P1     | 低                 |
| `attachment.go`      | 附件管理      | P1     | 中                 |
| `postgres.go`        | DB 初始化     | P1     | 高 - 需要实际 DB   |
| `memo_embedding.go`  | 向量嵌入      | P1     | 高 - 需要 pgvector |
| `agent_stats.go`     | 代理统计      | P2     | 中                 |
| `activity.go`        | 活动记录      | P2     | 低                 |
| `router_feedback.go` | 路由反馈      | P2     | 低                 |

### 测试策略

#### 1. 创建基础测试工具 (`postgres_test.go`)

```go
package postgres

import (
    "testing"
    "time"

    "github.com/hrygo/divinesense/store"
)

// testDB 提供**隔离**的测试数据库连接
func setupTestDB(t *testing.T) *PostgresDB {
    t.Helper()
    // 使用环境变量控制：TEST_DB_URL
    // CI 中使用 docker-compose 的测试数据库
}

// teardown 清理测试数据
func teardownTestDB(t *testing.T, db *PostgresDB) {
    t.Helper()
    // 清理所有测试表，保持测试独立
}
```

#### 2. 核心功能测试清单

**memo.go 测试**：
- [ ] `CreateMemo` - 正常创建
- [ ] `CreateMemo` - 重复 ID 处理
- [ ] `GetMemo` - 存在/不存在
- [ ] `UpdateMemo` - 正常更新
- [ ] `DeleteMemo` - 软删除
- [ ] `ListMemos` - 分页
- [ ] `ListMemos` - 过滤条件

**schedule.go 测试**：
- [ ] `CreateSchedule` - 正常创建
- [ ] `GetSchedule` - 存在/不存在
- [ ] `UpdateSchedule` - 时间冲突处理
- [ ] `DeleteSchedule` - 级联删除
- [ ] `ListSchedules` - 时间范围查询

**ai_conversation.go 测试**：
- [ ] `CreateConversation` - 新建对话
- [ ] `SaveMessage` - 保存消息
- [ ] `GetConversationContext` - 获取上下文
- [ ] `DeleteConversation` - 级联删除

#### 3. 跳过集成测试标记

```go
func TestPostgresMemo_Integration(t *testing.T) {
    if testing.Short() {
        t.Skip("跳过集成测试 - CI 中使用 -short 标志")
    }
    // 需要 pgvector 的完整集成测试
}
```

---

## Phase 2: server/router/api/v1 (1.7% → 30%)

### 现状分析

**目录结构**：
```
server/router/api/v1/
├── handler.go        # 主处理器
├── ai/
│   └── handler.go    # AI 处理器
├── memo.go          # Memo API
├── schedule.go      # Schedule API
├── auth.go          # 认证 API
└── ...
```

### 测试策略

#### 1. 创建 HTTP 测试辅助工具

```go
package v1

import (
    "net/http"
    "net/http/httptest"
    "testing"
)

// TestApp 提供**最小化**的测试应用
func setupTestApp(t *testing.T) *echo.Echo {
    t.Helper()
    // 只注册被测试的路由，不依赖完整服务器
}

// mockAuth 提供模拟认证
func mockAuth(next echo.HandlerFunc) echo.HandlerFunc {
    return func(c echo.Context) error {
        c.Set("user_id", int32(1))
        return next(c)
    }
}
```

#### 2. API 端点测试清单

**memo.go 测试**：
- [ ] `POST /api/v1/memos` - 创建成功
- [ ] `POST /api/v1/memos` - 参数验证失败
- [ ] `GET /api/v1/memos/:id` - 存在
- [ ] `GET /api/v1/memos/:id` - 不存在返回 404
- [ ] `PATCH /api/v1/memos/:id` - 更新成功
- [ ] `DELETE /api/v1/memos/:id` - 删除成功

**schedule.go 测试**：
- [ ] `POST /api/v1/schedules` - 创建成功
- [ ] `POST /api/v1/schedules` - 时间格式验证
- [ ] `GET /api/v1/schedules` - 列表查询

---

## 🚀 AI Native 测试原则

### 1. 简洁性

**❌ 避免**：
```go
// 过度工程化的测试辅助
type TestCase struct {
    Name string
    Setup func() (*Context, error)
    Teardown func(*Context) error
    Assert func(*testing.T, *Context)
}
```

**✅ 推荐**：
```go
// 直接、清晰的测试
func TestMemo_Create_Success(t *testing.T) {
    db := setupTestDB(t)
    defer teardownTestDB(t, db)

    memo := &store.Memo{Content: "test"}
    err := db.CreateMemo(ctx, memo)

    if err != nil {
        t.Fatalf("CreateMemo failed: %v", err)
    }
}
```

### 2. 快速执行

- 使用 `testing.Short()` 跳过慢速测试
- 避免网络调用、文件 I/O
- 使用 mock 替代外部依赖

### 3. 独立性

- 每个测试独立运行
- 不依赖执行顺序
- 清理副作用

### 4. 可读性

- 测试名称描述清楚：`TestMemo_Create_Success`
- 断言消息明确：`expected memo.ID to be set`
- AAA 模式：Arrange → Act → Assert

---

## 📋 执行时间表

| 阶段     | 目标                       | 预计时间 | 覆盖率提升 |
| -------- | -------------------------- | -------- | ---------- |
| Week 1   | store/db/postgres → 30%    | 2-3 天   | +5%        |
| Week 1-2 | server/router/api/v1 → 30% | 2-3 天   | +3%        |
| Week 2   | server/auth → 50%          | 1-2 天   | +2%        |
| Week 2-3 | ai/agents → 50%            | 2-3 天   | +3%        |

---

## 🔧 CI 集成

### 快速测试（PR 检查）

```bash
# CI 中运行（~30秒）
make test-fast
# 等价于：
go test -short ./... -count=1
```

### 完整测试（主分支）

```bash
# 夜间运行（~5分钟）
make test-full
# 等价于：
go test ./... -count=1
```

---

## 📈 进度跟踪

### 当前状态

- [ ] Phase 1.1: store/db/postgres 基础测试
- [ ] Phase 1.2: store/db/postgres 集成测试
- [ ] Phase 2.1: server/router/api/v1 端点测试
- [ ] Phase 2.2: server/auth 测试

### 覆盖率目标

| 模块                 | 当前  | Phase 1 目标 | Phase 2 目标 |
| -------------------- | ----- | ------------ | ------------ |
| store/db/postgres    | 0.9%  | 30%          | 50%          |
| server/router/api/v1 | 1.7%  | 30%          | 50%          |
| server/auth          | 27.1% | 35%          | 50%          |
| ai/agents            | 28.2% | 35%          | 50%          |

---

*最后更新：2026-02-09*
