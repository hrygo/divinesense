# DivineSense UI 设计系统

> **版本**: v1.1.0 | **更新时间**: 2026-02-11

本文档定义 DivineSense 项目的统一设计语言系统，包括色彩、排版、间距、圆角、阴影等设计 Token。

---

## 1. 设计原则

### 1.1 核心价值

- **清晰**：信息层级明确，视觉引导清晰
- **一致**：统一的视觉语言，可预测的交互体验
- **高效**：快速响应，流畅过渡
- **包容**：支持深色模式、减少动画偏好

### 1.2 设计哲学

DivineSense 采用 **"平静专注"** 的设计理念：
- 减少视觉干扰，专注内容创作
- 使用柔和的色彩和圆角，营造亲和感
- 适度的留白，避免信息过载

---

## 2. 色彩系统

### 2.1 语义化色彩 Token

```css
/* 主色 - 品牌蓝 */
--primary: oklch(0.45 0.08 250);
--primary-foreground: oklch(0.9818 0.0054 95.0986);

/* 次要色 */
--secondary: oklch(0.9245 0.0138 92.9892);
--secondary-foreground: oklch(0.4334 0.0177 98.6048);

/* 中性色 */
--muted: oklch(0.9341 0.0153 90.239);
--muted-foreground: oklch(0.5559 0.0075 97.4233);

/* 背景色 */
--background: oklch(0.9818 0.0054 95.0986);
--foreground: oklch(0.2438 0.0269 95.7226);

/* 卡片 */
--card: oklch(1 0 0);
--card-foreground: oklch(0.1908 0.002 106.5859);

/* 边框 */
--border: oklch(0.8847 0.0069 97.3627);
--input: oklch(0.7621 0.0156 98.3528);

/* 功能色 */
--destructive: oklch(0.35 0.02 250);
--destructive-foreground: oklch(0.95 0.005 250);
```

### 2.2 特殊模式色彩

#### Geek Mode（极客模式）
```css
--geek-primary: oklch(0.65 0.15 145);        /* 终端绿 */
--geek-cyan: oklch(0.7 0.12 210);            /* 赛博青 */
--geek-pink: oklch(0.65 0.15 330);            /* 赛博粉 */
--geek-bg: oklch(0.15 0.005 270);             /* 深色背景 */
```

#### Evolution Mode（进化模式）
```css
--evo-primary: oklch(0.6 0.15 280);            /* 进化紫 */
--evo-blue: oklch(0.65 0.12 240);              /* 科幻蓝 */
--evo-cyan: oklch(0.75 0.1 210);               /* 高亮青 */
--evo-bg: oklch(0.18 0.02 280);                /* 进化背景 */
```

---

## 3. 排版系统

### 3.1 字体栈

```css
--font-sans: ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont,
             "PingFang SC", "Microsoft YaHei", "Noto Sans SC",
             "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif,
             "Apple Color Emoji", "Segoe UI Emoji", "Noto Color Emoji";

--font-serif: ui-serif, Georgia, Cambria, "Times New Roman", Times, serif;

--font-mono: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas,
             "Liberation Mono", "Courier New", monospace;
```

### 3.2 字号规范

| Token | Tailwind | 值 | 用途 |
|:-----|:---------|:---|:-----|
| `--text-xs` | `text-xs` | 12px | 辅助信息、标签、徽章 |
| `--text-sm` | `text-sm` | 14px | 说明文字、次要信息 |
| `--text-base` | `text-base` | 16px | 正文内容（默认） |
| `--text-lg` | `text-lg` | 18px | 次级标题 |
| `--text-xl` | `text-xl` | 20px | 三级标题 |
| `--text-2xl` | `text-2xl` | 24px | 二级标题 |
| `--text-3xl` | `text-3xl` | 30px | 一级标题 |

### 3.3 字重规范

| 用途 | 字重 | Tailwind |
|:-----|:-----|:---------|
| 正文 | 400 | `font-normal` |
| 强调 | 500 | `font-medium` |
| 半粗 | 600 | `font-semibold` |
| 粗体 | 700 | `font-bold` |

### 3.4 行高规范

| 用途 | 行高 | Tailwind |
|:-----|:-----|:---------|
| 标题 | 1.25 | `leading-tight` |
| 正文 | 1.5 | `leading-relaxed` |
| 紧凑 | 1.25 | `leading-snug` |

---

## 4. 间距系统

### 4.1 基础间距 Token

```css
--spacing-xs: 0.25rem;  /* 4px */
--spacing-sm: 0.5rem;   /* 8px */
--spacing-md: 1rem;     /* 16px */
--spacing-lg: 1.5rem;   /* 24px */
--spacing-xl: 2rem;     /* 32px */
--spacing-2xl: 3rem;    /* 48px */
```

### 4.2 组件内边距规范

| 组件类型 | 内边距 | 说明 |
|:---------|:-------|:-----|
| 按钮（默认） | `px-3 py-1.5` | 水平 12px，垂直 6px |
| 按钮（小） | `px-2 py-1` | 水平 8px，垂直 4px |
| 按钮（大） | `px-4 py-2` | 水平 16px，垂直 8px |
| 输入框 | `px-3 py-1.5` | 与按钮一致 |
| 卡片 | `p-4` 或 `p-6` | 根据内容密度 |
| 对话框 | `p-6` | 充足的呼吸空间 |

---

## 5. 圆角系统

### 5.1 圆角规范

| Token | Tailwind | 值 | 用途 |
|:-----|:---------|:---|:-----|
| `--radius-sm` | `rounded-sm` | 4px | 菜单项、下拉选项 |
| `--radius-md` | `rounded-md` | 6px | 按钮、输入框 |
| `--radius-lg` | `rounded-lg` | 8px | 卡片、面板、对话框 |
| `--radius-xl` | `rounded-xl` | 12px | 大型容器、特殊面板 |
| `--radius-full` | `rounded-full` | 圆形（头像、图标按钮） |

**注意**：`--radius-xl` 为 `calc(var(--radius) + 4px)`，基于 `--radius: 0.5rem` (8px) 计算。

### 5.2 使用规则

**推荐使用**：
- **默认**：`rounded-md`（按钮、输入框）
- **卡片**：`rounded-lg`（内容卡片、面板、对话框）
- **菜单项**：`rounded-sm`（下拉菜单、选择器选项 - 保持紧凑）
- **头像**：`rounded-full`

**避免使用**：
- `rounded-3xl`（太大，过于圆润）
- `rounded-xl` 在小元素上（视觉不协调）

---

## 6. 阴影系统

### 6.1 阴影规范

| Token | Tailwind | 值 | 用途 |
|:-----|:---------|:---|:-----|
| `--shadow-2xs` | 自定义 | `0 1px 3px 0px hsl(0 0% 0% / 0.05)` | 按钮默认状态 |
| `--shadow-xs` | `shadow-xs` | 悬停状态 |
| `--shadow-sm` | `shadow-sm` | 轻微提升效果 |
| `--shadow` | `shadow` | 下拉菜单、弹出层 |
| `--shadow-md` | `shadow-md` | 卡片悬停 |
| `--shadow-lg` | `shadow-lg` | 对话框、模态框 |
| `--shadow-xl` | `shadow-xl` | 重要提示、通知 |
| `--shadow-2xl` | `shadow-2xl` | 特殊强调 |

### 6.2 使用规则

- **按钮默认**：`shadow-xs`（使用 --shadow-xs）
- **按钮悬停**：`shadow-sm`
- **卡片**：`shadow-sm`
- **下拉菜单**：`shadow`
- **模态框**：`shadow-lg`

---

## 7. 组件规范

### 7.1 按钮

| 尺寸 | 高度 | 水平内边距 | 字号 | 圆角 |
|:-----|:-----|-----------|:-----|:-----|
| Small | `h-7` | `px-2` | `text-sm` | `rounded-md` |
| Default | `h-8` | `px-3` | `text-sm` | `rounded-md` |
| Large | `h-9` | `px-4` | `text-sm` | `rounded-md` |
| Icon | `size-8` | - | - | `rounded-full` |

### 7.2 输入框

| 属性 | 值 |
|:-----|:---|
| 高度 | `h-9` |
| 内边距 | `px-3 py-1.5` |
| 圆角 | `rounded-md` |
| 边框 | `border border-input` |
| 聚焦阴影 | `ring-2 ring-ring ring-offset-2` |

### 7.3 卡片

| 属性 | 值 |
|:-----|:---|
| 内边距 | `p-4` 或 `p-6` |
| 圆角 | `rounded-lg` (8px) |
| 阴影 | `shadow-sm` |
| 边框 | `border border-border` |
| 背景 | `bg-card` |

### 7.4 对话框

| 属性 | 值 |
|:-----|:---|
| 内边距 | `p-6` |
| 圆角 | `rounded-lg` (8px) - **注意：不是 rounded-xl** |
| 阴影 | `shadow-lg` |
| 边框 | 无 |
| 背景 | `bg-background` |

**重要**：对话框使用 `rounded-lg`，而非 `rounded-xl`，与卡片保持一致的圆角大小。

---

## 8. 动画规范

### 8.1 过渡时长

**直接使用 Tailwind 类**：

| Tailwind 类 | 时长 | 用途 |
|:-----------|:-----|:-----|
| `duration-100` | 100ms | 极快反馈 |
| `duration-150` | 150ms | 快速交互 |
| `duration-200` | 200ms | 标准过渡 |
| `duration-300` | 300ms | 复杂动画 |
| `duration-500` | 500ms | 页面切换 |

**默认值**：`--default-transition-duration: 150ms` (在 `index.css` 中定义)

### 8.2 缓动函数

**直接使用 Tailwind 类**：

| Tailwind 类 | 曲线 | 用途 |
|:-----------|:-----|:-----|
| `ease-linear` | 线性 | 匀速动画 |
| `ease-in` | 加速 | 进入动画 |
| `ease-out` | 减速 | 离开动画 |
| `ease-in-out` | 先加速后减速 | 往返动画 |

### 8.3 减少动画支持

```css
@media (prefers-reduced-motion: reduce) {
  * {
    animation-duration: 0.01ms !important;
    animation-iteration-count: 1 !important;
    transition-duration: 0.01ms !important;
  }
}
```

---

## 9. 响应式断点

```css
/* Mobile First */
sm: 640px;   /* 小屏幕 */
md: 768px;   /* 中等屏幕 */
lg: 1024px;  /* 大屏幕 */
xl: 1280px;  /* 超大屏幕 */
2xl: 1536px; /* 极大屏幕 */
```

### 9.1 响应式内容宽度

```css
/* 移动优先 */
max-w-3xl;     /* 768px  - 默认 */
lg:max-w-4xl;  /* 896px  - 大屏幕 */
xl:max-w-5xl;  /* 1024px - 超大 */
2xl:max-w-6xl; /* 1152px - 极大 */
```

---

## 10. Z-index 层级规范

| 层级 | 值 | 用途 |
|:-----|:---|:-----|
| Dropdown | `z-10` | 下拉菜单 |
| Sticky | `z-20` | 粘性头部 |
| Sidebar | `z-30` | 侧边栏 |
| Modal Overlay | `z-40` | 模态遮罩 |
| Modal Content | `z-50` | 模态内容 |
| Toast/Notification | `z-50` | 通知提示 |

---

## 11. 表单状态规范

### 11.1 Focus 状态

- **Ring 大小**：`ring-2` 或 `ring-4` (输入框)
- **Ring 颜色**：`ring-primary/10` (浅色背景) 或 `ring-ring`
- **Ring 偏移**：`ring-offset-2`

**示例**：
```tsx
<input className="focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2" />
```

### 11.2 Disabled 状态

- **透明度**：`disabled:opacity-50`
- **指针事件**：`disabled:pointer-events-none`
- **光标**：`cursor: not-allowed` (全局 CSS 中定义)

### 11.3 Error 状态

- **边框颜色**：`border-destructive`
- **文本颜色**：`text-destructive`
- **图标颜色**：`text-destructive`

---

## 12. 图标尺寸规范

| 尺寸类 | 像素值 | 用途 |
|:-------|:------|:-----|
| `size-3` | 12px | 极小图标 |
| `size-4` | 16px | 小图标（按钮内、文本前） |
| `size-5` | 20px | 默认图标 |
| `size-6` | 24px | 中等图标 |
| `size-8` | 32px | 大图标（图标按钮） |

**推荐**：使用 `size-*` 类而非 `h-* w-*`，更简洁。

---

## 13. 使用指南

### 13.1 选择正确的圆角

```tsx
/* ✅ 正确 */
<button className="px-4 py-2 rounded-md">按钮</button>
<div className="p-4 rounded-lg shadow-sm">卡片</div>

/* ❌ 错误 - 过于圆润 */
<button className="rounded-3xl">按钮</button>
```

### 13.2 选择正确的阴影

```tsx
/* ✅ 正确 */
<div className="shadow-sm hover:shadow">卡片</div>
<Dialog className="shadow-lg">对话框</Dialog>

/* ❌ 错误 - 过重 */
<div className="shadow-2xl">普通卡片</div>
```

### 13.3 选择正确的字号

```tsx
/* ✅ 正确 */
<h1 className="text-3xl font-bold">一级标题</h1>
<p className="text-sm text-muted-foreground">说明文字</p>
<span className="text-xs">标签</span>

/* ❌ 错误 - 信息层级混乱 */
<h3 className="text-base">三级标题</h3>
```

---

## 14. 迁移指南

### 14.1 圆角迁移

| 旧值 | 新值 | 影响范围 |
|:-----|:-----|:---------|
| `rounded-xl` | `rounded-lg` | 对话框除外 |
| `rounded-2xl` | `rounded-xl` | 大头像除外 |
| `rounded-full`（非头像） | `rounded-md` | 图标按钮 |

### 14.2 阴影迁移

| 旧值 | 新值 | 影响范围 |
|:-----|:-----|:---------|
| 无阴影 | `shadow-sm` | 卡片 |
| `shadow` | `shadow-sm` | 按钮、卡片 |
| `shadow-lg` | `shadow` | 非关键弹窗 |

---

## 15. 版本历史

| 版本 | 日期 | 变更内容 |
|:-----|:-----|:---------|
| v1.1.0 | 2026-02-11 | 修正圆角规范（移除不存在的 radius-2xl），添加对话框规范，添加 Z-index/表单/图标规范，修正动画说明 |
| v1.0.0 | 2026-02-11 | 初始版本，建立统一设计系统 |

---

*本文档随项目演进持续更新。*
