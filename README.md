# DivineSense (神识)

**AI 驱动的个人第二大脑** — 通过智能代理自动化任务、过滤高价值信息、以技术杠杆提升生产力

[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE) [![Go](https://img.shields.io/badge/Go-1.25+-00ADD8.svg)](https://go.dev/) [![React](https://img.shields.io/badge/React-18-61DAFB.svg)](https://react.dev/)

[快速开始](#快速开始) • [功能特性](#功能特性) • [部署指南](#部署指南) • [开发文档](#开发文档)

---

## 快速开始

### 一键部署（推荐）

```bash
# 交互式安装（推荐新手）
curl -fsSL https://raw.githubusercontent.com/hrygo/divinesense/main/deploy/install.sh | sudo bash -s -- --interactive

# 二进制模式（Geek Mode）
curl -fsSL https://raw.githubusercontent.com/hrygo/divinesense/main/deploy/install.sh | sudo bash -s -- --mode=binary
```

### Docker 测试

```bash
docker run -d --name divinesense \
  -p 5230:5230 \
  -v ~/.divinesense:/var/opt/divinesense \
  hrygo/divinesense:stable
```

访问 http://localhost:5230

---

## 功能特性

| 类别               | 功能                                             |
| :----------------- | :----------------------------------------------- |
| **笔记**           | Markdown 编辑、语义搜索、AI 标签、附件管理       |
| **日程**           | 自然语言创建、冲突检测、多视图日历、周期事件     |
| **AI 代理**        | 五位智能代理（灰灰/时巧/折衷/极客/进化）协同工作 |
| **搜索**           | BM25 + 向量混合检索，精准定位内容                |
| **Geek Mode**      | Claude Code CLI 集成，自动化编码任务             |
| **Evolution Mode** | 系统自我进化，AI 修改源代码并提交 PR             |

---

## 部署指南

### 生产环境（云服务器）

```bash
# 二进制模式（Geek Mode 推荐）
curl -fsSL https://raw.githubusercontent.com/hrygo/divinesense/main/deploy/install.sh | sudo bash -s -- --mode=binary

# Docker 模式
curl -fsSL https://raw.githubusercontent.com/hrygo/divinesense/main/deploy/install.sh | sudo bash
```

**⚠️ 云服务器部署注意**：
- 安装后需在控制台开放安全组端口（默认 5230）
- 使用 80 端口需配置 `AmbientCapabilities=CAP_NET_BIND_SERVICE`
- 详见：[云服务器部署注意事项](docs/deployment/BINARY_DEPLOYMENT.md#云服务器部署注意事项)

**详细文档**：[部署指南](docs/deployment/BINARY_DEPLOYMENT.md) | [交互式向导](deploy/INTERACTIVE_WIZARD.md)

### 本地开发

```bash
git clone https://github.com/hrygo/divinesense.git && cd divinesense
make deps-all && make start
```

访问 http://localhost:25173，详见 [贡献指南](CONTRIBUTING.md)

---

## 开发文档

| 文档                                        | 说明                                        |
| :------------------------------------------ | :------------------------------------------ |
| [系统架构](docs/dev-guides/ARCHITECTURE.md) | AI 代理、数据流、项目结构                   |
| [后端开发](docs/dev-guides/BACKEND_DB.md)   | API、数据库、环境配置                       |
| [前端开发](docs/dev-guides/FRONTEND.md)     | 布局、组件、Tailwind 4                      |
| [Git 工作流](.claude/rules/git-workflow.md) | 分支管理、PR 规范                           |
| [贡献指南](CONTRIBUTING.md)                 | **入门必读**：环境搭建、开发规范、Checklist |

---

## 技术架构

**技术栈**：Go 1.25 + React 18 + PostgreSQL (pgvector) + DeepSeek V3

**AI 代理**：五位「鹦鹉」代理协同处理任务，支持意图路由、会话记忆、工具调用。

**Geek Mode**：集成 Claude Code CLI，全双工持久化会话架构。

**详细架构**：[系统架构文档](docs/dev-guides/ARCHITECTURE.md)

---

## 开源协议

[MIT](LICENSE) — 自由使用、修改和分发

---

## 致谢

本项目受到 [memos](https://github.com/usememos/memos) 启发。
