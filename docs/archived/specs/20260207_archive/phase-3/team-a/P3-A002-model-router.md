# P3-A002: 模型路由器

> **状态**: 🔲 待开发  
> **优先级**: P2 (增强)  
> **投入**: 3 人天  
> **负责团队**: 团队 A  
> **Sprint**: Sprint 6

---

## 1. 目标与背景

### 1.1 核心目标

实现智能模型路由，根据任务复杂度、网络状态、成本预算自动选择本地/云端模型。

### 1.2 用户价值

- 自动选择最优模型
- 离线时无缝降级
- 成本与质量平衡

---

## 2. 依赖关系

- [x] P3-A001: 本地模型集成
- [x] P1-A003: LLM 路由优化

---

## 3. 功能设计

### 3.1 路由策略

```
┌────────────────────────────────────────────────────────────┐
│                    模型路由决策树                           │
├────────────────────────────────────────────────────────────┤
│                                                            │
│   请求进入                                                  │
│       │                                                    │
│       ▼                                                    │
│   ┌─────────────────┐                                     │
│   │ 检查用户配置     │                                     │
│   │ prefer_local?   │                                     │
│   └─────────────────┘                                     │
│       │ Yes              │ No                              │
│       ▼                  ▼                                 │
│   ┌─────────────┐    ┌─────────────┐                      │
│   │ 本地可用?   │    │ 任务复杂度  │                      │
│   └─────────────┘    └─────────────┘                      │
│       │ Yes  │ No        │ Simple  │ Complex              │
│       ▼      ▼           ▼         ▼                      │
│   [Local]  [Cloud]   [Local]    [Cloud]                   │
│                                                            │
└────────────────────────────────────────────────────────────┘
```

### 3.2 核心实现

```go
// plugin/ai/llm/router.go

type ModelRouter struct {
    localProvider  LocalLLMProvider
    cloudProvider  LLMProvider
    config         *RouterConfig
}

type RouterConfig struct {
    PreferLocal      bool    `yaml:"prefer_local"`
    ComplexityThreshold int  `yaml:"complexity_threshold"`  // token 数阈值
    FallbackEnabled  bool    `yaml:"fallback_enabled"`
}

func (r *ModelRouter) Route(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error) {
    // 1. 检查用户偏好
    if r.config.PreferLocal && r.localProvider.IsAvailable(ctx) {
        return r.localProvider.Complete(ctx, req)
    }
    
    // 2. 简单任务用本地
    if r.isSimpleTask(req) && r.localProvider.IsAvailable(ctx) {
        return r.localProvider.Complete(ctx, req)
    }
    
    // 3. 复杂任务用云端
    resp, err := r.cloudProvider.Complete(ctx, req)
    if err != nil && r.config.FallbackEnabled {
        // 云端失败，降级本地
        return r.localProvider.Complete(ctx, req)
    }
    
    return resp, err
}

func (r *ModelRouter) isSimpleTask(req *CompletionRequest) bool {
    return len(req.Prompt) < r.config.ComplexityThreshold
}
```

### 3.3 配置

```yaml
model_router:
  prefer_local: false
  complexity_threshold: 500  # tokens
  fallback_enabled: true
  
  local_tasks:
    - "intent_classification"
    - "time_parsing"
    - "simple_qa"
    
  cloud_tasks:
    - "complex_reasoning"
    - "long_context"
```

---

## 4. 实现路径

| Day | 任务 |
|-----|------|
| 1 | Router 核心逻辑 |
| 2 | 复杂度判断 + 降级策略 |
| 3 | 配置化 + 测试 |

---

## 5. 验收标准

- [ ] 本地优先模式正常工作
- [ ] 云端失败自动降级本地
- [ ] 简单任务路由到本地

---

> **版本**: v1.0 | **更新时间**: 2026-01-27
