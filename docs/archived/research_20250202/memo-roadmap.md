# Memos 笔记 AI 增强 - 产品升级路线图

> **版本**: v1.0  
> **日期**: 2026-01-27  
> **定位**: 隐私优先的私人笔记 AI 增强  
> **范围**: 笔记本身能力的 AI 增强（智能助理、日程管理由其他团队负责）

**文档导航**: [主路线图](./00-master-roadmap.md) | [调研报告](./memo-research.md)

---

## 一、路线图总览

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                      产品升级路线图 (3 Phase / 5 Sprint)                      │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  Phase 1                   Phase 2                    Phase 3               │
│  智能检索增强               智能整理组织                知识沉淀回顾            │
│  ──────────────            ──────────────              ──────────────        │
│  Sprint 1-2                Sprint 3-4                  Sprint 5+            │
│                                                                             │
│  ┌──────────────┐          ┌──────────────┐           ┌──────────────┐      │
│  │ 搜索高亮 P0  │          │ 智能标签 P1  │           │ 知识图谱 P2  │      │
│  │ 上下文摘录P1│ →        │ 重复检测 P2  │ →        │ 智能回顾 P3  │      │
│  │ 相关推荐 P1 │          │ 自动分类 P2  │           │ 每周摘要 P3  │      │
│  └──────────────┘          └──────────────┘           └──────────────┘      │
│                                                                             │
│  交付里程碑:                交付里程碑:                 交付里程碑:           │
│  M1: 高亮搜索上线          M3: 标签建议完善            M5: 图谱可视化         │
│  M2: 相关推荐上线          M4: 重复检测上线            M6: 回顾系统           │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 1.1 核心价值主张

| 阶段 | 用户痛点 | 解决方案 | 核心价值 |
|:---|:---|:---|:---|
| **Phase 1** | 搜索结果难定位 | 高亮+摘录+推荐 | **找得快** |
| **Phase 2** | 整理费时费力 | 智能标签+去重 | **整得省** |
| **Phase 3** | 笔记沉睡无用 | 图谱+回顾 | **用得活** |

---

## 二、Phase 1: 智能检索增强

### 2.1 里程碑规划

| 里程碑 | 功能 | Sprint | 优先级 |
|:---|:---|:---:|:---:|
| **M1** | 搜索结果高亮 + 上下文摘录 | Sprint 1 | P0 |
| **M2** | 相关笔记推荐 | Sprint 2 | P1 |

### 2.2 技术方案: 搜索结果高亮 (M1)

#### 2.2.1 架构设计

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                          搜索高亮架构                                         │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  前端                         后端                          存储            │
│  ────                         ────                          ────            │
│                                                                             │
│  SearchInput                  ┌─────────────────┐                           │
│      │                        │ memo_ai_service │                           │
│      │ query                  │ .go             │                           │
│      ▼                        └────────┬────────┘                           │
│  ┌──────────────┐                     │                                    │
│  │SearchService│◄────────────────────┤                                    │
│  │  (RPC)      │   SearchWithHighlight                                     │
│  └──────┬───────┘                     │                                    │
│         │                             ▼                                    │
│         │                    ┌─────────────────┐                           │
│         │                    │ highlight.go    │                           │
│         │                    │ ├─ tokenize()   │                           │
│         │                    │ ├─ match()      │──────▶ PostgreSQL         │
│         │                    │ └─ extract()    │       (memo + embedding)  │
│         │                    └─────────────────┘                           │
│         ▼                                                                   │
│  ┌──────────────┐                                                          │
│  │HighlightedResult.tsx                                                    │
│  │ └─ <mark> 渲染                                                          │
│  └──────────────┘                                                          │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

#### 2.2.2 核心算法

```go
// server/service/memo/highlight.go

type HighlightService struct {
    retriever *retrieval.AdaptiveRetriever
}

// SearchWithHighlight 返回带高亮的搜索结果
func (s *HighlightService) SearchWithHighlight(
    ctx context.Context, 
    query string, 
    contextChars int,
) ([]HighlightedMemo, error) {
    // 1. 执行混合检索 (复用现有 RAG)
    results, err := s.retriever.Retrieve(ctx, query)
    if err != nil {
        return nil, err
    }
    
    // 2. 分词 (中文 jieba / 英文 whitespace)
    queryTokens := s.tokenize(query)
    
    // 3. 匹配高亮
    var highlighted []HighlightedMemo
    for _, result := range results {
        h := HighlightedMemo{
            Name:  result.Name,
            Score: result.Score,
        }
        
        // 查找匹配位置
        matches := s.findMatches(result.Content, queryTokens)
        
        // 提取上下文 (前后各 contextChars 字符)
        h.Snippet = s.extractSnippet(result.Content, matches, contextChars)
        h.Highlights = matches
        
        highlighted = append(highlighted, h)
    }
    
    return highlighted, nil
}

// extractSnippet 智能摘录核心逻辑
func (s *HighlightService) extractSnippet(
    content string, 
    matches []Match, 
    contextChars int,
) string {
    if len(matches) == 0 {
        // 无匹配，返回开头
        return truncate(content, contextChars*2)
    }
    
    // 取第一个匹配点为中心
    center := matches[0].Start
    start := max(0, center-contextChars)
    end := min(len(content), center+contextChars)
    
    snippet := content[start:end]
    
    // 添加省略号
    if start > 0 {
        snippet = "..." + snippet
    }
    if end < len(content) {
        snippet = snippet + "..."
    }
    
    return snippet
}
```

#### 2.2.3 前端实现

```tsx
// web/src/components/MemoSearch/HighlightedResult.tsx

interface HighlightedResultProps {
  memo: HighlightedMemo;
  query: string;
}

export function HighlightedResult({ memo, query }: HighlightedResultProps) {
  // 将高亮位置转换为 React 元素
  const renderHighlightedSnippet = () => {
    const { snippet, highlights } = memo;
    if (!highlights?.length) {
      return <span>{snippet}</span>;
    }

    const parts: React.ReactNode[] = [];
    let lastEnd = 0;

    highlights
      .sort((a, b) => a.start - b.start)
      .forEach((h, i) => {
        // 匹配前的文本
        if (h.start > lastEnd) {
          parts.push(
            <span key={`text-${i}`}>{snippet.slice(lastEnd, h.start)}</span>
          );
        }
        // 高亮文本
        parts.push(
          <mark
            key={`mark-${i}`}
            className="bg-yellow-200 dark:bg-yellow-700 rounded px-0.5"
          >
            {h.matchedText}
          </mark>
        );
        lastEnd = h.end;
      });

    // 剩余文本
    if (lastEnd < snippet.length) {
      parts.push(<span key="text-last">{snippet.slice(lastEnd)}</span>);
    }

    return <>{parts}</>;
  };

  return (
    <div className="p-3 border-b hover:bg-gray-50 dark:hover:bg-gray-800">
      <div className="text-sm text-gray-500 mb-1">
        {formatRelativeTime(memo.createdAt)}
      </div>
      <div className="text-base leading-relaxed">
        {renderHighlightedSnippet()}
      </div>
      <div className="flex items-center mt-2 text-xs text-gray-400">
        <span>相关度: {(memo.score * 100).toFixed(0)}%</span>
        <span className="mx-2">|</span>
        <Link to={`/m/${memo.name}`} className="text-blue-500">
          {t("search.view-full")}
        </Link>
      </div>
    </div>
  );
}
```

### 2.3 技术方案: 相关笔记推荐 (M2)

#### 2.3.1 架构设计

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         相关推荐架构                                         │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  触发场景:                                                                   │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐                       │
│  │ 编辑器输入   │  │ 笔记详情页   │  │ 保存笔记后   │                       │
│  │ (防抖 500ms) │  │ (侧边栏)     │  │ (后台推送)   │                       │
│  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘                       │
│         │                 │                 │                               │
│         └────────────────┬┴─────────────────┘                               │
│                          │                                                   │
│                          ▼                                                   │
│                 ┌─────────────────┐                                         │
│                 │  RelatedService │                                         │
│                 │  .GetRelated()  │                                         │
│                 └────────┬────────┘                                         │
│                          │                                                   │
│           ┌──────────────┼──────────────┐                                   │
│           ▼              ▼              ▼                                   │
│  ┌──────────────┐ ┌──────────────┐ ┌──────────────┐                         │
│  │ 向量相似度   │ │ 标签共现     │ │ 时间邻近     │                         │
│  │ (权重 0.6)   │ │ (权重 0.3)   │ │ (权重 0.1)   │                         │
│  └──────────────┘ └──────────────┘ └──────────────┘                         │
│                          │                                                   │
│                          ▼                                                   │
│                 ┌─────────────────┐                                         │
│                 │  Score Fusion   │                                         │
│                 │  + Dedup        │                                         │
│                 └────────┬────────┘                                         │
│                          │                                                   │
│                          ▼                                                   │
│                 ┌─────────────────┐                                         │
│                 │  Top-5 Related  │                                         │
│                 └─────────────────┘                                         │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

#### 2.3.2 核心实现

```go
// server/service/memo/related.go

type RelatedService struct {
    embeddingStore  *store.EmbeddingStore
    memoStore       *store.MemoStore
}

type RelatedMemo struct {
    Name       string  `json:"name"`
    Title      string  `json:"title"`      // 首行或截取
    Similarity float32 `json:"similarity"`
    SharedTags []string `json:"shared_tags"`
}

func (s *RelatedService) GetRelatedMemos(
    ctx context.Context,
    memoUID string,
    limit int,
) ([]RelatedMemo, error) {
    // 1. 获取当前笔记的向量
    currentEmb, err := s.embeddingStore.GetByMemoUID(ctx, memoUID)
    if err != nil {
        return nil, err
    }
    
    // 2. 向量相似度检索 (pgvector cosine similarity)
    vectorResults, err := s.embeddingStore.FindSimilar(ctx, currentEmb.Vector, limit*2)
    if err != nil {
        return nil, err
    }
    
    // 3. 获取当前笔记标签
    currentMemo, _ := s.memoStore.GetByUID(ctx, memoUID)
    currentTags := extractTags(currentMemo.Payload)
    
    // 4. 计算综合得分
    var results []RelatedMemo
    for _, v := range vectorResults {
        if v.MemoUID == memoUID {
            continue // 排除自身
        }
        
        memo, _ := s.memoStore.GetByUID(ctx, v.MemoUID)
        memoTags := extractTags(memo.Payload)
        
        // 标签共现得分
        sharedTags := intersect(currentTags, memoTags)
        tagScore := float32(len(sharedTags)) / float32(max(len(currentTags), 1))
        
        // 时间邻近得分 (7天内得分高)
        timeDiff := abs(currentMemo.CreatedTs - memo.CreatedTs)
        timeScore := max(0, 1.0 - float32(timeDiff)/(7*24*3600))
        
        // 加权融合
        finalScore := 0.6*v.Similarity + 0.3*tagScore + 0.1*timeScore
        
        results = append(results, RelatedMemo{
            Name:       memo.UID,
            Title:      extractTitle(memo.Content),
            Similarity: finalScore,
            SharedTags: sharedTags,
        })
    }
    
    // 5. 排序取 Top-N
    sort.Slice(results, func(i, j int) bool {
        return results[i].Similarity > results[j].Similarity
    })
    
    if len(results) > limit {
        results = results[:limit]
    }
    
    return results, nil
}
```

### 2.4 Phase 1 ROI 分析

#### 2.4.1 投入估算

| 资源 | M1 (搜索高亮) | M2 (相关推荐) | 合计 |
|:---|---:|---:|---:|
| **后端开发** | 3 人天 | 4 人天 | 7 人天 |
| **前端开发** | 2 人天 | 3 人天 | 5 人天 |
| **测试** | 1 人天 | 1 人天 | 2 人天 |
| **总计** | 6 人天 | 8 人天 | **14 人天** |

#### 2.4.2 收益预估

| 指标 | 当前基线 | 目标值 | 提升幅度 |
|:---|---:|---:|---:|
| 搜索点击次数/次成功定位 | 3.2 次 | 1.5 次 | **-53%** |
| 搜索平均耗时 | 45s | 20s | **-56%** |
| 笔记关联发现率 | 5% | 20% | **+300%** |
| 重复记录率 | 12% | 8% | **-33%** |

#### 2.4.3 ROI 计算

```
投入成本:
- 开发人力: 14 人天 × ¥2000/天 = ¥28,000
- 测试/修复: ¥5,000
- 总投入: ¥33,000

直接收益 (以 1000 活跃用户计):
- 搜索效率提升: 1000 × 25s × 10次/天 × 30天 = 2,083 人时/月
- 重复记录减少: 1000 × 4条/月 × 3分钟/条 = 200 人时/月
- 总节省: ~2,283 人时/月

间接收益:
- 用户粘性提升: +15% DAU
- 口碑传播: +20% 自然增长

投资回报周期: < 1 个月
```

---

## 三、Phase 2: 智能整理组织

### 3.1 里程碑规划

| 里程碑 | 功能 | Sprint | 优先级 |
|:---|:---|:---:|:---:|
| **M3** | 智能标签建议完善 | Sprint 3 | P1 |
| **M4** | 重复/相似笔记检测 | Sprint 4 | P2 |

### 3.2 技术方案: 智能标签建议 (M3)

#### 3.2.1 三层建议策略

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         智能标签三层策略                                      │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  输入: 笔记内容                                                              │
│         │                                                                   │
│         ▼                                                                   │
│  ┌──────────────────────────────────────────────────────────────┐           │
│  │ L1: 统计优先 (0ms, 本地)                                       │           │
│  │ ────────────────────────                                      │           │
│  │ • 用户历史高频标签 TOP-5                                        │           │
│  │ • 最近 7 天使用的标签                                          │           │
│  │ • 相似笔记的共用标签                                           │           │
│  │ • 实现: 内存缓存 + 简单统计                                    │           │
│  └──────────────────────────────────────────────────────────────┘           │
│         │ 输出: 候选标签集 A                                                 │
│         ▼                                                                   │
│  ┌──────────────────────────────────────────────────────────────┐           │
│  │ L2: 规则提取 (10ms, 本地)                                      │           │
│  │ ───────────────────────                                       │           │
│  │ • 专有名词识别 (技术栈: React/Go/Python...)                    │           │
│  │ • 日期时间模式 (#2026-01, #Q1-周报)                            │           │
│  │ • 情感/类型词 (#灵感, #问题, #待办, #读书)                     │           │
│  │ • 实现: 正则匹配 + 词典                                        │           │
│  └──────────────────────────────────────────────────────────────┘           │
│         │ 输出: 候选标签集 B                                                 │
│         ▼                                                                   │
│  ┌──────────────────────────────────────────────────────────────┐           │
│  │ L3: LLM 语义 (300ms, 可选)                                     │           │
│  │ ──────────────────────                                        │           │
│  │ • 主题分类 (#技术, #生活, #工作, #学习)                         │           │
│  │ • 新标签发现 (从内容推断未使用过的标签)                         │           │
│  │ • 实现: 调用 AI 服务 + 结果缓存                                 │           │
│  │ • 降级: 网络异常时跳过                                         │           │
│  └──────────────────────────────────────────────────────────────┘           │
│         │ 输出: 候选标签集 C                                                 │
│         ▼                                                                   │
│  ┌──────────────────────────────────────────────────────────────┐           │
│  │ 融合 & 去重 & 排序                                             │           │
│  │ Score = L1_weight × A + L2_weight × B + L3_weight × C         │           │
│  │ 输出: Top-5 建议标签                                           │           │
│  └──────────────────────────────────────────────────────────────┘           │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

#### 3.2.2 核心实现

```go
// server/service/memo/tags.go

type TagSuggestionService struct {
    memoStore    *store.MemoStore
    aiClient     *ai.Client  // 可选
    techDict     map[string]bool  // 技术词典
}

type SuggestedTag struct {
    Tag    string  `json:"tag"`
    Score  float32 `json:"score"`
    Source string  `json:"source"`  // "history" | "rule" | "ai"
}

func (s *TagSuggestionService) SuggestTags(
    ctx context.Context,
    userID int32,
    content string,
) ([]SuggestedTag, error) {
    var candidates []SuggestedTag
    
    // L1: 统计优先
    historyTags := s.getHistoryTags(ctx, userID)
    recentTags := s.getRecentTags(ctx, userID, 7*24*time.Hour)
    for _, tag := range mergeTags(historyTags, recentTags) {
        candidates = append(candidates, SuggestedTag{
            Tag: tag.Name, Score: tag.Frequency * 0.4, Source: "history",
        })
    }
    
    // L2: 规则提取
    ruleTags := s.extractByRules(content)
    for _, tag := range ruleTags {
        candidates = append(candidates, SuggestedTag{
            Tag: tag, Score: 0.3, Source: "rule",
        })
    }
    
    // L3: LLM 语义 (可选，带超时)
    if s.aiClient != nil {
        ctx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
        defer cancel()
        
        aiTags, err := s.aiClient.SuggestTags(ctx, content)
        if err == nil {
            for _, tag := range aiTags {
                candidates = append(candidates, SuggestedTag{
                    Tag: tag, Score: 0.5, Source: "ai",
                })
            }
        }
        // 超时或错误时静默降级
    }
    
    // 去重 & 排序
    return s.dedupeAndRank(candidates, 5), nil
}

// extractByRules 规则提取
func (s *TagSuggestionService) extractByRules(content string) []string {
    var tags []string
    
    // 技术词典匹配
    words := strings.Fields(strings.ToLower(content))
    for _, word := range words {
        if s.techDict[word] {
            tags = append(tags, word)
        }
    }
    
    // 日期模式
    dateRe := regexp.MustCompile(`(\d{4}-\d{2}|\d{4}年\d{1,2}月)`)
    if matches := dateRe.FindAllString(content, -1); len(matches) > 0 {
        tags = append(tags, matches[0])
    }
    
    // 情感词
    emotionKeywords := map[string]string{
        "灵感": "灵感", "问题": "问题", "待办": "待办",
        "TODO": "待办", "FIXME": "问题", "idea": "灵感",
    }
    for keyword, tag := range emotionKeywords {
        if strings.Contains(content, keyword) {
            tags = append(tags, tag)
        }
    }
    
    return tags
}
```

### 3.3 技术方案: 重复检测 (M4)

#### 3.3.1 检测流程

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         重复检测流程                                         │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  触发时机                         检测策略                                   │
│  ─────────                        ─────────                                 │
│                                                                             │
│  ┌──────────────┐                ┌──────────────────────────────┐           │
│  │ 创建新笔记时 │───────────────▶│ 实时检测 (向量相似度 > 0.9)  │           │
│  └──────────────┘                └──────────────────────────────┘           │
│         │                                     │                             │
│         │                                     ▼                             │
│         │                        ┌──────────────────────────────┐           │
│         │                        │ 相似度分级:                   │           │
│         │                        │ • > 0.95: 高度重复，强提示   │           │
│         │                        │ • 0.90-0.95: 相似，弱提示    │           │
│         │                        │ • < 0.90: 忽略               │           │
│         │                        └──────────────────────────────┘           │
│         │                                     │                             │
│         │                                     ▼                             │
│  ┌──────────────┐                ┌──────────────────────────────┐           │
│  │ 每日后台任务 │───────────────▶│ 批量扫描 (增量检测近7天笔记) │           │
│  └──────────────┘                └──────────────────────────────┘           │
│                                               │                             │
│                                               ▼                             │
│                                  ┌──────────────────────────────┐           │
│                                  │ 生成去重报告 (标记候选组)    │           │
│                                  └──────────────────────────────┘           │
│                                                                             │
│  用户操作选项:                                                               │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐                       │
│  │   合并笔记   │  │  建立关联    │  │    忽略      │                       │
│  │ (选择保留版) │  │ (双向链接)   │  │ (不再提示)   │                       │
│  └──────────────┘  └──────────────┘  └──────────────┘                       │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

#### 3.3.2 数据库设计

```sql
-- 新增表: 重复检测结果
CREATE TABLE memo_duplicate (
    id SERIAL PRIMARY KEY,
    memo_a_uid VARCHAR(36) NOT NULL,
    memo_b_uid VARCHAR(36) NOT NULL,
    similarity FLOAT NOT NULL,
    status VARCHAR(20) DEFAULT 'pending',  -- pending | merged | linked | ignored
    created_at TIMESTAMP DEFAULT NOW(),
    handled_at TIMESTAMP,
    
    UNIQUE(memo_a_uid, memo_b_uid)
);

CREATE INDEX idx_memo_duplicate_status ON memo_duplicate(status);
CREATE INDEX idx_memo_duplicate_memo_a ON memo_duplicate(memo_a_uid);
```

### 3.4 Phase 2 ROI 分析

#### 3.4.1 投入估算

| 资源 | M3 (智能标签) | M4 (重复检测) | 合计 |
|:---|---:|---:|---:|
| **后端开发** | 4 人天 | 5 人天 | 9 人天 |
| **前端开发** | 3 人天 | 4 人天 | 7 人天 |
| **测试** | 1 人天 | 2 人天 | 3 人天 |
| **总计** | 8 人天 | 11 人天 | **19 人天** |

#### 3.4.2 收益预估

| 指标 | 当前基线 | 目标值 | 提升幅度 |
|:---|---:|---:|---:|
| 标签建议采纳率 | 10% | 45% | **+350%** |
| 无标签笔记占比 | 40% | 15% | **-63%** |
| 重复笔记识别率 | 0% | 80% | **∞** |
| 整理耗时/笔记 | 30s | 10s | **-67%** |

#### 3.4.3 ROI 计算

```
投入成本:
- 开发人力: 19 人天 × ¥2000/天 = ¥38,000
- 测试/修复: ¥7,000
- 总投入: ¥45,000

直接收益 (以 1000 活跃用户计):
- 标签整理节省: 1000 × 20s × 5条/天 × 30天 = 833 人时/月
- 去重合并节省: 1000 × 2条/月 × 5分钟/条 = 167 人时/月
- 总节省: ~1,000 人时/月

间接收益:
- 笔记可检索性提升: +25% 标签覆盖率
- 知识库质量提升: -30% 冗余数据

投资回报周期: 1-2 个月
```

---

## 四、Phase 3: 知识沉淀回顾

### 4.1 里程碑规划

| 里程碑 | 功能 | Sprint | 优先级 |
|:---|:---|:---:|:---:|
| **M5** | 知识图谱可视化 | Sprint 5 | P2 |
| **M6** | 智能回顾 + 摘要 | Sprint 6 | P3 |

### 4.2 技术方案: 知识图谱 (M5)

#### 4.2.1 技术选型

| 方案 | 优点 | 缺点 | 推荐度 |
|:---|:---|:---|:---:|
| **D3.js** | 高度可定制，社区活跃 | 学习曲线陡 | ⭐⭐⭐⭐ |
| **vis.js** | 开箱即用，交互丰富 | 定制性一般 | ⭐⭐⭐⭐⭐ |
| **Cytoscape.js** | 专业图论库 | 过于复杂 | ⭐⭐⭐ |

**推荐**: vis.js (Network 模块) - 平衡易用性和功能性

#### 4.2.2 数据结构

```typescript
// 图谱节点
interface GraphNode {
  id: string;           // memo UID
  label: string;        // 首行/标题
  type: 'memo' | 'tag';
  size: number;         // 节点大小 = 关联数
  color: string;        // 按标签分类着色
  createdAt: number;
}

// 图谱边
interface GraphEdge {
  from: string;
  to: string;
  type: 'tag' | 'semantic' | 'manual';  // 标签共现 / 语义相似 / 手动关联
  strength: number;     // 边粗细 = 相似度
  dashes: boolean;      // 语义边用虚线
}

// API 响应
interface KnowledgeGraphResponse {
  nodes: GraphNode[];
  edges: GraphEdge[];
  clusters: {           // 聚类信息
    id: string;
    label: string;
    nodeIds: string[];
  }[];
}
```

#### 4.2.3 后端实现

```go
// server/service/memo/graph.go

func (s *GraphService) GetKnowledgeGraph(
    ctx context.Context,
    userID int32,
    options GraphOptions,
) (*KnowledgeGraphResponse, error) {
    // 1. 获取所有笔记
    memos, err := s.memoStore.ListByUser(ctx, userID)
    if err != nil {
        return nil, err
    }
    
    // 2. 构建节点
    nodes := make([]GraphNode, len(memos))
    for i, m := range memos {
        nodes[i] = GraphNode{
            ID:        m.UID,
            Label:     extractTitle(m.Content),
            Type:      "memo",
            CreatedAt: m.CreatedTs,
        }
    }
    
    // 3. 构建标签节点
    tagNodes := s.extractTagNodes(memos)
    nodes = append(nodes, tagNodes...)
    
    // 4. 构建边 (标签共现)
    tagEdges := s.buildTagEdges(memos)
    
    // 5. 构建边 (语义相似, 阈值 > 0.7)
    semanticEdges, err := s.buildSemanticEdges(ctx, memos, 0.7)
    if err != nil {
        return nil, err
    }
    
    // 6. 计算聚类
    clusters := s.computeClusters(nodes, append(tagEdges, semanticEdges...))
    
    return &KnowledgeGraphResponse{
        Nodes:    nodes,
        Edges:    append(tagEdges, semanticEdges...),
        Clusters: clusters,
    }, nil
}
```

### 4.3 技术方案: 智能回顾 (M6)

#### 4.3.1 间隔重复算法

```go
// server/service/memo/review.go

// 间隔重复算法 (简化版 SM-2)
type ReviewScheduler struct {
    intervals []int  // 天数: [1, 3, 7, 14, 30, 90]
}

func (r *ReviewScheduler) GetNextReviewDate(
    memo *Memo,
    reviewCount int,
    lastReviewAt time.Time,
) time.Time {
    if reviewCount >= len(r.intervals) {
        reviewCount = len(r.intervals) - 1
    }
    
    days := r.intervals[reviewCount]
    return lastReviewAt.Add(time.Duration(days) * 24 * time.Hour)
}

func (r *ReviewScheduler) GetDueReviews(
    ctx context.Context,
    userID int32,
    limit int,
) ([]ReviewSuggestion, error) {
    now := time.Now()
    
    // 查询到期的笔记
    rows, err := r.db.Query(ctx, `
        SELECT m.uid, m.content, r.review_count, r.next_review_at
        FROM memo m
        LEFT JOIN memo_review r ON m.uid = r.memo_uid
        WHERE m.creator_id = $1
          AND (r.next_review_at IS NULL OR r.next_review_at <= $2)
        ORDER BY COALESCE(r.next_review_at, m.created_ts) ASC
        LIMIT $3
    `, userID, now, limit)
    
    // ... 构建结果
}
```

#### 4.3.2 数据库设计

```sql
-- 回顾记录表
CREATE TABLE memo_review (
    memo_uid VARCHAR(36) PRIMARY KEY,
    review_count INT DEFAULT 0,
    last_review_at TIMESTAMP,
    next_review_at TIMESTAMP,
    is_important BOOLEAN DEFAULT FALSE,
    
    FOREIGN KEY (memo_uid) REFERENCES memo(uid) ON DELETE CASCADE
);

CREATE INDEX idx_memo_review_next ON memo_review(next_review_at);
```

### 4.4 Phase 3 ROI 分析

#### 4.4.1 投入估算

| 资源 | M5 (知识图谱) | M6 (智能回顾) | 合计 |
|:---|---:|---:|---:|
| **后端开发** | 5 人天 | 4 人天 | 9 人天 |
| **前端开发** | 8 人天 | 4 人天 | 12 人天 |
| **测试** | 2 人天 | 2 人天 | 4 人天 |
| **总计** | 15 人天 | 10 人天 | **25 人天** |

#### 4.4.2 收益预估

| 指标 | 当前基线 | 目标值 | 提升幅度 |
|:---|---:|---:|---:|
| 知识图谱周活跃率 | 0% | 25% | **∞** |
| 笔记回顾完成率 | 5% | 40% | **+700%** |
| 旧笔记再利用率 | 8% | 25% | **+213%** |

#### 4.4.3 ROI 计算

```
投入成本:
- 开发人力: 25 人天 × ¥2000/天 = ¥50,000
- 测试/修复: ¥10,000
- 总投入: ¥60,000

直接收益:
- 知识复用价值: 难以量化
- 学习效率提升: 间隔重复 ~30% 记忆留存率提升

间接收益:
- 差异化竞争力: 私人笔记软件中独有
- 用户粘性: +30% 月留存

投资回报周期: 3-6 个月 (长期价值型投入)
```

---

## 五、总体 ROI 汇总

### 5.1 投入产出对比

| 阶段 | 投入 (人天) | 投入 (成本) | 直接收益 | ROI | 回报周期 |
|:---|---:|---:|:---|:---:|:---|
| **Phase 1** | 14 | ¥33,000 | 2,283 人时/月 | **高** | < 1 月 |
| **Phase 2** | 19 | ¥45,000 | 1,000 人时/月 | **中高** | 1-2 月 |
| **Phase 3** | 25 | ¥60,000 | 长期价值 | **中** | 3-6 月 |
| **合计** | **58** | **¥138,000** | - | - | - |

### 5.2 推荐投资策略

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         投资策略建议                                         │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  短期 (Phase 1):                                                            │
│  ┌─────────────────────────────────────────────────────────────────────┐    │
│  │ 投资信心: ⭐⭐⭐⭐⭐ (确定性高，ROI 最佳)                              │    │
│  │ 建议: 立即启动，优先保证质量                                         │    │
│  └─────────────────────────────────────────────────────────────────────┘    │
│                                                                             │
│  中期 (Phase 2):                                                            │
│  ┌─────────────────────────────────────────────────────────────────────┐    │
│  │ 投资信心: ⭐⭐⭐⭐ (价值明确，需验证采纳率)                            │    │
│  │ 建议: Phase 1 上线后观察数据，再决定投入力度                          │    │
│  └─────────────────────────────────────────────────────────────────────┘    │
│                                                                             │
│  长期 (Phase 3):                                                            │
│  ┌─────────────────────────────────────────────────────────────────────┐    │
│  │ 投资信心: ⭐⭐⭐ (差异化价值，但收益周期长)                            │    │
│  │ 建议: 作为后续迭代方向，可拆分为多个小版本逐步交付                     │    │
│  └─────────────────────────────────────────────────────────────────────┘    │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## 六、风险评估与缓解

### 6.1 技术风险

| 风险 | 概率 | 影响 | 缓解措施 |
|:---|:---:|:---:|:---|
| 中文分词效果差 | 中 | 中 | 采用 jieba + 词典补充 |
| 向量检索性能瓶颈 | 低 | 高 | pgvector HNSW 索引 + 分页 |
| LLM 调用延迟/失败 | 中 | 中 | 设置超时 + 静默降级 |
| 图谱渲染卡顿 | 中 | 中 | 限制节点数 + 虚拟化 |

### 6.2 产品风险

| 风险 | 概率 | 影响 | 缓解措施 |
|:---|:---:|:---:|:---|
| 用户不习惯新交互 | 中 | 中 | 渐进式引导 + 可关闭 |
| 标签建议干扰创作 | 低 | 中 | 防抖 + 非侵入式 UI |
| 隐私担忧 (LLM 调用) | 高 | 高 | 本地模型 fallback + 明确说明 |

---

## 七、附录: API 接口规范

### 7.1 Phase 1 接口

```protobuf
// SearchWithHighlight
rpc SearchWithHighlight(SearchWithHighlightRequest) returns (SearchWithHighlightResponse);

message SearchWithHighlightRequest {
  string query = 1;
  int32 limit = 2;           // default: 20
  int32 context_chars = 3;   // default: 50
}

// GetRelatedMemos
rpc GetRelatedMemos(GetRelatedMemosRequest) returns (GetRelatedMemosResponse);

message GetRelatedMemosRequest {
  string memo_name = 1;
  int32 limit = 2;           // default: 5
}
```

### 7.2 Phase 2 接口

```protobuf
// SuggestTags
rpc SuggestTags(SuggestTagsRequest) returns (SuggestTagsResponse);

message SuggestTagsRequest {
  string content = 1;
  bool include_ai = 2;       // 是否包含 LLM 建议
}

// DetectDuplicates
rpc DetectDuplicates(DetectDuplicatesRequest) returns (DetectDuplicatesResponse);

message DetectDuplicatesRequest {
  string memo_name = 1;      // 当前笔记
  float threshold = 2;       // 相似度阈值, default: 0.9
}
```

### 7.3 Phase 3 接口

```protobuf
// GetKnowledgeGraph
rpc GetKnowledgeGraph(GetKnowledgeGraphRequest) returns (GetKnowledgeGraphResponse);

message GetKnowledgeGraphRequest {
  string filter_tag = 1;     // 可选，按标签过滤
  int32 limit = 2;           // 最大节点数, default: 100
}

// GetReviewSuggestions
rpc GetReviewSuggestions(GetReviewSuggestionsRequest) returns (GetReviewSuggestionsResponse);

message GetReviewSuggestionsRequest {
  int32 limit = 1;           // default: 5
}
```

---

> **文档维护**: 本路线图将随产品迭代持续更新
