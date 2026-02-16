# 架构速览

> DivineSense 核心架构组件和概念快速参考

---

## Orchestrator-Workers 多代理架构

```
用户输入
    ↓
┌─────────────────────────────────────────┐
│            Orchestrator                 │  ← LLM 驱动任务分解
│  ┌─────────────────────────────────┐   │
│  │  Decomposer (任务分解)           │   │
│  │  Executor  (并行执行)            │   │
│  │  Aggregator (结果聚合)           │   │
│  └─────────────────────────────────┘   │
└────────────────┬────────────────────────┘
                 │
        ┌────────┴────────┐
        ↓                 ↓
┌───────────────┐ ┌───────────────┐
│ MemoParrot    │ │ ScheduleParrot│  ← Expert Agents (配置化)
│ (灰灰)        │ │ (时巧)        │
└───────────────┘ └───────────────┘
        │                 │
        └────────┬────────┘
                 ↓
         Aggregated Response
```

### 核心组件

| 组件 | 文件 | 职责 |
|:---|:---|:---|
| Orchestrator | `ai/agents/orchestrator/orchestrator.go` | 任务编排入口 |
| Decomposer | `ai/agents/orchestrator/decomposer.go` | 任务分解（DAG 依赖） |
| Executor | `ai/agents/orchestrator/executor.go` | 并行执行 |
| Aggregator | `ai/agents/orchestrator/aggregator.go` | 结果聚合 |
| ExpertRegistry | `ai/agents/orchestrator/expert_registry.go` | 专家代理注册 |

### 配置文件

| 配置 | 路径 |
|:---|:---|
| Decomposer 提示词 | `config/orchestrator/decomposer.yaml` |
| Aggregator 提示词 | `config/orchestrator/aggregator.yaml` |

---

## 专家代理 (Expert Agents)

| 鹦鹉                   | 领域                 | 配置文件                      |
| :--------------------- | :------------------- | :---------------------------- |
| MemoParrot (灰灰)      | 笔记搜索             | `config/parrots/memo.yaml`    |
| ScheduleParrot (时巧)  | 日程管理             | `config/parrots/schedule.yaml` |

### 外部执行器 (External Executors)

| 鹦鹉                   | 领域                 | 实现方式     |
| :--------------------- | :------------------- | :----------- |
| GeekParrot (极客)      | Claude Code CLI 桥接 | 代码实现     |
| EvolutionParrot (进化) | 自我进化             | 代码实现     |

> **注意**: AmazingParrot 已被 Orchestrator 替代，其职责由 Orchestrator 动态协调 Expert Agents 完成。

---

## 路由流程

```
用户输入 → ChatRouter → [高置信度] → 直接响应
                ↓
         [低置信度/多意图]
                ↓
         Orchestrator → Expert Agents → Response
```

> **简化**: 路由层 LLM 已移除，复杂请求直接转 Orchestrator

---

## 关键概念

| 概念 | 实体             | 说明                 |
| :--- | :--------------- | :------------------- |
| 对话 | `AIConversation` | 包含多个 Block       |
| 块   | `AIBlock`        | 一个用户-AI 交互轮次 |
| 任务 | `TaskPlan`       | Orchestrator 分解的结构化任务 |
| 专家 | `ExpertAgent`    | 领域专家代理         |

### BlockMode vs AgentType

| BlockMode | 说明                     | AgentType | 说明           |
| :--------- | :----------------------- | :--------- | :------------- |
| `normal`   | 普通模式，由后端路由决定 | `AUTO`     | 路由标记       |
| `geek`     | 极客模式                 | `GEEK`     | Claude Code   |
| `evolution`| 进化模式                 | `EVOLUTION`| 自我进化       |

**重要**：Mode 和 Type 是独立维度

---

## UniversalParrot 架构（Expert Agents 基础）

```
ExpertAgent (配置驱动)
├── ParrotFactory (工厂)
├── ParrotConfig (配置加载)
├── ExecutionStrategy (执行策略)
│   ├── DirectExecutor (原生调用)
│   ├── ReActExecutor (思考-行动循环)
│   └── PlanningExecutor (两阶段规划)
└── ToolRegistry (工具注册表)
```

### 执行策略对比

| 策略       | 适用场景         | 特点               |
| :--------- | :--------------- | :----------------- |
| `direct`   | 简单 CRUD        | 快速，单次调用     |
| `react`    | 多步推理         | 思考-行动循环      |
| `planning` | 复杂多工具协作   | 两阶段，可并行     |
| `reflexion`| 需要自我优化     | 反思迭代改进       |

---

## 项目目录结构

### 根目录
```
divinesense/
├── ai/                  # AI 代理、工具、路由（Go 一级模块）
├── server/              # HTTP/gRPC 服务器
├── store/               # 数据访问层
├── proto/               # Protobuf 定义
├── web/                 # React 前端（**前端根目录**）
├── docs/                # 项目文档
├── docker/              # Docker 配置
├── deploy/              # 部署脚本
└── scripts/             # 工具脚本
```

### 后端关键路径

| 功能模块     | 路径                       |
| :----------- | :------------------------- |
| AI 代理      | `ai/agent/`                |
| AI 核心      | `ai/core/`                 |
| 工具系统     | `ai/agent/tools/`          |
| 意图路由     | `ai/agent/chat_router.go`  |
| 笔记服务     | `server/service/memo/`     |
| 日程服务     | `server/service/schedule/` |
| 数据迁移     | `store/migration/postgres/` |

### 缓存层

Store 结构包含三层内存缓存：

| 缓存              | 用途           | TTL   |
| :---------------- | :------------- | :---- |
| `instanceSettingCache` | 实例配置  | 10min |
| `userCache`       | 用户信息       | 10min |
| `userSettingCache`| 用户设置       | 10min |

### Background Runners

| Runner           | 职责               | 路径                   |
| :--------------- | :----------------- | :--------------------- |
| Embedding Runner | 向量嵌入后台任务   | `ai/embedding/runner.go` |
| OCR Runner       | 文本提取           | `ai/ocr/runner.go`     |

### 前端关键路径

| 功能模块     | 路径                               |
| :----------- | :--------------------------------- |
| 布局组件     | `web/src/layouts/`                 |
| AI 聊天      | `web/src/components/AIChat/`       |
| Memo 组件    | `web/src/components/Memo/`         |
| 编辑器模块   | `web/src/components/MemoEditor/`   |
| 状态管理     | `web/src/contexts/`                |
| API Hooks    | `web/src/hooks/useAIQueries.ts` 等 |
| 国际化       | `web/src/locales/`                 |

---

*文档路径：docs/dev-guides/ARCHITECTURE_SUMMARY.md*
