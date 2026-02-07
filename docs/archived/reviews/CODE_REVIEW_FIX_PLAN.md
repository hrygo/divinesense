# Code Review 问题修复计划

> 基于 main vs feat/ai-specs 分支的全面 Code Review
>
> **审查日期**: 2026-01-20
> **整体评分**: 7.6/10
> **问题总数**: 26 个（P0: 2, P1: 8, P2: 6, P3: 10）

---

## 📋 目录

- [P0 - 关键问题（必须修复）](#p0---关键问题必须修复)
- [P1 - 重要问题（强烈建议修复）](#p1---重要问题强烈建议修复)
- [P2 - 性能优化（建议改进）](#p2---性能优化建议改进)
- [P3 - 代码质量（可选改进）](#p3---代码质量可选改进)
- [测试验证清单](#测试验证清单)

---

## P0 - 关键问题（必须修复）

### P0-1: 修复 LLM 流式响应 Goroutine 泄漏

**优先级**: 🔴 紧急
**文件**: `plugin/ai/llm.go:92-121`
**预估时间**: 30 分钟

#### 问题描述

`ChatStream` 方法中启动的 goroutine 可能永远不会退出，导致资源泄漏：

1. 如果 `s.model.GenerateContent` 在 `ctx.Done()` 之后仍然阻塞，goroutine 将永远运行
2. 如果 `contentChan` 的接收方提前退出，goroutine 会在发送时永久阻塞
3. 缺少超时保护机制

#### 修复步骤

**步骤 1**: 修改 `plugin/ai/llm.go` 的 `ChatStream` 方法

```go
func (s *llmService) ChatStream(ctx context.Context, messages []Message) (<-chan string, <-chan error) {
	// 修改点 1: 添加缓冲，防止发送阻塞
	contentChan := make(chan string, 10)
	errChan := make(chan error, 1)

	go func() {
		defer close(contentChan)
		defer close(errChan)

		llmMessages := convertMessages(messages)

		// 修改点 2: 添加超时保护
		ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
		defer cancel()

		_, err := s.model.GenerateContent(ctx, llmMessages,
			llms.WithMaxTokens(s.maxTokens),
			llms.WithTemperature(float64(s.temperature)),
			llms.WithStreamingFunc(func(ctx context.Context, chunk []byte) error {
				// 修改点 3: 使用 select 检查 context 状态
				select {
				case contentChan <- string(chunk):
					return nil
				case <-ctx.Done():
					return ctx.Err()
				}
			}),
		)

		if err != nil {
			select {
			case errChan <- err:
			case <-ctx.Done():
				// Context 已取消，无法发送错误
			}
		}
	}()

	return contentChan, errChan
}
```

**步骤 2**: 添加测试验证

创建 `plugin/ai/llm_stream_test.go`:

```go
func TestLLMService_ChatStream_ContextCancellation(t *testing.T) {
	// ... 测试 context 取消时 goroutine 能正确退出
}

func TestLLMService_ChatStream_Timeout(t *testing.T) {
	// ... 测试超时保护
}
```

**步骤 3**: 运行测试验证

```bash
cd plugin/ai
go test -v -run TestLLMService_ChatStream
```

#### 验证标准

- ✅ Context 取消后 goroutine 能正确退出
- ✅ 超时后能返回错误
- ✅ 单元测试通过
- ✅ 无 goroutine 泄漏（使用 `runtime.NumGoroutine()` 检查）

---

### P0-2: Embedding Runner 添加 Context 取消检查

**优先级**: 🔴 紧急
**文件**: `server/runner/embedding/runner.go:58-86`
**预估时间**: 20 分钟

#### 问题描述

`processNewMemos` 方法在批量处理 embedding 时未检查 `ctx.Done()`，可能导致：
- 服务关闭时长时间阻塞
- 资源无法及时释放
- 优雅关闭失败

#### 修复步骤

**步骤 1**: 修改 `server/runner/embedding/runner.go` 的 `processNewMemos` 方法

```go
func (r *Runner) processNewMemos(ctx context.Context) {
	memos, err := r.findMemosWithoutEmbedding(ctx)
	if err != nil {
		slog.Error("failed to find memos without embedding", "error", err)
		return
	}

	if len(memos) == 0 {
		return
	}

	slog.Info("processing memos for embedding", "count", len(memos))

	for i := 0; i < len(memos); i += r.batchSize {
		// 修改点: 添加 context 取消检查
		select {
		case <-ctx.Done():
			slog.Info("embedding processing cancelled", "processed", i, "total", len(memos))
			return
		default:
			// 继续处理
		}

		end := i + r.batchSize
		if end > len(memos) {
			end = len(memos)
		}
		batch := memos[i:end]

		if err := r.processBatch(ctx, batch); err != nil {
			slog.Error("failed to process batch", "error", err, "batch", fmt.Sprintf("%d-%d", i, end))
			continue
		}
		slog.Info("batch processed", "count", len(batch), "progress", fmt.Sprintf("%d/%d", end, len(memos)))
	}
}
```

**步骤 2**: 同时检查 `processBatch` 方法

在 `processBatch` 方法开始处也添加检查：

```go
func (r *Runner) processBatch(ctx context.Context, memos []*store.Memo) error {
	// 添加 context 检查
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// ... 原有逻辑
}
```

**步骤 3**: 添加测试

```go
func TestRunner_ProcessNewMemos_ContextCancellation(t *testing.T) {
	// 测试 context 取消时能正确停止
}
```

#### 验证标准

- ✅ 服务关闭时能立即停止处理
- ✅ Context 取消日志正确输出
- ✅ 优雅关闭测试通过

---

## P1 - 重要问题（强烈建议修复）

### P1-1: 统一后端时区处理为 UTC

**优先级**: 🟠 高
**文件**: `plugin/ai/schedule/parser.go`
**预估时间**: 45 分钟

#### 修复步骤

**步骤 1**: 修改 LLM prompt，明确要求 UTC 时间

```go
systemPrompt := fmt.Sprintf(`You are an intelligent schedule parser...

Current Time (UTC): %s
User Timezone: %s

IMPORTANT RULES:
1. Always return start_time and end_time in UTC timezone
2. Format: YYYY-MM-DD HH:mm:ss (no timezone suffix)
3. Calculate times in UTC, then convert to the format above`,
	now.UTC().Format("2006-01-02 15:04:05"),
	p.location.String())
```

**步骤 2**: 修改时间解析逻辑

```go
parseTime := func(timeStr string) (int64, error) {
	// 统一解析为 UTC
	t, err := time.Parse("2006-01-02 15:04:05", timeStr)
	if err != nil {
		return 0, fmt.Errorf("failed to parse time: %w", err)
	}
	return t.Unix(), nil
}
```

**步骤 3**: 添加时间合理性验证

```go
// 验证解析的时间不在过去太久
if startTs < now.Add(-24*time.Hour).Unix() {
	return nil, fmt.Errorf("parsed start time is too far in the past: %d", startTs)
}

// 验证结束时间不早于开始时间
if endTs < startTs {
	return nil, fmt.Errorf("end time %d is before start time %d", endTs, startTs)
}
```

#### 验证标准

- ✅ 时区转换测试通过
- ✅ 跨时区用户看到正确时间
- ✅ 边界情况处理正确（夏令时、月末等）

---

### P1-2: 优化日程实例展开性能

**优先级**: 🟠 高
**文件**: `server/router/api/v1/schedule_service.go`
**预估时间**: 40 分钟

#### 修复步骤

**步骤 1**: 根据 PageSize 动态限制实例数

```go
// 在 ListSchedules 方法中
maxTotalInstances := 100 // 默认值
if req.PageSize > 0 {
	maxTotalInstances = int(req.PageSize) * 2 // 留一些余地
}
if maxTotalInstances > 500 {
	maxTotalInstances = 500 // 硬限制
}
```

**步骤 2**: 添加截断标志到响应

修改 `proto/api/v1/schedule_service.proto`:

```proto
message ListSchedulesResponse {
  repeated Schedule schedules = 1;
  bool truncated = 2;  // 添加此字段
}
```

**步骤 3**: 在达到限制时设置标志

```go
if len(expandedSchedules) >= maxTotalInstances {
	response.Truncated = true
	break
}
```

**步骤 4**: 添加日志警告

```go
if len(expandedSchedules) >= maxTotalInstances {
	slog.Warn("schedule instance expansion truncated",
		"count", len(expandedSchedules),
		"limit", maxTotalInstances)
}
```

#### 验证标准

- ✅ 大量日程时响应时间 < 1s
- ✅ 前端能正确显示截断提示
- ✅ 分页功能正常工作

---

### P1-3: 向量搜索添加输入验证

**优先级**: 🟠 高
**文件**: `server/router/api/v1/ai_service.go`
**预估时间**: 30 分钟

#### 修复步骤

**步骤 1**: 添加常量定义

```go
const (
	maxQueryLength = 1000
	minQueryLength = 2
)
```

**步骤 2**: 实现输入清理函数

```go
func sanitizeQuery(query string) string {
	// 移除多余空白
	query = strings.TrimSpace(query)
	query = strings.Join(strings.Fields(query), " ")

	// 移除控制字符
	query = strings.Map(func(r rune) rune {
		if r < 32 && r != '\n' && r != '\t' {
			return -1
		}
		return r
	}, query)

	return query
}
```

**步骤 3**: 在 `SemanticSearch` 方法中添加验证

```go
func (s *AIService) SemanticSearch(ctx context.Context, req *v1pb.SemanticSearchRequest) (*v1pb.SemanticSearchResponse, error) {
	// 验证非空
	if req.Query == "" {
		return nil, status.Errorf(codes.InvalidArgument, "query is required")
	}

	// 验证长度
	if len(req.Query) > maxQueryLength {
		return nil, status.Errorf(codes.InvalidArgument,
			"query too long: maximum %d characters, got %d", maxQueryLength, len(req.Query))
	}

	if len(strings.TrimSpace(req.Query)) < minQueryLength {
		return nil, status.Errorf(codes.InvalidArgument,
			"query too short: minimum %d characters", minQueryLength)
	}

	// 清理输入
	query := sanitizeQuery(req.Query)

	// Vectorize the query
	queryVector, err := s.EmbeddingService.Embed(ctx, query)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to process query")
	}

	// ... 后续逻辑
}
```

**步骤 4**: 添加测试

```go
func TestAIService_SemanticSearch_InputValidation(t *testing.T) {
	tests := []struct {
		name    string
		query   string
		wantErr bool
	}{
		{"empty query", "", true},
		{"too long", strings.Repeat("a", 1001), true},
		{"too short", "a", true},
		{"valid query", "test query", false},
		{"with extra spaces", "  test   query  ", false},
	}
	// ... 测试逻辑
}
```

#### 验证标准

- ✅ 空查询被拒绝
- ✅ 超长查询被拒绝
- ✅ 输入被正确清理
- ✅ 单元测试通过

---

### P1-4: SQL 查询使用占位符

**优先级**: 🟠 高
**文件**: `store/db/postgres/memo_embedding.go`
**预估时间**: 25 分钟

#### 修复步骤

**步骤 1**: 修改 `VectorSearch` 方法

```go
// 第 122-133 行
query := `
	SELECT
		memo.id, memo.creator_id, memo.created_ts, memo.updated_ts,
		memo.content, memo.visibility, memo.tags, memo.pinned,
		1 - (memo.embedding <=> $1) AS similarity
	FROM memo
	WHERE memo.creator_id = $2
		AND memo.embedding_model = $3
		AND memo.embedding IS NOT NULL
		AND (memo.visibility = 'PUBLIC' OR memo.creator_id = $2)
		AND memo.embedding <=> $1 < $4
	ORDER BY memo.embedding <=> $1
	LIMIT $5
`

rows, err := d.db.QueryContext(ctx, query,
	vector,           // $1
	opts.UserID,      // $2
	model,            // $3
	threshold,        // $4
	limit,            // $5 修改点：使用占位符
)
```

**步骤 2**: 修改 `FindMemosWithoutEmbedding` 方法

```go
// 第 204-210 行
query := `
	SELECT id, content
	FROM memo
	WHERE creator_id = $1
		AND (embedding IS NULL OR embedding_model != $2)
	ORDER BY created_ts DESC
	LIMIT $3
`

rows, err := d.db.QueryContext(ctx, query,
	find.UserID,    // $1
	find.Model,     // $2
	limit,          // $3 修改点：使用占位符
)
```

**步骤 3**: 同样修改 MySQL 和 SQLite 版本

- `store/db/mysql/memo_embedding.go`
- `store/db/sqlite/memo_embedding.go`

**步骤 4**: 运行测试

```bash
go test ./store/db/... -v -run TestMemoEmbedding
```

#### 验证标准

- ✅ 所有数据库层的测试通过
- ✅ SQL 注入扫描工具无警告
- ✅ 查询结果正确

---

### P1-5: 前端添加时区支持

**优先级**: 🟠 高
**文件**: `web/src/components/AIChat/ScheduleInput.tsx`
**预估时间**: 1 小时

#### 修复步骤

**步骤 1**: 安装 dayjs 时区插件

```bash
cd web
npm install dayjs
```

**步骤 2**: 配置 dayjs 插件

创建 `web/src/utils/dayjs.ts`:

```typescript
import dayjs from 'dayjs';
import utc from 'dayjs/plugin/utc';
import timezone from 'dayjs/plugin/timezone';

dayjs.extend(utc);
dayjs.extend(timezone);

export default dayjs;
```

**步骤 3**: 获取用户时区

在用户 store 中添加时区设置：

```typescript
// web/src/store/user.ts
export const useUserStore = create<UserState>((set) => ({
  // ... 其他状态
  timezone: Intl.DateTimeFormat().resolvedOptions().timeZone || 'Asia/Shanghai',
  setTimezone: (timezone: string) => set({ timezone }),
}));
```

**步骤 4**: 修改 ScheduleInput 组件

```typescript
import dayjs from '@/utils/dayjs';

const ScheduleInput = ({ ... }) => {
  const userTimezone = useUserStore(state => state.timezone) || 'Asia/Shanghai';

  // 显示时间时转换到用户时区
  const formatDateTime = (timestamp: bigint) => {
    return dayjs.unix(Number(timestamp))
      .tz(userTimezone)
      .format('YYYY-MM-DDTHH:mm');
  };

  // 提交时转换回 UTC
  const handleTimeChange = (field: 'startTs' | 'endTs', value: string) => {
    const ts = BigInt(dayjs.tz(value, userTimezone).unix());
    setParsedSchedule({ ...parsedSchedule, [field]: ts });
  };

  // ...
};
```

**步骤 5**: 添加时区选择器到用户设置

```typescript
// web/src/components/UserSettings.tsx
const timezoneOptions = [
  'Asia/Shanghai',
  'Asia/Tokyo',
  'America/New_York',
  'Europe/London',
  // ... 更多时区
];

<Select value={timezone} onValueChange={setTimezone}>
  {timezoneOptions.map(tz => (
    <option key={tz} value={tz}>{tz}</option>
  ))}
</Select>
```

#### 验证标准

- ✅ 不同时区用户看到正确时间
- ✅ 时区切换后时间正确更新
- ✅ 提交到后端的时间是 UTC

---

### P1-6: Reranker HTTP 添加超时

**优先级**: 🟠 高
**文件**: `plugin/ai/reranker.go`
**预估时间**: 15 分钟

#### 修复步骤

**步骤 1**: 修改 `NewRerankerService` 函数

```go
func NewRerankerService(cfg *RerankerConfig) RerankerService {
	return &rerankerService{
		enabled: cfg.Enabled,
		apiKey:  cfg.APIKey,
		baseURL: strings.TrimSuffix(cfg.BaseURL, "/"),
		model:   cfg.Model,
		client: &http.Client{
			// 修改点：添加超时
			Timeout: 30 * time.Second,
			// 添加连接池配置
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
				// 禁用 HTTP/2（避免连接复用问题）
				ForceAttemptHTTP2:   false,
			},
		},
	}
}
```

**步骤 2**: 添加超时测试

```go
func TestRerankerService_Timeout(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping timeout test in short mode")
	}

	// 启动一个延迟响应的服务
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(35 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := &RerankerConfig{
		Enabled: true,
		APIKey:  "test-key",
		BaseURL: server.URL,
		Model:   "test-model",
	}
	svc := NewRerankerService(cfg)

	// 应该超时
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
	defer cancel()

	docs := []Document{{Content: "test"}}
	_, err := svc.Rerank(ctx, docs, "test")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "timeout")
}
```

#### 验证标准

- ✅ 超时测试通过
- ✅ 正常请求不受影响
- ✅ 错误日志记录超时事件

---

### P1-7: 创建数据库迁移回滚脚本

**优先级**: 🟠 高
**文件**: `store/migration/`
**预估时间**: 50 分钟

#### 修复步骤

**步骤 1**: 创建 PostgreSQL 回滚脚本

创建 `store/migration/postgres/0.26/down/1__add_schedule.sql`:

```sql
-- ===== Down Migration for 0.26 =====
-- 回滚日程表

-- 删除更新时间触发器
DROP TRIGGER IF EXISTS trigger_schedule_updated_ts ON schedule;
DROP FUNCTION IF EXISTS update_schedule_updated_ts();

-- 删除索引
DROP INDEX IF EXISTS idx_schedule_uid;
DROP INDEX IF EXISTS idx_schedule_start_ts;
DROP INDEX IF EXISTS idx_schedule_creator_status;
DROP INDEX IF EXISTS idx_schedule_creator_start;
DROP INDEX IF EXISTS idx_schedule_updated_ts;

-- 删除表（级联删除提醒）
DROP TABLE IF EXISTS schedule_reminder;
DROP TABLE IF EXISTS schedule;

-- 记录日志
DO $$
BEGIN
	RAISE NOTICE 'Down migration 0.26 completed: schedule tables dropped';
END $$;
```

**步骤 2**: 创建 MySQL 回滚脚本

创建 `store/migration/mysql/0.26/down/1__add_schedule.sql`:

```sql
-- 删除触发器
DROP TRIGGER IF EXISTS trigger_schedule_updated_ts ON schedule;
DROP FUNCTION IF EXISTS update_schedule_updated_ts;

-- 删除索引
DROP INDEX IF EXISTS idx_schedule_uid ON schedule;
DROP INDEX IF EXISTS idx_schedule_start_ts ON schedule;
DROP INDEX IF EXISTS idx_schedule_creator_status ON schedule;
DROP INDEX IF EXISTS idx_schedule_creator_start ON schedule;

-- 删除表
DROP TABLE IF EXISTS schedule_reminder;
DROP TABLE IF EXISTS schedule;
```

**步骤 3**: 创建 SQLite 回滚脚本

创建 `store/migration/sqlite/0.26/down/1__add_schedule.sql`:

```sql
-- SQLite 不支持 DROP TRIGGER IF EXISTS，直接重建表
DROP TABLE IF EXISTS schedule_reminder;
DROP TABLE IF EXISTS schedule;
```

**步骤 4**: 创建 pgvector 回滚脚本

创建 `store/migration/postgres/0.30/down/1__add_pgvector.sql`:

```sql
-- 删除向量相似度搜索的函数
DROP FUNCTION IF EXISTS memo_similarity_search CASCADE;

-- 删除 pgvector 扩展（谨慎！这会影响所有使用 pgvector 的表）
-- 注意：只在没有其他表使用 pgvector 时才执行
-- DROP EXTENSION IF EXISTS vector CASCADE;

-- 而是只删除索引
DROP INDEX IF EXISTS memo_embedding_idx;
DROP INDEX IF EXISTS memo_embedding_model_idx;

-- 记录日志
DO $$
BEGIN
	RAISE NOTICE 'Down migration 0.30 completed: vector indexes dropped';
	RAISE NOTICE 'To drop vector extension, manually run: DROP EXTENSION IF EXISTS vector CASCADE;';
END $$;
```

**步骤 5**: 测试回滚

```bash
# PostgreSQL
go run cmd/memos/main.go --mode migration --database postgres --down

# MySQL
go run cmd/memos/main.go --mode migration --database mysql --down

# SQLite
go run cmd/memos/main.go --mode migration --database sqlite --down
```

#### 验证标准

- ✅ 回滚脚本无语法错误
- ✅ 回滚后数据库 schema 正确
- ✅ 能重新应用迁移

---

### P1-8: AI 聊天添加速率限制

**优先级**: 🟠 高
**文件**: `server/router/api/v1/`
**预估时间**: 1.5 小时

#### 修复步骤

**步骤 1**: 创建速率限制中间件

创建 `server/middleware/rate_limit.go`:

```go
package middleware

import (
	"context"
	"sync"
	"time"

	"golang.org/x/time/rate"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type RateLimiter struct {
	mu     sync.RWMutex
	limits map[string]*rate.Limiter
}

func NewRateLimiter() *RateLimiter {
	return &RateLimiter{
		limits: make(map[string]*rate.Limiter),
	}
}

func (rl *RateLimiter) getLimiter(userID string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	if limiter, ok := rl.mu.limits[userID]; ok {
		return limiter
	}

	// 每秒 10 个请求，允许突发 20 个
	limiter := rate.NewLimiter(rate.Every(time.Second/10), 20)
	rl.mu.limits[userID] = limiter
	return limiter
}

func (rl *RateLimiter) Allow(userID string) bool {
	return rl.getLimiter(userID).Allow()
}

// 全局限流器
var globalAILimiter = NewRateLimiter()
```

**步骤 2**: 修改 `ai_service.go`

```go
func (s *AIService) ChatWithMemos(req *v1pb.ChatWithMemosRequest, stream v1pb.AIService_ChatWithMemosServer) error {
	ctx := stream.Context()

	// 获取当前用户
	user, err := getCurrentUser(ctx, s.Store)
	if err != nil {
		return status.Errorf(codes.Unauthenticated, "unauthorized")
	}

	// 检查速率限制
	if !globalAILimiter.Allow(user.ID) {
		return status.Errorf(codes.ResourceExhausted,
			"rate limit exceeded: please wait before making another request")
	}

	// 检查每日配额
	quota, err := s.Store.CheckUserQuota(ctx, user.ID, "ai_chat_daily")
	if err != nil {
		slog.Error("failed to check quota", "user", user.ID, "error", err)
		// 继续处理，但记录错误
	}

	if quota != nil && quota.Remaining <= 0 {
		return status.Errorf(codes.ResourceExhausted,
			"daily quota exceeded: you have used all your AI chat credits for today")
	}

	// ... 继续处理聊天请求

	// 成功后扣减配额
	if quota != nil {
		if err := s.Store.DecrementQuota(ctx, user.ID, "ai_chat_daily", 1); err != nil {
			slog.Error("failed to decrement quota", "user", user.ID, "error", err)
		}
	}

	return nil
}
```

**步骤 3**: 添加配额表到数据库

创建迁移脚本 `store/migration/postgres/0.31/1__add_quota.sql`:

```sql
CREATE TABLE IF NOT EXISTS user_quota (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL,
    quota_type VARCHAR(50) NOT NULL,
    daily_limit INTEGER NOT NULL DEFAULT 100,
    used_today INTEGER NOT NULL DEFAULT 0,
    reset_date DATE NOT NULL DEFAULT CURRENT_DATE,
    created_ts BIGINT NOT NULL EXTRACT(EPOCH FROM NOW()) * 1000,
    updated_ts BIGINT NOT NULL EXTRACT(EPOCH FROM NOW()) * 1000,
    UNIQUE(user_id, quota_type)
);

CREATE INDEX idx_user_quota_user_type ON user_quota(user_id, quota_type);

COMMENT ON TABLE user_quota IS 'User API quota tracking';
COMMENT ON COLUMN user_quota.quota_type IS 'Quota type: ai_chat_daily, semantic_search_daily, etc.';
```

**步骤 4**: 在 store 中添加配额查询方法

修改 `store/user.go`:

```go
func (s *Store) CheckUserQuota(ctx context.Context, userID int32, quotaType string) (*Quota, error) {
	// 实现配额检查逻辑
}

func (s *Store) DecrementQuota(ctx context.Context, userID int32, quotaType string, amount int32) error {
	// 实现配额扣减逻辑
}
```

**步骤 5**: 添加测试

```go
func TestRateLimiter(t *testing.T) {
	limiter := NewRateLimiter()

	// 快速连续请求
	for i := 0; i < 25; i++ {
		allowed := limiter.Allow("user1")
		if i < 20 {
			assert.True(t, allowed, "request %d should be allowed", i)
		} else {
			assert.False(t, allowed, "request %d should be rate limited", i)
		}
	}
}
```

#### 验证标准

- ✅ 速率限制正常工作
- ✅ 配额扣减正确
- ✅ 用户收到友好的错误消息
- ✅ 单元测试通过

---

## P2 - 性能优化（建议改进）

### P2-1: 向量查询缓存

**优先级**: 🟡 中
**预估时间**: 2 小时

#### 实现建议

```go
// 创建缓存层
type SemanticSearchCache struct {
	cache *cache.Cache
	ttl   time.Duration
}

func NewSemanticSearchCache() *SemanticSearchCache {
	return &SemanticSearchCache{
		cache: cache.New(5*time.Minute, 10*time.Minute),
		ttl:   5 * time.Minute,
	}
}

func (c *SemanticSearchCache) Get(userID int32, query string) (*SearchResult, bool) {
	key := fmt.Sprintf("search:%d:%s", userID, hashQuery(query))
	if val, found := c.cache.Get(key); found {
		return val.(*SearchResult), true
	}
	return nil, false
}

func (c *SemanticSearchCache) Set(userID int32, query string, result *SearchResult) {
	key := fmt.Sprintf("search:%d:%s", userID, hashQuery(query))
	c.cache.Set(key, result, c.ttl)
}
```

---

### P2-2: Embedding 批大小动态调整

**优先级**: 🟡 中
**预估时间**: 1.5 小时

#### 实现建议

```go
type Runner struct {
	// ... 其他字段
	batchSize    int
	minBatchSize int
	maxBatchSize int
	lastDuration time.Duration
}

func (r *Runner) adjustBatchSize() {
	targetDuration := 3 * time.Second

	if r.lastDuration < targetDuration/2 {
		// 响应很快，增加批大小
		r.batchSize = min(r.batchSize*2, r.maxBatchSize)
	} else if r.lastDuration > targetDuration*2 {
		// 响应慢，减少批大小
		r.batchSize = max(r.batchSize/2, r.minBatchSize)
	}

	slog.Info("adjusted batch size", "new_size", r.batchSize, "last_duration", r.lastDuration)
}
```

---

### P2-3: 前端虚拟化

**优先级**: 🟡 中
**预估时间**: 1 小时

#### 实现建议

```bash
npm install react-virtuoso
```

```typescript
import { Virtuoso } from 'react-virtuoso';

<Virtuoso
  style={{ height: '100%' }}
  data={messages}
  itemContent={(index, message) => (
    <MessageBubble key={index} message={message} />
  )}
/>
```

---

### P2-4: 延迟展开重复日程

**优先级**: 🟡 中
**预估时间**: 1 小时

#### 实现建议

修改 API，添加 `expand_instances` 参数：

```proto
message ListSchedulesRequest {
  // ... 其他字段
  bool expand_instances = 10;  // 是否展开重复实例
}
```

默认返回重复规则，前端按需展开：

```typescript
const expandInstances = (schedule: Schedule, startDate: Date, endDate: Date) => {
  if (!schedule.recurrenceRule) return [schedule];

  const rule = JSON.parse(schedule.recurrenceRule);
  // 前端计算实例
  return generateInstances(rule, schedule.startTs, startDate, endDate);
};
```

---

### P2-5: 数据库连接池调优

**优先级**: 🟡 中
**预估时间**: 30 分钟

#### 实现建议

```go
// store/db/postgres/common.go
db.SetMaxOpenConns(10)  // 2C 环境，降低并发连接
db.SetMaxIdleConns(5)
db.SetConnMaxLifetime(1 * time.Hour)
db.SetConnMaxIdleTime(10 * time.Minute)
```

---

### P2-6: 图片懒加载

**优先级**: 🟡 中
**预估时间**: 20 分钟

#### 实现建议

```typescript
<img
  src={src}
  loading="lazy"
  decoding="async"
  alt={alt}
/>
```

---

## P3 - 代码质量（可选改进）

### P3-1: 定义常量

创建 `plugin/ai/constants.go`:

```go
const (
	// Embedding
	DefaultEmbeddingModel     = "text-embedding-3-small"
	DefaultEmbeddingDimension = 1024

	// Reranker
	DefaultRerankerThreshold = 0.5
	DefaultRerankerTopK      = 100

	// Schedule
	MaxReminders          = 100
	MaxScheduleTitleLength = 200
	DefaultQueryWindowDays = 30
)
```

---

### P3-2: 错误码国际化

创建 `server/errors/codes.go`:

```go
package errors

const (
	ErrScheduleTitleRequired    = "SCHEDULE_001"
	ErrScheduleInvalidName      = "SCHEDULE_002"
	ErrScheduleTimeConflict     = "SCHEDULE_003"
	ErrRateLimitExceeded        = "RATE_001"
	ErrQuotaExceeded            = "QUOTA_001"
)

// 前端根据错误码显示国际化消息
```

---

### P3-3: 更严格的类型定义

```go
type RecurrenceType string

const (
	RecurrenceTypeDaily   RecurrenceType = "daily"
	RecurrenceTypeWeekly  RecurrenceType = "weekly"
	RecurrenceTypeMonthly RecurrenceType = "monthly"
)

func (rt RecurrenceType) IsValid() bool {
	switch rt {
	case RecurrenceTypeDaily, RecurrenceTypeWeekly, RecurrenceTypeMonthly:
		return true
	default:
		return false
	}
}

type RecurrenceRule struct {
	Type     RecurrenceType `json:"type"`
	Interval int            `json:"interval"`
	// ...
}
```

---

### P3-4: 提高测试覆盖率

目标：70%+ 覆盖率

```bash
# 安装覆盖率工具
go install github.com/glebarez/go_plugin@latest

# 生成覆盖率报告
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

---

### P3-5: 统一日志规范

```go
// 使用结构化日志
slog.Info("schedule created",
	"user_id", user.ID,
	"schedule_id", schedule.ID,
	"title", schedule.Title,
)

// 避免使用
fmt.Printf("Schedule created: %v\n", schedule)
```

---

### P3-6: 添加代码注释

```go
// ScheduleParser converts natural language input into structured schedule information.
//
// It uses LLM to understand complex patterns like "every Monday at 3pm" or "next Friday morning".
// The parser is timezone-aware and converts all times to UTC for storage.
//
// Example:
//   parser := NewParser(llmService, "Asia/Shanghai")
//   result, err := parser.Parse(ctx, "明天下午3点开会")
type ScheduleParser struct {
	// ...
}
```

---

### P3-7: 添加 Proto 验证

安装 `protoc-gen-validate`:

```bash
go install github.com/envoyproxy/protoc-gen-validate@latest
```

添加验证规则：

```proto
import "validate/validate.proto";

message Schedule {
  string title = 1 [(validate.rules).string = {min_len: 1, max_len: 200}];
  int64 start_ts = 2 [(validate.rules).int64.gt = 0];
  int64 end_ts = 3 [(validate.rules).int64.gt = 0];
}
```

---

### P3-8: 消除代码重复

提取辅助函数：

```go
// marshalReminders converts protobuf reminders to JSON
func marshalReminders(reminders []*v1pb.Reminder) (string, error) {
	if len(reminders) == 0 {
		return "", nil
	}
	data, err := json.Marshal(reminders)
	if err != nil {
		return "", fmt.Errorf("failed to marshal reminders: %w", err)
	}
	return string(data), nil
}

// unmarshalReminders converts JSON to protobuf reminders
func unmarshalReminders(data string) ([]*v1pb.Reminder, error) {
	if data == "" {
		return nil, nil
	}
	var reminders []*v1pb.Reminder
	if err := json.Unmarshal([]byte(data), &reminders); err != nil {
		return nil, fmt.Errorf("failed to unmarshal reminders: %w", err)
	}
	return reminders, nil
}
```

---

### P3-9: 清理未使用代码

```bash
# Go
go vet ./...
goimports -w .

# TypeScript
npm run lint
npx eslint --fix web/src/
```

---

### P3-10: 改进配置管理

使用配置映射表：

```go
var providerConfigMap = map[string]struct {
	apiKeyField   *string
	baseURLField  *string
	modelField    *string
}{
	"siliconflow": {
		apiKeyField:  &profile.AISiliconFlowAPIKey,
		baseURLField: &profile.AISiliconFlowBaseURL,
		modelField:   &profile.AISiliconFlowModel,
	},
	"openai": {
		apiKeyField:  &profile.AIOpenAIAPIKey,
		baseURLField: &profile.AIOpenAIBaseURL,
		modelField:   &profile.AIOpenAIModel,
	},
	// ...
}
```

---

## 测试验证清单

### 单元测试

- [ ] `plugin/ai/llm_test.go` - 所有测试通过
- [ ] `plugin/ai/reranker_test.go` - 超时测试通过
- [ ] `plugin/ai/schedule/recurrence_test.go` - 时区测试通过
- [ ] `server/router/api/v1/ai_service_test.go` - 输入验证测试通过
- [ ] `server/router/api/v1/schedule_service_test.go` - 分页测试通过

### 集成测试

- [ ] `store/test/memo_embedding_test.go` - 向量搜索测试通过
- [ ] 数据库迁移回滚测试通过
- [ ] API 端到端测试通过

### 性能测试

- [ ] 速率限制功能正常
- [ ] 配额扣减正确
- [ ] 并发场景测试通过
- [ ] 内存泄漏检测通过

### 前端测试

- [ ] TypeScript 编译无错误
- [ ] ESLint 检查通过
- [ ] 时区切换测试通过
- [ ] 虚拟滚动测试通过

---

## 修复进度跟踪

| ID | 问题描述 | 优先级 | 状态 | 负责人 | 预估时间 |
|----|---------|--------|------|--------|----------|
| P0-1 | Goroutine 泄漏 | 🔴 | 待修复 | - | 30分钟 |
| P0-2 | Context 取消 | 🔴 | 待修复 | - | 20分钟 |
| P1-1 | 时区处理 | 🟠 | 待修复 | - | 45分钟 |
| P1-2 | 实例展开 | 🟠 | 待修复 | - | 40分钟 |
| P1-3 | 输入验证 | 🟠 | 待修复 | - | 30分钟 |
| P1-4 | SQL 占位符 | 🟠 | 待修复 | - | 25分钟 |
| P1-5 | 前端时区 | 🟠 | 待修复 | - | 1小时 |
| P1-6 | HTTP 超时 | 🟠 | 待修复 | - | 15分钟 |
| P1-7 | 回滚脚本 | 🟠 | 待修复 | - | 50分钟 |
| P1-8 | 速率限制 | 🟠 | 待修复 | - | 1.5小时 |

**总计预估时间**: P0 (50分钟) + P1 (6小时) + P2 (6.5小时) + P3 (4小时) = **约 17 小时**

---

## 修复建议优先级

**第一阶段（本周）**: 修复所有 P0 问题
- P0-1: Goroutine 泄漏
- P0-2: Context 取消

**第二阶段（本月）**: 修复所有 P1 问题
- P1-1 至 P1-8

**第三阶段（下季度）**: 实施 P2 性能优化
- 根据实际性能测试结果选择

**第四阶段（持续）**: P3 代码质量改进
- 作为技术债务逐步偿还

---

## 注意事项

1. **每个修复前先创建分支**: `fix/issue-{ID}-{short-description}`
2. **每个修复都需要测试**: 单元测试 + 集成测试
3. **修复后更新文档**: 如有 API 变更
4. **提交前代码审查**: 至少一人审查
5. **逐步合并**: 不要一次性合并所有修复

---

**文档版本**: 1.0
**最后更新**: 2026-01-20
**维护者**: 开发团队
