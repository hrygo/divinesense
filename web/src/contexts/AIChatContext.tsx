import { createContext, ReactNode, useCallback, useContext, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { aiServiceClient } from "@/connect";
import {
  AI_STORAGE_KEYS,
  AIChatContextValue,
  AIChatState,
  AIMode,
  ChatItem,
  Conversation,
  ConversationMessage,
  ConversationViewMode,
  isContextSeparator,
  ReferencedMemo,
  SidebarTab,
} from "@/types/aichat";
import { CapabilityStatus, CapabilityType } from "@/types/capability";
import { ParrotAgentType } from "@/types/parrot";
import { AgentType, AIConversation, AIMessage } from "@/types/proto/api/v1/ai_service_pb";

const MESSAGE_CACHE_LIMIT = 100; // Maximum MSG messages to cache per conversation

// LocalStorage key for current AI Mode preference
const AI_MODE_STORAGE_KEY = "divinesense.ai_mode";

// LocalStorage key for Immersive Mode preference
const IMMERSIVE_MODE_STORAGE_KEY = "divinesense.immersive_mode";

const generateId = () => `chat_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;

// Helper function to get default conversation title based on parrot type.
// Note: This returns a fallback English title. The actual display titles are
// localized by the backend using title keys (e.g., "chat.default.title").
function getDefaultTitle(parrotId: ParrotAgentType): string {
  const titles: Record<string, string> = {
    [ParrotAgentType.MEMO]: "Chat with Memo",
    [ParrotAgentType.SCHEDULE]: "Chat with Schedule",
    [ParrotAgentType.AMAZING]: "Chat with Amazing",
  };
  return titles[parrotId] || "AI Chat";
}

/**
 * 根据首条用户消息生成语义化标题
 * 截取前 20 字符，清理特殊字符
 * 返回 null 表示无法生成有效标题
 */
function generateSemanticTitle(message: string): string | null {
  const cleaned = message
    .replace(/[#@\n\r]/g, " ")
    .replace(/\s+/g, " ")
    .trim();

  // 空串或纯符号输入，返回 null 表示保持原标题
  if (cleaned.length === 0) return null;

  if (cleaned.length <= 20) return cleaned;
  return cleaned.slice(0, 20) + "...";
}

const DEFAULT_STATE: AIChatState = {
  conversations: [],
  currentConversationId: null,
  viewMode: "hub",
  sidebarTab: "history",
  sidebarOpen: true,
  currentCapability: CapabilityType.AUTO,
  capabilityStatus: "idle",
  currentMode: "normal",
  geekMode: false,
  evolutionMode: false,
  immersiveMode: false,
};

const AIChatContext = createContext<AIChatContextValue | null>(null);

export function useAIChat(): AIChatContextValue {
  const context = useContext(AIChatContext);
  if (!context) {
    throw new Error("useAIChat must be used within AIChatProvider");
  }
  return context;
}

interface AIChatProviderProps {
  children: ReactNode;
  initialState?: Partial<AIChatState>;
}

// FIFO: Enforce message cache limit (only counts MSG, keeps SEP between MSG)
function enforceFIFOMessages(messages: ChatItem[]): ChatItem[] {
  const reversed = [...messages].reverse();
  const result: ChatItem[] = [];
  let msgCount = 0;

  for (const item of reversed) {
    if (isContextSeparator(item)) {
      // Keep SEPARATOR only if it's between MSG (i.e., we have MSG after it)
      if (msgCount > 0 && result.length > 0 && !isContextSeparator(result[0])) {
        result.unshift(item);
      }
    } else {
      // Regular MSG message
      if (msgCount < MESSAGE_CACHE_LIMIT) {
        result.unshift(item);
        msgCount++;
      }
      // Drop excess MSG messages
    }
  }

  return result;
}

export function AIChatProvider({ children, initialState }: AIChatProviderProps) {
  const { t } = useTranslation();
  const [state, setState] = useState<AIChatState>(() => {
    // Load mode preferences from localStorage
    let savedAIMode: AIMode = "normal";
    let savedImmersiveMode = false;
    if (typeof window !== "undefined") {
      try {
        const savedMode = localStorage.getItem(AI_MODE_STORAGE_KEY);
        savedAIMode = savedMode === "geek" || savedMode === "evolution" ? savedMode : "normal";
        savedImmersiveMode = localStorage.getItem(IMMERSIVE_MODE_STORAGE_KEY) === "true";
      } catch {
        console.warn("Failed to load preferences");
      }
    }

    return {
      ...DEFAULT_STATE,
      ...initialState,
      currentMode: savedAIMode,
      geekMode: savedAIMode === "geek",
      evolutionMode: savedAIMode === "evolution",
      immersiveMode: savedImmersiveMode,
    };
  });

  // Track mount state to prevent setState after unmount
  const isMountedRef = useRef(true);

  useEffect(() => {
    isMountedRef.current = true;
    return () => {
      isMountedRef.current = false;
    };
  }, []);

  // Helper to localize conversation title from backend key to display title
  const localizeTitle = useCallback(
    (titleKey: string): string => {
      // Handle non-key strings (e.g., user custom titles)
      if (!titleKey || !titleKey.startsWith("chat.")) {
        return titleKey;
      }

      try {
        // Handle "chat.new" - backend now returns just "chat.new"
        // Numbering is handled by frontend based on conversation position
        if (titleKey === "chat.new") {
          const translated = t("chat.new");
          return translated || "New Chat"; // Fallback if translation is empty
        }

        // Handle legacy "chat.new.N" format for backward compatibility
        const newChatMatch = titleKey.match(/^chat\.new\.(\d+)$/);
        if (newChatMatch) {
          const translated = t("chat.new");
          return translated || "New Chat";
        }

        // Handle other "chat.*.title" format (e.g., "chat.memo.title")
        if (titleKey.endsWith(".title")) {
          const translated = t(titleKey);
          // If translation exists and is not empty, use it
          if (translated && translated !== titleKey) {
            return translated;
          }
          // Otherwise use hardcoded fallbacks for known keys
          const fallbacks: Record<string, string> = {
            "chat.memo.title": "Memo Chat",
            "chat.schedule.title": "Schedule Chat",
            "chat.amazing.title": "Amazing Chat",
          };
          return fallbacks[titleKey] || titleKey;
        }
      } catch (err) {
        // Fallback to original key if parsing or translation fails
        console.warn("Failed to localize title key:", titleKey, err);
      }

      return titleKey;
    },
    [t],
  );

  // Helper to get message count
  const getMessageCount = useCallback((conversation: Conversation): number => {
    return conversation.messages.filter((item) => !isContextSeparator(item)).length;
  }, []);

  // Computed values
  const currentConversation = useMemo(() => {
    return state.conversations.find((c) => c.id === state.currentConversationId) || null;
  }, [state.conversations, state.currentConversationId]);

  const conversationSummaries = useMemo(() => {
    return state.conversations
      .map((c) => ({
        id: c.id,
        title: c.title,
        parrotId: c.parrotId,
        updatedAt: c.updatedAt,
        // Prefer backend messageCount, fallback to local calculation
        messageCount: c.messageCount ?? getMessageCount(c),
      }))
      .sort((a, b) => {
        // Sort by updated time (newest first)
        return b.updatedAt - a.updatedAt;
      });
  }, [state.conversations, getMessageCount]);

  // Helpers to convert from Protobuf to local types
  // Note: convertMessageFromPb must be defined before convertConversationFromPb
  const convertMessageFromPb = useCallback((m: AIMessage): ChatItem => {
    if (m.type === "SEPARATOR") {
      return {
        type: "context-separator",
        id: String(m.id),
        uid: m.uid, // Include UID for incremental sync
        timestamp: Number(m.createdTs) * 1000,
        synced: true,
      };
    }
    // Safe JSON parse with fallback
    let metadata = {};
    try {
      metadata = JSON.parse(m.metadata || "{}");
    } catch {
      console.warn("Failed to parse message metadata", m.metadata);
    }
    return {
      id: String(m.id),
      uid: m.uid, // Include UID for incremental sync
      role: m.role.toLowerCase() as "user" | "assistant",
      content: m.content,
      timestamp: Number(m.createdTs) * 1000,
      metadata,
    };
  }, []);

  // Helper: Convert protobuf AgentType enum to ParrotAgentType string
  const convertAgentTypeToParrotId = useCallback((agentType: AgentType): ParrotAgentType => {
    switch (agentType) {
      case AgentType.MEMO:
        return ParrotAgentType.MEMO;
      case AgentType.SCHEDULE:
        return ParrotAgentType.SCHEDULE;
      default:
        // AMAZING, DEFAULT, CREATIVE all map to AMAZING
        return ParrotAgentType.AMAZING;
    }
  }, []);

  // Helper: Convert ParrotAgentType string to protobuf AgentType enum
  const convertParrotIdToAgentType = useCallback((parrotId: ParrotAgentType): AgentType => {
    switch (parrotId) {
      case ParrotAgentType.MEMO:
        return AgentType.MEMO;
      case ParrotAgentType.SCHEDULE:
        return AgentType.SCHEDULE;
      default:
        return AgentType.AMAZING;
    }
  }, []);

  const convertConversationFromPb = useCallback(
    (pb: AIConversation): Conversation => {
      return {
        id: String(pb.id),
        title: localizeTitle(pb.title),
        parrotId: convertAgentTypeToParrotId(pb.parrotId),
        createdAt: Number(pb.createdTs) * 1000,
        updatedAt: Number(pb.updatedTs) * 1000,
        messages: (pb.messages ?? []).map((m) => convertMessageFromPb(m)),
        referencedMemos: [], // Backend managed for RAG, but state can store it if needed
        messageCount: pb.messageCount, // Use backend-provided message count
      };
    },
    [convertMessageFromPb, localizeTitle, convertAgentTypeToParrotId],
  );

  // Sync state with backend
  const refreshConversations = useCallback(async () => {
    try {
      const response = await aiServiceClient.listAIConversations({});
      setState((prev) => {
        const newConversations = response.conversations.map((c) => convertConversationFromPb(c));

        // Preserve local temporary conversations (not yet synced to backend)
        const localConversations = prev.conversations.filter((c) => c.id.startsWith("chat_"));

        // Merge strategies:
        // 1. Preserve existing messages if new conversation (summary) has none
        // 2. Be careful not to overwrite richer local state with summary state
        const mergedConversations = newConversations.map((newConv) => {
          const existing = prev.conversations.find((c) => c.id === newConv.id);
          if (existing && existing.messages.length > 0 && newConv.messages.length === 0) {
            return {
              ...newConv,
              messages: existing.messages,
              messageCount: Math.max(newConv.messageCount ?? 0, existing.messageCount ?? 0),
              messageCache: existing.messageCache,
            };
          }
          return newConv;
        });

        return { ...prev, conversations: [...localConversations, ...mergedConversations] };
      });
    } catch (e) {
      console.error("Failed to fetch conversations:", e);
    }
  }, [convertConversationFromPb]);

  // Handle migration from localStorage
  const migrateFromStorage = useCallback(
    async (localConversations: Conversation[]) => {
      console.log("Migrating AI conversations to cloud storage...");
      for (const local of localConversations) {
        try {
          // Use the shared conversion helper
          const parrotId = convertParrotIdToAgentType(local.parrotId);

          await aiServiceClient.createAIConversation({
            title: local.title,
            parrotId,
          });

          // We don't bulk migrate history to avoid database bloat,
          // but the user's list is now persisted.
          console.log(`Migrated conversation: ${local.title}`);
        } catch (e) {
          console.error(`Failed to migrate conversation: ${local.title}`, e);
        }
      }
      // Clear localStorage once migrated
      localStorage.removeItem(AI_STORAGE_KEYS.CONVERSATIONS);
    },
    [convertParrotIdToAgentType],
  );

  // Conversation actions
  const createConversation = useCallback(
    (parrotId: ParrotAgentType, title?: string): { id: string; completed: Promise<string> } => {
      const tempId = generateId(); // Temporary ID for UI
      const now = Date.now();
      const defaultTitle = title || getDefaultTitle(parrotId);

      // Immediately add temporary conversation to state (optimistic update)
      // This allows addMessage to work immediately without waiting for backend
      const tempConversation: Conversation = {
        id: tempId,
        title: defaultTitle,
        parrotId,
        createdAt: now,
        updatedAt: now,
        messages: [],
        referencedMemos: [],
        messageCount: 0,
      };

      setState((prev) => ({
        ...prev,
        conversations: [tempConversation, ...prev.conversations],
        currentConversationId: tempId,
        viewMode: "chat",
      }));

      // Asynchronously create on backend
      const agentType = convertParrotIdToAgentType(parrotId);

      const completionPromise = new Promise<string>((resolve, reject) => {
        aiServiceClient
          .createAIConversation({
            title: defaultTitle,
            parrotId: agentType,
          })
          .then((pb) => {
            // Replace temp conversation with real one from backend
            const newConv = convertConversationFromPb(pb);

            setState((prev) => {
              // Find the temp conversation and preserve its messages
              const tempConv = prev.conversations.find((c) => c.id === tempId);
              const preservedMessages = tempConv?.messages || [];

              // Remove temp conversation and add real one with preserved messages
              // Also remove any existing conversation with the same ID (prevent duplicates)
              const filteredConversations = prev.conversations.filter((c) => c.id !== tempId && c.id !== newConv.id);
              const realConversation: Conversation = {
                ...newConv,
                messages: preservedMessages,
                messageCount: preservedMessages.filter((m) => !isContextSeparator(m)).length,
                // Initialize messageCache to prevent syncMessages from overwriting local messages
                messageCache: {
                  lastMessageUid: "",
                  totalCount: preservedMessages.filter((m) => !isContextSeparator(m)).length,
                  hasMore: false,
                },
              };

              return {
                ...prev,
                conversations: [realConversation, ...filteredConversations],
                currentConversationId: newConv.id,
              };
            });

            // Refresh in background to ensure sync
            refreshConversations();
            resolve(newConv.id);
          })
          .catch((err) => {
            console.error("Failed to create conversation:", err);
            // Don't delete the conversation on error. Keep it in local state.
            // This prevents the UI from flashing back to the greeting page if the backend fails.
            // The conversation will remain as a "local-only" or "failed" state conceptually.
            // In the future, we can add a visual indicator for sync failure.

            // We still reject the promise so the caller knows it failed
            reject(err);
          });
      });

      return { id: tempId, completed: completionPromise };
    },
    [convertConversationFromPb, convertParrotIdToAgentType, refreshConversations],
  );

  const deleteConversation = useCallback(
    (id: string) => {
      const numericId = parseInt(id);
      if (!isNaN(numericId)) {
        aiServiceClient
          .deleteAIConversation({ id: numericId })
          .then(() => {
            refreshConversations();
          })
          .catch((err) => {
            console.error("Failed to delete conversation:", err);
          });
      }

      setState((prev) => {
        const filtered = prev.conversations.filter((c) => c.id !== id);
        const newCurrentId = prev.currentConversationId === id ? (filtered.length > 0 ? filtered[0].id : null) : prev.currentConversationId;

        return {
          ...prev,
          conversations: filtered,
          currentConversationId: newCurrentId,
          viewMode: filtered.length === 0 && prev.currentConversationId === id ? "hub" : prev.viewMode,
        };
      });
    },
    [refreshConversations],
  );

  const selectConversation = useCallback((id: string) => {
    setState((prev) => ({
      ...prev,
      currentConversationId: id,
      viewMode: "chat",
    }));
  }, []);

  const updateConversationTitle = useCallback((id: string, title: string) => {
    const numericId = parseInt(id);
    if (!isNaN(numericId)) {
      aiServiceClient.updateAIConversation({ id: numericId, title });
    }
    setState((prev) => ({
      ...prev,
      conversations: prev.conversations.map((c) => (c.id === id ? { ...c, title, updatedAt: Date.now() } : c)),
    }));
  }, []);

  // Message actions
  const addMessage = useCallback((conversationId: string, message: Omit<ConversationMessage, "id" | "timestamp">): string => {
    // For cloud persistence, message IDs and timestamps are generated by the backend
    // during the chat stream. Here we just update local state for optimism.
    // Note: SEPARATOR messages are added via addContextSeparator, not here.
    const newMessageId = generateId();
    const now = Date.now();

    setState((prev) => {
      const conversation = prev.conversations.find((c) => c.id === conversationId);
      if (!conversation) return prev;

      // Check if this is the first user message - auto-generate semantic title
      const isFirstUserMessage =
        message.role === "user" && conversation.messages.filter((m) => !isContextSeparator(m) && m.role === "user").length === 0;

      const shouldUpdateTitle =
        isFirstUserMessage && (conversation.title.startsWith("chat.") || conversation.title === getDefaultTitle(conversation.parrotId));

      // Generate semantic title (may return null for invalid input like pure symbols)
      const newTitle = shouldUpdateTitle && message.content ? generateSemanticTitle(message.content) : null;

      // If first user message and valid title generated, update on backend
      if (newTitle) {
        const numericId = parseInt(conversationId);
        if (!isNaN(numericId)) {
          aiServiceClient.updateAIConversation({ id: numericId, title: newTitle });
        }
      }

      return {
        ...prev,
        conversations: prev.conversations.map((c) => {
          if (c.id !== conversationId) return c;

          // Increment messageCount for real messages (SEPARATOR uses addContextSeparator)
          const newMessageCount = (c.messageCount ?? 0) + 1;

          return {
            ...c,
            // Only update title if valid new title generated, otherwise keep original
            title: newTitle || c.title,
            messages: [
              ...c.messages,
              {
                ...message,
                id: newMessageId,
                timestamp: now,
                metadata: {
                  ...message.metadata,
                  mode: state.currentMode,
                },
              },
            ],
            messageCount: newMessageCount, // Update message count for conversation list
            updatedAt: now,
          };
        }),
      };
    });
    return newMessageId;
  }, []);

  const updateMessage = useCallback((conversationId: string, messageId: string, updates: Partial<ConversationMessage>) => {
    setState((prev) => ({
      ...prev,
      conversations: prev.conversations.map((c) => {
        if (c.id !== conversationId) return c;

        return {
          ...c,
          messages: c.messages.map((m) => {
            if (isContextSeparator(m)) return m;
            if (m.id !== messageId) return m;
            // Deep merge metadata to preserve existing fields
            if (updates.metadata) {
              return {
                ...m,
                ...updates,
                metadata: { ...m.metadata, ...updates.metadata },
              };
            }
            return { ...m, ...updates };
          }),
          updatedAt: Date.now(),
        };
      }),
    }));
  }, []);

  const deleteMessage = useCallback((conversationId: string, messageId: string) => {
    // Current backend doesn't support individual message deletion via API yet
    // but we update state for immediate UI feedback
    setState((prev) => ({
      ...prev,
      conversations: prev.conversations.map((c) => {
        if (c.id !== conversationId) return c;

        return {
          ...c,
          messages: c.messages.filter((m) => !isContextSeparator(m) || ("id" in m && m.id !== messageId)),
          updatedAt: Date.now(),
        };
      }),
    }));
  }, []);

  const clearMessages = useCallback((conversationId: string) => {
    const numericId = parseInt(conversationId);

    // Optimistic update: clear messages in local state immediately
    setState((prev) => ({
      ...prev,
      conversations: prev.conversations.map((c) =>
        c.id === conversationId ? { ...c, messages: [], messageCount: 0, updatedAt: Date.now() } : c,
      ),
    }));

    // Call backend API to persist the deletion
    if (!isNaN(numericId)) {
      aiServiceClient.clearConversationMessages({ conversationId: numericId }).catch((err) => {
        console.error("Failed to clear messages on backend:", err);
      });
    }
  }, []);

  const addContextSeparator = useCallback(
    (conversationId: string, _trigger: "manual" | "auto" | "shortcut" = "manual") => {
      const numericId = parseInt(conversationId);
      if (isNaN(numericId)) return "";

      // Call backend API to persist the SEPARATOR message
      // Backend is idempotent: won't create duplicate if last message is already SEPARATOR
      aiServiceClient
        .addContextSeparator({ conversationId: numericId })
        .then(() => {
          // After successful creation, refresh the conversation to show the new separator
          aiServiceClient.getAIConversation({ id: numericId }).then((pb) => {
            const fullConversation = convertConversationFromPb(pb);
            setState((prev) => ({
              ...prev,
              conversations: prev.conversations.map((c) => (c.id === conversationId ? fullConversation : c)),
            }));
          });
        })
        .catch((err) => {
          console.error("Failed to add context separator:", err);
        });

      // Note: Removed optimistic update to prevent duplicate SEPARATOR accumulation
      // Backend refresh is fast enough and the idempotent check prevents duplicates
      return "";
    },
    [convertConversationFromPb],
  );

  // Helper function to merge messages into state (pure function, safe for async callbacks)
  function mergeMessagesIntoState(
    prevState: AIChatState,
    conversationId: string,
    response: {
      messages: AIMessage[];
      hasMore: boolean;
      totalCount: number;
      latestMessageUid: string;
    },
    convertMessageFromPb: (m: AIMessage) => ChatItem,
  ): AIChatState {
    const newMessages = response.messages.map((m) => convertMessageFromPb(m));

    return {
      ...prevState,
      conversations: prevState.conversations.map((c) => {
        if (c.id !== conversationId) return c;

        // For first load (no existing cache), use new messages directly
        if (!c.messageCache) {
          return {
            ...c,
            messages: enforceFIFOMessages(newMessages),
            messageCache: {
              lastMessageUid: response.latestMessageUid,
              totalCount: response.totalCount,
              hasMore: response.hasMore,
            },
          };
        }

        // For incremental sync, merge with existing messages
        const existingMessages = c.messages || [];
        // Create a copy for mutation
        const mergedMessages = [...existingMessages];
        const existingUids = new Set(existingMessages.filter((m) => !isContextSeparator(m)).map((m) => ("uid" in m ? m.uid : m.id)));

        for (const msg of newMessages) {
          if (isContextSeparator(msg)) {
            mergedMessages.push(msg);
            continue;
          }

          const uid = "uid" in msg ? msg.uid : msg.id;

          // Case 1: Message already synced (UID match)
          if (existingUids.has(uid)) {
            continue;
          }

          // Case 2: Check for Optimistic Update (Local temporary message)
          // Look for a local message (with "chat_" prefix ID) that matches role and content
          const matchIndex = mergedMessages.findIndex(
            (localMsg) =>
              !isContextSeparator(localMsg) &&
              localMsg.id.startsWith("chat_") && // Is temporary
              localMsg.role === msg.role &&
              localMsg.content === msg.content,
          );

          if (matchIndex !== -1) {
            // Found a match! Replace local message with synced real message
            // Preserve local metadata (like mode) that might be missing from backend
            const localMsg = mergedMessages[matchIndex] as ConversationMessage;
            mergedMessages[matchIndex] = {
              ...msg,
              metadata: {
                ...localMsg.metadata,
                ...msg.metadata,
              },
            };
            // Add new UID to set to prevent duplicate additions
            existingUids.add(uid);
          } else {
            // Case 3: Totally new message, append it
            mergedMessages.push(msg);
            existingUids.add(uid);
          }
        }

        // Enforce FIFO limit
        return {
          ...c,
          messages: enforceFIFOMessages(mergedMessages),
          messageCache: {
            lastMessageUid: response.latestMessageUid,
            totalCount: response.totalCount,
            hasMore: response.hasMore,
          },
        };
      }),
    };
  }

  // Message sync actions with incremental sync and FIFO cache
  const syncMessages = useCallback(
    async (conversationId: string) => {
      const numericId = parseInt(conversationId);
      if (isNaN(numericId)) return;

      // Use functional state update to get current state without creating dependency
      setState((prev) => {
        const conversation = prev.conversations.find((c) => c.id === conversationId);
        const lastMessageUid = conversation?.messageCache?.lastMessageUid || "";
        const limit = 100; // Always request 100 messages

        // Fire-and-forget async operation (we can't await in setState updater)
        // Check mounted state before setState to prevent updates after unmount
        aiServiceClient
          .listMessages({
            conversationId: numericId,
            lastMessageUid,
            limit,
          })
          .then((response) => {
            if (!isMountedRef.current) return; // Skip update if unmounted

            if (response.syncRequired) {
              // Backend requires full refresh, clear cache and retry
              setState((innerPrev) => ({
                ...innerPrev,
                conversations: innerPrev.conversations.map((c) => {
                  if (c.id !== conversationId) return c;
                  return { ...c, messages: [], messageCache: undefined };
                }),
              }));
              // Retry with empty lastMessageUid
              aiServiceClient
                .listMessages({
                  conversationId: numericId,
                  lastMessageUid: "",
                  limit,
                })
                .then((retryResponse) => {
                  if (!isMountedRef.current) return; // Skip update if unmounted
                  setState((innerPrev) => mergeMessagesIntoState(innerPrev, conversationId, retryResponse, convertMessageFromPb));
                });
            } else {
              setState((innerPrev) => mergeMessagesIntoState(innerPrev, conversationId, response, convertMessageFromPb));
            }
          })
          .catch((e: unknown) => {
            console.error("Failed to sync messages:", e);
          });

        return prev; // Return previous state unchanged (async update will follow)
      });
    },
    [convertMessageFromPb],
  ); // Only depends on convertMessageFromPb

  const loadMoreMessages = useCallback(
    async (conversationId: string) => {
      const numericId = parseInt(conversationId);
      if (isNaN(numericId)) return;

      // Use functional state update to avoid stale closure
      setState((prev) => {
        const conversation = prev.conversations.find((c) => c.id === conversationId);
        if (!conversation?.messageCache?.hasMore) return prev; // No more messages to load

        // Find the oldest message UID for pagination
        const oldestMessage = conversation.messages.find((m) => !isContextSeparator(m));
        if (!oldestMessage) return prev;

        const oldestUid = "uid" in oldestMessage ? oldestMessage.uid : oldestMessage.id;

        // Fire-and-forget async operation
        // Check mounted state before setState to prevent updates after unmount
        aiServiceClient
          .listMessages({
            conversationId: numericId,
            lastMessageUid: oldestUid as string, // Request messages before this UID
            limit: 100,
          })
          .then((response) => {
            if (!isMountedRef.current) return; // Skip update if unmounted

            // Prepend messages (older messages come first)
            const olderMessages = response.messages.map((m) => convertMessageFromPb(m));
            setState((innerPrev) => ({
              ...innerPrev,
              conversations: innerPrev.conversations.map((c) => {
                if (c.id !== conversationId) return c;

                return {
                  ...c,
                  messages: [...olderMessages, ...c.messages],
                  messageCache: c.messageCache
                    ? {
                        ...c.messageCache,
                        hasMore: response.hasMore,
                      }
                    : undefined,
                };
              }),
            }));
          })
          .catch((e: unknown) => {
            console.error("Failed to load more messages:", e);
          });

        return prev;
      });
    },
    [convertMessageFromPb],
  ); // Only depends on convertMessageFromPb

  // Referenced content actions
  const addReferencedMemos = useCallback((conversationId: string, memos: ReferencedMemo[]) => {
    setState((prev) => ({
      ...prev,
      conversations: prev.conversations.map((c) => {
        if (c.id !== conversationId) return c;

        const existingUids = new Set(c.referencedMemos.map((m) => m.uid));
        const newMemos = memos.filter((m) => !existingUids.has(m.uid));

        return {
          ...c,
          referencedMemos: [...c.referencedMemos, ...newMemos],
        };
      }),
    }));
  }, []);

  // UI actions
  const setViewMode = useCallback((mode: ConversationViewMode) => {
    setState((prev) => ({ ...prev, viewMode: mode }));
  }, []);

  const setSidebarTab = useCallback((tab: SidebarTab) => {
    setState((prev) => ({ ...prev, sidebarTab: tab }));
  }, []);

  const setSidebarOpen = useCallback((open: boolean) => {
    setState((prev) => ({ ...prev, sidebarOpen: open }));
  }, []);

  const toggleSidebar = useCallback(() => {
    setState((prev) => ({ ...prev, sidebarOpen: !prev.sidebarOpen }));
  }, []);

  // ============================================================
  // CAPABILITY ACTIONS (新增 - 能力管理)
  // ============================================================
  const setCurrentCapability = useCallback((capability: CapabilityType) => {
    setState((prev) => ({ ...prev, currentCapability: capability }));
  }, []);

  const setCapabilityStatus = useCallback((status: CapabilityStatus) => {
    setState((prev) => ({ ...prev, capabilityStatus: status }));
  }, []);

  // ============================================================
  // AI MODE ACTIONS (三态模式切换)
  // ============================================================
  const setMode = useCallback((mode: AIMode) => {
    setState((prev) => ({
      ...prev,
      currentMode: mode,
      geekMode: mode === "geek",
      evolutionMode: mode === "evolution",
    }));
    // Persist to localStorage
    if (typeof window !== "undefined") {
      try {
        localStorage.setItem(AI_MODE_STORAGE_KEY, mode);
      } catch (e) {
        console.error("Failed to save AI mode preference:", e);
      }
    }
  }, []);

  // ============================================================
  // GEEK MODE ACTIONS (极客模式) - 兼容旧代码
  // ============================================================
  const toggleGeekMode = useCallback(
    (enabled: boolean) => {
      setMode(enabled ? "geek" : "normal");
    },
    [setMode],
  );

  // ============================================================
  // EVOLUTION MODE ACTIONS (进化模式) - 兼容旧代码
  // ============================================================
  const toggleEvolutionMode = useCallback(
    (enabled: boolean) => {
      setMode(enabled ? "evolution" : "normal");
    },
    [setMode],
  );

  // ============================================================
  // IMMERSIVE MODE ACTIONS (沉浸模式)
  // ============================================================
  const toggleImmersiveMode = useCallback((enabled: boolean) => {
    setState((prev) => ({ ...prev, immersiveMode: enabled }));
    // Persist to localStorage
    if (typeof window !== "undefined") {
      try {
        localStorage.setItem(IMMERSIVE_MODE_STORAGE_KEY, String(enabled));
      } catch (e) {
        console.error("Failed to save immersive mode preference:", e);
      }
    }
  }, []);

  // Persistence
  const saveToStorage = useCallback(() => {
    try {
      // We still use localStorage for UI preferences, but not conversations
      localStorage.setItem(AI_STORAGE_KEYS.CURRENT_CONVERSATION, state.currentConversationId || "");
      localStorage.setItem(AI_STORAGE_KEYS.SIDEBAR_TAB, state.sidebarTab);
      // Geek mode is saved separately via toggleGeekMode to avoid circular dependencies
    } catch (e) {
      console.error("Failed to save AI chat state:", e);
    }
  }, [state.currentConversationId, state.sidebarTab]);

  const loadFromStorage = useCallback(async () => {
    try {
      const conversationsData = localStorage.getItem(AI_STORAGE_KEYS.CONVERSATIONS);
      const currentConversationData = localStorage.getItem(AI_STORAGE_KEYS.CURRENT_CONVERSATION);
      const sidebarTabData = localStorage.getItem(AI_STORAGE_KEYS.SIDEBAR_TAB);

      if (conversationsData) {
        const localConversations = JSON.parse(conversationsData);
        if (localConversations.length > 0) {
          await migrateFromStorage(localConversations);
        }
      }

      await refreshConversations();

      setState((prev) => ({
        ...prev,
        currentConversationId: currentConversationData || null,
        sidebarTab: sidebarTabData === "history" || sidebarTabData === "memos" ? (sidebarTabData as SidebarTab) : "history",
      }));
    } catch (e) {
      console.error("Failed to load AI chat state:", e);
      refreshConversations();
    }
  }, [migrateFromStorage, refreshConversations]);

  const clearStorage = useCallback(() => {
    try {
      localStorage.removeItem(AI_STORAGE_KEYS.CONVERSATIONS);
      localStorage.removeItem(AI_STORAGE_KEYS.CURRENT_CONVERSATION);
      localStorage.removeItem(AI_STORAGE_KEYS.SIDEBAR_TAB);
      setState({ ...DEFAULT_STATE });
    } catch (e) {
      console.error("Failed to clear AI chat state:", e);
    }
  }, []);

  // Auto-save to localStorage when state changes (debounced)
  useEffect(() => {
    const timer = setTimeout(() => {
      saveToStorage();
    }, 500); // 500ms debounce
    return () => clearTimeout(timer);
  }, [state, saveToStorage]);

  // Load from storage on mount (only once)
  useEffect(() => {
    loadFromStorage();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // Sync messages when conversation changes (only for valid numeric IDs)
  // Note: syncMessages uses functional state updates, so it's stable across renders
  useEffect(() => {
    if (state.currentConversationId) {
      const numericId = parseInt(state.currentConversationId);
      if (!isNaN(numericId)) {
        syncMessages(state.currentConversationId);
      }
    }
    // syncMessages is intentionally excluded from deps - it uses functional setState
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [state.currentConversationId]);

  const contextValue: AIChatContextValue = {
    state,
    currentConversation,
    conversations: state.conversations,
    conversationSummaries,
    createConversation,
    deleteConversation,
    selectConversation,
    updateConversationTitle,
    addMessage,
    updateMessage,
    deleteMessage,
    clearMessages,
    addContextSeparator,
    addReferencedMemos,
    setViewMode,
    setSidebarTab,
    setSidebarOpen,
    toggleSidebar,
    setCurrentCapability,
    setCapabilityStatus,
    setMode,
    toggleGeekMode,
    toggleEvolutionMode,
    toggleImmersiveMode,
    saveToStorage,
    loadFromStorage,
    clearStorage,
    syncMessages,
    loadMoreMessages,
  };

  return <AIChatContext.Provider value={contextValue}>{children}</AIChatContext.Provider>;
}
