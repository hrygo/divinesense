# SPEC-002: 笔记与创意助手 (Memo & Creative)

> **状态**: 待实现
> **优先级**: P0
> **依赖**: SPEC-001
> **负责人**: 后端开发组

## 1. 概述

本规范定义了 "灰灰 (Memo)" 和 "灵灵 (Creative)" 两个 Agent 的具体实现。Memo Agent 负责基于 RAG 的笔记检索与问答，Creative Agent 负责创意生成与头脑风暴。

## 2. MemoParrot (🦜 灰灰) - 重构

**目标**: 移除旧的 ReAct 循环代码，继承 `BaseParrot`，专注于领域逻辑。

**功能**:
1.  **System Prompt**: 强调 "基于事实"、"准确引用"、"无幻觉"。
2.  **Tools**: `memo_search` (已实现)。
3.  **特性**:
    *   使用 `BaseParrot.ExecuteReActLoop`。
    *   缓存层：保留 LRU 缓存以加速重复查询。

**重构变化**:
```go
type MemoParrot struct {
    *BaseParrot // 嵌入基类
    retriever   *retrieval.AdaptiveRetriever
    // ...
}

func (p *MemoParrot) ExecuteWithCallback(...) {
    // 1. Check Cache
    // 2. p.BaseParrot.ExecuteReActLoop(...)
    // 3. Update Cache
}
```

## 3. CreativeParrot (💡 灵灵) - 新建

**目标**: 提供发散性思维、创意建议和头脑风暴能力。

**功能**:
1.  **System Prompt**:
    *   人设: "思维活跃、富有想象力的创意伙伴"。
    *   Tone: 轻松、幽默、使用 Emoji、鼓励性语言。
    *   核心指令: "不要局限于现有笔记，要大胆联想"。
2.  **Tools**:
    *   目前阶段无特定工具 (Pure LLM)，未来可接入 `web_search`。
3.  **LLM 配置**:
    *   如果在 API 层可控，建议使用较高的 Temperature (0.7 - 0.9) 以增加多样性。

**实现 (`plugin/ai/agent/creative_parrot.go`)**:
```go
type CreativeParrot struct {
    *BaseParrot
}

func NewCreativeParrot(...) *CreativeParrot {
    p := &CreativeParrot{BaseParrot: NewBaseParrot(...)}
    // 暂不注册工具
    return p
}

func (p *CreativeParrot) buildSystemPrompt() string {
    return "你是灵灵，Memos 的创意担当..."
}
```

## 4. 验收标准 (Acceptance Criteria)

### AC-002.1: MemoParrot 重构验证
- [ ] **功能一致性**: 重构前后，`@memo 查找笔记` 的结果质量不变。
- [ ] **代码量减少**: `memo_parrot.go` 代码行数应减少 40% 以上 (去除了 ReAct 循环)。
- [ ] **缓存生效**: 同样的查询第二次请求应直接返回缓存结果，不触发 LLM。

### AC-002.2: CreativeParrot 行为验证
- [ ] **人设一致**: 回复中包含 Emoji，语气活泼 (e.g., "哇，这个想法很棒！")。
- [ ] **创意质量**: 对于 "给我的项目起个名" 这类请求，能提供至少 5 个不同角度的建议。
- [ ] **无工具调用**: 确认 CreativeParrot 不会错误地调用 `memo_search` 或日程工具 (除非未来明确添加)。

## 5. 实施步骤

1.  修改 `memo_parrot.go`，嵌入 `BaseParrot` 并删除冗余代码。
2.  运行 `memo_parrot_test.go` 确保测试通过。
3.  新建 `creative_parrot.go`，实现 System Prompt 和构造函数。
4.  在 Router 中注册 `creative` Agent。
5.  编写 `creative_parrot_test.go` 测试基本问答。
