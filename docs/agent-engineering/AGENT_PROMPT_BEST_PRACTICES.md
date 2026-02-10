# Agent 提示词工程指南

> **目标读者**: AI Agent 工程师、提示词优化者
> **核心原则**: 提示词工程 → 上下文工程

---

## 核心原则

### 1. 从提示词工程到上下文工程

随着 LLM 上下文窗口扩大，焦点从"找对词"转向"优化整个上下文配置"：

| 概念 | 说明 |
|:-----|:-----|
| **Context Rot** | token 数量增加会导致模型准确率下降 |
| **注意力预算有限** | 每个新 token 都消耗模型的注意力资源 |
| **上下文是有限资源** | 必须精心策划进入上下文的信息 |

### 2. System Prompt 应像"简短合同"

优秀的 System Prompt 特征：
- **明确性**：清晰定义行为边界
- **可验证**：易于检查是否符合预期
- **适度抽象**：避免硬编码脆弱逻辑，也避免过于模糊的高层指导

### 3. 结构化分节组织

```markdown
# Identity
角色、目的、沟通风格

# Instructions
具体规则和约束

# Tool Guidance
工具使用指南

# Output Description
输出格式要求
```

使用 XML 标签或 Markdown 标题分隔各部分（如 `<background_information>`、`<instructions>`）

### 4. 示例胜过千言万语

- 提供**多样化的代表性示例**而非穷举所有边缘情况
- 使用 XML 标签分隔示例和实际任务
- 示例应展示预期的行为模式，而非堆砌规则列表

### 5. 简洁优于冗长

前沿 LLM 可可靠遵循约 **150-200 条指令**，指令数量增加 → 性能线性/指数衰减。

### 6. 分层使用消息角色

| 角色 | 优先级 | 用途 |
|:-----|:-------|:-----|
| `developer` | 最高 | 系统规则、业务逻辑 |
| `user` | 中等 | 用户输入、配置参数 |
| `assistant` | - | 模型生成的响应 |

---

## 常见反模式及修复

### 反模式 1：过度硬编码逻辑

```markdown
❌ 错误：复杂 if-else 规则堆砌
"如果用户说X，则做Y；如果用户说A，则做B..."

✅ 正确：高层次的启发式指导
"你是X领域的专家助手。遵循Y原则，优先考虑Z..."
```

### 反模式 2：示例过载

```markdown
❌ 错误：堆砌边缘情况
"以下是20个可能的边缘情况及其处理方式..."

✅ 正确：精选代表性示例
"以下是3个代表性输入-输出对，展示预期模式..."
```

### 反模式 3：模糊的"假设共享上下文"

```markdown
❌ 错误："使用标准格式"（标准是什么？）
❌ 错误："像往常一样处理"（没有惯例可循）

✅ 正确：明确指定
"使用以下格式：[具体格式说明]"
```

### 反模式 4：忽略 Context Rot

```markdown
❌ 错误：将整个文档历史加载到上下文

✅ 正确：压缩 + 按需检索
- 维护外部记忆（NOTES.md）
- 使用子代理处理深度任务
- 只返回摘要信息
```

---

## 提示词模板

### 基础模板

```markdown
# Identity
你是 [角色描述]，你的核心目标是 [目标]。
你的沟通风格是 [风格描述]。

# Background Information
<context>
[必要的背景信息，使用可替换部分]
</context>

# Core Instructions
1. [首要规则]
2. [次要规则]
3. [约束条件]

# Tool Guidance
[如果有工具，说明如何使用]

# Output Format
<output_format>
[明确的输出格式要求，如 JSON schema]
</output_format>

# Examples
<example_input>
[示例输入]
</example_input>

<example_output>
[示例输出]
</example_output>
```

### 高级模板（用于 Agent）

```markdown
## System Message (Immutable)
<system>
# Identity
你是 [domain] 专家助手，专注于 [specific_focus]。

# Behavioral Constraints
- 必须：[must-do]
- 禁止：[must-not-do]
- 优先级：[priority-order]

# Interaction Protocol
1. 理解用户意图
2. 规划解决步骤
3. 执行（使用工具）
4. 验证结果
</system>

## Task Specification (Per-Request)
<task>
# Current Objective
[具体任务描述]

# Available Tools
- tool_name: [用途描述]
- tool_name: [用途描述]

# Context
[任务特定上下文]
</task>
```

### 参数化模板

```yaml
template_id: "customer_response_v1"
version: "2.0"

system_prompt: |
  You are a {{tone}} customer service agent for {{company_name}}.

  # Guidelines
  - Response length: {{max_length}} words
  - Include: {{required_elements}}
  - Exclude: {{forbidden_topics}}

variables:
  tone:
    type: enum
    values: [professional, friendly, empathetic]
    default: professional

  company_name:
    type: string
    required: true

  max_length:
    type: integer
    default: 100
```

---

## 思维链提示模式

### CoT（Chain-of-Thought）

```markdown
# 标准 CoT
"Let's think step by step. First... Then... Finally..."

# 结构化 CoT（推荐）
<thinking>
[推理过程]
</thinking>

<answer>
[最终答案]
</answer>
```

### ReAct 模式

```markdown
Thought: [分析当前状态]
Action: [选择工具/行动]
Observation: [观察结果]
... (重复)
Thought: [得出结论]
Answer: [最终答案]
```

### Reflexion（自反思）

```markdown
1. 初步尝试
2. 反思："哪里出错了？"
3. 修正："下次如何改进？"
4. 重试
```

---

## 长上下文管理策略

### 1. 压缩

当上下文接近窗口限制时触发：
- 保留关键信息（决策、bug、实现细节）
- 丢弃冗余内容（工具输出、重复消息）

### 2. 结构化笔记

```markdown
<!-- 项目记忆模板 -->
## Project: [名称]

### Completed
- [任务1]: 结果
- [任务2]: 结果

### In Progress
- [任务3]: 当前状态

### Decisions Made
- [决策1]: 理由

### Next Steps
1. [下一步1]
2. [下一步2]
```

### 3. 子代理架构

- 主代理：高层规划、结果综合
- 子代理：深度探索、返回摘要（1000-2000 tokens）

---

## Token 预算分配

```
Token 预算分配（带检索）：
┌─────────────────────────────────────────┐
│ System Prompt      │ 500 tokens（固定） │
│ User Preferences   │ 10%                │
│ Short-term Memory  │ 40%                │
│ Long-term Memory   │ 15%                │
│ Retrieval Results  │ 45%                │
└─────────────────────────────────────────┘
```

---

## 提示词 vs 代码：边界划分

### 核心原则

| 职责 | 提示词 | 代码 |
|:-----|:-------|:-----|
| **角色定义** | ✅ 80% | ❌ 20% |
| **输出格式** | ✅ 90% | ❌ 10% |
| **工具选择** | ✅ 40% | ❌ 60% |
| **数据验证** | ❌ 10% | ✅ 90% |
| **安全检查** | ❌ 10% | ✅ 90% |
| **重试逻辑** | ❌ 5% | ✅ 95% |
| **缓存策略** | ❌ 5% | ✅ 95% |
| **流式控制** | ❌ 5% | ✅ 95% |

**原则**：提示词用于行为指导、灵活性、可解释性；代码用于确定性逻辑、性能关键路径、安全性。

---

## 工具调用提示设计

### 工具描述模板

```yaml
tool_name: "search_customer_records"
description: >
  Search for customer records by name, email, or ID.
  Returns the most recent and relevant records first.

  Use this when:
  - User asks about customer information
  - Need to verify customer identity
  - Looking up purchase history

parameters:
  query:
    type: string
    description: "Customer name, email, or ID to search"
    required: true
  limit:
    type: integer
    description: "Maximum results to return (default: 10)"
    default: 10
```

### 最佳实践

1. **命名规范**：动词+名词，语义明确
   - `search_contacts` ✅
   - `list_contacts` ❌（Agent 需要自行过滤）

2. **功能整合**：一个工具处理多个相关操作
   - `schedule_event` ✅（整合查询和创建）
   - `list_users` + `create_event` ❌（功能分散）

3. **返回高信号信息**
   ```python
   enum ResponseFormat {
       DETAILED = "detailed"  # 包含 IDs，用于后续调用
       CONCISE = "concise"    # 仅内容，节省 token
   }
   ```

---

## 快速参考卡片

```markdown
## Agent 提示词优化 Quick Reference

### ✅ DO
- 结构化分节（Identity/Instructions/Tools/Output）
- 3个代表性示例
- JSON/XML 标签分隔
- 工具整合（功能相关）
- 按需检索

### ❌ DON'T
- 过度硬编码逻辑
- 示例过载
- "假设共享上下文"
- 工具功能重叠
- 加载整个文档

### 🎯 Token 预算
| 组成 | 占比 |
|:-----|:-----|
| System Prompt | 固定 ~500 |
| User Prefs | 10% |
| Short-term Memory | 40% |
| Long-term Memory | 15% |
| Retrieval | 45% |
```

---

## 延伸阅读

- [Anthropic - Effective Context Engineering](https://www.anthropic.com/engineering/effective-context-engineering-for-ai-agents)
- [Anthropic - Writing Tools for AI Agents](https://www.anthropic.com/engineering/writing-tools-for-agents)
- [OpenAI - Prompt Engineering Guide](https://platform.openai.com/docs/guides/prompt-engineering)
- [OWASP Gen AI - LLM01: Prompt Injection](https://genai.owasp.org/llmrisk/llm01-prompt-injection/)
