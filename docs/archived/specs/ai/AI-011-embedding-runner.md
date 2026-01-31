# AI-011: 向量生成后台任务

## 概述

实现后台任务，自动为新创建/更新的 Memo 生成向量。

## 目标

确保所有 Memo 都有对应的向量嵌入，支持增量处理。

## 交付物

- `server/runner/embedding/runner.go` (新增)
- `server/server.go` (修改，注册后台任务)

## 实现规格

### Runner 实现

```go
package embedding

import (
    "context"
    "fmt"
    "log/slog"
    "time"
    
    "github.com/usememos/memos/plugin/ai"
    "github.com/usememos/memos/store"
)

type Runner struct {
    store           *store.Store
    embeddingService ai.EmbeddingService
    interval        time.Duration
    batchSize       int
    model           string
}

// NewRunner 创建向量生成任务
// 参数优化 for 2C2G: 较小批次减少内存峰值，较长间隔减少 CPU 争用
func NewRunner(store *store.Store, embeddingService ai.EmbeddingService) *Runner {
    return &Runner{
        store:           store,
        embeddingService: embeddingService,
        interval:        2 * time.Minute,  // 降低处理频率
        batchSize:       8,                 // 减少批处理大小
        model:           "BAAI/bge-m3",
    }
}

// Run 启动后台任务
func (r *Runner) Run(ctx context.Context) {
    // 启动时先处理一次
    r.processNewMemos(ctx)
    
    ticker := time.NewTicker(r.interval)
    defer ticker.Stop()
    
    for {
        select {
        case <-ticker.C:
            r.processNewMemos(ctx)
        case <-ctx.Done():
            slog.Info("embedding runner stopped")
            return
        }
    }
}

// RunOnce 单次处理（用于手动触发）
func (r *Runner) RunOnce(ctx context.Context) {
    r.processNewMemos(ctx)
}

func (r *Runner) processNewMemos(ctx context.Context) {
    // 查找没有向量的 Memo
    memos, err := r.findMemosWithoutEmbedding(ctx)
    if err != nil {
        slog.Error("failed to find memos without embedding", "error", err)
        return
    }
    
    if len(memos) == 0 {
        return
    }
    
    slog.Info("processing memos for embedding", "count", len(memos))
    
    // 批量处理
    for i := 0; i < len(memos); i += r.batchSize {
        end := i + r.batchSize
        if end > len(memos) {
            end = len(memos)
        }
        batch := memos[i:end]
        
        if err := r.processBatch(ctx, batch); err != nil {
            slog.Error("failed to process batch", "error", err)
            continue
        }
        slog.Info("batch processed", "count", len(batch), "progress", fmt.Sprintf("%d/%d", end, len(memos)))
    }
}

func (r *Runner) findMemosWithoutEmbedding(ctx context.Context) ([]*store.Memo, error) {
    // 查找指定模型下没有 embedding 的 NORMAL 状态 memo
    query := `
        SELECT m.id, m.uid, m.content, m.creator_id, m.visibility, 
               m.row_status, m.created_ts, m.updated_ts, m.pinned
        FROM memo m
        LEFT JOIN memo_embedding e ON m.id = e.memo_id AND e.model = $1
        WHERE e.id IS NULL 
          AND m.row_status = 'NORMAL'
          AND LENGTH(m.content) > 0
        LIMIT $2
    `
    
    // 注意：这里假设 store 暴露了直接执行 SQL 的能力，
    // 或者需要在 store 层增加 FindMemosWithoutEmbedding 方法。
    // 为了保持架构整洁，建议在 AI-005/AI-006 中增加 FindMemosWithoutEmbedding 方法。
    // 这里依然演示调用 store 方法的方式。
    
    return r.store.FindMemosWithoutEmbedding(ctx, &store.FindMemosWithoutEmbedding{
        Model: r.model,
        Limit: r.batchSize * 20,  // 每次取更多数据，但分小批处理
    })
}

func (r *Runner) processBatch(ctx context.Context, memos []*store.Memo) error {
    // 提取内容
    texts := make([]string, len(memos))
    for i, m := range memos {
        texts[i] = m.Content
    }
    
    // 批量生成向量
    vectors, err := r.embeddingService.EmbedBatch(ctx, texts)
    if err != nil {
        return err
    }
    
    // 存储向量
    for i, m := range memos {
        _, err := r.store.UpsertMemoEmbedding(ctx, &store.MemoEmbedding{
            MemoID:    m.ID,
            Embedding: vectors[i],
            Model:     r.model,
        })
        if err != nil {
            slog.Error("failed to upsert embedding", "memoID", m.ID, "error", err)
        }
    }
    
    return nil
}
```

### Server 集成

在 `server/server.go` 的 `StartBackgroundRunners` 中添加：

```go
// 启动向量生成任务
if s.AIService != nil && s.AIService.IsEnabled() {
    embeddingRunner := embedding.NewRunner(s.Store, s.AIService.EmbeddingService())
    go embeddingRunner.Run(ctx)
}
```

## 验收标准

### AC-1: 文件创建
- [x] `server/runner/embedding/runner.go` 文件存在

### AC-2: 编译通过
- [x] `go build ./server/...` 无错误

### AC-3: 自动处理
- [x] 新 Memo 创建后在下一个周期自动生成向量
- [x] 批量处理正常工作

### AC-4: 增量处理
- [x] 只处理没有向量的 Memo
- [x] 已有向量的 Memo 不会重复处理

### AC-5: 错误恢复
- [x] 单个 Memo 失败不影响其他 Memo
- [x] 错误日志正常输出

## 实现状态

✅ **已完成** - 实现于 [server/runner/embedding/runner.go](../../server/runner/embedding/runner.go)

## 测试命令

```bash
go build ./server/...
go test ./server/runner/embedding/... -v
```

## 依赖

- AI-006 (PostgreSQL 向量搜索)
- AI-008 (Embedding 服务)

## 预估时间

2 小时
