#!/bin/bash
# 生成构建产物的 SHA256 校验和
# 用于验证生产构建的完整性

set -e

# 颜色
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
CHECKSUM_FILE="$ROOT_DIR/.checksums"
BIN_DIR="$ROOT_DIR/bin"
DIST_DIR="$ROOT_DIR/web/dist"

log_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${YELLOW}[ERROR]${NC} $1"; }

# 检查 shasum 命令
if command -v shasum &>/dev/null; then
    SHASUM_CMD="shasum -a 256"
elif command -v sha256sum &>/dev/null; then
    SHASUM_CMD="sha256sum"
else
    log_error "未找到 shasum 或 sha256sum 命令"
    exit 1
fi

log_info "生成构建产物校验和..."
echo ""

# 清空或创建校验和文件
: > "$CHECKSUM_FILE"

# 1. 二进制文件校验和
if [ -f "$BIN_DIR/divinesense" ]; then
    checksum=$($SHASUM_CMD "$BIN_DIR/divinesense" | awk '{print $1}')
    size=$(stat -f%z "$BIN_DIR/divinesense" 2>/dev/null || stat -c%s "$BIN_DIR/divinesense" 2>/dev/null)
    echo "bin/divinesense|$checksum|$size" >> "$CHECKSUM_FILE"
    log_info "✓ bin/divinesense ($size bytes)"
else
    log_warn "二进制文件不存在: $BIN_DIR/divinesense"
fi

# 2. 前端构建产物校验和 (关键文件)
if [ -d "$DIST_DIR" ]; then
    # index.html
    if [ -f "$DIST_DIR/index.html" ]; then
        checksum=$($SHASUM_CMD "$DIST_DIR/index.html" | awk '{print $1}')
        echo "web/dist/index.html|$checksum" >> "$CHECKSUM_FILE"
        log_info "✓ web/dist/index.html"
    fi

    # 主 JS 文件
    main_js=$(find "$DIST_DIR/assets" -name "index-*.js" -o -name "polyfills-*.js" 2>/dev/null | head -5)
    for js in $main_js; do
        checksum=$($SHASUM_CMD "$js" | awk '{print $1}')
        rel_path="web/dist/${js#$DIST_DIR/}"
        echo "$rel_path|$checksum" >> "$CHECKSUM_FILE"
        log_info "✓ $rel_path"
    done

    # 主 CSS 文件
    main_css=$(find "$DIST_DIR/assets" -name "index-*.css" 2>/dev/null)
    for css in $main_css; do
        checksum=$($SHASUM_CMD "$css" | awk '{print $1}')
        rel_path="web/dist/${css#$DIST_DIR/}"
        echo "$rel_path|$checksum" >> "$CHECKSUM_FILE"
        log_info "✓ $rel_path"
    done

    # 统计 assets 数量
    js_count=$(find "$DIST_DIR/assets" -name "*.js" 2>/dev/null | wc -l)
    css_count=$(find "$DIST_DIR/assets" -name "*.css" 2>/dev/null | wc -l)
    log_info "前端资源: $js_count JS files, $css_count CSS files"
else
    log_warn "前端构建目录不存在: $DIST_DIR"
fi

# 添加元数据
echo "" >> "$CHECKSUM_FILE"
echo "# Build Metadata" >> "$CHECKSUM_FILE"
echo "timestamp=$(date -u +"%Y-%m-%dT%H:%M:%SZ")" >> "$CHECKSUM_FILE"
echo "git_commit=$(git rev-parse HEAD 2>/dev/null || echo "unknown")" >> "$CHECKSUM_FILE"
echo "git_branch=$(git branch --show-current 2>/dev/null || echo "unknown")" >> "$CHECKSUM_FILE"

echo ""
log_info "校验和已保存到: $CHECKSUM_FILE"
echo ""
echo "验证命令:"
echo "  $SHASUM_CMD -c $CHECKSUM_FILE"
