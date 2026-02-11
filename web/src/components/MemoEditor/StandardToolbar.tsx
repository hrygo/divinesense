import { Globe, Link2, Lock, MapPin, Maximize2, MoreHorizontal, Paperclip, Send, Sparkles } from "lucide-react";
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
 * Design Principles:
 * - 统一使用 lucide-react 图标系统
 * - 按钮高度：桌面端 h-9 (36px)、移动端 h-11 (44px)
 * - 间距：gap-2 (8px) 桌面、gap-1.5 (6px) 移动
 * - 圆角：统一 rounded-xl
 *
 * Features:
 * - File upload attachment
 * - Link memo relationship
 * - Add location tag
 * - Focus mode toggle
 * - AI tag suggestion with subtle glow
 * - Visibility selector
 * - Save/Cancel buttons with micro-interactions
 *
 * Mobile (< 640px):
 * - 工具按钮收纳到"更多"菜单
 * - 简化可见性切换为图标
 * - AI 指示器替代完整建议
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

  // 移动端 AI 指示器：显示 Sparkles 图标，点击可触发标签建议
  const hasContentForAI = content && content.trim().length > 10;
  const showMobileAIIndicator = isMobile && hasContentForAI && onInsertTags;

  return (
    <>
      <div
        className={cn(
          "w-full flex items-center justify-between gap-1.5 sm:gap-3 px-3 sm:px-4 py-2.5 border-t border-border/50 bg-muted/20 backdrop-blur-sm",
          className,
        )}
      >
        {/* Left: Tools - 移动端只显示"更多"按钮 */}
        <div className="flex items-center gap-1 sm:gap-2">
          {isMobile ? (
            // 移动端：更多按钮
            <>
              {/* AI 指示器 - 有内容时显示 */}
              {showMobileAIIndicator && (
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => handleInsertTags(["#AI"])}
                  disabled={isLoading}
                  className={cn(
                    "h-9 w-9 rounded-xl text-purple-500 hover:text-purple-600 hover:bg-purple-50 dark:hover:bg-purple-950/20",
                    "relative overflow-hidden",
                  )}
                  title={t("editor.ai-suggest-tags-title")}
                >
                  <Sparkles className="w-4 h-4 relative z-10" />
                  {/* AI 微光效果 */}
                  <span className="absolute inset-0 bg-gradient-to-tr from-purple-500/0 via-purple-500/10 to-purple-500/0 animate-pulse" />
                </Button>
              )}

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
                    className="h-9 w-9 rounded-xl text-muted-foreground hover:text-foreground hover:bg-accent/50"
                    aria-label={t("editor.more-tools")}
                  >
                    <MoreHorizontal className="w-4 h-4" />
                  </Button>
                }
              />
            </>
          ) : (
            <>
              {/* 桌面端：显示所有工具按钮 */}
              {onUploadFile && (
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={onUploadFile}
                  disabled={isLoading}
                  className="h-9 w-9 rounded-xl text-muted-foreground hover:text-foreground hover:bg-accent/50 transition-all hover:scale-105 active:scale-95"
                  title={t("common.upload")}
                >
                  <Paperclip className="w-4 h-4" />
                </Button>
              )}

              {onLinkMemo && (
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={onLinkMemo}
                  disabled={isLoading}
                  className="h-9 w-9 rounded-xl text-muted-foreground hover:text-foreground hover:bg-accent/50 transition-all hover:scale-105 active:scale-95"
                  title={t("tooltip.link-memo")}
                >
                  <Link2 className="w-4 h-4" />
                </Button>
              )}

              {onAddLocation && (
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={onAddLocation}
                  disabled={isLoading}
                  className="h-9 w-9 rounded-xl text-muted-foreground hover:text-foreground hover:bg-accent/50 transition-all hover:scale-105 active:scale-95"
                  title={t("tooltip.select-location")}
                >
                  <MapPin className="w-4 h-4" />
                </Button>
              )}

              {onToggleFocusMode && (
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={onToggleFocusMode}
                  disabled={isLoading}
                  className="h-9 w-9 rounded-xl text-muted-foreground hover:text-foreground hover:bg-accent/50 transition-all hover:scale-105 active:scale-95"
                  title={t("editor.focus-mode")}
                >
                  <Maximize2 className="w-4 h-4" />
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
        <div className="flex items-center gap-1.5 sm:gap-2 flex-shrink-0">
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
                  onClick={() => {
                    const newVisibility = visibility === Visibility.PRIVATE ? Visibility.PUBLIC : Visibility.PRIVATE;
                    onVisibilityChange(newVisibility);
                  }}
                  disabled={isLoading}
                  className="h-9 w-9 p-0 rounded-xl text-muted-foreground hover:text-foreground hover:bg-accent/50 transition-all hover:scale-105 active:scale-95"
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
                "h-9 px-3 rounded-xl text-sm font-medium transition-all hover:scale-105 active:scale-95",
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
                "h-9 min-w-[80px] gap-2 px-4 rounded-xl text-sm font-medium transition-all",
                "hover:scale-105 active:scale-95",
                "relative overflow-hidden",
                isValid
                  ? "bg-primary text-primary-foreground hover:bg-primary/90 shadow-sm hover:shadow"
                  : "bg-muted text-muted-foreground cursor-not-allowed",
              )}
            >
              {isLoading ? (
                <>
                  <span className="animate-spin w-4 h-4 border-2 border-current border-t-transparent rounded-full" />
                  <span className="hidden sm:inline">{t("common.saving")}</span>
                </>
              ) : (
                <>
                  <span className="hidden sm:inline">{t("editor.save")}</span>
                  <Send className="w-4 h-4" />
                </>
              )}
            </Button>
          )}
        </div>
      </div>
    </>
  );
}
