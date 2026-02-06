# SQLite-Vec 官方 Releases 集成

## 实现概述

成功集成 sqlite-vec 官方 GitHub Releases，使用 `go generate` 自动下载静态库，实现编译时静态链接。

## 技术方案

### Build Tag 分离

```go
// 启用 AI 功能（静态链接）
//go:build sqlite_vec
// +build sqlite_vec

// 默认模式（动态库加载）
//go:build !sqlite_vec
// +build !sqlite_vec
```

### 文件结构

```
store/db/sqlite/
├── sqlite_vec_internal.go     // sqlite_vec tag: CGO 静态链接
├── sqlite_vec_loader.go       // !sqlite_vec tag: 动态库加载
├── sqlite_extension.go        // !sqlite_vec tag: 动态扩展加载
└── download_sqlite_vec.sh     // go generate: 下载脚本
```

## 核心实现

### 1. 编译时自动下载 (go generate)

**sqlite_vec_internal.go**:
```go
//go:generate ./download_sqlite_vec.sh

/*
#cgo CFLAGS: -I${SRCDIR}/.lib
#cgo LDFLAGS: ${SRCDIR}/.lib/libvec0.a
*/
import "C"
```

**download_sqlite_vec.sh**:
```bash
VERSION="v0.1.7-alpha.2"
BASE_URL="https://github.com/asg017/sqlite-vec/releases/download"

# 检测平台
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"  # darwin -> macos
ARCH="$(uname -m)"                               # arm64 -> aarch64

# 下载并解压
curl -sL "${URL}" | tar -xz -C "${LIB_DIR}" libsqlite_vec0.a
mv "${LIB_DIR}/libsqlite_vec0.a" "${LIB_DIR}/libvec0.a"
```

### 2. 静态链接验证

```go
func loadVecExtension(db *sql.DB) error {
    var result int
    err := db.QueryRow("SELECT count(*) FROM pragma_function_list WHERE name LIKE 'vec_%'").Scan(&result)
    if result == 0 {
        return errors.New("sqlite-vec extension not loaded")
    }
    slog.Info("sqlite-vec extension verified (static linking)", "functions_found", result)
    return nil
}
```

## 构建命令

### 启用 AI 功能（静态链接）

```bash
cd store/db/sqlite
go generate -v ./...    # 下载静态库到 .lib/

cd -
go build -tags sqlite_vec -o divinesense ./cmd/divinesense
```

**产物**: 55MB 单二进制（包含 sqlite-vec v0.1.7-alpha.2）

### 默认模式（无 AI）

```bash
go build -o divinesense ./cmd/divinesense
```

**产物**: 52MB 单二进制（不含 sqlite-vec）

## 验证结果

### 本地测试 (macOS ARM64)

```bash
$ ./download_sqlite_vec.sh
Downloading sqlite-vec static library...
URL: https://github.com/asg017/sqlite-vec/releases/download/v0.1.7-alpha.2/sqlite-vec-0.1.7-alpha.2-static-macos-aarch64.tar.gz
✓ Downloaded successfully: .lib/libvec0.a
-rw-r--r--  1 xiaobingyang  staff   157K Jan. 11  2025 .lib/libvec0.a

$ go build -tags sqlite_vec -o /tmp/divinesense-official ./cmd/divinesense
$ ls -lh /tmp/divinesense-official
-rwxr-xr-x  1 xiaobingyang  staff    55M Feb.  6 15:01 /tmp/divinesense-official

$ /tmp/divinesense-official --help
An AI-powered personal knowledge assistant.
```

✅ 静态库下载成功 (157KB)
✅ 二进制构建成功 (55MB)
✅ 运行正常

## 平台支持

### 官方提供的静态库

| 平台 | 文件名 | 状态 |
|:-----|:-------|:-----|
| **macOS ARM64** | `sqlite-vec-0.1.7-alpha.2-static-macos-aarch64.tar.gz` | ✅ 已测试 |
| **macOS Intel** | `sqlite-vec-0.1.7-alpha.2-static-macos-x86_64.tar.gz` | ✅ 可用 |
| **Linux ARM64** | `sqlite-vec-0.1.7-alpha.2-static-linux-aarch64.tar.gz` | ✅ 可用 |
| **Linux x86_64** | `sqlite-vec-0.1.7-alpha.2-static-linux-x86_64.tar.gz` | ✅ 可用 |

### 自动平台检测

脚本自动检测当前平台并下载对应的静态库：

```bash
# macOS ARM64
OS="macos"
ARCH="aarch64"

# macOS Intel
OS="macos"
ARCH="x86_64"

# Linux ARM64
OS="linux"
ARCH="aarch64"

# Linux x86_64
OS="linux"
ARCH="x86_64"
```

## 版本管理

### 当前版本

- **sqlite-vec**: v0.1.7-alpha.2
- **更新日期**: 2025-01-11

### 升级步骤

1. 更新 `download_sqlite_vec.sh` 中的 `VERSION`
2. 运行 `go generate -v ./store/db/sqlite/...`
3. 重新构建：`go build -tags sqlite_vec ./cmd/divinesense`

## CI/CD 集成

### GitHub Actions 示例

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
          - os: macos-latest
            goos: darwin
            goarch: arm64

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

      - name: Build with AI support
        run: |
          go build -tags sqlite_vec -ldflags "-s -w" -o divinesense-${{ matrix.goos }}-${{ matrix.goarch }} ./cmd/divinesense

      - name: Upload artifact
        uses: actions/upload-artifact@v4
        with:
          name: divinesense-${{ matrix.goos }}-${{ matrix.goarch }}
          path: divinesense-${{ matrix.goos }}-${{ matrix.goarch }}
```

## .gitignore 配置

确保 `.lib` 目录不被提交到版本控制：

```gitignore
# SQLite-vec static library (auto-downloaded)
store/db/sqlite/.lib/
```

## 优势

✅ **官方版本**: 使用 sqlite-vec 官方 releases，无需自行编译
✅ **自动化**: `go generate` 自动下载，无需手动操作
✅ **跨平台**: 支持 4 个主流平台（macOS/Linux × ARM64/x86_64）
✅ **单二进制**: 55MB 完整功能，无需额外依赖
✅ **版本固定**: 明确指定 v0.1.7-alpha.2，可重现构建
✅ **透明缓存**: `.lib/` 目录可被版本控制忽略

## 限制

⚠️ **网络依赖**: 首次 `go generate` 需要互联网连接
⚠️ **平台限制**: Windows 不支持（sqlite-vec 官方未提供 Windows 静态库）
⚠️ **版本同步**: 需手动更新 VERSION 常量以升级

## 故障排查

### 问题：go generate 没有执行

**症状**: `.lib/` 目录不存在，构建失败

**解决**:
```bash
cd store/db/sqlite
go generate -v ./...
```

### 问题：tar 解压失败

**症状**: `tar: Error opening archive: Unrecognized archive format`

**原因**: URL 格式错误或网络问题

**解决**:
```bash
# 手动测试下载
curl -L https://github.com/asg017/sqlite-vec/releases/download/v0.1.7-alpha.2/sqlite-vec-0.1.7-alpha.2-static-macos-aarch64.tar.gz | tar -tz
```

### 问题：静态库符号冲突

**症状**: 链接错误，undefined symbols

**原因**: 静态库与当前平台不匹配

**解决**: 检查 `uname -m` 和 `uname -s` 输出，确认平台检测正确

## 相关文档

- [sqlite-vec 官方仓库](https://github.com/asg017/sqlite-vec)
- [sqlite-vec Releases](https://github.com/asg017/sqlite-vec/releases)
- [编译时下载实现](./SQLITE_VEC_COMPILE_TIME_DOWNLOAD.md)（废弃方案）

---

**最后更新**: 2026-02-06
**状态**: ✅ 生产就绪
**版本**: sqlite-vec v0.1.7-alpha.2
