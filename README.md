# DivineSense (神识)

<p align="center">
  <strong>AI 驱动的个人第二大脑</strong> — 通过五位智能代理自动化任务、过滤高价值信息、以技术杠杆提升生产力
</p>

<p align="center">
  <a href="LICENSE"><img src="https://img.shields.io/badge/license-MIT-blue.svg" alt="License"></a>
  <a href="https://github.com/hrygo/divinesense/releases"><img src="https://img.shields.io/badge/version-v0.97.0-green.svg" alt="Version"></a>
  <a href="https://go.dev/"><img src="https://img.shields.io/badge/Go-1.25+-00ADD8.svg" alt="Go"></a>
  <a href="https://react.dev/"><img src="https://img.shields.io/badge/React-18-61DAFB.svg" alt="React"></a>
</p>

---

## 为什么选择 DivineSense？

- **AI 原生架构**：内置五位智能代理（Memo/Schedule/Amazing/Geek/Evolution），深度协同工作
- **数据完全私有**：100% 自托管，无遥测，所有数据存储在您自己的服务器
- **极简单文件部署**：Go 语言编译的单二进制文件，零依赖，极低资源占用
- **Chat Apps 无缝集成**：原生支持 Telegram 和 钉钉，双向对话，随时随地记录与交互
- **Geek Mode**：深度集成 Claude Code CLI，提供独立沙箱环境执行代码与自动化任务
- **智能日程管理**：支持自然语言创建日程（如"明天下午三点开会"），自动冲突检测

---

## 核心功能

### AI 代理系统 — 五位「鹦鹉」协同工作

| 代理                | 名称 | 执行策略     | 核心能力                        |
| :------------------ | :--- | :----------- | :------------------------------ |
| **MemoParrot**      | 灰灰 | ReAct 循环   | 语义搜索笔记，智能标签建议      |
| **ScheduleParrot**  | 时巧 | 原生工具调用 | 自然语言创建日程，冲突检测      |
| **AmazingParrot**   | 折衷 | 两阶段规划   | 综合助理，多工具并发执行        |
| **GeekParrot**      | 极客 | Playground   | Claude Code CLI 集成，AI 试验场 |
| **EvolutionParrot** | 进化 | 自我进化     | AI 修改源代码，提交 GitHub PR   |

**智能路由**：四层意图分类（Cache → Rule → History → LLM），响应延迟 0-400ms

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

- **多平台支持**：Telegram、钉钉、WhatsApp（预留）
- **双向对话**：聊天应用中发送消息，AI 自动回复
- **安全加密**：AES-256-GCM 加密存储访问令牌
- **Webhook 验证**：HMAC-SHA256 签名 + 时间戳防重放

### Geek Mode — 代码执行

- **Claude Code CLI 集成**：全双工持久化会话架构
- **沙箱隔离**：用户工作目录独立，安全执行
- **危险命令检测**：自动拦截 rm -rf 等破坏性操作
- **代码产物预览**：实时预览生成的网页和代码

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

详见：[部署指南](docs/deployment/BINARY_DEPLOYMENT.md)

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
- **CC Runner**：深度集成 Claude Code，通过 PTY 实现全双工交互与会话持久化
- **单二进制分发**：Go embed 打包前端静态资源，零依赖部署
- **Connect RPC**：gRPC-HTTP 转码，类型安全的 API
- **Unified Block Model**：AI 聊天对话持久化，支持流式渲染和会话恢复
- **智能缓存层**：LRU 缓存 + DeepSeek 上下文缓存，响应延迟 <1ms

**详细架构**：[系统架构文档](docs/dev-guides/ARCHITECTURE.md)

---

## 常见问题

### Q: SQLite 和 PostgreSQL 有什么区别？

**A**: PostgreSQL 支持完整 AI 功能（对话、记忆、路由），SQLite 仅支持向量搜索（开发测试）

### Q: Geek Mode 需要什么条件？

**A**: 需要安装 Claude Code CLI (`npm install -g @anthropic-ai/claude-code`) 并启用 `DIVINESENSE_CLAUDE_CODE_ENABLED=true`

### Q: 如何备份数据？

**A**: 使用 `/opt/divinesense/deploy-binary.sh backup` 或 `pg_dump` 导出数据库

### Q: AI 功能是否必须联网？

**A**: 是的，AI 功能需要调用外部 API（DeepSeek/SiliconFlow），但所有数据存储在您的服务器

---

## 开发文档

| 文档                                            | 说明                             |
| :---------------------------------------------- | :------------------------------- |
| [系统架构](docs/dev-guides/ARCHITECTURE.md)     | AI 代理、数据流、项目结构        |
| [后端开发](docs/dev-guides/BACKEND_DB.md)       | API、数据库、环境配置            |
| [前端开发](docs/dev-guides/FRONTEND.md)         | 布局、组件、Tailwind 4           |
| [Chat Apps 指南](docs/user-guides/CHAT_APPS.md) | Telegram/钉钉机器人接入          |
| [Git 工作流](.claude/rules/git-workflow.md)     | 分支管理、PR 规范                |
| [贡献指南](CONTRIBUTING.md)                     | **入门必读**：环境搭建、开发规范 |

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
