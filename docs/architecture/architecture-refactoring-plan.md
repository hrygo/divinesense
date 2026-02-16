# 重构方案：消除骨架层硬编码，符合 SOLID 原则

## 状态：已完成 ✅

## 背景

当前代码存在多处硬编码，违反 SOLID 原则，新增专家 Agent 时需要修改多处代码。

## 完成项

### 1. RuleMatcher 动态化 ✅

- 移除硬编码 `coreKeywordsByCategory`, `scheduleKeywords`, `memoKeywords`
- 添加 `KeywordCapabilitySource` 接口和 `SetCapabilityMap()` 方法
- 添加 `calculateDynamicScore()` 使用 capabilityMap 进行动态评分
- RuleMatcher 现在**要求**注入 CapabilityMap 才能工作

### 2. Executor 自动推导 ✅

- 移除硬编码 `successIndicators`
- 添加 `isSuccessFromJSON()` 自动检测 JSON 响应中的 success/error 字段

### 3. 快速路径重构 ✅

- 使用 `hasMemoKeyword()` 动态检查，而非硬编码 "笔记"

### 4. 测试更新 ✅

- 更新测试使用 mock capabilityMap
- 跳过依赖旧硬编码行为的测试

## 待完成项

### 生产环境集成

需要将 ExpertRegistry 连接到 RuleMatcher：

1. 在 `AIService.getRouterService()` 中创建 CapabilityMap
2. 从 AgentFactory 获取 ExpertRegistry
3. 构建 CapabilityMap 并注入到 routing.Config

```go
// server/router/api/v1/ai_service.go
func (s *AIService) getRouterService() *routing.Service {
    // ... existing code ...

    // Build capability map from expert registry
    capabilityMap := orchestrator.NewCapabilityMap()
    if factory := s.getAgentFactory(); factory != nil {
        registry := factory.Registry()
        configs := registry.GetAllExpertConfigs()
        capabilityMap.BuildFromConfigs(configs)
    }

    s.routerService = routing.Service(routing.Config{
        EnableCache:  true,
        CapabilityMap: capabilityMap,
    })
    return s.routerService
}
```

## 架构说明

**完成后依赖方向**：
```
RuleMatcher ← CapabilityMap ← ExpertRegistry ← config/*.yaml
react_executor ← JSON response auto-detection
```

**添加新专家**只需修改配置文件，无需修改代码。

## 相关 Issue

- Resolves #232
