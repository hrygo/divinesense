# API V1 优化重构执行方案 (Optimized Plan)

> **状态**: 阶段 1-3 已完成
> **分支**: `refactor/issue-268-api-v1-solid`
> **目标**: 针对 `server/router/api/v1` 的技术债务进行"先破除巨石，后精确打击"的高效重构。
> **策略**: 激进拆分，稳妥去重，重构上帝类。
> **最后更新**: 2026-02-20

---

## 阶段一：破除巨石 (高优激进拆分) ✅

鉴于当前无并发代码冲突，优先将巨石文件 `user_service.go` (1400+ 行) 根据业务领域进行**物理隔离**。

*   **已完成的物理结构拆分**：
    *   `user_service_crud.go`: 用户增删改查 (`GetUser`, `CreateUser`, `UpdateUser`, `DeleteUser`, `ListUsers`)
    *   `user_service_settings.go`: 用户设置 (`GetUserSetting`, `UpdateUserSetting`, `ListUserSettings`)
    *   `user_service_auth.go`: 个人访问令牌 PAT (`ListPersonalAccessTokens`, `Create...`, `Delete...`)
    *   `user_service_webhook.go`: Webhooks
    *   `user_service_notification.go`: 通知
    *   `user_service_stats.go`: 统计
    *   `user_service_converter.go`: Proto 转换辅助函数

---

## 阶段二：DRY 去重与标准化提取 ✅

*   **2.1 统一权限守卫 (`requireUserAccess`)** ✅：
    *   `permissions.go` — 提取为独立包级函数 `fetchCurrentUser(ctx, store)` 和 `requireUserAccess(ctx, store, userID)`
    *   保留 `*APIV1Service` 上的薄包装方法以保持向后兼容
    *   已在 `user_service_*.go` 和 `memo_service.go` 中替换
*   **2.2 AI 服务可用性检查 (`requireAI`)** ✅：
    *   `permissions.go` — `ConnectServiceHandler.requireAI()` 方法
    *   已替换 `connect_handler.go` 中 **35+ 处** 冗余的 `if s.AIService == nil` 检查
*   **2.3 资源名称解析** ⚠️ (部分完成)：
    *   `resource_name.go` 已存在，但 `memo_service.go` 中仍有部分 `strings.Split` 硬编码未全部清理 → 作为后续小任务

---

## 阶段三：结构健康与 OCP 治理 ✅

*   **3.1 Schedule 更新重构 (Field Mapper Pattern)** ✅：
    *   `schedule_service.go` — `scheduleFieldMappers` map 替代了两大块重复的 switch/if-else 硬编码
    *   新增字段只需在 map 添加一行，完全符合 OCP
*   **3.2 APIV1Service 上帝类彻底解耦拆分 (God Class Elimination)** ✅：
    *   **问题**: `APIV1Service` 寄生了 V1 下的十余种服务，其体积庞大，上下文依赖杂乱。
    *   **解决**: 将所有的服务实现转移到分别封装的领域结构体 (`UserService`, `MemoService`, `AuthService`, `AttachmentService` 等) 中，并为每个结构体内部嵌入各自的按需依赖，不让它持有不需要的多余权限。
    *   **连线**: `APIV1Service` 已变为纯粹的依赖倒置和路由透传对象层 (Composition Root)，不再亲自实现任何服务。并且在 `v1pb.RegisterXXXService(ctx, mux, s.XXXService)` 和 Connect 包裹层 `s.APIV1Service.XXXService.Method` 进行透明的接口挂载。

---

## 验证结果
- ✅ `go test ./server/router/api/v1/...` — 全部通过
- ✅ `go vet ./...` — 无警告
- ✅ `go fmt` — 无格式问题
- ✅ pre-commit hooks — 全部通过

## 提交历史
1. `refactor: split user_service.go into domain-specific files` — 阶段一
2. `refactor: extract requireUserAccess and requireAI to eliminate DRY violations` — 阶段二
3. `refactor: apply Field Mapper pattern to ScheduleService.UpdateSchedule` — 阶段三 3.1
4. `refactor: decouple fetchCurrentUser and requireUserAccess from APIV1Service god struct` — 阶段三 3.2
