/**
 * UnifiedMessageBlock - Warp Block 风格统一消息容器
 *
 * 将用户输入 + AI 回复 + 工具调用 + 会话统计封装为一个统一的可折叠 Block
 *
 * ## 架构
 * ```
 * ┌─────────────────────────────────────────────────────────┐
 * │  Block Header (用户消息 + 时间戳 + 状态)                │ ← 固定显示
 * ├─────────────────────────────────────────────────────────┤
 * │  Block Body (可折叠)                                    │
 * │  ├── ThinkingSection (思考过程)                        │
 * │  ├── ToolCallsSection (工具调用徽章)                    │
 * │  ├── ToolResultsSection (终端风格输出)                │
 * │  ├── AnswerSection (Markdown渲染 + 代码高亮)        │
 * │  └── SummarySection (会话统计)                          │
 * ├─────────────────────────────────────────────────────────┤
 * │  Block Footer (操作栏：折叠/展开/复制/删除)             │ ← 固定显示
 * └─────────────────────────────────────────────────────────┘
 * ```
 */

/**
 * UnifiedMessageBlock - Warp Block 风格统一消息容器
 *
 * 将用户输入 + AI 回复 + 工具调用 + 会话统计封装为一个统一的可折叠 Block
 *
 * ## 架构
 * ```
 * ┌─────────────────────────────────────────────────────────┐
 * │  Block Header (用户消息 + 时间戳 + 状态)                │ ← 固定显示
 * ├─────────────────────────────────────────────────────────┤
 * │  Block Body (可折叠)                                    │
 * │  ├── ThinkingSection (思考过程)                        │
 * │  ├── ToolCallsSection (工具调用徽章)                    │
 * │  ├── ToolResultsSection (终端风格输出)                │
 * │  ├── AnswerSection (Markdown渲染 + 代码高亮)        │
 * │  └── SummarySection (会话统计)                          │
 * ├─────────────────────────────────────────────────────────┤
 * │  Block Footer (操作栏：折叠/展开/复制/删除)             │ ← 固定显示
 * └─────────────────────────────────────────────────────────┘
 * ```
 *
 * ## UI/UX 优化 (Issue #104)
 *
 * 已集成模块化组件：
 * - TimelineNode: 统一时间线节点样式
 * - StreamingProgressBar: 流式进度条
 * - PendingSkeleton: 骨架屏加载状态
 * - ToolCallCard: 优化的工具卡片（hover 高亮）
 * - BlockHeader: 响应式两栏布局
 * - BlockFooter: 图标优先响应式设计
 * - useBlockCollapse: 折叠状态下沉 hook
 * - useStreamingProgress: 流式进度计算
 */

import { AlertCircle, BarChart3, ChevronDown, ChevronRight, ChevronUp, Loader2 } from "lucide-react";
import { memo, ReactNode, useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import ReactMarkdown from "react-markdown";
import { ROUND_TIMESTAMP_MULTIPLIER, TOOL_CALL_OFFSET_US, USER_INPUTS_EXPAND_THRESHOLD } from "@/components/AIChat/constants";
import { ExpandedSessionSummary } from "@/components/AIChat/ExpandedSessionSummary";
import StreamingMarkdown from "@/components/AIChat/StreamingMarkdown";
import { cn } from "@/lib/utils";
import { type ConversationMessage } from "@/types/aichat";
import { BlockSummary, PARROT_THEMES, ParrotAgentType } from "@/types/parrot";

// Simple hash function for generating stable storage keys from strings.
// Uses DJB2 algorithm variant for good distribution.
function simpleHash(str: string): string {
  let hash = 0;
  for (let i = 0; i < str.length; i++) {
    const char = str.charCodeAt(i);
    hash = (hash << 5) - hash + char;
    hash = hash & hash; // Convert to 32bit integer
  }
  return Math.abs(hash).toString(36);
}

// UI/UX 优化模块组件
import {
  type BlockFooterProps,
  type BlockFooterTheme,
  type BlockHeaderProps,
  type BlockHeaderTheme,
  BlockFooter as ModularBlockFooter,
  BlockHeader as ModularBlockHeader,
  TimelineNode,
} from "./UnifiedMessageBlock/components";

/**
 * Extract pure tool name from function call string
 *
 * Handles various formats:
 * - "search_files(query=\"xxx\")" -> { displayName: "search_files", fullCall: "search_files(query=\"xxx\")" }
 * - "read_file(path=\"/a/b\")" -> { displayName: "read_file", fullCall: "read_file(path=\"/a/b\")" }
 * - "memo_search" -> { displayName: "memo_search", fullCall: "memo_search" }
 */
function extractToolName(callName: string): { displayName: string; fullCall: string } {
  // Match function name before opening parenthesis
  const match = callName.match(/^([a-zA-Z_][a-zA-Z0-9_]*)\s*\(/);
  if (match) {
    return { displayName: match[1], fullCall: callName };
  }
  // If no parentheses found, the entire string is the tool name
  return { displayName: callName, fullCall: callName };
}

/**
 * CompactToolCall - 轻量级工具调用卡片
 */
interface CompactToolCallProps {
  displayName: string;
  fullCall: string;
  inputSummary?: string;
  filePath?: string;
  duration?: number;
  isError?: boolean;
  isRunning?: boolean;
  hasResult?: boolean;
  output?: string;
  isOutputError?: boolean;
}

function CompactToolCall({
  displayName,
  fullCall,
  inputSummary,
  filePath,
  duration,
  isError,
  isRunning,
  hasResult,
  output,
  isOutputError,
}: CompactToolCallProps) {
  const { t } = useTranslation();
  const isWriteOp = ["write", "edit", "bash", "run_command"].some((k) => displayName.toLowerCase().includes(k));

  // Interactive expand/collapse state with localStorage persistence
  // Storage key format: tool-expanded-{displayName}-{hash of inputSummary or displayName}
  // Uses proper hash function to avoid collisions from simple slice(0,8)
  const storageKey = useMemo(() => {
    // Include both displayName and full inputSummary for uniqueness
    const uniquePart = inputSummary ? `${displayName}-${simpleHash(inputSummary)}` : displayName;
    return `tool-expanded-${uniquePart}`;
  }, [displayName, inputSummary]);

  const [isExpanded, setIsExpanded] = useState(() => {
    // Initialize from localStorage if available
    if (typeof window !== "undefined") {
      try {
        const stored = localStorage.getItem(storageKey);
        return stored === "true";
      } catch (err) {
        // Log storage errors in development for debugging
        if (import.meta.env.DEV) {
          console.warn("[CompactToolCall] Failed to read localStorage:", storageKey, err);
        }
      }
    }
    return false;
  });

  // Persist expand state changes to localStorage
  useEffect(() => {
    if (typeof window !== "undefined") {
      try {
        localStorage.setItem(storageKey, String(isExpanded));
      } catch (err) {
        // Log storage errors in development for debugging
        if (import.meta.env.DEV) {
          console.warn("[CompactToolCall] Failed to write localStorage:", storageKey, err);
        }
      }
    }
  }, [storageKey, isExpanded]);

  // Determine expandable content: only output result is worth expanding
  // inputSummary is already shown in compact view, so don't count it
  const hasOutputResult = output && output.trim().length > 0;
  const hasExpandableContent = hasOutputResult;
  const showExpandHint = !isExpanded && hasExpandableContent;

  // Generate unique ID for ARIA attributes
  const contentId = `tool-content-${displayName}-${inputSummary ? simpleHash(inputSummary) : ""}`;

  const handleToggle = useCallback(() => {
    if (hasExpandableContent) {
      setIsExpanded((prev) => !prev);
    }
  }, [hasExpandableContent]);

  // Space key press tracking for proper ARIA pattern (keydown preventDefault, keyup trigger)
  const spacePressedRef = useRef(false);

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      if (!hasExpandableContent) return;

      if (e.key === "Enter") {
        e.preventDefault();
        handleToggle();
      } else if (e.key === " ") {
        // Prevent page scroll on keydown, track for keyup trigger
        e.preventDefault();
        spacePressedRef.current = true;
      } else if (e.key === "Escape" && isExpanded) {
        e.preventDefault();
        setIsExpanded(false);
      }
    },
    [hasExpandableContent, handleToggle, isExpanded],
  );

  const handleKeyUp = useCallback(
    (e: React.KeyboardEvent) => {
      if (!hasExpandableContent) return;

      if (e.key === " " && spacePressedRef.current) {
        spacePressedRef.current = false;
        handleToggle();
      }
    },
    [hasExpandableContent, handleToggle],
  );

  return (
    <div
      className={cn(
        "rounded-lg border px-3 py-2 transition-all duration-200",
        "bg-card hover:shadow-sm",
        // Add cursor pointer when expandable
        hasExpandableContent && "cursor-pointer",
        isWriteOp ? "border-purple-200/50 dark:border-purple-800/30 bg-purple-50/10" : "border-border/50",
      )}
      onClick={hasExpandableContent ? handleToggle : undefined}
      role={hasExpandableContent ? "button" : undefined}
      tabIndex={hasExpandableContent ? 0 : undefined}
      aria-expanded={hasExpandableContent ? isExpanded : undefined}
      aria-controls={hasExpandableContent ? contentId : undefined}
      onKeyDown={hasExpandableContent ? handleKeyDown : undefined}
      onKeyUp={hasExpandableContent ? handleKeyUp : undefined}
    >
      {/* Line 1: Tool Name + Status + Duration */}
      <div className="flex items-center justify-between gap-3">
        <div className="flex items-center gap-2 min-w-0">
          <span className={cn("font-semibold text-sm", isWriteOp ? "text-purple-700 dark:text-purple-300" : "text-foreground")}>
            {displayName}
          </span>
          {/* Status Indicator - priority: running > done > error > pending */}
          {isRunning ? (
            <span className="flex items-center gap-1 text-[11px] text-purple-600 dark:text-purple-400">
              <Loader2 className="w-3 h-3 animate-spin" /> {t("ai.events.running")}
            </span>
          ) : hasResult ? (
            isOutputError ? (
              <span className="flex items-center gap-1 text-[11px] text-red-600 dark:text-red-400">
                <AlertCircle className="w-3 h-3" /> {t("ai.events.error")}
              </span>
            ) : (
              <span className="flex items-center gap-1 text-[11px] text-green-600 dark:text-green-400">✓ {t("ai.events.done")}</span>
            )
          ) : (
            <span className="text-[11px] text-muted-foreground">{t("ai.events.pending")}</span>
          )}
        </div>
        {/* Duration */}
        {duration && (
          <span className="text-[11px] text-muted-foreground font-mono shrink-0">
            {duration > 1000 ? `${(duration / 1000).toFixed(1)}s` : `${duration}ms`}
          </span>
        )}
      </div>

      {/* Line 2: Function Call + Parameters (Compact) - always shown unless empty */}
      {(inputSummary || filePath || fullCall !== displayName) && (
        <div className="mt-1 flex items-center gap-2 min-w-0">
          {/* Function with params - prefer inputSummary, fallback to fullCall */}
          {inputSummary ? (
            <code
              className={cn(
                "text-xs font-mono truncate block",
                isError ? "text-red-600/80 dark:text-red-400/80" : "text-muted-foreground/70",
              )}
              title={inputSummary}
            >
              {inputSummary}
            </code>
          ) : filePath ? (
            <code className="text-xs font-mono text-muted-foreground/70 truncate block" title={filePath}>
              {filePath}
            </code>
          ) : fullCall !== displayName ? (
            <code
              className={cn(
                "text-xs font-mono truncate block",
                isError ? "text-red-600/80 dark:text-red-400/80" : "text-muted-foreground/70",
              )}
              title={fullCall}
            >
              {fullCall}
            </code>
          ) : null}
        </div>
      )}

      {/* Expand hint indicator - only show when there's output to reveal */}
      {showExpandHint && (
        <div className="mt-1 flex items-center gap-1 text-[10px] text-muted-foreground opacity-60">
          <ChevronRight className="w-3 h-3" />
          {t("ai.unified_block.click_details") || "点击查看详情"}
        </div>
      )}

      {/* Expanded Details View - only show output result (inputSummary already visible above) */}
      {isExpanded && (hasOutputResult || (hasResult && !hasOutputResult)) && (
        <div
          id={contentId}
          className="mt-3 space-y-3 animate-in fade-in slide-in-from-top-1 duration-200"
          role="region"
          aria-label={t("ai.unified_block.tool_details") || "工具调用详情"}
        >
          {/* Output Result */}
          <div>
            <div className="text-[11px] text-muted-foreground mb-1 flex items-center gap-1">
              {t("ai.unified_block.output") || "输出结果"}
            </div>
            {hasOutputResult ? (
              <pre
                className={cn(
                  "text-xs font-mono overflow-x-auto whitespace-pre-wrap break-words max-h-48 overflow-y-auto p-3 rounded border",
                  isOutputError ? "text-red-600/90 bg-red-50/50 border-red-200/50" : "text-muted-foreground bg-muted/30 border-border/50",
                )}
              >
                {output}
              </pre>
            ) : (
              <div className="text-xs text-muted-foreground italic p-3 rounded border border-border/50 bg-muted/20">
                {t("ai.unified_block.no_output") || "无输出结果"}
              </div>
            )}
          </div>
        </div>
      )}
    </div>
  );
}

// Tool call type
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

// ============================================================================
// Constants
// ============================================================================

/**
 * Timestamp multiplier for calculating round-based timestamps.
 *
 * This value ensures tool calls from the same round are grouped together
 * in the timeline, with microsecond precision to avoid conflicts.
 * Formula: baseTimestamp (microseconds) + index * offset
 *
 * @example Round 2, 3rd tool call: 2_000_000 + 2 * 1000 = 2002000 microseconds
 */
// Note: ROUND_TIMESTAMP_MULTIPLIER, TOOL_CALL_OFFSET_US, and USER_INPUTS_EXPAND_THRESHOLD imported from constants.ts

// ============================================================================
// Types
// ============================================================================

export interface BlockState {
  collapsed: boolean;
  isLatest: boolean;
  isStreaming: boolean;
}

export interface UnifiedMessageBlockProps {
  /** User message that triggered this block */
  userMessage: ConversationMessage;
  /** Additional user inputs appended during streaming (追加的用户输入) */
  additionalUserInputs?: ConversationMessage[];
  /** Assistant message (may be streaming) */
  assistantMessage?: ConversationMessage;
  /** Block summary for this chat round (Geek/Evolution modes) */
  blockSummary?: BlockSummary;
  /** Current parrot agent type */
  parrotId?: ParrotAgentType;
  /** Whether this is the latest block */
  isLatest?: boolean;
  /** Whether assistant is currently streaming */
  isStreaming?: boolean;
  /** Current streaming phase for animation */
  streamingPhase?: "thinking" | "tools" | "answer" | null;
  /** Actions */
  onCopy?: (content: string) => void;
  onRegenerate?: () => void;
  onDelete?: () => void;
  /** Cancel streaming callback (#113) */
  onCancel?: () => void;
  /** Block ID for edit/fork operations */
  blockId?: bigint;
  /** Block sequence number (1-based) for display */
  blockNumber?: number;
  /** Additional children to render in block body */
  children?: ReactNode;
  className?: string;
}

// ============================================================================
// Theme Configuration
// ============================================================================

/** Block themes based on BlockMode (NORMAL | GEEK | EVOLUTION) */
const BLOCK_THEMES: Record<
  "default" | "NORMAL" | "GEEK" | "EVOLUTION",
  {
    border: string;
    headerBg: string;
    footerBg: string;
    badgeBg: string;
    badgeText: string;
    ringColor: string;
  }
> = {
  default: {
    border: "border-zinc-200 dark:border-zinc-700",
    headerBg: "bg-zinc-50 dark:bg-zinc-900/50",
    footerBg: "bg-zinc-200/80 dark:bg-zinc-800/60",
    badgeBg: "bg-zinc-100 dark:bg-zinc-800",
    badgeText: "text-zinc-600 dark:text-zinc-400",
    ringColor: "ring-primary/20",
  },
  // NORMAL - 普通 AI 模式（MEMO/SCHEDULE/AMAZING 都用这个）
  NORMAL: {
    border: "border-amber-200 dark:border-amber-700",
    headerBg: "bg-amber-50 dark:bg-amber-900/20",
    footerBg: "bg-amber-200/80 dark:bg-amber-800/50",
    badgeBg: "bg-amber-100 dark:bg-amber-900/30",
    badgeText: "text-amber-600 dark:text-amber-400",
    ringColor: "ring-amber-500/20",
  },
  // GEEK - 极客模式（Claude Code CLI）
  GEEK: {
    border: "border-sky-200 dark:border-slate-700",
    headerBg: "bg-sky-50 dark:bg-slate-900/20",
    footerBg: "bg-sky-200/80 dark:bg-slate-800/50",
    badgeBg: "bg-sky-100 dark:bg-slate-900/30",
    badgeText: "text-sky-600 dark:text-slate-400",
    ringColor: "ring-sky-500/20",
  },
  // EVOLUTION - 进化模式（系统自我进化）
  EVOLUTION: {
    border: "border-emerald-200 dark:border-emerald-700",
    headerBg: "bg-emerald-50 dark:bg-emerald-900/20",
    footerBg: "bg-emerald-200/80 dark:bg-emerald-800/50",
    badgeBg: "bg-emerald-100 dark:bg-emerald-900/30",
    badgeText: "text-emerald-600 dark:text-emerald-400",
    ringColor: "ring-emerald-500/20",
  },
};

// ============================================================================
// Helper Functions
// ============================================================================

function getDefaultCollapseState(isLatest: boolean, isStreaming: boolean): boolean {
  if (isStreaming || isLatest) return false;
  return true;
}

// ============================================================================
// Sub-Components (Legacy - 保留用于 UserInputsSection)
// ============================================================================

// ============================================================================
// User Inputs Section Component (新增)
// ============================================================================

interface UserInputsSectionProps {
  userMessage: ConversationMessage;
  additionalUserInputs?: ConversationMessage[];
  isCollapsed?: boolean;
  isStreaming?: boolean;
}

function UserInputsSection({ userMessage, additionalUserInputs = [], isCollapsed }: UserInputsSectionProps) {
  const { t } = useTranslation();
  const [isExpanded, setIsExpanded] = useState(false);

  const allInputs = useMemo(() => [userMessage, ...additionalUserInputs], [userMessage, additionalUserInputs]);
  const hasMultiple = allInputs.length > 1;
  const totalLength = allInputs.reduce((sum, m) => sum + m.content.length, 0);
  const isLongContent = totalLength > USER_INPUTS_EXPAND_THRESHOLD;

  // 默认展开条件：内容不长或只有一个输入
  const shouldShowExpanded = !isLongContent || isExpanded;

  if (isCollapsed) return null;

  return (
    <div className="relative group">
      {/* Timeline Node - 使用统一组件 - #1 Fix: 统一使用 absolute -left-[2rem] 包裹 */}
      <div className="absolute -left-[2rem] top-0.5">
        <TimelineNode type="user" />
      </div>

      {/* Section Header */}
      <div className="flex items-center justify-between mb-3">
        {/* Section Title */}
        <div className="text-sm font-medium text-muted-foreground">
          <span>{t("ai.unified_block.user_inputs") || "用户输入"}</span>
        </div>

        {/* Expand/Collapse Button */}
        {(isLongContent || hasMultiple) && (
          <button
            type="button"
            onClick={() => setIsExpanded(!isExpanded)}
            className="text-xs text-muted-foreground hover:text-foreground transition-colors flex items-center gap-1 px-2 py-1 rounded hover:bg-muted/50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
          >
            {isExpanded ? (
              <>
                <ChevronUp className="w-3.5 h-3.5" />
                {t("common.collapse") || "收起"}
              </>
            ) : (
              <>
                <ChevronDown className="w-3.5 h-3.5" />
                {t("common.expand") || "展开"}
              </>
            )}
          </button>
        )}
      </div>

      {/* Inputs List */}
      <div className="space-y-3">
        {shouldShowExpanded ? (
          allInputs.map((input, index) => (
            <div
              key={input.id}
              className={cn("rounded-lg border p-3 transition-colors", "bg-muted/20 border-border/50 hover:border-border")}
            >
              <div className="flex items-start gap-2">
                <span className="flex-shrink-0 w-5 h-5 rounded-full bg-blue-100 dark:bg-blue-900/30 text-blue-600 dark:text-blue-400 flex items-center justify-center text-xs font-medium">
                  {index + 1}
                </span>
                <div className="flex-1 min-w-0">
                  <div className="text-sm text-foreground whitespace-pre-wrap break-words">{input.content}</div>
                </div>
              </div>
            </div>
          ))
        ) : (
          /* Collapsed Preview */
          <button
            type="button"
            onClick={() => setIsExpanded(true)}
            className="w-full text-left rounded-lg border p-3 bg-muted/20 border-border/50 cursor-pointer hover:bg-muted/30 transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
          >
            <div className="text-sm text-muted-foreground line-clamp-3 italic">
              {allInputs.map((m) => m.content.split("\n")[0]).join(" ")}...
            </div>
            <div className="text-xs text-muted-foreground/60 mt-2 flex items-center gap-1">
              <ChevronDown className="w-3 h-3" />
              {t("ai.unified_block.expand_to_read_all") || "点击展开查看全部内容"}
            </div>
          </button>
        )}
      </div>
    </div>
  );
}

interface BlockBodyProps {
  userMessage?: ConversationMessage;
  additionalUserInputs?: ConversationMessage[];
  assistantMessage?: ConversationMessage;
  blockSummary?: BlockSummary;
  parrotId?: ParrotAgentType;
  isCollapsed: boolean;
  themeColors: (typeof PARROT_THEMES)[keyof typeof PARROT_THEMES];
  streamingPhase?: "thinking" | "tools" | "answer" | null;
  isLatest?: boolean;
  isStreaming?: boolean;
  children?: ReactNode;
}

function BlockBody({
  userMessage,
  additionalUserInputs = [],
  assistantMessage,
  blockSummary,
  parrotId,
  isCollapsed,
  themeColors,
  streamingPhase = null,
  isLatest = false,
  isStreaming = false,
  children,
}: BlockBodyProps) {
  const { t } = useTranslation();
  const contentRef = useRef<HTMLDivElement>(null);

  // Helper: Get mode-specific thinking text
  const getThinkingText = useCallback(() => {
    if (parrotId === ParrotAgentType.GEEK) {
      return t("ai.geek_mode.thinking");
    }
    if (parrotId === ParrotAgentType.EVOLUTION) {
      return t("ai.evolution_mode.thinking");
    }
    return t("ai.states.thinking");
  }, [parrotId, t]);

  // Check for error state
  const hasError = assistantMessage?.error;

  const thinkingSteps = assistantMessage?.metadata?.thinkingSteps || [];
  const toolCalls = assistantMessage?.metadata?.toolCalls || [];
  const toolResults = assistantMessage?.metadata?.toolResults || [];
  const hasAnswer = assistantMessage?.content;

  // 聚合 Thinking 内容
  const allThinkingContent = useMemo(() => {
    // Build placeholder list dynamically from i18n to catch translated placeholders
    const placeholderTexts = [
      "处理中...",
      "Thinking...",
      "...",
      "AI is thinking",
      "思考中",
      // Translated placeholders from i18n
      t("ai.geek_mode.thinking"),
      t("ai.evolution_mode.thinking"),
      t("ai.states.thinking"),
      t("schedule.streaming-assistant.thinking"),
    ].filter(Boolean); // Remove undefined values

    return thinkingSteps
      .map((s) => s.content?.trim() || "")
      .filter((c) => {
        if (!c) return false;
        // Filter out placeholder texts (exact match or starts with)
        const isPlaceholder = placeholderTexts.some((p) => p && (c === p || c.startsWith(p)));
        return !isPlaceholder;
      })
      .join("\n\n");
  }, [thinkingSteps, t]);

  // States for collapsible sections - Default expand thinking if it's the latest message
  const [isThinkingExpanded, setIsThinkingExpanded] = useState(() => isLatest && allThinkingContent.length > 0);

  // Auto-expand thinking when content arrives for the latest message
  useEffect(() => {
    if (isLatest && allThinkingContent.length > 0 && streamingPhase === "thinking") {
      setIsThinkingExpanded(true);
    }
  }, [allThinkingContent.length, isLatest, streamingPhase]);

  // 构建时序事件列表（按时间顺序排列）
  type TimelineEvent =
    | { type: "thinking"; id: string; timestamp: number; data: { content: string }; isFirst: boolean }
    | { type: "tool_call"; id: string; timestamp: number; data: ToolCall }
    | { type: "tool_result"; id: string; timestamp: number; data: (typeof toolResults)[number] };

  const timelineEvents: TimelineEvent[] = [];

  // 1. Thinking Logic: 作为一个整体处理，而不是分散的事件
  if (allThinkingContent.length > 0) {
    timelineEvents.push({
      type: "thinking",
      id: "thinking-group",
      timestamp: thinkingSteps[0]?.timestamp || 0,
      data: { content: allThinkingContent },
      isFirst: true,
    });
  }

  // 2. Tool Logic
  toolCalls.forEach((call, index) => {
    const round = (typeof call === "object" ? call.round : 0) || 0;
    const baseTimestamp = round * ROUND_TIMESTAMP_MULTIPLIER;
    const callTimestamp = baseTimestamp + index * TOOL_CALL_OFFSET_US;
    timelineEvents.push({
      type: "tool_call",
      id: `toolcall-${round}-${index}`,
      timestamp: callTimestamp,
      data: call,
    });
  });

  // 按时间戳排序
  timelineEvents.sort((a, b) => a.timestamp - b.timestamp);

  // When collapsed, show minimal info
  if (isCollapsed) {
    return <div className="px-4 py-2 text-sm text-muted-foreground italic">{t("ai.unified_block.collapsed")}</div>;
  }

  return (
    <div className="px-4 py-4">
      {/* Timeline Flow */}
      <div className="relative">
        {/* Left Timeline Line - 优化：虚线/实线混合 */}
        <div className="absolute left-[11px] top-2 bottom-4 w-px bg-border/60" />

        <div className="relative pl-8 space-y-6">
          {/* User Inputs Section - 新增用户输入区域 (放在 timeline 内) */}
          {(userMessage || additionalUserInputs.length > 0) && (
            <UserInputsSection
              userMessage={userMessage || { id: "", role: "user" as const, content: "", timestamp: Date.now() }}
              additionalUserInputs={additionalUserInputs}
              isCollapsed={isCollapsed}
            />
          )}

          {/* 1. Thinking Section - 使用 TimelineNode 组件 */}
          {/* 显示条件: 有 thinkingSteps（即使只是占位符）或正在流式思考 */}
          {(thinkingSteps.length > 0 || streamingPhase === "thinking") && (
            <div className="relative group">
              {/* Timeline Node - thinking 类型，流式时显示加载动画 */}
              <div className="absolute -left-[2rem] top-0.5">
                {streamingPhase === "thinking" ? (
                  <div className="w-6 h-6 rounded-full bg-blue-100 dark:bg-blue-900/40 border-2 border-blue-500 flex items-center justify-center shrink-0 z-10 ring-4 ring-blue-50 dark:ring-blue-900/10">
                    <Loader2 className="w-3.5 h-3.5 text-blue-600 dark:text-blue-400 animate-spin" />
                  </div>
                ) : (
                  <TimelineNode type="thinking" />
                )}
              </div>

              <div className="flex flex-col">
                <button
                  onClick={() => allThinkingContent.length > 0 && setIsThinkingExpanded(!isThinkingExpanded)}
                  className="flex items-center gap-2 text-sm font-medium text-foreground hover:text-blue-600 dark:hover:text-blue-400 transition-colors text-left w-full focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring rounded"
                >
                  <span className="flex items-center gap-2">
                    {streamingPhase === "thinking" ? (
                      <>
                        <Loader2 className="w-3.5 h-3.5 animate-spin text-blue-500" />
                        <span className="text-blue-600 dark:text-blue-400">{getThinkingText()}</span>
                      </>
                    ) : allThinkingContent.length > 0 ? (
                      // #2 Fix: 已完成状态显示"思考完成"而不是"思考中"
                      <span className="text-muted-foreground flex items-center gap-1">
                        <span className="text-green-600 dark:text-green-400">✓</span>
                        {t("ai.unified_block.thinking_done") || "思考完成"}
                      </span>
                    ) : (
                      // 只有占位符，没有真实思考内容
                      <span className="text-muted-foreground">{t("ai.unified_block.thinking_process") || "思考过程"}</span>
                    )}
                  </span>
                  {/* 只有在有真实内容时才显示展开/收起图标 */}
                  {allThinkingContent.length > 0 && (
                    <span className="ml-auto">
                      {isThinkingExpanded ? (
                        <ChevronUp className="w-4 h-4 text-muted-foreground" />
                      ) : (
                        <ChevronDown className="w-4 h-4 text-muted-foreground" />
                      )}
                    </span>
                  )}
                </button>

                {/* Expanded View - 只在有真实内容时显示 */}
                {isThinkingExpanded && allThinkingContent.length > 0 && (
                  <div className="mt-2 text-sm text-muted-foreground bg-muted/30 p-3 rounded-lg border border-border/50 animate-in fade-in slide-in-from-top-1 duration-200 prose prose-xs dark:prose-invert max-w-none">
                    <ReactMarkdown>{allThinkingContent}</ReactMarkdown>
                  </div>
                )}
              </div>
            </div>
          )}

          {/* 2. Tool Calls Stream - 使用 ToolCallCard 组件 */}
          {timelineEvents.map((event) => {
            if (event.type !== "tool_call") return null;

            const call = event.data;
            const rawCallName = typeof call === "string" ? call : call.name;
            // Extract pure tool name for title, full call for content
            const { displayName, fullCall } = extractToolName(rawCallName);

            // #3 Fix: Improve tool result matching - use displayName instead of fullCall for matching
            // result.name contains only the tool name (e.g., "search_files")
            // while rawCallName may contain full function call (e.g., "search_files(query=\"xxx\")")
            const result = toolResults.find(
              (r) => r.name === displayName || (typeof call === "object" && call.toolId && r.toolId === call.toolId),
            );

            const isError = typeof call === "object" ? call.isError : assistantMessage?.error;

            // Determine running state:
            // - If streaming in tools phase AND this tool has no result yet, it's running
            // - This fixes the issue where only the last tool showed as running
            const isToolRunning = streamingPhase === "tools" && !result;
            const hasToolResult = !!result;
            const hasError = isError || result?.isError;

            return (
              <div key={event.id} className="relative group">
                {/* Timeline Node - 使用 TimelineNode 组件 */}
                <div className="absolute -left-[2rem] top-0">
                  {isToolRunning ? (
                    <div className="w-6 h-6 rounded-full bg-purple-100 dark:bg-purple-900/40 border-2 border-purple-500 flex items-center justify-center shrink-0 z-10 animate-pulse">
                      <Loader2 className="w-3 h-3 text-purple-600 animate-spin" />
                    </div>
                  ) : hasError ? (
                    <TimelineNode type="error" />
                  ) : (
                    <TimelineNode type="tool" />
                  )}
                </div>

                {/* Tool Call Card - 使用 CompactToolCall 组件 */}
                <CompactToolCall
                  displayName={displayName}
                  fullCall={fullCall}
                  inputSummary={typeof call === "object" ? call.inputSummary : undefined}
                  filePath={typeof call === "object" ? call.filePath : undefined}
                  duration={typeof call === "object" ? call.duration : undefined}
                  isError={hasError}
                  isRunning={isToolRunning}
                  hasResult={hasToolResult}
                  output={result?.outputSummary}
                  isOutputError={result?.isError}
                />
              </div>
            );
          })}

          {/* 3. AI Answer Section - 使用 TimelineNode */}
          {hasAnswer ? (
            <div className="relative pt-2">
              {/* Timeline Node - answer 类型 */}
              <div className="absolute -left-[2rem] top-3.5">
                {streamingPhase === "answer" ? (
                  <div className="w-6 h-6 rounded-full bg-amber-100 dark:bg-amber-900/40 border-2 border-amber-500 flex items-center justify-center shrink-0 z-10 animate-pulse">
                    <Loader2 className="w-3.5 h-3.5 text-amber-600 animate-spin" />
                  </div>
                ) : (
                  <TimelineNode type="answer" />
                )}
              </div>

              {/* Message bubble */}
              <div
                className={cn(
                  "relative rounded-xl shadow-sm transition-colors",
                  themeColors.bubbleBg,
                  themeColors.bubbleBorder,
                  themeColors.text,
                )}
              >
                {/* Markdown content with streaming effect */}
                <div ref={contentRef} className="px-5 py-4">
                  <StreamingMarkdown
                    content={assistantMessage.content || getThinkingText()}
                    isStreaming={isStreaming && streamingPhase === "answer"}
                    parrotId={parrotId ?? ParrotAgentType.AUTO}
                    enableTypingCursor={true}
                    className="break-words leading-normal font-sans text-[15px]"
                  />
                  {children}
                </div>
              </div>
            </div>
          ) : (
            /* Pending State (Cold Start / Initializing) */
            (isLatest || children) && (
              <div className="relative pt-2 px-1 animate-in fade-in duration-300">
                <div className="flex items-center gap-3 text-muted-foreground">
                  {/* Fallback spinner if valid children (cursor) is not provided */}
                  {children ? (
                    <div className="scale-90 origin-left">{children}</div>
                  ) : (
                    <>
                      <Loader2 className="w-4 h-4 animate-spin text-muted-foreground/50" />
                      <span className="text-sm italic opacity-70">{t("ai.states.initializing")}</span>
                    </>
                  )}
                </div>
              </div>
            )
          )}

          {/* 4. Error Section - 使用 TimelineNode */}
          {hasError && (
            <div className="relative group">
              {/* Timeline Node - 与其他 Section 一致的位置规范 */}
              <div className="absolute -left-[2rem] top-0">
                <TimelineNode type="error" />
              </div>
              <div className="p-3 rounded-lg bg-red-50 dark:bg-red-900/10 border border-red-200 dark:border-red-800/30 text-sm">
                <p className="font-semibold text-red-700 dark:text-red-300 flex items-center gap-2">
                  {t("ai.unified_block.error_occurred")}
                </p>
                <p className="mt-1 text-red-600/80 dark:text-red-400/80 font-mono text-xs break-all">{assistantMessage.error}</p>
              </div>
            </div>
          )}
          {/* 5. Block Summary (Detailed view for all modes if present) */}
          {blockSummary && (
            <div className="relative">
              <div className="absolute -left-[2rem] top-1 w-6 h-6 rounded-full bg-green-100 dark:bg-green-900/30 border border-green-500 flex items-center justify-center shrink-0 z-10 transition-colors">
                <BarChart3 className="w-3.5 h-3.5 text-green-600 dark:text-green-400" />
              </div>
              <div className="pl-0">
                <ExpandedSessionSummary summary={blockSummary} />
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

// ============================================================================
// Legacy BlockHeader/Footer - 已替换为模块化组件，保留此接口用于向后兼容
// ============================================================================

// 将主题类型转换
function themeToHeaderTheme(theme: (typeof BLOCK_THEMES)[keyof typeof BLOCK_THEMES]): BlockHeaderTheme {
  return {
    border: theme.border,
    headerBg: theme.headerBg,
    footerBg: theme.footerBg,
    badgeBg: theme.badgeBg,
    badgeText: theme.badgeText,
    ringColor: theme.ringColor,
  };
}

function themeToFooterTheme(theme: (typeof BLOCK_THEMES)[keyof typeof BLOCK_THEMES]): BlockFooterTheme {
  return {
    border: theme.border,
    headerBg: theme.headerBg,
    footerBg: theme.footerBg,
    badgeBg: theme.badgeBg,
    badgeText: theme.badgeText,
    ringColor: theme.ringColor,
  };
}

// 使用 ModularBlockHeader 替代内联实现
function BlockHeader({
  userMessage,
  assistantMessage,
  blockSummary,
  parrotId,
  theme,
  onToggle,
  isCollapsed,
  isStreaming,
  additionalUserInputs = [],
  blockNumber,
}: BlockHeaderProps) {
  const headerTheme = themeToHeaderTheme(theme);

  return (
    <ModularBlockHeader
      userMessage={userMessage}
      assistantMessage={assistantMessage}
      blockSummary={blockSummary}
      parrotId={parrotId}
      theme={headerTheme}
      onToggle={onToggle}
      isCollapsed={isCollapsed}
      isStreaming={isStreaming}
      additionalUserInputs={additionalUserInputs}
      blockNumber={blockNumber}
    />
  );
}

// 使用 ModularBlockFooter 替代内联实现
function BlockFooter({ isCollapsed, onToggle, onCopy, onRegenerate, onDelete, theme, isStreaming, onCancel }: BlockFooterProps) {
  const footerTheme = themeToFooterTheme(theme);

  return (
    <ModularBlockFooter
      isCollapsed={isCollapsed}
      onToggle={onToggle}
      onCopy={onCopy}
      onRegenerate={onRegenerate}
      onDelete={onDelete}
      theme={footerTheme}
      isStreaming={isStreaming}
      onCancel={onCancel}
    />
  );
}

// ============================================================================
// Main Component
// ============================================================================

export const UnifiedMessageBlock = memo(function UnifiedMessageBlock({
  userMessage,
  additionalUserInputs = [],
  assistantMessage,
  blockSummary,
  parrotId,
  isLatest = false,
  isStreaming = false,
  streamingPhase = null,
  onCopy,
  onRegenerate,
  onDelete,
  onCancel,
  blockId: _blockId,
  blockNumber,
  children,
  className,
}: UnifiedMessageBlockProps) {
  // Map ParrotAgentType to BlockMode for theme selection
  // AUTO/MEMO/SCHEDULE/AMAZING → NORMAL, GEEK → GEEK, EVOLUTION → EVOLUTION
  const getBlockModeFromParrot = (): "NORMAL" | "GEEK" | "EVOLUTION" => {
    switch (parrotId) {
      case ParrotAgentType.GEEK:
        return "GEEK";
      case ParrotAgentType.EVOLUTION:
        return "EVOLUTION";
      default:
        return "NORMAL";
    }
  };

  const blockMode = getBlockModeFromParrot();
  const blockTheme = BLOCK_THEMES[blockMode] || BLOCK_THEMES.default;
  const themeColors = PARROT_THEMES[parrotId || "AUTO"] || PARROT_THEMES.AUTO;

  // P1 Fix: Use ref for latest values to avoid closure traps in fast succession
  const isLatestRef = useRef(isLatest);
  const isStreamingRef = useRef(isStreaming);

  useEffect(() => {
    isLatestRef.current = isLatest;
    isStreamingRef.current = isStreaming;
  }, [isLatest, isStreaming]);

  const [collapsed, setCollapsed] = useState(() => getDefaultCollapseState(isLatest, isStreaming));

  useEffect(() => {
    setCollapsed(getDefaultCollapseState(isLatestRef.current, isStreamingRef.current));
  }, [isLatest, isStreaming]);

  const toggleCollapse = useCallback(() => {
    setCollapsed((prev) => !prev);
  }, []);

  // Build content for copying (包含追加用户输入)
  const contentForCopy = useMemo(() => {
    const userContents = [userMessage.content, ...additionalUserInputs.map((m) => m.content)];
    return [
      `User: ${userContents.join("\n> ")}`,
      assistantMessage?.content ? `Assistant: ${assistantMessage.content}` : "",
      assistantMessage?.metadata?.toolResults
        ? `\n\nTools:\n${assistantMessage.metadata.toolResults.map((r) => `- ${r.name}: ${r.duration}ms`).join("\n")}`
        : "",
    ]
      .filter(Boolean)
      .join("\n\n");
  }, [userMessage.content, additionalUserInputs, assistantMessage?.content, assistantMessage?.metadata?.toolResults]);

  const handleCopy = useCallback(() => {
    onCopy?.(contentForCopy);
  }, [contentForCopy, onCopy]);

  return (
    <div
      className={cn(
        "rounded-lg border overflow-hidden shadow-sm transition-all duration-300",
        blockTheme.border,
        // Active/Streaming state: Ring (no breathing animation to prevent visual flicker)
        isLatest && isStreaming && `ring-2 ${blockTheme.ringColor}`,
        isLatest && !isStreaming && `ring-1 ${blockTheme.ringColor}`,
        className,
      )}
    >
      {/* Block Header - 始终显示 */}
      <div className={cn("border-b", blockTheme.border)}>
        <BlockHeader
          userMessage={userMessage}
          assistantMessage={assistantMessage}
          blockSummary={blockSummary}
          parrotId={parrotId}
          theme={blockTheme}
          onToggle={toggleCollapse}
          isCollapsed={collapsed}
          isStreaming={isStreaming}
          additionalUserInputs={additionalUserInputs}
          blockNumber={blockNumber}
        />
      </div>

      {/* Block Body - 可折叠内容 */}
      <BlockBody
        userMessage={userMessage}
        additionalUserInputs={additionalUserInputs}
        assistantMessage={assistantMessage}
        blockSummary={blockSummary}
        parrotId={parrotId}
        isCollapsed={collapsed}
        themeColors={themeColors}
        streamingPhase={streamingPhase}
        isLatest={isLatest}
        isStreaming={isStreaming}
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
          isStreaming={isStreaming}
          onCancel={onCancel}
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
