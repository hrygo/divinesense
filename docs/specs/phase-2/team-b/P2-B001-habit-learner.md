# P2-B001: 用户习惯学习系统

> **状态**: 🔲 待开发  
> **优先级**: P1 (重要)  
> **投入**: 5 人天  
> **负责团队**: 团队 B  
> **Sprint**: Sprint 3

---

## 1. 目标与背景

### 1.1 核心目标

基于历史交互数据自动学习用户习惯（时间偏好、日程习惯、常用关键词），实现"越用越懂你"的个性化体验。

### 1.2 用户价值

- 减少用户操作 30%+（自动填充偏好）
- 智能推荐时间段
- 打造"懂我"的私人助理体验

### 1.3 技术价值

- 无 LLM 调用（纯模式分析）
- 为预测性交互（P3-B001）奠定基础
- 差异化竞争力

---

## 2. 依赖关系

### 2.1 前置依赖

- [x] P1-A001: 轻量记忆系统（情景记忆数据源）
- [x] P1-A002: 基础评估指标（交互记录）

### 2.2 并行依赖

- P2-B002: 快速创建模式（可并行）

### 2.3 后续依赖

- P3-B001: 预测性交互系统
- P3-B002: 主动提醒系统

---

## 3. 功能设计

### 3.1 架构图

```
                    习惯学习系统架构
┌────────────────────────────────────────────────────────────┐
│                                                            │
│   ┌────────────────────────────────────────────────────┐  │
│   │              HabitLearner (后台任务)                 │  │
│   │                                                     │  │
│   │  每日运行 → 分析 30 天数据 → 更新用户偏好            │  │
│   └────────────────────────────────────────────────────┘  │
│                            │                               │
│          ┌─────────────────┼─────────────────┐            │
│          │                 │                 │            │
│          ▼                 ▼                 ▼            │
│   ┌──────────────┐ ┌──────────────┐ ┌──────────────┐      │
│   │  时间习惯    │ │  日程习惯    │ │  搜索习惯    │      │
│   │              │ │              │ │              │      │
│   │ • 活跃时段   │ │ • 默认时长   │ │ • 常用关键词 │      │
│   │ • 偏好时间   │ │ • 偏好时段   │ │ • 搜索模式   │      │
│   │ • 工作日模式 │ │ • 常用地点   │ │ • 结果偏好   │      │
│   └──────────────┘ └──────────────┘ └──────────────┘      │
│          │                 │                 │            │
│          └─────────────────┼─────────────────┘            │
│                            ▼                               │
│                  ┌─────────────────────┐                   │
│                  │   UserPreferences   │                   │
│                  │   (JSONB 存储)      │                   │
│                  └─────────────────────┘                   │
│                                                            │
│   CPU 开销: 每日 1 次分析 | 存储: ~5KB/用户               │
│                                                            │
└────────────────────────────────────────────────────────────┘
```

### 3.2 可学习的习惯维度

```go
// plugin/ai/habit/dimensions.go

// 时间习惯
type TimeHabits struct {
    ActiveHours     []int     `json:"active_hours"`      // 活跃小时 [9, 10, 14, 15]
    PreferredTimes  []string  `json:"preferred_times"`   // ["09:00", "14:00"]
    ReminderLeadMin int       `json:"reminder_lead_min"` // 提醒提前量（分钟）
    WeekdayPattern  bool      `json:"weekday_pattern"`   // 工作日模式
}

// 日程习惯
type ScheduleHabits struct {
    DefaultDuration   int      `json:"default_duration"`   // 默认会议时长（分钟）
    PreferredSlots    []string `json:"preferred_slots"`    // 偏好时间段
    FrequentLocations []string `json:"frequent_locations"` // 常用地点
    TitlePatterns     []string `json:"title_patterns"`     // 常见标题模式
}

// 搜索习惯
type SearchHabits struct {
    FrequentKeywords []string `json:"frequent_keywords"` // 常用关键词
    SearchMode       string   `json:"search_mode"`       // "exact" / "fuzzy"
    ResultPreference string   `json:"result_preference"` // 偏好笔记类型
}

// 聚合习惯
type UserHabits struct {
    Time     *TimeHabits     `json:"time"`
    Schedule *ScheduleHabits `json:"schedule"`
    Search   *SearchHabits   `json:"search"`
    UpdatedAt time.Time      `json:"updated_at"`
}
```

### 3.3 习惯分析器

```go
// plugin/ai/habit/analyzer.go

type HabitAnalyzer interface {
    // 分析用户习惯
    Analyze(ctx context.Context, userID int32) (*UserHabits, error)
}

type habitAnalyzer struct {
    memoryService MemoryService
    lookbackDays  int  // 默认 30 天
}

func NewHabitAnalyzer(memSvc MemoryService) HabitAnalyzer {
    return &habitAnalyzer{
        memoryService: memSvc,
        lookbackDays:  30,
    }
}

func (a *habitAnalyzer) Analyze(ctx context.Context, userID int32) (*UserHabits, error) {
    // 获取历史交互记录
    since := time.Now().AddDate(0, 0, -a.lookbackDays)
    episodes, err := a.memoryService.GetEpisodicMemories(ctx, userID, since)
    if err != nil {
        return nil, err
    }
    
    // 过滤成功的交互
    successEpisodes := filterSuccessful(episodes)
    
    if len(successEpisodes) < 10 {
        // 数据不足，返回默认值
        return defaultHabits(), nil
    }
    
    // 并行分析各维度
    timeHabits := a.analyzeTimeHabits(successEpisodes)
    scheduleHabits := a.analyzeScheduleHabits(successEpisodes)
    searchHabits := a.analyzeSearchHabits(successEpisodes)
    
    return &UserHabits{
        Time:      timeHabits,
        Schedule:  scheduleHabits,
        Search:    searchHabits,
        UpdatedAt: time.Now(),
    }, nil
}
```

### 3.4 时间习惯分析

```go
// plugin/ai/habit/time_analyzer.go

func (a *habitAnalyzer) analyzeTimeHabits(episodes []*EpisodicMemory) *TimeHabits {
    // 统计活跃小时分布
    hourCounts := make(map[int]int)
    weekdayCount := 0
    weekendCount := 0
    
    for _, ep := range episodes {
        hour := ep.Timestamp.Hour()
        hourCounts[hour]++
        
        if ep.Timestamp.Weekday() >= time.Monday && ep.Timestamp.Weekday() <= time.Friday {
            weekdayCount++
        } else {
            weekendCount++
        }
    }
    
    // 找出 Top-5 活跃小时
    activeHours := topNHours(hourCounts, 5)
    
    // 推断偏好时间点
    preferredTimes := inferPreferredTimes(hourCounts)
    
    return &TimeHabits{
        ActiveHours:    activeHours,
        PreferredTimes: preferredTimes,
        WeekdayPattern: weekdayCount > weekendCount*2, // 工作日为主
    }
}

func inferPreferredTimes(hourCounts map[int]int) []string {
    // 找出峰值时间
    peaks := findPeaks(hourCounts)
    
    var times []string
    for _, hour := range peaks {
        times = append(times, fmt.Sprintf("%02d:00", hour))
    }
    return times
}

func findPeaks(hourCounts map[int]int) []int {
    // 简单峰值检测：超过平均值 1.5 倍的小时
    total := 0
    for _, count := range hourCounts {
        total += count
    }
    avg := total / max(len(hourCounts), 1)
    threshold := int(float64(avg) * 1.5)
    
    var peaks []int
    for hour, count := range hourCounts {
        if count >= threshold {
            peaks = append(peaks, hour)
        }
    }
    sort.Ints(peaks)
    return peaks
}
```

### 3.5 日程习惯分析

```go
// plugin/ai/habit/schedule_analyzer.go

func (a *habitAnalyzer) analyzeScheduleHabits(episodes []*EpisodicMemory) *ScheduleHabits {
    // 过滤日程相关交互
    scheduleEpisodes := filterByAgentType(episodes, "schedule")
    
    if len(scheduleEpisodes) < 5 {
        return defaultScheduleHabits()
    }
    
    // 分析默认时长
    durations := extractDurations(scheduleEpisodes)
    defaultDuration := calculateMedian(durations)
    
    // 分析常用地点
    locations := extractLocations(scheduleEpisodes)
    frequentLocations := topNStrings(locations, 3)
    
    // 分析偏好时段
    slots := extractTimeSlots(scheduleEpisodes)
    preferredSlots := topNStrings(slots, 3)
    
    return &ScheduleHabits{
        DefaultDuration:   defaultDuration,
        PreferredSlots:    preferredSlots,
        FrequentLocations: frequentLocations,
    }
}

func extractDurations(episodes []*EpisodicMemory) []int {
    var durations []int
    // 从 episode metadata 中提取时长
    for _, ep := range episodes {
        if duration, ok := ep.Metadata["duration"].(int); ok {
            durations = append(durations, duration)
        }
    }
    return durations
}

func calculateMedian(values []int) int {
    if len(values) == 0 {
        return 60 // 默认 1 小时
    }
    sort.Ints(values)
    return values[len(values)/2]
}
```

### 3.6 搜索习惯分析

```go
// plugin/ai/habit/search_analyzer.go

func (a *habitAnalyzer) analyzeSearchHabits(episodes []*EpisodicMemory) *SearchHabits {
    // 过滤搜索相关交互
    searchEpisodes := filterByAgentType(episodes, "memo")
    
    if len(searchEpisodes) < 5 {
        return defaultSearchHabits()
    }
    
    // 提取常用关键词
    keywords := extractKeywords(searchEpisodes)
    frequentKeywords := topNStrings(keywords, 10)
    
    // 分析搜索模式
    searchMode := inferSearchMode(searchEpisodes)
    
    return &SearchHabits{
        FrequentKeywords: frequentKeywords,
        SearchMode:       searchMode,
    }
}

func extractKeywords(episodes []*EpisodicMemory) []string {
    var keywords []string
    for _, ep := range episodes {
        // 简单分词提取关键词
        words := tokenize(ep.UserInput)
        keywords = append(keywords, words...)
    }
    return keywords
}

func inferSearchMode(episodes []*EpisodicMemory) string {
    exactCount := 0
    fuzzyCount := 0
    
    for _, ep := range episodes {
        if hasExactQuotes(ep.UserInput) {
            exactCount++
        } else {
            fuzzyCount++
        }
    }
    
    if exactCount > fuzzyCount {
        return "exact"
    }
    return "fuzzy"
}
```

### 3.7 后台学习任务

```go
// plugin/ai/habit/learner.go

type HabitLearner struct {
    analyzer      HabitAnalyzer
    memoryService MemoryService
    ticker        *time.Ticker
}

func NewHabitLearner(analyzer HabitAnalyzer, memSvc MemoryService) *HabitLearner {
    return &HabitLearner{
        analyzer:      analyzer,
        memoryService: memSvc,
    }
}

func (l *HabitLearner) Start(ctx context.Context) {
    // 每天凌晨 2 点运行
    l.ticker = time.NewTicker(24 * time.Hour)
    
    // 启动时立即运行一次
    go l.runAnalysis(ctx)
    
    go func() {
        for {
            select {
            case <-ctx.Done():
                l.ticker.Stop()
                return
            case <-l.ticker.C:
                l.runAnalysis(ctx)
            }
        }
    }()
}

func (l *HabitLearner) runAnalysis(ctx context.Context) {
    slog.Info("starting habit analysis")
    
    // 获取所有活跃用户
    userIDs, err := l.memoryService.GetActiveUsers(ctx, 30) // 30天内活跃
    if err != nil {
        slog.Error("failed to get active users", "error", err)
        return
    }
    
    for _, userID := range userIDs {
        habits, err := l.analyzer.Analyze(ctx, userID)
        if err != nil {
            slog.Error("failed to analyze habits", "user_id", userID, "error", err)
            continue
        }
        
        // 更新用户偏好
        err = l.memoryService.UpdateUserPreferences(ctx, userID, habits.ToPreferences())
        if err != nil {
            slog.Error("failed to update preferences", "user_id", userID, "error", err)
        }
    }
    
    slog.Info("habit analysis completed", "users_processed", len(userIDs))
}

// 习惯转换为用户偏好
func (h *UserHabits) ToPreferences() *UserPreferences {
    return &UserPreferences{
        PreferredTimes:    h.Time.PreferredTimes,
        DefaultDuration:   h.Schedule.DefaultDuration,
        FrequentLocations: h.Schedule.FrequentLocations,
        TagPreferences:    h.Search.FrequentKeywords,
    }
}
```

### 3.8 习惯应用

```go
// plugin/ai/habit/applier.go

type HabitApplier struct {
    memoryService MemoryService
}

// 应用习惯到日程创建
func (a *HabitApplier) ApplyToScheduleCreate(ctx context.Context, userID int32, input *ScheduleInput) *ScheduleInput {
    prefs, _ := a.memoryService.GetUserPreferences(ctx, userID)
    if prefs == nil {
        return input
    }
    
    // 自动填充默认时长
    if input.Duration == 0 && prefs.DefaultDuration > 0 {
        input.Duration = prefs.DefaultDuration
    }
    
    // 自动推荐时间
    if input.StartTime.IsZero() && len(prefs.PreferredTimes) > 0 {
        input.SuggestedTimes = prefs.PreferredTimes
    }
    
    // 自动填充常用地点
    if input.Location == "" && len(prefs.FrequentLocations) > 0 {
        input.SuggestedLocations = prefs.FrequentLocations
    }
    
    return input
}

// 应用习惯到时间推断
func (a *HabitApplier) InferTime(ctx context.Context, userID int32, query string) time.Time {
    prefs, _ := a.memoryService.GetUserPreferences(ctx, userID)
    if prefs == nil {
        return time.Time{}
    }
    
    // 如果只说了"下午"，推荐偏好时间
    if containsAfternoon(query) && len(prefs.PreferredTimes) > 0 {
        for _, t := range prefs.PreferredTimes {
            if isAfternoon(t) {
                return parseTime(t)
            }
        }
    }
    
    return time.Time{}
}
```

---

## 4. 实现路径

### Day 1-2: 习惯分析器

- [ ] 实现 `HabitAnalyzer` 接口
- [ ] 时间习惯分析
- [ ] 日程习惯分析
- [ ] 搜索习惯分析

### Day 3: 后台学习任务

- [ ] 实现 `HabitLearner`
- [ ] 定时任务调度
- [ ] 用户偏好更新

### Day 4: 习惯应用

- [ ] 实现 `HabitApplier`
- [ ] 集成到 ScheduleAgent
- [ ] 集成到时间解析

### Day 5: 测试与优化

- [ ] 单元测试
- [ ] 集成测试
- [ ] 性能优化

---

## 5. 交付物

### 5.1 代码产出

| 文件 | 说明 |
|:---|:---|
| `plugin/ai/habit/dimensions.go` | 习惯数据结构 |
| `plugin/ai/habit/analyzer.go` | 习惯分析器 |
| `plugin/ai/habit/time_analyzer.go` | 时间习惯分析 |
| `plugin/ai/habit/schedule_analyzer.go` | 日程习惯分析 |
| `plugin/ai/habit/search_analyzer.go` | 搜索习惯分析 |
| `plugin/ai/habit/learner.go` | 后台学习任务 |
| `plugin/ai/habit/applier.go` | 习惯应用 |
| `plugin/ai/habit/*_test.go` | 单元测试 |

### 5.2 配置项

```yaml
# configs/ai.yaml
habit_learner:
  lookback_days: 30
  min_samples: 10
  run_hour: 2  # 凌晨 2 点
  
  thresholds:
    peak_multiplier: 1.5
    min_keyword_frequency: 3
```

---

## 6. 验收标准

### 6.1 功能验收

- [ ] 时间习惯正确识别活跃时段
- [ ] 日程习惯正确计算默认时长
- [ ] 搜索习惯正确提取常用关键词
- [ ] 后台任务每日正常运行

### 6.2 性能验收

- [ ] 单用户分析 < 500ms
- [ ] 100 用户批量分析 < 1分钟
- [ ] 无 LLM 调用（纯本地计算）

### 6.3 测试用例

```go
func TestTimeHabitAnalysis(t *testing.T) {
    episodes := generateMockEpisodes(100, []int{9, 10, 14, 15})
    
    analyzer := &habitAnalyzer{}
    timeHabits := analyzer.analyzeTimeHabits(episodes)
    
    // 应该识别出 9, 10, 14, 15 为活跃时段
    assert.Contains(t, timeHabits.ActiveHours, 9)
    assert.Contains(t, timeHabits.ActiveHours, 14)
}

func TestScheduleHabitAnalysis(t *testing.T) {
    episodes := generateScheduleEpisodes(50, 60) // 50 条，平均 60 分钟
    
    analyzer := &habitAnalyzer{}
    scheduleHabits := analyzer.analyzeScheduleHabits(episodes)
    
    // 默认时长应该接近 60 分钟
    assert.InDelta(t, 60, scheduleHabits.DefaultDuration, 10)
}
```

---

## 7. ROI 分析

| 投入 | 产出 |
|:---|:---|
| 开发: 5 人天 | 用户操作减少 30% |
| 存储: ~5KB/用户 | 打造"懂我"体验 |
| CPU: 每日 1 次后台分析 | 产品差异化竞争力 |

### 收益计算

- "开会" → 自动推断 1 小时（减少 1 次交互）
- "明天下午" → 自动推荐 14:00（减少 1 次确认）
- 每次交互节省约 10 秒，每日 10 次 = 100 秒/天

---

## 8. 风险与缓解

| 风险 | 概率 | 影响 | 缓解措施 |
|:---|:---:|:---:|:---|
| 数据不足 | 中 | 低 | 设置最小样本量，返回默认值 |
| 习惯变化 | 中 | 低 | 30天滚动窗口自动适应 |
| 隐私顾虑 | 低 | 中 | 本地分析，不上传云端 |

---

## 9. 排期

| 日期 | 任务 | 负责人 |
|:---|:---|:---|
| Sprint 3 Day 1-2 | 习惯分析器 | TBD |
| Sprint 3 Day 3 | 后台学习任务 | TBD |
| Sprint 3 Day 4 | 习惯应用 | TBD |
| Sprint 3 Day 5 | 测试与优化 | TBD |

---

> **纲领来源**: [00-master-roadmap.md](../../../archived/research_20250202/00-master-roadmap.md)  
> **研究文档**: [assistant-roadmap.md](../../../research/assistant-roadmap.md)  
> **版本**: v1.0  
> **更新时间**: 2026-01-27
