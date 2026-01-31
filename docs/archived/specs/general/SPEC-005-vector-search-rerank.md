# SPEC-005: 向量检索与 Rerank 实现

**优先级**: P1 (核心功能)
**预计工时**: 8 小时
**依赖**: SPEC-002, SPEC-003

## 目标
实现基于 pgvector 的语义检索,并集成 Reranker 提升检索精准度。

## 实施内容

### 1. 向量存储层
**文件路径**: `server/ai/vector_store.go`

```go
package ai

import (
    "context"
    "github.com/usememos/memos/store"
)

type VectorStore struct {
    store *store.Store
}

// Search 执行向量相似度检索
func (vs *VectorStore) Search(ctx context.Context, queryEmbedding []float32, limit int) ([]*store.Memo, []float32, error) {
    // 调用 store.SearchMemosByVector
    memos, similarities, err := vs.store.SearchMemosByVector(ctx, queryEmbedding, limit)
    if err != nil {
        return nil, nil, err
    }

    return memos, similarities, nil
}
```

### 2. Reranker 接口
**文件路径**: `server/ai/reranker.go`

```go
package ai

type Reranker struct {
    provider *Provider
    model    string // 默认: "BAAI/bge-reranker-v2-m3"
}

// RerankRequest Rerank 请求
type RerankRequest struct {
    Query   string
    Docs    []string
    TopK    int
}

// RerankResponse Rerank 响应
type RerankResponse struct {
    Indices    []int
    Scores     []float32
}

// Rerank 重新排序文档
func (r *Reranker) Rerank(ctx context.Context, req *RerankRequest) (*RerankResponse, error) {
    // 调用 Reranker API
    // 伪代码:
    // POST /rerank
    // {
    //   "model": "bge-reranker-v2-m3",
    //   "query": req.Query,
    //   "documents": req.Docs,
    //   "top_n": req.TopK
    // }

    // 返回排序后的索引和分数
    return &RerankResponse{
        Indices: indices,
        Scores:  scores,
    }, nil
}
```

### 3. RAG Pipeline 核心
**文件路径**: `server/ai/rag.go`

```go
package ai

type RAGPipeline struct {
    provider    *Provider
    vectorStore *VectorStore
    reranker    *Reranker
    embedder    *Embedder
}

// RetrievalConfig 检索配置
type RetrievalConfig struct {
    TopK       int  // 初检返回数量 (默认: 20)
    RerankTopK int  // Rerank 后返回数量 (默认: 5)
    MinScore   float32 // 最低相似度阈值 (默认: 0.5)
}

// Retrieve 检索相关文档
func (rag *RAGPipeline) Retrieve(ctx context.Context, query string, cfg *RetrievalConfig) ([]*store.Memo, error) {
    // 1. Query Embedding
    queryEmb, err := rag.provider.Embedding(ctx, query)
    if err != nil {
        return nil, err
    }

    // 2. Vector Search (初检)
    memos, similarities, err := rag.vectorStore.Search(ctx, queryEmb, cfg.TopK)
    if err != nil {
        return nil, err
    }

    // 3. 过滤低分结果
    filtered := filterByScore(memos, similarities, cfg.MinScore)

    // 4. Rerank
    docs := make([]string, len(filtered))
    for i, memo := range filtered {
        docs[i] = memo.Content
    }

    rerankResult, err := rag.reranker.Rerank(ctx, &RerankRequest{
        Query: query,
        Docs:  docs,
        TopK:  cfg.RerankTopK,
    })
    if err != nil {
        // Rerank 失败,降级返回初检结果
        log.Warn("Rerank failed, fallback to vector search", zap.Error(err))
        return filtered[:min(cfg.RerankTopK, len(filtered))], nil
    }

    // 5. 重新排序
    result := make([]*store.Memo, len(rerankResult.Indices))
    for i, idx := range rerankResult.Indices {
        result[i] = filtered[idx]
    }

    return result, nil
}

// GenerateAnswer 生成答案 (预留接口,SPEC-006 实现)
func (rag *RAGPipeline) GenerateAnswer(ctx context.Context, query string, memos []*store.Memo) (string, error) {
    // ... SPEC-006 实现
    return "", nil
}
```

### 4. 工具函数
**文件路径**: `server/ai/rag.go`

```go
// filterByScore 过滤低相似度结果
func filterByScore(memos []*store.Memo, scores []float32, threshold float32) []*store.Memo {
    var result []*store.Memo
    for i, score := range scores {
        if score >= threshold {
            result = append(result, memos[i])
        }
    }
    return result
}

func min(a, b int) int {
    if a < b {
        return a
    }
    return b
}
```

## 验收标准

### AC-1: 向量检索测试
**测试文件**: `server/ai/vector_store_test.go`

```bash
# 准备测试数据
docker exec -it memos-db psql -U memos -d memos << EOF
INSERT INTO memos (creator_id, content, embedding)
VALUES
  (1, 'Go 是一种编程语言', '[0.1, 0.2, ...]'::vector),
  (1, 'Python 是一种编程语言', '[0.15, 0.25, ...]'::vector),
  (1, '今天天气很好', '[0.9, 0.8, ...]'::vector);
EOF

# 执行检索 (模拟)
curl -X POST http://localhost:8081/api/v1/ai/search/debug \
  -H "Authorization: Bearer <token>" \
  -d '{"query": "编程语言", "top_k": 3}'

# 预期结果
- 返回 3 条记录
- "Go 是一种编程语言" 和 "Python 是一种编程语言" 相似度 > 0.8
- "今天天气很好" 相似度 < 0.3
- 排序按相似度降序
```

### AC-2: Reranker 集成测试
```bash
# 执行
curl -X POST http://localhost:8081/api/v1/ai/rerank/debug \
  -H "Authorization: Bearer <token>" \
  -d '{
    "query": "如何学习 Go 语言",
    "docs": [
      "Go 语言由 Google 开发",
      "今天天气晴朗",
      "Go 语言适合并发编程",
      "Python 是一种脚本语言"
    ],
    "top_k": 2
  }'

# 预期结果
- 返回 2 条记录
- "Go 语言由 Google 开发" 和 "Go 语言适合并发编程" 排名更高
- "今天天气晴朗" 被过滤
- 返回 Rerank 分数
```

### AC-3: RAG Pipeline 端到端测试
**测试文件**: `server/ai/rag_test.go`

```go
func TestRAGPipeline(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test")
    }

    ctx := context.Background()
    rag := setupTestRAG(t)

    // 测试用例
    query := "如何使用 Go 进行并发编程?"
    memos, err := rag.Retrieve(ctx, query, &RetrievalConfig{
        TopK:       20,
        RerankTopK: 5,
        MinScore:   0.5,
    })

    assert.NoError(t, err)
    assert.NotEmpty(t, memos)
    assert.LessOrEqual(t, len(memos), 5)

    // 验证返回的 Memo 内容相关
    // ... 具体验证逻辑
}
```

### AC-4: 性能基准测试
```bash
# 执行 100 次检索,记录平均延迟
time for i in {1..100}; do
  curl -s -X POST http://localhost:8081/api/v1/ai/search/debug \
    -H "Authorization: Bearer <token>" \
    -d '{"query": "测试查询", "top_k": 5}' > /dev/null
done

# 预期结果
- 平均延迟 < 500ms (不含 Rerank)
- P99 延迟 < 2s (含 Rerank)
```

### AC-5: 降级逻辑测试
```bash
# 场景: Reranker API 失败
export MEMOS_AI_RERANKER_URL="http://invalid:9999"

curl -X POST http://localhost:8081/api/v1/ai/search \
  -H "Authorization: Bearer <token>" \
  -d '{"query": "测试", "top_k": 5}'

# 预期结果
- 返回向量检索结果 (未 Rerank)
- 日志中有 "Rerank failed, fallback to vector search" 警告
- 响应时间 < 1s
```

### AC-6: 代码覆盖率
```bash
# 执行
go test -coverprofile=coverage.out ./server/ai/...
go tool cover -html=coverage.out -o coverage.html

# 预期结果
- 代码覆盖率 > 80%
```

## 回滚方案
- Rerank 失败时自动降级到纯向量检索
- 保留向量检索接口,不影响现有功能

## 注意事项
- Reranker API 延迟可能较高,需设置合理超时 (10s)
- 初检 TopK 应大于 RerankTopK (建议 4:1)
- MinScore 需根据实际效果调优
- Rerank 失败不影响主流程,允许降级