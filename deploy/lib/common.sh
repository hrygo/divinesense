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

# 获取最新版本
get_latest_version() {
    curl -s --connect-timeout $CURL_CONNECT_TIMEOUT --max-time $CURL_MAX_TIME \
        "https://api.github.com/repos/hrygo/divinesense/releases/latest" | \
        grep -oP '"tag_name":\s*"v?\K[^"]+' | head -1
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
