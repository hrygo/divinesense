import { memo, ReactNode, useCallback, useEffect, useMemo, useRef } from "react";
import { useTranslation } from "react-i18next";
import TypingCursor from "@/components/AIChat/TypingCursor";
import { cn } from "@/lib/utils";
import { type AIMode, ConversationMessage, MessageRole } from "@/types/aichat";
// Block types (single source of truth for chat data)
import type { Block as AIBlock } from "@/types/block";
import { blockModeToParrotAgentType, getBlockModeName, isErrorStatus, isStreamingStatus } from "@/types/block";
import type { BlockSummary } from "@/types/parrot";
import { PARROT_THEMES, ParrotAgentType } from "@/types/parrot";
import type { SessionStats } from "@/types/proto/api/v1/ai_service_pb";
import { BlockType } from "@/types/proto/api/v1/ai_service_pb";
import { UnifiedMessageBlock } from "./UnifiedMessageBlock";
// Event transformation utilities
import { extractThinkingSteps, extractToolCalls, normalizeTimestamp, type ThinkingStep } from "./utils/eventTransformers";

// ============================================================================
// Helper Hooks for ChatMessages
// ============================================================================

/** Hook to check if the last block is currently streaming */
function useStreamingStatus(blocks: AIBlock[] | undefined, isStreaming: boolean): boolean {
  return useMemo(() => {
    if (blocks && blocks.length > 0) {
      const lastAIBlock = blocks[blocks.length - 1];
      return isStreamingStatus(lastAIBlock.status);
    }
    return isStreaming;
  }, [blocks, isStreaming]);
}

/** Hook to determine the effective parrot ID from Block.mode (single source of truth) */
function useEffectiveParrotId(currentParrotId: ParrotAgentType | undefined, blocks: AIBlock[] | undefined): ParrotAgentType {
  return useMemo(() => {
    // Block.mode is the single source of truth for mode determination
    if (blocks && blocks.length > 0) {
      const lastAIBlock = blocks[blocks.length - 1];
      return blockModeToParrotAgentType(lastAIBlock.mode);
    }

    return currentParrotId ?? ParrotAgentType.AMAZING;
  }, [currentParrotId, blocks]);
}

// ============================================================================

interface ChatMessagesProps {
  /** Block data - single source of truth for chat messages */
  blocks: AIBlock[];
  isTyping?: boolean;
  currentParrotId?: ParrotAgentType;
  onCopyMessage?: (content: string) => void;
  onRegenerate?: () => void;
  onDeleteMessage?: (index: number) => void;
  onQuickReply?: (text: string) => void;
  children?: ReactNode;
  className?: string;
  /** 流式渲染支持 */
  isStreaming?: boolean;
  streamingContent?: string;
  /** 取消流式请求回调 (#113) */
  onCancel?: () => void;
  /** @deprecated Block summary now comes from Block.sessionStats (1:1 binding) */
  blockSummary?: BlockSummary;
  /** Conversation ID for Block API operations (e.g., fork) */
  conversationId?: number;
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
  additionalUserInputs?: ConversationMessage[];
  assistantMessage?: ConversationMessage;
  isLatest: boolean;
  /** Block summary for this specific block (from Block.sessionStats) */
  blockSummary?: BlockSummary;
}

/**
 * Translate i18n keys in thinking steps content
 * Optimized to avoid creating new array when no translation is needed.
 * @param steps - Thinking steps with possibly raw i18n keys
 * @param t - Translation function
 * @returns Thinking steps with translated content
 */
function translateThinkingSteps(steps: ThinkingStep[], t: (key: string) => string): ThinkingStep[] {
  // Check if any step needs translation (starts with "ai.")
  const needsTranslation = steps.some((s) => s.content.startsWith("ai."));
  if (!needsTranslation) return steps;

  return steps.map((step) => ({
    ...step,
    content: step.content.startsWith("ai.") ? t(step.content) : step.content,
  }));
}

/**
 * Convert SessionStats from Block to BlockSummary format
 * This enables 1:1 binding between Block and its summary
 */
function sessionStatsToBlockSummary(sessionStats: SessionStats): BlockSummary | undefined {
  // Only create summary if we have meaningful data
  if (!sessionStats) return undefined;

  const hasStats =
    (sessionStats.inputTokens || 0) > 0 ||
    (sessionStats.outputTokens || 0) > 0 ||
    (sessionStats.totalDurationMs || 0) > 0 ||
    (sessionStats.toolCallCount || 0) > 0;

  if (!hasStats) return undefined;

  return {
    sessionId: sessionStats.sessionId || undefined,
    totalDurationMs: sessionStats.totalDurationMs ? Number(sessionStats.totalDurationMs) : undefined,
    thinkingDurationMs: sessionStats.thinkingDurationMs ? Number(sessionStats.thinkingDurationMs) : undefined,
    toolDurationMs: sessionStats.toolDurationMs ? Number(sessionStats.toolDurationMs) : undefined,
    generationDurationMs: sessionStats.generationDurationMs ? Number(sessionStats.generationDurationMs) : undefined,
    totalInputTokens: sessionStats.inputTokens || undefined,
    totalOutputTokens: sessionStats.outputTokens || undefined,
    totalCacheWriteTokens: sessionStats.cacheWriteTokens || undefined,
    totalCacheReadTokens: sessionStats.cacheReadTokens || undefined,
    toolCallCount: sessionStats.toolCallCount || undefined,
    toolsUsed: sessionStats.toolsUsed?.length ? sessionStats.toolsUsed : undefined,
    filesModified: sessionStats.filesModified || undefined,
    filePaths: sessionStats.filePaths?.length ? sessionStats.filePaths : undefined,
    totalCostUSD: sessionStats.totalCostUsd || undefined,
    status: sessionStats.isError ? "error" : "success",
    errorMsg: sessionStats.errorMessage || undefined,
  };
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
 *
 * IMPORTANT: BlockSummary is now 1:1 bound to each Block via its sessionStats
 * This ensures proper persistence across page refreshes.
 *
 * @param blocks - Array of AIBlock objects
 * @param t - Translation function for i18n keys
 */
function convertAIBlocksToMessageBlocks(blocks: AIBlock[], _t: (key: string) => string): MessageBlock[] {
  const messageBlocks: MessageBlock[] = [];

  for (const block of blocks) {
    // Skip context separator blocks
    if (block.blockType === BlockType.CONTEXT_SEPARATOR) {
      continue;
    }

    // Split userInputs: first as userMessage, rest as additionalUserInputs
    const firstInput = block.userInputs[0];
    const restInputs = block.userInputs.slice(1);

    const userMessage: ConversationMessage = {
      id: `block-${block.id.toString()}`,
      role: "user" as MessageRole,
      content: firstInput?.content || "",
      timestamp: normalizeTimestamp(firstInput?.timestamp || block.createdTs),
      metadata: {
        mode: getBlockModeName(block.mode) as AIMode,
      },
    };

    // Build additional user inputs (appended messages)
    const additionalUserInputs: ConversationMessage[] = restInputs
      .filter((ui) => ui.content)
      .map((ui, idx) => ({
        id: `block-${block.id.toString()}-additional-${idx}`,
        role: "user" as MessageRole,
        content: ui.content,
        timestamp: normalizeTimestamp(ui.timestamp || block.createdTs),
        metadata: {
          mode: getBlockModeName(block.mode) as AIMode,
        },
      }));

    // Build assistant message from assistantContent and eventStream
    const rawThinkingSteps = extractThinkingSteps(block.eventStream);
    const toolCalls = extractToolCalls(block.eventStream);

    // Extract toolResults from toolCalls (toolCalls with outputSummary are completed)
    const toolResults = toolCalls
      .filter((call) => call.outputSummary !== undefined)
      .map((call) => ({
        name: call.name,
        outputSummary: call.outputSummary,
        isError: call.isError ?? false,
        duration: call.duration,
      }));

    const assistantMessage: ConversationMessage = {
      id: `block-${block.id.toString()}-assistant`,
      role: "assistant" as MessageRole,
      content: block.assistantContent || "",
      timestamp: normalizeTimestamp(block.updatedTs),
      error: isErrorStatus(block.status),
      metadata: {
        mode: getBlockModeName(block.mode) as AIMode,
        // Parse eventStream to extract metadata for UI, translating i18n keys
        thinkingSteps: translateThinkingSteps(rawThinkingSteps, _t),
        toolCalls: toolCalls,
        toolResults: toolResults,
      },
    };

    const isLatest = false; // Will be determined after loop
    // Generate blockSummary from Block's sessionStats (1:1 binding)
    const blockSummary = block.sessionStats ? sessionStatsToBlockSummary(block.sessionStats) : undefined;

    messageBlocks.push({
      id: block.id.toString(),
      userMessage,
      additionalUserInputs: additionalUserInputs.length > 0 ? additionalUserInputs : undefined,
      assistantMessage,
      isLatest,
      blockSummary,
    });
  }

  // Mark last block as latest
  if (messageBlocks.length > 0) {
    const lastBlock = messageBlocks[messageBlocks.length - 1];
    lastBlock.isLatest = true;
  }

  return messageBlocks;
}

const ChatMessages = memo(function ChatMessages({
  blocks,
  isTyping = false,
  currentParrotId,
  onCopyMessage,
  onRegenerate,
  onDeleteMessage,
  onQuickReply,
  children,
  className,
  isStreaming = false,
  streamingContent = "",
  onCancel,
  // blockSummary prop deprecated - each Block now has its own summary via sessionStats
  blockSummary: _deprecatedBlockSummary,
  conversationId: _conversationId,
}: ChatMessagesProps) {
  // Suppress unused variable warnings
  void _deprecatedBlockSummary;
  void _conversationId;

  const scrollRef = useRef<HTMLDivElement>(null);
  const endRef = useRef<HTMLDivElement>(null);
  const rafIdRef = useRef<ReturnType<typeof requestAnimationFrame> | null>(null);
  const lastScrollTimeRef = useRef(0);
  const isUserScrollingRef = useRef(false);

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

  // P1-1 OPTIMIZATION: Combined scroll management with useLatest pattern
  // Reduces from 6 useEffect to 2, minimizing re-renders and RAF scheduling
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

  // P1-1 OPTIMIZATION: Unified scroll-to-bottom logic with single RAF
  // Replaces 4 separate useEffect with one optimized version
  const prevBlocksLengthRef = useRef(0);
  const prevStreamingContentLengthRef = useRef(0);
  const prevIsTypingRef = useRef(false);
  const prevIsStreamingRef = useRef(false);

  // eslint-disable-next-line react-hooks/exhaustive-deps -- scrollRef intentionally excluded as it's accessed conditionally
  useEffect(() => {
    const blocksLength = blocks?.length ?? 0;
    const hasNewMessage = blocksLength > prevBlocksLengthRef.current;
    prevBlocksLengthRef.current = blocksLength;

    // Streaming content increase detection
    const contentLength = streamingContent.length;
    const contentIncrease = contentLength - prevStreamingContentLengthRef.current;
    prevStreamingContentLengthRef.current = contentLength;

    // Update refs for state change tracking
    prevIsTypingRef.current = isTyping;
    prevIsStreamingRef.current = isStreaming;

    // Unified scroll trigger logic - single RAF for all cases
    // Reduced threshold from 50 to 15 for more responsive streaming
    const shouldScroll =
      (hasNewMessage && !isUserScrollingRef.current) ||
      (isStreaming && contentIncrease > 15 && !isUserScrollingRef.current) ||
      (isTyping && !isUserScrollingRef.current);

    if (shouldScroll) {
      requestAnimationFrame(() => {
        if (!isUserScrollingRef.current) {
          scrollToBottomLocked();
        }
      });
    }

    // Reset user scroll state when block length changes
    if (blocksLength > 0) {
      const { scrollTop, scrollHeight, clientHeight } = scrollRef.current || ({} as HTMLElement);
      const distanceToBottom = (scrollHeight || 0) - (scrollTop || 0) - (clientHeight || 0);
      if (distanceToBottom <= SCROLL_THRESHOLD && isUserScrollingRef.current) {
        isUserScrollingRef.current = false;
      }
    }
  }, [blocks?.length, streamingContent, isStreaming, isTyping, scrollToBottomLocked]);

  useEffect(() => {
    return () => {
      if (rafIdRef.current) {
        cancelAnimationFrame(rafIdRef.current);
      }
    };
  }, []);

  // Get translation function
  const { t } = useTranslation();

  // Group messages into blocks - Block data is single source of truth
  // P1-2 OPTIMIZATION: Track block IDs to only recompute when blocks actually change
  // This avoids re-converting historical blocks during streaming updates
  const blocksIdsRef = useRef<string>("");
  const messageBlocksRef = useRef<MessageBlock[]>([]);

  const messageBlocks = useMemo(() => {
    if (!blocks || blocks.length === 0) return [];

    // Create stable key from block IDs and their essential properties
    const currentIds = blocks.map((b) => `${b.id}-${b.status}-${b.updatedTs}`).join(",");

    // If only the last block changed (streaming scenario), reuse cached conversions
    if (blocksIdsRef.current && currentIds.startsWith(blocksIdsRef.current)) {
      // Extract the changed part (new or modified blocks at the end)
      const prevBlocks = blocksIdsRef.current.split(",").map((idWithStatus) => idWithStatus.split("-")[0]);
      const newBlocks = blocks.filter((b) => !prevBlocks.includes(b.id.toString()));

      if (newBlocks.length > 0 && newBlocks.length < blocks.length) {
        // Only convert new/modified blocks
        const newMessageBlocks = convertAIBlocksToMessageBlocks(newBlocks, t);
        // Update ref and return combined result
        messageBlocksRef.current = [...messageBlocksRef.current, ...newMessageBlocks];
        blocksIdsRef.current = currentIds;
        return messageBlocksRef.current;
      }
    }

    // Full conversion needed (initial load or significant changes)
    const result = convertAIBlocksToMessageBlocks(blocks, t);
    blocksIdsRef.current = currentIds;
    messageBlocksRef.current = result;
    return result;
  }, [blocks, t]);

  // Phase 4: Check streaming status from either blocks or props (using extracted hook)
  const isLastStreaming = useStreamingStatus(blocks, isStreaming ?? false);

  // 计算当前流式阶段（用于动画效果）
  // Optimization: Extract only last block's relevant data to avoid recalc on any block change
  const lastBlockKey = useMemo(() => {
    if (!blocks || blocks.length === 0) return null;
    const last = blocks[blocks.length - 1];
    const events = last.eventStream || [];
    // Track only properties that affect streaming phase calculation
    const lastEvent = events[events.length - 1];
    return {
      lastEventType: lastEvent?.type,
      hasToolResult: lastEvent?.type === "tool_result",
      assistantContent: last.assistantContent,
    };
  }, [blocks]);

  const streamingPhase = useMemo((): "thinking" | "tools" | "answer" | null => {
    if (!isLastStreaming || !lastBlockKey) return null;

    // Determine phase from lastBlockKey (derived from last block's actual data)
    if (lastBlockKey.lastEventType === "tool_use" && !lastBlockKey.hasToolResult) return "tools";
    if (lastBlockKey.lastEventType === "thinking") return "thinking";
    if (lastBlockKey.assistantContent) return "answer";

    return null;
  }, [isLastStreaming, lastBlockKey]);

  // Determine effective parrot ID from Block.mode (single source of truth)
  const effectiveParrotId = useEffectiveParrotId(currentParrotId, blocks);

  return (
    <div
      ref={scrollRef}
      onScroll={handleScrollThrottled}
      className={cn("flex-1 overflow-y-auto px-3 md:px-6 py-4 overscroll-contain", className)}
      style={{ overflowAnchor: "auto", scrollbarGutter: "auto", contain: "layout style paint" }}
    >
      {children}

      {messageBlocks.length > 0 && (
        <div className="max-w-3xl lg:max-w-4xl xl:max-w-5xl 2xl:max-w-6xl mx-auto space-y-3">
          {messageBlocks.map((block, index) => {
            const blockIsLast = index === messageBlocks.length - 1;

            // Phase 2: Determine effective parrot ID from Block.mode (single source of truth)
            const blockParrotId =
              blocks && blocks.length > 0 && index < blocks.length ? blockModeToParrotAgentType(blocks[index].mode) : effectiveParrotId;

            // Get blockId for edit functionality
            const blockId = blocks && blocks.length > 0 && index < blocks.length ? blocks[index].id : undefined;

            // Calculate block number (1-based sequential number)
            const blockNumber = index + 1;

            return (
              <UnifiedMessageBlock
                key={`${block.id}-${index}`}
                userMessage={block.userMessage}
                additionalUserInputs={block.additionalUserInputs}
                assistantMessage={block.assistantMessage}
                blockSummary={block.blockSummary}
                parrotId={blockParrotId}
                isLatest={block.isLatest}
                isStreaming={isLastStreaming && block.isLatest}
                streamingPhase={blockIsLast ? streamingPhase : null}
                onCopy={onCopyMessage}
                onRegenerate={block.isLatest ? onRegenerate : undefined}
                onDelete={block.isLatest && onDeleteMessage ? () => onDeleteMessage(0) : undefined}
                onCancel={block.isLatest && isLastStreaming ? onCancel : undefined}
                onQuickReply={block.isLatest ? onQuickReply : undefined}
                blockId={blockId}
                blockNumber={blockNumber}
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
