import { GitBranch, TreeDeciduous } from "lucide-react";
import { useTranslation } from "react-i18next";
import { cn } from "@/lib/utils";
import { type BlockBranch } from "@/types/block";

interface BranchIndicatorProps {
  branches?: BlockBranch[];
  /** Branch path (e.g., "A.1", "B.2.3") - displays this instead of branch count */
  branchPath?: string;
  /** Number of branches (legacy, used if branchPath not provided) */
  branchCount?: number;
  isActive?: boolean;
  className?: string;
  onClick?: () => void;
}

/**
 * BranchIndicator - Visual indicator showing branch path or count on blocks
 *
 * Displays the branch path (e.g., "A.1", "B.2.3") if provided,
 * otherwise shows a badge with the number of branches available.
 * Clicking opens the branch selector to switch between branches.
 */
export function BranchIndicator({ branches, branchPath, branchCount, isActive, className, onClick }: BranchIndicatorProps) {
  const { t } = useTranslation();

  // Use branchPath if provided, otherwise count branches
  const displayPath = branchPath;
  const count = branchCount ?? branches?.length ?? 0;

  // Don't render if no branch path and no branches
  if (!displayPath && count === 0) {
    return null;
  }

  // For display: if we have a branch path, show it; otherwise show count
  const hasBranches = displayPath ? displayPath.length > 0 : count > 0;

  if (!hasBranches) {
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
      title={displayPath || t("chat.branches.branches-available", { count })}
    >
      <GitBranch className="w-3.5 h-3.5" />
      <span className="font-mono">{displayPath || count}</span>
      {isActive && <span className="w-1.5 h-1.5 rounded-full bg-purple-500" />}
    </button>
  );
}

/**
 * CompactBranchIndicator - Minimal inline branch indicator with path
 */
interface CompactBranchIndicatorProps {
  branchPath?: string;
  branchCount?: number;
  className?: string;
}

export function CompactBranchIndicator({ branchPath, branchCount, className }: CompactBranchIndicatorProps) {
  // Use branchPath if provided, otherwise use branchCount
  const displayPath = branchPath;
  const count = branchCount ?? 0;

  if (!displayPath && count === 0) {
    return null;
  }

  return (
    <div
      className={cn(
        "flex items-center gap-1 px-1.5 py-0.5 rounded text-[11px] font-medium",
        "bg-purple-100 dark:bg-purple-950/30 text-purple-600 dark:text-purple-400",
        className,
      )}
      title={displayPath}
    >
      {displayPath ? (
        <>
          <TreeDeciduous className="w-3 h-3 shrink-0" />
          <span className="font-mono truncate max-w-16">{displayPath}</span>
        </>
      ) : (
        <>
          <GitBranch className="w-3 h-3 shrink-0" />
          <span>{count}</span>
        </>
      )}
    </div>
  );
}

/**
 * SimplePathIndicator - Minimal branch path display without badge
 *
 * Shows just the branch path as text, useful for inline display.
 */
interface SimplePathIndicatorProps {
  branchPath?: string;
  className?: string;
}

export function SimplePathIndicator({ branchPath, className }: SimplePathIndicatorProps) {
  if (!branchPath || branchPath.length === 0) {
    return null;
  }

  return (
    <span
      className={cn(
        "inline-flex items-center gap-1 px-1.5 py-0.5 rounded text-[11px] font-mono",
        "bg-purple-50 dark:bg-purple-950/20 text-purple-600 dark:text-purple-400",
        className,
      )}
      title={`Branch: ${branchPath}`}
    >
      <TreeDeciduous className="w-3 h-3 shrink-0" />
      <span className="truncate max-w-20">{branchPath}</span>
    </span>
  );
}
