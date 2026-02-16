# Commands vs Skills 深度分析

> **目标读者**: Claude Code 深度用户、插件开发者
> **核心问题**: 何时用 Commands，何时用 Skills，何时混合使用

---

## 一、根本区别

| 维度 | Commands | Skills |
|:-----|:---------|:--------|
| **触发方式** | 用户手动 (`/command`) | 手动 + Claude 自动匹配 |
| **本质定位** | 快捷指令别名 | 能力容器（可被发现） |
| **控制权** | 100% 用户控制 | 共享控制（Claude 可决定） |
| **可靠性** | 完全可预测 | 自动激活 ~50% 成功率 |
| **可见性** | 用户可见全部内容 | 支持 isMeta 隐藏内部指令 |
| **参数化** | 仅 argument-hint 提示 | 完整 YAML frontmatter |
| **复杂度** | 单文件 `.md` | 文件夹结构 + Workflows/Tools/ |
| **存储位置** | `~/.claude/commands/` | `~/.claude/skills/{name}/SKILL.md` |

---

## 二、技术机制

### Commands 工作流程

```
用户输入 /delegate
    ↓
Claude 读取 delegate.md 全部内容
    ↓
作为用户消息注入对话
    ↓
Claude 按照内容指导执行
```

**特点**：
- 完整内容注入，用户可见
- 无隐藏机制
- 简单直接

### Skills 工作流程

```
用户输入 /skill-name 或 Claude 匹配 description
    ↓
注入两条独立消息：
1. 可见元数据（命令消息，isMeta: false）
2. 隐藏完整指令（isMeta: true）
    ↓
Claude 按隐藏指令执行
```

**特点**：
- 双消息模式（用户看摘要，Claude 看完整）
- 支持 isMeta 隐藏实现细节
- 可包含脚本、参考文档、模板

### 文件结构对比

```bash
# Command - 单文件
~/.claude/commands/
└── delegate.md
    ---
    description: 委托 Agents Team 进行智能并行执行
    argument-hint: <任务描述>
    ---
    # 完整说明内容...

# Skill - 文件夹结构
~/.claude/skills/
└── delegation/
    ├── SKILL.md              # 核心配置
    ├── Workflows/            # 子流程
    │   ├── task-analysis.md
    │   ├── agent-matching.md
    │   └── result-aggregation.md
    ├── Tools/                # 可执行脚本
    │   └── helper.py
    └── References/           # 参考文档
        └── agent-matrix.md
```

---

## 三、自动激活的现实问题

根据 Scott Spence (2025-11) 的实战测试：

> **Skills 应该自动激活。实际上它们不会。即使显式 hook 指令也只有 50/50 的成功率。**

| 测试场景 | 成功率 |
|:---------|:-------|
| 全局 hook | 4/10 (40%) |
| 项目 hook | 5/10 (50%) |
| 基本等同于 | 抛硬币 |

### 描述工程 (Description Engineering)

通用描述完全失败：

```yaml
# ❌ 失败示例
description: Provides information about stakeholders

# ✅ 成功示例
description: Stakeholder context for Test Project when discussing product features,
UX research, or stakeholder interviews. Auto-invoke when user mentions Test Project,
product lead, or UX research. Do NOT load for general stakeholder discussions
unrelated to Test Project.
```

**关键成功因素**:
1. **WHEN + WHEN NOT 模式**: 明确边界防止误触发
2. **具体场景**: 必须告诉 Claude 内容、何时加载、何时不加载
3. **关键词限定**: 不要用过于宽泛的描述

---

## 四、决策框架

```
你想要什么？
│
├─ "想让 Claude 自动记住并主动使用"
│   └─ → Skills（但要做好心理准备：自动激活不稳定）
│
├─ "想自动化复杂工作流"
│   └─ → Skills + Workflows
│
├─ "频繁使用的明确任务，需要快捷方式"
│   └─ → Commands
│
├─ "想分享给团队使用"
│   └─ → Plugin（打包 Skills + Commands + Subagents）
│
└─ "需要隐藏内部实现细节"
    └─ → Skills（isMeta 支持）
```

### 何时用 Commands

| 场景 | 原因 |
|:-----|:-----|
| 用户显式触发的快捷操作 | 完全可控，100% 可靠 |
| 需要详细说明的复杂任务 | 完整上下文注入 |
| 频繁使用的标准流程 | 快速访问 |
| 不需要隐藏实现细节 | 透明度高 |

### 何时用 Skills

| 场景 | 原因 |
|:-----|:-----|
| 希望 Claude 在工作流中主动使用 | 可被发现 |
| 需要隐藏内部实现细节 | isMeta 支持 |
| 需要包含脚本、模板、参考文档 | 文件夹结构 |
| 复杂工作流需要模块化 | Workflows 支持 |

---

## 五、混合策略分析

### 结构

```
Command（入口点，用户视角）
    ↓ 简洁调用
Skill（能力库，实现视角）
    ↓ 包含
Workflows（子流程）
    ↓ 使用
Tools/Scripts（可执行资源）
```

### 真正的价值

#### 1. 分离"是什么"和"怎么做"

```
Command: "帮我并行调研 XXX"
  ↓ 只需要知道
Skill: 如何拆解、匹配、并行、聚合
  ↓ 隐藏复杂度
```

**好处**：
- 用户调用时不被实现细节淹没
- 技术细节可以独立演进
- 符合"接口与实现分离"原则

#### 2. 描述工程的精准控制

```yaml
# Skill 的 description 可以更精准
---
name: delegation
description: Parallel task orchestration using Task tool. USE WHEN user says
"delegate", "parallel", or explicitly asks to split work. Parse task into
3-6 independent subtasks, match optimal agent/MCP per task, launch ALL Task
calls in SINGLE response. Do NOT use for simple queries or single-agent tasks.
---

# Command 只需要简单描述
---
description: 委托 Agents Team 进行智能并行执行
---
```

#### 3. 渐进式复杂度

```
阶段1（起步）：只用 Command
/delegate → 所有说明在一个文件

阶段2（复杂度增加）：分离 Skill
/delegate（Command，5行） → delegation（Skill，详细逻辑）

阶段3（进一步复杂）：Skill 内部再拆分
delegation/
  ├── SKILL.md（入口路由）
  ├── Workflows/
  └── Tools/
```

### 成本分析

| 成本 | Command 单独 | Command + Skill |
|:-----|:-------------|:----------------|
| **文件数量** | 1 个 | 2+ 个 |
| **维护复杂度** | 低 | 中 |
| **调试难度** | 直接看一个文件 | 需要两个文件关联理解 |
| **学习曲线** | 低 | 中 |

### 何时混合策略不值得

```
场景判断：
│
├─ 任务简单（<50行说明能说清楚）
│   └─ → 单 Command 足够
│
├─ 仅供个人使用，不打算分享
│   └─ → 单 Command 够用
│
├─ 逻辑稳定，不经常变动
│   └─ → 单 Command 即可
│
└─ 任务复杂（>300行）、需要隐藏细节、可能演进
    └─ → Command + Skill 有价值
```

---

## 六、实战建议

### Daniel Miessler 的三层架构

```
┌─────────────────────────────────────────────────────────────┐
│                         AGENTS                              │
│            (Parallel workers - execute skills)              │
└─────────────────────────────────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────────┐
│                         SKILLS                              │
│              (Domain containers - 77+ skills)               │
│  ┌─────────────────────────────────────────────────────┐   │
│  │  Blogging/       Research/        Art/         ...  │   │
│  │  ├── SKILL.md    ├── SKILL.md    ├── SKILL.md       │   │
│  │  ├── Workflows/  ├── Workflows/  ├── Workflows/     │   │
│  │  └── Tools/      └── Tools/      └── Tools/         │   │
│  └─────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────────┐
│                       WORKFLOWS                             │
│           (Task procedures inside skills)                   │
└─────────────────────────────────────────────────────────────┘
```

### 推荐技术栈

```yaml
Orchestrator: UniversalParrot (统一 ReAct 循环)
Skills:
  - MemoSkill (SKILL.md + Workflows/)
  - ScheduleSkill (SKILL.md + Workflows/)
  - AmazingSkill (SKILL.md + Workflows/)
Commands:
  - /memo (快捷调用 MemoSkill)
  - /schedule (快捷调用 ScheduleSkill)
  - /amazing (快捷调用 AmazingSkill)
```

### 核心原则

> **复杂度足够时再拆分，而不是为了"最佳实践"而提前拆分。**

---

## 七、最佳实践总结

### Command 模板

```markdown
---
description: 简洁明确的功能描述
argument-hint: <参数提示>
---

# 命令标题

> **别名**: `/alias1`, `/alias2`

## 用法
```
/command <参数>
```

## 作用
[详细描述]

## 执行流程
[步骤说明]

## 示例
[使用示例]
```

### Skill 模板

```yaml
---
name: skill-name
description: WHEN to use this skill. Auto-invoke conditions. WHEN NOT to use.
version: 1.0.0
allowed-tools: "Read,Write,Bash"
---

# Skill Purpose

Brief statement of what this skill does.

## Workflow Routing

| Workflow | Trigger | File |
|----------|---------|------|
| **Task1** | "keyword1" | `Workflows/Task1.md` |
| **Task2** | "keyword2" | `Workflows/Task2.md` |

## Available Tools

- `tool_name`: Description

## Output Format
[输出格式说明]
```

### 混合策略模板

```bash
# 目录结构
~/.claude/
├── commands/
│   └── my-task.md          # 入口点，简洁
└── skills/
    └── my-task/
        ├── SKILL.md        # 详细逻辑
        ├── Workflows/      # 子流程
        └── Tools/          # 脚本
```

---

## 八、延伸阅读

- [Understanding Claude Code: Skills vs Commands vs Subagents](https://www.youngleaders.tech/p/claude-skills-commands-subagents-plugins) - Young Leaders Tech
- [When to Use Claude Code Skills vs Workflows vs Agents](https://danielmiessler.com/blog/when-to-use-skills-vs-commands-vs-agents) - Daniel Miessler
- [Complete Guide to Skills and Slash Commands](https://oneaway.io/blog/claude-code-skills-slash-commands) - OneAway
- [Claude Code Skills Don't Auto-Activate (a workaround)](https://scottspence.com/posts/claude-code-skills-dont-auto-activate) - Scott Spence
