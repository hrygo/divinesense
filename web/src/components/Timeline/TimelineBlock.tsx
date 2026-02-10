/**
 * TimelineBlock - Generic Timeline Block Component
 *
 * A reusable block component with:
 * - Collapsible header/footer
 * - Timeline visualization
 * - Theme support
 *
 * Used as base for:
 * - UnifiedMessageBlock (AIChat)
 * - MemoBlock (Memo)
 */

import { ChevronDown, ChevronUp } from "lucide-react";
import { memo, ReactNode } from "react";
import { cn } from "@/lib/utils";
import { TimelineNode } from "./TimelineNode";
import type { BlockAction, BlockTheme, TimelineBlockProps } from "./types";

export interface TimelineBlockRenderProps<T> {
  item: T;
  isExpanded: boolean;
  theme?: BlockTheme;
}

export interface TimelineBlockComponents<T> {
  renderHeader: (props: TimelineBlockRenderProps<T>) => ReactNode;
  renderTimeline?: (props: TimelineBlockRenderProps<T>) => ReactNode;
  renderContent: (props: TimelineBlockRenderProps<T>) => ReactNode;
  renderFooter: (props: { item: T; isExpanded: boolean; onToggle: () => void }) => ReactNode;
  actions?: BlockAction[];
}

/**
 * Generic Timeline Block Component
 */
export const TimelineBlock = memo(function TimelineBlock<T>({
  item,
  isExpanded,
  onToggle,
  theme,
  className,
  components,
}: TimelineBlockProps<T> & { components: TimelineBlockComponents<T> }) {
  const defaultTheme: BlockTheme = theme || {
    border: "border-zinc-200 dark:border-zinc-700",
    headerBg: "bg-zinc-50 dark:bg-zinc-900/50",
    footerBg: "bg-zinc-200/80 dark:bg-zinc-800/60",
    badgeBg: "bg-zinc-100 dark:bg-zinc-800",
    badgeText: "text-zinc-600 dark:text-zinc-400",
    ringColor: "ring-primary/20",
  };

  return (
    <div
      className={cn(
        "rounded-lg border shadow-sm transition-all duration-300",
        // Remove overflow-hidden to allow TimelineNode negative positioning to show
        // "overflow-hidden",
        defaultTheme.border,
        className,
      )}
    >
      {/* Header - Always visible */}
      <div
        className={cn("border-b px-4 py-2.5 select-none cursor-pointer transition-colors duration-200 rounded-t-lg", defaultTheme.headerBg)}
      >
        {components.renderHeader({ item, isExpanded, theme: defaultTheme })}
      </div>

      {/* Body - Collapsible */}
      {isExpanded && (
        <div className="px-4 py-4 animate-in fade-in slide-in-from-top-1 duration-200">
          {/* Timeline visualization */}
          {components.renderTimeline && (
            <div className="relative">
              {/* Timeline line */}
              <div className="absolute left-[11px] top-2 bottom-4 w-px bg-border/60" />
              <div className="relative pl-8 space-y-6">{components.renderTimeline({ item, isExpanded, theme: defaultTheme })}</div>
            </div>
          )}
          {/* Content */}
          {components.renderContent({ item, isExpanded, theme: defaultTheme })}
        </div>
      )}

      {/* Footer - Always visible */}
      <div className={cn("border-t px-4 py-2 rounded-b-lg", defaultTheme.border, defaultTheme.footerBg)}>
        {components.renderFooter({ item, isExpanded, onToggle })}
      </div>
    </div>
  );
});

// Set displayName using type assertion for better debugging experience
(TimelineBlock as ReturnType<typeof memo>).displayName = "TimelineBlock";

/**
 * Default toggle button component
 */
export interface BlockToggleButtonProps {
  isExpanded: boolean;
  onToggle: () => void;
  theme?: BlockTheme;
  "aria-label"?: string;
}

export const BlockToggleButton = memo(function BlockToggleButton({
  isExpanded,
  onToggle,
  theme,
  "aria-label": ariaLabel,
}: BlockToggleButtonProps) {
  const defaultTheme = theme || {
    badgeText: "text-zinc-600 dark:text-zinc-400",
  };

  return (
    <button
      type="button"
      className={cn(
        "p-1 rounded transition-colors",
        "hover:bg-black/10 dark:hover:bg-white/10",
        "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring",
        defaultTheme.badgeText,
      )}
      onClick={(e) => {
        e.stopPropagation();
        onToggle();
      }}
      aria-label={ariaLabel}
      aria-expanded={!isExpanded}
    >
      {isExpanded ? <ChevronUp className="w-4 h-4" /> : <ChevronDown className="w-4 h-4" />}
    </button>
  );
});

BlockToggleButton.displayName = "BlockToggleButton";

// Re-export TimelineNode for convenience
export { TimelineNode };
export type { TimelineNodeType } from "./types";
