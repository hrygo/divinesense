# DivineSense 多平台构建 Dockerfile
# 解决 CGO 交叉编译问题

# ============================================================================
# 阶段 1: 构建阶段 - 编译 sqlite-vec 静态库
# ============================================================================
FROM ubuntu:22.04 AS builder

# 安装依赖
RUN apt-get update && apt-get install -y \
    build-essential \
    git \
    python3 \
    curl \
    && rm -rf /var/lib/apt/lists/*

# 设置工作目录
WORKDIR /build

# 下载并编译 SQLite amalgamation
RUN curl -sL https://sqlite.org/2024/sqlite-amalgamation-3470200.zip -o sqlite.zip && \
    unzip sqlite.zip && \
    gcc -c -fPIC -DSQLITE_ENABLE_FTS5 -DSQLITE_ENABLE_RTREE -O2 \
        sqlite-amalgamation-3470200/sqlite3.c -o sqlite3.o

# 下载并编译 sqlite-vec
RUN git clone --depth 1 https://github.com/asg017/sqlite-vec.git && \
    cd sqlite-vec && \
    python3 -c "import subprocess; subprocess.run(['envsubst', '<', 'sqlite-vec.h.tmpl', '>', 'sqlite-vec.h'], shell=True)" && \
    gcc -c -fPIC -O2 sqlite-vec.c -o sqlite-vec.o && \
    ar rcs libvec0.a sqlite3.o sqlite-vec.o && \
    mkdir -p /usr/local/lib && \
    cp libvec0.a /usr/local/lib/

# 验证
RUN ls -lh /usr/local/lib/libvec0.a

# ============================================================================
# 阶段 2: Go 构建阶段 - 使用静态库编译 DivineSense
# ============================================================================
FROM golang:1.25 AS builder-go

WORKDIR /app

# 安装 CGO 依赖
RUN apt-get update && apt-get install -y \
    build-essential \
    gcc \
    git \
    && rm -rf /var/lib/apt/lists/*

# 复制静态库
COPY --from=builder /usr/local/lib/libvec0.a /usr/local/lib/

# 复制项目代码
COPY . .

# 设置编译环境
ENV CGO_ENABLED=1
ENV GO111MODULE=on

# 编译静态链接的二进制
RUN go build -tags=prod -ldflags '-linkmode external -extldflags "-static"' \
    -o /tmp/divinesense \
    ./cmd/divinesense

# 验证二进制
RUN ldd /tmp/divinesense || echo "Static linking successful"
RUN ls -lh /tmp/divinesense

# ============================================================================
# 阶段 3: 运行时阶段 - 最小化镜像
# ============================================================================
FROM alpine:latest

# 安装运行时依赖
RUN apk add --no-cache ca-certificates

WORKDIR /app

# 复制二进制
COPY --from=builder-go /tmp/divinesense /app/divinesense

# 验证
RUN /app/divinesense --help 2>&1 | head -5 || echo "Binary ready"

# 暴露端口
EXPOSE 5230

# 健康检查
HEALTHCHECK --interval=30s --timeout=3s \
  CMD ["/app/divinesense", "--mode=check"]

# 启动服务
CMD ["/app/divinesense"]
