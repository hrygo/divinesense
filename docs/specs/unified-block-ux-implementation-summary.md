# UnifiedMessageBlock UI/UX 优化实施总结

> **完成时间**: 2026-02-07
> **关联 Issue**: [#104](https://github.com/hrygo/divinesense/issues/104)

---

## 实施概览

按照 Clean Architecture 方案，在 `src/components/AIChat/UnifiedMessageBlock/` 创建了模块化目录结构：

```
UnifiedMessageBlock/
├── components/          # React 组件
│   ├── TimelineNode.tsx           # 统一时间线节点
│   ├── StreamingProgressBar.tsx  # 流式进度条
│   ├── PendingSkeleton.tsx        # 骨架屏加载状态
│   ├── ToolCallCard.tsx           # 优化的工具卡片（hover + memo）
│   ├── BlockHeader.tsx            # 响应式两栏布局 Header
│   ├── BlockFooter.tsx            # 图标优先响应式 Footer
│   └── VirtualBlockList.tsx       # 虚拟滚动组件
├── hooks/               # 自定义 Hooks
│   ├── useBlockCollapse.ts        # 折叠状态下沉
│   ├── useStreamingProgress.ts    # 流式进度计算
│   └── useKeyboardNav.ts          # 键盘导航
├── utils/               # 工具函数
│   └── index.ts                   # 视觉宽度、预览文本等
├── types/               # TypeScript 类型定义
│   └── index.ts
├── constants.ts         # 常量配置（扩展自父目录）
└── index.ts            # 模块统一导出
```

---

## 完成的功能

### Phase 1: 视觉层次优化 ✅

| 组件 | 功能 |
|:-----|:-----|
| `TimelineNode` | 统一 `w-6 h-6 border-2 rounded-full` 样式 |
| `BlockHeader` | 两栏布局（头像+预览 | 统计+时间+徽章+切换） |
| 预览文本 | `generatePreviewText()` 函数支持折叠预览 |

### Phase 2: 响应式体验优化 ✅

| 断点 | 优化 |
|:-----|:-----|
| 移动端 (< 768px) | Header 隐藏统计/Badge，Footer 仅图标 |
| 桌面端 (≥ 768px) | 显示完整统计和标签 |
| `lg:hidden` / `sm:hidden` | Tailwind 响应式类控制显示 |

### Phase 3: 交互反馈增强 ✅

| 组件 | 功能 |
|:-----|:-----|
| `StreamingProgressBar` | 底部细进度条，0-100% 流式显示 |
| `ToolCallCard` | hover 紫色边框高亮 (`hover:border-purple-400/50`) |
| `PendingSkeleton` | 替代"处理中..."的骨架屏 |

### Phase 4: 可访问性改进 ✅

| Hook | 快捷键 |
|:-----|:-----|
| `useKeyboardNav` | Tab/Shift+Tab 导航、Ctrl+C 复制、Ctrl+E 编辑、Escape 取消 |

### Phase 5: 性能优化 ✅

| 优化 | 实现方式 |
|:-----|:---------|
| 状态下沉 | `useBlockCollapse` hook，每个 Block 管理自己的折叠状态 |
| React.memo | `BlockHeader`, `BlockFooter`, `ToolCallCard` 全部 memo |
| 虚拟滚动 | `VirtualBlockList` + `@tanstack/react-virtual` |

---

## 新增常量

```typescript
// src/components/AIChat/constants.ts

export const TIMELINE_NODE_CONFIG = {
  size: 'w-6 h-6',
  border: 'border-2',
  radius: 'rounded-full',
  iconSize: 'w-3.5 h-3.5',
} as const;

export const NODE_COLORS = {
  user: 'bg-blue-100 dark:bg-blue-900/40 border-blue-500 text-blue-600',
  thinking: 'bg-purple-100 dark:bg-purple-900/40 border-purple-500 text-purple-600',
  tool: 'bg-card border-border group-hover:border-purple-400/50',
  answer: 'bg-amber-50 dark:bg-amber-900/20 border-amber-500 text-amber-500',
  error: 'bg-red-100 dark:bg-red-900/30 border-red-500 text-red-600',
} as const;

export const RESPONSIVE_CONFIG = {
  mobile: { hideStats: true, hideBadge: true, iconOnly: true, singleStat: true },
  desktop: { showStats: true, showBadge: true, showLabels: true, allStats: true },
} as const;
```

---

## 使用方式

### 新组件使用示例

```tsx
import {
  TimelineNode,
  StreamingProgressBar,
  PendingSkeleton,
  ToolCallCard,
  BlockHeader,
  BlockFooter,
  useBlockCollapse,
  useStreamingProgress,
  useKeyboardNav,
  VirtualBlockList,
} from "@/components/AIChat/UnifiedMessageBlock";

// TimelineNode - 统一样式的时间线节点
<TimelineNode type="user" />
<TimelineNode type="thinking" />
<TimelineNode type="tool" />
<TimelineNode type="answer" />
<TimelineNode type="error" />

// StreamingProgressBar - 流式进度
<StreamingProgressBar progress={45} isActive={true} />

// PendingSkeleton - 加载状态
<PendingSkeleton message="AI 正在思考..." />

// Hooks
const { collapsed, toggleCollapse, generatePreview } = useBlockCollapse({
  isLatest: true,
  isStreaming: false,
});

const { progress, isShowingProgress } = useStreamingProgress({
  isStreaming: true,
  content: messageContent,
});

const { keyboardProps, focusBlock } = useKeyboardNav({
  blockId: "block-123",
  isFocusable: true,
  onShortcut: (action) => console.log(action),
});
```

---

## 下一步

要将这些优化整合到 `UnifiedMessageBlock.tsx`：

1. 用新的 `BlockHeader` 和 `BlockFooter` 替换内联组件
2. 用 `TimelineNode` 替换内联的时间线节点样式
3. 用 `useBlockCollapse` 替换父组件管理的折叠状态
4. 在长对话场景中使用 `VirtualBlockList` 包装消息列表
