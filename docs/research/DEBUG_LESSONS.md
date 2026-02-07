# 调试经验教训

> 记录 DivineSense 开发过程中遇到的典型问题和解决方案，避免重复踩坑。
>
> **保鲜状态**: ✅ 2026-02-07

---

## 快速查找

### 按类型分类

| 类型 | 问题 | 状态 |
|:-----|:-----|:-----|
| **[前端](#前端问题)** | [布局宽度不统一](#前端布局宽度不统一) | ✅ 已解决 |
| **[前端](#前端问题)** | [空白页面滚动条溢出](#空白页面滚动条溢出) | ✅ 已解决 |
| **[后端](#后端问题)** | [Go embed 忽略下划线文件](#go-embed-忽略以下划线开头的文件) | ✅ 已解决 |
| **[后端](#后端问题)** | [调试日志管理规范](#调试日志管理规范) | ⚠️ 规范 |
| **[AI](#ai-问题)** | [Evolution Mode 路由失败](#evolution-mode-路由失败) | ✅ 已解决 |
| **[AI](#ai-问题)** | [AI Token 统计与缓存指标](#ai-token-统计与缓存指标) | ✅ 已解决 |
| **[部署](#部署问题)** | [二进制部署运维权限](#二进制部署运维权限问题) | ✅ 已解决 |
| **[开发流程](#开发流程问题)** | [环境意识不足](#环境意识不足导致的重复错误) | ✅ 已解决 |

### 按关键词索引

| 关键词 | 相关问题 |
|:-------|:---------|
| `Protobuf` | Evolution Mode 路由失败 |
| `Tailwind` | 布局宽度不统一、空白页面滚动条溢出 |
| `Go embed` | 忽略下划线文件 |
| `UTF-8` | AI Token 统计 |
| `Docker` | 运维权限、环境意识 |
| `Makefile` | 环境意识 |

---

## 前端问题

### 前端布局宽度不统一

**问题**：不同页面在大屏幕上的最大宽度不一致，用户体验不统一。

**根本原因**：
1. **布局层级混乱**：Layout 层和 Page 层都设置了 `max-w-*`
2. **组件内部限制**：`MasonryColumn` 组件内部有 `max-w-2xl` 限制
3. **语义化类名陷阱**：Tailwind v4 的 `max-w-md/lg/xl` 解析为 ~16px

**解决方案**：
```tsx
// 所有主内容页面统一使用
max-w-[100rem]  // 1600px
mx-auto         // 居中
px-4 sm:px-6   // 响应式左右内边距
```

**经验教训**：
| 问题 | 教练 |
|:-----|:-----|
| **宽度规范分散** | 建立统一的设计 token，单一数据源 |
| **组件内部限制** | 组件应适配容器宽度，而非预设宽度 |
| **Tailwind v4 变化** | 升级时仔细阅读 Breaking Changes |
| **响应式断点** | 使用 sm/md/lg 而非硬编码像素值 |

---

### 空白页面滚动条溢出

**问题**：AI 聊天页面在空白状态（无消息）时仍显示滚动条。

**根本原因**：双重 padding + h-full 组合导致高度溢出
```
内层总高度 = 100% + 64px（上下 padding）→ 内容溢出 → 触发滚动条
```

**解决方案**：
```tsx
// ChatMessages.tsx
style={{ scrollbarGutter: "auto", ... }}  // 按需显示

// PartnerGreeting.tsx
className="... min-h-0 w-full px-6 py-8"  // 允许收缩
```

**经验教训**：
| 问题 | 教练 |
|:-----|:-----|
| **h-full 在 flex 容器中的陷阱** | flex 子元素使用 `h-full` + padding 会溢出 |
| **min-h-0 的神奇作用** | 允许 flex 子元素正确收缩 |
| **scrollbarGutter: stable 副作用** | 始终保留滚动条空间，即使不需要 |
| **嵌套 padding 累积** | 外层 py-4 + 内层 py-8 = 实际超出 100% |

---

## 后端问题

### Go embed 忽略以下划线开头的文件

**问题**：部署到生产环境后，部分 JavaScript 文件无法加载。

**错误表现**：
```
Failed to fetch dynamically imported module: .../Inboxes-3qwxzD_s.js
_baseFlatten-CWeGY8aD.js:1 Failed to load module script (MIME type: text/html)
```

**根本原因**：Go 的 `//go:embed` 指令会忽略**以下划线 `_` 开头的文件**

```
lodash-es 内部模块：
- _baseFlatten-xxx.js   ❌ 被 Go embed 忽略
- _baseMap-xxx.js        ❌ 被 Go embed 忽略
- sortBy-xxx.js         ✅ 正常嵌入
```

**解决方案**：修改 Vite 配置，将 lodash-es 模块打包到单个 chunk
```typescript
// vite.config.mts
manualChunks(id) {
  if (id.includes("lodash-es") || id.includes("/_base")) {
    return "lodash-vendor";  // 避免 _ 开头的文件名
  }
}
```

**构建验证**：
```bash
ls web/dist/assets/ | grep "^_"  # 应该为空
```

**经验教训**：
| 问题 | 教练 |
|:-----|:-----|
| **Go embed 文件过滤规则** | 忽略 `_` 开头文件，类似 Unix 的 `.` 隐藏文件 |
| **第三方库内部模块命名** | lodash-es 使用 `_` 前缀，与 Go embed 冲突 |
| **Vite/Rollup 默认行为** | 默认拆分模块为独立 chunk |
| **错误消息误导性** | "Failed to fetch module" 实际是 404 |

---

### 调试日志管理规范

**前端日志**：
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

**后端日志**：
```go
// ✅ 正确：使用结构化日志
slog.Info("AI chat started",
    "agent_type", req.AgentType,
    "user_id", req.UserID,
)

// ❌ 错误：过度调试
slog.Debug("Every single step", ...)
```

---

## AI 问题

### Evolution Mode 路由失败

**问题**：进化模式 (`evolutionMode: true`) 无法正确路由到后端。

**根本原因**：Protobuf JSON 序列化行为
```
@bufbuild/protobuf 的 create() 函数：
- true → 保留在 JSON 中
- false → 省略（Protobuf JSON 规范优化）
- undefined → 省略
```

**解决方案**：
```typescript
// 前端 Workaround
if (params.evolutionMode && request.evolutionMode === undefined) {
  (request as any).evolutionMode = true;
}
```

**经验教训**：
| 问题 | 教练 |
|:-----|:-----|
| **Protobuf JSON 序列化不一致** | 明确测试 true/false/undefined 三种情况 |
| **默认值省略导致歧义** | 关键路由字段明确传递 false 而非省略 |
| **前后端类型不对等** | TypeScript `undefined` ≠ Go `false` |

---

### AI Token 统计与缓存指标

**问题**：日志显示 `content_length=451`，数据库 `LENGTH(content)=163`

**根本原因**：UTF-8 编码差异
```
日志：octet_length(assistant_content) = 451 字节
数据库：LENGTH(assistant_content) = 163 字符
原因：中文字符在 UTF-8 中占用 3 字节
```

**cache_read_tokens 数据流**：
```
DeepSeek API (prompt_cache_hit_tokens)
    ↓ go-openai 库映射
PromptTokensDetails.CachedTokens
    ↓ ai/llm.go:206
CacheReadTokens
    ↓ 数据库
ai_block.token_usage.cache_read_tokens
```

**DeepSeek 上下文缓存**：
- **缓存粒度**：64 token 块
- **工作原理**：相同会话前缀的后续请求自动命中缓存
- **缓存率**：5760 / 8000 ≈ 72%

**经验教训**：
| 问题 | 教练 |
|:-----|:-----|
| **字节 vs 字符混淆** | UTF-8 中文字符 = 3 字节，区分 `octet_length` 和 `LENGTH` |
| **缓存指标来源不明** | 追踪完整数据流：API → SDK → 业务代码 → 数据库 |

---

## 部署问题

### 二进制部署运维权限问题

**问题**：divine 用户执行运维操作遇到权限问题。

**错误表现**：
```
Authentication is required to restart 'divinesense.service'
permission denied while trying to connect to the docker API
```

**根本原因**：
1. **docker 组缺失**：用户未加入 docker 组
2. **sudoers 未配置**：无免密执行 systemctl 权限
3. **缺少运维工具**：无 Makefile 运维工具

**解决方案**：
```bash
# 1. 配置 docker 组
usermod -aG docker divine

# 2. 配置 sudoers 免密
cat > /etc/sudoers.d/divinesense << 'EOF'
divine ALL=(ALL) NOPASSWD: /bin/systemctl status divinesense.service
divine ALL=(ALL) NOPASSWD: /bin/systemctl restart divinesense.service
EOF

# 3. 创建用户运维 Makefile
# 4. 配置 bash 别名 (ds-status, ds-restart, ...)
```

**经验教训**：
| 问题 | 教练 |
|:-----|:-----|
| **部署脚本不完整** | 安装脚本应自动配置用户运维权限 |
| **docker 组需重新登录** | 加入 docker 组后必须重新登录生效 |
| **sudo 安全最小化** | 仅开放必要命令免密 |
| **运维工具缺失** | 创建友好的 Makefile 而非让用户直接敲命令 |

---

## 开发流程问题

### 环境意识不足导致的重复错误

**问题**：AI Agent 执行命令时频繁犯错。

**错误模式**：
| 错误操作 | 正确操作 | 原因 |
|:---------|:---------|:-----|
| `docker exec divinesense-postgres` | `make db-shell` | 容器名自动检测 |
| `pnpm build`（根目录）| `make build-web` | `package.json` 在 `web/` 下 |

**环境配置对照**：
| 环境 | 容器名 | 端口 | 用户 |
|:-----|:-------|:-----|:-----|
| **开发** | `divinesense-postgres-dev` | 25432 | `divinesense` |
| **生产** | `divinesense-postgres` | 无映射 | `divine` |

**解决方案**：

**1. 优先使用 Makefile wrapper**
```bash
make db-shell          # 自动检测容器
make build-web         # 自动处理目录
make web               # 启动前端
make ci-frontend       # lint + build
```

**2. Makefile 环境自动检测**
```makefile
POSTGRES_CONTAINER := $(shell docker ps --filter "name=postgres" --format "{{.Names}}" | head -1)
```

**3. 执行前检查清单**
- [ ] 当前工作目录是否正确？
- [ ] 目标文件/容器是否存在？
- [ ] 是否有 Makefile wrapper 可用？
- [ ] 环境变量是否正确（dev/prod）？

**经验教训**：
| 问题 | 教练 |
|:-----|:-----|
| **假设而非验证** | 执行前先检查环境，不要假设默认状态 |
| **忽略项目约定** | Makefile 是项目命令的标准入口 |
| **缺少上下文感知** | 多环境配置下必须明确当前环境 |
| **重复性错误** | 一次错误是疏忽，重复是流程问题 |

**预防措施**：
1. **优先使用 Makefile** — 所有操作通过 Makefile 执行
2. **添加容器检测** — Makefile 自动检测运行的容器
3. **环境前缀** — 开发环境资源名称加 `-dev` 后缀

---

## 贡献指南

当你遇到一个新的调试问题时：

1. **记录问题**：在此文档添加新章节，标题格式：`## 问题名称`
2. **描述现象**：用户可见的故障表现
3. **分析原因**：深入分析，不要停留在表面
4. **记录方案**：最终采用的解决方案
5. **提炼教训**：可复用的经验，避免重复踩坑
6. **更新索引**：在"快速查找"章节添加对应条目

**防腐原则**：
- 标题使用通用名称，不包含日期
- 描述与代码版本无关的通用问题
- 依赖 Git 历史追踪时间线
