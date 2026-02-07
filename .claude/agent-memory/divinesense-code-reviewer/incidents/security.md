# 安全事件记录

> DivineSense 历史安全问题及修复记录

---

## 开发模式固定密钥

### 问题
```go
// server/server.go:60-63
if profile.Mode == "dev" {
    secret = "divinesense"  // 固定密钥！
}
```

### 影响
- 生产环境误用开发模式会导致会话可被预测
- 所有 JWT 签名可被伪造
- 用户认证完全失效

### 修复建议
```go
// 即使开发模式也应使用随机密钥
secret = generateRandomKey()
// 或通过环境变量显式设置
secret = os.Getenv("DIVINESENSE_SECRET_KEY")
```

### 相关文件
- `server/server.go:60-63`

---

## 历史问题

### Go Embed 文件忽略

**问题**：Vite 打包生成 `_lodash-internal.js` 被 Go embed 忽略

**修复**：配置 `manualChunks` 将 lodash 打包为单个 chunk

**相关**：
- `vite.config.mts`
- `docs/research/DEBUG_LESSONS.md`

---

### 密钥比较时序攻击

**问题**：`len(key) != 32` 使用 `!=` 比较可能被时序攻击

**修复**：使用 `crypto/subtle.ConstantTimeCompare`

**相关文件**：
- `plugin/chat_apps/store/crypto.go:22`

---

### DSN 密码打印

**问题**：开发模式下 DSN 可能包含密码但直接打印

**修复**：打印前移除密码部分

**相关文件**：
- `cmd/divinesense/main.go:182-186`
