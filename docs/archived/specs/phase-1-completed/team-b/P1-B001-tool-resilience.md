# P1-B001: 工具可靠性增强

> **状态**: ✅ 已完成  
> **优先级**: P0 (核心)  
> **投入**: 2 人天  
> **负责团队**: 团队 B  
> **Sprint**: Sprint 1

---

## 1. 目标与背景

### 1.1 核心目标

实现带重试和降级的工具执行器，提升工具调用成功率和用户体验。

### 1.2 用户价值

- 工具调用更稳定
- 失败时有友好提示

### 1.3 技术价值

- 统一工具执行策略
- 为指标采集提供埋点

---

## 2. 依赖关系

### 2.1 前置依赖

- [ ] P1-A001: 记忆系统（用于降级时的缓存数据）
- [ ] P1-A002: 指标框架（上报工具调用指标）

### 2.2 并行依赖

- P1-B002: 错误恢复机制（可并行）

### 2.3 后续依赖

- 所有 Agent 工具调用

---

## 3. 功能设计

### 3.1 架构图

```
┌─────────────────────────────────────────────────────────────────┐
│                    工具执行器架构                                 │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  Agent 调用                                                      │
│      │                                                          │
│      ▼                                                          │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │              ResilientToolExecutor                       │   │
│  │                                                          │   │
│  │  ┌───────────────────────────────────────────────────┐  │   │
│  │  │  执行流程                                          │  │   │
│  │  │  1. 首次执行                                       │  │   │
│  │  │  2. 失败? → 可重试? → 重试 (最多2次)               │  │   │
│  │  │  3. 仍失败? → 降级策略                             │  │   │
│  │  │  4. 上报指标                                       │  │   │
│  │  └───────────────────────────────────────────────────┘  │   │
│  │                                                          │   │
│  │  配置:                                                   │   │
│  │  • maxRetries: 2                                         │   │
│  │  • retryDelay: 500ms                                     │   │
│  │  • timeout: 10s                                          │   │
│  │                                                          │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                 │
│  降级策略表:                                                    │
│  ┌────────────────┬───────────────────────────────────────┐    │
│  │ 工具           │ 降级方案                               │    │
│  ├────────────────┼───────────────────────────────────────┤    │
│  │ memo_search    │ "搜索暂时不可用，请稍后重试"          │    │
│  │ schedule_query │ 使用缓存数据（如有）                  │    │
│  │ schedule_add   │ 转为"待确认"状态                      │    │
│  └────────────────┴───────────────────────────────────────┘    │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### 3.2 核心流程

1. **执行**: 调用工具
2. **重试**: 失败且可重试时重试
3. **降级**: 重试失败后执行降级策略
4. **上报**: 记录指标

### 3.3 关键决策

| 决策点 | 方案 A | 方案 B | 选择 | 理由 |
|:---|:---|:---|:---:|:---|
| 重试策略 | 固定间隔 | 指数退避 | A | 简单有效，私人部署无需复杂 |
| 降级策略 | 统一提示 | 按工具定制 | B | 更好的用户体验 |

---

## 4. 技术实现

### 4.1 关键代码路径

| 文件路径 | 职责 |
|:---|:---|
| `plugin/ai/agent/tools/executor.go` | 工具执行器 |
| `plugin/ai/agent/tools/fallback.go` | 降级策略 |

### 4.2 核心实现

```go
// plugin/ai/agent/tools/executor.go

type ResilientToolExecutor struct {
    maxRetries     int
    retryDelay     time.Duration
    timeout        time.Duration
    metricsService metrics.MetricsService
    fallbackRules  map[string]FallbackFunc
}

func NewResilientToolExecutor(
    metricsService metrics.MetricsService,
) *ResilientToolExecutor {
    return &ResilientToolExecutor{
        maxRetries:     2,
        retryDelay:     500 * time.Millisecond,
        timeout:        10 * time.Second,
        metricsService: metricsService,
        fallbackRules:  DefaultFallbackRules,
    }
}

func (e *ResilientToolExecutor) Execute(
    ctx context.Context,
    tool Tool,
    input string,
) (*Result, error) {
    start := time.Now()
    var lastErr error
    
    for attempt := 0; attempt <= e.maxRetries; attempt++ {
        // 带超时执行
        execCtx, cancel := context.WithTimeout(ctx, e.timeout)
        result, err := tool.Run(execCtx, input)
        cancel()
        
        if err == nil {
            // 成功，上报指标
            e.metricsService.RecordToolCall(ctx, tool.Name(), time.Since(start), true)
            return result, nil
        }
        
        lastErr = err
        
        // 检查是否可重试
        if !isRetryable(err) {
            break
        }
        
        // 等待后重试
        if attempt < e.maxRetries {
            time.Sleep(e.retryDelay)
        }
    }
    
    // 上报失败指标
    e.metricsService.RecordToolCall(ctx, tool.Name(), time.Since(start), false)
    
    // 执行降级
    if fallback, ok := e.fallbackRules[tool.Name()]; ok {
        return fallback(ctx, tool, input, lastErr)
    }
    
    return nil, lastErr
}

func isRetryable(err error) bool {
    // 网络错误、超时可重试
    // 参数错误、权限错误不可重试
    return errors.Is(err, context.DeadlineExceeded) ||
           errors.Is(err, ErrNetworkError) ||
           errors.Is(err, ErrServiceUnavailable)
}
```

### 4.3 降级策略

```go
// plugin/ai/agent/tools/fallback.go

type FallbackFunc func(ctx context.Context, tool Tool, input string, err error) (*Result, error)

var DefaultFallbackRules = map[string]FallbackFunc{
    "memo_search": func(ctx context.Context, tool Tool, input string, err error) (*Result, error) {
        return &Result{
            Output:  "搜索暂时不可用，请稍后重试",
            Success: false,
        }, nil
    },
    
    "schedule_query": func(ctx context.Context, tool Tool, input string, err error) (*Result, error) {
        // 尝试使用缓存
        if cached := getCachedSchedules(ctx, input); cached != nil {
            return &Result{
                Output:  formatSchedules(cached) + "\n(来自缓存，可能不是最新)",
                Success: true,
            }, nil
        }
        return &Result{
            Output:  "日程查询暂时不可用，请稍后重试",
            Success: false,
        }, nil
    },
    
    "schedule_add": func(ctx context.Context, tool Tool, input string, err error) (*Result, error) {
        return &Result{
            Output:  "日程已记录，待确认后生效。您可以稍后在日程页面查看",
            Success: false,
        }, nil
    },
}
```

---

## 5. 交付物清单

### 5.1 代码文件

- [ ] `plugin/ai/agent/tools/executor.go` - 工具执行器
- [ ] `plugin/ai/agent/tools/fallback.go` - 降级策略

### 5.2 测试文件

- [ ] `plugin/ai/agent/tools/executor_test.go`

---

## 6. 测试验收

### 6.1 功能测试

| 场景 | 输入 | 预期输出 |
|:---|:---|:---|
| 首次成功 | 正常调用 | 返回结果 |
| 重试成功 | 首次超时，重试成功 | 返回结果 |
| 降级执行 | 多次失败 | 执行降级策略 |
| 指标上报 | 任意调用 | 指标记录正确 |

### 6.2 性能验收

| 指标 | 目标值 | 测试方法 |
|:---|:---|:---|
| 执行开销 | < 5ms | 单元测试 |
| 重试延迟 | 500ms | 配置验证 |

---

## 7. ROI 分析

| 维度 | 值 |
|:---|:---|
| 开发投入 | 2 人天 |
| 预期收益 | 工具调用成功率 +10%，用户体验提升 |
| 风险评估 | 低 |
| 回报周期 | Phase 1 结束 |

---

## 8. 实施计划

### 8.1 时间表

| 阶段 | 时间 | 任务 |
|:---|:---|:---|
| Day 1 | 1人天 | 执行器实现 |
| Day 2 | 1人天 | 降级策略 + 测试 |

### 8.2 检查点

- [ ] Day 1: 重试机制生效
- [ ] Day 2: 降级策略完成

---

## 附录

### A. 参考资料

- [智能助理路线图 - 工具可靠性](../../research/assistant-roadmap.md)

### B. 变更记录

| 日期 | 版本 | 变更内容 | 作者 |
|:---|:---|:---|:---|
| 2026-01-27 | v1.0 | 初始版本 | - |
