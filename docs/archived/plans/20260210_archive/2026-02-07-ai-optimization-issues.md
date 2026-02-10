# AI 智能助理优化建议 - GitHub Issues 规划

> **创建日期**: 2026-02-07  
> **来源**: NORMAL_MODE_ASSISTANT_ANALYSIS.md 深度分析报告  
> **状态**: ✅ 已创建所有 Issues

---

## ✅ 已创建的 GitHub Issues

| Issue #                                                 | 标题                 | 优先级 | 状态       |
| ------------------------------------------------------- | -------------------- | ------ | ---------- |
| [#91](https://github.com/hrygo/divinesense/issues/91)   | 语义缓存层实现       | P1     | 🔴 Open     |
| [#92](https://github.com/hrygo/divinesense/issues/92)   | 工具级检索结果缓存   | P1     | 🔴 Open     |
| [#93](https://github.com/hrygo/divinesense/issues/93)   | 动态 Token 预算分配  | P1     | 🔴 Open     |
| [#94](https://github.com/hrygo/divinesense/issues/94)   | 增量上下文更新       | P2     | 🔴 Open     |
| [#95](https://github.com/hrygo/divinesense/issues/95)   | 规则路由权重动态调整 | P2     | 🔴 Open     |
| [#96](https://github.com/hrygo/divinesense/issues/96)   | 轻量 ML 意图分类器   | P2     | 🔴 Open     |
| [#97](https://github.hrygo/divinesense/issues/97)       | 渐进式进度反馈       | P2     | 🔴 Open     |
| [#98](https://github.com/hrygo/divinesense/issues/98)   | 智能快捷回复生成     | P2     | 🔴 Open     |
| [#99](https://github.com/hrygo/divinesense/issues/99)   | 端到端追踪系统       | P2     | 🔴 Open     |
| [#100](https://github.com/hrygo/divinesense/issues/100) | Prometheus 指标增强  | P3     | 🔴 Open     |
| [#101](https://github.com/hrygo/divinesense/issues/101) | 敏感信息过滤器       | P3     | 🔴 Open     |
| [#102](https://github.com/hrygo/divinesense/issues/102) | 预测性预加载         | P3     | 🔴 Open     |
| [#103](https://github.com/hrygo/divinesense/issues/103) | **优化计划总览**     | -      | 📋 Tracking |

---

## Issue 分组策略

将优化建议拆分为 **12 个原子化 Issues**，按优先级和复杂度排序：

| ID  | Issue 标题           | 优先级 | 复杂度 | 标签                   |
| --- | -------------------- | ------ | ------ | ---------------------- |
| 1   | 语义缓存层实现       | P1     | 高     | ai, 性能优化, backend  |
| 2   | 检索结果缓存         | P1     | 中     | ai, 性能优化, backend  |
| 3   | 动态 Token 预算分配  | P1     | 中     | ai, 性能优化, backend  |
| 4   | 增量上下文更新       | P2     | 高     | ai, 性能优化, backend  |
| 5   | 规则路由权重动态调整 | P2     | 中     | ai, backend            |
| 6   | 轻量 ML 意图分类器   | P2     | 高     | ai, backend, research  |
| 7   | 渐进式进度反馈       | P2     | 中     | ai, frontend, ux       |
| 8   | 智能快捷回复生成     | P2     | 中     | ai, frontend, ux       |
| 9   | 端到端追踪系统       | P2     | 中     | ai, backend            |
| 10  | Prometheus 指标增强  | P3     | 低     | ai, backend            |
| 11  | 敏感信息过滤器       | P3     | 低     | ai, backend            |
| 12  | 预测性预加载         | P3     | 高     | ai, 性能优化, research |

---

## Issue 详细规划

### Issue 1: 语义缓存层实现

**优先级**: P1 (高)  
**复杂度**: 高  
**预估工时**: 3-5 天

#### 问题描述
当前 AmazingParrot 使用简单的 LRU 缓存，基于完整输入的 hash 进行精确匹配。这导致：
- 语义相似但措辞不同的查询无法命中缓存
- 缓存命中率预估仅 ~30%

#### 建议方案
实现基于 embedding 相似度的语义缓存层：

```go
type SemanticCache struct {
    vectors    []float32          // 已缓存查询的向量
    responses  []string           // 对应的响应
    threshold  float32            // 相似度阈值 (0.95)
    maxSize    int                // 最大缓存条目
    embedding  embedding.Service  // 向量化服务
}

func (c *SemanticCache) Get(query string) (string, bool) {
    vec := c.embedding.Encode(query)
    for i, cached := range c.vectors {
        if cosineSimilarity(vec, cached) >= c.threshold {
            return c.responses[i], true
        }
    }
    return "", false
}
```

#### 文件变更
- `ai/agent/cache.go` - 新增 SemanticCache 实现
- `ai/agent/amazing_parrot.go` - 集成多级缓存

#### 验收标准
- [ ] 单元测试覆盖率 >80%
- [ ] 基准测试显示命中率提升 >20%
- [ ] 不增加 P95 延迟超过 50ms

---

### Issue 2: 检索结果缓存

**优先级**: P1 (高)  
**复杂度**: 中  
**预估工时**: 1-2 天

#### 问题描述
日程查询和笔记搜索结果没有缓存，相同时间范围的重复查询每次都会访问数据库。

#### 建议方案
添加工具级别的结果缓存：

```go
type ToolResultCache struct {
    cache    *lru.Cache
    ttl      time.Duration
}

type CacheKey struct {
    ToolName   string
    UserID     int32
    InputHash  string
}

// 针对不同工具的 TTL 策略
var toolTTLs = map[string]time.Duration{
    "schedule_query": 30 * time.Second,  // 日程变化较少
    "memo_search":    5 * time.Minute,   // 笔记搜索结果稳定
    "find_free_time": 1 * time.Minute,   // 空闲时间需要较新
}
```

#### 文件变更
- `ai/agent/tools/cache.go` - 新增工具结果缓存
- `ai/agent/tools/scheduler.go` - 集成缓存
- `ai/agent/tools/memo_search.go` - 集成缓存

#### 验收标准
- [ ] 配置化的 TTL 策略
- [ ] 缓存失效机制 (日程变更时失效)
- [ ] 日志记录命中/未命中

---

### Issue 3: 动态 Token 预算分配

**优先级**: P1 (高)  
**复杂度**: 中  
**预估工时**: 2-3 天

#### 问题描述
当前 `BudgetAllocator` 使用固定比例分配 Token 预算，不考虑任务类型：
- 查询类应给检索更多空间
- 创建类应给历史上下文更多空间
- 闲聊类几乎不需要检索空间

#### 建议方案

```go
type AdaptiveBudgetAllocator struct {
    profiles map[Intent]*BudgetProfile
}

type BudgetProfile struct {
    ShortTermRatio float64
    LongTermRatio  float64
    RetrievalRatio float64
    UserPrefsRatio float64
}

var defaultProfiles = map[Intent]*BudgetProfile{
    IntentQuery:  {0.25, 0.10, 0.55, 0.10},
    IntentCreate: {0.50, 0.20, 0.20, 0.10},
    IntentChat:   {0.60, 0.25, 0.05, 0.10},
    IntentUpdate: {0.45, 0.15, 0.30, 0.10},
}
```

#### 文件变更
- `ai/context/budget.go` - 重构为自适应分配
- `ai/context/builder_impl.go` - 传递 Intent 信息

#### 验收标准
- [ ] 支持按 Intent 动态分配
- [ ] 配置化的 Profile 定义
- [ ] 单元测试覆盖各种场景

---

### Issue 4: 增量上下文更新

**优先级**: P2 (中)  
**复杂度**: 高  
**预估工时**: 3-5 天

#### 问题描述
每次对话都完整重建上下文，包括：
- 重新计算所有 Token 估算
- 重新排序所有上下文片段
- 重新截断到预算

多轮对话中，只有最新消息变化，其余可复用。

#### 建议方案

```go
type IncrementalContextBuilder struct {
    lastResult   *ContextResult
    lastChecksum string
    dirty        map[string]bool  // 标记哪些部分需要更新
}

func (b *IncrementalContextBuilder) Build(req *ContextRequest) (*ContextResult, error) {
    delta := b.computeDelta(req)
    
    switch {
    case delta.OnlyNewMessage:
        return b.appendOnly(req)
    case delta.RetrievalUnchanged:
        return b.updateConversationOnly(req)
    default:
        return b.fullRebuild(req)
    }
}
```

#### 文件变更
- `ai/context/incremental_builder.go` - 新增增量构建器
- `ai/context/builder_impl.go` - 添加增量模式支持

#### 验收标准
- [ ] 多轮对话场景性能提升 >30%
- [ ] 结果与完整构建一致
- [ ] 内存使用不显著增加

---

### Issue 5: 规则路由权重动态调整

**优先级**: P2 (中)  
**复杂度**: 中  
**预估工时**: 2-3 天

#### 问题描述
当前规则路由器使用静态优先级，规则效果无法量化，也无法根据实际使用调整。

#### 建议方案

```go
type AdaptiveRuleMatcher struct {
    rules       []RoutingRule
    hitCounters map[int]int64      // 规则命中计数
    successRate map[int]float64    // 规则成功率
}

func (m *AdaptiveRuleMatcher) RecordFeedback(ruleID int, wasCorrect bool) {
    m.hitCounters[ruleID]++
    if wasCorrect {
        // 提高规则权重
        m.adjustWeight(ruleID, +0.01)
    } else {
        // 降低规则权重
        m.adjustWeight(ruleID, -0.05)
    }
}
```

#### 文件变更
- `ai/router/rules.go` - 添加动态权重
- `ai/router/service.go` - 集成反馈机制

#### 验收标准
- [ ] 权重持久化存储
- [ ] 提供规则效果仪表板数据
- [ ] 支持手动重置权重

---

### Issue 6: 轻量 ML 意图分类器

**优先级**: P2 (中)  
**复杂度**: 高  
**预估工时**: 5-7 天

#### 问题描述
当规则匹配失败时，需要调用 LLM 进行意图分类，增加延迟 (~100ms) 和成本。

#### 建议方案
训练轻量级 ML 模型替代部分 LLM 调用：

```go
type TinyClassifier struct {
    model     *onnx.Model  // ONNX 格式的小模型
    tokenizer *Tokenizer
    threshold float32
}

func (c *TinyClassifier) Predict(input string) (AgentType, float32) {
    tokens := c.tokenizer.Encode(input)
    output := c.model.Run(tokens)
    
    // 获取最高置信度的类别
    maxIdx, maxConf := argmax(output)
    return AgentType(maxIdx), maxConf
}
```

#### 技术选型
- 模型: DistilBERT-tiny 或自定义 TextCNN
- 格式: ONNX (跨平台推理)
- 训练数据: 从历史日志提取

#### 文件变更
- `ai/router/classifier.go` - 新增 ML 分类器
- `ai/router/service.go` - 集成到路由流程
- `scripts/train_classifier.py` - 训练脚本

#### 验收标准
- [ ] 准确率 >85% 时启用
- [ ] 推理延迟 <10ms
- [ ] 模型大小 <10MB

---

### Issue 7: 渐进式进度反馈

**优先级**: P2 (中)  
**复杂度**: 中  
**预估工时**: 2-3 天

#### 问题描述
当前进度反馈较粗糙：thinking → tool_use → answer，用户在长时间等待时不清楚进度。

#### 建议方案

**后端增加细粒度事件**:
```go
const (
    EventTypePhaseChange = "phase_change"
    EventTypeProgress    = "progress"
)

type PhaseChangeData struct {
    Phase         string  `json:"phase"`     // analyzing, planning, retrieving, synthesizing
    EstimatedMs   int     `json:"estimated_ms"`
    ToolsPlanned  []string `json:"tools_planned"`
}

type ProgressData struct {
    Phase     string  `json:"phase"`
    Progress  int     `json:"progress"`  // 0-100
    SubPhase  string  `json:"sub_phase"`
}
```

**前端渲染**:
```tsx
function ProgressIndicator({ progress }) {
  return (
    <div className="progress-container">
      <div className="phase-label">{getPhaseLabel(progress.phase)}</div>
      <ProgressBar value={progress.progress} />
      <div className="tools-status">
        {progress.toolsInProgress.map(tool => (
          <ToolChip key={tool} name={tool} status="running" />
        ))}
      </div>
    </div>
  );
}
```

#### 文件变更
- `ai/agent/types.go` - 添加新事件类型
- `ai/agent/amazing_parrot.go` - 发送进度事件
- `web/src/components/chat/ProgressIndicator.tsx` - 新组件

#### 验收标准
- [ ] 每个阶段有明确的视觉反馈
- [ ] 工具执行进度实时更新
- [ ] 移动端适配

---

### Issue 8: 智能快捷回复生成

**优先级**: P2 (中)  
**复杂度**: 中  
**预估工时**: 2-3 天

#### 问题描述
用户完成一轮对话后，需要手动输入下一步操作，增加交互成本。

#### 建议方案

```go
func (p *AmazingParrot) GenerateQuickReplies(lastResponse string) []QuickReply {
    responseType := analyzeResponseType(lastResponse)
    
    switch responseType {
    case ResponseTypeScheduleCreated:
        return []QuickReply{
            {ID: "remind", Label: "设置提醒", Prompt: "帮我设置会议前15分钟提醒"},
            {ID: "view", Label: "查看当天", Prompt: "查看这天还有什么安排"},
            {ID: "modify", Label: "修改时间", Prompt: "改成其他时间"},
        }
    case ResponseTypeMemoFound:
        return []QuickReply{
            {ID: "more", Label: "查看更多", Prompt: "还有其他相关的吗"},
            {ID: "schedule", Label: "创建日程", Prompt: "基于这个笔记创建日程"},
            {ID: "summarize", Label: "总结", Prompt: "帮我总结这些笔记的要点"},
        }
    // ...
    }
}
```

#### 文件变更
- `ai/agent/quick_replies.go` - 快捷回复生成逻辑
- `ai/agent/types.go` - QuickReply 类型定义
- `web/src/components/chat/QuickReplies.tsx` - 前端组件

#### 验收标准
- [ ] 根据上下文动态生成
- [ ] 支持自定义快捷回复配置
- [ ] 点击后自动发送

---

### Issue 9: 端到端追踪系统

**优先级**: P2 (中)  
**复杂度**: 中  
**预估工时**: 2-3 天

#### 问题描述
难以追踪单次请求的完整链路，定位性能瓶颈和错误源困难。

#### 建议方案

```go
type TracingContext struct {
    TraceID   string
    SpanID    string
    StartTime time.Time
    
    Phases    map[string]*PhaseSpan
    ToolCalls []ToolCallSpan
    LLMCalls  []LLMCallSpan
}

type PhaseSpan struct {
    Name      string
    StartTime time.Time
    Duration  time.Duration
    Metadata  map[string]any
}

// 在 AmazingParrot 中使用
func (p *AmazingParrot) ExecuteWithCallback(...) error {
    trace := NewTracingContext()
    defer trace.Finish()
    
    trace.StartPhase("planning")
    plan, err := p.planRetrieval(ctx, ...)
    trace.EndPhase("planning")
    // ...
}
```

#### 文件变更
- `ai/tracing/context.go` - 追踪上下文
- `ai/tracing/exporter.go` - 导出到日志/Jaeger
- `ai/agent/amazing_parrot.go` - 埋点

#### 验收标准
- [ ] 每个请求生成完整 trace
- [ ] 支持导出到 Jaeger/Zipkin
- [ ] 提供 trace 查询 API

---

### Issue 10: Prometheus 指标增强

**优先级**: P3 (低)  
**复杂度**: 低  
**预估工时**: 1 天

#### 问题描述
缺少 AI 模块的业务指标监控。

#### 建议方案

```go
var (
    ChatLatency = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Namespace: "divinesense",
            Subsystem: "ai",
            Name:      "chat_latency_seconds",
            Buckets:   []float64{0.1, 0.5, 1, 2, 5, 10},
        },
        []string{"agent_type", "intent"},
    )
    
    ToolCallTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Namespace: "divinesense",
            Subsystem: "ai",
            Name:      "tool_call_total",
        },
        []string{"tool_name", "success"},
    )
    
    CacheHitRate = prometheus.NewGaugeVec(
        prometheus.GaugeOpts{
            Namespace: "divinesense",
            Subsystem: "ai",
            Name:      "cache_hit_rate",
        },
        []string{"cache_layer"},
    )
)
```

#### 文件变更
- `ai/metrics/prometheus.go` - 指标定义
- `ai/agent/amazing_parrot.go` - 埋点
- `ai/agent/tools/*.go` - 埋点

#### 验收标准
- [ ] 覆盖核心业务指标
- [ ] Grafana 仪表板模板

---

### Issue 11: 敏感信息过滤器

**优先级**: P3 (低)  
**复杂度**: 低  
**预估工时**: 1 天

#### 问题描述
AI 回复可能意外包含用户的敏感信息（手机号、身份证等）。

#### 建议方案

```go
type SensitiveFilter struct {
    patterns []*regexp.Regexp
    masks    map[string]string
}

var defaultPatterns = []*regexp.Regexp{
    regexp.MustCompile(`\b1[3-9]\d{9}\b`),           // 手机号
    regexp.MustCompile(`\b\d{17}[\dXx]\b`),          // 身份证
    regexp.MustCompile(`[\w.-]+@[\w.-]+\.\w+`),      // 邮箱
    regexp.MustCompile(`\d{4}[\s-]?\d{4}[\s-]?\d{4}[\s-]?\d{4}`), // 银行卡
}

func (f *SensitiveFilter) Filter(output string) string {
    for _, pattern := range f.patterns {
        output = pattern.ReplaceAllStringFunc(output, func(s string) string {
            return maskSensitive(s)
        })
    }
    return output
}
```

#### 文件变更
- `ai/security/sensitive_filter.go` - 过滤器实现
- `ai/agent/amazing_parrot.go` - 在 synthesis 后应用

#### 验收标准
- [ ] 覆盖常见敏感信息类型
- [ ] 支持自定义规则
- [ ] 记录过滤日志

---

### Issue 12: 预测性预加载

**优先级**: P3 (低)  
**复杂度**: 高  
**预估工时**: 5-7 天

#### 问题描述
用户首次查询时需要完整的检索流程，延迟较高。

#### 建议方案

```go
type PredictiveLoader struct {
    userPatterns map[int32]*UserPattern
    scheduler    *cron.Scheduler
}

type UserPattern struct {
    MorningScheduleQuery    bool      // 通常早上查日程
    FrequentMemoTopics      []string  // 常搜索的笔记主题
    ActiveHours             []int     // 活跃时段
}

func (l *PredictiveLoader) PrefetchForUser(userID int32) {
    pattern := l.userPatterns[userID]
    
    if pattern.MorningScheduleQuery && isMorning() {
        go l.warmupScheduleCache(userID, "today")
    }
    
    for _, topic := range pattern.FrequentMemoTopics {
        go l.warmupMemoEmbedding(userID, topic)
    }
}
```

#### 文件变更
- `ai/prefetch/loader.go` - 预加载器
- `ai/prefetch/patterns.go` - 用户模式分析
- `server/cron/prefetch_job.go` - 定时任务

#### 验收标准
- [ ] 基于历史行为分析用户模式
- [ ] 后台定时预加载
- [ ] 不影响系统负载（资源限制）

---

## 执行计划

### Phase 1 (Week 1-2): 基础性能优化
- Issue 1: 语义缓存层
- Issue 2: 检索结果缓存
- Issue 3: 动态 Token 预算

### Phase 2 (Week 3-4): 路由与体验优化
- Issue 5: 规则路由权重
- Issue 7: 渐进式进度反馈
- Issue 8: 智能快捷回复

### Phase 3 (Week 5-6): 高级特性
- Issue 4: 增量上下文
- Issue 9: 端到端追踪
- Issue 10: Prometheus 指标

### Phase 4 (Week 7-8): 研究性特性
- Issue 6: ML 意图分类器
- Issue 11: 敏感信息过滤
- Issue 12: 预测性预加载
