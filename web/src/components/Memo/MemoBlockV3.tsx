/**
 * MemoBlockV3 - "Sticky Note" Design DNA
 *
 * 设计哲学：「彩色便签纸」
 * - Apple Notes 风格彩色背景
 * - 标签自动映射颜色
 * - 微妙折角效果
 * - **保留 V2 所有能力**：折叠展开、Markdown渲染、滑动手势、任务操作
 *
 * ## Key Features (继承自 V2)
 * - Swipe gestures (mobile): left to archive, right to delete
 * - Long-press for quick actions menu
 * - Expand/collapse with spring animation
 * - MemoView for Markdown rendering (展开时)
 * - Task list actions
 *
 * ## New in V3
 * - Tag-based color mapping (6 color palette)
 * - Paper-like shadow and fold effect
 * - 200-char smart preview (折叠时)
 */

import { timestampDate } from "@bufbuild/protobuf/wkt";
import { useQueryClient } from "@tanstack/react-query";
import copy from "copy-to-clipboard";
import {
  Archive,
  ArchiveRestore,
  Bookmark,
  ChevronDown,
  ChevronUp,
  Copy,
  Edit3,
  Ellipsis,
  MessageCircle,
  Pin,
  PinOff,
  Share2,
  Trash2,
  X,
} from "lucide-react";
import { memo, useCallback, useEffect, useMemo, useRef, useState } from "react";
import { createPortal } from "react-dom";
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
import { Visibility } from "@/types/proto/api/v1/memo_service_pb";
import { hasCompletedTasks, removeCompletedTasks } from "@/utils/markdown-manipulation";
import { getMemoColorClasses } from "@/utils/tag-colors";
import { generatePreview } from "@/utils/text";

// ============================================================================
// Types
// ============================================================================

export interface MemoBlockV3Props {
  memo: Memo;
  onEdit?: (memoName: string) => void;
  className?: string;
}

type QuickAction = "pin" | "archive" | "delete" | "copy" | "share";
type SwipeDirection = "left" | "right" | null;

// ============================================================================
// Utilities
// ============================================================================

function formatRelativeTime(timestamp: number, t: (key: string, options?: Record<string, unknown>) => string): string {
  const date = new Date(timestamp);
  if (isNaN(date.getTime())) return t("common.unknown") || "?";

  const now = new Date();
  const diffMs = now.getTime() - date.getTime();
  const diffMins = Math.floor(diffMs / 60000);

  if (diffMins < 1) return t("common.now");
  if (diffMins < 60) return t("common.minutes-ago", { count: diffMins });
  if (diffMins < 1440) return t("common.hours-ago", { count: Math.floor(diffMins / 60) });
  return date.toLocaleDateString(undefined, { month: "short", day: "numeric" });
}

// ============================================================================
// Main Component
// ============================================================================

export const MemoBlockV3 = memo(function MemoBlockV3({ memo, onEdit, className }: MemoBlockV3Props) {
  const { t } = useTranslation();
  const location = useLocation();
  const navigateTo = useNavigateTo();
  const queryClient = useQueryClient();
  const { profile } = useInstance();
  const { mutateAsync: updateMemo } = useUpdateMemo();
  const { mutateAsync: deleteMemo } = useDeleteMemo();

  const isInMemoDetailPage = location.pathname.startsWith(`/${memo.name}`);
  const hasCompletedTaskList = hasCompletedTasks(memo.content);
  const isArchived = memo.state === State.ARCHIVED;
  const tags = memo.tags || [];

  // Get color scheme based on tags
  const colorClasses = useMemo(() => getMemoColorClasses(tags), [tags]);

  // States - initialize from localStorage or use default based on content length
  const [isExpanded, setIsExpanded] = useState(() => {
    const memoId = memo.name.split("/").pop() || memo.name;
    try {
      const saved = localStorage.getItem(`memo-block-collapsed-${memoId}`);
      if (saved !== null) {
        return saved === "false"; // saved value is "collapsed" state, so false = expanded
      }
    } catch {
      // localStorage unavailable, use default
    }
    // Default: expand short content
    const contentLength = memo.content.length;
    return contentLength < 300;
  });
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [swipeDirection, setSwipeDirection] = useState<SwipeDirection>(null);
  const [quickMenuOpen, setQuickMenuOpen] = useState(false);

  // Refs for swipe detection and click-outside
  const touchStartRef = useRef<{ x: number; y: number } | null>(null);
  const cardRef = useRef<HTMLDivElement>(null);
  const dropdownRef = useRef<HTMLDivElement>(null);
  const dropdownButtonRef = useRef<HTMLButtonElement>(null);
  const [dropdownPosition, setDropdownPosition] = useState<{
    top: number;
    right: number;
  } | null>(null);

  // Persist collapse state
  useEffect(() => {
    const memoId = memo.name.split("/").pop() || memo.name;
    try {
      localStorage.setItem(`memo-block-collapsed-${memoId}`, String(!isExpanded));
    } catch {
      // localStorage unavailable (private browsing, quota exceeded, etc.)
      // Non-critical: collapse state won't persist across sessions
    }
  }, [memo.name, isExpanded]);

  // Close dropdown when clicking outside
  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (dropdownRef.current && !dropdownRef.current.contains(event.target as Node)) {
        setQuickMenuOpen(false);
      }
    };

    if (quickMenuOpen) {
      document.addEventListener("mousedown", handleClickOutside);
      return () => document.removeEventListener("mousedown", handleClickOutside);
    }
  }, [quickMenuOpen]);

  // Close dropdown on escape key
  useEffect(() => {
    const handleEscape = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        setQuickMenuOpen(false);
      }
    };

    if (quickMenuOpen) {
      document.addEventListener("keydown", handleEscape);
      return () => document.removeEventListener("keydown", handleEscape);
    }
  }, [quickMenuOpen]);

  // Calculate dropdown position when opened
  useEffect(() => {
    if (quickMenuOpen && dropdownButtonRef.current) {
      const rect = dropdownButtonRef.current.getBoundingClientRect();
      setDropdownPosition({
        top: rect.bottom + 4,
        right: window.innerWidth - rect.right,
      });
    } else {
      setDropdownPosition(null);
    }
  }, [quickMenuOpen]);

  // Action handlers
  const handleToggle = useCallback(() => setIsExpanded((prev) => !prev), []);

  const handleEdit = useCallback(() => {
    onEdit?.(memo.name);
  }, [memo.name, onEdit]);

  const handleTogglePin = useCallback(async () => {
    try {
      await updateMemo({
        update: { name: memo.name, pinned: !memo.pinned },
        updateMask: ["pinned"],
      });
    } catch (error: unknown) {
      handleError(error, toast.error, {
        context: "Pin",
        fallbackMessage: t("message.failed-toggle-pin"),
      });
    }
  }, [memo.name, memo.pinned, updateMemo, t]);

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
        fallbackMessage: t("message.error-occurred"),
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
    try {
      await deleteMemo(memo.name);
      toast.success(t("message.deleted-successfully"));
      if (isInMemoDetailPage) navigateTo("/");
      queryClient.invalidateQueries({ queryKey: userKeys.stats() });
    } catch (error: unknown) {
      handleError(error, toast.error, {
        context: "Delete",
        fallbackMessage: t("message.failed-to-delete"),
      });
    }
  }, [memo.name, t, isInMemoDetailPage, navigateTo, queryClient, deleteMemo]);

  const handleRemoveTasks = useCallback(async () => {
    try {
      const newContent = removeCompletedTasks(memo.content);
      await updateMemo({
        update: { name: memo.name, content: newContent },
        updateMask: ["content"],
      });
      toast.success(t("message.remove-completed-task-list-items-successfully"));
      queryClient.invalidateQueries({ queryKey: userKeys.stats() });
    } catch (error: unknown) {
      handleError(error, toast.error, {
        context: "RemoveTasks",
        fallbackMessage: t("message.failed-remove-tasks"),
      });
    }
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
    return generatePreview(memo.content, { maxLength: 120 });
  }, [memo.content]);

  const relativeTime = useMemo(() => {
    const date = memo.displayTime ? timestampDate(memo.displayTime) : new Date();
    return formatRelativeTime(date.getTime(), t);
  }, [memo.displayTime, t]);

  const visibilityLabel = useMemo(() => {
    switch (memo.visibility) {
      case Visibility.PUBLIC:
        return t("memo.visibility.public");
      case Visibility.PROTECTED:
        return t("memo.visibility.protected");
      default:
        return t("memo.visibility.private");
    }
  }, [memo.visibility, t]);

  // Quick actions menu
  const quickActions = useMemo(() => {
    const actions: Array<{
      key: QuickAction;
      icon: typeof Edit3;
      label: string;
      action: () => void;
      danger?: boolean;
    }> = [];

    if (!isArchived) {
      actions.push({ key: "archive", icon: Archive, label: t("common.archive"), action: handleToggleArchive });
    } else {
      actions.push({
        key: "archive",
        icon: ArchiveRestore,
        label: t("common.restore"),
        action: handleToggleArchive,
      });
    }

    actions.push({
      key: "delete",
      icon: Trash2,
      label: t("common.delete"),
      action: () => setDeleteDialogOpen(true),
      danger: true,
    });

    return actions;
  }, [isArchived, handleToggleArchive, t]);

  return (
    <>
      <div
        ref={cardRef}
        className={cn(
          // Base card styles
          "group relative rounded-lg overflow-hidden",
          // Sticky note background
          colorClasses.bg,
          "border",
          colorClasses.border,
          // Paper shadow
          "shadow-sm hover:shadow-md transition-all duration-200",
          // Swipe indicators
          swipeDirection === "left" && "bg-amber-50/95 dark:bg-amber-950/20",
          swipeDirection === "right" && "bg-red-50/95 dark:bg-red-950/20",
          className,
        )}
        onTouchStart={handleTouchStart}
        onTouchMove={handleTouchMove}
        onTouchEnd={handleTouchEnd}
      >
        {/* Fold corner effect */}
        <div className="sticky-note-corner" />

        {/* Swipe action hints (mobile) */}
        {swipeDirection && (
          <div className="absolute inset-0 z-10 flex items-center justify-center bg-black/5 backdrop-blur-sm animate-in fade-in">
            <span
              className={cn(
                "text-sm font-medium px-4 py-2 rounded-full",
                swipeDirection === "left"
                  ? "text-amber-600 bg-amber-100 dark:bg-amber-900/30"
                  : "text-red-600 bg-red-100 dark:bg-red-900/30",
              )}
            >
              {swipeDirection === "left" ? t("memo.swipe_archive") : t("memo.swipe_delete")}
            </span>
          </div>
        )}

        {/* Status indicator line */}
        <div
          className={cn(
            "absolute left-0 top-0 bottom-0 w-1 rounded-l-lg transition-colors",
            memo.pinned && "bg-amber-500",
            memo.state === State.ARCHIVED && "bg-zinc-300 dark:bg-zinc-700",
            !memo.pinned && memo.state !== State.ARCHIVED && "bg-transparent",
          )}
        />

        {/* Main content */}
        <div className="transition-all duration-200 ease-out">
          {/* Compact Header - Always visible */}
          <MemoCompactHeader
            memo={memo}
            previewText={previewText}
            relativeTime={relativeTime}
            visibilityLabel={visibilityLabel}
            isExpanded={isExpanded}
            onToggle={handleToggle}
            isArchived={isArchived}
            colorClasses={colorClasses}
          />

          {/* Expandable Content - Markdown Rendering */}
          {isExpanded && (
            <div className="px-5 pb-4 animate-in slide-in-from-top-2 duration-200">
              <div className="border-t border-current/10 pt-4">
                <MemoView
                  memo={memo}
                  showCreator={true}
                  showVisibility={false}
                  showPinned={false}
                  hideActionMenu={true}
                  hideInteractionButtons={true}
                  compact={false}
                  className="border-0 bg-transparent shadow-none p-0"
                />
              </div>

              {/* Task action */}
              {hasCompletedTaskList && !isArchived && !memo.parent && (
                <button
                  onClick={handleRemoveTasks}
                  className={cn(
                    "mt-4 text-xs transition-colors px-3 py-1.5 rounded-lg",
                    "hover:bg-black/5 dark:hover:bg-white/10",
                    colorClasses.muted,
                  )}
                >
                  {t("memo.clear_completed_tasks")}
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
            onToggleArchive={handleToggleArchive}
            onDelete={() => setDeleteDialogOpen(true)}
            isArchived={isArchived}
            quickActions={quickActions}
            quickMenuOpen={quickMenuOpen}
            onQuickMenuToggle={() => setQuickMenuOpen((prev) => !prev)}
            dropdownRef={dropdownRef}
            dropdownButtonRef={dropdownButtonRef}
            dropdownPosition={dropdownPosition}
            colorClasses={colorClasses}
          />
        </div>
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

MemoBlockV3.displayName = "MemoBlockV3";

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
  colorClasses: { text: string; muted: string; tag: string };
}

function MemoCompactHeader({
  memo,
  previewText,
  relativeTime,
  visibilityLabel,
  isExpanded,
  onToggle,
  isArchived,
  colorClasses,
}: MemoCompactHeaderProps) {
  const { t } = useTranslation();

  return (
    <div className="flex items-start gap-3 p-4">
      {/* Icon indicator - clickable for toggle */}
      <button
        onClick={onToggle}
        className={cn(
          "mt-0.5 w-9 h-9 rounded-full flex items-center justify-center shrink-0 transition-all",
          "hover:scale-105 active:scale-95",
          memo.pinned
            ? "bg-amber-100 dark:bg-amber-900/30 text-amber-600 dark:text-amber-400 hover:bg-amber-200 dark:hover:bg-amber-900/50"
            : "bg-black/5 dark:bg-white/10 hover:bg-black/10 dark:hover:bg-white/20",
          colorClasses.text,
        )}
        aria-label={isExpanded ? t("common.collapse") : t("common.expand")}
      >
        {memo.pinned ? <Pin className="w-4 h-4" /> : <Bookmark className="w-4 h-4" />}
      </button>

      {/* Content preview - clickable for toggle */}
      <button onClick={onToggle} className="flex-1 min-w-0 text-left">
        <p className={cn("text-sm leading-relaxed text-left w-full", isArchived && "line-through opacity-60", colorClasses.text)}>
          {previewText}
        </p>

        {/* Metadata row */}
        <div className="flex items-center gap-2.5 mt-2 text-xs">
          <span className={colorClasses.muted}>{relativeTime}</span>
          <span
            className={cn(
              "px-2 py-0.5 rounded-full text-[11px] font-medium",
              memo.visibility === Visibility.PUBLIC
                ? "bg-emerald-500/20 text-emerald-700 dark:text-emerald-300"
                : memo.visibility === Visibility.PROTECTED
                  ? "bg-blue-500/20 text-blue-700 dark:text-blue-300"
                  : "bg-black/10 text-zinc-600 dark:text-zinc-400",
            )}
          >
            {visibilityLabel}
          </span>
          {memo.parent && (
            <span className={cn("flex items-center gap-1", colorClasses.muted)}>
              <MessageCircle className="w-3 h-3" />
              {t("memo.comment_label")}
            </span>
          )}
        </div>
      </button>

      {/* Right side - collapse/expand button */}
      <button
        onClick={onToggle}
        className={cn(
          "p-2 rounded-lg transition-all shrink-0",
          "hover:bg-black/5 dark:hover:bg-white/10",
          "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-current/30",
          colorClasses.muted,
        )}
        aria-label={isExpanded ? t("common.collapse") : t("common.expand")}
      >
        {isExpanded ? <ChevronUp className="w-5 h-5" /> : <ChevronDown className="w-5 h-5" />}
      </button>
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
  onToggleArchive: () => void;
  onDelete: () => void;
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
  dropdownRef: React.RefObject<HTMLDivElement>;
  dropdownButtonRef: React.RefObject<HTMLButtonElement>;
  dropdownPosition: { top: number; right: number } | null;
  colorClasses: { muted: string };
}

function MemoCompactFooter({
  memo,
  isExpanded,
  onToggle,
  onEdit,
  onTogglePin,
  onCopy,
  onShare,
  onToggleArchive,
  onDelete,
  isArchived,
  quickActions,
  quickMenuOpen,
  onQuickMenuToggle,
  dropdownRef,
  dropdownButtonRef,
  dropdownPosition,
  colorClasses,
}: MemoCompactFooterProps) {
  const { t } = useTranslation();

  return (
    <div
      className={cn("flex items-center justify-between px-4 py-2.5", "border-t border-current/10", "bg-black/[0.02] dark:bg-white/[0.02]")}
    >
      {/* Left: Collapse/Expand indicator */}
      <button
        onClick={onToggle}
        className={cn(
          "flex items-center gap-2 text-xs transition-colors px-2 py-1 rounded-lg",
          "hover:bg-black/5 dark:hover:bg-white/10",
          colorClasses.muted,
        )}
      >
        {isExpanded ? <ChevronUp className="w-3.5 h-3.5" /> : <ChevronDown className="w-3.5 h-3.5" />}
        <span className="hidden sm:inline">{isExpanded ? t("memo.show-less") : t("memo.show-more")}</span>
      </button>

      {/* Right: Primary actions */}
      <div className="flex items-center gap-0.5">
        {/* Edit button - always visible when not archived */}
        {!isArchived && <ActionButton icon={Edit3} label={t("common.edit")} onClick={onEdit} colorClasses={colorClasses} />}

        {/* Pin button - visible for root memos */}
        {!memo.parent && !isArchived && (
          <ActionButton
            icon={memo.pinned ? PinOff : Pin}
            label={memo.pinned ? t("common.unpin") : t("common.pin")}
            onClick={onTogglePin}
            colorClasses={colorClasses}
            className={memo.pinned ? "text-amber-600 dark:text-amber-400 bg-amber-50 dark:bg-amber-900/20" : undefined}
          />
        )}

        {/* Copy button */}
        {!isArchived && <ActionButton icon={Copy} label={t("common.copy")} onClick={onCopy} colorClasses={colorClasses} />}

        {/* Share button - if available */}
        {!isArchived && typeof navigator !== "undefined" && "share" in navigator && (
          <ActionButton icon={Share2} label={t("common.share")} onClick={onShare} colorClasses={colorClasses} />
        )}

        {/* Desktop (sm+): Show Archive button directly */}
        <div className="hidden sm:block">
          <ActionButton
            icon={isArchived ? ArchiveRestore : Archive}
            label={isArchived ? t("common.restore") : t("common.archive")}
            onClick={onToggleArchive}
            colorClasses={colorClasses}
          />
        </div>

        {/* Desktop (sm+): Show Delete button directly */}
        <div className="hidden sm:block">
          <ActionButton
            icon={Trash2}
            label={t("common.delete")}
            onClick={onDelete}
            colorClasses={colorClasses}
            className="text-red-600 hover:text-red-700 dark:text-red-400 dark:hover:text-red-300"
          />
        </div>

        {/* Mobile only: More menu dropdown */}
        <div className="sm:hidden relative">
          <ActionButton
            icon={Ellipsis}
            label={t("memo.more_actions")}
            onClick={onQuickMenuToggle}
            colorClasses={colorClasses}
            isActive={quickMenuOpen}
            buttonRef={dropdownButtonRef}
          />
        </div>
      </div>

      {/* Mobile only: Dropdown menu - rendered via Portal */}
      {quickMenuOpen &&
        dropdownPosition &&
        createPortal(
          <div
            ref={dropdownRef}
            className={cn(
              "fixed z-[100] sm:hidden",
              "w-48 py-1.5 bg-white dark:bg-zinc-900",
              "rounded-lg shadow-lg shadow-zinc-200/50 dark:shadow-black/50",
              "border border-zinc-200 dark:border-zinc-800",
              "animate-in fade-in slide-in-from-top-1 duration-150",
            )}
            style={{
              top: `${dropdownPosition.top}px`,
              right: `${dropdownPosition.right}px`,
            }}
          >
            {/* Header with close button */}
            <div className="flex items-center justify-between px-3 py-2 border-b border-zinc-100 dark:border-zinc-800">
              <span className="text-xs font-medium text-zinc-500 dark:text-zinc-400">{t("memo.actions")}</span>
              <button
                onClick={() => onQuickMenuToggle()}
                className="p-1 rounded hover:bg-zinc-100 dark:hover:bg-zinc-800 transition-colors"
                aria-label={t("common.close")}
              >
                <X className="w-3.5 h-3.5 text-zinc-400" />
              </button>
            </div>

            {/* Action items */}
            <div className="py-1">
              {quickActions.map((action) => (
                <button
                  key={action.key}
                  onClick={() => {
                    action.action();
                    onQuickMenuToggle();
                  }}
                  className={cn(
                    "w-full flex items-center gap-3 px-3 py-2 text-sm text-left",
                    "transition-colors duration-100",
                    "hover:bg-zinc-100 dark:hover:bg-zinc-800",
                    "focus-visible:outline-none focus-visible:bg-zinc-100 dark:focus-visible:bg-zinc-800",
                    action.danger && "text-red-600 dark:text-red-400 hover:bg-red-50 dark:hover:bg-red-950/30",
                  )}
                >
                  <action.icon className="w-4 h-4 shrink-0" />
                  <span>{action.label}</span>
                </button>
              ))}
            </div>
          </div>,
          document.body,
        )}
    </div>
  );
}

// ============================================================================
// Action Button Component
// ============================================================================

interface ActionButtonProps {
  icon: typeof Edit3;
  label: string;
  onClick: () => void;
  colorClasses: { muted: string };
  className?: string;
  isActive?: boolean;
  buttonRef?: React.RefObject<HTMLButtonElement>;
}

function ActionButton({ icon: Icon, label, onClick, colorClasses, className, isActive, buttonRef }: ActionButtonProps) {
  return (
    <button
      type="button"
      ref={buttonRef}
      onClick={onClick}
      className={cn(
        "p-2 rounded-lg transition-all duration-150",
        colorClasses.muted,
        "hover:bg-black/5 dark:hover:bg-white/10",
        "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-current/30",
        "active:scale-95",
        isActive && "bg-black/10 dark:bg-white/20",
        className,
      )}
      aria-label={label}
      title={label}
    >
      <Icon className="w-4 h-4" />
    </button>
  );
}
