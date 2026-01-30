# 调试经验教训

> 记录 DivineSense 开发过程中遇到的典型问题和解决方案，避免重复踩坑。

---

## Evolution Mode 路由失败 (2025-01)

### 问题描述
进化模式 (`evolutionMode: true`) 无法正确路由到后端，一直使用普通模式处理。

### 根本原因

**Protobuf JSON 序列化行为**：

```
@bufbuild/protobuf 的 create() 函数在 JSON 序列化时：
- true → 保留在 JSON 中
- false → 省略（Protobuf JSON 规范优化）
- undefined → 省略
```

当前端传递 `evolutionMode: true` 时正常工作，但早期版本存在以下边界情况：
1. connect-web 版本兼容性问题
2. proto 生成代码的边界情况
3. create() 函数在特定条件下无法正确设置字段

### 解决方案

**前端 Workaround**：
```typescript
// useAIQueries.ts
if (params.evolutionMode && request.evolutionMode === undefined) {
  (request as any).evolutionMode = true;
}
```

**后端日志验证**：
```go
// server/router/api/v1/ai/handler.go
slog.Info("AI chat handler received request",
    "evolution_mode", req.EvolutionMode,
    "evolution_mode_raw", fmt.Sprintf("%v", req.EvolutionMode),
)
```

### 经验教训

| 问题 | 教练 |
|:-----|:-----|
| **Protobuf JSON 序列化行为不一致** | 明确测试 true/false/undefined 三种情况 |
| **默认值省略导致歧义** | 对于关键路由字段，明确传递 false 而非省略 |
| **调试日志散落各处** | 使用统一的日志框架，方便开关 |
| **临时修复变成永久代码** | WORKAROUND 应标注过期时间或跟踪问题 |
| **前后端类型不完全对等** | TypeScript `undefined` ≠ Go `false`，需要显式转换 |

### 代码改进建议

```typescript
// 改进前：依赖 ?? 默认值
evolutionMode: params.evolutionMode ?? false

// 改进后：显式布尔转换（更安全）
evolutionMode: Boolean(params.evolutionMode)
```

---

## 前端布局宽度不统一 (2025-01)

### 问题描述
不同页面在大屏幕上的最大宽度不一致，用户体验不统一。

### 根本原因

1. **布局层级混乱**：Layout 层和 Page 层都设置了 `max-w-*`
2. **组件内部限制**：`MasonryColumn` 组件内部有 `max-w-2xl` 限制
3. **语义化类名陷阱**：Tailwind v4 的 `max-w-md/lg/xl` 解析为 ~16px

### 解决方案

**统一规范**：
```tsx
// 所有主内容页面统一使用
max-w-[100rem]  // 1600px
mx-auto         // 居中
px-4 sm:px-6   // 响应式左右内边距
```

**Layout 层统一**：
```tsx
// MemoLayout.tsx, ScheduleLayout.tsx 等
<div className={cn("flex-1 ...", lg ? "pl-72" : "")}>
  <div className={cn("w-full mx-auto px-4 sm:px-6 md:pt-6 pb-8", "max-w-[100rem]")}>
    <Outlet />
  </div>
</div>
```

### 经验教训

| 问题 | 教练 |
|:-----|:-----|
| **宽度规范分散** | 建立统一的设计 token，单一数据源 |
| **组件内部限制** | 组件应适配容器宽度，而非预设宽度 |
| **Tailwind v4 变化** | 升级时仔细阅读 Breaking Changes |
| **响应式断点** | 使用 sm/md/lg 而非硬编码像素值 |

---

## 调试日志管理规范

### 前端日志

```typescript
// ✅ 正确：生产环境移除，DEV 保留
if (import.meta.env.DEV) {
  console.debug("[Component] Debug info", data);
}

// ✅ 正确：错误日志始终保留
console.error("[Component] Error occurred:", error);

// ❌ 错误：无条件输出到控制台
console.log("[Component] Some info");
```

### 后端日志

```go
// ✅ 正确：使用结构化日志
slog.Info("AI chat started",
    "agent_type", req.AgentType,
    "user_id", req.UserID,
)

// ✅ 正确：关键路径记录
if req.EvolutionMode {
    slog.Info("Evolution mode detected, routing to EvolutionParrot")
}

// ❌ 错误：过度调试
slog.Debug("Every single step", ...)  // 应使用条件日志级别
```

---

## 贡献指南

当你遇到一个新的调试问题时：

1. **记录问题**：在此文档添加新章节，标题格式：`## 问题名称 (YYYY-MM)`
2. **描述现象**：用户可见的故障表现
3. **分析原因**：深入分析，不要停留在表面
4. **记录方案**：最终采用的解决方案
5. **提炼教训**：可复用的经验，避免重复踩坑

---

*文档维护：随项目演进持续更新*
