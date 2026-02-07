# React/TypeScript 反模式与陷阱

> DivineSense 前端开发中的常见错误和陷阱

---

## 🔴 Critical: Tailwind CSS 4 陷阱

### 问题：语义化类名解析错误

Tailwind CSS 4 重新定义了容器宽度类，导致意外的布局问题：

```tsx
// ❌ 错误：这些类解析为 ~16px（而非期望的容器宽度）
<DialogContent className="max-w-md">      // 期望 384px，实际 ~16px
<SheetContent className="sm:max-w-sm">   // 期望 320px，实际 ~16px
<Sheet className="max-w-lg">            // 期望 512px，实际 ~16px
```

**原因**：Tailwind v4 使用 `--spacing-*` 变量，约 16px（而非传统容器宽度）

```tsx
// ✅ 正确：使用显式 rem 值
<DialogContent className="max-w-[28rem]">  // 448px
<SheetContent className="sm:max-w-[24rem]"> // 384px
<Sheet className="max-w-[32rem]">          // 512px
```

**常用宽度参考**：
| 用途 | rem 值 | 像素 |
|:-----|:-------|:-----|
| 小对话框 | `max-w-[24rem]` | 384px |
| 标准对话框 | `max-w-[28rem]` | 448px |
| 大对话框 | `max-w-[32rem]` | 512px |
| 宽内容 | `max-w-[42rem]` | 672px |

---

## 🔴 Critical: Flex 容器溢出

### 问题：h-full + padding 导致高度溢出

```tsx
// ❌ 错误：内层高度 = 100% + padding，导致溢出
<div className="flex-1 overflow-y-auto px-3 py-4">
  <div className="h-full w-full px-6 py-8">
    {/* 内容 */}
  </div>
</div>
```

**原因**：`h-full` = 100%，加上 `py-8` (64px) 超出父容器

```tsx
// ✅ 正确：使用 min-h-0 允许收缩
<div className="flex-1 overflow-y-auto px-3 py-4">
  <div className="min-h-0 w-full px-6 py-8">
    {/* 内容 */}
  </div>
</div>
```

---

## 🟡 常见错误

### Grid 容器上使用 max-w-*

```tsx
// ❌ 错误：导致列宽挤压
<div className="grid grid-cols-2 gap-3 max-w-xs">
  <Card /> <Card />  {/* 每列 160px - 被挤压 */}
</div>

// ✅ 正确：让 gap 控制宽度
<div className="grid grid-cols-2 gap-3">
  <Card /> <Card />
</div>
```

### i18n 硬编码

```tsx
// ❌ 错误：硬编码文本
<Button>Submit</Button>
<div className="text-red-500">Error occurred</div>

// ✅ 正确：使用 t() 函数
<Button>{t("button.submit")}</Button>
<div className="text-red-500">{t("errors.network")}</div>
```

### 组件命名

```tsx
// ❌ 错误：非 PascalCase
const userProfile = () => { ... }
const User_Profile = () => { ... }

// ✅ 正确：PascalCase
const UserProfile = () => { ... }
```

### Hooks 命名

```tsx
// ❌ 错误：缺少 use 前缀
const getData = () => { ... }
const UserState = () => { ... }

// ✅ 正确：use 前缀
const useGetData = () => { ... }
const useUserState = () => { ... }
```

---

## 性能陷阱

### 未使用 React.memo 的列表项

```tsx
// ❌ 错误：每次父组件更新都重新渲染
{items.map(item => (
  <MemoCard key={item.id} memo={item} />
))}

// ✅ 正确：用 memo 包装
const MemoCard = memo(({ memo }) => {
  // ...
}, (prev, next) => prev.memo.id === next.memo.id);
```

### 未使用 useCallback 的事件处理

```tsx
// ❌ 错误：每次渲染创建新函数
<MemoCard onClick={() => handleClick(item.id)} />

// ✅ 正确：使用 useCallback
const handleClickItem = useCallback((id: string) => {
  handleClick(id);
}, [handleClick]);

<MemoCard onClick={handleClickItem} />
```

---

## TanStack Query 模式

```tsx
// ✅ 正确的数据获取模式
const { data, isLoading, error } = useQuery({
  queryKey: ["blocks", conversationId],
  queryFn: () => api.blocks.list(conversationId),
  staleTime: 5 * 60 * 1000,  // 5分钟
});

// ✅ 正确的变更模式
const mutation = useMutation({
  mutationFn: (block: BlockCreate) => api.blocks.create(block),
  onSuccess: () => {
    queryClient.invalidateQueries({ queryKey: ["blocks"] });
  },
});
```

---

## 文件结构陷阱

### 组件与 hooks 分离

```
// ✅ 正确结构
components/AIChat/
├── ChatMessages.tsx       # 组件
├── useChatMessages.ts     # 组件专用 hook
├── ChatMessages.test.tsx  # 测试
└── types.ts               # 类型定义

// ❌ 避免：组件逻辑全部塞在一个文件
components/AIChat/
├── ChatMessages.tsx       # 2000+ 行，包含 hooks、类型、逻辑
```
