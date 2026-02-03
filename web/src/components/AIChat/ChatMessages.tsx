import { Check, ChevronDown, ChevronUp, Copy, FileIcon, Scissors, Terminal as TerminalIcon } from "lucide-react";
import React, { memo, ReactNode, useCallback, useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import ReactMarkdown from "react-markdown";
import remarkBreaks from "remark-breaks";
import remarkGfm from "remark-gfm";
import { AnimatedAvatar } from "@/components/AIChat/AnimatedAvatar";
import { ExpandedSessionSummary } from "@/components/AIChat/ExpandedSessionSummary";
import MessageActions from "@/components/AIChat/MessageActions";
import { InlineToolCall } from "@/components/AIChat/ToolCallCard";
import TypingCursor from "@/components/AIChat/TypingCursor";
import { CodeBlock } from "@/components/MemoContent/CodeBlock";
import { GenerativeUIContainer } from "@/components/ScheduleAI/GenerativeUIContainer";
import type { GenerativeUIContainerProps } from "@/components/ScheduleAI/types";
import { cn } from "@/lib/utils";
import { ChatItem, ConversationMessage } from "@/types/aichat";
import type { SessionSummary } from "@/types/parrot";
import { PARROT_ICONS, PARROT_THEMES, ParrotAgentType } from "@/types/parrot";

type CodeComponentProps = React.ComponentProps<"code"> & { inline?: boolean };

// Tool call type for legacy string format and new object format
type ToolCall =
  | string
  | {
      name: string;
      toolId?: string;
      inputSummary?: string;
      outputSummary?: string;
      filePath?: string;
      duration?: number;
      isError?: boolean;
    };

// Tool result type from metadata
type ToolResult = {
  name: string;
  toolId?: string;
  inputSummary?: string;
  outputSummary?: string;
  duration?: number;
  isError?: boolean;
};

// Format message timestamp to relative time (e.g., "刚刚", "5分钟前")
// 格式化消息时间戳为相对时间（如"刚刚"、"5分钟前"）
function formatMessageTime(timestamp: number, t: (key: string, options?: Record<string, unknown>) => string): string {
  const date = new Date(timestamp);
  const now = new Date();
  const diffMs = now.getTime() - date.getTime();
  const diffMins = Math.floor(diffMs / 60000);
  const diffHours = Math.floor(diffMs / 3600000);
  const diffDays = Math.floor(diffMs / 86400000);

  if (diffMins < 1) return t("ai.aichat.sidebar.time-just-now");
  if (diffMins < 60) return t("ai.aichat.sidebar.time-minutes-ago", { count: diffMins });
  if (diffHours < 24) return t("ai.aichat.sidebar.time-hours-ago", { count: diffHours });
  if (diffDays < 7) return t("ai.aichat.sidebar.time-days-ago", { count: diffDays });

  // Show full date for messages older than a week
  // 超过一周显示完整日期
  return date.toLocaleDateString(undefined, { month: "short", day: "numeric", hour: "2-digit", minute: "2-digit" });
}

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
  const { t } = useTranslation();
  const scrollRef = useRef<HTMLDivElement>(null);
  const endRef = useRef<HTMLDivElement>(null);
  const rafIdRef = useRef<ReturnType<typeof requestAnimationFrame> | null>(null);
  const lastScrollTimeRef = useRef(0);
  const isUserScrollingRef = useRef(false);
  // Phase 1: 追踪流式内容长度，优化滚动触发
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

  // Phase 1: 优化的 MutationObserver - 仅监听子节点变化，忽略内容更新
  useEffect(() => {
    if (!scrollRef.current) return;

    const observer = new MutationObserver((mutations) => {
      // 仅在新增节点时触发滚动，忽略文本内容变化
      const hasNewNodes = mutations.some((m) => m.type === "childList" && m.addedNodes.length > 0);

      if (hasNewNodes && !isUserScrollingRef.current) {
        scrollToBottomLocked();
      }
    });

    const contentElement = scrollRef.current.firstElementChild;
    if (contentElement) {
      observer.observe(contentElement, {
        childList: true, // 仅监听子节点变化
        subtree: true, // 监听所有后代
      });
    }

    return () => observer.disconnect();
  }, [scrollToBottomLocked]);

  // Phase 1: 优化的消息数量变化监听
  const prevItemsLengthRef = useRef(items.length);
  useEffect(() => {
    const itemsLength = items.length;
    const hasNewMessage = itemsLength > prevItemsLengthRef.current;
    prevItemsLengthRef.current = itemsLength;

    if (hasNewMessage && !isUserScrollingRef.current) {
      scrollToBottomLocked();
    }
  }, [items.length, scrollToBottomLocked]);

  // Phase 2: 流式内容变化监听 - 仅在内容显著增加时滚动
  useEffect(() => {
    if (!isStreaming) return;

    const contentLength = streamingContent.length;
    const contentIncrease = contentLength - lastContentLengthRef.current;
    lastContentLengthRef.current = contentLength;

    // 每增加约 50 字符滚动一次，减少频繁操作
    if (contentIncrease > 50 && !isUserScrollingRef.current) {
      scrollToBottomLocked();
    }
  }, [streamingContent, isStreaming, scrollToBottomLocked]);

  useEffect(() => {
    if (isTyping && !isUserScrollingRef.current) {
      scrollToBottomLocked();
    }
  }, [isTyping, scrollToBottomLocked]);

  // Reset user scrolling flag when items change AND we're near bottom
  // Only depend on items.length to avoid triggering on every items reference change
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

  const theme = currentParrotId ? PARROT_THEMES[currentParrotId] || PARROT_THEMES.AMAZING : PARROT_THEMES.AMAZING;
  const currentIcon = currentParrotId ? PARROT_ICONS[currentParrotId] || PARROT_ICONS.AMAZING : PARROT_ICONS.AMAZING;

  return (
    <div
      ref={scrollRef}
      onScroll={handleScrollThrottled}
      className={cn("flex-1 overflow-y-auto px-3 md:px-6 py-4 overscroll-contain", className)}
      style={{ overflowAnchor: "auto", scrollbarGutter: "stable", contain: "layout style paint" }}
    >
      {children}

      {items.length > 0 && (
        <div className="max-w-3xl lg:max-w-4xl xl:max-w-5xl 2xl:max-w-6xl mx-auto space-y-4">
          {items.map((item, index) => {
            // Context separator - optimized visual design
            if ("type" in item && item.type === "context-separator") {
              return (
                <div
                  key={`separator-${index}`}
                  className="flex items-center justify-center gap-3 py-3 my-2 animate-in fade-in slide-in-from-top-2 duration-300"
                >
                  <div className="flex-1 h-px bg-gradient-to-r from-transparent via-border to-transparent" />
                  <div className="flex items-center gap-2 px-3 py-1.5 rounded-full bg-muted border border-border shadow-sm">
                    <Scissors className="w-3.5 h-3.5 text-muted-foreground rotate-[-45deg]" />
                    <span className="text-xs text-muted-foreground font-medium whitespace-nowrap">{t("ai.context-cleared")}</span>
                  </div>
                  <div className="flex-1 h-px bg-gradient-to-r from-transparent via-border to-transparent" />
                </div>
              );
            }

            const msg = item as ConversationMessage;
            const isLastMessage = index === items.length - 1;
            const isNew = Date.now() - msg.timestamp < 1000; // Animation for recent messages

            // WeChat-style timestamp: show when there's a gap of > 3 minutes from previous message
            // 微信风格时间戳：与上一条消息间隔超过 3 分钟时显示
            const shouldShowTimestamp =
              index === 0 ||
              (() => {
                const prevItem = items[index - 1];
                if ("type" in prevItem && prevItem.type === "context-separator") return true;
                const prevMsg = prevItem as ConversationMessage;
                return msg.timestamp - prevMsg.timestamp > 3 * 60 * 1000; // 3 minutes
              })();

            return (
              <React.Fragment key={msg.id}>
                {/* WeChat-style centered timestamp */}
                {/* 微信风格居中时间戳 */}
                {shouldShowTimestamp && (
                  <div className="flex items-center justify-center py-2">
                    <span className="text-xs text-muted-foreground/60 bg-muted/50 px-2.5 py-1 rounded">
                      {formatMessageTime(msg.timestamp, t)}
                    </span>
                  </div>
                )}

                <MessageBubble
                  message={msg}
                  theme={theme}
                  icon={msg.role === "user" ? undefined : currentIcon}
                  isLastAssistant={msg.role === "assistant" && isLastMessage}
                  isNew={isNew}
                  isTyping={isTyping}
                  onCopy={() => onCopyMessage?.(msg.content)}
                  onRegenerate={onRegenerate}
                  onDelete={() => onDeleteMessage?.(index)}
                >
                  {msg.role === "assistant" && isTyping && isLastMessage && !msg.error && (
                    <TypingCursor active={true} parrotId={currentParrotId} variant="dots" />
                  )}
                </MessageBubble>
              </React.Fragment>
            );
          })}

          {/* Amazing Insight Card - rendered in message flow with exact same alignment as assistant messages */}
          {amazingInsightCard && !isTyping && items.length > 0 && (
            <div className="flex gap-3 md:gap-4 animate-in fade-in duration-300">
              {/* Spacer for avatar alignment */}
              <div className="w-9 h-9 md:w-10 md:h-10 shrink-0 invisible" />
              <div className="flex-1 min-w-0">
                <div className="max-w-[85%] md:max-w-[80%]">{amazingInsightCard}</div>
              </div>
            </div>
          )}

          {/* Generative UI Tools - embedded in message flow like assistant messages */}
          {uiTools && uiTools.length > 0 && onUIAction && onUIDismiss && (
            <div className="flex gap-3 md:gap-4 animate-in fade-in duration-300">
              {/* Spacer for avatar alignment */}
              <div className="w-9 h-9 md:w-10 md:h-10 shrink-0 invisible" />
              <div className="flex-1 min-w-0">
                <div className="max-w-[85%] md:max-w-[80%]">
                  <GenerativeUIContainer tools={uiTools} onAction={onUIAction} onDismiss={onUIDismiss} />
                </div>
              </div>
            </div>
          )}

          {/* Session Summary Panel - for Geek/Evolution modes */}
          {sessionSummary && !isTyping && (
            <div className="flex gap-3 md:gap-4 animate-in fade-in duration-300">
              {/* Spacer for avatar alignment */}
              <div className="w-9 h-9 md:w-10 md:h-10 shrink-0 invisible" />
              <div className="flex-1 min-w-0">
                <ExpandedSessionSummary summary={sessionSummary} />
              </div>
            </div>
          )}

          {/* Typing indicator - AI Native design */}
          {isTyping &&
            (() => {
              const lastItem = items[items.length - 1];
              if (!lastItem) return true;
              if ("type" in lastItem && lastItem.type === "context-separator") return true;
              return "role" in lastItem && lastItem.role !== "assistant";
            })() && (
              <div className="flex gap-3 md:gap-4 animate-in fade-in duration-300">
                <div className="w-9 h-9 md:w-10 md:h-10 rounded-full flex items-center justify-center shadow-sm">
                  {currentIcon.startsWith("/") ? (
                    <img src={currentIcon} alt="" className="w-8 h-8 md:w-9 md:h-9 object-contain" />
                  ) : (
                    <span className="text-lg md:text-xl">{currentIcon}</span>
                  )}
                </div>
                <div className={cn("px-4 py-3 rounded-2xl border shadow-sm", theme.bubbleBg, theme.bubbleBorder)}>
                  <TypingCursor active={true} parrotId={currentParrotId} variant="dots" />
                </div>
              </div>
            )}
          {/* Scroll anchor */}
          <div ref={endRef} className="h-px" />
        </div>
      )}
    </div>
  );
});

export { ChatMessages };

interface MessageBubbleProps {
  message: ConversationMessage;
  theme: (typeof PARROT_THEMES)[keyof typeof PARROT_THEMES];
  icon?: string;
  isLastAssistant?: boolean;
  isNew?: boolean;
  isTyping?: boolean;
  onCopy?: () => void;
  onRegenerate?: () => void;
  onDelete?: () => void;
  children?: ReactNode;
}

const MAX_MESSAGE_HEIGHT = 200;

const MessageBubble = memo(function MessageBubble({
  message,
  theme,
  icon,
  isLastAssistant = false,
  isNew = false,
  isTyping = false,
  onCopy,
  onRegenerate,
  onDelete,
  children,
}: MessageBubbleProps) {
  const { role, content, error } = message;
  const contentRef = useRef<HTMLDivElement>(null);
  const [isFolded, setIsFolded] = useState(false);
  const [shouldShowFold, setShouldShowFold] = useState(false);
  const [copied, setCopied] = useState(false);
  const { t } = useTranslation();

  // Detect height for auto-folding
  useEffect(() => {
    if (contentRef.current) {
      const height = contentRef.current.scrollHeight;
      if (height > MAX_MESSAGE_HEIGHT) {
        setShouldShowFold(true);
      } else {
        setShouldShowFold(false);
      }
    }
  }, [content, children]);

  const toggleFold = useCallback(() => {
    setIsFolded((prev) => !prev);
  }, []);

  const handleCopy = useCallback(() => {
    if (onCopy) {
      onCopy();
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    }
  }, [onCopy]);

  return (
    <div
      className={cn(
        "flex gap-3 md:gap-4 group/row",
        role === "user" ? "flex-row-reverse" : "flex-row",
        isNew && "animate-in fade-in duration-300",
      )}
      style={{ contain: "layout style" }}
    >
      {/* Avatar */}
      {role === "user" ? (
        <AnimatedAvatar src="/user-avatar.webp" alt="User" size="md" />
      ) : icon?.startsWith("/") ? (
        <AnimatedAvatar src={icon} alt="" size="md" isThinking={isTyping && isLastAssistant} isTyping={isTyping && isLastAssistant} />
      ) : (
        <div className="w-9 h-9 md:w-10 md:h-10 rounded-full flex items-center justify-center shrink-0 shadow-sm bg-muted">
          <span className="text-lg md:text-xl">{icon || "🤖"}</span>
        </div>
      )}

      {/* Message content area */}
      <div className="flex-1 min-w-0 flex flex-col gap-1">
        {/* Assistant Actions Header */}
        {role === "assistant" && isLastAssistant && onRegenerate && onDelete && (
          <div className="flex items-center gap-2 mb-0.5 opacity-0 group-row:opacity-100 transition-opacity">
            <MessageActions onRegenerate={onRegenerate} onDelete={onDelete} />
          </div>
        )}

        {/* Event Badge for tool calls - support both single toolName and multiple toolCalls */}
        {role === "assistant" && (message.metadata?.toolName || message.metadata?.toolCalls) && (
          <div className="flex flex-wrap gap-2 mb-2">
            {message.metadata.toolCalls
              ? message.metadata.toolCalls.map((call: ToolCall, i: number) => (
                  <InlineToolCall
                    key={i}
                    toolName={typeof call === "string" ? call : call.name}
                    inputSummary={typeof call === "object" ? call.inputSummary : undefined}
                    filePath={typeof call === "object" ? call.filePath : undefined}
                    isError={typeof call === "object" ? call.isError : message.error}
                  />
                ))
              : message.metadata.toolName && <InlineToolCall toolName={message.metadata.toolName} isError={message.error} />}
          </div>
        )}

        {/* Tool Results Display - expanded cards showing output */}
        {role === "assistant" && message.metadata?.toolResults && message.metadata.toolResults.length > 0 && (
          <div className="space-y-2 mb-3">
            {message.metadata.toolResults.map((result: ToolResult, i: number) => {
              // Only show results that have actual output
              if (!result.outputSummary || result.outputSummary.length === 0) return null;

              return (
                <div key={i} className="relative">
                  {/* Tool result indicator line */}
                  <div className="absolute left-4 top-0 bottom-0 w-px bg-slate-200 dark:bg-slate-700" />
                  <div className="pl-8">
                    <div className="flex items-center gap-2 mb-1">
                      <div className="p-1 rounded bg-green-100 dark:bg-green-900/30 text-green-600 dark:text-green-400">
                        {result.name.toLowerCase().includes("bash") || result.name.toLowerCase().includes("run") ? (
                          <TerminalIcon className="w-3 h-3" />
                        ) : (
                          <FileIcon className="w-3 h-3" />
                        )}
                      </div>
                      <span className="text-xs font-medium text-slate-600 dark:text-slate-400">{result.name}</span>
                      {result.duration && (
                        <span className="text-xs text-slate-400">
                          {result.duration > 1000 ? `${(result.duration / 1000).toFixed(1)}s` : `${result.duration}ms`}
                        </span>
                      )}
                      {result.isError && <span className="text-xs text-red-500 font-medium">Error</span>}
                    </div>
                    {/* Output terminal */}
                    <div className="rounded-lg bg-slate-950 dark:bg-slate-950 border border-slate-800 dark:border-slate-800 overflow-hidden">
                      <div className="px-3 py-1.5 bg-slate-900/50 border-b border-slate-800 flex items-center gap-2">
                        <div className="flex gap-1.5">
                          <div className="w-2.5 h-2.5 rounded-full bg-red-500/80" />
                          <div className="w-2.5 h-2.5 rounded-full bg-yellow-500/80" />
                          <div className="w-2.5 h-2.5 rounded-full bg-green-500/80" />
                        </div>
                        <span className="text-xs text-slate-500 font-mono">Output</span>
                      </div>
                      <pre className="p-3 text-xs text-slate-300 font-mono overflow-x-auto whitespace-pre-wrap break-words max-h-48 overflow-y-auto">
                        {result.outputSummary}
                      </pre>
                    </div>
                  </div>
                </div>
              );
            })}
          </div>
        )}

        <div className={cn("flex items-start gap-2", role === "user" ? "flex-row-reverse" : "flex-row")}>
          {error ? (
            <div className="min-w-[120px] max-w-[85%] md:max-w-[80%] p-3 rounded-xl bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 shadow-sm">
              <p className="text-sm text-red-700 dark:text-red-300">{content}</p>
            </div>
          ) : (
            <div
              className={cn(
                "relative rounded-2xl shadow-sm transition-colors group/bubble min-w-[120px] max-w-[85%] md:max-w-[80%]",
                role === "user" ? theme.bubbleUser : cn(theme.bubbleBg, theme.bubbleBorder, theme.text),
                shouldShowFold && isFolded ? "overflow-hidden" : "max-h-none",
              )}
              style={shouldShowFold && isFolded ? { maxHeight: `${MAX_MESSAGE_HEIGHT}px` } : {}}
            >
              {/* Floating Copy Button - Internal Top Right */}
              {!error && (
                <div className="absolute top-2 right-2 z-30">
                  <button
                    onClick={handleCopy}
                    className={cn(
                      "p-1.5 rounded-lg border shadow-sm transition-all active:scale-90",
                      role === "user"
                        ? "bg-white/10 border-white/20 text-white/80 hover:bg-white/30"
                        : "bg-card/50 border-border text-muted-foreground hover:text-foreground backdrop-blur-sm",
                      copied &&
                        (role === "user"
                          ? "bg-white/40 border-white/40"
                          : "bg-green-50 dark:bg-green-900/20 border-green-200 text-green-600"),
                    )}
                  >
                    {copied ? <Check className="w-3.5 h-3.5" /> : <Copy className="w-3.5 h-3.5" />}
                  </button>
                </div>
              )}

              {/* Content and Markdown */}
              <div ref={contentRef} className="pl-4 pr-10 py-2.5">
                {role === "assistant" ? (
                  <div className="prose prose-sm dark:prose-invert max-w-none break-words text-sm font-normal font-sans">
                    <ReactMarkdown
                      remarkPlugins={[remarkGfm, remarkBreaks]}
                      components={{
                        a: ({ node, ...props }) => (
                          <a {...props} className="text-blue-500 hover:underline" target="_blank" rel="noopener noreferrer" />
                        ),
                        p: ({ node, ...props }) => <p {...props} className="mb-1 last:mb-0 text-sm leading-relaxed" />,
                        ul: ({ node, ...props }) => <ul {...props} className="list-disc pl-5 mb-2 space-y-1" />,
                        ol: ({ node, ...props }) => <ol {...props} className="list-decimal pl-5 mb-2 space-y-1" />,
                        li: ({ node, ...props }) => <li {...props} className="pl-1" />,
                        h1: ({ node, ...props }) => <h1 {...props} className="text-xl font-bold mb-2 mt-4 first:mt-0" />,
                        h2: ({ node, ...props }) => <h2 {...props} className="text-lg font-bold mb-2 mt-3" />,
                        h3: ({ node, ...props }) => <h3 {...props} className="text-base font-bold mb-1 mt-2" />,
                        blockquote: ({ node, ...props }) => (
                          <blockquote {...props} className="border-l-4 border-primary/30 pl-4 py-1 my-2 bg-muted/30 italic rounded-r-lg" />
                        ),
                        table: ({ node, ...props }) => (
                          <div className="my-4 w-full overflow-x-auto rounded-lg border border-border shadow-sm">
                            <table className="w-full text-sm" {...props} />
                          </div>
                        ),
                        thead: ({ node, ...props }) => <thead className="bg-muted/50 text-xs uppercase" {...props} />,
                        tbody: ({ node, ...props }) => <tbody className="divide-y divide-border" {...props} />,
                        tr: ({ node, ...props }) => <tr className="hover:bg-muted/50 transition-colors" {...props} />,
                        th: ({ node, ...props }) => (
                          <th className="px-4 py-2.5 text-left font-medium text-muted-foreground tracking-wider" {...props} />
                        ),
                        td: ({ node, ...props }) => <td className="px-4 py-2.5 whitespace-pre-wrap" {...props} />,
                        pre: ({ node, ...props }) => <CodeBlock {...props} hideCopy={true} />,
                        code: ({ className, children, inline, ...props }: CodeComponentProps) => {
                          return inline ? (
                            <code
                              className={cn("px-1.5 py-0.5 rounded-md bg-muted text-xs break-all whitespace-pre-wrap", className)}
                              {...props}
                            >
                              {children}
                            </code>
                          ) : (
                            <code className={className} {...props}>
                              {children}
                            </code>
                          );
                        },
                      }}
                    >
                      {content || t("ai.states.thinking") || "..."}
                    </ReactMarkdown>
                    {children}
                  </div>
                ) : (
                  <div className="whitespace-pre-wrap break-words text-sm font-sans">{content}</div>
                )}
              </div>

              {/* Fold Mask and Button */}
              {shouldShowFold && (
                <>
                  {isFolded && (
                    <div className="absolute inset-x-0 bottom-0 h-16 bg-gradient-to-t from-card via-card/40 to-transparent pointer-events-none" />
                  )}
                  <div className={cn("flex justify-center p-1.5", isFolded ? "absolute bottom-0 inset-x-0 z-10" : "relative")}>
                    <button
                      onClick={toggleFold}
                      className="flex items-center gap-1 px-2.5 py-1 rounded-full text-[10px] font-bold uppercase bg-card border border-border shadow-sm hover:bg-accent text-muted-foreground"
                    >
                      {isFolded ? (
                        <>
                          <ChevronDown className="w-3 h-3" />
                          {t("common.expand")}
                        </>
                      ) : (
                        <>
                          <ChevronUp className="w-3 h-3" />
                          {t("common.collapse")}
                        </>
                      )}
                    </button>
                  </div>
                </>
              )}
            </div>
          )}
        </div>
      </div>
    </div>
  );
});
