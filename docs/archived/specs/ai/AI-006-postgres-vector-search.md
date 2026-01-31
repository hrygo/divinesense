# AI-006: PostgreSQL 向量搜索

## 概述

实现 PostgreSQL 驱动的向量操作方法，包括 CRUD 和相似度搜索。

## 目标

提供完整的向量存储和搜索能力。

## 交付物

- `store/db/postgres/memo_embedding.go` (新增)

## 实现规格

### 文件内容

```go
package postgres

import (
    "context"
    "strconv"
    "strings"

    "github.com/usememos/memos/store"
)

// UpsertMemoEmbedding 插入或更新向量
func (d *DB) UpsertMemoEmbedding(ctx context.Context, embedding *store.MemoEmbedding) (*store.MemoEmbedding, error) {
    query := `
        INSERT INTO memo_embedding (memo_id, embedding, model, created_ts, updated_ts)
        VALUES ($1, $2::vector, $3, EXTRACT(EPOCH FROM NOW())::BIGINT, EXTRACT(EPOCH FROM NOW())::BIGINT)
        ON CONFLICT (memo_id, model) 
        DO UPDATE SET embedding = $2::vector, updated_ts = EXTRACT(EPOCH FROM NOW())::BIGINT
        RETURNING id, memo_id, model, created_ts, updated_ts
    `
    vectorStr := vectorToString(embedding.Embedding)
    
    result := &store.MemoEmbedding{}
    err := d.db.QueryRowContext(ctx, query, 
        embedding.MemoID, 
        vectorStr, 
        embedding.Model,
    ).Scan(&result.ID, &result.MemoID, &result.Model, &result.CreatedTs, &result.UpdatedTs)
    
    if err != nil {
        return nil, err
    }
    result.Embedding = embedding.Embedding
    return result, nil
}

// ListMemoEmbeddings 查询向量列表
func (d *DB) ListMemoEmbeddings(ctx context.Context, find *store.FindMemoEmbedding) ([]*store.MemoEmbedding, error) {
    // 实现查询逻辑
}

// DeleteMemoEmbedding 删除向量
func (d *DB) DeleteMemoEmbedding(ctx context.Context, memoID int32) error {
    _, err := d.db.ExecContext(ctx, "DELETE FROM memo_embedding WHERE memo_id = $1", memoID)
    return err
}

// SearchMemosByVector 向量相似度搜索
func (d *DB) SearchMemosByVector(ctx context.Context, opts *store.VectorSearchOptions) ([]*store.MemoWithScore, error) {
    query := `
        SELECT m.id, m.uid, m.content, m.creator_id, m.visibility, 
               m.row_status, m.created_ts, m.updated_ts, m.pinned,
               1 - (e.embedding <=> $1::vector) AS score
        FROM memo m
        JOIN memo_embedding e ON m.id = e.memo_id
        WHERE m.creator_id = $2 
          AND m.row_status = 'NORMAL'
        ORDER BY e.embedding <=> $1::vector
        LIMIT $3
    `
    
    vectorStr := vectorToString(opts.Vector)
    limit := opts.Limit
    if limit <= 0 {
        limit = 10
    }
    
    rows, err := d.db.QueryContext(ctx, query, vectorStr, opts.UserID, limit)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    
    var results []*store.MemoWithScore
    for rows.Next() {
        memo := &store.Memo{}
        var score float32
        err := rows.Scan(
            &memo.ID, &memo.UID, &memo.Content, &memo.CreatorID, &memo.Visibility,
            &memo.RowStatus, &memo.CreatedTs, &memo.UpdatedTs, &memo.Pinned,
            &score,
        )
        if err != nil {
            return nil, err
        }
        results = append(results, &store.MemoWithScore{Memo: memo, Score: score})
    }
    
    return results, rows.Err()
}

// vectorToString 将 float32 切片转为 pgvector 格式 (高性能版本)
func vectorToString(v []float32) string {
    var sb strings.Builder
    sb.Grow(len(v) * 12) // 预分配内存
    sb.WriteByte('[')
    for i, f := range v {
        if i > 0 {
            sb.WriteByte(',')
        }
        sb.WriteString(strconv.FormatFloat(float64(f), 'g', -1, 32))
    }
    sb.WriteByte(']')
    return sb.String()
}
```

## 验收标准

### AC-1: 文件创建
- [x] `store/db/postgres/memo_embedding.go` 文件存在

### AC-2: 编译通过
- [x] `go build ./store/db/postgres/...` 无错误

### AC-3: CRUD 功能
- [x] `UpsertMemoEmbedding` 能插入新向量
- [x] `UpsertMemoEmbedding` 能更新已存在向量 (ON CONFLICT)
- [x] `ListMemoEmbeddings` 能按 memo_id 查询
- [x] `DeleteMemoEmbedding` 能删除指定 memo 的向量

### AC-4: 向量搜索功能
- [x] `SearchMemosByVector` 返回按相似度排序的结果
- [x] 只返回指定用户的 Memo
- [x] 只返回 row_status = NORMAL 的 Memo
- [x] Score 值在 0-1 范围内

### AC-5: 单元测试
- [x] 测试文件 `store/test/memo_embedding_test.go` 存在
- [x] 所有测试通过

## 实现状态

✅ **已完成** - 实现于 [store/db/postgres/memo_embedding.go](../../store/db/postgres/memo_embedding.go)

## 测试命令

```bash
DRIVER=postgres DSN="postgres://user:pass@localhost:25432/memos_test?sslmode=disable" \
go test ./store/test/... -run TestMemoEmbedding -v
```

## 依赖

- AI-003 (数据库迁移)
- AI-004 (MemoEmbedding 模型)
- AI-005 (Driver 接口)

## 预估时间

3 小时
