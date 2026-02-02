# P1-A003: LLM 路由优化

> **状态**: 🔄 进行中  
> **优先级**: P0 (核心)  
> **投入**: 3 人天  
> **负责团队**: 团队 A  
> **Sprint**: Sprint 1-2

---

## 1. 目标与背景

### 1.1 核心目标

优化 LLM 路由策略，实现三层路由架构，将 80% 请求在规则层完成，LLM 调用减少 60%。

### 1.2 用户价值

- 响应延迟降低 300ms+
- 降低 API 成本

### 1.3 技术价值

- 统一路由服务供团队 B/C 调用
- 为模型路由器 (P3-A002) 奠定基础

---

## 2. 依赖关系

### 2.1 前置依赖

- [x] S0-interface-contract: 接口定义
- [ ] P1-A001: 记忆系统（用于历史模式匹配）

### 2.2 并行依赖

- 无

### 2.3 后续依赖

- P1-B004: 规则分类器扩展
- P2-C001: 智能标签建议

---

## 3. 功能设计

### 3.1 架构图

```
┌─────────────────────────────────────────────────────────────────┐
│                    三层 LLM 路由架构                              │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  用户输入                                                        │
│      │                                                          │
│      ▼                                                          │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │  Layer 1: 精确匹配 (0ms)                                 │   │
│  │  • 日程关键词: 明天/后天/开会/会议/安排...               │   │
│  │  • 笔记关键词: 搜索/查找/记录/写下...                    │   │
│  │  • 正则模式: 时间表达式、标签格式                        │   │
│  │  • 触发条件: 得分 >= 阈值                                │   │
│  └────────────────────────┬────────────────────────────────┘   │
│                           │ 未匹配                              │
│                           ▼                                     │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │  Layer 2: 历史模式匹配 (~10ms)                           │   │
│  │  • 查询情景记忆中相似历史                                │   │
│  │  • 复用历史路由决策                                      │   │
│  │  • 相似度阈值 > 0.8                                      │   │
│  └────────────────────────┬────────────────────────────────┘   │
│                           │ 未匹配                              │
│                           ▼                                     │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │  Layer 3: LLM 分类 (~400ms)                              │   │
│  │  • 仅对真正模糊的输入使用                                 │   │
│  │  • 结果写入历史模式库                                     │   │
│  │  • 置信度 < 0.7 时返回 Unknown                           │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                 │
│  目标: 80% 请求在 Layer 1/2 完成                                │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### 3.2 核心流程

1. **Layer 1**: 关键词权重匹配，0ms
2. **Layer 2**: 从记忆系统查询相似历史，~10ms
3. **Layer 3**: LLM 分类，~400ms，结果写入记忆

### 3.3 关键词权重表

| 意图 | 权重+2 关键词 | 权重+1 关键词 | 触发阈值 |
|:---|:---|:---|:---:|
| schedule | 日程/安排/会议/提醒/预约 | 今天/明天/后天/时间词 | >= 3 |
| memo | 笔记/搜索/查找/记录/写过 | 关于/提到/之前 | >= 2 |
| amazing | 综合/总结/分析/周报 | 本周/工作 | >= 2 |

---

## 4. 技术实现

### 4.1 接口定义

见 [S0-interface-contract](../sprint-0/S0-interface-contract.md)

### 4.2 关键代码路径

| 文件路径 | 职责 |
|:---|:---|
| `plugin/ai/router/service.go` | 路由服务主实现 |
| `plugin/ai/router/rule_matcher.go` | Layer 1 规则匹配 |
| `plugin/ai/router/history_matcher.go` | Layer 2 历史匹配 |
| `plugin/ai/router/llm_classifier.go` | Layer 3 LLM 分类 |

### 4.3 规则匹配实现

```go
// plugin/ai/router/rule_matcher.go

type RuleMatcher struct {
    scheduleKeywords map[string]int  // 关键词 -> 权重
    memoKeywords     map[string]int
    amazingKeywords  map[string]int
    timePatterns     []*regexp.Regexp
}

func (m *RuleMatcher) Match(input string) (Intent, float32, bool) {
    lower := strings.ToLower(input)
    
    scheduleScore := m.calculateScore(lower, m.scheduleKeywords)
    memoScore := m.calculateScore(lower, m.memoKeywords)
    amazingScore := m.calculateScore(lower, m.amazingKeywords)
    
    // 时间模式加分
    if m.hasTimePattern(lower) {
        scheduleScore += 1
    }
    
    // 选择最高分
    if scheduleScore >= 3 {
        return IntentScheduleQuery, float32(scheduleScore)/5.0, true
    }
    if memoScore >= 2 && scheduleScore < 2 {
        return IntentMemoSearch, float32(memoScore)/4.0, true
    }
    if amazingScore >= 2 {
        return IntentAmazing, float32(amazingScore)/4.0, true
    }
    
    return IntentUnknown, 0, false
}
```

---

## 5. 交付物清单

### 5.1 代码文件

- [ ] `plugin/ai/router/service.go` - 路由服务主实现
- [ ] `plugin/ai/router/rule_matcher.go` - 规则匹配器
- [ ] `plugin/ai/router/history_matcher.go` - 历史匹配器
- [ ] `plugin/ai/router/llm_classifier.go` - LLM 分类器

### 5.2 测试文件

- [ ] `plugin/ai/router/service_test.go`
- [ ] `plugin/ai/router/rule_matcher_test.go`

---

## 6. 测试验收

### 6.1 功能测试

| 场景 | 输入 | 预期输出 |
|:---|:---|:---|
| 明确日程 | "明天下午3点开会" | IntentScheduleCreate, >0.8 |
| 明确笔记 | "搜索关于 Go 的笔记" | IntentMemoSearch, >0.8 |
| 模糊输入 | "帮我看看" | IntentUnknown (需 LLM) |

### 6.2 性能验收

| 指标 | 目标值 | 测试方法 |
|:---|:---|:---|
| Layer 1 延迟 | < 1ms | 单元测试 |
| Layer 2 延迟 | < 50ms | 集成测试 |
| LLM 调用率 | < 20% | 统计测试 |

---

## 7. ROI 分析

| 维度 | 值 |
|:---|:---|
| 开发投入 | 3 人天 |
| 预期收益 | LLM 调用 -60%，延迟 -300ms |
| 风险评估 | 低 |
| 回报周期 | Phase 1 结束 |

---

## 8. 实施计划

### 8.1 时间表

| 阶段 | 时间 | 任务 |
|:---|:---|:---|
| Day 1 | 1人天 | Layer 1 规则匹配 |
| Day 2 | 1人天 | Layer 2 历史匹配 |
| Day 3 | 1人天 | Layer 3 + 集成测试 |

### 8.2 检查点

- [ ] Day 1: 规则匹配覆盖常见场景
- [ ] Day 3: LLM 调用率 < 20%

---

## 附录

### A. 参考资料

- [ChatRouter 现有实现](../../../plugin/ai/agent/chat_router.go)
- [智能助理路线图](../../research/assistant-roadmap.md)

### B. 变更记录

| 日期 | 版本 | 变更内容 | 作者 |
|:---|:---|:---|:---|
| 2026-01-27 | v1.0 | 初始版本 | - |
