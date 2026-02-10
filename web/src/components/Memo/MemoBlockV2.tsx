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
 *
 * ## UX Improvements (v2.1)
 * - Click-outside to close dropdown
 * - Proper dropdown positioning with boundary detection
 * - Synced collapse/expand state across all controls
 * - Better mobile touch handling
 * - Smooth animations and transitions
 */

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
import { hasCompletedTasks, removeCompletedTasks } from "@/utils/markdown-manipulation";

// ============================================================================
// Design Tokens
// ============================================================================

const FLUID_THEME = {
  // Card states
  card: {
    base: "bg-white/90 dark:bg-zinc-900/90 backdrop-blur-xl",
    hover: "hover:bg-white dark:hover:bg-zinc-900",
    border: "border-zinc-200/70 dark:border-zinc-800/70",
    shadow: "shadow-sm hover:shadow-sm transition-all duration-200",
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
    default: "transition-all duration-200 ease-out",
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

  const memoId = memo.name.split("/").pop() || memo.name;
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

  // Refs for swipe detection and click-outside
  const touchStartRef = useRef<{ x: number; y: number } | null>(null);
  const cardRef = useRef<HTMLDivElement>(null);
  const dropdownRef = useRef<HTMLDivElement>(null);
  const dropdownButtonRef = useRef<HTMLButtonElement>(null);
  const [dropdownPosition, setDropdownPosition] = useState<{ top: number; left: number; right: number } | null>(null);

  // Persist collapse state
  useEffect(() => {
    try {
      localStorage.setItem(`memo-block-collapsed-${memoId}`, String(!isExpanded));
    } catch {
      // ignore
    }
  }, [memoId, isExpanded]);

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
        left: rect.left,
        right: window.innerWidth - rect.right,
      });
    } else {
      setDropdownPosition(null);
    }
  }, [quickMenuOpen]);

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

  // Quick actions menu - only show actions not visible in footer
  const quickActions = useMemo(() => {
    const actions: Array<{ key: QuickAction; icon: typeof Edit3; label: string; action: () => void; danger?: boolean }> = [];

    // Archive/Restore (always in dropdown, not in footer)
    if (!isArchived) {
      actions.push({ key: "archive", icon: Archive, label: "Archive", action: handleToggleArchive });
    } else {
      actions.push({ key: "archive", icon: ArchiveRestore, label: "Restore", action: handleToggleArchive });
    }

    // Delete (always in dropdown, not in footer)
    actions.push({ key: "delete", icon: Trash2, label: "Delete", action: () => setDeleteDialogOpen(true), danger: true });

    return actions;
  }, [isArchived, handleToggleArchive]);

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
          swipeDirection === "left" && "bg-amber-50/95 dark:bg-amber-950/20",
          swipeDirection === "right" && "bg-red-50/95 dark:bg-red-950/20",
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
                "text-sm font-medium px-4 py-2 rounded-full",
                swipeDirection === "left"
                  ? "text-amber-600 bg-amber-100 dark:bg-amber-900/30"
                  : "text-red-600 bg-red-100 dark:bg-red-900/30",
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
                  className="mt-4 text-xs text-zinc-500 hover:text-zinc-700 dark:text-zinc-400 dark:hover:text-zinc-200 transition-colors px-3 py-1.5 rounded-lg hover:bg-zinc-100 dark:hover:bg-zinc-800"
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
            onToggleArchive={handleToggleArchive}
            onDelete={() => setDeleteDialogOpen(true)}
            isArchived={isArchived}
            quickActions={quickActions}
            quickMenuOpen={quickMenuOpen}
            onQuickMenuToggle={() => setQuickMenuOpen((prev) => !prev)}
            dropdownRef={dropdownRef}
            dropdownButtonRef={dropdownButtonRef}
            dropdownPosition={dropdownPosition}
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
      {/* Icon indicator - clickable for toggle */}
      <button
        onClick={onToggle}
        className={cn(
          "mt-0.5 w-9 h-9 rounded-full flex items-center justify-center shrink-0 transition-all",
          "hover:scale-105 active:scale-95",
          memo.pinned
            ? "bg-amber-100 dark:bg-amber-900/30 text-amber-600 dark:text-amber-400 hover:bg-amber-200 dark:hover:bg-amber-900/50"
            : "bg-zinc-100 dark:bg-zinc-800 text-zinc-500 dark:text-zinc-400 hover:bg-zinc-200 dark:hover:bg-zinc-700",
        )}
        aria-label={isExpanded ? "Collapse" : "Expand"}
      >
        {memo.pinned ? <Pin className="w-4 h-4" /> : <Bookmark className="w-4 h-4" />}
      </button>

      {/* Content preview - clickable for toggle */}
      <button onClick={onToggle} className="flex-1 min-w-0 text-left">
        <p
          className={cn(
            "text-sm leading-relaxed text-left w-full",
            isArchived ? "text-zinc-400 line-through" : FLUID_THEME.text.primary,
            "hover:text-zinc-700 dark:hover:text-zinc-300 transition-colors",
          )}
        >
          {previewText}
        </p>

        {/* Metadata row */}
        <div className="flex items-center gap-2.5 mt-2 text-xs">
          <span className={FLUID_THEME.text.muted}>{relativeTime}</span>
          <span
            className={cn(
              "px-2 py-0.5 rounded-full text-[11px] font-medium",
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
      </button>

      {/* Right side - collapse/expand button */}
      <button
        onClick={onToggle}
        className={cn(
          "p-2 rounded-lg transition-all shrink-0",
          "text-zinc-400 hover:text-zinc-600 hover:bg-zinc-100",
          "dark:text-zinc-500 dark:hover:text-zinc-300 dark:hover:bg-zinc-800",
          "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-violet-500/50",
        )}
        aria-label={isExpanded ? "Collapse" : "Expand"}
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
  dropdownPosition: { top: number; left: number; right: number } | null;
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
}: MemoCompactFooterProps) {
  return (
    <div className="flex items-center justify-between px-4 py-2.5 border-t border-zinc-200/60 dark:border-zinc-800/60 bg-zinc-50/50 dark:bg-zinc-900/30">
      {/* Left: Collapse/Expand indicator */}
      <button
        onClick={onToggle}
        className={cn(
          "flex items-center gap-2 text-xs text-zinc-500 hover:text-zinc-700 dark:text-zinc-400 dark:hover:text-zinc-200",
          "transition-colors px-2 py-1 rounded-lg hover:bg-zinc-100 dark:hover:bg-zinc-800",
        )}
      >
        {isExpanded ? <ChevronUp className="w-3.5 h-3.5" /> : <ChevronDown className="w-3.5 h-3.5" />}
        <span className="hidden sm:inline">{isExpanded ? "Show less" : "Show more"}</span>
      </button>

      {/* Right: Primary actions */}
      <div className="flex items-center gap-0.5">
        {/* Edit button - always visible when not archived */}
        {!isArchived && <ActionButton icon={Edit3} label="Edit" onClick={onEdit} />}

        {/* Pin button - visible for root memos */}
        {!memo.parent && !isArchived && (
          <ActionButton
            icon={memo.pinned ? PinOff : Pin}
            label={memo.pinned ? "Unpin" : "Pin"}
            onClick={onTogglePin}
            className={memo.pinned ? "text-amber-600 dark:text-amber-400 bg-amber-50 dark:bg-amber-900/20" : undefined}
          />
        )}

        {/* Copy button */}
        {!isArchived && <ActionButton icon={Copy} label="Copy" onClick={onCopy} />}

        {/* Share button - if available */}
        {!isArchived && typeof navigator !== "undefined" && "share" in navigator && (
          <ActionButton icon={Share2} label="Share" onClick={onShare} />
        )}

        {/* Desktop (sm+): Show Archive button directly */}
        <div className="hidden sm:block">
          <ActionButton icon={isArchived ? ArchiveRestore : Archive} label={isArchived ? "Restore" : "Archive"} onClick={onToggleArchive} />
        </div>

        {/* Desktop (sm+): Show Delete button directly */}
        <div className="hidden sm:block">
          <ActionButton
            icon={Trash2}
            label="Delete"
            onClick={onDelete}
            className="text-red-600 hover:text-red-700 dark:text-red-400 dark:hover:text-red-300"
          />
        </div>

        {/* Mobile only: More menu dropdown */}
        <div className="sm:hidden relative">
          <ActionButton
            icon={Ellipsis}
            label="More actions"
            onClick={onQuickMenuToggle}
            isActive={quickMenuOpen}
            className={quickMenuOpen ? "bg-zinc-200 dark:bg-zinc-700" : undefined}
            buttonRef={dropdownButtonRef}
          />
        </div>
      </div>

      {/* Mobile only: Dropdown menu - rendered via Portal to avoid overflow clipping */}
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
              left: "auto",
              right: `${dropdownPosition.right}px`,
            }}
          >
            {/* Header with close button */}
            <div className="flex items-center justify-between px-3 py-2 border-b border-zinc-100 dark:border-zinc-800">
              <span className="text-xs font-medium text-zinc-500 dark:text-zinc-400">Actions</span>
              <button
                onClick={() => onQuickMenuToggle()}
                className="p-1 rounded hover:bg-zinc-100 dark:hover:bg-zinc-800 transition-colors"
                aria-label="Close menu"
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

interface ActionButtonProps {
  icon: typeof Edit3;
  label: string;
  onClick: () => void;
  className?: string;
  isActive?: boolean;
  buttonRef?: React.RefObject<HTMLButtonElement>;
}

function ActionButton({ icon: Icon, label, onClick, className, isActive, buttonRef }: ActionButtonProps) {
  return (
    <button
      ref={buttonRef}
      onClick={onClick}
      className={cn(
        "p-2 rounded-lg transition-all duration-150",
        "text-zinc-500 hover:text-zinc-700 dark:text-zinc-400 dark:hover:text-zinc-200",
        "hover:bg-zinc-100 dark:hover:bg-zinc-800",
        "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-violet-500/50",
        "active:scale-95",
        isActive && "bg-zinc-200 dark:bg-zinc-800",
        className,
      )}
      aria-label={label}
      title={label}
    >
      <Icon className="w-4 h-4" />
    </button>
  );
}
