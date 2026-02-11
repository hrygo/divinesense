// Package universal provides execution strategy comparison and selection guide.
package universal

/*
EXECUTION STRATEGY COMPARISON GUIDE

Three execution strategies are available for UniversalParrot, each designed for
different types of tasks. Choosing the right strategy is crucial for optimal performance
and user experience.

┌────────────────┬──────────────────┬──────────────────┬─────────────────────────────────────┐
│ Strategy        │ Direct           │ ReAct             │ Planning                               │
├────────────────┼──────────────────┼──────────────────┼─────────────────────────────────────┤
│ Philosophy     │ "Just do it"     │ "Think then act"  │ "Plan then execute"                  │
│ LLM Calls      │ 1-2 (fast)       │ 2-5 (medium)     │ 3-6 (slow, with planning)            │
│ Streaming      │ ✓ Yes            │ ✓ Yes            │ ✓ Yes                                  │
│ Multi-step     │ ✗ Limited        │ ✓ Excellent      │ ✓ Designed for it                     │
│ Reasoning      │ ✗ None           │ ✓ Per iteration  │ ✓ Dedicated phase                     │
└────────────────┴──────────────────┴──────────────────┴─────────────────────────────────────┘

═════════════════════════════════════════════════════════════════════════════════
DIRECT EXECUTOR - "Just Do It"
═════════════════════════════════════════════════════════════════════════════════

BEST FOR:
  • Simple, one-shot tool calls
  • CRUD operations (create, update, delete)
  • Actions where the user provides all necessary information upfront
  • Scenarios requiring minimal reasoning or explanation

EXAMPLES:
  ✓ "创建日程：明天下午3点开会" (Create schedule: meeting tomorrow 3pm)
  ✓ "把明天的会议改到4点" (Change tomorrow's meeting to 4pm)
  ✓ "删除这个提醒" (Delete this reminder)

NOT SUITABLE FOR:
  ✗ Queries requiring analysis (e.g., "下午有空闲时间吗？")
  ✗ Multi-step planning
  ✗ Tasks requiring data synthesis from multiple sources

WHY IT FAILS ON QUERIES:
  The LLM is expected to return tool_calls + answer in one response. For complex
  queries, the LLM may keep calling tools instead of generating the final answer,
  leading to infinite loops or iteration limits.

═════════════════════════════════════════════════════════════════════════════════
REACT EXECUTOR - "Think Then Act"
═════════════════════════════════════════════════════════════════════════════════

BEST FOR:
  • Single-tool scenarios with reasoning
  • Queries requiring data interpretation
  • Search and retrieval with explanation
  • Tasks where the LLM needs to "think" before/after tool use

EXAMPLES:
  ✓ "下午有空闲时间吗？" (Is there free time this afternoon?)
  ✓ "搜索我的笔记关于人工智能的内容" (Search my notes about AI)
  ✓ "根据日程告诉我什么时候有空" (Tell me when I'm free based on schedule)

ALGORITHM:
  1. LLM streams thinking content (visible to user as it generates)
  2. LLM decides to call a tool (e.g., TOOL: schedule_query(INPUT: {...}))
  3. Execute tool, stream result to user
  4. LLM generates final answer based on tool result
  5. Repeat 2-4 until final answer is reached

WHY IT WORKS FOR QUERIES:
  The explicit "thinking" phase helps the LLM reason about the user's intent.
  After getting tool results, the LLM naturally transitions to answering without
  feeling the need to "double-check" with another tool call.

═════════════════════════════════════════════════════════════════════════════════
PLANNING EXECUTOR - "Plan Then Execute"
═════════════════════════════════════════════════════════════════════════════════

BEST FOR:
  • Multi-tool coordination
  • Complex tasks requiring upfront planning
  • Scenarios where tools can be executed in parallel
  • Tasks with clear planning phase vs execution phase

EXAMPLES:
  ✓ "规划一次旅行：查天气、订酒店、安排行程"
    (Plan a trip: check weather, book hotel, arrange itinerary)
  ✓ "分析我的日程和笔记，找出优化时间的建议"
    (Analyze my schedule and notes, suggest time optimization)

ALGORITHM:
  Phase 1 (Planning):
    1. LLM generates a plan without calling tools
    2. Plan is displayed to user for confirmation
  Phase 2 (Execution):
    3. Execute planned tool calls (potentially in parallel)
    4. LLM synthesizes results into final answer

WHY USE IT:
  • Parallel tool execution saves time
  • User can review/modify the plan before execution
  • Better for complex, multi-step tasks

═════════════════════════════════════════════════════════════════════════════════
SELECTION DECISION TREE
═════════════════════════════════════════════════════════════════════════════════

User request comes in:
       │
       ├─ Is it a simple CRUD/creation action?
       │   └─ YES → Use DIRECT
       │
       ├─ Does it require multi-tool coordination or parallel execution?
       │   └─ YES → Use PLANNING
       │
       └─ Does it involve query/search/reasoning with a single tool?
           └─ YES → Use REACT

CONFIGURATION:
  In your parrot YAML file, set the strategy field:
    strategy: direct    # for simple actions
    strategy: react      # for queries and single-tool reasoning
    strategy: planning   # for complex multi-tool tasks

*/
