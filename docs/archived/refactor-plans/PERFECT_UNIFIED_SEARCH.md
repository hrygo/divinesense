# 笔记 + 日程完美联合检索方案

## 🎯 设计目标

打造一个**智能、高效、用户友好**的联合检索系统，实现：

1. ✅ **统一检索体验**：一次查询，同时检索笔记和日程
2. ✅ **智能意图识别**：自动判断用户需求
3. ✅ **最优检索策略**：根据数据特点选择最佳算法
4. ✅ **完美融合排序**：RRF + 业务规则混合排序
5. ✅ **流畅的用户体验**：清晰的结果展示和交互

---

## 📊 核心设计理念

### 数据差异分析

| 维度 | 笔记 (Memo) | 日程 (Schedule) |
|------|-------------|-----------------|
| **内容特征** | 长文本（100-2000字） | 短文本（10-100字） |
| **时间敏感度** | 低（创建时间） | 高（执行时间） |
| **检索重点** | 内容语义 | 时间 + 内容 |
| **排序依据** | 相关度 | 时间 + 相关度 |
| **用户期望** | 找到相关信息 | 按时间顺序列出 |

### 检索策略对比

```
笔记检索：
  ├─ BM25: 精确关键词匹配
  ├─ Semantic: 语义理解
  └─ 融合: RRF
  └─ 排序: 相关度优先

日程检索：
  ├─ Time Filter: 时间范围（必需）
  ├─ BM25: 标题/地点匹配
  ├─ Semantic: 描述语义
  └─ 融合: RRF + 时间权重
  └─ 排序: 时间优先，相关度辅助
```

---

## 🏗️ 完美架构设计

### 总体架构图

```
┌─────────────────────────────────────────────────────────┐
│                    用户查询输入                          │
│            "今天下午关于AI项目的会议"                     │
└────────────────────┬────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────┐
│              Phase 1: 智能意图分析引擎                   │
├─────────────────────────────────────────────────────────┤
│                                                          │
│  1.1 时间维度分析                                        │
│      ├─ 检测："今天"、"下午" → 时间范围                │
│      ├─ 计算：14:00 - 18:00                            │
│      └─ 输出：time_range = {start, end}                │
│                                                          │
│  1.2 内容维度分析                                        │
│      ├─ 检测："AI项目"、"会议" → 关键词               │
│      ├─ 提取：实体识别（项目名、人名）                 │
│      └─ 输出：semantic_query = "AI项目会议"            │
│                                                          │
│  1.3 数据源分析                                          │
│      ├─ 笔记关键词：备忘、记录、搜索、笔记             │
│      ├─ 日程关键词：会议、安排、日程、今天              │
│      └─ 输出：target_sources = ["memo", "schedule"]    │
│                                                          │
│  1.4 查询类型分类                                        │
│      ├─ 纯笔记：memo_only (40%)                         │
│      ├─ 纯日程：schedule_only (30%)                     │
│      ├─ 混合查询：mixed (20%)                           │
│      └─ 通用问答：general (10%)                         │
│                                                          │
└────────────────────┬────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────┐
│         Phase 2: 并行混合检索（2路并行）                 │
├─────────────────────────────────────────────────────────┤
│                                                          │
│  【A路】笔记检索通道                                     │
│  ┌────────────────────────────────────┐                │
│  │ 2.1 BM25 关键词检索                │                │
│  │     └─ Top 20, threshold ≥ 0.3    │                │
│  │                                    │                │
│  │ 2.2 语义向量检索                   │                │
│  │     └─ Top 20, threshold ≥ 0.5    │                │
│  │                                    │                │
│  │ 2.3 RRF 融合                       │                │
│  │     └─ Top 20 笔记                 │                │
│  └────────────────────────────────────┘                │
│                          ↓                             │
│  【B路】日程检索通道                                     │
│  ┌────────────────────────────────────┐                │
│  │ 2.1 时间过滤（SQL）                │                │
│  │     └─ 今天 14:00 - 18:00          │                │
│  │                                    │                │
│  │ 2.2 BM25 关键词检索                │                │
│  │     └─ 标题/地点匹配, Top 20       │                │
│  │                                    │                │
│  │ 2.3 语义向量检索                   │                │
│  │     └─ 描述相似度, Top 20          │                │
│  │                                    │                │
│  │ 2.4 混合融合（时间加权RRF）         │                │
│  │     └─ score = rrf + time_weight   │                │
│  │     └─ Top 20 日程                 │                │
│  └────────────────────────────────────┘                │
│                                                          │
└────────────────────┬────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────┐
│          Phase 3: 智能融合与重排序                       │
├─────────────────────────────────────────────────────────┤
│                                                          │
│  3.1 笔记和日程独立排序                                  │
│      ├─ 笔记：按相关度降序                              │
│      └─ 日程：按时间升序 + 相关度降序                   │
│                                                          │
│  3.2 业务规则应用                                        │
│      ├─ 今日日程：提升权重 × 1.5                        │
│      ├─ 紧急日程：提升权重 × 1.3                        │
│      ├─ 最近笔记：提升权重 × 1.2                        │
│      └─ 重要标签：提升权重 × 1.1                        │
│                                                          │
│  3.3 Reranker 重排序（可选）                            │
│      └─ 对 Top 10 使用 Reranker                        │
│      └─ 提升语义相关性                                 │
│                                                          │
└────────────────────┬────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────┐
│              Phase 4: 结果分组与格式化                   │
├─────────────────────────────────────────────────────────┤
│                                                          │
│  结果结构：                                              │
│  {                                                       │
│    "query_type": "mixed",                              │
│    "total_results": 15,                                │
│    "memos": {                                          │
│      "count": 8,                                       │
│      "items": [...]                                   │
│    },                                                   │
│    "schedules": {                                      │
│      "count": 7,                                       │
│      "items": [...],                                  │
│      "grouped": {                                      │
│        "today": [...],                                 │
│        "tomorrow": [...],                              │
│        "upcoming": [...]                               │
│      }                                                  │
│    },                                                   │
│    "metadata": {                                       │
│      "time_range_detected": true,                      │
│      "semantic_query": "AI项目会议",                   │
│      "confidence": 0.92                                │
│    }                                                    │
│  }                                                       │
└────────────────────┬────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────┐
│           Phase 5: LLM 智能回复生成                      │
├─────────────────────────────────────────────────────────┤
│                                                          │
│  5.1 构建上下文                                          │
│      ├─ 添加笔记内容（最多5条，3000字符）               │
│      ├─ 添加日程信息（结构化）                          │
│      └─ 添加用户查询                                    │
│                                                          │
│  5.2 选择回复策略                                        │
│      ├─ schedule_only: 简短总结 + 结构化数据            │
│      ├─ memo_only: 详细说明 + 引用笔记                  │
│      └─ mixed: 分段回复 + 结构化数据                   │
│                                                          │
│  5.3 生成回复                                            │
│      └─ 流式输出                                        │
│                                                          │
└────────────────────┬────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────┐
│            Phase 6: 前端智能渲染                         │
├─────────────────────────────────────────────────────────┤
│                                                          │
│  6.1 根据 query_type 渲染                                │
│      ├─ schedule_only: 日程卡片（时间线）               │
│      ├─ memo_only: AI 回复 + 笔记列表                   │
│      └─ mixed: AI 回复 + 日程卡片                       │
│                                                          │
│  6.2 日程分组展示                                        │
│      ├─ 今日日程（红色标记）                            │
│      ├─ 明日日程（蓝色标记）                            │
│      └─ 即将到来（灰色标记）                            │
│                                                          │
│  6.3 交互功能                                            │
│      ├─ 点击笔记 → 跳转详情                             │
│      ├─ 点击日程 → 打开编辑                             │
│      └─ 快速操作（新建、删除、移动）                    │
│                                                          │
└─────────────────────────────────────────────────────────┘
```

---

## 🔬 核心算法详解

### 1. 智能意图分析引擎

```go
// IntentEngine 意图分析引擎
type IntentEngine struct {
    // 时间关键词库
    timeKeywords map[string]TimeRange
    // 内容关键词库
    contentKeywords map[string]string
    // LLM 客户端（用于复杂意图判断）
    llm LLMService
}

type QueryIntent struct {
    // 基础信息
    OriginalQuery   string

    // 时间维度
    HasTimeKeyword  bool
    TimeRange       *TimeRange
    TimeExpressions []string  // ["今天", "下午"]

    // 内容维度
    SemanticQuery   string
    Keywords        []string
    Entities        []Entity   // 人名、项目名、地点

    // 数据源
    TargetSources   []string  // ["memo", "schedule"]

    // 查询类型
    QueryType       string    // "memo_only", "schedule_only", "mixed", "general"

    // 置信度
    Confidence      float32

    // 业务规则
    Priority        []string  // ["today", "urgent"]
}

// Analyze 分析查询意图（多阶段）
func (e *IntentEngine) Analyze(query string) *QueryIntent {
    intent := &QueryIntent{
        OriginalQuery: query,
    }

    // 阶段1: 快速规则匹配（95%场景）
    if intent := e.quickMatch(query); intent != nil {
        return intent
    }

    // 阶段2: 复杂意图识别（5%场景）
    return e.deepAnalysis(query)
}

// quickMatch 快速匹配（基于规则）
func (e *IntentEngine) quickMatch(query string) *QueryIntent {
    intent := &QueryIntent{}

    // 1. 时间关键词检测
    timeKeywords := e.extractTimeKeywords(query)
    if len(timeKeywords) > 0 {
        intent.HasTimeKeyword = true
        intent.TimeRange = e.calculateTimeRange(timeKeywords)
        intent.TimeExpressions = timeKeywords
    }

    // 2. 数据源检测
    hasMemoKeyword := containsAny(query, []string{"笔记", "备忘", "记录", "搜索", "查找"})
    hasScheduleKeyword := containsAny(query, []string{"日程", "会议", "安排", "计划", "今天", "明天"})

    // 3. 查询类型判断
    if intent.HasTimeKeyword && hasScheduleKeyword {
        intent.QueryType = "schedule_only"
        intent.TargetSources = []string{"schedule"}
        intent.Confidence = 0.95
    } else if hasMemoKeyword && !intent.HasTimeKeyword {
        intent.QueryType = "memo_only"
        intent.TargetSources = []string{"memo"}
        intent.Confidence = 0.90
    } else if intent.HasTimeKeyword || hasScheduleKeyword {
        intent.QueryType = "mixed"
        intent.TargetSources = []string{"memo", "schedule"}
        intent.Confidence = 0.85
    } else {
        intent.QueryType = "general"
        intent.TargetSources = []string{"memo", "schedule"}
        intent.Confidence = 0.70
    }

    // 4. 提取语义查询
    intent.SemanticQuery = e.extractSemanticQuery(query, intent.TimeExpressions)

    return intent
}

// extractSemanticQuery 提取语义查询（去除时间词）
func (e *IntentEngine) extractSemanticQuery(query string, timeExpressions []string) string {
    semanticQuery := query

    // 移除时间表达式
    for _, timeExpr := range timeExpressions {
        semanticQuery = strings.ReplaceAll(semanticQuery, timeExpr, "")
    }

    // 移除停用词
    stopWords := []string{"的", "有什么", "查询", "搜索", "查找"}
    for _, stopWord := range stopWords {
        semanticQuery = strings.ReplaceAll(semanticQuery, stopWord, "")
    }

    return strings.TrimSpace(semanticQuery)
}

// TimeRange 时间范围
type TimeRange struct {
    Start    time.Time
    End      time.Time
    Label    string  // "今天", "本周", 等
}

// calculateTimeRange 计算时间范围
func (e *IntentEngine) calculateTimeRange(expressions []string) *TimeRange {
    now := time.Now()

    // 单个时间词
    for _, expr := range expressions {
        switch expr {
        case "今天":
            return &TimeRange{
                Start: time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()),
                End:   time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 0, now.Location()),
                Label: "今天",
            }
        case "明天":
            tomorrow := now.AddDate(0, 0, 1)
            return &TimeRange{
                Start: time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), 0, 0, 0, 0, now.Location()),
                End:   time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), 23, 59, 59, 0, now.Location()),
                Label: "明天",
            }
        case "后天":
            dayAfter := now.AddDate(0, 0, 2)
            return &TimeRange{
                Start: time.Date(dayAfter.Year(), dayAfter.Month(), dayAfter.Day(), 0, 0, 0, 0, now.Location()),
                End:   time.Date(dayAfter.Year(), dayAfter.Month(), dayAfter.Day(), 23, 59, 59, 0, dayAfter.Location()),
                Label: "后天",
            }
        case "本周":
            weekday := now.Weekday()
            if weekday == time.Sunday {
                weekday = 7
            }
            startOfWeek := time.Date(now.Year(), now.Month(), now.Day()-int(weekday)+1, 0, 0, 0, 0, now.Location())
            endOfWeek := startOfWeek.AddDate(0, 0, 7)
            return &TimeRange{
                Start: startOfWeek,
                End:   endOfWeek,
                Label: "本周",
            }
        case "下周":
            weekday := now.Weekday()
            if weekday == time.Sunday {
                weekday = 7
            }
            startOfNextWeek := time.Date(now.Year(), now.Month(), now.Day()-int(weekday)+1+7, 0, 0, 0, 0, now.Location())
            endOfNextWeek := startOfNextWeek.AddDate(0, 0, 7)
            return &TimeRange{
                Start: startOfNextWeek,
                End:   endOfNextWeek,
                Label: "下周",
            }
        }
    }

    // 组合时间词（如"今天下午"）
    if contains(expressions, "今天") && contains(expressions, "下午") {
        start := time.Date(now.Year(), now.Month(), now.Day(), 13, 0, 0, 0, now.Location())
        end := time.Date(now.Year(), now.Month(), now.Day(), 18, 0, 0, 0, now.Location())
        return &TimeRange{
            Start: start,
            End:   end,
            Label: "今天下午",
        }
    }

    // 默认：未来7天
    start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
    end := start.AddDate(0, 0, 7)
    return &TimeRange{
        Start: start,
        End:   end,
        Label: "近期",
    }
}
```

### 2. 并行混合检索引擎

```go
// UnifiedSearchEngine 统一检索引擎
type UnifiedSearchEngine struct {
    store  *store.Store
    embedding ai.EmbeddingService
    reranker  ai.RerankerService
}

// Search 统一检索（笔记 + 日程）
func (e *UnifiedSearchEngine) Search(ctx context.Context, intent *QueryIntent) (*UnifiedSearchResult, error) {
    result := &UnifiedSearchResult{
        QueryIntent: intent,
        Memos:       make([]*MemoWithScore, 0),
        Schedules:   make([]*ScheduleWithScore, 0),
    }

    // 并行检索笔记和日程
    var (
        memoResults     []*store.SearchResult
        scheduleResults []*store.SearchResult
        memoErr         error
        scheduleErr     error
        wg              sync.WaitGroup
    )

    wg.Add(2)

    // 路径 A: 笔记检索
    go func() {
        defer wg.Done()
        if contains(intent.TargetSources, "memo") {
            memoResults, memoErr = e.searchMemos(ctx, intent)
        }
    }()

    // 路径 B: 日程检索
    go func() {
        defer wg.Done()
        if contains(intent.TargetSources, "schedule") {
            scheduleResults, scheduleErr = e.searchSchedules(ctx, intent)
        }
    }()

    wg.Wait()

    // 处理错误
    if memoErr != nil {
        return nil, fmt.Errorf("memo search failed: %w", memoErr)
    }
    if scheduleErr != nil {
        return nil, fmt.Errorf("schedule search failed: %w", scheduleErr)
    }

    // 转换结果
    result.Memos = convertMemoResults(memoResults)
    result.Schedules = convertScheduleResults(scheduleResults)

    // 应用业务规则
    e.applyBusinessRules(ctx, result)

    return result, nil
}

// searchMemos 检索笔记（BM25 + Semantic + RRF）
func (e *UnifiedSearchEngine) searchMemos(ctx context.Context, intent *QueryIntent) ([]*store.SearchResult, error) {
    opts := &store.HybridSearchOptions{
        UserID:       intent.UserID,
        Query:        intent.SemanticQuery,
        SearchTypes:  []string{"memo"},
        Limit:        20,
        RRFK:         60,
    }

    return e.store.HybridSearch(ctx, opts)
}

// searchSchedules 检索日程（时间过滤 + BM25 + Semantic + 时间加权RRF）
func (e *UnifiedSearchEngine) searchSchedules(ctx context.Context, intent *QueryIntent) ([]*store.SearchResult, error) {
    // 1. 构建时间过滤条件
    var startTime, endTime *int64
    if intent.HasTimeKeyword && intent.TimeRange != nil {
        start := intent.TimeRange.Start.Unix()
        end := intent.TimeRange.End.Unix()
        startTime = &start
        endTime = &end
    } else {
        // 默认时间范围：未来7天
        now := time.Now()
        start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).Unix()
        end := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).AddDate(0, 0, 7).Unix()
        startTime = &start
        endTime = &end
    }

    opts := &store.HybridSearchOptions{
        UserID:       intent.UserID,
        Query:        intent.SemanticQuery,
        SearchTypes:  []string{"schedule"},
        StartTime:    startTime,
        EndTime:      endTime,
        Limit:        20,
        RRFK:         60,
    }

    results, err := e.store.HybridSearch(ctx, opts)
    if err != nil {
        return nil, err
    }

    // 2. 应用时间权重
    e.applyTimeWeight(results, intent.TimeRange)

    return results, nil
}

// applyTimeWeight 应用时间权重
func (e *UnifiedSearchEngine) applyTimeWeight(results []*store.SearchResult, timeRange *TimeRange) {
    now := time.Now()

    for _, result := range results {
        if result.Type != "schedule" || result.Schedule == nil {
            continue
        }

        schedule := result.Schedule
        scheduleTime := time.Unix(schedule.StartTs, 0)

        // 时间权重计算
        var timeWeight float32 = 1.0

        // 今日日程：权重 × 1.5
        if isSameDay(scheduleTime, now) {
            timeWeight = 1.5
        }
        // 明日日程：权重 × 1.2
        else if isSameDay(scheduleTime, now.AddDate(0, 0, 1)) {
            timeWeight = 1.2
        }
        // 本周日程：权重 × 1.1
        else if isSameWeek(scheduleTime, now) {
            timeWeight = 1.1
        }

        // 更新分数
        result.Score = result.Score * timeWeight
    }
}

// applyBusinessRules 应用业务规则
func (e *UnifiedSearchEngine) applyBusinessRules(ctx context.Context, result *UnifiedSearchResult) {
    // 规则1: 今日日程优先
    now := time.Now()
    for _, sched := range result.Schedules {
        scheduleTime := time.Unix(sched.StartTs, 0)
        if isSameDay(scheduleTime, now) {
            sched.Score = sched.Score * 1.3
        }
    }

    // 规则2: 重要标签提升
    for _, memo := range result.Memos {
        if memo.HasTag("important") || memo.HasTag("紧急") {
            memo.Score = memo.Score * 1.2
        }
    }

    // 规则3: 最近笔记提升（7天内）
    weekAgo := now.AddDate(0, 0, -7)
    for _, memo := range result.Memos {
        memoTime := time.Unix(memo.CreatedTs, 0)
        if memoTime.After(weekAgo) {
            memo.Score = memo.Score * 1.1
        }
    }
}
```

### 3. 智能结果分组

```go
// ResultGrouper 结果分组器
type ResultGrouper struct{}

// Group 分组结果
func (g *ResultGrouper) Group(result *UnifiedSearchResult) *GroupedResult {
    grouped := &GroupedResult{
        Memos:     result.Memos,
        Schedules: make(map[string][]*ScheduleWithScore),
    }

    // 按时间分组日程
    now := time.Now()
    today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
    tomorrow := today.AddDate(0, 0, 1)
    thisWeek := today.AddDate(0, 0, 7)

    for _, sched := range result.Schedules {
        scheduleTime := time.Unix(sched.StartTs, 0)

        if scheduleTime.Before(today.AddDate(0, 0, 1)) {
            // 今日日程
            grouped.Schedules["today"] = append(grouped.Schedules["today"], sched)
        } else if scheduleTime.Before(tomorrow.AddDate(0, 0, 1)) {
            // 明日日程
            grouped.Schedules["tomorrow"] = append(grouped.Schedules["tomorrow"], sched)
        } else if scheduleTime.Before(thisWeek) {
            // 本周日程
            grouped.Schedules["this_week"] = append(grouped.Schedules["this_week"], sched)
        } else {
            // 未来日程
            grouped.Schedules["upcoming"] = append(grouped.Schedules["upcoming"], sched)
        }
    }

    return grouped
}
```

---

## 📡 API 设计

### Protocol Buffers

```protobuf
syntax = "proto3";

package api.v1;

service AIService {
  rpc UnifiedChat(UnifiedChatRequest) returns (stream UnifiedChatResponse);
}

message UnifiedChatRequest {
  string message = 1;
  repeated string history = 2;
  // 可选：强制指定数据源
  repeated string force_sources = 3;  // ["memo"], ["schedule"], or ["memo", "schedule"]
}

message UnifiedChatResponse {
  // 流式内容
  string content = 1;

  // 查询元数据
  QueryMetadata query_metadata = 2;

  // 笔记结果
  repeated MemoResult memos = 3;

  // 日程结果（分组）
  ScheduleResults schedules = 4;

  // 完成标记
  bool done = 5;
}

message QueryMetadata {
  string query_type = 1;  // "memo_only", "schedule_only", "mixed", "general"
  float confidence = 2;

  // 时间信息
  bool has_time_keyword = 3;
  string time_range_label = 4;  // "今天", "本周", etc.

  // 语义信息
  string semantic_query = 5;

  // 结果统计
  int32 total_memos = 6;
  int32 total_schedules = 7;
}

message MemoResult {
  string uid = 1;
  string content = 2;
  string snippet = 3;
  float score = 4;
  repeated string tags = 5;
  int64 created_ts = 6;
}

message ScheduleResults {
  int32 total = 1;
  map<string, ScheduleGroup> groups = 2;  // "today", "tomorrow", "this_week", "upcoming"
}

message ScheduleGroup {
  string label = 1;  // "今日日程", "明日日程"
  int32 count = 2;
  repeated ScheduleItem items = 3;
}

message ScheduleItem {
  string uid = 1;
  string title = 2;
  int64 start_ts = 3;
  int64 end_ts = 4;
  string location = 5;
  float score = 6;
}
```

---

## 🎨 前端渲染示例

```tsx
// 组件：UnifiedSearchResult.tsx

interface UnifiedSearchResultProps {
  queryMetadata: QueryMetadata;
  memos: MemoResult[];
  schedules: ScheduleResults;
  aiContent: string;
}

export function UnifiedSearchResult({
  queryMetadata,
  memos,
  schedules,
  aiContent
}: UnifiedSearchResultProps) {
  return (
    <div className="unified-search-result">
      {/* AI 回复 */}
      {aiContent && (
        <AIMessage content={aiContent} />
      )}

      {/* 日程结果（按时间分组） */}
      {schedules.total > 0 && (
        <div className="schedule-section">
          {Object.entries(schedules.groups).map(([key, group]) => (
            <ScheduleGroup key={key} group={group} />
          ))}
        </div>
      )}

      {/* 笔记结果 */}
      {memos.length > 0 && (
        <div className="memo-section">
          <h3>相关笔记 ({memos.length})</h3>
          {memos.map(memo => (
            <MemoCard key={memo.uid} memo={memo} />
          ))}
        </div>
      )}
    }
  </div>
  );
}

// 日程分组组件
function ScheduleGroup({ group }: { group: ScheduleGroup }) {
  return (
    <div className="schedule-group">
      <h3 className="group-title">
        {group.label} ({group.count})
      </h3>
      <div className="schedule-list">
        {group.items.map(schedule => (
          <ScheduleCard key={schedule.uid} schedule={schedule} />
        ))}
      </div>
    </div>
  );
}
```

---

## ⚡ 性能优化策略

### 1. 三级缓存

```go
type CacheStrategy struct {
    L1Cache *sync.Map  // 内存缓存（热点查询）
    L2Cache *redis.Cache  // Redis 缓存
    L3Cache *store.Store  // 数据库
}

func (c *CacheStrategy) Get(ctx context.Context, key string) (interface{}, error) {
    // L1: 内存缓存（10ms）
    if val, ok := c.L1Cache.Load(key); ok {
        return val, nil
    }

    // L2: Redis 缓存（50ms）
    val, err := c.L2Cache.Get(ctx, key)
    if err == nil {
        c.L1Cache.Store(key, val)
        return val, nil
    }

    // L3: 数据库查询（200ms）
    val, err = c.L3Cache.Query(ctx, key)
    if err == nil {
        c.L2Cache.Set(ctx, key, val, 30*time.Second)
        c.L1Cache.Store(key, val)
    }

    return val, err
}
```

### 2. 并行查询优化

```go
// 使用 goroutine 并行执行
func (e *UnifiedSearchEngine) SearchParallel(ctx context.Context, intent *QueryIntent) (*UnifiedSearchResult, error) {
    ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
    defer cancel()

    type resultPair struct {
        results []*store.SearchResult
        err     error
    }

    memoCh := make(chan resultPair, 1)
    scheduleCh := make(chan resultPair, 1)

    // 并行检索
    go func() {
        results, err := e.searchMemos(ctx, intent)
        memoCh <- resultPair{results, err}
    }()

    go func() {
        results, err := e.searchSchedules(ctx, intent)
        scheduleCh <- resultPair{results, err}
    }()

    // 等待结果
    memoResults := <-memoCh
    scheduleResults := <-scheduleCh

    // 处理结果
    // ...
}
```

### 3. 索引优化

```sql
-- 笔记：复合索引
CREATE INDEX idx_memo_user_tsv_created
  ON memo (creator_id, row_status) INCLUDE (content_tsv, created_ts);

-- 日程：复合索引
CREATE INDEX idx_schedule_user_time_search
  ON schedule (creator_id, start_ts, end_ts, row_status) INCLUDE (search_text);

-- 向量索引：IVFFlat
CREATE INDEX idx_memo_embedding_ivfflat
  ON memo_embedding USING ivfflat (embedding vector_cosine_ops)
  WITH (lists = 100);

CREATE INDEX idx_schedule_embedding_ivfflat
  ON schedule_embedding USING ivfflat (embedding vector_cosine_ops)
  WITH (lists = 100);
```

---

## 📊 效果评估

### 检索质量指标

| 指标 | 目标 | 验证方法 |
|------|------|---------|
| **意图识别准确率** | >95% | 人工标注测试集 |
| **笔记检索 NDCG@10** | >0.85 | 离线评估 |
| **日程检索准确率** | >90% | 时间范围匹配 |
| **混合排序满意度** | >4.0/5.0 | 用户反馈 |
| **端到端响应时间** | <500ms | 性能监控 |

### A/B 测试方案

```
对照组：当前实现（纯语义检索）
实验组：混合检索（BM25 + Semantic + RRF）

评估维度：
1. 检索准确率（离线指标）
2. 用户满意度（在线反馈）
3. 响应时间（性能监控）
4. 转化率（点击率、使用率）
```

---

## 🚀 实施路线图

### Phase 1: 基础设施（Week 1-2）
- [ ] 添加 BM25 索引（memo + schedule）
- [ ] 实现意图分析引擎
- [ ] 实现混合检索 Store 层

### Phase 2: 核心功能（Week 3-4）
- [ ] 统一检索引擎
- [ ] 业务规则引擎
- [ ] 结果分组器

### Phase 3: AI 服务集成（Week 5）
- [ ] 改造 ChatWithMemos
- [ ] 更新 Protocol Buffers
- [ ] 流式响应优化

### Phase 4: 前端适配（Week 6）
- [ ] 统一结果组件
- [ ] 日程分组展示
- [ ] 交互功能优化

### Phase 5: 测试与优化（Week 7-8）
- [ ] 单元测试
- [ ] 集成测试
- [ ] 性能优化
- [ ] A/B 测试

**总计：8 周**

---

## ✅ 总结

### 完美方案的三大支柱

1. **智能意图识别**
   - 时间维度分析
   - 内容语义提取
   - 查询类型分类

2. **混合检索策略**
   - 笔记：BM25 + 语义 + RRF
   - 日程：时间过滤 + BM25 + 语义 + 时间加权RRF
   - 并行执行，性能优化

3. **业务规则增强**
   - 今日日程优先
   - 重要标签提升
   - 最近笔记加权

### 核心优势

- ✅ **准确率高**：结合多种检索算法
- ✅ **智能排序**：业务规则 + 相关度
- ✅ **用户体验**：分组展示，清晰直观
- ✅ **性能优越**：并行检索，三级缓存

准备好了吗？让我们一起打造完美的联合检索系统！🚀
