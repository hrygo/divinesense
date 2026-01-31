# DivineSense (神识)

**AI 驱动的个人第二大脑** — 通过智能代理自动化任务、过滤高价值信息、以技术杠杆提升生产力

[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE) [![Go](https://img.shields.io/badge/Go-1.25+-00ADD8.svg)](https://go.dev/) [![React](https://img.shields.io/badge/React-18-61DAFB.svg)](https://react.dev/)

[快速开始](#快速开始) • [功能特性](#功能特性) • [部署指南](#部署指南) • [开发文档](#开发文档)



---

## 为什么选择 DivineSense？

| 🎯 **效率** | 🧠 **知识** | 🤖 **AI 代理** | 🔒 **隐私** |
| :--------: | :--------: | :-----------: | :--------: |
| 自动化任务 |  智能存储  |   意图路由    |   自托管   |
|  节省时间  |  语义搜索  |  多代理协作   |  数据隐私  |

---

## 功能特性

### 📝 笔记管理

- **Markdown 编辑器**：完整支持 KaTeX 数学公式、Mermaid 图表、GFM
- **智能标签系统**：AI 自动推荐相关标签
- **语义搜索**：BM25 + 向量混合检索，精准定位内容
- **笔记关联**：自动检测重复内容，建立知识网络
- **附件管理**：支持图片、文档等多类型附件
- **版本历史**：笔记修改全程可追溯

### 📅 日程管理

- **自然语言创建**：「明天下午3点开会」一句话搞定
- **智能冲突检测**：自动发现时间冲突并建议调整
- **多视图日历**：月/周/日/列表视图随心切换
- **周期事件**：支持每日/每周/每月/自定义重复
- **时区支持**：跨时区日程自动转换

### 🦜 AI 智能代理

五位「鹦鹉」代理，协同处理你的任务：

| 代理  | 名称     | 类型  | 定位             | 工作目录     | 产出                    |
| :---: | :------- | :---: | :--------------- | :----------- | :---------------------- |
|   🦜   | **灰灰** | 常规  | 知识检索         | —            | 搜索结果                |
|   🦜   | **金刚** | 常规  | 日程管理         | —            | 日程创建/查询           |
|   🦜   | **惊奇** | 常规  | 综合助理         | —            | 笔记+日程组合           |
|   🦜   | **极客** | 特殊  | 通用任务助手     | 用户沙箱     | 代码产物（用户下载）    |
|   🧬   | **进化** | 特殊  | 系统自我进化引擎 | 源代码根目录 | **GitHub PR**（需审查） |

**特殊模式详解**：

| 维度           | 🤖 极客模式 (Geek Mode)            | 🧬 进化模式 (Evolution Mode) |
| :------------- | :-------------------------------- | :-------------------------- |
| **定位**       | AI 帮你写代码                     | AI 修改系统自身代码         |
| **目标用户**   | 所有已登录用户                    | 仅管理员                    |
| **工作目录**   | `~/.divinesense/claude/user_{id}` | DivineSense 源代码根目录    |
| **产出物用途** | 用户浏览/下载                     | **强制 GitHub PR**          |
| **安全等级**   | 中（沙箱隔离）                    | 高（权限+PR审核）           |
| **视觉风格**   | Matrix 绿 / 终端                  | DNA 紫 / 科幻               |

### 📚 研究与最佳实践

- **[极客模式：CLI Agent 最佳实践](docs/research/BEST_PRACTICE_CLI_AGENT.md)** - 深入探讨如何优化类似 Claude Code 的 CLI 代理，实现更高效的自动化编码流程。
- **[进化模式规格说明书](docs/specs/evolution/EVOLUTION_MODE_SPEC.md)** - 详细了解系统自我进化的设计原则、安全机制与实现方案。
- **[系统架构图](docs/dev-guides/ARCHITECTURE.md)** - 了解 5 代理系统的通信机制与数据流向。

**智能路由**：
- 规则匹配（0ms）—— 常见模式瞬间响应
- 历史感知（~10ms）—— 结合对话上下文
- LLM 降级（~400ms）—— 复杂语义理解

**模式路由优先级**：
```
EvolutionMode (最高) → GeekMode → 常规代理路由
```

**会话记忆**：
- 跨会话上下文持续
- 30天自动保留
- 每个代理独立记忆空间

### 🧠 AI 增强功能

- **间隔重复复习**：基于记忆曲线的智能复习系统
- **知识图谱**：可视化笔记与日程的关联网络
- **每日回顾**：AI 生成的每日总结与洞察
- **向量检索**：pgvector 驱动的语义搜索
- **结果重排**：BGE-reranker 优化搜索精度

---

## 快速开始

### Docker 一键启动（基础笔记功能）

```bash
docker run -d --name divinesense \
  -p 5230:5230 \
  -v ~/.divinesense:/var/opt/divinesense \
  hrygo/divinesense:stable
```

访问 http://localhost:5230

### 完整 AI 功能（需要 PostgreSQL）

```bash
# 1. 克隆仓库
git clone https://github.com/hrygo/divinesense.git && cd divinesense

# 2. 配置环境变量
cp .env.example .env
# 编辑 .env，添加你的 API Keys

# 3. 安装依赖
make deps-all

# 4. 启动所有服务（PostgreSQL + 后端 + 前端）
make start
```

访问 http://localhost:25173

<details>
<summary><b>服务管理命令</b></summary>

```bash
make status   # 查看服务状态
make logs     # 查看日志
make stop     # 停止服务
make restart  # 重启服务
```

</details>

---

## 部署指南

### Docker 部署（推荐）

```bash
docker run -d --name divinesense \
  -p 5230:5230 \
  -v ~/.divinesense:/var/opt/divinesense \
  hrygo/divinesense:stable
```

### 二进制部署（Geek Mode 专用）

二进制部署提供更高性能和原生 Geek Mode 支持。

```bash
# 一键安装（默认 Docker 模式）
curl -fsSL https://raw.githubusercontent.com/hrygo/divinesense/main/deploy/aliyun/install.sh | sudo bash

# 二进制模式（支持 Geek Mode）
curl -fsSL https://raw.githubusercontent.com/hrygo/divinesense/main/deploy/aliyun/install.sh | sudo bash -s -- --mode=binary
```

**优势**：
- ✅ 原生 Geek Mode（Claude Code CLI 集成）
- ✅ 更快的启动速度，更低的资源开销
- ✅ 更便捷的升级流程

**详细文档**：[二进制部署指南](docs/deployment/BINARY_DEPLOYMENT.md)

---

## 技术架构

### 技术栈

| 层级        | 技术选型                                                |
| :---------- | :------------------------------------------------------ |
| **后端**    | Go 1.25+, Echo 框架, Connect RPC                        |
| **前端**    | React 18, Vite 7, TypeScript, Tailwind CSS 4, Radix UI  |
| **数据库**  | PostgreSQL 16+ (pgvector) [生产] / SQLite [开发，无 AI] |
| **AI 模型** | DeepSeek V3, Qwen2.5-7B, bge-m3, bge-reranker-v2-m3     |

### 混合 RAG 检索

```
查询 → 查询路由器 → BM25 + pgvector (HNSW) → 重排器 → RRF 融合
```

| 组件           | 技术                    | 用途       |
| :------------- | :---------------------- | :--------- |
| **向量搜索**   | pgvector + HNSW 索引    | 语义相似度 |
| **全文搜索**   | PostgreSQL FTS + BM25   | 关键词匹配 |
| **结果重排**   | BAAI/bge-reranker-v2-m3 | 结果精炼   |
| **文本向量化** | BAAI/bge-m3 (1024维)    | 文本向量化 |
| **大语言模型** | DeepSeek V3 / Qwen2.5   | 响应生成   |

### AI 代理架构

```
ChatRouter (意图分类)
    ├── EvolutionMode? ─Yes→ EvolutionParrot (进化) - 自我进化
    ├── GeekMode? ─Yes→ GeekParrot (极客) - Claude Code CLI 直接执行
    ├── 规则引擎 (0ms) - 关键词、模式匹配
    ├── 历史感知 (~10ms) - 对话上下文
    └── LLM 降级 (~400ms) - 语义理解

路由到：
    ├── EvolutionParrot (进化) - 源代码修改 + PR 创建
    ├── GeekParrot (极客) - Claude Code CLI 通信层（零 LLM）
    ├── MemoParrot (灰灰) - memo_search 工具
    ├── ScheduleParrotV2 (金刚) - schedule_add/query/update/find_free_time
    └── AmazingParrot (惊奇) - 并发多工具编排
```

---

## 开发指南

```bash
make start     # 启动所有服务
make stop      # 停止服务
make status    # 查看服务状态
make logs      # 查看日志
make test      # 运行测试
make check-all # 运行所有检查（构建、测试、i18n）
```

**开发文档**：
- [后端与数据库](docs/dev-guides/BACKEND_DB.md) - API、数据库结构、环境配置
- [前端架构](docs/dev-guides/FRONTEND.md) - 布局、Tailwind 注意事项、组件
- [系统架构](docs/dev-guides/ARCHITECTURE.md) - 项目结构、AI 代理、数据流

---

## 数据库架构

| 表名              | 用途                     |
| :---------------- | :----------------------- |
| `memo`            | 笔记主体内容             |
| `memo_embedding`  | 笔记向量嵌入（语义搜索） |
| `schedule`        | 日程主体                 |
| `ai_conversation` | AI 对话历史              |
| `episodic_memory` | 长期用户记忆和偏好       |
| `user_preference` | 用户沟通设置             |
| `agent_metrics`   | 代理性能追踪（A/B 测试） |

**数据库说明**：
- **PostgreSQL**：生产环境，完整 AI 支持（向量搜索、会话记忆、长期记忆）
- **SQLite**：开发环境，仅基础功能（**不支持 AI 功能**）

> 💡 **SQLite AI 支持计划**：详见 [#9](https://github.com/hrygo/divinesense/issues/9) - 探索 SQLite 向量搜索可行性

---

## 开源协议

[MIT](LICENSE) — 自由使用、修改和分发

---

## 致谢

本项目受到优秀的 [memos](https://github.com/usememos/memos) 项目启发。usememos 社区在隐私优先的笔记管理方面的工作，为 DivineSense 的许多核心功能奠定了基础。
