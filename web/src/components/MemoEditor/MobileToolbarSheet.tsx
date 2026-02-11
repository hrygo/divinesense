import { Link2, MapPin, Paperclip } from "lucide-react";
import { useCallback } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";
import { Sheet, SheetContent, SheetHeader, SheetTitle, SheetTrigger } from "@/components/ui/sheet";
import { cn } from "@/lib/utils";

/**
 * MobileToolbarSheet - iOS Share Sheet style slide-out toolbar
 *
 * Design Principles:
 * - 圆角：统一 rounded-xl / rounded-2xl（容器）
 * - 图标容器：w-11 h-11 (44px) 触摸友好
 * - 间距：gap-3 网格间距
 * - 颜色：紫色系（AI）、蓝色系（关联）、绿色系（位置）
 *
 * Provides mobile-optimized tool access with bottom sheet pattern:
 * - File upload attachment
 * - Link memo relationship
 * - Add location tag
 *
 * Only renders on mobile devices (< md breakpoint)
 */

interface MobileToolbarSheetProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onUploadFile?: () => void;
  onLinkMemo?: () => void;
  onAddLocation?: () => void;
  trigger?: React.ReactNode;
  triggerClassName?: string;
}

export function MobileToolbarSheet({
  open,
  onOpenChange,
  onUploadFile,
  onLinkMemo,
  onAddLocation,
  trigger,
  triggerClassName,
}: MobileToolbarSheetProps) {
  const { t } = useTranslation();

  const handleAction = useCallback(
    (action: () => void) => {
      // Close sheet after action
      onOpenChange(false);
      action();
    },
    [onOpenChange],
  );

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      {trigger && (
        <SheetTrigger asChild>
          <span className={triggerClassName}>{trigger}</span>
        </SheetTrigger>
      )}
      <SheetContent side="bottom" className="h-[45vh] rounded-t-2xl border-t border-border/50 bg-background/95 backdrop-blur-md p-0">
        {/* Drag Handle Indicator */}
        <div className="flex justify-center pt-3 pb-2">
          <div className="w-12 h-1.5 bg-muted-foreground/20 rounded-full" />
        </div>

        <SheetHeader className="px-6 pb-3">
          <SheetTitle className="text-center text-sm font-medium text-foreground/80">{t("editor.more-tools")}</SheetTitle>
        </SheetHeader>

        {/* Tool Actions Grid */}
        <div className="grid grid-cols-3 gap-4 px-6">
          {/* File Upload */}
          {onUploadFile && (
            <Button
              variant="ghost"
              className="flex flex-col gap-3 h-auto py-4 rounded-2xl hover:bg-accent/50 active:bg-accent transition-all active:scale-95"
              onClick={() => handleAction(onUploadFile)}
            >
              <div className="w-11 h-11 mx-auto rounded-xl bg-gradient-to-br from-primary/20 to-primary/5 flex items-center justify-center shadow-sm">
                <Paperclip className="w-5 h-5 text-primary" />
              </div>
              <span className="text-xs text-foreground/80">{t("editor.upload-file")}</span>
            </Button>
          )}

          {/* Link Memo */}
          {onLinkMemo && (
            <Button
              variant="ghost"
              className="flex flex-col gap-3 h-auto py-4 rounded-2xl hover:bg-accent/50 active:bg-accent transition-all active:scale-95"
              onClick={() => handleAction(onLinkMemo)}
            >
              <div className="w-11 h-11 mx-auto rounded-xl bg-gradient-to-br from-blue-500/20 to-blue-500/5 flex items-center justify-center shadow-sm">
                <Link2 className="w-5 h-5 text-blue-500" />
              </div>
              <span className="text-xs text-foreground/80">{t("editor.link-memo")}</span>
            </Button>
          )}

          {/* Add Location */}
          {onAddLocation && (
            <Button
              variant="ghost"
              className="flex flex-col gap-3 h-auto py-4 rounded-2xl hover:bg-accent/50 active:bg-accent transition-all active:scale-95"
              onClick={() => handleAction(onAddLocation)}
            >
              <div className="w-11 h-11 mx-auto rounded-xl bg-gradient-to-br from-green-500/20 to-green-500/5 flex items-center justify-center shadow-sm">
                <MapPin className="w-5 h-5 text-green-500" />
              </div>
              <span className="text-xs text-foreground/80">{t("editor.add-location")}</span>
            </Button>
          )}
        </div>

        {/* Cancel Button */}
        <div className="px-6 pb-8 pt-2">
          <Button variant="outline" className="w-full h-11 rounded-xl font-medium" onClick={() => onOpenChange(false)}>
            {t("cancel")}
          </Button>
        </div>
      </SheetContent>
    </Sheet>
  );
}

interface MobileToolbarTriggerProps {
  onClick: () => void;
  disabled?: boolean;
  className?: string;
}

/**
 * MobileToolbarTrigger - Button to open the mobile toolbar sheet
 * Shows expand icon with "More" indicator
 */
export function MobileToolbarTrigger({ onClick, disabled, className }: MobileToolbarTriggerProps) {
  const { t } = useTranslation();

  return (
    <Button
      variant="ghost"
      size="sm"
      onClick={onClick}
      disabled={disabled}
      className={cn("h-7 px-2 text-xs text-muted-foreground hover:text-foreground", className)}
      aria-label={t("editor.more-tools")}
    >
      {/* More indicator dots */}
      <span className="flex items-center gap-0.5 mr-1">
        <span className="w-1 h-1 bg-current rounded-full opacity-50" />
        <span className="w-1 h-1 bg-current rounded-full opacity-50" />
        <span className="w-1 h-1 bg-current rounded-full opacity-50" />
      </span>
      {t("editor.more-tools")}
    </Button>
  );
}
