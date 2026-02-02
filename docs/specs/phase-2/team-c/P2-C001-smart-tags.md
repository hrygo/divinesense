# P2-C001: 智能标签建议

> **状态**: 🔲 待开发  
> **优先级**: P1 (重要)  
> **投入**: 7 人天  
> **负责团队**: 团队 C  
> **Sprint**: Sprint 3

---

## 1. 目标与背景

### 1.1 核心目标

实现三层渐进式智能标签建议系统：统计优先（0ms）→ 规则提取（10ms）→ LLM 语义（可选），提升标签采纳率 350%+。

### 1.2 用户价值

- 标签输入成本降低 70%
- 标签一致性提升
- 更好的笔记组织体验

### 1.3 技术价值

- 三层降级策略，高可用
- LLM 调用可选，成本可控
- 为知识图谱（P3-C001）奠定基础

---

## 2. 依赖关系

### 2.1 前置依赖

- [x] P1-A003: LLM 路由优化（LLM 调用基础）
- [x] P1-A005: 通用缓存层（缓存标签建议）

### 2.2 并行依赖

- P2-C002: 重复检测系统（可并行）

### 2.3 后续依赖

- P3-C001: 知识图谱可视化

---

## 3. 功能设计

### 3.1 架构图

```
                    三层渐进式标签建议
┌────────────────────────────────────────────────────────────┐
│                                                            │
│   笔记内容输入                                              │
│         │                                                  │
│         ▼                                                  │
│   ┌─────────────────────────────────────────────────────┐ │
│   │            Layer 1: 统计优先 (0ms)                   │ │
│   │                                                      │ │
│   │  • 用户历史高频标签 (TOP-5)                          │ │
│   │  • 最近 7 天使用的标签                               │ │
│   │  • 相似笔记的标签                                    │ │
│   │                                                      │ │
│   │  无 LLM 调用，毫秒级响应                             │ │
│   └─────────────────────────────────────────────────────┘ │
│         │                                                  │
│         ▼                                                  │
│   ┌─────────────────────────────────────────────────────┐ │
│   │            Layer 2: 规则提取 (10ms)                  │ │
│   │                                                      │ │
│   │  • 专有名词识别 (React, Python, AI 等)               │ │
│   │  • 日期/时间模式 (#2024-01, #Q1)                     │ │
│   │  • 情感词识别 (#灵感, #问题, #待办)                   │ │
│   │                                                      │ │
│   │  正则表达式匹配，本地处理                            │ │
│   └─────────────────────────────────────────────────────┘ │
│         │                                                  │
│         ▼                                                  │
│   ┌─────────────────────────────────────────────────────┐ │
│   │            Layer 3: LLM 语义 (300ms，可选)           │ │
│   │                                                      │ │
│   │  • 主题分类 (#技术, #生活, #工作)                    │ │
│   │  • 新标签发现                                        │ │
│   │  • 语义理解                                          │ │
│   │                                                      │ │
│   │  降级策略：网络异常跳过，仅返回 L1/L2                │ │
│   └─────────────────────────────────────────────────────┘ │
│         │                                                  │
│         ▼                                                  │
│   ┌─────────────────────────────────────────────────────┐ │
│   │            合并去重 → 排序 → 返回 TOP-N              │ │
│   └─────────────────────────────────────────────────────┘ │
│                                                            │
└────────────────────────────────────────────────────────────┘
```

### 3.2 核心接口定义

```go
// plugin/ai/tags/suggester.go

type TagSuggester interface {
    // 获取标签建议
    Suggest(ctx context.Context, req *TagSuggestRequest) (*TagSuggestResponse, error)
}

type TagSuggestRequest struct {
    UserID   int32
    MemoID   string  // 可选，编辑已有笔记时
    Content  string  // 笔记内容
    Title    string  // 笔记标题
    MaxTags  int     // 最大返回数量 (默认 5)
    UseLLM   bool    // 是否使用 LLM (默认 true)
}

type TagSuggestResponse struct {
    Tags     []TagSuggestion `json:"tags"`
    Latency  time.Duration   `json:"latency"`
    Sources  []string        `json:"sources"`  // ["statistics", "rules", "llm"]
}

type TagSuggestion struct {
    Name       string  `json:"name"`
    Confidence float64 `json:"confidence"`  // 0.0 - 1.0
    Source     string  `json:"source"`      // "statistics", "rules", "llm"
    Reason     string  `json:"reason,omitempty"`
}
```

### 3.3 Layer 1: 统计优先

```go
// plugin/ai/tags/layer1_statistics.go

type StatisticsLayer struct {
    memoStore MemoStore
    cache     CacheService
}

func (l *StatisticsLayer) Suggest(ctx context.Context, userID int32, content string) []TagSuggestion {
    var suggestions []TagSuggestion
    
    // 1. 用户高频标签 (TOP-5)
    frequentTags := l.getFrequentTags(ctx, userID, 5)
    for _, tag := range frequentTags {
        suggestions = append(suggestions, TagSuggestion{
            Name:       tag.Name,
            Confidence: normalizeFrequency(tag.Count),
            Source:     "statistics",
            Reason:     fmt.Sprintf("使用 %d 次", tag.Count),
        })
    }
    
    // 2. 最近 7 天使用的标签
    recentTags := l.getRecentTags(ctx, userID, 7)
    for _, tag := range recentTags {
        if !containsTag(suggestions, tag.Name) {
            suggestions = append(suggestions, TagSuggestion{
                Name:       tag.Name,
                Confidence: 0.7,
                Source:     "statistics",
                Reason:     "最近使用",
            })
        }
    }
    
    // 3. 相似笔记的标签
    similarTags := l.getSimilarMemoTags(ctx, userID, content, 3)
    for _, tag := range similarTags {
        if !containsTag(suggestions, tag.Name) {
            suggestions = append(suggestions, TagSuggestion{
                Name:       tag.Name,
                Confidence: tag.Similarity * 0.8,
                Source:     "statistics",
                Reason:     "相似笔记使用",
            })
        }
    }
    
    return suggestions
}

func (l *StatisticsLayer) getFrequentTags(ctx context.Context, userID int32, limit int) []TagFrequency {
    // 缓存检查
    cacheKey := fmt.Sprintf("user:%d:frequent_tags", userID)
    if cached, ok := l.cache.Get(cacheKey); ok {
        return cached.([]TagFrequency)
    }
    
    // 查询数据库
    tags, _ := l.memoStore.GetFrequentTags(ctx, userID, limit)
    
    // 缓存 1 小时
    l.cache.Set(cacheKey, tags, time.Hour)
    
    return tags
}

func (l *StatisticsLayer) getSimilarMemoTags(ctx context.Context, userID int32, content string, limit int) []TagWithSimilarity {
    // 使用向量相似度查找相似笔记
    similarMemos, _ := l.memoStore.FindSimilarMemos(ctx, userID, content, limit)
    
    var result []TagWithSimilarity
    for _, memo := range similarMemos {
        for _, tag := range memo.Tags {
            result = append(result, TagWithSimilarity{
                Name:       tag,
                Similarity: memo.Similarity,
            })
        }
    }
    
    return result
}
```

### 3.4 Layer 2: 规则提取

```go
// plugin/ai/tags/layer2_rules.go

type RulesLayer struct {
    techTerms     []string
    emotionTerms  map[string]string
    datePatterns  []*regexp.Regexp
}

func NewRulesLayer() *RulesLayer {
    return &RulesLayer{
        techTerms: []string{
            "React", "Vue", "Angular", "Python", "Go", "Java", "JavaScript",
            "TypeScript", "Docker", "Kubernetes", "AI", "ML", "机器学习",
            "深度学习", "PostgreSQL", "MySQL", "Redis", "API", "REST",
        },
        emotionTerms: map[string]string{
            "灵感":  "#灵感",
            "想法":  "#想法",
            "问题":  "#问题",
            "待办":  "#待办",
            "TODO": "#待办",
            "BUG":  "#问题",
            "记录":  "#记录",
            "学习":  "#学习",
        },
        datePatterns: []*regexp.Regexp{
            regexp.MustCompile(`20\d{2}[-/]?\d{2}`),        // 2024-01
            regexp.MustCompile(`Q[1-4]`),                   // Q1, Q2
            regexp.MustCompile(`(第[一二三四]季度)`),         // 第一季度
        },
    }
}

func (l *RulesLayer) Suggest(ctx context.Context, content string, title string) []TagSuggestion {
    var suggestions []TagSuggestion
    text := title + " " + content
    
    // 1. 专有名词识别
    for _, term := range l.techTerms {
        if strings.Contains(strings.ToLower(text), strings.ToLower(term)) {
            suggestions = append(suggestions, TagSuggestion{
                Name:       term,
                Confidence: 0.9,
                Source:     "rules",
                Reason:     "技术术语",
            })
        }
    }
    
    // 2. 情感词识别
    for keyword, tag := range l.emotionTerms {
        if strings.Contains(text, keyword) {
            tagName := strings.TrimPrefix(tag, "#")
            if !containsTag(suggestions, tagName) {
                suggestions = append(suggestions, TagSuggestion{
                    Name:       tagName,
                    Confidence: 0.85,
                    Source:     "rules",
                    Reason:     "情感/状态词",
                })
            }
        }
    }
    
    // 3. 日期模式提取
    for _, pattern := range l.datePatterns {
        if matches := pattern.FindAllString(text, -1); len(matches) > 0 {
            for _, match := range matches {
                suggestions = append(suggestions, TagSuggestion{
                    Name:       match,
                    Confidence: 0.8,
                    Source:     "rules",
                    Reason:     "时间标记",
                })
            }
        }
    }
    
    return suggestions
}
```

### 3.5 Layer 3: LLM 语义

```go
// plugin/ai/tags/layer3_llm.go

type LLMLayer struct {
    llmClient LLMClient
    timeout   time.Duration
}

func NewLLMLayer(client LLMClient) *LLMLayer {
    return &LLMLayer{
        llmClient: client,
        timeout:   500 * time.Millisecond,
    }
}

const tagSuggestPrompt = `请为以下笔记内容建议 3-5 个合适的标签。

笔记标题: {{.Title}}
笔记内容: {{.Content}}

要求:
1. 标签应该简洁，1-4 个字
2. 优先使用常见分类词（技术、生活、工作、学习等）
3. 可以包含主题词（如具体技术名称）
4. 以 JSON 数组格式返回，如: ["标签1", "标签2", "标签3"]

只返回 JSON 数组，不要其他内容。`

func (l *LLMLayer) Suggest(ctx context.Context, title, content string) []TagSuggestion {
    // 设置超时
    ctx, cancel := context.WithTimeout(ctx, l.timeout)
    defer cancel()
    
    // 准备 prompt
    prompt := renderTemplate(tagSuggestPrompt, map[string]string{
        "Title":   title,
        "Content": truncate(content, 500),
    })
    
    // 调用 LLM
    response, err := l.llmClient.Complete(ctx, prompt)
    if err != nil {
        slog.Warn("llm tag suggestion failed", "error", err)
        return nil  // 降级：返回空，不影响 L1/L2
    }
    
    // 解析响应
    var tags []string
    if err := json.Unmarshal([]byte(response), &tags); err != nil {
        slog.Warn("failed to parse llm response", "response", response)
        return nil
    }
    
    var suggestions []TagSuggestion
    for _, tag := range tags {
        suggestions = append(suggestions, TagSuggestion{
            Name:       tag,
            Confidence: 0.75,
            Source:     "llm",
            Reason:     "AI 建议",
        })
    }
    
    return suggestions
}
```

### 3.6 组合建议器

```go
// plugin/ai/tags/suggester_impl.go

type tagSuggester struct {
    layer1 *StatisticsLayer
    layer2 *RulesLayer
    layer3 *LLMLayer
    cache  CacheService
}

func NewTagSuggester(memoStore MemoStore, llmClient LLMClient, cache CacheService) TagSuggester {
    return &tagSuggester{
        layer1: &StatisticsLayer{memoStore: memoStore, cache: cache},
        layer2: NewRulesLayer(),
        layer3: NewLLMLayer(llmClient),
        cache:  cache,
    }
}

func (s *tagSuggester) Suggest(ctx context.Context, req *TagSuggestRequest) (*TagSuggestResponse, error) {
    start := time.Now()
    var allSuggestions []TagSuggestion
    var sources []string
    
    // Layer 1: 统计优先 (同步，必选)
    l1Suggestions := s.layer1.Suggest(ctx, req.UserID, req.Content)
    allSuggestions = append(allSuggestions, l1Suggestions...)
    if len(l1Suggestions) > 0 {
        sources = append(sources, "statistics")
    }
    
    // Layer 2: 规则提取 (同步，必选)
    l2Suggestions := s.layer2.Suggest(ctx, req.Content, req.Title)
    allSuggestions = append(allSuggestions, l2Suggestions...)
    if len(l2Suggestions) > 0 {
        sources = append(sources, "rules")
    }
    
    // Layer 3: LLM 语义 (可选)
    if req.UseLLM && len(allSuggestions) < req.MaxTags {
        l3Suggestions := s.layer3.Suggest(ctx, req.Title, req.Content)
        allSuggestions = append(allSuggestions, l3Suggestions...)
        if len(l3Suggestions) > 0 {
            sources = append(sources, "llm")
        }
    }
    
    // 合并去重 + 排序
    finalTags := s.mergeAndRank(allSuggestions, req.MaxTags)
    
    return &TagSuggestResponse{
        Tags:    finalTags,
        Latency: time.Since(start),
        Sources: sources,
    }, nil
}

func (s *tagSuggester) mergeAndRank(suggestions []TagSuggestion, limit int) []TagSuggestion {
    // 去重，保留高置信度
    tagMap := make(map[string]TagSuggestion)
    for _, sug := range suggestions {
        existing, ok := tagMap[sug.Name]
        if !ok || sug.Confidence > existing.Confidence {
            tagMap[sug.Name] = sug
        }
    }
    
    // 转为切片并排序
    var result []TagSuggestion
    for _, sug := range tagMap {
        result = append(result, sug)
    }
    
    sort.Slice(result, func(i, j int) bool {
        return result[i].Confidence > result[j].Confidence
    })
    
    // 限制数量
    if len(result) > limit {
        result = result[:limit]
    }
    
    return result
}
```

### 3.7 API 与前端

```go
// server/router/api/v1/tag_suggest_handler.go

// POST /api/v1/tags/suggest
func (h *TagHandler) HandleSuggest(c *gin.Context) {
    var req TagSuggestRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    
    req.UserID = getUserID(c)
    if req.MaxTags == 0 {
        req.MaxTags = 5
    }
    
    response, err := h.suggester.Suggest(c.Request.Context(), &req)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    
    c.JSON(http.StatusOK, response)
}
```

```tsx
// web/src/components/ai/AITagSuggestPopover.tsx

interface AITagSuggestPopoverProps {
  content: string;
  title: string;
  existingTags: string[];
  onTagSelect: (tag: string) => void;
}

export function AITagSuggestPopover({
  content,
  title,
  existingTags,
  onTagSelect,
}: AITagSuggestPopoverProps) {
  const [suggestions, setSuggestions] = useState<TagSuggestion[]>([]);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    if (content.length > 10) {
      fetchSuggestions();
    }
  }, [content, title]);

  const fetchSuggestions = async () => {
    setLoading(true);
    try {
      const response = await api.post('/tags/suggest', {
        content,
        title,
        max_tags: 5,
      });
      setSuggestions(response.data.tags.filter(
        (t: TagSuggestion) => !existingTags.includes(t.name)
      ));
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="flex flex-wrap gap-2">
      {loading && <Spinner size="sm" />}
      {suggestions.map((tag) => (
        <button
          key={tag.name}
          onClick={() => onTagSelect(tag.name)}
          className="rounded-full bg-blue-100 px-3 py-1 text-sm text-blue-700 hover:bg-blue-200"
          title={tag.reason}
        >
          #{tag.name}
          {tag.confidence >= 0.9 && <span className="ml-1">✨</span>}
        </button>
      ))}
    </div>
  );
}
```

---

## 4. 实现路径

### Day 1-2: Layer 1 统计层

- [ ] 高频标签查询
- [ ] 最近标签查询
- [ ] 相似笔记标签
- [ ] 缓存策略

### Day 3: Layer 2 规则层

- [ ] 专有名词词典
- [ ] 情感词匹配
- [ ] 日期模式提取

### Day 4-5: Layer 3 LLM 层

- [ ] Prompt 设计
- [ ] LLM 调用封装
- [ ] 降级策略

### Day 6: 组合与 API

- [ ] 组合建议器
- [ ] 去重排序
- [ ] API Handler

### Day 7: 前端与测试

- [ ] 前端组件
- [ ] 单元测试
- [ ] 端到端测试

---

## 5. 交付物

### 5.1 代码产出

| 文件 | 说明 |
|:---|:---|
| `plugin/ai/tags/suggester.go` | 接口定义 |
| `plugin/ai/tags/layer1_statistics.go` | 统计层 |
| `plugin/ai/tags/layer2_rules.go` | 规则层 |
| `plugin/ai/tags/layer3_llm.go` | LLM 层 |
| `plugin/ai/tags/suggester_impl.go` | 组合实现 |
| `server/router/api/v1/tag_suggest_handler.go` | API |
| `web/src/components/ai/AITagSuggestPopover.tsx` | 前端 |

### 5.2 配置项

```yaml
# configs/ai.yaml
tag_suggester:
  max_tags: 5
  use_llm: true
  llm_timeout: 500ms
  
  layer1:
    frequent_limit: 5
    recent_days: 7
    cache_ttl: 1h
    
  layer2:
    tech_terms_file: "configs/tech_terms.txt"
    
  layer3:
    model: "qwen2.5-7b-instruct"
```

---

## 6. 验收标准

### 6.1 功能验收

| 场景 | 期望结果 |
|:---|:---|
| 包含 "React" | 建议 #React (规则层) |
| 用户常用 #学习 | 建议 #学习 (统计层) |
| 内容关于生活 | 建议 #生活 (LLM层) |
| LLM 超时 | 返回 L1/L2 结果 |

### 6.2 性能验收

- [ ] L1+L2 延迟 < 50ms
- [ ] 含 LLM 延迟 < 500ms
- [ ] 标签采纳率 > 40%

### 6.3 测试用例

```go
func TestTagSuggestion(t *testing.T) {
    suggester := NewTagSuggester(mockStore, mockLLM, mockCache)
    
    req := &TagSuggestRequest{
        UserID:  1,
        Content: "今天学习了 React Hooks 的使用方法",
        Title:   "React 学习笔记",
        MaxTags: 5,
        UseLLM:  false,  // 仅测试 L1/L2
    }
    
    resp, err := suggester.Suggest(context.Background(), req)
    
    assert.NoError(t, err)
    assert.True(t, containsTag(resp.Tags, "React"))
    assert.True(t, resp.Latency < 50*time.Millisecond)
}
```

---

## 7. ROI 分析

| 投入 | 产出 |
|:---|:---|
| 开发: 7 人天 | 标签采纳率提升 350%+ |
| LLM 成本: 可选 | 标签输入成本降低 70% |
| 维护: 词典可配置 | 更好的笔记组织 |

---

## 8. 风险与缓解

| 风险 | 概率 | 影响 | 缓解措施 |
|:---|:---:|:---:|:---|
| LLM 延迟高 | 中 | 低 | 三层降级，L3 可选 |
| 建议不准确 | 中 | 低 | 用户可忽略，不强制 |
| 词典维护 | 低 | 低 | 配置文件，热更新 |

---

## 9. 排期

| 日期 | 任务 | 负责人 |
|:---|:---|:---|
| Sprint 3 Day 1-2 | Layer 1 统计层 | TBD |
| Sprint 3 Day 3 | Layer 2 规则层 | TBD |
| Sprint 3 Day 4-5 | Layer 3 LLM 层 | TBD |
| Sprint 3 Day 6 | 组合与 API | TBD |
| Sprint 3 Day 7 | 前端与测试 | TBD |

---

> **纲领来源**: [00-master-roadmap.md](../../../archived/research_20250202/00-master-roadmap.md)  
> **研究文档**: [memo-roadmap.md](../../../research/memo-roadmap.md)  
> **版本**: v1.0  
> **更新时间**: 2026-01-27
