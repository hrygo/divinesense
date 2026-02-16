# Agent Tools (`ai/agents/tools`)

`tools` 包包含了所有 Agent 可使用的具体工具实现。

## 架构设计

```mermaid
classDiagram
    class ToolWithSchema {
        <<interface>>
        +Name() string
        +Description() string
        +Parameters() JSONSchema
        +Execute(ctx, args) string
    }
    class MemoSearchTool {
        +Execute()
    }
    class ScheduleAddTool {
        +Execute()
    }
    class FindFreeTimeTool {
        +Execute()
    }
    
    ToolWithSchema <|.. MemoSearchTool
    ToolWithSchema <|.. ScheduleAddTool
    ToolWithSchema <|.. FindFreeTimeTool
```


## 工具列表

### Memo Related
*   **`memo_search`**: 语义化笔记搜索。支持时间过滤、标签过滤。

### 日程相关 (Schedule)
*   **`schedule_add`**: 创建日程。
    *   *输入*: `{"title": "...", "start_time": "..."}`
    *   *特点*: 支持 ISO8601 时间格式，需先调用 `schedule_query` 查冲突。
*   **`schedule_query`**: 查询日程。
    *   *输入*: `{"start_time": "...", "end_time": "..."}`
*   **`schedule_update`**: 更新日程状态或详情。
    *   *输入*: `{"id": 123, "title": "..."}`
*   **`schedule_delete`**: 删除日程。
    *   *输入*: `{"id": 123}`
*   **`find_free_time`**: 查找空闲时间段。
    *   *输入*: `{"date": "YYYY-MM-DD"}`

## 工具开发规范

所有工具必须实现 `ToolWithSchema` 接口：

```go
type ToolWithSchema interface {
    Name() string             // 工具唯一标识 (snake_case)
    Description() string      // 给 LLM 看的功能描述
    Parameters() string       // JSON Schema 格式的参数定义
    Execute(ctx, args) (string, error) // 执行逻辑
}
```

## 注册机制
使用 `registry` 包进行注册，以便 `UniversalParrot` 根据配置动态加载。
