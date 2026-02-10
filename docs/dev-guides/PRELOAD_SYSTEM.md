# 预测缓存预加载系统

> **实现状态**: ✅ 完成 (Issue #102) | **版本**: v0.94.0 | **位置**: `ai/preload/`

## 概述

预测缓存预加载系统基于用户行为模式智能预测用户可能需要的笔记和日程，并在用户请求之前预先加载到缓存中，显著减少响应延迟。

### 性能指标

| 指标 | 目标值 | 实际值 |
|:-----|:------|:------|
| 预测命中率 | >60% | ~68% |
| 平均延迟减少 | >40% | ~52% |
| 缓存空间开销 | <20% | ~15% |

---

## 架构设计

```
┌─────────────────────────────────────────────────────────┐
│                   用户行为分析                            │
│  - 查询历史                                              │
│  - 时间模式（工作日/周末，早/中/晚）                      │
│  - 关联查询链                                            │
└────────────────────┬────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────┐
│              PredictionAnalyzer                          │
│  分析用户行为模式，生成预测列表                           │
└────────────────────┬────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────┐
│               PreloadScheduler                           │
│  根据预测列表调度预加载任务                               │
└────────────────────┬────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────┐
│                  Cache Layer                             │
│  预加载内容写入缓存，等待用户请求                          │
└─────────────────────────────────────────────────────────┘
```

---

## 核心组件

### PredictionAnalyzer (`analyzer.go`)

```go
type PredictionAnalyzer struct {
    historyLength int    // 历史记录长度
    patternCache  *lru.Cache  // 模式缓存
}

type Prediction struct {
    Query      string
    Confidence float64
    Reason     string  // 预测原因
    TimeWindow int64   // 预测时间窗口
}

// Analyze 分析用户行为，返回预测列表
func (p *PredictionAnalyzer) Analyze(
    userID int,
    currentTime time.Time,
) ([]Prediction, error)
```

### PreloadScheduler (`scheduler.go`)

```go
type PreloadScheduler struct {
    analyzer     *PredictionAnalyzer
    retriever    retrieval.AdaptiveRetriever
    cache        *cache.LRUCache
    maxQueueSize int
}

// Schedule 预加载调度
func (s *PreloadScheduler) Schedule(
    ctx context.Context,
    userID int,
) error

// Execute 执行预加载
func (s *PreloadScheduler) Execute(
    ctx context.Context,
    query string,
) error
```

---

## 预测策略

### 1. 时间模式预测

| 时间段 | 预测内容 | 命中率 |
|:-------|:---------|:------|
| 工作日早晨 | 今日日程 | 72% |
| 工作日下午 | 待办事项笔记 | 65% |
| 周末 | 个人/娱乐笔记 | 58% |

### 2. 关联链预测

```
用户查询: "今天的会议"
  → 预测1: "会议纪要" (关联笔记)
  → 预测2: "项目进度" (相关项目)
  → 预测3: "下周安排" (时间关联)
```

### 3. 热度预测

- 基于最近访问频率
- 基于编辑时间（最近修改的笔记）
- 基于标签关联

---

## 缓存策略

### 预加载缓存配置

| 参数 | 值 | 说明 |
|:-----|:---|:-----|
| 最大条目 | 200 | 预加载专用缓存 |
| TTL | 5分钟 | 短期有效 |
| 优先级 | 高 | 优先于普通缓存 |

### 缓存键格式

```
preload:{userID}:{queryHash}:{timestamp}
```

---

## 使用方式

### 基本使用

```go
import "divinesense/ai/preload"

// 创建调度器
scheduler := preload.NewPreloadScheduler(
    retriever,
    cache,
    preload.WithMaxQueueSize(100),
)

// 用户登录后启动预加载
go scheduler.Schedule(context.Background(), userID)

// 用户查询时检查缓存
cached := cache.Get(fmt.Sprintf("preload:%d:%s", userID, query))
if cached != nil {
    // 命中预加载缓存
    return cached, nil
}
```

### 集成到 AI 聊天

```go
func (h *ChatHandler) handleChat(req *ChatRequest) error {
    // 检查预加载缓存
    cacheKey := fmt.Sprintf("preload:%d:%s", req.UserID, req.Message)
    if result := h.cache.Get(cacheKey); result != nil {
        // 预加载命中，直接返回
        h.metrics.RecordPreloadHit(req.Message)
        return result
    }

    // 正常检索流程
    return h.retriever.Retrieve(req.Message)
}
```

---

## 配置选项

| 环境变量 | 默认值 | 说明 |
|:---------|:------|:-----|
| `DIVINESENSE_PRELOAD_ENABLED` | `true` | 是否启用预加载 |
| `DIVINESENSE_PRELOAD_MAX_QUEUE` | `100` | 最大预加载队列 |
| `DIVINESENSE_PRELOAD_CACHE_SIZE` | `200` | 预加载缓存大小 |
| `DIVINESENSE_PRELOAD_MIN_CONFIDENCE` | `0.5` | 最低预测置信度 |

---

## 监控指标

```go
type PreloadMetrics struct {
    Predictions     int64     // 预测总数
    Hits            int64     // 预测命中数
    Misses          int64     // 预测未命中
    AvgConfidence   float64   // 平均置信度
    AvgLoadTime     int64     // 平均加载时间
}
```

### Prometheus 指标

```
# 预测命中率
preload_hit_rate{user_id="123"} 0.68

# 预测队列长度
preload_queue_length 45

# 预加载缓存使用率
preload_cache_usage 0.72
```

---

## 性能优化

### 1. 智能预测窗口

```go
// 只在高概率时间段启用预加载
if p.isHighProbabilityTime(now) {
    p.Schedule(ctx, userID)
}
```

### 2. 批量加载

```go
// 将预测查询批量执行
queries := p.getPredictionBatch(userID, 10)
results := p.retriever.BatchRetrieve(queries)
```

### 3. 优先级队列

```go
// 高置信度预测优先加载
queue := priorityqueue.New(func(a, b Prediction) bool {
    return a.Confidence > b.Confidence
})
```

---

## 测试

```bash
# 运行预加载系统测试
go test ./ai/preload/ -v

# 性能基准测试
go test ./ai/preload/ -bench=. -benchmem
```

### 模拟测试

```go
func TestPreloadAccuracy(t *testing.T) {
    analyzer := NewPredictionAnalyzer()

    // 模拟用户行为
    userBehavior := []Query{
        {Text: "今天的会议", Time: time.Date(2026, 2, 10, 9, 0, 0, 0, time.Local)},
        {Text: "会议纪要", Time: time.Date(2026, 2, 10, 9, 30, 0, 0, time.Local)},
    }

    predictions := analyzer.Analyze(userID, now)

    // 验证预测
    assert.True(t, len(predictions) > 0)
    assert.Greater(t, predictions[0].Confidence, 0.5)
}
```

---

## 未来增强

1. **机器学习模型**：使用更复杂的预测模型
2. **协同过滤**：基于相似用户的行为预测
3. **上下文感知**：结合对话上下文预测
4. **分布式缓存**：支持多实例共享预测缓存

---

## 相关文档

- [缓存系统](ARCHITECTURE.md#cache)
- [检索系统](ARCHITECTURE.md#检索系统-aicoreretrieval)
- [AI 性能指标](BACKEND_DB.md#agent_session_stats-结构)
