# P1-B004: 规则分类器扩展

> **状态**: ✅ 已完成  
> **优先级**: P0 (核心)  
> **投入**: 2 人天  
> **负责团队**: 团队 B  
> **Sprint**: Sprint 2

---

## 1. 目标与背景

### 1.1 核心目标

基于团队 A 的 RouterService，在日程 Agent 层面扩展规则分类器，覆盖 90%+ 日程相关场景。

### 1.2 用户价值

- 日程意图识别更准确
- 响应延迟降低 300ms+

### 1.3 技术价值

- 减少 LLM 调用 70%+
- 与 RouterService 协同工作

---

## 2. 依赖关系

### 2.1 前置依赖

- [ ] P1-A003: LLM 路由优化（核心依赖）

### 2.2 并行依赖

- P1-B003: 时间解析加固

### 2.3 后续依赖

- P2-B001: 用户习惯学习

---

## 3. 功能设计

### 3.1 架构图

```
┌─────────────────────────────────────────────────────────────────┐
│                    日程规则分类器                                 │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  用户输入                                                        │
│      │                                                          │
│      ▼                                                          │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │  规则匹配 (0ms)                                           │   │
│  │                                                          │   │
│  │  ┌─────────────────────────────────────────────────────┐│   │
│  │  │ 创建模式                                             ││   │
│  │  │ • (时间词) + (动作/事件)                             ││   │
│  │  │   "明天3点开会" → SimpleCreate                       ││   │
│  │  │   "安排下周一面试" → SimpleCreate                    ││   │
│  │  └─────────────────────────────────────────────────────┘│   │
│  │                                                          │   │
│  │  ┌─────────────────────────────────────────────────────┐│   │
│  │  │ 查询模式                                             ││   │
│  │  │ • (疑问词) + (时间)                                  ││   │
│  │  │   "今天有什么安排" → SimpleQuery                     ││   │
│  │  │   "明天几点有会" → SimpleQuery                       ││   │
│  │  └─────────────────────────────────────────────────────┘│   │
│  │                                                          │   │
│  │  ┌─────────────────────────────────────────────────────┐│   │
│  │  │ 修改模式                                             ││   │
│  │  │ • (修改动词) + (目标)                                ││   │
│  │  │   "把会议改到3点" → SimpleUpdate                     ││   │
│  │  │   "取消明天的会" → SimpleUpdate                      ││   │
│  │  └─────────────────────────────────────────────────────┘│   │
│  │                                                          │   │
│  │  ┌─────────────────────────────────────────────────────┐│   │
│  │  │ 批量模式                                             ││   │
│  │  │ • (重复关键词)                                       ││   │
│  │  │   "每周一站会" → BatchCreate                         ││   │
│  │  │   "工作日早上9点" → BatchCreate                      ││   │
│  │  └─────────────────────────────────────────────────────┘│   │
│  │                                                          │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                 │
│  未匹配 → 调用 RouterService.ClassifyIntent()                   │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### 3.2 核心流程

1. **模式匹配**: 尝试规则匹配
2. **兜底路由**: 未匹配时调用 RouterService

### 3.3 关键词库

| 意图 | 高权重关键词 | 低权重关键词 | 触发阈值 |
|:---|:---|:---|:---:|
| SimpleCreate | 安排/约/预约/会议/开会/面试 | 时间词 | >= 3 |
| SimpleQuery | 什么安排/有什么/忙吗/有空/几点 | 时间词 | >= 2 |
| SimpleUpdate | 改/换/调/推迟/提前/取消/删除 | 会议/日程 | >= 2 |
| BatchCreate | 每天/每周/每月/工作日/周一到周五 | - | >= 1 |

---

## 4. 技术实现

### 4.1 关键代码路径

| 文件路径 | 职责 |
|:---|:---|
| `plugin/ai/schedule/intent_classifier.go` | 意图分类器 |

### 4.2 核心实现

```go
// plugin/ai/schedule/intent_classifier.go

type TaskIntent int

const (
    IntentUnknown TaskIntent = iota
    IntentSimpleCreate
    IntentSimpleQuery
    IntentSimpleUpdate
    IntentBatchCreate
)

type IntentClassifier struct {
    routerService router.RouterService
    patterns      []intentPattern
}

type intentPattern struct {
    regex  *regexp.Regexp
    intent TaskIntent
}

func NewIntentClassifier(routerService router.RouterService) *IntentClassifier {
    return &IntentClassifier{
        routerService: routerService,
        patterns: []intentPattern{
            // 创建: 时间 + 动作/事件
            {regexp.MustCompile(`(明天|后天|下周|今天).*(点|时).*(开会|会议|面试|约|见)`), IntentSimpleCreate},
            {regexp.MustCompile(`(上午|下午|晚上|早上).*(安排|约|预约)`), IntentSimpleCreate},
            {regexp.MustCompile(`安排.*(会议|面试|约会)`), IntentSimpleCreate},
            
            // 查询: 疑问词 + 时间
            {regexp.MustCompile(`(今天|明天|这周|下周).*(有什么|什么安排|忙吗|有空)`), IntentSimpleQuery},
            {regexp.MustCompile(`(查|看|显示).*(日程|安排|计划)`), IntentSimpleQuery},
            {regexp.MustCompile(`(几点|什么时候).*(会|开始)`), IntentSimpleQuery},
            
            // 修改: 修改动词 + 目标
            {regexp.MustCompile(`(改|换|调|推迟|提前|取消|删除).*(会议|日程|安排)`), IntentSimpleUpdate},
            {regexp.MustCompile(`把.*(改|换|调)到`), IntentSimpleUpdate},
            
            // 批量: 重复关键词
            {regexp.MustCompile(`每(天|周|月|年)|工作日|周一到周五`), IntentBatchCreate},
        },
    }
}

// Classify 分类用户意图
func (ic *IntentClassifier) Classify(ctx context.Context, input string) (TaskIntent, float32) {
    lowerInput := strings.ToLower(input)
    
    // 规则匹配
    for _, p := range ic.patterns {
        if p.regex.MatchString(lowerInput) {
            return p.intent, 0.9 // 高置信度
        }
    }
    
    // 兜底: 有时间词+动作词 → 创建
    if ic.hasTimeAndAction(lowerInput) {
        return IntentSimpleCreate, 0.7
    }
    
    // 调用 RouterService
    intent, confidence, err := ic.routerService.ClassifyIntent(ctx, input)
    if err != nil {
        return IntentUnknown, 0
    }
    
    return ic.mapRouterIntent(intent), confidence
}

// hasTimeAndAction 检查是否有时间词和动作词
func (ic *IntentClassifier) hasTimeAndAction(input string) bool {
    timeWords := []string{"今天", "明天", "后天", "下周", "点", "时"}
    actionWords := []string{"开会", "会议", "面试", "约", "安排"}
    
    hasTime := false
    hasAction := false
    
    for _, w := range timeWords {
        if strings.Contains(input, w) {
            hasTime = true
            break
        }
    }
    
    for _, w := range actionWords {
        if strings.Contains(input, w) {
            hasAction = true
            break
        }
    }
    
    return hasTime && hasAction
}
```

---

## 5. 交付物清单

### 5.1 代码文件

- [ ] `plugin/ai/schedule/intent_classifier.go` - 意图分类器

### 5.2 测试文件

- [ ] `plugin/ai/schedule/intent_classifier_test.go`

---

## 6. 测试验收

### 6.1 功能测试

| 场景 | 输入 | 预期输出 |
|:---|:---|:---|
| 简单创建 | "明天下午3点开会" | IntentSimpleCreate |
| 简单查询 | "今天有什么安排" | IntentSimpleQuery |
| 简单修改 | "把会议改到4点" | IntentSimpleUpdate |
| 批量创建 | "每周一9点站会" | IntentBatchCreate |

### 6.2 性能验收

| 指标 | 目标值 | 测试方法 |
|:---|:---|:---|
| 规则覆盖率 | > 90% | 测试用例集 |
| 分类延迟 | < 10ms | 单元测试 |

---

## 7. ROI 分析

| 维度 | 值 |
|:---|:---|
| 开发投入 | 2 人天 |
| 预期收益 | LLM 调用 -70%，延迟 -300ms |
| 风险评估 | 低 |
| 回报周期 | Phase 1 结束 |

---

## 8. 实施计划

### 8.1 时间表

| 阶段 | 时间 | 任务 |
|:---|:---|:---|
| Day 1 | 1人天 | 规则模式实现 |
| Day 2 | 1人天 | 测试用例 + 调优 |

### 8.2 检查点

- [ ] Day 1: 核心规则完成
- [ ] Day 2: 覆盖率 > 90%

---

## 附录

### A. 参考资料

- [日程路线图 - 规则分类器](../../research/schedule-roadmap.md)
- [ChatRouter 现有实现](../../../plugin/ai/agent/chat_router.go)

### B. 变更记录

| 日期 | 版本 | 变更内容 | 作者 |
|:---|:---|:---|:---|
| 2026-01-27 | v1.0 | 初始版本 | - |
