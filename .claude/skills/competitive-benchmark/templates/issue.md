# GitHub Issue 模板 - 竞品对标

> competitive-benchmark skill 生成的 Issue 模板

---

## [类型] 功能标题

### 竞品来源

**对标产品**: OpenClaw (https://github.com/openclaw/openclaw)
**对标版本**: {{openclaw_version}}
**参考功能**: {{reference_features}}

### 问题描述

<!-- 描述 OpenClaw 中的这个功能是什么 -->

### 当前行为

<!-- DivineSense 当前的行为（如不存在则写"不支持"） -->

### 期望行为

<!-- 描述期望实现的功能 -->

### 解决方案

#### 功能范围

**包含**：
- [功能点 1]
- [功能点 2]

**不包含**：
- [未来扩展]

#### 技术方案

**后端变更**：
- [API 变更说明]
- [数据模型变更]

**前端变更**：
- [新增/修改的页面和组件]
- [路由变更]
- [i18n keys 位置]

**AI 代理**（如适用）：
- [涉及的代理类型]
- [新增的工具]

#### 参考资源

- [OpenClaw 参考实现]({{openclaw_reference}})
- [相关文档链接]
- 📄 详细对标报告：`docs/research/benchmark/{{report_filename}}`

### 对标分析

| 维度 | 评分 | 说明 |
|:-----|:-----|:-----|
| **技术契合度** | {{tech_fit}}/1.0 | {{tech_fit_notes}} |
| **用户价值** | {{user_value}}/1.0 | {{user_value_notes}} |
| **优先级** | {{priority}}/1.0 | 技术契合度×0.4 + 用户价值×0.6 |

### 实现复杂度

- **工作量估算**: {{effort}} 人周
- **风险等级**: {{risk_level}}

### 验收标准

- [ ] [可测试的标准 1]
- [ ] [可测试的标准 2]
- [ ] `make check-all` 通过
- [ ] 已更新文档（如需要）

### 依赖项

- [ ] 前置 Issue #xxx
- [ ] 其他依赖

---

**对标生成时间**: {{timestamp}}
**对标版本**: v1.3
