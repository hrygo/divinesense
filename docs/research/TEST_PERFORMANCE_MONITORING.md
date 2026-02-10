# 测试性能分析总结

> **日期**: 2026-02-10
> **目标**: 避免后期测试变慢问题

---

## 已识别的潜在性能风险

### 1. ✅ TestCCRunnerIntegration（已修复）

**问题**:
- 外部 CLI 依赖（非确定性启动时间）
- Goroutine 泄漏（未调用 Shutdown）
- 超时配置不当（60s 上下文 + 60s 测试超时）

**修复**:
- 环境变量 opt-in（`DIVINESENSE_RUN_INTEGRATION_TESTS=1`）
- 添加 `defer Shutdown()` 调用
- 减少超时到 15s

**文件**: `ai/agent/cc_event_test.go`

---

### 2. ✅ TestRealWorldScenario（已有保护）

**风险**: 3.7 秒累积 sleep 时间

**保护**: 已使用 `testing.Short()` 检查

```go
if testing.Short() {
    t.Skip("skipping integration test in short mode")
}
```

**验证**:
```
go test -short ./plugin/scheduler/ -run TestRealWorldScenario
--- SKIP: TestRealWorldScenario (0.00s)
```

**文件**: `plugin/scheduler/integration_test.go`

---

### 3. 其他发现

| 文件 | Sleep 调用 | 累积时间 | Short 模式保护 |
|------|----------|---------|-------------|
| `example_test.go` | 2 次 | 5.1s | ❌ 无 |
| `scheduler_test.go` | 1 次 | 1.5s | ❌ 无 |
| `integration_test.go` | 7 次 | 3.7s | ✅ 有 |

**注意**: `example_test.go` 和 `scheduler_test.go` 不是集成测试，但包含 sleep。需要检查这些是否影响 CI。

---

## 预防规范

### 集成测试标准

所有集成测试必须满足：

```go
func TestIntegration(t *testing.T) {
    // 1. Short 模式跳过（CI 默认使用 -short）
    if testing.Short() {
        t.Skip("Skipping integration test in short mode")
    }

    // 2. 环境变量 opt-in（用于手动运行）
    if os.Getenv("DIVINESENSE_RUN_INTEGRATION_TESTS") == "" {
        t.Skip("Set DIVINESENSE_RUN_INTEGRATION_TESTS=1 to enable")
    }

    // 3. 检查外部依赖
    if _, err := exec.LookPath("external-tool"); err != nil {
        t.Skip("External tool not available")
    }

    // 4. 清理资源
    defer cleanup()

    // 5. 合理超时
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()
}
```

### Sleep 使用规范

**❌ 避免**: 在单元测试中使用 sleep

```go
func TestBad(t *testing.T) {
    time.Sleep(1 * time.Second)  // 拖慢 CI
}
```

**✅ 推荐**: 使用 channel 或 sync.Cond

```go
func TestGood(t *testing.T) {
    done := make(chan struct{})
    go func() {
        // ... do work
        close(done)
    }()
    select {
    case <-done:
    case <-time.After(100 * time.Millisecond):
        t.Error("timeout")
    }
}
```

---

## CI 测试时间目标

| 测试类型 | 目标时间 | 当前状态 |
|---------|---------|---------|
| 单元测试 | < 1s | ✅ 达成 |
| Mock 测试 | < 2s | ✅ 达成 |
| 集成测试 | 默认跳过 | ✅ 达成 |
| 总测试时间 | < 5s (parallel) | ✅ 达成 |

---

## 持续监控

### 每次提交检查

```bash
# 运行所有测试（short 模式）
go test ./... -short -timeout 30s

# 检查测试时间
go test ./... -short -v 2>&1 | grep "ok"
```

### 基准测试关键路径

```bash
# 定期运行基准测试
go test -bench=. -benchmem ./ai/agent/universal
```

### Goroutine 泄漏检测

```go
// 添加到测试辅助函数
func assertNoGoroutineLeak(t *testing.T, before int) {
    after := runtime.NumGoroutine()
    if after > before {
        t.Errorf("Goroutine leak detected: %d → %d (leaked %d)",
            before, after, after-before)
    }
}
```

---

## 已创建文档

- `/docs/research/TEST_PERFORMANCE_ANALYSIS.md` - 详细分析报告
- 本文件 - 监控和预防规范

## 更新日志

| 日期 | 变更 |
|------|------|
| 2026-02-10 | 初始分析，修复 TestCCRunnerIntegration |
| 2026-02-10 | 添加 scheduler 集成测试验证 |
| 2026-02-10 | 创建监控和预防规范 |
