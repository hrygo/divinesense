import copy from "copy-to-clipboard";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import toast from "react-hot-toast";
import { useTranslation } from "react-i18next";

import { ChatHeader } from "@/components/AIChat/ChatHeader";
import { ChatInput } from "@/components/AIChat/ChatInput";
import { ChatMessages } from "@/components/AIChat/ChatMessages";
import { PartnerGreeting } from "@/components/AIChat/PartnerGreeting";
import { PendingQueueBar } from "@/components/AIChat/PendingQueueBar";
// SessionBar 已移除 - PC 端 SessionStats 已整合到 ChatHeader
// import { SessionBar } from "@/components/AIChat/SessionBar";
import ConfirmDialog from "@/components/ConfirmDialog";
import { isDefaultTitle, useAIChat } from "@/contexts/AIChatContext";
import { useChat } from "@/hooks/useAIQueries";
import { useBlocks } from "@/hooks/useBlockQueries";
import { useCapabilityRouter } from "@/hooks/useCapabilityRouter";
import { usePendingQueue } from "@/hooks/usePendingQueue";
import { cn } from "@/lib/utils";
import type { AIMode } from "@/types/aichat";
import type { Block as AIBlock } from "@/types/block";
import { isActiveStatus } from "@/types/block";
import { CapabilityStatus, CapabilityType, capabilityToParrotAgent } from "@/types/capability";
import type { BlockSummary } from "@/types/parrot";
import { ParrotAgentType } from "@/types/parrot";
import type { SessionStats } from "@/types/proto/api/v1/ai_service_pb";
import { BlockStatus as BlockStatusEnum } from "@/types/proto/api/v1/ai_service_pb";

// ============================================================
// UNIFIED CHAT VIEW - 单一对话视图
// ============================================================
interface UnifiedChatViewProps {
  input: string;
  setInput: (value: string) => void;
  onSend: (messageContent?: string) => void;
  onStop: () => void;
  onNewChat: () => void;
  isTyping: boolean;
  isThinking: boolean;
  clearDialogOpen: boolean;
  setClearDialogOpen: (open: boolean) => void;
  onClearChat: () => void;
  onClearContext: () => void;
  blockSummary?: BlockSummary;
  blocks?: AIBlock[];
  currentCapability: CapabilityType;
  capabilityStatus: CapabilityStatus;
  recentMemoCount?: number;
  upcomingScheduleCount?: number;
  currentMode: AIMode;
  onModeChange: (mode: AIMode) => void;
  isAdmin?: boolean;
  conversationId?: number;
}

function UnifiedChatView({
  input,
  setInput,
  onSend,
  onStop,
  onNewChat,
  isTyping,
  isThinking,
  clearDialogOpen,
  setClearDialogOpen,
  onClearChat,
  onClearContext,
  blockSummary,
  blocks,
  currentCapability,
  capabilityStatus,
  recentMemoCount,
  upcomingScheduleCount,
  currentMode,
  onModeChange,
  conversationId,
}: UnifiedChatViewProps) {
  const { t } = useTranslation();

  const handleInputChange = (value: string) => {
    setInput(value);
  };

  const handleCopyMessage = (content: string) => {
    copy(content);
  };

  const handleDeleteMessage = () => {
    // TODO: Implement message deletion
  };

  // Handle quick reply - fills input with suggested text (#98)
  const handleQuickReply = useCallback(
    (text: string) => {
      // Translate i18n keys in the payload
      const translatedText = text.startsWith("ai.quick_replies.payload_") || text.startsWith("ai.") ? t(text) : text;
      setInput(translatedText);
      // Auto-focus input for immediate editing
      setTimeout(() => {
        const textarea = document.querySelector("textarea[placeholder*='Chat'], textarea[placeholder*='对话']") as HTMLTextAreaElement;
        if (textarea) {
          textarea.focus();
          // Move cursor to end of text
          textarea.setSelectionRange(translatedText.length, translatedText.length);
        }
      }, 50);
    },
    [setInput, t],
  );

  // Get mode-specific container classes
  const getModeContainerClass = (mode: AIMode) => {
    switch (mode) {
      case "geek":
        return "geek-matrix-grid";
      case "evolution":
        return "evo-flow-bg";
      default:
        return "";
    }
  };

  return (
    <div className={cn("w-full h-full flex flex-col relative bg-background", getModeContainerClass(currentMode))}>
      {/* Header - desktop only */}
      <ChatHeader
        className="hidden lg:flex"
        currentCapability={currentCapability}
        capabilityStatus={capabilityStatus}
        isThinking={isThinking}
        currentMode={currentMode}
        blocks={blocks}
      />

      {/* SessionBar - 已移除（PC 端 SessionStats 已整合到 ChatHeader） */}

      {/* Messages Area with Welcome */}
      <ChatMessages
        blocks={blocks ?? []}
        isTyping={isTyping}
        currentParrotId={ParrotAgentType.GENERAL}
        onCopyMessage={handleCopyMessage}
        onDeleteMessage={handleDeleteMessage}
        onQuickReply={handleQuickReply}
        onCancel={onStop}
        blockSummary={blockSummary}
        conversationId={conversationId}
      >
        {/* Welcome message - 统一入口，示例提问直接发送 */}
        {(blocks?.length ?? 0) === 0 && (
          <PartnerGreeting
            recentMemoCount={recentMemoCount}
            upcomingScheduleCount={upcomingScheduleCount}
            onSendMessage={onSend}
            currentMode={currentMode}
          />
        )}
      </ChatMessages>

      {/* Pending Queue Bar (Issue #121) */}
      <PendingQueueBar className="flex-shrink-0" />

      {/* Input Area */}
      <ChatInput
        value={input}
        onChange={handleInputChange}
        onSend={onSend}
        onStop={onStop}
        onNewChat={onNewChat}
        onClearContext={onClearContext}
        onClearChat={() => setClearDialogOpen(true)}
        onModeChange={onModeChange}
        isTyping={isTyping}
        currentMode={currentMode}
      />

      {/* Clear Chat Confirmation Dialog */}
      <ConfirmDialog
        open={clearDialogOpen}
        onOpenChange={setClearDialogOpen}
        title={t("ai.clear-chat")}
        confirmLabel={t("common.confirm")}
        description={t("ai.clear-chat-confirm")}
        cancelLabel={t("common.cancel")}
        onConfirm={onClearChat}
        confirmVariant="destructive"
      />
    </div>
  );
}

// ============================================================
// MAIN AI CHAT PAGE - 重构为单一对话入口
// ============================================================
const AIChat = () => {
  const chatHook = useChat();
  const aiChat = useAIChat();
  const capabilityRouter = useCapabilityRouter();

  // Local state
  const [input, setInput] = useState("");
  const [isTyping, setIsTyping] = useState(false);
  const [isThinking, setIsThinking] = useState(false);

  const [clearDialogOpen, setClearDialogOpen] = useState(false);
  const [blockSummary, setBlockSummary] = useState<BlockSummary | undefined>();

  const timeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const messageIdRef = useRef(0);
  const streamingContentRef = useRef<string>("");
  const isCreatingConversationRef = useRef(false);
  // Track conversations that have already triggered title generation (Block 0 completed)
  const titledConversationsRef = useRef<Set<string>>(new Set());
  // Phase 3: Removed refs that are no longer needed:
  // - lastAssistantMessageIdRef: Block ID tracking handled by useAIQueries
  // - thinkingStepsRef: Written to Block.eventStream
  // - toolCallsRef: Written to Block.eventStream
  // - currentRoundRef: No longer needed

  // Get current conversation and capability from context
  const {
    currentConversation,
    conversations,
    createConversation,
    selectConversation,
    // Phase 3: No longer need addMessage/updateMessage - Block API handles this
    addReferencedMemos,
    addContextSeparator,
    clearMessages,
    state,
    setCurrentCapability,
    setCapabilityStatus,
    setMode,
    // Phase 4: Block methods
    appendUserInput,
    incrementMessageCount,
    // Title management
    scheduleTitleRefresh,
    // Phase 5: Pending Queue
    addToPendingQueue,
  } = aiChat;

  const currentCapability = state.currentCapability || CapabilityType.AUTO;
  const capabilityStatus = state.capabilityStatus || "idle";
  const currentMode = state.currentMode || "normal";

  // ============================================================
  // Phase 4: Unified Block Model - Use blocks as primary data source
  // ============================================================
  const currentConversationIdNum = useMemo(() => {
    const id = currentConversation?.id;
    return id ? parseInt(id, 10) : 0;
  }, [currentConversation?.id]);

  // Phase 2: Use Block API as single source of truth (no more items fallback)
  // Always keep active=true to ensure real-time streaming updates are reflected in UI
  // The optimistic updates from useAIQueries.ts rely on React Query's cache changes being observed
  const blocksQuery = useBlocks(
    currentConversationIdNum,
    undefined, // No filters
    { isActive: true }, // Always active to receive streaming event updates
  );

  const blocks = blocksQuery.data?.blocks || [];

  const { t } = useTranslation();

  // Clear timeout on unmount
  useEffect(() => {
    return () => {
      if (timeoutRef.current) {
        clearTimeout(timeoutRef.current);
      }
    };
  }, []);

  // Clean up titledConversationsRef when switching conversations to prevent memory leak
  // Keep only the current conversation's ID in the set
  useEffect(() => {
    if (currentConversation?.id) {
      titledConversationsRef.current = new Set(Array.from(titledConversationsRef.current).filter((id) => id === currentConversation.id));
    }
  }, [currentConversation?.id]);

  const resetTypingState = useCallback(() => {
    if (timeoutRef.current) {
      clearTimeout(timeoutRef.current);
      timeoutRef.current = null;
    }
    setIsTyping(false);
  }, []);

  // Handle parrot chat with callbacks
  // Phase 3: Simplified - stream events written to Block.eventStream by useAIQueries.ts
  const handleParrotChat = useCallback(
    async (conversationId: string, parrotId: ParrotAgentType, userMessage: string, _conversationIdNum: number) => {
      setIsTyping(true);
      setIsThinking(true);
      setCapabilityStatus("thinking");
      const _messageId = ++messageIdRef.current;

      const explicitMessage = userMessage;

      // Prepare stream params
      const streamParams = {
        message: explicitMessage,
        agentType: parrotId,
        userTimezone: Intl.DateTimeFormat().resolvedOptions().timeZone,
        conversationId: _conversationIdNum,
        geekMode: currentMode === "geek",
        evolutionMode: currentMode === "evolution",
      };

      try {
        await chatHook.stream(streamParams, {
          // Phase 3: Stream events are automatically written to Block.eventStream
          // by useAIQueries.ts updateBlockEventStream function
          onThinking: (_msg) => {
            // UI state update only - content already in Block.eventStream
            setIsThinking(true);
          },
          onToolUse: (_toolName, _meta) => {
            // UI state update only - tool calls already in Block.eventStream
            setCapabilityStatus("processing");
          },
          onToolResult: (_result, _meta) => {
            // Tool results already in Block.eventStream
          },
          onBlockSummary: (summary) => {
            setBlockSummary(summary);
          },
          onMemoQueryResult: (result) => {
            if (_messageId === messageIdRef.current) {
              addReferencedMemos(conversationId, result.memos);
            }
          },
          onScheduleQueryResult: (_result) => {
            // Schedule query results - reserved for future use
          },
          onContent: (content) => {
            streamingContentRef.current += content;
            // Note: Block content is updated in useAIQueries.ts stream handler
            // via optimistic cache updates (updateBlockEventStream)
          },
          onDone: () => {
            setIsTyping(false);
            setIsThinking(false);
            setCapabilityStatus("idle");
            streamingContentRef.current = "";
            // No need to refetch - the streaming handler has already updated the block
            // with complete content via optimistic updates. Refetching would cause flicker.
            // Increment message count for the conversation
            const convId = String(_conversationIdNum);
            if (_conversationIdNum > 0) {
              incrementMessageCount(convId);
            }
            // Schedule title refresh ONLY when Block 0 (first message) completes
            // The backend generates titles asynchronously in generateTitleAsync (~2-5s)
            // This schedules a delayed refresh to fetch the auto-generated title
            if (_conversationIdNum > 0 && !titledConversationsRef.current.has(convId)) {
              const conv = conversations.find((c) => c.id === convId);
              // Only refresh if still has default title (user hasn't manually edited)
              if (conv && isDefaultTitle(conv.title)) {
                // Mark as titled to prevent duplicate scheduling
                titledConversationsRef.current.add(convId);
                // Schedule delayed refresh to fetch backend-generated title
                scheduleTitleRefresh(convId);
              }
            }
          },
          onError: (error) => {
            setIsTyping(false);
            setIsThinking(false);
            setCapabilityStatus("idle");
            streamingContentRef.current = "";
            console.error("[Parrot Error]", error);
            // Error state already written to Block by useAIQueries.ts
          },
        });
      } catch (error) {
        setIsTyping(false);
        setIsThinking(false);
        setCapabilityStatus("idle");
        console.error("[Parrot Chat Error]", error);
      }
    },
    [chatHook, addReferencedMemos, setCapabilityStatus, currentMode, scheduleTitleRefresh, incrementMessageCount, conversations],
  );

  const handleSend = useCallback(
    async (messageContent?: string) => {
      const userMessage = (messageContent || input).trim();
      if (!userMessage) return;

      // Phase 4 & 5: Check if there's a Block in streaming state
      // - Geek/Evolution Mode: Append user input to that Block (Interactive CLI style)
      // - Normal Mode: Queue message to PendingQueue (Issue #121)
      const streamingBlock = blocks?.find((b) => isActiveStatus(b.status));
      if (streamingBlock) {
        // Issue #121: In Normal Mode, queue messages instead of appending
        // This prevents interruption of the assistant's flow
        if (currentMode === "normal") {
          addToPendingQueue(userMessage);
          setInput("");
          return;
        }

        // Geek/Evolution Mode: Append input to running block
        const blockId = Number(streamingBlock.id);
        const convId = Number(streamingBlock.conversationId);
        // Validate conversationId matches current conversation to prevent unauthorized access
        if (blockId > 0 && convId > 0 && convId === currentConversationIdNum) {
          try {
            // Pass conversationId for optimistic update
            await appendUserInput(blockId, userMessage, convId);
            // Only clear input AFTER successful append
            setInput("");
            return;
          } catch (e) {
            console.error("[AI Chat] Failed to append user input to block:", e);
            // Don't clear input - let user retry
            toast.error(t("ai.error-send-failed") || "Failed to send message. Please try again.");
            return; // Stop execution, don't fall through to normal send
          }
        }
      }

      // 智能路由：根据输入内容自动识别能力
      const intentResult = capabilityRouter.route(userMessage, currentCapability);
      const targetCapability = intentResult.capability;

      // 如果识别出不同的能力，切换能力
      if (targetCapability !== currentCapability && targetCapability !== CapabilityType.AUTO) {
        setCurrentCapability(targetCapability);
      }

      // 确定使用哪个 Agent
      const targetParrotId = capabilityToParrotAgent(targetCapability);

      // Ensure we have a conversation
      let targetConversationId = currentConversation?.id;
      let creationPromise: Promise<string> | null = null;

      if (!targetConversationId) {
        // Prevent double creation due to race conditions/double clicks
        if (isCreatingConversationRef.current) return;

        // No active conversation - create one with AUTO agent (由后端路由决定)
        const existingConversation = conversations.find((c) => !c.parrotId || c.parrotId === ParrotAgentType.AUTO);
        if (existingConversation) {
          targetConversationId = existingConversation.id;
          selectConversation(existingConversation.id);
        } else {
          // Set lock before creating
          isCreatingConversationRef.current = true;
          const { id, completed } = createConversation(ParrotAgentType.AUTO);
          targetConversationId = id;
          creationPromise = completed.finally(() => {
            // Release lock when creation completes (success or failure)
            isCreatingConversationRef.current = false;
          });
        }
      }

      if (!targetConversationId) {
        console.error("[AI Chat] Failed to determine conversation");
        return;
      }

      // Phase 3: User message will be included in Block created by backend
      // No need for optimistic addMessage - Block API handles this

      // Special handling for cutting line (context separator)
      if (userMessage === "---") {
        setInput("");
        // Wait for real ID if we just created it
        const finalId = creationPromise ? await creationPromise : targetConversationId;
        const targetConversationIdNum = parseInt(finalId, 10);
        await handleParrotChat(finalId, targetParrotId, userMessage, targetConversationIdNum);
        return;
      }

      // Phase 3: No need to add empty assistant message - Block is created by backend
      // and automatically updated via useAIQueries.ts optimistic cache

      streamingContentRef.current = "";
      setInput("");

      // Wait for real ID if we just created the conversation
      // This is crucial to avoid "Chat with ID 0" which creates a duplicate session on backend
      let finalId = targetConversationId;
      if (creationPromise) {
        try {
          finalId = await creationPromise;
        } catch (e) {
          console.error("[AI Chat] Creation failed, using temp ID for optimistic UI", e);
          // Fallback to temp ID, request will likely fail on backend but UI remains stable
        }
      }

      const targetConversationIdNum = parseInt(finalId, 10);
      const conversationIdNum = isNaN(targetConversationIdNum) ? 0 : targetConversationIdNum;

      await handleParrotChat(finalId, targetParrotId, userMessage, conversationIdNum);
    },
    [
      input,
      currentConversation,
      currentCapability,
      capabilityRouter,
      setCurrentCapability,
      conversations,
      selectConversation,
      createConversation,
      // Phase 3: No addMessage dependency - Block created by backend
      handleParrotChat,
      resetTypingState,
      // Phase 4: Block append dependencies
      blocks,
      appendUserInput,
      // Phase 5: Pending queue
      addToPendingQueue,
      currentMode,
    ],
  );

  const handleStop = useCallback(() => {
    chatHook.stop();
    resetTypingState();
  }, [chatHook, resetTypingState]);

  const handleClearChat = useCallback(() => {
    if (currentConversation) {
      clearMessages(currentConversation.id);
    }
    setClearDialogOpen(false);
  }, [currentConversation, clearMessages]);

  const handleNewChat = useCallback(() => {
    createConversation(ParrotAgentType.AUTO);
  }, [createConversation]);

  const handleClearContext = useCallback(
    (trigger: "manual" | "auto" | "shortcut" = "manual") => {
      if (currentConversation) {
        addContextSeparator(currentConversation.id, trigger);
        toast.success(t("ai.context-cleared-toast"), {
          duration: 2000,
          icon: "✂️",
          className: "dark:bg-zinc-800 dark:border-zinc-700",
        });
      }
    },
    [currentConversation, addContextSeparator, t],
  );

  // Phase 5: Pending Queue Hook (Issue #121)
  // Automatically flush pending queue when active block finishes
  // Note: We pass handleSend directly. The hook handles the logic of when to call it.
  usePendingQueue(blocks, handleSend);

  // Handle custom event for sending messages (from suggested prompts)
  useEffect(() => {
    const handler = (e: CustomEvent<string>) => {
      setInput(e.detail);
      setTimeout(() => {
        setInput("");
        handleSend(e.detail);
      }, 100);
    };

    window.addEventListener("aichat-send-message", handler as EventListener);
    return () => {
      window.removeEventListener("aichat-send-message", handler as EventListener);
    };
  }, [handleSend]);

  // Keyboard shortcuts: ⌘K clear context, ⌘N new chat, ⌘L clear chat
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (!(e.metaKey || e.ctrlKey)) return;

      switch (e.key.toLowerCase()) {
        case "k":
          e.preventDefault();
          if (currentConversation) {
            handleClearContext("shortcut");
          }
          break;
        case "n":
          e.preventDefault();
          handleNewChat();
          break;
        case "l":
          e.preventDefault();
          if (currentConversation) {
            setClearDialogOpen(true);
          }
          break;
      }
    };

    window.addEventListener("keydown", handleKeyDown);
    return () => {
      window.removeEventListener("keydown", handleKeyDown);
    };
  }, [currentConversation, handleClearContext, handleNewChat]);

  // ============================================================
  // P2-#6: Restore blockSummary from persisted Block.sessionStats
  // When blocks are loaded (e.g., after refresh), restore blockSummary
  // from the latest block's sessionStats field
  // ============================================================

  /**
   * Safely convert bigint to number with precision loss warning
   */
  const safeBigIntToNumber = useCallback((value: bigint | undefined | null, fieldName: string): number | undefined => {
    if (value === undefined || value === null) return undefined;
    const num = Number(value);
    // Check if value exceeds safe integer range (2^53 - 1) or is NaN
    if (value > BigInt(Number.MAX_SAFE_INTEGER) || isNaN(num)) {
      console.warn(`[BlockSummary] ${fieldName} (${value}) exceeds MAX_SAFE_INTEGER or is NaN, precision may be lost`);
      return undefined;
    }
    return num;
  }, []);

  useEffect(() => {
    // Skip if we already have blockSummary from streaming
    if (blockSummary) return;

    // Skip if no blocks available
    if (!blocks || blocks.length === 0) return;

    // Find the latest completed block with sessionStats
    const latestBlock = blocks
      .filter((b) => b.status === BlockStatusEnum.COMPLETED || b.status === BlockStatusEnum.STREAMING)
      .sort((a, b) => Number(b.roundNumber) - Number(a.roundNumber))[0];

    if (!latestBlock?.sessionStats) return;

    const sessionStats: SessionStats = latestBlock.sessionStats;

    // Convert SessionStats to BlockSummary format (with safe bigint conversion)
    const restoredSummary: BlockSummary = {
      sessionId: sessionStats.sessionId || undefined,
      totalDurationMs: safeBigIntToNumber(sessionStats.totalDurationMs, "totalDurationMs"),
      thinkingDurationMs: safeBigIntToNumber(sessionStats.thinkingDurationMs, "thinkingDurationMs"),
      toolDurationMs: safeBigIntToNumber(sessionStats.toolDurationMs, "toolDurationMs"),
      generationDurationMs: safeBigIntToNumber(sessionStats.generationDurationMs, "generationDurationMs"),
      totalInputTokens: sessionStats.inputTokens || undefined,
      totalOutputTokens: sessionStats.outputTokens || undefined,
      totalCacheWriteTokens: sessionStats.cacheWriteTokens || undefined,
      totalCacheReadTokens: sessionStats.cacheReadTokens || undefined,
      toolCallCount: sessionStats.toolCallCount || undefined,
      toolsUsed: sessionStats.toolsUsed?.length ? sessionStats.toolsUsed : undefined,
      filesModified: sessionStats.filesModified || undefined,
      filePaths: sessionStats.filePaths?.length ? sessionStats.filePaths : undefined,
      totalCostUSD: sessionStats.totalCostUsd || undefined,
      status: sessionStats.isError ? "error" : sessionStats.updatedAt ? "success" : undefined,
      errorMsg: sessionStats.errorMessage || undefined,
    };

    setBlockSummary(restoredSummary);
  }, [blocks, blockSummary, setBlockSummary, safeBigIntToNumber]);

  // ============================================================
  // RENDER
  // ============================================================
  return (
    <UnifiedChatView
      input={input}
      setInput={setInput}
      onSend={handleSend}
      onStop={handleStop}
      onNewChat={handleNewChat}
      isTyping={isTyping}
      isThinking={isThinking}
      clearDialogOpen={clearDialogOpen}
      setClearDialogOpen={setClearDialogOpen}
      onClearChat={handleClearChat}
      onClearContext={handleClearContext}
      blockSummary={blockSummary}
      blocks={blocks}
      currentCapability={currentCapability}
      capabilityStatus={capabilityStatus}
      currentMode={currentMode}
      onModeChange={setMode}
      isAdmin={true}
      conversationId={currentConversationIdNum}
    />
  );
};

export default AIChat;
