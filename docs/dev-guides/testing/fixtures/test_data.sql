-- E2E 测试数据脚本
-- 用途: 为 DivineSense E2E 测试预埋测试数据
-- 执行方式: make db-shell 后执行 \i docs/testing/fixtures/test_data.sql

-- 设置当前时间为测试基准时间 (2026-02-15 10:00:00 UTC)
-- 所有时间戳基于此基准计算

-- ============================================================================
-- 1. 测试用户
-- ============================================================================
INSERT INTO "user" (id, username, nickname, email, role, created_ts, updated_ts, row_status, timezone)
VALUES (1, 'test_user', 'Test User', 'test@example.com', 'USER', 1739613600, 1739613600, 'NORMAL', 'Asia/Shanghai')
ON CONFLICT (id) DO UPDATE SET
    username = EXCLUDED.username,
    nickname = EXCLUDED.nickname,
    email = EXCLUDED.email;

-- ============================================================================
-- 2. 测试笔记 (50+ 条)
-- ============================================================================
INSERT INTO memo (uid, creator_id, content, visibility, row_status, created_ts, updated_ts) VALUES
-- Go 相关笔记 (10 条)
('e2e_memo_001', 1, 'Go 语言学习笔记：今天学习了 Go 的并发编程，包括 goroutine 和 channel 的使用。goroutine 是轻量级线程，由 Go 运行时管理。channel 用于 goroutine 之间的通信和同步。', 'PRIVATE', 'NORMAL', 1739613600, 1739613600),
('e2e_memo_002', 1, 'Go 进阶：深入理解 Go 的调度器、GMP 模型和垃圾回收机制。G 代表 goroutine，M 代表线程，P 代表处理器，三者共同构成了 Go 的并发模型。', 'PRIVATE', 'NORMAL', 1739527200, 1739527200),
('e2e_memo_003', 1, 'Go 项目架构：DDD 领域驱动设计在 Go 项目中的实践。包括实体、值对象、聚合根、领域服务等概念的应用。', 'PRIVATE', 'NORMAL', 1739440800, 1739440800),
('e2e_memo_004', 1, 'Go 标准库学习：context 包的使用。context 用于在 API 边界之间传递截止时间、取消信号和其他请求范围的值。', 'PRIVATE', 'NORMAL', 1739354400, 1739354400),
('e2e_memo_005', 1, 'Go 测试：单元测试、集成测试和基准测试的编写。使用 testing 包和 testify 框架编写高质量测试。', 'PRIVATE', 'NORMAL', 1739268000, 1739268000),
('e2e_memo_006', 1, 'Go 性能优化：pprof 性能分析、内存分配优化、并发模式优化。', 'PRIVATE', 'NORMAL', 1739181600, 1739181600),
('e2e_memo_007', 1, 'Go Web 开发：使用 Gin 框架构建 RESTful API。路由、中间件、参数绑定、响应封装。', 'PRIVATE', 'NORMAL', 1739095200, 1739095200),
('e2e_memo_008', 1, 'Go gRPC：Protocol Buffers 和 gRPC 的使用。定义 .proto 文件，生成 Go 代码，双向流通信。', 'PRIVATE', 'NORMAL', 1739008800, 1739008800),
('e2e_memo_009', 1, 'Go 数据库编程：使用 GORM 和 sqlx 操作数据库。连接池、事务、预编译语句。', 'PRIVATE', 'NORMAL', 1738922400, 1738922400),
('e2e_memo_010', 1, 'Go 错误处理：使用 errors 包和 wrap 错误。自定义错误类型、错误检查模式。', 'PRIVATE', 'NORMAL', 1738836000, 1738836000),

-- Python 相关笔记 (10 条)
('e2e_memo_011', 1, 'Python 入门指南：变量、数据类型、条件语句和循环的基础用法。Python 是动态类型语言，语法简洁优雅。', 'PRIVATE', 'NORMAL', 1739613600, 1739613600),
('e2e_memo_012', 1, 'Python 进阶：装饰器、生成器、上下文管理器的使用。装饰器是高阶函数，用于扩展函数功能。', 'PRIVATE', 'NORMAL', 1739527200, 1739527200),
('e2e_memo_013', 1, 'Python 数据分析：Pandas、NumPy、Matplotlib 使用笔记。数据处理、可视化、统计分析。', 'PRIVATE', 'NORMAL', 1739440800, 1739440800),
('e2e_memo_014', 1, 'Python Web 开发：Django 和 Flask 框架对比。Django 是全栈框架，Flask 是微框架。', 'PRIVATE', 'NORMAL', 1739354400, 1739354400),
('e2e_memo_015', 1, 'Python 异步编程：asyncio 模块的使用。async/await 语法、事件循环、协程。', 'PRIVATE', 'NORMAL', 1739268000, 1739268000),
('e2e_memo_016', 1, 'Python 类型提示：typing 模块的使用。泛型、Protocol、TypeVar。', 'PRIVATE', 'NORMAL', 1739181600, 1739181600),
('e2e_memo_017', 1, 'Python 单元测试：pytest 框架的使用。fixture、parametrize、mock。', 'PRIVATE', 'NORMAL', 1739095200, 1739095200),
('e2e_memo_018', 1, 'Python 机器学习：scikit-learn 入门。分类、回归、聚类算法。', 'PRIVATE', 'NORMAL', 1739008800, 1739008800),
('e2e_memo_019', 1, 'Python 爬虫：Scrapy 框架使用。爬取网页、解析内容、存储数据。', 'PRIVATE', 'NORMAL', 1738922400, 1738922400),
('e2e_memo_020', 1, 'Python 面试题整理：常见编程题和算法题解法。', 'PRIVATE', 'NORMAL', 1738836000, 1738836000),

-- 会议纪要 (8 条)
('e2e_memo_021', 1, '会议纪要：Q1 规划会议，讨论了产品路线图和技术债务。会议确定了 Q1 的三个主要目标：性能优化、新功能开发、代码重构。', 'PRIVATE', 'NORMAL', 1739613600, 1739613600),
('e2e_memo_022', 1, '项目会议：Sprint 3 评审会议纪要，总结本周完成的工作和下週计划。本周完成了用户认证模块、搜索功能优化。', 'PRIVATE', 'NORMAL', 1739527200, 1739527200),
('e2e_memo_023', 1, '团队例会：关于代码审查规范的讨论，最终确定了审查清单。包括命名规范、注释要求、测试覆盖率要求。', 'PRIVATE', 'NORMAL', 1739440800, 1739440800),
('e2e_memo_024', 1, '技术评审会：微服务架构方案评审。讨论了服务拆分、API 设计、熔断降级策略。', 'PRIVATE', 'NORMAL', 1739354400, 1739354400),
('e2e_memo_025', 1, '产品需求会：讨论了下个版本的功能需求。用户反馈集中在性能、用户体验两个方面。', 'PRIVATE', 'NORMAL', 1739268000, 1739268000),
('e2e_memo_026', 1, '设计评审会：数据库 schema 优化方案。讨论了索引设计、分表策略、缓存方案。', 'PRIVATE', 'NORMAL', 1739181600, 1739181600),
('e2e_memo_027', 1, '运维会议：监控告警和故障复盘。总结了最近一次线上故障的原因和改进措施。', 'PRIVATE', 'NORMAL', 1739095200, 1739095200),
('e2e_memo_028', 1, '项目启动会：新项目 Kickoff 会议。确定了项目目标、人员分工、时间计划。', 'PRIVATE', 'NORMAL', 1739008800, 1739008800),

-- 读书笔记 (6 条)
('e2e_memo_029', 1, '《代码整洁之道》读书笔记：代码可读性的重要性及实践方法。命名规范、函数设计、注释技巧。', 'PRIVATE', 'NORMAL', 1739613600, 1739613600),
('e2e_memo_030', 1, '《架构整洁之道》笔记：分层架构、依赖倒置原则的理解。整洁架构的四个层次：实体、用例、接口适配器、框架驱动。', 'PRIVATE', 'NORMAL', 1739527200, 1739527200),
('e2e_memo_031', 1, '《人月神话》笔记：软件项目管理的心得体会。没有银弹、 Brooks 法则、原型设计的重要性。', 'PRIVATE', 'NORMAL', 1739440800, 1739440800),
('e2e_memo_032', 1, '《设计模式》笔记：GOF 23 种设计模式。创建型、结构型、行为型模式的应用场景。', 'PRIVATE', 'NORMAL', 1739354400, 1739354400),
('e2e_memo_033', 1, '《深入理解计算机系统》笔记：程序的机器级表示、处理器架构、优化程序性能。', 'PRIVATE', 'NORMAL', 1739268000, 1739268000),
('e2e_memo_034', 1, '《算法导论》笔记：排序、搜索、图算法。复杂度分析、算法设计技巧。', 'PRIVATE', 'NORMAL', 1739181600, 1739181600),

-- 学习笔记 (10 条)
('e2e_memo_035', 1, '学习笔记：HTTP 协议详解，包括请求方法、状态码、缓存机制。GET、POST、PUT、DELETE 方法的使用场景。', 'PRIVATE', 'NORMAL', 1739613600, 1739613600),
('e2e_memo_036', 1, '学习笔记：Git 工作流最佳实践，Gitflow vs Trunk-based。分支策略、代码审查、合并冲突处理。', 'PRIVATE', 'NORMAL', 1739527200, 1739527200),
('e2e_memo_037', 1, '学习笔记：Docker 容器化技术，镜像构建和网络配置。Dockerfile 最佳实践、Compose 编排。', 'PRIVATE', 'NORMAL', 1739440800, 1739440800),
('e2e_memo_038', 1, '学习笔记：Kubernetes 入门。Pod、Deployment、Service、ConfigMap 的使用。', 'PRIVATE', 'NORMAL', 1739354400, 1739354400),
('e2e_memo_039', 1, '学习笔记：Redis 缓存策略。缓存穿透、缓存击穿、缓存雪崩的解决方案。', 'PRIVATE', 'NORMAL', 1739268000, 1739268000),
('e2e_memo_040', 1, '学习笔记：消息队列 Kafka。分区、消费者组、Exactly-Once 语义。', 'PRIVATE', 'NORMAL', 1739181600, 1739181600),
('e2e_memo_041', 1, '学习笔记：微服务架构。服务注册与发现、负载均衡、熔断降级、链路追踪。', 'PRIVATE', 'NORMAL', 1739095200, 1739095200),
('e2e_memo_042', 1, '学习笔记：OAuth 2.0 授权流程。授权码模式、隐式模式、客户端凭证模式。', 'PRIVATE', 'NORMAL', 1739008800, 1739008800),
('e2e_memo_043', 1, '学习笔记：PostgreSQL 高级特性。JSONB、数组类型、窗口函数、CTE。', 'PRIVATE', 'NORMAL', 1738922400, 1738922400),
('e2e_memo_044', 1, '学习笔记：Elasticsearch 全文搜索。倒排索引、分词器、聚合查询。', 'PRIVATE', 'NORMAL', 1738836000, 1738836000),

-- 工作相关 (8 条)
('e2e_memo_045', 1, '工作日志：本周完成了用户认证模块的重构，优化了登录流程。引入 JWT 令牌，安全性提升。', 'PRIVATE', 'NORMAL', 1739613600, 1739613600),
('e2e_memo_046', 1, 'TODO：处理线上工单 #1234，用户反馈登录失败问题。定位到是 Token 过期时间配置错误。', 'PRIVATE', 'NORMAL', 1739527200, 1739527200),
('e2e_memo_047', 1, '项目文档：API 接口文档 v2.0，包含所有新增接口。', 'PRIVATE', 'NORMAL', 1739440800, 1739440800),
('e2e_memo_048', 1, '技术方案：搜索功能优化方案。使用 Elasticsearch 替代数据库模糊查询，响应时间从 500ms 降至 50ms。', 'PRIVATE', 'NORMAL', 1739354400, 1739354400),
('e2e_memo_049', 1, '性能分析报告：API 性能优化。热点接口分析、数据库慢查询优化、缓存命中率提升。', 'PRIVATE', 'NORMAL', 1739268000, 1739268000),
('e2e_memo_050', 1, '架构设计：实时消息推送系统设计。使用 WebSocket 长连接，配合 Redis 发布订阅。', 'PRIVATE', 'NORMAL', 1739181600, 1739181600),
('e2e_memo_051', 1, '代码审查记录：PR #456 代码审查。提出了 5 个改进建议，已全部采纳。', 'PRIVATE', 'NORMAL', 1739095200, 1739095200),
('e2e_memo_052', 1, '故障复盘：上周服务宕机原因分析。数据库连接池耗尽，已调整配置参数。', 'PRIVATE', 'NORMAL', 1739008800, 1739008800),

-- 边界测试用例 (2 条)
('e2e_memo_053', 1, '短内容：A', 'PRIVATE', 'NORMAL', 1739613600, 1739613600),
('e2e_memo_054', 1, '中等长度内容：这是一个包含多个关键词的测试笔记，用于验证搜索功能的各种场景。关键词包括：测试、搜索、功能、验证、场景等。', 'PRIVATE', 'NORMAL', 1739527200, 1739527200);

-- ============================================================================
-- 3. 测试日程 (30+ 条)
-- ============================================================================
-- 今日: 2026-02-15
-- 明日: 2026-02-16
-- 本周: 2026-02-15 ~ 2026-02-21
-- 下周: 2026-02-22 ~ 2026-02-28

INSERT INTO schedule (uid, creator_id, title, description, location, start_ts, end_ts, all_day, timezone, row_status, created_ts, updated_ts) VALUES

-- 今日日程 (3 条) - 2026-02-15
('e2e_sched_001', 1, '团队周会', '每周例会，讨论本周进展', '会议室A', 1739620800, 1739624400, false, 'Asia/Shanghai', 'NORMAL', 1739613600, 1739613600),
('e2e_sched_002', 1, '代码审查', 'Review PR #123', '线上', 1739628000, 1739631600, false, 'Asia/Shanghai', 'NORMAL', 1739613600, 1739613600),
('e2e_sched_003', 1, '专注工作时间', '封闭开发', '', 1739613600, 1739653200, true, 'Asia/Shanghai', 'NORMAL', 1739613600, 1739613600),

-- 明日日程 (4 条) - 2026-02-16
('e2e_sched_004', 1, '项目评审', 'Q2 项目规划评审', '会议室B', 1739707200, 1739710800, false, 'Asia/Shanghai', 'NORMAL', 1739613600, 1739613600),
('e2e_sched_005', 1, '客户演示', '产品演示会议', '线上', 1739714400, 1739718000, false, 'Asia/Shanghai', 'NORMAL', 1739613600, 1739613600),
('e2e_sched_006', 1, '技术调研', '调研新技术选型', '', 1739721600, 1739725200, false, 'Asia/Shanghai', 'NORMAL', 1739613600, 1739613600),
('e2e_sched_007', 1, '每日站会', 'Sprint 每日同步', '线上', 1739620800, 1739622600, false, 'Asia/Shanghai', 'NORMAL', 1739613600, 1739613600),

-- 本周其他日程 (6 条) - 2026-02-17 ~ 2026-02-21
('e2e_sched_008', 1, '技术分享', 'Go 并发编程实践', '会议室C', 1739793600, 1739797200, false, 'Asia/Shanghai', 'NORMAL', 1739613600, 1739613600),
('e2e_sched_009', 1, '一对一沟通', '与产品经理同步需求', '办公室', 1739880000, 1739883600, false, 'Asia/Shanghai', 'NORMAL', 1739613600, 1739613600),
('e2e_sched_010', 1, 'Sprint 规划会', 'Sprint 4 计划会议', '会议室A', 1739966400, 1739973600, false, 'Asia/Shanghai', 'NORMAL', 1739613600, 1739613600),
('e2e_sched_011', 1, 'Bug 评审', '线上问题评审', '线上', 1739790000, 1739793600, false, 'Asia/Shanghai', 'NORMAL', 1739613600, 1739613600),
('e2e_sched_012', 1, 'UI 设计评审', '新功能界面评审', '会议室B', 1739876400, 1739880000, false, 'Asia/Shanghai', 'NORMAL', 1739613600, 1739613600),
('e2e_sched_013', 1, '数据库维护', '数据库备份和优化', '', 1740052800, 1740056400, false, 'Asia/Shanghai', 'NORMAL', 1739613600, 1739613600),

-- 下周日程 (8 条) - 2026-02-22 ~ 2026-02-28
('e2e_sched_014', 1, 'Sprint 启动会', 'Sprint 4 启动', '会议室A', 1740571200, 1740574800, false, 'Asia/Shanghai', 'NORMAL', 1739613600, 1739613600),
('e2e_sched_015', 1, '性能测试', '新版本性能测试', '', 1740657600, 1740664800, false, 'Asia/Shanghai', 'NORMAL', 1739613600, 1739613600),
('e2e_sched_016', 1, '发布评审', '版本发布评审会议', '会议室B', 1740744000, 1740747600, false, 'Asia/Shanghai', 'NORMAL', 1739613600, 1739613600),
('e2e_sched_017', 1, '外部培训', '新技术培训', '线上', 1740830400, 1740837600, false, 'Asia/Shanghai', 'NORMAL', 1739613600, 1739613600),
('e2e_sched_018', 1, '架构设计会', '新模块架构设计', '会议室C', 1740916800, 1740920400, false, 'Asia/Shanghai', 'NORMAL', 1739613600, 1739613600),
('e2e_sched_019', 1, '安全审计', '代码安全审计', '', 1741003200, 1741006800, false, 'Asia/Shanghai', 'NORMAL', 1739613600, 1739613600),
('e2e_sched_020', 1, '技术交流', '与外部团队技术交流', '线上', 1741089600, 1741093200, false, 'Asia/Shanghai', 'NORMAL', 1739613600, 1739613600),
('e2e_sched_021', 1, 'Sprint 回顾会', 'Sprint 3 回顾', '会议室A', 1741176000, 1741179600, false, 'Asia/Shanghai', 'NORMAL', 1739613600, 1739613600),

-- 跨天日程 (2 条) - 用于测试冲突检测
('e2e_sched_022', 1, '长会议', '跨天战略会议', '会议室A', 1739620800, 1739707200, false, 'Asia/Shanghai', 'NORMAL', 1739613600, 1739613600),
('e2e_sched_023', 1, '全天研讨会', '技术研讨会', '会议中心', 1740220800, 1740307200, false, 'Asia/Shanghai', 'NORMAL', 1739613600, 1739613600),

-- 定期日程 (2 条)
('e2e_sched_024', 1, '每周例会', '周例会', '会议室A', 1740571200, 1740574800, false, 'Asia/Shanghai', 'NORMAL', 1739613600, 1739613600),

-- 更多分散日程 (5 条)
('e2e_sched_025', 1, '临时会议', '紧急问题讨论', '线上', 1739703600, 1739705400, false, 'Asia/Shanghai', 'NORMAL', 1739613600, 1739613600),
('e2e_sched_026', 1, '面试', '技术面试', '会议室B', 1739962800, 1739970000, false, 'Asia/Shanghai', 'NORMAL', 1739613600, 1739613600),
('e2e_sched_027', 1, '文档评审', '技术文档评审', '', 1740049200, 1740052800, false, 'Asia/Shanghai', 'NORMAL', 1739613600, 1739613600),
('e2e_sched_028', 1, '客户支持', '客户问题支持', '线上', 1740567600, 1740571200, false, 'Asia/Shanghai', 'NORMAL', 1739613600, 1739613600),
('e2e_sched_029', 1, '内部培训', '新员工培训', '会议室C', 1740650400, 1740657600, false, 'Asia/Shanghai', 'NORMAL', 1739613600, 1739613600);

-- ============================================================================
-- 4. 测试标签 (50+ 条)
-- ============================================================================
INSERT INTO memo_tags (memo_id, tag, confidence, source, created_ts)
SELECT id, 'go', 1.0, 'user', 1739613600
FROM memo WHERE content LIKE '%Go%' OR content LIKE '%go%';

INSERT INTO memo_tags (memo_id, tag, confidence, source, created_ts)
SELECT id, 'python', 1.0, 'user', 1739613600
FROM memo WHERE content LIKE '%Python%' OR content LIKE '%python%';

INSERT INTO memo_tags (memo_id, tag, confidence, source, created_ts)
SELECT id, '编程', 1.0, 'user', 1739613600
FROM memo WHERE content LIKE '%编程%' OR content LIKE '%代码%' OR content LIKE '%算法%';

INSERT INTO memo_tags (memo_id, tag, confidence, source, created_ts)
SELECT id, '会议', 1.0, 'user', 1739613600
FROM memo WHERE content LIKE '%会议%' OR content LIKE '%评审%';

INSERT INTO memo_tags (memo_id, tag, confidence, source, created_ts)
SELECT id, '学习', 1.0, 'user', 1739613600
FROM memo WHERE content LIKE '%学习%' OR content LIKE '%笔记%' OR content LIKE '%读书%';

INSERT INTO memo_tags (memo_id, tag, confidence, source, created_ts)
SELECT id, '工作', 1.0, 'user', 1739613600
FROM memo WHERE content LIKE '%工作%' OR content LIKE '%项目%' OR content LIKE '%任务%';

INSERT INTO memo_tags (memo_id, tag, confidence, source, created_ts)
SELECT id, '技术', 1.0, 'user', 1739613600
FROM memo WHERE content LIKE '%技术%' OR content LIKE '%架构%' OR content LIKE '%设计%';

-- ============================================================================
-- 5. 长期记忆 (20+ 条) - 用于上下文工程测试
-- ============================================================================
INSERT INTO episodic_memory (user_id, agent_type, user_input, outcome, summary, importance, created_ts) VALUES
(1, 'memo', '搜索 Go 学习笔记', 'success', '用户搜索了 Go 语言相关的学习笔记，返回了 10 条相关结果', 0.8, 1739527200),
(1, 'memo', '查找 Q1 规划会议纪要', 'success', '用户查找了 Q1 规划会议的纪要，找到了相关文档', 0.9, 1739440800),
(1, 'schedule', '安排团队周会', 'success', '用户创建了每周一次的团队周会，时间为每周五上午 10 点', 0.9, 1739354400),
(1, 'memo', '搜索 Python 笔记', 'success', '用户搜索了 Python 相关笔记，返回了 10 条结果', 0.7, 1739268000),
(1, 'schedule', '查询本周日程', 'success', '用户查询了本周的日程安排，显示了 8 个日程', 0.8, 1739181600),
(1, 'memo', '搜索 Docker 笔记', 'success', '用户搜索了 Docker 容器化相关的学习笔记', 0.7, 1739095200),
(1, 'schedule', '安排项目评审', 'success', '用户创建了 Q2 项目规划评审会议', 0.8, 1739008800),
(1, 'memo', '搜索技术分享笔记', 'success', '用户搜索了技术分享相关的笔记', 0.6, 1738922400),
(1, 'schedule', '查询明天日程', 'success', '用户查询了明天的日程安排', 0.7, 1738836000),
(1, 'memo', '查找代码审查记录', 'success', '用户找到了代码审查相关的记录', 0.8, 1738749600),
(1, 'schedule', '删除临时会议', 'success', '用户删除了一个临时会议日程', 0.6, 1738663200),
(1, 'memo', '搜索微服务架构', 'success', '用户搜索了微服务架构相关的学习笔记', 0.7, 1738576800),
(1, 'schedule', '修改会议时间', 'success', '用户修改了团队周会的时间', 0.8, 1738490400),
(1, 'memo', '搜索 Redis 缓存', 'success', '用户搜索了 Redis 缓存策略相关的笔记', 0.7, 1738404000),
(1, 'schedule', '查找空闲时间', 'success', '用户查找了明天上午的空闲时间', 0.6, 1738317600),
(1, 'memo', '搜索 API 设计', 'success', '用户搜索了 API 设计相关的文档', 0.8, 1738231200),
(1, 'schedule', '创建每日站会', 'success', '用户创建了每日的站会日程', 0.7, 1738144800),
(1, 'memo', '查找性能优化笔记', 'success', '用户搜索了性能优化相关的笔记', 0.8, 1738058400),
(1, 'schedule', '查询下周安排', 'success', '用户查询了下周的日程安排', 0.7, 1737972000),
(1, 'memo', '搜索数据库设计', 'success', '用户搜索了数据库设计相关的学习笔记', 0.8, 1737885600);

-- ============================================================================
-- 6. 对话历史 (10+ 条) - 用于短期记忆测试
-- ============================================================================
INSERT INTO ai_conversation (user_id, title, model, created_ts, updated_ts)
VALUES (1, 'Go 学习讨论', 'glm-4', 1739613600, 1739613600);

-- 获取刚创建的对话 ID
-- 注意: 这里使用子查询获取最新创建的对话
INSERT INTO ai_message (conversation_id, role, content, model, token_count, created_ts)
SELECT (SELECT id FROM ai_conversation ORDER BY id DESC LIMIT 1), 'user', '我最近在学习 Go 语言，能帮我找一下之前的 Go 学习笔记吗？', 'glm-4', 50, 1739613600;

INSERT INTO ai_message (conversation_id, role, content, model, token_count, created_ts)
SELECT (SELECT id FROM ai_conversation ORDER BY id DESC LIMIT 1), 'assistant', '好的，我帮您找到了以下 Go 学习笔记：\n\n1. Go 语言学习笔记：goroutine 和 channel\n2. Go 进阶：GMP 模型\n3. Go 项目架构：DDD 实践\n...', 'glm-4', 200, 1739613600;

-- 继续添加更多对话历史（用于测试上下文窗口）
INSERT INTO ai_message (conversation_id, role, content, model, token_count, created_ts)
SELECT (SELECT id FROM ai_conversation ORDER BY id DESC LIMIT 1), 'user', '这些笔记很好，能帮我总结一下 Go 并发编程的重点吗？', 'glm-4', 40, 1739527200;

INSERT INTO ai_message (conversation_id, role, content, model, token_count, created_ts)
SELECT (SELECT id FROM ai_conversation ORDER BY id DESC LIMIT 1), 'assistant', 'Go 并发编程的重点包括：\n\n1. Goroutine：轻量级线程\n2. Channel：通信机制\n3. Select：多路复用\n4. Context：取消和超时\n5. Sync 包：同步原语', 'glm-4', 180, 1739527200;

INSERT INTO ai_message (conversation_id, role, content, model, token_count, created_ts)
SELECT (SELECT id FROM ai_conversation ORDER BY id DESC LIMIT 1), 'user', '谢谢！那 Channel 有哪些使用场景？', 'glm-4', 30, 1739440800;

INSERT INTO ai_message (conversation_id, role, content, model, token_count, created_ts)
SELECT (SELECT id FROM ai_conversation ORDER BY id DESC LIMIT 1), 'assistant', 'Channel 的主要使用场景：\n\n1. 任务分发：生产者-消费者模式\n2. 信号通知：通知协程停止工作\n3. 定时任务：超时控制\n4. 并发控制：限制并发数量\n5. 数据传递：在协程间传递数据', 'glm-4', 220, 1739440800;

-- ============================================================================
-- 7. 验证数据
-- ============================================================================
SELECT '测试用户创建成功' AS status, COUNT(*) AS count FROM "user" WHERE id = 1;
SELECT '测试笔记创建成功' AS status, COUNT(*) AS count FROM memo WHERE uid LIKE 'e2e_memo_%';
SELECT '测试日程创建成功' AS status, COUNT(*) AS count FROM schedule WHERE uid LIKE 'e2e_sched_%';
SELECT '测试标签创建成功' AS status, COUNT(*) AS count FROM memo_tags;
SELECT '长期记忆创建成功' AS status, COUNT(*) AS count FROM episodic_memory WHERE user_id = 1;
SELECT '对话历史创建成功' AS status, COUNT(*) AS count FROM ai_message;

-- 输出完成信息
\echo '=========================================='
\echo 'E2E 测试数据预埋完成！'
\echo '=========================================='
