import { memo, ReactNode, useCallback, useEffect, useMemo, useRef } from "react";
import TypingCursor from "@/components/AIChat/TypingCursor";
import { GenerativeUIContainer } from "@/components/ScheduleAI/GenerativeUIContainer";
import type { GenerativeUIContainerProps } from "@/components/ScheduleAI/types";
import { cn } from "@/lib/utils";
import { type AIMode, ChatItem, ConversationMessage, isContextSeparator, MessageRole } from "@/types/aichat";
import type { SessionSummary } from "@/types/parrot";
import { PARROT_THEMES, ParrotAgentType } from "@/types/parrot";
import { UnifiedMessageBlock } from "./UnifiedMessageBlock";

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
  /** Generative UI tools to render in message flow */
  uiTools?: GenerativeUIContainerProps["tools"];
  onUIAction?: GenerativeUIContainerProps["onAction"];
  onUIDismiss?: GenerativeUIContainerProps["onDismiss"];
  /** Phase 2: 流式渲染支持 */
  isStreaming?: boolean;
  streamingContent?: string;
  /** Session summary for Geek/Evolution modes */
  sessionSummary?: SessionSummary;
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
 * Group messages into user-assistant pairs
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
  isTyping = false,
  currentParrotId,
  onCopyMessage,
  onRegenerate,
  onDeleteMessage,
  children,
  className,
  amazingInsightCard,
  uiTools,
  onUIAction,
  onUIDismiss,
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
  // Use items.length and last item ID as dependencies to avoid recalculation
  // when the parent component re-renders with the same array content
  const messageBlocks = useMemo(
    () => groupMessagesIntoBlocks(items, !!sessionSummary),
    [items.length, items[items.length - 1]?.id, sessionSummary],
  );

  // Check if last assistant message is streaming
  const lastBlock = messageBlocks[messageBlocks.length - 1];
  const isLastStreaming = lastBlock?.assistantMessage ? isStreaming : false;

  // 计算当前流式阶段（用于动画效果）
  const streamingPhase = useMemo((): "thinking" | "tools" | "answer" | null => {
    if (!isLastStreaming || !lastBlock?.assistantMessage) return null;
    const metadata = lastBlock.assistantMessage.metadata;
    if (!metadata) return null;

    // 有工具调用正在执行（有 toolCalls 但没有 outputSummary）
    const hasPendingTools = metadata.toolCalls?.some((tc) => typeof tc === "object" && !tc.outputSummary);
    if (hasPendingTools) return "tools";

    // 有内容正在流式输出
    if (lastBlock.assistantMessage.content) return "answer";

    // 正在思考（有 thinkingSteps 或 thinking 但还没有内容）
    if ((metadata.thinkingSteps?.length ?? 0) > 0 || metadata.thinking) return "thinking";

    return null;
  }, [isLastStreaming, lastBlock, isStreaming]);

  // Determine effective parrot ID based on session mode (Geek/Evolution override normal parrotId)
  const effectiveParrotId = useMemo(() => {
    if (sessionSummary?.mode === "geek") return ParrotAgentType.GEEK;
    if (sessionSummary?.mode === "evolution") return ParrotAgentType.EVOLUTION;
    return currentParrotId;
  }, [currentParrotId, sessionSummary?.mode]);

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

            // Determine effective parrot ID for this specific block based on message metadata
            // detailed order:
            // 1. Assistant message metadata mode
            // 2. User message metadata mode
            // 3. Session summary mode (legacy/fallback)
            // 4. Current global parrotId (fallback)
            const blockMode: AIMode | undefined = block.assistantMessage?.metadata?.mode || block.userMessage?.metadata?.mode;

            let blockParrotId = effectiveParrotId;
            if (blockMode === "geek") {
              blockParrotId = ParrotAgentType.GEEK;
            } else if (blockMode === "evolution") {
              blockParrotId = ParrotAgentType.EVOLUTION;
            } else if (blockMode === "normal") {
              blockParrotId = ParrotAgentType.AMAZING;
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

      {/* Generative UI Tools */}
      {uiTools && uiTools.length > 0 && onUIAction && onUIDismiss && (
        <div className="max-w-3xl lg:max-w-4xl xl:max-w-5xl 2xl:max-w-6xl mx-auto mt-3">
          <GenerativeUIContainer tools={uiTools} onAction={onUIAction} onDismiss={onUIDismiss} />
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
