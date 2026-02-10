/**
 * MemoBlockV2 - "Fluid Card" Design for AI-Native Note Taking
 *
 * ## Design Philosophy
 * - Content-first: UI gets out of the way
 * - Gesture-driven: Swipe actions on mobile
 * - Progressive disclosure: Summary → Content → Actions
 * - AI-aware: Subtle visual cues for AI features
 *
 * ## Key Features
 * - Swipe gestures (mobile): left to archive, right to delete
 * - Long-press for quick actions menu
 * - Expand/collapse with spring animation
 * - Contextual AI chip (when AI is relevant)
 * - Adaptive layout: 2-column grid on desktop, single on mobile
 */

import { useQueryClient } from "@tanstack/react-query";
import copy from "copy-to-clipboard";
import {
  Archive,
  ArchiveRestore,
  Bookmark,
  ChevronDown,
  Copy,
  Edit3,
  MessageCircle,
  MoreVertical,
  Pin,
  PinOff,
  Share2,
  Trash2,
} from "lucide-react";
import { memo, useCallback, useEffect, useMemo, useRef, useState } from "react";
import toast from "react-hot-toast";
import { useTranslation } from "react-i18next";
import { useLocation } from "react-router-dom";
import ConfirmDialog from "@/components/ConfirmDialog";
import MemoView from "@/components/MemoView/MemoView";
import { useInstance } from "@/contexts/InstanceContext";
import { useDeleteMemo, useUpdateMemo } from "@/hooks/useMemoQueries";
import useNavigateTo from "@/hooks/useNavigateTo";
import { userKeys } from "@/hooks/useUserQueries";
import { handleError } from "@/lib/error";
import { cn } from "@/lib/utils";
import { State } from "@/types/proto/api/v1/common_pb";
import type { Memo } from "@/types/proto/api/v1/memo_service_pb";
import { hasCompletedTasks, removeCompletedTasks } from "@/utils/markdown-manipulation";

// ============================================================================
// Design Tokens
// ============================================================================

const FLUID_THEME = {
  // Card states
  card: {
    base: "bg-white/80 dark:bg-zinc-900/80 backdrop-blur-xl",
    hover: "hover:bg-white dark:hover:bg-zinc-900",
    border: "border-zinc-200/60 dark:border-zinc-800/60",
    shadow: "shadow-sm hover:shadow-lg transition-all duration-300",
  },
  // Typography
  text: {
    primary: "text-zinc-900 dark:text-zinc-100",
    secondary: "text-zinc-500 dark:text-zinc-400",
    muted: "text-zinc-400 dark:text-zinc-500",
  },
  // Accents
  accent: {
    primary: "text-violet-600 dark:text-violet-400",
    bg: "bg-violet-500/10",
    border: "border-violet-200 dark:border-violet-800/50",
  },
  // Status indicators
  status: {
    pinned: "text-amber-500",
    archived: "text-zinc-400",
    ai: "text-violet-500",
  },
  // Motion
  spring: {
    default: "transition-all duration-300 cubic-bezier(0.34, 1.56, 0.64, 1)",
  },
} as const;

// ============================================================================
// Utilities
// ============================================================================

function formatRelativeTime(timestamp: number, t: (key: string, options?: Record<string, unknown>) => string): string {
  const date = new Date(timestamp);
  if (isNaN(date.getTime())) return t("common.unknown") || "?";

  const now = new Date();
  const diffMs = now.getTime() - date.getTime();
  const diffMins = Math.floor(diffMs / 60000);

  if (diffMins < 1) return "now";
  if (diffMins < 60) return `${diffMins}m`;
  if (diffMins < 1440) return `${Math.floor(diffMins / 60)}h`;
  return date.toLocaleDateString(undefined, { month: "short", day: "numeric" });
}

// ============================================================================
// Utilities
// ============================================================================

function getMemoId(memo: Memo): string {
  return memo.name.split("/").pop() || memo.name;
}

function getCollapseStorageKey(memoName: string): string {
  return `memo-block-collapsed-${memoName}`;
}

// ============================================================================
// Types
// ============================================================================

export interface MemoBlockV2Props {
  memo: Memo;
  isLatest?: boolean;
  onEdit?: (memo: Memo) => void;
  className?: string;
}

type QuickAction = "pin" | "archive" | "delete" | "copy" | "share";
type SwipeDirection = "left" | "right" | null;

// ============================================================================
// Main Component
// ============================================================================

export const MemoBlockV2 = memo(function MemoBlockV2({ memo, isLatest = false, onEdit, className }: MemoBlockV2Props) {
  const { t } = useTranslation();
  const location = useLocation();
  const navigateTo = useNavigateTo();
  const queryClient = useQueryClient();
  const { profile } = useInstance();
  const { mutateAsync: updateMemo } = useUpdateMemo();
  const { mutateAsync: deleteMemo } = useDeleteMemo();

  const memoId = getMemoId(memo);
  const isInMemoDetailPage = location.pathname.startsWith(`/${memo.name}`);
  const hasCompletedTaskList = hasCompletedTasks(memo.content);
  const isArchived = memo.state === State.ARCHIVED;

  // States
  const [isExpanded, setIsExpanded] = useState(() => {
    const contentLength = memo.content.length;
    return contentLength < 300 || isLatest;
  });
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [swipeDirection, setSwipeDirection] = useState<SwipeDirection>(null);
  const [quickMenuOpen, setQuickMenuOpen] = useState(false);

  // Refs for swipe detection
  const touchStartRef = useRef<{ x: number; y: number } | null>(null);
  const cardRef = useRef<HTMLDivElement>(null);

  // Persist collapse state
  useEffect(() => {
    try {
      localStorage.setItem(getCollapseStorageKey(memoId), String(!isExpanded));
    } catch {
      // ignore
    }
  }, [memoId, isExpanded]);

  // Action handlers
  const handleToggle = useCallback(() => setIsExpanded((prev) => !prev), []);

  const handleEdit = useCallback(() => {
    onEdit?.(memo);
  }, [memo, onEdit]);

  const handleTogglePin = useCallback(async () => {
    try {
      await updateMemo({
        update: { name: memo.name, pinned: !memo.pinned },
        updateMask: ["pinned"],
      });
    } catch {
      // silent
    }
  }, [memo.name, memo.pinned, updateMemo]);

  const handleToggleArchive = useCallback(async () => {
    const newState = memo.state === State.ARCHIVED ? State.NORMAL : State.ARCHIVED;
    const message = newState === State.ARCHIVED ? t("message.archived-successfully") : t("message.restored-successfully");

    try {
      await updateMemo({
        update: { name: memo.name, state: newState },
        updateMask: ["state"],
      });
      toast.success(message);
    } catch (error: unknown) {
      handleError(error, toast.error, {
        context: newState === State.ARCHIVED ? "Archive" : "Restore",
        fallbackMessage: "An error occurred",
      });
      return;
    }

    if (isInMemoDetailPage) {
      navigateTo(memo.state === State.ARCHIVED ? "/" : "/archived");
    }
    queryClient.invalidateQueries({ queryKey: userKeys.stats() });
  }, [memo.name, memo.state, t, isInMemoDetailPage, navigateTo, queryClient, updateMemo]);

  const handleCopy = useCallback(() => {
    copy(memo.content);
    toast.success(t("message.succeed-copy-content"));
  }, [memo.content, t]);

  const handleShare = useCallback(() => {
    const host = profile.instanceUrl || window.location.origin;
    const url = `${host}/${memo.name}`;

    if (navigator.share) {
      navigator.share({ title: t("memo.share-memo"), url });
    } else {
      copy(url);
      toast.success(t("message.succeed-copy-link"));
    }
  }, [memo.name, t, profile.instanceUrl]);

  const confirmDelete = useCallback(async () => {
    await deleteMemo(memo.name);
    toast.success(t("message.deleted-successfully"));
    if (isInMemoDetailPage) navigateTo("/");
    queryClient.invalidateQueries({ queryKey: userKeys.stats() });
  }, [memo.name, t, isInMemoDetailPage, navigateTo, queryClient, deleteMemo]);

  const handleRemoveTasks = useCallback(async () => {
    const newContent = removeCompletedTasks(memo.content);
    await updateMemo({
      update: { name: memo.name, content: newContent },
      updateMask: ["content"],
    });
    toast.success(t("message.remove-completed-task-list-items-successfully"));
    queryClient.invalidateQueries({ queryKey: userKeys.stats() });
  }, [memo.name, memo.content, t, queryClient, updateMemo]);

  // Touch handlers for swipe gestures
  const handleTouchStart = useCallback((e: React.TouchEvent) => {
    touchStartRef.current = { x: e.touches[0].clientX, y: e.touches[0].clientY };
  }, []);

  const handleTouchMove = useCallback((e: React.TouchEvent) => {
    if (!touchStartRef.current) return;

    const deltaX = e.touches[0].clientX - touchStartRef.current.x;
    const deltaY = Math.abs(e.touches[0].clientY - touchStartRef.current.y);

    // Only trigger if horizontal swipe (not vertical scroll)
    if (Math.abs(deltaX) > 30 && deltaY < 50) {
      setSwipeDirection(deltaX > 0 ? "right" : "left");
    }
  }, []);

  const handleTouchEnd = useCallback(() => {
    if (swipeDirection === "left") {
      handleToggleArchive(); // Swipe left to archive
    } else if (swipeDirection === "right") {
      setDeleteDialogOpen(true); // Swipe right to delete
    }
    touchStartRef.current = null;
    setSwipeDirection(null);
  }, [swipeDirection, handleToggleArchive]);

  // Memo metadata
  const previewText = useMemo(() => {
    const firstLine = memo.content.split("\n")[0];
    return firstLine.length > 120 ? firstLine.slice(0, 120) + "..." : firstLine;
  }, [memo.content]);

  const relativeTime = useMemo(() => {
    const timestamp = memo.displayTime ? Number(memo.displayTime) : Date.now();
    return formatRelativeTime(timestamp, t);
  }, [memo.displayTime, t]);

  const visibilityLabel = useMemo(() => {
    switch (memo.visibility) {
      case 1:
        return "Public";
      case 2:
        return "Protected";
      default:
        return "Private";
    }
  }, [memo.visibility]);

  // Quick actions menu
  const quickActions = useMemo(() => {
    const actions: Array<{ key: QuickAction; icon: typeof Edit3; label: string; action: () => void; danger?: boolean }> = [];

    if (!isArchived) {
      actions.push(
        { key: "pin", icon: memo.pinned ? PinOff : Pin, label: memo.pinned ? "Unpin" : "Pin", action: handleTogglePin },
        { key: "archive", icon: Archive, label: "Archive", action: handleToggleArchive },
      );
    } else {
      actions.push({ key: "archive", icon: ArchiveRestore, label: "Restore", action: handleToggleArchive });
    }

    if (!isArchived) {
      actions.push(
        { key: "copy", icon: Copy, label: "Copy", action: handleCopy },
        { key: "share", icon: Share2, label: "Share", action: handleShare },
      );
    }

    actions.push({ key: "delete", icon: Trash2, label: "Delete", action: () => setDeleteDialogOpen(true), danger: true });

    return actions;
  }, [isArchived, memo.pinned, handleTogglePin, handleToggleArchive, handleCopy, handleShare]);

  return (
    <>
      <div
        ref={cardRef}
        className={cn(
          // Base card styles
          "group relative rounded-lg overflow-hidden",
          FLUID_THEME.card.base,
          FLUID_THEME.card.border,
          FLUID_THEME.card.shadow,
          // Swipe indicators
          swipeDirection === "left" && "bg-amber-50 dark:bg-amber-950/30",
          swipeDirection === "right" && "bg-red-50 dark:bg-red-950/30",
          className,
        )}
        onTouchStart={handleTouchStart}
        onTouchMove={handleTouchMove}
        onTouchEnd={handleTouchEnd}
      >
        {/* Swipe action hints (mobile) */}
        {swipeDirection && (
          <div className="absolute inset-0 z-10 flex items-center justify-center bg-black/5 backdrop-blur-sm animate-in fade-in">
            <span
              className={cn(
                "text-sm font-medium",
                swipeDirection === "left" ? "text-amber-600 dark:text-amber-400" : "text-red-600 dark:text-red-400",
              )}
            >
              {swipeDirection === "left" ? "Archive →" : "← Delete"}
            </span>
          </div>
        )}

        {/* Main content */}
        <div className={FLUID_THEME.spring.default}>
          {/* Compact Header - Always visible */}
          <MemoCompactHeader
            memo={memo}
            previewText={previewText}
            relativeTime={relativeTime}
            visibilityLabel={visibilityLabel}
            isExpanded={isExpanded}
            onToggle={handleToggle}
            isArchived={isArchived}
          />

          {/* Expandable Content */}
          {isExpanded && (
            <div className="px-5 pb-4 animate-in slide-in-from-top-2 duration-200">
              <div className="border-t border-zinc-200/60 dark:border-zinc-800/60 pt-4">
                <MemoView
                  memo={memo}
                  showVisibility={false}
                  showPinned={false}
                  hideActionMenu={true}
                  hideInteractionButtons={false}
                  compact={false}
                  className="!border-0 !bg-transparent !shadow-none !p-0"
                />
              </div>

              {/* Task action */}
              {hasCompletedTaskList && !isArchived && !memo.parent && (
                <button
                  onClick={handleRemoveTasks}
                  className="mt-4 text-xs text-zinc-500 hover:text-zinc-700 dark:text-zinc-400 dark:hover:text-zinc-200 transition-colors"
                >
                  ✓ Clear completed tasks
                </button>
              )}
            </div>
          )}

          {/* Footer Actions - Compact bar */}
          <MemoCompactFooter
            memo={memo}
            isExpanded={isExpanded}
            onToggle={handleToggle}
            onEdit={handleEdit}
            onTogglePin={handleTogglePin}
            onCopy={handleCopy}
            onShare={handleShare}
            isArchived={isArchived}
            quickActions={quickActions}
            quickMenuOpen={quickMenuOpen}
            onQuickMenuToggle={() => setQuickMenuOpen((prev) => !prev)}
          />
        </div>

        {/* Status indicator line */}
        <div
          className={cn(
            "absolute left-0 top-0 bottom-0 w-1 transition-colors",
            memo.pinned && "bg-amber-500",
            memo.state === State.ARCHIVED && "bg-zinc-300 dark:bg-zinc-700",
            !memo.pinned && memo.state !== State.ARCHIVED && "bg-transparent",
          )}
        />
      </div>

      {/* Delete confirmation */}
      <ConfirmDialog
        open={deleteDialogOpen}
        onOpenChange={setDeleteDialogOpen}
        title={t("memo.delete-confirm")}
        confirmLabel={t("common.delete")}
        description={t("memo.delete-confirm-description")}
        cancelLabel={t("common.cancel")}
        onConfirm={confirmDelete}
        confirmVariant="destructive"
      />
    </>
  );
});

MemoBlockV2.displayName = "MemoBlockV2";

// ============================================================================
// Sub-components
// ============================================================================

interface MemoCompactHeaderProps {
  memo: Memo;
  previewText: string;
  relativeTime: string;
  visibilityLabel: string;
  isExpanded: boolean;
  onToggle: () => void;
  isArchived: boolean;
}

function MemoCompactHeader({ memo, previewText, relativeTime, visibilityLabel, isExpanded, onToggle, isArchived }: MemoCompactHeaderProps) {
  return (
    <div className="flex items-start gap-3 p-4">
      {/* Icon indicator */}
      <div
        onClick={onToggle}
        className={cn(
          "mt-0.5 w-8 h-8 rounded-full flex items-center justify-center shrink-0 cursor-pointer",
          memo.pinned
            ? "bg-amber-100 dark:bg-amber-900/30 text-amber-600 dark:text-amber-400"
            : "bg-zinc-100 dark:bg-zinc-800 text-zinc-500 dark:text-zinc-400",
        )}
      >
        {memo.pinned ? <Pin className="w-4 h-4" /> : <Bookmark className="w-4 h-4" />}
      </div>

      {/* Content preview */}
      <div onClick={onToggle} className="flex-1 min-w-0 cursor-pointer">
        <p className={cn("text-sm leading-relaxed", isArchived ? "text-zinc-400 line-through" : FLUID_THEME.text.primary)}>{previewText}</p>

        {/* Metadata row */}
        <div className="flex items-center gap-3 mt-2 text-xs">
          <span className={FLUID_THEME.text.muted}>{relativeTime}</span>
          <span
            className={cn(
              "px-2 py-0.5 rounded-full",
              memo.visibility === 1
                ? "bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-400"
                : memo.visibility === 2
                  ? "bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400"
                  : "bg-zinc-100 text-zinc-600 dark:bg-zinc-800 dark:text-zinc-400",
            )}
          >
            {visibilityLabel}
          </span>
          {memo.parent && (
            <span className="flex items-center gap-1 text-zinc-400">
              <MessageCircle className="w-3 h-3" />
              Comment
            </span>
          )}
        </div>
      </div>

      {/* Right side actions */}
      <div className="flex items-center gap-1 shrink-0">
        {/* Expand/Collapse chevron */}
        <button
          onClick={onToggle}
          className={cn(
            "p-1.5 rounded-lg transition-all",
            "text-zinc-400 hover:text-zinc-600 hover:bg-zinc-100",
            "dark:text-zinc-500 dark:hover:text-zinc-300 dark:hover:bg-zinc-800",
            "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-violet-500/50",
          )}
          aria-label={isExpanded ? "Collapse" : "Expand"}
        >
          <ChevronDown className={cn("w-5 h-5 transition-transform duration-200", !isExpanded && "-rotate-90")} />
        </button>
      </div>
    </div>
  );
}

interface MemoCompactFooterProps {
  memo: Memo;
  isExpanded: boolean;
  onToggle: () => void;
  onEdit: () => void;
  onTogglePin: () => void;
  onCopy: () => void;
  onShare: () => void;
  isArchived: boolean;
  quickActions: Array<{
    key: QuickAction;
    icon: typeof Edit3;
    label: string;
    action: () => void;
    danger?: boolean;
  }>;
  quickMenuOpen: boolean;
  onQuickMenuToggle: () => void;
}

function MemoCompactFooter({
  memo,
  isExpanded,
  onToggle,
  onEdit,
  onTogglePin,
  onCopy,
  onShare,
  isArchived,
  quickActions,
  quickMenuOpen,
  onQuickMenuToggle,
}: MemoCompactFooterProps) {
  return (
    <div className="flex items-center justify-between px-4 py-2 border-t border-zinc-200/60 dark:border-zinc-800/60 bg-zinc-50/50 dark:bg-zinc-900/30">
      {/* Left: Toggle button */}
      <button
        onClick={onToggle}
        className="flex items-center gap-1.5 text-xs text-zinc-500 hover:text-zinc-700 dark:text-zinc-400 dark:hover:text-zinc-200 transition-colors"
      >
        <ChevronDown className={cn("w-4 h-4 transition-transform", !isExpanded && "-rotate-90")} />
        <span className="hidden sm:inline">{isExpanded ? "Collapse" : "Expand"}</span>
      </button>

      {/* Right: Primary actions */}
      <div className="flex items-center gap-1">
        {/* Edit button - always visible when not archived */}
        {!isArchived && <ActionButton icon={Edit3} label="Edit" onClick={onEdit} />}

        {/* Pin button - visible for root memos */}
        {!memo.parent && !isArchived && (
          <ActionButton
            icon={memo.pinned ? PinOff : Pin}
            label={memo.pinned ? "Unpin" : "Pin"}
            onClick={onTogglePin}
            className={memo.pinned ? "text-amber-600 dark:text-amber-400" : undefined}
          />
        )}

        {/* Copy button */}
        {!isArchived && <ActionButton icon={Copy} label="Copy" onClick={onCopy} />}

        {/* Share button - if available */}
        {!isArchived && typeof navigator !== "undefined" && "share" in navigator && (
          <ActionButton icon={Share2} label="Share" onClick={onShare} />
        )}

        {/* More menu */}
        <div className="relative">
          <ActionButton icon={MoreVertical} label="More" onClick={onQuickMenuToggle} isActive={quickMenuOpen} />

          {/* Dropdown menu */}
          {quickMenuOpen && (
            <div className="absolute right-0 bottom-full mb-2 w-48 py-2 bg-white dark:bg-zinc-900 rounded-lg shadow-lg border border-zinc-200 dark:border-zinc-800 z-50 animate-in fade-in slide-in-from-bottom-2 duration-150">
              {quickActions.map((action) => (
                <button
                  key={action.key}
                  onClick={() => {
                    action.action();
                    onQuickMenuToggle();
                  }}
                  className={cn(
                    "w-full flex items-center gap-3 px-4 py-2 text-sm text-left hover:bg-zinc-100 dark:hover:bg-zinc-800 transition-colors",
                    action.danger && "text-red-600 dark:text-red-400 hover:bg-red-50 dark:hover:bg-red-950/30",
                  )}
                >
                  <action.icon className="w-4 h-4" />
                  <span>{action.label}</span>
                </button>
              ))}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

interface ActionButtonProps {
  icon: typeof Edit3;
  label: string;
  onClick: () => void;
  className?: string;
  isActive?: boolean;
}

function ActionButton({ icon: Icon, label, onClick, className, isActive }: ActionButtonProps) {
  return (
    <button
      onClick={onClick}
      className={cn(
        "p-2 rounded-lg text-zinc-500 hover:text-zinc-700 hover:bg-zinc-100 dark:text-zinc-400 dark:hover:text-zinc-200 dark:hover:bg-zinc-800 transition-all",
        "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-violet-500/50",
        isActive && "bg-zinc-200 dark:bg-zinc-800",
        className,
      )}
      aria-label={label}
    >
      <Icon className="w-4 h-4" />
    </button>
  );
}
