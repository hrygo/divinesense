# API V1 上帝类 (APIV1Service) 彻底拆分重构方案

> **状态**: 计划中
> **目标**: 将 `v1.go` 中超过 10 种指派与实现杂糅的巨块 `APIV1Service`，彻底拆分为符合单一职责原则的内聚性高的小结构体。
> **前提**: 阶段二和阶段三（早期）已经剥离了与包级别强相关的鉴权实用函数（如 `fetchCurrentUser` / `requireUserAccess`），并完成了 `schedule_service.go` 的重构，目前已经扫清了解耦合的技术屏障。

## 1. 核心问题扫描
当前 `APIV1Service` 在 `server/router/api/v1/v1.go` 充当了名副其实的"上帝结构体"。它直接或间接地承担了整个 V1 路由网关的逻辑：
* **结构体堆叠**：通过嵌套混入了诸如 `UnimplementedUserServiceServer`、`UnimplementedMemoServiceServer`、`UnimplementedAuthServiceServer` 等总计数十个服务的伪存根。
* **依赖腐化**：它的字段混合包含了所有服务所需的基建上下文，例如 `Secret`, `Store`, `Profile`, `MarkdownService`, `thumbnailSemaphore` 等等，这导致任何微小的服务都必须挂历在 `*APIV1Service` 上，无差别的获得了它并不需要的变量权限。
* **调用堆砌**：由于必须兼容并直接由 `v1pb.RegisterXXXServiceHandlerServer(ctx, gwMux, s)` 的网关方法来直接挂载 API 对象，它必须囊括实现所有功能接口。

## 2. 解决方案：职责切分的洋葱架构重构

参考 `ScheduleService` 与 `AIService` 成功的范例，我们将对现存的所有存量服务彻底实施结构体切割。

### 2.1 新的底层服务域类图设计
我们将创建以下单独的服务领域对象，移除 `APIV1Service` 对对应域服务的寄生依赖，直接构造它们：
1. **`UserService`**: 依赖 `Store`。
2. **`AuthService`**: 依赖 `Store`, `Secret`。
3. **`MemoService`**: 依赖 `Store`, `AIService`, `MarkdownService`。
4. **`AttachmentService`**: 依赖 `Store`, `Profile`, `thumbnailSemaphore`。
5. **`ShortcutService`**: 依赖 `Store`, `Profile`（为了识别 db driver 使用的 dialect filter）。
6. **`ActivityService`**: 依赖 `Store`。
7. **`IdentityProviderService`**: 依赖 `Store`。
8. **`InstanceService`**: 依赖 `Store`, `Profile`。
9. **`ChatAppService`**: 依赖 `Store`, `AIService`, `Secret`, `chatChannelRouter`, `chatAppStore`。

### 2.2 具体操作实施步骤

进行全流程覆盖切割时的详细任务流：

- **Step 1：定义服务 Struct 与构造函数**
  在服务的主文件（诸如 `user_service_crud.go`, `memo_service.go` 等）定义自身的结构体，并实现 `v1pb.UnimplementedXXXServer`。
  
- **Step 2：方法的平滑移花接木**
  将所有的 `func (s *APIV1Service) Foo(ctx... )` 签名中的 Receiver 大面积直接替换为对应的新设的服务的指针，例如：`func (s *UserService) ListUsers(...)`。

- **Step 3：替换 `fetchCurrentUser` 的封装依赖**
  通过已预先抽离的孤立函数 `fetchCurrentUser(ctx, store)` 以及 `requireUserAccess(ctx, store, id)`，将孤儿服务内对应的挂载项直接变更为对纯函数的依赖调用。
  
- **Step 4：重新组装上帝调度节点 (`v1.go`)**
  让 `APIV1Service` 蜕变为纯粹的 **Gateway 路由器**（Composition Root）。从其定义内擦除所有 `UnimplementedXXXX` 并在其内部成员中持有这些剥离服务：
  ```go
  type APIV1Service struct {
      UserService       *UserService
      MemoService       *MemoService
      AuthService       *AuthService
      // ... 等等
  }
  ```
  在 `NewAPIV1Service` 构建函数中，显式向他们注入单独定制的高内聚依赖组件。

- **Step 5：重接 GPRC / Connect 拦截器与路由节点**
  - **Gateway Node (`v1.go:RegisterGateway`)**：由于子业务对象已经实现了协议方法，将其由 `v1pb.RegisterXXXService(ctx, mux, s)` 转挂载为 `v1pb.RegisterXXXService(ctx, mux, s.XXXService)`。
  - **Connect Node (`connect_services.go`)**：由于现有的拦截网关 `ConnectServiceHandler` 单独包裹了 `APIV1Service`，更改它对应的透传代理，比如将旧有的：`return s.APIV1Service.ListUsers(ctx, req.Msg)` 变更为：`return s.APIV1Service.UserService.ListUsers(ctx, req.Msg)`。

## 3. 落地推进与验收标准
受限于工作量，本次拆分将彻底清洗所有的 gRPC / Connect 代理服务，确保 V1 领域彻底拥抱**领域独立架构 (Domain Independent Service Structures)** 原则。此架构能通过了严格的单元编译 `go test -c` 与代码规范清洗 `goimports` 即视为成功解耦。
