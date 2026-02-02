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

# 脚本目录
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# 项目根目录
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

# 验证项目根目录
validate_project_root() {
    if [[ ! -f "$PROJECT_ROOT/go.mod" ]] || [[ ! -d "$PROJECT_ROOT/plugin/ai" ]]; then
        echo "错误: 必须在 DivineSense 项目根目录下运行" >&2
        exit 1
    fi
}

validate_project_root

# 切换到项目根目录
cd "$PROJECT_ROOT" || { echo "错误: 无法切换到项目目录 $PROJECT_ROOT" >&2; exit 1; }

# 路径常量（P2: 提取硬编码路径）
readonly AGENT_DIR="plugin/ai/agent"
readonly TOOLS_DIR="$AGENT_DIR/tools"
readonly PAGES_DIR="web/src/pages"
readonly MIGRATION_DIR="store/migration/postgres"

# 日志函数
log_info() { echo "[$(date -u +"%Y-%m-%d %H:%M:%S")][INFO] $*" >&2; }
log_error() { echo "[$(date -u +"%Y-%m-%d %H:%M:%S")][ERROR] $*" >&2; }

# 扫描结果缓存（P2: 避免重复扫描）
declare -A SCAN_CACHE

# scan_parrots - 扫描 AI 代理数量
# 支持两种命名模式：
#   - *_parrot.go (标准 Parrot)
#   - *_parrot_v2.go (V2 变体，如 schedule_parrot_v2.go)
# 输出: 代理数量
scan_parrots() {
    if [[ -z "${SCAN_CACHE[parrots]:-}" ]]; then
        # 扫描标准 Parrot 和 V2 变体
        SCAN_CACHE[parrots]=$(
            (
                find "$AGENT_DIR" -name "*_parrot.go" 2>/dev/null
                find "$AGENT_DIR" -name "*_parrot_v2.go" 2>/dev/null
            ) | wc -l | tr -d ' '
        )
    fi
    echo "${SCAN_CACHE[parrots]}"
}

# scan_parrot_names - 扫描 AI 代理名称列表
# 输出: 代理名称（每行一个）
scan_parrot_names() {
    if [[ -z "${SCAN_CACHE[parrot_names]:-}" ]]; then
        SCAN_CACHE[parrot_names]=$(
            (
                find "$AGENT_DIR" -name "*_parrot.go" -o -name "*_parrot_v2.go" 2>/dev/null
            ) | sed 's/.*\///' | \
            sed 's/_parrot_v2\.go$//' | sed 's/_parrot\.go$//' | sort -u
        )
    fi
    echo "${SCAN_CACHE[parrot_names]}"
}

# scan_tools - 扫描工具数量
# 输出: 工具数量
scan_tools() {
    if [[ -z "${SCAN_CACHE[tools]:-}" ]]; then
        # P2: 排除测试文件
        SCAN_CACHE[tools]=$(find "$TOOLS_DIR" -name "*.go" -not -name "*_test.go" 2>/dev/null | wc -l | tr -d ' ')
    fi
    echo "${SCAN_CACHE[tools]}"
}

# scan_tool_names - 扫描工具名称列表
# 输出: 工具名称（每行一个）
scan_tool_names() {
    if [[ -z "${SCAN_CACHE[tool_names]:-}" ]]; then
        SCAN_CACHE[tool_names]=$(find "$TOOLS_DIR" -name "*.go" -not -name "*_test.go" 2>/dev/null | \
            sed 's/.*\///' | sed 's/\.go$//' | sort)
    fi
    echo "${SCAN_CACHE[tool_names]}"
}

# scan_pages - 扫描前端页面数量
# 输出: 页面数量
scan_pages() {
    if [[ -z "${SCAN_CACHE[pages]:-}" ]]; then
        SCAN_CACHE[pages]=$(find "$PAGES_DIR" -name "*.tsx" 2>/dev/null | wc -l | tr -d ' ')
    fi
    echo "${SCAN_CACHE[pages]}"
}

# scan_page_names - 扫描前端页面名称列表
# 输出: 页面名称（每行一个）
scan_page_names() {
    if [[ -z "${SCAN_CACHE[page_names]:-}" ]]; then
        SCAN_CACHE[page_names]=$(find "$PAGES_DIR" -name "*.tsx" 2>/dev/null | \
            sed 's/.*\///' | sed 's/\.tsx$//' | sort)
    fi
    echo "${SCAN_CACHE[page_names]}"
}

# scan_tables - 扫描数据库表数量
# 输出: 表数量
scan_tables() {
    if [[ -z "${SCAN_CACHE[tables]:-}" ]]; then
        # P1: 添加 -- 分隔符防止参数注入
        SCAN_CACHE[tables]=$(grep -r "CREATE TABLE" -- "$MIGRATION_DIR/" 2>/dev/null | \
            sed 's/.*CREATE TABLE IF NOT EXISTS //' | sed 's/ .*//' | sort -u | wc -l | tr -d ' ')
    fi
    echo "${SCAN_CACHE[tables]}"
}

# scan_table_names - 扫描数据库表名称列表
# 输出: 表名称（每行一个）
scan_table_names() {
    if [[ -z "${SCAN_CACHE[table_names]:-}" ]]; then
        SCAN_CACHE[table_names]=$(grep -r "CREATE TABLE" -- "$MIGRATION_DIR/" 2>/dev/null | \
            sed 's/.*CREATE TABLE IF NOT EXISTS //' | sed 's/ .*//' | sort -u)
    fi
    echo "${SCAN_CACHE[table_names]}"
}

# has_feature - 检查是否已实现某功能
# 参数: $1 - 搜索模式 (固定字符串，非正则)
# 输出: 0 (已实现) 或 1 (未实现)
has_feature() {
    local pattern=$1

    # P1: 参数验证
    if [[ -z "$pattern" ]]; then
        log_error "has_feature 需要提供搜索模式"
        return 1
    fi

    # P0: 验证输入只包含安全字符
    if [[ ! "$pattern" =~ ^[a-zA-Z0-9_./:-]+$ ]]; then
        log_error "搜索模式包含非法字符: $pattern"
        return 1
    fi

    # P0: 使用 grep -F 禁用正则，防止注入
    grep -rqF -- "$pattern" "$AGENT_DIR/" "web/src/" 2>/dev/null
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
        if [[ -z "${2:-}" ]]; then
            echo "错误: 'has' 命令需要搜索模式参数" >&2
            echo "" >&2
            echo "Usage: $0 has <pattern>" >&2
            echo "示例: $0 has \"session.*prun\"" >&2
            exit 1
        fi
        if has_feature "$2"; then
            # P3: 终端兼容性检查
            if [[ -t 1 && "$TERM" != "dumb" && -n "${TERM:-}" ]]; then
                echo "✅ 已实现: $2"
            else
                echo "[PASS] 已实现: $2"
            fi
            exit 0
        else
            if [[ -t 1 && "$TERM" != "dumb" && -n "${TERM:-}" ]]; then
                echo "❌ 未实现: $2"
            else
                echo "[FAIL] 未实现: $2"
            fi
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
