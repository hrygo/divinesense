# AI 核心服务 (`ai/core`)

`core` 包提供了 DivineSense 全局使用的基础 AI 服务接口与实现。

## 概览

这些是底层的、经过充分测试的服务，构成了上层 AI 功能的基石。

```mermaid
graph TD
    App[应用层 / Agents] --> Core
    
    subgraph Core [AI 核心层]
        LLM[llm: 统一客户端]
        Embed[embedding: 向量化]
        Rerank[reranker: 重排序]
        Ret[retrieval: 混合检索]
        
        Ret --> LLM
        Ret --> Embed
        Ret --> Rerank
    end
    
    Core --> Providers[外部服务商 (OpenAI, DeepSeek, SiliconFlow)]
```

## 子包说明

### `embedding` (向量服务)
虽然是一个重度依赖模型的服务，但接口定义非常简洁：`Embed(text)` 返回 `[]float32`。
*   **统一接口**: 屏蔽了不同服务商（OpenAI, SiliconFlow, DashScope）的差异。
*   **批处理**: 支持 Batch 操作以提高吞吐量。

### `llm` (大语言模型)
提供统一的 Chat 接口，支持流式输出 (Streaming) 和工具调用 (Function Calling)。
*   **适配器模式**: 内置多个 Provider 的适配器。
*   **可观测性**: 集成了 Tracing 和 Metrics，自动记录 Token 消耗。

### `reranker` (重排序)
用于 RAG (检索增强生成) 的最后一步，对检索回来的文档进行语义重排序，提升准确率。

### `retrieval` (检索)
实现了混合检索逻辑：
1.  **Keyword Search**: 传统的关键词匹配 (BM25等)。
2.  **Vector Search**: 向量相似度检索。
3.  **Hybrid Merge**: 加权合并两者的结果。
