import { useQueryClient } from "@tanstack/react-query";
import { createContext, ReactNode, useCallback, useContext, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { aiServiceClient } from "@/connect";
// Import blockKeys for consistent query cache management
import { blockKeys } from "@/hooks/useBlockQueries";
import {
  AI_STORAGE_KEYS,
  AIChatContextValue,
  AIChatState,
  AIMode,
  Conversation,
  ConversationViewMode,
  ReferencedMemo,
  SidebarTab,
} from "@/types/aichat";
import { CapabilityStatus, CapabilityType } from "@/types/capability";
import { ParrotAgentType } from "@/types/parrot";
import { AgentType, AIConversation } from "@/types/proto/api/v1/ai_service_pb";

// LocalStorage key for current AI Mode preference
const AI_MODE_STORAGE_KEY = "divinesense.ai_mode";

// LocalStorage key for Immersive Mode preference
const IMMERSIVE_MODE_STORAGE_KEY = "divinesense.immersive_mode";

// Timeout for conversation creation (ms)
const CONVERSATION_CREATE_TIMEOUT = 10000;

const generateId = () => `chat_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;

// Helper function to get default conversation title based on parrot type.
// Returns i18n key format to enable backend detection and auto-title generation.
// The frontend will translate these keys for display using t().
function getDefaultTitle(parrotId: ParrotAgentType): string {
  const titles: Record<string, string> = {
    [ParrotAgentType.MEMO]: "chat.default.memo",
    [ParrotAgentType.SCHEDULE]: "chat.default.schedule",
    [ParrotAgentType.AMAZING]: "chat.default.amazing",
    [ParrotAgentType.GEEK]: "chat.default.geek",
    [ParrotAgentType.EVOLUTION]: "chat.default.evolution",
  };
  return titles[parrotId] || "chat.default.auto";
}

// Checks if a title is a default i18n key (should trigger auto-generation)
export function isDefaultTitle(title: string): boolean {
  return title.startsWith("chat.default.") || title === "chat.new" || title.startsWith("chat.");
}

// Translates a title: if it's an i18n key, translates it; otherwise returns as-is.
export function translateTitle(title: string, t: (key: string) => string): string {
  if (isDefaultTitle(title)) {
    return t(title);
  }
  return title;
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
  blocksByConversation: {}, // Phase 4: Initialize empty blocks map
  pendingQueue: {
    messages: [],
    maxSize: 10,
  },
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

export function AIChatProvider({ children, initialState }: AIChatProviderProps) {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
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

        // Handle "chat.default.*" format (e.g., "chat.default.auto", "chat.default.memo")
        const defaultMatch = titleKey.match(/^chat\.default\.(.+)$/);
        if (defaultMatch) {
          const translated = t(titleKey);
          return translated || titleKey; // Fallback to key if translation is empty
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
        // Use backend messageCount directly (UI uses Block API now)
        messageCount: c.messageCount ?? 0,
      }))
      .sort((a, b) => {
        // Sort by updated time (newest first)
        return b.updatedAt - a.updatedAt;
      });
  }, [state.conversations]);

  // Phase 4: Current conversation blocks
  const currentBlocks = useMemo(() => {
    if (!state.currentConversationId) return [];
    return state.blocksByConversation[state.currentConversationId] || [];
  }, [state.currentConversationId, state.blocksByConversation]);

  // Helper: Convert protobuf AgentType enum to ParrotAgentType string
  const convertAgentTypeToParrotId = useCallback((agentType: AgentType): ParrotAgentType => {
    switch (agentType) {
      case AgentType.MEMO:
        return ParrotAgentType.MEMO;
      case AgentType.SCHEDULE:
        return ParrotAgentType.SCHEDULE;
      default:
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
      // UI now uses Block API directly for messages
      return {
        id: String(pb.id),
        title: localizeTitle(pb.title),
        parrotId: convertAgentTypeToParrotId(pb.parrotId),
        createdAt: Number(pb.createdTs) * 1000,
        updatedAt: Number(pb.updatedTs) * 1000,
        referencedMemos: [],
        messageCount: pb.blockCount,
      };
    },
    [localizeTitle, convertAgentTypeToParrotId],
  );

  // Sync state with backend
  const refreshConversations = useCallback(async () => {
    try {
      const response = await aiServiceClient.listAIConversations({});
      setState((prev) => {
        const newConversations = response.conversations.map((c) => convertConversationFromPb(c));

        // Preserve local temporary conversations (not yet synced to backend)
        const localConversations = prev.conversations.filter((c) => c.id.startsWith("chat_"));

        return { ...prev, conversations: [...localConversations, ...newConversations] };
      });
    } catch (e) {
      console.error("Failed to fetch conversations:", e);
    }
  }, [convertConversationFromPb]);

  // Handle migration from localStorage
  const migrateFromStorage = useCallback(
    async (localConversations: Conversation[]) => {
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
      const tempConversation: Conversation = {
        id: tempId,
        title: defaultTitle,
        parrotId,
        createdAt: now,
        updatedAt: now,
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
        // Create timeout promise
        const timeoutPromise = new Promise<never>((_, timeoutReject) => {
          setTimeout(() => {
            timeoutReject(new Error(`Conversation creation timeout after ${CONVERSATION_CREATE_TIMEOUT}ms`));
          }, CONVERSATION_CREATE_TIMEOUT);
        });

        // Race between actual API call and timeout
        Promise.race([
          aiServiceClient.createAIConversation({
            title: defaultTitle,
            parrotId: agentType,
          }),
          timeoutPromise,
        ])
          .then((pb) => {
            // Replace temp conversation with real one from backend
            const newConv = convertConversationFromPb(pb);

            setState((prev) => {
              // Remove temp conversation and add real one from backend
              // Also remove any existing conversation with the same ID (prevent duplicates)
              const filteredConversations = prev.conversations.filter((c) => c.id !== tempId && c.id !== newConv.id);

              return {
                ...prev,
                conversations: [newConv, ...filteredConversations],
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
    // Only update local state - API call is handled by the caller
    // This avoids duplicate API calls when updating from dialogs
    setState((prev) => ({
      ...prev,
      conversations: prev.conversations.map((c) => (c.id === id ? { ...c, title, updatedAt: Date.now() } : c)),
    }));
  }, []);

  const generateConversationTitle = useCallback(async (id: string) => {
    const numericId = parseInt(id);
    if (isNaN(numericId)) return null;

    try {
      const response = await aiServiceClient.generateConversationTitle({ id: numericId });

      setState((prev) => ({
        ...prev,
        conversations: prev.conversations.map((c) => {
          if (c.id === id) {
            return {
              ...c,
              title: response.title,
              updatedAt: Date.now(),
            };
          }
          return c;
        }),
      }));
      return response.title;
    } catch (error) {
      console.error("Failed to generate conversation title:", error);
      return null;
    }
  }, []);

  // scheduleTitleRefresh schedules a delayed refresh to fetch the auto-generated title.
  // The backend generates titles asynchronously in generateTitleAsync, which takes ~2-5 seconds.
  // This method should be called after Block 0 completes to fetch the generated title.
  const scheduleTitleRefresh = useCallback((conversationId: string) => {
    // Delay 3 seconds to allow backend async title generation to complete
    setTimeout(async () => {
      try {
        const response = await aiServiceClient.listAIConversations({});
        const updatedConv = response.conversations.find((c) => String(c.id) === conversationId);
        if (updatedConv) {
          setState((prev) => ({
            ...prev,
            conversations: prev.conversations.map((c) => {
              if (c.id === conversationId) {
                return {
                  ...c,
                  title: updatedConv.title,
                  updatedAt: Number(updatedConv.updatedTs) * 1000,
                };
              }
              return c;
            }),
          }));
        }
      } catch (error) {
        // Silently fail - title refresh is best-effort
        if (import.meta.env.DEV) {
          console.debug("[AI Chat] Title refresh failed (non-critical):", error);
        }
      }
    }, 3000);
  }, []);

  // Phase 4: Removed addMessage, updateMessage, deleteMessage - Block API handles this
  // Message actions
  const clearMessages = useCallback((conversationId: string) => {
    const numericId = parseInt(conversationId);

    // Optimistic update: clear message count in local state immediately
    setState((prev) => ({
      ...prev,
      conversations: prev.conversations.map((c) => (c.id === conversationId ? { ...c, messageCount: 0, updatedAt: Date.now() } : c)),
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

  // ============================================================
  // PHASE 4: BLOCK ACTIONS (Unified Block Model)
  // ============================================================

  /**
   * Load blocks for a conversation from the backend
   * Uses React Query for caching and optimistic updates
   * Also updates the messageCount in conversations list
   */
  const loadBlocks = useCallback(async (conversationId: string) => {
    const numericId = parseInt(conversationId);
    if (isNaN(numericId)) return;

    // We use the API client directly here to avoid circular dependency
    // The useBlocks hook can be used by components for reactive updates
    try {
      const response = await aiServiceClient.listBlocks({ conversationId: numericId });
      const blocks = response.blocks || [];

      setState((prev) => ({
        ...prev,
        blocksByConversation: {
          ...prev.blocksByConversation,
          [conversationId]: blocks,
        },
        // Update messageCount in conversations list
        conversations: prev.conversations.map((c) => (c.id === conversationId ? { ...c, messageCount: blocks.length } : c)),
      }));
    } catch (e) {
      console.error("Failed to load blocks:", e);
    }
  }, []);

  /**
   * Append user input to an existing block
   * Used during multi-turn conversations within the same block
   *
   * Uses invalidate + refetch pattern:
   * 1. Call API to persist the change
   * 2. Invalidate and refetch to get updated data
   */
  const appendUserInput = useCallback(
    async (blockId: number, content: string, conversationId: number) => {
      try {
        // Call API to persist the change
        await aiServiceClient.appendUserInput({
          id: BigInt(blockId),
          input: {
            content,
            timestamp: BigInt(Date.now()),
            metadata: JSON.stringify({}),
          },
        });

        // Invalidate specific conversation's blocks to trigger refetch
        await queryClient.invalidateQueries({
          queryKey: blockKeys.list(conversationId),
        });
      } catch (e) {
        console.error("Failed to append user input:", e);
        // On error, invalidate cache to trigger refetch of correct state
        await queryClient.invalidateQueries({ queryKey: blockKeys.lists() });
        throw e;
      }
    },
    [queryClient],
  );

  /**
   * Increment message count for a conversation (optimistic update)
   * Called when a new block is created/completed
   */
  const incrementMessageCount = useCallback((conversationId: string) => {
    setState((prev) => ({
      ...prev,
      conversations: prev.conversations.map((c) => (c.id === conversationId ? { ...c, messageCount: (c.messageCount || 0) + 1 } : c)),
    }));
  }, []);

  /**
   * Update block status locally (optimistic update)
   * Used during streaming to show real-time status changes
   */
  const updateBlockStatus = useCallback((blockId: number, _status: "pending" | "streaming" | "completed" | "error") => {
    setState((prev) => {
      const updated = { ...prev.blocksByConversation };

      for (const convId of Object.keys(updated)) {
        const blockIndex = updated[convId].findIndex((b) => Number(b.id) === blockId);
        if (blockIndex !== -1) {
          // Create a new array with the updated block
          updated[convId] = [...updated[convId]];
          // Convert status string to BlockStatus enum
          // Update status - we need to import BlockStatus enum
          // For now, just trigger a reload by removing the block from cache
          updated[convId].splice(blockIndex, 1);
        }
      }

      return { ...prev, blocksByConversation: updated };
    });
  }, []);

  // ============================================================
  // PHASE 5: PENDING QUEUE ACTIONS (Issue #121)
  // ============================================================
  const addToPendingQueue = useCallback((content: string) => {
    setState((prev) => {
      const queue = prev.pendingQueue;
      // Check if queue is full
      if (queue.messages.length >= queue.maxSize) {
        // Drop the oldest message (FIFO)
        const newMessages = [
          ...queue.messages.slice(1),
          {
            id: generateId(),
            content,
            timestamp: Date.now(),
          },
        ];
        return {
          ...prev,
          pendingQueue: {
            ...queue,
            messages: newMessages,
          },
        };
      }

      // Add to queue
      return {
        ...prev,
        pendingQueue: {
          ...queue,
          messages: [
            ...queue.messages,
            {
              id: generateId(),
              content,
              timestamp: Date.now(),
            },
          ],
        },
      };
    });
  }, []);

  const removeFromPendingQueue = useCallback((id: string) => {
    setState((prev) => ({
      ...prev,
      pendingQueue: {
        ...prev.pendingQueue,
        messages: prev.pendingQueue.messages.filter((m) => m.id !== id),
      },
    }));
  }, []);

  const clearPendingQueue = useCallback(() => {
    setState((prev) => ({
      ...prev,
      pendingQueue: {
        ...prev.pendingQueue,
        messages: [],
      },
    }));
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

  const contextValue: AIChatContextValue = {
    state,
    currentConversation,
    conversations: state.conversations,
    conversationSummaries,
    currentBlocks, // Phase 4: Current conversation blocks
    createConversation,
    deleteConversation,
    selectConversation,
    updateConversationTitle,
    generateConversationTitle,
    scheduleTitleRefresh,
    refreshConversations,
    // Phase 4: Removed addMessage, updateMessage, deleteMessage - Block API handles this
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
    // Phase 4: Block actions
    loadBlocks,
    appendUserInput,
    updateBlockStatus,
    incrementMessageCount,
    saveToStorage,
    loadFromStorage,
    clearStorage,
    addToPendingQueue,
    removeFromPendingQueue,
    clearPendingQueue,
  };

  return <AIChatContext.Provider value={contextValue}>{children}</AIChatContext.Provider>;
}
