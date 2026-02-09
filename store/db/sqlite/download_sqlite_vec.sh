#!/bin/bash
#
# download_sqlite_vec.sh
# Downloads sqlite-vec static library from official GitHub releases
#

set -e

VERSION="v0.1.7-alpha.2"
BASE_URL="https://github.com/asg017/sqlite-vec/releases/download"
LIB_DIR=".lib"

# Detect platform
# Priority: GOOS/GOARCH environment variables > uname
if [ -n "${GOOS}" ]; then
    OS="${GOOS}"
else
    OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
fi

if [ -n "${GOARCH}" ]; then
    ARCH="${GOARCH}"
else
    ARCH="$(uname -m)"
fi

# Convert Go OS names to sqlite-vec naming
case "${OS}" in
    darwin)
        OS="macos"
        ;;
    linux)
        OS="linux"
        ;;
esac

# Convert Go ARCH names to sqlite-vec naming
case "${ARCH}" in
    amd64)
        ARCH="x86_64"
        ;;
    arm64)
        ARCH="aarch64"
        ;;
    386)
        echo "Error: 386 architecture is not supported by sqlite-vec"
        exit 1
        ;;
    x86_64|aarch64)
        # Already in correct format
        ;;
    *)
        echo "Unsupported architecture: ${ARCH}"
        exit 1
        ;;
esac

# Remove 'v' prefix from VERSION for filename
VERSION_NO_V="${VERSION#v}"
FILENAME="sqlite-vec-${VERSION_NO_V}-static-${OS}-${ARCH}.tar.gz"
URL="${BASE_URL}/${VERSION}/${FILENAME}"

echo "Downloading sqlite-vec static library..."
echo "URL: ${URL}"
echo "Target: ${LIB_DIR}/libvec0.a"

# Create lib directory
mkdir -p "${LIB_DIR}"

# Download with retry logic
MAX_RETRIES=3
RETRY_DELAY=2

DOWNLOAD_SUCCESS=false
for i in $(seq 1 $MAX_RETRIES); do
    if curl -sL --proto =https "${URL}" | tar -xz -C "${LIB_DIR}" libsqlite_vec0.a 2>/dev/null; then
        DOWNLOAD_SUCCESS=true
        break
    else
        if [ $i -lt $MAX_RETRIES ]; then
            echo "⚠️  Download failed, retrying in ${RETRY_DELAY}s... (attempt $i/$MAX_RETRIES)"
            sleep $RETRY_DELAY
            # Clean up partial download
            rm -f "${LIB_DIR}/libsqlite_vec0.a" 2>/dev/null || true
        else
            echo "❌ Download failed after $MAX_RETRIES attempts"
            exit 1
        fi
    fi
done

if [ "$DOWNLOAD_SUCCESS" = false ]; then
    echo "❌ Failed to download sqlite-vec static library"
    exit 1
fi

# Verify downloaded file
if [ ! -s "${LIB_DIR}/libsqlite_vec0.a" ]; then
    echo "❌ Downloaded file is empty or corrupted"
    exit 1
fi

# Check if it's a valid ar archive (static library)
if ! file "${LIB_DIR}/libsqlite_vec0.a" 2>/dev/null | grep -q "archive"; then
    if ! ar t "${LIB_DIR}/libsqlite_vec0.a" >/dev/null 2>&1; then
        echo "❌ Downloaded file is not a valid static library"
        echo "   Expected: ar archive format"
        echo "   Got: $(file "${LIB_DIR}/libsqlite_vec0.a" 2>/dev/null || echo "unknown")"
        rm -f "${LIB_DIR}/libsqlite_vec0.a"
        exit 1
    fi
fi

# Rename to expected name
mv "${LIB_DIR}/libsqlite_vec0.a" "${LIB_DIR}/libvec0.a"

echo "✓ Downloaded successfully: ${LIB_DIR}/libvec0.a"
ls -lh "${LIB_DIR}/libvec0.a"
