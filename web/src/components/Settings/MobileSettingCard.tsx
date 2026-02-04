import { ChevronRightIcon } from "lucide-react";
import { ReactNode } from "react";
import { cn } from "@/lib/utils";

interface MobileSettingCardProps {
  title: string;
  description?: string;
  value?: ReactNode;
  icon?: ReactNode;
  onClick?: () => void;
  className?: string;
  children?: ReactNode;
  disabled?: boolean;
}

/**
 * Mobile-optimized setting card with large touch targets (44px+ min height).
 * Displays settings in a card format suitable for mobile list views.
 */
const MobileSettingCard = ({ title, description, value, icon, onClick, className, children, disabled = false }: MobileSettingCardProps) => {
  return (
    <div
      className={cn(
        "flex items-center gap-3 px-4 py-3 bg-background border-b border-border min-h-[56px]",
        !disabled && onClick && "active:bg-muted/50 cursor-pointer",
        disabled && "opacity-50 cursor-not-allowed",
        className,
      )}
      onClick={!disabled ? onClick : undefined}
      role={onClick ? "button" : undefined}
      tabIndex={onClick && !disabled ? 0 : undefined}
    >
      {icon && <div className="shrink-0 w-8 h-8 flex items-center justify-center text-muted-foreground">{icon}</div>}

      <div className="flex-1 min-w-0">
        <div className="text-sm font-medium text-foreground truncate">{title}</div>
        {description && <div className="text-xs text-muted-foreground truncate mt-0.5">{description}</div>}
      </div>

      {value !== undefined && <div className="shrink-0 text-sm text-muted-foreground">{value}</div>}

      {children && <div className="shrink-0">{children}</div>}

      {onClick && !value && !children && <ChevronRightIcon className="shrink-0 w-5 h-5 text-muted-foreground" />}
    </div>
  );
};

export default MobileSettingCard;
