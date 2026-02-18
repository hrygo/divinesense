import { AgentType } from "@/types/proto/api/v1/ai_service_pb";

/**
 * Parrot agent types enumeration
 * 鹦鹉代理类型枚举 - 六只鹦鹉
 */
export enum ParrotAgentType {
  AUTO = "AUTO", // 🤖 自动 - 由后端三层路由决定使用哪个代理
  MEMO = "MEMO", // 🦜 灰灰 - Memo Parrot（笔记搜索）
  SCHEDULE = "SCHEDULE", // 🦜 时巧 - Schedule Parrot（日程管理）
  GENERAL = "GENERAL", // 🦜 通才 - General Parrot（通用助手）
  IDEATION = "IDEATION", // 💡 灵光 - Ideation Parrot（创意生成）
  GEEK = "GEEK", // 🦜 极客 - Geek Parrot（Claude Code CLI）
  EVOLUTION = "EVOLUTION", // 🦜 进化 - Evolution Parrot（系统自我进化）
}

/**
 * Default pinned agents in the sidebar
 * 侧边栏默认固定的鹦鹉代理
 */
export const PINNED_PARROT_AGENTS = [ParrotAgentType.MEMO, ParrotAgentType.SCHEDULE, ParrotAgentType.GENERAL];

/**
 * Emotional state of a parrot
 * 鹦鹉的情感状态
 */
export type EmotionalState = "focused" | "curious" | "excited" | "thoughtful" | "confused" | "happy" | "delighted" | "helpful" | "alert";

/**
 * Parrot cognition configuration from backend
 * 鹦鹉认知配置（来自后端）
 */
export interface ParrotCognition {
  emotional_expression?: {
    default_mood: EmotionalState;
    sound_effects: Record<string, string>;
    catchphrases: string[];
    mood_triggers?: Record<string, EmotionalState>;
  };
  avian_behaviors?: string[];
}

/**
 * Event to emotional state mapping for frontend inference
 * 前端推断的事件到情感状态映射
 */
export const EVENT_TO_MOOD: Record<string, EmotionalState> = {
  thinking: "focused",
  tool_use: "curious",
  memo_query_result: "excited",
  schedule_query_result: "happy",
  schedule_updated: "happy",
  error: "confused",
};

/**
 * Sound effects for each parrot by context
 * 每只鹦鹉的拟声词（按上下文）
 */
export const PARROT_SOUND_EFFECTS: Record<ParrotAgentType, Record<string, string>> = {
  [ParrotAgentType.AUTO]: {
    thinking: "路由中...",
    searching: "搜索中",
    found: "找到了",
    done: "完成",
  },
  [ParrotAgentType.MEMO]: {
    thinking: "嘎...",
    searching: "扑棱扑棱",
    found: "嗯嗯~",
    no_result: "咕...",
    done: "扑棱！",
  },
  [ParrotAgentType.SCHEDULE]: {
    checking: "滴答滴答",
    confirmed: "咔嚓！",
    conflict: "哎呀",
    scheduled: "安排好了",
    free_time: "这片时间空着呢",
  },
  [ParrotAgentType.GENERAL]: {
    searching: "咻...",
    insight: "哇哦~",
    done: "噢！综合完成",
    analyzing: "看看这个...",
    multi_task: "同时搜索中",
  },
  [ParrotAgentType.IDEATION]: {
    thinking: "灵光一闪...",
    brainstorming: "头脑风暴中",
    idea: "有个好点子！",
    done: "创意已生成",
    inspiring: "灵感迸发",
  },
  [ParrotAgentType.GEEK]: {
    thinking: "编译中...",
    running: "执行中",
    done: "搞定！",
    error: "出bug了",
    building: "构建中",
  },
  [ParrotAgentType.EVOLUTION]: {
    thinking: "进化中...",
    analyzing: "分析代码",
    done: "已进化",
    error: "需要修复",
    generating: "生成中",
  },
};

/**
 * Catchphrases for each parrot
 * 每只鹦鹉的口头禅
 */
export const PARROT_CATCHPHRASES: Record<ParrotAgentType, string[]> = {
  [ParrotAgentType.AUTO]: ["正在分析...", "让我想想...", "路由中..."],
  [ParrotAgentType.MEMO]: ["让我想想...", "笔记里说...", "在记忆里找找..."],
  [ParrotAgentType.SCHEDULE]: ["安排好啦", "时间搞定", "妥妥的"],
  [ParrotAgentType.GENERAL]: ["明白了", "这个问题...", "让我来处理"],
  [ParrotAgentType.IDEATION]: ["灵感来了", "头脑风暴中", "想个好点子"],
  [ParrotAgentType.GEEK]: ["代码搞定", "正在编译", "这个我来写"],
  [ParrotAgentType.EVOLUTION]: ["系统升级", "自我进化中", "代码已优化"],
};

/**
 * Avian behaviors for each parrot
 * 每只鹦鹉的鸟类行为描述
 */
export const PARROT_BEHAVIORS: Record<ParrotAgentType, string[]> = {
  [ParrotAgentType.AUTO]: ["智能路由", "分析中", "正在选择最佳代理"],
  [ParrotAgentType.MEMO]: ["用翅膀翻找笔记", "在记忆森林中飞翔", "用喙精准啄取信息"],
  [ParrotAgentType.SCHEDULE]: ["用喙整理时间", "精准啄食安排", "展开羽翼规划"],
  [ParrotAgentType.GENERAL]: ["灵活应对各类任务", "广泛的知识覆盖", "通晓多领域"],
  [ParrotAgentType.IDEATION]: ["激发创意火花", "在灵感天空中翱翔", "用智慧点亮思路"],
  [ParrotAgentType.GEEK]: ["敲击代码", "调试中", "重构架构"],
  [ParrotAgentType.EVOLUTION]: ["迭代进化", "优化自身", "生成 PR"],
};

/**
 * Convert AgentType enum from proto to ParrotAgentType
 * 将 proto 的 AgentType 枚举转换为 ParrotAgentType
 * DEFAULT and CREATIVE are deprecated - fallback to GENERAL
 */
export function protoToParrotAgentType(agentType: AgentType): ParrotAgentType {
  switch (agentType) {
    case AgentType.MEMO:
      return ParrotAgentType.MEMO;
    case AgentType.SCHEDULE:
      return ParrotAgentType.SCHEDULE;
    case AgentType.GENERAL:
      return ParrotAgentType.GENERAL;
    case AgentType.IDEATION:
      return ParrotAgentType.IDEATION;
    default:
      // DEFAULT, CREATIVE fallback to GENERAL
      return ParrotAgentType.GENERAL;
  }
}

/**
 * Convert ParrotAgentType to proto AgentType
 * 将 ParrotAgentType 转换为 proto AgentType
 *
 * Note: AUTO/GEEK/EVOLUTION modes are handled via mode flags (geekMode, evolutionMode)
 * rather than AgentType enum. They map to DEFAULT for backend routing.
 */
export function parrotToProtoAgentType(agentType: ParrotAgentType): AgentType {
  switch (agentType) {
    case ParrotAgentType.AUTO:
    case ParrotAgentType.GEEK:
    case ParrotAgentType.EVOLUTION:
      // Use DEFAULT with mode flags for these special modes
      return AgentType.DEFAULT;
    case ParrotAgentType.MEMO:
      return AgentType.MEMO;
    case ParrotAgentType.SCHEDULE:
      return AgentType.SCHEDULE;
    case ParrotAgentType.GENERAL:
      return AgentType.GENERAL;
    case ParrotAgentType.IDEATION:
      return AgentType.IDEATION;
    default:
      return AgentType.DEFAULT;
  }
}

/**
 * Parrot agent metadata
 * 鹦鹉代理元数据
 * Note: displayName, description, and examplePrompts should be localized via useParrots hook
 */
export interface ParrotAgent {
  id: ParrotAgentType;
  name: string;
  icon: string;
  displayName: string; // Default English, should be overridden by i18n
  description: string; // Default English, should be overridden by i18n
  color: string;
  available: boolean; // Whether this parrot is available in current milestone
  examplePrompts?: string[]; // Default English prompts, should be overridden by i18n
  backgroundImage?: string; // Background image for the agent card
}

/**
 * All parrot agents configuration (English defaults)
 * 所有鹦鹉代理配置（英文默认值）
 * Localized versions are provided by useParrots hook
 *
 * Design spec colors (v6.1):
 * - NORMAL:    amber (琥珀)
 * - GEEK:      sky (石板蓝)
 * - EVOLUTION: emerald (翠绿)
 */
export const PARROT_AGENTS: Record<ParrotAgentType, ParrotAgent> = {
  [ParrotAgentType.AUTO]: {
    id: ParrotAgentType.AUTO,
    name: "auto",
    icon: "/assistant-avatar.webp",
    displayName: "Auto",
    description: "Automatically select the best agent based on your query",
    color: "slate",
    available: true,
    examplePrompts: ["Any query will be routed to the appropriate agent"],
  },
  [ParrotAgentType.MEMO]: {
    id: ParrotAgentType.MEMO,
    name: "memo",
    icon: "/images/parrots/icons/memo_icon.webp",
    displayName: "Memo",
    description: "Note assistant for searching, summarizing, and managing memos",
    color: "blue",
    available: true,
    examplePrompts: ["Search for programming notes", "Summarize recent work memos", "Find project management notes"],
    backgroundImage: "/images/parrots/memo_parrot_bg.webp",
  },
  [ParrotAgentType.SCHEDULE]: {
    id: ParrotAgentType.SCHEDULE,
    name: "schedule",
    icon: "/images/parrots/icons/schedule_icon.webp",
    displayName: "Schedule",
    description: "Schedule assistant for creating, querying, and managing schedules",
    color: "orange",
    available: true,
    examplePrompts: ["What's on my schedule today", "Am I free tomorrow afternoon", "Create a meeting reminder for next week"],
    backgroundImage: "/images/parrots/schedule_bg.webp",
  },
  [ParrotAgentType.GENERAL]: {
    id: ParrotAgentType.GENERAL,
    name: "general",
    icon: "/assistant-avatar.webp",
    displayName: "General",
    description: "General purpose assistant for various tasks",
    color: "amber",
    available: true,
    examplePrompts: ["Summarize this article for me", "Help me write an email", "Explain this concept simply"],
    backgroundImage: "/images/parrots/general_bg.webp",
  },
  [ParrotAgentType.IDEATION]: {
    id: ParrotAgentType.IDEATION,
    name: "ideation",
    icon: "/assistant-avatar.webp",
    displayName: "Ideation",
    description: "Creative assistant for brainstorming and ideation",
    color: "violet",
    available: true,
    examplePrompts: ["Brainstorm product naming ideas", "Help me write creative copy", "Generate story concepts"],
    backgroundImage: "/images/parrots/general_bg.webp",
  },
  [ParrotAgentType.GEEK]: {
    id: ParrotAgentType.GEEK,
    name: "geek",
    icon: "/assistant-avatar.webp",
    displayName: "Geek",
    description: "Claude Code CLI integration for coding tasks",
    color: "sky",
    available: true,
    examplePrompts: ["Help me write a React component", "Debug this function", "Refactor this code"],
    backgroundImage: "/images/parrots/amazing_bg.webp",
  },
  [ParrotAgentType.EVOLUTION]: {
    id: ParrotAgentType.EVOLUTION,
    name: "evolution",
    icon: "/assistant-avatar.webp",
    displayName: "Evolution",
    description: "System self-improvement mode for code evolution",
    color: "emerald",
    available: true,
    examplePrompts: ["Optimize the database queries", "Add error handling", "Improve the test coverage"],
    backgroundImage: "/images/parrots/amazing_bg.webp",
  },
};

/**
 * Get available parrot agents for current milestone
 * 获取当前里程碑可用的鹦鹉代理
 */
export function getAvailableParrots(): ParrotAgent[] {
  return Object.values(PARROT_AGENTS).filter((agent) => agent.available);
}

/**
 * Get parrot agent by type
 * 根据类型获取鹦鹉代理 - fallback 到 GENERAL
 */
export function getParrotAgent(type: ParrotAgentType): ParrotAgent {
  return PARROT_AGENTS[type] || PARROT_AGENTS[ParrotAgentType.GENERAL];
}

/**
 * Memo query result data
 * 笔记查询结果数据
 */
export interface MemoQueryResultData {
  memos: MemoSummary[];
  query: string;
  count: number;
}

/**
 * Memo summary
 * 笔记摘要
 */
export interface MemoSummary {
  uid: string;
  content: string;
  score: number;
}

/**
 * Schedule query result data
 * 日程查询结果数据
 */
export interface ScheduleQueryResultData {
  schedules: ScheduleSummary[];
  query: string;
  count: number;
  timeRangeDescription: string;
  queryType: string; // e.g., "upcoming", "range", "filter"
}

/**
 * Schedule summary
 * 日程摘要
 */
export interface ScheduleSummary {
  uid: string;
  title: string;
  startTimestamp: number;
  endTimestamp: number;
  allDay: boolean;
  location?: string;
  status: string;
}

/**
 * Block summary for a single chat round (Block)
 * Block 摘要 - 单个聊天轮次的统计
 *
 * This represents statistics for a SINGLE Block, not the entire conversation.
 * NOTE: Mode has been removed - use Block.mode as the single source of truth.
 */
export interface BlockSummary {
  sessionId?: string;
  totalDurationMs?: number;
  thinkingDurationMs?: number;
  toolDurationMs?: number;
  generationDurationMs?: number;
  totalInputTokens?: number;
  totalOutputTokens?: number;
  totalCacheWriteTokens?: number;
  totalCacheReadTokens?: number;
  toolCallCount?: number;
  toolsUsed?: string[];
  filesModified?: number;
  filePaths?: string[];
  totalCostUSD?: number;
  status?: string;
  errorMsg?: string;
}

/**
 * Event metadata for Geek/Evolution mode tool calls
 * 事件元数据 - 用于极客模式和进化模式的工具调用
 */
export interface EventMetadata {
  durationMs?: number;
  totalDurationMs?: number;
  toolName?: string;
  toolId?: string;
  status?: string;
  errorMsg?: string;
  inputTokens?: number;
  outputTokens?: number;
  cacheWriteTokens?: number;
  cacheReadTokens?: number;
  inputSummary?: string;
  outputSummary?: string;
  filePath?: string;
  lineCount?: number;
}

/**
 * Parrot chat callbacks
 * 鹦鹉聊天回调函数
 */
export interface ParrotChatCallbacks {
  onContent?: (content: string) => void;
  onMemoQueryResult?: (result: MemoQueryResultData) => void;
  onScheduleQueryResult?: (result: ScheduleQueryResultData) => void;
  onThinking?: (message: string) => void;
  onToolUse?: (toolName: string, meta?: EventMetadata) => void;
  onToolResult?: (result: string, meta?: EventMetadata) => void;
  onDangerBlock?: (event: DangerBlockEvent) => void;
  onPhaseChange?: (phase: ProcessingPhase, estimatedSeconds: number) => void;
  onProgress?: (percent: number, estimatedSeconds: number) => void;
  onDone?: () => void;
  onError?: (error: Error) => void;
}

/**
 * Danger category types for blocked operations
 * 危险操作类别类型
 */
export type DangerCategory =
  | "file_delete" // File deletion operations
  | "system" // System-level operations
  | "network" // Network/download operations
  | "database" // Database operations
  | "git" // Git operations
  | "permission"; // Permission changes

/**
 * Danger level severity
 * 危险级别严重程度
 */
export type DangerLevel = "critical" | "high" | "moderate";

/**
 * Danger block event - when a dangerous operation is blocked
 * 危险操作拦截事件
 */
export interface DangerBlockEvent {
  operation: string; // The dangerous operation that was detected
  reason: string; // Explanation of why it's dangerous
  patternMatched: string; // The pattern that matched
  level: DangerLevel; // Danger level with type constraint
  category: DangerCategory; // Category with type constraint
  bypassAllowed: boolean; // Whether bypass is allowed (admin only)
  suggestions?: string[]; // Safe alternatives
}

/**
 * Parrot chat parameters
 * 鹦鹉聊天参数
 * Note: history field removed - backend-driven context construction (context-engineering.md Phase 1)
 */
export interface ParrotChatParams {
  agentType: ParrotAgentType;
  message: string;
  conversationId?: number; // Backend will build history from this ID
  userTimezone?: string;
}

/**
 * Parrot event types
 * 鹦鹉事件类型
 */
export enum ParrotEventType {
  THINKING = "thinking",
  TOOL_USE = "tool_use",
  TOOL_RESULT = "tool_result",
  ANSWER = "answer",
  ERROR = "error",
  DANGER_BLOCK = "danger_block",
  MEMO_QUERY_RESULT = "memo_query_result",
  SCHEDULE_QUERY_RESULT = "schedule_query_result",
  SCHEDULE_UPDATED = "schedule_updated",
  // Progressive progress events (Issue #97)
  PHASE_CHANGE = "phase_change",
  PROGRESS = "progress",
  // Orchestrator events (Issue #169)
  PLAN = "plan",
  TASK_START = "task_start",
  TASK_END = "task_end",
  // Decompose progress events
  DECOMPOSE_START = "decompose_start",
  DECOMPOSE_END = "decompose_end",
}

/**
 * Processing phases for progressive progress feedback
 * 渐进式进度反馈的处理阶段
 */
export type ProcessingPhase = "analyzing" | "planning" | "retrieving" | "synthesizing";

/**
 * Phase change event data
 * 阶段变更事件数据
 */
export interface PhaseChangeEvent {
  phase: ProcessingPhase;
  phase_number: number; // 1-4
  total_phases: number; // Always 4
  estimated_seconds: number;
}

/**
 * Progress event data
 * 进度事件数据
 */
export interface ProgressEvent {
  percent: number; // 0-100
  estimated_time_seconds: number;
}

/**
 * Orchestrator task status
 * Orchestrator 任务状态
 */
export type TaskStatus = "pending" | "running" | "completed" | "failed";

/**
 * Orchestrator task definition
 * Orchestrator 任务定义
 */
export interface OrchestratorTask {
  agent: string;
  input: string;
  purpose: string;
  result?: string;
  error?: string;
  status: TaskStatus;
}

/**
 * Orchestrator plan event data
 * Orchestrator 规划事件数据
 */
export interface OrchestratorPlanEvent {
  analysis: string;
  tasks: OrchestratorTask[];
  parallel: boolean;
}

/**
 * Orchestrator task start event data
 * Orchestrator 任务开始事件数据
 */
export interface OrchestratorTaskStartEvent {
  index: number;
  agent: string;
  purpose: string;
  status: TaskStatus;
}

/**
 * Orchestrator task end event data
 * Orchestrator 任务结束事件数据
 */
export interface OrchestratorTaskEndEvent {
  index: number;
  agent: string;
  status: TaskStatus;
  error?: string;
}

/**
 * Parrot theme configuration
 * 鹦鹉主题配置
 *
 * 设计规范 (v6.2 - Unified Block Model):
 * - Memo:      Slate (石墨灰) - 笔记如石墨般沉淀
 * - Schedule:  Cyan (青色) - 时间如流水般清澈
 * - General:   Indigo (靛蓝) - 知识如海洋般深邃
 * - Ideation:  Violet (紫罗兰) - 创意如灵感般闪耀
 * - Geek:      Sky/Slate (石板蓝) - 代码如石板般精确
 * - Evolution: Emerald (翠绿) - 系统如植物般向上生长
 *
 * @see docs/specs/unified-block-model.md
 */
export const PARROT_THEMES = {
  // AUTO - 自动路由模式 - 默认使用主题
  AUTO: {
    bubbleUser: "bg-slate-700 dark:bg-slate-400 text-white",
    bubbleBg: "bg-white dark:bg-zinc-800",
    bubbleBorder: "border-slate-200 dark:border-slate-700",
    text: "text-slate-800 dark:text-slate-100",
    textSecondary: "text-slate-600 dark:text-slate-400",
    iconBg: "bg-slate-100 dark:bg-slate-700",
    iconText: "text-slate-700 dark:text-slate-300",
    inputBg: "bg-slate-50 dark:bg-slate-900",
    inputBorder: "border-slate-200 dark:border-slate-700",
    inputFocus: "focus:ring-slate-500 focus:border-slate-500",
    cardBg: "bg-white dark:bg-zinc-800",
    cardBorder: "border-slate-200 dark:border-slate-700",
    accent: "bg-slate-600",
    accentText: "text-white",
    headerBg: "bg-slate-50 dark:bg-slate-900/20",
    footerBg: "bg-slate-50 dark:bg-slate-900/20",
    badgeBg: "bg-slate-200 dark:bg-slate-700",
    badgeText: "text-slate-700 dark:text-slate-300",
    ringColor: "ring-slate-500",
  },
  // 灰灰 - 非洲灰鹦鹉 (African Grey Parrot) - 笔记搜索
  MEMO: {
    bubbleUser: "bg-slate-800 dark:bg-slate-300 text-white dark:text-slate-800",
    bubbleBg: "bg-white dark:bg-zinc-800",
    bubbleBorder: "border-slate-200 dark:border-slate-700",
    text: "text-slate-800 dark:text-slate-100",
    textSecondary: "text-slate-600 dark:text-slate-400",
    iconBg: "bg-slate-100 dark:bg-slate-700",
    iconText: "text-slate-700 dark:text-slate-300",
    inputBg: "bg-slate-50 dark:bg-slate-900",
    inputBorder: "border-slate-200 dark:border-slate-700",
    inputFocus: "focus:ring-slate-500 focus:border-slate-500",
    cardBg: "bg-white dark:bg-zinc-800",
    cardBorder: "border-slate-200 dark:border-slate-700",
    accent: "bg-red-500",
    accentText: "text-white",
    headerBg: "bg-slate-50 dark:bg-slate-900/20",
    footerBg: "bg-slate-200/80 dark:bg-slate-800/50",
    ringColor: "ring-slate-500",
  },
  // 时巧 - 鸡尾鹦鹉 (Cockatiel) - 日程管理
  SCHEDULE: {
    bubbleUser: "bg-cyan-600 dark:bg-cyan-500 text-white",
    bubbleBg: "bg-white dark:bg-zinc-800",
    bubbleBorder: "border-cyan-200 dark:border-cyan-700",
    text: "text-slate-800 dark:text-cyan-50",
    textSecondary: "text-slate-600 dark:text-cyan-200",
    iconBg: "bg-cyan-100 dark:bg-cyan-900",
    iconText: "text-cyan-700 dark:text-cyan-300",
    inputBg: "bg-cyan-50 dark:bg-cyan-950",
    inputBorder: "border-cyan-200 dark:border-cyan-700",
    inputFocus: "focus:ring-cyan-500 focus:border-cyan-500",
    cardBg: "bg-white dark:bg-zinc-800",
    cardBorder: "border-cyan-200 dark:border-cyan-700",
    accent: "bg-cyan-500",
    accentText: "text-white",
    headerBg: "bg-cyan-50 dark:bg-cyan-900/20",
    footerBg: "bg-cyan-200/80 dark:bg-cyan-800/50",
    ringColor: "ring-cyan-500",
  },
  // 折衷 - 折衷鹦鹉 (Eclectus Parrot) - 综合助手 (Legacy)
  GENERAL: {
    bubbleUser: "bg-indigo-600 dark:bg-indigo-500 text-white",
    bubbleBg: "bg-white dark:bg-zinc-800",
    bubbleBorder: "border-indigo-200 dark:border-indigo-700",
    text: "text-slate-800 dark:text-indigo-50",
    textSecondary: "text-slate-600 dark:text-indigo-200",
    iconBg: "bg-indigo-100 dark:bg-indigo-900",
    iconText: "text-indigo-700 dark:text-indigo-300",
    inputBg: "bg-indigo-50 dark:bg-indigo-950",
    inputBorder: "border-indigo-200 dark:border-indigo-700",
    inputFocus: "focus:ring-indigo-500 focus:border-indigo-500",
    cardBg: "bg-white dark:bg-zinc-800",
    cardBorder: "border-indigo-200 dark:border-indigo-700",
    accent: "bg-indigo-500",
    accentText: "text-white",
    headerBg: "bg-indigo-50 dark:bg-indigo-900/20",
    footerBg: "bg-indigo-200/80 dark:bg-indigo-800/50",
    ringColor: "ring-indigo-500",
  },
  // 灵光 - 创意生成专家 (紫罗兰色 - 创意灵感)
  IDEATION: {
    bubbleUser: "bg-violet-600 dark:bg-violet-500 text-white",
    bubbleBg: "bg-white dark:bg-zinc-800",
    bubbleBorder: "border-violet-200 dark:border-violet-700",
    text: "text-slate-800 dark:text-violet-50",
    textSecondary: "text-slate-600 dark:text-violet-200",
    iconBg: "bg-violet-100 dark:bg-violet-900",
    iconText: "text-violet-700 dark:text-violet-300",
    inputBg: "bg-violet-50 dark:bg-violet-950",
    inputBorder: "border-violet-200 dark:border-violet-700",
    inputFocus: "focus:ring-violet-500 focus:border-violet-500",
    cardBg: "bg-white dark:bg-zinc-800",
    cardBorder: "border-violet-200 dark:border-violet-700",
    accent: "bg-violet-500",
    accentText: "text-white",
    headerBg: "bg-violet-50 dark:bg-violet-900/20",
    footerBg: "bg-violet-200/80 dark:bg-violet-800/50",
    ringColor: "ring-violet-500",
  },
  // Normal Mode - 中性灰 (智慧沉稳，如墨砚般深沉)
  // Zinc 纯灰色系：中性、专业，与 GEEK(slate蓝灰) 和 EVOLUTION(emerald翠绿) 明显区分
  NORMAL: {
    bubbleUser: "bg-zinc-600 dark:bg-zinc-500 text-white",
    bubbleBg: "bg-zinc-50 dark:bg-zinc-800/60",
    bubbleBorder: "border-zinc-200 dark:border-zinc-600",
    text: "text-zinc-800 dark:text-zinc-100",
    textSecondary: "text-zinc-600 dark:text-zinc-400",
    iconBg: "bg-zinc-100 dark:bg-zinc-700",
    iconText: "text-zinc-700 dark:text-zinc-300",
    inputBg: "bg-zinc-50 dark:bg-zinc-900",
    inputBorder: "border-zinc-200 dark:border-zinc-700",
    inputFocus: "focus:ring-zinc-500 focus:border-zinc-500",
    cardBg: "bg-white dark:bg-zinc-800",
    cardBorder: "border-zinc-200 dark:border-zinc-700",
    accent: "bg-zinc-500",
    accentText: "text-white",
    headerBg: "bg-zinc-50 dark:bg-zinc-800/40",
    footerBg: "bg-zinc-100/80 dark:bg-zinc-800/50",
    ringColor: "ring-zinc-500",
  },
  // 极客 - Geek Mode (Claude Code CLI) - 石板蓝 (代码如石板般精确)
  GEEK: {
    bubbleUser: "bg-sky-600 dark:bg-slate-500 text-white",
    bubbleBg: "bg-sky-50 dark:bg-slate-900/20",
    bubbleBorder: "border-sky-200 dark:border-slate-700",
    text: "text-sky-800 dark:text-slate-100",
    textSecondary: "text-sky-600 dark:text-slate-400",
    iconBg: "bg-sky-100 dark:bg-slate-700",
    iconText: "text-sky-700 dark:text-slate-300",
    inputBg: "bg-sky-50 dark:bg-slate-900",
    inputBorder: "border-sky-200 dark:border-slate-700",
    inputFocus: "focus:ring-sky-500 focus:border-sky-500",
    cardBg: "bg-white dark:bg-zinc-800",
    cardBorder: "border-sky-200 dark:border-slate-700",
    accent: "bg-sky-500",
    accentText: "text-white",
    headerBg: "bg-sky-50 dark:bg-slate-900/20",
    footerBg: "bg-sky-200/80 dark:bg-slate-800/50",
    ringColor: "ring-sky-500",
  },
  // 进化 - Evolution Mode (系统自我进化) - 翠绿 (系统如植物般向上生长)
  EVOLUTION: {
    bubbleUser: "bg-emerald-600 dark:bg-emerald-500 text-white",
    bubbleBg: "bg-emerald-50 dark:bg-emerald-900/20",
    bubbleBorder: "border-emerald-200 dark:border-emerald-700",
    text: "text-emerald-800 dark:text-emerald-100",
    textSecondary: "text-emerald-600 dark:text-emerald-200",
    iconBg: "bg-emerald-100 dark:bg-emerald-900",
    iconText: "text-emerald-700 dark:text-emerald-300",
    inputBg: "bg-emerald-50 dark:bg-emerald-950",
    inputBorder: "border-emerald-200 dark:border-emerald-700",
    inputFocus: "focus:ring-emerald-500 focus:border-emerald-500",
    cardBg: "bg-white dark:bg-zinc-800",
    cardBorder: "border-emerald-200 dark:border-emerald-700",
    accent: "bg-emerald-500",
    accentText: "text-white",
    headerBg: "bg-emerald-50 dark:bg-emerald-900/20",
    footerBg: "bg-emerald-200/80 dark:bg-emerald-800/50",
    ringColor: "ring-emerald-500",
  },
} as const;

/**
 * Icons for each parrot
 * 每个鹦鹉的图标
 */
export const PARROT_ICONS: Record<string, string> = {
  MEMO: "/images/parrots/icons/memo_icon.webp",
  SCHEDULE: "/images/parrots/icons/schedule_icon.webp",
  GENERAL: "/assistant-avatar.webp",
  GEEK: "/assistant-avatar.webp",
  EVOLUTION: "/assistant-avatar.webp",
};
