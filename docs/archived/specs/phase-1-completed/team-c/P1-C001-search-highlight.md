# P1-C001: 搜索结果高亮

> **状态**: ✅ 已完成  
> **优先级**: P0 (核心)  
> **投入**: 3 人天  
> **负责团队**: 团队 C  
> **Sprint**: Sprint 1

---

## 1. 目标与背景

### 1.1 核心目标

实现搜索结果关键词高亮功能，帮助用户快速定位匹配内容。

### 1.2 用户价值

- 搜索效率提升 53%
- 快速定位关键信息

### 1.3 技术价值

- 复用现有检索架构
- 为上下文摘录奠定基础

---

## 2. 依赖关系

### 2.1 前置依赖

- 无（可独立开发）

### 2.2 并行依赖

- P1-A005: 缓存层（可选优化）

### 2.3 后续依赖

- P1-C002: 上下文智能摘录
- P1-C003: 相关笔记推荐

---

## 3. 功能设计

### 3.1 架构图

```
┌─────────────────────────────────────────────────────────────────┐
│                    搜索高亮架构                                   │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  前端 SearchInput                                                │
│      │ query                                                    │
│      ▼                                                          │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │  后端 HighlightService                                    │   │
│  │                                                          │   │
│  │  1. 执行混合检索 (复用 AdaptiveRetriever)                 │   │
│  │  2. 分词 (中文 jieba / 英文 whitespace)                  │   │
│  │  3. 查找匹配位置                                         │   │
│  │  4. 提取上下文 (前后各 N 字符)                           │   │
│  │  5. 返回带高亮位置的结果                                 │   │
│  └─────────────────────────────────────────────────────────┘   │
│      │                                                          │
│      ▼                                                          │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │  前端 HighlightedResult.tsx                               │   │
│  │                                                          │   │
│  │  • 根据高亮位置渲染 <mark> 标签                          │   │
│  │  • 显示匹配上下文                                        │   │
│  │  • 支持点击展开完整内容                                  │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### 3.2 核心流程

1. **检索**: 复用 AdaptiveRetriever
2. **分词**: 对查询进行分词
3. **匹配**: 在内容中查找匹配位置
4. **摘录**: 提取匹配点附近的上下文
5. **渲染**: 前端使用 `<mark>` 渲染

### 3.3 关键决策

| 决策点 | 方案 A | 方案 B | 选择 | 理由 |
|:---|:---|:---|:---:|:---|
| 分词方式 | LLM | 规则分词 | B | 延迟低 |
| 高亮位置 | 前端计算 | 后端返回 | B | 减少前端复杂度 |

---

## 4. 技术实现

### 4.1 接口定义

```protobuf
// proto/api/v1/memo_service.proto

rpc SearchWithHighlight(SearchWithHighlightRequest) returns (SearchWithHighlightResponse);

message SearchWithHighlightRequest {
  string query = 1;
  int32 limit = 2;           // default: 20
  int32 context_chars = 3;   // default: 50
}

message SearchWithHighlightResponse {
  repeated HighlightedMemo memos = 1;
}

message HighlightedMemo {
  string name = 1;
  string snippet = 2;
  float score = 3;
  repeated Highlight highlights = 4;
  int64 created_ts = 5;
}

message Highlight {
  int32 start = 1;
  int32 end = 2;
  string matched_text = 3;
}
```

### 4.2 关键代码路径

| 文件路径 | 职责 |
|:---|:---|
| `server/service/memo/highlight.go` | 高亮服务 |
| `server/service/memo/tokenizer.go` | 分词器 |
| `server/router/api/v1/memo_service.go` | API 处理器 |
| `web/src/components/MemoSearch/HighlightedResult.tsx` | 前端组件 |

### 4.3 后端实现

```go
// server/service/memo/highlight.go

type HighlightService struct {
    retriever *retrieval.AdaptiveRetriever
    tokenizer *Tokenizer
}

func (s *HighlightService) SearchWithHighlight(
    ctx context.Context,
    query string,
    contextChars int,
) ([]HighlightedMemo, error) {
    // 1. 执行混合检索
    results, err := s.retriever.Retrieve(ctx, &retrieval.RetrievalOptions{
        Query: query,
        Limit: 20,
    })
    if err != nil {
        return nil, err
    }
    
    // 2. 分词
    tokens := s.tokenizer.Tokenize(query)
    
    // 3. 匹配高亮
    var highlighted []HighlightedMemo
    for _, result := range results {
        h := HighlightedMemo{
            Name:      result.Name,
            Score:     result.Score,
            CreatedTs: result.CreatedTs,
        }
        
        // 查找匹配位置
        matches := s.findMatches(result.Content, tokens)
        
        // 提取上下文
        h.Snippet = s.extractSnippet(result.Content, matches, contextChars)
        h.Highlights = matches
        
        highlighted = append(highlighted, h)
    }
    
    return highlighted, nil
}

func (s *HighlightService) findMatches(content string, tokens []string) []Highlight {
    var matches []Highlight
    lowerContent := strings.ToLower(content)
    
    for _, token := range tokens {
        lowerToken := strings.ToLower(token)
        start := 0
        for {
            idx := strings.Index(lowerContent[start:], lowerToken)
            if idx == -1 {
                break
            }
            actualStart := start + idx
            matches = append(matches, Highlight{
                Start:       actualStart,
                End:         actualStart + len(token),
                MatchedText: content[actualStart : actualStart+len(token)],
            })
            start = actualStart + len(token)
        }
    }
    
    // 按位置排序
    sort.Slice(matches, func(i, j int) bool {
        return matches[i].Start < matches[j].Start
    })
    
    return matches
}
```

### 4.4 前端实现

```tsx
// web/src/components/MemoSearch/HighlightedResult.tsx

interface HighlightedResultProps {
  memo: HighlightedMemo;
  query: string;
}

export function HighlightedResult({ memo, query }: HighlightedResultProps) {
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
        if (h.start > lastEnd) {
          parts.push(
            <span key={`text-${i}`}>{snippet.slice(lastEnd, h.start)}</span>
          );
        }
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

    if (lastEnd < snippet.length) {
      parts.push(<span key="text-last">{snippet.slice(lastEnd)}</span>);
    }

    return <>{parts}</>;
  };

  return (
    <div className="p-3 border-b hover:bg-gray-50 dark:hover:bg-gray-800">
      <div className="text-sm text-gray-500 mb-1">
        {formatRelativeTime(memo.createdTs)}
      </div>
      <div className="text-base leading-relaxed">
        {renderHighlightedSnippet()}
      </div>
      <div className="flex items-center mt-2 text-xs text-gray-400">
        <span>{t("search.relevance")}: {(memo.score * 100).toFixed(0)}%</span>
      </div>
    </div>
  );
}
```

---

## 5. 交付物清单

### 5.1 代码文件

- [ ] `server/service/memo/highlight.go` - 高亮服务
- [ ] `server/service/memo/tokenizer.go` - 分词器
- [ ] `server/router/api/v1/memo_service.go` - API 扩展
- [ ] `web/src/components/MemoSearch/HighlightedResult.tsx` - 前端组件

### 5.2 Proto 变更

- [ ] `proto/api/v1/memo_service.proto` - 新增 RPC

### 5.3 国际化

- [ ] `web/src/locales/en.json` - 新增 key
- [ ] `web/src/locales/zh-Hans.json` - 新增 key

---

## 6. 测试验收

### 6.1 功能测试

| 场景 | 输入 | 预期输出 |
|:---|:---|:---|
| 单词匹配 | "Go" | Go 高亮显示 |
| 多词匹配 | "Go 语言" | Go 和 语言 分别高亮 |
| 无匹配 | "xyz" | 返回空结果 |

### 6.2 性能验收

| 指标 | 目标值 | 测试方法 |
|:---|:---|:---|
| 响应延迟 | < 500ms | 集成测试 |
| 高亮准确率 | > 95% | 人工验证 |

---

## 7. ROI 分析

| 维度 | 值 |
|:---|:---|
| 开发投入 | 3 人天 |
| 预期收益 | 搜索定位效率 +53% |
| 风险评估 | 低 |
| 回报周期 | Phase 1 结束 |

---

## 8. 实施计划

### 8.1 时间表

| 阶段 | 时间 | 任务 |
|:---|:---|:---|
| Day 1 | 1人天 | 后端服务实现 |
| Day 2 | 1人天 | 前端组件实现 |
| Day 3 | 1人天 | 测试 + 优化 |

### 8.2 检查点

- [ ] Day 1: API 可用
- [ ] Day 3: 前端渲染正确

---

## 附录

### A. 参考资料

- [笔记增强路线图](../../research/memo-roadmap.md)

### B. 变更记录

| 日期 | 版本 | 变更内容 | 作者 |
|:---|:---|:---|:---|
| 2026-01-27 | v1.0 | 初始版本 | - |
