# 📋 上游功能逆向与追齐规范 (Upstream Feature Specs)

**分析时间**: `2026-01-21`
**分析范围**: `dc7ec8a8` -> `324f7959`

## 🎯 目标摘要
本次同步的核心目标是跟进上游在 **v0.26** 版本引入的两个重大破坏性变更：
1.  **用户角色架构重构**：移除 `HOST` 角色，合并至 `ADMIN`。
2.  **启动配置简化**：移除 `--mode` 参数，引入 `--demo` 标志，并清理相关环境变量逻辑。

同时，采纳关于 OOM 的重要修复和数据目录处理的优化，提升生产环境稳定性。

## 🚫 排除项 (Ignored)
*   **Go.mod 依赖升级**: 上游可能涉及大量内部依赖变动，本次暂不全量同步，仅同步代码中显式引用的部分。
*   **不稳定的迁移测试脚本**: `store/test/migrator_test.go` 变更巨大（-375 lines），暂不引入，以免破坏本地测试环境。

## 🗺️ 功能逆向与实现细则

### 1. 🛑 架构与破坏性变更 (Critical / Breaking)

#### 用户角色重构 (Migrate HOST to ADMIN)
- **来源**: `0f3c9a46` (refactor: migrate HOST roles to ADMIN)
- **必要性**: ⭐⭐⭐⭐⭐ (数据库模型/权限核心)
- **原理解析 (Reverse Engineering)**:
    - **本质**: 系统简化了权限模型，原本的 `HOST` (宿主) 角色被视作特殊的 `ADMIN`，现在彻底去除了 `HOST` 枚举，统一使用 `ADMIN`。
    - **关键变动**:
        - `proto/api/v1/user_service.proto`:删除 `Role.HOST`。
        - `store/migrator.go`: 数据迁移逻辑，需将表中所有 `HOST` 用户刷为 `ADMIN`。
        - SQL 迁移脚本: `migrage_host_to_admin.sql`。
- **本地移植规范**:
    - **数据层**: 必须复制 `store/migration/sqlite/0.26/03__alter_user_role.sql` 和 `04__migrate_host_to_admin.sql` (以及 Postgres/MySQL 对应脚本)。
    - **应用层**: 全局搜索代码中的 `Role_HOST`，将其逻辑替换为 `Role_ADMIN` 并移除相关 `if` 判断（如果逻辑是特有的，需要重新评估是否保留业务逻辑）。
    - **API 层**: 确保前端不再发送 `HOST` 角色类型，否则 protobuf 解析会失败。

#### 启动参数重构 (Remove mode, add demo)
- **来源**: `47ebb04d` (refactor: remove mode flag and introduce explicit demo flag)
- **必要性**: ⭐⭐⭐⭐⭐ (运维部署)
- **原理解析**:
    - **本质**: `mode` 字段（dev/prod/demo）被认为设计过度。现在由 `Demo` bool 字段显式控制演示模式，默认就是生产模式。
    - **关键变动**:
        - `cmd/memos/main.go`: 移除 `viper.SetDefault("mode", "dev")`，替换为 `viper.SetDefault("demo", false)`。
        - `internal/profile`: 结构体字段 `Mode` -> `Demo`。
- **本地移植规范**:
    - 修改 `main.go` 中的 Flag 定义。
    - 修改所有启动脚本 (`start-dev.sh`, `Makefile`)，将 `--mode dev` 替换为 `--demo`。
    - 检查本地是否还在使用环境变量 `MEMOS_MODE`，若有需废弃。

### 2. ⚡ 核心逻辑与修复 (High Priority)

#### OOM 防护 (Mmap size setting)
- **来源**: `05f31e45` (fix: add mmap size setting to database connection to prevent OOM errors)
- **必要性**: ⭐⭐⭐⭐ (生产稳定性)
- **逻辑分析**:
    - **原因**: SQLite 在大内存机器上默认 mmap 行为可能导致 Go 进程虚拟内存暴涨，引发 OOM Killer。
    - **修复**: 在 `store/db/sqlite/sqlite.go` 连接初始化时，显式设置 `pragma mmap_size`。
- **移植建议**:
    - 直接 Copy 该 commit 中对 `sqlite.go` 的修改。

#### 数据目录处理优化
- **来源**: `324f7959` (fix: improve default data directory handling)
- **必要性**: ⭐⭐⭐ (健壮性)
- **逻辑分析**:
    - **改进**: 优化了 `FromEnv` 或默认路径判断逻辑，处理了绝对路径转换的边缘情况。
- **移植建议**:
    - 对比 `internal/profile/profile.go`，重点同步 `checkDataDir` 函数的变动。

### 3. ✨ 有价值的新特性 (Features)
*(本次无重大新功能引入，主要为重构与修复)*

## ✅ 验证清单
- [ ] **构建验证**: `make build` 成功。
- [ ] **启动验证**: 使用 `--demo` 参数启动，确认日志无报错。
- [ ] **权限验证**: 登录原有 `HOST` 账号，确认已自动变为 `ADMIN` 且权限正常。
- [ ] **数据验证**: 检查数据库 `migration_history` 表，确认 `migrate_host_to_admin` 已执行。
