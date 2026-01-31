#!/bin/bash
# =============================================================================
# DivineSense Release Build Script
# =============================================================================
#
# Cross-compiles DivineSense for multiple platforms with embedded frontend.
#
# Usage:
#   ./scripts/release/build-release.sh [version]
#
# Platforms:
#   - linux/amd64, linux/arm64
#   - darwin/amd64, darwin/arm64
#   - windows/amd64, windows/arm64
#
# Output:
#   dist/divinesense-<version>-<platform>
#
# =============================================================================

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
DIST_DIR="${PROJECT_ROOT}/dist"
VERSION="${1:-dev}"
BUILD_TIME="$(date -u '+%Y-%m-%d_%H:%M:%S')"
LDFLAGS="-X main.Version=${VERSION} -X main.BuildTime=${BUILD_TIME} -s -w"

# Supported platforms
PLATFORMS=("linux/amd64" "linux/arm64" "darwin/amd64" "darwin/arm64" "windows/amd64" "windows/arm64")

# Logging
log_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
log_success() { echo -e "${GREEN}[OK]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

# Print banner
print_banner() {
    echo ""
    echo -e "${BLUE}╔════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${BLUE}║${NC}  ${GREEN}DivineSense Release Build${NC}                                  ${BLUE}║${NC}"
    echo -e "${BLUE}╚════════════════════════════════════════════════════════════╝${NC}"
    echo ""
    log_info "Version: ${VERSION}"
    log_info "Build Time: ${BUILD_TIME}"
    echo ""
}

# Check dependencies
check_dependencies() {
    log_info "Checking dependencies..."

    # Check Go
    if ! command -v go &>/dev/null; then
        log_error "Go is not installed"
        exit 1
    fi

    local go_version=$(go version | grep -oP 'go[0-9.]+' | head -1)
    log_success "Go: $(go version | grep -oP 'go[0-9.]+' | head -1)"

    # Check pnpm/npm
    if ! command -v pnpm &>/dev/null && ! command -v npm &>/dev/null; then
        log_error "Neither pnpm nor npm is installed"
        exit 1
    fi

    if command -v pnpm &>/dev/null; then
        log_success "Frontend: pnpm $(pnpm --version)"
    else
        log_success "Frontend: npm $(npm --version)"
    fi

    # Check for cross-compilation tools
    if ! go env | grep -q "CC"; then
        log_warn "Cross-compilation C compiler not configured"
        log_info "For ARM64 builds on macOS: brew install gnu-coreutils"
        log_info "For ARM64 builds on Linux: apt-get install gcc-aarch64-linux-gnu"
    fi
}

# Build frontend
build_frontend() {
    log_info "Building frontend..."

    cd "${PROJECT_ROOT}/web"

    # Install dependencies if needed
    if [ ! -d "node_modules" ]; then
        log_info "Installing frontend dependencies..."
        if command -v pnpm &>/dev/null; then
            pnpm install --frozen-lockfile
        else
            npm install
        fi
    fi

    # Build production bundle
    if command -v pnpm &>/dev/null; then
        pnpm build
    else
        npm run build
    fi

    log_success "Frontend built to web/dist/"
}

# Build backend for a platform
build_platform() {
    local platform=$1
    local GOOS=$(echo $platform | cut -d'/' -f1)
    local GOARCH=$(echo $platform | cut -d'/' -f2)
    local output_name="divinesense-${VERSION}-${GOOS}-${GOARCH}"
    if [ "$GOOS" == "windows" ]; then
        output_name="${output_name}.exe"
    fi
    local output_path="${DIST_DIR}/${output_name}"

    log_info "Building for ${platform}..."

    # Set cross-compilation environment
    export GOOS=$GOOS
    export GOARCH=$GOARCH
    export CGO_ENABLED=0

    # Build
    cd "${PROJECT_ROOT}"
    go build -ldflags "${LDFLAGS}" -o "${output_path}" ./cmd/divinesense

    # Verify build
    if [ -f "${output_path}" ]; then
        local size=$(du -h "${output_path}" | cut -f1)
        log_success "Built ${platform} → ${output_name} (${size})"
    else
        log_error "Failed to build ${platform}"
        return 1
    fi

    # Create checksum
    if command -v sha256sum &>/dev/null; then
        cd "${DIST_DIR}"
        sha256sum "${output_name}" >> "${DIST_DIR}/checksums.txt"
    elif command -v shasum &>/dev/null; then
        cd "${DIST_DIR}"
        shasum -a 256 "${output_name}" >> "${DIST_DIR}/checksums.txt"
    fi
}

# Main build process
main() {
    print_banner
    check_dependencies

    # Clean and create dist directory
    rm -rf "${DIST_DIR}"
    mkdir -p "${DIST_DIR}"

    # Build frontend first
    build_frontend

    # Embed frontend into Go
    log_info "Embedding frontend assets..."

    # Build for each platform
    for platform in "${PLATFORMS[@]}"; do
        build_platform "$platform" || exit 1
    done

    # Copy systemd service file
    log_info "Copying service files..."
    cp "${SCRIPT_DIR}/divinesense.service" "${DIST_DIR}/"

    echo ""
    log_success "=========================================="
    log_success "Release build complete!"
    log_success "=========================================="
    echo ""
    log_info "Output directory: ${DIST_DIR}"
    echo ""
    log_info "Artifacts:"
    ls -lh "${DIST_DIR}" | grep -v "^total" | grep -v "^d" | awk '{printf "  %-40s %s\n", $9, $5}'
    echo ""

    if [ -f "${DIST_DIR}/checksums.txt" ]; then
        log_info "Checksums:"
        cat "${DIST_DIR}/checksums.txt"
    fi
}

main "$@"
