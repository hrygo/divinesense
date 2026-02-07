# P2-C002: 重复检测系统

> **状态**: 🔄 进行中  
> **优先级**: P2 (增强)  
> **投入**: 9 人天  
> **负责团队**: 团队 C  
> **Sprint**: Sprint 4

---

## 1. 目标与背景

### 1.1 核心目标

实现笔记重复检测系统，在创建新笔记时自动检测相似内容，支持合并、关联或忽略操作，重复识别率达到 80%+。

### 1.2 用户价值

- 减少重复记录
- 发现相关笔记，促进知识关联
- 更干净的知识库

### 1.3 技术价值

- 三维相似度计算
- 为知识图谱（P3-C001）奠定基础
- 可复用的相似度服务

---

## 2. 依赖关系

### 2.1 前置依赖

- [x] P1-A005: 通用缓存层（缓存检测结果）
- [x] P1-C001: 搜索结果高亮（向量检索基础）

### 2.2 并行依赖

- P2-C001: 智能标签建议（可并行）

### 2.3 后续依赖

- P3-C001: 知识图谱可视化
- P3-C002: 智能回顾系统

---

## 3. 功能设计

### 3.1 架构图

```
                    重复检测系统架构
┌────────────────────────────────────────────────────────────┐
│                                                            │
│   新笔记创建                                                │
│         │                                                  │
│         ▼                                                  │
│   ┌─────────────────────────────────────────────────────┐ │
│   │              DuplicateDetector                       │ │
│   │                                                      │ │
│   │  Step 1: 向量化新笔记                                │ │
│   │          └─ Embedding API                           │ │
│   │                                                      │ │
│   │  Step 2: 三维相似度检索                              │ │
│   │          ├─ 向量相似度 (0.5)                        │ │
│   │          ├─ 标签共现 (0.3)                          │ │
│   │          └─ 时间邻近 (0.2)                          │ │
│   │                                                      │ │
│   │  Step 3: 分级决策                                    │ │
│   │          ├─ >90%: 可能重复 → 警告                   │ │
│   │          ├─ 70-90%: 相关内容 → 提示                 │ │
│   │          └─ <70%: 正常 → 无提示                     │ │
│   └─────────────────────────────────────────────────────┘ │
│         │                                                  │
│         ▼                                                  │
│   ┌─────────────────────────────────────────────────────┐ │
│   │              用户决策                                │ │
│   │                                                      │ │
│   │  [合并] → 合并到已有笔记                             │ │
│   │  [关联] → 建立双向链接                               │ │
│   │  [忽略] → 继续创建                                   │ │
│   └─────────────────────────────────────────────────────┘ │
│                                                            │
└────────────────────────────────────────────────────────────┘
```

### 3.2 核心接口定义

```go
// plugin/ai/duplicate/detector.go

type DuplicateDetector interface {
    // 检测重复
    Detect(ctx context.Context, req *DetectRequest) (*DetectResponse, error)
    
    // 合并笔记
    Merge(ctx context.Context, sourceID, targetID string) error
    
    // 建立关联
    Link(ctx context.Context, memoID1, memoID2 string) error
}

type DetectRequest struct {
    UserID   int32
    Title    string
    Content  string
    Tags     []string
    TopK     int  // 返回最相似的 K 条 (默认 5)
}

type DetectResponse struct {
    HasDuplicate   bool             `json:"has_duplicate"`
    HasRelated     bool             `json:"has_related"`
    Duplicates     []SimilarMemo    `json:"duplicates,omitempty"`
    Related        []SimilarMemo    `json:"related,omitempty"`
}

type SimilarMemo struct {
    ID             string   `json:"id"`
    Name           string   `json:"name"`
    Title          string   `json:"title"`
    Snippet        string   `json:"snippet"`
    Similarity     float64  `json:"similarity"`
    SharedTags     []string `json:"shared_tags,omitempty"`
    Level          string   `json:"level"`  // "duplicate", "related"
}
```

### 3.3 三维相似度计算

```go
// plugin/ai/duplicate/similarity.go

type SimilarityCalculator struct {
    vectorStore VectorStore
    memoStore   MemoStore
}

type SimilarityWeights struct {
    Vector    float64  // 向量相似度权重
    TagCoOccur float64 // 标签共现权重
    TimeProx  float64  // 时间邻近权重
}

var DefaultWeights = SimilarityWeights{
    Vector:    0.5,
    TagCoOccur: 0.3,
    TimeProx:  0.2,
}

func (c *SimilarityCalculator) Calculate(ctx context.Context, userID int32, newMemo *MemoInput, candidateID string) (float64, *SimilarityBreakdown, error) {
    // 获取候选笔记
    candidate, err := c.memoStore.GetMemo(ctx, candidateID)
    if err != nil {
        return 0, nil, err
    }
    
    breakdown := &SimilarityBreakdown{}
    
    // 1. 向量相似度 (cosine)
    vectorSim, err := c.calculateVectorSimilarity(ctx, newMemo.Content, candidate.Content)
    if err != nil {
        vectorSim = 0
    }
    breakdown.Vector = vectorSim
    
    // 2. 标签共现率
    tagSim := c.calculateTagCoOccurrence(newMemo.Tags, candidate.Tags)
    breakdown.TagCoOccur = tagSim
    
    // 3. 时间邻近度 (7天衰减)
    timeSim := c.calculateTimeProximity(time.Now(), candidate.CreatedAt)
    breakdown.TimeProx = timeSim
    
    // 加权求和
    total := vectorSim*DefaultWeights.Vector +
             tagSim*DefaultWeights.TagCoOccur +
             timeSim*DefaultWeights.TimeProx
    
    return total, breakdown, nil
}

type SimilarityBreakdown struct {
    Vector    float64 `json:"vector"`
    TagCoOccur float64 `json:"tag_co_occur"`
    TimeProx  float64 `json:"time_prox"`
}
```

### 3.4 向量相似度

```go
// plugin/ai/duplicate/vector_similarity.go

func (c *SimilarityCalculator) calculateVectorSimilarity(ctx context.Context, content1, content2 string) (float64, error) {
    // 获取向量
    vec1, err := c.vectorStore.GetOrCreateEmbedding(ctx, content1)
    if err != nil {
        return 0, err
    }
    
    vec2, err := c.vectorStore.GetOrCreateEmbedding(ctx, content2)
    if err != nil {
        return 0, err
    }
    
    // 计算余弦相似度
    return cosineSimilarity(vec1, vec2), nil
}

func cosineSimilarity(a, b []float32) float64 {
    var dotProduct, normA, normB float64
    
    for i := range a {
        dotProduct += float64(a[i]) * float64(b[i])
        normA += float64(a[i]) * float64(a[i])
        normB += float64(b[i]) * float64(b[i])
    }
    
    if normA == 0 || normB == 0 {
        return 0
    }
    
    return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
}
```

### 3.5 标签共现率

```go
// plugin/ai/duplicate/tag_similarity.go

func (c *SimilarityCalculator) calculateTagCoOccurrence(tags1, tags2 []string) float64 {
    if len(tags1) == 0 && len(tags2) == 0 {
        return 0
    }
    
    // 构建集合
    set1 := make(map[string]bool)
    for _, tag := range tags1 {
        set1[strings.ToLower(tag)] = true
    }
    
    set2 := make(map[string]bool)
    for _, tag := range tags2 {
        set2[strings.ToLower(tag)] = true
    }
    
    // 计算交集
    var intersection int
    for tag := range set1 {
        if set2[tag] {
            intersection++
        }
    }
    
    // Jaccard 相似度
    union := len(set1) + len(set2) - intersection
    if union == 0 {
        return 0
    }
    
    return float64(intersection) / float64(union)
}
```

### 3.6 时间邻近度

```go
// plugin/ai/duplicate/time_similarity.go

const TimeDecayDays = 7  // 7天衰减周期

func (c *SimilarityCalculator) calculateTimeProximity(newTime, candidateTime time.Time) float64 {
    // 计算时间差（天）
    daysDiff := newTime.Sub(candidateTime).Hours() / 24
    
    if daysDiff < 0 {
        daysDiff = -daysDiff
    }
    
    // 指数衰减: e^(-days/7)
    return math.Exp(-daysDiff / TimeDecayDays)
}
```

### 3.7 重复检测器实现

```go
// plugin/ai/duplicate/detector_impl.go

const (
    DuplicateThreshold = 0.9  // >90% 为重复
    RelatedThreshold   = 0.7  // 70-90% 为相关
)

type duplicateDetector struct {
    calculator *SimilarityCalculator
    memoStore  MemoStore
    vectorStore VectorStore
    cache      CacheService
}

func NewDuplicateDetector(memoStore MemoStore, vectorStore VectorStore, cache CacheService) DuplicateDetector {
    return &duplicateDetector{
        calculator: &SimilarityCalculator{
            vectorStore: vectorStore,
            memoStore:   memoStore,
        },
        memoStore:   memoStore,
        vectorStore: vectorStore,
        cache:       cache,
    }
}

func (d *duplicateDetector) Detect(ctx context.Context, req *DetectRequest) (*DetectResponse, error) {
    response := &DetectResponse{}
    
    // Step 1: 向量检索候选笔记
    candidates, err := d.vectorStore.SearchSimilar(ctx, req.UserID, req.Content, req.TopK*2)
    if err != nil {
        return nil, fmt.Errorf("vector search failed: %w", err)
    }
    
    // Step 2: 精确计算三维相似度
    var similarities []SimilarMemo
    for _, candidate := range candidates {
        score, breakdown, err := d.calculator.Calculate(ctx, req.UserID, &MemoInput{
            Title:   req.Title,
            Content: req.Content,
            Tags:    req.Tags,
        }, candidate.ID)
        
        if err != nil {
            continue
        }
        
        if score >= RelatedThreshold {
            level := "related"
            if score >= DuplicateThreshold {
                level = "duplicate"
            }
            
            similarities = append(similarities, SimilarMemo{
                ID:         candidate.ID,
                Name:       candidate.Name,
                Title:      extractTitle(candidate.Content),
                Snippet:    truncate(candidate.Content, 100),
                Similarity: score,
                SharedTags: findSharedTags(req.Tags, candidate.Tags),
                Level:      level,
            })
        }
    }
    
    // Step 3: 分类
    for _, sim := range similarities {
        if sim.Level == "duplicate" {
            response.Duplicates = append(response.Duplicates, sim)
            response.HasDuplicate = true
        } else {
            response.Related = append(response.Related, sim)
            response.HasRelated = true
        }
    }
    
    // 排序：相似度降序
    sortBySimilarity(response.Duplicates)
    sortBySimilarity(response.Related)
    
    // 限制数量
    if len(response.Duplicates) > req.TopK {
        response.Duplicates = response.Duplicates[:req.TopK]
    }
    if len(response.Related) > req.TopK {
        response.Related = response.Related[:req.TopK]
    }
    
    return response, nil
}
```

### 3.8 合并与关联

```go
// plugin/ai/duplicate/merge.go

func (d *duplicateDetector) Merge(ctx context.Context, sourceID, targetID string) error {
    // 获取源笔记
    source, err := d.memoStore.GetMemo(ctx, sourceID)
    if err != nil {
        return err
    }
    
    // 获取目标笔记
    target, err := d.memoStore.GetMemo(ctx, targetID)
    if err != nil {
        return err
    }
    
    // 合并内容
    mergedContent := target.Content + "\n\n---\n\n" + source.Content
    
    // 合并标签
    mergedTags := mergeTags(target.Tags, source.Tags)
    
    // 更新目标笔记
    err = d.memoStore.UpdateMemo(ctx, targetID, &MemoUpdate{
        Content: mergedContent,
        Tags:    mergedTags,
    })
    if err != nil {
        return err
    }
    
    // 删除源笔记（或标记为已合并）
    err = d.memoStore.ArchiveMemo(ctx, sourceID, "merged_to:"+targetID)
    
    return err
}

func (d *duplicateDetector) Link(ctx context.Context, memoID1, memoID2 string) error {
    // 建立双向关联
    err := d.memoStore.AddRelation(ctx, memoID1, memoID2, "related")
    if err != nil {
        return err
    }
    
    return d.memoStore.AddRelation(ctx, memoID2, memoID1, "related")
}
```

### 3.9 API 与前端

```go
// server/router/api/v1/duplicate_handler.go

// POST /api/v1/memos/duplicate-check
func (h *MemoHandler) HandleDuplicateCheck(c *gin.Context) {
    var req DetectRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    
    req.UserID = getUserID(c)
    if req.TopK == 0 {
        req.TopK = 5
    }
    
    response, err := h.detector.Detect(c.Request.Context(), &req)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    
    c.JSON(http.StatusOK, response)
}
```

```tsx
// web/src/components/memo/DuplicateWarning.tsx

interface DuplicateWarningProps {
  duplicates: SimilarMemo[];
  related: SimilarMemo[];
  onMerge: (targetId: string) => void;
  onLink: (memoId: string) => void;
  onIgnore: () => void;
}

export function DuplicateWarning({
  duplicates,
  related,
  onMerge,
  onLink,
  onIgnore,
}: DuplicateWarningProps) {
  if (duplicates.length === 0 && related.length === 0) {
    return null;
  }

  return (
    <div className="rounded-lg border border-yellow-200 bg-yellow-50 p-4">
      {duplicates.length > 0 && (
        <div className="mb-4">
          <h4 className="flex items-center gap-2 font-medium text-yellow-800">
            <AlertTriangle className="h-4 w-4" />
            发现相似笔记
          </h4>
          <div className="mt-2 space-y-2">
            {duplicates.map((memo) => (
              <div
                key={memo.id}
                className="flex items-center justify-between rounded bg-white p-2"
              >
                <div>
                  <p className="font-medium">{memo.title}</p>
                  <p className="text-sm text-gray-500">{memo.snippet}</p>
                  <p className="text-xs text-yellow-600">
                    相似度: {(memo.similarity * 100).toFixed(0)}%
                  </p>
                </div>
                <div className="flex gap-2">
                  <Button size="sm" onClick={() => onMerge(memo.id)}>
                    合并
                  </Button>
                  <Button size="sm" variant="outline" onClick={() => onLink(memo.id)}>
                    关联
                  </Button>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      {related.length > 0 && (
        <div>
          <h4 className="text-sm font-medium text-gray-600">相关笔记</h4>
          <div className="mt-2 flex flex-wrap gap-2">
            {related.map((memo) => (
              <button
                key={memo.id}
                onClick={() => onLink(memo.id)}
                className="rounded-full bg-gray-100 px-3 py-1 text-sm hover:bg-gray-200"
              >
                {memo.title}
              </button>
            ))}
          </div>
        </div>
      )}

      <div className="mt-4 flex justify-end">
        <Button variant="ghost" onClick={onIgnore}>
          忽略，继续创建
        </Button>
      </div>
    </div>
  );
}
```

---

## 4. 实现路径

### Day 1-2: 相似度计算

- [ ] 向量相似度
- [ ] 标签共现率
- [ ] 时间邻近度
- [ ] 加权计算

### Day 3-4: 检测器实现

- [ ] 候选检索
- [ ] 精确计算
- [ ] 分级决策

### Day 5-6: 合并与关联

- [ ] 合并逻辑
- [ ] 关联逻辑
- [ ] 数据库操作

### Day 7-8: API 与前端

- [ ] API Handler
- [ ] 前端组件
- [ ] 交互流程

### Day 9: 测试与优化

- [ ] 单元测试
- [ ] 端到端测试
- [ ] 性能优化

---

## 5. 交付物

### 5.1 代码产出

| 文件 | 说明 |
|:---|:---|
| `plugin/ai/duplicate/detector.go` | 接口定义 |
| `plugin/ai/duplicate/similarity.go` | 相似度计算 |
| `plugin/ai/duplicate/vector_similarity.go` | 向量相似度 |
| `plugin/ai/duplicate/tag_similarity.go` | 标签相似度 |
| `plugin/ai/duplicate/time_similarity.go` | 时间相似度 |
| `plugin/ai/duplicate/detector_impl.go` | 检测器实现 |
| `plugin/ai/duplicate/merge.go` | 合并与关联 |
| `server/router/api/v1/duplicate_handler.go` | API |
| `web/src/components/memo/DuplicateWarning.tsx` | 前端 |

### 5.2 配置项

```yaml
# configs/ai.yaml
duplicate_detection:
  enabled: true
  duplicate_threshold: 0.9
  related_threshold: 0.7
  top_k: 5
  
  weights:
    vector: 0.5
    tag_co_occur: 0.3
    time_prox: 0.2
    
  time_decay_days: 7
```

---

## 6. 验收标准

### 6.1 功能验收

| 场景 | 期望结果 |
|:---|:---|
| 相似度 >90% | 显示重复警告，提供合并选项 |
| 相似度 70-90% | 显示相关提示，提供关联选项 |
| 相似度 <70% | 无提示，正常创建 |
| 用户选择合并 | 内容合并，源笔记归档 |
| 用户选择关联 | 建立双向链接 |

### 6.2 性能验收

- [ ] 检测延迟 < 500ms
- [ ] 重复识别率 > 80%
- [ ] 误报率 < 10%

### 6.3 测试用例

```go
func TestDuplicateDetection(t *testing.T) {
    detector := NewDuplicateDetector(mockMemoStore, mockVectorStore, mockCache)
    
    // 创建测试数据
    existingMemo := &Memo{
        ID:      "memo-1",
        Content: "React Hooks 是 React 16.8 引入的新特性",
        Tags:    []string{"React", "学习"},
    }
    mockMemoStore.Create(context.Background(), 1, existingMemo)
    
    // 检测相似内容
    req := &DetectRequest{
        UserID:  1,
        Title:   "React 学习笔记",
        Content: "今天学习了 React Hooks 的用法",
        Tags:    []string{"React"},
        TopK:    5,
    }
    
    resp, err := detector.Detect(context.Background(), req)
    
    assert.NoError(t, err)
    assert.True(t, resp.HasDuplicate || resp.HasRelated)
}
```

---

## 7. ROI 分析

| 投入 | 产出 |
|:---|:---|
| 开发: 9 人天 | 重复笔记减少 30%+ |
| 存储: 关联索引 | 知识发现能力 |
| 维护: 阈值可配置 | 更干净的知识库 |

---

## 8. 风险与缓解

| 风险 | 概率 | 影响 | 缓解措施 |
|:---|:---:|:---:|:---|
| 向量检索慢 | 中 | 中 | 索引优化 + 限制候选数 |
| 误报过多 | 中 | 中 | 调整阈值，用户可忽略 |
| 合并冲突 | 低 | 中 | 乐观锁 + 冲突提示 |

---

## 9. 排期

| 日期 | 任务 | 负责人 |
|:---|:---|:---|
| Sprint 4 Day 1-2 | 相似度计算 | TBD |
| Sprint 4 Day 3-4 | 检测器实现 | TBD |
| Sprint 4 Day 5-6 | 合并与关联 | TBD |
| Sprint 4 Day 7-8 | API 与前端 | TBD |
| Sprint 4 Day 9 | 测试与优化 | TBD |

---

> **纲领来源**: [00-master-roadmap.md](../../../archived/research_20250202/00-master-roadmap.md)  
> **研究文档**: [memo-roadmap.md](../../../research/memo-roadmap.md)  
> **版本**: v1.0  
> **更新时间**: 2026-01-27
