# OpenClaw 对标报告

**生成时间**: {{timestamp}}
**OpenClaw 版本**: {{openclaw_version}}
**DivineSense 版本**: {{divinesense_version}}

---

## 执行摘要

| 指标 | 数值 |
|:-----|:-----|
| 扫描功能数 | {{total_features}} |
| 差距功能数 | {{gap_count}} |
| 高优先级 | {{high_priority_count}} |
| 中优先级 | {{medium_priority_count}} |
| 过滤功能数 | {{filtered_count}} |

---

## 功能差距矩阵

{% for category in categories %}
### {{category.name}}

| 功能 | OpenClaw | DivineSense | 技术契合 | 用户价值 | 优先级 |
|:-----|:---------|:------------|:---------|:---------|:-------|
{% for feature in category.features %}
| {{feature.name}} | ✅ | {% if feature.exists %}✅{% else %}❌{% endif %} | {{feature.tech_fit}} | {{feature.user_value}} | {{feature.priority}} |
{% endfor %}

{% endfor %}

---

## 高优先级功能 (≥0.7)

{% for feature in high_priority_features %}
### {{feature.name}}

- **技术契合度**: {{feature.tech_fit}}
- **用户价值**: {{feature.user_value}}
- **优先级**: {{feature.priority}}
- **参考文件**: `{{feature.openclaw_files}}`

{% endfor %}

---

## 生成的 Issue

{% for issue in issues %}
- **[#{{issue.number}}] {{issue.title}}**
  - 优先级: {{issue.priority}}
  - 包含功能: {{issue.features}}

{% endfor %}

---

## 过滤的功能

| 功能 | 过滤原因 |
|:-----|:---------|
{% for feature in filtered_features %}
| {{feature.name}} | {{feature.filter_reason}} |
{% endfor %}

---

## 附录

### 对标方法论

**优先级计算公式**:
```
优先级 = 技术契合度 × 0.4 + 用户价值 × 0.6
```

**过滤规则**:
- TypeScript 特异性功能（如 jiti、tsx）
- 多渠道集成（不同产品定位）
- 需要外部服务的功能

### 相关文件

- 对标配置: `.claude/skills/competitive-benchmark/`
- 历史报告: `docs/research/benchmark/`

---

*本报告由 competitive-benchmark skill 自动生成*
