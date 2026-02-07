/**
 * BlockFooter Component - Optimized Version
 *
 * Icon-first responsive design:
 * - Desktop: Icon + Label
 * - Mobile: Icon only (with tooltip)
 *
 * Phase 2: Responsive Experience
 * Phase 5: React.memo optimization
 */

import { Brain, Check, ChevronDown, ChevronUp, Copy, StopCircle } from "lucide-react";
import { memo, useCallback, useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { cn } from "@/lib/utils";

export interface BlockFooterTheme {
  border: string;
  headerBg: string;
  footerBg: string;
  badgeBg: string;
  badgeText: string;
  ringColor: string;
}

export interface BlockFooterProps {
  isCollapsed: boolean;
  onToggle: () => void;
  onCopy: () => void;
  onRegenerate?: () => void;
  onDelete?: () => void;
  theme: BlockFooterTheme;
  isStreaming?: boolean;
  /** Cancel streaming callback (#113) */
  onCancel?: () => void;
}

/**
 * Memo comparison for BlockFooter
 */
const areBlockFooterPropsEqual = (prev: BlockFooterProps, next: BlockFooterProps): boolean => {
  return (
    prev.isCollapsed === next.isCollapsed &&
    prev.isStreaming === next.isStreaming &&
    prev.onCopy === next.onCopy &&
    prev.onToggle === next.onToggle &&
    prev.onCancel === next.onCancel
  );
};

/**
 * BlockFooter component
 *
 * Optimized with React.memo and responsive icon-first design.
 */
export const BlockFooter = memo(function BlockFooter({
  isCollapsed,
  onToggle,
  onCopy,
  onRegenerate,
  onDelete,
  theme,
  isStreaming,
  onCancel,
}: BlockFooterProps) {
  const { t } = useTranslation();
  const [copied, setCopied] = useState(false);
  const timeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const handleCopy = useCallback(() => {
    onCopy();
    setCopied(true);

    if (timeoutRef.current) {
      clearTimeout(timeoutRef.current);
    }

    timeoutRef.current = setTimeout(() => {
      setCopied(false);
      timeoutRef.current = null;
    }, 2000);
  }, [onCopy]);

  useEffect(() => {
    return () => {
      if (timeoutRef.current) {
        clearTimeout(timeoutRef.current);
      }
    };
  }, []);

  // Toggle icon
  const ToggleIcon = isCollapsed ? ChevronDown : ChevronUp;
  const toggleLabelKey = isCollapsed ? "common.expand" : "common.collapse";

  return (
    <div className={cn("flex items-center justify-between px-4 py-2 border-t", theme.border, theme.footerBg)}>
      {/* Left: Collapse/Expand Toggle */}
      <button
        type="button"
        onClick={onToggle}
        className={cn(
          "flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-medium transition-colors",
          "hover:bg-black/10 dark:hover:bg-white/10",
          "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring",
          theme.badgeText,
        )}
      >
        <ToggleIcon className="w-3.5 h-3.5" />
        <span className="hidden sm:inline">{t(toggleLabelKey)}</span>
      </button>

      {/* Right: Action Buttons - Responsive Icon-First */}
      <div className="flex items-center gap-2">
        {/* Cancel Button (Only show during streaming) */}
        {isStreaming && onCancel && (
          <button
            type="button"
            onClick={onCancel}
            className={cn(
              "flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-medium transition-colors",
              "bg-red-50 hover:bg-red-100 dark:bg-red-900/20 dark:hover:bg-red-900/30",
              "text-red-600 dark:text-red-400",
              "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-red-500",
            )}
            title={t("ai.actions.cancel") || "取消"}
          >
            <StopCircle className="w-3.5 h-3.5" />
            <span className="hidden sm:inline">{t("ai.actions.cancel") || "取消"}</span>
          </button>
        )}

        {/* Regenerate Button */}
        {onRegenerate && (
          <button
            type="button"
            onClick={onRegenerate}
            className={cn(
              "px-3 py-1.5 rounded-lg text-xs font-medium transition-colors",
              "hover:bg-black/10 dark:hover:bg-white/10",
              "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring",
              theme.badgeText,
            )}
            title={t("ai.regenerate")}
          >
            <span className="hidden sm:inline">{t("ai.regenerate")}</span>
            <span className="sm:hidden">↻</span>
          </button>
        )}

        {/* Forget/Remove Button */}
        <button
          type="button"
          className={cn(
            "flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-medium transition-colors opacity-60 hover:opacity-100",
            "hover:bg-black/10 dark:hover:bg-white/10",
            "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:opacity-100",
            theme.badgeText,
          )}
          title={t("ai.unified_block.forget_tooltip")}
        >
          <Brain className="w-3.5 h-3.5" />
          <span className="hidden lg:inline">{t("ai.unified_block.forget")}</span>
        </button>

        {/* Copy Button */}
        <button
          type="button"
          onClick={handleCopy}
          className={cn(
            "flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-medium transition-colors",
            "hover:bg-black/10 dark:hover:bg-white/10",
            "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring",
            copied && "bg-green-100 dark:bg-green-900/30 text-green-600 dark:text-green-400",
            !copied && theme.badgeText,
          )}
          title={copied ? t("common.copied") : t("common.copy")}
        >
          {copied ? <Check className="w-3.5 h-3.5" /> : <Copy className="w-3.5 h-3.5" />}
          <span className="hidden sm:inline">{copied ? t("common.copied") : t("common.copy")}</span>
        </button>

        {/* Delete Button */}
        {onDelete && (
          <button
            type="button"
            onClick={onDelete}
            className={cn(
              "flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-medium transition-colors",
              "hover:bg-red-100 dark:hover:bg-red-900/30",
              "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-red-500",
              "text-red-600 dark:text-red-400",
            )}
            title={t("common.delete")}
          >
            <span className="hidden sm:inline">{t("common.delete")}</span>
            <span className="sm:hidden">🗑</span>
          </button>
        )}
      </div>
    </div>
  );
}, areBlockFooterPropsEqual);
