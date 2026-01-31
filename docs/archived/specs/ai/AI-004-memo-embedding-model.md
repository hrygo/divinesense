# AI-004: MemoEmbedding 模型

## 概述

定义 MemoEmbedding 相关的 Go 结构体和基础方法。

## 目标

提供 Memo 向量存储的数据模型定义。

## 交付物

- `store/memo_embedding.go` (新增)

## 实现规格

### 结构体定义

```go
package store

import "context"

// MemoEmbedding 表示 Memo 的向量嵌入
type MemoEmbedding struct {
    ID        int32
    MemoID    int32
    Embedding []float32  // 1024 维向量
    Model     string     // 模型标识，如 "BAAI/bge-m3"
    CreatedTs int64
    UpdatedTs int64
}

// FindMemoEmbedding 查询条件
type FindMemoEmbedding struct {
    MemoID *int32
    Model  *string
}

// MemoWithScore 向量搜索结果
type MemoWithScore struct {
    Memo  *Memo
    Score float32  // 相似度分数 (0-1, 越大越相似)
}

// VectorSearchOptions 向量搜索选项
type VectorSearchOptions struct {
    UserID int32       // 必填，只搜索该用户的 Memo
    Vector []float32   // 查询向量
    Limit  int         // 返回数量，默认 10
}
```

### Store 方法

```go
// UpsertMemoEmbedding 插入或更新向量
func (s *Store) UpsertMemoEmbedding(ctx context.Context, embedding *MemoEmbedding) (*MemoEmbedding, error) {
    return s.driver.UpsertMemoEmbedding(ctx, embedding)
}

// GetMemoEmbedding 获取指定 Memo 的向量
func (s *Store) GetMemoEmbedding(ctx context.Context, memoID int32, model string) (*MemoEmbedding, error) {
    list, err := s.driver.ListMemoEmbeddings(ctx, &FindMemoEmbedding{
        MemoID: &memoID,
        Model:  &model,
    })
    if err != nil {
        return nil, err
    }
    if len(list) == 0 {
        return nil, nil
    }
    return list[0], nil
}

// ListMemoEmbeddings 列出向量
func (s *Store) ListMemoEmbeddings(ctx context.Context, find *FindMemoEmbedding) ([]*MemoEmbedding, error) {
    return s.driver.ListMemoEmbeddings(ctx, find)
}

// DeleteMemoEmbedding 删除向量
func (s *Store) DeleteMemoEmbedding(ctx context.Context, memoID int32) error {
    return s.driver.DeleteMemoEmbedding(ctx, memoID)
}
```

## 验收标准

### AC-1: 文件创建
- [x] `store/memo_embedding.go` 文件存在
- [x] 结构体定义符合规格

### AC-2: 编译通过
- [x] `go build ./store/...` 无错误

### AC-3: 结构体完整
- [x] `MemoEmbedding` 包含所有必需字段
- [x] `MemoWithScore` 包含 `Memo` 和 `Score`
- [x] `VectorSearchOptions` 包含 `UserID`, `Vector`, `Limit`

### AC-4: 方法签名正确
- [x] `UpsertMemoEmbedding` 方法存在
- [x] `GetMemoEmbedding` 方法存在
- [x] `ListMemoEmbeddings` 方法存在
- [x] `DeleteMemoEmbedding` 方法存在

## 实现状态

✅ **已完成** - 实现于 [store/memo_embedding.go](../../store/memo_embedding.go)

## 测试命令

```bash
go build ./store/...
```

## 依赖

- AI-003 (数据库迁移)

## 预估时间

0.5 小时
