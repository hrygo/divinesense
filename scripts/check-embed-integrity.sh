#!/bin/bash
# 前端嵌入完整性检查
# 验证 web/dist/ 中的所有资源都被正确引用，且所有引用的文件都存在

set -e

# 颜色
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DIST_DIR="$ROOT_DIR/web/dist"

log_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

# 检查 dist 目录是否存在
if [ ! -d "$DIST_DIR" ]; then
    log_error "dist 目录不存在: $DIST_DIR"
    log_info "请先运行: cd web && pnpm build"
    exit 1
fi

# 提取 index.html 中引用的所有资源
INDEX_FILE="$DIST_DIR/index.html"
if [ ! -f "$INDEX_FILE" ]; then
    log_error "index.html 不存在: $INDEX_FILE"
    exit 1
fi

log_info "检查前端构建完整性..."
echo ""

# 提取 index.html 中引用的所有资源 (兼容 macOS grep)
# 使用 sed 提取 href 和 src 属性
referenced_files=$(sed -n 's/.*href="\([^"]*\.[js][^"]*\)".*/\1/p; s/.*src="\([^"]*\.[js][^"]*\)".*/\1/p' "$INDEX_FILE" | sort -u)

missing_count=0
total_count=0

for file in $referenced_files; do
    total_count=$((total_count + 1))
    full_path="$DIST_DIR/$file"

    if [ ! -f "$full_path" ]; then
        log_error "缺失文件: $file"
        missing_count=$((missing_count + 1))
    fi
done

# 检查是否有未使用的资源 (反向检查)
# 忽略一些特殊的文件
actual_files=$(find "$DIST_DIR" -type f \( -name "*.js" -o -name "*.css" \) | sed "s|^$DIST_DIR/||" | sort)

unused_count=0
for file in $actual_files; do
    # 跳过不需要在 index.html 中直接引用的文件
    if [[ "$file" =~ ^assets/index-.*\.js$ ]] || [[ "$file" =~ ^assets/.*-vendor.*\.js$ ]] || [[ "$file" =~ ^assets/.*-chunk.*\.js$ ]]; then
        continue
    fi

    # 检查是否被动态加载 (可能是懒加载的路由)
    if [[ "$file" =~ ^assets/[A-Z][a-zA-Z]+-.*\.js$ ]]; then
        # 这是动态导入的路由组件，检查是否在代码中被引用
        # 这里只做警告，不算错误
        if ! grep -q "$file" "$DIST_DIR"/assets/*.js 2>/dev/null; then
            log_warn "可能未使用的路由组件: $file"
        fi
    fi
done

echo ""
log_info "检查结果:"
echo "  总引用文件: $total_count"
echo "  缺失文件: $missing_count"

if [ $missing_count -gt 0 ]; then
    echo ""
    log_error "前端构建完整性检查失败!"
    echo "建议运行: cd web && rm -rf dist && pnpm build"
    exit 1
else
    log_info "前端构建完整性检查通过! ✅"
    exit 0
fi
