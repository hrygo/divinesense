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
*   **3.2 APIV1Service 上帝类解耦** ✅ (已解耦基础设施)：
    *   `fetchCurrentUser` 和 `requireUserAccess` 已提取为 standalone 函数
    *   修复了 `requireAI` 的递归调用 bug
    *   `AIService` 和 `ScheduleService` 已作为独立 struct 存在

### 3.2 的架构约束说明

`APIV1Service` 作为"上帝结构体"的存在受限于 **gRPC-Gateway 注册机制**：
```go
// gRPC-Gateway 要求 handler 必须实现完整的 proto 接口
v1pb.RegisterUserServiceHandlerServer(ctx, gwMux, s)  // s 必须实现 UserServiceServer
v1pb.RegisterMemoServiceHandlerServer(ctx, gwMux, s)  // s 必须实现 MemoServiceServer
```

完全消灭 `APIV1Service` 需要：
1. 为每个域创建独立 struct (如 `UserService`, `MemoService`)
2. 将 `UnimplementedXxxServer` 移到各域 struct
3. 更新所有 gRPC-Gateway 注册为各域 struct
4. 更新 `ConnectServiceHandler` 的委托链

**当前策略**: 已将共享工具函数解耦为 standalone，使得未来按需提取各域服务成为可能，但不在本 PR 中做全量提取，避免爆炸半径过大。

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
