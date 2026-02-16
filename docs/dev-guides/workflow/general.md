# 开发工作流

> **保鲜状态**: ✅ 已验证 (2026-02-12) | **版本**: v0.99.0
>
> DivineSense 项目的标准开发流程和命令

---

## 多任务管理

**何时创建 TODO LIST**：3+ 个优化点、任务 > 1 小时

```
TaskCreate → TaskList → TaskUpdate(in_progress) → TaskUpdate(completed)
```

---

## 开发命令

### 服务控制
```bash
make start              # 启动所有服务（PostgreSQL + 后端 + 前端）
make stop               # 停止所有服务
make status             # 查看服务状态
make logs               # 查看日志
make restart            # 重启后端服务（需手动确认）
make run                # 仅启动后端
make web                # 仅启动前端
```

### 数据库
```bash
make db-shell           # 进入 PostgreSQL shell
make db-reset           # 重置数据库（破坏性）
make db-vector          # 验证 pgvector 扩展
```

### 检查和测试
```bash
make check-all          # 完整检查（格式 + vet + 测试 + i18n）
make check-i18n         # i18n 完整性检查
make ci-check           # 模拟 CI 检查
make test               # 运行所有测试
make test-ai            # AI 相关测试
```

### 构建
```bash
make build              # 构建后端二进制
make build-web          # 构建前端静态资源
make build-all          # 同时构建前后端
```

### 依赖
```bash
make deps-all           # 安装所有依赖（Go + pnpm）
```

---

## 服务重启规范

**⚠️ 禁止直接执行启停命令**

修改后端代码后，通知用户手动执行 `make restart`

---

## Git 提交流程

```
make check-all → feat/fix 分支 → PR → 合并
```

详细规范：@.claude/rules/git-workflow.md

### 分支命名
- 功能：`feat/<issue-id>-description`
- 修复：`fix/<issue-id>-description`
- 重构：`refactor/<issue-id>-description`

### Commit 格式
```
<type>(<scope>): <description>

Refs #<issue-id>

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>
```

---

## 前端开发

### 从 web/ 目录运行
```bash
pnpm dev               # 开发服务器
pnpm build             # 生产构建
pnpm lint              # Lint 检查
pnpm lint:fix          # 自动修复
pnpm check-i18n        # i18n 完整性检查
```

### 从项目根目录
```bash
make web               # 启动前端
make build-web         # 构建前端
```

### 添加新组件
1. 创建组件：`web/src/components/YourComponent.tsx`
2. 添加样式：使用 Tailwind 类（避免 `max-w-*` 语义类）
3. 国际化：使用 `t("key")` 包裹文本

### 添加新页面
1. 创建文件：`web/src/pages/your-page/index.tsx`
2. 配置路由：`web/src/router/index.tsx`
3. 添加导航：更新侧边栏或菜单

---

## 后端开发

### 基础命令
```bash
make run               # 启动后端
make lint              # golangci-lint
make vet               # go vet
make test-ai           # AI 相关测试
```

### 添加新 API
1. 定义 Proto：`proto/api/v1/your_service.proto`
2. 生成代码：`make generate`
3. 实现服务：`server/service/your_service/`
4. 注册路由：`server/router/v1/`
5. 测试：`go test ./server/service/your_service/...`

### 添加数据库迁移
在 `store/migration/postgres/migrate/` 目录下创建新的迁移 SQL 文件。

### 调试 AI 代理
```bash
# 查看代理日志
make logs | grep -i parrot

# 单独运行测试
go test -v ./ai/agent/... -run TestYourCase
```

---

## Proto 变更流程

1. 修改 `.proto` 文件
2. 运行 `make generate` 重新生成代码
3. 更新前后端绑定
4. 提交变更

---

## 构建与部署

### 本地构建
```bash
make build-all          # 构建全部
make build              # 仅构建后端
cd web && pnpm build    # 仅构建前端
```

### 发布版本
```bash
# 1. 更新版本号
# 2. 更新 CHANGELOG
# 3. 创建 Git Tag
git tag v1.x.x
git push origin v1.x.x
```

---

## 常见问题

### 端口占用
```bash
# 查看端口占用
lsof -i :25173  # 前端
lsof -i :28081  # 后端

# 杀掉进程
kill -9 <PID>
```

### 依赖问题
```bash
# 重新安装依赖
make deps-all

# 清理缓存
go clean -cache
cd web && rm -rf node_modules && pnpm install
```

---

## 相关文档

- [系统架构](../../architecture/overview.md) | [架构摘要](../../architecture/summary.md)
- [后端开发](../backend/database.md)
- [前端开发](../frontend/overview.md)
- [Agent 工作流](../agent/workflow.md)

---

*文档路径：docs/dev-guides/WORKFLOW.md*
