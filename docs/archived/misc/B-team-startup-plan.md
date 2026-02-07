# B 团队启动工作规划

> **生成时间**: 2026-01-27  
> **状态**: ✅ Sprint 1 + Sprint 2 全部完成

---

## Sprint 0 验收状态

| 检查项 | 状态 | 说明 |
|:---|:---:|:---|
| 接口定义 (7个) | ✅ | `plugin/ai/*/interface.go` 全部完成 |
| Mock 实现 (7个) | ✅ | `plugin/ai/*/mock.go` 全部完成 |
| 契约测试 | ✅ | 4 个核心服务测试全部通过 |
| 现有代码分析 | ✅ | 发现可复用组件 |

---

## Sprint 1 完成状态 ✅

| Spec ID | 任务 | 状态 | 交付物 |
|:---|:---|:---:|:---|
| **P1-B002** | 错误恢复机制 | ✅ | `errors.go`, `recovery.go`, `recovery_test.go` |
| **P1-B001** | 工具可靠性增强 | ✅ | `tools/executor.go`, `tools/fallback.go`, `tools/executor_test.go` |

### Code Review 修复项

| Issue | 优先级 | 修复状态 |
|:---|:---:|:---:|
| ctx取消时重试循环未正确退出 | HIGH | ✅ |
| ExecuteDetailed fallback错误被丢弃 | HIGH | ✅ |
| 删除未使用的 maxRetries 字段 | MEDIUM | ✅ |
| fallbackRules 复制避免并发风险 | MEDIUM | ✅ |
| 预编译正则表达式 | LOW | ✅ |
| WithTimezone 返回新实例 | LOW | ✅ |

---

## Sprint 2 完成状态 ✅

| Spec ID | 任务 | 状态 | 交付物 |
|:---|:---|:---:|:---|
| **P1-B003** | 时间解析加固 | ✅ | `schedule/time_hardener.go`, `schedule/time_hardener_test.go` |
| **P1-B004** | 规则分类器扩展 | ✅ | `schedule/schedule_intent_classifier.go`, `schedule/schedule_intent_classifier_test.go` |

### P1-B003 时间解析加固

- **中文数字转换**: 支持 "三点" → "3点", "十一点" → "11点", "二十一" → "21"
- **格式标准化**: "早上" → "上午", "点钟" → "点", "点半" → "点30分"
- **智能日期推断**: 无日期时根据当前时间推断今天/明天
- **时间验证**: 不能早于现在、不能超过一年
- **测试覆盖**: 20+ 测试用例

### P1-B004 规则分类器扩展

- **规则匹配 (0ms)**: 预编译正则实现快速分类
- **四种意图**: SimpleCreate, SimpleQuery, SimpleUpdate, BatchCreate
- **智能区分**: Query vs Create (有什么安排 vs 安排会议)
- **LLM 降级**: 规则未匹配时调用 RouterService
- **测试覆盖**: 25+ 测试用例，含 Benchmark

---

## 代码组织结构 (实际)

```
plugin/ai/
├── agent/
│   ├── tools/
│   │   ├── executor.go      [P1-B001] ✅
│   │   ├── fallback.go      [P1-B001] ✅
│   │   └── executor_test.go [P1-B001] ✅
│   ├── recovery.go          [P1-B002] ✅
│   ├── recovery_test.go     [P1-B002] ✅
│   ├── errors.go            [P1-B002] ✅
│   └── ...
├── schedule/
│   ├── time_hardener.go              [P1-B003] ✅
│   ├── time_hardener_test.go         [P1-B003] ✅
│   ├── schedule_intent_classifier.go      [P1-B004] ✅
│   └── schedule_intent_classifier_test.go [P1-B004] ✅
└── ...
```

---

## 开发进度清单

- [x] Sprint 0 接口契约验证通过
- [x] Mock 实现可用于并行开发
- [x] 现有代码分析完成
- [x] **P1-B002**: 错误恢复机制 ✅
- [x] **P1-B001**: 工具可靠性增强 ✅
- [x] Code Review + 修复 ✅
- [x] **P1-B003**: 时间解析加固 ✅
- [x] **P1-B004**: 规则分类器扩展 ✅

---

## 风险与缓解

| 风险 | 缓解措施 | 状态 |
|:---|:---|:---:|
| Mock 与真实实现行为差异 | 契约测试约束 + 集成测试验证 | 监控中 |
| A 团队 Sprint 1 延期 | Sprint 2 任务使用 Mock 先行开发 | ✅ 已完成 |
| 时间解析边界情况 | TimeService Mock 已覆盖中文场景 | ✅ 已验证 |

---

## 总结

B 团队 Phase 1 全部任务已完成：

| Sprint | 任务数 | 状态 |
|:---|:---:|:---:|
| Sprint 0 | 接口契约 | ✅ |
| Sprint 1 | P1-B001, P1-B002 | ✅ |
| Sprint 2 | P1-B003, P1-B004 | ✅ |

**测试统计**:
- P1-B001 工具可靠性: 22 测试
- P1-B002 错误恢复: 30+ 测试
- P1-B003 时间加固: 20+ 测试
- P1-B004 规则分类: 25+ 测试

等待与 A 团队真实实现集成验证。
