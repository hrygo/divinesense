/**
 * UnifiedMessageBlock - Warp Block 风格统一消息容器
 *
 * 将用户输入 + AI 回复 + 工具调用 + 会话统计封装为一个统一的可折叠 Block
 *
 * @component UnifiedMessageBlock
 * @description Warp Block 风格的消息容器，解决 UI 割裂问题
 *
 * ## 架构
 * ```
 * ┌─────────────────────────────────────────────────────────┐
 * │  Block Header (用户消息 + 时间戳 + 状态)                │
 * ├─────────────────────────────────────────────────────────┤
 * │  Block Body (可折叠)                                    │
 * │  ├── ThinkingSection (思考过程)                        │
 * │  ├── ToolCallsSection (工具调用)                        │
 * │  ├── AnswerSection (最终回答)                          │
 * │  └── SummarySection (会话统计)                          │
 * ├─────────────────────────────────────────────────────────┤
 * │  Block Footer (操作栏)                                  │
 * └─────────────────────────────────────────────────────────┘
 * ```
 *
 * ## 主题适配
 * - Normal: border-zinc-200/300
 * - Geek: border-green-500/30
 * - Evolution: border-purple-500/30
 *
 * ## 折叠策略
 * - 新 Block（流式中）→ 展开
 * - 最新 Block（刚完成）→ 展开
 * - 历史 Block（非最新）→ 折叠
 */

import { Check, ChevronDown, ChevronUp, Clock, Copy } from "lucide-react";
import { memo, ReactNode, useCallback, useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { cn } from "@/lib/utils";
import { ConversationMessage } from "@/types/aichat";
import { PARROT_THEMES, ParrotAgentType, SessionSummary } from "@/types/parrot";
import { SessionSummaryPanel } from "./SessionSummaryPanel";
import { ToolCallCard } from "./ToolCallCard";

// ============================================================================
// Types
// ============================================================================

/**
 * Block state management
 */
export interface BlockState {
  collapsed: boolean;
  isLatest: boolean;
  isStreaming: boolean;
}

/**
 * UnifiedMessageBlock props
 */
export interface UnifiedMessageBlockProps {
  /** User message that triggered this block */
  userMessage: ConversationMessage;
  /** Assistant message (may be streaming) */
  assistantMessage?: ConversationMessage;
  /** Session summary for Geek/Evolution modes */
  sessionSummary?: SessionSummary;
  /** Current parrot agent type */
  parrotId?: ParrotAgentType;
  /** Whether this is the latest block */
  isLatest?: boolean;
  /** Whether assistant is currently streaming */
  isStreaming?: boolean;
  /** Actions */
  onCopy?: (content: string) => void;
  onRegenerate?: () => void;
  onDelete?: () => void;
  /** Additional children to render in block body */
  children?: ReactNode;
  className?: string;
}

// ============================================================================
// Theme Configuration
// ============================================================================

/**
 * Block theme configuration - extends PARROT_THEMES with Block-specific styles
 */
const BLOCK_THEMES: Record<
  ParrotAgentType | "default",
  {
    border: string;
    headerBg: string;
    badgeBg: string;
    badgeText: string;
  }
> = {
  default: {
    border: "border-zinc-200 dark:border-zinc-700",
    headerBg: "bg-zinc-50 dark:bg-zinc-900/50",
    badgeBg: "bg-zinc-100 dark:bg-zinc-800",
    badgeText: "text-zinc-600 dark:text-zinc-400",
  },
  MEMO: {
    border: "border-slate-200 dark:border-slate-700",
    headerBg: "bg-slate-50 dark:bg-slate-900/50",
    badgeBg: "bg-slate-100 dark:bg-slate-800",
    badgeText: "text-slate-600 dark:text-slate-400",
  },
  SCHEDULE: {
    border: "border-cyan-200 dark:border-cyan-700",
    headerBg: "bg-cyan-50 dark:bg-cyan-900/20",
    badgeBg: "bg-cyan-100 dark:bg-cyan-900/30",
    badgeText: "text-cyan-600 dark:text-cyan-400",
  },
  AMAZING: {
    border: "border-emerald-200 dark:border-emerald-700",
    headerBg: "bg-emerald-50 dark:bg-emerald-900/20",
    badgeBg: "bg-emerald-100 dark:bg-emerald-900/30",
    badgeText: "text-emerald-600 dark:text-emerald-400",
  },
  GEEK: {
    border: "border-green-500/30 dark:border-green-500/30",
    headerBg: "bg-green-50 dark:bg-green-900/20",
    badgeBg: "bg-green-100 dark:bg-green-900/30",
    badgeText: "text-green-600 dark:text-green-400",
  },
  EVOLUTION: {
    border: "border-purple-500/30 dark:border-purple-500/30",
    headerBg: "bg-purple-50 dark:bg-purple-900/20",
    badgeBg: "bg-purple-100 dark:bg-purple-900/30",
    badgeText: "text-purple-600 dark:text-purple-400",
  },
};

// ============================================================================
// Helper Functions
// ============================================================================

/**
 * Format timestamp to relative time
 */
function formatTime(timestamp: number, t: (key: string, options?: Record<string, unknown>) => string): string {
  const date = new Date(timestamp);
  const now = new Date();
  const diffMs = now.getTime() - date.getTime();
  const diffMins = Math.floor(diffMs / 60000);

  if (diffMins < 1) return t("ai.aichat.sidebar.time-just-now");
  if (diffMins < 60) return t("ai.aichat.sidebar.time-minutes-ago", { count: diffMins });
  if (diffMins < 1440) return t("ai.aichat.sidebar.time-hours-ago", { count: Math.floor(diffMs / 60) });
  return date.toLocaleDateString(undefined, { month: "short", day: "numeric" });
}

/**
 * Determine default collapse state based on block status
 */
function getDefaultCollapseState(isLatest: boolean, isStreaming: boolean): boolean {
  // New or streaming blocks are always expanded
  if (isStreaming || isLatest) return false;
  // Historical blocks are collapsed by default
  return true;
}

// ============================================================================
// Sub-Components
// ============================================================================

interface BlockHeaderProps {
  userMessage: ConversationMessage;
  parrotId?: ParrotAgentType;
  isCollapsed: boolean;
  onToggle: () => void;
  theme: (typeof BLOCK_THEMES)[keyof typeof BLOCK_THEMES];
}

function BlockHeader({ userMessage, parrotId, isCollapsed, onToggle, theme }: BlockHeaderProps) {
  const { t } = useTranslation();

  return (
    <div
      className={cn(
        "flex items-center justify-between px-4 py-2.5 cursor-pointer select-none",
        "hover:bg-black/5 dark:hover:bg-white/5",
        "transition-colors duration-200",
        theme.headerBg,
        "border-b",
        theme.border,
      )}
      onClick={onToggle}
    >
      {/* Left: User message preview */}
      <div className="flex items-center gap-3 flex-1 min-w-0">
        {/* User avatar */}
        <div className="w-7 h-7 rounded-full bg-slate-800 dark:bg-slate-300 flex items-center justify-center text-white dark:text-slate-800 text-xs font-medium shrink-0">
          U
        </div>
        {/* Message preview */}
        <div className="flex-1 min-w-0">
          <p className="text-sm font-medium text-foreground truncate">
            {userMessage.content.slice(0, 60)}
            {userMessage.content.length > 60 ? "..." : ""}
          </p>
        </div>
      </div>

      {/* Right: Timestamp + Toggle */}
      <div className="flex items-center gap-3 shrink-0">
        {/* Timestamp */}
        <div className={cn("flex items-center gap-1 text-xs", theme.badgeText)}>
          <Clock className="w-3 h-3" />
          <span>{formatTime(userMessage.timestamp, t)}</span>
        </div>

        {/* Status badge (for Geek/Evolution) */}
        {(parrotId === "GEEK" || parrotId === "EVOLUTION") && (
          <span className={cn("px-2 py-0.5 rounded-full text-xs font-medium", theme.badgeBg, theme.badgeText)}>
            {parrotId === "GEEK" ? "Geek" : "Evolution"}
          </span>
        )}

        {/* Toggle icon */}
        <button
          type="button"
          className={cn("p-1 rounded transition-colors", "hover:bg-black/10 dark:hover:bg-white/10", theme.badgeText)}
          onClick={(e) => {
            e.stopPropagation();
            onToggle();
          }}
        >
          {isCollapsed ? <ChevronDown className="w-4 h-4" /> : <ChevronUp className="w-4 h-4" />}
        </button>
      </div>
    </div>
  );
}

interface BlockBodyProps {
  assistantMessage?: ConversationMessage;
  sessionSummary?: SessionSummary;
  isCollapsed: boolean;
  parrotId?: ParrotAgentType;
  theme: (typeof BLOCK_THEMES)[keyof typeof BLOCK_THEMES];
  children?: ReactNode;
}

function BlockBody({ assistantMessage, sessionSummary, isCollapsed, parrotId, theme, children }: BlockBodyProps) {
  const { t } = useTranslation();
  const themeColors = PARROT_THEMES[parrotId || "AMAZING"] || PARROT_THEMES.AMAZING;

  if (isCollapsed) {
    return null;
  }

  const hasThinking = assistantMessage?.metadata?.thinking;
  const hasToolCalls = assistantMessage?.metadata?.toolCalls && assistantMessage.metadata.toolCalls.length > 0;
  const hasToolResults = assistantMessage?.metadata?.toolResults && assistantMessage.metadata.toolResults.length > 0;
  const hasAnswer = assistantMessage?.content;

  return (
    <div className="px-4 py-3 space-y-4">
      {/* Thinking Section */}
      {hasThinking && (
        <div className="flex items-start gap-2 text-sm text-muted-foreground">
          <span className="font-medium">💭</span>
          <p className="italic">{assistantMessage.metadata!.thinking}</p>
        </div>
      )}

      {/* Tool Calls Section */}
      {hasToolCalls && (
        <div className="space-y-2">
          <div className={cn("text-xs font-medium uppercase tracking-wide", theme.badgeText)}>{t("ai.events.tools") || "Tools"}</div>
          <div className="space-y-2">
            {assistantMessage.metadata!.toolCalls!.map((call, i) => (
              <ToolCallCard
                key={i}
                data={{
                  toolName: call.name,
                  toolId: call.toolId,
                  input: call.inputSummary ? { command: call.inputSummary } : undefined,
                  isError: call.isError,
                }}
              />
            ))}
          </div>
        </div>
      )}

      {/* Tool Results Section */}
      {hasToolResults && (
        <div className="space-y-2">
          <div className={cn("text-xs font-medium uppercase tracking-wide", theme.badgeText)}>{t("ai.events.results") || "Results"}</div>
          <div className="space-y-2">
            {assistantMessage.metadata!.toolResults!.map((result, i) => (
              <div
                key={i}
                className={cn(
                  "p-3 rounded-lg border",
                  theme.border,
                  result.isError ? "bg-red-50 dark:bg-red-900/20" : "bg-slate-50 dark:bg-slate-900/50",
                )}
              >
                <div className="flex items-center justify-between mb-2">
                  <span className="text-sm font-medium text-foreground">{result.name}</span>
                  {result.duration && <span className="text-xs text-muted-foreground">{result.duration}ms</span>}
                </div>
                {result.outputSummary && (
                  <pre className="text-xs text-muted-foreground whitespace-pre-wrap font-mono">{result.outputSummary}</pre>
                )}
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Answer Section */}
      {hasAnswer && (
        <div className="space-y-2">
          <div className={cn("text-xs font-medium uppercase tracking-wide", theme.badgeText)}>{t("ai.events.answer") || "Answer"}</div>
          <div
            className={cn(
              "p-3 rounded-lg border text-sm leading-relaxed",
              themeColors.bubbleBg,
              themeColors.bubbleBorder,
              themeColors.text,
            )}
          >
            {assistantMessage.content}
          </div>
        </div>
      )}

      {/* Custom children (for typing cursor, etc.) */}
      {children}

      {/* Session Summary Section */}
      {sessionSummary && (
        <div className="pt-2 border-t border-border/50">
          <SessionSummaryPanel summary={sessionSummary} />
        </div>
      )}
    </div>
  );
}

interface BlockFooterProps {
  isCollapsed: boolean;
  onCopy: () => void;
  onRegenerate?: () => void;
  onDelete?: () => void;
  theme: (typeof BLOCK_THEMES)[keyof typeof BLOCK_THEMES];
}

function BlockFooter({ isCollapsed, onCopy, onRegenerate, onDelete, theme }: BlockFooterProps) {
  const { t } = useTranslation();
  const [copied, setCopied] = useState(false);

  const handleCopy = useCallback(() => {
    onCopy();
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  }, [onCopy]);

  if (isCollapsed) {
    return null;
  }

  return (
    <div className={cn("flex items-center justify-end gap-2 px-4 py-2 border-t", theme.border, theme.headerBg)}>
      {onRegenerate && (
        <button
          type="button"
          onClick={onRegenerate}
          className={cn(
            "px-3 py-1.5 rounded-lg text-xs font-medium transition-colors",
            "hover:bg-black/10 dark:hover:bg-white/10",
            theme.badgeText,
          )}
        >
          {t("ai.regenerate") || "Regenerate"}
        </button>
      )}
      <button
        type="button"
        onClick={handleCopy}
        className={cn(
          "flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-medium transition-colors",
          "hover:bg-black/10 dark:hover:bg-white/10",
          copied && "bg-green-100 dark:bg-green-900/30 text-green-600 dark:text-green-400",
          !copied && theme.badgeText,
        )}
      >
        {copied ? <Check className="w-3.5 h-3.5" /> : <Copy className="w-3.5 h-3.5" />}
        {copied ? t("common.copied") || "Copied" : t("common.copy") || "Copy"}
      </button>
      {onDelete && (
        <button
          type="button"
          onClick={onDelete}
          className={cn(
            "flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-medium transition-colors",
            "hover:bg-red-100 dark:hover:bg-red-900/30",
            "text-red-600 dark:text-red-400",
          )}
        >
          {t("common.delete") || "Delete"}
        </button>
      )}
    </div>
  );
}

// ============================================================================
// Main Component
// ============================================================================

/**
 * UnifiedMessageBlock - Warp Block 风格统一消息容器
 *
 * @example
 * ```tsx
 * <UnifiedMessageBlock
 *   userMessage={userMsg}
 *   assistantMessage={assistantMsg}
 *   sessionSummary={summary}
 *   parrotId="GEEK"
 *   isLatest={true}
 *   isStreaming={false}
 *   onCopy={() => navigator.clipboard.writeText(content)}
 *   onRegenerate={() => regenerate()}
 *   onDelete={() => deleteMessage()}
 * />
 * ```
 */
export const UnifiedMessageBlock = memo(function UnifiedMessageBlock({
  userMessage,
  assistantMessage,
  sessionSummary,
  parrotId,
  isLatest = false,
  isStreaming = false,
  onCopy,
  onRegenerate,
  onDelete,
  children,
  className,
}: UnifiedMessageBlockProps) {
  // Get theme for block styling
  const blockTheme = (parrotId && BLOCK_THEMES[parrotId]) || BLOCK_THEMES.default;

  // Collapse state with automatic behavior based on block status
  const [collapsed, setCollapsed] = useState(() => getDefaultCollapseState(isLatest, isStreaming));

  // Update collapse state when isLatest or isStreaming changes
  useEffect(() => {
    setCollapsed(getDefaultCollapseState(isLatest, isStreaming));
  }, [isLatest, isStreaming]);

  const toggleCollapse = useCallback(() => {
    setCollapsed((prev) => !prev);
  }, []);

  // Build content for copying
  const contentForCopy = [`User: ${userMessage.content}`, assistantMessage?.content ? `Assistant: ${assistantMessage.content}` : ""]
    .filter(Boolean)
    .join("\n\n");

  const handleCopy = useCallback(() => {
    onCopy?.(contentForCopy);
  }, [contentForCopy, onCopy]);

  return (
    <div
      className={cn(
        "rounded-lg border overflow-hidden shadow-sm transition-all duration-200",
        blockTheme.border,
        isLatest && "ring-2 ring-primary/20",
        className,
      )}
    >
      {/* Block Header */}
      <BlockHeader userMessage={userMessage} parrotId={parrotId} isCollapsed={collapsed} onToggle={toggleCollapse} theme={blockTheme} />

      {/* Block Body (collapsible) */}
      <BlockBody
        assistantMessage={assistantMessage}
        sessionSummary={sessionSummary}
        isCollapsed={collapsed}
        parrotId={parrotId}
        theme={blockTheme}
      >
        {children}
      </BlockBody>

      {/* Block Footer */}
      <BlockFooter isCollapsed={collapsed} onCopy={handleCopy} onRegenerate={onRegenerate} onDelete={onDelete} theme={blockTheme} />
    </div>
  );
});

// ============================================================================
// Hook for Block State Management
// ============================================================================

/**
 * useBlockState - Manages collapse state for multiple message blocks
 *
 * @example
 * ```tsx
 * const blockStates = useBlockState(messages);
 *
 * {messages.map((msg, i) => (
 *   <UnifiedMessageBlock
 *     key={msg.id}
 *     isLatest={i === messages.length - 1}
 *     isCollapsed={blockStates.get(msg.id)?.collapsed ?? false}
 *     ...
 *   />
 * ))}
 * ```
 */
export function useBlockState(messages: ConversationMessage[]) {
  const [blockStates, setBlockStates] = useState<Record<string, BlockState>>(() => {
    const initial: Record<string, BlockState> = {};
    messages.forEach((msg, i) => {
      const isLatest = i === messages.length - 1;
      initial[msg.id] = {
        collapsed: getDefaultCollapseState(isLatest, false),
        isLatest,
        isStreaming: false,
      };
    });
    return initial;
  });

  const updateBlockState = useCallback((messageId: string, updates: Partial<BlockState>) => {
    setBlockStates((prev) => ({
      ...prev,
      [messageId]: { ...prev[messageId], ...updates },
    }));
  }, []);

  const toggleBlock = useCallback((messageId: string) => {
    setBlockStates((prev) => ({
      ...prev,
      [messageId]: { ...prev[messageId], collapsed: !prev[messageId]?.collapsed },
    }));
  }, []);

  return {
    blockStates,
    updateBlockState,
    toggleBlock,
  };
}

export default UnifiedMessageBlock;
