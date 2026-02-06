# CLAUDE.md

> DivineSense 项目开发纲领 — Claude Code 辅助开发的核心指导文档

## 第一性原理

**DivineSense (神识)** = AI 代理驱动的个人「第二大脑」

```
核心使命：通过智能代理自动化任务、过滤高价值信息、以技术杠杆提升生产力

技术本质：Go 后端 + React 前端的单二进制分发应用
架构核心：五位「鹦鹉」AI 代理 + 统一块模型 (Unified Block Model)
```

---

## 架构原则

### 核心概念映射

| 概念     | 实体             | 关系                   |
| :------- | :--------------- | :--------------------- |
| **对话** | `AIConversation` | 包含多个 Block         |
| **块**   | `AIBlock`        | 一个用户-AI 交互轮次   |
| **代理** | `ParrotAgent`    | 处理用户请求的 AI 实体 |
| **路由** | `ChatRouter`     | 决定使用哪只鹦鹉       |

### 关键架构决策

**1. BlockMode ≠ ParrotAgentType** （最常混淆）
- `BlockMode.NORMAL/GEEK/EVOLUTION` — 消息块的结构模式
- `ParrotAgentType.AUTO/MEMO/SCHEDULE/...` — 哪只鹦鹉处理请求
- **无映射关系**：不要在代码中相互转换

**2. AUTO 不是鹦鹉**
- `AUTO` 是「请后端决定」的标记
- 后端三层路由：规则匹配 → 历史感知 → LLM 降级

**3. 数据库选择影响功能**
- PostgreSQL → 完整 AI 功能（向量搜索、对话持久化）
- SQLite → 仅开发环境，AI 功能禁用

---

## 工作流

### 多任务管理（重要）

> **原则**：始终使用 TODO LIST 跟踪多任务状态，避免"失忆"或迷失方向。

**何时创建 TODO LIST**：
- 分析日志/代码后发现**多个**优化点时
- 用户要求"逐个击破"多个问题时
- 任务预计超过 1 小时或包含多个步骤时

**操作流程**：
```bash
# 1. 分析完成后，为每个子任务创建 TODO
TaskCreate("优化项1标题", "详细描述...")
TaskCreate("优化项2标题", "详细描述...")

# 2. 查看当前任务列表
TaskList

# 3. 开始任务前，标记为 in_progress
TaskUpdate(taskId, status="in_progress")

# 4. 完成后标记为 completed
TaskUpdate(taskId, status="completed")
```

**状态流转**：
```
pending → in_progress → completed
    ↓                      ↓
  (开始)                (完成/删除)
```

**示例**（本会话实践）：
```
用户：分析日志优化空间 → 发现 5 个问题
AI：创建 4 个 TODO（第 1 个立即处理）
    → #1 优化 SessionStats [pending]
    → #2 合并数据库查询 [pending]     ← 下一个
    → #3 修复零值日志 [pending]
    → #4 删除重复日志 [pending]
```

### 开发前
```bash
make deps-all      # 安装依赖
make start         # 启动全栈
```

### 开发中
```bash
make check-all     # 提交前检查
make ci-check      # 模拟 CI
```

### 提交流程
1. `make check-all` 通过
2. 分支命名：`feat/xxx`、`fix/xxx`、`evolution/xxx`
3. 禁止直接 push 到 main
4. 通过 PR 合并

详细规范：@.claude/rules/git-workflow.md

---

## 编码规范

### DRY & SOLID 原则

> **减法 > 加法**：优先通过删除重复代码、合并相似功能来优化架构，而非添加新的抽象层。

#### DRY (Don't Repeat Yourself)

**核心原则**：每一处知识在系统中都必须有单一、无歧义、权威的表示。

```go
// ❌ 违反 DRY：重复的逻辑
func CreateMemo(name, content) { hashPassword(...) }
func UpdateMemo(id, name, content) { hashPassword(...) }  // 重复
func DeleteMemo(id) { hashPassword(...) }                   // 重复

// ✅ 遵循 DRY：提取单一函数
func (s *Service) hashPassword(pwd string) string { ... }
```

**实践案例**：
- 路由逻辑统一：`ai/router/Service` 提供三层路由，`ai/agent/chat_router.go` 复用而非重复实现
- 删除 492 行重复路由代码（v0.93.1）

#### SOLID 原则

| 原则 | 简记 | 说明 |
|:-----|:-----|:-----|
| **S** | 单一职责 | 每个模块只做一件事 |
| **O** | 开闭原则 | 扩展开放，修改封闭 |
| **L** | 里氏替换 | 子类可替换父类 |
| **I** | 接口隔离 | 拆分胖接口，客户端只依赖需要的接口 |
| **D** | 依赖倒置 | 依赖抽象而非具体实现 |

```go
// SRP + ISP：接口隔离，职责分离
type LLMClient interface {
    Complete(ctx, prompt, config) (string, error)  // 单一方法
}

// DIP：依赖接口，具体实现可替换
type Service struct {
    llmClient LLMClient  // 依赖抽象，非具体实现
}

// OCP：扩展新 LLM 无需修改 Service
func (s *Service) SetLLMClient(client LLMClient) {
    s.llmClient = client
}
```

**实践案例**：
- `router.LLMClient` 接口：定义清晰的契约
- `routerLLMClient` / `routerIntentLLMClient`：可互换的实现
- `router.Service` 通过接口依赖，而非直接依赖具体实现

### Go
- 文件：`snake_case.go`
- 日志：`log/slog`
- 错误：始终检查并处理

### React/TypeScript
- 组件：`PascalCase.tsx`
- Hooks：`use` 前缀
- 文本：`t("key")` 国际化

### Tailwind v4
- ❌ `max-w-md/lg/xl` → 解析错误 (~16px)
- ✅ `max-w-[24rem]` → 显式值

---

## 导航索引

| 任务         | 文档                                  |
| :----------- | :------------------------------------ |
| **理解架构** | @docs/dev-guides/ARCHITECTURE.md      |
| **后端开发** | @docs/dev-guides/BACKEND_DB.md        |
| **前端开发** | @docs/dev-guides/FRONTEND.md          |
| **部署**     | @docs/deployment/BINARY_DEPLOYMENT.md |
| **调试问题** | @docs/research/DEBUG_LESSONS.md       |
| **数据库迁移** | @store/migration/postgres/CLAUDE.md   |

---

## 产品能力边界

| 功能           | 状态                                 |
| :------------- | :----------------------------------- |
| 笔记           | ✅ Markdown + 语义搜索                |
| 日程           | ✅ 自然语言 + 冲突检测                |
| AI 代理        | ✅ 五位鹦鹉协同                       |
| Geek Mode      | ✅ Claude Code CLI 集成，用于复杂任务 |
| Evolution Mode | ✅ Claude Code CLI 集成，系统自我进化 |

---

*本文档随项目演进自动更新。新增功能时同步更新架构原则和导航索引。*
