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

import { AlertCircle, BarChart3, Brain, Check, ChevronDown, ChevronUp, Clock, Copy, Terminal, Wrench, Zap } from "lucide-react";
import { memo, ReactNode, useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import ReactMarkdown from "react-markdown";
import remarkBreaks from "remark-breaks";
import remarkGfm from "remark-gfm";
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

function getDefaultCollapseState(isLatest: boolean, isStreaming: boolean): boolean {
  if (isStreaming || isLatest) return false;
  return true;
}

const MAX_MESSAGE_HEIGHT = 200;

// ============================================================================
// Sub-Components
// ============================================================================

interface BlockHeaderProps {
  userMessage: ConversationMessage;
  parrotId?: ParrotAgentType;
  theme: (typeof BLOCK_THEMES)[keyof typeof BLOCK_THEMES];
  onToggle: () => void;
  isCollapsed: boolean;
}

function BlockHeader({ userMessage, parrotId, theme, onToggle, isCollapsed }: BlockHeaderProps) {
  const { t } = useTranslation();
  const userInitial = extractUserInitial(userMessage.content);

  return (
    <div className={cn("flex items-center justify-between px-4 py-2.5 select-none cursor-pointer", theme.headerBg)} onClick={onToggle}>
      {/* Left: User avatar + message preview */}
      <div className="flex items-center gap-3 flex-1 min-w-0">
        <div className="w-7 h-7 rounded-full bg-slate-800 dark:bg-slate-300 flex items-center justify-center text-white dark:text-slate-800 text-xs font-medium shrink-0">
          {userInitial}
        </div>
        <p className="text-sm font-medium text-foreground truncate">
          {userMessage.content.slice(0, 60)}
          {userMessage.content.length > 60 ? "..." : ""}
        </p>
      </div>

      {/* Right: Timestamp + Badge + Toggle */}
      <div className="flex items-center gap-3 shrink-0">
        <div className={cn("flex items-center gap-1 text-xs", theme.badgeText)}>
          <Clock className="w-3 h-3" />
          <span>{formatTime(userMessage.timestamp, t)}</span>
        </div>

        {(parrotId === "GEEK" || parrotId === "EVOLUTION") && (
          <span className={cn("px-2 py-0.5 rounded-full text-xs font-medium", theme.badgeBg, theme.badgeText)}>
            {parrotId === "GEEK" ? "Geek" : "Evolution"}
          </span>
        )}

        <button
          type="button"
          className={cn("p-1 rounded transition-colors", "hover:bg-black/10 dark:hover:bg-white/10", theme.badgeText)}
          onClick={(e) => e.stopPropagation()}
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
  themeColors: (typeof PARROT_THEMES)[keyof typeof PARROT_THEMES];
  children?: ReactNode;
}

function BlockBody({ assistantMessage, sessionSummary, isCollapsed, themeColors, children }: BlockBodyProps) {
  const { t } = useTranslation();
  const contentRef = useRef<HTMLDivElement>(null);
  const [isFolded, setIsFolded] = useState(false);
  const [shouldShowFold, setShouldShowFold] = useState(false);

  // Detect height for auto-folding
  useEffect(() => {
    if (contentRef.current && assistantMessage?.content && !isCollapsed) {
      const height = contentRef.current.scrollHeight;
      if (height > MAX_MESSAGE_HEIGHT) {
        setShouldShowFold(true);
      } else {
        setShouldShowFold(false);
      }
    }
  }, [assistantMessage?.content, isCollapsed]);

  // Check for error state
  const hasError = assistantMessage?.error;

  // 按轮次分组数据
  const thinkingSteps = assistantMessage?.metadata?.thinkingSteps || [];
  const toolCalls = assistantMessage?.metadata?.toolCalls || [];
  const toolResults = assistantMessage?.metadata?.toolResults || [];
  const hasAnswer = assistantMessage?.content;
  const hasSessionSummary =
    sessionSummary &&
    ((sessionSummary.totalDurationMs || 0) > 0 ||
      (sessionSummary.totalInputTokens || 0) + (sessionSummary.totalOutputTokens || 0) > 0 ||
      (sessionSummary.totalCostUSD || 0) > 0);

  // 计算最大轮次
  const maxRound = Math.max(
    0,
    ...thinkingSteps.map((s) => s.round),
    ...toolCalls.map((c) => c.round || 0),
    ...toolResults.map((r) => r.round || 0),
  );

  // 按轮次分组
  const rounds = Array.from({ length: maxRound + 1 }, (_, i) => {
    const step = thinkingSteps.find((s) => s.round === i);
    const roundToolCalls = toolCalls.filter((c) => (c.round || 0) === i);
    const roundToolResults = toolResults.filter((r) => (r.round || 0) === i);
    return { round: i, step, toolCalls: roundToolCalls, toolResults: roundToolResults };
  });

  // When collapsed, show minimal info
  if (isCollapsed) {
    return <div className="px-4 py-2 text-sm text-muted-foreground italic">{t("ai.collapsed") || "Click expand to view details"}</div>;
  }

  return (
    <div className="px-4 py-3">
      {/* Timeline Flow */}
      <div className="relative">
        {/* Left Timeline Line */}
        <div className="absolute left-[11px] top-0 bottom-0 w-0.5 bg-border" />

        <div className="relative pl-6 space-y-4">
          {/* 渲染每一轮思考 */}
          {rounds.map((round) => {
            const hasRoundContent = round.step || round.toolCalls.length > 0 || round.toolResults.length > 0;
            if (!hasRoundContent) return null;

            const roundNumber = round.round;
            const isMultiRound = maxRound > 0;

            return (
              <div key={round.round} className="space-y-3">
                {/* 轮次标题（如果有多个轮次） */}
                {isMultiRound && (
                  <div className="flex items-center gap-2 pl-2">
                    <span className="text-[10px] font-medium text-muted-foreground uppercase tracking-wide">Round {roundNumber + 1}</span>
                    <div className="flex-1 h-px bg-border/50" />
                  </div>
                )}

                {/* AI Thinking */}
                {round.step && (
                  <div className="relative">
                    <div className="absolute -left-6 top-1.5 w-5 h-5 rounded-full bg-blue-100 dark:bg-blue-900/30 border-2 border-blue-500 flex items-center justify-center shrink-0">
                      <Brain className="w-2.5 h-2.5 text-blue-600 dark:text-blue-400" />
                    </div>
                    <div className="pl-2">
                      <div className="inline-flex items-center gap-2 px-3 py-1.5 rounded-lg bg-blue-50/50 dark:bg-blue-900/20 border border-blue-200/50 dark:border-blue-700/30">
                        <span className="text-xs font-semibold text-blue-700 dark:text-blue-300">AI Thinking</span>
                      </div>
                      <p className="mt-2 text-sm text-muted-foreground italic">{round.step.content}</p>
                    </div>
                  </div>
                )}

                {/* Tools (tool_use → tool_result pairs) for this round */}
                {round.toolCalls.length > 0 || round.toolResults.length > 0 ? (
                  <div className="relative space-y-3">
                    <div className="absolute -left-6 top-0 w-5 h-5 rounded-full bg-purple-100 dark:bg-purple-900/30 border-2 border-purple-500 flex items-center justify-center shrink-0">
                      <Wrench className="w-2.5 h-2.5 text-purple-600 dark:text-purple-400" />
                    </div>

                    <div className="pl-2">
                      <div className="inline-flex items-center gap-2 px-3 py-1.5 rounded-lg bg-purple-50/50 dark:bg-purple-900/20 border border-purple-200/50 dark:border-purple-700/30 mb-3">
                        <span className="text-xs font-semibold text-purple-700 dark:text-purple-300">Tools</span>
                        <span className="text-xs text-muted-foreground">
                          {round.toolCalls.length + round.toolResults.length} operations
                        </span>
                      </div>

                      {/* Tool pairs: tool_use → tool_result */}
                      <div className="space-y-3">
                        {round.toolCalls.map((call: ToolCall, callIndex) => {
                          const callName = typeof call === "string" ? call : call.name;
                          // Find corresponding result in this round
                          const result = round.toolResults.find(
                            (r) => r.name === callName || (typeof call === "object" && call.toolId && r.toolId === call.toolId),
                          );
                          const isError = typeof call === "object" ? call.isError : assistantMessage?.error;

                          return (
                            <div key={callIndex} className="relative pl-4 border-l-2 border-purple-200 dark:border-purple-700/50">
                              {/* Tool Use */}
                              <div className="flex items-center gap-2 mb-1.5">
                                <div
                                  className={cn(
                                    "w-5 h-5 rounded flex items-center justify-center shrink-0",
                                    isError
                                      ? "bg-red-100 dark:bg-red-900/30 text-red-600"
                                      : "bg-slate-100 dark:bg-slate-800 text-slate-600",
                                  )}
                                >
                                  <Terminal className="w-3 h-3" />
                                </div>
                                <div className="flex-1">
                                  <div className="flex items-center gap-2">
                                    <span className="text-xs font-medium">{callName}</span>
                                    {typeof call === "object" && call.duration && (
                                      <span className="text-[10px] text-muted-foreground">
                                        {call.duration > 1000 ? `${(call.duration / 1000).toFixed(1)}s` : `${call.duration}ms`}
                                      </span>
                                    )}
                                    {typeof call === "object" && call.inputSummary && (
                                      <span className="text-[10px] text-muted-foreground truncate max-w-[200px]" title={call.inputSummary}>
                                        {call.inputSummary}
                                      </span>
                                    )}
                                  </div>
                                  {typeof call === "object" && call.filePath && (
                                    <div className="text-[10px] text-muted-foreground font-mono bg-muted/50 px-2 py-0.5 rounded mt-0.5">
                                      {call.filePath}
                                    </div>
                                  )}
                                </div>
                              </div>

                              {/* Tool Result (if available) */}
                              {result && result.outputSummary && result.outputSummary.length > 0 && (
                                <div className="mt-2">
                                  <div className="flex items-center gap-2 text-[10px] text-muted-foreground mb-1">
                                    <ChevronDown className="w-3 h-3" />
                                    <span>
                                      Output (
                                      {result.duration && result.duration > 1000
                                        ? `${(result.duration / 1000).toFixed(1)}s`
                                        : `${result.duration || 0}ms`}
                                      )
                                    </span>
                                  </div>
                                  <div className="rounded-lg bg-slate-950 dark:bg-slate-950 border border-slate-800 overflow-hidden">
                                    <div className="px-3 py-1 bg-slate-900/50 border-b border-slate-800 flex items-center gap-2">
                                      <div className="flex gap-1">
                                        <div className="w-1.5 h-1.5 rounded-full bg-red-500/80" />
                                        <div className="w-1.5 h-1.5 rounded-full bg-yellow-500/80" />
                                        <div className="w-1.5 h-1.5 rounded-full bg-green-500/80" />
                                      </div>
                                      <span className="text-[10px] text-slate-500 font-mono">Output</span>
                                    </div>
                                    <pre className="p-2.5 text-xs text-slate-300 font-mono overflow-x-auto whitespace-pre-wrap break-words max-h-48 overflow-y-auto">
                                      {result.outputSummary}
                                    </pre>
                                  </div>
                                </div>
                              )}

                              {/* Waiting for result indicator (if no result yet) */}
                              {!result && (
                                <div className="mt-2 flex items-center gap-2 text-[10px] text-muted-foreground">
                                  <div className="w-2 h-2 rounded-full bg-slate-400 animate-pulse" />
                                  <span>Waiting for result...</span>
                                </div>
                              )}
                            </div>
                          );
                        })}
                      </div>
                    </div>
                  </div>
                ) : null}
              </div>
            );
          })}

          {/* 2. Tools (tool_use → tool_result pairs) */}
          {toolCalls.length > 0 || toolResults.length > 0 ? (
            <div className="relative space-y-3">
              {/* Tool indicator header */}
              <div className="absolute -left-6 top-0 w-5 h-5 rounded-full bg-purple-100 dark:bg-purple-900/30 border-2 border-purple-500 flex items-center justify-center shrink-0">
                <Wrench className="w-2.5 h-2.5 text-purple-600 dark:text-purple-400" />
              </div>

              <div className="pl-2">
                <div className="inline-flex items-center gap-2 px-3 py-1.5 rounded-lg bg-purple-50/50 dark:bg-purple-900/20 border border-purple-200/50 dark:border-purple-700/30 mb-3">
                  <span className="text-xs font-semibold text-purple-700 dark:text-purple-300">Tools</span>
                  <span className="text-xs text-muted-foreground">{toolCalls.length + toolResults.length} operations</span>
                </div>

                {/* Tool pairs: tool_use → tool_result */}
                <div className="space-y-3">
                  {toolCalls.map((call: ToolCall, callIndex) => {
                    const callName = typeof call === "string" ? call : call.name;
                    // Find corresponding result
                    const result = toolResults.find(
                      (r) => r.name === callName || (typeof call === "object" && call.toolId && r.toolId === call.toolId),
                    );
                    const isError = typeof call === "object" ? call.isError : assistantMessage?.error;

                    return (
                      <div key={callIndex} className="relative pl-4 border-l-2 border-purple-200 dark:border-purple-700/50">
                        {/* Tool Use */}
                        <div className="flex items-center gap-2 mb-1.5">
                          <div
                            className={cn(
                              "w-5 h-5 rounded flex items-center justify-center shrink-0",
                              isError ? "bg-red-100 dark:bg-red-900/30 text-red-600" : "bg-slate-100 dark:bg-slate-800 text-slate-600",
                            )}
                          >
                            <Terminal className="w-3 h-3" />
                          </div>
                          <div className="flex-1">
                            <div className="flex items-center gap-2">
                              <span className="text-xs font-medium">{callName}</span>
                              {typeof call === "object" && call.duration && (
                                <span className="text-[10px] text-muted-foreground">
                                  {call.duration > 1000 ? `${(call.duration / 1000).toFixed(1)}s` : `${call.duration}ms`}
                                </span>
                              )}
                              {typeof call === "object" && call.inputSummary && (
                                <span className="text-[10px] text-muted-foreground truncate max-w-[200px]" title={call.inputSummary}>
                                  {call.inputSummary}
                                </span>
                              )}
                            </div>
                            {typeof call === "object" && call.filePath && (
                              <div className="text-[10px] text-muted-foreground font-mono bg-muted/50 px-2 py-0.5 rounded mt-0.5">
                                {call.filePath}
                              </div>
                            )}
                          </div>
                        </div>

                        {/* Tool Result (if available) */}
                        {result && result.outputSummary && result.outputSummary.length > 0 && (
                          <div className="mt-2">
                            <div className="flex items-center gap-2 text-[10px] text-muted-foreground mb-1">
                              <ChevronDown className="w-3 h-3" />
                              <span>
                                Output (
                                {result.duration && result.duration > 1000
                                  ? `${(result.duration / 1000).toFixed(1)}s`
                                  : `${result.duration || 0}ms`}
                                )
                              </span>
                            </div>
                            <div className="rounded-lg bg-slate-950 dark:bg-slate-950 border border-slate-800 overflow-hidden">
                              <div className="px-3 py-1 bg-slate-900/50 border-b border-slate-800 flex items-center gap-2">
                                <div className="flex gap-1">
                                  <div className="w-1.5 h-1.5 rounded-full bg-red-500/80" />
                                  <div className="w-1.5 h-1.5 rounded-full bg-yellow-500/80" />
                                  <div className="w-1.5 h-1.5 rounded-full bg-green-500/80" />
                                </div>
                                <span className="text-[10px] text-slate-500 font-mono">Output</span>
                              </div>
                              <pre className="p-2.5 text-xs text-slate-300 font-mono overflow-x-auto whitespace-pre-wrap break-words max-h-48 overflow-y-auto">
                                {result.outputSummary}
                              </pre>
                            </div>
                          </div>
                        )}

                        {/* Waiting for result indicator (if no result yet) */}
                        {!result && (
                          <div className="mt-2 flex items-center gap-2 text-[10px] text-muted-foreground">
                            <div className="w-2 h-2 rounded-full bg-slate-400 animate-pulse" />
                            <span>Waiting for result...</span>
                          </div>
                        )}
                      </div>
                    );
                  })}
                </div>
              </div>
            </div>
          ) : null}

          {/* 3. AI Answer */}
          {hasAnswer && (
            <div className="relative">
              {/* Timeline Dot */}
              <div className="absolute -left-6 top-1.5 w-5 h-5 rounded-full bg-amber-100 dark:bg-amber-900/30 border-2 border-amber-500 flex items-center justify-center shrink-0">
                <Zap className="w-2.5 h-2.5 text-amber-600 dark:text-amber-400" />
              </div>
              {/* Content */}
              <div className="pl-2">
                <div className="inline-flex items-center gap-2 px-3 py-1.5 rounded-lg bg-amber-50/50 dark:bg-amber-900/20 border border-amber-200/50 dark:border-amber-700/30 mb-2">
                  <span className="text-xs font-semibold text-amber-700 dark:text-amber-300">AI Answer</span>
                </div>
                {/* Message bubble with Markdown */}
                <div
                  className={cn(
                    "relative rounded-2xl shadow-sm transition-colors min-w-[120px] max-w-full",
                    themeColors.bubbleBg,
                    themeColors.bubbleBorder,
                    themeColors.text,
                    shouldShowFold && isFolded && "overflow-hidden",
                  )}
                  style={shouldShowFold && isFolded ? { maxHeight: `${MAX_MESSAGE_HEIGHT}px` } : {}}
                >
                  {/* Floating Copy Button */}
                  <div className="absolute top-2 right-2 z-30">
                    <button
                      onClick={() => {
                        if (assistantMessage) {
                          navigator.clipboard.writeText(assistantMessage.content);
                        }
                      }}
                      className={cn(
                        "p-1.5 rounded-lg border shadow-sm transition-all active:scale-90",
                        "bg-card/50 border-border text-muted-foreground hover:text-foreground backdrop-blur-sm",
                      )}
                      title={t("common.copy") || "Copy"}
                    >
                      <Copy className="w-3.5 h-3.5" />
                    </button>
                  </div>

                  {/* Markdown content */}
                  <div ref={contentRef} className="pl-4 pr-10 py-2.5">
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
                            <blockquote
                              {...props}
                              className="border-l-4 border-primary/30 pl-4 py-1 my-2 bg-muted/30 italic rounded-r-lg"
                            />
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
                        {assistantMessage.content || t("ai.states.thinking") || "..."}
                      </ReactMarkdown>
                      {children}
                    </div>
                  </div>

                  {/* Fold Mask and Button */}
                  {shouldShowFold && (
                    <>
                      {isFolded && (
                        <div className="absolute inset-x-0 bottom-0 h-16 bg-gradient-to-t from-card via-card/40 to-transparent pointer-events-none" />
                      )}
                      <div className={cn("flex justify-center p-1.5", isFolded ? "absolute bottom-0 inset-x-0 z-10" : "relative")}>
                        <button
                          onClick={() => setIsFolded((prev) => !prev)}
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
              </div>
            </div>
          )}

          {/* 4. Error */}
          {hasError && (
            <div className="relative">
              {/* Timeline Dot */}
              <div className="absolute -left-6 top-1.5 w-5 h-5 rounded-full bg-red-100 dark:bg-red-900/30 border-2 border-red-500 flex items-center justify-center shrink-0">
                <AlertCircle className="w-2.5 h-2.5 text-red-600 dark:text-red-400" />
              </div>
              {/* Content */}
              <div className="pl-2">
                <div className="inline-flex items-center gap-2 px-3 py-1.5 rounded-lg bg-red-50/50 dark:bg-red-900/20 border border-red-200/50 dark:border-red-700/30">
                  <span className="text-xs font-semibold text-red-700 dark:text-red-300">Error</span>
                </div>
                <p className="mt-2 text-sm text-red-600 dark:text-red-400">{assistantMessage.error}</p>
              </div>
            </div>
          )}

          {/* 5. SessionSummary */}
          {hasSessionSummary && (
            <div className="relative">
              {/* Timeline Dot */}
              <div className="absolute -left-6 top-1.5 w-5 h-5 rounded-full bg-green-100 dark:bg-green-900/30 border-2 border-green-500 flex items-center justify-center shrink-0">
                <BarChart3 className="w-2.5 h-2.5 text-green-600 dark:text-green-400" />
              </div>
              {/* Content */}
              <div className="pl-2">
                <div className="inline-flex items-center gap-2 px-3 py-1.5 rounded-lg bg-green-50/50 dark:bg-green-900/20 border border-green-200/50 dark:border-green-700/30 mb-2">
                  <span className="text-xs font-semibold text-green-700 dark:text-green-300">Session Summary</span>
                </div>
                {/* ExpandedSessionSummary embedded */}
                <div className="pl-2">
                  <ExpandedSessionSummary summary={sessionSummary} />
                </div>
              </div>
            </div>
          )}
        </div>
      </div>

      {/* Custom children (typing cursor, etc.) */}
      {children}
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
  const themeColors = PARROT_THEMES[parrotId || "AMAZING"] || PARROT_THEMES.AMAZING;

  const [collapsed, setCollapsed] = useState(() => getDefaultCollapseState(isLatest, isStreaming));

  useEffect(() => {
    setCollapsed(getDefaultCollapseState(isLatest, isStreaming));
  }, [isLatest, isStreaming]);

  const toggleCollapse = useCallback(() => {
    setCollapsed((prev) => !prev);
  }, []);

  // Build content for copying
  const contentForCopy = useMemo(
    () =>
      [
        `User: ${userMessage.content}`,
        assistantMessage?.content ? `Assistant: ${assistantMessage.content}` : "",
        assistantMessage?.metadata?.toolResults
          ? `\n\nTools:\n${assistantMessage.metadata.toolResults.map((r) => `- ${r.name}: ${r.duration}ms`).join("\n")}`
          : "",
      ]
        .filter(Boolean)
        .join("\n\n"),
    [userMessage.content, assistantMessage?.content, assistantMessage?.metadata?.toolResults],
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
        <BlockHeader userMessage={userMessage} parrotId={parrotId} theme={blockTheme} onToggle={toggleCollapse} isCollapsed={collapsed} />
      </div>

      {/* Block Body - 可折叠内容 */}
      <BlockBody assistantMessage={assistantMessage} sessionSummary={sessionSummary} isCollapsed={collapsed} themeColors={themeColors}>
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
