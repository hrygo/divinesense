# Claude Code 并行代理能力调研总结

> **调研日期**: 2026-02-09
> **目的**: 了解 Claude Code 的 Agents Team 能力，优化 DivineSense 项目 CLAUDE.md

---

## 核心发现

### Claude Code 并行代理能力

Claude Code (Opus 4.6) 支持真正的多代理并行执行，而非简单的串行任务调度。

**关键特性**：
- **并行执行**: 多个代理可同时运行，独立处理任务
- **点对点通信**: 代理间可以直接通信协调
- **共享上下文**: 所有代理访问相同的项目上下文
- **结果聚合**: 主代理负责汇总各代理结果

### 启动并行代理的条件

满足以下任一条件时，应考虑启动并行代理：

1. **2+ 个无依赖关系的子任务** - 最核心条件
2. **子任务涉及不同代码区域** - 前端/后端/数据库独立
3. **子任务需要不同专业能力** - 分析/搜索/测试/代码生成
4. **任务总耗时预计 > 10 分钟** - 有显著加速价值

### 典型应用场景

#### 场景 1: 代码重构分析
```
任务：分析 universal parrot 重构影响
├─ Agent A: 分析 ai/agent/universal/ 依赖关系
├─ Agent B: 检查 server/router/api/v1/ai/ 调用点
└─ Agent C: 搜索配置文件 config/parrots/ 引用

加速效果：串行 ~8 分钟 → 并行 ~3 分钟
```

#### 场景 2: PR 全面审查
```
任务：全面审查变更
├─ Agent A: pr-test-analyzer（测试覆盖）
├─ Agent B: code-simplifier（代码简化）
├─ Agent C: comment-analyzer（注释质量）
└─ Agent D: silent-failure-hunter（错误处理）

加速效果：串行 ~15 分钟 → 并行 ~5 分钟
```

#### 场景 3: 跨模块功能开发
```
任务：添加新的 AI 工具
├─ Agent A: 设计工具接口（proto/）
├─ Agent B: 实现后端逻辑（ai/agent/tools/）
└─ Agent C: 更新前端 UI（web/src/components/）

加速效果：串行 ~20 分钟 → 并行 ~8 分钟
```

## 技术实现

### 模式 1: 单次消息多并行调用
```text
用户: "同时完成以下独立任务：
1. 分析前端 AIBlock 组件架构
2. 检查后端 ai/agent/ 目录结构
3. 搜索数据库迁移文件模式"
```
Claude 将在单个响应中并行调用多个 Task/Explore 代理。

### 模式 2: 显式指定并行执行
```text
用户: "并行执行以下任务：
- 运行 pr-test-analyzer 检查测试覆盖
- 运行 comment-analyzer 检查注释质量
- 运行 code-explorer 分析架构"
```

## 最佳实践

| 实践 | 说明 |
|:-----|:-----|
| **明确边界** | 每个代理的任务边界清晰，无重叠 |
| **独立状态** | 避免共享可变状态，使用只读输入 |
| **结果聚合** | 指定主代理负责汇总各代理结果 |
| **超时控制** | 每个代理设置合理超时（默认 5-10 分钟） |
| **错误隔离** | 单个代理失败不影响其他代理执行 |

## DivineSense 项目应用

### 已集成的并行代理技能

- `superpowers:dispatching-parallel-agents` - 并行代理调度框架
- `pr-review-toolkit:review-pr` - 多代理并行 PR 审查
- `feature-dev:code-explorer` - 架构分析代理
- `pr-review-toolkit:code-simplifier` - 代码简化代理

### 双层代理架构

DivineSense 项目有两层代理架构：

| 维度 | DivineSense (应用层) | Claude Code (开发层) |
|:-----|:---------------------|:---------------------|
| **用途** | 产品功能（AI 助手） | 开发效率（代码协作） |
| **通信** | 工具调用 + 共享内存 | 上下文共享 + 消息传递 |
| **并发** | 单 LLM 串行工具调用 | 真正多进程并行 |
| **状态** | 数据库持久化 | 会话级别 |

**关键洞察**：DivineSense 的五位鹦鹉通过**路由决策**实现专业分工，Claude Code 通过**任务分解**实现并行加速。两者互补，共同提升项目开发体验。

## 参考资源

- [Claude Code 并行代理指南](https://www.marc0.dev/en/blog/claude-code-agent-teams-multiple-ai-agents-working-in-parallel-setup-guide-1770317684454)
- [Anthropic 多代理研究系统](https://www.anthropic.com/engineering/multi-agent-research-system)
- [并行编码代理生活方式](https://simonwillison.net/2025/Oct/5/parallel-coding-agents/)
- [Building agents with the Claude Agent SDK](https://claude.com/blog/building-agents-with-the-claude-agent-sdk)
- [How to Use Claude Code Subagents to Parallelize Development](https://zachwills.net/how-to-use-claude-code-subagents-to-parallelize-development/)
