/**
 * EditorToolbar - 禅意工具栏 (ZenToolbar)
 *
 * 设计哲学：「禅意智识」
 * - 不折叠：所有功能直观呈现
 * - 呼吸感：与 logo 同步的 3000ms 呼吸周期
 * - 留白：4px 基础间距单位
 * - 响应式：PC 水平布局 / 移动两行布局
 *
 * ## 设计规范
 * - 按钮尺寸：PC 36×36px / 移动 40×40px
 * - 圆角：统一 rounded-xl (12px)
 * - 动画：300ms ease-out
 */

import React, { memo, useCallback, useEffect, useState } from "react";
import { useDebounce } from "react-use";
import { useReverseGeocoding } from "@/components/map";
import { Button } from "@/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import VisibilityIcon from "@/components/VisibilityIcon";
import { cn } from "@/lib/utils";
import type { MemoRelation } from "@/types/proto/api/v1/memo_service_pb";
import { Visibility } from "@/types/proto/api/v1/memo_service_pb";
import { useTranslate } from "@/utils/i18n";
import { useFileUpload, useLinkMemo, useLocation } from "../hooks";
import { validationService } from "../services";
import { useEditorContext } from "../state";
import type { EditorToolbarProps } from "../types";
import type { LocalFile } from "../types/attachment";
import { LinkMemoDialog } from ".";

// Lazy load LocationDialog
const LocationDialog = React.lazy(() => import(".").then((module) => ({ default: module.LocationDialog })));

// ============================================================================
// 常量
// ============================================================================

const BREATH_DURATION = 3000; // 与 logo-breathe-gentle 同步

// ============================================================================
// 子组件
// ============================================================================

interface ToolButtonProps {
  icon: React.ComponentType<{ className?: string }>;
  label: string;
  shortcut?: string;
  isActive?: boolean;
  isLoading?: boolean;
  isDisabled?: boolean;
  tooltipSide?: "top" | "bottom" | "left" | "right";
  onClick: () => void;
  className?: string;
  children?: React.ReactNode;
}

const ToolButton = memo(function ToolButton({
  icon: Icon,
  label,
  shortcut,
  isActive = false,
  isLoading = false,
  isDisabled = false,
  tooltipSide = "top",
  onClick,
  className,
  children,
}: ToolButtonProps) {
  const [isHovered, setIsHovered] = useState(false);

  return (
    <Tooltip delayDuration={300}>
      <TooltipTrigger asChild>
        <button
          type="button"
          onClick={onClick}
          disabled={isDisabled || isLoading}
          onMouseEnter={() => setIsHovered(true)}
          onMouseLeave={() => setIsHovered(false)}
          className={cn(
            "relative group",
            // 响应式尺寸：移动端稍大
            "h-9 w-9 sm:h-10 sm:w-10",
            "rounded-xl",
            "transition-all duration-300 ease-out",
            // 状态
            isActive ? "bg-primary/10 text-primary shadow-inner" : "text-muted-foreground hover:text-foreground hover:bg-muted/50",
            isDisabled && "opacity-40 cursor-not-allowed",
            "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/50",
            className,
          )}
          aria-label={label}
        >
          {/* 呼吸光晕 */}
          {(isHovered || isActive) && !isDisabled && (
            <span
              className="absolute inset-0 rounded-xl bg-primary/10 animate-pulse"
              style={{ animationDuration: `${BREATH_DURATION}ms` }}
            />
          )}

          {/* 加载中 */}
          {isLoading ? (
            <span className="h-4 w-4 animate-spin rounded-full border-2 border-current border-t-transparent" />
          ) : (
            <Icon className="h-4 w-4 relative z-10" />
          )}

          {children}
        </button>
      </TooltipTrigger>
      <TooltipContent side={tooltipSide} className="flex items-center gap-2">
        <span>{label}</span>
        {shortcut && <kbd className="rounded-md bg-muted/50 px-1.5 py-0.5 text-[10px] font-mono text-muted-foreground">{shortcut}</kbd>}
      </TooltipContent>
    </Tooltip>
  );
});

interface VisibilitySelectorProps {
  value: Visibility;
  onChange: (value: Visibility) => void;
}

const VISIBILITY_OPTIONS = [
  { value: Visibility.PRIVATE, label: "memo.visibility.private" },
  { value: Visibility.PROTECTED, label: "memo.visibility.protected" },
  { value: Visibility.PUBLIC, label: "memo.visibility.public" },
] as const;

const VisibilitySelector = memo(function VisibilitySelector({ value, onChange }: VisibilitySelectorProps) {
  const t = useTranslate();
  const [isExpanded, setIsExpanded] = useState(false);

  // 获取当前可见性文本
  const getCurrentLabel = () => {
    if (value === Visibility.PRIVATE) return t("memo.visibility.private");
    if (value === Visibility.PROTECTED) return t("memo.visibility.protected");
    return t("memo.visibility.public");
  };

  return (
    <div className="relative" onMouseEnter={() => setIsExpanded(true)} onMouseLeave={() => setIsExpanded(false)}>
      {/* 展开面板 */}
      {isExpanded && (
        <div
          className={cn(
            "absolute bottom-full left-1/2 -translate-x-1/2 mb-2",
            "flex items-center gap-1 p-1",
            "rounded-xl bg-background border border-border/50 shadow-lg",
            "animate-in fade-in slide-in-from-bottom-1 duration-200 z-50",
          )}
        >
          {VISIBILITY_OPTIONS.map((option) => {
            const isSelected = value === option.value;
            return (
              <Tooltip key={option.value} delayDuration={200}>
                <TooltipTrigger asChild>
                  <button
                    type="button"
                    onClick={() => onChange(option.value)}
                    className={cn(
                      "h-9 w-9 rounded-lg flex items-center justify-center",
                      "transition-all duration-200",
                      isSelected ? "bg-primary text-primary-foreground" : "text-muted-foreground hover:text-foreground hover:bg-muted/50",
                    )}
                  >
                    <VisibilityIcon visibility={option.value} className="h-4 w-4" />
                  </button>
                </TooltipTrigger>
                <TooltipContent side="top">{t(option.label)}</TooltipContent>
              </Tooltip>
            );
          })}
        </div>
      )}

      {/* 当前选择按钮 */}
      <Tooltip delayDuration={300}>
        <TooltipTrigger asChild>
          <button
            type="button"
            onClick={() => {
              const currentIndex = VISIBILITY_OPTIONS.findIndex((opt) => opt.value === value);
              const nextIndex = (currentIndex + 1) % VISIBILITY_OPTIONS.length;
              onChange(VISIBILITY_OPTIONS[nextIndex].value);
            }}
            className={cn(
              "h-9 sm:h-10 w-9 sm:w-10",
              "rounded-xl flex items-center justify-center",
              "text-muted-foreground hover:text-foreground hover:bg-muted/50",
              "transition-all duration-300",
            )}
          >
            <VisibilityIcon visibility={value} className="h-4 w-4" />
          </button>
        </TooltipTrigger>
        <TooltipContent side="top">
          <div className="flex items-center gap-2">
            <span>{getCurrentLabel()}</span>
            <span className="text-muted-foreground text-xs">→ 点击切换</span>
          </div>
        </TooltipContent>
      </Tooltip>
    </div>
  );
});

// ============================================================================
// 主组件
// ============================================================================

export const EditorToolbar: React.FC<EditorToolbarProps> = ({ onSave, onCancel, memoName }) => {
  const t = useTranslate();
  const { state, actions, dispatch } = useEditorContext();
  const { valid } = validationService.canSave(state);

  const isSaving = state.ui.isLoading.saving;
  const [linkDialogOpen, setLinkDialogOpen] = useState(false);
  const [locationDialogOpen, setLocationDialogOpen] = useState(false);

  // 文件上传
  const { fileInputRef, selectingFlag, handleFileInputChange, handleUploadClick } = useFileUpload((newFiles: LocalFile[]) => {
    newFiles.forEach((file) => dispatch(actions.addLocalFile(file)));
  });

  // 链接笔记
  const linkMemo = useLinkMemo({
    isOpen: linkDialogOpen,
    currentMemoName: memoName,
    existingRelations: state.metadata.relations,
    onAddRelation: (relation: MemoRelation) => {
      const relations = state.metadata.relations.some((r) => r.relatedMemo?.name === relation.relatedMemo?.name)
        ? state.metadata.relations
        : [...state.metadata.relations, relation];
      dispatch(actions.setMetadata({ relations }));
      setLinkDialogOpen(false);
    },
  });

  // 位置
  const location = useLocation(state.metadata.location);
  const [debouncedPosition, setDebouncedPosition] = useState<{ lat: number; lng: number } | undefined>(undefined);

  useDebounce(
    () => {
      setDebouncedPosition(location.state.position);
    },
    1000,
    [location.state.position],
  );

  const { data: displayName } = useReverseGeocoding(debouncedPosition?.lat, debouncedPosition?.lng);

  useEffect(() => {
    if (displayName) {
      location.setPlaceholder(displayName);
    }
  }, [displayName, location]);

  const isUploading = selectingFlag || state.ui.isLoading.uploading;

  // 处理函数
  const handleLocationClick = useCallback(() => {
    setLocationDialogOpen(true);
    if (!state.metadata.location && !location.locationInitialized) {
      if (navigator.geolocation) {
        navigator.geolocation.getCurrentPosition(
          (position) => {
            location.handlePositionChange({
              lat: position.coords.latitude,
              lng: position.coords.longitude,
            });
          },
          (error) => {
            console.error("Geolocation error:", error);
          },
        );
      }
    }
  }, [state.metadata.location, location, location.locationInitialized]);

  const handleLocationConfirm = useCallback(() => {
    const newLocation = location.getLocation();
    if (newLocation) {
      dispatch(actions.setMetadata({ location: newLocation }));
      setLocationDialogOpen(false);
    }
  }, [location, dispatch]);

  const handleLocationCancel = useCallback(() => {
    location.reset();
    setLocationDialogOpen(false);
  }, [location]);

  const handlePositionChange = useCallback(
    (position: { lat: number; lng: number }) => {
      location.handlePositionChange(position);
    },
    [location],
  );

  const handleToggleFocusMode = useCallback(() => {
    dispatch(actions.toggleFocusMode());
  }, [dispatch]);

  const handleVisibilityChange = useCallback(
    (visibility: Visibility) => {
      dispatch(actions.setMetadata({ visibility }));
    },
    [dispatch],
  );

  return (
    <>
      {/* 桌面端工具栏 */}
      <div className="hidden sm:flex items-center justify-between gap-3">
        {/* 左侧：插入功能 */}
        <div className="flex items-center gap-1">
          <ToolButton
            icon={({ className }) => (
              <svg
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                strokeWidth="2"
                strokeLinecap="round"
                strokeLinejoin="round"
                className={className}
              >
                <rect x="3" y="3" width="18" height="18" rx="2" ry="2" />
                <circle cx="8.5" cy="8.5" r="1.5" />
                <polyline points="21 15 16 10 5 21" />
              </svg>
            )}
            label={t("common.upload")}
            isLoading={isUploading}
            onClick={handleUploadClick}
          />

          <ToolButton
            icon={({ className }) => (
              <svg
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                strokeWidth="2"
                strokeLinecap="round"
                strokeLinejoin="round"
                className={className}
              >
                <path d="M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71" />
                <path d="M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71" />
              </svg>
            )}
            label={t("tooltip.link-memo")}
            onClick={() => setLinkDialogOpen(true)}
          />

          <ToolButton
            icon={({ className }) => (
              <svg
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                strokeWidth="2"
                strokeLinecap="round"
                strokeLinejoin="round"
                className={className}
              >
                <path d="M21 10c0 7-9 13-9 13s-9-6-9-13a9 9 0 0 1 18 0z" />
                <circle cx="12" cy="10" r="3" />
              </svg>
            )}
            label={t("tooltip.select-location")}
            isActive={!!state.metadata.location}
            onClick={handleLocationClick}
          />

          <ToolButton
            icon={({ className }) => (
              <svg
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                strokeWidth="2"
                strokeLinecap="round"
                strokeLinejoin="round"
                className={className}
              >
                <polyline points="15 3 21 3 21 9" />
                <polyline points="9 21 3 21 3 15" />
                <line x1="21" x2="14" y1="3" y2="10" />
                <line x1="3" x2="10" y1="21" y2="14" />
              </svg>
            )}
            label={t("editor.focus-mode")}
            shortcut="⌘⇧F"
            onClick={handleToggleFocusMode}
          />
        </div>

        {/* 右侧：可见性 + 操作 */}
        <div className="flex items-center gap-1">
          <VisibilitySelector value={state.metadata.visibility} onChange={handleVisibilityChange} />

          {onCancel && (
            <Button
              variant="ghost"
              size="sm"
              onClick={onCancel}
              disabled={isSaving}
              className="h-9 px-3 text-muted-foreground hover:text-foreground"
            >
              {t("common.cancel")}
            </Button>
          )}

          <Button onClick={onSave} disabled={!valid || isSaving} className="h-9 px-4">
            {isSaving ? (
              <span className="flex items-center gap-2">
                <span className="h-4 w-4 animate-spin rounded-full border-2 border-current border-t-transparent" />
                {t("common.saving")}
              </span>
            ) : (
              <span className="flex items-center gap-2">
                {t("editor.save")}
                <svg
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  strokeWidth="2"
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  className="h-3.5 w-3.5"
                >
                  <line x1="22" y1="2" x2="11" y2="13" />
                  <polygon points="22 2 15 22 11 13 11 22" />
                </svg>
              </span>
            )}
          </Button>
        </div>
      </div>

      {/* 移动端工具栏 */}
      <div className="sm:hidden flex flex-col gap-2">
        {/* 第一行：插入功能 */}
        <div className="flex items-center justify-between gap-2">
          <div className="flex items-center gap-1">
            <ToolButton
              icon={({ className }) => (
                <svg
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  strokeWidth="2"
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  className={className}
                >
                  <rect x="3" y="3" width="18" height="18" rx="2" ry="2" />
                  <circle cx="8.5" cy="8.5" r="1.5" />
                  <polyline points="21 15 16 10 5 21" />
                </svg>
              )}
              label={t("common.upload")}
              isLoading={isUploading}
              onClick={handleUploadClick}
            />

            <ToolButton
              icon={({ className }) => (
                <svg
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  strokeWidth="2"
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  className={className}
                >
                  <path d="M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71" />
                  <path d="M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71" />
                </svg>
              )}
              label={t("tooltip.link-memo")}
              onClick={() => setLinkDialogOpen(true)}
            />

            <ToolButton
              icon={({ className }) => (
                <svg
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  strokeWidth="2"
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  className={className}
                >
                  <path d="M21 10c0 7-9 13-9 13s-9-6-9-13a9 9 0 0 1 18 0z" />
                  <circle cx="12" cy="10" r="3" />
                </svg>
              )}
              label={t("tooltip.select-location")}
              isActive={!!state.metadata.location}
              onClick={handleLocationClick}
            />

            <ToolButton
              icon={({ className }) => (
                <svg
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  strokeWidth="2"
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  className={className}
                >
                  <polyline points="15 3 21 3 21 9" />
                  <polyline points="9 21 3 21 3 15" />
                  <line x1="21" x2="14" y1="3" y2="10" />
                  <line x1="3" x2="10" y1="21" y2="14" />
                </svg>
              )}
              label={t("editor.focus-mode")}
              onClick={handleToggleFocusMode}
            />
          </div>

          <VisibilitySelector value={state.metadata.visibility} onChange={handleVisibilityChange} />
        </div>

        {/* 第二行：操作按钮 */}
        <div className="flex items-center justify-end gap-2">
          {onCancel && (
            <Button
              variant="ghost"
              size="sm"
              onClick={onCancel}
              disabled={isSaving}
              className="h-10 px-4 text-muted-foreground hover:text-foreground"
            >
              {t("common.cancel")}
            </Button>
          )}

          <Button onClick={onSave} disabled={!valid || isSaving} className="h-10 px-4">
            {isSaving ? (
              <span className="flex items-center gap-2">
                <span className="h-4 w-4 animate-spin rounded-full border-2 border-current border-t-transparent" />
                {t("common.saving")}
              </span>
            ) : (
              <span className="flex items-center gap-2">
                {t("editor.save")}
                <svg
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  strokeWidth="2"
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  className="h-4 w-4"
                >
                  <line x1="22" y1="2" x2="11" y2="13" />
                  <polygon points="22 2 15 22 11 13 11 22" />
                </svg>
              </span>
            )}
          </Button>
        </div>
      </div>

      {/* 隐藏的文件输入 */}
      <input
        className="hidden"
        ref={fileInputRef}
        disabled={isUploading}
        onChange={handleFileInputChange}
        type="file"
        multiple
        accept="*"
      />

      {/* 链接笔记对话框 */}
      <LinkMemoDialog
        open={linkDialogOpen}
        onOpenChange={setLinkDialogOpen}
        searchText={linkMemo.searchText}
        onSearchChange={linkMemo.setSearchText}
        filteredMemos={linkMemo.filteredMemos}
        isFetching={linkMemo.isFetching}
        onSelectMemo={linkMemo.addMemoRelation}
      />

      {/* 位置选择对话框 */}
      {locationDialogOpen && (
        <React.Suspense fallback={null}>
          <LocationDialog
            open={locationDialogOpen}
            onOpenChange={setLocationDialogOpen}
            state={location.state}
            locationInitialized={location.locationInitialized}
            onPositionChange={handlePositionChange}
            onUpdateCoordinate={location.updateCoordinate}
            onPlaceholderChange={location.setPlaceholder}
            onCancel={handleLocationCancel}
            onConfirm={handleLocationConfirm}
          />
        </React.Suspense>
      )}
    </>
  );
};
