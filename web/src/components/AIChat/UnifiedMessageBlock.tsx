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

import {
  AlertCircle,
  BarChart3,
  Brain,
  Check,
  ChevronDown,
  ChevronUp,
  Clock,
  Copy,
  Loader2,
  Terminal,
  User,
  Wrench,
  Zap,
} from "lucide-react";
import { memo, ReactNode, useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import ReactMarkdown from "react-markdown";
import remarkBreaks from "remark-breaks";
import remarkGfm from "remark-gfm";
// Import constants
import { BADGE_WIDTH_OFFSET, HEADER_VISUAL_WIDTH, TOOL_CALL_OFFSET_MS, USER_INPUTS_EXPAND_THRESHOLD } from "@/components/AIChat/constants";
import { ExpandedSessionSummary } from "@/components/AIChat/ExpandedSessionSummary";
import { CodeBlock } from "@/components/MemoContent/CodeBlock";
import { cn } from "@/lib/utils";
import { ConversationMessage } from "@/types/aichat";
import { PARROT_THEMES, ParrotAgentType, SessionSummary } from "@/types/parrot";

type CodeComponentProps = React.ComponentProps<"code"> & { inline?: boolean };

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

/** Timestamp multiplier for calculating round-based timestamps */
const ROUND_TIMESTAMP_MULTIPLIER = 1_000_000;

// Note: TOOL_CALL_OFFSET_MS and USER_INPUTS_EXPAND_THRESHOLD imported from constants.ts

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
  /** Session summary for Geek/Evolution modes */
  sessionSummary?: SessionSummary;
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
  /** Additional children to render in block body */
  children?: ReactNode;
  className?: string;
}

// ============================================================================
// Theme Configuration
// ============================================================================

const BLOCK_THEMES: Record<
  ParrotAgentType | "default",
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
  MEMO: {
    border: "border-slate-200 dark:border-slate-700",
    headerBg: "bg-slate-50 dark:bg-slate-900/50",
    footerBg: "bg-slate-200/80 dark:bg-slate-800/60",
    badgeBg: "bg-slate-100 dark:bg-slate-800",
    badgeText: "text-slate-600 dark:text-slate-400",
    ringColor: "ring-slate-500/20",
  },
  SCHEDULE: {
    border: "border-cyan-200 dark:border-cyan-700",
    headerBg: "bg-cyan-50 dark:bg-cyan-900/20",
    footerBg: "bg-cyan-200/80 dark:bg-cyan-800/50",
    badgeBg: "bg-cyan-100 dark:bg-cyan-900/30",
    badgeText: "text-cyan-600 dark:text-cyan-400",
    ringColor: "ring-cyan-500/20",
  },
  AMAZING: {
    border: "border-emerald-200 dark:border-emerald-700",
    headerBg: "bg-emerald-50 dark:bg-emerald-900/20",
    footerBg: "bg-emerald-200/80 dark:bg-emerald-800/50",
    badgeBg: "bg-emerald-100 dark:bg-emerald-900/30",
    badgeText: "text-emerald-600 dark:text-emerald-400",
    ringColor: "ring-emerald-500/20",
  },
  GEEK: {
    border: "border-violet-200 dark:border-violet-700",
    headerBg: "bg-violet-50 dark:bg-violet-900/20",
    footerBg: "bg-violet-200/80 dark:bg-violet-800/50",
    badgeBg: "bg-violet-100 dark:bg-violet-900/30",
    badgeText: "text-violet-600 dark:text-violet-400",
    ringColor: "ring-violet-500/20",
  },
  EVOLUTION: {
    border: "border-rose-200 dark:border-rose-700",
    headerBg: "bg-rose-50 dark:bg-rose-900/20",
    footerBg: "bg-rose-200/80 dark:bg-rose-800/50",
    badgeBg: "bg-rose-100 dark:bg-rose-900/30",
    badgeText: "text-rose-600 dark:text-rose-400",
    ringColor: "ring-rose-500/20",
  },
};

// ============================================================================
// Helper Functions
// ============================================================================

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

function extractUserInitial(content: string): string {
  const trimmed = content.trim();
  if (trimmed.length === 0) return "U";
  const match = trimmed.match(/[a-zA-Z\u4e00-\u9fa5]/);
  return match ? match[0].toUpperCase() : "U";
}

/**
 * 计算字符串的视觉宽度（改进版）
 * - ASCII 字符（英文字母、数字、半角符号）= 1
 * - 中文字符、全角符号、Emoji = 2
 * - 某些组合字符（如皮肤修饰符）不计入宽度
 */
function getVisualWidth(str: string): number {
  let width = 0;
  // 使用 [...str] 而不是 for...of 来正确处理代理对（surrogate pairs）
  const chars = [...str];
  for (let i = 0; i < chars.length; i++) {
    const char = chars[i];
    const code = char.codePointAt(0) || 0;

    // 跳过零宽连接符和变体选择器
    if (code === 0x200b || code === 0xfe0e || code === 0xfe0f || (code >= 0x1f3fb && code <= 0x1f3ff)) {
      continue;
    }

    // ASCII: 0-127
    if (code < 128) {
      width += 1;
    } else if (
      code >= 0x1100 &&
      // Hangul Jamo
      (code <= 0x11ff ||
        // CJK Radicals Supplement
        (code >= 0x2e80 && code <= 0x9fff) ||
        // CJK Ideographs
        (code >= 0x3400 && code <= 0x4dbf) ||
        // CJK Unified Ideographs Extension A
        (code >= 0x20000 && code <= 0x2ebef))
    ) {
      // CJK 字符统一为 2
      width += 2;
    } else {
      // 其他字符（包括 emoji）默认为 2
      // 对于复杂的 emoji 序列，这只是一个近似值
      width += 2;
    }
  }
  return width;
}

/**
 * 按视觉宽度截取字符串
 * @param str 原字符串
 * @param maxVisualWidth 最大视觉宽度
 * @returns 截取后的字符串
 */
function truncateByVisualWidth(str: string, maxVisualWidth: number): string {
  let currentWidth = 0;
  let result = "";

  for (const char of str) {
    const code = char.codePointAt(0) || 0;
    const charWidth = code < 128 ? 1 : 2;

    if (currentWidth + charWidth > maxVisualWidth) {
      return result + "...";
    }

    result += char;
    currentWidth += charWidth;
  }

  return result;
}

function getDefaultCollapseState(isLatest: boolean, isStreaming: boolean): boolean {
  if (isStreaming || isLatest) return false;
  return true;
}

// ============================================================================
// Sub-Components
// ============================================================================

interface BlockHeaderProps {
  userMessage: ConversationMessage;
  assistantMessage?: ConversationMessage;
  sessionSummary?: SessionSummary;
  parrotId?: ParrotAgentType;
  theme: (typeof BLOCK_THEMES)[keyof typeof BLOCK_THEMES];
  onToggle: () => void;
  isCollapsed: boolean;
  isStreaming?: boolean;
  /** 追加的用户输入列表 (支持多个) */
  additionalUserInputs?: ConversationMessage[];
}

function BlockHeader({
  userMessage,
  assistantMessage,
  sessionSummary,
  parrotId,
  theme,
  onToggle,
  isCollapsed,
  isStreaming,
  additionalUserInputs = [],
}: BlockHeaderProps) {
  const { t } = useTranslation();
  const userInitial = extractUserInitial(userMessage.content);

  // 计算用户输入预览文本 (按视觉宽度截取)
  // 24 视觉宽度 ≈ 12 个中文字符 或 24 个英文字符
  const userInputPreview = useMemo(() => {
    const inputs = [userMessage.content, ...additionalUserInputs.map((m) => m.content)];
    const firstLine = inputs[0].split("\n")[0];

    // 单个输入直接截取
    if (inputs.length === 1) {
      const visualWidth = getVisualWidth(firstLine);
      return visualWidth > HEADER_VISUAL_WIDTH ? truncateByVisualWidth(firstLine, HEADER_VISUAL_WIDTH) : firstLine;
    }

    // 多输入时：第一个输入截取 + 追加数量
    const truncated = truncateByVisualWidth(firstLine, HEADER_VISUAL_WIDTH - BADGE_WIDTH_OFFSET); // 预留 " +N" 空间
    if (inputs.length === 2) {
      return `${truncated} +1`;
    }
    return `${truncated} +${inputs.length - 1}`;
  }, [userMessage.content, additionalUserInputs]);

  // 计算追加输入的总数
  const totalInputCount = useMemo(() => 1 + additionalUserInputs.length, [additionalUserInputs.length]);

  // Calculate stats for Outcome Badge

  // Determine border color based on status (Status Bubbling)
  // Determine border color based on status (Status Bubbling)
  const statusBorderClass = useMemo(() => {
    if (isStreaming) return "border-l-4 border-l-blue-500/50 dark:border-l-blue-400"; // Breathing animation handled in parent
    if (assistantMessage?.error) return "border-l-4 border-l-red-500 dark:border-l-red-400";
    return "border-l-4 border-l-transparent";
  }, [isStreaming, assistantMessage]);

  // Geek Mode Summary Info
  const geekSummary = useMemo(() => {
    if (!sessionSummary || (parrotId !== "GEEK" && parrotId !== "EVOLUTION")) return null;

    const cost = sessionSummary.totalCostUSD ? `$${sessionSummary.totalCostUSD.toFixed(4)}` : "";
    const tokens =
      sessionSummary.totalInputTokens && sessionSummary.totalOutputTokens
        ? `${((sessionSummary.totalInputTokens + sessionSummary.totalOutputTokens) / 1000).toFixed(1)}k token`
        : "";
    const time = sessionSummary.totalDurationMs ? `${(sessionSummary.totalDurationMs / 1000).toFixed(1)}s` : "";

    return { cost, tokens, time };
  }, [sessionSummary, parrotId]);

  return (
    <div
      className={cn(
        "flex items-center justify-between px-4 py-2.5 select-none cursor-pointer transition-colors duration-200",
        theme.headerBg,
        statusBorderClass,
      )}
      onClick={onToggle}
    >
      {/* Left: User avatar + message preview */}
      <div className="flex items-center gap-3 flex-1 min-w-0">
        <div className="relative">
          <div className="w-7 h-7 rounded-full bg-slate-800 dark:bg-slate-300 flex items-center justify-center text-white dark:text-slate-800 text-xs font-medium shrink-0 shadow-sm">
            {userInitial}
          </div>
          {/* 追加输入计数徽章 */}
          {totalInputCount > 1 && (
            <div className="absolute -top-1 -right-1 min-w-[16px] h-4 px-0.5 rounded-full bg-blue-500 flex items-center justify-center text-[10px] font-bold text-white border-2 border-background shadow-sm">
              {totalInputCount}
            </div>
          )}
        </div>
        <div className="min-w-0 flex-1">
          <p className="text-sm font-medium text-foreground truncate" title={userMessage.content}>
            {userInputPreview}
          </p>
        </div>
      </div>

      {/* Right: Timestamp + Badge + Geek Summary + Toggle */}
      <div className="flex items-center gap-2 sm:gap-3 shrink-0 ml-1 sm:ml-2">
        {/* Geek/Evolution Summary stats - Compact View */}
        {geekSummary && (
          <>
            {/* Desktop: Full stats */}
            <div className="hidden lg:flex items-center gap-3 text-[11px] font-mono opacity-70 mr-1 bg-muted/50 px-2 py-1 rounded border border-border/50">
              <span className="font-semibold text-muted-foreground/80 uppercase tracking-wider text-[11px]">
                {t("ai.unified_block.session")}
              </span>
              {geekSummary.time && (
                <span className="flex items-center gap-1" title={t("ai.unified_block.session_duration")}>
                  <Clock className="w-3 h-3" /> {geekSummary.time}
                </span>
              )}
              {geekSummary.tokens && (
                <span className="flex items-center gap-1" title={t("ai.unified_block.session_tokens")}>
                  <Brain className="w-3 h-3" /> {geekSummary.tokens}
                </span>
              )}
              {geekSummary.cost && (
                <span className="flex items-center gap-1 text-green-600 dark:text-green-400" title={t("ai.unified_block.session_cost")}>
                  <span className="font-bold">$</span> {geekSummary.cost}
                </span>
              )}
            </div>
            {/* Mobile: Simplified cost indicator */}
            <div className="lg:hidden flex items-center gap-1 text-[10px] font-mono opacity-80">
              {geekSummary.cost && (
                <span className="flex items-center gap-0.5 text-green-600 dark:text-green-400 bg-green-50 dark:bg-green-900/20 px-1.5 py-0.5 rounded">
                  <span className="font-bold">$</span>
                  {geekSummary.cost}
                </span>
              )}
              {!geekSummary.cost && geekSummary.tokens && <span className="text-muted-foreground">{geekSummary.tokens}</span>}
            </div>
          </>
        )}

        <div className={cn("flex items-center gap-1 text-xs", theme.badgeText)}>
          <Clock className="w-3 h-3" />
          <span>{formatTime(userMessage.timestamp, t)}</span>
        </div>

        {(parrotId === "GEEK" || parrotId === "EVOLUTION" || parrotId === "AMAZING") && (
          <span className={cn("inline-flex px-2 py-0.5 rounded-full text-xs font-medium", theme.badgeBg, theme.badgeText)}>
            {parrotId === "GEEK" ? t("ai.mode.geek") : parrotId === "EVOLUTION" ? t("ai.mode.evolution") : t("ai.mode.normal")}
          </span>
        )}

        <button
          type="button"
          className={cn(
            "p-1 rounded transition-colors",
            "hover:bg-black/10 dark:hover:bg-white/10",
            "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring",
            theme.badgeText,
          )}
          onClick={(e) => {
            e.stopPropagation();
            onToggle();
          }}
          aria-label={isCollapsed ? t("common.expand") : t("common.collapse")}
          aria-expanded={!isCollapsed}
        >
          {isCollapsed ? <ChevronDown className="w-4 h-4" /> : <ChevronUp className="w-4 h-4" />}
        </button>
      </div>
    </div>
  );
}

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
      {/* Timeline Node */}
      <div className="absolute -left-8 top-1 w-6 h-6 rounded-full bg-blue-100 dark:bg-blue-900/40 border border-blue-500 flex items-center justify-center shrink-0 z-10 transition-colors group-hover:bg-blue-200 dark:group-hover:bg-blue-900/60">
        <User className="w-3.5 h-3.5 text-blue-600 dark:text-blue-400" />
      </div>

      {/* Section Header */}
      <div className="flex items-center justify-end mb-3">
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
  sessionSummary?: SessionSummary;
  isCollapsed: boolean;
  themeColors: (typeof PARROT_THEMES)[keyof typeof PARROT_THEMES];
  streamingPhase?: "thinking" | "tools" | "answer" | null;
  isLatest?: boolean;
  children?: ReactNode;
}

function BlockBody({
  userMessage,
  additionalUserInputs = [],
  assistantMessage,
  sessionSummary,
  isCollapsed,
  themeColors,
  streamingPhase = null,
  isLatest = false,
  children,
}: BlockBodyProps) {
  const { t } = useTranslation();
  const contentRef = useRef<HTMLDivElement>(null);

  // Check for error state
  const hasError = assistantMessage?.error;

  const thinkingSteps = assistantMessage?.metadata?.thinkingSteps || [];
  const toolCalls = assistantMessage?.metadata?.toolCalls || [];
  const toolResults = assistantMessage?.metadata?.toolResults || [];
  const hasAnswer = assistantMessage?.content;

  // 聚合 Thinking 内容
  const allThinkingContent = useMemo(() => {
    const placeholderTexts = ["处理中...", "Thinking...", "...", "AI is thinking", "思考中"];
    return thinkingSteps
      .map((s) => s.content?.trim() || "")
      .filter((c) => c && !placeholderTexts.some((p) => c === p || c.startsWith(p)))
      .join("\n\n");
  }, [thinkingSteps]);

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
    const callTimestamp = baseTimestamp + index * TOOL_CALL_OFFSET_MS;
    timelineEvents.push({
      type: "tool_call",
      id: `toolcall-${round}-${index}`,
      timestamp: callTimestamp,
      data: call,
    });
  });

  // 按时间戳排序
  timelineEvents.sort((a, b) => a.timestamp - b.timestamp);

  // 计算各类事件的最后一个索引（用于动效判断）
  const lastToolIndex = timelineEvents
    .map((e, i) => (e.type === "tool_call" ? i : -1))
    .filter((i) => i >= 0)
    .pop();

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

          {/* 1. Thinking Section - Simplified Collapsible Style */}
          {allThinkingContent.length > 0 && (
            <div className="relative group">
              <div
                className={cn(
                  "absolute -left-8 top-0.5 w-6 h-6 rounded-full flex items-center justify-center shrink-0 z-10 transition-colors",
                  streamingPhase === "thinking"
                    ? "bg-blue-100 dark:bg-blue-900/40 text-blue-600 dark:text-blue-400 ring-4 ring-blue-50 dark:ring-blue-900/10"
                    : "bg-muted text-muted-foreground group-hover:bg-blue-50 dark:group-hover:bg-blue-900/20 group-hover:text-blue-500",
                )}
              >
                {streamingPhase === "thinking" ? <Loader2 className="w-3.5 h-3.5 animate-spin" /> : <Brain className="w-3.5 h-3.5" />}
              </div>

              <div className="flex flex-col">
                <button
                  onClick={() => setIsThinkingExpanded(!isThinkingExpanded)}
                  className="flex items-center gap-2 text-sm font-medium text-foreground hover:text-blue-600 dark:hover:text-blue-400 transition-colors text-left w-full focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring rounded"
                >
                  <span className="flex items-center gap-2">
                    {streamingPhase === "thinking" ? (
                      <>
                        <Loader2 className="w-3.5 h-3.5 animate-spin text-blue-500" />
                        <span className="text-blue-600 dark:text-blue-400">{t("ai.states.thinking") || "思考中..."}</span>
                      </>
                    ) : (
                      <>
                        <Brain className="w-3.5 h-3.5 text-muted-foreground" />
                        <span className="text-muted-foreground">{t("ai.unified_block.thinking_process") || "思考过程"}</span>
                      </>
                    )}
                  </span>
                  <span className="ml-auto">
                    {isThinkingExpanded ? (
                      <ChevronUp className="w-4 h-4 text-muted-foreground" />
                    ) : (
                      <ChevronDown className="w-4 h-4 text-muted-foreground" />
                    )}
                  </span>
                </button>

                {/* Expanded View */}
                {isThinkingExpanded && (
                  <div className="mt-2 text-sm text-muted-foreground bg-muted/30 p-3 rounded-lg border border-border/50 animate-in fade-in slide-in-from-top-1 duration-200 prose prose-xs dark:prose-invert max-w-none">
                    <ReactMarkdown>{allThinkingContent}</ReactMarkdown>
                  </div>
                )}
              </div>
            </div>
          )}

          {/* 2. Tool Calls Stream */}
          {timelineEvents.map((event, eventIndex) => {
            if (event.type !== "tool_call") return null;

            const calling = streamingPhase === "tools" && eventIndex === lastToolIndex;
            const call = event.data;
            const callName = typeof call === "string" ? call : call.name;

            // Find result
            const result = toolResults.find(
              (r) => r.name === callName || (typeof call === "object" && call.toolId && r.toolId === call.toolId),
            );

            const isError = typeof call === "object" ? call.isError : assistantMessage?.error;
            // Determine operation type (Read vs Write)
            const isWriteOp = ["write", "edit", "bash", "run_command"].some((k) => callName.toLowerCase().includes(k));

            return (
              <div key={event.id} className="relative group">
                {/* Timeline Node */}
                <div
                  className={cn(
                    "absolute -left-8 top-0 w-6 h-6 rounded-full flex items-center justify-center shrink-0 z-10 border transition-all",
                    calling
                      ? "bg-purple-100 dark:bg-purple-900/40 border-purple-500 animate-pulse"
                      : "bg-card border-border group-hover:border-purple-400/50",
                    isError && "bg-red-50 border-red-200",
                  )}
                >
                  {calling ? (
                    <Loader2 className="w-3 h-3 text-purple-600 animate-spin" />
                  ) : (
                    <Wrench className={cn("w-3 h-3", isError ? "text-red-500" : "text-muted-foreground group-hover:text-purple-500")} />
                  )}
                </div>

                {/* Card Container */}
                <div
                  className={cn(
                    "rounded-lg border overflow-hidden transition-all duration-200",
                    "bg-card hover:shadow-sm",
                    isWriteOp ? "border-purple-200/50 dark:border-purple-800/30 bg-purple-50/10" : "border-border/50",
                    // Write operations get subtle highlighting
                  )}
                >
                  {/* Tool Header */}
                  <div className="flex items-center justify-between px-3 py-2 bg-muted/20 text-xs">
                    <div className="flex items-center gap-2">
                      <span className={cn("font-semibold", isWriteOp ? "text-purple-700 dark:text-purple-300" : "text-foreground")}>
                        {callName}
                      </span>
                      {typeof call === "object" && call.duration && (
                        <span className="text-[11px] text-muted-foreground bg-background/50 px-1.5 py-0.5 rounded">
                          {call.duration > 1000 ? `${(call.duration / 1000).toFixed(1)}s` : `${call.duration}ms`}
                        </span>
                      )}
                    </div>
                    {typeof call === "object" && call.filePath && (
                      <span className="font-mono text-[11px] text-muted-foreground truncate max-w-[150px]" title={call.filePath}>
                        {call.filePath}
                      </span>
                    )}
                  </div>

                  {/* Tool Input Preview (Argument) */}
                  {typeof call === "object" && call.inputSummary && (
                    <div className="px-3 py-2 border-t border-border/30 bg-background/50">
                      <div className="text-[11px] uppercase text-muted-foreground font-semibold mb-0.5">{t("ai.unified_block.input")}</div>
                      <code className="text-xs text-muted-foreground/80 font-mono break-all line-clamp-2 hover:line-clamp-none transition-all">
                        {call.inputSummary}
                      </code>
                    </div>
                  )}

                  {/* Tool Result (Output) */}
                  {result && result.outputSummary && (
                    <div className="px-3 py-2 border-t border-border/30 bg-muted/10">
                      <div className="flex items-center justify-between mb-1">
                        <div className="text-[11px] uppercase text-muted-foreground font-semibold flex items-center gap-1">
                          <Terminal className="w-3 h-3" /> {t("ai.unified_block.output")}
                        </div>
                        {result.isError && (
                          <span className="text-[11px] bg-red-100 text-red-600 px-1.5 rounded font-medium">{t("ai.events.error")}</span>
                        )}
                      </div>
                      <pre
                        className={cn(
                          "text-xs font-mono overflow-x-auto whitespace-pre-wrap break-words max-h-32 overflow-y-auto p-1.5 rounded bg-black/5 dark:bg-black/20 text-muted-foreground",
                          result.isError && "text-red-600/90 bg-red-50/50",
                        )}
                      >
                        {result.outputSummary}
                      </pre>
                    </div>
                  )}
                </div>
              </div>
            );
          })}

          {/* 3. AI Answer Section */}
          {/* 3. AI Answer Section */}
          {hasAnswer ? (
            <div className="relative pt-2">
              <div
                className={cn(
                  "absolute -left-8 top-3.5 w-6 h-6 rounded-full flex items-center justify-center shrink-0 z-10 transition-colors",
                  streamingPhase === "answer"
                    ? "bg-amber-100 dark:bg-amber-900/40 border border-amber-500 animate-pulse text-amber-600"
                    : "bg-amber-50 dark:bg-amber-900/20 border border-amber-200 dark:border-amber-700/50 text-amber-500",
                )}
              >
                {streamingPhase === "answer" ? <Loader2 className="w-3.5 h-3.5 animate-spin" /> : <Zap className="w-3.5 h-3.5" />}
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
                {/* Floating Copy Button */}
                <div className="absolute top-2 right-2 z-30 opacity-0 group-hover:opacity-100 transition-opacity">
                  <button
                    onClick={() => {
                      if (assistantMessage) {
                        navigator.clipboard.writeText(assistantMessage.content);
                      }
                    }}
                    className="p-1.5 rounded-md bg-background/50 hover:bg-background border border-border/50 text-muted-foreground shadow-sm transition-all focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                    title={t("common.copy")}
                  >
                    <Copy className="w-3.5 h-3.5" />
                  </button>
                </div>

                {/* Markdown content */}
                <div ref={contentRef} className="px-5 py-4">
                  <div className="prose prose-sm dark:prose-invert max-w-none break-words leading-normal font-sans text-[15px]">
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
                              className={cn("px-1.5 py-0.5 rounded-md bg-muted/80 font-mono text-xs text-secondary-foreground", className)}
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
                      {assistantMessage.content || t("ai.states.thinking")}
                    </ReactMarkdown>
                    {children}
                  </div>
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

          {/* 4. Error Section */}
          {hasError && (
            <div className="relative group">
              <div className="absolute -left-8 top-1 w-6 h-6 rounded-full bg-red-100 dark:bg-red-900/30 border border-red-500 flex items-center justify-center shrink-0 z-10 transition-colors group-hover:bg-red-200 dark:group-hover:bg-red-900/50">
                <AlertCircle className="w-3.5 h-3.5 text-red-600 dark:text-red-400" />
              </div>
              <div className="p-3 rounded-lg bg-red-50 dark:bg-red-900/10 border border-red-200 dark:border-red-800/30 text-sm">
                <p className="font-semibold text-red-700 dark:text-red-300 flex items-center gap-2">
                  {t("ai.unified_block.error_occurred")}
                </p>
                <p className="mt-1 text-red-600/80 dark:text-red-400/80 font-mono text-xs break-all">{assistantMessage.error}</p>
              </div>
            </div>
          )}
          {/* 5. Session Summary (Detailed view for all modes if present) */}
          {sessionSummary && (
            <div className="relative">
              <div className="absolute -left-8 top-1 w-6 h-6 rounded-full bg-green-100 dark:bg-green-900/30 border border-green-500 flex items-center justify-center shrink-0 z-10 transition-colors">
                <BarChart3 className="w-3.5 h-3.5 text-green-600 dark:text-green-400" />
              </div>
              <div className="pl-0">
                <ExpandedSessionSummary summary={sessionSummary} />
              </div>
            </div>
          )}
        </div>
      </div>
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
    <div className={cn("flex items-center justify-between px-4 py-2 border-t", theme.border, theme.footerBg)}>
      {/* Left: Collapse/Expand Toggle */}
      <button
        type="button"
        onClick={onToggle}
        className={cn(
          "flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-medium transition-colors",
          "hover:bg-black/10 dark:hover:bg-white/10",
          "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring",
          theme.badgeText,
        )}
      >
        {isCollapsed ? <ChevronDown className="w-3.5 h-3.5" /> : <ChevronUp className="w-3.5 h-3.5" />}
        {isCollapsed ? t("common.expand") : t("common.collapse")}
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
              "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring",
              theme.badgeText,
            )}
          >
            {t("ai.regenerate")}
          </button>
        )}
        {/* Context Pinning / Remove - Visual Only for now */}
        <button
          type="button"
          className={cn(
            "flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-medium transition-colors opacity-60 hover:opacity-100",
            "hover:bg-black/10 dark:hover:bg-white/10",
            "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:opacity-100",
            theme.badgeText,
          )}
          title={t("ai.unified_block.forget_tooltip")}
        >
          <Brain className="w-3.5 h-3.5" />
          <span className="hidden lg:inline">{t("ai.unified_block.forget")}</span>
        </button>

        <button
          type="button"
          onClick={handleCopy}
          className={cn(
            "flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-medium transition-colors",
            "hover:bg-black/10 dark:hover:bg-white/10",
            "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring",
            copied && "bg-green-100 dark:bg-green-900/30 text-green-600 dark:text-green-400",
            !copied && theme.badgeText,
          )}
        >
          {copied ? <Check className="w-3.5 h-3.5" /> : <Copy className="w-3.5 h-3.5" />}
          {copied ? t("common.copied") : t("common.copy")}
        </button>
        {onDelete && (
          <button
            type="button"
            onClick={onDelete}
            className={cn(
              "flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-medium transition-colors",
              "hover:bg-red-100 dark:hover:bg-red-900/30",
              "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-red-500",
              "text-red-600 dark:text-red-400",
            )}
          >
            {t("common.delete")}
          </button>
        )}
      </div>
    </div>
  );
}

// ============================================================================
// Main Component
// ============================================================================

export const UnifiedMessageBlock = memo(function UnifiedMessageBlock({
  userMessage,
  additionalUserInputs = [],
  assistantMessage,
  sessionSummary,
  parrotId,
  isLatest = false,
  isStreaming = false,
  streamingPhase = null,
  onCopy,
  onRegenerate,
  onDelete,
  children,
  className,
}: UnifiedMessageBlockProps) {
  const blockTheme = (parrotId && BLOCK_THEMES[parrotId]) || BLOCK_THEMES.default;
  const themeColors = PARROT_THEMES[parrotId || "AMAZING"] || PARROT_THEMES.AMAZING;

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
        // Active/Streaming state: Breathing border + Ring
        isLatest && isStreaming && `ring-2 ${blockTheme.ringColor} animate-block-pulse`,
        isLatest && !isStreaming && `ring-1 ${blockTheme.ringColor}`,
        className,
      )}
    >
      {/* Block Header - 始终显示 */}
      <div className={cn("border-b", blockTheme.border)}>
        <BlockHeader
          userMessage={userMessage}
          assistantMessage={assistantMessage}
          sessionSummary={sessionSummary}
          parrotId={parrotId}
          theme={blockTheme}
          onToggle={toggleCollapse}
          isCollapsed={collapsed}
          isStreaming={isStreaming}
          additionalUserInputs={additionalUserInputs}
        />
      </div>

      {/* Block Body - 可折叠内容 */}
      <BlockBody
        userMessage={userMessage}
        additionalUserInputs={additionalUserInputs}
        assistantMessage={assistantMessage}
        sessionSummary={sessionSummary}
        isCollapsed={collapsed}
        themeColors={themeColors}
        streamingPhase={streamingPhase}
        isLatest={isLatest}
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
