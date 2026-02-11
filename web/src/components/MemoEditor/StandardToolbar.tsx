import { Globe, Lock, Maximize2Icon, MoreHorizontalIcon, SendIcon } from "lucide-react";
import { useCallback, useState } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";
import useMediaQuery from "@/hooks/useMediaQuery";
import { cn } from "@/lib/utils";
import { Visibility } from "@/types/proto/api/v1/memo_service_pb";
import { AITagSuggestPopover } from "./components/AITagSuggestPopover";
import { MobileToolbarSheet } from "./MobileToolbarSheet";
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
 * StandardToolbar - 标准模式完整工具栏（响应式）
 *
 * Features:
 * - File upload attachment
 * - Link memo relationship
 * - Add location tag
 * - Focus mode toggle
 * - AI tag suggestion
 * - Visibility selector
 * - Save/Cancel buttons
 *
 * Mobile (< 640px):
 * - 工具按钮收纳到"更多"菜单
 * - 简化可见性切换为图标
 * - 隐藏 Focus Mode 和 AI 建议
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
  const isMobile = useMediaQuery("sm"); // < 640px
  const [mobileToolsOpen, setMobileToolsOpen] = useState(false);

  const handleInsertTags = useCallback(
    (tags: string[]) => {
      if (onInsertTags && tags.length > 0) {
        onInsertTags(tags);
      }
    },
    [onInsertTags],
  );

  return (
    <>
      <div
        className={cn(
          "w-full flex items-center justify-between gap-1.5 sm:gap-3 px-2 sm:px-4 py-2 border-t border-border bg-muted/30",
          className,
        )}
      >
        {/* Left: Tools - 移动端只显示"更多"按钮 */}
        <div className="flex items-center gap-0.5 sm:gap-1">
          {isMobile ? (
            // 移动端：更多按钮
            <MobileToolbarSheet
              open={mobileToolsOpen}
              onOpenChange={setMobileToolsOpen}
              onUploadFile={onUploadFile}
              onLinkMemo={onLinkMemo}
              onAddLocation={onAddLocation}
              trigger={
                <Button
                  variant="ghost"
                  size="sm"
                  disabled={isLoading}
                  className="h-8 w-8 text-muted-foreground hover:text-foreground"
                  aria-label={t("editor.more-tools")}
                >
                  <MoreHorizontalIcon className="w-4 h-4" />
                </Button>
              }
            />
          ) : (
            <>
              {/* 桌面端：显示所有工具按钮 */}
              {onUploadFile && (
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={onUploadFile}
                  disabled={isLoading}
                  className="h-8 px-2 text-muted-foreground hover:text-foreground"
                  title={t("common.upload")}
                >
                  <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4.5v15m7.5-7.5h-15" />
                  </svg>
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
                  <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path
                      strokeLinecap="round"
                      strokeLinejoin="round"
                      strokeWidth={2}
                      d="M13.828 10.172a4 4 0 00-5.656 0l-4 4a4 4 0 105.656 5.656l1.102-1.101m-.758-4.899a4 4 0 005.656 0l4-4a4 4 0 00-5.656-5.656l-1.1 1.1"
                    />
                  </svg>
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
                  <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path
                      strokeLinecap="round"
                      strokeLinejoin="round"
                      strokeWidth={2}
                      d="M17.657 16.657L13.414 20.9a1.998 1.998 0 01-2.827 0l-4.244-4.243a8 8 0 1111.314 0z"
                    />
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 11a3 3 0 11-6 0 3 3 0 016 0z" />
                  </svg>
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
            </>
          )}
        </div>

        {/* Center: AI Tag Suggestion - 仅在桌面端显示 */}
        {!isMobile && content && onInsertTags && (
          <AITagSuggestPopover content={content} onInsertTags={handleInsertTags} disabled={isLoading} />
        )}

        {/* Right: Visibility + Actions */}
        <div className="flex items-center gap-1 sm:gap-2 flex-shrink-0">
          {/* 可见性选择器 */}
          {onVisibilityChange && (
            <>
              {/* 桌面端：完整选择器 */}
              <div className="hidden sm:block">
                <VisibilitySelector value={visibility ?? Visibility.PRIVATE} onChange={onVisibilityChange} />
              </div>

              {/* 移动端：简化图标按钮 */}
              {isMobile && (
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => onVisibilityChange(visibility === Visibility.PRIVATE ? Visibility.PUBLIC : Visibility.PRIVATE)}
                  disabled={isLoading}
                  className="h-8 w-8 p-0 text-muted-foreground hover:text-foreground"
                  title={visibility === Visibility.PUBLIC ? t("visibility.public") : t("visibility.private")}
                >
                  {visibility === Visibility.PUBLIC ? <Globe className="w-4 h-4" /> : <Lock className="w-4 h-4" />}
                </Button>
              )}
            </>
          )}

          {onCancel && (
            <Button
              variant="ghost"
              size="sm"
              onClick={onCancel}
              disabled={isLoading}
              className={cn(
                "h-8 px-2 text-sm",
                isMobile && "hidden", // 移动端隐藏取消按钮，节省空间
              )}
            >
              {t("common.cancel")}
            </Button>
          )}

          {onSave && (
            <Button
              size="sm"
              onClick={onSave}
              disabled={!isValid || isLoading}
              className={cn(
                "h-8 gap-1.5 px-2 sm:px-3 text-sm",
                isValid ? "bg-primary text-primary-foreground hover:bg-primary/90" : "bg-muted text-muted-foreground",
              )}
            >
              <span className="hidden sm:inline">{isLoading ? t("common.saving") : t("editor.save")}</span>
              {!isLoading && <SendIcon className="w-3.5 h-3.5 sm:w-4 sm:h-4" />}
              {isLoading && <span className="sm:hidden">{t("common.saving")}</span>}
            </Button>
          )}
        </div>
      </div>
    </>
  );
}
