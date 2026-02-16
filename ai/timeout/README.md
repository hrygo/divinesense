# AI Timeout Constants (`ai/timeout`)

`timeout` 包集中管理 AI 模块所有的超时常量配置，确保系统在面对外部调用（如 LLM API）时的健壮性。

## 核心常量

### Agent Execution
*   `AgentTimeout` / `AgentExecutionTimeout`: **2 分钟**。整个 Agent 处理流程（思考+工具调用）的最大时长。
*   `MaxIterations`: **5 次**。ReAct 循环的最大思考轮数，防止死循环。

### LLM & Tools
*   `StreamTimeout`: **5 分钟**。流式响应的最大保持时间。
*   `ToolExecutionTimeout`: **30 秒**。单个工具调用的超时限制。
*   `EmbeddingTimeout`: **30 秒**。向量生成的超时限制。

### Fault Tolerance
*   `MaxToolFailures`: **3 次**。允许工具连续失败的最大次数，超过则中断任务。
*   `MaxRecentToolCalls`: **10**。循环检测窗口大小。

## 使用建议
直接导入该包使用常量，而非在代码中硬编码数字，以便于统一调整系统策略。
```go
ctx, cancel := context.WithTimeout(parentCtx, timeout.AgentTimeout)
defer cancel()
```
