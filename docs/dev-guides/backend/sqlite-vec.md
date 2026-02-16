# SQLite-Vec 向量搜索使用指南

> DivineSense SQLite 向量搜索（语义检索）功能说明

> **⚠️ 注意**: 当前仅支持向量搜索功能。完整的 AI 功能（对话持久化、情景记忆等）需要使用 PostgreSQL，或等待后续 PR 实现。详见 [#134](https://github.com/hrygo/divinesense/issues/134)。

---

## 📋 目录

1. [功能范围](#功能范围)
2. [快速开始](#快速开始)
3. [两种模式对比](#两种模式对比)
4. [本地开发](#本地开发)
5. [生产构建](#生产构建)
6. [CI/CD 自动化](#cicd-自动化)
7. [故障排查](#故障排查)

---

## 功能范围

### ✅ 当前支持（PR #131）

| 功能        | 状态 | 说明                                       |
| :---------- | :--- | :----------------------------------------- |
| 向量搜索    | ✅    | 使用 sqlite-vec 扩展实现 O(log n) KNN 搜索 |
| 向量存储    | ✅    | BLOB (vec0) + TEXT (JSON) 双格式           |
| Go Fallback | ✅    | 扩展不可用时应用层计算                     |
| 全文搜索    | ✅    | FTS5 或 LIKE fallback                      |

### 🚧 待实现（后续 PR）

| 功能       | 规划 PR | 说明                          |
| :--------- | :------ | :---------------------------- |
| 对话持久化 | #132    | `AIBlock` SQLite 支持         |
| 情景记忆   | #133    | `EpisodicMemory` SQLite 支持  |
| 用户偏好   | #134    | `UserPreferences` SQLite 支持 |
| 代理指标   | #134    | `AgentMetrics` SQLite 支持    |

**💡 推荐**: 如需完整 AI 功能，请使用 PostgreSQL。

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

**产物**: 55MB 二进制，包含向量搜索功能

---

## 两种模式对比

### 模式对比

| 特性       | 默认模式      | AI 模式        |
| :--------- | :------------ | :------------- |
| 向量搜索   | ❌             | ✅ (sqlite-vec) |
| 全文搜索   | ✅ (FTS5/LIKE) | ✅ (FTS5/LIKE)  |
| 构建       | 纯 Go         | 需要 CGO       |
| 二进制大小 | ~52MB         | ~55MB          |
| 交叉编译   | 简单          | 需要工具链     |

### 依赖说明

> ⚠️ **重要变更**: AI 模式使用 `mattn/go-sqlite3` 替代 `modernc.org/sqlite`

| 驱动                 | CGO  | 扩展支持     |
| :------------------- | :--- | :----------- |
| `modernc.org/sqlite` | ❌    | ❌ (默认模式) |
| `mattn/go-sqlite3`   | ✅    | ✅ (AI 模式)  |

---

## 本地开发

### 启动 SQLite + AI 模式

```bash
# 一键启动（自动下载静态库）
make start-sqlite-vec

# 或手动启动
SQLITE_VEC=true make start
```

### 环境变量

```bash
# 启用 sqlite-vec
export SQLITE_VEC=true

# 或同时启用 AI 服务
export DIVINESENSE_AI_ENABLED=true
export SQLITE_VEC=true
```

---

## 生产构建

### Linux 构建

```bash
# 下载 Linux 静态库
cd store/db/sqlite
GOOS=linux GOARCH=amd64 go generate -v ./...

# 构建
cd -
GOOS=linux GOARCH=amd64 CGO_ENABLED=1 \
  go build -tags sqlite_vec -o divinesense-linux ./cmd/divinesense
```

### macOS 构建

```bash
# 下载 macOS 静态库
cd store/db/sqlite
GOOS=darwin GOARCH=arm64 go generate -v ./...

# 构建
cd -
GOOS=darwin GOARCH=arm64 CGO_ENABLED=1 \
  go build -tags sqlite_vec -o divinesense-macos ./cmd/divinesense
```

---

## CI/CD 自动化

### GitHub Actions

见 `.github/workflows/build-multi-platform.yml`，自动构建：
- Linux (amd64, arm64)
- macOS (amd64, arm64)
- Windows (amd64)

---

## 故障排查

### 静态库下载失败

```bash
# 手动下载
cd store/db/sqlite
bash download_sqlite_vec.sh
```

### CGO 错误

```bash
# 确保 CGO 已启用
export CGO_ENABLED=1

# 安装 GCC (Linux)
sudo apt-get install build-essential

# 安装 Xcode Command Line Tools (macOS)
xcode-select --install
```

### 扩展加载失败

检查日志中的 "vec0 not found" 错误，确保：
1. 静态库已下载
2. 使用 `-tags sqlite_vec` 构建
3. `DIVINESENSE_AI_ENABLED=true`

---

## 相关文档

- **技术调研**: `docs/research/SQLITE_VEC_OFFICIAL_RELEASES.md`
- **完整规划**: [#134](https://github.com/hrygo/divinesense/issues/134)
- **后端指南**: `database.md`
