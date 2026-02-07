// Phase 4: Import Block type for currentBlocks
import type { Block } from "./block";
import { CapabilityStatus, CapabilityType } from "./capability";
import { ParrotAgentType } from "./parrot";

// Re-export capability types for convenience
export type { CapabilityType, CapabilityStatus };

/**
 * Message role in conversation
 */
export type MessageRole = "user" | "assistant" | "system";

/**
 * Single message in a conversation
 */
export interface ConversationMessage {
  id: string;
  uid?: string; // Backend UID for incremental sync
  role: MessageRole;
  content: string;
  timestamp: number;
  error?: boolean;
  metadata?: {
    referencedMemos?: string[];
    referencedSchedules?: string[];
    toolName?: string;
    toolCalls?: Array<{
      name: string;
      toolId?: string;
      inputSummary?: string;
      outputSummary?: string;
      filePath?: string;
      duration?: number;
      isError?: boolean;
      round?: number; // 第几轮思考（0-based）
    }>; // List of tools called by the agent
    toolResults?: Array<{
      name: string;
      toolId?: string;
      inputSummary?: string;
      outputSummary?: string;
      duration?: number;
      isError?: boolean;
      round?: number; // 第几轮思考（0-based）
    }>; // List of tool execution results
    // 多轮思考支持：thinkingSteps 数组或单一 thinking 字符串（向后兼容）
    thinkingSteps?: Array<{
      content: string;
      timestamp: number;
      round: number; // 第几轮（0-based）
    }>;
    thinking?: string; // 保留单一字符串向后兼容
    mode?: AIMode; // 消息生成时的 AI 模式
  };
}

/**
 * Referenced memo in conversation
 */
export interface ReferencedMemo {
  uid: string;
  content: string;
  score: number;
  timestamp?: number;
}

/**
 * Referenced schedule in conversation
 */
export interface ReferencedSchedule {
  uid: string;
  title: string;
  startTimestamp: number;
  endTimestamp: number;
  allDay: boolean;
  location?: string;
  status: string;
}

/**
 * Conversation state type
 */
export type ConversationViewMode = "hub" | "chat";

/**
 * AI Mode type - 三态循环模式
 * - normal: 普通模式 - AI 智能助理
 * - geek: 极客模式 - Claude Code CLI 代码执行
 * - evolution: 进化模式 - 系统自我进化
 */
export type AIMode = "normal" | "geek" | "evolution";

/**
 * Sidebar tab type
 */
export type SidebarTab = "history" | "memos";

/**
 * Single conversation
 */
export interface Conversation {
  id: string;
  title: string;
  parrotId: ParrotAgentType;
  createdAt: number;
  updatedAt: number;
  referencedMemos: ReferencedMemo[];
  messageCount?: number; // Optional: backend-provided message count
}

/**
 * Conversation summary for sidebar display
 */
export interface ConversationSummary {
  id: string;
  title: string;
  parrotId: ParrotAgentType;
  updatedAt: number;
  messageCount: number;
}

/**
 * AI Chat state
 */
export interface AIChatState {
  conversations: Conversation[];
  currentConversationId: string | null;
  viewMode: ConversationViewMode;
  sidebarTab: SidebarTab;
  sidebarOpen: boolean;
  // 能力状态 (新增 - 支持"个人专属助手"模式)
  currentCapability?: CapabilityType;
  capabilityStatus?: CapabilityStatus;
  // AI 模式 - 三态循环 (普通/极客/进化)
  currentMode: AIMode;
  // 兼容旧代码的 getter (computed from currentMode)
  geekMode: boolean;
  evolutionMode: boolean;
  // 沉浸模式 (沉浸模式 - 全屏沉浸体验)
  immersiveMode: boolean;
  // Phase 4: Block data (Unified Block Model support)
  // Maps conversationId to its blocks array
  blocksByConversation: Record<string, Block[]>;
}

/**
 * AI Chat context value
 */
export interface AIChatContextValue {
  // State
  state: AIChatState;

  // Computed values
  currentConversation: Conversation | null;
  conversations: Conversation[];
  conversationSummaries: ConversationSummary[];
  // Phase 4: Current conversation blocks
  currentBlocks: Block[];

  // Conversation actions
  createConversation: (parrotId: ParrotAgentType, title?: string) => { id: string; completed: Promise<string> };
  deleteConversation: (id: string) => void;
  selectConversation: (id: string) => void;
  updateConversationTitle: (id: string, title: string) => void;

  // Message actions
  // Phase 4: Removed addMessage, updateMessage, deleteMessage, syncMessages, loadMoreMessages - Block API handles this
  clearMessages: (conversationId: string) => void;
  addContextSeparator: (conversationId: string, trigger?: "manual" | "auto" | "shortcut") => string;

  // Referenced content actions
  addReferencedMemos: (conversationId: string, memos: ReferencedMemo[]) => void;

  // UI actions
  setViewMode: (mode: ConversationViewMode) => void;
  setSidebarTab: (tab: SidebarTab) => void;
  setSidebarOpen: (open: boolean) => void;
  toggleSidebar: () => void;

  // Capability actions (新增 - 能力管理)
  setCurrentCapability: (capability: CapabilityType) => void;
  setCapabilityStatus: (status: CapabilityStatus) => void;

  // AI Mode action (三态模式切换)
  setMode: (mode: AIMode) => void;

  // Geek Mode action (新增 - 极客模式) - 兼容旧代码
  toggleGeekMode: (enabled: boolean) => void;

  // Evolution Mode action (进化模式 - 自我进化) - 兼容旧代码
  toggleEvolutionMode: (enabled: boolean) => void;

  // Immersive Mode action (沉浸模式 - 全屏沉浸体验)
  toggleImmersiveMode: (enabled: boolean) => void;

  // Phase 4: Block actions (Unified Block Model support)
  loadBlocks: (conversationId: string) => Promise<void>;
  appendUserInput: (blockId: number, content: string, conversationId: number) => Promise<void>;
  updateBlockStatus: (blockId: number, status: "pending" | "streaming" | "completed" | "error") => void;

  // Persistence
  saveToStorage: () => void;
  loadFromStorage: () => void;
  clearStorage: () => void;
}

/**
 * Parrot theme configuration
 * 鹦鹉主题配置 - AI Native 配色系统
 */
export interface ParrotTheme {
  bgLight: string;
  bgDark: string;
  bubbleUser: string;
  bubbleBg: string;
  bubbleBorder: string;
  text: string;
  textSecondary: string;
  iconBg: string;
  iconText: string;
  inputBg: string;
  inputBorder: string;
  inputFocus: string;
  cardBg: string;
  cardBorder: string;
  accent: string;
  accentText: string;
}

/**
 * Local storage keys
 */
export const AI_STORAGE_KEYS = {
  CONVERSATIONS: "aichat_conversations",
  CURRENT_CONVERSATION: "aichat_current_conversation",
  SIDEBAR_TAB: "aichat_sidebar_tab",
} as const;

/**
 * CC Runner Stream Event types (async mode)
 * CC Runner 流式事件类型（异步模式）
 */
export type CcEventType = "thinking" | "tool_use" | "tool_result" | "answer" | "error";

/**
 * CC Runner Stream Event metadata
 * CC Runner 流式事件元数据
 */
export interface CcEventMeta {
  tool_name?: string;
  tool_id?: string;
  is_error?: boolean;
  file_path?: string;
  session_id?: string;
  exit_code?: number;
  duration_ms?: number;
  input?: Record<string, unknown>;
}

/**
 * CC Runner Stream Event
 * CC Runner 流式事件
 */
export interface CcStreamEvent {
  type: CcEventType;
  content: string;
  meta?: CcEventMeta;
  timestamp: number;
}
