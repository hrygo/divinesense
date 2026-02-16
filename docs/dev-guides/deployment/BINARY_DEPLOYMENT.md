# DivineSense 部署指南

> **版本**: v0.99.0 | **更新时间**: 2026-02-12

---

## 📖 目录

- [快速开始](#快速开始)
- [部署模式对比](#部署模式对比)
- [Docker 模式部署](#docker-模式部署)
- [二进制模式部署（推荐）](#二进制模式部署推荐)
- [Geek Mode 配置](#geek-mode-配置)
- [云服务器部署](#云服务器部署)
- [升级指南](#升级指南)
- [备份与恢复](#备份与恢复)
- [配置说明](#配置说明)
- [故障排查](#故障排查)
- [卸载指南](#卸载指南)

---

## ⚡ 快速开始

### 一键部署（推荐）

```bash
# 交互式安装（推荐新手）
curl -fsSL https://raw.githubusercontent.com/hrygo/divinesense/main/deploy/install.sh | sudo bash -s -- --interactive

# 二进制模式（Geek Mode 推荐）
curl -fsSL https://raw.githubusercontent.com/hrygo/divinesense/main/deploy/install.sh | sudo bash -s -- --mode=binary

# Docker 模式
curl -fsSL https://raw.githubusercontent.com/hrygo/divinesense/main/deploy/install.sh | sudo bash -s -- --mode=docker
```

### 查看帮助

```bash
curl -fsSL https://raw.githubusercontent.com/hrygo/divinesense/main/deploy/install.sh | sudo bash -s -- --help
```

---

## 部署模式对比

| 特性           | Docker 模式    | 二进制模式          |
| :------------- | :------------- | :------------------ |
| Geek Mode 支持 | ⚠️ 需额外配置   | ✅ 原生支持          |
| Evolution Mode 支持 | ❌ 不支持 | ✅ 原生支持 |
| 资源占用       | 高 (容器开销)  | 低                  |
| 启动速度       | 慢             | 快                  |
| 更新方式       | 重建镜像       | 替换二进制          |
| 数据隔离       | 容器隔离       | 需手动配置          |
| 适用场景       | 快速部署、测试 | Geek Mode、生产环境 |

---

## Docker 模式

### 安装流程

1. 自动安装 Docker 和 Docker Compose
2. 克隆项目文件
3. 生成配置文件
4. 拉取镜像
5. 启动容器

### 服务管理

```bash
cd /opt/divinesense
./deploy.sh status     # 查看状态
./deploy.sh logs       # 查看日志
./deploy.sh restart    # 重启服务
./deploy.sh backup     # 备份数据
./deploy.sh restore    # 恢复数据
```

---

## 二进制模式 (推荐 Geek Mode)

### 安装流程

1. 检测系统架构
2. 下载二进制文件并校验完整性
3. 创建用户和目录
4. 安装 systemd 服务
5. 配置 PostgreSQL (Docker 或系统)
6. **配置用户运维权限**（自动完成）：
   - divine 用户加入 docker 组
   - 配置 sudoers 免密（仅限服务管理命令）
   - 创建 `~/Makefile` 运维工具
   - 配置 bash 快捷别名
7. 启动服务

### 服务管理

**方式一：用户 Makefile（推荐，无需 sudo 密码）**

安装完成后，divine 用户主目录会自动创建运维 Makefile：

```bash
# SSH 登录后使用（以 divine 用户）
make help          # 查看所有命令
make status        # 查看服务状态
make health        # 健康检查
make restart       # 重启服务
make logs          # 查看日志
make db-backup     # 备份数据库
make db-shell      # 进入数据库 Shell
make upgrade       # 升级到最新版本
make clone-source  # 克隆源码（Evolution Mode）
```

**快捷别名**（重新登录生效）：

```bash
ds-status    # 等同于 make status
ds-restart   # 等同于 make restart
ds-health    # 等同于 make health
ds-backup    # 等同于 make db-backup
ds-db        # 等同于 make db-shell
```

**方式二：systemd 命令**

```bash
sudo systemctl status divinesense    # 查看状态
sudo systemctl restart divinesense   # 重启服务
sudo journalctl -u divinesense -f    # 查看日志
```

---

## Geek Mode 配置

Geek Mode 允许 DivineSense 通过 Claude Code CLI 处理代码相关任务。

### 安装 Claude Code CLI

**方法 1: 官方 NPM 包（推荐）**

```bash
npm install -g @anthropic-ai/claude-code
claude auth login
```

**方法 2: 智谱 Coding Helper（国内网络优化）**

```bash
npx @z_ai/coding-helper
```

参考: [智谱 AI Claude Code 文档](https://docs.bigmodel.cn/cn/coding-plan/tool/claude)

### 启用 Geek Mode

编辑 `/etc/divinesense/config`：

```bash
# 启用 Geek Mode 功能模块
# 开启后，前端聊天界面会出现 Geek Mode 切换开关
DIVINESENSE_CLAUDE_CODE_ENABLED=true
DIVINESENSE_CLAUDE_CODE_WORKDIR=/opt/divinesense/data
```

重启服务：

```bash
sudo systemctl restart divinesense
```

### 启用 Evolution Mode (进化模式)

Evolution Mode 是 Geek Mode 的高级形态，解锁完整的 Claude Code Agent 能力。

编辑 `/etc/divinesense/config`：

```bash
# 启用 Evolution Mode (需先启用 Geek Mode)
DIVINESENSE_CLAUDE_CODE_ENABLED=true
DIVINESENSE_EVOLUTION_ENABLED=true

# 可选: 仅管理员可用
DIVINESENSE_EVOLUTION_ADMIN_ONLY=true
```

重启服务：

```bash
sudo systemctl restart divinesense
```

### 验证

1. 进入 DivineSense 聊天界面
2. 点击输入框上方的模式切换开关，进入 Geek Mode
3. 发送代码相关指令（如"帮我修复这段代码的 bug"），此时系统将调用 Claude Code CLI 处理请求

---

## 云服务器部署注意事项

### 阿里云 ECS / 腾讯云 CVM

部署到云服务器时，需要注意以下事项：

#### 1. 安全组配置

安装完成后，需要在云控制台开放对应端口：

| 服务 | 开发环境端口 | 生产环境端口 | 说明 |
|:-----|:------------|:-------------|:-----|
| DivineSense (后端) | 28081 | 5230 | HTTP/gRPC API 端口 |
| DivineSense (前端) | 25173 | - | Vite 开发服务器（生产环境嵌入二进制） |
| PostgreSQL | 25432 | 25432 | 数据库端口（建议仅内网） |

**注意**：
- 开发环境：后端使用 28081，前端开发服务器使用 25173
- 生产环境：Web 服务默认使用 5230（可通过 `DIVINESENSE_PORT` 修改）

**阿里云操作路径**：
1. ECS 实例 → 安全组 → 配置规则
2. 添加入方向规则：端口 `5230/5230`，授权对象 `0.0.0.0/0`

#### 2. 使用标准 80 端口

如果需要使用 80 端口（HTTP 标准端口），需要特殊配置：

```bash
# 修改配置文件
sudo nano /etc/divinesense/config
# 修改: DIVINESENSE_PORT=80

# 修改 systemd 服务文件，添加低端口绑定权限
sudo nano /etc/systemd/system/divinesense.service
# 在 [Service] 段添加: AmbientCapabilities=CAP_NET_BIND_SERVICE
# 修改 ExecStart 为: ExecStart=/opt/divinesense/bin/divinesense --port 80 --data /opt/divinesense/data

# 重载并重启
sudo systemctl daemon-reload
sudo systemctl restart divinesense
```

#### 3. 域名配置（可选）

配置域名后，可以使用 Nginx 反向代理：

```nginx
server {
    listen 80;
    server_name your-domain.com;

    location / {
        proxy_pass http://localhost:5230;  # 生产环境默认端口，可通过 DIVINESENSE_PORT 修改
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

---

## 升级

### Docker 模式

```bash
cd /opt/divinesense
./deploy.sh upgrade
```

### 二进制模式

```bash
/opt/divinesense/deploy-binary.sh upgrade
```

---

## 备份与恢复

### Docker 模式

```bash
cd /opt/divinesense
./deploy.sh backup              # 备份
./deploy.sh restore <文件名>     # 恢复
```

### 二进制模式

```bash
/opt/divinesense/deploy-binary.sh backup              # 备份
/opt/divinesense/deploy-binary.sh restore <文件名>     # 恢复
```

### 手动备份 PostgreSQL

```bash
# Docker PostgreSQL
docker exec divinesense-postgres pg_dump -U divinesense divinesense | gzip > backup.sql.gz

# 系统 PostgreSQL
pg_dump -U divinesense divinesense | gzip > backup.sql.gz
```

---

## 配置文件

### Docker 模式

- 配置文件: `/opt/divinesense/.env.prod`
- 数据库密码: `/opt/divinesense/.db_password`

### 二进制模式

- 配置文件: `/etc/divinesense/config`
- 数据库密码: `/etc/divinesense/.db_password`

---

## 故障排查

### 服务无法启动

```bash
# 查看详细错误
sudo journalctl -u divinesense -n 50 --no-pager

# 检查配置文件
sudo cat /etc/divinesense/config

# 检查端口占用
sudo ss -tlnp | grep 5230  # 生产环境
# 或
sudo ss -tlnp | grep 28081  # 开发环境
```

### 数据库连接失败

```bash
# 检查 PostgreSQL 容器
docker ps | grep divinesense-postgres

# 测试连接
docker exec divinesense-postgres pg_isready -U divinesense

# 查看 PostgreSQL 日志
docker logs divinesense-postgres
```

### AI 功能不可用

确保：
1. 使用 PostgreSQL（SQLite 不支持 AI）
2. API Key 已配置且有效
3. `DIVINESE_AI_ENABLED=true`
4. pgvector 扩展已安装

> 💡 **SQLite AI 支持研究**：详见 [#9](https://github.com/hrygo/divinesense/issues/9) - 探索开发环境 AI 功能可能性

```bash
# 验证 pgvector
docker exec divinesense-postgres psql -U divinesense -d divinesense -c "SELECT extname FROM pg_extension WHERE extname = 'vector';"
```

### Geek Mode 不可用

确保：
1. Claude Code CLI 已安装
2. `claude` 命令在 PATH 中
3. `DIVINESENSE_CLAUDE_CODE_ENABLED=true`
4. 工作目录可写

```bash
# 验证 Claude Code CLI
which claude
claude --version

# 验证权限
ls -la /opt/divinesense/data
```

---

## 卸载

### Docker 模式

```bash
cd /opt/divinesense
./deploy.sh stop
# 然后手动删除容器和镜像
```

### 二进制模式

```bash
sudo /opt/divinesense/deploy-binary.sh uninstall
```

---

## 目录结构

### Docker 模式

```
/opt/divinesense/          # 项目根目录
├── .env.prod              # 环境配置
├── .db_password          # 数据库密码
├── docker/               # Docker 配置
│   └── compose/
│       └── prod.yml
├── backups/              # 备份目录
└── deploy.sh            # 管理脚本
```

### 二进制模式

```
/opt/divinesense/          # 安装根目录
├── bin/                   # 二进制文件
│   └── divinesense
├── data/                  # 工作目录 (Geek Mode)
├── logs/                  # 日志目录
├── backups/               # 数据库备份
├── docker/                # PostgreSQL Docker 配置 (可选)
│   ├── postgres.yml
│   └── .env
└── deploy-binary.sh      # 管理脚本

/etc/divinesense/          # 配置目录
└── config                 # 环境变量配置
└── .db_password          # 数据库密码

/etc/systemd/system/       # systemd 服务
└── divinesense.service
```
