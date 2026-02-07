# P2-A001: Self-RAG 检索优化

> **状态**: 🔲 待开发  
> **优先级**: P1 (重要)  
> **投入**: 3 人天  
> **负责团队**: 团队 A  
> **Sprint**: Sprint 3

---

## 1. 目标与背景

### 1.1 核心目标

实现轻量级 Self-RAG（规则驱动，非 LLM），在检索前判断"是否需要检索"，检索后评估"结果是否有用"，减少无效检索 40%+。

### 1.2 用户价值

- 响应速度提升（跳过不必要的检索）
- 回答质量提升（避免无关信息干扰）

### 1.3 技术价值

- API 成本降低 40%+
- 为 Phase 3 本地模型铺路

---

## 2. 依赖关系

### 2.1 前置依赖

- [x] P1-A005: 通用缓存层（缓存检索结果）
- [x] P1-A003: LLM 路由优化（意图分类基础）

### 2.2 并行依赖

- P2-A002: 上下文增强构建器（可并行）

### 2.3 后续依赖

- P2-C001: 智能标签建议（依赖检索服务）
- P2-C002: 重复检测（依赖相似度计算）

---

## 3. 功能设计

### 3.1 架构图

```
                    Self-RAG 决策流程
┌────────────────────────────────────────────────────────────┐
│                                                            │
│    Query                                                   │
│      │                                                     │
│      ▼                                                     │
│  ┌──────────────────┐                                     │
│  │  Layer 1: 需检索?  │  规则判断 (0ms)                     │
│  │                   │                                     │
│  │  - 闲聊类 → No    │                                     │
│  │  - 系统命令 → No  │                                     │
│  │  - 检索词 → Yes   │                                     │
│  └──────────────────┘                                     │
│           │                                                │
│     ┌─────┴─────┐                                         │
│     │           │                                         │
│    Yes          No                                         │
│     │           │                                         │
│     ▼           ▼                                         │
│  ┌───────┐  ┌────────────┐                               │
│  │Retrieve│  │Direct Answer│                               │
│  └───────┘  └────────────┘                               │
│     │                                                      │
│     ▼                                                      │
│  ┌──────────────────┐                                     │
│  │  Layer 2: 有用?   │  分数判断 (0ms)                     │
│  │                   │                                     │
│  │  - Top1 > 0.6 Yes│                                     │
│  │  - 空结果 → No   │                                     │
│  └──────────────────┘                                     │
│           │                                                │
│     ┌─────┴─────┐                                         │
│     │           │                                         │
│    Yes          No                                         │
│     │           │                                         │
│     ▼           ▼                                         │
│  ┌─────────┐ ┌───────────┐                               │
│  │Grounded │ │Retry/Expand│                               │
│  │ Answer  │ │ or Direct  │                               │
│  └─────────┘ └───────────┘                               │
│                                                            │
└────────────────────────────────────────────────────────────┘
```

### 3.2 检索决策规则

```go
// plugin/ai/rag/retrieval_decision.go

type RetrievalDecision struct {
    ShouldRetrieve bool
    Reason         string
}

// 关键词列表
var (
    chitchatPatterns = []string{"你好", "谢谢", "再见", "哈哈", "好的", "嗯"}
    systemCommands   = []string{"帮助", "设置", "退出", "清空"}
    retrievalTriggers = []string{"搜索", "查找", "找到", "记录", "笔记", "日程"}
)

func DecideRetrieval(query string) *RetrievalDecision {
    // Rule 1: 闲聊不检索
    for _, pattern := range chitchatPatterns {
        if strings.HasPrefix(strings.TrimSpace(query), pattern) {
            return &RetrievalDecision{
                ShouldRetrieve: false,
                Reason:         "chitchat_detected",
            }
        }
    }
    
    // Rule 2: 系统命令不检索
    for _, cmd := range systemCommands {
        if strings.Contains(query, cmd) {
            return &RetrievalDecision{
                ShouldRetrieve: false,
                Reason:         "system_command",
            }
        }
    }
    
    // Rule 3: 检索触发词
    for _, trigger := range retrievalTriggers {
        if strings.Contains(query, trigger) {
            return &RetrievalDecision{
                ShouldRetrieve: true,
                Reason:         "retrieval_trigger",
            }
        }
    }
    
    // Rule 4: 默认检索
    return &RetrievalDecision{
        ShouldRetrieve: true,
        Reason:         "default",
    }
}
```

### 3.3 结果有效性评估

```go
// plugin/ai/rag/result_evaluator.go

const (
    UsefulScoreThreshold = 0.6  // Top1 分数阈值
    MinResultsForRerank  = 5    // 最小重排数量
)

type EvaluationResult struct {
    IsUseful       bool
    Reason         string
    SuggestedAction string  // "use", "expand", "direct"
}

func EvaluateResults(results []*SearchResult) *EvaluationResult {
    // 空结果
    if len(results) == 0 {
        return &EvaluationResult{
            IsUseful:        false,
            Reason:          "empty_results",
            SuggestedAction: "direct",
        }
    }
    
    // Top1 分数判断
    top1Score := results[0].Score
    if top1Score > UsefulScoreThreshold {
        return &EvaluationResult{
            IsUseful:        true,
            Reason:          "high_relevance",
            SuggestedAction: "use",
        }
    }
    
    // 低分数：扩展查询
    return &EvaluationResult{
        IsUseful:        false,
        Reason:          "low_relevance",
        SuggestedAction: "expand",
    }
}
```

### 3.4 Reranker 触发条件

```go
// plugin/ai/rag/reranker.go

const (
    ScoreDiffThreshold = 0.15  // 分差阈值
)

func ShouldRerank(query string, results []*SearchResult) bool {
    // 结果太少不重排
    if len(results) < MinResultsForRerank {
        return false
    }
    
    // 简单关键词查询不重排
    if isSimpleKeywordQuery(query) {
        return false
    }
    
    // 分差大不重排（Top1 明显胜出）
    if len(results) >= 2 {
        scoreDiff := results[0].Score - results[1].Score
        if scoreDiff > ScoreDiffThreshold {
            return false
        }
    }
    
    return true
}

func isSimpleKeywordQuery(query string) bool {
    words := strings.Fields(query)
    return len(words) <= 2
}
```

### 3.5 混合检索策略

```go
// plugin/ai/rag/hybrid_search.go

type SearchStrategy string

const (
    StrategyBM25Only        SearchStrategy = "schedule_bm25_only"
    StrategySemanticOnly    SearchStrategy = "memo_semantic_only"
    StrategyHybridStandard  SearchStrategy = "hybrid_standard"
    StrategyHybridBM25Heavy SearchStrategy = "hybrid_bm25_weighted"
    StrategyFullPipeline    SearchStrategy = "full_pipeline_with_reranker"
)

type StrategyConfig struct {
    BM25Weight   float64
    VectorWeight float64
    UseReranker  bool
}

var strategyConfigs = map[SearchStrategy]StrategyConfig{
    StrategyBM25Only:        {BM25Weight: 1.0, VectorWeight: 0.0, UseReranker: false},
    StrategySemanticOnly:    {BM25Weight: 0.0, VectorWeight: 1.0, UseReranker: false},
    StrategyHybridStandard:  {BM25Weight: 0.5, VectorWeight: 0.5, UseReranker: false},
    StrategyHybridBM25Heavy: {BM25Weight: 0.7, VectorWeight: 0.3, UseReranker: false},
    StrategyFullPipeline:    {BM25Weight: 0.5, VectorWeight: 0.5, UseReranker: true},
}

func SelectStrategy(intent Intent) SearchStrategy {
    switch intent {
    case IntentScheduleQuery:
        return StrategyBM25Only  // 日程用 BM25
    case IntentMemoSearch:
        return StrategySemanticOnly  // 笔记用向量
    default:
        return StrategyHybridStandard  // 默认混合
    }
}
```

### 3.6 RRF 倒数排名融合

```go
// plugin/ai/rag/rrf.go

const RRFDampingFactor = 60  // k = 60

// RRF(d) = Σ weight_i / (k + rank_i(d))
func FuseWithRRF(bm25Results, vectorResults []*SearchResult, config StrategyConfig) []*SearchResult {
    scoreMap := make(map[string]float64)
    
    // BM25 分数贡献
    for rank, result := range bm25Results {
        score := config.BM25Weight / float64(RRFDampingFactor+rank+1)
        scoreMap[result.ID] += score
    }
    
    // 向量分数贡献
    for rank, result := range vectorResults {
        score := config.VectorWeight / float64(RRFDampingFactor+rank+1)
        scoreMap[result.ID] += score
    }
    
    // 合并排序
    return sortByScore(scoreMap)
}
```

---

## 4. 实现路径

### Day 1: 检索决策层

- [ ] 实现 `retrieval_decision.go`
- [ ] 规则配置外部化
- [ ] 单元测试覆盖

### Day 2: 结果评估层

- [ ] 实现 `result_evaluator.go`
- [ ] 实现 `reranker.go` 触发逻辑
- [ ] 集成 bge-reranker（可选）

### Day 3: 混合检索与集成

- [ ] 实现 `hybrid_search.go`
- [ ] 实现 `rrf.go`
- [ ] 与现有 Agent 集成
- [ ] 端到端测试

---

## 5. 交付物

### 5.1 代码产出

| 文件 | 说明 |
|:---|:---|
| `plugin/ai/rag/retrieval_decision.go` | 检索决策规则 |
| `plugin/ai/rag/result_evaluator.go` | 结果有效性评估 |
| `plugin/ai/rag/reranker.go` | Reranker 触发逻辑 |
| `plugin/ai/rag/hybrid_search.go` | 混合检索策略 |
| `plugin/ai/rag/rrf.go` | RRF 融合算法 |
| `plugin/ai/rag/*_test.go` | 单元测试 |

### 5.2 配置项

```yaml
# configs/ai.yaml
self_rag:
  useful_score_threshold: 0.6
  min_results_for_rerank: 5
  score_diff_threshold: 0.15
  rrf_damping_factor: 60
  
  retrieval_patterns:
    chitchat:
      - "你好"
      - "谢谢"
      - "再见"
    triggers:
      - "搜索"
      - "查找"
      - "笔记"
```

---

## 6. 验收标准

### 6.1 功能验收

- [ ] 闲聊类查询跳过检索（"你好" → 直接回复）
- [ ] 检索触发词正确检索（"搜索笔记" → 执行检索）
- [ ] 低相关性结果触发扩展/直接回答

### 6.2 性能验收

- [ ] 检索决策延迟 < 1ms
- [ ] 无效检索减少 40%+
- [ ] API 调用成本降低（可度量）

### 6.3 测试用例

```go
func TestRetrievalDecision(t *testing.T) {
    tests := []struct {
        query    string
        expected bool
    }{
        {"你好", false},           // 闲聊
        {"谢谢帮助", false},        // 闲聊
        {"搜索我的笔记", true},      // 触发词
        {"明天的日程", true},        // 默认
        {"帮助", false},           // 系统命令
    }
    
    for _, tt := range tests {
        decision := DecideRetrieval(tt.query)
        assert.Equal(t, tt.expected, decision.ShouldRetrieve)
    }
}
```

---

## 7. ROI 分析

| 投入 | 产出 |
|:---|:---|
| 开发: 3 人天 | 无效检索减少 40% |
| 存储: 0 | API 成本降低 20%+ |
| 维护: 规则可配置 | 响应速度提升 30%+ |

### 收益计算

- 假设每日 1000 次 AI 查询
- 当前 100% 执行检索 → 优化后 60% 执行
- 每次检索成本约 ¥0.01
- 月节省: 1000 × 30 × 40% × ¥0.01 = ¥120

---

## 8. 风险与缓解

| 风险 | 概率 | 影响 | 缓解措施 |
|:---|:---:|:---:|:---|
| 规则误判 | 中 | 中 | 添加监控指标，持续优化规则 |
| 阈值不当 | 中 | 低 | 配置外部化，支持动态调整 |
| Reranker 延迟 | 低 | 中 | 设置超时，降级为不重排 |

---

## 9. 排期

| 日期 | 任务 | 负责人 |
|:---|:---|:---|
| Sprint 3 Day 1 | 检索决策层 | TBD |
| Sprint 3 Day 2 | 结果评估层 | TBD |
| Sprint 3 Day 3 | 混合检索与集成测试 | TBD |

---

> **纲领来源**: [00-master-roadmap.md](../../../archived/research_20250202/00-master-roadmap.md)  
> **研究文档**: [memo-roadmap.md](../../../research/memo-roadmap.md)  
> **版本**: v1.0  
> **更新时间**: 2026-01-27
