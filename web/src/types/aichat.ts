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
    thinking?: string;
  };
}

/**
 * Context separator type for clearing conversation context
 *
 * Design Notes:
 * - `id`: Unique identifier for sync with backend (future)
 * - `synced`: Whether this separator is synced to server (future)
 * - `trigger`: How the context was cleared (for analytics)
 */
export interface ContextSeparator {
  type: "context-separator";
  id?: string; // Future: sync ID for conversation storage
  timestamp: number;
  synced?: boolean; // Future: sync status
  trigger?: "manual" | "auto" | "shortcut"; // How context was cleared
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
 * Chat item - union of message and separator
 */
export type ChatItem = ConversationMessage | ContextSeparator;

/**
 * Conversation state type
 */
export type ConversationViewMode = "hub" | "chat";

/**
 * Sidebar tab type
 */
export type SidebarTab = "history" | "memos";

/**
 * Message cache state for incremental sync
 */
export interface MessageCache {
  lastMessageUid: string; // Latest message UID from backend
  totalCount: number; // Total MSG count from backend
  hasMore: boolean; // Whether more messages exist before the first cached message
}

/**
 * Single conversation
 */
export interface Conversation {
  id: string;
  title: string;
  parrotId: ParrotAgentType;
  createdAt: number;
  updatedAt: number;
  messages: ChatItem[];
  referencedMemos: ReferencedMemo[];
  messageCount?: number; // Optional: backend-provided message count (excludes SEPARATOR)
  messageCache?: MessageCache; // Local message cache state for incremental sync
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
  // 极客模式 (Geek Mode for Claude Code CLI)
  geekMode: boolean;
  // 进化模式 (Evolution Mode for self-evolution)
  evolutionMode: boolean;
  // 沉浸模式 (沉浸模式 - 全屏沉浸体验)
  immersiveMode: boolean;
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

  // Conversation actions
  createConversation: (parrotId: ParrotAgentType, title?: string) => { id: string; completed: Promise<string> };
  deleteConversation: (id: string) => void;
  selectConversation: (id: string) => void;
  updateConversationTitle: (id: string, title: string) => void;

  // Message actions
  addMessage: (conversationId: string, message: Omit<ConversationMessage, "id" | "timestamp">) => string;
  updateMessage: (conversationId: string, messageId: string, updates: Partial<ConversationMessage>) => void;
  deleteMessage: (conversationId: string, messageId: string) => void;
  clearMessages: (conversationId: string) => void;
  addContextSeparator: (conversationId: string, trigger?: "manual" | "auto" | "shortcut") => string;
  syncMessages: (conversationId: string) => Promise<void>; // Incremental message sync with FIFO cache
  loadMoreMessages: (conversationId: string) => Promise<void>; // Load older messages (paginate back)

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

  // Geek Mode action (新增 - 极客模式)
  toggleGeekMode: (enabled: boolean) => void;

  // Evolution Mode action (进化模式 - 自我进化)
  toggleEvolutionMode: (enabled: boolean) => void;

  // Immersive Mode action (沉浸模式 - 全屏沉浸体验)
  toggleImmersiveMode: (enabled: boolean) => void;

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
