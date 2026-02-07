# Web 性能与渲染稳定性优化调研报告

> **调研日期**: 2026-01-31
> **关联 Issue**: [#19](https://github.com/hrygo/divinesense/issues/19)
> **状态**: 待实施

---

## 执行摘要

优化 DivineSense Web 前端性能，重点关注 **Chat 页面的 TTI（Time To Interactive）和渲染稳定性**，解决用户输入时的跳动闪动问题。

---

## 一、问题分析

### 1.1 用户报告的问题

| 问题 | 场景 | 影响 |
|:-----|:-----|:-----|
| **输入跳动** | Chat 页面输入时 | 输入框高度变化导致聊天区域上下跳动 |
| **流式输出跳动** | AI 回复流式渲染时 | 每次新增内容导致布局抖动 |
| **模式切换闪动** | 切换 Geek/Evolution 模式 | 侧边栏条件渲染导致布局偏移 |
| **路由加载延迟** | 页面切换 | 白屏时间较长，无加载提示 |

### 1.2 根因分析

#### ChatInput 跳动根因

**文件**: `web/src/components/AIChat/ChatInput.tsx:77-81`

```tsx
const handleInput = useCallback((e: React.FormEvent<HTMLTextAreaElement>) => {
  const target = e.target as HTMLTextAreaElement;
  target.style.height = "auto";           // ← 重置为 auto，容器瞬间收缩
  const newHeight = Math.min(target.scrollHeight, 120);
  target.style.height = `${newHeight}px`;  // ← 恢复高度，容器下移
}, []);
```

**影响链路**：
```
用户输入 → height="auto" → 输入框变矮 → 聊天区域上移
         → height=80px  → 输入框恢复 → 聊天区域下移
         → 结果：整个聊天区域上下跳动
```

#### 侧边栏条件渲染问题

**文件**: `web/src/layouts/AIChatLayout.tsx:162-172`

```tsx
{lg && !immersiveMode && (
  <div className="fixed top-0 left-16 ... w-72">
    <AIChatSidebar />
  </div>
)}
```

**问题**: 条件为 false 时 div 被完全移除 DOM，导致主内容区域的 `pl-72` padding 失效，布局重新计算。

---

## 二、当前状态基线

### 2.1 已有的优化

| 优化项 | 状态 | 说明 |
|:------|:-----|:-----|
| 路由懒加载 | ✅ 已实现 | 所有页面除 Home 外已使用 `lazy()` |
| 代码分割 | ✅ 已实现 | 8 个 vendor chunks 已分离 |
| 流式 Markdown | ✅ 已实现 | StreamingMarkdown 句子边界检测 |
| Mermaid 动态导入 | ✅ 已实现 | `await import('mermaid')` |

### 2.2 性能瓶颈

| 指标 | 现状 | 目标 |
|:-----|:-----|:-----|
| 最大包 | markdown-vendor 1.3MB | 无变化 |
| 输入响应 | 跳动闪动 | 平滑无跳动 |
| 模式切换 | 布局偏移 | 无偏移 |
| TTI | 未测量 | 改善 20%+ |

---

## 三、解决方案

### 3.1 P0 修复（必须）

#### A. ChatInput 高度调整优化

```tsx
// 修复方案：RAF 批处理 + 缓存优化
const handleInput = useCallback((e: React.FormEvent<HTMLTextAreaElement>) => {
  const target = e.target as HTMLTextAreaElement;

  // 缓存当前高度，避免不必要的 auto 重置
  const currentHeight = target.scrollHeight;

  requestAnimationFrame(() => {
    const newHeight = Math.min(currentHeight, 120);
    // 只在高度差异 > 4px 时才更新
    if (Math.abs(parseInt(target.style.height || "44") - newHeight) > 4) {
      target.style.height = `${newHeight}px`;
    }
  });
}, []);
```

#### B. CSS Contain 隔离

```css
.chat-container {
  contain: layout style paint;
  content-visibility: auto;
}

.chat-input-wrapper {
  contain: layout;
}
```

#### C. 侧边栏占位修复

```tsx
// 使用 hidden 替代条件渲染
<div className={cn(
  "fixed top-0 left-16 shrink-0 h-svh border-r w-72 pt-2",
  lg ? (immersiveMode ? "hidden" : "block") : "hidden"
)}>
  <AIChatSidebar />
</div>

// 主内容 padding 固定
<div className={cn("flex-1 min-h-0 overflow-hidden", "lg:pl-72")}>
```

### 3.2 P1 优化（重要）

#### D. useDeferredValue 降级

```tsx
const ChatMessages = memo(function ChatMessages({ items, ... }) {
  // 降低消息列表更新优先级
  const deferredItems = useDeferredValue(items);
  const displayItems = deferredItems.length === items.length ? items : deferredItems;

  return <div ...>{displayItems.map(...)}</div>;
});
```

#### E. 路由骨架屏

**新建文件**: `web/src/components/PageSkeleton.tsx`

### 3.3 P2 优化（可选）

#### F. React Compiler

```tsx
// vite.config.mts
react({
  babel: {
    plugins: [["babel-plugin-react-compiler", {}]]
  }
})
```

---

## 四、实施计划

| 阶段 | 任务 | 工作量 | 优先级 |
|:----:|:-----|:-------|:-------|
| Phase 1 | ChatInput 优化 | 0.5 人日 | P0 |
| Phase 1 | ChatMessages useDeferredValue | 0.5 人日 | P0 |
| Phase 1 | AIChatLayout 侧边栏占位 | 0.5 人日 | P0 |
| Phase 2 | PageSkeleton 组件 | 1 人日 | P1 |
| Phase 2 | Home 懒加载 | 0.5 人日 | P1 |
| Phase 3 | React Compiler | 1 人日 | P2 |

**总计**: 3-4 人日

---

## 五、验收标准

### 功能验收
- [ ] 输入时无跳动闪动
- [ ] 流式输出时布局稳定
- [ ] 模式切换时无布局偏移
- [ ] 路由切换有骨架屏

### 技术验收
- [ ] `make check-all` 通过
- [ ] 无 Console 错误
- [ ] 移动端键盘适配正常

### 性能验收
- [ ] FCP 改善 20%+
- [ ] TTI 改善 20%+
- [ ] CLS < 0.1

---

## 六、参考资源

- [React useTransition](https://react.dev/reference/react/useTransition)
- [React useDeferredValue](https://react.dev/reference/react/useDeferredValue)
- [CSS Contain](https://developer.mozilla.org/en-US/docs/Web/CSS/contain)
- [React Compiler](https://react.dev/learn/react-compiler)
- [content-visibility](https://developer.mozilla.org/en-US/docs/Web/CSS/content-visibility)

---

## 七、附录

### A. 文件变更清单

| 文件 | 类型 | 变更内容 |
|:-----|:-----|:---------|
| `web/src/components/AIChat/ChatInput.tsx` | 修改 | RAF 批处理 + 缓存优化 |
| `web/src/components/AIChat/ChatMessages.tsx` | 修改 | useDeferredValue + CSS contain |
| `web/src/layouts/AIChatLayout.tsx` | 修改 | 侧边栏 hidden 占位 |
| `web/src/components/PageSkeleton.tsx` | 新建 | 统一骨架屏组件 |
| `web/src/router/index.tsx` | 修改 | Home 懒加载 |
| `web/vite.config.mts` | 修改 | React Compiler 配置 |

### B. 关键代码片段

详见 Issue [#19](https://github.com/hrygo/divinesense/issues/19)。

---

*报告版本: v1.0 | 最后更新: 2026-01-31*
