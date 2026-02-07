import { Hash } from "lucide-react";
import { useTranslation } from "react-i18next";
import { cn } from "@/lib/utils";

interface BlockNumberIndicatorProps {
  /** Block number (1-based, displayed as-is) */
  blockNumber: number;
  /** Optional label for tooltip */
  label?: string;
  className?: string;
  /** Whether this is the current/active block */
  isActive?: boolean;
}

/**
 * BlockNumberIndicator - Visual indicator showing block sequence number
 *
 * Displays the block number (e.g., "1", "2", "3") instead of branch path.
 * Simplified from the original branch indicator to show sequential block numbers.
 */
export function BlockNumberIndicator({ blockNumber, label, className, isActive }: BlockNumberIndicatorProps) {
  const { t } = useTranslation();

  // Don't render for invalid block numbers
  if (blockNumber <= 0) {
    return null;
  }

  const displayNumber = String(blockNumber);
  // Try multiple key paths for different locale file structures
  const defaultLabel =
    label ||
    t("chat.block-number", { number: blockNumber }) ||
    t("ai.unified_block.block_number", { number: blockNumber }) ||
    t("chat.blocks.block-number", { number: blockNumber, defaultValue: `Block ${blockNumber}` });

  return (
    <div
      className={cn(
        "inline-flex items-center gap-1.5 px-2 py-1 rounded-md text-xs font-medium transition-colors",
        "bg-slate-100 dark:bg-slate-800/50",
        "text-slate-700 dark:text-slate-400",
        "border border-slate-200 dark:border-slate-700/50",
        className,
      )}
      title={defaultLabel}
    >
      <Hash className="w-3.5 h-3.5" />
      <span className="font-mono">{displayNumber}</span>
      {isActive && <span className="w-1.5 h-1.5 rounded-full bg-slate-500 dark:bg-slate-400" />}
    </div>
  );
}

/**
 * CompactBlockNumberIndicator - Minimal inline block number indicator
 */
interface CompactBlockNumberIndicatorProps {
  blockNumber: number;
  className?: string;
  label?: string;
}

export function CompactBlockNumberIndicator({ blockNumber, className, label }: CompactBlockNumberIndicatorProps) {
  if (blockNumber <= 0) {
    return null;
  }

  const displayNumber = String(blockNumber);
  const defaultLabel = label || `#${blockNumber}`;

  return (
    <div
      className={cn(
        "flex items-center gap-1 px-1.5 py-0.5 rounded text-[11px] font-medium",
        "bg-slate-100 dark:bg-slate-800/50 text-slate-600 dark:text-slate-400",
        className,
      )}
      title={defaultLabel}
    >
      <span className="font-mono">{displayNumber}</span>
    </div>
  );
}

/**
 * SimpleBlockNumberIndicator - Minimal block number display without badge
 */
interface SimpleBlockNumberIndicatorProps {
  blockNumber: number;
  className?: string;
}

export function SimpleBlockNumberIndicator({ blockNumber, className }: SimpleBlockNumberIndicatorProps) {
  if (blockNumber <= 0) {
    return null;
  }

  return (
    <span
      className={cn(
        "inline-flex items-center justify-center w-5 h-5 rounded-full text-[10px] font-mono font-medium",
        "bg-slate-200 dark:bg-slate-700 text-slate-600 dark:text-slate-400",
        className,
      )}
      title={`Block ${blockNumber}`}
    >
      {blockNumber}
    </span>
  );
}

// Legacy exports for backward compatibility
// These now map to block number indicators with branchPath converted to block number
interface BranchIndicatorProps {
  /** @deprecated Use blockNumber instead */
  branchPath?: string;
  /** @deprecated Use blockNumber instead */
  branches?: never;
  /** @deprecated Use blockNumber instead */
  branchCount?: never;
  isActive?: boolean;
  className?: string;
  onClick?: () => void;
  /** Block number (preferred over branchPath) */
  blockNumber?: number;
}

/**
 * @deprecated Use BlockNumberIndicator instead
 * Kept for backward compatibility during transition
 */
export function BranchIndicator({ branchPath, blockNumber, isActive, className }: BranchIndicatorProps) {
  // If blockNumber is provided, use it directly
  if (blockNumber !== undefined) {
    return <BlockNumberIndicator blockNumber={blockNumber} isActive={isActive} className={className} />;
  }

  // Legacy: extract block number from branchPath if available
  // branchPath format was "0/1/2" or "A.1.2" - we'll use the last number as block number
  let displayBlockNumber = 0;
  if (branchPath) {
    const parts = branchPath.split("/").filter(Boolean);
    if (parts.length > 0) {
      const lastPart = parts[parts.length - 1];
      const num = parseInt(lastPart, 10);
      if (!isNaN(num)) {
        displayBlockNumber = num + 1; // Convert 0-based to 1-based
      }
    }
  }

  if (displayBlockNumber <= 0) {
    return null;
  }

  return <BlockNumberIndicator blockNumber={displayBlockNumber} isActive={isActive} className={className} />;
}

/**
 * @deprecated Use CompactBlockNumberIndicator instead
 */
interface CompactBranchIndicatorProps {
  /** @deprecated Use blockNumber instead */
  branchPath?: string;
  /** @deprecated Use blockNumber instead */
  branchCount?: never;
  /** Block number (preferred over branchPath) */
  blockNumber?: number;
  className?: string;
}

export function CompactBranchIndicator({ branchPath, blockNumber, className }: CompactBranchIndicatorProps) {
  if (blockNumber !== undefined) {
    return <CompactBlockNumberIndicator blockNumber={blockNumber} className={className} />;
  }

  let displayBlockNumber = 0;
  if (branchPath) {
    const parts = branchPath.split("/").filter(Boolean);
    if (parts.length > 0) {
      const lastPart = parts[parts.length - 1];
      const num = parseInt(lastPart, 10);
      if (!isNaN(num)) {
        displayBlockNumber = num + 1;
      }
    }
  }

  if (displayBlockNumber <= 0) {
    return null;
  }

  return <CompactBlockNumberIndicator blockNumber={displayBlockNumber} className={className} />;
}

/**
 * @deprecated Use SimpleBlockNumberIndicator instead
 */
interface SimplePathIndicatorProps {
  /** @deprecated Use blockNumber instead */
  branchPath?: string;
  /** Block number (preferred over branchPath) */
  blockNumber?: number;
  className?: string;
}

export function SimplePathIndicator({ branchPath, blockNumber, className }: SimplePathIndicatorProps) {
  if (blockNumber !== undefined) {
    return <SimpleBlockNumberIndicator blockNumber={blockNumber} className={className} />;
  }

  let displayBlockNumber = 0;
  if (branchPath) {
    const parts = branchPath.split("/").filter(Boolean);
    if (parts.length > 0) {
      const lastPart = parts[parts.length - 1];
      const num = parseInt(lastPart, 10);
      if (!isNaN(num)) {
        displayBlockNumber = num + 1;
      }
    }
  }

  if (displayBlockNumber <= 0) {
    return null;
  }

  return <SimpleBlockNumberIndicator blockNumber={displayBlockNumber} className={className} />;
}
