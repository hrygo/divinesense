import { Brain, ChevronDown, ChevronUp, Clock, DollarSign, FileEdit, PenLine, Wrench, Zap } from "lucide-react";
import { memo, useState } from "react";
import { useTranslation } from "react-i18next";
import { cn } from "@/lib/utils";

interface SessionSummaryData {
  sessionId?: string;
  mode?: string; // "geek" | "evolution" | "normal"
  totalDurationMs?: number;
  thinkingDurationMs?: number;
  toolDurationMs?: number;
  generationDurationMs?: number;
  totalInputTokens?: number;
  totalOutputTokens?: number;
  totalCacheWriteTokens?: number;
  totalCacheReadTokens?: number;
  toolCallCount?: number;
  toolsUsed?: string[];
  filesModified?: number;
  filePaths?: string[];
  totalCostUSD?: number;
  status?: string;
}

interface ExpandedSessionSummaryProps {
  summary: SessionSummaryData;
  className?: string;
}

/**
 * ExpandedSessionSummary - Detailed session summary card for Geek/Evolution modes
 *
 * Shows all metrics in an expandable card with:
 * - Mode indicator (geek/evolution)
 * - Status and session ID
 * - Duration breakdown (thinking, tool, generation)
 * - Token breakdown (input, output, cache)
 * - Cost analysis
 * - Tool calls summary
 * - Files modified
 */
export const ExpandedSessionSummary = memo(function ExpandedSessionSummary({ summary, className }: ExpandedSessionSummaryProps) {
  const { t } = useTranslation();
  // Only enable expand on desktop (lg breakpoint and above)
  const [isExpanded, setIsExpanded] = useState(false);

  // Format duration in human-readable format
  const formatDuration = (ms: number) => {
    if (ms < 1000) return `${ms}ms`;
    if (ms < 60000) return `${(ms / 1000).toFixed(2)}s`;
    return `${(ms / 60000).toFixed(1)}m`;
  };

  // Format large numbers with unit suffix
  const formatNumber = (num: number) => {
    if (num >= 1000000) return `${(num / 1000000).toFixed(2)}M`;
    if (num >= 1000) return `${(num / 1000).toFixed(1)}K`;
    return num.toString();
  };

  // Calculate totals and percentages
  const totalTokens = (summary.totalInputTokens || 0) + (summary.totalOutputTokens || 0); // Billed tokens only
  const totalProcessedTokens =
    (summary.totalCacheReadTokens || 0) +
    (summary.totalCacheWriteTokens || 0) +
    (summary.totalInputTokens || 0) +
    (summary.totalOutputTokens || 0); // All tokens including cache
  const totalDuration = summary.totalDurationMs || 0;

  // Duration breakdown percentages
  const thinkingPercent =
    totalDuration > 0 && summary.thinkingDurationMs ? Math.round((summary.thinkingDurationMs / totalDuration) * 100) : 0;
  const toolPercent = totalDuration > 0 && summary.toolDurationMs ? Math.round((summary.toolDurationMs / totalDuration) * 100) : 0;
  const generationPercent =
    totalDuration > 0 && summary.generationDurationMs ? Math.round((summary.generationDurationMs / totalDuration) * 100) : 0;

  // Status color
  const statusColor =
    summary.status?.toLowerCase() === "error"
      ? "text-red-600 dark:text-red-400"
      : summary.status?.toLowerCase() === "cancelled"
        ? "text-amber-600 dark:text-amber-400"
        : "text-emerald-600 dark:text-emerald-400";

  return (
    <div
      className={cn(
        "rounded-xl border overflow-hidden",
        "bg-gradient-to-br from-slate-50 to-slate-100/50 dark:from-slate-900/50 dark:to-slate-950/50",
        "border-slate-200 dark:border-slate-700",
        className,
      )}
    >
      {/* Header - always visible, clickable only on desktop */}
      <div
        className={cn(
          "flex items-center justify-between px-4 py-3",
          "lg:cursor-pointer lg:hover:bg-slate-100/50 dark:lg:hover:bg-slate-800/50",
          "transition-colors",
        )}
        onClick={() => setIsExpanded(!isExpanded)}
      >
        <div className="flex items-center gap-3">
          {/* Status indicator */}
          <div className={cn("flex items-center gap-1.5 px-2 py-1 rounded-full text-xs font-semibold", statusColor, "bg-current/10")}>
            {summary.status?.toLowerCase() === "error" ? (
              <span>✗</span>
            ) : summary.status?.toLowerCase() === "cancelled" ? (
              <span>⚠</span>
            ) : (
              <span>✓</span>
            )}
            <span className="capitalize">{summary.status || t("ai.session_stats.status-success")}</span>
          </div>

          {/* Quick stats - single line */}
          <div className="flex items-center gap-3 text-xs text-muted-foreground">
            {totalDuration > 0 && (
              <span className="flex items-center gap-1">
                <Clock className="w-3.5 h-3.5" />
                <span className="font-mono font-medium">{formatDuration(totalDuration)}</span>
              </span>
            )}
            {totalTokens > 0 && (
              <span className="flex items-center gap-1">
                <Zap className="w-3.5 h-3.5 text-amber-500" />
                <span className="font-mono font-medium">{formatNumber(totalTokens)}</span>
              </span>
            )}
            {summary.totalCostUSD && summary.totalCostUSD > 0 && (
              <span className="flex items-center gap-1">
                <DollarSign className="w-3.5 h-3.5 text-green-500" />
                <span className="font-mono font-medium">
                  {t("ai.session_stats.currency_symbol")}
                  {summary.totalCostUSD.toFixed(4)}
                </span>
              </span>
            )}
          </div>
        </div>

        {/* Expand/collapse button - desktop only */}
        <button
          type="button"
          className="hidden lg:block p-1.5 rounded-lg hover:bg-slate-200 dark:hover:bg-slate-700 transition-colors"
          onClick={(e) => {
            e.stopPropagation();
            setIsExpanded(!isExpanded);
          }}
          aria-label={isExpanded ? t("ai.session_stats.collapse") : t("ai.session_stats.expand")}
        >
          {isExpanded ? <ChevronUp className="w-4 h-4 text-muted-foreground" /> : <ChevronDown className="w-4 h-4 text-muted-foreground" />}
        </button>
      </div>

      {/* Expanded details - desktop only */}
      {isExpanded && (
        <div className="hidden lg:block">
          <div className="px-4 pb-4 space-y-4">
            {/* Session ID */}
            {summary.sessionId && (
              <div className="flex items-center justify-between text-xs">
                <span className="text-muted-foreground">{t("ai.session_stats.session_id")}</span>
                <span className="font-mono text-muted-foreground/70 select-all break-all">{summary.sessionId}</span>
              </div>
            )}

            {/* Duration Breakdown - Bar chart */}
            {totalDuration > 0 && (
              <div className="space-y-2">
                <div className="flex items-center justify-between text-xs">
                  <span className="text-muted-foreground font-medium">{t("ai.session_stats.duration_breakdown")}</span>
                  <span className="font-mono text-muted-foreground">
                    {formatDuration(totalDuration)} {t("ai.session_stats.total")}
                  </span>
                </div>

                {/* Progress bar visualization */}
                <div className="h-6 rounded-full overflow-hidden flex bg-slate-200 dark:bg-slate-700">
                  {summary.thinkingDurationMs && summary.thinkingDurationMs > 0 && (
                    <div
                      className="bg-blue-500/80 flex items-center justify-center text-[10px] text-white font-medium"
                      style={{ width: `${thinkingPercent}%` }}
                      title={`${t("ai.session_stats.thinking")}: ${formatDuration(summary.thinkingDurationMs)}`}
                    >
                      {thinkingPercent >= 10 && <Brain className="w-3 h-3" />}
                    </div>
                  )}
                  {summary.toolDurationMs && summary.toolDurationMs > 0 && (
                    <div
                      className="bg-purple-500/80 flex items-center justify-center text-[10px] text-white font-medium"
                      style={{ width: `${toolPercent}%` }}
                      title={`${t("ai.session_stats.tool_execution")}: ${formatDuration(summary.toolDurationMs)}`}
                    >
                      {toolPercent >= 10 && <Wrench className="w-3 h-3" />}
                    </div>
                  )}
                  {summary.generationDurationMs && summary.generationDurationMs > 0 && (
                    <div
                      className="bg-emerald-500/80 flex items-center justify-center text-[10px] text-white font-medium"
                      style={{ width: `${generationPercent}%` }}
                      title={`${t("ai.session_stats.generation")}: ${formatDuration(summary.generationDurationMs)}`}
                    >
                      {generationPercent >= 10 && <PenLine className="w-3 h-3" />}
                    </div>
                  )}
                </div>

                {/* Legend */}
                <div className="flex flex-wrap gap-3 text-xs">
                  {summary.thinkingDurationMs && summary.thinkingDurationMs > 0 && (
                    <div className="flex items-center gap-1.5">
                      <div className="w-2 h-2 rounded-full bg-blue-500/80" />
                      <span className="text-muted-foreground">
                        {t("ai.session_stats.thinking")}: {formatDuration(summary.thinkingDurationMs)}
                      </span>
                    </div>
                  )}
                  {summary.toolDurationMs && summary.toolDurationMs > 0 && (
                    <div className="flex items-center gap-1.5">
                      <div className="w-2 h-2 rounded-full bg-purple-500/80" />
                      <span className="text-muted-foreground">
                        {t("ai.session_stats.tools")}: {formatDuration(summary.toolDurationMs)}
                      </span>
                    </div>
                  )}
                  {summary.generationDurationMs && summary.generationDurationMs > 0 && (
                    <div className="flex items-center gap-1.5">
                      <div className="w-2 h-2 rounded-full bg-emerald-500/80" />
                      <span className="text-muted-foreground">
                        {t("ai.session_stats.generation")}: {formatDuration(summary.generationDurationMs)}
                      </span>
                    </div>
                  )}
                </div>
              </div>
            )}

            {/* Grid layout for other metrics */}
            <div className="grid grid-cols-2 gap-3">
              {/* Tokens - always show if we have token data */}
              {(summary.totalInputTokens !== undefined || summary.totalOutputTokens !== undefined) && (
                <div className="p-3 rounded-lg bg-amber-50/50 dark:bg-amber-900/10 border border-amber-200/50 dark:border-amber-700/30">
                  <div className="flex items-center gap-2 mb-2">
                    <Zap className="w-4 h-4 text-amber-500" />
                    <span className="text-xs font-medium text-muted-foreground">{t("ai.session_stats.tokens")}</span>
                  </div>
                  <div className="space-y-1">
                    {/* Cache Read (discounted 90%) */}
                    {(summary.totalCacheReadTokens || 0) > 0 && (
                      <div className="flex justify-between text-xs items-center">
                        <span className="text-green-600 dark:text-green-400 flex items-center gap-1">
                          <span className="w-1.5 h-1.5 rounded-full bg-green-500" />
                          {t("ai.session_stats.cache_read_discount")}
                        </span>
                        <span className="font-mono text-green-600 dark:text-green-400">
                          {formatNumber(summary.totalCacheReadTokens || 0)}
                        </span>
                      </div>
                    )}
                    {/* Cache Write (1.25x base price) */}
                    {(summary.totalCacheWriteTokens || 0) > 0 && (
                      <div className="flex justify-between text-xs items-center">
                        <span className="text-amber-600 dark:text-amber-400 flex items-center gap-1">
                          <span className="w-1.5 h-1.5 rounded-full bg-amber-500" />
                          {t("ai.session_stats.cache_write_premium")}
                        </span>
                        <span className="font-mono text-amber-600 dark:text-amber-400">
                          {formatNumber(summary.totalCacheWriteTokens || 0)}
                        </span>
                      </div>
                    )}
                    {/* New Input (billed at base rate) */}
                    {(summary.totalInputTokens || 0) > 0 && (
                      <div className="flex justify-between text-xs items-center">
                        <span className="text-muted-foreground/70 flex items-center gap-1">
                          <span className="w-1.5 h-1.5 rounded-full bg-amber-500" />
                          {t("ai.session_stats.new_input")}
                        </span>
                        <span className="font-mono">{formatNumber(summary.totalInputTokens || 0)}</span>
                      </div>
                    )}
                    {/* Output (billed) */}
                    {(summary.totalOutputTokens || 0) > 0 && (
                      <div className="flex justify-between text-xs items-center">
                        <span className="text-muted-foreground/70 flex items-center gap-1">
                          <span className="w-1.5 h-1.5 rounded-full bg-amber-500" />
                          {t("ai.session_stats.output")}
                        </span>
                        <span className="font-mono">{formatNumber(summary.totalOutputTokens || 0)}</span>
                      </div>
                    )}
                    {/* Total Processed - always show */}
                    <div className="pt-1 border-t border-amber-200/30 dark:border-amber-700/30 flex justify-between text-xs font-medium">
                      <span className="text-muted-foreground">{t("ai.session_stats.total_processed")}</span>
                      <span className="font-mono text-amber-600 dark:text-amber-400">{formatNumber(totalProcessedTokens)}</span>
                    </div>
                  </div>
                </div>
              )}

              {/* Cost - actual cost from backend */}
              {summary.totalCostUSD !== undefined && summary.totalCostUSD >= 0 && (
                <div className="p-3 rounded-lg bg-green-50/50 dark:bg-green-900/10 border border-green-200/50 dark:border-green-700/30">
                  <div className="flex items-center justify-between mb-2">
                    <div className="flex items-center gap-2">
                      <DollarSign className="w-4 h-4 text-green-500" />
                      <span className="text-xs font-medium text-muted-foreground">{t("ai.session_stats.total_cost")}</span>
                    </div>
                    <span className="text-2xl font-mono font-bold text-green-600 dark:text-green-400">
                      {t("ai.session_stats.currency_symbol")}
                      {summary.totalCostUSD.toFixed(4)}
                    </span>
                  </div>

                  {/* Cost per 1K tokens */}
                  {totalTokens > 0 && (
                    <div className="text-[10px] text-muted-foreground text-center">
                      {t("ai.session_stats.currency_symbol")}
                      {((summary.totalCostUSD / totalTokens) * 1000).toFixed(4)} {t("ai.session_stats.per_1k_tokens")}
                    </div>
                  )}

                  {/* Cache hit rate indicator */}
                  {totalProcessedTokens > 0 && (summary.totalCacheReadTokens || 0) > 0 && (
                    <div className="mt-2 pt-2 border-t border-green-200/30 dark:border-green-700/30">
                      <div className="flex items-center justify-between text-[11px]">
                        <span className="text-muted-foreground">{t("ai.session_stats.cache_hit_rate")}</span>
                        <span className="font-mono text-green-600 dark:text-green-400">
                          {Math.round(((summary.totalCacheReadTokens || 0) / totalProcessedTokens) * 100)}
                          {t("ai.session_stats.percent")}
                        </span>
                      </div>
                    </div>
                  )}
                </div>
              )}

              {/* Tool Calls - always show if we have tool data */}
              {summary.toolCallCount !== undefined && summary.toolCallCount > 0 && (
                <div className="p-3 rounded-lg bg-purple-50/50 dark:bg-purple-900/10 border border-purple-200/50 dark:border-purple-700/30">
                  <div className="flex items-center gap-2 mb-2">
                    <Wrench className="w-4 h-4 text-purple-500" />
                    <span className="text-xs font-medium text-muted-foreground">{t("ai.session_stats.tool_calls")}</span>
                  </div>
                  <div className="text-center">
                    <span className="text-2xl font-mono font-bold text-purple-600 dark:text-purple-400">{summary.toolCallCount}</span>
                    {summary.toolDurationMs && (
                      <div className="text-[11px] text-muted-foreground mt-1">
                        {t("ai.session_stats.avg")}: {formatDuration(summary.toolDurationMs / summary.toolCallCount)}{" "}
                        {t("ai.session_stats.per_call")}
                      </div>
                    )}
                  </div>
                  {summary.toolsUsed && summary.toolsUsed.length > 0 && (
                    <div className="mt-2 pt-2 border-t border-purple-200/30 dark:border-purple-700/30">
                      <div className="flex flex-wrap gap-1">
                        {summary.toolsUsed.map((tool) => (
                          <span
                            key={tool}
                            className="px-2 py-0.5 rounded-full text-[11px] font-medium bg-purple-100 dark:bg-purple-900/30 text-purple-700 dark:text-purple-300"
                          >
                            {tool}
                          </span>
                        ))}
                      </div>
                    </div>
                  )}
                </div>
              )}

              {/* Files Modified - show when has files */}
              {summary.filesModified !== undefined && summary.filesModified > 0 && (
                <div className="p-3 rounded-lg bg-emerald-50/50 dark:bg-emerald-900/10 border border-emerald-200/50 dark:border-emerald-700/30">
                  <div className="flex items-center gap-2 mb-2">
                    <FileEdit className="w-4 h-4 text-emerald-500" />
                    <span className="text-xs font-medium text-muted-foreground">{t("ai.session_stats.files_modified")}</span>
                  </div>
                  <div className="text-center">
                    <span className="text-2xl font-mono font-bold text-emerald-600 dark:text-emerald-400">{summary.filesModified}</span>
                    {summary.filePaths && summary.filePaths.length > 0 && (
                      <div className="mt-2 text-[10px] text-muted-foreground">
                        <div className="max-h-16 overflow-y-auto space-y-0.5">
                          {summary.filePaths.map((path, i) => (
                            <div key={i} className="truncate font-mono" title={path}>
                              {path}
                            </div>
                          ))}
                        </div>
                      </div>
                    )}
                  </div>
                </div>
              )}
            </div>
          </div>
        </div>
      )}
    </div>
  );
});
