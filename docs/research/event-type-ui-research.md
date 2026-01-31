# 前端事件类型展示调研报告

**调研日期**: 2026-02-01
**相关 Issue**: [#24](https://github.com/hrygo/divinesense/issues/24)
**状态**: 完成

---

## 1. 需求背景

### 1.1 痛点分析

当前 Geek/Evolution Mode (CCRunner) 用户体验问题：

| 痛点 | 描述 | 影响 |
|:-----|:-----|:-----|
| **黑盒执行** | 用户无法感知 Claude Code 的实时状态 | 焦虑、重复操作 |
| **无视觉区分** | 所有事件类型用相同聊天气泡展示 | 信息层级混乱 |
| **缺少反馈** | tool_use/tool_result 无专门 UI 组件 | 执行过程不透明 |

### 1.2 目标用户

- 使用 Geek Mode 的开发者（代码执行、文件操作）
- 使用 Evolution Mode 的管理员（系统自我进化）
- 需要实时反馈的重度用户

---

## 2. 技术调研

### 2.1 现有基础

**后端已完成** (PR #23)：
- `StreamEvent` 结构定义
- 事件类型：`thinking`, `tool_use`, `tool_result`, `answer`, `error`
- WebSocket 双向通信

**前端已有**：
- `ParrotEventType` 枚举定义
- `useParrotChat` 事件处理框架
- 聊天气泡基础组件

### 2.2 竞品分析

| 产品 | 实现 | 特点 |
|:-----|:-----|:-----|
| **Cursor AI** | 右侧面板实时展示 | 内联工具调用结果 |
| **GitHub Copilot Workspace** | 时间线式展示 | 多步骤执行可视化 |
| **v0.dev** | 内联展示工具调用 | 简洁明了 |

### 2.3 技术方案

**新增组件**：
```
components/AIChat/
├── EventBadge.tsx          # 事件类型徽章
├── ToolCallCard.tsx        # 工具调用卡片
├── EventStream.tsx         # 事件流容器
└── TerminalOutput.tsx      # 终端输出展示

hooks/
└── useEventStream.ts       # 事件流 Hook
```

**样式规范**：
| 事件类型 | 图标 | 颜色 | 动画 |
|:---------|:-----|:-----|:-----|
| `thinking` | Sparkles | 灰色 | 脉冲 |
| `tool_use` | Wrench | 蓝色 | 淡入 |
| `tool_result` (成功) | CheckCircle | 绿色 | 缩放 |
| `tool_result` (失败) | XCircle | 红色 | 抖动 |
| `error` | AlertCircle | 红色 | 闪烁 |

---

## 3. 设计方案

### 3.1 组件层次

```
EventStream (容器)
    └── MessageBubble (消息)
        ├── EventBadge (类型徽章)
        ├── ToolCallCard (工具调用)
        │   └── TerminalOutput (输出)
        └── Content (文本内容)
```

### 3.2 数据流

```
WebSocket Event
    ↓
useEventStream Hook
    ↓
EventBadge + ToolCallCard
    ↓
UI 更新
```

---

## 4. 风险与缓解

| 风险 | 影响 | 缓解措施 |
|:-----|:-----|:---------|
| WebSocket 断线 | 高 | 自动重连 + 状态恢复 |
| 大量输出性能 | 中 | 虚拟滚动 + 分页 |
| CLI 格式变更 | 低 | 后端兼容层 |

---

## 5. 验收标准

- [x] Issue 创建 (#24)
- [ ] `make check-all` 通过
- [ ] Geek Mode 实时展示工具调用
- [ ] 事件类型视觉区分
- [ ] Bash 输出语法高亮
- [ ] WebSocket 自动重连

---

## 6. 参考链接

- [Issue #24](https://github.com/hrygo/divinesense/issues/24)
- [Issue #15](https://github.com/hrygo/divinesense/issues/15) - CCRunner 异步架构
- [PR #23](https://github.com/hrygo/divinesense/pull/23) - 后端实现
- [架构规格](../specs/cc_runner_async_arch.md)
