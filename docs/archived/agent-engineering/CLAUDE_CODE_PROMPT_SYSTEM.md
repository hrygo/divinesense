# Claude Code 提示词系统机制

> **目标读者**: Claude Code 深度用户、技能开发者
> **核心内容**: 系统提示词注入、Skills 系统、上下文管理

---

## 系统架构

### 多层提示词注入

```
┌─────────────────────────────────────────────────────────────────┐
│                     Claude Code 提示词架构                       │
├─────────────────────────────────────────────────────────────────┤
│  层级        │ 位置               │ 注入方式                    │
├─────────────────────────────────────────────────────────────────┤
│  System      │ 硬编码系统提示词   │ API 请求的 system 字段     │
│  CLAUDE.md   │ 项目/用户配置     │ User 消息 + isMeta: false  │
│  Auto Memory │ ~/.claude/projects/ │ User 消息（首 200 行）     │
│  <system-reminder> │ 工具调用结果  │ 工具结果中动态注入         │
│  Skills      │ .claude/skills/    │ User 消息 + isMeta: true   │
└─────────────────────────────────────────────────────────────────┘
```

### 缓存控制

Claude Code 使用 `cache_control: {type: "ephemeral"}` 实现系统提示词缓存：

```javascript
{
  "model": "claude-sonnet-4-5-20250929",
  "system": [
    {
      "text": "You are Claude Code...",
      "type": "text",
      "cache_control": {"type": "ephemeral"}
    }
  ],
  "messages": [...]
}
```

---

## <system-reminder> 标签机制

这是 Claude Code 最核心的设计之一，贯穿整个系统。

### 用途

```markdown
<system-reminder>
As you answer the user's questions, you can use the following context:
# important-instruction-reminders
Do what has been asked; nothing more, nothing less.
NEVER create files unless they're absolutely necessary...
ALWAYS prefer editing an existing file to creating a new one...
...
IMPORTANT: this context may or may not be relevant to your tasks.
</system-reminder>
```

### 注入位置

- 系统提示词中
- 用户消息中
- 工具调用结果中
- Bash 工具执行前（命令前缀检测）
- TodoWrite 工具结果中

---

## Skills 系统

### Skills 本质

Skills **不是可执行代码**，而是**提示词模板**，通过上下文注入修改 Claude 的行为：

| 特征 | Traditional Tools | Skills |
|:-----|:------------------|:--------|
| 执行模型 | 同步直接执行 | 提示词扩展 |
| 目的 | 执行特定操作 | 指导复杂工作流 |
| 返回值 | 立即结果 | 对话上下文 + 执行上下文修改 |
| 类型 | 各种类型 | 总是 `"prompt"` |

### Skill 定义结构

```
my-skill/
├── SKILL.md              # 核心提示词和指令
├── scripts/              # 可执行 Python/Bash 脚本
├── references/           # 加载到上下文的文档
└── assets/               # 模板和二进制文件
```

### SKILL.md 模板

```yaml
---
name: skill-name
description: Clear description of when to use this skill
allowed-tools: "Read,Write,Bash,Glob,Grep"
license: MIT
version: 1.0.0
---

# Skill Purpose

Brief statement of what this skill does.

## Overview

[What this skill does, when to use it, what it provides]

## Instructions

### Step 1: [First Action]
[Imperative instructions]

### Step 2: [Next Action]
[Imperative instructions]

## Output Format
[How to structure results]

## Error Handling
[What to do when things fail]

## Resources
- {baseDir}/scripts/helper.py - Helper script
- {baseDir}/references/spec.md - Detailed specs
```

### 双消息模式

Skills 执行时注入**两条独立用户消息**：

```javascript
// 消息 1：用户可见的元数据
{
  role: "user",
  content: `<command-message>The "pdf" skill is loading</command-message>
<command-name>pdf</command-name>`,
  // isMeta: false (默认) → 可见
}

// 消息 2：隐藏的完整技能提示词
{
  role: "user",
  content: "You are a PDF processing specialist...\n## Process\n1. Validate...",
  isMeta: true  // 隐藏但发送给 API
}
```

### Skill 选择机制

**无算法路由** - Skills 依赖**纯 LLM 推理**：

```
用户: "Extract text from report.pdf"
      ↓
Claude 推理:
- User wants to extract text from PDF
- "pdf": Extract text from PDF documents
- Match! Invoke Skill tool
      ↓
Tool Call: {name: "Skill", input: {command: "pdf"}}
```

---

## 层级记忆系统

### 七种记忆类型

| 类型 | 位置 | 用途 | 共享范围 |
|:-----|:-----|:-----|:---------|
| Managed policy | 系统级目录 | 组织级指令 | 所有用户 |
| Project memory | `./CLAUDE.md` | 项目级指令 | 团队（版本控制） |
| Project rules | `./.claude/rules/*.md` | 模块化指令 | 团队（版本控制） |
| User memory | `~/.claude/CLAUDE.md` | 个人偏好 | 仅自己（所有项目） |
| Project local | `./CLAUDE.local.md` | 个人项目偏好 | 仅自己（当前项目） |
| Auto memory | `~/.claude/projects/<project>/memory/` | 自动学习笔记 | 仅自己（每个项目） |

### 加载优先级

更具体的指令覆盖更广泛的指令：
```
Project rules > Project memory > User memory > Managed policy
```

### Auto Memory

```
~/.claude/projects/<project>/memory/
├── MEMORY.md          # 索引文件（首 200 行自动加载）
├── debugging.md       # 按需加载的主题文件
├── api-conventions.md
└── ...
```

**特点**：
- `MEMORY.md` 首 200 行在每次会话开始时加载
- 主题文件按需读取
- Claude 在会话中读写这些文件

---

## CLAUDE.md 最佳实践

### 核心原则：Less is More

- 前沿 LLM 可可靠遵循约 **150-200 条指令**
- 指令数量增加 → 性能**线性/指数衰减**
- LLM 偏向关注**提示词首尾**的指令

### 应包含的内容

遵循 **WHY-WHAT-HOW** 结构：

```markdown
# WHY - 项目目的
## 项目概述
[项目核心目的，解决的问题]

# WHAT - 技术栈和结构
## 技术栈
- Go 1.25 + Echo + Connect RPC

## 目录结构
| 目录 | 说明 |
|:-----|:-----|
| ai/ | AI 核心模块 |

# HOW - 工作方式
## 开发命令
- 启动: `make start`
- 测试: `make test`

## 核心规范
### 命名约定
- Go: `snake_case.go`
- React: `PascalCase.tsx`
```

### 应避免的内容

| 类型 | 示例 | 原因 |
|:-----|:-----|:-----|
| 过长 | >300 行 | 降低指令遵循率 |
| 代码风格 | 具体格式规则 | 使用 linter 替代 |
| 罕见任务 | 数据库迁移流程 | 使用 Progressive Disclosure |
| 过时信息 | 旧的 API 调用 | 保持同步困难 |

### Progressive Disclosure 模式

将任务特定指令放在独立文件中：

```
docs/
├── building_the_project.md
├── running_tests.md
├── code_conventions.md
└── service_architecture.md
```

在 CLAUDE.md 中指向这些文件：

```markdown
## 上下文文档
在开始工作前，阅读相关的 @docs/ 下的文档：
- building_the_project.md - 构建流程
- running_tests.md - 测试约定
```

### 推荐模板

```markdown
# <项目名称> Claude 指南

> **保鲜状态**: ✅ YYYY-MM-DD | **最后检查**: v0.XX.0

## 项目概述
[1-2 句话描述项目核心目的]

## 技术栈
- **后端**: [语言] + [框架] + [协议]
- **前端**: [框架] + [构建工具] + [UI 库]

## 目录结构
| 路径 | 用途 |
|:-----|:-----|
| `dir1/` | 说明 |
| `dir2/` | 说明 |

## 开发命令
| 命令 | 用途 |
|:-----|:-----|
| `make cmd` | 说明 |

## 核心规范
### 命名约定
- **Go**: `snake_case.go`
- **React**: `PascalCase.tsx`

### 错误处理
- [具体规则]

## 关键链接
- 详细架构: @docs/ARCHITECTURE.md
- API 文档: @docs/API.md

## 环境变量
```bash
KEY1=value1
KEY2=value2
```

## 常见陷阱
| 问题 | 说明 |
|:-----|:-----|
| 陷阱1 | 解决方案 |
```

---

## 模块化 Rules 系统

`.claude/rules/` 目录支持路径特定规则：

```markdown
---
paths:
  - "src/api/**/*.ts"
  - "lib/**/*.ts"
---

# API 开发规则
- 所有端点必须包含输入验证
- 使用标准错误响应格式
```

### 特性

- 支持 glob 模式匹配
- 支持 brace expansion: `src/**/*.{ts,tsx}`
- 支持子目录组织
- 支持 symlinks 共享规则
- 无 `paths` 字段的规则全局应用

---

## CLAUDE.md 导入机制

支持 `@path` 语法递归导入：

```markdown
# 项目概览
See @README for overview and @package.json for commands.

# 额外指令
- Git 工作流 @docs/git-instructions.md
- 个人指令 @~/.claude/my-preferences.md
```

**规则**：
- 相对路径相对于包含文件解析
- 最大递归深度：5 跳
- 不会在代码块中评估导入

---

## 持久化提示词知识库组织

### 推荐目录结构

```
project-root/
├── CLAUDE.md                    # 主入口（< 60 行推荐）
├── CLAUDE.local.md              # 个人本地配置（不提交）
├── .claude/
│   ├── CLAUDE.md                # 项目级指令
│   └── rules/
│       ├── code-style.md        # 代码风格
│       ├── testing.md           # 测试约定
│       └── paths/
│           ├── api/
│           │   └── endpoints.md  # API 特定规则
│           └── frontend/
│               └── components.md # 前端特定规则
├── docs/
│   └── agent_docs/              # 详细文档（Progressive Disclosure）
│       ├── architecture.md
│       ├── workflows.md
│       └── conventions.md
└── ~/.claude/
    ├── CLAUDE.md                # 全局个人偏好
    ├── projects/
    │   └── <project-hash>/
    │       └── memory/
    │           ├── MEMORY.md    # Auto memory 索引
    │           └── patterns.md  # 学习到的模式
    └── skills/                  # 自定义 Skills
        └── my-skill/
            └── SKILL.md
```

### 内容组织原则

1. **主 CLAUDE.md 保持简洁** (< 300 行，< 60 行更佳)
2. **使用 Progressive Disclosure** - 详细内容放在独立文件
3. **优先指针而非副本** - 引用文件路径而非复制内容
4. **层级覆盖** - 具体 > 一般，项目 > 用户 > 组织
5. **路径特定规则** - 使用 `.claude/rules/` + `paths` 字段

---

## 常见 Skill 模式

### Pattern 1: Script Automation

```yaml
---
name: analyzer
description: Analyze codebase for security vulnerabilities
allowed-tools: "Bash(python {baseDir}/scripts/*:*), Read, Write"
---

Run analysis:
```bash
python {baseDir}/scripts/analyzer.py --path "$USER_PATH"
```
Parse and report findings.
```

### Pattern 2: Read-Process-Write

```yaml
---
name: transformer
description: Transform data between formats
allowed-tools: "Read, Write"
---

## Workflow
1. Read input file
2. Transform per specifications
3. Write output
4. Report completion
```

### Pattern 3: Search-Analyze-Report

```yaml
---
name: auditor
description: Security audit for specific patterns
allowed-tools: "Grep, Read"
---

## Process
1. Use Grep to find patterns
2. Read matched files
3. Generate report
```

---

## 延伸阅读

- [Claude Code Docs - Manage Memory](https://code.claude.com/docs/en/memory)
- [Claude Skills Architecture Decoded](https://medium.com/aimonks/claude-skills-architecture-decoded-from-prompt-engineering-to-context-engineering-a6625ddaf53c)
- [Writing a good CLAUDE.md](https://www.humanlayer.dev/blog/writing-a-good-claude-md)
