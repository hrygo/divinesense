# MemoList 状态组件设计说明

> **设计哲学**：「禅意智识」
> - 呼吸感：与 logo-breathe-gentle 同步的韵律动画
> - 留白：东方美学的空灵意境
> - 意识流：思绪浮现的视觉隐喻
> - 渐进：信息温和呈现，不打断心流

---

## 组件概览

| 组件 | 文件 | 用途 |
|:-----|:-----|:-----|
| `LoadingSkeleton` | `MemoListStates.tsx` | 初始加载骨架屏 |
| `PaginationSkeleton` | `MemoListStates.tsx` | 分页加载骨架屏 |
| `EmptyState` | `MemoListStates.tsx` | 空状态 |
| `EndIndicator` | `MemoListStates.tsx` | 列表结束指示器 |
| `FixedEditor` | `FixedEditor.tsx` | 固定底部编辑器 |
| `QuickEditor` | `FixedEditor.tsx` | 快速编辑器（可选） |

---

## 设计规范

### 动画周期

```typescript
const BREATH_DURATION = 3000; // 与 logo-breathe-gentle 同步
```

所有呼吸动画使用相同的 3000ms 周期，确保整个应用的韵律统一。

### 间距系统

| 类 | 值 | 用途 |
|:-----|:-----|:-----|
| `gap-4` | 16px | 卡片间距 |
| `py-3 sm:py-4` | 12px/16px | 编辑器内边距 |
| `py-8 sm:py-10` | 32px/40px | 结束指示器上下边距 |
| `py-16 sm:py-24` | 64px/96px | 空状态上下边距 |

### 圆角系统

| 类 | 值 | 用途 |
|:-----|:-----|:-----|
| `rounded-xl` | 12px | 卡片、编辑器 |
| `rounded-2xl` | 16px | 搜索框、编辑器聚焦 |
| `rounded-full` | 50% | 装饰性元素 |

---

## 组件详解

### 1. LoadingSkeleton - 初始加载骨架屏

**设计要点**：
- 使用透明渐变 (`from-transparent via-muted/60 to-transparent`) 而非硬边 shimmer
- 每条线条独立脉动，模拟自然呼吸
- 保持与 MemoBlockV2 相同的结构

```tsx
<ZenSkeletonLine width="100%" delay={index * 100} />
<ZenSkeletonLine width="85%" delay={index * 100 + 100} />
```

**动画**：
- `animate-pulse` 自带，但每个线条有不同的 `animationDuration`
- 延迟递增，创造波纹效果

---

### 2. PaginationSkeleton - 分页加载骨架屏

**设计要点**：
- 只显示 2 个卡片（而非 4 个），暗示「正在加载更多」
- 使用更轻的视觉重量
- 底部有微妙的加载指示器

```tsx
<div className="flex items-center justify-center gap-2 py-4">
  <Loader2 className="h-4 w-4 animate-spin" style={{ animationDuration: "2s" }} />
  <span className="text-xs">正在加载...</span>
</div>
```

---

### 3. EmptyState - 空状态

**设计要点**：
- 呼吸光晕背景，与整体设计呼应
- 图标轻柔浮动 (`gentle-float` 动画)
- 根据场景显示不同文案

**场景类型**：

| 类型 | 图标 | 标题 | 副标题 |
|:-----|:-----|:-----|:-----|
| `all` | BookOpen | 开始你的记录之旅 | 每一个想法都值得被记住 |
| `search` | Sparkles | 没有找到相关笔记 | 试试其他关键词，或用智能搜索发现更多 |
| `filtered` | Feather | 没有符合条件的笔记 | 试试调整筛选条件 |

---

### 4. EndIndicator - 列表结束指示器

**设计要点**：
- 东方美学的「止」— 不是结束，而是停顿
- 使用三个小圆点，依次呼吸
- 两侧装饰线，营造平衡感

```tsx
<div className="flex items-center justify-center gap-3 py-8">
  <div className="h-px w-8 bg-gradient-to-r from-transparent to-border/50" />
  <div className="flex gap-2">
    <span className="h-1 w-1 rounded-full" style={{ animation: `dot-breathe 3s infinite` }} />
    <span className="h-1 w-1 rounded-full" style={{ animation: `dot-breathe 3s infinite 1s` }} />
    <span className="h-1 w-1 rounded-full" style={{ animation: `dot-breathe 3s infinite 2s` }} />
  </div>
  <div className="h-px w-8 bg-gradient-to-l from-transparent to-border/50" />
</div>
```

---

### 5. FixedEditor - 固定底部编辑器

**设计要点**：
- 聚焦时产生「意识场」光晕
- 顶部渐变边框
- 聚焦时微妙缩放 (`scale-[1.01]`)
- 底部快捷键提示（未聚焦时显示）

**意识场光晕**：
```tsx
<div
  className="absolute bottom-0 left-1/2 -translate-x-1/2 h-32 w-[120%] rounded-t-full bg-primary/3 blur-3xl"
  style={{ animation: `consciousness-field ${BREATH_DURATION}ms infinite` }}
/>
```

---

### 6. QuickEditor - 快速编辑器

**设计要点**：
- 紧凑布局，适合快速输入场景
- 自动调整高度（max 200px）
- 聚焦时显示光晕
- `⌘ Enter` 发送快捷键

---

## 动画定义

```css
/* 呼吸动画 - 与 logo 同步 */
@keyframes breathe {
  0%, 100% { opacity: 0.4; transform: scale(1); }
  50% { opacity: 0.8; transform: scale(1.08); }
}

/* 圆点呼吸 - 更轻柔 */
@keyframes dot-breathe {
  0%, 100% { opacity: 0.3; transform: scale(0.8); }
  50% { opacity: 0.7; transform: scale(1.2); }
}

/* 轻柔浮动 */
@keyframes gentle-float {
  0%, 100% { transform: translateY(0); }
  50% { transform: translateY(-6px); }
}

/* 意识场光晕 */
@keyframes consciousness-field {
  0%, 100% { opacity: 0.3; transform: translateX(-50%) scale(1); }
  50% { opacity: 0.6; transform: translateX(-50%) scale(1.05); }
}
```

---

## 使用示例

```tsx
import {
  EmptyState,
  EndIndicator,
  LoadingSkeleton,
  PaginationSkeleton,
} from "@/components/Memo/MemoListStates";

// 初始加载
{isLoading && <LoadingSkeleton count={4} />}

// 分页加载
{isFetchingNextPage && <PaginationSkeleton />}

// 空状态
{!isFetchingNextPage && memos.length === 0 && (
  <EmptyState type={getEmptyType()} />
)}

// 列表结束
{!isFetchingNextPage && !hasNextPage && memos.length > 0 && (
  <EndIndicator />
)}
```

---

## 设计原则

1. **呼吸感**：所有动画与 3000ms 呼吸周期同步
2. **留白**：充足的间距让内容呼吸
3. **温和**：所有过渡都是柔和的，无突兀变化
4. **渐进**：信息逐步呈现，不造成视觉冲击
5. **一致**：与 HeroSection、MemoBlockV2 保持相同的设计语言

---

## 文件结构

```
web/src/components/Memo/
├── MemoListStates.tsx    # 状态组件（骨架屏、空状态、结束指示器）
├── FixedEditor.tsx       # 固定底部编辑器 + QuickEditor
└── MemoList.tsx          # 使用上述状态组件
```

---

## i18n 键

| 键 | 值 |
|:-----|:-----|
| `memo.empty_title` | 开始你的记录之旅 |
| `memo.empty_subtitle` | 每一个想法都值得被记住 |
| `search.empty_title` | 没有找到相关笔记 |
| `search.empty_subtitle` | 试试其他关键词，或用智能搜索发现更多 |
| `filter.empty_title` | 没有符合条件的笔记 |
| `filter.empty_subtitle` | 试试调整筛选条件 |
