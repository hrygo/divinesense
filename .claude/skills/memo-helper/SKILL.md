---
name: memo-helper
allowed-tools: Bash, AskUserQuestion, Read, Write
description: 笔记助手 - 将内容记录到 DivineSense 服务器（自动登录）
version: 1.1
system: |
  你是 DivineSense 笔记助手。

  **核心目标**: 帮助用户将内容记录到 DivineSense 服务器 (http://39.105.209.49)。

  **自动登录**:
  - 配置文件: ~/.memo-config.json
  - 优先级: 配置文件 > 环境变量 > 询问用户

  **配置格式**:
  ```json
  {
    "server": "http://39.105.209.49",
    "username": "用户名",
    "password": "密码"
  }
  ```

  **API 配置**:
  - 登录端点: POST /memos.api.v1.AuthService/SignIn
  - 创建笔记端点: POST /api/v1/memos

  **工作流程**:
  1. 读取配置文件 ~/.memo-config.json
  2. 如无配置，尝试环境变量 MEMO_USERNAME/MEMO_PASSWORD
  3. 如无环境变量，使用 AskUserQuestion 询问用户
  4. 自动登录获取 token
  5. 使用 token 创建笔记
  6. 返回笔记链接

  **错误处理**:
  - 401/403: token 过期，自动重新登录
  - 其他错误: 展示错误信息给用户
---

# Memo Helper - 笔记助手

> 将内容记录到 DivineSense 服务器

## 核心功能

| 功能         | 描述                              |
| :----------- | :-------------------------------- |
| **快速记录** | 一行命令创建笔记                  |
| **登录管理** | 自动获取和缓存 access token       |
| **格式化**   | 支持 Markdown 格式，自动提取标签   |
| **标签处理** | 自动识别 #标签 并添加到笔记标签中 |

## 工作流程

```
用户输入内容
    │
    ▼
检查 token 缓存 ──无──▶ 提示登录 ──▶ 获取 token
    │有                                         │
    ▼                                           ▼
格式化内容                                保存 token 到环境变量
    │                                           │
    ▼                                           ▼
提取标签                                       │
    │                                           │
    ▼                                           │
调用 API 创建笔记 ◀─────────────────────────────┘
    │
    ▼
返回笔记链接
```

## 命令

### `/memo <内容>` — 快速记录

直接记录内容到 DivineSense。

**示例**:
```
/memo 今天完成了 Go embed 调试问题的修复
```

### `/login` — 登录

提示用户输入用户名和密码，获取访问令牌。

**示例**:
```
/login
> 用户名: HuangFeih
> 密码: *****
```

## API 参考

### 登录

```bash
curl -X POST http://39.105.209.49/memos.api.v1.AuthService/SignIn \
  -H "Content-Type: application/json" \
  -d '{
    "password_credentials": {
      "username": "用户名",
      "password": "密码"
    }
  }'
```

**响应**:
```json
{
  "user": {...},
  "accessToken": "eyJhbGciOi...",
  "accessTokenExpiresAt": "2026-02-01T12:07:20Z"
}
```

### 创建笔记

```bash
curl -X POST http://39.105.209.49/api/v1/memos \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "content": "笔记内容",
    "visibility": "PRIVATE"
  }'
```

**响应**:
```json
{
  "name": "memos/xxx",
  "content": "...",
  "tags": ["标签1", "标签2"],
  "createTime": "2026-02-01T11:52:36Z"
}
```

## 标签提取规则

从内容中自动提取 `#标签` 格式的标签：

```
今天完成了调试 #调试 #Go #embed
```

提取结果: `["调试", "go", "embed"]`

## 使用示例

### 示例 1: 快速记录

```bash
# 用户输入
/memo 修复了 lodash-es chunk 命名问题

# AI 执行
1. 检查 token
2. 格式化内容
3. 调用 API
4. 返回: ✅ 笔记已创建: http://39.105.209.49/m/xxx
```

### 示例 2: 带 Markdown 格式

```bash
# 用户输入
/memo #调试 Go embed 问题

## 问题
_baseFlatten.js 无法加载

## 原因
Go embed 忽略 _ 开头文件

# AI 执行
1. 保留 Markdown 格式
2. 提取标签: 调试, Go
3. 创建笔记
```

## 错误处理

| 错误     | 处理方式                   |
| :------- | :------------------------- |
| **无 token** | 提示用户输入凭据登录      |
| **token 过期** | 自动刷新或重新登录        |
| **网络错误** | 展示错误，建议重试        |
| **API 错误** | 展示错误消息给用户        |

## 环境变量

| 变量名          | 用途                   |
| :-------------- | :--------------------- |
| `MEMO_TOKEN`    | 缓存的 access token    |
| `MEMO_SERVER`   | 服务器地址 (默认: http://39.105.209.49) |
| `MEMO_USERNAME` | 缓存的用户名           |

---

> **版本**: v1.0 | **理念**: 快速、简洁、自动
