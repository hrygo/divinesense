# P1-B002: 错误恢复机制

> **状态**: ✅ 已完成  
> **优先级**: P0 (核心)  
> **投入**: 1 人天  
> **负责团队**: 团队 B  
> **Sprint**: Sprint 1

---

## 1. 目标与背景

### 1.1 核心目标

实现 Agent 执行层的错误自动恢复机制，减少用户重新输入的次数。

### 1.2 用户价值

- 错误自动恢复，无需重新输入
- 友好的错误提示

### 1.3 技术价值

- 统一错误处理逻辑
- 提升系统健壮性

---

## 2. 依赖关系

### 2.1 前置依赖

- 无

### 2.2 并行依赖

- P1-B001: 工具可靠性增强（可并行）

### 2.3 后续依赖

- 所有 Agent 执行

---

## 3. 功能设计

### 3.1 架构图

```
┌─────────────────────────────────────────────────────────────────┐
│                    错误恢复机制                                   │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  Agent 执行                                                      │
│      │                                                          │
│      ▼                                                          │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │              执行结果                                     │   │
│  │              │                                            │   │
│  │     成功 ◄───┴───► 失败                                   │   │
│  │      │              │                                     │   │
│  │      ▼              ▼                                     │   │
│  │   返回结果      ┌───────────────────────────────────┐     │   │
│  │                 │  tryRecover(err, input)           │     │   │
│  │                 │                                   │     │   │
│  │                 │  可恢复?                          │     │   │
│  │                 │  ├─ Yes → 修正输入 → 重试         │     │   │
│  │                 │  └─ No  → 友好错误提示            │     │   │
│  │                 └───────────────────────────────────┘     │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                 │
│  可恢复错误类型:                                                │
│  • ErrInvalidTimeFormat → 重新解析时间                         │
│  • ErrToolNotFound → 重新路由                                  │
│  • ErrParseError → 提取关键信息重试                            │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### 3.2 核心流程

1. **执行**: Agent 执行任务
2. **错误检测**: 捕获执行错误
3. **恢复尝试**: 尝试自动修正
4. **友好提示**: 无法恢复时返回友好提示

### 3.3 关键决策

| 决策点 | 方案 A | 方案 B | 选择 | 理由 |
|:---|:---|:---|:---:|:---|
| 重试次数 | 1次 | 无限 | A | 避免死循环 |
| 错误提示 | 技术错误 | 用户友好 | B | 更好的体验 |

---

## 4. 技术实现

### 4.1 关键代码路径

| 文件路径 | 职责 |
|:---|:---|
| `plugin/ai/agent/recovery.go` | 错误恢复逻辑 |

### 4.2 核心实现

```go
// plugin/ai/agent/recovery.go

type ErrorRecovery struct {
    timeNormalizer *time.TimeNormalizer
    maxRetries     int
}

func NewErrorRecovery(timeNormalizer *time.TimeNormalizer) *ErrorRecovery {
    return &ErrorRecovery{
        timeNormalizer: timeNormalizer,
        maxRetries:     1,
    }
}

// ExecuteWithRecovery 带自动恢复的执行
func (r *ErrorRecovery) ExecuteWithRecovery(
    ctx context.Context,
    executor func(context.Context, string) (string, error),
    input string,
) (string, error) {
    result, err := executor(ctx, input)
    if err == nil {
        return result, nil
    }
    
    // 尝试恢复
    if recovered, fixedInput := r.tryRecover(err, input); recovered {
        result, err = executor(ctx, fixedInput)
        if err == nil {
            return result, nil
        }
    }
    
    // 返回友好错误
    return r.formatUserFriendlyError(err), nil
}

// tryRecover 尝试自动恢复
func (r *ErrorRecovery) tryRecover(err error, input string) (bool, string) {
    switch {
    case errors.Is(err, ErrInvalidTimeFormat):
        // 时间格式错误 → 尝试重新解析
        if normalized := r.normalizeTimeInInput(input); normalized != input {
            return true, normalized
        }
        
    case errors.Is(err, ErrToolNotFound):
        // 工具不存在 → 不修改，让路由重新选择
        return true, input
        
    case errors.Is(err, ErrParseError):
        // 解析错误 → 尝试提取关键信息
        if simplified := r.simplifyInput(input); simplified != input {
            return true, simplified
        }
    }
    
    return false, ""
}

// normalizeTimeInInput 尝试标准化输入中的时间
func (r *ErrorRecovery) normalizeTimeInInput(input string) string {
    // 提取时间表达式并尝试标准化
    // 例如: "明天3点" → "2026-01-28T15:00:00"
    return r.timeNormalizer.NormalizeInText(input)
}

// formatUserFriendlyError 格式化用户友好的错误信息
func (r *ErrorRecovery) formatUserFriendlyError(err error) string {
    switch {
    case errors.Is(err, ErrInvalidTimeFormat):
        return "抱歉，我没能理解时间。请尝试更明确的表达，比如"明天下午3点""
        
    case errors.Is(err, ErrToolNotFound):
        return "抱歉，我暂时无法处理这个请求"
        
    case errors.Is(err, ErrNetworkError):
        return "网络连接出现问题，请稍后重试"
        
    case errors.Is(err, context.DeadlineExceeded):
        return "处理时间较长，请稍后重试"
        
    default:
        return "抱歉，处理遇到问题，请稍后重试"
    }
}
```

---

## 5. 交付物清单

### 5.1 代码文件

- [ ] `plugin/ai/agent/recovery.go` - 错误恢复逻辑
- [ ] `plugin/ai/agent/errors.go` - 错误定义

### 5.2 测试文件

- [ ] `plugin/ai/agent/recovery_test.go`

---

## 6. 测试验收

### 6.1 功能测试

| 场景 | 输入 | 预期输出 |
|:---|:---|:---|
| 时间错误恢复 | 错误时间格式 | 自动修正后重试 |
| 不可恢复错误 | 未知错误 | 友好提示 |
| 正常执行 | 正确输入 | 正常返回 |

### 6.2 性能验收

| 指标 | 目标值 | 测试方法 |
|:---|:---|:---|
| 恢复开销 | < 50ms | 单元测试 |

---

## 7. ROI 分析

| 维度 | 值 |
|:---|:---|
| 开发投入 | 1 人天 |
| 预期收益 | 用户重试率 -50% |
| 风险评估 | 低 |
| 回报周期 | Phase 1 结束 |

---

## 8. 实施计划

### 8.1 时间表

| 阶段 | 时间 | 任务 |
|:---|:---|:---|
| Day 1 | 1人天 | 恢复逻辑 + 测试 |

### 8.2 检查点

- [ ] Day 1: 时间错误可自动恢复

---

## 附录

### A. 参考资料

- [日程路线图 - 错误恢复](../../research/schedule-roadmap.md)

### B. 变更记录

| 日期 | 版本 | 变更内容 | 作者 |
|:---|:---|:---|:---|
| 2026-01-27 | v1.0 | 初始版本 | - |
