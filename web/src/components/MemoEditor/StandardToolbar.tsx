import { FileIcon, LinkIcon, MapPinIcon, Maximize2Icon, SendIcon } from "lucide-react";
import { useCallback } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { Visibility } from "@/types/proto/api/v1/memo_service_pb";
import { AITagSuggestPopover } from "./components/AITagSuggestPopover";
import VisibilitySelector from "./Toolbar/VisibilitySelector";

interface StandardToolbarProps {
  /** Current content for AI tag suggestion */
  content: string;
  /** Whether an action is in progress */
  isLoading?: boolean;
  /** Whether the content is valid for saving */
  isValid?: boolean;
  /** Current visibility value */
  visibility?: number;
  /** Callback when visibility changes */
  onVisibilityChange?: (visibility: number) => void;
  /** Callback when file upload is clicked */
  onUploadFile?: () => void;
  /** Callback when link memo is clicked */
  onLinkMemo?: () => void;
  /** Callback when add location is clicked */
  onAddLocation?: () => void;
  /** Callback when focus mode is toggled */
  onToggleFocusMode?: () => void;
  /** Callback when tags should be inserted */
  onInsertTags?: (tags: string[]) => void;
  /** Callback when save is clicked */
  onSave?: () => void;
  /** Callback when cancel is clicked */
  onCancel?: () => void;
  /** Additional CSS classes */
  className?: string;
}

/**
 * StandardToolbar - PC端标准模式完整工具栏
 *
 * Features:
 * - File upload attachment
 * - Link memo relationship
 * - Add location tag
 * - Focus mode toggle
 * - AI tag suggestion
 * - Visibility selector
 * - Save/Cancel buttons
 */
export function StandardToolbar({
  content,
  isLoading = false,
  isValid = true,
  visibility,
  onVisibilityChange,
  onUploadFile,
  onLinkMemo,
  onAddLocation,
  onToggleFocusMode,
  onInsertTags,
  onSave,
  onCancel,
  className,
}: StandardToolbarProps) {
  const { t } = useTranslation();

  const handleInsertTags = useCallback(
    (tags: string[]) => {
      if (onInsertTags && tags.length > 0) {
        onInsertTags(tags);
      }
    },
    [onInsertTags],
  );

  return (
    <div className={cn("w-full flex items-center justify-between gap-3 px-4 py-2 border-t border-border bg-muted/30", className)}>
      {/* Left: Tools */}
      <div className="flex items-center gap-1">
        {onUploadFile && (
          <Button
            variant="ghost"
            size="sm"
            onClick={onUploadFile}
            disabled={isLoading}
            className="h-8 px-2 text-muted-foreground hover:text-foreground"
            title={t("common.upload")}
          >
            <FileIcon className="w-4 h-4" />
          </Button>
        )}

        {onLinkMemo && (
          <Button
            variant="ghost"
            size="sm"
            onClick={onLinkMemo}
            disabled={isLoading}
            className="h-8 px-2 text-muted-foreground hover:text-foreground"
            title={t("tooltip.link-memo")}
          >
            <LinkIcon className="w-4 h-4" />
          </Button>
        )}

        {onAddLocation && (
          <Button
            variant="ghost"
            size="sm"
            onClick={onAddLocation}
            disabled={isLoading}
            className="h-8 px-2 text-muted-foreground hover:text-foreground"
            title={t("tooltip.select-location")}
          >
            <MapPinIcon className="w-4 h-4" />
          </Button>
        )}

        {onToggleFocusMode && (
          <Button
            variant="ghost"
            size="sm"
            onClick={onToggleFocusMode}
            disabled={isLoading}
            className="h-8 px-2 text-muted-foreground hover:text-foreground"
            title={t("editor.focus-mode")}
          >
            <Maximize2Icon className="w-4 h-4" />
          </Button>
        )}
      </div>

      {/* Center: AI Tag Suggestion */}
      {content && onInsertTags && <AITagSuggestPopover content={content} onInsertTags={handleInsertTags} disabled={isLoading} />}

      {/* Right: Visibility + Actions */}
      <div className="flex items-center gap-2">
        {onVisibilityChange && <VisibilitySelector value={visibility ?? Visibility.PRIVATE} onChange={onVisibilityChange} />}

        {onCancel && (
          <Button variant="ghost" size="sm" onClick={onCancel} disabled={isLoading} className="h-8">
            {t("common.cancel")}
          </Button>
        )}

        {onSave && (
          <Button
            size="sm"
            onClick={onSave}
            disabled={!isValid || isLoading}
            className={cn("h-8 gap-1.5", isValid && isLoading ? "bg-primary text-primary-foreground" : "bg-muted text-muted-foreground")}
          >
            {isLoading ? t("common.saving") : t("editor.save")}
            {!isLoading && <SendIcon className="w-3.5 h-3.5" />}
          </Button>
        )}
      </div>
    </div>
  );
}
