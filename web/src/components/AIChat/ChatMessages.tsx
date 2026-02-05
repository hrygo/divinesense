import { memo, ReactNode, useCallback, useEffect, useMemo, useRef } from "react";
import TypingCursor from "@/components/AIChat/TypingCursor";
import { cn } from "@/lib/utils";
import { type AIMode, ChatItem, ConversationMessage, isContextSeparator, MessageRole } from "@/types/aichat";
// Phase 4: Import Block types
import type { Block as AIBlock } from "@/types/block";
import { BLOCK_STATUS, blockModeToParrotAgentType, EVENT_TYPE, getBlockModeName } from "@/types/block";
import type { SessionSummary } from "@/types/parrot";
import { PARROT_THEMES, ParrotAgentType } from "@/types/parrot";
import { BlockType } from "@/types/proto/api/v1/ai_service_pb";
import { UnifiedMessageBlock } from "./UnifiedMessageBlock";
// Event transformation utilities
import { extractThinkingSteps, extractToolCalls, normalizeTimestamp } from "./utils/eventTransformers";

// ============================================================================
// Helper Hooks for ChatMessages
// ============================================================================

/** Hook to check if the last block is currently streaming */
function useStreamingStatus(blocks: AIBlock[] | undefined, isStreaming: boolean): boolean {
  return useMemo(() => {
    if (blocks && blocks.length > 0) {
      const lastAIBlock = blocks[blocks.length - 1];
      return String(lastAIBlock.status) === String(BLOCK_STATUS.STREAMING);
    }
    return isStreaming;
  }, [blocks, isStreaming]);
}

/** Hook to determine the effective parrot ID considering session and block modes */
function useEffectiveParrotId(
  currentParrotId: ParrotAgentType | undefined,
  sessionSummary: SessionSummary | undefined,
  blocks: AIBlock[] | undefined,
): ParrotAgentType {
  return useMemo(() => {
    // Session summary has highest priority
    if (sessionSummary?.mode === "geek") return ParrotAgentType.GEEK;
    if (sessionSummary?.mode === "evolution") return ParrotAgentType.EVOLUTION;

    // Check last Block mode
    if (blocks && blocks.length > 0) {
      const lastAIBlock = blocks[blocks.length - 1];
      return blockModeToParrotAgentType(lastAIBlock.mode);
    }

    return currentParrotId ?? ParrotAgentType.AMAZING;
  }, [currentParrotId, sessionSummary?.mode, blocks]);
}

// ============================================================================

interface ChatMessagesProps {
  items: ChatItem[];
  isTyping?: boolean;
  currentParrotId?: ParrotAgentType;
  onCopyMessage?: (content: string) => void;
  onRegenerate?: () => void;
  onDeleteMessage?: (index: number) => void;
  children?: ReactNode;
  className?: string;
  amazingInsightCard?: ReactNode;
  /** Phase 2: 流式渲染支持 */
  isStreaming?: boolean;
  streamingContent?: string;
  /** Session summary for Geek/Evolution modes */
  sessionSummary?: SessionSummary;
  /** Phase 4: Block data support */
  blocks?: AIBlock[];
}

const SCROLL_THRESHOLD = 150;
const SCROLL_THROTTLE_MS = 50;

/**
 * Pair user and assistant messages into blocks
 * Each block contains: user message + optional assistant response
 */
interface MessageBlock {
  id: string;
  userMessage: ConversationMessage;
  assistantMessage?: ConversationMessage;
  isLatest: boolean;
  /** Session summary attached to this block (only for last block) */
  attachSessionSummary?: boolean;
}

/**
 * Convert AIBlock[] to MessageBlock[] format
 * Phase 4: New function to handle Block data structure
 *
 * Each AIBlock contains:
 * - userInputs: UserInput[] (can have multiple user inputs per block)
 * - assistantContent: string (the assistant response)
 * - mode: BlockMode (normal/geek/evolution)
 * - status: BlockStatus (pending/streaming/completed/error)
 * - eventStream: BlockEvent[] (thinking/tool_use/tool_result/answer events)
 * - sessionStats: SessionStats (for Geek/Evolution modes)
 */
function convertAIBlocksToMessageBlocks(blocks: AIBlock[], hasSessionSummary: boolean): MessageBlock[] {
  const messageBlocks: MessageBlock[] = [];

  for (const block of blocks) {
    // Skip context separator blocks
    if (block.blockType === BlockType.CONTEXT_SEPARATOR) {
      continue;
    }

    // Combine all user inputs into a single message
    const userContent = block.userInputs
      .map((ui) => ui.content)
      .filter(Boolean)
      .join("\n");

    const userMessage: ConversationMessage = {
      id: `block-${block.id}`,
      role: "user" as MessageRole,
      content: userContent,
      timestamp: normalizeTimestamp(block.createdTs),
      metadata: {
        mode: getBlockModeName(block.mode) as AIMode,
      },
    };

    // Build assistant message from assistantContent and eventStream
    const assistantMessage: ConversationMessage = {
      id: `block-${block.id}-assistant`,
      role: "assistant" as MessageRole,
      content: block.assistantContent || "",
      timestamp: normalizeTimestamp(block.updatedTs),
      error: String(block.status) === String(BLOCK_STATUS.ERROR),
      metadata: {
        mode: getBlockModeName(block.mode) as AIMode,
        // Parse eventStream to extract metadata for UI
        thinkingSteps: extractThinkingSteps(block.eventStream),
        toolCalls: extractToolCalls(block.eventStream),
      },
    };

    const isLatest = false; // Will be determined after loop
    const attachSessionSummary = false; // Will be set for last block

    messageBlocks.push({
      id: String(block.id),
      userMessage,
      assistantMessage,
      isLatest,
      attachSessionSummary,
    });
  }

  // Mark last block as latest and attach session summary if available
  if (messageBlocks.length > 0) {
    const lastBlock = messageBlocks[messageBlocks.length - 1];
    lastBlock.isLatest = true;
    if (hasSessionSummary && lastBlock.assistantMessage) {
      lastBlock.attachSessionSummary = true;
    }
  }

  return messageBlocks;
}

/**
 * Group messages into user-assistant pairs
 * Legacy function for ChatItem[] support (backward compatibility)
 */
function groupMessagesIntoBlocks(items: ChatItem[], hasSessionSummary: boolean): MessageBlock[] {
  const blocks: MessageBlock[] = [];
  let pendingUser: ConversationMessage | null = null;

  for (const item of items) {
    // Skip context separators for now (they could be rendered separately)
    if (isContextSeparator(item)) {
      if (pendingUser) {
        blocks.push({
          id: pendingUser.id,
          userMessage: pendingUser,
          isLatest: false,
        });
        pendingUser = null;
      }
      continue;
    }

    const msg = item as ConversationMessage;

    if (msg.role === "user") {
      // If we have a pending user message, flush it first
      if (pendingUser) {
        blocks.push({
          id: pendingUser.id,
          userMessage: pendingUser,
          isLatest: false,
        });
      }
      pendingUser = msg;
    } else if (msg.role === "assistant") {
      // Pair with user message if available
      if (pendingUser) {
        blocks.push({
          id: pendingUser.id,
          userMessage: pendingUser,
          assistantMessage: msg,
          isLatest: false,
        });
        pendingUser = null;
      } else {
        // Orphan assistant message (shouldn't happen normally)
        blocks.push({
          id: msg.id,
          userMessage: {
            id: `system-${msg.id}`,
            role: "user" as MessageRole,
            content: "",
            timestamp: msg.timestamp,
          },
          assistantMessage: msg,
          isLatest: false,
        });
      }
    }
  }

  // Flush remaining user message
  if (pendingUser) {
    blocks.push({
      id: pendingUser.id,
      userMessage: pendingUser,
      isLatest: true,
    });
  }

  // Mark last block as latest and attach session summary if available
  if (blocks.length > 0) {
    const lastBlock = blocks[blocks.length - 1];
    lastBlock.isLatest = true;
    // Only attach session summary to the last block if it has an assistant message
    if (hasSessionSummary && lastBlock.assistantMessage) {
      lastBlock.attachSessionSummary = true;
    }
  }

  return blocks;
}

const ChatMessages = memo(function ChatMessages({
  items,
  blocks,
  isTyping = false,
  currentParrotId,
  onCopyMessage,
  onRegenerate,
  onDeleteMessage,
  children,
  className,
  amazingInsightCard,
  isStreaming = false,
  streamingContent = "",
  sessionSummary,
}: ChatMessagesProps) {
  const scrollRef = useRef<HTMLDivElement>(null);
  const endRef = useRef<HTMLDivElement>(null);
  const rafIdRef = useRef<ReturnType<typeof requestAnimationFrame> | null>(null);
  const lastScrollTimeRef = useRef(0);
  const isUserScrollingRef = useRef(false);
  const lastContentLengthRef = useRef(0);

  const scrollToBottomLocked = useCallback(() => {
    if (rafIdRef.current) return;

    rafIdRef.current = requestAnimationFrame(() => {
      rafIdRef.current = null;

      if (scrollRef.current && !isUserScrollingRef.current) {
        const { scrollTop, scrollHeight, clientHeight } = scrollRef.current;
        const distanceToBottom = scrollHeight - scrollTop - clientHeight;

        if (distanceToBottom < SCROLL_THRESHOLD) {
          scrollRef.current.scrollTop = scrollHeight;
        }
      }
    });
  }, []);

  const handleScroll = useCallback(() => {
    if (!scrollRef.current) return;

    const { scrollTop, scrollHeight, clientHeight } = scrollRef.current;
    const distanceToBottom = scrollHeight - scrollTop - clientHeight;
    const shouldBeScrolling = distanceToBottom > SCROLL_THRESHOLD;

    if (isUserScrollingRef.current !== shouldBeScrolling) {
      isUserScrollingRef.current = shouldBeScrolling;
    }
  }, []);

  const handleScrollThrottled = useCallback(() => {
    const now = Date.now();
    if (now - lastScrollTimeRef.current < SCROLL_THROTTLE_MS) return;
    lastScrollTimeRef.current = now;
    handleScroll();
  }, [handleScroll]);

  useEffect(() => {
    if (!scrollRef.current) return;

    const observer = new MutationObserver((mutations) => {
      const hasNewNodes = mutations.some((m) => m.type === "childList" && m.addedNodes.length > 0);

      if (hasNewNodes && !isUserScrollingRef.current) {
        scrollToBottomLocked();
      }
    });

    const contentElement = scrollRef.current.firstElementChild;
    if (contentElement) {
      observer.observe(contentElement, {
        childList: true,
        subtree: true,
      });
    }

    return () => observer.disconnect();
  }, [scrollToBottomLocked]);

  const prevItemsLengthRef = useRef(items.length);
  useEffect(() => {
    const itemsLength = items.length;
    const hasNewMessage = itemsLength > prevItemsLengthRef.current;
    prevItemsLengthRef.current = itemsLength;

    if (hasNewMessage && !isUserScrollingRef.current) {
      scrollToBottomLocked();
    }
  }, [items.length, scrollToBottomLocked]);

  useEffect(() => {
    if (!isStreaming) return;

    const contentLength = streamingContent.length;
    const contentIncrease = contentLength - lastContentLengthRef.current;
    lastContentLengthRef.current = contentLength;

    if (contentIncrease > 50 && !isUserScrollingRef.current) {
      scrollToBottomLocked();
    }
  }, [streamingContent, isStreaming, scrollToBottomLocked]);

  useEffect(() => {
    if (isTyping && !isUserScrollingRef.current) {
      scrollToBottomLocked();
    }
  }, [isTyping, scrollToBottomLocked]);

  const itemsLengthRef = useRef(items.length);
  useEffect(() => {
    const lengthChanged = items.length !== itemsLengthRef.current;
    itemsLengthRef.current = items.length;

    if (!lengthChanged) return;

    if (scrollRef.current) {
      const { scrollTop, scrollHeight, clientHeight } = scrollRef.current;
      const distanceToBottom = scrollHeight - scrollTop - clientHeight;
      if (distanceToBottom <= SCROLL_THRESHOLD && isUserScrollingRef.current) {
        isUserScrollingRef.current = false;
      }
    }
  }, [items.length]);

  useEffect(() => {
    return () => {
      if (rafIdRef.current) {
        cancelAnimationFrame(rafIdRef.current);
      }
    };
  }, []);

  // Group messages into blocks
  // Phase 4: Use blocks if provided, otherwise fall back to items (backward compatibility)
  const messageBlocks = useMemo(() => {
    if (blocks && blocks.length > 0) {
      // Use new Block data structure
      return convertAIBlocksToMessageBlocks(blocks, !!sessionSummary);
    }
    // Legacy: use ChatItem[] structure
    return groupMessagesIntoBlocks(items, !!sessionSummary);
  }, [blocks, items, sessionSummary]);

  // Phase 4: Check streaming status from either blocks or props (using extracted hook)
  const isLastStreaming = useStreamingStatus(blocks, isStreaming ?? false);

  // 计算当前流式阶段（用于动画效果）
  // Phase 4: Enhanced to read from Block eventStream
  const streamingPhase = useMemo((): "thinking" | "tools" | "answer" | null => {
    if (!isLastStreaming) return null;

    // Phase 4: Check Block eventStream first
    if (blocks && blocks.length > 0) {
      const lastAIBlock = blocks[blocks.length - 1];
      const events = lastAIBlock.eventStream || [];

      // Get the most recent non-answer event
      const recentEvents = events.filter((e) => e.type !== EVENT_TYPE.ANSWER);
      const lastNonAnswerEvent = recentEvents[recentEvents.length - 1];

      if (lastNonAnswerEvent) {
        if (lastNonAnswerEvent.type === EVENT_TYPE.TOOL_USE) {
          // Check if there's a corresponding tool_result
          const hasResult = events.some((e, i) => e.type === EVENT_TYPE.TOOL_RESULT && i > events.indexOf(lastNonAnswerEvent));
          if (!hasResult) return "tools";
        }
        if (lastNonAnswerEvent.type === EVENT_TYPE.THINKING) return "thinking";
      }

      // Check if we have answer content
      if (lastAIBlock.assistantContent) return "answer";

      return null;
    }

    // Legacy: use message metadata from messageBlocks
    const lastMessageBlock = messageBlocks[messageBlocks.length - 1];
    if (!lastMessageBlock?.assistantMessage) return null;
    const metadata = lastMessageBlock.assistantMessage.metadata;
    if (!metadata) return null;

    // 有工具调用正在执行（有 toolCalls 但没有 outputSummary）
    const hasPendingTools = metadata.toolCalls?.some((tc) => typeof tc === "object" && !tc.outputSummary);
    if (hasPendingTools) return "tools";

    // 有内容正在流式输出
    if (lastMessageBlock.assistantMessage.content) return "answer";

    // 正在思考（有 thinkingSteps 或 thinking 但还没有内容）
    if ((metadata.thinkingSteps?.length ?? 0) > 0 || metadata.thinking) return "thinking";

    return null;
  }, [isLastStreaming, blocks, messageBlocks, isStreaming]);

  // Determine effective parrot ID based on session mode (Geek/Evolution override normal parrotId)
  // Phase 4: Also consider Block mode (using extracted hook)
  const effectiveParrotId = useEffectiveParrotId(currentParrotId, sessionSummary, blocks);

  return (
    <div
      ref={scrollRef}
      onScroll={handleScrollThrottled}
      className={cn("flex-1 overflow-y-auto px-3 md:px-6 py-4 overscroll-contain", className)}
      style={{ overflowAnchor: "auto", scrollbarGutter: "stable", contain: "layout style paint" }}
    >
      {children}

      {messageBlocks.length > 0 && (
        <div className="max-w-3xl lg:max-w-4xl xl:max-w-5xl 2xl:max-w-6xl mx-auto space-y-3">
          {messageBlocks.map((block, index) => {
            const blockIsLast = index === messageBlocks.length - 1;

            // Phase 4: Determine effective parrot ID for this specific block
            // Priority order:
            // 1. Block's own mode (when using blocks prop)
            // 2. Message metadata mode (legacy items)
            // 3. Session summary mode (fallback)
            // 4. Current global parrotId (fallback)
            let blockParrotId = effectiveParrotId;

            if (blocks && blocks.length > 0 && index < blocks.length) {
              // Direct from Block mode
              blockParrotId = blockModeToParrotAgentType(blocks[index].mode);
            } else {
              // Legacy: from message metadata
              const blockMode: AIMode | undefined = block.assistantMessage?.metadata?.mode || block.userMessage?.metadata?.mode;
              if (blockMode === "geek") {
                blockParrotId = ParrotAgentType.GEEK;
              } else if (blockMode === "evolution") {
                blockParrotId = ParrotAgentType.EVOLUTION;
              } else if (blockMode === "normal") {
                blockParrotId = ParrotAgentType.AMAZING;
              }
            }

            return (
              <UnifiedMessageBlock
                key={block.id}
                userMessage={block.userMessage}
                assistantMessage={block.assistantMessage}
                sessionSummary={block.attachSessionSummary ? sessionSummary : undefined}
                parrotId={blockParrotId}
                isLatest={block.isLatest}
                isStreaming={isLastStreaming && block.isLatest}
                streamingPhase={blockIsLast ? streamingPhase : null}
                onCopy={onCopyMessage}
                onRegenerate={block.isLatest ? onRegenerate : undefined}
                onDelete={block.isLatest && onDeleteMessage ? () => onDeleteMessage(0) : undefined}
              >
                {/* Typing cursor for streaming messages */}
                {block.isLatest && isTyping && !block.assistantMessage?.error && (
                  <TypingCursor active={true} parrotId={effectiveParrotId || ParrotAgentType.AMAZING} variant="dots" />
                )}
              </UnifiedMessageBlock>
            );
          })}
        </div>
      )}

      {/* Amazing Insight Card - rendered separately */}
      {amazingInsightCard && !isTyping && messageBlocks.length > 0 && (
        <div className="max-w-3xl lg:max-w-4xl xl:max-w-5xl 2xl:max-w-6xl mx-auto mt-3">{amazingInsightCard}</div>
      )}

      {/* Typing indicator when no messages yet */}
      {isTyping && messageBlocks.length === 0 && (
        <div className="flex gap-3 md:gap-4 animate-in fade-in duration-300">
          <div className="w-9 h-9 md:w-10 md:h-10 rounded-full flex items-center justify-center shadow-sm bg-muted">
            <span className="text-lg md:text-xl">🤖</span>
          </div>
          <div className={cn("px-4 py-3 rounded-2xl border shadow-sm", PARROT_THEMES.AMAZING.bubbleBg, PARROT_THEMES.AMAZING.bubbleBorder)}>
            <TypingCursor active={true} parrotId={effectiveParrotId || ParrotAgentType.AMAZING} variant="dots" />
          </div>
        </div>
      )}

      {/* Scroll anchor */}
      <div ref={endRef} className="h-px" />
    </div>
  );
});

export { ChatMessages };
