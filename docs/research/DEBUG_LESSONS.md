# 调试经验教训

> 记录 DivineSense 开发过程中遇到的典型问题和解决方案，避免重复踩坑。

---

## Evolution Mode 路由失败

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

## Go embed 忽略以下划线开头的文件

### 问题描述
部署到生产环境后，部分 JavaScript 文件无法加载，返回 404（实际上是 index.html fallback）。

**错误表现**：
```
Failed to fetch dynamically imported module: http://39.105.209.49/assets/Inboxes-3qwxzD_s.js
```

浏览器控制台显示：
```
_baseFlatten-CWeGY8aD.js:1 Failed to load module script: Expected a JavaScript module script but the server responded with a MIME type of "text/html"
```

### 根本原因

**Go embed 文件过滤规则**：

Go 的 `//go:embed` 指令会忽略**以下划线 `_` 开头的文件**。这是一个设计决策，与 Unix 忽略以 `.` 开头的隐藏文件类似。

```
lodash-es 内部模块被 Vite/Rollup 拆分为独立 chunk：
- _baseFlatten-xxx.js   (331 bytes)  ❌ 被 Go embed 忽略
- _baseMap-xxx.js        (199 bytes)  ❌ 被 Go embed 忽略
- sortBy-xxx.js         (1181 bytes)  ✅ 正常嵌入
- uniq-xxx.js            (98 bytes)   ✅ 正常嵌入
```

**问题链条**：
1. `lodash-es` 是一个模块化的 lodash 库，包含大量内部模块（`_baseFlatten`、`_baseMap` 等）
2. Vite/Rollup 默认将这些模块拆分为独立的 chunk 文件
3. 这些内部模块以 `_` 开头，被 Go embed 忽略
4. 浏览器请求这些文件时，收到的是 index.html（404 fallback）
5. HTML 作为 JavaScript 解析失败，导致整个应用崩溃

### 解决方案

**修改 Vite 配置，将 lodash-es 模块打包到单个 chunk**：

```typescript
// vite.config.mts
build: {
  rollupOptions: {
    output: {
      manualChunks(id) {
        // lodash-es internal modules - bundle into a single chunk
        if (id.includes("lodash-es") || id.includes("/_base")) {
          return "lodash-vendor";  // 生成 lodash-vendor-xxx.js
        }
        // ... 其他 vendor chunks
      },
    },
  },
}
```

**构建验证**：
```bash
# 检查是否有以下划线开头的文件
ls web/dist/assets/ | grep "^_"  # 应该为空

# 验证 lodash 被打包
ls web/dist/assets/ | grep lodash  # 应该看到 lodash-vendor-xxx.js
```

### 经验教训

| 问题 | 教练 |
|:-----|:-----|
| **Go embed 文件过滤规则** | `//go:embed` 忽略 `_` 开头文件，类似 Unix 的 `.` 隐藏文件 |
| **第三方库内部模块命名** | lodash-es 等库使用 `_` 前缀表示内部模块，与 Go embed 冲突 |
| **Vite/Rollup 默认行为** | 默认会拆分模块为独立 chunk，需为 Go embed 特殊配置 |
| **错误消息误导性** | "Failed to fetch module" 实际是 404，而非网络问题 |
| **SPA fallback 行为** | http.FileServer 的 HTML5 fallback 会返回 index.html，掩盖真实问题 |

### 预防措施

1. **构建时检查**：在 CI/CD 中添加检查脚本
   ```bash
   # 检查嵌入目录中是否有以下划线开头的文件
   if find server/router/frontend/dist/assets -name "_*" | grep -q .; then
     echo "ERROR: Found files starting with '_' which will be ignored by Go embed"
     exit 1
   fi
   ```

2. **Vite 配置规范**：为单二进制 Go 项目添加特定配置
   ```typescript
   // 避免生成 Go embed 不支持的文件名
   chunkFileNames: "assets/[name]-[hash].js",
   entryFileNames: "assets/[name]-[hash].js",
   assetFileNames: "assets/[name]-[hash].[ext]",
   ```

3. **测试清单**：
   - [ ] 验证所有 vendor chunks 都能正确加载
   - [ ] 检查浏览器控制台无 404 错误
   - [ ] 测试懒加载路由（如 Inboxes 页面）

---

## 贡献指南

当你遇到一个新的调试问题时：

1. **记录问题**：在此文档添加新章节，标题格式：`## 问题名称`
2. **描述现象**：用户可见的故障表现
3. **分析原因**：深入分析，不要停留在表面
4. **记录方案**：最终采用的解决方案
5. **提炼教训**：可复用的经验，避免重复踩坑

**防腐原则**：
- 标题使用通用名称，不包含日期
- 描述与代码版本无关的通用问题
- 依赖 Git 历史追踪时间线
