# ZenToolbar 设计说明

> **设计哲学**: 「禅意智识」
> **呼吸周期**: 3000ms（与 logo-breathe-gentle 同步）

---

## 概述

ZenToolbar 是 DivineSense MemoEditor 的全新工具栏设计，遵循「禅意智识」设计哲学，提供**不折叠**的工具栏体验，同时完美兼顾 PC 端与移动端。

---

## 设计原则

### 1. 呼吸感 (Breathing)
所有动画与 logo 的 `logo-breathe-gentle` 动效同步，使用 3000ms 呼吸周期：
- 悬停时的光晕脉冲
- 激活状态的生命感
- 过渡动画的柔和节奏

### 2. 留白 (Whitespace)
- 4px 基础间距单位系统
- 充足的按钮内边距
- 舒适的视觉呼吸空间

### 3. 不折叠 (No Collapse)
- 所有功能直观呈现
- 无需点击展开菜单
- 提高操作效率

### 4. 微妙 (Subtle)
- 柔和的状态过渡
- 轻柔的视觉反馈
- 不喧宾夺主

---

## 布局策略

### PC 端 (≥ 640px)
```
┌──────────────────────────────────────────────────────────────────────────────┐
│  [📎] [🔗] [📍] [⛶]          [🪄]          [👁️]  [取消]  [保存 →]          │
│  插入功能                    AI 辅助         可见性  操作                      │
└──────────────────────────────────────────────────────────────────────────────┘
```

**三段式布局**：
- **左侧**：插入功能（文件、链接、位置、专注模式）
- **中间**：AI 辅助（标签建议）
- **右侧**：可见性 + 操作按钮

### 移动端 (< 640px)
```
┌─────────────────────────────────────────────┐
│  [📎] [🔗] [📍] [⛶]              [🪄]      │
│  插入功能                        AI 辅助    │
├─────────────────────────────────────────────┤
│  [👁️]                      [取消] [保存 →] │
│  可见性                    操作             │
└─────────────────────────────────────────────┘
```

**两行式布局**：
- **第一行**：插入功能 + AI
- **第二行**：可见性 + 操作

---

## 组件规范

### 按钮尺寸

| 断点 | 尺寸 | 圆角 | 图标 |
|:-----|:-----|:-----|:-----|
| PC (sm+) | `h-9 w-9` (36px) | `rounded-xl` (12px) | 16px |
| 移动端 | `h-10 w-10` (40px) | `rounded-xl` (12px) | 16px |

**设计理由**：移动端按钮稍大，便于拇指操作。

### 间距系统

| 用途 | 类名 | 值 |
|:-----|:-----|:-----|
| 按钮之间 | `gap-1` | 4px |
| 功能组之间 | `gap-3` | 12px |
| 两行之间 | `mb-2` | 8px |

### 颜色状态

| 状态 | 背景色 | 文字色 | 光晕 |
|:-----|:-------|:-------|:-----|
| 默认 | - | `text-muted-foreground` | - |
| 悬停 | `hover:bg-muted/50` | `hover:text-foreground` | 呼吸脉冲 |
| 激活 | `bg-primary/10` | `text-primary` | 内阴影 |
| 禁用 | - | `opacity-40` | - |

---

## 动画规范

### 呼吸动画 (Breathe)

```css
@keyframes zen-breathe {
  0%, 100% {
    opacity: 0.3;
    transform: scale(1);
  }
  50% {
    opacity: 0.6;
    transform: scale(1.05);
  }
}
```

**使用场景**：
- 悬停时的背景光晕
- 面板展开的强调效果

### 进入动画 (Slide In)

```css
@keyframes zen-slide-in {
  from {
    opacity: 0;
    transform: translateY(8px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}
```

**使用场景**：
- 展开面板的进入
- 下拉菜单的显示

### 过渡时长

| 场景 | 时长 | 缓动函数 |
|:-----|:-----|:---------|
| 悬停效果 | 300ms | `ease-out` |
| 面板展开 | 200ms | `ease-out` |
| 状态切换 | 200ms | `ease-out` |

---

## 功能按钮

### 1. 文件上传 (FileImage)
- **图标**: `FileImage` (lucide-react)
- **快捷键**: 无
- **状态**: 上传中显示加载动画

### 2. 关联笔记 (Link2)
- **图标**: `Link2` (lucide-react)
- **快捷键**: 无
- **交互**: 点击打开笔记搜索对话框

### 3. 添加位置 (MapPin)
- **图标**: `MapPin` (lucide-react)
- **状态**: 已添加位置时保持激活状态
- **交互**: 点击打开位置选择对话框

### 4. 专注模式 (Maximize2)
- **图标**: `Maximize2` (lucide-react)
- **快捷键**: `⌘⇧F`
- **功能**: 切换全屏专注模式

### 5. AI 标签 (Wand2)
- **图标**: `Wand2` (lucide-react)
- **功能**: 点击展开标签建议面板
- **交互**: 点击标签直接插入到内容末尾

### 6. 可见性选择器 (VisibilitySelector)
- **设计**: 圆形切换器 + 展开面板
- **交互**:
  - 点击按钮：循环切换可见性
  - 悬停展开：显示完整选项面板
- **选项**:
  - 私有 (PRIVATE) - 锁图标
  - 工作区 (PROTECTED) - 工作区图标
  - 公开 (PUBLIC) - 地球图标

### 7. 保存按钮 (Send)
- **图标**: `Send` (lucide-react)
- **状态**:
  - 保存中: 显示加载动画
  - 禁用: 内容无效时
  - 可用: 内容有效时

---

## 响应式断点

```tsx
// 桌面端检测
const isMobile = window.innerWidth < 640; // Tailwind sm 断点

// 或者使用 CSS
.hidden sm:block  {/* 桌面端显示 */}
.sm:hidden       {/* 移动端显示 */}
```

---

## 可访问性

### 键盘导航
- `Tab`: 在按钮间导航
- `Enter/Space`: 激活按钮
- `Escape`: 关闭展开的面板
- `⌘⇧F`: 切换专注模式

### ARIA 属性
- `aria-label`: 按钮描述
- `aria-disabled`: 禁用状态
- `aria-haspopup`: 有子菜单的按钮

### 触摸目标
- 移动端最小点击区域: 40×40px (h-10 w-10)
- 符合 WCAG 2.1 AAA 标准

---

## 与现有组件集成

### 替换方案

```tsx
// 原来的 EditorToolbar
import { EditorToolbar } from "@/components/MemoEditor";

// 新的 ZenToolbar
import { ZenToolbar } from "@/components/MemoEditor/Toolbar";

// 接口完全兼容
<ZenToolbar onSave={handleSave} onCancel={handleCancel} memoName={memoName} />
```

### 渐进迁移

1. 在 `MemoEditor/index.tsx` 中添加条件渲染
2. 根据功能标志选择使用哪个工具栏
3. 保留旧组件作为回退

---

## 文件结构

```
web/src/components/MemoEditor/Toolbar/
├── ZenToolbar.tsx       # 主工具栏组件
├── InsertMenu.tsx        # 原有插入菜单（保留）
├── VisibilitySelector.tsx # 原有可见性选择器（保留）
└── index.ts              # 导出文件
```

---

## 设计变体

### Geek Mode 变体（未来）

```tsx
// 可以添加 mode 属性
<ZenToolbar mode="geek" />
```

**特点**：
- 终端绿主题
- 故障效果动画
- 等宽字体

### Evolution Mode 变体（未来）

```tsx
<ZenToolbar mode="evolution" />
```

**特点**：
- 紫色渐变主题
- 有机脉动动画
- 发光效果

---

## 性能优化

### memo 优化
所有子组件使用 `memo` 包裹，避免不必要的重渲染。

### useCallback
所有事件处理器使用 `useCallback` 缓存。

### 懒加载
- LocationDialog 使用 `lazy` 动态导入
- 减少初始加载体积

---

## i18n 键

| 键 | 值 |
|:---|:-----|
| `editor.save` | 保存 |
| `common.cancel` | 取消 |
| `common.saving` | 保存中 |
| `common.upload` | 上传 |
| `tooltip.link-memo` | 关联笔记 |
| `tooltip.select-location` | 选择位置 |
| `editor.focus-mode` | 专注模式 |
| `memo.visibility.private` | 私有 |
| `memo.visibility.protected` | 工作区 |
| `memo.visibility.public` | 公开 |

---

## 未来扩展

### 可配置的工具栏

```tsx
interface ZenToolbarConfig {
  showAI?: boolean;
  showLocation?: boolean;
  showFocusMode?: boolean;
  customButtons?: ToolButton[];
}

<ZenToolbar config={customConfig} />
```

### 插件系统

```tsx
<ZenToolbar
  plugins={[
    { id: "custom", icon: CustomIcon, action: handleCustom }
  ]}
/>
```

---

*设计版本: v1.0*
*最后更新: 2026-02-11*
