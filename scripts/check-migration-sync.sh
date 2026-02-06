#!/bin/bash
# 检查 LATEST.sql 与迁移文件的表同步状态

set -e

MIGRATE_DIR="store/migration/postgres/migrate"
LATEST_SQL="store/migration/postgres/schema/LATEST.sql"
TMP_DIR=$(mktemp -d)

echo "=== 检查数据库迁移同步状态 ==="
echo ""

# 提取迁移文件中的表（排除 IF NOT EXISTS）
grep -h "^CREATE TABLE" "$MIGRATE_DIR"/*.up.sql 2>/dev/null | \
    sed 's/CREATE TABLE IF NOT EXISTS //' | \
    sed 's/CREATE TABLE //' | \
    sed 's/ (.*//' | \
    sort -u > "$TMP_DIR/migrate_tables.txt"

# 提取 LATEST.sql 中的表
grep "^CREATE TABLE" "$LATEST_SQL" 2>/dev/null | \
    sed 's/CREATE TABLE //' | \
    sed 's/ (.*//' | \
    sort > "$TMP_DIR/latest_tables.txt"

# 找出 LATEST.sql 缺少的表
MISSING_TABLES=$(comm -13 "$TMP_DIR/latest_tables.txt" "$TMP_DIR/migrate_tables.txt" || true)

if [ -n "$MISSING_TABLES" ]; then
    echo "❌ LATEST.sql 缺少以下表："
    echo "$MISSING_TABLES" | while read -r table; do
        echo "   - $table"
    done
    echo ""
    echo "⚠️  请将缺失的表定义同步到 $LATEST_SQL"
    echo ""
    rm -rf "$TMP_DIR"
    exit 1
else
    echo "✅ LATEST.sql 与迁移文件表同步正常"
    echo ""
    echo "📊 统计信息："
    echo "   - 迁移文件表数量: $(wc -l < "$TMP_DIR/migrate_tables.txt" | tr -d ' ')"
    echo "   - LATEST.sql 表数量: $(wc -l < "$TMP_DIR/latest_tables.txt" | tr -d ' ')"
    echo ""
    rm -rf "$TMP_DIR"
    exit 0
fi
