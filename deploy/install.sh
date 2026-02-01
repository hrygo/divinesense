#!/bin/bash
#
# DivineSense 一键安装脚本 v4.0
#
set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

# Config
INTERACTIVE=false
DEPLOY_MODE="${DEPLOY_MODE:-binary}"
PORT="${PORT:-5230}"
DB_TYPE="${DB_TYPE:-docker}"

log_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
log_success() { echo -e "${GREEN}[OK]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }
log_step() { echo -e "${CYAN}[STEP]${NC} $1"; }

print_banner() {
    echo ""
    echo -e "${CYAN}╔════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${CYAN}║${NC}  ${GREEN}DivineSense 安装向导 v4.0${NC}                                  ${CYAN}║${NC}"
    echo -e "${CYAN}╠════════════════════════════════════════════════════════════╣${NC}"
    echo -e "${CYAN}║${NC}  ${YELLOW}AI 驱动的个人第二大脑${NC}                                      ${CYAN}║${NC}"
    echo -e "${CYAN}╚════════════════════════════════════════════════════════════╝${NC}"
    echo ""
}

# Check root
check_root() {
    if [ "$EUID" -ne 0 ]; then
        log_error "需要 root 权限"
        exit 1
    fi
}

# Detect OS
detect_os() {
    if [ -f /etc/os-release ]; then
        . /etc/os-release
        OS="$ID"
    else
        log_error "无法检测操作系统"
        exit 1
    fi
    
    case "$OS" in
        alpine|arch) PKG_MANAGER="apk" ;;
        debian|ubuntu) PKG_MANAGER="apt" ;;
        centos|rhel|fedora|rocky) PKG_MANAGER="yum" ;;
        *) PKG_MANAGER="unknown" ;;
    esac
}

# Install dependencies
install_base_tools() {
    log_step "安装依赖..."
    case "$PKG_MANAGER" in
        apt)
            export DEBIAN_FRONTEND=noninteractive
            apt-get update -qq
            apt-get install -y -qq curl git 2>/dev/null
            ;;
        yum)
            yum install -y -q curl git 2>/dev/null
            ;;
        apk)
            apk add --no-cache curl git 2>/dev/null
            ;;
    esac
    log_success "依赖已安装"
}

# Generate password
generate_password() {
    if command -v openssl &>/dev/null; then
        openssl rand -hex 16 | head -c 20
    else
        tr -dc A-Za-z0-9 </dev/urandom 2>/dev/null | head -c 20
    fi
}

# Get server IP
get_server_ip() {
    curl -s --connect-timeout 3 -4 ifconfig.me 2>/dev/null || \
    curl -s --connect-timeout 3 -4 icanhazip.com 2>/dev/null || \
    hostname -I | awk '{print $1}'
}

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
    echo "  模式:   $DEPLOY_MODE"
    echo "  端口:   $PORT"
    echo "  数据库: $DB_TYPE"
    echo "  AI:     $ENABLE_AI"
    echo "  Geek:   $ENABLE_GEEK"
    echo ""
    echo -ne "确认开始安装? [Y/n]: "
    read -n 1 -r confirm
    if [[ ! "$confirm" =~ ^[Yy]$ ]] && [ -n "$confirm" ]; then
        log_info "已取消"
        exit 0
    fi
}

prompt() {
    local default="$2"
    echo -ne "${CYAN}▸${NC} $1 [${GREEN}${default}${NC}]: "
    read -r result
    echo "${result:-$default}"
}

print_box() {
    local text="$1"
    echo -e "${CYAN}┌$(printf '─%.0s' "$((40))")┐${NC}"
    echo -e "${CYAN}│${NC} ${text}$(printf ' %.0s' "$((39 - ${#text}))")${CYAN}│${NC}"
    echo -e "${CYAN}└$(printf '─%.0s' "$((40))")┘${NC}"
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
    mkdir -p /opt/divinesense
    cd /opt/divinesense

    if [ ! -d .git ]; then
        rm -rf /opt/divinesense/* 2>/dev/null || true
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

    local arch=$(uname -m)
    case "$arch" in
        x86_64) BINARY_ARCH="amd64" ;;
        aarch64) BINARY_ARCH="arm64" ;;
        *) log_error "不支持的架构: $arch"; exit 1 ;;
    esac

    mkdir -p /opt/divinesense/{bin,data,logs,backups,docker}
    mkdir -p /etc/divinesense

    if ! id divinesense &>/dev/null; then
        useradd -r -s /bin/false -d /opt/divinesense divinesense
    fi

    local download_url="https://github.com/hrygo/divinesense/releases/latest/download/divinesense-linux-${BINARY_ARCH}"
    local checksum_url="${download_url}.sha256"
    local tmp_binary="/tmp/divinesense-${BINARY_ARCH}"
    local tmp_checksum="/tmp/divinesense-${BINARY_ARCH}.sha256"

    # 下载二进制和校验和
    log_info "从 $download_url 下载..."
    if ! curl -fsSL "$download_url" -o "$tmp_binary"; then
        log_error "下载失败"
        exit 1
    fi

    # 下载校验和（可选）
    if curl -fsSL "$checksum_url" -o "$tmp_checksum" 2>/dev/null; then
        cd /tmp
        if sha256sum -c "$tmp_checksum" 2>/dev/null; then
            log_success "校验和验证通过"
        else
            log_warn "校验和验证失败，继续安装..."
        fi
    fi

    # 移动到目标位置
    mv "$tmp_binary" /opt/divinesense/bin/divinesense
    rm -f "$tmp_checksum"
    chmod +x /opt/divinesense/bin/divinesense
    
    local db_password=$(generate_password)
    local server_ip=$(get_server_ip)
    
    cat > /etc/divinesense/config << EOF
DIVINESENSE_INSTANCE_URL=http://${server_ip}:${PORT}
DIVINESENSE_PORT=${PORT}
DIVINESENSE_MODE=prod
DIVINESENSE_DATA=/opt/divinesense/data
DIVINESENSE_DRIVER=postgres
DIVINESENSE_DSN=postgres://divinesense:${db_password}@localhost:25432/divinesense?sslmode=disable
DIVINESENSE_AI_ENABLED=${ENABLE_AI}
DIVINESENSE_CLAUDE_CODE_ENABLED=${ENABLE_GEEK}
DIVINESENSE_CLAUDE_CODE_WORKDIR=/opt/divinesense/data
DIVINESENSE_EVOLUTION_ENABLED=${ENABLE_EVOLUTION}
DIVINESENSE_EVOLUTION_ADMIN_ONLY=true
EOF
    
    echo "$db_password" > /etc/divinesense/.db_password
    chmod 600 /etc/divinesense/.db_password
    
    # PostgreSQL in Docker
    cat > /opt/divinesense/docker/postgres.yml << EOF
version: '3.8'
services:
  postgres:
    image: pgvector/pgvector:pg16
    container_name: divinesense-postgres
    restart: unless-stopped
    environment:
      POSTGRES_DB: divinesense
      POSTGRES_USER: divinesense
      POSTGRES_PASSWORD: \${POSTGRES_PASSWORD}
    volumes:
      - postgres_data:/var/lib/postgresql/data
    ports:
      - "25432:5432"
volumes:
  postgres_data:
EOF
    
    cat > /opt/divinesense/docker/.env << EOF
POSTGRES_PASSWORD=${db_password}
EOF
    
    if command -v docker &>/dev/null; then
        cd /opt/divinesense/docker
        docker compose -f postgres.yml up -d
        sleep 5
    fi
    
    # Systemd service
    cat > /etc/systemd/system/divinesense.service << EOF
[Unit]
Description=DivineSense AI-Powered Personal Second Brain
After=network-online.target
Wants=network-online.target

[Service]
Type=exec
User=divinesense
EnvironmentFile=-/etc/divinesense/config
ExecStart=/opt/divinesense/bin/divinesense
Restart=always
RestartSec=10s

[Install]
WantedBy=multi-user.target
EOF
    
    systemctl daemon-reload
    systemctl enable divinesense
    systemctl start divinesense
    
    chown -R divinesense:divinesense /opt/divinesense
    
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
            --help|-h)
                echo "用法: $0 [--interactive] [--mode=binary|docker]"
                echo ""
                echo "  --interactive, -i  交互式配置向导"
                echo "  --mode=MODE       部署模式"
                exit 0
                ;;
        esac
        shift
    done
    
    print_banner
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
    echo ""
    print_box "安装完成"
    echo ""
    local server_ip=$(get_server_ip)
    echo -e "  访问: ${YELLOW}http://${server_ip}:${PORT}${NC}"
    echo ""
    echo -e "  管理:"
    echo -e "    状态: ${CYAN}systemctl status divinesense${NC}"
    echo -e "    日志: ${CYAN}journalctl -u divinesense -f${NC}"
    echo -e "    重启: ${CYAN}systemctl restart divinesense${NC}"
    echo ""
}

main "$@"
