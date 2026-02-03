# v0.90.0 - AI Core Module Refactoring

> **Major Release**: AI 模块重大架构升级

---

## 核心变更 (Core Changes)

### 🏗️ AI 模块提升为一级模块

**Pull Requests**:
- [#54](https://github.com/hrygo/divinesense/pull/54) - `refactor(ai): promote AI module from plugin/ai to first-level ai/`
- [#56](https://github.com/hrygo/divinesense/pull/56) - `refactor(ai): AI Core module promotion - 提升 AI 为一级模块`

**变更范围**:
- **193 files changed**: `plugin/ai/*` → `ai/*`
- 新的模块结构：
  ```
  ai/                          # 一级 AI 模块 (新)
  ├── agent/                   # AI 代理
  │   ├── chat_router.go       # 聊天路由器
  │   ├── geek_parrot.go       # 极客代理
  │   ├── evolution_parrot.go  # 进化代理
  │   ├── memo_parrot.go       # 笔记代理
  │   ├── schedule_parrot_v2.go # 日程代理
  │   ├── amazing_parrot.go    # 综合代理
  │   ├── cc_runner/           # CC Runner (持久会话)
  │   └── tools/               # 代理工具
  ├── context/                 # 上下文构建器
  ├── memory/                  # 记忆系统
  ├── metrics/                 # 性能指标
  ├── router/                  # 意图路由
  ├── session/                 # 会话管理
  ├── vector/                  # 向量嵌入
  └── cache/                   # LRU 缓存
  ```

**架构优势**:
1. **模块独立性**: AI 不再作为插件，成为系统核心组件
2. **更清晰的导入路径**: `ai/agent/xxx` 替代 `plugin/ai/agent/xxx`
3. **为未来扩展铺路**: 支持更多 AI 能力集成

---

## 新增功能 (Features)

### 📋 会话嵌套模型调研

**Issue**: [#57](https://github.com/hrygo/divinesense/issues/57)

解决了三种模式之间的会话上下文割裂问题：

**问题**:
- 普通模式、极客模式、进化模式各自独立存储会话
- 跨模式切换时上下文丢失
- 前端无法统一显示所有会话历史

**方案**: 母会话-子会话嵌套模型
- Geek/Evolution 作为 Normal 的子会话运行
- 完整保存 Q+A（不生成摘要）
- 支持追加式输入（多 Q 单 A）
- 保留 cc_session_id 用于本地追溯

**文档**: [session-nested-model-research.md](https://github.com/hrygo/divinesense/blob/main/docs/research/session-nested-model-research.md)

---

## 其他更新 (Other Changes)

### 📡 Chat Apps 接入调研

新增 Telegram/WhatsApp/钉钉集成调研报告：
- [chat-apps-integration-research.md](https://github.com/hrygo/divinesense/blob/main/docs/research/chat-apps-integration-research.md)

### 🤖 Agent Skills 重组

- `.agent/skills/` 目录结构优化
- 更新技能加载逻辑

---

## 升级指南

### 开发者

更新导入路径：
```go
// 旧路径
import "github.com/hrygo/divinesense/plugin/ai/agent"

// 新路径
import "github.com/hrygo/divinesense/ai/agent"
```

### 部署者

无额外配置要求，AI 模块功能保持兼容。

---

## Full Changelog

**[v0.81.0...v0.90.0](https://github.com/hrygo/divinesense/compare/v0.81.0...v0.90.0)**

---

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>
