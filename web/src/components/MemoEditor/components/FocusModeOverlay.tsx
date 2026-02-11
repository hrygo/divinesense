import { Minimize2Icon } from "lucide-react";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { FOCUS_MODE_STYLES } from "../constants";
import type { FocusModeExitButtonProps, FocusModeOverlayProps } from "../types";

/**
 * FocusModeOverlay - 焦点模式遮罩层（支持动画）
 */
export function FocusModeOverlay({ isActive, onToggle, isExiting = false }: FocusModeOverlayProps & { isExiting?: boolean }) {
  if (!isActive && !isExiting) return null;

  return (
    <button
      type="button"
      className={cn(
        FOCUS_MODE_STYLES.backdrop,
        // 退出动画
        isExiting && "opacity-0 transition-opacity duration-200",
      )}
      onClick={onToggle}
      aria-label="Exit focus mode"
    />
  );
}

/**
 * FocusModeExitButton - 焦点模式退出按钮
 */
export function FocusModeExitButton({ isActive, onToggle, title }: FocusModeExitButtonProps) {
  if (!isActive) return null;

  return (
    <Button variant="ghost" size="icon" className={FOCUS_MODE_STYLES.exitButton} onClick={onToggle} title={title}>
      <Minimize2Icon className="w-4 h-4" />
    </Button>
  );
}
