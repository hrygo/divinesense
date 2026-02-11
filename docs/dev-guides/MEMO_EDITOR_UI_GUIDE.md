# MemoEditor UI 设计规范

> **版本**: v1.0 | **更新时间**: 2026-02-11 | **保鲜状态**: ✅

本文档定义了 DivineSense MemoEditor 组件的 UI 设计规范，确保整个编辑器系统的视觉一致性和用户体验质量。

---

## 设计原则

### 1. 禅意智识
- **呼吸感**: 组件间留白充足，使用 4px 倍数间距
- **流动感**: 状态切换动画流畅自然
- **专注感**: 内容为王，工具退居幕后

### 2. 统一规范
- **圆角**: 统一 `rounded-xl` (12px) 用于按钮和小组件，`rounded-2xl` (16px) 用于容器
- **按钮高度**: 移动端 `h-11` (44px)，桌面端 `h-9` (36px)
- **图标系统**: 统一使用 `lucide-react` 图标库

### 3. 微交互
- **悬停**: `hover:scale-105` + `hover:bg-accent/50`
- **点击**: `active:scale-95`
- **过渡**: `transition-all duration-200`

---

## 组件规范

### QuickInput（快速输入）

```tsx
// 容器
<div className="shrink-0 p-3 md:p-4 border-t border-border/50 bg-background">

  {/* 快捷键提示 */}
  <div className="mb-2">
    <kbd className="px-1.5 py-0.5 bg-muted/70 rounded-md text-[10px] font-mono">Enter</kbd>
  </div>

  {/* 输入框容器 */}
  <div className="flex items-end gap-2 md:gap-3 p-3 rounded-2xl border shadow-sm
              bg-muted/30 hover:bg-muted/50
              focus-within:ring-2 focus-within:ring-primary/20">

    {/* 文本域 */}
    <Textarea className="flex-1 min-h-[44px] max-h-[120px]" />

    {/* 发送按钮 */}
    <Button className="h-11 min-w-[44px] rounded-xl" />
  </div>
</div>
```

**规范要点**:
- 输入框最小高度 44px（移动端触摸友好）
- 发送按钮带微交互：`hover:scale-105 active:scale-95`
- AI 就绪提示：内容超过 20 字符时显示紫色 Sparkles 图标

### StandardToolbar（标准工具栏）

```tsx
// 工具栏容器
<div className="w-full flex items-center justify-between
                gap-1.5 sm:gap-3 px-3 sm:px-4 py-2.5
                border-t border-border/50 bg-muted/20 backdrop-blur-sm">

  {/* 左侧工具按钮 */}
  <div className="flex items-center gap-1 sm:gap-2">
    <Button className="h-9 w-9 rounded-xl
                   hover:bg-accent/50
                   hover:scale-105 active:scale-95" />
  </div>

  {/* 右侧操作按钮 */}
  <div className="flex items-center gap-1.5 sm:gap-2">
    <Button className="h-9 px-3 rounded-xl" />           {/* 取消 */}
    <Button className="h-9 min-w-[80px] rounded-xl" />    {/* 保存 */}
  </div>
</div>
```

**桌面端工具按钮**:
- `Paperclip` - 上传文件
- `Link2` - 关联笔记
- `MapPin` - 添加位置
- `Maximize2` - 聚焦模式

**移动端**:
- AI 指示器：内容 > 10 字符时显示紫色 Sparkles 图标
- 更多按钮：打开 MobileToolbarSheet

### MobileToolbarSheet（移动端工具面板）

```tsx
<SheetContent side="bottom"
              className="h-[45vh] rounded-t-2xl
                         border-t border-border/50
                         bg-background/95 backdrop-blur-md">

  {/* 拖动手柄 */}
  <div className="w-12 h-1.5 bg-muted-foreground/20 rounded-full" />

  {/* 工具网格 */}
  <div className="grid grid-cols-3 gap-4">
    <Button className="flex flex-col gap-3 py-4 rounded-2xl
                    h-auto hover:bg-accent/50 active:scale-95">
      <div className="w-11 h-11 rounded-xl
                  bg-gradient-to-br from-primary/20 to-primary/5">
        <Icon className="w-5 h-5 text-primary" />
      </div>
    </Button>
  </div>
</SheetContent>
```

**工具图标容器颜色**:
- 上传文件：`from-primary/20 to-primary/5` + `text-primary`
- 关联笔记：`from-blue-500/20 to-blue-500/5` + `text-blue-500`
- 添加位置：`from-green-500/20 to-green-500/5` + `text-green-500`

### FocusModeEditor（聚焦模式编辑器）

```tsx
<div className="fixed z-50 max-w-5xl mx-auto
                shadow-lg border-border bg-background rounded-lg
                top-2 left-2 right-2 bottom-2
                sm:top-4 sm:left-4 sm:right-4 sm:bottom-4
                animate-in fade-in-0 zoom-in-95 duration-300">

  {/* 头部 */}
  <div className="flex items-center justify-between
                  px-5 py-3 border-b border-border/50 bg-muted/30">
    <span className="text-sm font-medium text-muted-foreground/80">
      {t("editor.focus-mode")}
    </span>
    <button className="px-3 py-1.5 rounded-xl hover:bg-accent/50">
      <Minimize2 className="w-4 h-4" />
      Exit (ESC)
    </button>
  </div>

  {/* 编辑区 */}
  <div className="min-h-[300px] max-h-[60vh] px-6 py-4">
    <Editor className="min-h-[300px]" />
  </div>

  {/* 工具栏 */}
  <StandardToolbar />
</div>
```

**动画规范**:
- 进入：`animate-in fade-in-0 zoom-in-95 duration-300`
- 退出：`animate-out fade-out-0 zoom-out-95 duration-200`

---

## 颜色系统

### 主题色

| 用途 | CSS 变量 | Tailwind 类 |
|:-----|:---------|:-----------|
| 主色 | `--primary` | `bg-primary` |
| AI 指示 | - | `text-purple-500` / `text-purple-600` |
| 成功 | - | `text-green-500` |
| 背景 | `--background` | `bg-background` |
| 面板 | `--muted` | `bg-muted/30` |

### 语义化颜色

| 状态 | 颜色类 | 用途 |
|:-----|:-------|:-----|
| 默认 | `text-muted-foreground` | 未激活的工具按钮 |
| 悬停 | `hover:text-foreground` | 工具按钮悬停 |
| 激活 | `text-foreground` | 当前激活状态 |
| 禁用 | `opacity-50 cursor-not-allowed` | 禁用状态 |

---

## 间距系统

### 基础单位
- **基准**: 4px
- **小间距**: 8px (`gap-2`)
- **中间距**: 12px (`gap-3`)
- **大间距**: 16px (`gap-4`)

### 组件内间距

| 组件 | padding | gap |
|:-----|:-------|:-----|
| QuickInput | `p-3 md:p-4` | `gap-2 md:gap-3` |
| StandardToolbar | `px-3 sm:px-4 py-2.5` | `gap-1.5 sm:gap-3` |
| MobileToolbarSheet | `px-6` | `gap-4` |
| FocusModeEditor | `px-6 py-4` | - |

---

## 图标规范

### 图标库
- **统一使用**: `lucide-react`
- **禁止**: 自定义 SVG 代码

### 图标尺寸

| 位置 | 尺寸 | 用途 |
|:-----|:-----|:-----|
| 工具按钮 | `w-4 h-4` | 桌面端工具栏 |
| 工具按钮 | `w-5 h-5` | 移动端工具面板 |
| 发送按钮 | `w-5 h-5` | 发送图标 |
| 小指示器 | `w-3 h-3` | AI 就绪提示 |

---

## 按钮规范

### 尺寸

| 类型 | 高度 | 宽度 | 圆角 | 场景 |
|:-----|:-----|:-----|:-----|:-----|
| 移动端按钮 | `h-11` (44px) | `w-11` | `rounded-xl` | 触摸友好 |
| 桌面端按钮 | `h-9` (36px) | `w-9` | `rounded-xl` | 工具按钮 |
| 主要按钮 | `h-9` | `min-w-[80px]` | `rounded-xl` | 保存/发送 |

### 状态

```tsx
// 禁用状态
className="opacity-50 cursor-not-allowed disabled:hover:scale-100"

// 默认状态
className="bg-muted text-muted-foreground hover:bg-accent/50"

// 激活状态
className="bg-primary text-primary-foreground hover:bg-primary/90 shadow-sm hover:shadow"
```

### 微交互

```tsx
className="transition-all duration-200
           hover:scale-105
           active:scale-95"
```

---

## 响应式断点

| 断点 | 宽度 | 行为 |
|:-----|:-----|:-----|
| `sm` | 640px | 显示完整工具栏 |
| `md` | 768px | 标准布局 |
| `lg` | 1024px | 大屏布局 |
| 移动端 | < 640px | 简化工具栏 + Sheet |

---

## 动画规范

### 过渡时长

| 类型 | 时长 | 用途 |
|:-----|:-----|:-----|
| 快速 | 150ms | 按钮交互 |
| 标准 | 200ms | 一般过渡 |
| 缓慢 | 300ms | 进入/退出动画 |

### 关键帧动画

```tsx
// 进入动画
"animate-in fade-in-0 zoom-in-95 duration-300"

// 退出动画
"animate-out fade-out-0 zoom-out-95 duration-200"

// 脉冲动画（AI 指示）
"animate-pulse"

// 旋转动画（加载中）
"animate-spin"
```

---

## 可访问性

### 触摸目标
- **最小尺寸**: 44×44px（移动端）
- **推荐尺寸**: 48×48px

### 键盘导航
- `ESC`: 退出聚焦模式
- `Enter`: 发送笔记
- `Ctrl/Cmd + Enter`: 插入换行

### ARIA 标签
```tsx
aria-label={t("editor.send")}
aria-label={t("editor.more-tools")}
```

---

## 实现检查清单

在修改 MemoEditor 组件时，请确认以下检查项：

- [ ] 使用 `lucide-react` 图标，无自定义 SVG
- [ ] 按钮圆角为 `rounded-xl`
- [ ] 移动端按钮高度 `h-11`，桌面端 `h-9`
- [ ] 间距使用 4px 倍数
- [ ] 包含微交互 `hover:scale-105 active:scale-95`
- [ ] 过渡动画 `transition-all duration-200`
- [ ] AI 指示器使用紫色 (`text-purple-500`)
- [ ] i18n key 同步（运行 `make check-i18n`）
- [ ] 代码通过 lint 检查（运行 `pnpm lint`）

---

*本文档随 MemoEditor 组件演进自动更新。*
