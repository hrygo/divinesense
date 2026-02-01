#!/bin/bash
# 后端嵌入完整性检查
# 验证 go:embed 引用的 dist/ 目录存在且包含必要文件

set -e

# 颜色
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DIST_DIR="$ROOT_DIR/web/dist"
EMBED_FILE="$ROOT_DIR/server/router/frontend/frontend_prod.go"

log_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

log_info "检查后端嵌入配置..."
echo ""

# 1. 检查 embed 指令是否存在
if [ ! -f "$EMBED_FILE" ]; then
    log_error "embed 文件不存在: $EMBED_FILE"
    exit 1
fi

if ! grep -q "//go:embed dist/" "$EMBED_FILE"; then
    log_error "未找到 //go:embed 指令"
    exit 1
fi
log_info "✓ embed 指令存在"

# 2. 检查 dist 目录是否存在
if [ ! -d "$DIST_DIR" ]; then
    log_error "dist 目录不存在: $DIST_DIR"
    log_info "请先运行: cd web && pnpm build"
    exit 1
fi
log_info "✓ dist 目录存在"

# 3. 检查关键文件是否存在
critical_files=(
    "$DIST_DIR/index.html"
    "$DIST_DIR/assets"
)

for file in "${critical_files[@]}"; do
    if [ ! -e "$file" ]; then
        log_error "关键文件不存在: $file"
        exit 1
    fi
done
log_info "✓ 关键文件存在"

# 4. 检查是否有 JS 和 CSS 文件
js_count=$(find "$DIST_DIR/assets" -name "*.js" 2>/dev/null | wc -l)
css_count=$(find "$DIST_DIR/assets" -name "*.css" 2>/dev/null | wc -l)

if [ "$js_count" -eq 0 ]; then
    log_error "未找到 JS 文件，构建可能不完整"
    exit 1
fi
if [ "$css_count" -eq 0 ]; then
    log_error "未找到 CSS 文件，构建可能不完整"
    exit 1
fi

log_info "✓ 找到 $js_count 个 JS 文件, $css_count 个 CSS 文件"

# 5. 检查构建标签
if ! grep -q "//go:build !noui" "$EMBED_FILE"; then
    log_warn "embed 文件缺少构建标签条件，可能影响测试"
else
    log_info "✓ 构建标签正确 (!noui)"
fi

echo ""
log_info "后端嵌入完整性检查通过! ✅"
exit 0
