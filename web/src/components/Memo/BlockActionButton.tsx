/**
 * BlockActionButton - Icon-first responsive action button
 *
 * Used in MemoBlock footer for actions like edit, delete, archive.
 * Shows icon-only on mobile, icon+label on desktop.
 */

import { memo } from "react";
import type { BlockAction } from "@/components/Timeline";
import { cn } from "@/lib/utils";

export interface BlockActionButtonProps {
  action: BlockAction;
}

export const BlockActionButton = memo(function BlockActionButton({ action }: BlockActionButtonProps) {
  // Don't render actions that should only show during streaming
  if (action.isStreaming === false) return null;

  return (
    <button
      type="button"
      onClick={action.onClick}
      className={cn(
        "flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-medium transition-colors",
        "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring",
        // Variant styles
        action.variant === "danger"
          ? "text-red-600 dark:text-red-400 hover:bg-red-100 dark:hover:bg-red-900/30"
          : "text-amber-600 dark:text-amber-400 hover:bg-amber-100 dark:hover:bg-amber-900/30",
        // Hide on mobile if showOnMobile is false
        !action.showOnMobile && "hidden lg:flex",
      )}
      title={action.label}
    >
      <action.icon className="w-3.5 h-3.5 shrink-0" />
      <span className="hidden sm:inline">{action.label}</span>
    </button>
  );
});

BlockActionButton.displayName = "BlockActionButton";
