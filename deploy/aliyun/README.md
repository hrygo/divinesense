# DivineSense 单机部署指南

适用于阿里云/腾讯云服务器的生产环境部署方案。

---

## 一键安装

```bash
curl -fsSL https://raw.githubusercontent.com/hrygo/divinesense/main/deploy/install.sh | sudo bash -s -- --mode=docker
```

**自动完成：**
- ✅ 安装 Docker + Docker Compose
- ✅ 配置国内镜像加速
- ✅ 下载 DivineSense 镜像
- ✅ 生成安全密码
- ✅ 初始化 PostgreSQL + pgvector
- ✅ 启动服务
- ✅ 配置防火墙
- ✅ 设置每日自动备份

**安装完成后：**

1. 配置 AI API Keys：
```bash
vi /opt/divinesense/.env.prod

# 修改以下两项：
DIVINESENSE_AI_SILICONFLOW_API_KEY=sk-xxx
DIVINESENSE_AI_DEEPSEEK_API_KEY=sk-xxx

# 重启服务
cd /opt/divinesense && ./deploy.sh restart
```

2. 访问服务：`http://your-server-ip:5230`

---

## 架构

```
┌─────────────────────────────────────────────────┐
│              服务器                               │
│                                                 │
│  ┌──────────────────────────────────────────┐  │
│  │           Docker Network                 │  │
│  │                                          │  │
│  │  ┌──────────────┐  ┌─────────────────┐  │  │
│  │  │  PostgreSQL  │  │   DivineSense   │  │  │
│  │  │  pg16+vector │  │   自定义资源    │  │  │
│  │  │              │──│  :5230 ────────►│───┼──► 公网
│  │  │  :5432       │  │                 │  │  │
│  │  └──────────────┘  └─────────────────┘  │  │
│  └──────────────────────────────────────────┘  │
│                                                 │
│  数据卷: postgres_data, divinesense_data        │
└─────────────────────────────────────────────────┘
```

**资源分配建议**

| 服务        | CPU    | 内存    | 说明        |
| ----------- | ------ | ------- | ----------- |
| PostgreSQL  | 可配置 | 可配置   | 数据库      |
| DivineSense | 可配置 | 可配置   | 应用服务    |
| 系统预留    | >=0.5核 | >=512M  | OS + Docker |

> 💡 **提示**：根据服务器配置调整资源分配，建议预留至少 512MB 给系统。

---

## AI 配置

DivineSense 需要 2 个 API Key（国内推荐）：

| API Key     | 用途               | 获取地址                      |
| ----------- | ------------------ | ----------------------------- |
| SiliconFlow | 向量/重排/意图分类 | https://cloud.siliconflow.cn  |
| DeepSeek    | 对话 LLM           | https://platform.deepseek.com |

**其他方案：**
- 纯 SiliconFlow（单一供应商）
- OpenAI（海外用户）
- Ollama（本地离线）

详见 `.env.prod` 文件内注释。

### 🤓 Geek Mode (Claude Code) 配置

**Geek Mode** 是一项供开发者使用的高级功能，允许 Agent 通过 `Claude Code CLI` 直接操作服务器环境。
出于安全性考虑，**默认安装脚本不会启用该功能**，需要手动配置。

**前置条件：**
1. 获取 API Key: [智谱开放平台](https://bigmodel.cn/usercenter/proj-mgmt/apikeys) (推荐) 或 Anthropic 官方 Key。
2. 确保服务器已安装 `Node.js 18+` 环境。

**配置指南：**

#### 🅰️ 二进制部署 (推荐)
直接在服务器终端执行：

1. **安装工具**:
   ```bash
   npm install -g @anthropic-ai/claude-code
   ```
2. **自动配置认证**:
   ```bash
   npx @z_ai/coding-helper
   ```
3. **启用功能**:
   修改配置 `/etc/divinesense/config`:
   ```bash
   DIVINESENSE_CLAUDE_CODE_ENABLED=true
   ```
4. **重启服务**:
   ```bash
   systemctl restart divinesense
   ```

#### 🅱️ Docker 部署
需要进入容器内部执行安装（数据卷持久化）：

1. **安装工具 (需 Root 权限)**:
   ```bash
   #这是在容器内安装，无需担心污染宿主机
   docker exec -u 0 -it divinesense npm install -g @anthropic-ai/claude-code
   ```
2. **自动配置认证**:
   ```bash
   docker exec -it divinesense npx @z_ai/coding-helper
   ```
3. **启用功能**:
   修改 `/opt/divinesense/.env.prod` 文件：
   ```bash
   DIVINESENSE_CLAUDE_CODE_ENABLED=true
   ```
4. **重启服务**:
   ```bash
   cd /opt/divinesense && ./deploy.sh restart
   ```

### 🧬 Evolution Mode (进化模式) 配置

**Evolution Mode** 是最高级功能，允许 AI 自我修改 DivineSense 源代码。

**工作目录**: `/home/divine/source/divinesense`

**安全提示**: 仅限管理员使用，所有代码变更通过 GitHub PR 审核。

#### 二进制部署

1. **克隆源码**:
   ```bash
   sudo -u divine git clone https://github.com/hrygo/divinesense.git /home/divine/source/divinesense
   ```

2. **启用功能**:
   修改配置 `/etc/divinesense/config`:
   ```bash
   DIVINESENSE_CLAUDE_CODE_ENABLED=true
   DIVINESENSE_EVOLUTION_ENABLED=true
   DIVINESENSE_EVOLUTION_ADMIN_ONLY=true
   DIVINESENSE_EVOLUTION_SOURCE_DIR=/home/divine/source/divinesense
   ```

3. **重启服务**:
   ```bash
   systemctl restart divinesense
   ```

#### Docker 部署

修改 `/opt/divinesense/.env.prod` 文件：
```bash
DIVINESENSE_CLAUDE_CODE_ENABLED=true
DIVINESENSE_EVOLUTION_ENABLED=true
DIVINESENSE_EVOLUTION_ADMIN_ONLY=true
DIVINESENSE_EVOLUTION_SOURCE_DIR=/home/divine/source/divinesense
```

重启服务：
```bash
cd /opt/divinesense && ./deploy.sh restart
```

---

## 运维命令

### Docker 模式
```bash
cd /opt/divinesense

./deploy.sh status     # 查看状态
./deploy.sh logs       # 查看日志
./deploy.sh restart    # 重启服务
./deploy.sh stop       # 停止服务
./deploy.sh backup     # 手动备份
./deploy.sh upgrade    # 升级版本
```

### 二进制模式
```bash
systemctl status divinesense    # 查看状态
journalctl -u divinesense -f    # 查看日志
systemctl restart divinesense   # 重启服务
systemctl stop divinesense      # 停止服务

# 备份与升级
curl -fsSL https://raw.githubusercontent.com/hrygo/divinesense/main/deploy/install.sh | sudo bash -s -- --mode=binary
```

---

## 备份

**自动备份：** 每天凌晨 2 点（安装时已配置）

**手动备份：**
- Docker: `cd /opt/divinesense && ./deploy.sh backup`
- Binary: 使用 systemd 服务备份脚本

**恢复备份：**
- Docker: `./deploy.sh restore backups/backup-file.gz`
- Binary: 使用 pg_restore 或 sqlite 恢复

---

## 常见问题

| 问题           | 解决方案                         |
| -------------- | -------------------------------- |
| 镜像拉取慢     | 一键安装脚本已自动配置国内镜像源 |
| 服务无法启动   | 查看日志 (logs命令)              |
| 忘记数据库密码 | 查看 `.db_password` 文件         |
| 防火墙问题     | 确保开放 5230 端口               |

---

## 文件位置

**默认路径** (可通过环境变量 `DIVINE_INSTALL_DIR` 和 `DIVINE_CONFIG_DIR` 自定义)

### Docker 模式
```
/opt/divinesense/         # DIVINE_INSTALL_DIR
├── .env.prod             # 环境配置
├── .db_password          # 数据库密码
├── deploy.sh             # 运维脚本
└── backups/              # 备份目录

/home/divine/            # divine 用户家目录
├── .divinesense/        # Geek Mode 工作目录
└── source/              # Evolution Mode 源码目录
```

### 二进制模式
```
/opt/divinesense/         # DIVINE_INSTALL_DIR (默认)
├── bin/                  # 二进制文件
│   └── divinesense
├── data/                 # 数据目录
├── logs/                 # 日志目录
├── backups/              # 数据库备份
└── docker/               # PostgreSQL Docker 配置 (可选)
    ├── postgres.yml
    └── .env

/home/divine/            # divine 用户家目录
├── .divinesense/        # Geek Mode 工作目录
└── source/              # Evolution Mode 源码目录
    └── divinesense/     # 项目源码

/etc/divinesense/         # DIVINE_CONFIG_DIR (默认)
├── config                # 配置文件
└── .db_password          # 数据库密码 (640 权限, root:divine)

/etc/systemd/system/      # systemd 服务
└── divinesense.service   # User=divine
```
