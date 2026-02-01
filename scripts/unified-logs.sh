#!/bin/bash
# DivineSense 统一日志查看器
# 用法: ./scripts/unified-logs.sh [backend|frontend|all] [--level=DEBUG|INFO|WARN|ERROR] [-f]

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
MAGENTA='\033[0;35m'
CYAN='\033[0;36m'
GRAY='\033[0;90m'
NC='\033[0m'

# 项目目录
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
LOG_DIR="$ROOT_DIR/.logs"
BACKEND_LOG="$LOG_DIR/backend.log"
FRONTEND_LOG="$LOG_DIR/frontend.log"

# 参数解析
SERVICE="${1:-all}"
LOG_LEVEL=""
FOLLOW=false

shift 2>/dev/null || true

while [[ $# -gt 0 ]]; do
    case "$1" in
        --level=*)
            LOG_LEVEL="${1#*=}"
            ;;
        -f|--follow)
            FOLLOW=true
            ;;
        *)
            ;;
    esac
    shift
done

# 日志级别映射
get_level_color() {
    local level="$1"
    case "$level" in
        DEBUG|TRACE) echo "$GRAY" ;;
        INFO) echo "$GREEN" ;;
        WARN|WARNING) echo "$YELLOW" ;;
        ERROR|FATAL) echo "$RED" ;;
        *) echo "$NC" ;;
    esac
}

get_level_short() {
    local level="$1"
    case "$level" in
        DEBUG|TRACE) echo "D" ;;
        INFO) echo "I" ;;
        WARN|WARNING) echo "W" ;;
        ERROR|FATAL) echo "E" ;;
        *) echo "?" ;;
    esac
}

# 过滤日志级别
filter_log_level() {
    local input="$1"
    if [ -n "$LOG_LEVEL" ]; then
        case "$LOG_LEVEL" in
            DEBUG)
                echo "$input"
                ;;
            INFO)
                echo "$input" | grep -v -E "\[DEBUG\]|\[TRACE\]" || true
                ;;
            WARN)
                echo "$input" | grep -E "\[WARN\]|\[ERROR\]|\[FATAL\]" || true
                ;;
            ERROR)
                echo "$input" | grep -E "\[ERROR\]|\[FATAL\]" || true
                ;;
        esac
    else
        echo "$input"
    fi
}

# 格式化后端日志
format_backend_log() {
    local line="$1"
    local color="$NC"
    local level="?"
    local timestamp=""
    local message=""

    # 尝试解析结构化日志
    if echo "$line" | grep -q "level="; then
        # slog 格式: level=INFO msg=...
        level=$(echo "$line" | grep -oP 'level=\K\w+' || echo "?")
        timestamp=$(echo "$line" | grep -oP '\d{4}/\d{2}/\d{2} \d{2}:\d{2}:\d{2}' || echo "")
        message=$(echo "$line" | sed -E 's/.*level='"$level"' //; s/ method=[^ ]*//; s/ duration_ms=[0-9]+//')
    elif echo "$line" | grep -q "\[INFO\]\|\[DEBUG\]\|\[WARN\]\|\[ERROR\]"; then
        # 自定义格式
        level=$(echo "$line" | grep -oP '\[(DEBUG|INFO|WARN|ERROR)\]' | tr -d '[]' | head -1 || echo "?")
        message="$line"
    else
        message="$line"
    fi

    color=$(get_level_color "$level")
    level_short=$(get_level_short "$level")

    echo -e "${CYAN}[BE]${NC} ${color}[${level_short}]${NC} ${message}"
}

# 格式化前端日志
format_frontend_log() {
    local line="$1"
    local color="$NC"
    local level="?"

    # Vite 日志格式
    if echo "$line" | grep -q "hmr update"; then
        level="INFO"
        message="$line"
    elif echo "$line" | grep -q "error"; then
        level="ERROR"
        message="$line"
    elif echo "$line" | grep -q "warning"; then
        level="WARN"
        message="$line"
    else
        message="$line"
    fi

    color=$(get_level_color "$level")
    level_short=$(get_level_short "$level")

    echo -e "${MAGENTA}[FE]${NC} ${color}[${level_short}]${NC} ${message}"
}

# 显示日志
show_logs() {
    if [ "$FOLLOW" = "true" ]; then
        # 实时跟踪模式
        if [ "$SERVICE" = "all" ]; then
            # 使用 tail -f 同时监控两个文件
            tail -f "$BACKEND_LOG" 2>/dev/null | while read -r line; do
                formatted=$(format_backend_log "$line")
                filtered=$(filter_log_level "$formatted")
                [ -n "$filtered" ] && echo "$filtered"
            done &
            BACKEND_PID=$!

            tail -f "$FRONTEND_LOG" 2>/dev/null | while read -r line; do
                formatted=$(format_frontend_log "$line")
                filtered=$(filter_log_level "$formatted")
                [ -n "$filtered" ] && echo "$filtered"
            done &
            FRONTEND_PID=$!

            trap "kill $BACKEND_PID $FRONTEND_PID 2>/dev/null; exit 0" INT TERM

            wait
        elif [ "$SERVICE" = "backend" ]; then
            tail -f "$BACKEND_LOG" 2>/dev/null | while read -r line; do
                formatted=$(format_backend_log "$line")
                filtered=$(filter_log_level "$formatted")
                [ -n "$filtered" ] && echo "$filtered"
            done
        elif [ "$SERVICE" = "frontend" ]; then
            tail -f "$FRONTEND_LOG" 2>/dev/null | while read -r line; do
                formatted=$(format_frontend_log "$line")
                filtered=$(filter_log_level "$formatted")
                [ -n "$filtered" ] && echo "$filtered"
            done
        fi
    else
        # 静态查看模式 (最后 50 行)
        if [ "$SERVICE" = "all" ] || [ "$SERVICE" = "backend" ]; then
            if [ -f "$BACKEND_LOG" ]; then
                echo -e "${CYAN}=== 后端日志 (最近 50 行) ===${NC}"
                tail -50 "$BACKEND_LOG" 2>/dev/null | while read -r line; do
                    formatted=$(format_backend_log "$line")
                    filtered=$(filter_log_level "$formatted")
                    [ -n "$filtered" ] && echo "$filtered"
                done
                echo ""
            fi
        fi

        if [ "$SERVICE" = "all" ] || [ "$SERVICE" = "frontend" ]; then
            if [ -f "$FRONTEND_LOG" ]; then
                echo -e "${MAGENTA}=== 前端日志 (最近 50 行) ===${NC}"
                tail -50 "$FRONTEND_LOG" 2>/dev/null | while read -r line; do
                    formatted=$(format_frontend_log "$line")
                    filtered=$(filter_log_level "$formatted")
                    [ -n "$filtered" ] && echo "$filtered"
                done
            fi
        fi
    fi
}

# 显示使用帮助
show_help() {
    cat << EOF
DivineSense 统一日志查看器

用法: $0 [service] [options]

服务:
  backend    仅查看后端日志
  frontend   仅查看前端日志
  all        查看所有日志 (默认)

选项:
  --level=LEVEL    过滤日志级别 (DEBUG|INFO|WARN|ERROR)
  -f, --follow     实时跟踪日志

示例:
  $0                    # 查看所有日志
  $0 backend -f        # 实时跟踪后端日志
  $0 --level=ERROR     # 仅显示错误级别日志
  $0 all --level=WARN  # 显示警告和错误日志

日志级别颜色:
  ${GREEN}INFO${NC}    - 一般信息
  ${YELLOW}WARN${NC}    - 警告信息
  ${RED}ERROR${NC}    - 错误信息
  ${GRAY}DEBUG${NC}    - 调试信息
EOF
}

# 检查日志文件是否存在
check_log_files() {
    local missing=false

    if [ "$SERVICE" = "all" ] || [ "$SERVICE" = "backend" ]; then
        if [ ! -f "$BACKEND_LOG" ]; then
            echo -e "${YELLOW}警告: 后端日志文件不存在: $BACKEND_LOG${NC}"
            missing=true
        fi
    fi

    if [ "$SERVICE" = "all" ] || [ "$SERVICE" = "frontend" ]; then
        if [ ! -f "$FRONTEND_LOG" ]; then
            echo -e "${YELLOW}警告: 前端日志文件不存在: $FRONTEND_LOG${NC}"
            missing=true
        fi
    fi

    if [ "$missing" = true" ]; then
        echo ""
        echo "提示: 请先启动服务: make start"
        exit 1
    fi
}

# 主逻辑
case "${SERVICE:-}" in
    -h|--help|help)
        show_help
        exit 0
        ;;
    backend|frontend|all)
        check_log_files
        show_logs
        ;;
    *)
        echo "错误: 未知服务 '$SERVICE'"
        echo ""
        show_help
        exit 1
        ;;
esac
