# SQLite-Vec 使用指南

> DivineSense SQLite AI 支持的完整使用说明

---

## 📋 目录

1. [快速开始](#快速开始)
2. [两种模式对比](#两种模式对比)
3. [本地开发](#本地开发)
4. [生产构建](#生产构建)
5. [CI/CD 自动化](#cicd-自动化)
6. [故障排查](#故障排查)

---

## 快速开始

### 默认模式（无 AI）

```bash
# 直接构建（无 tag）
go build -o divinesense ./cmd/divinesense

# 或使用 Makefile
make build
```

**产物**: 52MB 二进制，无 AI 功能

### AI 模式（启用 sqlite-vec）

```bash
# 1. 下载静态库（首次）
cd store/db/sqlite
go generate -v ./...

# 2. 构建（带 tag）
cd -
go build -tags sqlite_vec -o divinesense ./cmd/divinesense

# 或使用 Makefile（如果已配置）
make build-ai
```

**产物**: 55MB 二进制，包含完整 AI 功能

---

## 两种模式对比

### 模式 1: 默认模式（无 tag）- **动态库加载**

**构建命令**:
```bash
go build -o divinesense ./cmd/divinesense
```

**特性**:
- ✅ 无需 CGO（如果使用 modernc.org/sqlite）
- ✅ 构建简单，无需下载静态库
- ⚠️  AI 功能**取决于**运行时动态库

**AI 支持条件**:
```
1. 系统安装了 sqlite-vec 动态库：
   - macOS: brew install sqlite-vec
   - Linux: 从源码编译或下载 .so 文件

2. 或本地构建了动态库：
   - ./store/db/sqlite/.lib/libvec0.dylib (macOS)
   - ./store/db/sqlite/.lib/libvec0.so (Linux)

3. 运行时自动尝试加载这些路径
```

**降级策略**:
- 扩展加载失败 → 使用 Go 应用层向量搜索（慢，但可用）
- 适合开发环境快速测试

### 模式 2: AI 模式（带 tag）- **静态链接**

**构建命令**:
```bash
# 1. 首次使用：下载静态库
cd store/db/sqlite && go generate -v ./...

# 2. 构建（带 tag）
cd -
go build -tags sqlite_vec -o divinesense ./cmd/divinesense
```

**特性**:
- ✅ 完整 AI 支持（vec0 虚拟表，O(log n) 搜索）
- ✅ 静态链接，无需运行时依赖
- ✅ 单二进制分发
- ⚠️ 需要 CGO（编译时依赖）

**AI 支持**:
- 始终启用（内置在二进制中）
- 使用官方 sqlite-vec v0.1.7-alpha.2
- 向量搜索性能最优

**适用场景**:
- 生产环境部署
- Geek Mode（极客模式）
- 单二进制分发

---

## 本地开发

### 场景 1: 快速开发（不需要 AI）

```bash
# 克隆仓库
git clone https://github.com/hrygo/divinesense.git
cd divinesense

# 安装依赖
make deps

# 启动服务（默认模式）
make start
```

**说明**:
- 使用默认 SQLite 驱动
- 无需 CGO
- 无需下载静态库
- 快速启动开发

### 场景 2: 开发 AI 功能

#### 步骤 1: 环境准备

```bash
# 1. 检查 CGO 是否可用
go env CGO_ENABLED
# 输出应该是 "1" 或空（默认启用）

# 2. 检查编译器（需要 gcc 或 clang）
# macOS: 通常已安装 Xcode Command Line Tools
# Linux: sudo apt install build-essential
```

#### 步骤 2: 下载静态库

```bash
cd store/db/sqlite

# 运行 go generate 下载官方静态库
go generate -v ./...

# 输出示例:
# Downloading sqlite-vec static library...
# URL: https://github.com/asg017/sqlite-vec/releases/download/v0.1.7-alpha.2/...
# ✓ Downloaded successfully: .lib/libvec0.a
# -rw-r--r--  1 xiaobingyang  staff   157K Jan  11  2025 .lib/libvec0.a

# 验证文件存在
ls -lh .lib/libvec0.a
```

**下载的文件**:
```
store/db/sqlite/.lib/libvec0.a  (157KB)
```

**平台检测**:
- macOS ARM64: `sqlite-vec-0.1.7-alpha.2-static-macos-aarch64.tar.gz`
- macOS Intel: `sqlite-vec-0.1.7-alpha.2-static-macos-x86_64.tar.gz`
- Linux ARM64: `sqlite-vec-0.1.7-alpha.2-static-linux-aarch64.tar.gz`
- Linux AMD64: `sqlite-vec-0.1.7-alpha.2-static-linux-x86_64.tar.gz`

#### 步骤 3: 构建 AI 版本

```bash
# 返回项目根目录
cd -

# 构建 AI 版本
go build -tags sqlite_vec -o divinesense ./cmd/divinesense

# 验证
ls -lh divinesense
# -rwxr-xr-x  1 user  staff   55M Feb  6 15:30 divinesense
```

#### 步骤 4: 运行和验证

```bash
# 启动服务
./divinesense --driver sqlite --data ./data

# 查看日志，确认 sqlite-vec 加载成功
# 日志应该显示:
# INFO sqlite-vec extension verified (static linking) functions_found=18
```

#### 步骤 5: 测试 AI 功能

```bash
# 使用 Web UI 或 API 测试向量搜索
# 例如：创建笔记并搜索

# API 示例:
curl -X POST http://localhost:28081/api/v1/memo \
  -H "Content-Type: application/json" \
  -d '{"content": "测试向量搜索功能"}'

curl -X POST http://localhost:28081/api/v1/chat \
  -H "Content-Type: application/json" \
  -d '{"content": "搜索关于向量的笔记"}'
```

---

## 生产构建

### 场景 1: 多平台 Release 构建

#### 方法 A: 使用构建脚本（推荐）

```bash
# 1. 确保已安装 Zig（用于交叉编译）
brew install zig  # macOS
# 或访问 https://ziglang.org/

# 2. 下载所有平台的静态库
cd store/db/sqlite

# macOS ARM64 (当前平台)
go generate -v ./...

# macOS Intel (交叉编译)
GOOS=darwin GOARCH=amd64 go generate -v ./...

# Linux AMD64 (交叉编译)
GOOS=linux GOARCH=amd64 go generate -v ./...

# Linux ARM64 (交叉编译)
GOOS=linux GOARCH=arm64 go generate -v ./...
```

**问题**: `go generate` 会在**当前平台**下载，无法直接交叉编译下载。

**解决方案**: 使用 GitHub Actions 自动下载（见 CI/CD 章节）

#### 方法 B: 使用 GitHub Actions（自动化）

**文件**: `.github/workflows/build-multi-platform.yml`

```yaml
name: Build with AI Support

on:
  push:
    tags: ['v*']

jobs:
  build:
    runs-on: ${{ matrix.os }}
    strategy:
      matrix:
        include:
          - os: ubuntu-latest
            goos: linux
            goarch: amd64
          - os: ubuntu-latest
            goos: linux
            goarch: arm64
          - os: macos-latest
            goos: darwin
            goarch: arm64
          - os: macos-latest
            goos: darwin
            goarch: amd64

    steps:
      - uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.25'

      - name: Download sqlite-vec static library
        run: |
          cd store/db/sqlite
          go generate -v ./...
        env:
          GOOS: ${{ matrix.goos }}
          GOARCH: ${{ matrix.goarch }}

      - name: Build with AI support
        run: |
          go build -tags sqlite_vec -ldflags "-s -w" -o divinesense-${{ matrix.goos }}-${{ matrix.goarch }} ./cmd/divinesense
        env:
          GOOS: ${{ matrix.goos }}
          GOARCH: ${{ matrix.goarch }}
          CGO_ENABLED: 1

      - name: Upload artifact
        uses: actions/upload-artifact@v4
        with:
          name: divinesense-${{ matrix.goos }}-${{ matrix.goarch }}
          path: divinesense-${{ matrix.goos }}-${{ matrix.goarch }}
```

**运行**:
```bash
# 推送 tag 触发构建
git tag v0.1.0
git push origin v0.1.0

# 或手动触发（GitHub Web UI）
```

### 场景 2: 单平台构建（本地发布）

#### macOS ARM64（示例）

```bash
# 1. 下载静态库
cd store/db/sqlite
go generate -v ./...

# 2. 构建
cd -
CGO_ENABLED=1 go build -tags sqlite_vec \
  -ldflags "-s -w -X main.Version=v0.1.0" \
  -o divinesense-macos-arm64 \
  ./cmd/divinesense

# 3. 验证
ls -lh divinesense-macos-arm64
# -rwxr-xr-x  1 user  staff   55M Feb  6 15:30 divinesense-macos-arm64

# 4. 测试
./divinesense-macos-arm64 --help
```

#### Linux AMD64（在 macOS 上交叉编译）

```bash
# 1. 安装 Zig
brew install zig

# 2. 设置环境
export GOOS=linux
export GOARCH=amd64
export CC=zig cc
export CGO_ENABLED=1

# 3. 下载静态库（需要手动）
# 下载: https://github.com/asg017/sqlite-vec/releases/download/v0.1.7-alpha.2/sqlite-vec-0.1.7-alpha.2-static-linux-x86_64.tar.gz
mkdir -p store/db/sqlite/.lib
curl -sL https://github.com/asg017/sqlite-vec/releases/download/v0.1.7-alpha.2/sqlite-vec-0.1.7-alpha.2-static-linux-x86_64.tar.gz | \
  tar -xz -C store/db/sqlite/.lib libsqlite_vec0.a
mv store/db/sqlite/.lib/libsqlite_vec0.a store/db/sqlite/.lib/libvec0.a

# 4. 构建
go build -tags sqlite_vec \
  -ldflags "-s -w" \
  -o divinesense-linux-amd64 \
  ./cmd/divinesense

# 5. 验证
file divinesense-linux-amd64
# divinesense-linux-amd64: ELF 64-bit LSB executable, x86-64, ...
```

---

## CI/CD 自动化

### GitHub Actions 完整流程

**文件**: `.github/workflows/build-multi-platform.yml`

```yaml
name: Build Multi-Platform with AI

on:
  push:
    branches: [main]
    tags: ['v*']
  pull_request:
    branches: [main]

jobs:
  build:
    name: Build ${{ matrix.goos }}/${{ matrix.goarch }}
    runs-on: ${{ matrix.os }}

    strategy:
      fail-fast: false
      matrix:
        include:
          # Linux AMD64
          - os: ubuntu-latest
            goos: linux
            goarch: amd64
            zig_target: x86_64-linux-musl

          # Linux ARM64
          - os: ubuntu-latest
            goos: linux
            goarch: arm64
            zig_target: aarch64-linux-musl

          # macOS ARM64 (M1/M2)
          - os: macos-latest
            goos: darwin
            goarch: arm64

          # macOS Intel
          - os: macos-latest
            goos: darwin
            goarch: amd64

    steps:
      - name: Checkout code
        uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.25'
          cache: true

      - name: Install Zig (for Linux cross-compilation)
        if: matrix.goos == 'linux'
        uses: goto-bus/setup-zig@v2
        with:
          version: 0.13.0

      - name: Download sqlite-vec static library
        run: |
          cd store/db/sqlite
          go generate -v ./...
          echo "Downloaded library:"
          ls -lh .lib/libvec0.a
        env:
          GOOS: ${{ matrix.goos }}
          GOARCH: ${{ matrix.goarch }}

      - name: Build with AI support
        run: |
          set -x
          if [ "${{ matrix.goos }}" = "linux" ]; then
            export CC="zig cc -target ${{ matrix.zig_target }}"
            export CXX="zig c++ -target ${{ matrix.zig_target }}"
          fi

          go build -tags sqlite_vec \
            -ldflags "-s -w -X main.Version=${{ github.ref_name }}" \
            -o "divinesense-${{ matrix.goos }}-${{ matrix.goarch }}" \
            ./cmd/divinesense

          ls -lh "divinesense-${{ matrix.goos }}-${{ matrix.goarch }}"
        env:
          GOOS: ${{ matrix.goos }}
          GOARCH: ${{ matrix.goarch }}
          CGO_ENABLED: 1

      - name: Test binary (optional)
        if: matrix.goos == 'darwin' && matrix.goarch == 'arm64'
        run: |
          ./divinesense-darwin-arm64 --help

      - name: Upload artifact
        uses: actions/upload-artifact@v4
        with:
          name: divinesense-${{ matrix.goos }}-${{ matrix.goarch }}
          path: divinesense-${{ matrix.goos }}-${{ matrix.goarch }}
          retention-days: 7

  release:
    name: Create Release
    needs: build
    runs-on: ubuntu-latest
    if: startsWith(github.ref, 'refs/tags/v')

    steps:
      - name: Download all artifacts
        uses: actions/download-artifact@v4

      - name: Create GitHub Release
        uses: softprops/action-gh-release@v2
        with:
          files: divinesense-*/*
          draft: false
          prerelease: false
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

**使用方法**:
```bash
# 1. 推送 tag
git tag v0.1.0
git push origin v0.1.0

# 2. GitHub Actions 自动：
#    - 下载 4 个平台的静态库
#    - 构建 4 个平台的二进制
#    - 上传 artifacts
#    - 创建 GitHub Release

# 3. 下载 Release
# 访问: https://github.com/hrygo/divinesense/releases/tag/v0.1.0
```

---

## 故障排查

### 问题 1: go generate 下载失败

**症状**:
```bash
$ go generate -v ./...
❌ Download failed after 3 attempts
```

**原因**: 网络问题或 GitHub Releases 访问受限

**解决方案**:
```bash
# 1. 检查网络连接
curl -I https://github.com

# 2. 手动下载
cd store/db/sqlite
VERSION="v0.1.7-alpha.2"
OS="macos"  # 或 linux
ARCH="aarch64"  # 或 x86_64

curl -LO "https://github.com/asg017/sqlite-vec/releases/download/${VERSION}/sqlite-vec-${VERSION#v}-static-${OS}-${ARCH}.tar.gz"
tar -xzf sqlite-vec-*.tar.gz
mv libsqlite_vec0.a .lib/libvec0.a
rm sqlite-vec-*.tar.gz

# 3. 验证
ls -lh .lib/libvec0.a
```

### 问题 2: 构建失败 - CGO 错误

**症状**:
```bash
$ go build -tags sqlite_vec
# github.com/hrygo/divinesense/store/db/sqlite
./sqlite_vec_internal.go:10:2: error: #cgo LDFLAGS: invalid command
```

**原因**: CGO 未启用或编译器未找到

**解决方案**:
```bash
# 检查 CGO_ENABLED
go env CGO_ENABLED
# 应该是 "1" 或空

# macOS: 安装 Xcode Command Line Tools
xcode-select --install

# Linux: 安装 build-essential
sudo apt install build-essential

# 设置环境变量
export CGO_ENABLED=1
```

### 问题 3: 二进制运行时找不到 sqlite-vec

**症状**:
```bash
$ ./divinesense
INFO sqlite-vec extension not loaded, vector search will use Go fallback
```

**可能原因**:
1. 使用了默认模式（无 tag）且系统没有动态库
2. 静态库路径错误

**解决方案**:
```bash
# 检查构建模式
strings divinesense | grep vec0
# 应该看到 "vec0" 符号（如果静态链接成功）

# 如果没有，重新构建
cd store/db/sqlite
go generate -v ./...
cd -
go build -tags sqlite_vec -o divinesense ./cmd/divinesense
```

### 问题 4: 跨平台构建失败

**症状**:
```bash
$ GOOS=linux GOARCH=amd64 go build -tags sqlite_vec
cannot find -lvec0_linux_amd64
```

**原因**: 静态库平台不匹配

**解决方案**:
```bash
# 方法 1: 为目标平台下载正确的静态库
# Linux AMD64
curl -LO "https://github.com/asg017/sqlite-vec/releases/download/v0.1.7-alpha.2/sqlite-vec-0.1.7-alpha.2-static-linux-x86_64.tar.gz"
mkdir -p store/db/sqlite/.lib
tar -xzf sqlite-vec-*.tar.gz -C store/db/sqlite/.lib libsqlite_vec0.a
mv store/db/sqlite/.lib/libsqlite_vec0.a store/db/sqlite/.lib/libvec0.a

# 方法 2: 使用 GitHub Actions 自动化
# 推荐：让 CI/CD 在对应平台运行器上构建
```

---

## 最佳实践

### 开发环境

```bash
# 1. 默认模式（快速开发）
make start
# 无需下载静态库，无 CGO

# 2. AI 模式（开发 AI 功能）
cd store/db/sqlite && go generate -v ./...
cd -
make build-ai  # 或: go build -tags sqlite_vec
```

### 生产构建

```bash
# 推荐：使用 GitHub Actions
# 1. 推送 tag
git tag v0.1.0
git push origin v0.1.0

# 2. 自动构建 4 个平台
# 3. 从 GitHub Releases 下载

# 或手动构建（需要 Zig）
make build-sqlite-vec-all
```

### 持续集成

```yaml
# .github/workflows/ci.yml
- name: Download sqlite-vec
  run: cd store/db/sqlite && go generate -v ./...

- name: Build
  run: go build -tags sqlite_vec ./cmd/divinesense
  env:
    CGO_ENABLED: 1
```

---

## 版本管理

### 升级 sqlite-vec 版本

**当前版本**: v0.1.7-alpha.2

**升级步骤**:
```bash
# 1. 编辑 download_sqlite_vec.sh
vim store/db/sqlite/download_sqlite_vec.sh

# 修改第 9 行:
VERSION="v0.1.8"  # 新版本

# 2. 测试下载
cd store/db/sqlite
rm -rf .lib
./download_sqlite_vec.sh

# 3. 测试构建
cd -
go build -tags sqlite_vec ./cmd/divinesense

# 4. 提交
git add store/db/sqlite/download_sqlite_vec.sh
git commit -m "chore(sqlite-vec): upgrade to v0.1.8"
```

---

## 性能对比

### 向量搜索性能

| 方法 | 复杂度 | 10K 向量 | 100K 向量 |
|:-----|:-------|:--------:|:---------:|
| **vec0 虚拟表** | O(log n) | ~5ms | ~10ms |
| **Go 应用层** | O(n) | ~50ms | ~500ms |

### 二进制大小

| 构建模式 | 大小 | AI 支持 |
|:---------|:----:|:--------|
| **默认（无 tag）** | 52MB | ❌ |
| **AI 模式（sqlite_vec）** | 55MB | ✅ |

---

## 总结

### 什么时候用哪种模式？

| 场景 | 推荐模式 | 构建命令 |
|:-----|:---------|:---------|
| **快速开发** | 默认模式 | `go build` |
| **开发 AI 功能** | AI 模式 | `go build -tags sqlite_vec` |
| **本地测试** | 默认模式 | `make start` |
| **生产部署** | AI 模式 | GitHub Actions (自动) |
| **单二进制分发** | AI 模式 | `go build -tags sqlite_vec` |

### 核心命令

```bash
# 下载静态库（首次）
cd store/db/sqlite && go generate -v ./...

# 构建 AI 版本
go build -tags sqlite_vec ./cmd/divinesense

# 运行
./divinesense --driver sqlite
```

---

**更新日期**: 2026-02-06
**sqlite-vec 版本**: v0.1.7-alpha.2
**相关 Issue**: #9
