# SPEC-004: 前端交互与 UI/UX 优化

> **状态**: 待实现
> **优先级**: P1
> **依赖**: 后端 API 就绪
> **负责人**: 前端开发组

## 1. 概述

本规范定义了鹦鹉助手家族的前端实现细节，重点在于 **UI/UX 效果优化**，确保交互的流畅性、自然性和高级感。目标是打造 "沉浸式" 的 AI 协作体验。

## 2. 视觉设计规范

### 2.1 设计语言 (Unified Style)
- **色彩**: 每个 Agent 拥有专属主题色 (Tailwind CSS)，但在卡片结构和字体排版上保持一致。
    - 🦜 Memo: `blue-500` / `bg-blue-50`
    - ⏰ Schedule: `orange-500` / `bg-orange-50`
    - 🌟 Amazing: `purple-500` / `bg-purple-50`
    - 💡 Creative: `yellow-500` / `bg-yellow-50`
- **圆角**: 统一使用 `rounded-lg` 或 `rounded-xl`。
- **动效**: 使用 `transition-all duration-200` 处理 Hover 和切换状态。

### 2.2 核心组件

#### ParrotQuickActions (顶部快捷栏)
- **位置**: AI Chat 窗口顶部。
- **交互**:
    - **Hover**: 微微上浮 (`-translate-y-0.5`) + 阴影加深 (`shadow-md`)。
    - **Click**: 瞬间激活对应 Agent，输入框 Placeholder 变更。
    - **Active**: 选中项高亮，非选中项半透明。
- **优化**: 确保在该组件上使用 `AnimatePresence` (如果引入 Framer Motion) 或 CSS Transition 实现平滑切换。

#### ParrotSelector (@ 菜单)
- **触发**: 输入框键入 `@`。
- **样式**: 悬浮菜单 (Popover)，支持键盘 `↑` `↓` `Enter` 选择。
- **内容**: 列表展示 Agent 图标、名称、简短描述。

#### AI Response Cards (结果卡片)
- **MemoQueryResult**:
    - 展示笔记列表摘要。
    - 点击可跳转到笔记详情。
    - 支持 "引用" 样式高亮。
- **ScheduleQueryResult**:
    - 票据式布局 (Ticket Style)。
    - 清晰展示 时间、标题、冲突状态 (红色警告)。
- **AmazingQueryResult**:
    - 聚合视图。
    - 顶部 Tab 或 分区展示 (笔记区/日程区)。

### 2.3 交互体验优化 (UX)

#### 状态反馈 (Thinking State)
- **动画**: 
    - 使用 "波浪式" 或 "呼吸式" 的 Loading 动画，而不是传统的旋转圆圈。
    - 文本提示: "灰灰正在翻阅笔记...", "金刚正在查询日历..." (随机化文案增加趣味性)。
- **流式渲染**:
    - 确保 Markdown 渲染平滑，避免闪烁。
    - 自动滚动到底部 (Auto-scroll) 但允许用户手动阻断。

#### 错误处理
- **优雅降级**: 如果 Agent 调用失败，显示友好的 Toast 提示，并保留当前输入内容供用户重试。

## 3. 验收标准 (Acceptance Criteria)

### AC-004.1: Agent 切换体验
- [ ] **视觉反馈**: 点击快捷卡片时，切换无延迟，主题色随之渐变。
- [ ] **输入框**: Placeholder 立即更新为对应 Agent 的欢迎语 (e.g., "@Memo: 搜些什么？")。

### AC-004.2: 结果卡片渲染
- [ ] **Memo 卡片**: 正确渲染 Markdown 摘要，链接可点击。
- [ ] **Schedule 卡片**: 时间格式化为本地时间，冲突日程有明显红字标识。
- [ ] **Amazing 卡片**: 在同一气泡内清晰分隔笔记和日程结果，布局不拥挤。

### AC-004.3: 动效与性能
- [ ] **FPS**: 动画过程中帧率稳定在 60fps。
- [ ] **打字机效果**: AI 回复时文本逐字出现，光标闪烁自然。

## 4. 实施步骤

1.  检查现有 `ParrotQuickActions.tsx`，添加 Hover 动效和主题色逻辑。
2.  优化 `ParrotSelector.tsx` 的键盘交互和弹出动画。
3.  开发 `MemoQueryResult.tsx` 组件。
4.  开发 `AmazingQueryResult.tsx` 组件。
5.  在 `MessageList` 中集成各类型卡片的渲染逻辑。
6.  全局通过 CSS/Tailwind 调整字体和间距，统一视觉风格。
