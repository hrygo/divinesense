# AIChat @ 符号选择专家 Agent 功能调研

> 调研时间: 2026-02-17
> Issue: #259
> 状态: 已确认，待实施

---

## 需求概述

AIChat 输入框支持 `@` 符号触发专家 Agent 选择，允许用户直接指定具体 Agent 处理任务。

### 核心约束

1. **选择范围**：仅显示可提及的专家代理（排除 AUTO/AMAZING/GEEK/EVOLUTION/GENERAL）
2. **位置限制**：仅支持消息头部或尾部插入 `@灰灰` 标记
3. **多 Agent**：支持同时指定多个 Agent
4. **无快捷键**：不实现 `@h` 快速选择等功能

---

## 技术调研

### 数据来源

从 `GET /api/v1/ai/parrots` 动态加载 Agent 列表。

**响应结构**:
```typescript
interface ParrotInfo {
  agent_type: AgentType;
  name: string;
  self_cognition: {
    name: string;
    emoji: string;
    title: string;
    capabilities: string[];
    working_style: string;
  };
}
```

### 配置文件

Agent 配置位于 `config/parrots/`:
- `memo.yaml` - 灰灰（笔记搜索）
- `schedule.yaml` - 时巧（日程管理）
- `general.yaml` - 通用代理

### 过滤逻辑

```typescript
function isMentionable(parrot: ParrotInfo): boolean {
  const excludedTypes = ['DEFAULT', 'AMAZING', 'GEEK', 'EVOLUTION'];
  return !excludedTypes.includes(parrot.agent_type);
}
```

---

## 实现方案

### 组件结构

```
ChatInput
  └── AgentMentionPopover
        ├── useParrotsList() hook
        ├── Popover (Radix)
        └── AgentList
```

### 文件变更

| 文件 | 操作 | 说明 |
|:-----|:-----|:-----|
| `web/src/hooks/useParrotsList.ts` | 新增 | 调用 ListParrots API |
| `web/src/components/AIChat/ChatInput.tsx` | 修改 | 集成 @ 检测 + Popover |
| `web/src/components/AIChat/AgentMentionPopover.tsx` | 新增 | Agent 选择弹窗组件 |
| `web/src/utils/agentMention.ts` | 新增 | 解析/位置判断工具函数 |
| `web/src/locales/zh-Hans.json` | 修改 | 新增翻译 |
| `web/src/locales/en.json` | 修改 | 新增翻译 |

### 位置判断逻辑

```typescript
function isValidMentionPosition(value: string, cursorPos: number): boolean {
  const beforeCursor = value.slice(0, cursorPos).trim();
  const afterCursor = value.slice(cursorPos).trim();
  return beforeCursor === '' || afterCursor === '';
}
```

### 多 Agent 解析

```typescript
function parseMentionedAgents(message: string): {
  agents: string[];
  cleanMessage: string;
} {
  const agentPattern = /@(灰灰|时巧|memo|schedule)\s*/gi;
  const agents: string[] = [];

  let match;
  while ((match = agentPattern.exec(message)) !== null) {
    const name = match[1].toLowerCase();
    if (name === '灰灰' || name === 'memo') {
      if (!agents.includes('memo')) agents.push('memo');
    } else if (name === '时巧' || name === 'schedule') {
      if (!agents.includes('schedule')) agents.push('schedule');
    }
  }

  const cleanMessage = message.replace(agentPattern, '').trim();
  return { agents, cleanMessage };
}
```

---

## 工作量评估

| 模块 | 工作量 |
|:-----|:-------|
| AgentMention 组件 | 2h |
| ChatInput 集成 | 2h |
| 发送逻辑修改 | 1h |
| i18n | 0.5h |
| **总计** | **~0.7 人周** |

---

## 风险与缓解

| 风险 | 影响 | 措施 |
|:-----|:-----|:-----|
| 光标定位 | 中 | 使用 `selectionStart`/`selectionEnd` API |
| 中文输入法 | 低 | 监听 `compositionend` 事件 |
| 多 Agent 发送 | 中 | MVP 阶段仅使用第一个 Agent |

---

## 验收标准

- [ ] `make check-all` 通过
- [ ] 输入框 @ 触发 Agent 选择弹窗
- [ ] 支持头部/尾部位置插入
- [ ] 支持中文名 (`@灰灰`) 和英文名 (`@memo`) 匹配
- [ ] 已更新 i18n 翻译
- [ ] 已测试键盘导航（上下键 + Enter）

---

## 参考资料

- [Issue #259](https://github.com/hrygo/divinesense/issues/259)
- [ListParrots API](/proto/api/v1/ai_service.proto)
- [Parrot 配置](/config/parrots/)
