#!/bin/bash
# =============================================================================
# Baileys WhatsApp Bridge 部署管理脚本
# =============================================================================
#
# 用法: ./baileys-bridge.sh [命令]
#
# 命令:
#   install   - 安装并启动 WhatsApp Bridge 服务
#   start     - 启动服务
#   stop      - 停止服务
#   restart   - 重启服务
#   status    - 查看服务状态
#   logs      - 查看日志
#   health    - 健康检查
#   uninstall - 卸载服务
#   qr        - 显示 QR 码 (用于配对)
#   update    - 更新到最新版本
#   backup    - 备份认证数据
#
# =============================================================================

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

# 日志函数
log_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
log_success() { echo -e "${GREEN}[OK]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }
log_step() { echo -e "${CYAN}[STEP]${NC} $1"; }

# 配置变量
BRIDGE_DIR="${DIVINE_BRIDGE_DIR:-/opt/divinesense/plugin/chat_apps/channels/whatsapp/bridge}"
SERVICE_NAME="baileys-bridge"
LOG_DIR="/var/log/baileys-bridge"
CONFIG_FILE="${BRIDGE_DIR}/.env"
AUTH_FILE="${BRIDGE_DIR}/baileys_auth_info.json"
SYSTEMD_SERVICE="/etc/systemd/system/${SERVICE_NAME}.service"
PM2_ECOSYSTEM="${BRIDGE_DIR}/ecosystem.config.cjs"
INSTALL_DIR="${DIVINE_INSTALL_DIR:-/opt/divinesense}"
WEBHOOK_URL="${DIVINESENSE_WEBHOOK_URL:-http://localhost:5230/api/v1/chat_apps/webhook}"

# 检查是否已安装
is_installed() {
    [ -f "${SYSTEMD_SERVICE}" ] || pm2 list | grep -q "${SERVICE_NAME}"
}

# 检查服务状态
is_running() {
    if systemctl is-active --quiet "${SERVICE_NAME}" 2>/dev/null; then
        return 0
    elif pm2 list | grep -q "online|connected.*${SERVICE_NAME}"; then
        return 0
    fi
    return 1
}

# 检查 Node.js
check_nodejs() {
    if ! command -v node &>/dev/null; then
        log_error "Node.js 未安装"
        log_info "请先安装 Node.js 18+:"
        log_info "  curl -fsSL https://deb.nodesource.com/setup_18.x | sudo -E bash -"
        exit 1
    fi

    local node_version=$(node -v | sed 's/v//' | cut -d. -f1)
    if [ "$node_version" -lt 18 ]; then
        log_error "Node.js 版本过低 (需要 >= 18.x)"
        exit 1
    fi

    log_success "Node.js $(node -v) 检查通过"
}

# 安装依赖
install_dependencies() {
    log_step "安装 Node.js 依赖..."

    cd "${BRIDGE_DIR}"

    if [ ! -d "node_modules" ]; then
        npm install
    else
        log_info "依赖已安装，检查更新..."
        npm ci || npm install
    fi

    log_success "依赖安装完成"
}

# 创建目录
create_directories() {
    log_step "创建目录结构..."

    mkdir -p "${BRIDGE_DIR}"
    mkdir -p "${LOG_DIR}"
    mkdir -p "${BRIDGE_DIR}/.baileys"

    # 设置权限
    chown -R divine:divine "${BRIDGE_DIR}" 2>/dev/null || true
    chmod -R 750 "${BRIDGE_DIR}"
    chown -R divine:divine "${LOG_DIR}" 2>/dev/null || true

    log_success "目录已创建: ${BRIDGE_DIR}"
}

# 创建配置文件
create_config() {
    log_step "创建配置文件..."

    if [ ! -f "${CONFIG_FILE}" ]; then
        cp "${BRIDGE_DIR}/.env.example" "${CONFIG_FILE}" 2>/dev/null || cat > "${CONFIG_FILE}" << 'EOF'
PORT=3001
DIVINESENSE_WEBHOOK_URL=${WEBHOOK_URL}
BAILEYS_AUTH_FILE=./baileys_auth_info.json
NODE_ENV=production
EOF
        log_success "配置文件已创建: ${CONFIG_FILE}"
    else
        log_info "配置文件已存在: ${CONFIG_FILE}"
    fi
}

# 使用 PM2 安装
install_pm2() {
    if ! command -v pm2 &>/dev/null; then
        log_step "安装 PM2..."
        npm install -g pm2
        log_success "PM2 已安装"
    fi

    cd "${BRIDGE_DIR}"

    # 启动服务
    log_step "启动 WhatsApp Bridge (PM2)..."
    pm2 start ecosystem.config.cjs
    pm2 save

    log_success "WhatsApp Bridge 已启动 (PM2)"
}

# 使用 systemd 安装
install_systemd() {
    log_step "配置 systemd 服务..."

    # 复制服务文件
    cp "${BRIDGE_DIR}/baileys-bridge.service" "${SYSTEMD_SERVICE}"

    # 重新加载 systemd
    systemctl daemon-reload

    # 启用服务
    systemctl enable "${SERVICE_NAME}"

    # 启动服务
    systemctl start "${SERVICE_NAME}"

    log_success "WhatsApp Bridge 已启动 (systemd)"
}

# 主安装函数
install() {
    log_info "=========================================="
    log_info "Baileys WhatsApp Bridge 安装"
    log_info "=========================================="
    echo ""

    check_nodejs
    create_directories
    install_dependencies
    create_config

    # 选择进程管理器
    if command -v pm2 &>/dev/null; then
        install_pm2
    else
        install_systemd
    fi

    # 等待服务启动
    log_step "等待服务启动..."
    sleep 5

    # 健康检查
    if health_check; then
        log_success "健康检查通过"
    else
        log_warn "服务可能未成功启动，请检查日志:"
        log_info "  $0 logs"
    fi

    echo ""
    log_success "=========================================="
    log_success "安装完成！"
    log_success "=========================================="
    echo ""
    show_connection_info
}

# 启动服务
start() {
    if is_running; then
        log_info "服务已在运行"
        return
    fi

    log_info "启动 WhatsApp Bridge..."

    if command -v pm2 &>/dev/null && pm2 list | grep -q "${SERVICE_NAME}"; then
        pm2 start "${SERVICE_NAME}"
    else
        systemctl start "${SERVICE_NAME}"
    fi

    sleep 3
    log_success "服务已启动"
    show_connection_info
}

# 停止服务
stop() {
    log_info "停止 WhatsApp Bridge..."

    if command -v pm2 &>/dev/null && pm2 list | grep -q "${SERVICE_NAME}"; then
        pm2 stop "${SERVICE_NAME}"
    else
        systemctl stop "${SERVICE_NAME}"
    fi

    log_success "服务已停止"
}

# 重启服务
restart() {
    log_info "重启 WhatsApp Bridge..."

    if command -v pm2 &>/dev/null && pm2 list | grep -q "${SERVICE_NAME}"; then
        pm2 restart "${SERVICE_NAME}"
    else
        systemctl restart "${SERVICE_NAME}"
    fi

    sleep 3
    log_success "服务已重启"
    show_connection_info
}

# 显示状态
status() {
    echo ""
    echo "=== 服务状态 ==="

    if command -v pm2 &>/dev/null && pm2 list | grep -q "${SERVICE_NAME}"; then
        pm2 show "${SERVICE_NAME}"
    else
        systemctl status "${SERVICE_NAME}" --no-pager || true
    fi

    echo ""
    health_check
}

# 显示日志
logs() {
    if command -v pm2 &>/dev/null && pm2 list | grep -q "${SERVICE_NAME}"; then
        pm2 logs "${SERVICE_NAME}"
    else
        journalctl -u "${SERVICE_NAME}" -f
    fi
}

# 健康检查
health_check() {
    local url="http://localhost:3001/health"

    echo -n "服务状态: "
    if curl -sf "${url}" >/dev/null 2>&1; then
        echo -e "${GREEN}运行中${NC}"

        # 检查连接状态
        local info=$(curl -s "${url/health}/info" 2>/dev/null)
        if [ -n "$info" ]; then
            local connected=$(echo "$info" | grep -o '"connected":[^,]*' | cut -d: -f2)
            if [ "$connected" = "true" ]; then
                echo -e "  WhatsApp: ${GREEN}已连接${NC}"
            else
                echo -e "  WhatsApp: ${YELLOW}未连接 (需扫码配对)${NC}"
            fi

            local phone=$(echo "$info" | grep -o '"phone":[^,]*' | cut -d'"' -f4 | cut -c1-)
            if [ -n "$phone" ] && [ "$phone" != "null" ]; then
                echo "  手机号: $phone"
            fi
        fi
    else
        echo -e "${RED}未运行${NC}"
        return 1
    fi

    return 0
}

# 显示 QR 码
show_qr() {
    local url="http://localhost:3001/info"

    log_info "获取 QR 码信息..."

    local info=$(curl -s "$url" 2>/dev/null)
    if [ -z "$info" ]; then
        log_error "无法连接到服务"
        return 1
    fi

    local qrcode=$(echo "$info" | grep -o '"qrcode":[^,]*' | cut -d'"' -f4 | sed 's/\\n/\n/g')
    local connected=$(echo "$info" | grep -o '"connected":[^,]*' | cut -d: -f2)

    echo ""

    if [ "$connected" = "true" ]; then
        local phone=$(echo "$info" | grep -o '"phone":[^,]*' | cut -d'"' -f4)
        log_success "WhatsApp 已连接!"
        echo ""
        echo "  手机号: $phone"
        echo ""
        echo "如需重新配对，请先删除认证文件："
        echo "  sudo rm ${AUTH_FILE}"
        echo "  ${0} restart"
    else
        echo "=================================================="
        echo "  QR Code - Scan with WhatsApp"
        echo "  Settings → Linked Devices → Link a Device"
        echo "=================================================="
        echo ""

        if command -v qrencode &>/dev/null; then
            echo "$qrcode" | qrencode -t ANSIUTF8
        else
            echo "$qrcode"
        fi

        echo ""
        log_info "或者访问: $url"
        echo ""
    fi
}

# 卸载
uninstall() {
    log_warn "这将卸载 WhatsApp Bridge 服务！"
    read -p "确认继续? (yes/no): " confirm

    if [ "$confirm" != "yes" ]; then
        log_info "已取消"
        exit 0
    fi

    log_info "停止服务..."
    stop 2>/dev/null || true

    # PM2 清理
    if command -v pm2 &>/dev/null && pm2 list | grep -q "${SERVICE_NAME}"; then
        pm2 delete "${SERVICE_NAME}"
        pm2 save
    fi

    # systemd 清理
    if [ -f "${SYSTEMD_SERVICE}" ]; then
        systemctl disable "${SERVICE_NAME}"
        rm -f "${SYSTEMD_SERVICE}"
        systemctl daemon-reload
    fi

    # 询问数据处理
    echo ""
    read -p "删除认证数据? (yes/no): " remove_data

    if [ "$remove_data" = "yes" ]; then
        rm -rf "${BRIDGE_DIR}/.baileys"/*
        log_info "认证数据已删除"
    else
        log_info "认证数据已保留: ${BRIDGE_DIR}/.baileys/"
    fi

    log_success "卸载完成"
}

# 备份认证数据
backup() {
    local backup_dir="${BRIDGE_DIR}/backups"
    local backup_file="${backup_dir}/baileys_auth_$(date +%Y%m%d_%H%M%S).tar.gz"

    mkdir -p "$backup_dir"

    if [ -d "${BRIDGE_DIR}/.baileys" ]; then
        log_info "备份认证数据..."
        tar -czf "$backup_file" -C "${BRIDGE_DIR}/.baileys" .

        local size=$(du -h "$backup_file" | cut -f1)
        log_success "备份完成: ${backup_file} (${size})"

        # 清理 7 天前的备份
        find "$backup_dir" -name "baileys_auth_*.tar.gz" -mtime +7 -delete 2>/dev/null || true
    else
        log_info "没有认证数据需要备份"
    fi
}

# 更新
update() {
    log_info "更新 WhatsApp Bridge..."

    cd "${BRIDGE_DIR}"

    # 备份
    backup

    # 拉取更新
    if [ -d .git ]; then
        git fetch origin
        git reset --hard origin/main
    else
        log_warn "非 git 仓库，跳过更新"
    fi

    # 安装依赖
    install_dependencies

    # 重启服务
    restart

    log_success "更新完成"
}

# 显示连接信息
show_connection_info() {
    echo ""
    echo "=========================================="
    echo "  WhatsApp Bridge 信息"
    echo "=========================================="
    echo ""
    echo "  API 地址: http://localhost:3001"
    echo "  健康检查: http://localhost:3001/health"
    echo "  QR 码获取: http://localhost:3001/info"
    echo ""
    echo "  DivineSense Webhook: ${WEBHOOK_URL}"
    echo ""
    echo "  管理命令:"
    echo "    状态: $0 status"
    echo "    日志: $0 logs"
    echo "    重启: $0 restart"
    echo "    QR 码: $0 qr"
    echo ""
}

# 显示帮助
show_help() {
    echo ""
    echo "Baileys WhatsApp Bridge 管理脚本"
    echo ""
    echo "用法: $0 [命令]"
    echo ""
    echo "命令:"
    echo "  install   - 安装并启动服务"
    echo "  start     - 启动服务"
    echo "  stop      - 停止服务"
    echo "  restart   - 重启服务"
    echo "  status    - 查看服务状态"
    echo "  logs      - 查看日志"
    echo "  health    - 健康检查"
    echo "  qr        - 显示配对 QR 码"
    echo "  backup    - 备份认证数据"
    echo "  update    - 更新到最新版本"
    echo "  uninstall - 卸载服务"
    echo ""
    echo "环境变量:"
    echo "  DIVINE_INSTALL_DIR    安装目录 (默认: /opt/divinesense)"
    echo "  DIVINE_WEBHOOK_URL     DivineSense Webhook URL"
    echo ""
    echo "示例:"
    echo "  $0 install              # 首次安装"
    echo "  $0 qr                   # 显示 QR 码"
    echo "  $0 status               # 查看状态"
    echo "  $0 logs                 # 查看日志"
    echo ""
}

# 主函数
case "${1:-help}" in
    install)
        install
        ;;
    start)
        start
        ;;
    stop)
        stop
        ;;
    restart)
        restart
        ;;
    status)
        status
        ;;
    logs)
        logs
        ;;
    health)
        health_check
        ;;
    qr)
        show_qr
        ;;
    backup)
        backup
        ;;
    update)
        update
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
