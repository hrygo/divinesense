#!/bin/bash
# =============================================================================
# DivineSense 部署共享函数库
# =============================================================================
#
# 此文件包含部署脚本共用的函数和常量
# 使用方式: source /path/to/deploy/lib/common.sh
#
# =============================================================================

# 颜色定义
export RED='\033[0;31m'
export GREEN='\033[0;32m'
export YELLOW='\033[1;33m'
export BLUE='\033[0;34m'
export CYAN='\033[0;36m'
export NC='\033[0m'

# 配置常量
export INSTALL_DIR="${DIVINE_INSTALL_DIR:-/opt/divinesense}"
export CONFIG_DIR="${DIVINE_CONFIG_DIR:-/etc/divinesense}"
export BACKUP_DIR="${INSTALL_DIR}/backups"
export SERVICE_NAME="divinesense"

# 网络超时设置
export CURL_CONNECT_TIMEOUT=30
export CURL_MAX_TIME=300

# 日志函数
log_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
log_success() { echo -e "${GREEN}[OK]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }
log_step() { echo -e "${CYAN}[STEP]${NC} $1"; }

# 检查 root 权限
check_root() {
    if [ "$EUID" -ne 0 ]; then
        log_error "需要 root 权限"
        exit 1
    fi
}

# 检测操作系统
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
    export OS PKG_MANAGER
}

# 安装基础工具
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

# 检测架构
detect_arch() {
    local arch=$(uname -m)
    case "$arch" in
        x86_64) echo "amd64" ;;
        aarch64|arm64) echo "arm64" ;;
        *)
            log_error "不支持的架构: $arch"
            exit 1
            ;;
    esac
}

# 生成随机密码
generate_password() {
    if command -v openssl &>/dev/null; then
        openssl rand -hex 16 | head -c 20
    else
        tr -dc A-Za-z0-9 </dev/urandom 2>/dev/null | head -c 20
    fi
}

# 获取服务器 IP
get_server_ip() {
    curl -s --connect-timeout 3 -4 ifconfig.me 2>/dev/null || \
    curl -s --connect-timeout 3 -4 icanhazip.com 2>/dev/null || \
    hostname -I | awk '{print $1}'
}

# 打印横幅
print_banner() {
    local version="${1:-v4.0}"
    echo ""
    echo -e "${CYAN}╔════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${CYAN}║${NC}  ${GREEN}DivineSense 安装向导 ${version}${NC}                              ${CYAN}║${NC}"
    echo -e "${CYAN}╠════════════════════════════════════════════════════════════╣${NC}"
    echo -e "${CYAN}║${NC}  ${YELLOW}AI 驱动的个人第二大脑${NC}                                      ${CYAN}║${NC}"
    echo -e "${CYAN}╚════════════════════════════════════════════════════════════╝${NC}"
    echo ""
}

# 打印确认框
print_box() {
    local text="$1"
    local width="${2:-40}"
    local padding=$((width - ${#text} - 1))
    echo -e "${CYAN}┌$(printf '─%.0s' $(seq 1 $width))┐${NC}"
    echo -e "${CYAN}│${NC} ${text}$(printf ' %.0s' $(seq 1 $padding))${CYAN}│${NC}"
    echo -e "${CYAN}└$(printf '─%.0s' $(seq 1 $width))┘${NC}"
}

# 交互式提示
prompt() {
    local default="$2"
    echo -ne "${CYAN}▸${NC} $1 [${GREEN}${default}${NC}]: "
    read -r result
    echo "${result:-$default}"
}

# 选择菜单
choose() {
    local prompt="$1"
    shift
    local options=("$@")
    local default="${options[0]}"

    echo ""
    echo "$prompt"
    local i=1
    for opt in "${options[@]}"; do
        echo "  $i) $opt"
        i=$((i + 1))
    done
    echo ""
    echo -ne "选择 [1-${#options[@]}]: "
    read -r choice

    local idx=$((choice - 1))
    if [ "$idx" -ge 0 ] && [ "$idx" -lt "${#options[@]}" ]; then
        echo "${options[$idx]}"
    else
        echo "$default"
    fi
}

# 验证下载文件
verify_checksum() {
    local binary_file="$1"
    local checksum_file="$2"

    if [ ! -f "$checksum_file" ]; then
        log_warn "校验文件不存在，跳过验证"
        return 0
    fi

    cd "$(dirname "$binary_file")"
    local expected_checksum=$(cat "$checksum_file" | cut -d' ' -f1)
    local actual_checksum=$(sha256sum "$(basename "$binary_file")" | cut -d' ' -f1)

    if [ "$expected_checksum" != "$actual_checksum" ]; then
        log_error "校验和不匹配!"
        log_error "预期: $expected_checksum"
        log_error "实际: $actual_checksum"
        return 1
    fi

    log_success "校验和验证通过"
    return 0
}

# 下载二进制文件（带校验）
download_binary() {
    local url="$1"
    local output="$2"
    local arch="$3"

    local tmp_file="/tmp/divinesense-${arch}"
    local checksum_url="${url}.sha256"
    local tmp_checksum="/tmp/divinesense-${arch}.sha256"

    log_info "从 $url 下载..."
    if ! curl -fsSL --connect-timeout $CURL_CONNECT_TIMEOUT --max-time $CURL_MAX_TIME "$url" -o "$tmp_file"; then
        log_error "下载失败"
        rm -f "$tmp_file" "$tmp_checksum"
        return 1
    fi

    # 尝试下载校验和
    if curl -fsSL --connect-timeout $CURL_CONNECT_TIMEOUT "$checksum_url" -o "$tmp_checksum" 2>/dev/null; then
        if ! verify_checksum "$tmp_file" "$tmp_checksum"; then
            rm -f "$tmp_file" "$tmp_checksum"
            return 1
        fi
    else
        log_warn "校验和文件不可用，跳过验证"
    fi

    mv "$tmp_file" "$output"
    rm -f "$tmp_checksum"
    chmod +x "$output"
    return 0
}

# 获取最新版本（带 v 前缀，如 v0.80.6）
get_latest_version() {
    curl -s --connect-timeout $CURL_CONNECT_TIMEOUT --max-time $CURL_MAX_TIME \
        "https://api.github.com/repos/hrygo/divinesense/releases/latest" | \
        grep -oP '"tag_name":\s*"\K[^"]+' | head -1
}

# 检查服务状态
check_service() {
    if [ ! -f "${INSTALL_DIR}/bin/divinesense" ] && [ ! -f "${INSTALL_DIR}/divinesense" ]; then
        log_error "DivineSense 未安装"
        return 1
    fi
    return 0
}

# 显示完成信息
show_complete() {
    local port="$1"
    local server_ip=$(get_server_ip)

    echo ""
    print_box "安装完成"
    echo ""
    echo -e "  访问: ${YELLOW}http://${server_ip}:${port}${NC}"
    echo ""
    echo -e "  管理:"
    echo -e "    状态: ${CYAN}systemctl status ${SERVICE_NAME}${NC}"
    echo -e "    日志: ${CYAN}journalctl -u ${SERVICE_NAME} -f${NC}"
    echo -e "    重启: ${CYAN}systemctl restart ${SERVICE_NAME}${NC}"
    echo ""
}

# ============================================================================
# 权限配置 ( divine 用户运维权限 )
# ============================================================================

# 获取 divine 用户家目录（动态）
get_divine_home() {
    getent passwd divine 2>/dev/null | cut -d: -f6 || echo "/home/divine"
}

# 验证 divine 用户存在
validate_divine_user() {
    if ! id divine &>/dev/null; then
        log_error "divine 用户不存在，无法配置权限"
        return 1
    fi
    return 0
}

# 配置 docker 组权限
configure_docker_group() {
    # 验证用户存在
    validate_divine_user || return 1

    if ! command -v docker &>/dev/null; then
        log_warn "Docker 未安装，跳过 docker 组配置"
        return 0
    fi

    log_step "配置 docker 组权限..."

    # 确保 docker 组存在
    if ! grep -q "^docker:" /etc/group 2>/dev/null; then
        groupadd docker 2>/dev/null || true
    fi

    # 将 divine 用户添加到 docker 组
    if ! groups divine 2>/dev/null | grep -q docker; then
        usermod -aG docker divine
        log_success "divine 用户已加入 docker 组"
    else
        log_info "divine 用户已在 docker 组中"
    fi
}

# 配置 sudoers 免密 (仅限 DivineSense 运维命令)
configure_sudoers() {
    # 验证用户存在
    validate_divine_user || return 1

    log_step "配置 sudoers 免密..."

    local sudoers_file="/etc/sudoers.d/divinesense"
    local sudoers_dir="/etc/sudoers.d"

    # 验证 sudoers.d 目录存在
    if [ ! -d "$sudoers_dir" ]; then
        log_error "sudoers.d 目录不存在: $sudoers_dir"
        return 1
    fi

    # 写入 sudoers 配置
    cat > "$sudoers_file" << EOF
# DivineSense 运维 - divine 用户免密执行特定命令
# 安全说明: 仅允许管理 divinesense 服务，不包括其他系统操作
divine ALL=(ALL) NOPASSWD: /bin/systemctl status divinesense.service
divine ALL=(ALL) NOPASSWD: /bin/systemctl start divinesense.service
divine ALL=(ALL) NOPASSWD: /bin/systemctl stop divinesense.service
divine ALL=(ALL) NOPASSWD: /bin/systemctl restart divinesense.service
divine ALL=(ALL) NOPASSWD: /bin/journalctl -u divinesense *
# WhatsApp Bridge 管理
divine ALL=(ALL) NOPASSWD: /opt/divinesense/deploy/baileys-bridge.sh *
divine ALL=(ALL) NOPASSWD: /bin/systemctl status baileys-bridge.service
divine ALL=(ALL) NOPASSWD: /bin/systemctl start baileys-bridge.service
divine ALL=(ALL) NOPASSWD: /bin/systemctl stop baileys-bridge.service
divine ALL=(ALL) NOPASSWD: /bin/systemctl restart baileys-bridge.service
divine ALL=(ALL) NOPASSWD: /bin/journalctl -u baileys-bridge *
EOF

    chmod 440 "$sudoers_file"

    # 验证 sudoers 语法
    if ! visudo -c >/dev/null 2>&1; then
        log_error "sudoers 语法验证失败，正在回滚..."
        rm -f "$sudoers_file"
        return 1
    fi

    log_success "sudoers 配置完成"
}

# 创建用户运维 Makefile
create_user_makefile() {
    # 验证用户存在
    validate_divine_user || return 1

    log_step "创建用户运维工具..."

    local divine_home
    divine_home=$(get_divine_home)
    local makefile="${divine_home}/Makefile"

    # 确保家目录存在
    if [ ! -d "$divine_home" ]; then
        log_error "家目录不存在: $divine_home"
        return 1
    fi

    cat > "$makefile" << 'MAKEFILE_EOF'
.PHONY: help status start stop restart logs logs-follow health db-shell db-backup db-restore db-reset upgrade pull-source check-version clone-source source-status whatsapp-status whatsapp-qr whatsapp-logs

# ============================================================
# 配置
# ============================================================
DB_NAME     = divinesense
DB_USER     = divine
DB_CONTAINER= divinesense-postgres
BACKUP_DIR  = /opt/divinesense/backups
SOURCE_DIR  = /home/divine/source/divinesense
SOURCE_REPO = https://github.com/hrygo/divinesense.git
GITHUB_API  = https://api.github.com/repos/hrygo/divinesense/releases/latest

# 从配置文件读取端口（如果存在）
-include /etc/divinesense/config
DIVINESENSE_PORT ?= 5230

# 命令（systemctl 已配置免密 sudo）
SYSTEMCTL   = sudo systemctl
JOURNALCTL  = sudo journalctl

# ============================================================
# 默认目标：显示帮助
# ============================================================
help:
	@echo "DivineSense 运维工具"
	@echo ""
	@echo "服务管理:"
	@echo "  make status          - 查看服务状态"
	@echo "  make start           - 启动服务"
	@echo "  make stop            - 停止服务"
	@echo "  make restart         - 重启服务"
	@echo "  make logs            - 查看日志（最近 50 行）"
	@echo "  make logs-follow     - 实时跟踪日志"
	@echo "  make health          - 健康检查"
	@echo ""
	@echo "数据库管理:"
	@echo "  make db-shell        - 进入数据库 Shell"
	@echo "  make db-backup       - 备份数据库"
	@echo "  make db-restore FILE=<文件> - 恢复数据库"
	@echo "  make db-reset        - 重置数据库（危险！）"
	@echo ""
	@echo "版本管理:"
	@echo "  make check-version   - 检查当前和最新版本"
	@echo "  make upgrade         - 升级到最新版本"
	@echo ""
	@echo "源码管理 (Evolution Mode):"
	@echo "  make clone-source    - 克隆源码仓库"
	@echo "  make pull-source     - 拉取源码更新"
	@echo "  make source-status   - 查看源码状态"
	@echo ""
	@echo "WhatsApp Bridge:"
	@echo "  make whatsapp-status  - 查看 WhatsApp Bridge 状态"
	@echo "  make whatsapp-qr      - 显示配对 QR 码"
	@echo "  make whatsapp-logs    - 查看 WhatsApp Bridge 日志"
	@echo "  make whatsapp-install - 安装 WhatsApp Bridge"

# ============================================================
# 服务管理
# ============================================================
status:
	@echo "=== DivineSense 服务状态 ==="
	@$(SYSTEMCTL) status divinesense --no-pager
	@echo ""
	@echo "=== PostgreSQL 容器状态 ==="
	@docker ps --filter name=$(DB_CONTAINER) --format "table {{.Names}}\t{{.Status}}"

start:
	@echo "启动 DivineSense 服务..."
	@$(SYSTEMCTL) start divinesense
	@echo "服务已启动"
	@make status

stop:
	@echo "停止 DivineSense 服务..."
	@$(SYSTEMCTL) stop divinesense
	@echo "服务已停止"

restart:
	@echo "重启 DivineSense 服务..."
	@$(SYSTEMCTL) restart divinesense
	@echo "服务已重启"
	@make status

logs:
	@echo "=== 最近 50 条日志 ==="
	@$(JOURNALCTL) -u divinesense -n 50 --no-pager

logs-follow:
	@echo "=== 实时日志 (Ctrl+C 退出) ==="
	@$(JOURNALCTL) -u divinesense -f

health:
	@echo "=== 健康检查 ==="
	@echo -n "服务状态: "
	@$(SYSTEMCTL) is-active divinesense 2>/dev/null || echo "unknown"
	@echo -n "HTTP 响应: "
	@curl -s -o /dev/null -w "%{http_code}\n" http://localhost:$(DIVINESENSE_PORT)/health || echo "failed"
	@echo -n "数据库连接: "
	@docker exec $(DB_CONTAINER) pg_isready -U $(DB_USER) 2>/dev/null && echo "OK" || echo "FAILED"

# ============================================================
# 数据库管理
# ============================================================
db-shell:
	@echo "进入 PostgreSQL Shell (输入 \\q 退出)"
	@docker exec -it $(DB_CONTAINER) psql -U $(DB_USER) -d $(DB_NAME)

db-backup:
	@echo "备份数据库..."
	@mkdir -p $(BACKUP_DIR)
	@docker exec $(DB_CONTAINER) pg_dump -U $(DB_USER) $(DB_NAME) | gzip > $(BACKUP_DIR)/divinesense_$$(date +%Y%m%d_%H%M%S).sql.gz
	@echo "备份完成: $(BACKUP_DIR)/divinesense_$$(date +%Y%m%d_%H%M%S).sql.gz"
	@ls -lh $(BACKUP_DIR)/divinesense_$$(date +%Y%m%d_%H%M%S).sql.gz

db-restore:
	@if [ -z "$(FILE)" ]; then \
		echo "错误: 请指定备份文件，例如: make db-restore FILE=divinesense_20260201_120000.sql.gz"; \
		exit 1; \
	fi
	@if [ ! -f "$(FILE)" ]; then \
		echo "错误: 文件不存在: $(FILE)"; \
		exit 1; \
	fi
	@echo "警告: 即将恢复数据库，现有数据将被覆盖！"
	@read -p "确认继续？[y/N] " -n 1 -r; \
	echo; \
	if [[ $$REPLY =~ ^[Yy]$$ ]]; then \
		gunzip -c "$(FILE)" | docker exec -i $(DB_CONTAINER) psql -U $(DB_USER) -d $(DB_NAME); \
		echo "数据库恢复完成"; \
	else \
		echo "已取消"; \
	fi

db-reset:
	@echo "警告: 此操作将删除所有数据！"
	@read -p "确认重置数据库？[y/N] " -n 1 -r; \
	echo; \
	if [[ $$REPLY =~ ^[Yy]$$ ]]; then \
		docker exec -i $(DB_CONTAINER) psql -U $(DB_USER) -d $(DB_NAME) -c "DROP SCHEMA public CASCADE; CREATE SCHEMA public;"; \
		echo "数据库已重置，请重启服务以执行迁移"; \
	else \
		echo "已取消"; \
	fi

# ============================================================
# 版本管理
# ============================================================
check-version:
	@echo "=== 版本信息 ==="
	@echo -n "当前版本: "
	@/opt/divinesense/bin/divinesense --version 2>/dev/null || echo "未知"
	@echo -n "最新版本: "
	@curl -s $(GITHUB_API) | grep '"tag_name"' | sed -E 's/.*"tag_name": *"([^"]+)".*/\1/'

upgrade:
	@echo "=== 升级 DivineSense ==="
	@echo "1. 备份当前数据..."
	@make db-backup
	@echo ""
	@echo "2. 获取最新版本并下载..."
	@cd /tmp && \
		LATEST_VERSION=$$(curl -s $(GITHUB_API) | grep '"tag_name"' | sed -E 's/.*"tag_name": *"([^"]+)".*/\1/') && \
		echo "最新版本: $$LATEST_VERSION" && \
		rm -f divinesense_linux_amd64.tar.gz && \
		curl -sSL https://github.com/hrygo/divinesense/releases/download/$$LATEST_VERSION/divinesense_linux_amd64.tar.gz -o divinesense_linux_amd64.tar.gz
	@echo ""
	@echo "3. 停止服务..."
	@$(SYSTEMCTL) stop divinesense
	@echo ""
	@echo "4. 替换二进制文件..."
	@tar -xzf /tmp/divinesense_linux_amd64.tar.gz -C /opt/divinesense/bin/ --strip-components=1
	@chmod +x /opt/divinesense/bin/divinesense
	@echo ""
	@echo "5. 启动服务..."
	@$(SYSTEMCTL) start divinesense
	@echo ""
	@echo "升级完成！"
	@make status

# ============================================================
# 源码管理 (Evolution Mode)
# ============================================================
clone-source:
	@if [ -d "$(SOURCE_DIR)/.git" ]; then \
		echo "源码已存在，请使用 'make pull-source' 更新"; \
	else \
		echo "克隆源码仓库..."; \
		git clone $(SOURCE_REPO) $(SOURCE_DIR); \
		echo "源码已克隆到: $(SOURCE_DIR)"; \
	fi

pull-source:
	@if [ -d "$(SOURCE_DIR)/.git" ]; then \
		echo "拉取源码更新..."; \
		cd $(SOURCE_DIR) && git pull origin main; \
		echo "源码已更新"; \
	else \
		echo "源码不存在，请先运行 'make clone-source'"; \
	fi

source-status:
	@if [ -d "$(SOURCE_DIR)/.git" ]; then \
		echo "=== 源码状态 ==="; \
		cd $(SOURCE_DIR) && git status; \
		echo ""; \
		echo "=== 最近提交 ==="; \
		cd $(SOURCE_DIR) && git log --oneline -5; \
	else \
		echo "源码不存在，请先运行 'make clone-source'"; \
	fi

# ============================================================
# WhatsApp Bridge 管理
# ============================================================
WHATSAPP_BRIDGE_DIR = /opt/divinesense/plugin/chat_apps/channels/whatsapp/bridge
WHATSAPP_SCRIPT     = /opt/divinesense/deploy/baileys-bridge.sh

whatsapp-status:
	@if [ -f "$(WHATSAPP_SCRIPT)" ]; then \
		$(WHATSAPP_SCRIPT) status; \
	else \
		echo "WhatsApp Bridge 未安装"; \
	fi

whatsapp-qr:
	@if [ -f "$(WHATSAPP_SCRIPT)" ]; then \
		$(WHATSAPP_SCRIPT) qr; \
	else \
		echo "WhatsApp Bridge 未安装"; \
	fi

whatsapp-logs:
	@if [ -f "$(WHATSAPP_SCRIPT)" ]; then \
		$(WHATSAPP_SCRIPT) logs; \
	else \
		echo "WhatsApp Bridge 未安装"; \
	fi

whatsapp-install:
	@if [ -f "$(WHATSAPP_SCRIPT)" ]; then \
		sudo $(WHATSAPP_SCRIPT) install; \
	else \
		echo "找不到 baileys-bridge.sh 脚本"; \
	fi
MAKEFILE_EOF

    chown divine:divine "$makefile"
    log_success "Makefile 运维工具已创建: ~/Makefile"
}

# 配置 bash 别名
configure_bash_aliases() {
    # 验证用户存在
    validate_divine_user || return 1

    log_step "配置 bash 别名..."

    local divine_home
    divine_home=$(get_divine_home)
    local bashrc="${divine_home}/.bashrc"
    local alias_marker="# ===== DivineSense 运维快捷别名 ====="

    # 确保家目录存在
    if [ ! -d "$divine_home" ]; then
        log_warn "家目录不存在: $divine_home，跳过别名配置"
        return 1
    fi

    # 创建 .bashrc 如果不存在
    if [ ! -f "$bashrc" ]; then
        touch "$bashrc"
        chown divine:divine "$bashrc"
    fi

    # 检查是否已配置
    if grep -q "$alias_marker" "$bashrc" 2>/dev/null; then
        log_info "别名已配置"
        return 0
    fi

    cat >> "$bashrc" << EOF

$alias_marker
alias ds='make -C ${divine_home}'
alias ds-help='make -C ${divine_home} help'
alias ds-status='make -C ${divine_home} status'
alias ds-start='make -C ${divine_home} start'
alias ds-stop='make -C ${divine_home} stop'
alias ds-restart='make -C ${divine_home} restart'
alias ds-logs='make -C ${divine_home} logs'
alias ds-health='make -C ${divine_home} health'
alias ds-db='make -C ${divine_home} db-shell'
alias ds-backup='make -C ${divine_home} db-backup'
alias ds-upgrade='make -C ${divine_home} upgrade'
alias ds-pull='make -C ${divine_home} pull-source'
alias ds-wa='make -C ${divine_home} whatsapp-status'
alias ds-qr='make -C ${divine_home} whatsapp-qr'
alias ds-wa-logs='make -C ${divine_home} whatsapp-logs'
EOF

    chown divine:divine "$bashrc"
    log_success "bash 别名已配置 (重新登录生效)"
}
