# P1-B003: 时间解析加固

> **状态**: ✅ 已完成  
> **优先级**: P0 (核心)  
> **投入**: 2 人天  
> **负责团队**: 团队 B  
> **Sprint**: Sprint 1-2

---

## 1. 目标与背景

### 1.1 核心目标

基于团队 A 的 TimeService，在日程 Agent 层面加固时间解析，确保 LLM 生成的时间能被正确处理。

### 1.2 用户价值

- 日程创建成功率从 85% 提升至 98%
- 自然语言时间表达更可靠

### 1.3 技术价值

- 补充 LLM 输出的时间格式兼容
- 与 TimeService 协同工作

---

## 2. 依赖关系

### 2.1 前置依赖

- [ ] P1-A004: 时间解析服务（核心依赖）

### 2.2 并行依赖

- P1-B004: 规则分类器扩展

### 2.3 后续依赖

- P2-B002: 快速创建模式

---

## 3. 功能设计

### 3.1 架构图

```
┌─────────────────────────────────────────────────────────────────┐
│                    时间解析加固层                                 │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  LLM 输出的时间                                                  │
│      │                                                          │
│      ▼                                                          │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │  Step 1: LLM 输出格式兼容                                 │   │
│  │                                                          │   │
│  │  • "2026年1月28日下午3点" → 标准化                       │   │
│  │  • "tomorrow 3pm" → 标准化                               │   │
│  │  • "15:00" (缺日期) → 补充今天/明天                      │   │
│  │  • 错误格式 → 尝试修复                                   │   │
│  └────────────────────────┬────────────────────────────────┘   │
│                           │                                     │
│                           ▼                                     │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │  Step 2: 调用 TimeService                                 │   │
│  │                                                          │   │
│  │  TimeService.Normalize(input, timezone)                  │   │
│  └────────────────────────┬────────────────────────────────┘   │
│                           │                                     │
│                           ▼                                     │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │  Step 3: 合理性检查                                       │   │
│  │                                                          │   │
│  │  • 时间是否在合理范围内（不早于现在）                     │   │
│  │  • 是否在未来 1 年内                                      │   │
│  │  • 工作时间偏好检查                                       │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### 3.2 核心流程

1. **格式兼容**: 处理 LLM 输出的各种格式
2. **标准化**: 调用 TimeService
3. **合理性检查**: 验证时间合理性

### 3.3 LLM 输出格式表

| LLM 可能输出 | 处理方式 |
|:---|:---|
| `2026年1月28日下午3点` | 中文格式解析 |
| `tomorrow 3pm` | 英文混合解析 |
| `15:00` | 补充日期（智能推断） |
| `下午三点` | 中文数字转换 |
| `明天下午` | 默认时间（14:00） |

---

## 4. 技术实现

### 4.1 关键代码路径

| 文件路径 | 职责 |
|:---|:---|
| `plugin/ai/schedule/time_hardener.go` | 时间加固层 |

### 4.2 核心实现

```go
// plugin/ai/schedule/time_hardener.go

type TimeHardener struct {
    timeService aitime.TimeService
    timezone    *time.Location
    now         func() time.Time
}

func NewTimeHardener(timeService aitime.TimeService, timezone *time.Location) *TimeHardener {
    return &TimeHardener{
        timeService: timeService,
        timezone:    timezone,
        now:         time.Now,
    }
}

// HardenTime 加固时间解析
func (h *TimeHardener) HardenTime(ctx context.Context, input string) (time.Time, error) {
    // Step 1: 预处理 LLM 输出
    normalized := h.preprocessLLMOutput(input)
    
    // Step 2: 调用 TimeService
    t, err := h.timeService.Normalize(ctx, normalized, h.timezone.String())
    if err != nil {
        return time.Time{}, fmt.Errorf("时间解析失败: %w", err)
    }
    
    // Step 3: 合理性检查
    if err := h.validateTime(t); err != nil {
        return time.Time{}, err
    }
    
    return t, nil
}

// preprocessLLMOutput 预处理 LLM 输出
func (h *TimeHardener) preprocessLLMOutput(input string) string {
    // 中文数字转阿拉伯数字
    input = h.convertChineseNumbers(input)
    
    // 处理缺失日期的情况
    if h.hasTimeButNoDate(input) {
        input = h.inferDate(input)
    }
    
    // 标准化中文格式
    input = h.normalizeChineseFormat(input)
    
    return input
}

// convertChineseNumbers 中文数字转换
func (h *TimeHardener) convertChineseNumbers(input string) string {
    replacer := strings.NewReplacer(
        "一", "1", "二", "2", "三", "3", "四", "4", "五", "5",
        "六", "6", "七", "7", "八", "8", "九", "9", "十", "10",
        "十一", "11", "十二", "12",
    )
    return replacer.Replace(input)
}

// inferDate 推断日期
func (h *TimeHardener) inferDate(input string) string {
    // 如果只有时间没有日期，根据时间推断
    // 如果时间已过，推断为明天；否则为今天
    now := h.now()
    
    // 提取时间
    hour := h.extractHour(input)
    if hour < now.Hour() || (hour == now.Hour() && h.extractMinute(input) <= now.Minute()) {
        // 时间已过，推断为明天
        return "明天" + input
    }
    return "今天" + input
}

// validateTime 验证时间合理性
func (h *TimeHardener) validateTime(t time.Time) error {
    now := h.now()
    
    // 不能是过去的时间
    if t.Before(now) {
        return fmt.Errorf("时间不能早于现在")
    }
    
    // 不能太久远（1年内）
    oneYearLater := now.AddDate(1, 0, 0)
    if t.After(oneYearLater) {
        return fmt.Errorf("时间太远，请选择一年内的时间")
    }
    
    return nil
}
```

---

## 5. 交付物清单

### 5.1 代码文件

- [ ] `plugin/ai/schedule/time_hardener.go` - 时间加固层

### 5.2 测试文件

- [ ] `plugin/ai/schedule/time_hardener_test.go`

---

## 6. 测试验收

### 6.1 功能测试

| 场景 | 输入 | 预期输出 |
|:---|:---|:---|
| 中文格式 | "2026年1月28日下午3点" | 正确时间 |
| 缺失日期 | "下午3点" | 今天/明天下午3点 |
| 中文数字 | "下午三点" | 下午3点 |
| 英文混合 | "tomorrow 3pm" | 明天15:00 |

### 6.2 性能验收

| 指标 | 目标值 | 测试方法 |
|:---|:---|:---|
| 解析成功率 | > 98% | 测试用例集 |
| 解析延迟 | < 50ms | 单元测试 |

---

## 7. ROI 分析

| 维度 | 值 |
|:---|:---|
| 开发投入 | 2 人天 |
| 预期收益 | 日程创建成功率 +13% |
| 风险评估 | 低 |
| 回报周期 | Phase 1 结束 |

---

## 8. 实施计划

### 8.1 时间表

| 阶段 | 时间 | 任务 |
|:---|:---|:---|
| Day 1 | 1人天 | 预处理逻辑 |
| Day 2 | 1人天 | 测试用例 + 边界处理 |

### 8.2 检查点

- [ ] Day 1: 核心预处理完成
- [ ] Day 2: 成功率 > 98%

---

## 附录

### A. 参考资料

- [日程路线图 - 时间解析加固](../../research/schedule-roadmap.md)

### B. 变更记录

| 日期 | 版本 | 变更内容 | 作者 |
|:---|:---|:---|:---|
| 2026-01-27 | v1.0 | 初始版本 | - |
