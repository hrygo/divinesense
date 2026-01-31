# DivineSense 会话管理进化路线规划

> **版本**: v1.1 (更正)
> **日期**: 2026-01-31
> **作者**: Claude Opus 4.5
> **状态**: 待评审
> **参考**: [openclaw](https://github.com/openclaw/openclaw) 会话管理架构

---

## 一、执行摘要

本文档基于 DivineSense 现有会话管理机制（详见 `SESSION_MANAGEMENT_REPORT.md`）与 openclaw 项目的深度对比分析，规划 DivineSense 会话管理功能的进化路线。

**重要更正**: 经深入代码调研，DivineSense 已具备基础的**滑动窗口**和**优先级截断**机制，但在**工具结果智能修剪**和**会话压缩**方面与 openclaw 存在显著差距。

---

## 二、架构对比分析

### 2.1 核心架构对比

| 维度 | DivineSense | openclaw | 差距评估 |
|:-----|:-----------|:----------|:---------|
| **会话存储** | PostgreSQL + JSONB | JSONL 文件 + sessions.json | 不同方案，各有利弊 |
| **固定会话** | ✅ `(userID << 8) \| offset` | ✅ `agent:<agentId>:<mainKey>` | 相当 |
| **会话隔离** | 按代理类型 | 按 dmScope + channel | openclaw 更灵活 |
| **消息修剪** | ✅ 滑动窗口 (20 条) | ✅ 会话修剪 + 压缩 | **实现方式不同** |
| **Token 预算** | ✅ 动态分配 | ✅ 动态估算 | 相当 |
| **工具结果修剪** | ❌ 简化摘要 | ✅ 智能修剪 | **显著差距** |
| **会话压缩** | ❌ 无 | ✅ LLM 驱动 | **显著差距** |
| **会话重置** | ❌ 无用户命令 | ✅ `/new`, `/reset` | 可学习 |
| **上下文检查** | ❌ 无 API | ✅ `/context`, `/status` | 可学习 |

### 2.2 DivineSense 现有功能详解

#### 已实现的修剪机制

**1. 滑动窗口** (`plugin/ai/session/recovery.go`):
```go
const MaxMessagesPerSession = 20

func (r *SessionRecovery) AppendMessage(...) error {
    session.Messages = append(session.Messages, *msg)
    // 应用滑动窗口
    if len(session.Messages) > MaxMessagesPerSession {
        session.Messages = session.Messages[len(session.Messages)-MaxMessagesPerSession:]
    }
}
```

**2. 优先级截断** (`plugin/ai/context/priority.go`):
```go
func (r *PriorityRanker) RankAndTruncate(segments []*ContextSegment, budget int) []*ContextSegment {
    // 按优先级排序
    sort.Slice(sorted, func(i, j int) bool {
        return sorted[i].Priority > sorted[j].Priority
    })
    // 截断超出预算的段
    if usedTokens+seg.TokenCost <= budget {
        result = append(result, seg)
    } else if remaining >= MinSegmentTokens {
        truncated := truncateToTokens(seg.Content, remaining)
        // ...
    }
}
```

**3. 工具调用摘要** (`plugin/ai/agent/context.go:348-364`):
```go
// 简化的工具摘要: "Assistant (Action: tool_name): [Success/Fail]"
toolsUsed := make([]string, 0, len(turn.ToolCalls))
for _, tc := range turn.ToolCalls {
    status := "OK"
    if !tc.Success {
        status = "Failed"
    }
    toolsUsed = append(toolsUsed, fmt.Sprintf("%s (%s)", tc.Tool, status))
}
sb.WriteString(fmt.Sprintf("System: Agent used tools: %s\n", strings.Join(toolsUsed, ", ")))
```

### 2.3 差距分析

#### DivineSense 的局限性

| 局面 | DivineSense 实现 | openclaw 实现 | 改进空间 |
|:-----|:---------------|:--------------|:---------|
| **修剪粒度** | 整体滑动窗口 (消息级) | 工具结果级修剪 | 可降低 token 使用 |
| **修剪策略** | 简单截断 | 软修剪(头尾)+ 硬清除 | 可保留关键信息 |
| **修剪触发** | 固定 20/10 条消息 | TTL 感知 + 配置化 | 更智能 |
| **工具结果处理** | 简化为 "tool_name (OK)" | 完整截断/清除 | 更精细控制 |
| **会话压缩** | ❌ 无 | ✅ LLM 驱动摘要 | 可延长会话寿命 |

---

## 三、进化路线规划

### Phase 1: 基础增强（短期，1-2 周）

**目标**: 补齐用户体验差距，添加诊断能力

| 功能 | 优先级 | 复杂度 | 依赖 |
|:-----|:-------|:-------|:-----|
| **会话重置命令** | P0 | 低 | 无 |
| **会话状态 API** | P0 | 低 | 无 |
| **会话标题生成** | P1 | 中 | AI |
| **上下文检查命令** | P1 | 低 | 无 |

**详细设计**:

#### 1.1 会话重置命令
```go
// plugin/ai/session/commands.go
type ResetCommand struct {
    ConversationID int64
    UserID         int32
    Mode           string // "soft" (保留历史) | "hard" (清空)
}

// 触发词: /new, /reset
```

#### 1.2 会话状态 API
```protobuf
message SessionStatusResponse {
  int32 message_count = 1;
  int64 created_ts = 2;
  int64 updated_ts = 3;
  int32 estimated_tokens = 4;  // 当前上下文估算
  int32 max_messages = 5;      // 配置的最大消息数
  string reset_policy = 6;
}
```

---

### Phase 2: 智能修剪（中期，2-4 周）

**目标**: 实现 openclaw 风格的工具结果智能修剪

| 功能 | 优先级 | 复杂度 | 依赖 |
|:-----|:-------|:-------|:-----|
| **工具结果修剪** | P0 | 中 | Phase 1 |
| **软/硬修剪模式** | P1 | 中 | Phase 2 其他 |
| **TTL 感知修剪** | P2 | 中 | Phase 2 其他 |
| **Token 估算优化** | P1 | 低 | 无 |

**详细设计**:

#### 2.1 工具结果修剪 (Session Pruning)
```go
// plugin/ai/session/pruning.go
type PruningConfig struct {
    Mode             string // "off" | "cache-ttl" | "always"
    TTL              time.Duration
    KeepLastN        int  // 保留最近 N 条 assistant 消息后的工具结果
    SoftTrimRatio    float64 // 30% - 保留头尾比例
    HardClearRatio   float64 // 50% - 超过此比例完全清除
    MinToolChars     int    // 小于此值不修剪 (default: 50000)
}

type ToolResultPruner struct {
    config PruningConfig
}

func (p *ToolResultPruner) PruneToolResults(messages []Message) []Message {
    // 1. 检查是否需要修剪（TTL 超时 或 超过大小限制）
    // 2. 软修剪：保留头 + 尾，中间用 ... 替代
    // 3. 硬清除：完全替换为 [Content cleared]
}
```

#### 2.2 与现有系统集成
```go
// 在 plugin/ai/agent/context.go 中集成
func (c *ConversationContext) ToHistoryPrompt(pruneConfig *PruningConfig) string {
    // 现有逻辑...

    // 应用工具结果修剪
    if pruneConfig != nil && pruneConfig.Mode != "off" {
        c.applyToolResultPruning(pruneConfig)
    }
}
```

---

### Phase 3: 会话压缩（中期，2-4 周）

**目标**: 实现 LLM 驱动的会话摘要压缩

| 功能 | 优先级 | 复杂度 | 依赖 |
|:-----|:-------|:-------|:-----|
| **会话压缩服务** | P0 | 高 | Phase 2 |
| **压缩命令** | P0 | 低 | Phase 3 压缩服务 |
| **压缩历史管理** | P1 | 中 | Phase 3 压缩服务 |

**详细设计**:

#### 3.1 会话压缩服务
```go
// plugin/ai/session/compaction.go
type CompactionService struct {
    llmClient LLMClient
}

func (c *CompactionService) CompactSession(ctx context.Context, sessionID string) error {
    // 1. 读取会话历史
    // 2. 使用 LLM 生成摘要（保留关键信息、决策、结论）
    // 3. 将摘要作为 system 消息插入
    // 4. 归档或删除旧消息
}
```

#### 3.2 压缩触发条件
- 会话消息数 > 50 条
- 估算 token > 模型窗口的 70%
- 用户主动触发 `/compact`
- 定时自动压缩（可配置）

---

### Phase 4: 高级特性（长期，1-2 月）

| 功能 | 优先级 | 复杂度 | 依赖 |
|:-----|:-------|:-------|:-----|
| **会话分支** | P2 | 高 | Phase 3 |
| **会话导出** | P2 | 中 | Phase 1 |
| **会话分享** | P3 | 高 | Phase 1 |

---

## 四、技术建议

### 4.1 数据库演进 (Phase 1)

**现有表** (已有):
```sql
CREATE TABLE conversation_context (
  id            SERIAL PRIMARY KEY,
  session_id    VARCHAR(64) NOT NULL UNIQUE,
  user_id       INTEGER NOT NULL,
  agent_type    VARCHAR(20) NOT NULL,
  context_data  JSONB NOT NULL DEFAULT '{}',
  created_ts    BIGINT NOT NULL,
  updated_ts    BIGINT NOT NULL
);
```

**新增字段** (Phase 1):
```sql
ALTER TABLE conversation_context ADD COLUMN title VARCHAR(512);
ALTER TABLE conversation_context ADD COLUMN reset_policy VARCHAR(32) DEFAULT 'daily';
ALTER TABLE conversation_context ADD COLUMN expires_at BIGINT;
ALTER TABLE conversation_context ADD COLUMN message_count INTEGER DEFAULT 0;
ALTER TABLE conversation_context ADD COLUMN estimated_tokens INTEGER DEFAULT 0;
ALTER TABLE conversation_context ADD COLUMN tool_call_count INTEGER DEFAULT 0;
```

### 4.2 API 设计 (Phase 1)

```protobuf
// proto/api/v1/ai_service.proto

service AIService {
  // 现有接口...

  // 会话管理接口
  rpc ResetSession(ResetSessionRequest) returns (ResetSessionResponse);
  rpc GetSessionStatus(SessionStatusRequest) returns (SessionStatusResponse);
  rpc CompactSession(CompactSessionRequest) returns (CompactSessionResponse);
}

message ResetSessionRequest {
  int64 conversation_id = 1;
  string mode = 2; // "soft" | "hard"
}
```

---

## 五、更正总结

### DivineSense 已有但未充分利用的功能

| 功能 | 现有实现 | 改进方向 |
|:-----|:---------|:---------|
| **滑动窗口** | 20 条消息硬编码 | 可配置化 |
| **优先级截断** | 基础实现 | 可扩展到工具结果级 |
| **Token 预算** | 固定比例 | 可根据会话类型动态调整 |
| **工具摘要** | 简化为名称列表 | 可选择性保留详细信息 |

### DivineSense 缺少的关键功能

1. **工具结果智能修剪** - 最大差距
2. **会话压缩 (Compaction)** - 次大差距
3. **用户可见的会话控制** - 可学习
4. **会话诊断命令** - 可学习

---

## 六、参考资料

- DivineSense 会话管理报告: `docs/research/SESSION_MANAGEMENT_REPORT.md`
- openclaw 项目: https://github.com/openclaw/openclaw
- openclaw 会话文档: `~/openclaw/docs/concepts/session.md`
- openclaw 会话修剪: `~/openclaw/docs/concepts/session-pruning.md`
- DivineSense 现有代码:
  - `plugin/ai/session/recovery.go`
  - `plugin/ai/context/priority.go`
  - `plugin/ai/context/budget.go`
  - `plugin/ai/agent/context.go`

---

*文档维护：随项目演进持续更新*
