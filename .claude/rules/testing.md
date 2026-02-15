# 测试规范

## 核心原则
1. **测试金字塔**：大量单元 + 少量集成 + 最少 E2E
2. **测试驱动**：关键功能先写测试
3. **覆盖率目标**：核心 > 80%，整体 > 60%

## Go 表格驱动测试
```go
func TestFunc(t *testing.T) {
    tests := []struct{
        name string
        input string
        want  bool
    }{
        {"valid", "test", true},
        {"invalid", "", false},
    }
    for _, tt := range tests { ... }
}
```

## 测试命令
```bash
make test          # 所有测试
make test-ai       # AI 测试
go test ./path -v  # 详细输出
go test -race ./... # 竞态检测
```

## 单元测试规范

### 禁止使用 time.Sleep

**禁止在单元测试中使用 `time.Sleep`** - 这会严重影响 CI 效率并导致测试不稳定。

常见问题与解决方案：

| 问题 | 错误做法 | 正确做法 |
|------|----------|----------|
| 等待异步处理 | `time.Sleep(50 * time.Millisecond)` | 禁用异步（如 `p.dedupEnabled.Store(false)`）或等待完成信号 |
| 测试超时 | `time.Sleep(1 * time.Second)` | 使用 `context.WithTimeout` 或 `assert.Eventually` |
| Race condition | `Sleep` 后检查状态 | 使用同步机制（channel、waitgroup）或重构代码 |

### 正确的异步测试模式
```go
// 错误：使用 sleep
time.Sleep(50 * time.Millisecond)
if queue.Size() != 0 { ... }

// 正确：禁用异步特性
p.dedupEnabled.Store(false)

// 正确：等待完成信号
select {
case <-done:
case <-time.After(timeout):
    t.Fatal("timeout")
}
```
