/**
 * HeaderSessionStats - Header 内嵌的会话统计组件
 *
 * 展示当前会话的统计数据（成本、Tokens、耗时）
 * 根据 Mode 差异化展示内容
 * PC 端集成到 ChatHeader，移动端使用独立的 SessionBar
 *
 * @see docs/specs/block-design/ai-chat-interface-gap-analysis.md
 */

import { Brain, Clock, Wrench } from "lucide-react";
import { memo, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { cn } from "@/lib/utils";
import type { AIMode } from "@/types/aichat";
import type { Block as AIBlock } from "@/types/block";

interface SessionStatsData {
  primary: string;
  secondary: string;
  icon: "clock" | "token" | "tool";
}

/**
 * Conversion factor: milli-cents to USD
 *
 * Backend stores costs in "milli-cents" (1/1000 of a cent) as integers:
 * - 1 cent = 1000 milli-cents
 * - 1 USD = 100 cents = 100,000 milli-cents
 *
 * Example: $0.0087 = 0.87 cents = 870 milli-cents
 */
const MILLI_CENTS_TO_USD = 100000;

interface HeaderSessionStatsProps {
  /** Blocks to aggregate stats from */
  blocks?: AIBlock[];
  /** Current AI mode */
  mode?: AIMode;
  /** Compact display for mobile */
  compact?: boolean;
  /** className for styling */
  className?: string;
}

/**
 * Calculate aggregated session statistics from blocks
 * P2-#2: Limit processing to recent blocks for performance (last 100)
 * Trade-off: For very long conversations, this provides recent stats
 * rather than full historical totals, but prevents UI lag.
 */
const MAX_BLOCKS_TO_PROCESS = 100;

function calculateSessionStats(blocks: AIBlock[] | undefined, mode: AIMode = "normal"): SessionStatsData | null {
  if (!blocks || blocks.length === 0) return null;

  // Only process the most recent blocks for performance
  const blocksToProcess = blocks.slice(-MAX_BLOCKS_TO_PROCESS);

  let totalCost = 0;
  let totalTokens = 0;
  let totalDuration = 0;
  let toolCallCount = 0;
  let filesModified = 0;

  for (const block of blocksToProcess) {
    // Add cost estimate (from costEstimate field in milli-cents)
    if (block.costEstimate && block.costEstimate > 0) {
      totalCost += Number(block.costEstimate) / MILLI_CENTS_TO_USD;
    }

    // Add tokens from tokenUsage
    if (block.tokenUsage) {
      totalTokens += block.tokenUsage.totalTokens || 0;
    }

    // Track total duration
    const created = Number(block.createdTs);
    const updated = Number(block.updatedTs);
    totalDuration += updated - created;

    // Count tools and files (from sessionStats if available)
    if (block.sessionStats) {
      toolCallCount += block.sessionStats.toolCallCount || 0;
      filesModified += block.sessionStats.filesModified || 0;
    }
  }

  // Format functions
  const formatCost = (cost: number) => `$${cost.toFixed(4)}`;
  const formatTokens = (tokens: number) => `${(tokens / 1000).toFixed(1)}k`;
  const formatTime = (ms: number) => {
    if (ms < 1000) return "<1s";
    const s = Math.floor(ms / 1000);
    if (s < 60) return `${s}s`;
    const m = Math.floor(s / 60);
    return `${m}m`;
  };

  // Return mode-specific stats
  switch (mode) {
    case "geek":
      return {
        primary: formatTime(totalDuration),
        secondary: toolCallCount > 0 ? `${toolCallCount}` : "",
        icon: "clock",
      };

    case "evolution":
      return {
        primary: formatTime(totalDuration),
        secondary: filesModified > 0 ? `${filesModified} 文件` : "",
        icon: "clock",
      };

    case "normal":
    default:
      return {
        primary: formatTokens(totalTokens),
        secondary: formatCost(totalCost),
        icon: "token",
      };
  }
}

export const HeaderSessionStats = memo(function HeaderSessionStats({
  blocks,
  mode = "normal",
  compact = false,
  className,
}: HeaderSessionStatsProps) {
  const { t } = useTranslation();

  const stats = useMemo(() => calculateSessionStats(blocks, mode), [blocks, mode]);

  if (!stats) return null;

  // Desktop: Full stats row
  if (!compact) {
    return (
      <div
        className={cn(
          "hidden lg:flex items-center gap-3 text-[11px] font-mono opacity-70 bg-muted/30 px-2 py-1 rounded border border-border/50",
          className,
        )}
      >
        <span className="font-semibold text-muted-foreground/60 uppercase tracking-wider text-[10px]">{t("ai.unified_block.session")}</span>

        {mode === "normal" && (
          <>
            {stats.primary && (
              <span className="flex items-center gap-1" title={t("ai.unified_block.session_tokens")}>
                <Brain className="w-3 h-3" /> {stats.primary}
              </span>
            )}
            {stats.secondary && (
              <span className="flex items-center gap-1 text-green-600 dark:text-green-400" title={t("ai.unified_block.session_cost")}>
                <span className="font-bold">$</span> {stats.secondary}
              </span>
            )}
          </>
        )}

        {(mode === "geek" || mode === "evolution") && (
          <>
            {stats.primary && (
              <span className="flex items-center gap-1" title={t("ai.unified_block.session_duration")}>
                <Clock className="w-3 h-3" /> {stats.primary}
              </span>
            )}
            {stats.secondary && (
              <span className="flex items-center gap-1" title={mode === "geek" ? t("ai.stats.tool_calls") : t("ai.stats.files_modified")}>
                {mode === "geek" ? <Wrench className="w-3 h-3" /> : <Clock className="w-3 h-3" />}
                {stats.secondary}
              </span>
            )}
          </>
        )}
      </div>
    );
  }

  // Mobile: Simplified indicator
  return (
    <div className={cn("flex items-center gap-1 text-[10px] font-mono opacity-80", className)}>
      {mode === "normal" && stats.secondary && (
        <span className="flex items-center gap-0.5 text-green-600 dark:text-green-400 bg-green-50 dark:bg-green-900/20 px-1.5 py-0.5 rounded">
          <span className="font-bold">$</span>
          {stats.secondary}
        </span>
      )}
      {(mode === "geek" || mode === "evolution") && stats.primary && (
        <span className="flex items-center gap-0.5 text-muted-foreground bg-muted/50 px-1.5 py-0.5 rounded">
          <Clock className="w-3 h-3" /> {stats.primary}
        </span>
      )}
    </div>
  );
});

HeaderSessionStats.displayName = "HeaderSessionStats";

export default HeaderSessionStats;
