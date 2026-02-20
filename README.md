# DivineSense (神识)

<p align="center">
  <strong>AI 驱动的个人第二大脑</strong> — Orchestrator-Workers 多代理架构，动态任务分解与智能协作
</p>

<p align="center">
  <a href="LICENSE"><img src="https://img.shields.io/badge/license-MIT-blue.svg" alt="License"></a>
  <a href="https://github.com/hrygo/divinesense/releases"><img src="https://img.shields.io/badge/version-v0.101.0-green.svg" alt="Version"></a>
  <a href="https://go.dev/"><img src="https://img.shields.io/badge/Go-1.25+-00ADD8.svg" alt="Go"></a>
  <a href="https://react.dev/"><img src="https://img.shields.io/badge/React-18-61DAFB.svg" alt="React"></a>
</p>

---

## 为什么选择 DivineSense？

- **Orchestrator-Workers 架构**：LLM 动态分解任务、并行执行 Expert Agents、智能聚合结果
- **数据完全私有**：100% 自托管，无遥测，所有数据存储在您自己的服务器
- **极简单文件部署**：Go 语言编译的单二进制文件，零依赖，极低资源占用
- **Chat Apps 无缝集成**：原生支持 Telegram 和 钉钉，双向对话，随时随地记录与交互
- **Geek Mode**: 深度集成 Claude Code CLI，采用 **Hot-Multiplexing (热多路复用)** 架构，秒级响应代码执行与自动化任务
- **智能日程管理**：支持自然语言创建日程（如"明天下午三点开会"），自动冲突检测

---

## 核心功能

### Orchestrator-Workers 多代理架构

采用 Anthropic 推荐的 Orchestrator-Workers 模式，LLM 动态分解任务、并行执行、智能聚合：

```
用户输入 → Orchestrator → 任务分解 → Expert Agents 并行执行 → 结果聚合
```

| 组件             | 职责                            |
| :--------------- | :------------------------------ |
| **Orchestrator** | LLM 驱动的任务编排入口          |
| **Decomposer**   | 智能分解复杂请求，支持 DAG 依赖 |
| **Executor**     | 并行执行独立任务，降低延迟      |
| **Aggregator**   | 合并多代理结果，统一输出格式    |

### 专家代理 (Expert Agents)

| 代理               | 名称 | 核心能力                   | 实现方式                        |
| :----------------- | :--- | :------------------------- | :------------------------------ |
| **MemoParrot**     | 灰灰 | 语义搜索笔记，智能标签建议 | UniversalParrot + memo.yaml     |
| **ScheduleParrot** | 时巧 | 自然语言创建日程，冲突检测 | UniversalParrot + schedule.yaml |
| **GeneralParrot**  | 通才 | 通用任务，直接响应         | UniversalParrot + general.yaml  |

> **架构**: 所有领域代理基于 **UniversalParrot** 配置驱动实现，通过 YAML 配置定义行为

### 外部执行器 (External Executors)

| 代理                | 名称 | 核心能力                           |
| :------------------ | :--- | :--------------------------------- |
| **GeekParrot**      | 极客 | Claude Code CLI 集成，代码执行沙箱 |
| **EvolutionParrot** | 进化 | 自我进化，AI 修改源代码提交 PR     |

> **Note**: AmazingParrot 已被 Orchestrator 替代，其职责由 Orchestrator 动态协调 Expert Agents 完成。

**智能路由 (两层)**：Cache (LRU) → Rule Matcher (配置驱动)，置信度阈值 0.8

### 搜索与检索

- **混合检索**：BM25 + 向量搜索 + RRF 融合
- **自适应策略**：根据查询复杂度自动选择最佳检索方式
- **重排优化**：BAAI/bge-reranker-v2-m3 精炼结果，准确率提升 60%+

### 笔记管理

- **Markdown 编辑**：支持 KaTeX 公式、Mermaid 图表、GFM
- **智能标签**：AI 自动建议标签和分类
- **重复检测**：基于内容相似性自动去重
- **附件管理**：支持图片、文件等媒体内容

### 日程管理

- **自然语言解析**：支持"明天下午3点开会"等表达
- **冲突检测**：自动检测日程冲突并建议解决方案
- **多视图日历**：FullCalendar 集成，支持日/周/月视图
- **智能提醒**：基于用户习惯的个性化提醒

### Chat Apps 集成

- **多平台支持**：Telegram、钉钉、WhatsApp（实验性支持）
- **双向对话**：聊天应用中发送消息，AI 自动回复
- **安全加密**：AES-256-GCM 加密存储访问令牌
- **Webhook 验证**：HMAC-SHA256 签名 + 时间戳防重放

### Geek Mode — 代码执行

- **Claude Code CLI 集成**：Architecture v2.0 全双工热多路复用会话架构，零冷启动延迟
- **沙箱隔离**：基于 **UUID v5 确定性命名空间**，确保跨用户、跨模式状态物理级绝对隔离
- **危险命令检测**：正则防火墙自动拦截 `rm -rf` 等破坏性操作
- **生命周期管控**：支持 **PGID 进程组级回收**，随服务关闭自动清除孤儿进程，确保内存零泄漏
- **代码产物预览**：实时预览生成的网页和代码产物

### 成本追踪

- **实时统计**：Token 使用量、费用估算
- **缓存优化**：DeepSeek 上下文缓存，成本降低 70%+
- **预算告警**：支持每日预算设置和超额提醒

---

## 快速开始

### 一键安装（推荐）

```bash
# 交互式安装（推荐新手）
curl -fsSL https://raw.githubusercontent.com/hrygo/divinesense/main/deploy/install.sh | sudo bash -s -- --interactive

# 二进制模式（Geek Mode 推荐）
curl -fsSL https://raw.githubusercontent.com/hrygo/divinesense/main/deploy/install.sh | sudo bash -s -- --mode=binary

# Docker 模式
curl -fsSL https://raw.githubusercontent.com/hrygo/divinesense/main/deploy/install.sh | sudo bash -s -- --mode=docker
```

### Docker 快速测试

```bash
docker run -d --name divinesense \
  -p 5230:5230 \
  -v ~/.divinesense:/var/opt/divinesense \
  ghcr.io/hrygo/divinesense:stable
```

访问 http://localhost:5230

**注意**：生产环境部署需在云控制台开放安全组端口（默认 5230）

---

## 部署指南

### 部署模式对比

| 特性           | Docker 模式    | 二进制模式          |
| :------------- | :------------- | :------------------ |
| Geek Mode 支持 | ⚠️ 需额外配置   | ✅ 原生支持          |
| Evolution Mode | ❌ 不支持       | ✅ 原生支持          |
| 资源占用       | 高（容器开销） | 低                  |
| 启动速度       | 慢             | 快                  |
| 适用场景       | 快速部署、测试 | Geek Mode、生产环境 |

### 云服务器部署注意事项

1. **安全组配置**：安装后在控制台开放端口 5230
2. **使用 80 端口**：需配置 `AmbientCapabilities=CAP_NET_BIND_SERVICE`
3. **Nginx 反向代理**：可选，用于域名绑定

详见：[部署指南](docs/dev-guides/deployment/BINARY_DEPLOYMENT.md)

### 本地开发

```bash
git clone https://github.com/hrygo/divinesense.git && cd divinesense
make deps-all && make start
```

访问 http://localhost:25173 。详见 [贡献指南](CONTRIBUTING.md)

---

## 技术架构

**技术栈**：Go 1.25 + React 18 + Vite 7 + PostgreSQL (pgvector) + Tailwind CSS 4

**AI 模型**：
- **对话 LLM**：Z.AI GLM (`glm-4.7`)
- **向量 Embedding**：SiliconFlow (`BAAI/bge-m3`, 1024维)
- **意图分类**：SiliconFlow (`Qwen/Qwen2.5-7B-Instruct`)
- **重排 Rerank**：SiliconFlow (`BAAI/bge-reranker-v2-m3`)

**架构亮点**：
- **Orchestrator-Workers**：LLM 驱动的任务分解、并行执行、结果聚合
- **CC Runner v2.0**：深度集成 Claude Code，通过 **Hot-Multiplexing** 实现 Stdin/Stdout 流复用，挂载 PGID 进程组实现优雅销毁
- **单二进制分发**：Go embed 打包前端静态资源，零依赖部署
- **Connect RPC**：gRPC-HTTP 转码，类型安全的 API
- **Unified Block Model**：AI 聊天对话持久化，支持流式渲染和会话恢复
- **智能缓存层**：LRU 缓存 + DeepSeek 上下文缓存，响应延迟 <1ms

**详细架构**：[系统架构文档](docs/architecture/overview.md)

---

## 常见问题

### Q: SQLite 和 PostgreSQL 有什么区别？

**A**: PostgreSQL 支持完整 AI 功能（对话、记忆、路由），SQLite 仅支持向量搜索（开发测试）

### Q: Geek Mode 需要什么条件？

**A**: 需要安装 Claude Code CLI (`npm install -g @anthropic-ai/claude-code`) 并启用 `DIVINESENSE_CLAUDE_CODE_ENABLED=true`

### Q: 如何备份数据？

**A**: 使用 `/opt/divinesense/deploy-binary.sh backup` 或 `pg_dump` 导出数据库

### Q: AI 功能是否必须联网？

**A**: 是的，AI 功能需要调用外部 API（Z.AI GLM/SiliconFlow），但所有数据存储在您的服务器

---

## 开发文档

| 文档                                                        | 说明                             |
| :---------------------------------------------------------- | :------------------------------- |
| [系统架构](docs/architecture/overview.md)                   | AI 代理、数据流、项目结构        |
| [后端开发](docs/dev-guides/backend/database.md)             | API、数据库、环境配置            |
| [前端开发](docs/dev-guides/frontend/overview.md)            | 布局、组件、Tailwind 4           |
| [Chat Apps 指南](docs/dev-guides/user-manuals/chat-apps.md) | Telegram/钉钉机器人接入          |
| [Git 工作流](.claude/rules/git-workflow.md)                 | 分支管理、PR 规范                |
| [贡献指南](CONTRIBUTING.md)                                 | **入门必读**：环境搭建、开发规范 |

---

## 安全与隐私

- ✅ 无遥测、无数据上传
- ✅ 所有数据存储在您的服务器
- ✅ AES-256-GCM 加密（Chat Apps 凭证）
- ✅ 支持离线部署（内网环境）

---

## 社区

- **GitHub**: [hrygo/divinesense](https://github.com/hrygo/divinesense)
- **Issues**: [问题反馈](https://github.com/hrygo/divinesense/issues)
- **Discussions**: [功能讨论](https://github.com/hrygo/divinesense/discussions)

---

## 更新日志

查看 [CHANGELOG.md](CHANGELOG.md) 了解版本历史

---

## 开源协议

[MIT](LICENSE) — 自由使用、修改和分发

---

## 致谢

本项目受到 [memos](https://github.com/usememos/memos) 启发。
