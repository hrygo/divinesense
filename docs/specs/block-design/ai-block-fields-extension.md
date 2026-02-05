# Spec: ai_block 字段扩展设计

> **Status**: 📝 Proposed | **Version**: 1.0 | **Created**: 2026-02-05
> **Priority**: P1 (Enhancement) | **Parent Spec**: [Unified Block Model](./unified-block-model.md)

---

## 1. 背景与目标

### 1.1 当前问题

`ai_block` 表 (Unified Block Model v2) 已实现核心对话持久化功能，但在以下方面存在不足：

| 问题 | 描述 | 影响 |
|:-----|:-----|:-----|
| **Token 使用不可查询** | Token 数据存储在 `session_stats` JSONB 中 | 无法按 token 排序/筛选/聚合 |
| **成本数据不精确** | 成本存储为浮点数（USD） | 存在精度误差，不适合财务计算 |
| **模型版本无追踪** | 无法记录使用的 LLM 模型版本 | 无法分析不同模型效果 |
| **无用户反馈机制** | 用户无法对 AI 回复评分 | 无法收集质量数据 |
| **错误信息缺失** | `status=error` 时无详细错误描述 | 调试困难，用户体验差 |
| **无重新生成追踪** | 无法记录 Block 被重新生成的次数 | 无法分析重试模式 |
| **无软删除支持** | Block 删除即永久消失 | 无法实现"回收站"功能 |

### 1.2 设计目标

| 目标 | 描述 | 优先级 |
|:-----|:-----|:-------|
| **成本可计算** | Token 使用独立存储，支持精确查询 | P0 |
| **质量可追踪** | 用户反馈 + 模型版本，支持质量分析 | P1 |
| **错误可调试** | 详细错误信息，便于排查问题 | P1 |
| **数据可恢复** | 软删除支持，实现回收站 | P2 |

---

## 2. 字段设计

### 2.1 新增字段概览

| 字段名 | 类型 | 默认值 | 约束 | 说明 |
|:-------|:-----|:-------|:-----|:-----|
| `token_usage` | jsonb | `'{}'::jsonb` | NOT NULL | Token 使用明细 |
| `cost_estimate` | bigint | `0` | NOT NULL | 成本估算（毫厘） |
| `model_version` | text | `''` | | LLM 模型版本标识 |
| `user_feedback` | integer | | 1-5 或 NULL | 用户评分 |
| `error_message` | text | | | 错误详情 |
| `regeneration_count` | integer | `0` | NOT NULL | 重新生成次数 |
| `archived_at` | bigint | | | 软删除时间戳 |

### 2.2 详细设计

#### 2.2.1 token_usage (JSONB)

**目的**: 独立存储 Token 使用数据，支持查询和聚合。

**结构**:
```json
{
  "prompt_tokens": 150,
  "completion_tokens": 300,
  "total_tokens": 450,
  "cache_read_tokens": 50,
  "cache_write_tokens": 0
}
```

**字段说明**:
| 字段 | 类型 | 说明 |
|:-----|:-----|:-----|
| `prompt_tokens` | integer | 输入 Token 数 |
| `completion_tokens` | integer | 输出 Token 数 |
| `total_tokens` | integer | 总 Token 数 |
| `cache_read_tokens` | integer | 缓存命中 Token 数（如 Claude Prompt Caching） |
| `cache_write_tokens` | integer | 缓存写入 Token 数 |

**索引**:
```sql
-- 支持 GIN 索引查询 JSONB 内部字段
CREATE INDEX idx_ai_block_token_usage ON ai_block USING GIN (token_usage);

-- 或支持特定字段的查询
CREATE INDEX idx_ai_block_total_tokens ON ai_block
  ((token_usage->>'total_tokens')::bigint) DESC;
```

**查询示例**:
```sql
-- 查询 Token 使用最多的 Blocks
SELECT id, (token_usage->>'total_tokens')::int as tokens
FROM ai_block
ORDER BY tokens DESC LIMIT 10;

-- 统计总 Token 使用
SELECT
  SUM((token_usage->>'total_tokens')::int) as total_tokens,
  AVG((token_usage->>'total_tokens')::int) as avg_tokens
FROM ai_block
WHERE status = 'completed';
```

#### 2.2.2 cost_estimate (BIGINT)

**目的**: 精确存储成本估算，避免浮点误差。

**单位**: 毫厘 (milli-cents, 1/1000 美分)
- `$0.001` = `1000` 毫厘
- `$0.01` = `10000` 毫厘
- `$1.00` = `1000000` 毫厘

**计算公式**:
```
cost_estimate (milli-cents) = (total_tokens / 1M) * price_per_1M_tokens * 1000000
```

**示例**:
| 模型 | 价格 | 输入 1000 tokens | 输出 1000 tokens |
|:-----|:-----|:----------------|:----------------|
| DeepSeek V3 | $0.14/1M 输入, $0.28/1M 输出 | 140 毫厘 | 280 毫厘 |

**优势**:
- 整数运算，无浮点误差
- 适合数据库聚合（SUM, AVG）
- 前端显示时除以 1000000

**索引**:
```sql
CREATE INDEX idx_ai_block_cost_estimate ON ai_block(cost_estimate DESC);
```

#### 2.2.3 model_version (TEXT)

**目的**: 记录使用的 LLM 模型版本。

**格式**: `{provider}/{model_name}`
- `deepseek/deepseek-chat`
- `openai/gpt-4o`
- `anthropic/claude-3-5-sonnet-20241022`

**用途**:
- 分析不同模型的效果
- 追踪模型版本更新
- A/B 测试不同模型

**索引**:
```sql
CREATE INDEX idx_ai_block_model_version ON ai_block(model_version);
```

#### 2.2.4 user_feedback (INTEGER)

**目的**: 收集用户对 AI 回复的质量反馈。

**取值**: `1` | `2` | `3` | `4` | `5` | `NULL`

**约束**:
```sql
CONSTRAINT chk_user_feedback_range
  CHECK (user_feedback IS NULL OR (user_feedback >= 1 AND user_feedback <= 5))
```

**UI 设计**:
```
┌─────────────────────────────────────────────────────────┐
│  🔴🔴🔴🔴⚪  这条回复有帮助吗？                          │
│  [👍 有帮助]  [👎 没帮助]  [重新生成]                     │
└─────────────────────────────────────────────────────────┘
```

#### 2.2.5 error_message (TEXT)

**目的**: 当 `status='error'` 时，存储详细错误信息。

**示例**:
```
"Rate limit exceeded: 120 requests per minute exceeded"
"Invalid API key: please check your configuration"
"Timeout: LLM provider did not respond within 30s"
```

**用途**:
- 用户可见的错误提示
- 后端调试和日志分析
- 错误分类统计

#### 2.2.6 regeneration_count (INTEGER)

**目的**: 记录 Block 被用户"重新生成"的次数。

**用途**:
- 分析用户不满意率
- 优化提示词
- 检测模型问题

**统计查询**:
```sql
-- 查询重新生成最多的 Blocks
SELECT id, regeneration_count
FROM ai_block
WHERE regeneration_count > 0
ORDER BY regeneration_count DESC;

-- 计算重新生成率
SELECT
  COUNT(*) FILTER (WHERE regeneration_count > 0)::float / COUNT(*) as regeneration_rate
FROM ai_block
WHERE status = 'completed';
```

#### 2.2.7 archived_at (BIGINT)

**目的**: 软删除支持，实现回收站功能。

**行为**:
- `NULL`: 正常状态
- `非 NULL`: 已归档（时间戳）

**查询**:
```sql
-- 未归档的 Blocks
SELECT * FROM ai_block WHERE archived_at IS NULL;

-- 已归档的 Blocks
SELECT * FROM ai_block WHERE archived_at IS NOT NULL;

-- 恢复归档
UPDATE ai_block SET archived_at = NULL WHERE id = ?;
```

**索引**:
```sql
CREATE INDEX idx_ai_block_archived_at ON ai_block(archived_at)
  WHERE archived_at IS NOT NULL;
```

---

## 3. 数据库迁移

### 3.1 迁移脚本

```sql
-- =============================================================================
-- Migration: ai_block 字段扩展
-- Version: 20260205000002
-- Author: Claude
-- =============================================================================

-- 1. 添加 token_usage 字段
ALTER TABLE ai_block
  ADD COLUMN IF NOT EXISTS token_usage jsonb NOT NULL DEFAULT '{
    "prompt_tokens": 0,
    "completion_tokens": 0,
    "total_tokens": 0,
    "cache_read_tokens": 0,
    "cache_write_tokens": 0
  }'::jsonb;

COMMENT ON COLUMN ai_block.token_usage IS 'Token 使用明细 (prompt/completion/cache)';

-- 2. 添加 cost_estimate 字段
ALTER TABLE ai_block
  ADD COLUMN IF NOT EXISTS cost_estimate bigint NOT NULL DEFAULT 0;

COMMENT ON COLUMN ai_block.cost_estimate IS '成本估算（毫厘，1/1000 美分）';

-- 3. 添加 model_version 字段
ALTER TABLE ai_block
  ADD COLUMN IF NOT EXISTS model_version text;

COMMENT ON COLUMN ai_block.model_version IS 'LLM 模型版本 (如 deepseek/deepseek-chat)';

-- 4. 添加 user_feedback 字段
ALTER TABLE ai_block
  ADD COLUMN IF NOT EXISTS user_feedback integer;

COMMENT ON COLUMN ai_block.user_feedback IS '用户评分 (1-5, NULL 表示未评分)';

-- 5. 添加 error_message 字段
ALTER TABLE ai_block
  ADD COLUMN IF NOT EXISTS error_message text;

COMMENT ON COLUMN ai_block.error_message IS '错误详情（当 status=error 时填充）';

-- 6. 添加 regeneration_count 字段
ALTER TABLE ai_block
  ADD COLUMN IF NOT EXISTS regeneration_count integer NOT NULL DEFAULT 0;

COMMENT ON COLUMN ai_block.regeneration_count IS '重新生成次数';

-- 7. 添加 archived_at 字段
ALTER TABLE ai_block
  ADD COLUMN IF NOT EXISTS archived_at bigint;

COMMENT ON COLUMN ai_block.archived_at IS '软删除时间戳（NULL 表示正常）';

-- 8. 添加约束
ALTER TABLE ai_block
  ADD CONSTRAINT IF NOT EXISTS chk_user_feedback_range
  CHECK (user_feedback IS NULL OR (user_feedback >= 1 AND user_feedback <= 5));

-- 9. 创建索引
CREATE INDEX IF NOT EXISTS idx_ai_block_total_tokens ON ai_block
  ((token_usage->>'total_tokens')::bigint DESC)
  WHERE (token_usage->>'total_tokens')::bigint > 0;

CREATE INDEX IF NOT EXISTS idx_ai_block_cost_estimate ON ai_block(cost_estimate DESC)
  WHERE cost_estimate > 0;

CREATE INDEX IF NOT EXISTS idx_ai_block_model_version ON ai_block(model_version)
  WHERE model_version IS NOT NULL AND model_version != '';

CREATE INDEX IF NOT EXISTS idx_ai_block_archived_at ON ai_block(archived_at)
  WHERE archived_at IS NOT NULL;

-- 10. 更新触发器（在 status 变为 error 时，必须填充 error_message）
CREATE OR REPLACE FUNCTION validate_error_status()
RETURNS TRIGGER AS $$
BEGIN
  IF NEW.status = 'error' AND (NEW.error_message IS NULL OR NEW.error_message = '') THEN
    RAISE EXCEPTION 'error_message cannot be empty when status is error';
  END IF;
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_validate_error_status
  BEFORE UPDATE ON ai_block
  FOR EACH ROW
  WHEN (NEW.status = 'error' AND OLD.status != 'error')
  EXECUTE FUNCTION validate_error_status();
```

### 3.2 回滚脚本

```sql
-- =============================================================================
-- Rollback: ai_block 字段扩展
-- =============================================================================

DROP TRIGGER IF EXISTS trigger_validate_error_status ON ai_block;
DROP FUNCTION IF EXISTS validate_error_status();

DROP INDEX IF EXISTS idx_ai_block_archived_at;
DROP INDEX IF EXISTS idx_ai_block_model_version;
DROP INDEX IF EXISTS idx_ai_block_cost_estimate;
DROP INDEX IF EXISTS idx_ai_block_total_tokens;

ALTER TABLE ai_block DROP CONSTRAINT IF EXISTS chk_user_feedback_range;

ALTER TABLE ai_block DROP COLUMN IF EXISTS archived_at;
ALTER TABLE ai_block DROP COLUMN IF EXISTS regeneration_count;
ALTER TABLE ai_block DROP COLUMN IF EXISTS error_message;
ALTER TABLE ai_block DROP COLUMN IF EXISTS user_feedback;
ALTER TABLE ai_block DROP COLUMN IF EXISTS model_version;
ALTER TABLE ai_block DROP COLUMN IF EXISTS cost_estimate;
ALTER TABLE ai_block DROP COLUMN IF EXISTS token_usage;
```

---

## 4. API 变更

### 4.1 Proto 更新

```protobuf
// proto/api/v1/ai_service.proto

message TokenUsage {
  int32 prompt_tokens = 1;
  int32 completion_tokens = 2;
  int32 total_tokens = 3;
  int32 cache_read_tokens = 4;
  int32 cache_write_tokens = 5;
}

message AIBlock {
  // ... 现有字段 ...

  // 新增字段
  TokenUsage token_usage = 20;
  int64 cost_estimate = 21;        // 毫厘
  string model_version = 22;
  int32 user_feedback = 23;        // 1-5, 0 表示未评分
  string error_message = 24;
  int32 regeneration_count = 25;
  int64 archived_at = 26;
}

message UpdateBlockRequest {
  // ... 现有字段 ...

  // 新增可更新字段
  int32 user_feedback = 10;
  string error_message = 11;
  int32 regeneration_count = 12;
  int64 archived_at = 13;
}
```

### 4.2 Store 接口更新

```go
// store/block.go

type AIBlock struct {
    // ... 现有字段 ...

    // 新增字段
    TokenUsage       *TokenUsage
    CostEstimate     int64    // 毫厘
    ModelVersion     *string
    UserFeedback     *int32   // 1-5, nil 表示未评分
    ErrorMessage     *string
    RegenerationCount int32
    ArchivedAt       *int64
}

type TokenUsage struct {
    PromptTokens     int32
    CompletionTokens int32
    TotalTokens      int32
    CacheReadTokens  int32
    CacheWriteTokens int32
}

type UpdateBlock struct {
    // ... 现有字段 ...

    // 新增可更新字段
    UserFeedback     *int32
    ErrorMessage     *string
    RegenerationCount *int32
    ArchivedAt       *int64
}
```

---

## 5. 前端变更

### 5.1 类型定义

```typescript
// web/src/types/block.ts

export interface TokenUsage {
  promptTokens: number;
  completionTokens: number;
  totalTokens: number;
  cacheReadTokens: number;
  cacheWriteTokens: number;
}

export interface AIBlock {
  // ... 现有字段 ...

  // 新增字段
  tokenUsage?: TokenUsage;
  costEstimate?: number;      // 毫厘
  modelVersion?: string;
  userFeedback?: 1 | 2 | 3 | 4 | 5;  // undefined 表示未评分
  errorMessage?: string;
  regenerationCount?: number;
  archivedAt?: number;        // undefined 表示正常
}

// 辅助函数：将毫厘转换为美元
export function milliCentsToUSD(milliCents: number): string {
  return `$${(milliCents / 1000000).toFixed(4)}`;
}

// 辅助函数：格式化 Token 显示
export function formatTokenUsage(tokens: number): string {
  if (tokens >= 1000000) return `${(tokens / 1000000).toFixed(1)}M`;
  if (tokens >= 1000) return `${(tokens / 1000).toFixed(1)}K`;
  return tokens.toString();
}
```

### 5.2 UI 组件

**成本徽章**:
```tsx
// BlockCostBadge.tsx
interface BlockCostBadgeProps {
  costEstimate: number;
  tokenUsage: TokenUsage;
}

export function BlockCostBadge({ costEstimate, tokenUsage }: BlockCostBadgeProps) {
  return (
    <div className="flex items-center gap-2 text-xs text-muted-foreground">
      <span>{formatTokenUsage(tokenUsage.totalTokens)} tokens</span>
      <span>·</span>
      <span>{milliCentsToUSD(costEstimate)}</span>
    </div>
  );
}
```

**用户反馈组件**:
```tsx
// UserFeedbackRating.tsx
interface UserFeedbackRatingProps {
  blockId: string;
  currentRating?: 1 | 2 | 3 | 4 | 5;
  onRate: (rating: 1 | 2 | 3 | 4 | 5) => void;
}

export function UserFeedbackRating({ currentRating, onRate }: UserFeedbackRatingProps) {
  return (
    <div className="flex items-center gap-1">
      {[1, 2, 3, 4, 5].map((star) => (
        <button
          key={star}
          onClick={() => onRate(star as 1 | 2 | 3 | 4 | 5)}
          className={cn(
            "text-lg transition-colors",
            currentRating && star <= currentRating
              ? "text-yellow-400 fill-yellow-400"
              : "text-gray-300"
          )}
        >
          <Star className="h-4 w-4" />
        </button>
      ))}
    </div>
  );
}
```

---

## 6. 实施计划

| 阶段 | 任务 | 投入 | 依赖 |
|:-----|:-----|:-----|:-----|
| **Phase 1** | 数据库迁移 | 0.5 人天 | 无 |
| **Phase 2** | Proto + Store 接口 | 1 人天 | Phase 1 |
| **Phase 3** | Chat Handler 集成 | 1.5 人天 | Phase 2 |
| **Phase 4** | 前端类型 + Hooks | 1 人天 | Phase 2 |
| **Phase 5** | UI 组件（成本徽章、反馈评分） | 1.5 人天 | Phase 4 |
| **Phase 6** | 测试与验证 | 1 人天 | Phase 5 |

**总计**: 6.5 人天

---

## 7. 验收标准

### 7.1 功能验收

| 场景 | 验收标准 |
|:-----|:---------|
| **Token 追踪** | 完成 Block 后 `token_usage` 正确填充 |
| **成本计算** | `cost_estimate` 精确到毫厘，无浮点误差 |
| **模型版本** | 每个 Block 记录使用的 LLM 模型 |
| **用户评分** | 用户可对 Block 评分 1-5 星 |
| **错误信息** | `status=error` 时 `error_message` 不为空 |
| **重新生成** | 重新生成时 `regeneration_count++` |
| **软删除** | 删除 Block 仅设置 `archived_at`，数据保留 |

### 7.2 查询验收

```sql
-- Token 使用统计
SELECT
  model_version,
  SUM((token_usage->>'total_tokens')::int) as total_tokens,
  SUM(cost_estimate) / 1000000.0 as total_cost_usd
FROM ai_block
WHERE archived_at IS NULL
GROUP BY model_version;

-- 用户反馈统计
SELECT
  user_feedback,
  COUNT(*) as block_count,
  AVG((token_usage->>'total_tokens')::int) as avg_tokens
FROM ai_block
WHERE user_feedback IS NOT NULL
GROUP BY user_feedback
ORDER BY user_feedback DESC;

-- 重新生成分析
SELECT
  regeneration_count,
  COUNT(*) as block_count,
  AVG(user_feedback) as avg_rating
FROM ai_block
WHERE regeneration_count > 0
GROUP BY regeneration_count
ORDER BY regeneration_count DESC;
```

---

## 8. 风险与缓解

| 风险 | 影响 | 缓解措施 |
|:-----|:-----|:---------|
| **存储开销** | JSONB 字段增加存储 | 定期清理归档数据，使用 TOAST |
| **查询性能** | JSONB 查询较慢 | 添加表达式索引 |
| **数据一致性** | Token 与成本计算错误 | 后端统一计算逻辑，添加单元测试 |
| **前端复杂性** | 新增字段增加 UI 复杂度 | 渐进式添加，可选显示 |

---

## 9. 附录

### 9.1 成本计算参考

| 模型 | 输入价格 | 输出价格 | 1000 tokens 成本 |
|:-----|:---------|:---------|:----------------|
| DeepSeek V3 | $0.14/1M | $0.28/1M | $0.00042 (420 毫厘) |
| GPT-4o | $2.50/1M | $10.00/1M | $0.0125 (12500 毫厘) |
| Claude 3.5 Sonnet | $3.00/1M | $15.00/1M | $0.018 (18000 毫厘) |

### 9.2 变更记录

| 日期 | 版本 | 变更内容 |
|:-----|:-----|:---------|
| 2026-02-05 | v1.0 | 初始版本 |

---

*Spec Created: 2026-02-05*
*Related Issue: 待创建*
