# SPEC-004: Memo Embedding 生成与入库

**优先级**: P1 (核心功能)
**预计工时**: 6 小时
**依赖**: SPEC-002, SPEC-003

## 目标
实现 Memo 创建/更新时自动生成 Embedding 并入库的完整流程。

## 实施内容

### 1. 文档切片逻辑
**文件路径**: `server/ai/chunker.go`

```go
package ai

// ChunkSize 每个切片的最大字符数
const ChunkSize = 500

// ChunkOverlap 切片重叠字符数
const ChunkOverlap = 50

// ChunkDocument 将长文档切分为多个片段
func ChunkDocument(content string) []string {
    // 1. 按段落分割
    // 2. 合并短段落
    // 3. 超长段落强制切分
    // 4. 添加重叠上下文
    // ... 实现细节
}
```

### 2. Embedding 生成与存储
**文件路径**: `server/ai/embedder.go`

```go
package ai

type Embedder struct {
    provider *Provider
    store    *store.Store
}

// EmbedMemo 为单个 Memo 生成 Embedding
func (e *Embedder) EmbedMemo(ctx context.Context, memo *store.Memo) error {
    // 1. 切片文档
    chunks := ChunkDocument(memo.Content)

    // 2. 批量调用 Embedding API
    embeddings := make([][]float32, len(chunks))
    for i, chunk := range chunks {
        emb, err := e.provider.Embedding(ctx, chunk)
        if err != nil {
            return fmt.Errorf("failed to embed chunk %d: %w", i, err)
        }
        embeddings[i] = emb
    }

    // 3. 平均池化 (多个 chunk -> 单个 vector)
    avgEmbedding := averageEmbeddings(embeddings)

    // 4. 存入数据库
    return e.store.UpdateMemoEmbedding(ctx, memo.ID, avgEmbedding)
}

// EmbedMemoBatch 批量为 Memo 生成 Embedding (用于历史数据迁移)
func (e *Embedder) EmbedMemoBatch(ctx context.Context, memos []*store.Memo) <-chan error {
    // 控制并发数不超过 3 (避免内存溢出)
    // ... 实现
}
```

### 3. 集成到 Memo 服务
**文件路径**: `server/router/api/v1/memo_service.go`

在 `CreateMemo` 和 `UpdateMemo` 后触发 Embedding:
```go
func (s *APIV1Service) CreateMemo(ctx context.Context, request *connect.Request[v1.CreateMemoRequest]) (*connect.Response[v1.Memo], error) {
    // ... 原有逻辑 ...

    // 异步生成 Embedding (不阻塞响应)
    go func() {
        if err := s.embedder.EmbedMemo(context.Background(), memo); err != nil {
            log.Error("Failed to embed memo", zap.Int32("id", memo.ID), zap.Error(err))
        }
    }()

    return connect.NewResponse(memo), nil
}
```

### 4. 向量平均算法
**文件路径**: `server/ai/embedder.go`

```go
// averageEmbeddings 对多个向量取平均
func averageEmbeddings(embeddings [][]float32) []float32 {
    if len(embeddings) == 0 {
        return nil
    }

    n := len(embeddings[0])
    result := make([]float32, n)

    for _, emb := range embeddings {
        for i := 0; i < n; i++ {
            result[i] += emb[i]
        }
    }

    for i := 0; i < n; i++ {
        result[i] /= float32(len(embeddings))
    }

    return result
}
```

## 验收标准

### AC-1: 文档切片逻辑测试
**测试文件**: `server/ai/chunker_test.go`

```go
func TestChunkDocument(t *testing.T) {
    cases := []struct {
        name     string
        content  string
        expected []string
    }{
        {
            name:    "短文档不切片",
            content: "Hello world",
            expected: []string{"Hello world"},
        },
        {
            name:    "长文档切片",
            content: strings.Repeat("test ", 200), // 1000 字符
            expected: func() []string {
                // 预期 3 个切片,每个约 500 字符
                // ... 具体断言
            }(),
        },
        {
            name: "保留段落完整性",
            content: `
                第一段
                第二段
                第三段
            `,
            expected: []string{
                "第一段\n第二段",
                "第二段\n第三段",
            },
        },
    }

    for _, tc := range cases {
        t.Run(tc.name, func(t *testing.T) {
            result := ChunkDocument(tc.content)
            assert.Equal(t, tc.expected, result)
        })
    }
}
```

### AC-2: Embedding 生成测试
**测试文件**: `server/ai/embedder_test.go`

```go
func TestEmbedMemo(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test")
    }

    ctx := context.Background()
    embedder := setupTestEmbedder(t)

    memo := &store.Memo{
        ID:      123,
        Content: "这是一个测试 memo",
    }

    err := embedder.EmbedMemo(ctx, memo)
    assert.NoError(t, err)

    // 验证数据库中有 embedding
    // ...
}
```

### AC-3: 端到端测试
**测试文件**: `server/router/api/v1/memo_service_test.go`

```bash
# 1. 创建 Memo
curl -X POST http://localhost:8081/api/v1/memos \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"content": "测试自动 embedding"}'

# 2. 等待 3 秒 (异步处理)

# 3. 查询数据库
docker exec -it memos-db psql -U memos -d memos << EOF
SELECT id, content, embedding IS NOT NULL as has_embedding
FROM memos
ORDER BY created_ts DESC
LIMIT 1;
EOF

# 预期结果
- has_embedding = true
- embedding 不为 NULL
```

### AC-4: 性能测试
```bash
# 创建 10 个 Memo,记录总时间
time for i in {1..10}; do
  curl -s -X POST http://localhost:8081/api/v1/memos \
    -H "Authorization: Bearer <token>" \
    -H "Content-Type: application/json" \
    -d "{\"content\": \"测试 memo $i\"}"
done

# 预期结果
- 总耗时 < 30 秒
- 无 OOM 错误
- 数据库连接数 < 10
```

### AC-5: 错误处理测试
```bash
# 场景 1: Embedding API 失败
# 1. 设置错误的 API Key
export MEMOS_AI_API_KEY="invalid-key"
# 2. 创建 Memo
# 3. 预期: Memo 创建成功,但日志中有 "Failed to embed memo" 错误

# 场景 2: 网络超时
# 1. 设置不可达的 BaseURL
export MEMOS_AI_BASE_URL="http://invalid:9999"
# 2. 创建 Memo
# 3. 预期: 重试 3 次后失败,Memo 仍然创建成功
```

## 回滚方案
- 从 `memo_service.go` 中移除 Embedding 调用
- 数据库中的 `embedding` 列保留(允许为 NULL)

## 注意事项
- Embedding 生成必须异步执行,不阻塞主流程
- 并发数控制在 3 以内,避免 2G 内存溢出
- 失败的 Embedding 记录日志,不影响 Memo 创建
- 文档切片保留段落边界,提升语义完整性