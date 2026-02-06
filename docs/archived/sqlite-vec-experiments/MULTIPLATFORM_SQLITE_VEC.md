# SQLite AI 支持的多平台编译与分发策略

> 分析日期: 2026-02-04
> 问题: 如何在多平台架构下编译 sqlite-vec 并保证二进制制品架构兼容

---

## 📋 问题分析

### 当前实现的核心挑战

#### 1. CGO 依赖问题

**问题**: `scripts/release/build-release.sh:134`
```bash
export CGO_ENABLED=0  # ❌ 禁用 CGO
```

**影响**:
- ❌ 当前发布版本**完全不支持 AI 功能**
- ❌ sqlite-vec 扩展无法加载（需要 CGO）
- ❌ 向量搜索功能缺失
- ✅ 只支持基础功能（笔记、日程）

#### 2. 扩展分发问题

**问题**: sqlite-vec 是动态库，无法嵌入 Go 二进制

```go
// 当前实现：硬编码本地路径
extensionPaths := []string{
    "./internal/sqlite-vec/libvec0.dylib",  // ❌ 开发环境专用
    "/usr/local/lib/libvec0.dylib",        // ❌ 用户需手动安装
}
```

**影响**:
- 🔴 **单二进制分发失败** - 扩展文件必须单独分发
- 🔴 **用户体验差** - 需要手动安装依赖
- 🔴 **跨平台复杂** - 每个平台需要不同的扩展文件

#### 3. 多平台编译限制

**当前支持平台**:
| 平台 | CGO | sqlite-vec | 状态 |
|:-----|:----|:-----------|:-----|
| linux/amd64 | ❌ | ❌ | 基础功能 |
| linux/arm64 | ❌ | ❌ | 基础功能 |
| darwin/amd64 | ❌ | ❌ | 基础功能 |
| darwin/arm64 | ✅ (dev) | ✅ (dev) | 完整 AI 功能 |
| windows/amd64 | ❌ | ❌ | 基础功能 |

---

## 🔧 解决方案

### 方案 1: 静态链接 sqlite-vec（推荐）⭐

**原理**: 将 sqlite-vec 编译为静态库，在 Go 编译时静态链接

**优点**:
- ✅ **真正单二进制** - 扩展代码直接编译进 Go
- ✅ **跨平台支持** - 使用 Zig 交叉编译
- ✅ **用户体验** - 无需额外依赖

**缺点**:
- ⚠️ Go 交叉编译 CGO 复杂（需要 Zig）
- ⚠️ 二进制体积增加（约 +1-2MB）

**实现步骤**:

#### 1.1 修改构建脚本

```bash
# scripts/release/build-release.sh
build_platform() {
    local platform=$1
    local GOOS=$(echo $platform | cut -d'/' -f1)
    local GOARCH=$(echo $platform | cut -d'/' -f2)

    # ✅ 启用 CGO
    export CGO_ENABLED=1

    # ✅ 使用 Zig 作为交叉编译器
    export CC=zig cc
    export CXX=zig c++

    # ✅ 静态链接 sqlite-vec
    local SQLITE_VEC_ARCHIVE="${PROJECT_ROOT}/internal/sqlite-vec/libvec0.a"

    if [ ! -f "$SQLITE_VEC_ARCHIVE" ]; then
        log_error "sqlite-vec static library not found: $SQLITE_VEC_ARCHIVE"
        log_error "Please run: make build-sqlite-vec"
        exit 1
    fi

    # ✅ 添加 LDFLAGS 指定静态库
    local EXTRA_LDFLAGS="-L${PROJECT_ROOT}/internal/sqlite-vec -lvec0"

    go build -tags sqlite_vec_static \
        -ldflags "${LDFLAGS} ${EXTRA_LDFLAGS}" \
        -o "${output_path}" \
        ./cmd/divinesense
}
```

#### 1.2 使用 Zig 交叉编译

**安装 Zig**:
```bash
# macOS
brew install zig

# Linux
curl -O https://ziglang.org/builds/zig/linux-x86_64-0.13.0.tar.xz
tar xf zig-linux-x86_64-0.13.0.tar.xz
export PATH=$PATH:$(pwd)/zig-linux-x86_64-0.13.0
```

**构建静态库**:
```bash
# scripts/build-sqlite-vec-static.sh
#!/bin/bash

set -e

OS=$1  # linux|darwin|windows
ARCH=$2  # amd64|arm64

echo "Building sqlite-vec for ${OS}/${ARCH}..."

# 下载 SQLite amalgamation
curl -sL https://sqlite.org/2024/sqlite-amalgamation-3470200.zip -o sqlite.zip
unzip -q sqlite.zip

# 编译 SQLite
zig cc -target ${ARCH}-linux-musl -fPIC -DSQLITE_ENABLE_FTS5 \
    -c sqlite-amalgamation-3470200/sqlite3.c -o sqlite3.o

# 下载并编译 sqlite-vec
git clone --depth 1 https://github.com/asg017/sqlite-vec.git
cd sqlite-vec
envsubst < sqlite-vec.h.tmpl > sqlite-vec.h

zig cc -target ${ARCH}-linux-musl -fPIC -c sqlite-vec.c -o sqlite-vec.o
zig ar rcs libvec0.a sqlite3.o sqlite-vec.o

# 复制到项目目录
mkdir -p ../../internal/sqlite-vec/
cp libvec0.a ../../internal/sqlite-vec/

echo "✅ Built: internal/sqlite-vec/libvec0.a"
```

**构建多平台静态库**:
```bash
# Linux amd64
./scripts/build-sqlite-vec-static.sh linux amd64

# Linux arm64
./scripts/build-sqlite-vec-static.sh linux arm64

# macOS amd64 (交叉编译)
./scripts/build-sqlite-vec-static.sh darwin amd64

# macOS arm64 (交叉编译)
./scripts/build-sqlite-vec-static.sh darwin arm64
```

#### 1.3 Go 代码修改

**添加 build tag 文件**: `store/db/sqlite/sqlite_vec_static.go`
```go
//go:build sqlite_vec_static

package sqlite

/*
#cgo CFLAGS: -I../../internal/sqlite-vec
#cgo LDFLAGS: -L../../internal/sqlite-vec -lvec0

#include "sqlite-vec.h"
*/
import "C"
```

**在 Windows/Linux 上使用动态链接**: `store/db/sqlite/sqlite_vec_dynamic.go`
```go
//go:build !sqlite_vec_static

package sqlite

import (
    "context"
    "database/sql"
    "fmt"
    "log/slog"

    _ "github.com/mattn/go-sqlite3"
    "github.com/mattn/go-sqlite3"
)

// loadExtension loads sqlite-vec from dynamic library
func loadExtension(db *sql.DB, extensionPath string) error {
    conn, err := db.Conn(context.Background())
    if err != nil {
        return fmt.Errorf("failed to get connection: %w", err)
    }
    defer conn.Close()

    err = conn.Raw(func(driverConn interface{}) error {
        sqliteConn, ok := driverConn.(*sqlite3.SQLiteConn)
        if !ok {
            return fmt.Errorf("unexpected driver connection type: %T", driverConn)
        }
        return sqliteConn.LoadExtension(extensionPath, "sqlite3_vec_init")
    })

    return err
}
```

---

### 方案 2: 分发架构（Docker 模式）

**原理**: Docker 容器内包含扩展，用户运行容器

**优点**:
- ✅ 解决依赖问题
- ✅ 隔离运行环境
- ✅ 易于更新

**缺点**:
- ❌ 不是单二进制
- ❌ Docker 开销
- ❌ Geek Mode 不友好

**实现**:

#### 2.1 修改 docker/builder.Dockerfile

```dockerfile
# 多阶段构建 - 每个平台单独构建

# ========== Linux amd64 ==========
FROM ubuntu:22.04 AS builder-linux-amd64
RUN # ... 编译 sqlite-vec 静态库 ...
FROM golang:1.25 AS builder-go-linux-amd64
COPY --from=builder-linux-amd64 /usr/local/lib/libvec0.a /usr/local/lib/
RUN CGO_ENABLED=1 go build -o divinesense-linux-amd64 ./cmd/divinesense
FROM alpine:latest
COPY --from=builder-go-linux-amd64 /tmp/divinesense /app/divinesense

# ========== Linux arm64 ==========
FROM --platform=linux/arm64 ubuntu:22.04 AS builder-linux-arm64
RUN # ... 编译 sqlite-vec 静态库 ...
FROM --platform=linux/arm64 golang:1.25 AS builder-go-linux-arm64
COPY --from=builder-linux-arm64 /usr/local/lib/libvec0.a /usr/local/lib/
RUN CGO_ENABLED=1 go build -o divinesense-linux-arm64 ./cmd/divinesense
FROM alpine:latest
COPY --from=builder-go-linux-arm64 /tmp/divinesense /app/divinesense
```

#### 2.2 构建多架构镜像

```bash
# 构建并推送到 Docker Hub
docker buildx build --platform linux/amd64,linux/arm64 \
    -t hrygo/divinesense:latest \
    --push .
```

---

### 方案 3: Go fallback 模式（当前实现）

**原理**: 不使用 sqlite-vec，在 Go 层实现向量搜索

**优点**:
- ✅ 单二进制
- ✅ 跨平台
- ✅ 无依赖

**缺点**:
- ❌ 性能差（O(n) vs O(log n)）
- ❌ 内存占用高
- ❌ 不适合大数据集

**当前状态**: ✅ 已实现作为 fallback

---

## 📊 方案对比

| 方案 | 单二进制 | 跨平台 | 性能 | 复杂度 | 推荐度 |
|:-----|:--------|:-------|:-----|:--------|:------:|
| **静态链接** | ✅ | ✅ | ⭐⭐⭐⭐⭐ | 🔴 高 | ⭐⭐⭐⭐⭐ |
| Docker 模式 | ❌ | ✅ | ⭐⭐⭐⭐ | 🟡 中 | ⭐⭐⭐ |
| Go Fallback | ✅ | ✅ | ⭐⭐ | 🟢 低 | ⭐⭐ |

---

## 🎯 推荐实施路径

### 短期（1-2 周）- Docker 模式

1. ✅ 使用现有 `docker/builder.Dockerfile`
2. ✅ 发布 `hrygo/divinesense:full-ai` 镜像
3. ✅ 支持 Linux amd64/arm64
4. ✅ 文档说明 Docker 模式支持完整 AI

### 中期（1-2 月）- 静态链接

1. ⚠️ 学习 Zig 交叉编译
2. ⚠️ 构建多平台静态库（6 个平台 × 2 架构）
3. ⚠️ 修改构建脚本支持 CGO
4. ⚠️ 测试单二进制分发

### 长期（2-3 月）- 混合模式

1. **开发环境**: 本地动态库（当前）
2. **Docker 用户**: 容器内完整 AI 支持
3. **二进制用户**: 静态链接完整 AI 支持
4. **无 AI 用户**: 当前 `CGO_ENABLED=0` 版本

---

## 🛠️ 技术细节

### Zig 交叉编译示例

```bash
# 设置 Zig 编译器
export CC=zig cc
export CXX=zig c++
export CGO_ENABLED=1

# 交叉编译到 Linux arm64
export GOOS=linux GOARCH=arm64
export CGO_LDFLAGS="-target aarch64-linux-musl"

go build -o divinesense-linux-arm64 ./cmd/divinesense
```

### 静态库集成

```c
// internal/sqlite-vec/sqlite3_vec_init.c
#include "sqlite-vec.h"

// 入口点
#ifdef _WIN32
  __declspec(dllexport)
#endif
int sqlite3_vec_init(sqlite3 *db, char **pzErrMsg, const sqlite3_api_routines *pApi) {
    // 初始化代码
    return SQLITE_OK;
}
```

---

## 📝 结论

### 当前问题
- 🔴 发布版本**完全不支持 AI 功能**（CGO_ENABLED=0）
- 🔴 只有开发环境有完整 AI 支持
- 🔴 缺乏多平台编译策略

### 推荐方案
1. **短期**: Docker 模式（快速上线）
2. **中期**: 静态链接（单二进制）
3. **长期**: 混合模式（用户自选）

### 优先级
- 🔥 **P0**: 修复 `build-release.sh` 的 CGO_ENABLED=0
- 🔥 **P0**: 实现静态链接方案
- 🟡 **P1**: 完善多平台静态库构建
- 🟢 **P2**: 提供无 AI 版本（减少二进制体积）

---

**分析完成** ✅
**下一步**: 选择实施方案，制定详细计划
