#!/usr/bin/env bash
#
# DivineSense Interactive Deployment Wizard
# Self-contained installer - works on zero-base Linux servers
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/hrygo/divinesense/main/deploy/interactive/wizard.sh | sudo bash
#

set -e

# Colors for output
readonly RED='\033[0;31m'
readonly GREEN='\033[0;32m'
readonly YELLOW='\033[1;33m'
readonly BLUE='\033[0;34m'
readonly CYAN='\033[0;36m'
readonly NC='\033[0m' # No Color

# Configuration
readonly REPO="hrygo/divinesense"
readonly BRANCH="${WIZARD_BRANCH:-main}"
readonly WORK_DIR="/tmp/divinesense-wizard-$$"

# Cleanup handler
cleanup() {
    local exit_code=$?
    if [[ "${exit_code}" -ne 0 ]]; then
        error "Wizard exited with error code ${exit_code}"
    fi
    rm -rf "${WORK_DIR}" 2>/dev/null || true
    exit ${exit_code}
}
trap cleanup EXIT INT TERM

# Print message with color
info() { echo -e "${BLUE}[INFO]${NC} $*"; }
success() { echo -e "${GREEN}[OK]${NC} $*"; }
warn() { echo -e "${YELLOW}[WARN]${NC} $*"; }
error() { echo -e "${RED}[ERROR]${NC} $*"; }
step() { echo -e "${CYAN}[STEP]${NC} $*"; }

# Check if running as root
check_root() {
    if [[ $EUID -ne 0 ]]; then
        error "This script must be run as root (use sudo)"
        exit 1
    fi
}

# Detect OS
detect_os() {
    if [[ -f /etc/os-release ]]; then
        . /etc/os-release
        OS="${ID}"
        OS_VERSION="${VERSION_ID}"
    else
        error "Cannot detect OS. Only Linux is supported."
        exit 1
    fi
    info "Detected OS: ${OS} ${OS_VERSION}"
}

# Detect architecture
detect_arch() {
    local arch=$(uname -m)
    case "${arch}" in
        x86_64) ARCH="amd64" ;;
        aarch64) ARCH="arm64" ;;
        *) error "Unsupported architecture: ${arch}"; exit 1 ;;
    esac
    info "Detected architecture: ${ARCH}"
}

# Install required packages
install_dependencies() {
    step "Installing dependencies..."

    case "${OS}" in
        alpine)
            apk add --no-cache curl ca-certificates bash git 2>/dev/null
            ;;
        ubuntu|debian)
            export DEBIAN_FRONTEND=noninteractive
            apt-get update -qq 2>/dev/null || true
            apt-get install -y -qq curl ca-certificates git 2>/dev/null
            ;;
        centos|rhel|rocky|almalinux)
            if command -v dnf &>/dev/null; then
                dnf install -y -q curl ca-certificates git 2>/dev/null
            else
                yum install -y -q curl ca-certificates git 2>/dev/null
            fi
            ;;
        fedora)
            dnf install -y -q curl ca-certificates git 2>/dev/null
            ;;
        *)
            warn "Unknown OS: ${OS}, hoping curl and git are available..."
            ;;
    esac
    success "Dependencies ready"
}

# Download and build the wizard
build_wizard() {
    mkdir -p "${WORK_DIR}"
    cd "${WORK_DIR}"

    step "Downloading wizard source..."

    # Try shallow clone for speed
    if git clone --depth 1 --single-branch --branch "${BRANCH}" \
        "https://github.com/${REPO}.git" divinesense 2>/dev/null; then
        success "Source downloaded"
    else
        error "Failed to clone repository"
        info "Please check your internet connection and try again"
        exit 1
    fi

    cd divinesense

    # Check if Go exists
    if command -v go &>/dev/null; then
        info "Using existing Go $(go version | awk '{print $3}')"
    else
        step "Installing Go temporarily..."
        install_go_temp
    fi

    step "Building wizard (this may take 1-2 minutes)..."

    # Build the wizard
    if go build -o /tmp/divinesense-wizard ./cmd/deploy-wizard/main.go 2>/dev/null; then
        mv /tmp/divinesense-wizard "${WORK_DIR}/wizard"
        success "Wizard built successfully"
    else
        error "Build failed"
        info "Go version: $(go version 2>/dev/null || echo 'not installed')"
        exit 1
    fi

    # Cleanup Go if we installed it
    if [[ "${TEMP_GO}" == "1" ]]; then
        cleanup_temp_go
    fi

    # Cleanup source
    rm -rf "${WORK_DIR}/divinesense"
}

# Install Go temporarily to /tmp/go-temp
install_go_temp() {
    local GO_VERSION="1.23.6"
    local GO_ARCH="${ARCH}"
    unset TEMP_GO

    local GO_URL="https://go.dev/dl/go${GO_VERSION}.linux-${GO_ARCH}.tar.gz"
    local GO_INSTALL_DIR="/tmp/go-temp-$$"

    info "Downloading Go ${GO_VERSION} for ${GO_ARCH}..."
    mkdir -p "${GO_INSTALL_DIR}"

    if ! curl -fsSL "${GO_URL}" | tar -xz -C "${GO_INSTALL_DIR}" 2>/dev/null; then
        error "Failed to download Go"
        exit 1
    fi

    export PATH="${GO_INSTALL_DIR}/go/bin:${PATH}"
    TEMP_GO="1"
    TEMP_GO_DIR="${GO_INSTALL_DIR}"
}

cleanup_temp_go() {
    if [[ -n "${TEMP_GO_DIR}" ]]; then
        rm -rf "${TEMP_GO_DIR}"
    fi
}

# Run the wizard
run_wizard() {
    cd "${WORK_DIR}"
    chmod +x wizard

    echo ""
    echo -e "${BLUE}═══════════════════════════════════════════════════════${NC}"
    echo -e "${CYAN}         DivineSense Deployment Wizard${NC}"
    echo -e "${BLUE}═══════════════════════════════════════════════════════${NC}"
    echo ""

    ./wizard

    echo ""
    echo -e "${BLUE}═══════════════════════════════════════════════════════${NC}"
    success "Wizard completed!"
    echo -e "${BLUE}═══════════════════════════════════════════════════════${NC}"
}

# Main execution
main() {
    clear
    echo -e "${CYAN}"
    echo "╔═══════════════════════════════════════════════════════╗"
    echo "║     DivineSense Interactive Deployment Wizard         ║"
    echo "║     Version 1.0.0                                     ║"
    echo "║                                                      ║"
    echo "║     This will guide you through installing           ║"
    echo "║     DivineSense on your server.                       ║"
    echo "╚═══════════════════════════════════════════════════════╝"
    echo -e "${NC}"
    echo ""

    check_root
    detect_os
    detect_arch
    install_dependencies
    build_wizard
    run_wizard
}

# Run main
main "$@"
