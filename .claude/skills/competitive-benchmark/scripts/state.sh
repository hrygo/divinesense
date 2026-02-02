#!/usr/bin/env bash
# Competitive Benchmark - State Management
#
# 管理对标状态文件的读写操作
#
# Usage:
#   source state.sh
#   append_state "2026-02-02T10:00:00Z" "abc123" "def456' '["feat1"]' '[30]'
#   get_latest_state
#   query_state "openclaw_sha"
#
# Or direct execution:
#   ./state.sh append "..." "..." "..." '[]' '[]'
#   ./state.sh get
#   ./state.sh query openclaw_sha
#   ./state.sh summary
#   ./state.sh init

set -euo pipefail

# Bash 版本说明
# 此脚本兼容 Bash 3.2+（不使用关联数组等 Bash 4 特性）

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

# 默认状态文件路径（相对于项目根目录）
DEFAULT_STATE_FILE="$PROJECT_ROOT/docs/research/benchmark/state.jsonl"

# 规范化路径并验证在项目目录内
validate_state_path() {
    local input_path="${1:-$DEFAULT_STATE_FILE}"

    # 支持相对路径
    if [[ "$input_path" != /* ]]; then
        input_path="$PROJECT_ROOT/$input_path"
    fi

    # 规范化路径
    local target_path
    target_path="$(cd "$PROJECT_ROOT" && realpath -m "$input_path" 2>/dev/null || echo "$input_path")"

    # 验证路径在项目目录内
    if [[ "$target_path" != "$PROJECT_ROOT"/* ]]; then
        echo "错误: 状态文件必须在项目目录内" >&2
        exit 1
    fi

    STATE_FILE="$target_path"
}

# 初始化 STATE_FILE
validate_state_path "${STATE_FILE:-}"

# 检查依赖
check_dependencies() {
    local missing=()

    for cmd in jq; do
        if ! command -v "$cmd" >/dev/null 2>&1; then
            missing+=("$cmd")
        fi
    done

    if [ ${#missing[@]} -gt 0 ]; then
        echo "错误: 缺少必需依赖: ${missing[*]}" >&2
        echo "请安装: brew install jq / apt install jq" >&2
        exit 1
    fi
}

check_dependencies

# 空状态结构常量
EMPTY_STATE='{"timestamp":"","openclaw_sha":"","divinesense_sha":"","analyzed_features":[],"discovered_functions":[],"created_issues":[]}'

# 日志函数
log_info() { echo "[$(date -u +"%Y-%m-%d %H:%M:%S")][INFO] $*" >&2; }
log_error() { echo "[$(date -u +"%Y-%m-%d %H:%M:%S")][ERROR] $*" >&2; }
log_warn() { echo "[$(date -u +"%Y-%m-%d %H:%M:%S")][WARN] $*" >&2; }

# 验证 ISO 8601 时间戳格式
validate_timestamp() {
    local ts=$1
    if ! [[ "$ts" =~ ^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}(Z|[\+\-][0-9]{2}:[0-9]{2})$ ]]; then
        log_error "无效的时间戳格式: $ts (期望 ISO 8601, 如 2026-02-02T10:00:00Z)"
        return 1
    fi
}

# 验证 Git SHA 格式
validate_sha() {
    local sha=$1
    local name=$2
    if [[ -z "$sha" ]]; then
        log_error "$name 不能为空"
        return 1
    fi
    if [[ ! "$sha" =~ ^[a-fA-F0-9]{7,64}$ ]]; then
        log_error "无效的 SHA 格式: $sha"
        return 1
    fi
}

# 验证 JSON 数组格式
validate_json_array() {
    local input=$1
    local name=$2

    if [[ -z "$input" ]]; then
        log_error "$name 不能为空"
        return 1
    fi

    if ! echo "$input" | jq -e '. | if type == "array" then true else false end' >/dev/null 2>&1; then
        log_error "$name 必须是有效的 JSON 数组: $input"
        return 1
    fi
}

# append_state - 追加新状态记录（使用文件锁保证并发安全）
# 参数:
#   $1 - timestamp (ISO 8601)
#   $2 - openclaw_sha
#   $3 - divinesense_sha
#   $4 - analyzed_features (JSON array)
#   $5 - created_issues (JSON array)
append_state() {
    local timestamp=$1
    local oc_sha=$2
    local ds_sha=$3
    local features=$4
    local issues=$5

    # 验证输入
    validate_timestamp "$timestamp" || return 1
    validate_sha "$oc_sha" "openclaw_sha" || return 1
    validate_sha "$ds_sha" "divinesense_sha" || return 1
    validate_json_array "$features" "analyzed_features" || return 1
    validate_json_array "$issues" "created_issues" || return 1

    # 确保目录存在
    mkdir -p "$(dirname "$STATE_FILE")"

    # 使用文件锁保证并发安全
    local lock_file="$STATE_FILE.lock"
    local max_wait=10

    if ! (
        flock -w "$max_wait" 9 || {
            log_error "无法获取文件锁（超时 ${max_wait}s）"
            exit 1
        }

        # 使用 jq 安全构建 JSON（避免注入）
        jq -n \
            --arg ts "$timestamp" \
            --arg oc "$oc_sha" \
            --arg ds "$ds_sha" \
            --argjson features "$features" \
            --argjson issues "$issues" \
            '{timestamp: $ts, openclaw_sha: $oc, divinesense_sha: $ds,
              analyzed_features: $features, discovered_functions: [],
              created_issues: $issues}' >> "$STATE_FILE"

        log_info "状态已追加: oc_sha=$oc_sha, features=$(
            echo "$features" | jq 'length'
        ) 个, issues=$(
            echo "$issues" | jq 'length'
        ) 个"
    ) 9>"$lock_file"; then
        log_error "追加状态失败"
        return 1
    fi

    rm -f "$lock_file"
}

# get_latest_state - 获取最新状态记录
# 输出: JSON 格式的状态记录
get_latest_state() {
    if [ -f "$STATE_FILE" ] && [ -s "$STATE_FILE" ]; then
        local last_line
        last_line=$(tail -1 "$STATE_FILE")

        # 验证 JSON 格式
        if echo "$last_line" | jq -e '.' >/dev/null 2>&1; then
            echo "$last_line" | jq -r '.'
        else
            log_error "状态文件最后一条记录格式无效"
            echo "$EMPTY_STATE"
            return 1
        fi
    else
        # 返回空状态结构
        echo "$EMPTY_STATE"
    fi
}

# query_state - 查询状态中的特定字段
# 参数:
#   $1 - 字段名 (如 openclaw_sha, timestamp)
# 输出: 字段值
query_state() {
    local field=$1
    get_latest_state | jq -r ".$field"
}

# state_exists - 检查状态文件是否存在且有内容
# 输出: 0 (存在) 或 1 (不存在)
state_exists() {
    [ -f "$STATE_FILE" ] && [ -s "$STATE_FILE" ]
}

# get_state_count - 获取状态记录数量
# 输出: 记录数量
get_state_count() {
    if state_exists; then
        wc -l < "$STATE_FILE" | tr -d ' '
    else
        echo 0
    fi
}

# show_state_summary - 显示状态摘要
show_state_summary() {
    if ! state_exists; then
        echo "状态文件不存在，首次运行"
        echo "建议运行: ./.claude/skills/competitive-benchmark/scripts/benchmark.sh init"
        return
    fi

    local state_json
    if ! state_json=$(get_latest_state); then
        log_error "无法读取状态文件"
        return 1
    fi

    local last_run=$(echo "$state_json" | jq -r '.timestamp')
    local oc_sha=$(echo "$state_json" | jq -r '.openclaw_sha')
    local ds_sha=$(echo "$state_json" | jq -r '.divinesense_sha')
    local analyzed_count=$(echo "$state_json" | jq -r '.analyzed_features | length')
    local issues_count=$(echo "$state_json" | jq -r '.created_issues | length')

    echo "对标状态摘要:"
    echo "  上次对标: ${last_run:-无}"
    echo "  OpenClaw SHA: ${oc_sha:-无}"
    echo "  DivineSense SHA: ${ds_sha:-无}"
    echo "  已分析功能: ${analyzed_count:-0} 个"
    echo "  已创建 Issue: ${issues_count:-0} 个"
}

# init_state - 初始化状态文件（首次运行）
init_state() {
    if state_exists; then
        echo "状态文件已存在: $STATE_FILE"
        echo "如需重新初始化，请先删除现有文件"
        return 1
    fi

    mkdir -p "$(dirname "$STATE_FILE")"

    # 获取当前 SHA
    local current_sha
    current_sha=$(git rev-parse HEAD 2>/dev/null || echo "unknown")

    # 写入初始状态
    local timestamp=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

    jq -n \
        --arg ts "$timestamp" \
        --arg ds "$current_sha" \
        '{timestamp: $ts, openclaw_sha: "", divinesense_sha: $ds,
          analyzed_features: [], discovered_functions: [],
          created_issues: []}' > "$STATE_FILE"

    echo "状态文件已初始化: $STATE_FILE"
    echo "DivineSense SHA: $current_sha"
    echo "建议运行: ./.claude/skills/competitive-benchmark/scripts/benchmark.sh run"
}

# 如果直接执行脚本（非 source），允许子命令调用
if [ "$(basename "$0")" = "state.sh" ]; then
    validate_project_root

    case "${1:-}" in
        append)
            shift
            append_state "$@"
            ;;
        get|latest)
            get_latest_state
            ;;
        query)
            if [[ -z "${2:-}" ]]; then
                echo "错误: 'query' 命令需要字段名参数" >&2
                exit 1
            fi
            query_state "$2"
            ;;
        summary)
            show_state_summary
            ;;
        count)
            get_state_count
            ;;
        init)
            init_state
            ;;
        *)
            echo "Usage: $0 {append|get|query|summary|count|init}" >&2
            echo "" >&2
            echo "Commands:" >&2
            echo "  append <ts> <oc_sha> <ds_sha> <features> <issues>" >&2
            echo "  get/latest  - 获取最新状态" >&2
            echo "  query <field> - 查询状态字段" >&2
            echo "  summary     - 显示状态摘要" >&2
            echo "  count       - 获取记录数量" >&2
            echo "  init        - 初始化状态文件（首次运行）" >&2
            exit 1
            ;;
    esac
fi
