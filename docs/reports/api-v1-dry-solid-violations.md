# server/router/api/v1 包 DRY/SOLID 违约扫描报告

> 扫描日期: 2026-02-18
> 扫描范围: server/router/api/v1/**/*.go

---

## 一、DRY 违约 (Don't Repeat Yourself)

### 1.1 AI 服务可用性检查重复

**位置**: `connect_handler.go`

**问题描述**: 多个 Connect wrapper 方法中存在重复的 AI 服务检查逻辑：

```go
// 重复出现 30+ 次的模式
if s.AIService == nil || !s.AIService.IsEnabled() {
    return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("AI features are disabled"))
}
```

**涉及方法** (部分列表):
- `SuggestTags` (L100-103)
- `Format` (L111-114)
- `Summary` (L122-125)
- `SemanticSearch` (L133-136)
- `DetectDuplicates` (L530-533)
- `MergeMemos` (L541-544)
- `LinkMemos` (L552-555)
- `GetKnowledgeGraph` (L563-566)
- `GetDueReviews` (L574-577)
- `RecordReview` (L585-588)
- `RecordRouterFeedback` (L596-599)
- `GetReviewStats` (L607-610)
- `ListBlocks` (L630-633)
- `GetBlock` (L641-644)
- `CreateBlock` (L652-655)
- `UpdateBlock` (L663-666)
- `DeleteBlock` (L674-677)
- `AppendUserInput` (L685-688)
- `AppendEvent` (L696-699)
- `ForkBlock` (L709-712)
- `ListBlockBranches` (L720-723)
- `SwitchBranch` (L731-734)
- `DeleteBranch` (L742-745)

**建议修复**: 创建辅助函数或中间件

```go
func (s *ConnectServiceHandler) requireAI(ctx context.Context) error {
    if s.AIService == nil || !s.AIService.IsEnabled() {
        return connect.NewError(connect.CodeUnavailable, fmt.Errorf("AI features are disabled"))
    }
    return nil
}

// 或使用泛型 wrapper
func wrapAIResponse[T, R any](s *ConnectServiceHandler, ctx context.Context, req *connect.Request[T], fn func(context.Context, *T) (*R, error)) (*connect.Response[R], error) {
    if s.AIService == nil || !s.AIService.IsEnabled() {
        return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("AI features are disabled"))
    }
    resp, err := fn(ctx, req.Msg)
    if err != nil {
        return nil, convertGRPCError(err)
    }
    return connect.NewResponse(resp), nil
}
```

---

### 1.2 用户获取模式重复

**位置**: 多个文件

**问题描述**: `fetchCurrentUser` 模式在多个服务中重复：

| 文件 | 方法 | 位置 |
|------|------|------|
| `auth_service.go` | `fetchCurrentUser` | L403-418 |
| `ai_service.go` | `getCurrentUser` | L357-373 |

两个方法功能几乎相同，仅返回值处理略有不同。

**建议修复**: 统一到 `common.go` 中，使用单一实现

---

### 1.3 资源名称解析模式重复

**位置**: `user_service.go`

**问题描述**: 多个资源名称解析函数存在相似模式：

```go
// 模式重复
func ExtractUserIDFromName(name string) (int32, error)          // 通用
func ExtractUserIDAndSettingKeyFromName(name string) (...)      // L969
func parseUserWebhookName(name string) (string, int32, error)   // L878
func ExtractNotificationIDFromName(name string) (int32, error)  // L1428
```

所有这些函数都执行类似的 `strings.Split` 和验证逻辑。

**建议修复**: 创建通用的资源名称解析器

```go
type ResourceName struct {
    Parent   string
    Resource string
    ID       string
}

func ParseResourceName(name string, expectedPattern string) (*ResourceName, error)
```

---

### 1.4 权限检查模式重复

**位置**: `user_service.go`, `memo_service.go`

**问题描述**: 权限检查代码在多个方法中重复：

```go
// user_service.go 中重复 10+ 次的模式
currentUser, err := s.fetchCurrentUser(ctx)
if err != nil {
    return nil, status.Errorf(codes.Internal, "failed to get current user: %v", err)
}
if currentUser == nil {
    return nil, status.Errorf(codes.Unauthenticated, "user not authenticated")
}
if currentUser.ID != userID && currentUser.Role != store.RoleHost && currentUser.Role != store.RoleAdmin {
    return nil, status.Errorf(codes.PermissionDenied, "permission denied")
}
```

**涉及方法**:
- `ListUserWebhooks` (L685-694)
- `CreateUserWebhook` (L717-726)
- `UpdateUserWebhook` (L757-766)
- `DeleteUserWebhook` (L829-838)
- `ListPersonalAccessTokens` (L542-548)
- `CreatePersonalAccessToken` (L595-601)
- `DeletePersonalAccessToken` (L664-670)

**建议修复**: 创建权限检查辅助函数

```go
func (s *APIV1Service) requireUserOrAdmin(ctx context.Context, targetUserID int32) (*store.User, error) {
    currentUser, err := s.fetchCurrentUser(ctx)
    if err != nil {
        return nil, status.Errorf(codes.Internal, "failed to get current user: %v", err)
    }
    if currentUser == nil {
        return nil, status.Errorf(codes.Unauthenticated, "user not authenticated")
    }
    if currentUser.ID != targetUserID && !isSuperUser(currentUser) {
        return nil, status.Errorf(codes.PermissionDenied, "permission denied")
    }
    return currentUser, nil
}
```

---

### 1.5 Schedule Reminders 转换重复

**位置**: `schedule_service.go`

**问题描述**: reminders 转换逻辑在多处重复：

- `scheduleToStore` (L142-168): pb.Reminders → JSON
- `UpdateSchedule` with update_mask (L459-475): 同样逻辑
- `UpdateSchedule` without update_mask (L510-524): 同样逻辑

```go
// 重复的 reminders 转换代码
if len(reminders) > 0 {
    remindersList := make([]*v1pb.Reminder, 0, len(reminders))
    for _, r := range reminders {
        remindersList = append(remindersList, &v1pb.Reminder{
            Type:  r.Type,
            Value: r.Value,
            Unit:  r.Unit,
        })
    }
    remindersStr, err := aischedule.MarshalReminders(remindersList)
    // ...
}
```

**建议修复**: 提取为独立函数

```go
func convertRemindersToJSON(reminders []*v1pb.Reminder) (string, error)
```

---

### 1.6 Connect 错误转换重复

**位置**: `connect_handler.go`

**问题描述**: 几乎所有 wrapper 方法都有相同的错误转换模式：

```go
resp, err := s.ScheduleService.CreateSchedule(ctx, req.Msg)
if err != nil {
    return nil, convertGRPCError(err)
}
return connect.NewResponse(resp), nil
```

**建议修复**: 创建泛型 wrapper 或使用中间件

---

## 二、SOLID 违约

### 2.1 单一职责原则 (SRP) 违约 - APIV1Service

**位置**: `v1.go`, 各 service 文件

**问题描述**: `APIV1Service` 结构体承担了过多职责：

```go
type APIV1Service struct {
    v1pb.UnimplementedAIServiceServer      // AI 服务
    v1pb.UnimplementedUserServiceServer    // 用户服务
    v1pb.UnimplementedScheduleServiceServer // 日程服务
    v1pb.UnimplementedAttachmentServiceServer
    v1pb.UnimplementedShortcutServiceServer
    v1pb.UnimplementedActivityServiceServer
    v1pb.UnimplementedIdentityProviderServiceServer
    v1pb.UnimplementedAuthServiceServer
    v1pb.UnimplementedInstanceServiceServer
    v1pb.UnimplementedMemoServiceServer
    v1pb.UnimplementedChatAppServiceServer
    // ... 更多字段
}
```

单个服务实现了 **11 个** gRPC 服务接口，违反 SRP。

**影响**:
- 代码难以维护
- 测试困难
- 部署灵活性低

**建议修复**: 拆分为独立的服务结构体，`APIV1Service` 仅作为聚合器

---

### 2.2 单一职责原则 (SRP) 违约 - user_service.go

**位置**: `user_service.go`

**问题描述**: `user_service.go` 文件包含多个不相关的功能域：

| 功能域 | 行数 |
|--------|------|
| 用户 CRUD | L32-323 |
| 用户设置 | L325-518 |
| Personal Access Tokens | L520-677 |
| Webhooks | L679-865 |
| 通知 | L1233-1426 |
| 转换器函数 | L904-1082 |

**文件总行数**: 1444 行

**建议修复**: 拆分为独立文件：
- `user_service_crud.go`
- `user_service_settings.go`
- `user_service_tokens.go`
- `user_service_webhooks.go`
- `user_service_notifications.go`
- `user_service_converter.go`

---

### 2.3 开闭原则 (OCP) 违约 - ScheduleService UpdateMask

**位置**: `schedule_service.go` L430-525

**问题描述**: `UpdateSchedule` 方法中，字段更新逻辑硬编码，新增字段需要修改方法：

```go
switch path {
case "title":
    update.Title = &req.Schedule.Title
case "description":
    update.Description = &req.Schedule.Description
case "location":
    update.Location = &req.Schedule.Location
// ... 更多 case
}
```

且存在两套逻辑（with/without update_mask），违反 DRY 和 OCP。

**建议修复**: 使用反射或字段映射器模式

```go
var fieldMappers = map[string]func(*v1pb.Schedule, *store.UpdateSchedule){
    "title": func(s *v1pb.Schedule, u *store.UpdateSchedule) {
        u.Title = &s.Title
    },
    // ...
}
```

---

### 2.4 接口隔离原则 (ISP) 违约 - AIService

**位置**: `ai_service.go`

**问题描述**: `AIService` 结构体包含大量字段，但并非所有客户端都需要所有功能：

```go
type AIService struct {
    v1pb.UnimplementedAIServiceServer
    RerankerService          pluginai.RerankerService
    EmbeddingService         pluginai.EmbeddingService
    LLMService               pluginai.LLMService
    IntentLLMService         pluginai.LLMService
    conversationService      *aichat.ConversationService
    AdaptiveRetriever        *retrieval.AdaptiveRetriever
    IntentClassifierConfig   *pluginai.IntentClassifierConfig
    UniversalParrotConfig    *pluginai.UniversalParrotConfig
    agentFactory             *aichat.AgentFactory
    routerService            *routing.Service
    chatEventBus             *aichat.EventBus
    Store                    *store.Store
    contextBuilder           *aichat.ContextBuilder
    conversationSummarizer   *aichat.ConversationSummarizer
    TitleGenerator           *pluginai.TitleGenerator
    EmbeddingModel           string
    persister                *aistats.Persister
    enrichmentTrigger        *enrichment.Trigger
    // ... 7 个互斥锁
}
```

**影响**: 测试时需要 mock 大量不相关的依赖

**建议修复**: 拆分为功能接口

```go
type EmbeddingCapable interface {
    IsEnabled() bool
    Embed(ctx context.Context, text string) ([]float32, error)
}

type ChatCapable interface {
    IsLLMEnabled() bool
    Chat(req *v1pb.ChatRequest, stream v1pb.AIService_ChatServer) error
}
```

---

### 2.5 依赖倒置原则 (DIP) 违约 - 直接依赖具体实现

**位置**: `ai_service.go` L133-201

**问题描述**: `getRouterService` 方法直接创建具体实现：

```go
func (s *AIService) getRouterService() *routing.Service {
    // 直接依赖 dbpostgres.DB
    if db, ok := driver.(*dbpostgres.DB); ok {
        return routing.NewPostgresWeightStorage(db)
    }
    // 回退到内存实现
    return routing.NewInMemoryWeightStorage()
}
```

**影响**: 难以测试和替换实现

**建议修复**: 依赖注入

```go
type AIService struct {
    weightStorage routing.RouterWeightStorage // 接口类型
    // ...
}
```

---

### 2.6 单一职责原则 (SRP) 违约 - connect_handler.go

**位置**: `connect_handler.go`

**问题描述**: `ConnectServiceHandler` 包含与 Connect 协议无关的业务逻辑：

```go
// 这些方法包含业务逻辑，不应该在 handler 中
func (s *ConnectServiceHandler) GetParrotSelfCognition(...)  // L311-318
func (s *ConnectServiceHandler) ListParrots(...)             // L321-343
func getParrotSelfCognition(agentType v1pb.AgentType)        // L346-409
func getParrotNameByAgentType(agentType v1pb.AgentType)      // L412-425
```

**建议修复**: 将 parrot 相关逻辑移至 `ai_service.go` 或专门的 `parrot_service.go`

---

## 三、总结统计

| 类型 | 数量 | 严重程度 |
|------|------|----------|
| DRY 违约 | 6 | 中 |
| SRP 违约 | 4 | 高 |
| OCP 违约 | 1 | 中 |
| ISP 违约 | 1 | 中 |
| DIP 违约 | 1 | 低 |
| **总计** | **13** | - |

---

## 四、优先修复建议

1. **高优先级** (SRP 违约):
   - 拆分 `user_service.go` 为多个文件
   - 考虑将 `APIV1Service` 拆分为独立服务

2. **中优先级** (DRY 违约):
   - 创建 AI 服务检查辅助函数
   - 创建权限检查辅助函数
   - 统一资源名称解析逻辑

3. **低优先级** (其他):
   - 重构 `ScheduleService.UpdateSchedule`
   - 拆分 `AIService` 接口

---

*报告生成时间: 2026-02-18*
