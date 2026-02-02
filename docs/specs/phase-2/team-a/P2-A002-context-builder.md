# P2-A002: 上下文增强构建器

> **状态**: 🔲 待开发  
> **优先级**: P1 (重要)  
> **投入**: 3 人天  
> **负责团队**: 团队 A  
> **Sprint**: Sprint 3

---

## 1. 目标与背景

### 1.1 核心目标

构建智能上下文组装器，将短期记忆、长期记忆、检索结果、用户偏好统一编排，为 LLM 提供最优上下文窗口。

### 1.2 用户价值

- 对话连贯性提升（记忆跨会话）
- 个性化回答（融入用户偏好）
- 信息精准度提升（相关记忆优先）

### 1.3 技术价值

- 统一上下文管理入口
- Token 使用效率优化 30%+
- 为多 Agent 协作提供标准化上下文

---

## 2. 依赖关系

### 2.1 前置依赖

- [x] P1-A001: 轻量记忆系统（短期+长期记忆）
- [x] P1-A005: 通用缓存层（缓存构建结果）

### 2.2 并行依赖

- P2-A001: Self-RAG 检索优化（可并行）

### 2.3 后续依赖

- P2-B001: 用户习惯学习（使用上下文）
- P3-B001: 预测性交互（依赖上下文）

---

## 3. 功能设计

### 3.1 架构图

```
                    上下文构建流程
┌────────────────────────────────────────────────────────────┐
│                                                            │
│                  ┌─────────────────────┐                   │
│                  │   ContextBuilder    │                   │
│                  └─────────────────────┘                   │
│                            │                               │
│          ┌─────────────────┼─────────────────┐            │
│          │                 │                 │            │
│          ▼                 ▼                 ▼            │
│   ┌──────────────┐ ┌──────────────┐ ┌──────────────┐      │
│   │  短期记忆层   │ │  长期记忆层   │ │  检索结果层   │      │
│   │              │ │              │ │              │      │
│   │ • 最近10轮   │ │ • 情景记忆   │ │ • RAG结果    │      │
│   │ • 滑动窗口   │ │ • 用户偏好   │ │ • 相关笔记   │      │
│   │ • ~2K tokens│ │ • ~500 tokens│ │ • ~1K tokens │      │
│   └──────────────┘ └──────────────┘ └──────────────┘      │
│          │                 │                 │            │
│          └─────────────────┼─────────────────┘            │
│                            ▼                               │
│                  ┌─────────────────────┐                   │
│                  │   Token 预算分配    │                   │
│                  │   (总计 4K tokens)  │                   │
│                  └─────────────────────┘                   │
│                            │                               │
│                            ▼                               │
│                  ┌─────────────────────┐                   │
│                  │  优先级排序 + 截断   │                   │
│                  └─────────────────────┘                   │
│                            │                               │
│                            ▼                               │
│                  ┌─────────────────────┐                   │
│                  │  最终 Prompt 组装   │                   │
│                  └─────────────────────┘                   │
│                                                            │
└────────────────────────────────────────────────────────────┘
```

### 3.2 核心接口定义

```go
// plugin/ai/context/builder.go

type ContextBuilder interface {
    // 构建完整上下文
    Build(ctx context.Context, req *ContextRequest) (*ContextResult, error)
    
    // 获取上下文统计
    GetStats() *ContextStats
}

type ContextRequest struct {
    UserID          int32
    SessionID       string
    CurrentQuery    string
    AgentType       string           // memo/schedule/amazing
    RetrievalResults []*SearchResult  // RAG 检索结果
    MaxTokens       int              // 总 token 预算 (默认 4096)
}

type ContextResult struct {
    SystemPrompt    string           // 系统提示词
    ConversationContext string       // 对话上下文
    RetrievalContext    string       // 检索上下文
    UserPreferences     string       // 用户偏好摘要
    TotalTokens     int
    TokenBreakdown  *TokenBreakdown
}

type TokenBreakdown struct {
    SystemPrompt    int
    ShortTermMemory int
    LongTermMemory  int
    Retrieval       int
    UserPrefs       int
}
```

### 3.3 Token 预算分配策略

```go
// plugin/ai/context/budget.go

type TokenBudget struct {
    Total           int
    SystemPrompt    int  // 固定 500
    ShortTermMemory int  // 动态 40%
    LongTermMemory  int  // 动态 15%
    Retrieval       int  // 动态 35%
    UserPrefs       int  // 固定 10%
}

func AllocateBudget(total int, hasRetrieval bool) *TokenBudget {
    budget := &TokenBudget{
        Total:        total,
        SystemPrompt: 500,  // 固定
        UserPrefs:    int(float64(total) * 0.10),
    }
    
    remaining := total - budget.SystemPrompt - budget.UserPrefs
    
    if hasRetrieval {
        // 有检索结果时的分配
        budget.ShortTermMemory = int(float64(remaining) * 0.40)
        budget.LongTermMemory = int(float64(remaining) * 0.15)
        budget.Retrieval = int(float64(remaining) * 0.45)
    } else {
        // 无检索时，更多分配给记忆
        budget.ShortTermMemory = int(float64(remaining) * 0.55)
        budget.LongTermMemory = int(float64(remaining) * 0.30)
        budget.Retrieval = 0
    }
    
    return budget
}
```

### 3.4 上下文优先级排序

```go
// plugin/ai/context/priority.go

type ContextPriority int

const (
    PrioritySystem      ContextPriority = 100  // 系统提示词最高
    PriorityUserQuery   ContextPriority = 90   // 当前查询
    PriorityRecentTurns ContextPriority = 80   // 最近 3 轮对话
    PriorityRetrieval   ContextPriority = 70   // 检索结果
    PriorityEpisodic    ContextPriority = 60   // 情景记忆
    PriorityPreferences ContextPriority = 50   // 用户偏好
    PriorityOlderTurns  ContextPriority = 40   // 较早对话
)

type ContextSegment struct {
    Content   string
    Priority  ContextPriority
    TokenCost int
    Source    string  // "short_term", "long_term", "retrieval"
}

// 按优先级排序并截断到预算内
func PrioritizeAndTruncate(segments []*ContextSegment, budget int) []*ContextSegment {
    // 按优先级降序排列
    sort.Slice(segments, func(i, j int) bool {
        return segments[i].Priority > segments[j].Priority
    })
    
    var result []*ContextSegment
    usedTokens := 0
    
    for _, seg := range segments {
        if usedTokens+seg.TokenCost <= budget {
            result = append(result, seg)
            usedTokens += seg.TokenCost
        } else {
            // 尝试截断
            remaining := budget - usedTokens
            if remaining > 100 { // 至少保留 100 tokens
                truncated := truncateToTokens(seg.Content, remaining)
                result = append(result, &ContextSegment{
                    Content:   truncated,
                    Priority:  seg.Priority,
                    TokenCost: remaining,
                    Source:    seg.Source,
                })
            }
            break
        }
    }
    
    return result
}
```

### 3.5 短期记忆提取

```go
// plugin/ai/context/short_term.go

type ShortTermExtractor struct {
    memoryService MemoryService
}

func (e *ShortTermExtractor) Extract(ctx context.Context, sessionID string, maxTurns int) ([]*Message, error) {
    // 获取最近 N 轮对话
    messages, err := e.memoryService.GetRecentMessages(ctx, sessionID, maxTurns)
    if err != nil {
        return nil, err
    }
    
    // 按时间正序排列
    sort.Slice(messages, func(i, j int) bool {
        return messages[i].Timestamp.Before(messages[j].Timestamp)
    })
    
    return messages, nil
}

// 格式化为对话格式
func FormatConversation(messages []*Message) string {
    var sb strings.Builder
    for _, msg := range messages {
        if msg.Role == "user" {
            sb.WriteString(fmt.Sprintf("用户: %s\n", msg.Content))
        } else {
            sb.WriteString(fmt.Sprintf("助手: %s\n", msg.Content))
        }
    }
    return sb.String()
}
```

### 3.6 长期记忆提取

```go
// plugin/ai/context/long_term.go

type LongTermExtractor struct {
    memoryService MemoryService
}

func (e *LongTermExtractor) Extract(ctx context.Context, userID int32, query string) (*LongTermContext, error) {
    // 1. 获取相关情景记忆
    episodes, err := e.memoryService.SearchEpisodicMemory(ctx, userID, query, 3)
    if err != nil {
        return nil, err
    }
    
    // 2. 获取用户偏好
    prefs, err := e.memoryService.GetUserPreferences(ctx, userID)
    if err != nil {
        // 偏好可选，不影响主流程
        prefs = &UserPreferences{}
    }
    
    return &LongTermContext{
        EpisodicMemories: episodes,
        Preferences:      prefs,
    }, nil
}

// 格式化情景记忆
func FormatEpisodes(episodes []*EpisodicMemory) string {
    if len(episodes) == 0 {
        return ""
    }
    
    var sb strings.Builder
    sb.WriteString("### 相关历史记录\n")
    for _, ep := range episodes {
        sb.WriteString(fmt.Sprintf("- [%s] %s\n", 
            ep.Timestamp.Format("01-02"), 
            ep.Summary))
    }
    return sb.String()
}

// 格式化用户偏好
func FormatPreferences(prefs *UserPreferences) string {
    var parts []string
    
    if prefs.Timezone != "" {
        parts = append(parts, fmt.Sprintf("时区: %s", prefs.Timezone))
    }
    if prefs.DefaultDuration > 0 {
        parts = append(parts, fmt.Sprintf("默认会议时长: %d分钟", prefs.DefaultDuration))
    }
    if len(prefs.PreferredTimes) > 0 {
        parts = append(parts, fmt.Sprintf("偏好时间: %s", strings.Join(prefs.PreferredTimes, ", ")))
    }
    
    if len(parts) == 0 {
        return ""
    }
    
    return "### 用户偏好\n" + strings.Join(parts, " | ")
}
```

### 3.7 完整构建实现

```go
// plugin/ai/context/builder_impl.go

type contextBuilder struct {
    shortTerm *ShortTermExtractor
    longTerm  *LongTermExtractor
    tokenizer Tokenizer
    cache     CacheService
}

func NewContextBuilder(memSvc MemoryService, cache CacheService) ContextBuilder {
    return &contextBuilder{
        shortTerm: &ShortTermExtractor{memoryService: memSvc},
        longTerm:  &LongTermExtractor{memoryService: memSvc},
        tokenizer: NewSimpleTokenizer(),
        cache:     cache,
    }
}

func (b *contextBuilder) Build(ctx context.Context, req *ContextRequest) (*ContextResult, error) {
    // 缓存检查
    cacheKey := fmt.Sprintf("context:%s:%s", req.SessionID, hashQuery(req.CurrentQuery))
    if cached, ok := b.cache.Get(cacheKey); ok {
        return cached.(*ContextResult), nil
    }
    
    // 1. 分配 Token 预算
    hasRetrieval := len(req.RetrievalResults) > 0
    budget := AllocateBudget(req.MaxTokens, hasRetrieval)
    
    // 2. 提取各层上下文
    shortTermMsgs, _ := b.shortTerm.Extract(ctx, req.SessionID, 10)
    longTermCtx, _ := b.longTerm.Extract(ctx, req.UserID, req.CurrentQuery)
    
    // 3. 构建上下文段
    segments := b.buildSegments(shortTermMsgs, longTermCtx, req.RetrievalResults)
    
    // 4. 优先级排序与截断
    finalSegments := PrioritizeAndTruncate(segments, budget.Total-budget.SystemPrompt)
    
    // 5. 组装最终上下文
    result := b.assembleResult(req.AgentType, finalSegments, budget)
    
    // 缓存结果 (5分钟)
    b.cache.Set(cacheKey, result, 5*time.Minute)
    
    return result, nil
}
```

---

## 4. 实现路径

### Day 1: 核心接口与预算分配

- [ ] 定义 `ContextBuilder` 接口
- [ ] 实现 `TokenBudget` 分配逻辑
- [ ] 实现 `ContextPriority` 排序

### Day 2: 记忆提取器

- [ ] 实现 `ShortTermExtractor`
- [ ] 实现 `LongTermExtractor`
- [ ] 格式化函数（对话、情景、偏好）

### Day 3: 集成与测试

- [ ] 实现完整 `ContextBuilder`
- [ ] 与 Agent 集成
- [ ] 单元测试 + 集成测试

---

## 5. 交付物

### 5.1 代码产出

| 文件 | 说明 |
|:---|:---|
| `plugin/ai/context/builder.go` | 接口定义 |
| `plugin/ai/context/budget.go` | Token 预算分配 |
| `plugin/ai/context/priority.go` | 优先级排序 |
| `plugin/ai/context/short_term.go` | 短期记忆提取 |
| `plugin/ai/context/long_term.go` | 长期记忆提取 |
| `plugin/ai/context/builder_impl.go` | 完整实现 |
| `plugin/ai/context/*_test.go` | 单元测试 |

### 5.2 配置项

```yaml
# configs/ai.yaml
context_builder:
  max_tokens: 4096
  max_turns: 10
  
  budget:
    system_prompt: 500
    user_prefs_ratio: 0.10
    retrieval_ratio: 0.35
    
  cache:
    ttl: 5m
```

---

## 6. 验收标准

### 6.1 功能验收

- [ ] 短期记忆正确提取最近 10 轮
- [ ] 长期记忆正确提取相关情景
- [ ] Token 预算不超限
- [ ] 优先级排序正确

### 6.2 性能验收

- [ ] 构建延迟 < 50ms（不含检索）
- [ ] Token 使用效率提升 30%+
- [ ] 缓存命中率 > 60%

### 6.3 测试用例

```go
func TestTokenBudgetAllocation(t *testing.T) {
    budget := AllocateBudget(4096, true)
    
    total := budget.SystemPrompt + budget.ShortTermMemory + 
             budget.LongTermMemory + budget.Retrieval + budget.UserPrefs
    
    assert.LessOrEqual(t, total, 4096)
    assert.Equal(t, 500, budget.SystemPrompt)
}

func TestPriorityTruncation(t *testing.T) {
    segments := []*ContextSegment{
        {Content: "...", Priority: PrioritySystem, TokenCost: 500},
        {Content: "...", Priority: PriorityRecentTurns, TokenCost: 2000},
        {Content: "...", Priority: PriorityRetrieval, TokenCost: 2000},
    }
    
    result := PrioritizeAndTruncate(segments, 3000)
    
    // 应该保留 System + RecentTurns，截断 Retrieval
    assert.Equal(t, 2, len(result))
}
```

---

## 7. ROI 分析

| 投入 | 产出 |
|:---|:---|
| 开发: 3 人天 | Token 使用效率 +30% |
| 存储: 0 | 对话连贯性显著提升 |
| 维护: 配置化 | 个性化回答能力 |

### 收益计算

- Token 效率提升 30% 意味着相同 Token 预算下包含更多有效信息
- 减少无效上下文导致的误解
- 跨会话记忆减少用户重复说明 50%+

---

## 8. 风险与缓解

| 风险 | 概率 | 影响 | 缓解措施 |
|:---|:---:|:---:|:---|
| Token 计算不准 | 中 | 中 | 使用 tiktoken 精确计算 |
| 记忆服务延迟 | 低 | 中 | 设置超时，降级为仅短期 |
| 上下文过长 | 中 | 低 | 强制截断，保证 Prompt 完整 |

---

## 9. 排期

| 日期 | 任务 | 负责人 |
|:---|:---|:---|
| Sprint 3 Day 1 | 核心接口与预算分配 | TBD |
| Sprint 3 Day 2 | 记忆提取器 | TBD |
| Sprint 3 Day 3 | 集成与测试 | TBD |

---

> **纲领来源**: [00-master-roadmap.md](../../../archived/research_20250202/00-master-roadmap.md)  
> **研究文档**: [assistant-roadmap.md](../../../research/assistant-roadmap.md)  
> **版本**: v1.0  
> **更新时间**: 2026-01-27
