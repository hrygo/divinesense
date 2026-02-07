# P3-B001: 预测性交互系统

> **状态**: 🔲 待开发  
> **优先级**: P2 (增强)  
> **投入**: 5 人天  
> **负责团队**: 团队 B  
> **Sprint**: Sprint 5

---

## 1. 目标与背景

### 1.1 核心目标

基于用户习惯和上下文，预测用户下一步操作，主动提供快捷入口，实现"比用户更快一步"的体验。

### 1.2 用户价值

- 减少操作步骤 40%
- 更智能的助理体验
- "懂我"的私人助理

---

## 2. 依赖关系

- [x] P1-A001: 轻量记忆系统
- [x] P2-B001: 用户习惯学习

---

## 3. 功能设计

### 3.1 预测场景

```
┌────────────────────────────────────────────────────────────┐
│                    预测性交互场景                           │
├────────────────────────────────────────────────────────────┤
│                                                            │
│  场景 1: 时间触发                                          │
│  ────────────────                                          │
│  周一早上 9:00 → 预测"创建本周会议"                        │
│  下班前 17:30 → 预测"明天日程预览"                         │
│                                                            │
│  场景 2: 上下文触发                                        │
│  ────────────────                                          │
│  刚创建日程 → 预测"设置提醒"                               │
│  查看笔记后 → 预测"搜索相关笔记"                           │
│                                                            │
│  场景 3: 模式触发                                          │
│  ────────────────                                          │
│  每周一查看周报 → 主动推送周报入口                         │
│  月底整理笔记 → 提示"本月笔记回顾"                         │
│                                                            │
└────────────────────────────────────────────────────────────┘
```

### 3.2 核心实现

```go
// plugin/ai/prediction/engine.go

type PredictionEngine struct {
    habitService  *HabitAnalyzer
    memoryService MemoryService
    timeService   TimeService
}

type Prediction struct {
    Type       string  `json:"type"`        // "action", "query", "reminder"
    Label      string  `json:"label"`       // 显示文本
    Confidence float64 `json:"confidence"`
    Action     string  `json:"action"`      // 触发的动作
    Payload    any     `json:"payload"`
}

func (e *PredictionEngine) Predict(ctx context.Context, userID int32) ([]Prediction, error) {
    var predictions []Prediction
    
    // 1. 时间触发预测
    timePredictions := e.predictByTime(ctx, userID)
    predictions = append(predictions, timePredictions...)
    
    // 2. 上下文触发预测
    contextPredictions := e.predictByContext(ctx, userID)
    predictions = append(predictions, contextPredictions...)
    
    // 3. 模式触发预测
    patternPredictions := e.predictByPattern(ctx, userID)
    predictions = append(predictions, patternPredictions...)
    
    // 排序并返回 Top-3
    sort.Slice(predictions, func(i, j int) bool {
        return predictions[i].Confidence > predictions[j].Confidence
    })
    
    if len(predictions) > 3 {
        predictions = predictions[:3]
    }
    
    return predictions, nil
}

func (e *PredictionEngine) predictByTime(ctx context.Context, userID int32) []Prediction {
    now := time.Now()
    var predictions []Prediction
    
    // 周一早上 → 本周会议
    if now.Weekday() == time.Monday && now.Hour() >= 8 && now.Hour() <= 10 {
        predictions = append(predictions, Prediction{
            Type:       "action",
            Label:      "查看本周日程",
            Confidence: 0.8,
            Action:     "view_week_schedule",
        })
    }
    
    // 下班前 → 明日预览
    if now.Hour() >= 17 && now.Hour() <= 18 {
        predictions = append(predictions, Prediction{
            Type:       "action",
            Label:      "明天有什么安排？",
            Confidence: 0.7,
            Action:     "view_tomorrow",
        })
    }
    
    return predictions
}
```

### 3.3 前端展示

```tsx
// web/src/components/ai/PredictionChips.tsx

export function PredictionChips({ predictions, onSelect }: Props) {
  return (
    <div className="flex gap-2 overflow-x-auto pb-2">
      {predictions.map((pred) => (
        <button
          key={pred.action}
          onClick={() => onSelect(pred)}
          className="shrink-0 rounded-full bg-blue-100 px-4 py-2 text-sm text-blue-700 hover:bg-blue-200"
        >
          {pred.label}
        </button>
      ))}
    </div>
  );
}
```

---

## 4. 实现路径

| Day | 任务 |
|-----|------|
| 1-2 | 预测引擎核心逻辑 |
| 3 | 三种触发场景实现 |
| 4 | 前端组件 |
| 5 | 测试与调优 |

---

## 5. 验收标准

- [ ] 周一早上显示"本周日程"预测
- [ ] 预测准确率 > 60%
- [ ] 用户点击率 > 30%

---

> **版本**: v1.0 | **更新时间**: 2026-01-27
