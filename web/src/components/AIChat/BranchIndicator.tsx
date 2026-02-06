import { GitBranch } from "lucide-react";
import { useTranslation } from "react-i18next";
import { cn } from "@/lib/utils";
import { type BlockBranch } from "@/types/block";

interface BranchIndicatorProps {
  branches?: BlockBranch[];
  isActive?: boolean;
  className?: string;
  onClick?: () => void;
}

/**
 * BranchIndicator - Visual indicator showing branch count on blocks
 *
 * Displays a small badge showing the number of branches available from this block.
 * Clicking opens the branch selector to switch between branches.
 */
export function BranchIndicator({ branches, isActive, className, onClick }: BranchIndicatorProps) {
  const { t } = useTranslation();

  // Count total branches (including all descendants)
  const branchCount = branches?.length ?? 0;

  if (branchCount === 0) {
    return null;
  }

  return (
    <button
      type="button"
      onClick={onClick}
      className={cn(
        "flex items-center gap-1.5 px-2 py-1 rounded-md text-xs font-medium transition-colors",
        "bg-purple-100 dark:bg-purple-950/30",
        "text-purple-700 dark:text-purple-400",
        "hover:bg-purple-200 dark:hover:bg-purple-950/50",
        "border border-purple-200 dark:border-purple-800/50",
        onClick && "cursor-pointer",
        className,
      )}
      title={t("chat.branches.branches-available", { count: branchCount })}
    >
      <GitBranch className="w-3.5 h-3.5" />
      <span>{branchCount}</span>
      {isActive && <span className="w-1.5 h-1.5 rounded-full bg-purple-500" />}
    </button>
  );
}

/**
 * CompactBranchIndicator - Minimal inline branch indicator
 */
interface CompactBranchIndicatorProps {
  branchCount: number;
  className?: string;
}

export function CompactBranchIndicator({ branchCount, className }: CompactBranchIndicatorProps) {
  if (branchCount === 0) {
    return null;
  }

  return (
    <div
      className={cn(
        "flex items-center gap-1 px-1.5 py-0.5 rounded text-[11px] font-medium",
        "bg-purple-100 dark:bg-purple-950/30 text-purple-600 dark:text-purple-400",
        className,
      )}
    >
      <GitBranch className="w-3 h-3" />
      <span>{branchCount}</span>
    </div>
  );
}
