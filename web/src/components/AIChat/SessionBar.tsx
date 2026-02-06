/**
 * SessionBar - 会话汇总栏
 *
 * 聚合当前会话所有 Blocks 的统计数据：
 * - 总成本 = Σ block.cost_estimate
 * - 总 Token = Σ block.token_usage.total_tokens
 * - 总时间 = (最后更新 - 最先创建)
 *
 * 特性：
 * - 可折叠（点击折叠/展开）
 * - 实时更新（流式完成后刷新）
 * - 移动端隐藏或简化显示
 *
 * @see docs/specs/block-design/ai-chat-interface-gap-analysis.md P1-A002
 */

import { ChevronDown, ChevronUp, Coins, Timer, Wallet } from "lucide-react";
import { memo, useCallback, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { cn } from "@/lib/utils";
import type { Block as AIBlock } from "@/types/block";
import type { BlockSummary } from "@/types/parrot";

interface SessionStats {
  totalCost: number; // in USD
  totalTokens: number;
  totalDuration: number; // in milliseconds
  blockCount: number;
}

/** Conversion factor: milli-cents to USD (100000 milli-cents = 1 USD) */
const MILLI_CENTS_TO_USD = 100000;

interface SessionBarProps {
  /** Blocks to aggregate stats from */
  blocks?: AIBlock[];
  /** Optional BlockSummary for additional stats (Geek/Evolution modes) */
  blockSummary?: BlockSummary;
  /** Initial collapsed state (default: false) */
  defaultCollapsed?: boolean;
  /** Mobile-only display (simplified) */
  mobileOnly?: boolean;
  /** className for styling */
  className?: string;
}

/**
 * Calculate aggregated session statistics from blocks
 */
function calculateSessionStats(blocks: AIBlock[] | undefined, blockSummary?: BlockSummary): SessionStats {
  if (!blocks || blocks.length === 0) {
    return { totalCost: 0, totalTokens: 0, totalDuration: 0, blockCount: 0 };
  }

  let totalCost = 0;
  let totalTokens = 0;
  let minTimestamp = Number.MAX_VALUE;
  let maxTimestamp = 0;

  for (const block of blocks) {
    // Add cost estimate (from costEstimate field in milli-cents)
    if (block.costEstimate && block.costEstimate > 0) {
      totalCost += Number(block.costEstimate) / MILLI_CENTS_TO_USD;
    }

    // Add tokens from tokenUsage
    if (block.tokenUsage) {
      totalTokens += block.tokenUsage.totalTokens || 0;
    }

    // Track time range
    const created = Number(block.createdTs);
    const updated = Number(block.updatedTs);
    if (created < minTimestamp) minTimestamp = created;
    if (updated > maxTimestamp) maxTimestamp = updated;
  }

  // If we have blockSummary (from Geek/Evolution modes), use its data as well
  if (blockSummary) {
    // blockSummary might have additional stats not in blocks
    // For now, we'll rely on blocks data
  }

  const totalDuration = maxTimestamp - minTimestamp;

  return {
    totalCost,
    totalTokens,
    totalDuration,
    blockCount: blocks.length,
  };
}

/**
 * Format duration for display
 */
function formatDuration(milliseconds: number, t: (key: string) => string): string {
  if (milliseconds < 1000) {
    return t("ai.session_bar.less_than_1s") || "<1s";
  }
  const seconds = Math.floor(milliseconds / 1000);
  if (seconds < 60) {
    return `${seconds}s`;
  }
  const minutes = Math.floor(seconds / 60);
  const remainingSeconds = seconds % 60;
  if (minutes < 60) {
    return remainingSeconds > 0 ? `${minutes}m ${remainingSeconds}s` : `${minutes}m`;
  }
  const hours = Math.floor(minutes / 60);
  const remainingMinutes = minutes % 60;
  return remainingMinutes > 0 ? `${hours}h ${remainingMinutes}m` : `${hours}h`;
}

/**
 * Format token count for display
 */
function formatTokenCount(tokens: number): string {
  if (tokens < 1000) {
    return tokens.toString();
  }
  if (tokens < 1000000) {
    return `${(tokens / 1000).toFixed(1)}k`;
  }
  return `${(tokens / 1000000).toFixed(2)}M`;
}

/**
 * Format cost for display
 */
function formatCost(cost: number): string {
  if (cost < 0.01) {
    return `$${(cost * 100).toFixed(2)}¢`;
  }
  return `$${cost.toFixed(4)}`;
}

/**
 * SessionBar Component
 *
 * Displays aggregated session statistics in a collapsible bar.
 * Hidden on mobile by default (can be enabled with mobileOnly).
 */
export function SessionBar({ blocks, blockSummary, defaultCollapsed = false, mobileOnly = false, className }: SessionBarProps) {
  const { t } = useTranslation();
  const [isCollapsed, setIsCollapsed] = useState(defaultCollapsed);

  const stats = useMemo(() => calculateSessionStats(blocks, blockSummary), [blocks, blockSummary]);

  const toggleCollapse = useCallback(() => {
    setIsCollapsed((prev) => !prev);
  }, []);

  // Don't render if no blocks
  if (stats.blockCount === 0) {
    return null;
  }

  const durationDisplay = formatDuration(stats.totalDuration, t);
  const tokensDisplay = formatTokenCount(stats.totalTokens);
  const costDisplay = formatCost(stats.totalCost);

  return (
    <div
      className={cn(
        "border-b border-border/50 bg-background/95 backdrop-blur-sm transition-all duration-200",
        // Hide on mobile unless mobileOnly is true
        !mobileOnly && "hidden md:flex",
        // Collapsed state
        isCollapsed ? "h-10" : "h-auto min-h-12",
        "flex-col items-center justify-center gap-1 md:gap-2 px-3 py-2 md:py-1.5",
        className,
      )}
    >
      {/* Header with toggle */}
      <button
        type="button"
        onClick={toggleCollapse}
        className="flex items-center gap-2 text-xs text-muted-foreground hover:text-foreground transition-colors cursor-pointer w-full justify-center"
        aria-label={isCollapsed ? t("ai.session_bar.expand") : t("ai.session_bar.collapse")}
      >
        <span className="font-medium">{t("ai.session_bar.title")}</span>
        <span className="text-muted-foreground/50">•</span>
        <span>
          {stats.blockCount} {t("ai.session_bar.blocks", { count: stats.blockCount })}
        </span>
        {isCollapsed ? <ChevronDown className="w-3.5 h-3.5" /> : <ChevronUp className="w-3.5 h-3.5" />}
      </button>

      {/* Stats - hidden when collapsed */}
      {!isCollapsed && (
        <div className="flex items-center justify-center gap-4 md:gap-6 text-xs flex-wrap">
          {/* Cost */}
          <div
            className="flex items-center gap-1.5 text-muted-foreground hover:text-foreground transition-colors"
            title={t("ai.session_bar.total_cost_tooltip")}
          >
            <Wallet className="w-3.5 h-3.5 text-green-500" />
            <span className="font-medium tabular-nums">{costDisplay}</span>
          </div>

          {/* Tokens */}
          <div
            className="flex items-center gap-1.5 text-muted-foreground hover:text-foreground transition-colors"
            title={t("ai.session_bar.total_tokens_tooltip")}
          >
            <Coins className="w-3.5 h-3.5 text-amber-500" />
            <span className="font-medium tabular-nums">{tokensDisplay}</span>
            <span className="text-muted-foreground/70">{t("ai.session_bar.tokens")}</span>
          </div>

          {/* Duration */}
          <div
            className="flex items-center gap-1.5 text-muted-foreground hover:text-foreground transition-colors"
            title={t("ai.session_bar.total_duration_tooltip")}
          >
            <Timer className="w-3.5 h-3.5 text-blue-500" />
            <span className="font-medium tabular-nums">{durationDisplay}</span>
          </div>
        </div>
      )}
    </div>
  );
}

/**
 * SessionBarMobile - Simplified mobile version
 *
 * Always visible on mobile, shows minimal info.
 */
export const SessionBarMobile = memo(function SessionBarMobile(props: Omit<SessionBarProps, "mobileOnly">) {
  return <SessionBar {...props} mobileOnly={true} defaultCollapsed={true} />;
});

/**
 * Hook: useSessionStats
 *
 * Calculate session statistics from blocks
 */
export function useSessionStats(blocks: AIBlock[] | undefined, blockSummary?: BlockSummary): SessionStats {
  return useMemo(() => calculateSessionStats(blocks, blockSummary), [blocks, blockSummary]);
}

export default memo(SessionBar);
