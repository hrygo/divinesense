/**
 * VisibilityToggleGroup - 可见性快速切换按钮组
 *
 * 设计哲学：「禅意智识」
 * - 直观：三个按钮一目了然
 * - 快捷：单击直接切换，无需下拉菜单
 * - 反馈：当前状态高亮显示
 */

import { Globe, Lock, Shield } from "lucide-react";
import { type FC, memo, useMemo } from "react";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { cn } from "@/lib/utils";
import { Visibility } from "@/types/proto/api/v1/memo_service_pb";
import { useTranslate } from "@/utils/i18n";

// ============================================================================
// VisibilityToggleGroup Component
// ============================================================================

interface VisibilityToggleGroupProps {
  value: Visibility;
  onChange: (value: Visibility) => void;
  disabled?: boolean;
  compact?: boolean;
}

export const VisibilityToggleGroup: FC<VisibilityToggleGroupProps> = memo(function VisibilityToggleGroup({
  value,
  onChange,
  disabled = false,
  compact = false,
}) {
  const t = useTranslate();

  const options = useMemo(
    () => [
      { value: Visibility.PRIVATE, label: t("memo.visibility.private"), icon: Lock },
      { value: Visibility.PROTECTED, label: t("memo.visibility.protected"), icon: Shield },
      { value: Visibility.PUBLIC, label: t("memo.visibility.public"), icon: Globe },
    ],
    [t],
  );

  return (
    <div className={cn("flex items-center bg-muted/30 rounded-xl p-0.5", disabled && "opacity-50 pointer-events-none")}>
      {options.map((option) => {
        const isActive = value === option.value;
        const Icon = option.icon;

        const button = (
          <button
            type="button"
            onClick={() => onChange(option.value)}
            disabled={disabled}
            className={cn(
              "flex items-center gap-1 transition-all duration-200",
              "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/30",
              compact
                ? cn(
                    "h-9 w-9 px-0 justify-center rounded-xl",
                    isActive && "bg-primary/15 text-primary",
                    !isActive && "text-muted-foreground hover:text-foreground hover:bg-muted/50",
                  )
                : cn(
                    "h-9 px-2 rounded-xl",
                    isActive && "bg-primary/15 text-primary",
                    !isActive && "text-muted-foreground hover:text-foreground hover:bg-muted/50",
                  ),
            )}
            aria-label={option.label}
            aria-pressed={isActive}
          >
            <Icon className="w-4 h-4" />
            {!compact && <span className="text-xs hidden lg:inline">{option.label}</span>}
          </button>
        );

        return (
          <Tooltip key={option.value} delayDuration={300}>
            <TooltipTrigger asChild>{button}</TooltipTrigger>
            <TooltipContent side="top">
              <p>{option.label}</p>
            </TooltipContent>
          </Tooltip>
        );
      })}
    </div>
  );
});
