# SPEC-008: Chat UI/UX Optimization (Parrot Agents)

> **Status**: Draft
> **Author**: Antigravity (Product Manager & UI/UX Designer)
> **Date**: 2026-01-25

## 1. 背景与问题分析 (Background & Problem)

当前 `AIChat` 界面存在以下体验问题：
1.  **入口冗余**：用户可以通过左侧“历史记录”进入对话，也可以通过主界面的“滑动鹦鹉卡片”新建对话。两者功能重叠。
2.  **空间浪费**：主界面的滑动卡片占据了大量核心区域，对于高频用户而言，这些卡片并未提供持续价值（每次都要看一遍介绍）。
3.  **Agent 访问路径长**：用户想要快速找特定 Agent（如“日程助手”），需要在历史记录里翻找或重新从卡片创建。

## 2. 核心目标 (Core Goals)

1.  **固定入口**：将 5 个鹦鹉 Agent (Default, Memo, Schedule, Amazing, Creative) 作为**一级常驻入口**固定在左侧边栏。
2.  **净化主屏**：移除主界面的滑动卡片，提供更纯净的对话体验或“空状态”。
3.  **高效交互**：点击左侧固定头像，立即进入该 Agent 的“最新”或“默认”会话，无需确认。

## 3. 设计方案 (Design Proposal)

### 3.1 左侧边栏重构 (Sidebar Redesign)

左侧边栏将分为两个区域（Vertical Split）：

*   **Zone A: 常驻智能体 (Pinned Agents)**
    *   位于列表顶部，固定不滚动。
    *   展示 5 个 Agent 的头像（高亮显示当前选中）。
    *   支持悬停显示 Tooltip（Agent 名称）。
    *   **交互逻辑**：
        *   点击 Agent 头像 -> 检查是否有该 Agent 的历史会话。
        *   有 -> 跳转至最近一次会话。
        *   无 -> 自动创建新会话并进入。

*   **Zone B: 最近会话 (Recent Chats)**
    *   位于 Zone A 下方，支持滚动。
    *   展示所有历史会话列表（按时间倒序）。
    *   保持现有 `ConversationHistoryPanel` 的核心功能（删除、重命名等）。

### 3.2 主界面优化 (Main Area Cleanup)

*   **移除 `HubView`**：彻底删除包含滑动卡片的 `HubView` 组件。
*   **默认状态 (Empty State)**：
    *   当没有任何会话选中且作为初始进入状态时，展示一个极简的欢迎页。
    *   文案示例：“选择左侧助手或直接开始对话”。
    *   背景：纯净或带水印 Logo，无干扰元素。

### 3.3 视觉模拟 (Visual Mockup)

![Chat UI Mockup](/Users/huangzhonghui/.gemini/antigravity/brain/29d1d8e5-0bd6-42f0-ae5b-b01dabf2ca8f/chat_sidebar_mockup_1769308681125.png)

*(注：实际开发中将严格遵循现有 Design System 的色彩和组件风格)*

## 4. 交互细节 (Interaction Details)

| 动作                         | 预期结果                                                                       |
| :--------------------------- | :----------------------------------------------------------------------------- |
| **点击左侧「日程助手」头像** | 立即打开与日程助手的对话窗口。如果之前聊过，加载历史；如果是新的，显示欢迎语。 |
| **在对话框输入 `@`**         | (保持现有功能) 弹出 Agent 选择列表，允许临时切换或多 Agent 协作。              |
| **点击「清除上下文」**       | 保持现有功能，在当前 Agent 窗口内清除记忆。                                    |

## 5. 预期收益 (Expected Outcome)

*   **屏幕利用率提升 30%**：移除冗余卡片后，用户更聚焦于内容及侧边导航。
*   **操作路径缩短 50%**：从进页面到开始与特定 Agent 对话，由“寻找卡片->点击”变为“侧边栏直达”。
