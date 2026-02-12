import { Globe, Link2, Loader2, Lock, type LucideIcon, MapPin, Maximize2, Paperclip, Shield, Sparkles } from "lucide-react";
import type { FC } from "react";
import { forwardRef, lazy, Suspense, useCallback, useEffect, useMemo, useState } from "react";
import { useDebounce } from "react-use";
import { useReverseGeocoding } from "@/components/map";
import { Button } from "@/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import useMediaQuery from "@/hooks/useMediaQuery";
import { cn } from "@/lib/utils";
import type { MemoRelation } from "@/types/proto/api/v1/memo_service_pb";
import { Visibility } from "@/types/proto/api/v1/memo_service_pb";
import { useTranslate } from "@/utils/i18n";
import { TOOLBAR_BUTTON_STYLES } from "../constants";
import { useFileUpload, useLinkMemo, useLocation } from "../hooks";
import { useEditorContext } from "../state";
import type { EditorToolbarProps } from "../types";
import type { LocalFile } from "../types/attachment";
import { AIFormatButton } from "./AIFormatButton";
import { AITagButton } from "./AITagButton";
import { LinkMemoDialog } from "./LinkMemoDialog";
import { VisibilityToggleGroup } from "./VisibilityToggleGroup";

// Lazy load LocationDialog
const LocationDialog = lazy(() => import("./LocationDialog").then((module) => ({ default: module.LocationDialog })));

// ============================================================================
// Constants
// ============================================================================

const BREATH_DURATION = 3000; // 与 logo-breathe-gentle 同步

const VISIBILITY_CYCLE: Visibility[] = [Visibility.PRIVATE, Visibility.PROTECTED, Visibility.PUBLIC];

// ============================================================================
// VisibilityCycleButton - 移动端可见性循环切换按钮
// ============================================================================

interface VisibilityCycleButtonProps {
  value: Visibility;
  onChange: (value: Visibility) => void;
}

const VisibilityCycleButton: FC<VisibilityCycleButtonProps> = ({ value, onChange }) => {
  const t = useTranslate();

  const { icon: Icon, label } = useMemo(() => {
    switch (value) {
      case Visibility.PRIVATE:
        return { icon: Lock, label: t("memo.visibility.private") };
      case Visibility.PROTECTED:
        return { icon: Shield, label: t("memo.visibility.protected") };
      case Visibility.PUBLIC:
        return { icon: Globe, label: t("memo.visibility.public") };
      default:
        return { icon: Lock, label: t("memo.visibility.private") };
    }
  }, [value, t]);

  const cycleVisibility = () => {
    const currentIndex = VISIBILITY_CYCLE.indexOf(value);
    const nextIndex = (currentIndex + 1) % VISIBILITY_CYCLE.length;
    onChange(VISIBILITY_CYCLE[nextIndex]);
  };

  return (
    <Tooltip delayDuration={300}>
      <TooltipTrigger asChild>
        <button
          type="button"
          onClick={cycleVisibility}
          className={cn(
            "h-9 w-9 rounded-xl flex items-center justify-center",
            "text-muted-foreground hover:text-foreground",
            "hover:bg-muted/50 active:scale-95",
            "transition-all duration-200",
            "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/30",
          )}
          aria-label={label}
        >
          <Icon className="w-4 h-4" />
        </button>
      </TooltipTrigger>
      <TooltipContent side="top">
        <p>{label}</p>
      </TooltipContent>
    </Tooltip>
  );
};

// ============================================================================
// EditorToolbar Component
// ============================================================================

/**
 * EditorToolbar - 禅意智识风格的编辑器工具栏
 *
 * 设计基因来自 HeroSection：
 * - 统一圆角：rounded-xl
 * - 微交互：hover:scale-105 active:scale-95
 * - 呼吸感：subtle transitions
 * - 视觉层次：分组清晰的工具栏
 * - 响应式：
 *   - PC: [📎][📍][🔗] | [✨AI标签][🪄格式化] | Spacer | [🔒][👥][🌐] | [⛶] | [Save]
 *   - Mobile: [📎][📍][🔗] | [✨AI][🪄] | [🔒] | [Save]
 */
export const EditorToolbar: FC<EditorToolbarProps> = ({
  onSave,
  onCancel,
  onLinkMemo,
  onToggleFocusMode,
  onVisibilityChange,
  onInsertTags,
  onFormatContent,
  memoName,
}) => {
  const t = useTranslate();
  const lg = useMediaQuery("lg");
  const { state, actions, dispatch } = useEditorContext();

  const hasCancel = !!onCancel;
  const hasContent = state.content.trim().length > 0;
  const isSaving = state.ui.isLoading.saving;

  // Location dialog state
  const [locationDialogOpen, setLocationDialogOpen] = useState(false);

  // Link memo dialog state
  const [linkMemoDialogOpen, setLinkMemoDialogOpen] = useState(false);

  // File upload hook
  const { fileInputRef, selectingFlag, handleFileInputChange, handleUploadClick } = useFileUpload((newFiles: LocalFile[]) => {
    newFiles.forEach((file) => dispatch(actions.addLocalFile(file)));
  });

  // Location hook
  const location = useLocation(state.metadata.location);
  const [debouncedPosition, setDebouncedPosition] = useState<{ lat: number; lng: number } | undefined>(undefined);

  useDebounce(
    () => {
      setDebouncedPosition(location.state.position);
    },
    1000,
    [location.state.position],
  );

  // Reverse geocoding for location display name
  const { data: displayName } = useReverseGeocoding(debouncedPosition?.lat, debouncedPosition?.lng);

  useEffect(() => {
    if (displayName) {
      location.setPlaceholder(displayName);
    }
  }, [displayName, location]);

  // Link memo hook
  const linkMemo = useLinkMemo({
    isOpen: linkMemoDialogOpen,
    currentMemoName: memoName,
    existingRelations: state.metadata.relations,
    onAddRelation: (relation: MemoRelation) => {
      dispatch(actions.addRelation(relation));
      setLinkMemoDialogOpen(false);
    },
  });

  const isUploading = selectingFlag || state.ui.isLoading.uploading;

  const handleSave = () => {
    onSave?.();
  };

  // Location handlers
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
  }, [state.metadata.location, location]);

  const handleLocationConfirm = useCallback(() => {
    const newLocation = location.getLocation();
    if (newLocation) {
      dispatch(actions.setMetadata({ location: newLocation }));
      setLocationDialogOpen(false);
    }
  }, [location, dispatch, actions]);

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

  return (
    <>
      <div className="w-full flex items-center justify-between gap-2 sm:gap-3 px-4 sm:px-5 py-3 border-t border-border/40 bg-muted/20 backdrop-blur-sm">
        {/* Left: Tool buttons group + AI tags */}
        <div className="flex items-center gap-1.5 sm:gap-2">
          {/* Upload attachment - always visible */}
          <ToolbarButton
            icon={Paperclip}
            ariaLabel={t("editor.add-attachment")}
            tooltip={t("editor.add-attachment")}
            onClick={handleUploadClick}
            isLoading={isUploading}
          />

          {/* Location button - now visible on both desktop and mobile */}
          <ToolbarButton
            icon={MapPin}
            ariaLabel={t("editor.add-location")}
            tooltip={t("editor.add-location")}
            onClick={handleLocationClick}
            isActive={!!state.metadata.location}
          />

          {/* Link memo button - both desktop and mobile */}
          {onLinkMemo && (
            <ToolbarButton
              icon={Link2}
              ariaLabel={t("editor.link-memo")}
              tooltip={t("editor.link-memo")}
              onClick={() => setLinkMemoDialogOpen(true)}
            />
          )}

          {/* Divider */}
          {(onInsertTags || onFormatContent) && <div className="w-px h-5 bg-border/50 mx-1" />}

          {/* AI Tag button - both desktop and mobile */}
          {onInsertTags && <AITagButton content={state.content} onInsertTags={onInsertTags} compact={!lg} />}

          {/* AI Format button - both desktop and mobile */}
          {onFormatContent && <AIFormatButton content={state.content} onFormat={onFormatContent} compact={!lg} />}
        </div>

        {/* Right: Settings and action buttons */}
        <div className="flex items-center gap-1.5 sm:gap-2">
          {/* Visibility: PC = button group, Mobile = cycle button */}
          {onVisibilityChange &&
            (lg ? (
              <VisibilityToggleGroup value={state.metadata.visibility} onChange={onVisibilityChange} />
            ) : (
              <VisibilityCycleButton value={state.metadata.visibility} onChange={onVisibilityChange} />
            ))}

          {/* Focus mode button - Desktop only */}
          {lg && onToggleFocusMode && (
            <ToolbarButton
              icon={Maximize2}
              ariaLabel={t("editor.focus-mode")}
              tooltip={t("editor.focus-mode")}
              onClick={onToggleFocusMode}
            />
          )}

          {/* Save button - styled with breathing effect when has content */}
          <button
            type="button"
            onClick={handleSave}
            disabled={!hasContent || isSaving}
            className={cn(
              "shrink-0 h-9 px-4 min-w-[64px] rounded-xl transition-all duration-300",
              "hover:scale-105 active:scale-95",
              "text-sm font-medium",
              isSaving
                ? "bg-muted text-muted-foreground"
                : hasContent
                  ? "bg-primary text-primary-foreground shadow-md shadow-primary/20 hover:bg-primary/90"
                  : "bg-muted text-muted-foreground hover:bg-muted/80",
              "disabled:opacity-50 disabled:cursor-not-allowed disabled:hover:scale-100",
            )}
            aria-label={t("editor.save")}
          >
            {isSaving ? (
              <Sparkles className="w-4 h-4 opacity-50 animate-pulse mx-auto" />
            ) : (
              <span>{memoName ? t("common.update") : t("editor.save")}</span>
            )}
          </button>

          {/* Cancel button - when cancel callback exists */}
          {hasCancel && (
            <Button
              variant="ghost"
              size="sm"
              className="h-9 px-4 rounded-xl hover:bg-accent/60 active:scale-95 transition-all duration-200 text-sm font-medium"
              onClick={onCancel}
            >
              {t("common.cancel")}
            </Button>
          )}
        </div>
      </div>

      {/* Hidden file input */}
      <input
        className="hidden"
        ref={fileInputRef}
        disabled={isUploading}
        onChange={handleFileInputChange}
        type="file"
        multiple
        accept="*"
      />

      {/* Link Memo Dialog */}
      <LinkMemoDialog
        open={linkMemoDialogOpen}
        onOpenChange={setLinkMemoDialogOpen}
        searchText={linkMemo.searchText}
        onSearchChange={linkMemo.setSearchText}
        filteredMemos={linkMemo.filteredMemos}
        isFetching={linkMemo.isFetching}
        onSelectMemo={linkMemo.addMemoRelation}
      />

      {/* Location Dialog */}
      {locationDialogOpen && (
        <Suspense fallback={null}>
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
        </Suspense>
      )}
    </>
  );
};

// ============================================================================
// ToolbarButton - Enhanced tool button with micro-interactions and Tooltip
// ============================================================================

interface ToolbarButtonProps {
  icon: LucideIcon;
  ariaLabel: string;
  tooltip?: string;
  onClick?: () => void;
  className?: string;
  isActive?: boolean;
  isLoading?: boolean;
}

const ToolbarButton = forwardRef<HTMLButtonElement, ToolbarButtonProps>(
  ({ icon: Icon, ariaLabel, tooltip, onClick, className, isActive = false, isLoading = false }, ref) => {
    const [isHovered, setIsHovered] = useState(false);

    const button = (
      <button
        ref={ref}
        type="button"
        onClick={onClick}
        onMouseEnter={() => setIsHovered(true)}
        onMouseLeave={() => setIsHovered(false)}
        disabled={isLoading}
        className={cn(
          TOOLBAR_BUTTON_STYLES.base,
          TOOLBAR_BUTTON_STYLES.ghost,
          "relative group",
          "h-9 w-9 rounded-xl",
          "flex items-center justify-center",
          isActive ? "bg-primary/10 text-primary" : "text-muted-foreground hover:text-foreground",
          isLoading && "opacity-50 cursor-not-allowed",
          "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/30",
          "transition-all duration-300 ease-out",
          className,
        )}
        aria-label={ariaLabel}
      >
        {/* 呼吸光晕 - hover 时显示 */}
        {(isHovered || isActive) && !isLoading && (
          <span className="absolute inset-0 rounded-xl bg-primary/10 animate-pulse" style={{ animationDuration: `${BREATH_DURATION}ms` }} />
        )}
        {isLoading ? <Loader2 className="w-4 h-4 animate-spin" /> : <Icon className="w-4 h-4 relative z-10" />}
      </button>
    );

    if (tooltip) {
      return (
        <Tooltip delayDuration={300}>
          <TooltipTrigger asChild>{button}</TooltipTrigger>
          <TooltipContent side="top">
            <p>{tooltip}</p>
          </TooltipContent>
        </Tooltip>
      );
    }

    return button;
  },
);

ToolbarButton.displayName = "ToolbarButton";

EditorToolbar.displayName = "EditorToolbar";
