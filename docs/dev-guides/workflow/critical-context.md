# Critical Context（必读上下文）

> DivineSense 项目关键信息速查 — 避免常见陷阱

---

## 项目结构

| 目录      | 说明                                    |
| :-------- | :-------------------------------------- |
| `web/`    | 前端根目录 — **始终从此处运行前端命令** |
| `ai/`     | AI 核心模块（Go 一级模块）              |
| `server/` | HTTP/gRPC 服务器                        |
| `store/`  | 数据访问层                              |
| `proto/`  | Protobuf 定义（修改后需重新生成）       |

---

## 关键配置

| 配置              | 值                         |
| :---------------- | :------------------------- |
| PostgreSQL 容器名 | `divinesense-postgres-dev` |
| 前端端口          | 25173                      |
| 后端端口          | 28081                      |
| 数据库端口        | 25432                      |

---

## 常见陷阱

| 陷阱                | 说明                                           |
| :------------------ | :--------------------------------------------- |
| `max-w-md` 等语义类 | Tailwind v4 解析为 ~16px，用 `max-w-[24rem]`   |
| i18n 不同步         | `make check-i18n` 检查 en.json 和 zh-Hans.json |
| 服务重启            | 修改后端代码后通知用户手动 `make restart`      |
| SQLite 无 AI        | 生产 AI 功能必须用 PostgreSQL                  |

### Tailwind v4 陷阱详解

**问题**：`max-w-sm/md/lg/xl` 在 Tailwind v4 中解析为约 16px（而非传统容器宽度）

**解决方案**：使用显式 rem 值
```tsx
// ❌ 错误（会坍缩到 ~16px）
<DialogContent className="max-w-md">

// ✅ 正确
<DialogContent className="max-w-[28rem]">  {/* 448px */}
```

### Go embed 忽略下划线文件

**问题**：`//go:embed` 忽略以下划线 `_` 开头的文件

**解决方案**：在 `vite.config.mts` 配置 `manualChunks`
```typescript
manualChunks(id) {
  if (id.includes("lodash-es") || id.includes("/_base")) {
    return "lodash-vendor";
  }
}
```

---

## 数据库策略

### PostgreSQL（生产环境）
- **AI 功能**：完整支持（pgvector、混合搜索、重排）
- **Tree Branching**：完整支持（ForkBlock、SwitchBranch）
- **推荐用途**：所有生产部署
- **维护状态**：积极维护

### SQLite（仅开发环境）
- **向量搜索**：✅ 支持（sqlite-vec）
- **AI 功能**：❌ 不支持（AIBlock、AgentStats 返回错误）
- **Tree Branching**：❌ 不支持
- **推荐用途**：仅限非 AI 功能的本地开发
- **维护状态**：仅对非 AI 功能尽力维护

---

## Background Runners

| Runner           | 职责               | 触发条件           |
| :--------------- | :----------------- | :----------------- |
| Embedding Runner | 向量嵌入后台任务   | Memo 创建/更新     |
| OCR Runner       | 图片文本提取       | 附件上传           |

> 运行在服务器启动时，自动处理异步任务

---

*文档路径：docs/essentials/CRITICAL_CONTEXT.md*
