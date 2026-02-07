import { create } from "@bufbuild/protobuf";
import { memo, ReactNode, useCallback, useEffect, useMemo, useRef } from "react";
import { useTranslation } from "react-i18next";
import TypingCursor from "@/components/AIChat/TypingCursor";
import { useForkBlock } from "@/hooks/useBlockQueries";
import { cn } from "@/lib/utils";
import { type AIMode, ConversationMessage, MessageRole } from "@/types/aichat";
// Block types (single source of truth for chat data)
import type { Block as AIBlock } from "@/types/block";
import { blockModeToParrotAgentType, getBlockModeName, isErrorStatus, isStreamingStatus } from "@/types/block";
import type { BlockSummary } from "@/types/parrot";
import { PARROT_THEMES, ParrotAgentType } from "@/types/parrot";
import type { SessionStats } from "@/types/proto/api/v1/ai_service_pb";
import { BlockType, UserInputSchema } from "@/types/proto/api/v1/ai_service_pb";
// BlockEditDialog for editing user inputs
import { BlockEditDialog, useBlockEditDialog } from "./BlockEditDialog";
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
  children,
  className,
  isStreaming = false,
  streamingContent = "",
  onCancel,
  // blockSummary prop deprecated - each Block now has its own summary via sessionStats
  blockSummary: _deprecatedBlockSummary,
  conversationId,
}: ChatMessagesProps) {
  // Suppress unused variable warning
  void _deprecatedBlockSummary;

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

  const prevBlocksLengthRef = useRef(0);
  useEffect(() => {
    const blocksLength = blocks?.length ?? 0;
    const hasNewMessage = blocksLength > prevBlocksLengthRef.current;
    prevBlocksLengthRef.current = blocksLength;

    if (hasNewMessage && !isUserScrollingRef.current) {
      scrollToBottomLocked();
    }
  }, [blocks?.length, scrollToBottomLocked]);

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

  const blocksLengthRef = useRef(0);
  useEffect(() => {
    const blocksLength = blocks?.length ?? 0;
    const lengthChanged = blocksLength !== blocksLengthRef.current;
    blocksLengthRef.current = blocksLength;

    if (!lengthChanged) return;

    if (scrollRef.current) {
      const { scrollTop, scrollHeight, clientHeight } = scrollRef.current;
      const distanceToBottom = scrollHeight - scrollTop - clientHeight;
      if (distanceToBottom <= SCROLL_THRESHOLD && isUserScrollingRef.current) {
        isUserScrollingRef.current = false;
      }
    }
  }, [blocks?.length]);

  useEffect(() => {
    return () => {
      if (rafIdRef.current) {
        cancelAnimationFrame(rafIdRef.current);
      }
    };
  }, []);

  // Get translation function
  const { t } = useTranslation();

  // Fork block mutation
  const forkBlock = useForkBlock();

  // Block edit dialog state management
  const editDialog = useBlockEditDialog();

  // Handle edit confirmation - call ForkBlock API with new user input
  const handleEditConfirm = useCallback(
    async (editedMessage: string, blockId: bigint, _convId: number) => {
      try {
        // Create new UserInput with edited message
        // Fix: Use create() to properly construct protobuf message with $typeName
        const newUserInput = create(UserInputSchema, {
          content: editedMessage,
          timestamp: BigInt(Date.now()),
          metadata: "{}",
        });

        // Fork block with replaced user input
        await forkBlock.mutateAsync({
          blockId,
          reason: `User edited message: "${editedMessage}"`,
          replaceUserInputs: [newUserInput],
        });

        // The forked block will appear in the block list with the new user input
        // User can continue the conversation by sending a new message
        editDialog.closeDialog();
      } catch (error) {
        console.error("Failed to fork block:", error);
      }
    },
    [forkBlock, editDialog],
  );

  // Handle edit button click - merge all user inputs for editing
  const handleEdit = useCallback(
    (blockId: bigint, block: MessageBlock) => {
      if (!conversationId) return;

      // Merge all user inputs (primary + additional) into a single message
      const allInputs = [block.userMessage, ...(block.additionalUserInputs || [])];
      const mergedMessage = allInputs
        .map((msg) => msg.content)
        .filter((content) => content)
        .join("\n");

      editDialog.openDialog(blockId, conversationId, mergedMessage);
    },
    [conversationId, editDialog],
  );

  // Group messages into blocks - Block data is single source of truth
  // Note: t is excluded from deps as translateThinkingSteps handles it conditionally
  const messageBlocks = useMemo(() => {
    if (!blocks || blocks.length === 0) return [];
    return convertAIBlocksToMessageBlocks(blocks, t);
  }, [blocks]); // t is stable from i18next, no need to track

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

            // Get branchPath from block
            const branchPath = blocks && blocks.length > 0 && index < blocks.length ? blocks[index].branchPath : undefined;

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
                onEdit={blockId ? () => handleEdit(blockId, block) : undefined}
                onCancel={block.isLatest && isLastStreaming ? onCancel : undefined}
                blockId={blockId}
                branchPath={branchPath}
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

      {/* Block Edit Dialog */}
      <BlockEditDialog
        originalMessage={editDialog.originalMessage}
        blockId={editDialog.blockId}
        conversationId={editDialog.conversationId}
        open={editDialog.open}
        onOpenChange={editDialog.setOpen}
        onConfirm={handleEditConfirm}
      />
    </div>
  );
});

export { ChatMessages };
