#!/bin/bash
# DivineSense 开发环境管理脚本
# 用法: ./scripts/dev.sh [start|stop|restart|status|logs]

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 项目根目录
# 项目根目录
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)"
cd "$ROOT_DIR"

# PID 文件目录
PID_DIR="$ROOT_DIR/.pids"
mkdir -p "$PID_DIR"

# 日志目录
LOG_DIR="$ROOT_DIR/.logs"
mkdir -p "$LOG_DIR"

# 服务配置
POSTGRES_CONTAINER="divinesense-postgres-dev"
BACKEND_PID_FILE="$PID_DIR/backend.pid"
FRONTEND_PID_FILE="$PID_DIR/frontend.pid"

# 端口配置
BACKEND_PORT=28081
FRONTEND_PORT=25173

# 日志文件
BACKEND_LOG="$LOG_DIR/backend.log"
FRONTEND_LOG="$LOG_DIR/frontend.log"

# ============================================================================
# 辅助函数
# ============================================================================

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[OK]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 检查端口是否被占用
check_port() {
    local port=$1
    if lsof -i ":$port" &>/dev/null; then
        return 0
    fi
    return 1
}

# 等待端口可用
wait_for_port() {
    local port=$1
    local service=$2
    local max_wait=${3:-30}
    local count=0

    while ! check_port "$port"; do
        if [ $count -ge $max_wait ]; then
            log_error "$service 启动超时"
            return 1
        fi
        sleep 1
        count=$((count + 1))
        echo -n "."
    done
    echo ""
    return 0
}

# 检查 Docker 是否运行
check_docker() {
    if ! docker info &>/dev/null; then
        log_error "Docker 未运行，请先启动 Docker"
        exit 1
    fi
}

# 加载 .env 文件
load_env() {
    if [ -f "$ROOT_DIR/.env" ]; then
        set -a
        source "$ROOT_DIR/.env"
        set +a
    fi
}

# ============================================================================
# 服务状态检查
# ============================================================================

postgres_status() {
    if docker ps --format '{{.Names}}' | grep -q "^${POSTGRES_CONTAINER}$"; then
        echo "running"
    elif docker ps -a --format '{{.Names}}' | grep -q "^${POSTGRES_CONTAINER}$"; then
        echo "stopped"
    else
        echo "not_found"
    fi
}

backend_status() {
    if [ -f "$BACKEND_PID_FILE" ]; then
        local pid=$(cat "$BACKEND_PID_FILE")
        if ps -p "$pid" &>/dev/null; then
            echo "running"
        else
            echo "stopped"
        fi
    else
        echo "not_found"
    fi
}

frontend_status() {
    if [ -f "$FRONTEND_PID_FILE" ]; then
        local pid=$(cat "$FRONTEND_PID_FILE")
        if ps -p "$pid" &>/dev/null; then
            echo "running"
        else
            echo "stopped"
        fi
    else
        echo "not_found"
    fi
}

# ============================================================================
# 启动服务
# ============================================================================

start_postgres() {
    local status=$(postgres_status)

    case $status in
        running)
            log_info "PostgreSQL 已在运行"
            return 0
            ;;
        stopped)
            log_info "启动 PostgreSQL..."
            docker compose -f docker/compose/dev.yml up -d
            ;;
        not_found)
            log_info "启动 PostgreSQL..."
            docker compose -f docker/compose/dev.yml up -d
            ;;
    esac

    # 等待 PostgreSQL 启动
    echo -n "等待 PostgreSQL 启动"
    if wait_for_port 25432 "PostgreSQL" 30; then
        log_success "PostgreSQL 已启动"
        return 0
    else
        log_error "PostgreSQL 启动失败"
        return 1
    fi
}

start_backend() {
    local status=$(backend_status)

    case $status in
        running)
            log_info "后端已在运行 (PID: $(cat $BACKEND_PID_FILE))"
            return 0
            ;;
    esac

    log_info "启动后端..."

    # 确保日志目录存在
    mkdir -p "$(dirname "$BACKEND_LOG")"

    # 加载环境变量
    load_env

    # 启动后端（后台运行）
    nohup go run -tags=noui ./cmd/divinesense --mode dev --port $BACKEND_PORT \
        > "$BACKEND_LOG" 2>&1 &

    local pid=$!
    echo $pid > "$BACKEND_PID_FILE"

    # 等待后端启动
    echo -n "等待后端启动"
    if wait_for_port $BACKEND_PORT "后端" 30; then
        log_success "后端已启动 (PID: $pid, http://localhost:$BACKEND_PORT)"
        return 0
    else
        log_error "后端启动失败，查看日志: $BACKEND_LOG"
        rm -f "$BACKEND_PID_FILE"
        return 1
    fi
}

start_frontend() {
    local status=$(frontend_status)

    case $status in
        running)
            log_info "前端已在运行 (PID: $(cat $FRONTEND_PID_FILE))"
            return 0
            ;;
    esac

    log_info "启动前端..."

    # 确保日志目录存在
    mkdir -p "$(dirname "$FRONTEND_LOG")"

    # 启动前端（后台运行）
    cd web
    nohup pnpm dev > "$FRONTEND_LOG" 2>&1 &
    cd ..

    local pid=$!
    echo $pid > "$FRONTEND_PID_FILE"

    # 等待前端启动
    echo -n "等待前端启动"
    if wait_for_port $FRONTEND_PORT "前端" 60; then
        log_success "前端已启动 (PID: $pid, http://localhost:$FRONTEND_PORT)"
        return 0
    else
        log_error "前端启动失败，查看日志: $FRONTEND_LOG"
        rm -f "$FRONTEND_PID_FILE"
        return 1
    fi
}

# ============================================================================
# 停止服务
# ============================================================================

stop_postgres() {
    local status=$(postgres_status)

    case $status in
        running)
            log_info "停止 PostgreSQL..."
            docker compose -f docker/compose/dev.yml down
            log_success "PostgreSQL 已停止"
            ;;
        stopped|not_found)
            log_info "PostgreSQL 未运行"
            ;;
    esac
}

# 验证进程是否是 memos 后端进程
verify_backend_process() {
    local pid=$1
    if [ -z "$pid" ]; then
        return 1
    fi

    # 检查进程是否存在
    if ! ps -p "$pid" &>/dev/null; then
        return 1
    fi

    # 获取进程的完整命令行和工作目录
    local cmdline=$(ps -p "$pid" -o command= 2>/dev/null)
    # macOS 不支持 ps -o cwd，使用 lsof 代替
    local cwd=$(lsof -p "$pid" 2>/dev/null | grep cwd | awk '{print $NF}' | tr -d ' ')

    # 调试信息 (如果需要调试打开注释)
    # echo "Debug verify_backend: PID=$pid CWD=$cwd ROOT=$ROOT_DIR CMD=$cmdline" >> "$LOG_DIR/debug.lock"

    # 策略 1: 严格匹配 - CWD 匹配且命令行包含特征
    if [ -n "$cmdline" ] && [ "$cwd" = "$ROOT_DIR" ]; then
        if echo "$cmdline" | grep -qE "(go run.*cmd/divinesense|divinesense.*--mode dev|divinesense.*--port $BACKEND_PORT)"; then
            return 0
        fi
    fi

    # 策略 2: 宽松匹配 - 针对 go run 产生的临时二进制文件 (CWD 可能不匹配)
    if [ -n "$cmdline" ]; then
        # 必须满足以下强特征之一，防止误杀：
        
        # 特征 A: 命令行包含项目名 "divinesense" 且包含开发模式参数 "--mode dev"
        # (覆盖 go run 产生的 /tmp/.../exe/divinesense --mode dev ... 情况)
        if echo "$cmdline" | grep -q "divinesense" && echo "$cmdline" | grep -q "\-\-mode dev"; then
             return 0
        fi
        
        # 特征 B: 命令行包含端口参数 且 包含开发模式参数
        # (覆盖二进制文件名不含 divinesense 但参数完全匹配的情况)
        if echo "$cmdline" | grep -q "\-\-port $BACKEND_PORT" && echo "$cmdline" | grep -q "\-\-mode dev"; then
             return 0
        fi
    fi

    return 1
}

stop_backend() {
    local status=$(backend_status)

    case $status in
        running)
            local pid=$(cat "$BACKEND_PID_FILE")
            log_info "停止后端 (PID: $pid)..."
            kill "$pid" 2>/dev/null || true
            rm -f "$BACKEND_PID_FILE"
            log_success "后端已停止"
            ;;
        stopped)
            log_warn "后端已停止，清理 PID 文件"
            rm -f "$BACKEND_PID_FILE"
            ;;
        not_found)
            log_info "后端未运行"
            ;;
    esac

    # 额外检查：确保端口没有被占用（解决 go run 孤儿进程问题）
    if check_port $BACKEND_PORT; then
        log_warn "端口 $BACKEND_PORT 仍被占用，检查进程..."

        # 获取占用端口的进程列表
        local port_pids=$(lsof -ti ":$BACKEND_PORT" 2>/dev/null)

        if [ -n "$port_pids" ]; then
            for port_pid in $port_pids; do
                # 验证进程是否是我们启动的 divinesense 后端
                if verify_backend_process "$port_pid"; then
                    log_info "终止 divinesense 后端进程 (PID: $port_pid)..."
                    kill "$port_pid" 2>/dev/null || true
                    sleep 1
                    # 如果还没终止，强制杀死
                    if ps -p "$port_pid" &>/dev/null; then
                        kill -9 "$port_pid" 2>/dev/null || true
                    fi
                    log_success "已清理端口 $BACKEND_PORT 的 divinesense 进程"
                else
                    log_warn "端口 $BACKEND_PORT 被其他进程占用 (PID: $port_pid)"
                    local proc_cmd=$(ps -p "$port_pid" -o command=)
                    log_warn "  Command: $proc_cmd" 
                    log_warn "  (未匹配到 divinesense 特征，为防止误杀，跳过处理)"
                    log_warn "  如需终止该进程，请手动执行: kill $port_pid"
                fi
            done
        fi
    fi
}

# 验证进程是否是 memos 前端进程 (pnpm dev / vite)
verify_frontend_process() {
    local pid=$1
    if [ -z "$pid" ]; then
        return 1
    fi

    # 检查进程是否存在
    if ! ps -p "$pid" &>/dev/null; then
        return 1
    fi

    # 获取进程的完整命令行和工作目录
    local cmdline=$(ps -p "$pid" -o command= 2>/dev/null)
    # macOS 不支持 ps -o cwd，使用 lsof 代替
    local cwd=$(lsof -p "$pid" 2>/dev/null | grep cwd | awk '{print $NF}' | tr -d ' ')
    local web_dir="$ROOT_DIR/web"
    
    # Debug info
    # echo "Debug verify_frontend: PID=$pid CWD=$cwd WEB_DIR=$web_dir CMD=$cmdline" >> "$LOG_DIR/debug.lock"

    # 策略 1: 严格匹配
    if [ -n "$cmdline" ] && [ "$cwd" = "$web_dir" ]; then
        if echo "$cmdline" | grep -qE "(pnpm dev|vite|node.*vite.*dev)"; then
            return 0
        fi
    fi

    # 策略 2: 宽松匹配 - 只要包含 vite/pnpm 且监听了端口 (caller logic ensures listening)
    # 结合端口占用检查，这足够精准
    if [ -n "$cmdline" ]; then
        if echo "$cmdline" | grep -qE "(vite|pnpm)"; then
            return 0
        fi
    fi

    return 1
}

stop_frontend() {
    local status=$(frontend_status)

    case $status in
        running)
            local pid=$(cat "$FRONTEND_PID_FILE")
            log_info "停止前端 (PID: $pid)..."
            kill "$pid" 2>/dev/null || true
            rm -f "$FRONTEND_PID_FILE"
            log_success "前端已停止"
            ;;
        stopped)
            log_warn "前端已停止，清理 PID 文件"
            rm -f "$FRONTEND_PID_FILE"
            ;;
        not_found)
            log_info "前端未运行"
            ;;
    esac

    # 额外检查：确保端口没有被占用
    if check_port $FRONTEND_PORT; then
        log_warn "端口 $FRONTEND_PORT 仍被占用，检查进程..."

        # 获取占用端口的进程列表
        local port_pids=$(lsof -ti ":$FRONTEND_PORT" 2>/dev/null)

        if [ -n "$port_pids" ]; then
            for port_pid in $port_pids; do
                # 验证进程是否是我们启动的前端开发服务器
                if verify_frontend_process "$port_pid"; then
                    log_info "终止前端开发服务器进程 (PID: $port_pid)..."
                    kill "$port_pid" 2>/dev/null || true
                    sleep 1
                    if ps -p "$port_pid" &>/dev/null; then
                        kill -9 "$port_pid" 2>/dev/null || true
                    fi
                    log_success "已清理端口 $FRONTEND_PORT 的前端进程"
                else
                    log_warn "端口 $FRONTEND_PORT 被其他进程占用 (PID: $port_pid)，跳过终止"
                    log_warn "如需终止该进程，请手动执行: kill $port_pid"
                fi
            done
        fi
    fi
}

# ============================================================================
# 状态显示
# ============================================================================

show_status() {
    echo ""
    echo "=== DivineSense 开发环境状态 ==="
    echo ""

    # PostgreSQL
    local pg_status=$(postgres_status)
    case $pg_status in
        running)
            echo -e "PostgreSQL: ${GREEN}运行中${NC}"
            ;;
        stopped)
            echo -e "PostgreSQL: ${YELLOW}已停止${NC}"
            ;;
        not_found)
            echo -e "PostgreSQL: ${YELLOW}未创建${NC}"
            ;;
    esac

    # Backend
    local be_status=$(backend_status)
    case $be_status in
        running)
            local pid=$(cat "$BACKEND_PID_FILE")
            echo -e "后端:       ${GREEN}运行中${NC} (PID: $pid, http://localhost:$BACKEND_PORT)"
            ;;
        stopped)
            echo -e "后端:       ${RED}已停止${NC}"
            ;;
        not_found)
            echo -e "后端:       ${YELLOW}未运行${NC}"
            ;;
    esac

    # Frontend
    local fe_status=$(frontend_status)
    case $fe_status in
        running)
            local pid=$(cat "$FRONTEND_PID_FILE")
            echo -e "前端:       ${GREEN}运行中${NC} (PID: $pid, http://localhost:$FRONTEND_PORT)"
            ;;
        stopped)
            echo -e "前端:       ${RED}已停止${NC}"
            ;;
        not_found)
            echo -e "前端:       ${YELLOW}未运行${NC}"
            ;;
    esac

    echo ""
}

# ============================================================================
# 日志查看
# ============================================================================

show_logs() {
    local service=${1:-all}
    local follow=${2:-false}

    if [ "$follow" = "true" ]; then
        local tail_opts="-f"
    else
        local tail_opts="-20"
    fi

    case $service in
        postgres|pg)
            docker logs -f "$POSTGRES_CONTAINER"
            ;;
        backend|be)
            if [ -f "$BACKEND_LOG" ]; then
                tail $tail_opts "$BACKEND_LOG"
            else
                log_warn "后端日志文件不存在"
            fi
            ;;
        frontend|fe)
            if [ -f "$FRONTEND_LOG" ]; then
                tail $tail_opts "$FRONTEND_LOG"
            else
                log_warn "前端日志文件不存在"
            fi
            ;;
        all|"")
            echo "=== 后端日志 (最后 20 行) ==="
            if [ -f "$BACKEND_LOG" ]; then
                tail -20 "$BACKEND_LOG"
            fi
            echo ""
            echo "=== 前端日志 (最后 20 行) ==="
            if [ -f "$FRONTEND_LOG" ]; then
                tail -20 "$FRONTEND_LOG"
            fi
            ;;
        *)
            log_error "未知服务: $service"
            echo "可用服务: postgres, backend, frontend, all"
            exit 1
            ;;
    esac
}

# ============================================================================
# 主命令
# ============================================================================

cmd_start() {
    local detach=${1:-false}

    echo ""
    log_info "启动 DivineSense 开发环境..."
    echo ""

    check_docker

    # 按顺序启动服务
    start_postgres || exit 1
    sleep 2
    start_backend || exit 1
    sleep 1
    start_frontend || exit 1

    echo ""
    log_success "所有服务已启动！"
    echo ""
    echo "服务地址:"
    echo "  - 后端: http://localhost:$BACKEND_PORT"
    echo "  - 前端: http://localhost:$FRONTEND_PORT"
    echo ""
    echo "查看日志: ./scripts/dev.sh logs [postgres|backend|frontend]"
    echo "查看状态: ./scripts/dev.sh status"
    echo "停止服务: ./scripts/dev.sh stop"
    echo ""

    if [ "$detach" = "true" ]; then
        log_info "后台运行模式 (-d)，不自动显示日志"
    else
        # 显示实时日志
        log_info "显示实时日志 (Ctrl+C 退出日志查看，服务继续运行)..."
        echo ""
        show_logs backend true
    fi
}

cmd_stop() {
    echo ""
    log_info "停止 DivineSense 开发环境..."
    echo ""

    # 按逆序停止服务
    stop_frontend
    stop_backend
    stop_postgres

    echo ""
    log_success "所有服务已停止"
    echo ""
}

cmd_restart() {
    local detach=${1:-false}

    echo ""
    log_info "重启所有服务（PostgreSQL + 后端 + 前端）..."
    echo ""

    # 停止所有服务（包括PostgreSQL）
    stop_frontend
    stop_backend
    stop_postgres

    sleep 2

    # 启动 PostgreSQL
    check_docker
    log_info "启动 PostgreSQL..."
    start_postgres || exit 1
    sleep 2

    # 重启应用服务
    start_backend || exit 1
    sleep 1
    start_frontend || exit 1

    echo ""
    log_success "应用已重启！"
    echo ""
    echo "服务地址:"
    echo "  - 后端: http://localhost:$BACKEND_PORT"
    echo "  - 前端: http://localhost:$FRONTEND_PORT"
    echo ""
    echo "查看日志: ./scripts/dev.sh logs [postgres|backend|frontend]"
    echo "查看状态: ./scripts/dev.sh status"
    echo "停止服务: ./scripts/dev.sh stop"
    echo ""

    if [ "$detach" = "true" ]; then
        log_info "后台运行模式 (-d)，不自动显示日志"
    else
        # 显示实时日志
        log_info "显示实时日志 (Ctrl+C 退出日志查看，服务继续运行)..."
        echo ""
        show_logs backend true
    fi
}

cmd_status() {
    show_status
}

cmd_logs() {
    local service=${1:-all}
    local follow=false

    if [ "$service" = "-f" ] || [ "$2" = "-f" ]; then
        follow=true
        [ "$service" = "-f" ] && service="all"
    fi

    show_logs "$service" "$follow"
}

# ============================================================================
# 入口
# ============================================================================

case "${1:-}" in
    start)
        if [ "$2" = "-d" ]; then
            cmd_start true
        else
            cmd_start false
        fi
        ;;
    stop)
        cmd_stop
        ;;
    restart)
        if [ "$2" = "-d" ]; then
            cmd_restart true
        else
            cmd_restart false
        fi
        ;;
    status)
        cmd_status
        ;;
    logs)
        cmd_logs "${2:-}" "${3:-}"
        ;;
    *)
        echo "DivineSense 开发环境管理脚本"
        echo ""
        echo "用法: $0 [command]"
        echo ""
        echo "命令:"
        echo "  start          启动所有服务 (PostgreSQL -> 后端 -> 前端)"
        echo "  stop           停止所有服务"
        echo "  restart        重启所有服务"
        echo "  status         查看服务状态"
        echo "  logs [service] 查看日志 (可选: postgres|backend|frontend, 默认: all)"
        echo "                  加 -f 参数实时跟踪日志"
        echo ""
        echo "示例:"
        echo "  $0 start              # 启动所有服务"
        echo "  $0 status             # 查看状态"
        echo "  $0 logs backend       # 查看后端日志"
        echo "  $0 logs backend -f    # 实时查看后端日志"
        echo ""
        exit 1
        ;;
esac
