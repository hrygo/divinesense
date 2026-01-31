# AI-005: Driver 接口扩展

## 概述

扩展 Store Driver 接口，添加 MemoEmbedding 相关方法签名。

## 目标

为数据库驱动定义向量操作的统一接口。

## 交付物

- `store/driver.go` (修改)

## 实现规格

### 新增接口方法

在 `Driver` interface 中添加：

```go
type Driver interface {
    // ... 现有方法 ...

    // MemoEmbedding 相关方法
    UpsertMemoEmbedding(ctx context.Context, embedding *MemoEmbedding) (*MemoEmbedding, error)
    ListMemoEmbeddings(ctx context.Context, find *FindMemoEmbedding) ([]*MemoEmbedding, error)
    DeleteMemoEmbedding(ctx context.Context, memoID int32) error
    
    // 向量搜索
    SearchMemosByVector(ctx context.Context, opts *VectorSearchOptions) ([]*MemoWithScore, error)
}
```

### 方法说明

| 方法                  | 说明                                        |
| --------------------- | ------------------------------------------- |
| `UpsertMemoEmbedding` | 插入或更新向量，按 memo_id + model 唯一约束 |
| `ListMemoEmbeddings`  | 根据条件查询向量列表                        |
| `DeleteMemoEmbedding` | 删除指定 memo 的所有向量                    |
| `SearchMemosByVector` | 执行向量相似度搜索，返回 Memo 和分数        |

## 验收标准

### AC-1: 接口更新
- [x] `Driver` 接口包含 4 个新方法
- [x] 方法签名正确

### AC-2: 编译检查
- [x] `go build ./store/...` 无错误 (会报 not implemented，符合预期)

### AC-3: SQLite/MySQL 占位实现
- [x] SQLite 驱动添加占位方法 (返回 not supported 错误)
- [x] MySQL 驱动添加占位方法 (返回 not supported 错误)

## 实现状态

✅ **已完成** - 实现于 [store/driver.go](../../store/driver.go)

## 测试命令

```bash
go build ./store/...
```

## 依赖

- AI-004 (MemoEmbedding 模型)

## 预估时间

0.5 小时
