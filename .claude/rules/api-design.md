# API 设计规范

## 核心原则
1. **Protocol First**：先修改 `.proto`，再生成代码
2. **版本化**：`/api/v1/`、`/api/v2/`
3. **向后兼容**：新版本不破坏旧客户端

## Connect RPC
- 定义：`proto/api/v1/xxx_service.proto`
- 错误：通过 Connect metadata 传递

## RESTful
| 操作 | 方法 | URL |
|:-----|:-----|:-----|
| 列表 | GET | `/api/v1/resources` |
| 详情 | GET | `/api/v1/resources/{id}` |
| 创建 | POST | `/api/v1/resources` |
| 更新 | PUT/PATCH | `/api/v1/resources/{id}` |
| 删除 | DELETE | `/api/v1/resources/{id}` |

## 命名
- 复数：`/api/v1/users`
- kebab-case：`/api/v1/schedule-events`
