#!/bin/bash
#
# cleanup-sqlite-vec.sh
# 清理不需要的文件（基于官方 releases 方案）
#

set -e

echo "=== SQLite-Vec 集成清理脚本 ==="
echo ""
echo "基于官方 releases 方案，清理不需要的文件"
echo ""

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 1. 删除自行编译脚本
echo "🗑️  删除自行编译脚本..."
if [ -f "scripts/build-sqlite-vec-static.sh" ]; then
    rm -v scripts/build-sqlite-vec-static.sh
fi
if [ -f "scripts/build-sqlite-vec.sh" ]; then
    rm -v scripts/build-sqlite-vec.sh
fi

# 2. 删除自行编译的静态库
echo ""
echo "🗑️  删除自行编译的静态库..."
if [ -d "internal/sqlite-vec" ]; then
    rm -rv internal/sqlite-vec
fi
if [ -d ".lib" ]; then
    rm -rv .lib
fi

# 3. 删除废弃文档
echo ""
echo "🗑️  删除废弃文档..."
if [ -f "docs/research/SQLITE_VEC_COMPILE_TIME_DOWNLOAD.md" ]; then
    rm -v docs/research/SQLITE_VEC_COMPILE_TIME_DOWNLOAD.md
fi

# 4. 删除临时文件
echo ""
echo "🗑️  删除临时文件..."
rm -fv divinesense.db* 2>/dev/null || true
rm -fv web/divinesense.db 2>/dev/null || true
rm -fv store/db/sqlite/memo_embedding.go.backup 2>/dev/null || true

# 5. 归档研究文档
echo ""
echo "📦 归档研究文档..."
ARCHIVE_DIR="docs/archived/sqlite-vec-experiments"
mkdir -p "${ARCHIVE_DIR}"

if [ -f "CODE_REVIEW_SQLITE_AI.md" ]; then
    mv -v CODE_REVIEW_SQLITE_AI.md "${ARCHIVE_DIR}/"
fi
if [ -f "FIX_REPORT_SQLITE_AI.md" ]; then
    mv -v FIX_REPORT_SQLITE_AI.md "${ARCHIVE_DIR}/"
fi
if [ -f "docs/research/MULTIPLATFORM_SQLITE_VEC.md" ]; then
    mv -v docs/research/MULTIPLATFORM_SQLITE_VEC.md "${ARCHIVE_DIR}/"
fi
if [ -f "docs/research/STATIC_LINKING_IMPLEMENTATION.md" ]; then
    mv -v docs/research/STATIC_LINKING_IMPLEMENTATION.md "${ARCHIVE_DIR}/"
fi

# 6. 创建 README
echo ""
echo "📝 创建归档说明..."
cat > "${ARCHIVE_DIR}/README.md" << 'EOF'
# SQLite-Vec 实验性功能归档

本目录包含 DivineSense 在探索 SQLite AI 支持过程中的实验性文档和方案。

## 最终方案

我们采用了 **sqlite-vec 官方 releases + go generate** 的方案：

- **文档**: `docs/research/SQLITE_VEC_OFFICIAL_RELEASES.md`
- **实现**: `store/db/sqlite/sqlite_vec_internal.go`
- **下载**: `store/db/sqlite/download_sqlite_vec.sh`

## 归档内容

### 代码审查报告
- `CODE_REVIEW_SQLITE_AI.md` - 代码审查发现的 P0/P1 问题

### 修复报告
- `FIX_REPORT_SQLITE_AI.md` - 详细修复记录

### 多平台编译研究
- `MULTIPLATFORM_SQLITE_VEC.md` - 多平台编译分析

### 静态链接实现
- `STATIC_LINKING_IMPLEMENTATION.md` - 静态链接方案实验

## 历史背景

1. **初始方案**: 尝试自行编译静态库（已废弃）
2. **中间方案**: init() 编译时下载（已废弃）
3. **最终方案**: 使用官方 releases + go generate（✅ 采用）

## 为什么这些方案被废弃？

1. **自行编译**: 需要维护多平台编译脚本和静态库，成本高
2. **init() 下载**: CGO 编译阶段问题，init() 执行时机晚于 cgo 指令处理
3. **官方 releases**: 简单可靠，由 sqlite-vec 官方维护

---

**归档时间**: 2026-02-06
**最终方案**: docs/research/SQLITE_VEC_OFFICIAL_RELEASES.md
EOF

echo ""
echo "✅ 清理完成！"
echo ""
echo "剩余文件："
echo "  ✅ 核心集成: store/db/sqlite/sqlite_vec_*.go"
echo "  ✅ AI 功能: store/db/sqlite/memo_embedding.go"
echo "  ✅ 文档: docs/research/SQLITE_VEC_OFFICIAL_RELEASES.md"
echo "  📦 归档: ${ARCHIVE_DIR}/"
echo ""
echo "下一步："
echo "  1. 运行 'go generate -v ./store/db/sqlite/...' 下载静态库"
echo "  2. 更新 Makefile 删除 build-sqlite-vec-* 命令"
echo "  3. 更新 build-release.sh 使用 go generate 方案"
echo "  4. 提交代码（建议拆分为 3 个 commit）"
