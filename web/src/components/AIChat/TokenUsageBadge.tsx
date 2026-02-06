import { ChevronDown, ChevronUp, Coins } from "lucide-react";
import { useTranslation } from "react-i18next";
import { cn } from "@/lib/utils";
import { type TokenUsage as TokenUsageType } from "@/types/block";

interface TokenUsageBadgeProps {
  tokenUsage?: TokenUsageType;
  className?: string;
}

/**
 * TokenUsageBadge - Compact badge displaying token usage statistics
 *
 * Shows total tokens with option to expand for detailed breakdown:
 * - Input/Output tokens
 * - Cache read/write tokens
 */
export function TokenUsageBadge({ tokenUsage, className }: TokenUsageBadgeProps) {
  const { t } = useTranslation();
  const [expanded, setExpanded] = React.useState(false);

  if (!tokenUsage) {
    return null;
  }

  const { promptTokens = 0, completionTokens = 0, totalTokens = 0, cacheReadTokens = 0, cacheWriteTokens = 0 } = tokenUsage;

  // Calculate display values
  const displayTotal = totalTokens || promptTokens + completionTokens;
  const hasCache = cacheReadTokens > 0 || cacheWriteTokens > 0;

  // Format large numbers
  const formatNumber = (num: number) => {
    if (num >= 1000000) return `${(num / 1000000).toFixed(1)}M`;
    if (num >= 1000) return `${(num / 1000).toFixed(1)}K`;
    return num.toString();
  };

  const ExpandIcon = expanded ? ChevronUp : ChevronDown;

  return (
    <div className={cn("relative group", className)}>
      {/* Collapsed state - simple badge */}
      <button
        type="button"
        onClick={() => setExpanded(!expanded)}
        className={cn(
          "flex items-center gap-1.5 px-2 py-1 rounded-md text-xs font-medium transition-colors",
          "bg-amber-100 dark:bg-amber-950/30",
          "text-amber-700 dark:text-amber-400",
          "hover:bg-amber-200 dark:hover:bg-amber-950/50",
          "border border-amber-200 dark:border-amber-800/50",
        )}
      >
        <Coins className="w-3.5 h-3.5" />
        <span>{formatNumber(displayTotal)}</span>
        <ExpandIcon className="w-3 h-3 opacity-60" />
      </button>

      {/* Expanded state - detailed breakdown */}
      {expanded && (
        <div
          className={cn(
            "absolute z-50 mt-1 p-2 rounded-lg shadow-lg border min-w-[140px]",
            "bg-background dark:bg-gray-900",
            "border-border",
          )}
        >
          <div className="space-y-1 text-xs">
            {/* Total */}
            <div className="flex justify-between items-center gap-4 font-medium text-foreground">
              <span>{t("chat.block-summary.total-tokens")}</span>
              <span className="text-amber-600 dark:text-amber-400">{formatNumber(displayTotal)}</span>
            </div>

            {/* Input/Output breakdown */}
            <div className="pt-1 space-y-1 border-t border-border/50">
              <div className="flex justify-between items-center gap-4 text-muted-foreground">
                <span>{t("chat.block-summary.input-tokens")}</span>
                <span>{formatNumber(promptTokens)}</span>
              </div>
              <div className="flex justify-between items-center gap-4 text-muted-foreground">
                <span>{t("chat.block-summary.output-tokens")}</span>
                <span>{formatNumber(completionTokens)}</span>
              </div>
            </div>

            {/* Cache tokens (if any) */}
            {hasCache && (
              <div className="pt-1 space-y-1 border-t border-border/50">
                <div className="flex justify-between items-center gap-4 text-muted-foreground">
                  <span>{t("chat.block-summary.cache-read")}</span>
                  <span className="text-emerald-600 dark:text-emerald-400">-{formatNumber(cacheReadTokens)}</span>
                </div>
                <div className="flex justify-between items-center gap-4 text-muted-foreground">
                  <span>{t("chat.block-summary.cache-write")}</span>
                  <span className="text-blue-600 dark:text-blue-400">+{formatNumber(cacheWriteTokens)}</span>
                </div>
              </div>
            )}
          </div>
        </div>
      )}
    </div>
  );
}

// Import React for useState
import React from "react";
