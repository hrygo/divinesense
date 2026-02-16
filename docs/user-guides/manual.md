# E2E 用户测试手册

> **测试范围**: MemoParrot (灰灰) + ScheduleParrot (时巧) + Orchestrator
> **系统版本**: 基于 2026-02-16 架构
> **测试级别**: L1 逻辑集成 + L2 真实 E2E（需要真实数据库和 LLM）
> **测试策略**: 完整覆盖核心功能、编排协作、上下文工程、可观测性

---

## 测试前准备

### 1. 环境检查

```bash
# 确认服务运行中
make status

# 确认 PostgreSQL 可用
docker exec -it divinesense-postgres-dev psql -U divinesense -c "SELECT 1"

# 确认 LLM 配置（可选，用于 L2 测试）
cat .env | grep -E "LLM_|ZHIYUAN_"
```

### 2. 开启 L2 测试模式

```bash
# 必须在 CI 环境变量未设置的情况下
export ENABLE_MANUAL_E2E=true
export CI=  # 确保 CI 未设置
```

---

## 测试用例

### 第一部分：MemoParrot (灰灰) 测试

#### TC-MEMO-001: 关键词搜索

**测试目标**: 验证笔记搜索专家能正确执行关键词搜索

**操作步骤**:
1. 打开 AI Chat 界面
2. 选择或创建一个对话
3. 输入: `查找关于 Go 语言的笔记`
4. 观察返回结果

**预期结果**:
- 返回包含 "Go" 关键词的笔记
- 显示相关度分数
- 响应时间 < 3s

**验证点**:
- [ ] 结果包含 Go 相关笔记
- [ ] 笔记内容正确显示
- [ ] 可以点击跳转到笔记详情

---

#### TC-MEMO-002: 语义向量搜索

**测试目标**: 验证语义搜索能力

**前置条件**: 已有至少 5 条包含 "编程" 标签的笔记

**操作步骤**:
1. 输入: `我之前记录的编程学习资料`
2. 观察返回结果

**预期结果**:
- 返回语义相关的结果（即使不包含 "编程" 关键词）
- 结果按相关度排序

**验证点**:
- [ ] 返回与 "编程学习" 语义相关的笔记
- [ ] 不包含关键词但相关内容也被召回

---

#### TC-MEMO-003: 时间过滤搜索

**测试目标**: 验证时间过滤器

**操作步骤**:
1. 输入: `查看今天的笔记`
2. 输入: `查看上周的笔记`
3. 输入: `查看最近7天的笔记`

**预期结果**:
| 输入 | 预期时间范围 |
|------|-------------|
| 今天 | 今日 00:00 ~ 现在 |
| 昨天 | 昨日 00:00 ~ 24:00 |
| 本周 | 本周周一 ~ 现在 |
| 上周 | 上周周一 ~ 周日 |
| 最近7天 | 7天前 ~ 现在 |

**验证点**:
- [ ] "今天的笔记" 只返回今天创建的笔记
- [ ] "上周的笔记" 正确返回上周的笔记
- [ ] 跨日期边界查询正确

---

#### TC-MEMO-004: 标签过滤搜索

**测试目标**: 验证标签过滤功能

**前置条件**: 已有带标签的笔记

**操作步骤**:
1. 输入: `查看 #work 标签的笔记`
2. 输入: `查看关于 Python 且带 #学习 标签的笔记`

**预期结果**:
- 返回指定标签的笔记
- 支持组合条件查询

**验证点**:
- [ ] 单标签过滤正确
- [ ] 多条件组合查询正确

---

#### TC-MEMO-005: 无结果处理

**测试目标**: 验证无搜索结果时的友好提示

**操作步骤**:
1. 输入: `查找一个不存在的关键词 xyzabc123`

**预期结果**:
- 返回友好的无结果提示
- 建议用户尝试其他搜索词

**验证点**:
- [ ] 不返回空结果
- [ ] 有友好的提示信息

---

### 第二部分：ScheduleParrot (时巧) 测试

#### TC-SCHEDULE-001: 创建日程（简单）

**测试目标**: 验证日程创建功能

**操作步骤**:
1. 输入: `下周五下午三点安排团队会议`
2. 观察解析结果
3. 确认创建

**预期结果**:
- 正确解析相对时间 "下周五下午三点"
- 显示创建确认
- 日程成功保存到数据库

**验证点**:
- [ ] 时间解析正确（下周五 15:00）
- [ ] 返回创建成功的日程信息
- [ ] 数据库中可查询到该日程

---

#### TC-SCHEDULE-002: 创建日程（含冲突检测）

**测试目标**: 验证日程冲突检测

**前置条件**: 已存在 14:00-16:00 的会议

**操作步骤**:
1. 输入: `今天下午三点安排项目评审`
2. 观察冲突提示

**预期结果**:
- 检测到与现有日程冲突
- 显示冲突详情（冲突的日程标题和时间）
- 提供解决方案建议

**验证点**:
- [ ] 正确检测时间冲突
- [ ] 显示冲突的日程信息
- [ ] 建议调整时间或继续创建

---

#### TC-SCHEDULE-003: 查询今日日程

**测试目标**: 验证日程查询功能

**前置条件**: 已有今日日程

**操作步骤**:
1. 输入: `今天有什么安排？`
2. 输入: `查看今天的日程`

**预期结果**:
- 返回今日所有日程
- 按时间排序
- 显示日程标题、时间、地点

**验证点**:
- [ ] 返回今日所有日程
- [ ] 时间排序正确
- [ ] 信息完整（标题、时间、地点）

---

#### TC-SCHEDULE-004: 查询本周日程

**测试目标**: 验证周期查询

**操作步骤**:
1. 输入: `这周有什么安排？`
2. 输入: `查看下周一的日程`

**预期结果**:
- 返回本周/下周所有日程
- 按日期分组显示

**验证点**:
- [ ] 正确返回本周/下周日程
- [ ] 按日期正确分组
- [ ] 不包含非本周日程

---

#### TC-SCHEDULE-005: 修改日程

**测试目标**: 验证日程更新功能

**前置条件**: 已有待修改的日程

**操作步骤**:
1. 输入: `把团队会议改到下周三下午四点`
2. 观察修改结果

**预期结果**:
- 正确识别要修改的日程
- 更新时间
- 返回更新后的日程信息

**验证点**:
- [ ] 正确识别目标日程
- [ ] 时间更新正确
- [ ] 数据库中已更新

---

#### TC-SCHEDULE-006: 删除日程

**测试目标**: 验证日程删除功能

**前置条件**: 已有待删除的日程

**操作步骤**:
1. 输入: `删除今天下午三点的项目评审`
2. 确认删除

**预期结果**:
- 删除指定的日程
- 返回删除确认

**验证点**:
- [ ] 指定日程被删除
- [ ] 数据库中已删除

---

#### TC-SCHEDULE-007: 查找空闲时间

**测试目标**: 验证空闲时间查找功能

**操作步骤**:
1. 输入: `查看明天有什么空闲时间`
2. 输入: `帮我找一下这周下午2点到5点之间的空档`

**预期结果**:
- 返回指定日期/时间段的空闲时间
- 显示可用时间段

**验证点**:
- [ ] 正确识别已有日程
- [ ] 返回正确的空闲时间段

---

#### TC-SCHEDULE-008: 相对时间解析

**测试目标**: 验证各种相对时间表达

**测试用例**:
| 输入 | 预期解析结果 |
|------|-------------|
| 明天上午9点 | 明天 09:00 |
| 后天下午2点 | 明天下午 14:00 |
| 下周一上午10点 | 下周一 10:00 |
| 这周五下班前 | 本周五 18:00 |
| 3天后 | 3天后 00:00 |
| 2周后的周三 | 2周后的周三 00:00 |

**验证点**:
- [ ] 所有相对时间正确解析
- [ ] 中文和英文都支持

---

### 第三部分：多 Agent 协作测试

#### TC-ORCH-001: 简单任务路由

**测试目标**: 验证单任务正确路由

**操作步骤**:
1. 输入: `搜索我记录的 Go 学习笔记`

**预期结果**:
- 识别为 memo 任务
- 路由到 MemoParrot
- 返回搜索结果

**验证点**:
- [ ] 正确识别任务类型
- [ ] 路由到正确的 Agent
- [ ] 返回正确的结果

---

#### TC-ORCH-002: 复杂任务分解

**测试目标**: 验证多任务分解和 DAG 调度

**操作步骤**:
1. 输入: `搜索上次项目会议的纪要，并帮我安排下周的跟进会议`

**预期结果**:
- 分解为 2 个任务：
  1. 搜索会议纪要 (memo)
  2. 创建日程 (schedule)
- DAG 依赖：先搜索，后创建日程
- 按依赖顺序执行

**验证点**:
- [ ] 任务正确分解为 2 个
- [ ] 依赖关系正确
- [ ] 结果聚合展示

---

#### TC-ORCH-003: 并行任务执行

**测试目标**: 验证无依赖任务并行执行

**操作步骤**:
1. 输入: `帮我搜索 Go 笔记，同时看看这周有什么日程`

**预期结果**:
- 分解为 2 个独立任务
- 并行执行
- 结果同时展示

**验证点**:
- [ ] 识别为可并行任务
- [ ] 两个任务同时执行
- [ ] 结果聚合展示

---

#### TC-ORCH-004: 自动转交 (Handoff)

**测试目标**: 验证任务失败时自动转交

**场景**:
1. 当前 Agent 无法处理
2. 自动转交给其他 Agent

**操作步骤**:
1. 输入: `帮我安排明天的会议`（在 Memo 对话中）

**预期结果**:
- 检测到需要 schedule 能力
- 自动转交给 ScheduleParrot
- 成功创建日程

**验证点**:
- [ ] 识别需要转交的场景
- [ ] 成功转交到目标 Agent
- [ ] 任务完成

---

### 第五部分：上下文工程测试

#### TC-CTX-001: 长期记忆检索

**测试目标**: 验证从 episodic memory 检索历史交互

**前置条件**: 已存在历史交互记录

**操作步骤**:
1. 预先插入历史交互记录
2. 触发新的查询
3. 验证历史相关记录被检索

**预期结果**: 返回与当前查询相关的历史交互

---

#### TC-CTX-002: 用户偏好提取

**测试目标**: 验证用户偏好被正确加载

**预期输出**: 返回用户时区设置、通信风格偏好

---

#### TC-CTX-003: 对话历史提取

**测试目标**: 验证最近 N 轮对话被正确加载

**操作步骤**: 多轮对话后，输入新查询

**预期输出**: 返回最近 10 轮对话（默认配置）

---

### 第六部分：可观测性测试

#### TC-OBS-001: 追踪链路完整性

**测试目标**: 验证完整调用链被追踪

**验证点**: Span 包含操作名称、开始/结束时间、元数据

---

#### TC-OBS-002: 请求指标记录

**测试目标**: 验证请求指标被正确记录

**验证点**: TotalRequests、AvgLatencyMs 正确

---

#### TC-OBS-003: 工具调用统计

**测试目标**: 验证工具调用次数被记录

**验证点**: CallCount、AvgLatencyMs 正确

---

### 第七部分：集成场景测试

#### TC-JOURNEY-001: 笔记搜索完整流程

**测试步骤**:
1. 用户输入: "查找我之前记录的 Go 学习笔记"
2. 上下文工程加载历史偏好
3. 检索相关笔记
4. 返回结果
5. 记录指标和日志

**验证点**: Tracer 包含完整链路、Metrics 记录请求

---

#### TC-JOURNEY-002: 复杂任务编排

**测试步骤**:
1. 用户输入: "帮我搜索上次项目会议的纪要，然后安排下周一的项目跟进会"
2. 任务分解为 2 个子任务
3. DAG 调度执行
4. 结果聚合
5. 返回综合响应

**验证点**: 正确分解为 2 个任务、依赖关系正确 (memo → schedule)

---

### 第四部分：交互体验测试

#### TC-INTERACT-001: 多轮澄清

**测试目标**: 验证必填信息缺失时的追问能力

**操作步骤**:
1. 输入: `帮我安排个会`
2. 观察澄清问题

**预期结果**:
- 不直接创建日程
- 返回澄清问题: "请问会议的主题是什么？计划在什么时间开始？"
- 保持会话状态

**验证点**:
- [ ] 不创建不完整的日程
- [ ] 返回有意义的澄清问题
- [ ] 会话上下文保持

---

#### TC-INTERACT-002: 流式响应

**测试目标**: 验证 Thinking 过程和最终结果的流式传输

**操作步骤**:
1. 输入一个复杂查询
2. 观察响应过程

**预期结果**:
- 收到流式输出
- 事件序列:
  1. `thinking_start`: "正在分析用户意图..."
  2. `tool_call`: "正在搜索笔记..."
  3. `thinking_end`
  4. `content`: 最终结果

**验证点**:
- [ ] 流式输出可见
- [ ] Thinking 过程可见
- [ ] 工具调用状态可见

---

#### TC-INTERACT-003: 错误处理

**测试目标**: 验证错误情况的友好提示

**操作步骤**:
1. 输入一个无效请求
2. 观察错误处理

**预期结果**:
- 返回友好的错误提示
- 不泄露内部错误详情
- 建议用户如何操作

**验证点**:
- [ ] 错误信息友好
- [ ] 不暴露内部细节
- [ ] 提供解决建议

---

## 测试数据准备

> **重要**: 测试数据是 E2E 测试的基础，必须预埋充足的数据才能覆盖所有测试场景。

### 数据概览

| 数据类型 | 数量 | 用途 |
|----------|------|------|
| 用户 | 1 | 测试用户 (ID=1) |
| 笔记 (Memo) | 50+ | 关键词搜索、语义搜索、混合检索、时间过滤、标签过滤 |
| 日程 (Schedule) | 30+ | 日程创建、查询、修改、删除、冲突检测、空闲时间查找 |
| 标签 (MemoTags) | 50+ | 标签过滤测试 |
| 长期记忆 (EpisodicMemory) | 20+ | 上下文工程测试 |
| 对话历史 (AIMessage) | 10+ | 短期记忆测试 |

### 快速开始：使用测试数据脚本

```bash
# 方式 1：使用 SQL 脚本（推荐）
make db-shell
\i docs/testing/fixtures/test_data.sql

# 方式 2：使用 Go Fixtures 运行测试
ENABLE_MANUAL_E2E=true go test -tags=e2e_manual ./ai/e2e/... -v
```

### 创建测试笔记（50+ 条）

```sql
-- 在 PostgreSQL 中创建测试笔记
INSERT INTO memo (uid, creator_id, content, visibility, row_status, created_ts, updated_ts)
VALUES
  -- Go 相关笔记
  ('test_memo_001', 1, 'Go 语言学习笔记：今天学习了 Go 的并发编程，包括 goroutine 和 channel 的使用。', 'PRIVATE', 'NORMAL', EXTRACT(EPOCH FROM NOW()), EXTRACT(EPOCH FROM NOW())),
  ('test_memo_002', 1, 'Go 进阶：深入理解 Go 的调度器、GMP 模型和垃圾回收机制。', 'PRIVATE', 'NORMAL', EXTRACT(EPOCH FROM NOW()), EXTRACT(EPOCH FROM NOW())),
  ('test_memo_003', 1, 'Go 项目架构：DDD 领域驱动设计在 Go 项目中的实践。', 'PRIVATE', 'NORMAL', EXTRACT(EPOCH FROM NOW()), EXTRACT(EPOCH FROM NOW())),

  -- Python 相关笔记
  ('test_memo_004', 1, 'Python 入门指南：变量、数据类型、条件语句和循环的基础用法。', 'PRIVATE', 'NORMAL', EXTRACT(EPOCH FROM NOW()), EXTRACT(EPOCH FROM NOW())),
  ('test_memo_005', 1, 'Python 进阶：装饰器、生成器、上下文管理器的使用。', 'PRIVATE', 'NORMAL', EXTRACT(EPOCH FROM NOW()), EXTRACT(EPOCH FROM NOW())),
  ('test_memo_006', 1, 'Python 数据分析：Pandas、NumPy、Matplotlib 使用笔记。', 'PRIVATE', 'NORMAL', EXTRACT(EPOCH FROM NOW()), EXTRACT(EPOCH FROM NOW())),

  -- 会议纪要
  ('test_memo_007', 1, '会议纪要：Q1 规划会议，讨论了产品路线图和技术债务。', 'PRIVATE', 'NORMAL', EXTRACT(EPOCH FROM NOW()), EXTRACT(EPOCH FROM NOW())),
  ('test_memo_008', 1, '项目会议：Sprint 3 评审会议纪要，总结本周完成的工作和下週计划。', 'PRIVATE', 'NORMAL', EXTRACT(EPOCH FROM NOW()), EXTRACT(EPOCH FROM NOW())),
  ('test_memo_009', 1, '团队例会：关于代码审查规范的讨论，最终确定了审查清单。', 'PRIVATE', 'NORMAL', EXTRACT(EPOCH FROM NOW()), EXTRACT(EPOCH FROM NOW())),

  -- 读书笔记
  ('test_memo_010', 1, '《代码整洁之道》读书笔记：代码可读性的重要性及实践方法。', 'PRIVATE', 'NORMAL', EXTRACT(EPOCH FROM NOW()), EXTRACT(EPOCH FROM NOW())),
  ('test_memo_011', 1, '《架构整洁之道》笔记：分层架构、依赖倒置原则的理解。', 'PRIVATE', 'NORMAL', EXTRACT(EPOCH FROM NOW()), EXTRACT(EPOCH FROM NOW())),
  ('test_memo_012', 1, '《人月神话》笔记：软件项目管理的心得体会。', 'PRIVATE', 'NORMAL', EXTRACT(EPOCH FROM NOW()), EXTRACT(EPOCH FROM NOW())),

  -- 学习笔记
  ('test_memo_013', 1, '学习笔记：HTTP 协议详解，包括请求方法、状态码、缓存机制。', 'PRIVATE', 'NORMAL', EXTRACT(EPOCH FROM NOW()), EXTRACT(EPOCH FROM NOW())),
  ('test_memo_014', 1, '学习笔记：Git 工作流最佳实践，Gitflow vs Trunk-based。', 'PRIVATE', 'NORMAL', EXTRACT(EPOCH FROM NOW()), EXTRACT(EPOCH FROM NOW())),
  ('test_memo_015', 1, '学习笔记：Docker 容器化技术，镜像构建和网络配置。', 'PRIVATE', 'NORMAL', EXTRACT(EPOCH FROM NOW()), EXTRACT(EPOCH FROM NOW())),

  -- 工作相关
  ('test_memo_016', 1, '工作日志：本周完成了用户认证模块的重构，优化了登录流程。', 'PRIVATE', 'NORMAL', EXTRACT(EPOCH FROM NOW()), EXTRACT(EPOCH FROM NOW())),
  ('test_memo_017', 1, 'TODO：处理线上工单 #1234，用户反馈登录失败问题。', 'PRIVATE', 'NORMAL', EXTRACT(EPOCH FROM NOW()), EXTRACT(EPOCH FROM NOW())),
  ('test_memo_018', 1, '项目文档：API 接口文档 v2.0，包含所有新增接口。', 'PRIVATE', 'NORMAL', EXTRACT(EPOCH FROM NOW()), EXTRACT(EPOCH FROM NOW())),

  -- 生活记录
  ('test_memo_019', 1, '周末计划：学习新技术 Terraform，实践基础设施即代码。', 'PRIVATE', 'NORMAL', EXTRACT(EPOCH FROM NOW()), EXTRACT(EPOCH FROM NOW())),
  ('test_memo_020', 1, '读书计划：准备阅读《深入理解计算机系统》。', 'PRIVATE', 'NORMAL', EXTRACT(EPOCH FROM NOW()), EXTRACT(EPOCH FROM NOW())),

  -- 更多笔记（用于边界测试）
  ('test_memo_021', 1, '短内容：A', 'PRIVATE', 'NORMAL', EXTRACT(EPOCH FROM NOW()), EXTRACT(EPOCH FROM NOW())),
  ('test_memo_022', 1, '中等长度内容：这是一个包含多个关键词的测试笔记，用于验证搜索功能的各种场景。关键词包括：测试、搜索、功能、验证、场景等。', 'PRIVATE', 'NORMAL', EXTRACT(EPOCH FROM NOW()), EXTRACT(EPOCH FROM NOW()));
```

> **提示**: 完整测试数据脚本请参考 `docs/testing/fixtures/test_data.sql`

### 创建测试日程（30+ 条）

```sql
-- 创建测试日程（分散在不同日期）
INSERT INTO schedule (uid, creator_id, title, description, location, start_ts, end_ts, all_day, timezone, row_status, created_ts, updated_ts)
VALUES
  -- 今日日程
  ('test_sched_001', 1, '团队周会', '每周例会，讨论本周进展', '会议室A', EXTRACT(EPOCH FROM NOW()) + 3600, EXTRACT(EPOCH FROM NOW()) + 7200, false, 'Asia/Shanghai', 'NORMAL', EXTRACT(EPOCH FROM NOW()), EXTRACT(EPOCH FROM NOW())),
  ('test_sched_002', 1, '代码审查', 'Review PR #123', '线上', EXTRACT(EPOCH FROM NOW()) + 10800, EXTRACT(EPOCH FROM NOW()) + 14400, false, 'Asia/Shanghai', 'NORMAL', EXTRACT(EPOCH FROM NOW()), EXTRACT(EPOCH FROM NOW())),

  -- 明日日程
  ('test_sched_003', 1, '项目评审', 'Q2 项目规划评审', '会议室B', EXTRACT(EPOCH FROM NOW()) + 86400 + 3600, EXTRACT(EPOCH FROM NOW()) + 86400 + 7200, false, 'Asia/Shanghai', 'NORMAL', EXTRACT(EPOCH FROM NOW()), EXTRACT(EPOCH FROM NOW())),

  -- 本周其他日程
  ('test_sched_004', 1, '技术分享', 'Go 并发编程实践', '会议室C', EXTRACT(EPOCH FROM NOW()) + 172800 + 3600, EXTRACT(EPOCH FROM NOW()) + 172800 + 7200, false, 'Asia/Shanghai', 'NORMAL', EXTRACT(EPOCH FROM NOW()), EXTRACT(EPOCH FROM NOW())),
  ('test_sched_005', 1, '一对一沟通', '与产品经理同步需求', '办公室', EXTRACT(EPOCH FROM NOW()) + 259200 + 3600, EXTRACT(EPOCH FROM NOW()) + 259200 + 5400, false, 'Asia/Shanghai', 'NORMAL', EXTRACT(EPOCH FROM NOW()), EXTRACT(EPOCH FROM NOW())),

  -- 下周日程（用于测试"下周"查询）
  ('test_sched_006', 1, 'Sprint 计划会', 'Sprint 4 计划会议', '会议室A', EXTRACT(EPOCH FROM NOW()) + 604800 + 3600, EXTRACT(EPOCH FROM NOW()) + 604800 + 7200, false, 'Asia/Shanghai', 'NORMAL', EXTRACT(EPOCH FROM NOW()), EXTRACT(EPOCH FROM NOW())),
  ('test_sched_007', 1, '客户演示', '产品演示会议', '线上', EXTRACT(EPOCH FROM NOW()) + 691200 + 3600, EXTRACT(EPOCH FROM NOW()) + 691200 + 5400, false, 'Asia/Shanghai', 'NORMAL', EXTRACT(EPOCH FROM NOW()), EXTRACT(EPOCH FROM NOW())),

  -- 全天日程
  ('test_sched_008', 1, '专注工作日', '封闭开发日', '', EXTRACT(EPOCH FROM NOW()) + 86400, NULL, true, 'Asia/Shanghai', 'NORMAL', EXTRACT(EPOCH FROM NOW()), EXTRACT(EPOCH FROM NOW()));
```

### 创建测试标签

```sql
-- 为笔记创建标签
INSERT INTO memo_tags (memo_id, tag, confidence, source, created_ts)
SELECT id, 'go', 1.0, 'user', EXTRACT(EPOCH FROM NOW())
FROM memo WHERE content LIKE '%Go%' OR content LIKE '%go%';

INSERT INTO memo_tags (memo_id, tag, confidence, source, created_ts)
SELECT id, 'python', 1.0, 'user', EXTRACT(EPOCH FROM NOW())
FROM memo WHERE content LIKE '%Python%' OR content LIKE '%python%';

INSERT INTO memo_tags (memo_id, tag, confidence, source, created_ts)
SELECT id, '编程', 1.0, 'user', EXTRACT(EPOCH FROM NOW())
FROM memo WHERE content LIKE '%编程%' OR content LIKE '%代码%';

INSERT INTO memo_tags (memo_id, tag, confidence, source, created_ts)
SELECT id, '会议', 1.0, 'user', EXTRACT(EPOCH FROM NOW())
FROM memo WHERE content LIKE '%会议%';

INSERT INTO memo_tags (memo_id, tag, confidence, source, created_ts)
SELECT id, '学习', 1.0, 'user', EXTRACT(EPOCH FROM NOW())
FROM memo WHERE content LIKE '%学习%' OR content LIKE '%笔记%';

INSERT INTO memo_tags (memo_id, tag, confidence, source, created_ts)
SELECT id, '工作', 1.0, 'user', EXTRACT(EPOCH FROM NOW())
FROM memo WHERE content LIKE '%工作%' OR content LIKE '%项目%';
```

### 创建长期记忆（可选，用于上下文工程测试）

```sql
-- 创建历史交互记录
INSERT INTO episodic_memory (user_id, agent_type, user_input, outcome, summary, importance, created_ts)
VALUES
  (1, 'memo', '搜索 Go 学习笔记', 'success', '用户搜索了 Go 语言相关的学习笔记', 0.8, EXTRACT(EPOCH FROM NOW()) - 86400),
  (1, 'schedule', '安排团队周会', 'success', '用户创建了每周一次的团队周会', 0.9, EXTRACT(EPOCH FROM NOW()) - 172800),
  (1, 'memo', '查找会议纪要', 'success', '用户查找了 Q1 规划会议的纪要', 0.7, EXTRACT(EPOCH FROM NOW()) - 259200);
```

---

## 测试检查清单

### 每日冒烟测试

- [ ] TC-MEMO-001: 关键词搜索
- [ ] TC-MEMO-002: 语义向量搜索
- [ ] TC-SCHEDULE-001: 创建日程
- [ ] TC-SCHEDULE-003: 查询今日日程
- [ ] TC-ORCH-001: 简单任务路由

### 完整回归测试

#### 核心功能测试
- [ ] TC-MEMO-001: 关键词搜索
- [ ] TC-MEMO-002: 语义向量搜索
- [ ] TC-MEMO-003: 时间过滤搜索
- [ ] TC-MEMO-004: 标签过滤搜索
- [ ] TC-MEMO-005: 无结果处理
- [ ] TC-SCHEDULE-001: 创建日程（简单）
- [ ] TC-SCHEDULE-002: 创建日程（含冲突检测）
- [ ] TC-SCHEDULE-003: 查询今日日程
- [ ] TC-SCHEDULE-004: 查询本周日程
- [ ] TC-SCHEDULE-005: 修改日程
- [ ] TC-SCHEDULE-006: 删除日程
- [ ] TC-SCHEDULE-007: 查找空闲时间
- [ ] TC-SCHEDULE-008: 相对时间解析

#### 编排与协作测试
- [ ] TC-ORCH-001: 简单任务路由
- [ ] TC-ORCH-002: 复杂任务分解
- [ ] TC-ORCH-003: 并行任务执行
- [ ] TC-ORCH-004: 自动转交 (Handoff)

#### 交互体验测试
- [ ] TC-INTERACT-001: 多轮澄清
- [ ] TC-INTERACT-002: 流式响应
- [ ] TC-INTERACT-003: 错误处理

#### 上下文工程测试
- [ ] TC-CTX-001: 长期记忆检索
- [ ] TC-CTX-002: 用户偏好提取
- [ ] TC-CTX-003: 对话历史提取

#### 可观测性测试
- [ ] TC-OBS-001: 追踪链路完整性
- [ ] TC-OBS-002: 请求指标记录
- [ ] TC-OBS-003: 工具调用统计

#### 集成场景测试
- [ ] TC-JOURNEY-001: 笔记搜索完整流程
- [ ] TC-JOURNEY-002: 复杂任务编排

---

## 常见问题排查

### 问题 1: 日程创建失败

**可能原因**:
- 时间解析失败
- 数据库连接问题

**排查步骤**:
1. 检查输入的时间表达是否支持
2. 检查数据库连接
3. 查看服务端日志

### 问题 2: 搜索无结果

**可能原因**:
- 索引未构建
- 关键词不匹配

**排查步骤**:
1. 确认笔记已创建
2. 检查 pgvector 扩展是否安装
3. 尝试语义搜索

### 问题 3: Agent 转交失败

**可能原因**:
- 目标 Agent 不可用
- Handoff 深度超限

**排查步骤**:
1. 检查专家配置
2. 查看 Handoff 日志

---

## 测试报告模板

```markdown
## 测试报告

**测试日期**: YYYY-MM-DD
**测试人员**: XXX
**测试环境**: dev/staging

### 测试结果

| 用例 | 状态 | 备注 |
|------|------|------|
| TC-MEMO-001 | PASS/FAIL | 备注 |
| ... | ... | ... |

### 问题记录

| 问题 | 严重程度 | 状态 |
|------|---------|------|
| 问题描述 | P0/P1/P2/P3 | Open/Resolved |
```

---

## 参考资料

- 测试用例文档: `docs/testing/e2e-ai-agent-test-cases.md`
- 专家配置:
  - `config/parrots/memo.yaml`
  - `config/parrots/schedule.yaml`
- Agent 实现:
  - `ai/agents/tools/memo_search.go`
  - `ai/agents/tools/schedule/tools.go`
