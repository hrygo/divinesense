#!/bin/bash
# =============================================================================
# DivineSense 二进制部署管理脚本
# =============================================================================
#
# 用法: ./deploy-binary.sh [命令]
#
# 命令:
#   upgrade   - 升级到最新版本
#   restart   - 重启服务
#   start     - 启动服务
#   stop      - 停止服务
#   status    - 查看服务状态
#   logs      - 查看服务日志
#   backup    - 备份数据库
#   restore   - 恢复数据库 <文件>
#   uninstall - 卸载 DivineSense
#
# =============================================================================

set -e

# 获取脚本目录
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
LIB_DIR="${SCRIPT_DIR}/../lib"

# 加载共享函数库
if [ -f "${LIB_DIR}/common.sh" ]; then
    source "${LIB_DIR}/common.sh"
else
    echo "错误: 找不到共享库 ${LIB_DIR}/common.sh"
    # 降级：内联最小必需函数
    RED='\033[0;31m'
    GREEN='\033[0;32m'
    YELLOW='\033[1;33m'
    BLUE='\033[0;34m'
    NC='\033[0m'
    INSTALL_DIR="${DIVINE_INSTALL_DIR:-/opt/divinesense}"
    CONFIG_DIR="${DIVINE_CONFIG_DIR:-/etc/divinesense}"
    BACKUP_DIR="${INSTALL_DIR}/backups"
    SERVICE_NAME="divinesense"
    CURL_CONNECT_TIMEOUT=30
    CURL_MAX_TIME=300
    log_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
    log_success() { echo -e "${GREEN}[OK]${NC} $1"; }
    log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
    log_error() { echo -e "${RED}[ERROR]${NC} $1"; }
fi

# GitHub 配置
DOWNLOAD_URL="${DOWNLOAD_URL:-https://github.com/hrygo/divinesense/releases}"
GITHUB_API_URL="${GITHUB_API_URL:-https://api.github.com/repos/hrygo/divinesense}"

# 检查是否已安装
check_installed() {
    if [ ! -f "${INSTALL_DIR}/bin/divinesense" ]; then
        log_error "DivineSense 未安装"
        log_info "请先运行: curl -fsSL https://raw.githubusercontent.com/hrygo/divinesense/main/deploy/install.sh | sudo bash -s -- --mode=binary"
        exit 1
    fi
}

# 获取当前版本
get_current_version() {
    "${INSTALL_DIR}/bin/divinesense" --version 2>/dev/null | grep -oP 'v?\K[0-9.]+' || echo "unknown"
}

# 获取最新版本
get_latest_version() {
    curl -s --connect-timeout $CURL_CONNECT_TIMEOUT --max-time $CURL_MAX_TIME \
        "${GITHUB_API_URL}/releases/latest" | grep -oP '"tag_name":\s*"v?\K[^"]+' | head -1
}

# 升级服务
upgrade() {
    log_info "升级 DivineSense..."

    check_installed

    local current_version=$(get_current_version)
    local latest_version=$(get_latest_version)

    log_info "当前版本: ${current_version}"
    log_info "最新版本: ${latest_version}"

    if [ "$current_version" = "$latest_version" ]; then
        log_info "已是最新版本"
        return 0
    fi

    # 备份前升级
    log_warn "升级前自动备份..."
    backup_auto

    # 下载新二进制
    local BINARY_ARCH=$(detect_arch)
    local binary_name="divinesense-${latest_version}-linux-${BINARY_ARCH}"
    local download_url="${DOWNLOAD_URL}/download/${latest_version}/${binary_name}"

    log_info "下载 ${latest_version}..."

    if ! download_binary "$download_url" "${INSTALL_DIR}/bin/divinesense" "$BINARY_ARCH"; then
        log_error "下载失败"
        exit 1
    fi

    # 停止服务
    log_info "停止服务..."
    systemctl stop "$SERVICE_NAME"

    # 启动服务
    log_info "启动服务..."
    systemctl start "$SERVICE_NAME"

    # 等待服务就绪
    sleep 3

    if systemctl is-active --quiet "$SERVICE_NAME"; then
        log_success "已升级到 ${latest_version}"
    else
        log_error "服务启动失败"
        log_info "查看日志: journalctl -u ${SERVICE_NAME} -n 50"
        exit 1
    fi
}

# 重启服务
restart() {
    check_installed
    log_info "重启服务..."
    systemctl restart "$SERVICE_NAME"
    log_success "服务已重启"
}

# 启动服务
start() {
    check_installed
    log_info "启动服务..."
    systemctl start "$SERVICE_NAME"
    log_success "服务已启动"
}

# 停止服务
stop() {
    check_installed
    log_info "停止服务..."
    systemctl stop "$SERVICE_NAME"
    log_success "服务已停止"
}

# 显示状态
status() {
    check_installed

    echo ""
    echo "=== 服务状态 ==="
    systemctl status "$SERVICE_NAME" --no-pager || true

    echo ""
    echo "=== 版本信息 ==="
    echo "二进制: $(${INSTALL_DIR}/bin/divinesense --version 2>/dev/null || echo "unknown")"

    echo ""
    echo "=== 资源使用 ==="
    systemctl show "$SERVICE_NAME" -p CPUUsage,MemoryCurrent --no-pager 2>/dev/null || true

    echo ""
    echo "=== Geek Mode ==="
    if grep -q "DIVINESENSE_CLAUDE_CODE_ENABLED=true" "${CONFIG_DIR}/config" 2>/dev/null; then
        echo -e "  状态: ${GREEN}已启用${NC}"
        if command -v claude &>/dev/null; then
            echo -e "  CLI: ${GREEN}已安装${NC} ($(claude --version 2>/dev/null || echo "版本未知"))"
        else
            echo -e "  CLI: ${YELLOW}未安装${NC} (运行: npm install -g @anthropic-ai/claude-code)"
        fi
    else
        echo "  状态: 未启用"
    fi

    echo ""
}

# 查看日志
logs() {
    check_installed
    journalctl -u "$SERVICE_NAME" -f
}

# 备份数据库
backup() {
    check_installed

    local backup_file="${BACKUP_DIR}/divinesense-backup-$(date +%Y%m%d-%H%M%S).sql.gz"

    mkdir -p "$BACKUP_DIR"

    # 读取配置
    source "${CONFIG_DIR}/config" 2>/dev/null || true

    if [ "${DIVINESENSE_DRIVER:-sqlite}" = "postgres" ]; then
        log_info "备份 PostgreSQL..."

        # 提取 DSN 组件
        local dsn="${DIVINESENSE_DSN}"
        local db_user=$(echo "$dsn" | grep -oP '://\K[^:]+' || echo "divinesense")
        local db_name=$(echo "$dsn" | grep -oP '/[^?]*' | sed 's/\///' || echo "divinesense")

        if command -v docker &>/dev/null && docker ps | grep -q divinesense-postgres; then
            # Docker PostgreSQL
            docker exec divinesense-postgres pg_dump -U "$db_user" "$db_name" 2>&1 | gzip > "$backup_file"
        elif command -v pg_dump &>/dev/null; then
            # 系统 PostgreSQL
            pg_dump -U "$db_user" "$db_name" 2>&1 | gzip > "$backup_file"
        else
            log_error "找不到 PostgreSQL，请检查服务状态"
            exit 1
        fi
    else
        log_info "备份 SQLite..."
        local db_file="${DIVINESENSE_DATA:-${INSTALL_DIR}/data}/divinesense.db"
        if [ -f "$db_file" ]; then
            cp "$db_file" "${BACKUP_DIR}/divinesense-backup-$(date +%Y%m%d-%H%M%S).db"
            log_success "SQLite 数据库已备份"
            return 0
        else
            log_warn "数据库文件不存在: $db_file"
            return 0
        fi
    fi

    # 验证备份文件
    if [ -f "$backup_file" ]; then
        local size=$(stat -c%s "$backup_file" 2>/dev/null || stat -f%z "$backup_file" 2>/dev/null)
        if [ "$size" -gt 0 ]; then
            local size_human=$(du -h "$backup_file" | cut -f1)
            log_success "备份已创建: ${backup_file} (${size_human})"
        else
            log_error "备份文件为空"
            rm -f "$backup_file"
            exit 1
        fi
    else
        log_error "备份失败"
        exit 1
    fi
}

# 自动备份 (静默)
backup_auto() {
    local backup_file="${BACKUP_DIR}/.auto-backup-$(date +%Y%m%d-%H%M%S).sql.gz"

    mkdir -p "$BACKUP_DIR"

    source "${CONFIG_DIR}/config" 2>/dev/null || true

    if [ "${DIVINESENSE_DRIVER:-sqlite}" = "postgres" ]; then
        local dsn="${DIVINESENSE_DSN}"
        local db_user=$(echo "$dsn" | grep -oP '://\K[^:]+' || echo "divinesense")
        local db_name=$(echo "$dsn" | grep -oP '/[^?]*' | sed 's/\///' || echo "divinesense")

        if command -v docker &>/dev/null && docker ps | grep -q divinesense-postgres; then
            docker exec divinesense-postgres pg_dump -U "$db_user" "$db_name" 2>&1 | gzip > "$backup_file"
        elif command -v pg_dump &>/dev/null; then
            pg_dump -U "$db_user" "$db_name" 2>&1 | gzip > "$backup_file"
        fi
    fi

    # 清理旧自动备份 (保留最近 3 个)
    ls -t "${BACKUP_DIR}"/.auto-backup-*.sql.gz 2>/dev/null | tail -n +4 | xargs rm -f 2>/dev/null || true
}

# 恢复数据库
restore() {
    check_installed

    local backup_file="$2"

    if [ -z "${backup_file}" ]; then
        log_error "请指定备份文件"
        echo "用法: $0 restore <backup-file>"
        exit 1
    fi

    if [ ! -f "${backup_file}" ]; then
        log_error "备份文件不存在: ${backup_file}"
        exit 1
    fi

    log_warn "这将替换当前数据库!"
    read -p "确认继续? (yes/no): " confirm

    if [ "${confirm}" != "yes" ]; then
        log_info "已取消"
        exit 0
    fi

    source "${CONFIG_DIR}/config" 2>/dev/null || true

    if [ "${DIVINESENSE_DRIVER:-sqlite}" = "postgres" ]; then
        local dsn="${DIVINESENSE_DSN}"
        local db_user=$(echo "$dsn" | grep -oP '://\K[^:]+' || echo "divinesense")
        local db_name=$(echo "$dsn" | grep -oP '/[^?]*' | sed 's/\///' || echo "divinesense")

        log_info "恢复 PostgreSQL..."
        stop

        if command -v docker &>/dev/null && docker ps | grep -q divinesense-postgres; then
            # 删除并重建数据库
            docker exec divinesense-postgres psql -U "$db_user" -c "DROP DATABASE IF EXISTS ${db_name};"
            docker exec divinesense-postgres psql -U "$db_user" -c "CREATE DATABASE ${db_name};"
            # 恢复
            gunzip < "${backup_file}" | docker exec -i divinesense-postgres psql -U "$db_user" "$db_name"
        elif command -v psql &>/dev/null; then
            psql -U "$db_user" -c "DROP DATABASE IF EXISTS ${db_name};"
            psql -U "$db_user" -c "CREATE DATABASE ${db_name};"
            gunzip < "${backup_file}" | psql -U "$db_user" "$db_name"
        fi

        start
    else
        log_info "恢复 SQLite..."
        local db_file="${DIVINESENSE_DATA:-${INSTALL_DIR}/data}/divinesense.db"
        stop
        cp "${backup_file}" "$db_file"
        start
    fi

    log_success "恢复完成"
}

# 卸载
uninstall() {
    check_installed

    log_warn "这将删除 DivineSense!"
    read -p "确认继续? (yes/no): " confirm

    if [ "${confirm}" != "yes" ]; then
        log_info "已取消"
        exit 0
    fi

    # 询问数据处理
    echo ""
    read -p "删除数据目录? (yes/no): " remove_data

    log_info "停止服务..."
    systemctl stop "$SERVICE_NAME"
    systemctl disable "$SERVICE_NAME"

    log_info "删除文件..."
    rm -f "/etc/systemd/system/${SERVICE_NAME}.service"
    systemctl daemon-reload

    if [ "${remove_data}" = "yes" ]; then
        rm -rf "$INSTALL_DIR"
        rm -rf "$CONFIG_DIR"
        log_info "数据目录已删除"
    else
        log_info "数据目录保留: ${INSTALL_DIR}"
    fi

    # 询问用户删除
    echo ""
    read -p "删除 divine 用户? (yes/no): " remove_user
    if [ "${remove_user}" = "yes" ]; then
        userdel divine 2>/dev/null || true
        log_info "用户已删除"
    fi

    log_success "卸载完成"
}

# 显示帮助
show_help() {
    echo ""
    echo "DivineSense 二进制部署管理"
    echo ""
    echo "用法: $0 [命令] [参数]"
    echo ""
    echo "命令:"
    echo "  upgrade   - 升级到最新版本"
    echo "  restart   - 重启服务"
    echo "  start     - 启动服务"
    echo "  stop      - 停止服务"
    echo "  status    - 查看服务状态"
    echo "  logs      - 查看服务日志"
    echo "  backup    - 备份数据库"
    echo "  restore   - 恢复数据库 <文件>"
    echo "  uninstall - 卸载 DivineSense"
    echo ""
    echo "环境变量:"
    echo "  DIVINE_INSTALL_DIR  安装目录 (默认: /opt/divinesense)"
    echo "  DIVINE_CONFIG_DIR   配置目录 (默认: /etc/divinesense)"
    echo ""
    echo "示例:"
    echo "  $0 status              # 查看状态"
    echo "  $0 logs                # 查看日志"
    echo "  $0 backup              # 创建备份"
    echo "  $0 restore backup.sql.gz # 恢复备份"
    echo ""
}

# 主函数
case "${1:-help}" in
    upgrade)
        upgrade
        ;;
    restart)
        restart
        ;;
    start)
        start
        ;;
    stop)
        stop
        ;;
    status)
        status
        ;;
    logs)
        logs
        ;;
    backup)
        backup
        ;;
    restore)
        restore "$@"
        ;;
    uninstall)
        uninstall
        ;;
    help|--help|-h)
        show_help
        ;;
    *)
        log_error "未知命令: $1"
        show_help
        exit 1
        ;;
esac
