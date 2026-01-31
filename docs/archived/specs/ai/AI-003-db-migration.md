# AI-003: 数据库迁移 pgvector

## 概述

创建 PostgreSQL 数据库迁移，启用 pgvector 扩展并创建 memo_embedding 表。

## 目标

为 Memo 向量存储提供数据库基础设施。

## 交付物

- `store/migration/postgres/0.30/1__add_pgvector.sql`

## 实现规格

### 迁移 SQL

```sql
-- 启用 pgvector 扩展
CREATE EXTENSION IF NOT EXISTS vector;

-- Memo 向量嵌入表
CREATE TABLE memo_embedding (
    id SERIAL PRIMARY KEY,
    memo_id INTEGER NOT NULL,
    embedding vector(1024) NOT NULL,
    model VARCHAR(100) NOT NULL DEFAULT 'BAAI/bge-m3',
    created_ts BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW())::BIGINT,
    updated_ts BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW())::BIGINT,
    
    -- 外键约束
    CONSTRAINT fk_memo_embedding_memo 
        FOREIGN KEY (memo_id) 
        REFERENCES memo(id) 
        ON DELETE CASCADE,
    
    -- 每个 memo 每个模型只有一个向量
    CONSTRAINT uq_memo_embedding_memo_model 
        UNIQUE (memo_id, model)
);

-- HNSW 向量索引 (余弦相似度)
-- 参数优化 for 2C2G: m=8 (连接数), ef_construction=32 (构建候选数)
-- 较低参数减少内存/CPU 使用，牺牲少量精度
CREATE INDEX idx_memo_embedding_hnsw
ON memo_embedding USING hnsw (embedding vector_cosine_ops)
WITH (m = 8, ef_construction = 32);

-- memo_id 索引 (加速 JOIN)
CREATE INDEX idx_memo_embedding_memo_id 
ON memo_embedding (memo_id);

-- 更新时间触发器
CREATE OR REPLACE FUNCTION update_memo_embedding_updated_ts()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_ts = EXTRACT(EPOCH FROM NOW())::BIGINT;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_memo_embedding_updated_ts
    BEFORE UPDATE ON memo_embedding
    FOR EACH ROW
    EXECUTE FUNCTION update_memo_embedding_updated_ts();
```

### 目录结构

```
store/migration/postgres/
├── 0.29/
│   └── ...
├── 0.30/
│   └── 1__add_pgvector.sql   # 新增
└── LATEST.sql                 # 需更新
```

### LATEST.sql 更新

在 `LATEST.sql` 末尾追加 memo_embedding 表定义。

## 验收标准

### AC-1: 迁移文件存在
- [x] `store/migration/postgres/0.30/1__add_pgvector.sql` 文件存在
- [x] 文件内容符合规格

### AC-2: 迁移执行成功
- [x] 在干净的 PostgreSQL 数据库上执行迁移无错误
- [x] `memo_embedding` 表创建成功
- [x] `idx_memo_embedding_hnsw` 索引创建成功
- [x] 触发器创建成功

### AC-3: 表结构正确
- [x] `embedding` 列类型为 `vector(1024)`
- [x] `memo_id` 外键指向 `memo(id)`
- [x] `ON DELETE CASCADE` 生效

### AC-4: 级联删除测试
- [x] 删除 memo 时，对应的 embedding 自动删除

## 实现状态

✅ **已完成** - 实现于 [store/migration/postgres/0.30/1__add_pgvector.sql](../../store/migration/postgres/0.30/1__add_pgvector.sql)

## 测试命令

```bash
# 在 PostgreSQL 中执行
psql -d memos_test -f store/migration/postgres/0.30/1__add_pgvector.sql

# 验证表结构
psql -d memos_test -c "\d memo_embedding"

# 验证索引
psql -d memos_test -c "\di idx_memo_embedding*"

# 测试级联删除
psql -d memos_test -c "
INSERT INTO memo (id, uid, content, creator_id, visibility, row_status, created_ts, updated_ts) 
VALUES (99999, 'test-ai-001', 'test', 1, 'PRIVATE', 'NORMAL', 0, 0);
INSERT INTO memo_embedding (memo_id, embedding, model) 
VALUES (99999, ARRAY_FILL(0.1, ARRAY[1024])::vector, 'test');
DELETE FROM memo WHERE id = 99999;
SELECT COUNT(*) FROM memo_embedding WHERE memo_id = 99999;  -- 应返回 0
"
```

## 依赖

- PostgreSQL 安装 pgvector 扩展

## 预估时间

1 小时
