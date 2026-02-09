# sqlite-vec 使用指南

sqlite-vec 是一个可选功能，可以通过 Make 参数或不同的 Dockerfile 启用。

## 本地开发

### 方法一：使用 Make 命令（推荐）

**启动 PostgreSQL 版本（默认）**：
```bash
make start
```

**启动 SQLite + sqlite-vec 版本**：
```bash
make start-sqlite-vec
```

**区别**：
| 命令 | 数据库 | AI 功能 | 适用场景 |
|:-----|:-------|:-------|:---------|
| `make start` | PostgreSQL | ✅ | 开发/生产（推荐） |
| `make start-sqlite-vec` | SQLite + sqlite-vec | ✅ | 轻量开发/离线 |

### 方法二：直接使用 Make 参数

```bash
# SQLite + sqlite-vec 模式
SQLITE_VEC=true make run

# PostgreSQL 模式（默认）
make run
```

### 验证 sqlite-vec 是否启用

查看启动日志：
```
✓ 成功：
[INFO] sqlite-vec extension registered as auto-extension
[INFO] sqlite-vec extension verified (static linking) functions_found=18

✗ 降级：
[WARN] sqlite-vec extension not loaded, vector search will use Go fallback
```

---

## Docker 部署

### PostgreSQL 版本（默认）

```bash
# 使用默认 Dockerfile
docker build -t divinesense:postgres -f docker/Dockerfile .

# 运行
docker run -d \
  --name divinesense \
  -p 5230:5230 \
  -e DIVINESENSE_DRIVER=postgres \
  -e DIVINESENSE_DSN="postgres://user:pass@host:5432/db" \
  divinesense:postgres
```

### SQLite + sqlite-vec 版本

```bash
# 使用专门的 Dockerfile
docker build -t divinesense:sqlite-vec -f docker/Dockerfile.sqlite-vec .

# 运行
docker run -d \
  --name divinesense \
  -p 5230:5230 \
  -v $(pwd)/data:/var/opt/divinesense \
  divinesense:sqlite-vec
```

### Docker Compose（推荐）

**PostgreSQL 版本**：
```bash
docker compose -f docker/compose/prod.yml up -d
```

**SQLite 版本**：
```bash
docker compose -f docker/compose/sqlite-vec.yml up -d
```

**Make 命令快捷方式**：
```bash
# PostgreSQL
make docker-prod-up      # 启动
make docker-prod-down    # 停止
make docker-prod-logs    # 查看日志

# SQLite + sqlite-vec
make docker-sqlite-vec-up      # 启动
make docker-sqlite-vec-down    # 停止
make docker-sqlite-vec-logs    # 查看日志
```

---

## 二进制部署

### 本地构建

**macOS (Apple Silicon)**：
```bash
CGO_ENABLED=1 \
GOOS=darwin \
GOARCH=arm64 \
go build -tags sqlite_vec -o divinesense-darwin-arm64 ./cmd/divinesense
```

**Linux (AMD64)**：
```bash
CGO_ENABLED=1 \
GOOS=linux \
GOARCH=amd64 \
go build -tags sqlite_vec -o divinesense-linux-amd64 ./cmd/divinesense
```

### 交叉编译（从 macOS 到 Linux）

需要先安装 Zig：
```bash
# macOS
brew install zig

# 下载 sqlite-vec 静态库（目标平台）
cd store/db/sqlite
GOOS=linux GOARCH=amd64 bash download_sqlite_vec.sh
cd ../../../

# 下载 SQLite amalgamation
curl -sL https://www.sqlite.org/2024/sqlite-amalgamation-3450000.zip -o /tmp/sqlite.zip
unzip -q /tmp/sqlite.zip -d /tmp/
cp /tmp/sqlite-amalgamation-3450000/sqlite3.h store/db/sqlite/.lib/

# 交叉编译
CGO_ENABLED=1 \
GOOS=linux \
GOARCH=amd64 \
CC="zig cc -target x86_64-linux-musl -Istore/db/sqlite/.lib" \
CGO_CFLAGS="-Istore/db/sqlite/.lib" \
CGO_LDFLAGS="-Lstore/db/sqlite/.lib" \
go build -tags sqlite_vec -o divinesense-linux-amd64 ./cmd/divinesense
```

### 部署到服务器

```bash
# 上传二进制
scp divinesense-linux-amd64 user@server:/opt/divinesense/bin/divinesense

# 启动
ssh user@server
cd /opt/divinesense
./bin/divinesense --driver sqlite --dsn /var/lib/divinesense/divinesense.db
```

---

## 环境变量配置

创建 `.env` 文件：

```bash
# 数据库（SQLite）
DIVINESENSE_DRIVER=sqlite
DIVINESENSE_DSN=divinesense.db?_loc=auto&_allow_load_extension=1

# AI 功能（必需）
DIVINESENSE_AI_ENABLED=true
DIVINESENSE_AI_EMBEDDING_PROVIDER=siliconflow
DIVINESENSE_AI_EMBEDDING_MODEL=BAAI/bge-m3
DIVINESENSE_AI_SILICONFLOW_API_KEY=your_key_here

# LLM（可选）
DIVINESENSE_AI_LLM_PROVIDER=deepseek
DIVINESENSE_AI_LLM_MODEL=deepseek-chat
DIVINESENSE_AI_DEEPSEEK_API_KEY=your_key_here
```

---

## 故障排查

### CGO 相关错误

**错误**：`cgo: C compiler not found`

**解决**：
```bash
# macOS
xcode-select --install

# Linux
sudo apt-get install build-essential  # Debian/Ubuntu
sudo yum groupinstall "Development Tools"  # CentOS/RHEL
```

### sqlite-vec 扩展未加载

**错误**：`WARN sqlite-vec extension not loaded`

**解决**：
```bash
# 1. 检查静态库
ls -lh store/db/sqlite/.lib/libvec0.a

# 2. 重新下载
cd store/db/sqlite
rm -rf .lib
bash download_sqlite_vec.sh

# 3. 重新编译（带 tags）
go build -tags sqlite_vec ...
```

### 交叉编译缺少头文件

**错误**：`fatal error: 'sqlite3.h' file not found`

**解决**：
```bash
# 下载 SQLite amalgamation
curl -sL https://www.sqlite.org/2024/sqlite-amalgamation-3450000.zip -o /tmp/sqlite.zip
unzip -q /tmp/sqlite.zip
cp /tmp/sqlite-amalgamation-3450000/sqlite3.h store/db/sqlite/.lib/

# 添加 CFLAGS
CGO_CFLAGS="-Istore/db/sqlite/.lib" \
go build -tags sqlite_vec ...
```

---

## 性能对比

| 数据集大小 | PostgreSQL + pgvector | SQLite + sqlite-vec | 建议 |
|:-----------|:---------------------|:-------------------|:-----|
| < 1k 向量   | ~10ms                | ~15ms              | 相当 |
| 1k-10k      | ~20ms                | ~30ms              | SQLite 可用 |
| > 10k       | ~50ms                | ~100ms+            | 推荐 PostgreSQL |

**选择建议**：
- ✅ **PostgreSQL + pgvector**：生产环境、大规模部署
- ✅ **SQLite + sqlite-vec**：开发环境、个人使用、轻量部署

---

## 快速验证

```bash
# 克隆项目
git clone https://github.com/hrygo/divinesense.git
cd divinesense

# 方式一：PostgreSQL（默认）
make start

# 方式二：SQLite + sqlite-vec
make start-sqlite-vec

# 访问
open http://localhost:28081
```
