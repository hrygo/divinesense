# SPEC-002: pgvector 扩展启用与数据表迁移

**优先级**: P0 (阻塞)
**预计工时**: 3 小时
**依赖**: SPEC-001

## 目标
启用 pgvector 扩展,并创建支持向量检索的数据库表结构。

## 实施内容

### 1. 创建迁移文件
**文件路径**: `store/migration/postgres/0.22/1__enable_pgvector.sql`
```sql
-- 启用 pgvector 扩展
CREATE EXTENSION IF NOT EXISTS vector;
```

**文件路径**: `store/migration/postgres/0.22/2__add_embedding_column.sql`
```sql
-- 为 memos 表添加 embedding 字段
ALTER TABLE memos ADD COLUMN IF NOT EXISTS embedding vector(1024);

-- 创建 HNSW 索引 (提升检索性能)
CREATE INDEX IF NOT EXISTS memos_embedding_idx
ON memos USING hnsw (embedding vector_cosine_ops)
WITH (m = 16, ef_construction = 64);

-- 添加 GIN 索引优化文本检索
CREATE INDEX IF NOT EXISTS memos_content_search_idx
ON memos USING gin (to_tsvector('simple', content));
```

### 2. 更新 Store 接口
**文件**: `store/memo.go`

新增方法:
```go
// UpdateMemoEmbedding 更新 memo 的向量表示
UpdateMemoEmbedding(ctx context.Context, id int32, embedding []float32) error

// SearchMemosByVector 语义检索
SearchMemosByVector(ctx context.Context, embedding []float32, limit int) ([]*Memo, []float32, error)
```

### 3. 实现层
**文件**: `store/db/postgres/memo.go`

实现 `UpdateMemoEmbedding` 和 `SearchMemosByVector`:
```go
func (d *Driver) SearchMemosByVector(ctx context.Context, embedding []float32, limit int) ([]*Memo, []float32, error) {
    query := `
        SELECT id, content, creator_id, created_ts, updated_ts,
               1 - (embedding <=> $1) as similarity
        FROM memos
        WHERE embedding IS NOT NULL
        ORDER BY embedding <=> $1
        LIMIT $2
    `
    // ... 实现细节
}
```

## 验收标准

### AC-1: 扩展启用成功
```bash
# 执行
docker exec -it memos-db psql -U memos -d memos -c "SELECT * FROM pg_extension WHERE extname = 'vector';"

# 预期结果
- vector 扩展已安装
- 版本 > 0.5.0
```

### AC-2: 表结构验证
```bash
# 执行
docker exec -it memos-db psql -U memos -d memos -c "\d memos" | grep embedding

# 预期结果
- embedding 列存在
- 类型为 vector(1024)
```

### AC-3: 索引创建验证
```bash
# 执行
docker exec -it memos-db psql -U memos -d memos -c "\d memos"

# 预期结果
- memos_embedding_idx 索引存在 (USING hnsw)
- memos_content_search_idx 索引存在 (USING gin)
```

### AC-4: 手动向量插入测试
```bash
# 执行
docker exec -it memos-db psql -U memos -d memos << EOF
INSERT INTO memos (creator_id, content, embedding)
VALUES (1, '测试 memo', '[0.1, 0.2, 0.3]'::vector);

SELECT id, content, 1 - (embedding <=> '[0.1, 0.2, 0.3]'::vector) as similarity
FROM memos
WHERE embedding IS NOT NULL
ORDER BY embedding <=> '[0.1, 0.2, 0.3]'::vector
LIMIT 5;
EOF

# 预期结果
- 插入成功
- 相似度查询返回结果
- similarity 值在 [0, 1] 范围内
```

### AC-5: 代码编译通过
```bash
# 执行
cd /path/to/memos
go build ./store/...
go test ./store/...

# 预期结果
- 编译无错误
- 测试全部通过
```

## 回滚方案
```sql
-- 回滚迁移
DROP INDEX IF EXISTS memos_embedding_idx;
DROP INDEX IF EXISTS memos_content_search_idx;
ALTER TABLE memos DROP COLUMN IF EXISTS embedding;
DROP EXTENSION IF EXISTS vector;
```

## 注意事项
- vector 维度(1024)需与 Embedding 模型匹配
- HNSW 索引参数(m=16, ef_construction=64)针对 2G 内存优化
- cosine 距离适用于语义检索