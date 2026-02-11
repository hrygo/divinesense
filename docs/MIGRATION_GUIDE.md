# DivineSense 迁移指南

从 Memos 迁移到 DivineSense 的平滑升级指南。

---

## 新用户快速开始

**如果你是首次使用 DivineSense**，不需要迁移！直接：

```bash
# 1. 配置环境变量
cp .env.example .env

# 2. 启动服务（会自动创建 divinesense 数据库）
make start
```

数据库将自动初始化，默认配置：
- 数据库: `divinesense`
- 用户: `divinesense`
- 密码: `divinesense`
- 端口: `25432`

---

## 从 Memos 迁移

**适用场景**: 现有 Memos 用户，希望迁移到 DivineSense

**⚠️ 重要变更**: DivineSense 已移除对 `MEMOS_*` 环境变量的向后兼容支持，迁移时必须更新所有环境变量。

---

## 迁移前准备

### 1. 备份数据

**Docker 部署**:
```bash
# 备份数据库
docker exec memos-postgres pg_dump -U memos memos | gzip > backup_pre_migration_$(date +%Y%m%d).sql.gz

# 备份数据卷（可选）
docker run --rm -v memos_postgres_data:/data -v $(pwd):/backup alpine tar czf /backup/postgres_volume_$(date +%Y%m%d).tar.gz -C /data .
```

**本地开发**:
```bash
# 备份 SQLite 数据库
cp /var/opt/memos/memos_prod.db /var/opt/memos/backup_$(date +%Y%m%d).db

# 或 Windows
copy "C:\ProgramData\memos\memos_prod.db" "C:\ProgramData\memos\backup_%DATE%.db"
```

### 2. 记录当前配置

```bash
# 导出当前环境变量（用于参考）
env | grep MEMOS_ > memos_env_backup.txt
```

---

## 迁移方案

### 方案 A: 自动化迁移脚本（推荐）

**适用场景**: 从 memos 数据库迁移到 divinesense 数据库

```bash
# 运行迁移脚本
./scripts/migrate_to_divinesense.sh \
  --old-db memos \
  --new-db divinesense \
  --user memos \
  --port 5432

# 脚本会自动：
# 1. 备份 memos 数据库
# 2. 创建 divinesense 数据库
# 3. 迁移所有数据
# 4. 验证迁移结果
```

### 方案 B: 一次性迁移

**适用场景**: 新安装或开发环境

**步骤**:

1. **停止服务**
   ```bash
   make stop
   # 或
   docker-compose -f docker/compose/prod.yml down
   ```

2. **重命名 Docker 卷** (如果使用)
   ```bash
   docker volume rename memos_data divinesense_data
   docker volume rename memos_postgres_data divinesense_postgres_data
   ```

3. **更新环境变量**
   ```bash
   # ⚠️ 必须更新所有环境变量前缀
   # 批量替换 .env 文件
   sed -i 's/MEMOS_/DIVINESENSE_/g' .env
   sed -i 's/memos:/divinesense:/g' .env
   sed -i 's@/memos@/divinesense@g' .env
   ```

   **重要**: 由于 `MEMOS_*` 前缀不再被支持，必须将所有环境变量更新为 `DIVINESENSE_*`。

4. **更新数据库连接**

   在 PostgreSQL 中创建新数据库并迁移数据：
   ```bash
   # 连接到 PostgreSQL
   docker exec -it divinesense-postgres psql -U divinesense

   -- 在 psql 中执行:
   CREATE DATABASE divinesense;
   \q

   # 迁移数据
   docker exec memos-postgres pg_dump -U memos memos | docker exec -i divinesense-postgres psql -U divinesense -d divinesense
   ```

5. **启动服务**
   ```bash
   make start
   ```

---

## 环境变量映射表

| 旧变量 (MEMOS_*) | 新变量 (DIVINESENSE_*) | 状态 |
|:------------------|:----------------------|:-----|
| `MEMOS_DRIVER` | `DIVINESENSE_DRIVER` | **必须更新** |
| `MEMOS_DSN` | `DIVINESENSE_DSN` | **必须更新** |
| `MEMOS_MODE` | `DIVINESENSE_MODE` | **必须更新** |
| `MEMOS_PORT` | - | 移除 (使用固定端口) |
| `MEMOS_DATA` | - | 移除 (使用固定路径) |
| `MEMOS_AI_ENABLED` | `DIVINESENSE_AI_ENABLED` | **必须更新** |
| `MEMOS_AI_*_PROVIDER` | `DIVINESENSE_AI_*_PROVIDER` | **必须更新** |
| `MEMOS_AI_*_API_KEY` | `DIVINESENSE_AI_*_API_KEY` | **必须更新** |
| `MEMOS_OCR_ENABLED` | `DIVINESENSE_OCR_ENABLED` | **必须更新** |

⚠️ **重要**: DivineSense 不再支持 `MEMOS_*` 前缀的环境变量，必须在迁移时全部更新。

---

## 数据目录变更

| 平台 | 旧路径 | 新路径 |
|:-----|:-------|:-------|
| Linux | `/var/opt/memos` | `/var/opt/divinesense` |
| Windows | `C:\ProgramData\memos` | `C:\ProgramData\divinesense` |
| macOS | `/Library/Application Support/memos` | `/Library/Application Support/divinesense` |

**迁移数据目录**:
```bash
# Linux
sudo mv /var/opt/memos /var/opt/divinesense

# Windows
robocopy "C:\ProgramData\memos" "C:\ProgramData\divinesense" /E /R:0 /W:0
```

---

## Docker 容器名称变更

| 旧名称 | 新名称 |
|:-------|:-------|
| `memos-postgres` | `divinesense-postgres` |
| `memos-postgres-dev` | `divinesense-postgres-dev` |
| `memos` | `divinesense` |
| `memos_network` | `divinesense_network` |
| `memos_data` | `divinesense_data` |
| `memos_postgres_data` | `divinesense_postgres_data` |

---

## 验证清单

迁移完成后，验证以下项目：

- [ ] 服务正常启动 (`make status`)
- [ ] 用户可以登录
- [ ] 笔记数据完整
- [ ] AI 对话功能正常
- [ ] 日程管理功能正常
- [ ] 附件可以访问
- [ ] 日志无错误 (`make logs`)

---

## 回滚方案

### 快速回滚

```bash
# 停止服务
make stop

# 恢复数据库备份
gunzip < backup_pre_migration_*.sql.gz | docker exec -i divinesense-postgres psql -U divinesense -d divinesense

# 恢复代码
git checkout HEAD~1

# 重新构建并启动
make build-all
make start
```

### Docker 卷回滚

```bash
# 重命名卷回去
docker volume rename divinesense_data memos_data
docker volume rename divinesense_postgres_data memos_postgres_data
```

---

## 常见问题

### Q: 迁移后登录失败？

**A**: 检查数据库名称和连接字符串：
```bash
# 验证数据库连接
docker exec divinesense-postgres psql -U divinesense -d divinesense -c "SELECT 1;"
```

### Q: AI 功能不工作？

**A**: 确认环境变量已正确设置（必须使用 `DIVINESENSE_*` 前缀）：
```bash
docker exec divinesense env | grep DIVINESENSE_AI
```

### Q: 旧环境变量还支持吗？

**A**: ❌ 不再支持。从 v0.98.0 开始，`MEMOS_*` 前缀已被移除，必须使用 `DIVINESENSE_*` 前缀。

### Q: 需要重新训练 AI 模型吗？

**A**: 不需要，AI 模型配置无需更改。

---

## 获取帮助

如遇到问题，请：

1. 检查日志: `make logs`
2. 验证配置: `cat .env | grep DIVINESENSE`
3. 提交 Issue: https://github.com/hrygo/divinesense/issues

---

**最后更新**: 2026-02-11 (v0.98.0 - 移除 MEMOS_ 前缀支持)
