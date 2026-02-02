#!/bin/bash
#
# DivineSense 一键安装脚本
#
set -e

# 获取脚本目录
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
LIB_DIR="${SCRIPT_DIR}/lib"

# 加载共享函数库
if [ -f "${LIB_DIR}/common.sh" ]; then
    source "${LIB_DIR}/common.sh"
else
    echo "错误: 找不到共享库 ${LIB_DIR}/common.sh"
    exit 1
fi

# 配置变量（可通过环境变量覆盖）
INTERACTIVE=false
DEPLOY_MODE="${DEPLOY_MODE:-binary}"
PORT="${PORT:-5230}"
DB_TYPE="${DB_TYPE:-docker}"
ENABLE_AI="${ENABLE_AI:-true}"
ENABLE_GEEK="${ENABLE_GEEK:-true}"
ENABLE_EVOLUTION="${ENABLE_EVOLUTION:-false}"

# 获取项目版本（如果存在）
get_project_version() {
    if [ -f "${SCRIPT_DIR}/../internal/version/version.go" ]; then
        grep -oP '(?<=Version = ")[^"]+' "${SCRIPT_DIR}/../internal/version/version.go" 2>/dev/null || echo "dev"
    else
        echo "dev"
    fi
}

PROJECT_VERSION=$(get_project_version)

# ============================================================================
# Interactive Wizard
# ============================================================================

run_interactive_wizard() {
    echo ""
    echo -e "${GREEN}欢迎使用 DivineSense!${NC}"
    echo ""

    # Deploy mode
    echo ""
    echo "选择部署模式:"
    echo "  1) Binary (推荐 - Geek Mode 原生支持)"
    echo "  2) Docker (测试)"
    echo ""
    echo -ne "选择 [1-2]: "
    read -r mode_choice
    case "$mode_choice" in
        2|"docker") DEPLOY_MODE="docker" ;;
        *) DEPLOY_MODE="binary" ;;
    esac

    # Port
    echo ""
    PORT=$(prompt "服务端口" "5230")

    # Database
    echo ""
    echo "数据库方式:"
    echo "  1) Docker (推荐)"
    echo "  2) 系统安装"
    echo "  3) 远程连接"
    echo ""
    echo -ne "选择 [1-3]: "
    read -r db_choice
    case "$db_choice" in
        2|"system") DB_TYPE="system" ;;
        3|"remote") DB_TYPE="remote" ;;
        *) DB_TYPE="docker" ;;
    esac

    # AI features
    echo ""
    echo -ne "启用 AI 功能? [Y/n]: "
    read -n 1 -r ai_confirm
    ENABLE_AI=true
    [[ ! "$ai_confirm" =~ ^[Yy]$ ]] && [[ -n "$ai_confirm" ]] && ENABLE_AI=false

    # Geek Mode
    echo ""
    echo -ne "启用 Geek Mode? [Y/n]: "
    read -n 1 -r geek_confirm
    ENABLE_GEEK=true
    [[ ! "$geek_confirm" =~ ^[Yy]$ ]] && [[ -n "$geek_confirm" ]] && ENABLE_GEEK=false

    # Evolution Mode
    echo ""
    echo -ne "启用 Evolution Mode (仅管理员)? [y/N]: "
    read -n 1 -r evo_confirm
    ENABLE_EVOLUTION=false
    [[ "$evo_confirm" =~ ^[Yy]$ ]] && ENABLE_EVOLUTION=true

    # Admin account
    echo ""
    ADMIN_USERNAME=$(prompt "管理员用户名" "admin")
    ADMIN_PASSWORD=$(prompt "管理员密码 (留空自动生成)" "")
    [ -z "$ADMIN_PASSWORD" ] && ADMIN_PASSWORD=$(generate_password)

    # Confirm
    echo ""
    print_box "配置确认"
    echo ""
    echo "  模式:    $DEPLOY_MODE"
    echo "  端口:    $PORT"
    echo "  数据库:  $DB_TYPE"
    echo "  AI:      $ENABLE_AI"
    echo "  Geek:    $ENABLE_GEEK"
    echo "  Evolution: $ENABLE_EVOLUTION"
    echo ""
    echo -ne "确认开始安装? [Y/n]: "
    read -n 1 -r confirm
    if [[ ! "$confirm" =~ ^[Yy]$ ]] && [ -n "$confirm" ]; then
        log_info "已取消"
        exit 0
    fi
}

# ============================================================================
# Installation
# ============================================================================

install_docker_mode() {
    log_step "安装 Docker..."

    if ! command -v docker &>/dev/null; then
        curl -fsSL https://get.docker.com | sh
        systemctl enable docker 2>/dev/null || true
        systemctl start docker
    fi
    log_success "Docker 已就绪"

    log_step "下载 DivineSense..."
    mkdir -p "$INSTALL_DIR"
    cd "$INSTALL_DIR"

    if [ ! -d .git ]; then
        rm -rf "${INSTALL_DIR:?}"/* 2>/dev/null || true
        if ! git clone --depth 1 https://github.com/hrygo/divinesense.git . 2>/dev/null; then
            log_error "Git clone 失败"
            exit 1
        fi
    fi

    local db_password=$(generate_password)
    local server_ip=$(get_server_ip)

    cat > .env.prod << EOF
DIVINESENSE_INSTANCE_URL=http://${server_ip}:${PORT}
DIVINESENSE_PORT=${PORT}
POSTGRES_PASSWORD=${db_password}
EOF

    echo "$db_password" > .db_password
    chmod 600 .db_password

    log_step "启动服务..."
    if ! docker compose -f docker/compose/prod.yml --env-file .env.prod up -d; then
        log_error "Docker compose 启动失败"
        log_info "检查日志: docker compose -f docker/compose/prod.yml logs"
        exit 1
    fi

    log_success "安装完成"
}

install_binary_mode() {
    log_step "下载 DivineSense..."

    local BINARY_ARCH=$(detect_arch)

    mkdir -p "$INSTALL_DIR"/{bin,data,logs,backups,docker}
    mkdir -p "$CONFIG_DIR"

    if ! id divine &>/dev/null; then
        useradd -r -s /bin/bash -d /home/divine -m divine
    fi

    # 创建 Geek Mode 和 Evolution Mode 工作目录
    mkdir -p /home/divine/.divinesense
    mkdir -p /home/divine/source
    chown -R divine:divine /home/divine

    # 获取最新版本号并构建正确的下载 URL
    local LATEST_VERSION=$(get_latest_version)
    log_info "最新版本: ${LATEST_VERSION}"

    # 文件名格式: divinesense-v0.80.4-linux-amd64
    local binary_name="divinesense-${LATEST_VERSION}-linux-${BINARY_ARCH}"
    local download_url="https://github.com/hrygo/divinesense/releases/download/${LATEST_VERSION}/${binary_name}"

    if ! download_binary "$download_url" "$INSTALL_DIR/bin/divinesense" "$BINARY_ARCH"; then
        log_error "下载二进制文件失败"
        exit 1
    fi

    local db_password=$(generate_password)
    local server_ip=$(get_server_ip)

    # 使用交互式配置的值
    cat > "$CONFIG_DIR/config" << EOF
DIVINESENSE_INSTANCE_URL=http://${server_ip}:${PORT}
DIVINESENSE_PORT=${PORT}
DIVINESENSE_MODE=prod
DIVINESENSE_DATA=${INSTALL_DIR}/data
DIVINESENSE_DRIVER=postgres
DIVINESENSE_DSN=postgres://divine:${db_password}@localhost:25432/divinesense?sslmode=disable
DIVINESENSE_AI_ENABLED=${ENABLE_AI}
DIVINESENSE_CLAUDE_CODE_ENABLED=${ENABLE_GEEK}
DIVINESENSE_CLAUDE_CODE_WORKDIR=/home/divine/.divinesense
DIVINESENSE_EVOLUTION_ENABLED=${ENABLE_EVOLUTION}
DIVINESENSE_EVOLUTION_ADMIN_ONLY=true
DIVINESENSE_EVOLUTION_SOURCE_DIR=/home/divine/source/divinesense
EOF

    echo "$db_password" > "$CONFIG_DIR/.db_password"
    chmod 600 "$CONFIG_DIR/.db_password"

    # 配置目录权限：divine 用户需要读取配置文件
    chown -R root:divine "$CONFIG_DIR"
    chmod 750 "$CONFIG_DIR"
    chmod 640 "$CONFIG_DIR/config"
    chmod 640 "$CONFIG_DIR/.db_password"

    # PostgreSQL in Docker
    cat > "$INSTALL_DIR/docker/postgres.yml" << EOF
version: '3.8'
services:
  postgres:
    image: pgvector/pgvector:pg16
    container_name: divinesense-postgres
    restart: unless-stopped
    environment:
      POSTGRES_DB: divinesense
      POSTGRES_USER: divine
      POSTGRES_PASSWORD: \${POSTGRES_PASSWORD}
    volumes:
      - postgres_data:/var/lib/postgresql/data
    ports:
      - "25432:5432"
volumes:
  postgres_data:
EOF

    cat > "$INSTALL_DIR/docker/.env" << EOF
POSTGRES_PASSWORD=${db_password}
EOF

    if command -v docker &>/dev/null; then
        cd "$INSTALL_DIR/docker"
        docker compose -f postgres.yml up -d
        sleep 5
    fi

    # Systemd service
    # 注意: 使用 --data 参数确保数据目录正确，使用 --port 参数设置端口
    # AmbientCapabilities=CAP_NET_BIND_SERVICE 允许非 root 用户绑定 1024 以下端口
    cat > /etc/systemd/system/${SERVICE_NAME}.service << EOF
[Unit]
Description=DivineSense AI-Powered Personal Second Brain
After=network-online.target docker.service
Wants=network-online.target

[Service]
Type=exec
User=divine
EnvironmentFile=-${CONFIG_DIR}/config
AmbientCapabilities=CAP_NET_BIND_SERVICE
ExecStart=${INSTALL_DIR}/bin/divinesense --port \${DIVINESENSE_PORT:-5230} --data ${INSTALL_DIR}/data
Restart=always
RestartSec=10s

# 安全加固
NoNewPrivileges=false
PrivateTmp=true

[Install]
WantedBy=multi-user.target
EOF

    systemctl daemon-reload
    systemctl enable "$SERVICE_NAME"
    systemctl start "$SERVICE_NAME"

    chown -R divine:divine "$INSTALL_DIR"

    # ============================================================================
    # 配置用户运维权限和工具
    # ============================================================================
    log_step "配置用户运维权限和工具..."

    # 配置 docker 组（非关键）
    if ! configure_docker_group; then
        log_warn "Docker 组配置失败，继续安装..."
    fi

    # 配置 sudoers（关键）
    if ! configure_sudoers; then
        log_warn "sudoers 配置失败，用户可能需要输入密码来管理服务"
    fi

    # 创建 Makefile（非关键）
    if ! create_user_makefile; then
        log_warn "Makefile 创建失败"
    fi

    # 配置 bash 别名（非关键）
    if ! configure_bash_aliases; then
        log_warn "bash 别名配置失败"
    fi

    log_success "安装完成"
}

# ============================================================================
# Main
# ============================================================================

main() {
    # Parse args
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --interactive|-i) INTERACTIVE="true" ;;
            --mode=*) DEPLOY_MODE="${1#*=}" ;;
            --port=*) PORT="${1#*=}" ;;
            --help|-h)
                echo "用法: $0 [选项]"
                echo ""
                echo "选项:"
                echo "  --interactive, -i     交互式配置向导"
                echo "  --mode=MODE          部署模式 (binary|docker)"
                echo "  --port=PORT          服务端口 (默认: 5230)"
                echo "  --help, -h            显示此帮助"
                echo ""
                echo "环境变量:"
                echo "  DEPLOY_MODE           部署模式 (binary|docker)"
                echo "  PORT                  服务端口"
                echo "  ENABLE_AI            启用 AI 功能 (true|false)"
                echo "  ENABLE_GEEK          启用 Geek Mode (true|false)"
                echo "  ENABLE_EVOLUTION     启用 Evolution Mode (true|false)"
                echo "  DIVINE_INSTALL_DIR    安装目录 (默认: /opt/divinesense)"
                echo "  DIVINE_CONFIG_DIR     配置目录 (默认: /etc/divinesense)"
                exit 0
                ;;
        esac
        shift
    done

    print_banner "$PROJECT_VERSION"
    check_root
    detect_os
    install_base_tools

    if [ "$INTERACTIVE" = "true" ]; then
        run_interactive_wizard
    fi

    if [ "$DEPLOY_MODE" = "docker" ]; then
        install_docker_mode
    else
        install_binary_mode
    fi

    # Show result
    show_complete "$PORT"
}

main "$@"
