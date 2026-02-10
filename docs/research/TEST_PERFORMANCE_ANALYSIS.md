# 测试性能分析报告

> **日期**: 2026-02-10
> **问题**: TestCCRunnerIntegration 拖慢 CI（超时 60+ 秒）
> **状态**: ✅ 已修复

---

## 根本原因分析

### 1. 外部依赖问题

**问题**: `TestCCRunnerIntegration` 是一个集成测试，依赖外部 CLI (`claude`)

**性能风险**:
- CLI 启动时间不确定（0.1s - 10s+）
- CLI 执行时间取决于系统负载
- 网络延迟（如果 CLI 需要认证）

**教训**: 集成测试不应该在 CI 默认运行，因为它们是非确定性的

### 2. Goroutine 泄漏

**问题**: `CCSessionManager` 在 `NewCCSessionManager` 中启动了一个 `cleanupLoop` goroutine

```go
func NewCCSessionManager(logger *slog.Logger, timeout time.Duration) *CCSessionManager {
    // ...
    go sm.cleanupLoop()  // ← 永久运行的 goroutine
    return sm
}
```

**性能风险**:
- 测试不调用 `Shutdown()` 导致 goroutine 泄漏
- Go 测试框架会等待所有 goroutines 退出
- 泄漏的 goroutine 会导致测试挂起

**修复**:
```go
runner, err := NewCCRunner(10*time.Second, nil)
defer runner.GetSessionManager().Shutdown()  // ← 必须调用
```

### 3. 超时配置不当

**问题**: 测试超时与 `go test` 超时冲突

```go
// ❌ 错误：60 秒测试超时 + 60 秒上下文超时 = 可能 120 秒
ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)

// ✅ 正确：15 秒上下文超时 < 30 秒 go test 超时
ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
```

---

## 预防措施

### 1. 集成测试规范

```go
// ✅ 正确的集成测试模式
func TestIntegration(t *testing.T) {
    // 1. 默认跳过（使用 testing.Short()）
    if testing.Short() {
        t.Skip("Skipping integration test in short mode")
    }

    // 2. 环境变量 opt-in（用于手动运行）
    if os.Getenv("DIVINESENSE_RUN_INTEGRATION_TESTS") == "" {
        t.Skip("Set DIVINESENSE_RUN_INTEGRATION_TESTS=1 to enable")
    }

    // 3. 检查外部依赖
    if _, err := exec.LookPath("claude"); err != nil {
        t.Skip("External dependency not available")
    }

    // 4. 使用 defer 确保清理
    defer cleanup()

    // 5. 使用更短的超时
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()
}
```

### 2. 资源管理规范

**原则**: 任何启动 goroutine 的类型必须提供关闭方法

```go
// ✅ 正确模式
type Manager struct {
    done chan struct{}
}

func NewManager() *Manager {
    m := &Manager{done: make(chan struct{})}
    go m.backgroundLoop()
    return m
}

func (m *Manager) Shutdown() {
    close(m.done)  // 通知 goroutine 退出
}

// 测试中
manager := NewManager()
defer manager.Shutdown()  // ← 必须调用
```

### 3. 测试超时层级

```
go test -timeout 60s          # CI 超时（最外层）
├── context.WithTimeout(30s) # 测试上下文超时（中间层）
    └── runner timeout(10s)    # 组件超时（最内层）
```

**规则**: 内层超时 < 中层超时 < 外层超时

---

## 当前测试性能状态

### 测试执行时间（`go test ./ai/agent/`）

| 测试类型 | 时间 | 状态 |
|---------|------|------|
| 单元测试 | 0.00s | ✅ 快速 |
| Mock 测试 | 0.00s | ✅ 快速 |
| 集成测试 | SKIP | ✅ 已跳过 |

### 总测试时间

```
ai/agent:        0.330s
ai/agent/tools:  0.600s
ai/agent/universal: 0.774s
```

所有测试在 1 秒内完成，符合 CI 性能要求。

---

## 长期监控建议

### 1. 添加测试超时监控

在 CI 中添加测试超时告警：

```yaml
# .github/workflows/test.yml
- name: Run tests
  run: go test ./... -timeout 60s
  timeout-minutes: 5  # GitHub Actions 超时
```

### 2. 定期检查 goroutine 泄漏

使用 `runtime.NumGoroutine()` 监控：

```go
func TestNoGoroutineLeak(t *testing.T) {
    before := runtime.NumGoroutine()

    // 运行测试
    runner := NewRunner()
    defer runner.Shutdown()

    after := runtime.NumGoroutine()
    if after != before {
        t.Errorf("Goroutine leak: %d → %d", before, after)
    }
}
```

### 3. 基准测试关键路径

为关键组件添加基准测试：

```go
func BenchmarkCCRunnerCreation(b *testing.B) {
    for i := 0; i < b.N; i++ {
        _, _ = NewCCRunner(30*time.Second, nil)
    }
}
```

---

## 决策矩阵

| 测试类型 | 是否应放在 CI? | 超时限制 | 外部依赖 |
|---------|--------------|----------|---------|
| 单元测试 | ✅ 是 | 1s | ❌ 无 |
| Mock 集成测试 | ✅ 是 | 5s | ❌ 无 |
| 真实集成测试 | ❌ 否 | N/A | ✅ 有 |

---

## 参考资料

- [Go Testing Best Practices](https://go.dev/doc/testing)
- [Table-Driven Tests](https://go.dev/wiki/TableDrivenTests)
- [Testing for Failures](https://go.dev/doc/tutorial/add-a-test)
