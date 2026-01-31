# SPEC-001: PostgreSQL + pgvector 基础设施

**优先级**: P0 (阻塞)
**预计工时**: 2 小时
**依赖**: 无

## 目标
将现有 SQLite 数据库迁移到 PostgreSQL + pgvector,为语义检索提供基础设施。

## 实施内容

### 1. 修改 `docker-compose.yml`
- 使用 `pgvector/pgvector:pg16` 镜像
- 针对 2G 内存优化配置参数:
  - `shared_buffers=128MB`
  - `work_mem=4MB`
  - `maintenance_work_mem=64MB`
  - `max_connections=50`
- 配置数据持久化卷

### 2. 更新环境变量配置
- 新增 `MEMOS_DRIVER=postgres`
- 新增 `MEMOS_DSN=postgres://memos:memos@db:5432/memos`

### 3. 文档更新
- 在 `README.md` 中添加 PostgreSQL 部署说明
- 添加环境变量说明文档

## 验收标准

### AC-1: Docker Compose 启动成功
```bash
# 执行
docker-compose up -d

# 预期结果
- db 容器状态为 healthy
- memos 容器成功连接数据库
- 日志无错误
```

### AC-2: PostgreSQL 参数验证
```bash
# 执行
docker exec -it memos-db psql -U memos -d memos -c "SHOW shared_buffers;"
docker exec -it memos-db psql -U memos -d memos -c "SHOW work_mem;"
docker exec -it memos-db psql -U memos -d memos -c "SHOW max_connections;"

# 预期结果
- shared_buffers = 128MB
- work_mem = 4MB
- max_connections = 50
```

### AC-3: 基础连接测试
```bash
# 执行
curl http://localhost:8081/api/v1/status

# 预期结果
- HTTP 200
- 数据库状态为 healthy
```

### AC-4: 内存使用检查
```bash
# 执行
docker stats memos-db --no-stream

# 预期结果
- 内存使用 < 1GB
- 无 OOM 警告
```

## 回滚方案
保留原有 SQLite 配置,通过环境变量切换:
```bash
MEMOS_DRIVER=sqlite docker-compose up
```

## 注意事项
- 本变更放弃对 SQLite/MySQL 的兼容性支持
- 数据迁移方案在后续 Spec 中定义
- 确保 PostgreSQL 版本为 16(支持 pgvector)