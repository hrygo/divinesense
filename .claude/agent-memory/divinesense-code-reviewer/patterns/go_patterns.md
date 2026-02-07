# Go 惯用模式

> DivineSense Go 代码的惯用写法和良好实践

---

## 命名规范

### 文件命名
```go
// ✅ 正确：snake_case.go
session_manager.go
intent_classifier.go
block_manager.go

// ❌ 错误：kebab-case 或 PascalCase
session-manager.go
IntentClassifier.go
```

### 包命名
```go
// ✅ 简单小写单词
package agent
package router
package context

// ❌ 下划线或混合大小写
package ai_agent
package aiRouter
```

---

## 错误处理

### 标准错误检查
```go
// ✅ 始终检查错误
rows, err := db.Query(ctx, query)
if err != nil {
    return fmt.Errorf("failed to query: %w", err)
}
defer rows.Close()

// ❌ 忽略错误
rows, err := db.Query(ctx, query)
// 继续使用 rows 而不检查 err
```

### 错误包装
```go
// ✅ 使用 %w 保留错误链
return fmt.Errorf("failed to create block: %w", err)

// ❌ 使用 %v 断开错误链
return fmt.Errorf("failed to create block: %v", err)
```

### 事务回滚
```go
// ✅ 正确处理嵌套事务
if err := fn(tx); err != nil {
    if isNewTx {
        if rbErr := tx.Rollback(); rbErr != nil {
            return fmt.Errorf("original: %w (rollback: %v)", err, rbErr)
        }
    }
    return err
}
```

---

## 日志规范

### 结构化日志
```go
// ✅ 使用 log/slog 结构化日志
slog.Info("AI chat started",
    "agent_type", req.AgentType,
    "user_id", req.UserID,
    "mode", req.Mode,
)

// ❌ 使用 fmt.Printf 或 log.Printf
fmt.Printf("AI chat started: %+v\n", req)
```

### 日志级别
```go
slog.Debug("详细调试信息")     // 开发环境
slog.Info("常规业务操作")       // 重要事件
slog.Warn("可恢复的异常")       // 需要注意但不影响运行
slog.Error("需要关注的错误")    // 错误已处理但需要跟踪
```

---

## Context 使用

```go
// ✅ 传递带超时的 context
ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
defer cancel()

// ❌ 使用 Background()（除非确实必要）
ctx := context.Background()
```

---

## Go Embed 兼容

```go
// ❌ 以下划线开头的文件会被 embed 忽略
_lodash_internal.js
_utils.go

// ✅ 修改配置避免生成以下划线开头的文件
// vite.config.mts
manualChunks(id) {
    if (id.includes("lodash-es") || id.includes("/_base")) {
        return "lodash-vendor";  // 生成 lodash-vendor-xxx.js
    }
}
```

---

## 数据库操作

```go
// ✅ 使用参数化查询
rows, err := db.Query(ctx, "SELECT * FROM user WHERE id = $1", userID)

// ❌ 字符串拼接（SQL 注入风险）
query := fmt.Sprintf("SELECT * FROM user WHERE id = %s", userID)
```

---

## 常见模式

### 接口模式
```go
// store/ 接口定义
type BlockStore interface {
    CreateBlock(ctx context.Context, block *AIBlock) (*AIBlock, error)
    GetBlock(ctx context.Context, id int64) (*AIBlock, error)
}

// 具体实现
type postgresBlockStore struct {
    db *sql.DB
}
```

### 工厂模式
```go
// ai/agent/ 工厂函数
func NewParrotAgent(
    agentType ParrotAgentType,
    config *AgentConfig,
    deps *AgentDeps,
) (ParrotAgent, error) {
    switch agentType {
    case ParrotAgentType.MEMO:
        return NewMemoParrot(config, deps), nil
    case ParrotAgentType.GEEK:
        return NewGeekParrot(config, deps), nil
    default:
        return nil, fmt.Errorf("unknown agent type: %s", agentType)
    }
}
```
