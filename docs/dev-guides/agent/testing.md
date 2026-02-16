# AI 代理测试指南

> **保鲜状态**: ✅ 已验证 (2026-02-12) | **最后检查**: v0.99.0 | **建议**: 测试流程已验证

> DivineSense AI 代理 (Parrot) 的测试与验证方法

---

## 三种测试方式

### 方式 1: 交互式测试脚本（最简单）

```bash
# 1. 启动服务
make start

# 2. 运行测试脚本
chmod +x scripts/test_schedule_agent.sh
./scripts/test_schedule_agent.sh
```

---

### 方式 2: Go 测试程序（推荐）

```bash
# 1. 确保数据库运行
make docker-up

# 2. 配置 .env 文件
cat >> .env << 'EOF'
DIVINESENSE_AI_ENABLED=true
DIVINESENSE_AI_LLM_PROVIDER=deepseek
DIVINESENSE_AI_LLM_MODEL=deepseek-chat
DIVINESENSE_AI_LLM_API_KEY=your_key_here
# Embedding (Required for RAG/Memory)
DIVINESENSE_AI_EMBEDDING_PROVIDER=siliconflow
DIVINESENSE_AI_EMBEDDING_API_KEY=your_embedding_key
EOF

# 3. 运行测试程序
go run ./cmd/test-agent/main.go
```

---

### 方式 3: 手动 API 测试

#### 步骤 1: 启动服务

```bash
# 启动所有服务
make start

# 或分别启动
make docker-up  # 数据库
make run        # 后端（新终端）
make web       # 前端（新终端）
```

#### 步骤 2: 获取 Token

```bash
# 登录获取 token
curl -X POST http://localhost:28081/api/v1/auth/signin \
  -H "Content-Type: application/json" \
  -d '{
    "username": "your_username",
    "password": "your_password"
  }'
```

#### 步骤 3: 测试 API

```bash
# 设置 token
export TOKEN="your_access_token_here"

# 测试 1: 查询日程
curl -X POST http://localhost:28081/api/v1/ai/chat \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "message": "查看明天有什么安排",
    "user_timezone": "Asia/Shanghai"
  }'

# 测试 2: 创建日程
curl -X POST http://localhost:28081/api/v1/ai/chat \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "message": "后天上午10点开个产品会",
    "user_timezone": "Asia/Shanghai"
  }'
```

---

## 验证清单

### 基础验证

```bash
# 1. 数据库连接
make db-connect
# 应该进入 psql shell

# 2. 后端服务
curl http://localhost:28081
# 应该返回 404 或 API 信息

# 3. AI 功能
curl http://localhost:28081/api/v1/status
# 检查 ai.enabled 是否为 true
```

### 智能体验证

#### 测试查询
```
输入: "明天有什么安排？"
预期: 返回日程列表或"暂无日程"
```

#### 测试创建
```
输入: "后天下午2点开个会"
预期:
  - 如果无冲突: "成功创建日程..."
  - 如果有冲突: "发现冲突..."
```

---

## 常见问题

### ❌ "AI features are disabled"

```bash
# 检查环境变量
echo $DIVINESENSE_AI_ENABLED

# 修复
echo "DIVINESENSE_AI_ENABLED=true" >> .env
make stop && make start
```

### ❌ "Failed to create LLM service"

```bash
# 检查配置
cat .env | grep AI

# 确保 API key 正确
echo $DIVINESENSE_AI_LLM_API_KEY
```

### ❌ "Database connection failed"

```bash
# 检查数据库
make docker-up
make db-connect

# 重置数据库（如果需要）
make db-reset
```

---

## 查看日志

```bash
# 实时查看所有日志
make logs

# 只查看后端日志
make logs-follow-backend

# 过滤代理相关日志
make logs-follow-backend | grep -i "agent\|schedule"
```

---

## 运行单元测试

```bash
# 测试 Service 层
go test ./server/service/schedule/... -v

# 测试工具层
go test ./ai/agents/tools/... -v

# 测试智能体
go test ./ai/agents/... -v

# 查看覆盖率
go test ./server/service/schedule/... -cover
```

---

## 🔗 相关文档

- [架构文档](ARCHITECTURE.md) - AI 代理系统详情
- [后端开发指南](BACKEND_DB.md) - 后端开发和数据库策略
- [前端开发指南](FRONTEND.md) - 前端架构和布局模式
