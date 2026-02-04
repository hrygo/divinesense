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
 * │  Block Header (用户消息 + 时间戳 + 状态)                │ ← 固定显示
 * ├─────────────────────────────────────────────────────────┤
 * │  Block Body (可折叠)                                    │
 * │  ├── ThinkingSection (思考过程)                        │
 * │  ├── ToolCallsSection (工具调用)                        │
 * │  ├── AnswerSection (最终回答)                          │
 * │  └── SummarySection (会话统计)                          │
 * ├─────────────────────────────────────────────────────────┤
 * │  Block Footer (操作栏：折叠/展开/复制/删除)             │ ← 固定显示
 * └─────────────────────────────────────────────────────────┘
 * ```
 *
 * ## 主题适配
 * - Normal: border-zinc-200/300
 * - Geek: border-violet-500/30
 * - Evolution: border-rose-500/30
 *
 * ## 折叠策略
 * - 新 Block（流式中）→ 展开
 * - 最新 Block（刚完成）→ 展开
 * - 历史 Block（非最新）→ 折叠
 */

import { Check, ChevronDown, ChevronUp, Clock, Copy } from "lucide-react";
import { memo, ReactNode, useCallback, useEffect, useMemo, useRef, useState } from "react";
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
 * Block theme configuration - 使用与 PARROT_THEMES 一致的颜色系统
 * GEEK 使用 violet 色系，EVOLUTION 使用 rose 色系
 */
const BLOCK_THEMES: Record<
  ParrotAgentType | "default",
  {
    border: string;
    headerBg: string;
    badgeBg: string;
    badgeText: string;
    ringColor: string;
  }
> = {
  default: {
    border: "border-zinc-200 dark:border-zinc-700",
    headerBg: "bg-zinc-50 dark:bg-zinc-900/50",
    badgeBg: "bg-zinc-100 dark:bg-zinc-800",
    badgeText: "text-zinc-600 dark:text-zinc-400",
    ringColor: "ring-primary/20",
  },
  MEMO: {
    border: "border-slate-200 dark:border-slate-700",
    headerBg: "bg-slate-50 dark:bg-slate-900/50",
    badgeBg: "bg-slate-100 dark:bg-slate-800",
    badgeText: "text-slate-600 dark:text-slate-400",
    ringColor: "ring-slate-500/20",
  },
  SCHEDULE: {
    border: "border-cyan-200 dark:border-cyan-700",
    headerBg: "bg-cyan-50 dark:bg-cyan-900/20",
    badgeBg: "bg-cyan-100 dark:bg-cyan-900/30",
    badgeText: "text-cyan-600 dark:text-cyan-400",
    ringColor: "ring-cyan-500/20",
  },
  AMAZING: {
    border: "border-emerald-200 dark:border-emerald-700",
    headerBg: "bg-emerald-50 dark:bg-emerald-900/20",
    badgeBg: "bg-emerald-100 dark:bg-emerald-900/30",
    badgeText: "text-emerald-600 dark:text-emerald-400",
    ringColor: "ring-emerald-500/20",
  },
  GEEK: {
    border: "border-violet-200 dark:border-violet-700",
    headerBg: "bg-violet-50 dark:bg-violet-900/20",
    badgeBg: "bg-violet-100 dark:bg-violet-900/30",
    badgeText: "text-violet-600 dark:text-violet-400",
    ringColor: "ring-violet-500/20",
  },
  EVOLUTION: {
    border: "border-rose-200 dark:border-rose-700",
    headerBg: "bg-rose-50 dark:bg-rose-900/20",
    badgeBg: "bg-rose-100 dark:bg-rose-900/30",
    badgeText: "text-rose-600 dark:text-rose-400",
    ringColor: "ring-rose-500/20",
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
  if (diffMins < 1440) return t("ai.aichat.sidebar.time-hours-ago", { count: Math.floor(diffMins / 60) });
  return date.toLocaleDateString(undefined, { month: "short", day: "numeric" });
}

/**
 * Extract user initial from content for avatar
 */
function extractUserInitial(content: string): string {
  const trimmed = content.trim();
  if (trimmed.length === 0) return "U";
  const match = trimmed.match(/[a-zA-Z\u4e00-\u9fa5]/);
  return match ? match[0].toUpperCase() : "U";
}

/**
 * Determine default collapse state based on block status
 */
function getDefaultCollapseState(isLatest: boolean, isStreaming: boolean): boolean {
  if (isStreaming || isLatest) return false;
  return true;
}

// ============================================================================
// Sub-Components
// ============================================================================

interface BlockHeaderProps {
  userMessage: ConversationMessage;
  parrotId?: ParrotAgentType;
  theme: (typeof BLOCK_THEMES)[keyof typeof BLOCK_THEMES];
}

function BlockHeader({ userMessage, parrotId, theme }: BlockHeaderProps) {
  const { t } = useTranslation();
  const userInitial = extractUserInitial(userMessage.content);

  return (
    <div className={cn("flex items-center justify-between px-4 py-2.5 select-none", theme.headerBg)}>
      {/* Left: User message preview */}
      <div className="flex items-center gap-3 flex-1 min-w-0">
        {/* User avatar with initial */}
        <div className="w-7 h-7 rounded-full bg-slate-800 dark:bg-slate-300 flex items-center justify-center text-white dark:text-slate-800 text-xs font-medium shrink-0">
          {userInitial}
        </div>
        {/* Message preview */}
        <div className="flex-1 min-w-0">
          <p className="text-sm font-medium text-foreground truncate">
            {userMessage.content.slice(0, 60)}
            {userMessage.content.length > 60 ? "..." : ""}
          </p>
        </div>
      </div>

      {/* Right: Timestamp + Status Badge */}
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

  const hasThinking = assistantMessage?.metadata?.thinking;
  const hasToolCalls = assistantMessage?.metadata?.toolCalls && assistantMessage.metadata.toolCalls.length > 0;
  const hasToolResults = assistantMessage?.metadata?.toolResults && assistantMessage.metadata.toolResults.length > 0;
  const hasAnswer = assistantMessage?.content;

  // When collapsed, render minimal placeholder
  if (isCollapsed) {
    return <div className="px-4 py-2 text-sm text-muted-foreground italic">{t("ai.collapsed") || "Click expand to view details"}</div>;
  }

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
  onToggle: () => void;
  onCopy: () => void;
  onRegenerate?: () => void;
  onDelete?: () => void;
  theme: (typeof BLOCK_THEMES)[keyof typeof BLOCK_THEMES];
}

function BlockFooter({ isCollapsed, onToggle, onCopy, onRegenerate, onDelete, theme }: BlockFooterProps) {
  const { t } = useTranslation();
  const [copied, setCopied] = useState(false);
  const timeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const handleCopy = useCallback(() => {
    onCopy();
    setCopied(true);

    if (timeoutRef.current) {
      clearTimeout(timeoutRef.current);
    }

    timeoutRef.current = setTimeout(() => {
      setCopied(false);
      timeoutRef.current = null;
    }, 2000);
  }, [onCopy]);

  useEffect(() => {
    return () => {
      if (timeoutRef.current) {
        clearTimeout(timeoutRef.current);
      }
    };
  }, []);

  return (
    <div className={cn("flex items-center justify-between px-4 py-2 border-t", theme.border, theme.headerBg)}>
      {/* Left: Collapse/Expand Toggle */}
      <button
        type="button"
        onClick={onToggle}
        className={cn(
          "flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-medium transition-colors",
          "hover:bg-black/10 dark:hover:bg-white/10",
          theme.badgeText,
        )}
      >
        {isCollapsed ? <ChevronDown className="w-3.5 h-3.5" /> : <ChevronUp className="w-3.5 h-3.5" />}
        {isCollapsed ? t("common.expand") || "Expand" : t("common.collapse") || "Collapse"}
      </button>

      {/* Right: Action Buttons */}
      <div className="flex items-center gap-2">
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
  const blockTheme = (parrotId && BLOCK_THEMES[parrotId]) || BLOCK_THEMES.default;

  const [collapsed, setCollapsed] = useState(() => getDefaultCollapseState(isLatest, isStreaming));

  useEffect(() => {
    setCollapsed(getDefaultCollapseState(isLatest, isStreaming));
  }, [isLatest, isStreaming]);

  const toggleCollapse = useCallback(() => {
    setCollapsed((prev) => !prev);
  }, []);

  const contentForCopy = useMemo(
    () =>
      [`User: ${userMessage.content}`, assistantMessage?.content ? `Assistant: ${assistantMessage.content}` : ""]
        .filter(Boolean)
        .join("\n\n"),
    [userMessage.content, assistantMessage?.content],
  );

  const handleCopy = useCallback(() => {
    onCopy?.(contentForCopy);
  }, [contentForCopy, onCopy]);

  return (
    <div
      className={cn(
        "rounded-lg border overflow-hidden shadow-sm transition-all duration-200",
        blockTheme.border,
        isLatest && `ring-2 ${blockTheme.ringColor}`,
        className,
      )}
    >
      {/* Block Header - 始终显示 */}
      <div className={cn("border-b", blockTheme.border)}>
        <BlockHeader userMessage={userMessage} parrotId={parrotId} theme={blockTheme} />
      </div>

      {/* Block Body - 可折叠内容 */}
      <BlockBody
        assistantMessage={assistantMessage}
        sessionSummary={sessionSummary}
        isCollapsed={collapsed}
        parrotId={parrotId}
        theme={blockTheme}
      >
        {children}
      </BlockBody>

      {/* Block Footer - 始终显示 */}
      <div className={cn("border-t", blockTheme.border)}>
        <BlockFooter
          isCollapsed={collapsed}
          onToggle={toggleCollapse}
          onCopy={handleCopy}
          onRegenerate={onRegenerate}
          onDelete={onDelete}
          theme={blockTheme}
        />
      </div>
    </div>
  );
});

UnifiedMessageBlock.displayName = "UnifiedMessageBlock";

// ============================================================================
// Hook for Block State Management
// ============================================================================

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

  const messageIds = useMemo(() => messages.map((m) => m.id).join(","), [messages]);
  useEffect(() => {
    setBlockStates((prev) => {
      const currentIds = new Set(messages.map((m) => m.id));
      const prevIds = new Set(Object.keys(prev));

      if (currentIds.size === prevIds.size && [...currentIds].every((id) => prevIds.has(id))) {
        let hasChanges = false;
        const updated: Record<string, BlockState> = { ...prev };
        messages.forEach((msg, i) => {
          const isLatest = i === messages.length - 1;
          if (prev[msg.id]?.isLatest !== isLatest) {
            updated[msg.id] = { ...prev[msg.id], isLatest };
            hasChanges = true;
          }
        });
        return hasChanges ? updated : prev;
      }

      const updated: Record<string, BlockState> = {};
      messages.forEach((msg, i) => {
        const isLatest = i === messages.length - 1;
        const existing = prev[msg.id];
        updated[msg.id] = {
          collapsed: existing?.collapsed ?? getDefaultCollapseState(isLatest, false),
          isLatest,
          isStreaming: false,
        };
      });

      return updated;
    });
  }, [messageIds, messages.length]);

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
