# PostgreSQL 迁移开发指南

> **保鲜状态**: ✅ 已验证 (2026-02-06) | **最后检查**: v0.93.0

## 概述

本目录包含 PostgreSQL 数据库的迁移脚本和 schema 定义。

**核心原则**：**任何数据库变更必须同时更新两处**
1. `migrate/YYYYMMDDHHMMSS_description.up.sql` - 增量迁移（用于已部署数据库升级）
2. `schema/LATEST.sql` - 完整 schema（用于全新安装）

---

## 目录结构

```
store/migration/postgres/
├── CLAUDE.md              # 本文档
├── migrate/               # 增量迁移文件
│   ├── 20260203000000_baseline.up.sql
│   ├── 20260203000000_baseline.down.sql
│   └── ...
└── schema/
    └── LATEST.sql         # 完整 schema（全新安装用）
```

---

## 添加新迁移的完整流程

### 步骤 1：创建增量迁移文件

```bash
# 创建迁移文件（命名格式：YYYYMMDDHHMMSS_description.up.sql）
touch store/migration/postgres/migrate/$(date +%Y%m%d%H%M%S)_add_feature.up.sql
touch store/migration/postgres/migrate/$(date +%Y%m%d%H%M%S)_add_feature.down.sql
```

**命名规范**：
- 时间戳：14 位数字（YYYYMMDDHHMMSS）
- 描述：小写下划线分隔（`add_feature`）
- 文件后缀：`.up.sql`（应用）、`.down.sql`（回滚）

### 步骤 2：编写迁移内容

**`migrate/20260206000001_add_feature.up.sql`**：
```sql
-- Add feature table for new functionality
CREATE TABLE feature (
  id SERIAL PRIMARY KEY,
  name TEXT NOT NULL,
  created_ts BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW())
);

CREATE INDEX idx_feature_name ON feature(name);
```

**`migrate/20260206000001_add_feature.down.sql`**：
```sql
-- Rollback feature table
DROP INDEX IF EXISTS idx_feature_name;
DROP TABLE IF EXISTS feature;
```

### 步骤 3：**同步到 LATEST.sql** ⚠️ 关键步骤

编辑 `schema/LATEST.sql`，在合适位置添加相同的表定义。

**示例**：在 `schema/LATEST.sql` 中添加：
```sql
-- feature (V0.93.0)
-- Stores feature data for new functionality
CREATE TABLE feature (
  id SERIAL PRIMARY KEY,
  name TEXT NOT NULL,
  created_ts BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW())
);

CREATE INDEX idx_feature_name ON feature(name);
```

### 步骤 4：验证一致性

```bash
# 检查 LATEST.sql 中的表数量
grep -c "^CREATE TABLE" schema/LATEST.sql
```

---

## 常见错误

### ❌ 错误 1：只创建迁移文件，忘记更新 LATEST.sql

**症状**：全新安装缺少表，出现 "relation does not exist" 错误

**修复**：立即将变更同步到 `schema/LATEST.sql`

### ❌ 错误 2：LATEST.sql 和迁移文件内容不一致

**症状**：全新安装和升级后的数据库结构不同

**修复**：确保两处定义完全一致（包括索引、约束、触发器）

---

## 验证清单

在提交迁移变更前，确认以下检查项：

- [ ] 迁移文件命名正确（14位时间戳）
- [ ] up.sql 和 down.sql 成对存在
- [ ] **变更已同步到 `schema/LATEST.sql`** ⚠️
- [ ] LATEST.sql 内容与迁移文件一致
- [ ] 索引、约束、触发器都已同步
- [ ] 代码编译通过（`go build ./...`）
- [ ] 本地测试通过（全新安装 + 升级场景）

---

## 快速检查命令

### 检查 LATEST.sql 缺少的表

```bash
# 提取迁移文件中的表
grep -h "^CREATE TABLE" migrate/*.up.sql 2>/dev/null | sed 's/CREATE TABLE //' | sed 's/ (.*//' | sort -u > /tmp/migrate_tables.txt

# 提取 LATEST.sql 中的表
grep "^CREATE TABLE" schema/LATEST.sql | sed 's/CREATE TABLE //' | sed 's/ (.*//' | sort > /tmp/latest_tables.txt

# 找出差异
diff /tmp/migrate_tables.txt /tmp/latest_tables.txt
```

### 统计当前表数量

```bash
echo "LATEST.sql 表数量: $(grep -c '^CREATE TABLE' schema/LATEST.sql)"
echo "迁移文件表数量: $(grep -h '^CREATE TABLE' migrate/*.up.sql 2>/dev/null | wc -l)"
```

---

## 相关文档

| 文档 | 描述 |
|:-----|:-----|
| `../../store/migrator.go` | 迁移系统实现代码 |
| `../../../docs/dev-guides/BACKEND_DB.md` | 后端与数据库指南 |
| `../../../docs/research/DEBUG_LESSONS.md` | 调试经验教训 |
