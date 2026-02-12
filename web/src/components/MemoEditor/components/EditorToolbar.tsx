import { Globe, Link2, Lock, type LucideIcon, MapPin, Maximize2, MoreHorizontal, Paperclip, Plus, Shield, Sparkles } from "lucide-react";
import type { FC } from "react";
import { forwardRef, memo, useState } from "react";
import { Button } from "@/components/ui/button";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from "@/components/ui/dropdown-menu";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import VisibilityIcon from "@/components/VisibilityIcon";
import useMediaQuery from "@/hooks/useMediaQuery";
import { cn } from "@/lib/utils";
import { Visibility } from "@/types/proto/api/v1/memo_service_pb";
import { useTranslate } from "@/utils/i18n";
import { TOOLBAR_BUTTON_STYLES } from "../constants";
import { useEditorContext } from "../state";
import type { EditorToolbarProps } from "../types";

// ============================================================================
// Constants
// ============================================================================

const BREATH_DURATION = 3000; // 与 logo-breathe-gentle 同步

const VISIBILITY_OPTIONS = [
  { value: Visibility.PRIVATE, label: "memo.visibility.private" },
  { value: Visibility.PROTECTED, label: "memo.visibility.protected" },
  { value: Visibility.PUBLIC, label: "memo.visibility.public" },
] as const;

// ============================================================================
// VisibilitySelector Component
// ============================================================================

interface VisibilitySelectorProps {
  value: Visibility;
  onChange: (value: Visibility) => void;
  isDesktop?: boolean;
}

const VisibilitySelector = memo(function VisibilitySelector({ value, onChange, isDesktop }: VisibilitySelectorProps) {
  const t = useTranslate();

  const getCurrentLabel = () => {
    if (value === Visibility.PRIVATE) return t("memo.visibility.private");
    if (value === Visibility.PROTECTED) return t("memo.visibility.protected");
    return t("memo.visibility.public");
  };

  const cycleVisibility = () => {
    const currentIndex = VISIBILITY_OPTIONS.findIndex((opt) => opt.value === value);
    const nextIndex = (currentIndex + 1) % VISIBILITY_OPTIONS.length;
    onChange(VISIBILITY_OPTIONS[nextIndex].value);
  };

  // Desktop: Text button with cycle
  if (isDesktop) {
    return (
      <Tooltip>
        <TooltipTrigger asChild>
          <Button
            type="button"
            variant="ghost"
            size="sm"
            onClick={cycleVisibility}
            className="h-9 px-2.5 rounded-xl text-xs text-muted-foreground hover:text-foreground hover:bg-accent/60 active:scale-95 transition-all duration-200"
            aria-label="Visibility"
          >
            <VisibilityIcon visibility={value} className="w-4 h-4 mr-1" />
            {getCurrentLabel()}
          </Button>
        </TooltipTrigger>
        <TooltipContent side="top">
          <p>点击切换可见性</p>
        </TooltipContent>
      </Tooltip>
    );
  }

  // Mobile: Icon button with dropdown menu
  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <ToolbarButton
          icon={value === Visibility.PRIVATE ? Lock : value === Visibility.PROTECTED ? Shield : Globe}
          ariaLabel={getCurrentLabel()}
        />
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="w-40">
        {VISIBILITY_OPTIONS.map((option) => (
          <DropdownMenuItem key={option.value} onClick={() => onChange(option.value)} className={cn(value === option.value && "bg-accent")}>
            <VisibilityIcon visibility={option.value} className="w-4 h-4 mr-2" />
            {t(option.label)}
          </DropdownMenuItem>
        ))}
      </DropdownMenuContent>
    </DropdownMenu>
  );
});

/**
 * EditorToolbar - 禅意智识风格的编辑器工具栏
 *
 * 设计基因来自 HeroSection：
 * - 统一圆角：rounded-xl
 * - 微交互：hover:scale-105 active:scale-95
 * - 呼吸感：subtle transitions
 * - 视觉层次：分组清晰的工具栏
 * - 响应式：移动端显示附件+关联，其他收纳到更多菜单
 */
export const EditorToolbar: FC<EditorToolbarProps> = ({
  onSave,
  onCancel,
  onUploadAttachment,
  onLinkMemo,
  onToggleFocusMode,
  onVisibilityChange,
  onOpenMobileTools,
  memoName,
}) => {
  const t = useTranslate();
  const md = useMediaQuery("md");
  const [moreMenuOpen, setMoreMenuOpen] = useState(false);
  const { state } = useEditorContext();

  const hasCancel = !!onCancel;
  const hasContent = state.content.trim().length > 0;
  const isSaving = state.ui.isLoading.saving;

  const handleSave = () => {
    onSave?.();
  };

  return (
    <div className="w-full flex items-center justify-between gap-2 sm:gap-3 px-4 sm:px-5 py-3 border-t border-border/40 bg-muted/20 backdrop-blur-sm">
      {/* Left: Tool buttons group */}
      <div className="flex items-center gap-1.5 sm:gap-2">
        {/* Upload attachment - always visible */}
        <ToolbarButton
          icon={Paperclip}
          ariaLabel={t("editor.add-attachment")}
          tooltip={t("editor.add-attachment")}
          onClick={onUploadAttachment}
        />

        {/* Desktop: Location button */}
        {md && (
          <ToolbarButton
            icon={MapPin}
            ariaLabel={t("editor.add-location")}
            tooltip={t("editor.add-location")}
            onClick={() => {
              // TODO: Implement location picker
              console.log("Add location");
            }}
          />
        )}

        {/* Mobile: Open tools sheet button */}
        {!md && onOpenMobileTools && <ToolbarButton icon={Plus} ariaLabel="Tools" tooltip="Tools" onClick={onOpenMobileTools} />}

        {/* Link memo button - both desktop and mobile */}
        {onLinkMemo && (
          <ToolbarButton icon={Link2} ariaLabel={t("editor.link-memo")} tooltip={t("editor.link-memo")} onClick={onLinkMemo} />
        )}
      </div>

      {/* Right: Settings and action buttons */}
      <div className="flex items-center gap-1.5 sm:gap-2">
        {/* Visibility selector */}
        {onVisibilityChange && <VisibilitySelector value={state.metadata.visibility} onChange={onVisibilityChange} isDesktop={md} />}

        {/* Focus mode button */}
        {onToggleFocusMode && (
          <ToolbarButton icon={Maximize2} ariaLabel={t("editor.focus-mode")} tooltip={t("editor.focus-mode")} onClick={onToggleFocusMode} />
        )}

        {/* Mobile: More menu with additional tools */}
        {!md && (
          <DropdownMenu open={moreMenuOpen} onOpenChange={setMoreMenuOpen}>
            <DropdownMenuTrigger asChild>
              <ToolbarButton icon={MoreHorizontal} ariaLabel="More tools" tooltip="More tools" />
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end" className="w-48">
              <DropdownMenuItem onClick={onToggleFocusMode}>
                <Maximize2 className="w-4 h-4 mr-2" />
                {t("editor.focus-mode")}
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
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
  );
};

/**
 * ToolbarButton - Enhanced tool button with micro-interactions and Tooltip
 */
interface ToolbarButtonProps {
  icon: LucideIcon;
  ariaLabel: string;
  tooltip?: string;
  onClick?: () => void;
  className?: string;
}

const ToolbarButton = forwardRef<HTMLButtonElement, ToolbarButtonProps>(({ icon: Icon, ariaLabel, tooltip, onClick, className }, ref) => {
  const [isHovered, setIsHovered] = useState(false);

  const button = (
    <button
      ref={ref}
      type="button"
      onClick={onClick}
      onMouseEnter={() => setIsHovered(true)}
      onMouseLeave={() => setIsHovered(false)}
      className={cn(
        TOOLBAR_BUTTON_STYLES.base,
        TOOLBAR_BUTTON_STYLES.ghost,
        "relative group",
        // 响应式尺寸：移动端稍大
        "h-9 w-9 sm:h-9 sm:w-9",
        "rounded-xl",
        "flex items-center justify-center",
        "text-muted-foreground hover:text-foreground",
        "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/30",
        "transition-all duration-300 ease-out",
        className,
      )}
      aria-label={ariaLabel}
    >
      {/* 呼吸光晕 - hover 时显示 */}
      {isHovered && (
        <span className="absolute inset-0 rounded-xl bg-primary/10 animate-pulse" style={{ animationDuration: `${BREATH_DURATION}ms` }} />
      )}
      <Icon className="w-4 h-4 relative z-10" />
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
});

ToolbarButton.displayName = "ToolbarButton";

EditorToolbar.displayName = "EditorToolbar";
