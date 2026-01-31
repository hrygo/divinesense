# SPEC-003: 惊奇助手 (Amazing Agent)

> **状态**: 待实现
> **优先级**: P1
> **依赖**: SPEC-001, SPEC-002
> **负责人**: 后端开发组

## 1. 概述

"惊奇 (Amazing)" (🌟 惊奇) 是鹦鹉家族的 **元 Agent (Meta-Agent)**。它不直接管理数据，而是作为协调者，并行调度 `MemoParrot` 和 `ScheduleParrot`，为用户提供跨领域的综合服务。

## 2. 核心架构

### 2.1 编排逻辑 (Orchestrator)

**目标**: 智能判断用户意图，决定调用哪些子 Agent。

**判断逻辑**:
1.  **关键词检测**:
    *   日程词 (日程, 会议, 时间...) -> `NeedsSchedule = true`
    *   笔记词 (笔记, 搜索, 总结...) -> `NeedsMemo = true`
2.  **默认策略**: 如果无法明确区分，则默认为 **Both (同时查询)**，以提供最全面的信息。

### 2.2 并发执行模型

**实现 (`plugin/ai/agent/amazing_parrot.go`)**:
使用 Go `errgroup` 实现并发调用。

```go
func (p *AmazingParrot) Execute(ctx context.Context, input string) (string, error) {
    g, ctx := errgroup.WithContext(ctx)
    
    // 省略结果通道定义...

    // 1. 调用 MemoParrot
    if needsMemo {
        g.Go(func() error {
            // 调用 p.memoParrot.Execute...
            return nil
        })
    }

    // 2. 调用 ScheduleParrot
    if needsSchedule {
        g.Go(func() error {
            // 调用 p.scheduleParrot.Execute...
            return nil
        })
    }

    if err := g.Wait(); err != nil {
        return "", err
    }

    // 3. 结果合成
    return p.synthesizeResults(memoResult, scheduleResult)
}
```

### 2.3 结果合成

**目标**: 将零散的工具结果整合成流畅的自然语言回答。

**Prompt 策略**:
> "你拥有以下信息：
> 1. 笔记搜索结果: [内容...]
> 2. 日程查询结果: [内容...]
> 请综合上述信息回答用户问题。如果某方面没有结果，请忽略该部分。"

## 3. 混合检索与 RAG 优化 (基于调研报告)

虽然 `MemoParrot` 内部已经 (或将要) 实现 BM25 + Vector 的混合检索，`AmazingParrot` 在应用层实现的是 **联邦搜索 (Federated Search)**。

**性能优化**:
1.  **Fail-Fast**: 如果任何一个子 Agent 快速报错 (e.g., 数据库超时)，不应阻塞其他 Agent 的结果。
2.  **Result Cache**: 对于 "我的今天" 这类高频组合查询，AmazingParrot 可在这一层做短时缓存 (1 min)。

## 4. 验收标准 (Acceptance Criteria)

### AC-003.1: 意图路由
- [ ] **纯笔记**: 输入 "搜索代码笔记"，只调用 MemoParrot。
- [ ] **纯日程**: 输入 "在这个时间段有空吗"，只调用 ScheduleParrot。
- [ ] **混合查询**: 输入 "查看关于项目的笔记和我下周的会议"，**并行** 调用两者。

### AC-003.2: 并发性能
- [ ] **延迟**: 混合查询的总耗时应接近于最慢的那个子 Agent 耗时，而不是两者之和 (T_total ≈ max(T_memo, T_schedule))。
- [ ] **超时**: 整体超时控制在 2 分钟。

### AC-003.3: 结果质量
- [ ] **合成回答**: 最终回复应逻辑通顺，比如 "我找到了 3 条相关笔记，此外你下周还有 2 个相关会议..."。
- [ ] **结构化数据**: 前端能接收到 `memo_query_result` 和 `schedule_query_result` 事件，并渲染对应的 UI 卡片。

## 5. 实施步骤

1.  创建 `amazing_parrot.go`，注入 `MemoParrot` 和 `ScheduleParrot` 实例。
2.  实现 `analyzeIntent` 函数 (简单的关键词或 LLM 判断)。
3.  实现 `errgroup` 并发调用逻辑。
4.  实现 `synthesizeResults` Prompt 构建。
5.  注册到 Router。
