# P2-B002: 快速创建模式

> **状态**: 🔲 待开发  
> **优先级**: P1 (重要)  
> **投入**: 4 人天  
> **负责团队**: 团队 B  
> **Sprint**: Sprint 3

---

## 1. 目标与背景

### 1.1 核心目标

实现日程快速创建模式，用户输入简短语句（如"明天下午3点开会"）即可一键创建日程，无需多轮确认。

### 1.2 用户价值

- 日程创建从 3 步减少到 1 步
- 创建时间从 30 秒减少到 5 秒
- "说一句话就搞定"的体验

### 1.3 技术价值

- 减少 LLM 调用次数
- 降低交互延迟
- 为批量创建（P3-B003）奠定基础

---

## 2. 依赖关系

### 2.1 前置依赖

- [x] P1-A004: 时间解析服务（核心依赖）
- [x] P1-B003: 时间解析加固（LLM 输出规范化）
- [x] P1-B004: 规则分类器（意图识别）

### 2.2 并行依赖

- P2-B001: 用户习惯学习（习惯应用）

### 2.3 后续依赖

- P3-B003: 批量日程支持

---

## 3. 功能设计

### 3.1 架构图

```
                    快速创建流程
┌────────────────────────────────────────────────────────────┐
│                                                            │
│   用户输入: "明天下午3点开会"                                │
│                     │                                      │
│                     ▼                                      │
│   ┌─────────────────────────────────────────────────────┐ │
│   │            FastCreateParser                          │ │
│   │                                                      │ │
│   │  Step 1: 意图识别 (规则优先)                          │ │
│   │          └─ 匹配 SimpleCreate 模式                   │ │
│   │                                                      │ │
│   │  Step 2: 时间提取                                    │ │
│   │          └─ "明天下午3点" → 2026-01-28 15:00         │ │
│   │                                                      │ │
│   │  Step 3: 动作提取                                    │ │
│   │          └─ "开会" → title                          │ │
│   │                                                      │ │
│   │  Step 4: 缺省填充 (用户习惯)                         │ │
│   │          └─ duration: 60min (默认)                  │ │
│   └─────────────────────────────────────────────────────┘ │
│                     │                                      │
│                     ▼                                      │
│   ┌─────────────────────────────────────────────────────┐ │
│   │            验证 & 确认                               │ │
│   │                                                      │ │
│   │  如果信息完整:                                       │ │
│   │    → 显示预览卡片 + 一键确认                         │ │
│   │                                                      │ │
│   │  如果信息不足:                                       │ │
│   │    → 降级到普通创建流程                              │ │
│   └─────────────────────────────────────────────────────┘ │
│                     │                                      │
│                     ▼                                      │
│              日程创建成功                                   │
│                                                            │
└────────────────────────────────────────────────────────────┘
```

### 3.2 快速创建解析器

```go
// plugin/ai/agent/schedule/fast_create.go

type FastCreateResult struct {
    CanFastCreate bool              // 是否可以快速创建
    Schedule      *ScheduleRequest  // 解析出的日程
    MissingFields []string          // 缺失字段
    Confidence    float64           // 置信度
}

type FastCreateParser struct {
    timeService   TimeService
    habitApplier  *HabitApplier
    ruleClassifier *RuleClassifier
}

func NewFastCreateParser(timeSvc TimeService, habitApplier *HabitApplier) *FastCreateParser {
    return &FastCreateParser{
        timeService:   timeSvc,
        habitApplier:  habitApplier,
        ruleClassifier: NewRuleClassifier(),
    }
}

func (p *FastCreateParser) Parse(ctx context.Context, userID int32, input string) (*FastCreateResult, error) {
    result := &FastCreateResult{
        Schedule: &ScheduleRequest{},
    }
    
    // Step 1: 意图识别
    intent := p.ruleClassifier.Classify(input)
    if intent != IntentSimpleCreate {
        result.CanFastCreate = false
        result.MissingFields = []string{"intent_unclear"}
        return result, nil
    }
    
    // Step 2: 时间提取
    parsedTime, err := p.timeService.Parse(ctx, input, userID)
    if err != nil || parsedTime.IsZero() {
        result.CanFastCreate = false
        result.MissingFields = append(result.MissingFields, "time")
        return result, nil
    }
    result.Schedule.StartTime = parsedTime
    
    // Step 3: 动作/标题提取
    title := p.extractTitle(input)
    if title == "" {
        result.CanFastCreate = false
        result.MissingFields = append(result.MissingFields, "title")
        return result, nil
    }
    result.Schedule.Title = title
    
    // Step 4: 缺省填充
    p.applyDefaults(ctx, userID, result.Schedule)
    
    // 计算置信度
    result.Confidence = p.calculateConfidence(result.Schedule)
    result.CanFastCreate = result.Confidence >= 0.8
    
    return result, nil
}
```

### 3.3 标题提取

```go
// plugin/ai/agent/schedule/title_extractor.go

var (
    // 时间词移除模式
    timePatterns = []string{
        `今天|明天|后天|大后天`,
        `周[一二三四五六日天]|下周[一二三四五六日天]`,
        `\d{1,2}月\d{1,2}[日号]`,
        `[上下]午`,
        `\d{1,2}[点时](\d{1,2}分)?`,
        `早上|中午|晚上|傍晚`,
    }
    
    // 动作词映射
    actionMappings = map[string]string{
        "开会":   "会议",
        "meeting": "Meeting",
        "约":    "约会",
        "面试":   "面试",
        "汇报":   "工作汇报",
        "电话":   "电话会议",
        "讨论":   "讨论",
    }
)

func (p *FastCreateParser) extractTitle(input string) string {
    // 移除时间表达式
    cleaned := input
    for _, pattern := range timePatterns {
        re := regexp.MustCompile(pattern)
        cleaned = re.ReplaceAllString(cleaned, "")
    }
    
    // 清理空格和标点
    cleaned = strings.TrimSpace(cleaned)
    cleaned = strings.Trim(cleaned, "，。、")
    
    // 映射常见动作词
    for action, title := range actionMappings {
        if strings.Contains(cleaned, action) {
            return title
        }
    }
    
    // 如果还有内容，直接作为标题
    if len(cleaned) > 0 && len(cleaned) <= 50 {
        return cleaned
    }
    
    return ""
}
```

### 3.4 缺省值填充

```go
// plugin/ai/agent/schedule/defaults.go

func (p *FastCreateParser) applyDefaults(ctx context.Context, userID int32, schedule *ScheduleRequest) {
    // 应用用户习惯
    if p.habitApplier != nil {
        schedule = p.habitApplier.ApplyToScheduleCreate(ctx, userID, schedule)
    }
    
    // 默认时长
    if schedule.Duration == 0 {
        schedule.Duration = 60 // 默认 1 小时
    }
    
    // 默认提醒
    if schedule.ReminderMinutes == 0 {
        schedule.ReminderMinutes = 15 // 提前 15 分钟
    }
    
    // 计算结束时间
    if schedule.EndTime.IsZero() && !schedule.StartTime.IsZero() {
        schedule.EndTime = schedule.StartTime.Add(time.Duration(schedule.Duration) * time.Minute)
    }
}
```

### 3.5 置信度计算

```go
// plugin/ai/agent/schedule/confidence.go

func (p *FastCreateParser) calculateConfidence(schedule *ScheduleRequest) float64 {
    var score float64 = 1.0
    
    // 时间完整性
    if schedule.StartTime.IsZero() {
        score -= 0.4
    } else {
        // 检查时间是否合理（不是过去）
        if schedule.StartTime.Before(time.Now()) {
            score -= 0.2
        }
    }
    
    // 标题完整性
    if schedule.Title == "" {
        score -= 0.4
    } else if len(schedule.Title) < 2 {
        score -= 0.1
    }
    
    // 时长合理性
    if schedule.Duration <= 0 || schedule.Duration > 480 {
        score -= 0.1
    }
    
    return max(0, score)
}
```

### 3.6 快速创建处理器

```go
// plugin/ai/agent/schedule/fast_create_handler.go

type FastCreateHandler struct {
    parser        *FastCreateParser
    scheduleStore ScheduleStore
}

func (h *FastCreateHandler) Handle(ctx context.Context, userID int32, input string) (*AgentResponse, error) {
    // 尝试快速解析
    result, err := h.parser.Parse(ctx, userID, input)
    if err != nil {
        return nil, err
    }
    
    if !result.CanFastCreate {
        // 降级到普通流程
        return &AgentResponse{
            Type:    ResponseTypeFallback,
            Message: "需要更多信息，请确认以下内容：",
            Data: map[string]any{
                "missing_fields": result.MissingFields,
            },
        }, nil
    }
    
    // 生成预览卡片
    preview := h.generatePreview(result.Schedule)
    
    return &AgentResponse{
        Type:    ResponseTypeFastCreate,
        Message: "已识别日程，请确认：",
        Data: map[string]any{
            "preview":    preview,
            "schedule":   result.Schedule,
            "confidence": result.Confidence,
        },
        Actions: []Action{
            {Type: "confirm", Label: "确认创建", Data: result.Schedule},
            {Type: "edit", Label: "修改", Data: result.Schedule},
            {Type: "cancel", Label: "取消"},
        },
    }, nil
}

func (h *FastCreateHandler) generatePreview(schedule *ScheduleRequest) string {
    return fmt.Sprintf(
        "📅 %s\n⏰ %s - %s\n⏱️ %d 分钟",
        schedule.Title,
        schedule.StartTime.Format("01月02日 15:04"),
        schedule.EndTime.Format("15:04"),
        schedule.Duration,
    )
}
```

### 3.7 前端预览卡片

```tsx
// web/src/components/ai/FastCreatePreview.tsx

interface FastCreatePreviewProps {
  schedule: ScheduleRequest;
  confidence: number;
  onConfirm: () => void;
  onEdit: () => void;
  onCancel: () => void;
}

export function FastCreatePreview({
  schedule,
  confidence,
  onConfirm,
  onEdit,
  onCancel,
}: FastCreatePreviewProps) {
  return (
    <div className="rounded-lg border border-blue-200 bg-blue-50 p-4">
      <div className="flex items-start gap-3">
        <CalendarIcon className="h-5 w-5 text-blue-600" />
        <div className="flex-1">
          <h3 className="font-medium text-gray-900">{schedule.title}</h3>
          <p className="text-sm text-gray-600">
            {formatDateTime(schedule.startTime)} - {formatTime(schedule.endTime)}
          </p>
          <p className="text-xs text-gray-500">
            时长: {schedule.duration} 分钟
            {confidence >= 0.9 && " • 高置信度"}
          </p>
        </div>
      </div>
      
      <div className="mt-3 flex gap-2">
        <Button size="sm" onClick={onConfirm}>
          确认创建
        </Button>
        <Button size="sm" variant="outline" onClick={onEdit}>
          修改
        </Button>
        <Button size="sm" variant="ghost" onClick={onCancel}>
          取消
        </Button>
      </div>
    </div>
  );
}
```

---

## 4. 实现路径

### Day 1: 快速创建解析器

- [ ] 实现 `FastCreateParser`
- [ ] 标题提取逻辑
- [ ] 置信度计算

### Day 2: 缺省值与习惯应用

- [ ] 实现缺省值填充
- [ ] 集成习惯应用
- [ ] 处理边界情况

### Day 3: 处理器与集成

- [ ] 实现 `FastCreateHandler`
- [ ] 与 ScheduleAgent 集成
- [ ] 降级逻辑

### Day 4: 前端与测试

- [ ] 预览卡片组件
- [ ] 单元测试
- [ ] 端到端测试

---

## 5. 交付物

### 5.1 代码产出

| 文件 | 说明 |
|:---|:---|
| `plugin/ai/agent/schedule/fast_create.go` | 快速创建解析器 |
| `plugin/ai/agent/schedule/title_extractor.go` | 标题提取 |
| `plugin/ai/agent/schedule/defaults.go` | 缺省值填充 |
| `plugin/ai/agent/schedule/confidence.go` | 置信度计算 |
| `plugin/ai/agent/schedule/fast_create_handler.go` | 处理器 |
| `web/src/components/ai/FastCreatePreview.tsx` | 预览卡片 |
| `*_test.go` | 单元测试 |

### 5.2 配置项

```yaml
# configs/ai.yaml
fast_create:
  enabled: true
  confidence_threshold: 0.8
  
  defaults:
    duration: 60
    reminder_minutes: 15
    
  action_mappings:
    开会: 会议
    面试: 面试
    汇报: 工作汇报
```

---

## 6. 验收标准

### 6.1 功能验收

| 输入 | 期望输出 |
|:---|:---|
| "明天下午3点开会" | 快速创建：会议 2026-01-28 15:00-16:00 |
| "后天早上9点面试" | 快速创建：面试 2026-01-29 09:00-10:00 |
| "周五开会" | 降级：缺少具体时间 |
| "明天做点什么" | 降级：意图不明确 |

### 6.2 性能验收

- [ ] 解析延迟 < 100ms（不含 LLM）
- [ ] 置信度 ≥ 0.8 才快速创建
- [ ] LLM 调用减少 50%+

### 6.3 测试用例

```go
func TestFastCreateParsing(t *testing.T) {
    parser := NewFastCreateParser(mockTimeSvc, mockHabitApplier)
    
    tests := []struct {
        input    string
        canFast  bool
        title    string
    }{
        {"明天下午3点开会", true, "会议"},
        {"后天早上9点面试", true, "面试"},
        {"周五开会", false, ""},       // 缺少时间
        {"明天做点什么", false, ""},   // 意图不明
    }
    
    for _, tt := range tests {
        result, _ := parser.Parse(context.Background(), 1, tt.input)
        assert.Equal(t, tt.canFast, result.CanFastCreate)
        if tt.canFast {
            assert.Equal(t, tt.title, result.Schedule.Title)
        }
    }
}
```

---

## 7. ROI 分析

| 投入 | 产出 |
|:---|:---|
| 开发: 4 人天 | 日程创建效率提升 80% |
| 存储: 0 | LLM 调用减少 50%+ |
| 维护: 规则可配置 | 用户满意度提升 |

### 收益计算

- 原流程: 3 轮对话 × 10 秒/轮 = 30 秒
- 新流程: 1 句话 + 确认 = 5 秒
- 效率提升: (30-5)/30 = 83%

---

## 8. 风险与缓解

| 风险 | 概率 | 影响 | 缓解措施 |
|:---|:---:|:---:|:---|
| 误创建 | 中 | 高 | 置信度阈值 + 预览确认 |
| 时间解析错误 | 中 | 中 | 依赖 TimeService 加固 |
| 用户习惯不准 | 低 | 低 | 默认值兜底 |

---

## 9. 排期

| 日期 | 任务 | 负责人 |
|:---|:---|:---|
| Sprint 3 Day 1 | 快速创建解析器 | TBD |
| Sprint 3 Day 2 | 缺省值与习惯应用 | TBD |
| Sprint 3 Day 3 | 处理器与集成 | TBD |
| Sprint 3 Day 4 | 前端与测试 | TBD |

---

> **纲领来源**: [00-master-roadmap.md](../../../archived/research_20250202/00-master-roadmap.md)  
> **研究文档**: [schedule-roadmap.md](../../../research/schedule-roadmap.md)  
> **版本**: v1.0  
> **更新时间**: 2026-01-27
