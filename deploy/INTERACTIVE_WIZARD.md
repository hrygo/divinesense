# DivineSense 交互式部署向导

## 概述

DivineSense 提供交互式 TUI（终端用户界面）部署向导，引导用户完成完整的系统配置和部署。

## 快速开始

### 交互式向导（推荐）

```bash
# 下载并运行交互式向导
curl -fsSL https://raw.githubusercontent.com/hrygo/divinesense/main/deploy/interactive/wizard.sh | sudo bash

# 或从源码构建
go run ./cmd/deploy-wizard/main.go
```

### 一键脚本（无交互）

```bash
# Docker 模式（默认）
curl -fsSL https://raw.githubusercontent.com/hrygo/divinesense/main/deploy/aliyun/install.sh | sudo bash

# 二进制模式（Geek Mode 推荐）
curl -fsSL https://raw.githubusercontent.com/hrygo/divinesense/main/deploy/aliyun/install.sh | sudo bash -s -- --mode=binary
```

## 部署模式对比

| 特性 | Docker 模式 | 二进制模式 |
|:-----|:-----------|:---------|
| **Geek Mode 支持** | ⚠️ 需额外配置 | ✅ 原生支持 |
| **Evolution Mode 支持** | ❌ 不支持 | ✅ 原生支持 |
| **资源占用** | 高（容器开销） | 低 |
| **启动速度** | 慢 | 快 |
| **更新方式** | 重建镜像 | 替换二进制 |
| **适用场景** | 快速部署/测试 | 生产环境/Geek Mode |

## 交互式向导功能

### 1. 系统检查与依赖检测

自动检测以下依赖：
- Docker（可选）
- PostgreSQL（必需）
- Node.js + npm（Geek Mode 需要）
- Git（Evolution Mode 需要）
- Claude Code CLI（Geek Mode 需要）

### 2. 部署模式选择

支持两种部署模式：
- **Docker 模式**：容器化部署，环境隔离
- **二进制模式**：原生部署，性能最优

### 3. 数据库配置

支持三种 PostgreSQL 部署方式：
- **Docker 容器**（推荐）：自动部署独立容器
- **系统包安装**：使用系统包管理器安装
- **远程数据库**：连接已有 PostgreSQL 实例

### 4. AI 功能配置

支持多家 AI 提供商：
- **SiliconFlow**（推荐 - 国内网络优化）
- **DeepSeek**
- **OpenAI**（官方）

### 5. Geek Mode 配置

- 自动检测 Claude Code CLI 安装状态
- 引导用户完成 Claude Code CLI 安装
- 配置工作目录（默认 `/opt/divinesense/data`）

### 6. Evolution Mode 配置（高级）

- 检测 Git 仓库状态
- 验证源代码目录
- 配置管理员权限限制

### 7. 管理员账户配置

支持首次访问时创建或向导中配置

### 8. 配置摘要与确认

显示完整配置摘要，包括：
- 部署模式
- 安装目录
- 数据库配置
- AI 配置
- 高级模式状态

### 9. 自动安装

根据配置生成并执行安装脚本

## Geek Mode 部署

### 前置要求

- DivineSense 二进制模式部署
- Node.js 16+ 或 Claude Code CLI
- AI 功能已启用（配置 API Key）

### 安装 Claude Code CLI

```bash
# 方法 1: 官方 NPM 包（推荐）
npm install -g @anthropic-ai/claude-code
claude auth login

# 方法 2: 智谱 Coding Helper（国内网络优化）
npx @z_ai/coding-helper
```

### 配置 Geek Mode

编辑 `/etc/divinesense/config`：

```bash
# 启用 Geek Mode
DIVINESENSE_CLAUDE_CODE_ENABLED=true

# Geek Mode 工作目录（Claude Code CLI 执行目录）
DIVINESENSE_CLAUDE_CODE_WORKDIR=/opt/divinesense/data

# Claude Code CLI 路径（可选，自动检测）
# DIVINESENSE_CLAUDE_CODE_PATH=/usr/local/bin/claude
```

重启服务：

```bash
sudo systemctl restart divinesense
```

## Evolution Mode 部署

### 前置要求

- Geek Mode 已启用
- Git 已安装
- DivineSense 源代码目录可访问

### 配置 Evolution Mode

编辑 `/etc/divinesense/config`：

```bash
# 启用 Evolution Mode（需先启用 Geek Mode）
DIVINESENSE_CLAUDE_CODE_ENABLED=true
DIVINESENSE_EVOLUTION_ENABLED=true

# 仅管理员可用（推荐）
DIVINESENSE_EVOLUTION_ADMIN_ONLY=true
```

### 源代码目录配置

Evolution Mode 需要在 DivineSense 源代码根目录运行：

```bash
# 克隆源码（如果需要）
git clone https://github.com/hrygo/divinesense.git /opt/divinesense-src
cd /opt/divinesense-src

# 配置指向源码目录
export DIVINESENSE_CLAUDE_CODE_WORKDIR=/opt/divinesense-src
```

## 部署后管理

### 查看服务状态

```bash
# 二进制模式
sudo systemctl status divinesense

# Docker 模式
cd /opt/divinesense && ./deploy.sh status
```

### 查看日志

```bash
# 二进制模式
sudo journalctl -u divinesense -f

# Docker 模式
cd /opt/divinesense && ./deploy.sh logs
```

### 备份与恢复

```bash
# 二进制模式
/opt/divinesense/deploy-binary.sh backup
/opt/divinesense/deploy-binary.sh restore <backup-file>

# Docker 模式
cd /opt/divinesense && ./deploy.sh backup
cd /opt/divinesense && ./deploy.sh restore <backup-file>
```

### 升级版本

```bash
# 二进制模式
/opt/divinesense/deploy-binary.sh upgrade

# Docker 模式
cd /opt/divinesense && ./deploy.sh upgrade
```

## 故障排查

### 服务无法启动

```bash
# 查看详细错误
sudo journalctl -u divinesense -n 50 --no-pager

# 检查配置文件
sudo cat /etc/divinesense/config

# 检查端口占用
sudo ss -tlnp | grep 5230
```

### 数据库连接失败

```bash
# 检查 PostgreSQL 容器
docker ps | grep divinesense-postgres

# 测试连接
docker exec divinesense-postgres pg_isready -U divinesense

# 查看日志
docker logs divinesense-postgres
```

### Geek Mode 不可用

```bash
# 验证 Claude Code CLI
which claude
claude --version

# 检查配置
grep CLAUDE_CODE /etc/divinesense/config

# 检查权限
ls -la /opt/divinesense/data
```

### Evolution Mode 不可用

确保满足以下条件：
1. Geek Mode 已启用
2. Git 已安装
3. 工作目录指向源码根目录
4. 有管理员权限

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
├── bin/                   # 二进制
│   └── divinesense
├── data/                  # 工作目录 (Geek Mode)
├── logs/                  # 日志
├── backups/               # 数据库备份
├── docker/                # PostgreSQL Docker 配置
│   ├── postgres.yml
│   └── .env
└── deploy-binary.sh      # 管理脚本

/etc/divinesense/          # 配置目录
└── config                 # 环境变量
└── .db_password          # 数据库密码

/etc/systemd/system/       # systemd 服务
└── divinesense.service
```
