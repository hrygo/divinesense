import { DollarSign } from "lucide-react";
import { useTranslation } from "react-i18next";
import { cn } from "@/lib/utils";

interface BlockCostBadgeProps {
  costEstimate?: bigint;
  className?: string;
}

/**
 * BlockCostBadge - Compact badge displaying cost estimate
 *
 * Shows estimated cost in milli-cents (1/1000 of a US cent)
 * Converts to display currency (cents or dollars based on magnitude)
 */
export function BlockCostBadge({ costEstimate, className }: BlockCostBadgeProps) {
  const { t } = useTranslation();

  if (!costEstimate || costEstimate === 0n) {
    return null;
  }

  // Convert milli-cents to display format
  // costEstimate is in 1/1000 of a cent (1/100000 USD)
  const costInCents = Number(costEstimate) / 1000; // Convert to cents
  const costInUSD = costInCents / 100; // Convert to dollars

  // Format for display
  const formatCost = (usd: number) => {
    if (usd < 0.01) {
      // Show in milli-cents for very small amounts
      return `${costEstimate}m¢`;
    }
    if (usd < 1) {
      // Show in cents
      return `${costInCents.toFixed(2)}¢`;
    }
    // Show in dollars
    return `$${usd.toFixed(4)}`;
  };

  const displayCost = formatCost(costInUSD);

  // Determine color based on cost tier
  const getCostColor = () => {
    const usd = costInUSD;
    if (usd < 0.01) {
      return "bg-emerald-100 dark:bg-emerald-950/30 text-emerald-700 dark:text-emerald-400 border-emerald-200 dark:border-emerald-800/50";
    }
    if (usd < 0.1) {
      return "bg-green-100 dark:bg-green-950/30 text-green-700 dark:text-green-400 border-green-200 dark:border-green-800/50";
    }
    if (usd < 1) {
      return "bg-amber-100 dark:bg-amber-950/30 text-amber-700 dark:text-amber-400 border-amber-200 dark:border-amber-800/50";
    }
    return "bg-orange-100 dark:bg-orange-950/30 text-orange-700 dark:text-orange-400 border-orange-200 dark:border-orange-800/50";
  };

  return (
    <div
      className={cn(
        "flex items-center gap-1.5 px-2 py-1 rounded-md text-xs font-medium transition-colors border",
        getCostColor(),
        className,
      )}
      title={`${t("chat.block-summary.estimated-cost")}: $${costInUSD.toFixed(6)}`}
    >
      <DollarSign className="w-3.5 h-3.5" />
      <span>{displayCost}</span>
    </div>
  );
}
