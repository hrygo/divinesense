import copy from "copy-to-clipboard";
import { X } from "lucide-react";
import { useCallback, useDeferredValue, useEffect, useMemo, useRef, useState } from "react";
import toast from "react-hot-toast";
import { useTranslation } from "react-i18next";
import { AmazingInsightCard } from "@/components/AIChat/AmazingInsightCard";
import { ChatHeader } from "@/components/AIChat/ChatHeader";
import { ChatInput } from "@/components/AIChat/ChatInput";
import { ChatMessages } from "@/components/AIChat/ChatMessages";
import { ParrotHub } from "@/components/AIChat/ParrotHub";
import { PartnerGreeting } from "@/components/AIChat/PartnerGreeting";
import ConfirmDialog from "@/components/ConfirmDialog";
import { useAIChat } from "@/contexts/AIChatContext";
import { useChat } from "@/hooks/useAIQueries";
import { useAITools } from "@/hooks/useAITools";
import { useCapabilityRouter } from "@/hooks/useCapabilityRouter";
import useMediaQuery from "@/hooks/useMediaQuery";
import { cn } from "@/lib/utils";
import type { AIMode, ChatItem } from "@/types/aichat";
import { CapabilityStatus, CapabilityType, capabilityToParrotAgent } from "@/types/capability";
import type { MemoQueryResultData, ScheduleQueryResultData, SessionSummary } from "@/types/parrot";
import { ParrotAgentType } from "@/types/parrot";

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
  memoQueryResults: MemoQueryResultData[];
  scheduleQueryResults: ScheduleQueryResultData[];
  sessionSummary?: SessionSummary;
  items: ChatItem[];
  currentCapability: CapabilityType;
  capabilityStatus: CapabilityStatus;
  recentMemoCount?: number;
  upcomingScheduleCount?: number;
  uiTools: ReturnType<typeof useAITools>;
  currentMode: AIMode;
  onModeChange: (mode: AIMode) => void;
  immersiveMode: boolean;
  onImmersiveModeToggle: (enabled: boolean) => void;
  isAdmin?: boolean;
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
  memoQueryResults,
  scheduleQueryResults,
  sessionSummary,
  items,
  currentCapability,
  capabilityStatus,
  recentMemoCount,
  upcomingScheduleCount,
  uiTools,
  currentMode,
  onModeChange,
  immersiveMode,
  onImmersiveModeToggle,
  isAdmin = true,
}: UnifiedChatViewProps) {
  const { t } = useTranslation();
  const md = useMediaQuery("md");

  // P1-5: Concurrent rendering optimizations
  // Defer non-critical UI updates (query results) to improve input responsiveness
  const deferredMemoResults = useDeferredValue(memoQueryResults);
  const deferredScheduleResults = useDeferredValue(scheduleQueryResults);

  const handleInputChange = (value: string) => {
    setInput(value);
  };

  const handleCopyMessage = (content: string) => {
    copy(content);
  };

  const handleDeleteMessage = () => {
    // TODO: Implement message deletion
  };

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
      {/* Desktop Header */}
      {md && (
        <ChatHeader
          currentCapability={currentCapability}
          capabilityStatus={capabilityStatus}
          isThinking={isThinking}
          currentMode={currentMode}
          onModeChange={onModeChange}
          immersiveMode={immersiveMode}
          onImmersiveModeToggle={onImmersiveModeToggle}
          isAdmin={isAdmin}
        />
      )}

      {/* Messages Area with Welcome */}
      <ChatMessages
        items={items}
        isTyping={isTyping}
        currentParrotId={ParrotAgentType.AMAZING}
        onCopyMessage={handleCopyMessage}
        onDeleteMessage={handleDeleteMessage}
        amazingInsightCard={
          currentCapability === CapabilityType.AMAZING && (deferredMemoResults.length > 0 || deferredScheduleResults.length > 0) ? (
            <AmazingInsightCard memos={deferredMemoResults[0]?.memos ?? []} schedules={deferredScheduleResults[0]?.schedules ?? []} />
          ) : undefined
        }
        uiTools={uiTools.tools}
        onUIAction={uiTools.handleAction}
        onUIDismiss={uiTools.dismissTool}
        sessionSummary={sessionSummary}
      >
        {/* Welcome message - 统一入口，示例提问直接发送 */}
        {items.length === 0 && (
          <PartnerGreeting
            recentMemoCount={recentMemoCount}
            upcomingScheduleCount={upcomingScheduleCount}
            onSendMessage={onSend}
            currentMode={currentMode}
          />
        )}
      </ChatMessages>

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
// CAPABILITY PANEL VIEW - 能力面板视图
// ============================================================
interface CapabilityPanelViewProps {
  currentCapability: CapabilityType;
  capabilityStatus: CapabilityStatus;
  onCapabilitySelect: (capability: CapabilityType) => void;
  onBack: () => void;
}

function CapabilityPanelView({ currentCapability, capabilityStatus, onCapabilitySelect, onBack }: CapabilityPanelViewProps) {
  const md = useMediaQuery("md");
  const { t } = useTranslation();

  return (
    <div className="w-full h-full flex flex-col relative bg-background">
      {/* Mobile Sub-Header */}
      {!md && (
        <header className="flex items-center justify-between px-3 py-2 border-b border-border bg-background/80 backdrop-blur-md sticky top-0 z-20">
          <button
            onClick={onBack}
            className="flex items-center gap-1.5 px-2 py-1.5 rounded-lg text-muted-foreground hover:text-foreground hover:bg-muted transition-all"
          >
            <X className="w-4 h-4" />
            <span className="text-xs font-medium">{t("common.close") || "Close"}</span>
          </button>
          <span className="text-sm font-medium text-foreground">{t("ai.capability.title") || "我的能力"}</span>
          <div className="w-16" />
        </header>
      )}

      {/* Capability Panel */}
      <ParrotHub currentCapability={currentCapability} capabilityStatus={capabilityStatus} onCapabilitySelect={onCapabilitySelect} />
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
  const uiTools = useAITools();

  // Local state
  const [input, setInput] = useState("");
  const [isTyping, setIsTyping] = useState(false);
  const [isThinking, setIsThinking] = useState(false);

  const [clearDialogOpen, setClearDialogOpen] = useState(false);
  const [memoQueryResults, setMemoQueryResults] = useState<MemoQueryResultData[]>([]);
  const [scheduleQueryResults, setScheduleQueryResults] = useState<ScheduleQueryResultData[]>([]);
  const [sessionSummary, setSessionSummary] = useState<SessionSummary | undefined>();
  const [showCapabilityPanel, setShowCapabilityPanel] = useState(false);

  const timeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const messageIdRef = useRef(0);
  const lastAssistantMessageIdRef = useRef<string | null>(null);
  const streamingContentRef = useRef<string>("");
  const isCreatingConversationRef = useRef(false);
  const toolCallsRef = useRef<
    Array<{
      name: string;
      toolId?: string;
      inputSummary?: string;
      outputSummary?: string;
      filePath?: string;
    }>
  >([]);

  // Get current conversation and capability from context
  const {
    currentConversation,
    conversations,
    createConversation,
    selectConversation,
    addMessage,
    updateMessage,
    addReferencedMemos,
    addContextSeparator,
    clearMessages,
    state,
    setCurrentCapability,
    setCapabilityStatus,
    setMode,
    toggleImmersiveMode,
  } = aiChat;

  const currentCapability = state.currentCapability || CapabilityType.AUTO;
  const capabilityStatus = state.capabilityStatus || "idle";
  const currentMode = state.currentMode || "normal";
  const immersiveMode = state.immersiveMode || false;

  // Get messages from current conversation (memoized to prevent unnecessary re-renders)
  const items = useMemo(() => currentConversation?.messages || [], [currentConversation?.messages]);

  const { t } = useTranslation();

  // Clear timeout on unmount
  useEffect(() => {
    return () => {
      if (timeoutRef.current) {
        clearTimeout(timeoutRef.current);
      }
    };
  }, []);

  const resetTypingState = useCallback(() => {
    if (timeoutRef.current) {
      clearTimeout(timeoutRef.current);
      timeoutRef.current = null;
    }
    setIsTyping(false);
  }, []);

  // Handle parrot chat with callbacks
  const handleParrotChat = useCallback(
    async (conversationId: string, parrotId: ParrotAgentType, userMessage: string, _conversationIdNum: number) => {
      setIsTyping(true);
      setIsThinking(true);
      setCapabilityStatus("thinking");
      setMemoQueryResults([]);
      setScheduleQueryResults([]);
      toolCallsRef.current = []; // Reset tool calls for new message
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
          onThinking: (msg) => {
            if (lastAssistantMessageIdRef.current) {
              // Handle i18n keys from backend (e.g., "ai.geek_mode.thinking")
              const content = msg.startsWith("ai.") ? t(msg) : msg;
              updateMessage(conversationId, lastAssistantMessageIdRef.current, {
                content,
              });
            }
          },
          onToolUse: (toolName, meta) => {
            console.debug("[Geek/Evolution Mode] Tool use event:", toolName, meta);
            setCapabilityStatus("processing");
            // Accumulate tool calls for this message
            toolCallsRef.current.push({
              name: toolName,
              toolId: meta?.toolId,
              inputSummary: meta?.inputSummary,
              outputSummary: meta?.outputSummary,
              filePath: meta?.filePath,
            });
            if (lastAssistantMessageIdRef.current) {
              updateMessage(conversationId, lastAssistantMessageIdRef.current, {
                metadata: {
                  toolCalls: [...toolCallsRef.current], // Copy to avoid reference issues
                },
              });
            }
          },
          onToolResult: (_result, _meta) => {
            // Tool result received, metadata can be used for debugging
            console.debug("[Geek/Evolution Mode] Tool result:", _result, _meta);
          },
          onSessionSummary: (summary) => {
            console.debug("[Geek/Evolution Mode] Session summary:", summary);
            setSessionSummary(summary);
          },
          onMemoQueryResult: (result) => {
            if (_messageId === messageIdRef.current) {
              setMemoQueryResults((prev) => [...prev, result]);
              addReferencedMemos(conversationId, result.memos);
            }
          },
          onScheduleQueryResult: (result) => {
            if (_messageId === messageIdRef.current) {
              const transformedResult: ScheduleQueryResultData = {
                schedules: result.schedules.map((s) => ({
                  uid: s.uid,
                  title: s.title,
                  startTimestamp: Number(s.startTs),
                  endTimestamp: Number(s.endTs),
                  allDay: s.allDay,
                  location: s.location || undefined,
                  status: s.status,
                })),
                query: "",
                count: result.schedules.length,
                timeRangeDescription: result.timeRangeDescription,
                queryType: result.queryType,
              };
              setScheduleQueryResults((prev) => [...prev, transformedResult]);
            }
          },
          onUIMemoPreview: (data) => {
            if (_messageId === messageIdRef.current) {
              uiTools.processEvent({
                type: "ui_memo_preview",
                data: JSON.stringify(data),
                uiType: "ui_memo_preview",
                uiData: data,
              });
            }
          },
          onContent: (content) => {
            if (lastAssistantMessageIdRef.current) {
              streamingContentRef.current += content;
              updateMessage(conversationId, lastAssistantMessageIdRef.current, {
                content: streamingContentRef.current,
              });
            }
          },
          onDone: () => {
            setIsTyping(false);
            setIsThinking(false);
            setCapabilityStatus("idle");
          },
          onError: (error) => {
            setIsTyping(false);
            setIsThinking(false);
            setCapabilityStatus("idle");
            console.error("[Parrot Error]", error);
            if (lastAssistantMessageIdRef.current) {
              updateMessage(conversationId, lastAssistantMessageIdRef.current, {
                content: streamingContentRef.current || t("ai.error-generic") || "Sorry, something went wrong. Please try again.",
                error: true,
              });
            }
          },
        });
      } catch (error) {
        setIsTyping(false);
        setIsThinking(false);
        setCapabilityStatus("idle");
        console.error("[Parrot Chat Error]", error);
      }
    },
    [chatHook, updateMessage, addReferencedMemos, setCapabilityStatus, t, uiTools],
  );

  const handleSend = useCallback(
    async (messageContent?: string) => {
      const userMessage = (messageContent || input).trim();
      if (!userMessage) return;

      // Block sending when AI is still typing/replying
      // 用户可以在输入框输入文本，但当 AI 正在回复时不能发送新消息
      if (isTyping) {
        return;
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

        // No active conversation - create one with AMAZING agent (综合助手)
        // (会话不再绑定特定Agent，能力可以在会话中动态切换)
        const existingConversation = conversations.find((c) => !c.parrotId || c.parrotId === ParrotAgentType.AMAZING);
        if (existingConversation) {
          targetConversationId = existingConversation.id;
          selectConversation(existingConversation.id);
        } else {
          // Set lock before creating
          isCreatingConversationRef.current = true;
          const { id, completed } = createConversation(ParrotAgentType.AMAZING);
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

      // Add user message (optimistic update using tempId or realId)
      addMessage(targetConversationId, {
        role: "user",
        content: userMessage,
      });

      // Special handling for cutting line (context separator)
      if (userMessage === "---") {
        setInput("");
        // Wait for real ID if we just created it
        const finalId = creationPromise ? await creationPromise : targetConversationId;
        const targetConversationIdNum = parseInt(finalId, 10);
        await handleParrotChat(finalId, targetParrotId, userMessage, targetConversationIdNum);
        return;
      }

      // Add empty assistant message
      const newMessage = {
        role: "assistant" as const,
        content: "",
      };

      // Note: addMessage returns messageID. We don't use it for streaming logic
      // but we need it to update the specific message later if needed.
      const assistantMessageId = addMessage(targetConversationId, newMessage);
      lastAssistantMessageIdRef.current = assistantMessageId;

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
      isTyping,
      currentConversation,
      currentCapability,
      capabilityRouter,
      setCurrentCapability,
      conversations,
      selectConversation,
      createConversation,
      addMessage,
      handleParrotChat,
      resetTypingState,
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
    createConversation(ParrotAgentType.AMAZING);
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
  // RENDER
  // ============================================================
  return showCapabilityPanel ? (
    <CapabilityPanelView
      currentCapability={currentCapability}
      capabilityStatus={capabilityStatus}
      onCapabilitySelect={(cap) => {
        setCurrentCapability(cap);
        setShowCapabilityPanel(false);
      }}
      onBack={() => setShowCapabilityPanel(false)}
    />
  ) : (
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
      memoQueryResults={memoQueryResults}
      scheduleQueryResults={scheduleQueryResults}
      sessionSummary={sessionSummary}
      items={items}
      currentCapability={currentCapability}
      capabilityStatus={capabilityStatus}
      uiTools={uiTools}
      currentMode={currentMode}
      onModeChange={setMode}
      immersiveMode={immersiveMode}
      onImmersiveModeToggle={toggleImmersiveMode}
      isAdmin={true}
    />
  );
};

export default AIChat;
