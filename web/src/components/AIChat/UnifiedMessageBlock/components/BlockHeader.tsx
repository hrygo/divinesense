/**
 * BlockHeader Component - Optimized Version
 *
 * Two-column responsive layout:
 * - Left: Avatar + message preview + Block number
 * - Right: Stats + Badge + Toggle
 *
 * Phase 1: Visual Hierarchy
 * Phase 2: Responsive Experience
 * Phase 5: React.memo optimization
 */

import { ChevronDown, ChevronUp, Clock, Hash, Wrench } from "lucide-react";
import { memo, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { cn } from "@/lib/utils";
import type { AIMode, ConversationMessage } from "@/types/aichat";
import type { BlockSummary, ParrotAgentType } from "@/types/parrot";
import { formatRelativeTime, getVisualWidth, truncateByVisualWidth } from "../utils";

export interface BlockHeaderTheme {
  border: string;
  headerBg: string;
  footerBg: string;
  badgeBg: string;
  badgeText: string;
  ringColor: string;
}

export interface BlockHeaderProps {
  userMessage: ConversationMessage;
  assistantMessage?: ConversationMessage;
  blockSummary?: BlockSummary;
  parrotId?: ParrotAgentType;
  theme: BlockHeaderTheme;
  onToggle: () => void;
  isCollapsed: boolean;
  isStreaming?: boolean;
  additionalUserInputs?: ConversationMessage[];
  /** Block sequence number (1-based) for display */
  blockNumber?: number;
}

/**
 * Extract user initial from content
 */
function extractUserInitial(content: string): string {
  const trimmed = content.trim();
  if (trimmed.length === 0) return "U";
  const match = trimmed.match(/[a-zA-Z\u4e00-\u9fa5]/);
  return match ? match[0].toUpperCase() : "U";
}

/**
 * Memo comparison for BlockHeader
 */
const areBlockHeaderPropsEqual = (prev: BlockHeaderProps, next: BlockHeaderProps): boolean => {
  return (
    prev.userMessage.id === next.userMessage.id &&
    prev.userMessage.content === next.userMessage.content &&
    prev.isCollapsed === next.isCollapsed &&
    prev.isStreaming === next.isStreaming &&
    prev.additionalUserInputs?.length === next.additionalUserInputs?.length &&
    prev.blockNumber === next.blockNumber
  );
};

/**
 * BlockNumberBadge - Simple block number display
 */
interface BlockNumberBadgeProps {
  blockNumber: number;
  isActive?: boolean;
}

function BlockNumberBadge({ blockNumber, isActive }: BlockNumberBadgeProps) {
  return (
    <div
      className={cn(
        "flex items-center gap-1 px-1.5 py-0.5 rounded text-[11px] font-medium",
        "bg-slate-100 dark:bg-slate-800/50",
        "text-slate-600 dark:text-slate-400",
        "border border-slate-200 dark:border-slate-700/50",
        isActive && "ring-1 ring-slate-400 dark:ring-slate-600",
      )}
      title={`Block ${blockNumber}`}
    >
      <Hash className="w-3 h-3 shrink-0" />
      <span className="font-mono">{blockNumber}</span>
    </div>
  );
}

/**
 * BlockHeader component
 *
 * Optimized with React.memo and responsive two-column layout.
 * Simplified to show block number instead of branch path.
 */
export const BlockHeader = memo(function BlockHeader({
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
  const { t } = useTranslation();
  const userInitial = extractUserInitial(userMessage.content);

  // Calculate user input preview
  const userInputPreview = useMemo(() => {
    const inputs = [userMessage.content, ...additionalUserInputs.map((m) => m.content)];
    const firstLine = inputs[0].split("\n")[0];

    // 24 visual width ≈ 12 Chinese or 24 English characters
    const HEADER_VISUAL_WIDTH = 24;
    const BADGE_WIDTH_OFFSET = 4;

    if (inputs.length === 1) {
      const visualWidth = getVisualWidth(firstLine);
      return visualWidth > HEADER_VISUAL_WIDTH ? truncateByVisualWidth(firstLine, HEADER_VISUAL_WIDTH) : firstLine;
    }

    // Multi-input: truncate first + count
    const truncated = truncateByVisualWidth(firstLine, HEADER_VISUAL_WIDTH - BADGE_WIDTH_OFFSET);
    if (inputs.length === 2) {
      return `${truncated} +1`;
    }
    return `${truncated} +${inputs.length - 1}`;
  }, [userMessage.content, additionalUserInputs]);

  const totalInputCount = useMemo(() => 1 + additionalUserInputs.length, [additionalUserInputs.length]);

  // Status border class
  const statusBorderClass = cn(
    "border-l-4",
    isStreaming
      ? "border-l-blue-500/50 dark:border-l-blue-400"
      : assistantMessage?.error
        ? "border-l-red-500 dark:border-l-red-400"
        : "border-l-transparent",
  );

  // Mode-specific summary with comprehensive stats
  const modeSummary = useMemo(() => {
    if (!blockSummary) return null;

    const currentMode: AIMode =
      userMessage.metadata?.mode || (parrotId === "GEEK" ? "geek" : parrotId === "EVOLUTION" ? "evolution" : "normal");

    const formatCost = (cost?: number) => (cost ? `${t("ai.session_stats.currency_symbol")}${cost.toFixed(4)}` : "");
    const formatTokens = (input?: number, output?: number) => {
      if (input && output) return `${((input + output) / 1000).toFixed(1)}k`;
      return "";
    };
    const formatTime = (ms?: number) => (ms ? `${(ms / 1000).toFixed(1)}s` : "");

    // Build stats array for flexible display
    type StatItem = { value: string; icon?: string; title?: string; className?: string };
    const stats: StatItem[] = [];

    // Duration (always show if available)
    if (blockSummary.totalDurationMs) {
      stats.push({
        value: formatTime(blockSummary.totalDurationMs),
        icon: "clock",
        title: t("ai.unified_block.session_duration"),
      });
    }

    // Tool calls (show if > 0)
    if (blockSummary.toolCallCount && blockSummary.toolCallCount > 0) {
      stats.push({
        value: `${blockSummary.toolCallCount}`,
        icon: "wrench",
        title: t("ai.stats.tool_calls"),
      });
    }

    // Tokens (for normal mode)
    if (currentMode === "normal") {
      const tokens = formatTokens(blockSummary.totalInputTokens, blockSummary.totalOutputTokens);
      if (tokens) {
        stats.push({
          value: tokens,
          icon: "token",
          title: t("ai.unified_block.session_tokens"),
          className: "text-amber-500",
        });
      }
    }

    // Cost (always show if available)
    const cost = formatCost(blockSummary.totalCostUSD);
    if (cost) {
      stats.push({
        value: cost,
        icon: "cost",
        title: t("ai.unified_block.session_cost"),
        className: "text-green-600 dark:text-green-400",
      });
    }

    // Files modified (for evolution mode)
    if (currentMode === "evolution" && blockSummary.filesModified) {
      stats.push({
        value: `${blockSummary.filesModified} ${t("ai.stats.files_modified_short")}`,
        icon: "file",
        title: t("ai.stats.files_modified"),
      });
    }

    return { stats };
  }, [blockSummary, userMessage.metadata?.mode, parrotId, t]);

  return (
    <div
      className={cn(
        "flex items-center justify-between px-4 py-2.5 select-none cursor-pointer transition-colors duration-200",
        theme.headerBg,
        statusBorderClass,
      )}
      onClick={onToggle}
    >
      {/* Left: Avatar + message preview + Block number */}
      <div className="flex items-center gap-3 flex-1 min-w-0">
        <div className="relative">
          <div className="w-7 h-7 rounded-full bg-slate-800 dark:bg-slate-300 flex items-center justify-center text-white dark:text-slate-800 text-xs font-medium shrink-0 shadow-sm">
            {userInitial}
          </div>
          {/* Badge for multiple inputs */}
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

        {/* Block Number Badge */}
        {blockNumber && blockNumber > 0 && <BlockNumberBadge blockNumber={blockNumber} isActive={isStreaming} />}
      </div>

      {/* Right: Stats + Badge + Toggle */}
      <div className="flex items-center gap-2 sm:gap-3 shrink-0 ml-1 sm:ml-2">
        {/* Mode-specific Session Summary - Responsive */}
        {modeSummary && modeSummary.stats.length > 0 && (
          <>
            {/* Desktop (≥ 1024px): Full stats */}
            <div className="hidden lg:flex items-center gap-2 text-[11px] font-mono opacity-70 mr-1 bg-muted/50 px-2 py-1 rounded border border-border/50">
              {modeSummary.stats.map((stat, idx) => (
                <span key={idx} className={cn("flex items-center gap-1", stat.className)} title={stat.title}>
                  {stat.icon === "clock" && <Clock className="w-3 h-3" />}
                  {stat.icon === "wrench" && <Wrench className="w-3 h-3" />}
                  {stat.icon === "token" && <span className="text-amber-500">⚡</span>}
                  {stat.icon === "cost" && <span className="font-bold">$</span>}
                  {stat.value}
                </span>
              ))}
            </div>

            {/* Mobile (< 1024px): Show first 2 stats */}
            <div className="lg:hidden flex items-center gap-1 text-[10px] font-mono opacity-80">
              {modeSummary.stats.slice(0, 2).map((stat, idx) => (
                <span
                  key={idx}
                  className={cn(
                    "flex items-center gap-0.5 px-1.5 py-0.5 rounded",
                    stat.className ? stat.className : "text-muted-foreground bg-muted/50",
                  )}
                  title={stat.title}
                >
                  {stat.icon === "clock" && <Clock className="w-3 h-3" />}
                  {stat.icon === "wrench" && <Wrench className="w-3 h-3" />}
                  {stat.icon === "cost" && <span className="font-bold">$</span>}
                  {stat.value}
                </span>
              ))}
            </div>
          </>
        )}

        {/* Timestamp */}
        <div className={cn("flex items-center gap-1 text-xs", theme.badgeText)}>
          <Clock className="w-3 h-3" />
          <span className="hidden sm:inline">{formatRelativeTime(userMessage.timestamp, t)}</span>
          <span className="sm:hidden">{formatRelativeTime(userMessage.timestamp, t)}</span>
        </div>

        {/* Parrot Badge - hidden on mobile */}
        {(parrotId === "GEEK" || parrotId === "EVOLUTION" || parrotId === "AMAZING") && (
          <span className={cn("hidden sm:inline-flex px-2 py-0.5 rounded-full text-xs font-medium", theme.badgeBg, theme.badgeText)}>
            {parrotId === "GEEK" ? t("ai.mode.geek") : parrotId === "EVOLUTION" ? t("ai.mode.evolution") : t("ai.mode.normal")}
          </span>
        )}

        {/* Toggle button */}
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
}, areBlockHeaderPropsEqual);
