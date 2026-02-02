#!/usr/bin/env bash
# Competitive Benchmark - Capability Scanning
#
# 扫描 DivineSense 项目的能力矩阵
#
# Usage:
#   ./scripts/benchmark/scan.sh          # 生成完整矩阵
#   ./scripts/benchmark/scan.sh parrots  # 扫描代理
#   ./scripts/benchmark/scan.sh tools    # 扫描工具

set -euo pipefail

# 项目根目录
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$PROJECT_ROOT"

# scan_parrots - 扫描 AI 代理数量
# 输出: 代理数量
scan_parrots() {
    find plugin/ai/agent -name "*_parrot.go" 2>/dev/null | wc -l | tr -d ' '
}

# scan_parrot_names - 扫描 AI 代理名称列表
# 输出: 代理名称（每行一个）
scan_parrot_names() {
    find plugin/ai/agent -name "*_parrot.go" 2>/dev/null | \
        sed 's/.*\///' | sed 's/_parrot.go//' | sort
}

# scan_tools - 扫描工具数量
# 输出: 工具数量
scan_tools() {
    find plugin/ai/agent/tools -name "*.go" 2>/dev/null | wc -l | tr -d ' '
}

# scan_tool_names - 扫描工具名称列表
# 输出: 工具名称（每行一个）
scan_tool_names() {
    find plugin/ai/agent/tools -name "*.go" 2>/dev/null | \
        sed 's/.*\///' | sed 's/\.go$//' | sort
}

# scan_pages - 扫描前端页面数量
# 输出: 页面数量
scan_pages() {
    find web/src/pages -name "*.tsx" 2>/dev/null | wc -l | tr -d ' '
}

# scan_page_names - 扫描前端页面名称列表
# 输出: 页面名称（每行一个）
scan_page_names() {
    find web/src/pages -name "*.tsx" 2>/dev/null | \
        sed 's/.*\///' | sed 's/\.tsx$//' | sort
}

# scan_tables - 扫描数据库表数量
# 输出: 表数量
scan_tables() {
    grep -r "CREATE TABLE" store/migration/postgres/ 2>/dev/null | \
        sed 's/.*CREATE TABLE IF NOT EXISTS //' | sed 's/ .*//' | sort -u | wc -l | tr -d ' '
}

# scan_table_names - 扫描数据库表名称列表
# 输出: 表名称（每行一个）
scan_table_names() {
    grep -r "CREATE TABLE" store/migration/postgres/ 2>/dev/null | \
        sed 's/.*CREATE TABLE IF NOT EXISTS //' | sed 's/ .*//' | sort -u
}

# has_feature - 检查是否已实现某功能
# 参数: $1 - 搜索模式 (正则或关键词)
# 输出: 0 (已实现) 或 1 (未实现)
has_feature() {
    local pattern=$1
    grep -rq "$pattern" plugin/ai/agent/ web/src/ 2>/dev/null
}

# generate_matrix - 生成完整能力矩阵 (JSON)
# 输出: JSON 格式的能力矩阵
generate_matrix() {
    local timestamp=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
    local ds_sha=$(git rev-parse HEAD 2>/dev/null || echo "unknown")

    cat <<EOF
{
  "parrots": $(scan_parrots),
  "tools": $(scan_tools),
  "pages": $(scan_pages),
  "tables": $(scan_tables),
  "divinesense_sha": "$ds_sha",
  "timestamp": "$timestamp"
}
EOF
}

# show_summary - 显示人类可读的能力摘要
show_summary() {
    echo "DivineSense 能力矩阵:"
    echo "  AI 代理: $(scan_parrots) 个"
    echo "  工具: $(scan_tools) 个"
    echo "  前端页面: $(scan_pages) 个"
    echo "  数据库表: $(scan_tables) 个"
    echo ""
    echo "AI 代理列表:"
    scan_parrot_names | sed 's/^/  - /'
    echo ""
    echo "工具列表:"
    scan_tool_names | sed 's/^/  - /'
}

# 主命令路由
case "${1:-matrix}" in
    parrots)
        scan_parrots
        ;;
    parrot-names)
        scan_parrot_names
        ;;
    tools)
        scan_tools
        ;;
    tool-names)
        scan_tool_names
        ;;
    pages)
        scan_pages
        ;;
    page-names)
        scan_page_names
        ;;
    tables)
        scan_tables
        ;;
    table-names)
        scan_table_names
        ;;
    has|has-feature)
        if has_feature "$2"; then
            echo "✅ 已实现: $2"
            exit 0
        else
            echo "❌ 未实现: $2"
            exit 1
        fi
        ;;
    summary)
        show_summary
        ;;
    matrix|json|"")
        generate_matrix
        ;;
    *)
        echo "Usage: $0 {parrots|tools|pages|tables|has|summary|matrix}" >&2
        echo "" >&2
        echo "Commands:" >&2
        echo "  parrots        - 扫描 AI 代理数量" >&2
        echo "  parrot-names   - 列出 AI 代理名称" >&2
        echo "  tools          - 扫描工具数量" >&2
        echo "  tool-names     - 列出工具名称" >&2
        echo "  pages          - 扫描前端页面数量" >&2
        echo "  page-names     - 列出页面名称" >&2
        echo "  tables         - 扫描数据库表数量" >&2
        echo "  table-names    - 列出表名称" >&2
        echo "  has <pattern>  - 检查是否实现某功能" >&2
        echo "  summary        - 显示人类可读摘要" >&2
        echo "  matrix         - 输出 JSON 格式能力矩阵" >&2
        exit 1
        ;;
esac
