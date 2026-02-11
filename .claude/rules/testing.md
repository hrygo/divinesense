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
