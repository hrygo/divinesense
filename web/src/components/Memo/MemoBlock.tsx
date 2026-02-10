/**
 * MemoBlock - Unified Timeline Block Design
 *
 * 完全对齐到 Chat 的 UnifiedMessageBlock 设计语言
 *
 * ## 设计哲学
 * - 相同的 Block 主题系统
 * - 相同的 Timeline 视觉元素
 * - 相同的 Header/Footer 布局
 * - 相同的状态指示方式
 *
 * ## 与 Chat 的差异
 * - 无 Avatar（使用 Bookmark 图标代替）
 * - 无流式状态
 * - 操作按钮不同（Edit/Pin/Archive/Delete vs Copy/Regenerate/Delete）
 * - 无 Session Summary 统计
 */

import {
  Archive,
  ArchiveRestore,
  Bookmark,
  ChevronDown,
  ChevronUp,
  Clock,
  Copy,
  Edit,
  MoreHorizontal,
  Pin,
  PinOff,
  Trash2,
} from "lucide-react";
import { memo, useCallback, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { BlockActionButton } from "@/components/Memo/BlockActionButton";
import MemoView from "@/components/MemoView/MemoView";
import { cn } from "@/lib/utils";
import type { Memo } from "@/types/proto/api/v1/memo_service_pb";

/**
 * Format timestamp to relative time string
 */
function formatRelativeTime(timestamp: number, t: (key: string, options?: Record<string, unknown>) => string): string {
  const date = new Date(timestamp);
  if (isNaN(date.getTime())) {
    return t("common.unknown") || "Unknown";
  }

  const now = new Date();
  const diffMs = now.getTime() - date.getTime();
  const diffMins = Math.floor(diffMs / 60000);

  if (diffMins < 1) return t("common.just-now") || "Just now";
  if (diffMins < 60) return t("common.minutes-ago", { count: diffMins }) || `${diffMins}m ago`;
  if (diffMins < 1440) return t("common.hours-ago", { count: Math.floor(diffMins / 60) }) || `${Math.floor(diffMins / 60)}h ago`;

  return date.toLocaleDateString(undefined, { month: "short", day: "numeric" });
}

/**
 * Block Action - 用于定义操作按钮
 */
export interface BlockAction {
  icon: React.ComponentType<{ className?: string }>;
  label: string;
  onClick: () => void;
  variant?: "default" | "danger";
  showOnMobile: boolean;
}

/**
 * Get state-specific border style (inspired by Chat's border-l-4 status indicator)
 */
function getStateBorderClass(memo: Memo, isLatest: boolean): string {
  // Latest item gets special treatment
  if (isLatest) {
    return "border-l-4 border-l-amber-500";
  }
  // Archived state
  if (memo.state === 2) {
    return "border-l-4 border-l-slate-400 dark:border-l-slate-500";
  }
  // Pinned items get visual emphasis
  if (memo.pinned) {
    return "border-l-4 border-l-rose-500";
  }
  // Normal state - no left border
  return "";
}

/**
 * Memo Block Theme - 使用与 Chat 相同的主题系统
 * 对应 NORMAL 模式（Memo 默认使用 NORMAL 主题）
 */
const MEMO_BLOCK_THEME = {
  border: "border-amber-200 dark:border-amber-700",
  headerBg: "bg-amber-50 dark:bg-amber-900/20",
  footerBg: "bg-amber-200/80 dark:bg-amber-800/50",
  badgeBg: "bg-amber-100 dark:bg-amber-900/30",
  badgeText: "text-amber-600 dark:text-amber-400",
  ringColor: "ring-amber-500/20",
} as const;

/**
 * Storage key for collapse state persistence
 */
function getCollapseStorageKey(memoName: string): string {
  return `memo-block-collapsed-${memoName}`;
}

/**
 * Get default collapse state based on content length
 */
function getDefaultCollapseState(memo: Memo, isLatest?: boolean): boolean {
  if (isLatest) return false;
  if (memo.content.length < 200) return false;
  return true;
}

/**
 * Extract memo ID from resource name
 */
function getMemoId(memo: Memo): string {
  return memo.name.split("/").pop() || memo.name;
}

export interface MemoBlockProps {
  memo: Memo;
  isLatest?: boolean;
  onEdit?: (memo: Memo) => void;
  onDelete?: (memo: Memo) => void;
  onArchive?: (name: string, archived: boolean) => void;
  onPin?: (name: string, pinned: boolean) => void;
  onCopy?: (content: string) => void;
  className?: string;
}

/**
 * MemoBlock Component - 完全对齐 Chat 设计语言
 */
export const MemoBlock = memo(function MemoBlock({
  memo,
  isLatest = false,
  onEdit,
  onDelete,
  onArchive,
  onPin,
  onCopy,
  className,
}: MemoBlockProps) {
  const { t } = useTranslation();
  const memoId = getMemoId(memo);

  const [isExpanded, setIsExpanded] = useState(() => {
    if (typeof window === "undefined") return getDefaultCollapseState(memo, isLatest);
    try {
      const stored = localStorage.getItem(getCollapseStorageKey(memoId));
      if (stored !== null) return stored === "false";
    } catch (err) {
      // Log storage errors (e.g., Safari private browsing mode)
      console.debug("[MemoBlock] localStorage access failed:", err);
    }
    return getDefaultCollapseState(memo, isLatest);
  });

  useEffect(() => {
    if (typeof window === "undefined") return;
    try {
      localStorage.setItem(getCollapseStorageKey(memoId), String(!isExpanded));
    } catch (err) {
      // Log storage errors (e.g., Safari private browsing mode)
      console.debug("[MemoBlock] localStorage access failed:", err);
    }
  }, [memoId, isExpanded]);

  const handleToggle = useCallback(() => {
    setIsExpanded((prev) => !prev);
  }, []);

  // Build memo actions
  const actions = useMemo((): BlockAction[] => {
    const actionList: BlockAction[] = [];

    if (onEdit) {
      actionList.push({
        icon: Edit,
        label: t("common.edit"),
        onClick: () => onEdit(memo),
        showOnMobile: true,
      });
    }

    if (onPin) {
      actionList.push({
        icon: memo.pinned ? PinOff : Pin,
        label: memo.pinned ? t("common.unpin") : t("common.pin"),
        onClick: () => onPin(memoId, !memo.pinned),
        showOnMobile: false,
      });
    }

    if (onArchive) {
      actionList.push({
        icon: memo.state === 2 ? ArchiveRestore : Archive,
        label: memo.state === 2 ? t("common.restore") : t("common.archive"),
        onClick: () => onArchive(memoId, memo.state !== 2),
        showOnMobile: true,
      });
    }

    if (onCopy) {
      actionList.push({
        icon: Copy,
        label: t("common.copy"),
        onClick: () => onCopy(memo.content),
        showOnMobile: true,
      });
    }

    if (onDelete) {
      actionList.push({
        icon: Trash2,
        label: t("common.delete"),
        onClick: () => onDelete(memo),
        variant: "danger",
        showOnMobile: false,
      });
    }

    return actionList;
  }, [memo, memoId, onEdit, onPin, onArchive, onCopy, onDelete, t]);

  // Memo preview text
  const previewText = useMemo(() => {
    const firstLine = memo.content.split("\n")[0];
    return firstLine.length > 100 ? firstLine.slice(0, 100) + "..." : firstLine;
  }, [memo.content]);

  // Relative time
  const relativeTime = useMemo(() => {
    const timestamp = memo.displayTime ? Number(memo.displayTime) : Date.now();
    return formatRelativeTime(timestamp, t);
  }, [memo.displayTime, t]);

  // Memo visibility label
  const visibilityLabel = useMemo(() => {
    switch (memo.visibility) {
      case 1:
        return t("visibility.public");
      case 2:
        return t("visibility.protected");
      default:
        return t("visibility.private");
    }
  }, [memo.visibility, t]);

  // Compute state border class
  const stateBorderClass = getStateBorderClass(memo, isLatest);

  return (
    <div
      className={cn(
        // 与 Chat 相同的基础样式
        "rounded-lg border overflow-hidden shadow-sm transition-all duration-300",
        MEMO_BLOCK_THEME.border,
        // State border indicator (inspired by Chat's design)
        stateBorderClass,
        className,
      )}
    >
      {/* Block Header - 使用与 Chat 相同的布局结构 */}
      <MemoBlockHeader
        memo={memo}
        previewText={previewText}
        relativeTime={relativeTime}
        visibilityLabel={visibilityLabel}
        isExpanded={isExpanded}
        onToggle={handleToggle}
        theme={MEMO_BLOCK_THEME}
      />

      {/* Block Body - 可折叠内容 */}
      {isExpanded && (
        <div className="px-4 py-4 animate-in fade-in slide-in-from-top-1 duration-200">
          <MemoBlockContent memo={memo} />
        </div>
      )}

      {/* Block Footer - 使用与 Chat 相同的布局结构 */}
      <MemoBlockFooter isExpanded={isExpanded} onToggle={handleToggle} actions={actions} theme={MEMO_BLOCK_THEME} />
    </div>
  );
});

MemoBlock.displayName = "MemoBlock";

/**
 * MemoBlockHeader - 对齐 Chat 的 BlockHeader 设计
 */
interface MemoBlockHeaderProps {
  memo: Memo;
  previewText: string;
  relativeTime: string;
  visibilityLabel: string;
  isExpanded?: boolean;
  onToggle: () => void;
  theme: typeof MEMO_BLOCK_THEME;
}

function MemoBlockHeader({ memo, previewText, relativeTime, visibilityLabel, isExpanded, onToggle, theme }: MemoBlockHeaderProps) {
  const { t } = useTranslation();
  return (
    <div
      className={cn(
        // 与 Chat 相同的 header 布局
        "flex items-center justify-between px-4 py-2.5 select-none cursor-pointer transition-colors duration-200",
        theme.headerBg,
      )}
      onClick={onToggle}
    >
      {/* Left: Icon + Preview */}
      <div className="flex items-center gap-3 flex-1 min-w-0">
        {/* Bookmark Icon - 替代 Avatar */}
        <div className="w-7 h-7 rounded-full bg-amber-100 dark:bg-amber-900/30 flex items-center justify-center shrink-0 shadow-sm">
          <Bookmark className="w-4 h-4 text-amber-600 dark:text-amber-400" />
        </div>

        {/* Preview Text */}
        <div className="min-w-0 flex-1">
          <p className="text-sm font-medium text-foreground truncate" title={memo.content}>
            {previewText}
          </p>
          <div className="flex items-center gap-2 mt-0.5">
            <span className="text-[11px] text-muted-foreground">{relativeTime}</span>
            <span className="text-[11px] px-1.5 py-0.5 rounded bg-muted/50 text-muted-foreground">{visibilityLabel}</span>
            {memo.pinned && (
              <span className="text-[11px] flex items-center gap-0.5 text-amber-600 dark:text-amber-400">
                <Pin className="w-2.5 h-2.5" /> {t("common.pinned")}
              </span>
            )}
          </div>
        </div>
      </div>

      {/* Right: Toggle + Time */}
      <div className="flex items-center gap-2 sm:gap-3 shrink-0 ml-1 sm:ml-2">
        {/* Timestamp - 与 Chat 相同的样式 */}
        <div className={cn("flex items-center gap-1 text-xs", theme.badgeText)}>
          <Clock className="w-3 h-3" />
          <span className="hidden sm:inline">{relativeTime}</span>
        </div>

        {/* Toggle button - 与 Chat 相同的样式 */}
        <button
          type="button"
          className={cn(
            "p-1 rounded transition-colors",
            "hover:bg-black/10 dark:hover:bg-white/10",
            "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring",
            theme.badgeText,
          )}
          onClick={(e) => {
            e.stopPropagation();
            onToggle();
          }}
          aria-label={isExpanded ? t("common.collapse") : t("common.expand")}
          aria-expanded={!isExpanded}
        >
          {isExpanded ? <ChevronUp className="w-4 h-4" /> : <ChevronDown className="w-4 h-4" />}
        </button>
      </div>
    </div>
  );
}

/**
 * MemoBlockContent - 干净的内容展示
 */
interface MemoBlockContentProps {
  memo: Memo;
}

function MemoBlockContent({ memo }: MemoBlockContentProps) {
  return (
    <MemoView
      memo={memo}
      showVisibility={false}
      showPinned={false}
      compact={false}
      className="!border-0 !bg-transparent !shadow-none !p-0"
    />
  );
}

/**
 * MemoBlockFooter - 对齐 Chat 的 BlockFooter 设计
 */
interface MemoBlockFooterProps {
  isExpanded: boolean;
  onToggle: () => void;
  actions: BlockAction[];
  theme: typeof MEMO_BLOCK_THEME;
}

function MemoBlockFooter({ isExpanded, onToggle, actions, theme }: MemoBlockFooterProps) {
  const { t } = useTranslation();

  // Split actions by visibility
  const primaryActions = actions.filter((a) => a.showOnMobile);
  const secondaryActions = actions.filter((a) => !a.showOnMobile);

  return (
    <div className={cn("flex items-center justify-between px-4 py-2 border-t", theme.border, theme.footerBg)}>
      {/* Left: Collapse/Expand Toggle - 与 Chat 相同的按钮样式 */}
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
        <ChevronDown className={cn("w-3.5 h-3.5 transition-transform duration-200", !isExpanded && "rotate-180")} />
        <span className="hidden sm:inline">{isExpanded ? t("common.collapse") : t("common.expand")}</span>
      </button>

      {/* Right: Action Buttons - 响应式图标优先设计 */}
      <div className="flex items-center gap-2">
        {/* Mobile: Primary actions + more menu */}
        <div className="flex items-center gap-1 sm:hidden">
          {primaryActions.map((action, index) => (
            <BlockActionButton key={`mobile-${index}`} action={action} />
          ))}
          {(secondaryActions.length > 0 || primaryActions.length > 3) && (
            <button
              type="button"
              className="p-1.5 rounded-lg text-amber-600 dark:text-amber-400 hover:bg-amber-100 dark:hover:bg-amber-900/30 transition-colors"
              aria-label={t("common.more")}
            >
              <MoreHorizontal className="w-4 h-4" />
            </button>
          )}
        </div>

        {/* Desktop: All actions */}
        <div className="hidden sm:flex items-center gap-1">
          {actions.map((action, index) => (
            <BlockActionButton key={`desktop-${index}`} action={action} />
          ))}
        </div>
      </div>
    </div>
  );
}
