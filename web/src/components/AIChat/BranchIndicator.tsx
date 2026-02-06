import { GitBranch, TreeDeciduous } from "lucide-react";
import { useTranslation } from "react-i18next";
import { cn } from "@/lib/utils";
import { type BlockBranch } from "@/types/block";

/**
 * Convert internal branch path (e.g., "0/1/2") to friendly format (e.g., "A.2.3")
 * - First level: 0=A, 1=B, 2=C, ... (26=AA, 27=AB, ...)
 * - Other levels: 0-based to 1-based (0->1, 1->2, ...)
 */
function formatBranchPath(internalPath: string): string {
  if (!internalPath) return "";

  const parts = internalPath.split("/").filter(Boolean);

  if (parts.length === 0) return "";

  // Convert each part
  return parts
    .map((part, index) => {
      const num = parseInt(part, 10);
      if (isNaN(num)) return part;

      // First level: 0=A, 1=B, ..., 25=Z, 26=AA, 27=AB, ...
      if (index === 0) {
        return toBase26(num);
      }

      // Other levels: convert 0-based to 1-based
      return String(num + 1);
    })
    .join(".");
}

/**
 * Convert number to Excel-style column notation (0=A, 1=B, ..., 25=Z, 26=AA, ...)
 * P2-#1: Added explicit iteration limit to prevent potential infinite loops
 */
function toBase26(num: number): string {
  if (num < 0) return "?";
  if (num < 26) return String.fromCharCode(65 + num); // A-Z

  // 26^10 is astronomically large (> 10^14), far beyond practical branch depth
  const MAX_ITERATIONS = 10;
  let result = "";
  let n = num;
  let iterations = 0;

  while (n >= 0 && iterations < MAX_ITERATIONS) {
    result = String.fromCharCode(65 + (n % 26)) + result;
    n = Math.floor(n / 26) - 1;
    iterations++;
  }

  // Fallback for absurdly large numbers (should never happen in practice)
  if (n >= 0) {
    console.warn(`[BranchIndicator] Branch depth ${num} exceeds reasonable limit, showing "?"`);
    return "?";
  }

  return result;
}

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

  // Convert internal path to friendly format
  const friendlyPath = branchPath ? formatBranchPath(branchPath) : "";
  const displayPath = friendlyPath;
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
  // Convert internal path to friendly format
  const friendlyPath = branchPath ? formatBranchPath(branchPath) : "";
  const displayPath = friendlyPath;
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
  // Convert internal path to friendly format
  const friendlyPath = branchPath ? formatBranchPath(branchPath) : "";

  if (!friendlyPath || friendlyPath.length === 0) {
    return null;
  }

  return (
    <span
      className={cn(
        "inline-flex items-center gap-1 px-1.5 py-0.5 rounded text-[11px] font-mono",
        "bg-purple-50 dark:bg-purple-950/20 text-purple-600 dark:text-purple-400",
        className,
      )}
      title={`Branch: ${branchPath} (${friendlyPath})`}
    >
      <TreeDeciduous className="w-3 h-3 shrink-0" />
      <span className="truncate max-w-20">{friendlyPath}</span>
    </span>
  );
}
