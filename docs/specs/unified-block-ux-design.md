# UnifiedMessageBlock UI/UX 优化设计

> **Issue**: [#104](https://github.com/hrygo/divinesense/issues/104)
> **状态**: Design
> **创建时间**: 2026-02-07
> **目标**: 优化 AI 聊天界面的视觉层次、响应式体验和交互反馈

---

## 1. 现状分析

### 1.1 当前架构

```
UnifiedMessageBlock
├── BlockHeader (用户消息 + 时间戳 + 统计 + Badge + 分支 + 折叠)
├── BlockBody
│   ├── TimelineContainer
│   │   ├── UserInputsSection
│   │   ├── ThinkingSection (可折叠)
│   │   ├── ToolCallsSection
│   │   └── AnswerSection
│   └── ErrorSection
└── BlockFooter (折叠 + 复制 + 编辑 + 删除 + 重新生成 + 遗忘)
```

### 1.2 存在问题

| 问题类别 | 具体表现 | 影响 |
|:---------|:---------|:-----|
| **视觉密度** | Header 信息过载、Timeline 节点样式不统一 | 信息层次混乱 |
| **响应式** | 移动端按钮密集、统计信息不可见 | 移动端体验差 |
| **交互反馈** | Streaming 动画不明显、工具卡片无 hover 状态 | 状态感知弱 |
| **性能** | 长对话（50+ blocks）滚动卡顿 | 性能瓶颈 |

---

## 2. 设计决策

### 2.1 Phase 1: 视觉层次优化

| 设计点 | 决策 | 实现细节 |
|:-------|:-----|:---------|
| **Timeline 节点** | 完全统一 | `w-6 h-6 rounded-full border-2`，仅颜色/图标区分 |
| **Header 布局** | 两栏布局 | 左（头像+消息+统计）- 右（时间+操作） |
| **折叠摘要** | 预览文本 | 显示 AI 回复前两行（~100 字符） |
| **错误样式** | 保持现状 | 红色边框方案已足够清晰 |

#### Timeline 节点规范

```typescript
// components/AIChat/TimelineNode.tsx
export const TIMELINE_NODE_CONFIG = {
  size: 'w-6 h-6',           // 统一尺寸
  border: 'border-2',        // 统一边框宽度
  radius: 'rounded-full',    // 统一圆角
  iconSize: 'w-3.5 h-3.5',   // 统一图标尺寸
} as const;

export const NODE_COLORS = {
  user: 'bg-blue-100 dark:bg-blue-900/40 border-blue-500 text-blue-600',
  thinking: 'bg-purple-100 dark:bg-purple-900/40 border-purple-500 text-purple-600',
  tool: 'bg-card border-border group-hover:border-purple-400/50',
  answer: 'bg-amber-50 dark:bg-amber-900/20 border-amber-500 text-amber-500',
  error: 'bg-red-100 dark:bg-red-900/30 border-red-500 text-red-600',
} as const;
```

#### Header 两栏布局

```tsx
// < 768px (移动端)
// ┌─────────────────────────────────────────────┐
// │ [头像] 消息预览...                    [▼]    │
// └─────────────────────────────────────────────┘

// ≥ 768px (桌面端)
// ┌──────────────────────────────────────────────────────────────┐
// │ [头像] 消息预览...    ⚡1.2k $0.0023    [10:30] [GEEK] [▼]    │
// └──────────────────────────────────────────────────────────────┘
```

### 2.2 Phase 2: 响应式体验优化

| 设计点 | 决策 | 断点 |
|:-------|:-----|:-----|
| **移动端 Header** | 最小化 | `<768px` 隐藏统计/Badge |
| **Footer 按钮** | 图标优先 | 移除文字，保留图标+tooltip |
| **统计信息** | 单指标 | 移动端仅显示关键指标（成本/耗时） |

#### 响应式断点

```typescript
export const RESPONSIVE_CONFIG = {
  breakpoints: {
    mobile: 'max-width: 767px',
    tablet: 'min-width: 768px',
    desktop: 'min-width: 1024px',
  },
  header: {
    mobile: { hideStats: true, hideBadge: true },
    desktop: { showStats: true, showBadge: true },
  },
  footer: {
    mobile: { iconOnly: true },
    desktop: { showLabels: true },
  },
} as const;
```

### 2.3 Phase 3: 交互反馈增强

| 设计点 | 决策 | 实现细节 |
|:-------|:-----|:---------|
| **Streaming 动画** | 进度条 | Block 底部添加细进度条 |
| **工具卡片 Hover** | 边框高亮 | 紫色到深紫色渐变 |
| **Pending 状态** | 骨架屏 | 替代「处理中...」文本 |

#### Streaming 进度条

```tsx
// components/AIChat/StreamingProgress.tsx
<div className="absolute bottom-0 left-0 h-1 bg-primary/20">
  <div
    className="h-full bg-primary transition-all duration-300 ease-out"
    style={{ width: `${progress}%` }}
  />
</div>
```

#### 工具卡片 Hover 效果

```css
/* 工具卡片 hover 边框渐变 */
.tool-card:hover {
  border-color: rgb(168 85 247); /* purple-500 */
  box-shadow: 0 0 0 1px rgb(168 85 247 / 0.1);
  transition: all 0.2s ease;
}
```

### 2.4 Phase 4: 可访问性改进

| 设计点 | 决策 | 备注 |
|:-------|:-----|:-----|
| **键盘导航** | Tab 焦点链 + 快捷键 | 支持 Tab 导航和 Ctrl+C/E 等快捷键 |
| **ARIA 标签** | ~跳过~ | 非目标用户群体 |
| **屏幕阅读器** | ~跳过~ | 追求 ROI |

#### 键盘快捷键

| 快捷键 | 功能 |
|:-------|:-----|
| `Tab` / `Shift+Tab` | 在 Block 间移动焦点 |
| `Ctrl/Cmd + C` | 复制当前 Block 内容 |
| `Ctrl/Cmd + E` | 编辑当前 Block |
| `Ctrl/Cmd + Enter` | 发送消息 |
| `Escape` | 关闭对话框/取消操作 |

### 2.5 Phase 5: 性能优化

| 设计点 | 决策 | 实现细节 |
|:-------|:-----|:---------|
| **状态管理** | 状态下沉 | collapse 状态移到 UnifiedMessageBlock 内部 |
| **React.memo** | 全面优化 | Header + Footer + ToolCallCard |
| **长对话渲染** | 虚拟滚动 | IntersectionObserver 仅渲染视口内 Blocks |

#### 状态下沉架构

```typescript
// Before: ChatMessages 管理所有 Block 状态
const [blockStates, setBlockStates] = useState<Record<string, BlockState>>({});

// After: 每个 Block 管理自己的折叠状态
function UnifiedMessageBlock({ isLatest, ... }) {
  const [collapsed, setCollapsed] = useState(() => !isLatest);
  // ...
}
```

#### 虚拟滚动实现

```typescript
// components/AIChat/VirtualBlockList.tsx
const observer = new IntersectionObserver((entries) => {
  entries.forEach(entry => {
    const blockId = entry.target.dataset.blockId;
    setShouldRender(blockId, entry.isIntersecting);
  });
}, { rootMargin: '200px' }); // 提前 200px 加载
```

---

## 3. 实施计划

### 3.1 任务分解

| Phase | 任务 | 优先级 | 预估时间 |
|:------|:-----|:-------|:---------|
| **1** | 创建 TimelineNode 组件 | P0 | 1h |
| **1** | 重构 Header 为两栏布局 | P0 | 2h |
| **1** | 添加折叠预览文本 | P0 | 1h |
| **2** | 实现响应式断点配置 | P0 | 1h |
| **2** | Footer 按钮图标优先 | P1 | 1h |
| **2** | 移动端统计单指标展示 | P1 | 1h |
| **3** | Streaming 进度条组件 | P1 | 2h |
| **3** | 工具卡片 hover 边框高亮 | P1 | 1h |
| **3** | Pending 骨架屏组件 | P1 | 2h |
| **4** | 键盘导航 Tab 焦点链 | P2 | 2h |
| **4** | 快捷键支持 | P2 | 2h |
| **5** | Block collapse 状态下沉 | P0 | 2h |
| **5** | React.memo 优化 Header/Footer | P0 | 1h |
| **5** | React.memo 优化 ToolCallCard | P1 | 1h |
| **5** | 虚拟滚动实现 | P1 | 3h |

**总计**: ~23 小时

### 3.2 依赖关系

```
Phase 5 (状态下沉) ──必须优先──┐
                              ├──→ Phase 1 (视觉优化)
Phase 2 (响应式) ──────────────┘
                              ├──→ Phase 3 (交互反馈)
                              └──→ Phase 4 (可访问性)
```

---

## 4. 验收标准

- [ ] 所有 Timeline 节点使用统一样式配置
- [ ] 移动端 Header 信息密度降低 30%
- [ ] 流式输出进度条可见且流畅
- [ ] 支持 Tab 键在 Block 间导航
- [ ] 长对话（50+ blocks）滚动流畅 60fps
- [ ] 通过 Lighthouse 性能测试（>90 分）

---

## 5. 参考资料

- [Notion AI Blocks](https://notion.so) - 折叠块设计参考
- [Cursor IDE](https://cursor.sh) - 代码块工具调用展示
- [Linear](https://linear.app) - 状态徽章设计
- [Tailwind CSS v4](https://tailwindcss.com) - 样式系统
