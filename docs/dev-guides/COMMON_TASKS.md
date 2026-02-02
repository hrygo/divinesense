# 常见开发任务步骤

> **保鲜状态**: ✅ 已验证 (2025-02-02) | **最后检查**: v6.0

> DivineSense 开发中的常用操作与最佳实践

---

## 🚀 日常开发

### 启动开发环境

```bash
# 一键启动全栈
make start

# 查看服务状态
make status

# 查看日志
make logs
```

### 提交代码前

```bash
# 1. 运行全量检查
make check-all

# 2. 格式化代码
cd web && pnpm lint:fix
go fmt ./...

# 3. 提交
git add .
git commit -m "feat: your message"
```

---

## 🔧 后端任务

### 添加新 API

1. **定义 Proto**: `proto/api/v1/your_service.proto`
2. **生成代码**: `make gen-proto`
3. **实现服务**: `server/service/your_service/`
4. **注册路由**: `server/router/v1/`
5. **测试**: `go test ./server/service/your_service/...`

### 添加数据库迁移

```bash
# 创建迁移文件
make migration-create NAME=add_new_table

# 应用迁移
make migration-up
```

### 调试 AI 代理

```bash
# 查看代理日志
make logs | grep -i parrot

# 单独运行测试
go test -v ./server/ai/parrot/... -run TestYourCase
```

---

## 🎨 前端任务

### 添加新组件

1. **创建组件**: `web/src/components/YourComponent.tsx`
2. **添加样式**: 使用 Tailwind 类
3. **导入使用**: 在页面中导入
4. **国际化**: 使用 `t("key")` 包裹文本

### 添加新页面

1. **创建文件**: `web/src/pages/your-page/index.tsx`
2. **配置路由**: `web/src/router/index.tsx`
3. **添加导航**: 更新侧边栏或菜单

### 检查国际化

```bash
cd web
pnpm check-i18n  # 检查未翻译的文本
```

---

## 📦 构建与部署

### 本地构建

```bash
# 构建全部
make build-all

# 仅构建后端
make build

# 仅构建前端
cd web && pnpm build
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

## 🐛 常见问题

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

## 🔗 相关文档

- [快速开始](../../README.md#快速开始)
- [系统架构](./ARCHITECTURE.md)
- [项目路径](./PROJECT_PATHS.md)
