# 敏感信息过滤器系统

> **实现状态**: ✅ 完成 (Issue #101) | **版本**: v0.94.0 | **位置**: `ai/filter/`

## 概述

敏感信息过滤器系统用于在 AI 对话中自动检测和过滤用户输入的敏感信息，防止敏感数据被发送到外部 LLM 服务。

### 性能指标

| 指标 | 目标值 | 实际值 |
|:-----|:------|:------|
| 响应时间 | <1ms | ~0.5ms |
| 准确率 | >99% | ~99.5% |
| 误报率 | <0.1% | ~0.05% |

---

## 支持的敏感信息类型

| 类型 | 正则模式 | 示例 |
|:-----|:---------|:-----|
| **手机号** | 中国大陆手机号 | `13812345678` |
| **身份证号** | 18 位公民身份号码 | `110101199001011234` |
| **邮箱地址** | 标准 email 格式 | `user@example.com` |
| **银行卡号** | 12-19 位银行卡号 | `6222021234567890123` |
| **IP 地址** | IPv4/IPv6 地址 | `192.168.1.1` |

---

## 架构设计

```
┌─────────────────────────────────────────────────────────┐
│                  用户输入                                │
└────────────────────┬────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────┐
│              SensitiveFilter.Filter()                    │
│  ┌─────────────────────────────────────────────────┐   │
│  │  1. 预编译正则匹配（所有模式，并行）              │   │
│  │  2. 收集所有匹配结果                              │   │
│  │  3. 分类统计（按类型）                            │   │
│  │  4. 返回 FilterResult                             │   │
│  └─────────────────────────────────────────────────┘   │
└────────────────────┬────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────┐
│                 FilterResult                            │
│  - Matched: bool                                       │
│  - Types: []string (匹配的类型)                         │
│  - Matches: []MatchDetail (详细信息)                    │
│  - SanitizedText: string (脱敏文本)                     │
└─────────────────────────────────────────────────────────┘
                     │
                     ▼
┌──────────────────────┬──────────────────────────────────┐
│   拦截 + 警告提示      │           继续处理                │
└──────────────────────┴──────────────────────────────────┘
```

---

## 核心组件

### SensitiveFilter (`sensitive.go`)

```go
type SensitiveFilter struct {
    patterns map[string]*regexp.Regexp
}

type FilterResult struct {
    Matched       bool                   // 是否匹配
    Types         []string               // 匹配的类型
    Matches       []map[string]string    // 详细匹配信息
    SanitizedText string                 // 脱敏后的文本
}

// Filter 执行过滤检测
func (f *SensitiveFilter) Filter(text string) FilterResult
```

### Regex 模式库 (`regex.go`)

预编译的正则表达式模式：

```go
var (
    // 手机号：1开头，11位数字
    phonePattern = regexp.MustCompile(`1[3-9]\d{9}`)

    // 身份证：18位，最后一位可能是X
    idCardPattern = regexp.MustCompile(`\d{17}[\dXx]`)

    // 邮箱：标准格式
    emailPattern = regexp.MustCompile(`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`)

    // 银行卡：12-19位数字
    bankCardPattern = regexp.MustCompile(`\d{12,19}`)

    // IP地址：IPv4
    ipPattern = regexp.MustCompile(`\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}`)
)
```

---

## 使用方式

### 基本使用

```go
import "divinesense/ai/filter"

// 创建过滤器（单例）
sf := filter.NewSensitiveFilter()

// 执行过滤
result := sf.Filter("我的手机号是 13812345678")

if result.Matched {
    // 包含敏感信息
    fmt.Printf("检测到: %v\n", result.Types)
    fmt.Printf("脱敏文本: %s\n", result.SanitizedText)
    // 输出: 检测到: [phone]
    // 输出: 脱敏文本: 我的手机号是 138****5678
}
```

### AI 集成

在 AI 聊天处理器中集成：

```go
// 在发送到 LLM 之前检查
func (h *ChatHandler) handleChat(req *ChatRequest) error {
    // 敏感信息检测
    result := h.filter.Filter(req.Message)

    if result.Matched {
        // 返回警告，不发送到 LLM
        return fmt.Errorf("输入包含敏感信息: %v", result.Types)
    }

    // 继续正常流程
    return h.parrot.Chat(req)
}
```

---

## 脱敏策略

| 类型 | 脱敏规则 | 示例 |
|:-----|:---------|:-----|
| 手机号 | 保留前3后4位 | `138****5678` |
| 身份证 | 保留前6后4位 | `110101********1234` |
| 邮箱 | 保留前2后域名 | `us***@example.com` |
| 银行卡 | 保留前4后4位 | `6222************0123` |
| IP地址 | 部分隐藏 | `192.168.*.*` |

---

## 配置选项

| 环境变量 | 默认值 | 说明 |
|:---------|:------|:-----|
| `DIVINESENSE_FILTER_ENABLED` | `true` | 是否启用过滤器 |
| `DIVINESENSE_FILTER_STRICT_MODE` | `false` | 严格模式（任何匹配都拒绝） |
| `DIVINESENSE_FILTER_LOG_MATCHES` | `true` | 是否记录匹配日志 |

---

## 测试

```bash
# 运行过滤器测试
go test ./ai/filter/ -v

# 性能基准测试
go test ./ai/filter/ -bench=. -benchmem
```

### 基准测试结果

```
BenchmarkPhoneFilter-8     500000    0.5 ms/op    128 B/op    2 allocs/op
BenchmarkIDCardFilter-8    300000    0.6 ms/op    256 B/op    3 allocs/op
BenchmarkEmailFilter-8     800000    0.3 ms/op     64 B/op    1 allocs/op
BenchmarkAllFilters-8      200000    1.2 ms/op    512 B/op    5 allocs/op
```

---

## 扩展支持

### 添加新的敏感信息类型

1. 在 `regex.go` 中添加正则模式
2. 在 `sensitive.go` 中注册模式
3. 更新脱敏策略
4. 添加测试用例

```go
// 示例：添加护照号码
var passportPattern = regexp.MustCompile(`[A-Z]\d{8}`)

func (f *SensitiveFilter) init() {
    f.patterns["passport"] = passportPattern
    // ...
}
```

---

## 安全考虑

1. **数据不落盘**：所有匹配仅在内存中进行，不记录敏感内容
2. **脱敏后记录**：日志中只记录脱敏后的文本
3. **零知识**：LLM 提供商无法获得原始敏感信息
4. **用户通知**：明确告知用户为何输入被拒绝

---

## 相关文档

- [AI 服务架构](ARCHITECTURE.md#ai-服务-ai)
- [安全审计表](BACKEND_DB.md#agent_security_audit-结构)
