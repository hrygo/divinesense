import { Globe, Link2, Lock, type LucideIcon, MapPin, Maximize2, Paperclip, Shield, Sparkles } from "lucide-react";
import type { FC } from "react";
import { forwardRef, useMemo, useState } from "react";
import { Button } from "@/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import useMediaQuery from "@/hooks/useMediaQuery";
import { cn } from "@/lib/utils";
import { Visibility } from "@/types/proto/api/v1/memo_service_pb";
import { useTranslate } from "@/utils/i18n";
import { TOOLBAR_BUTTON_STYLES } from "../constants";
import { useEditorContext } from "../state";
import type { EditorToolbarProps } from "../types";
import { AIFormatButton } from "./AIFormatButton";
import { AITagButton } from "./AITagButton";
import { VisibilityToggleGroup } from "./VisibilityToggleGroup";

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
  onUploadAttachment,
  onLinkMemo,
  onToggleFocusMode,
  onVisibilityChange,
  onInsertTags,
  onFormatContent,
  memoName,
}) => {
  const t = useTranslate();
  const md = useMediaQuery("md");
  const { state } = useEditorContext();

  const hasCancel = !!onCancel;
  const hasContent = state.content.trim().length > 0;
  const isSaving = state.ui.isLoading.saving;

  const handleSave = () => {
    onSave?.();
  };

  return (
    <div className="w-full flex items-center justify-between gap-2 sm:gap-3 px-4 sm:px-5 py-3 border-t border-border/40 bg-muted/20 backdrop-blur-sm">
      {/* Left: Tool buttons group + AI tags */}
      <div className="flex items-center gap-1.5 sm:gap-2">
        {/* Upload attachment - always visible */}
        <ToolbarButton
          icon={Paperclip}
          ariaLabel={t("editor.add-attachment")}
          tooltip={t("editor.add-attachment")}
          onClick={onUploadAttachment}
        />

        {/* Location button - now visible on both desktop and mobile */}
        <ToolbarButton
          icon={MapPin}
          ariaLabel={t("editor.add-location")}
          tooltip={t("editor.add-location")}
          onClick={() => {
            // TODO: Implement location picker
            console.log("Add location");
          }}
        />

        {/* Link memo button - both desktop and mobile */}
        {onLinkMemo && (
          <ToolbarButton icon={Link2} ariaLabel={t("editor.link-memo")} tooltip={t("editor.link-memo")} onClick={onLinkMemo} />
        )}

        {/* Divider */}
        {(onInsertTags || onFormatContent) && <div className="w-px h-5 bg-border/50 mx-1" />}

        {/* AI Tag button - both desktop and mobile */}
        {onInsertTags && <AITagButton content={state.content} onInsertTags={onInsertTags} compact={!md} />}

        {/* AI Format button - both desktop and mobile */}
        {onFormatContent && <AIFormatButton content={state.content} onFormat={onFormatContent} compact={!md} />}
      </div>

      {/* Right: Settings and action buttons */}
      <div className="flex items-center gap-1.5 sm:gap-2">
        {/* Visibility: PC = button group, Mobile = cycle button */}
        {onVisibilityChange &&
          (md ? (
            <VisibilityToggleGroup value={state.metadata.visibility} onChange={onVisibilityChange} />
          ) : (
            <VisibilityCycleButton value={state.metadata.visibility} onChange={onVisibilityChange} />
          ))}

        {/* Focus mode button - Desktop only */}
        {md && onToggleFocusMode && (
          <ToolbarButton icon={Maximize2} ariaLabel={t("editor.focus-mode")} tooltip={t("editor.focus-mode")} onClick={onToggleFocusMode} />
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

// ============================================================================
// ToolbarButton - Enhanced tool button with micro-interactions and Tooltip
// ============================================================================

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
        "h-9 w-9 rounded-xl",
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
