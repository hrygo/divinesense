#!/usr/bin/env bash
# Competitive Benchmark - State Management
#
# 管理对标状态文件的读写操作
#
# Usage:
#   source scripts/benchmark/state.sh
#   append_state "2026-02-02T10:00:00Z" "abc123" "def456" '["feat1"]' '[30]'
#   get_latest_state
#   query_state "openclaw_sha"

set -euo pipefail

# 状态文件路径（可通过环境变量覆盖）
STATE_FILE="${STATE_FILE:-docs/research/benchmark/state.jsonl}"

# append_state - 追加新状态记录
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

    # 确保目录存在
    mkdir -p "$(dirname "$STATE_FILE")"

    # 追加状态记录
    cat >> "$STATE_FILE" <<EOF
{"timestamp":"$timestamp","openclaw_sha":"$oc_sha","divinesense_sha":"$ds_sha","analyzed_features":$features,"discovered_functions":[],"created_issues":$issues}
EOF
}

# get_latest_state - 获取最新状态记录
# 输出: JSON 格式的状态记录
get_latest_state() {
    if [ -f "$STATE_FILE" ] && [ -s "$STATE_FILE" ]; then
        tail -1 "$STATE_FILE" | jq -r '.'
    else
        # 返回空状态结构
        echo '{"timestamp":"","openclaw_sha":"","divinesense_sha":"","analyzed_features":[],"discovered_functions":[],"created_issues":[]}'
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
        return
    fi

    local last_run=$(query_state "timestamp")
    local oc_sha=$(query_state "openclaw_sha")
    local analyzed_count=$(get_latest_state | jq -r '.analyzed_features | length')
    local issues_count=$(get_latest_state | jq -r '.created_issues | length')

    echo "对标状态摘要:"
    echo "  上次对标: ${last_run:-无}"
    echo "  OpenClaw SHA: ${oc_sha:-无}"
    echo "  已分析功能: ${analyzed_count:-0} 个"
    echo "  已创建 Issue: ${issues_count:-0} 个"
}

# 如果直接执行脚本（非 source），允许子命令调用
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    case "${1:-}" in
        append)
            shift
            append_state "$@"
            ;;
        get|latest)
            get_latest_state
            ;;
        query)
            query_state "$2"
            ;;
        summary)
            show_state_summary
            ;;
        count)
            get_state_count
            ;;
        *)
            echo "Usage: $0 {append|get|query|summary|count}" >&2
            echo "  append <timestamp> <oc_sha> <ds_sha> <features> <issues>" >&2
            echo "  get/latest  - 获取最新状态" >&2
            echo "  query <field> - 查询状态字段" >&2
            echo "  summary     - 显示状态摘要" >&2
            echo "  count       - 获取记录数量" >&2
            exit 1
            ;;
    esac
fi
