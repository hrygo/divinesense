import { Link2, MapPin, Paperclip } from "lucide-react";
import { useCallback } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";
import { Sheet, SheetContent, SheetHeader, SheetTitle, SheetTrigger } from "@/components/ui/sheet";
import { cn } from "@/lib/utils";

interface MobileToolbarSheetProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onUploadFile?: () => void;
  onLinkMemo?: () => void;
  onAddLocation?: () => void;
  trigger?: React.ReactNode;
  triggerClassName?: string;
}

/**
 * MobileToolbarSheet - iOS Share Sheet style slide-out toolbar
 *
 * Provides mobile-optimized tool access with bottom sheet pattern:
 - File upload attachment
 - Link memo relationship
 - Add location tag
 *
 * Only renders on mobile devices (< md breakpoint)
 */
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
        <SheetTrigger asChild className={triggerClassName}>
          {trigger}
        </SheetTrigger>
      )}
      <SheetContent side="bottom" className="h-[50vh] rounded-t-2xl border-t border-border/50 bg-background/95 backdrop-blur-sm p-0">
        {/* Drag Handle Indicator */}
        <div className="flex justify-center pt-3 pb-1">
          <div className="w-10 h-1 bg-muted-foreground/30 rounded-full" />
        </div>

        <SheetHeader className="px-4 pb-2">
          <SheetTitle className="text-center text-sm font-medium text-muted-foreground">{t("editor.more-tools")}</SheetTitle>
        </SheetHeader>

        {/* Tool Actions Grid */}
        <div className="grid grid-cols-3 gap-3 p-4">
          {/* File Upload */}
          {onUploadFile && (
            <Button
              variant="ghost"
              className="flex flex-col gap-2 h-auto py-4 rounded-lg hover:bg-accent"
              onClick={() => handleAction(onUploadFile)}
            >
              <div className="w-10 h-10 mx-auto rounded-full bg-primary/10 flex items-center justify-center">
                <Paperclip className="w-5 h-5 text-primary" />
              </div>
              <span className="text-xs text-foreground">{t("editor.upload-file")}</span>
            </Button>
          )}

          {/* Link Memo */}
          {onLinkMemo && (
            <Button
              variant="ghost"
              className="flex flex-col gap-2 h-auto py-4 rounded-lg hover:bg-accent"
              onClick={() => handleAction(onLinkMemo)}
            >
              <div className="w-10 h-10 mx-auto rounded-full bg-blue-500/10 flex items-center justify-center">
                <Link2 className="w-5 h-5 text-blue-500" />
              </div>
              <span className="text-xs text-foreground">{t("editor.link-memo")}</span>
            </Button>
          )}

          {/* Add Location */}
          {onAddLocation && (
            <Button
              variant="ghost"
              className="flex flex-col gap-2 h-auto py-4 rounded-lg hover:bg-accent"
              onClick={() => handleAction(onAddLocation)}
            >
              <div className="w-10 h-10 mx-auto rounded-full bg-green-500/10 flex items-center justify-center">
                <MapPin className="w-5 h-5 text-green-500" />
              </div>
              <span className="text-xs text-foreground">{t("editor.add-location")}</span>
            </Button>
          )}
        </div>

        {/* Cancel Button */}
        <div className="px-4 pb-6">
          <Button variant="outline" className="w-full rounded-lg" onClick={() => onOpenChange(false)}>
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
