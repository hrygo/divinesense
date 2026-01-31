---
allowed-tools: Bash, Read, Grep, Edit
description: 执行原子化 Git 提交，将变更按功能模块分组提交
disable-model-invocation: false
---

# 原子化提交技能

执行原子化 Git 提交，将变更按功能模块分组提交。

## 执行流程

1. **收集变更文件**：获取所有修改和新增的文件
2. **分类变更**：按功能模块分组变更
3. **预检查**：
   - 编译验证 (`go build ./...`)
   - **i18n 双语检查** (`make check-i18n`) - **强制执行**
   - 硬编码文本检查 (`make check-i18n-hardcode`)
   - **代码风格检查** (`make check-lint`) - **强制执行**
4. **生成提交消息**：为每组变更生成规范的提交消息
5. **执行提交**：按顺序执行 git add 和 git commit

## 提交消息格式

```
<type>(<scope>): <subject>

<body>

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>
```

## 类型说明

| 类型       | 说明      | 示例            |
| ---------- | --------- | --------------- |
| `feat`     | 新功能    | 添加 BM25 搜索  |
| `fix`      | 缺陷修复  | 修复并发竞态    |
| `refactor` | 重构      | 移除 MySQL 支持 |
| `perf`     | 性能优化  | 优化查询缓存    |
| `docs`     | 文档      | 更新 API 文档   |
| `test`     | 测试      | 添加单元测试    |
| `chore`    | 构建/工具 | 更新依赖        |

## 作用域说明

| 作用域      | 说明             |
| ----------- | ---------------- |
| `store`     | 数据存储层       |
| `server`    | HTTP/gRPC 服务器 |
| `plugin`    | 插件系统         |
| `web`       | 前端应用         |
| `api`       | API 路由         |
| `db`        | 数据库实现       |
| `cache`     | 缓存层           |
| `runner`    | 后台任务         |
| `scheduler` | 调度服务         |

## 原子化原则

每个提交必须：
- ✅ 只包含一个功能或修复
- ✅ 能够独立编译通过
- ✅ 可以独立回滚
- ✅ 提交消息清晰描述变更

## 排除文件

不提交以下文件：
- `.claude/settings.local.json` (本地配置)
- `.env` (环境变量)
- `node_modules/`, `dist/`, `bin/` (构建产物)
- `*.log`, `*.tmp` (临时文件)

---

## i18n 双语强制规则 ⚠️

**所有前端 UI 文本必须同时提供英文和中文翻译。**

### 检查规则

变更涉及以下文件时，强制执行 i18n 检查：
- `web/src/locales/en.json`
- `web/src/locales/zh-Hans.json`

### 提交前验证

```bash
# 必须通过 i18n 检查才能提交
make check-i18n
```

### 添加新文本流程

1. 在 `en.json` 添加英文翻译
2. 在 `zh-Hans.json` 添加中文翻译
3. 确保 key 路径完全一致
4. 运行 `make check-i18n` 验证

### 禁止行为

```tsx
// ❌ 禁止：硬编码文本
<button>Save</button>

// ❌ 禁止：只有英文翻译
// en.json: { "save": "Save" }
// zh-Hans.json: { "save": "保存" }
// 但在代码中仍用硬编码

// ✅ 正确：使用 i18n hook
<button>{t("common.save")}</button>
```

### 拒绝条件

以下情况将**拒绝提交**：
1. `en.json` 有 key 但 `zh-Hans.json` 没有
2. `zh-Hans.json` 有 key 但 `en.json` 没有
3. 前端代码中存在硬编码用户可见文本

---

## 使用示例

```bash
# 执行原子化提交
/commit

# 带参数
/commit "feat(web): add schedule page"
```
