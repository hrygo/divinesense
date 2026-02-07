# P3-C001: 知识图谱可视化

> **状态**: 🔲 待开发  
> **优先级**: P3 (可选)  
> **投入**: 13 人天 (Sprint 5: 8天 + Sprint 6: 5天)  
> **负责团队**: 团队 C  
> **Sprint**: Sprint 5-6

---

## 1. 目标与背景

### 1.1 核心目标

构建笔记知识图谱，可视化笔记之间的关联关系，支持图谱探索和知识发现。

### 1.2 用户价值

- 发现隐藏的知识关联
- 可视化个人知识体系
- 更好的知识管理体验

---

## 2. 依赖关系

- [x] P1-A005: 通用缓存层
- [x] P1-C003: 相关笔记推荐
- [x] P2-C002: 重复检测系统

---

## 3. 功能设计

### 3.1 架构图

```
┌────────────────────────────────────────────────────────────┐
│                    知识图谱架构                             │
├────────────────────────────────────────────────────────────┤
│                                                            │
│  数据层                                                     │
│  ├─ 节点: 笔记 (Memo)                                      │
│  ├─ 边: 关联关系                                           │
│  │   ├─ 显式链接 (用户创建)                                │
│  │   ├─ 标签共现                                           │
│  │   └─ 语义相似                                           │
│  └─ 属性: 标签、时间、重要性                               │
│                                                            │
│  计算层                                                     │
│  ├─ 图谱构建: 增量更新                                     │
│  ├─ 社区发现: Louvain 算法                                 │
│  └─ 中心性计算: PageRank                                   │
│                                                            │
│  展示层                                                     │
│  ├─ 力导向图 (D3.js / Cytoscape.js)                       │
│  ├─ 交互: 缩放、拖拽、聚焦                                 │
│  └─ 筛选: 按标签、时间、重要性                             │
│                                                            │
└────────────────────────────────────────────────────────────┘
```

### 3.2 图谱数据模型

```go
// plugin/ai/graph/model.go

type GraphNode struct {
    ID         string            `json:"id"`
    Label      string            `json:"label"`      // 笔记标题
    Type       string            `json:"type"`       // memo, tag
    Tags       []string          `json:"tags"`
    Importance float64           `json:"importance"` // PageRank
    Cluster    int               `json:"cluster"`    // 社区ID
    CreatedAt  time.Time         `json:"created_at"`
}

type GraphEdge struct {
    Source   string  `json:"source"`
    Target   string  `json:"target"`
    Type     string  `json:"type"`     // link, tag_co, semantic
    Weight   float64 `json:"weight"`
}

type KnowledgeGraph struct {
    Nodes []GraphNode `json:"nodes"`
    Edges []GraphEdge `json:"edges"`
}
```

### 3.3 图谱构建

```go
// plugin/ai/graph/builder.go

type GraphBuilder struct {
    memoStore    MemoStore
    vectorStore  VectorStore
    cache        CacheService
}

func (b *GraphBuilder) Build(ctx context.Context, userID int32) (*KnowledgeGraph, error) {
    // 缓存检查
    cacheKey := fmt.Sprintf("graph:%d", userID)
    if cached, ok := b.cache.Get(cacheKey); ok {
        return cached.(*KnowledgeGraph), nil
    }
    
    graph := &KnowledgeGraph{}
    
    // 1. 获取所有笔记作为节点
    memos, _ := b.memoStore.GetAllMemos(ctx, userID)
    for _, memo := range memos {
        graph.Nodes = append(graph.Nodes, GraphNode{
            ID:        memo.ID,
            Label:     extractTitle(memo.Content),
            Type:      "memo",
            Tags:      memo.Tags,
            CreatedAt: memo.CreatedAt,
        })
    }
    
    // 2. 构建边
    // 2.1 显式链接
    links, _ := b.memoStore.GetAllLinks(ctx, userID)
    for _, link := range links {
        graph.Edges = append(graph.Edges, GraphEdge{
            Source: link.SourceID,
            Target: link.TargetID,
            Type:   "link",
            Weight: 1.0,
        })
    }
    
    // 2.2 标签共现
    tagEdges := b.buildTagCoOccurrenceEdges(memos)
    graph.Edges = append(graph.Edges, tagEdges...)
    
    // 2.3 语义相似 (Top-3)
    semanticEdges := b.buildSemanticEdges(ctx, memos)
    graph.Edges = append(graph.Edges, semanticEdges...)
    
    // 3. 计算中心性
    b.computeImportance(graph)
    
    // 4. 社区发现
    b.detectCommunities(graph)
    
    // 缓存 10 分钟
    b.cache.Set(cacheKey, graph, 10*time.Minute)
    
    return graph, nil
}
```

### 3.4 前端可视化

```tsx
// web/src/components/graph/KnowledgeGraphView.tsx

import { useEffect, useRef } from 'react';
import * as d3 from 'd3';

export function KnowledgeGraphView({ graph }: { graph: KnowledgeGraph }) {
  const svgRef = useRef<SVGSVGElement>(null);

  useEffect(() => {
    if (!svgRef.current || !graph) return;

    const svg = d3.select(svgRef.current);
    const width = 800;
    const height = 600;

    // 力导向模拟
    const simulation = d3.forceSimulation(graph.nodes)
      .force('link', d3.forceLink(graph.edges).id(d => d.id))
      .force('charge', d3.forceManyBody().strength(-100))
      .force('center', d3.forceCenter(width / 2, height / 2));

    // 绘制边
    const links = svg.selectAll('line')
      .data(graph.edges)
      .join('line')
      .attr('stroke', '#999')
      .attr('stroke-opacity', d => d.weight);

    // 绘制节点
    const nodes = svg.selectAll('circle')
      .data(graph.nodes)
      .join('circle')
      .attr('r', d => 5 + d.importance * 10)
      .attr('fill', d => colorByCluster(d.cluster))
      .call(drag(simulation));

    // 标签
    const labels = svg.selectAll('text')
      .data(graph.nodes)
      .join('text')
      .text(d => d.label)
      .attr('font-size', 10);

    simulation.on('tick', () => {
      links
        .attr('x1', d => d.source.x)
        .attr('y1', d => d.source.y)
        .attr('x2', d => d.target.x)
        .attr('y2', d => d.target.y);

      nodes
        .attr('cx', d => d.x)
        .attr('cy', d => d.y);

      labels
        .attr('x', d => d.x + 8)
        .attr('y', d => d.y + 3);
    });
  }, [graph]);

  return <svg ref={svgRef} width={800} height={600} />;
}
```

---

## 4. 实现路径

| Sprint | Day | 任务 |
|--------|-----|------|
| 5 | 1-2 | 图谱数据模型 |
| 5 | 3-4 | 边构建（链接、标签、语义） |
| 5 | 5-6 | 中心性与社区发现 |
| 5 | 7-8 | API 与缓存 |
| 6 | 1-2 | 前端 D3 可视化 |
| 6 | 3-4 | 交互（缩放、筛选） |
| 6 | 5 | 测试与优化 |

---

## 5. 验收标准

- [ ] 图谱正确显示笔记节点
- [ ] 三种边类型正确渲染
- [ ] 支持缩放、拖拽
- [ ] 节点点击跳转笔记

---

> **版本**: v1.0 | **更新时间**: 2026-01-27
