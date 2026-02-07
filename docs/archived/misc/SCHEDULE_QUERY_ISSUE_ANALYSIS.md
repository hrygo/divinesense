# 日程查询功能缺失深度分析报告

**报告日期**: 2026-01-20
**问题描述**: 用户提问"我近期的日程"，助手未查询日程返回
**分析范围**: AI 聊天、日程系统集成
**严重程度**: 🔴 高优先级功能缺失

---

## 📋 问题复现

### 用户操作
1. 用户打开 AI 聊天界面
2. 用户输入："我近期的日程"
3. AI 助手回复：未返回日程信息

### 期望行为
- AI 助手应该查询用户的日程
- 返回近期（如未来7天）的日程列表
- 以友好的格式展示给用户

### 实际行为
- AI 只搜索用户笔记（memos）
- 没有查询日程表（schedules）
- 回复"没有相关信息"或基于笔记内容回答

---

## 🔍 根本原因分析

### 架构层面

#### 1. ChatWithMemos 功能范围限制 ⭐⭐⭐

**当前实现**: `server/router/api/v1/ai_service.go`

```go
// ChatWithMemos streams a chat response using memos as context.
func (s *AIService) ChatWithMemos(req *v1pb.ChatWithMemosRequest, stream v1pb.AIService_ChatWithMemosServer) error {
    // ...

    // 3. 两阶段检索：初步回捞 + Reranker 重排序
    queryVector, err := s.EmbeddingService.Embed(ctx, req.Message)
    if err != nil {
        return status.Errorf(codes.Internal, "failed to embed query: %v", err)
    }

    results, err := s.Store.VectorSearch(ctx, &store.VectorSearchOptions{
        UserID: user.ID,
        Vector: queryVector,
        Limit:  20, // ❌ 只搜索 memos，不搜索 schedules
    })
    // ...
}
```

**问题**:
- ✅ 向量搜索 `VectorSearch` 只搜索 `memos` 表
- ❌ 没有调用 `ListSchedules` 查询日程
- ❌ 日程数据不在 AI 上下文中

#### 2. Proto 定义缺少日程查询响应

**当前定义**: `proto/api/v1/ai_service.proto`

```protobuf
// ChatWithMemosResponse is the response for ChatWithMemos.
message ChatWithMemosResponse {
  string content = 1;           // streaming content chunk
  repeated string sources = 2;  // citation sources memos/{id}
  bool done = 3;                // stream end marker
  ScheduleCreationIntent schedule_creation_intent = 4;  // ❌ 只有创建意图，没有查询结果
}
```

**问题**:
- ❌ 缺少 `ScheduleQueryResult` 字段
- ❌ 无法返回查询到的日程列表
- ✅ 只有 `ScheduleCreationIntent`（创建意图检测）

#### 3. System Prompt 未包含日程查询指令

**当前 Prompt**: `server/router/api/v1/ai_service.go:398-423`

```go
systemPrompt = `你是一个基于用户个人笔记的AI助手。请根据以下笔记内容回答问题。

## 重要
在回复的最后，如果检测到用户想创建日程/提醒，请添加一行特殊标记：
<SCHEDULE_INTENT:{"detected":true,"description":"简短的日程描述"}>

如果没有创建意图，不要添加此标记。只有用户明确说"帮我创建"、"提醒我"、"安排"等表达时才添加。

## 回答要求
- 必须严格基于提供的笔记内容回答
- 不要编造或假设任何笔记中没有的信息
- 如果笔记中没有相关信息，请明确告知用户
- 使用中文，保持简洁准确`
```

**问题**:
- ❌ Prompt 强调"严格基于笔记内容"
- ❌ 没有提及日程查询能力
- ❌ AI 不知道可以访问日程数据

---

### 代码流程分析

#### 当前流程图

```
用户问"我近期的日程"
    ↓
AIChat.tsx → useChatWithMemos()
    ↓
ai_service.go → ChatWithMemos()
    ↓
EmbeddingService.Embed()  // 向量化查询
    ↓
Store.VectorSearch()      // ❌ 只搜索 memos 表
    ↓
构建上下文 (只有笔记)
    ↓
LLMService.ChatStream()   // AI 只看到笔记
    ↓
回复："没有找到相关信息"  // ❌ 日程不在上下文中
```

#### 缺失的环节

```
❌ Store.ListSchedules()  // 没有调用
❌ 日程数据注入上下文       // 没有实现
❌ Proto 返回日程         // 没有定义
```

---

## 💡 解决方案设计

### 方案 A: 扩展 ChatWithMemos 支持日程查询 ⭐⭐⭐⭐⭐

**推荐度**: ⭐⭐⭐⭐⭐ (最优方案)

#### 实施步骤

##### 1. 扩展 Proto 定义

```protobuf
// proto/api/v1/ai_service.proto

message ChatWithMemosResponse {
  string content = 1;
  repeated string sources = 2;
  bool done = 3;
  ScheduleCreationIntent schedule_creation_intent = 4;

  // ✅ 新增：日程查询结果
  ScheduleQueryResult schedule_query_result = 5;
}

// ✅ 新增：日程查询结果
message ScheduleQueryResult {
  bool detected = 1;                    // 是否检测到日程查询意图
  repeated Schedule schedules = 2;       // 查询到的日程列表
  string time_range = 3;                // 时间范围描述 (如"未来7天")
  string query_type = 4;                // 查询类型 (如"upcoming", "range")
}

// ✅ 新增：日程对象（简化版，避免循环依赖）
message Schedule {
  string uid = 1;
  string title = 2;
  int64 start_ts = 3;
  int64 end_ts = 4;
  string recurrence_rule = 5;           // JSON string
}
```

##### 2. 后端实现日程查询逻辑

```go
// server/router/api/v1/ai_service.go

func (s *AIService) ChatWithMemos(req *v1pb.ChatWithMemosRequest, stream v1pb.AIService_ChatWithMemosServer) error {
    // ... 现有代码 ...

    // ✅ 新增：检测日程查询意图
    scheduleQueryIntent := s.detectScheduleQueryIntent(req.Message)
    var scheduleQueryResult *v1pb.ScheduleQueryResult

    if scheduleQueryIntent.Detected {
        // ✅ 查询日程
        schedules, err := s.querySchedules(ctx, user.ID, scheduleQueryIntent)
        if err != nil {
            // 日程查询失败不影响聊天，记录日志
            fmt.Printf("Failed to query schedules: %v\n", err)
        } else {
            scheduleQueryResult = schedules
        }
    }

    // ✅ 将日程信息注入上下文
    if scheduleQueryResult != nil && len(scheduleQueryResult.Schedules) > 0 {
        contextBuilder.WriteString(fmt.Sprintf("\n## 用户日程\n"))
        for _, sched := range scheduleQueryResult.Schedules {
            startTime := time.Unix(sched.StartTs, 0).Format("2006-01-02 15:04")
            contextBuilder.WriteString(fmt.Sprintf("- %s: %s (%s)\n",
                startTime, sched.Title, sched.Uid))
        }
        contextBuilder.WriteString("\n")
    }

    // ... 继续发送响应 ...

    // ✅ 在最终响应中返回日程
    if scheduleQueryResult != nil {
        return stream.Send(&v1pb.ChatWithMemosResponse{
            Done: true,
            ScheduleQueryResult: scheduleQueryResult,
        })
    }
}

// ✅ 新增：检测日程查询意图
func (s *AIService) detectScheduleQueryIntent(message string) *v1pb.ScheduleQueryIntent {
    // 简单规则检测（也可以用 LLM）
    keywords := []string{
        "日程", "近期", "今天", "明天", "本周", "下周",
        "安排", "计划", "提醒", "有什么", "查询",
    }

    for _, kw := range keywords {
        if strings.Contains(message, kw) {
            return &v1pb.ScheduleQueryIntent{
                Detected:  true,
                QueryType: "upcoming",
                TimeRange: "7d", // 默认查询未来7天
            }
        }
    }
    return &v1pb.ScheduleQueryIntent{Detected: false}
}

// ✅ 新增：查询日程
func (s *AIService) querySchedules(ctx context.Context, userID int32, intent *v1pb.ScheduleQueryIntent) (*v1pb.ScheduleQueryResult, error) {
    // 计算时间范围
    now := time.Now()
    startTime := now
    endTime := now.Add(7 * 24 * time.Hour) // 未来7天

    // 查询日程
    find := &store.FindSchedule{
        CreatorID: &userID,
        StartTime: &startTime,
        EndTime:   &endTime,
    }

    schedules, err := s.Store.ListSchedules(ctx, find)
    if err != nil {
        return nil, err
    }

    // 转换为 proto 格式
    result := &v1pb.ScheduleQueryResult{
        Detected:  true,
        Schedules: make([]*v1pb.Schedule, len(schedules)),
        TimeRange: "未来7天",
        QueryType: intent.QueryType,
    }

    for i, sched := range schedules {
        result.Schedules[i] = &v1pb.Schedule{
            Uid:            "schedules/" + sched.UID,
            Title:          sched.Title,
            StartTs:        sched.StartTs,
            EndTs:          *sched.EndTs,
            RecurrenceRule: sched.RecurrenceRule,
        }
    }

    return result, nil
}
```

##### 3. 更新 System Prompt

```go
systemPrompt = `你是一个基于用户笔记和日程的AI助手。

## 功能
1. 笔记查询：基于用户笔记回答问题
2. 日程查询：查询用户近期日程安排
3. 日程创建：帮助用户创建新日程

## 日程相关
- 如果用户询问"近期日程"、"今天安排"等，请基于下方"用户日程"部分回答
- 日程信息已经包含在上下文中，直接引用即可
- 如果没有找到日程，请告知用户没有相关日程

## 回答要求
- 使用中文，简洁准确
- 严格基于提供的上下文回答
- 不要编造信息`
```

##### 4. 前端展示日程查询结果

```typescript
// web/src/pages/AIChat.tsx

// 在消息处理中添加日程查询结果展示
const handleScheduleQueryResult = (scheduleQueryResult: ScheduleQueryResult) => {
  if (scheduleQueryResult.detected && scheduleQueryResult.schedules.length > 0) {
    setQueryResultSchedules(scheduleQueryResult.schedules);
    setQueryTitle(scheduleQueryResult.timeRange || "近期日程");
    setShowScheduleQueryResult(true);
  }
};

// 在 useChatWithMemos callback 中处理
stream: async (params, callbacks) => {
  await chatHook.stream(params, {
    ...callbacks,
    onScheduleQueryResult: (result) => {
      handleScheduleQueryResult(result);
    },
  });
}
```

#### 优点
- ✅ 统一的聊天入口，用户体验好
- ✅ AI 可以结合笔记和日程回答
- ✅ 支持复杂查询（如"下周有哪些会议"）
- ✅ 可以扩展更多功能

#### 缺点
- ⚠️ 需要修改多个文件
- ⚠️ 测试工作量大

---

### 方案 B: 独立的日程查询接口 ⭐⭐⭐

**推荐度**: ⭐⭐⭐ (备选方案)

#### 实施步骤

##### 1. 添加新的 API

```protobuf
// proto/api/v1/schedule_service.proto

service ScheduleService {
  // ... 现有方法 ...

  // ✅ 新增：智能查询日程
  rpc QuerySchedulesWithAI(QuerySchedulesWithAIRequest) returns (QuerySchedulesWithAIResponse) {
    option (google.api.http) = {
      post: "/api/v1/schedules/query-ai"
      body: "*"
    };
  }
}

message QuerySchedulesWithAIRequest {
  string query = 1;  // 自然语言查询，如"下周的会议"
}

message QuerySchedulesWithAIResponse {
  repeated Schedule schedules = 1;
  string explanation = 2;  // AI 解释查询结果
}
```

##### 2. 前端调用新接口

```typescript
// 用户问"我近期的日程"
// 前端调用 /api/v1/schedules/query-ai
// 后端使用 LLM 解析意图并查询日程
```

#### 优点
- ✅ 职责分离，不影响现有 ChatWithMemos
- ✅ 实现简单，测试容易
- ✅ 可以独立优化

#### 缺点
- ❌ 需要用户在聊天和专门接口间切换
- ❌ AI 无法结合笔记和日程回答
- ❌ 用户体验较差

---

### 方案 C: 混合方案 ⭐⭐⭐⭐

**推荐度**: ⭐⭐⭐⭐ (平衡方案)

#### 实施步骤
1. **短期** (方案B)：先实现独立接口，快速上线
2. **长期** (方案A)：逐步集成到 ChatWithMemos

#### 优点
- ✅ 快速交付价值
- ✅ 逐步完善功能
- ✅ 降低风险

---

## 📊 方案对比

| 维度 | 方案 A (扩展) | 方案 B (独立) | 方案 C (混合) |
|------|-------------|-------------|-------------|
| **用户体验** | ⭐⭐⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐⭐⭐ |
| **实现复杂度** | ⭐⭐ | ⭐⭐⭐⭐⭐ | ⭐⭐⭐ |
| **测试工作量** | ⭐⭐⭐ | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐ |
| **可扩展性** | ⭐⭐⭐⭐⭐ | ⭐⭐ | ⭐⭐⭐⭐ |
| **交付速度** | ⭐⭐ | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐ |
| **维护成本** | ⭐⭐⭐ | ⭐⭐⭐⭐ | ⭐⭐⭐ |
| **总评分** | **4.0/5** | **3.2/5** | **3.8/5** |

---

## 🎯 推荐实施计划

### 阶段 1: 快速修复（1周）⚡
**目标**: 让用户能查询日程

1. ✅ 实现方案 B：独立日程查询接口
2. ✅ 前端添加快捷按钮"查询日程"
3. ✅ 基础测试

**产出**:
- `QuerySchedulesWithAI` API
- 用户可以通过专门入口查询日程

---

### 阶段 2: 深度集成（2周）🚀
**目标**: AI 聊天能理解日程查询

1. ✅ 实现方案 A：扩展 ChatWithMemos
2. ✅ 添加日程查询意图检测
3. ✅ 更新 System Prompt
4. ✅ 完整测试

**产出**:
- 用户可以直接在聊天中问"我近期的日程"
- AI 返回日程结果
- AI 可以结合笔记和日程回答

---

### 阶段 3: 智能增强（1周）✨
**目标**: 更智能的日程交互

1. ✅ 支持 LLM 解析复杂查询
2. ✅ 支持日程编辑意图检测
3. ✅ 支持日程删除意图检测
4. ✅ 性能优化

**产出**:
- "下周二下午把会议改到3点"
- "删除明天上午的提醒"
- 响应时间 < 2s

---

## ⚠️ 风险与挑战

### 技术风险
1. **性能风险**: 日程查询可能增加响应时间
   - **缓解**: 并行查询日程和笔记，缓存结果

2. **意图识别准确率**: 规则检测可能误判
   - **缓解**: 使用 LLM 进行意图分析，添加 fallback

3. **上下文窗口限制**: 日程信息占用 token
   - **缓解**: 限制返回日程数量（最多 10 条）

### 产品风险
1. **用户期望管理**: 用户期望 AI 能处理所有日程操作
   - **缓解**: 明确告知当前能力范围

2. **功能边界**: 日程查询 vs 日程管理
   - **缓解**: 专注查询功能，管理操作使用现有界面

---

## 📝 代码改动估算

### 方案 A: 扩展 ChatWithMemos

| 文件 | 改动行数 | 复杂度 |
|------|---------|--------|
| `proto/api/v1/ai_service.proto` | +30 | 中 |
| `server/router/api/v1/ai_service.go` | +150 | 高 |
| `web/src/hooks/useAIQueries.ts` | +20 | 低 |
| `web/src/pages/AIChat.tsx` | +50 | 中 |
| `web/src/components/AIChat/ScheduleQueryResult.tsx` | +100 | 中 |
| **总计** | **~350 行** | **中高** |

**工作量**: 3-5 天

---

## 🔧 技术细节

### 日程查询意图检测策略

#### 策略 1: 规则匹配（快速）⚡

```go
func (s *AIService) detectScheduleQueryIntent(message string) *ScheduleQueryIntent {
    queryPatterns := map[string]string{
        "近期日程": "upcoming:7d",
        "今天安排": "range:today",
        "明天.*安排": "range:tomorrow",
        "下周.*会议": "filter:week+1,meeting",
        // ... 更多规则
    }

    for pattern, queryType := range queryPatterns {
        if matched, _ := regexp.MatchString(pattern, message); matched {
            return &ScheduleQueryIntent{
                Detected:  true,
                QueryType: queryType,
            }
        }
    }
    return nil
}
```

**优点**: 快速、可控
**缺点**: 维护成本高、覆盖有限

#### 策略 2: LLM 分析（智能）🧠

```go
func (s *AIService) detectScheduleQueryIntentWithLLM(ctx context.Context, message string) *ScheduleQueryIntent {
    prompt := fmt.Sprintf(`分析用户问题是否为日程查询。

用户问题：%s

返回 JSON：
{
  "detected": true/false,
  "query_type": "upcoming/range/filter",
  "time_range": "7d/today/tomorrow/week",
  "filters": ["meeting", "urgent"]
}`, message)

    response, err := s.LLMService.Chat(ctx, []ai.Message{
        {Role: "user", Content: prompt},
    })
    // ... 解析响应
}
```

**优点**: 灵活、覆盖广
**缺点**: 额外 LLM 调用、成本高

**推荐**: 混合策略（规则优先，LLM fallback）

---

## 📈 预期效果

### 用户体验提升
- ✅ 自然语言查询日程
- ✅ 统一的 AI 交互入口
- ✅ 智能意图识别

### 功能完善度
| 功能 | 当前 | 实施后 |
|------|------|--------|
| 笔记查询 | ✅ | ✅ |
| 日程创建 | ✅ | ✅ |
| 日程查询 | ❌ | ✅ |
| 日程编辑 | ⚠️ (手动) | ⚠️ (手动) |
| 日程删除 | ⚠️ (手动) | ⚠️ (手动) |

---

## 🎓 总结

### 核心问题
**ChatWithMemos 只搜索笔记，不搜索日程**

### 根本原因
1. 架构设计时只考虑了笔记查询
2. 缺少日程查询的 API 定义
3. System Prompt 未包含日程指令
4. 没有日程查询意图检测逻辑

### 推荐方案
**方案 A: 扩展 ChatWithMemos**（最优）+ **方案 C: 混合**（平衡）

### 实施优先级
1. 🔴 高优先级：日程查询功能缺失
2. ⏱️ 预计工期：3-5 周（分阶段）
3. 🎯 预期收益：用户体验显著提升

---

**报告完成日期**: 2026-01-20
**下次审查**: 实施方案确定后
