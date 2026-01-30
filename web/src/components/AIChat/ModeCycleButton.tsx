import { Dna, MessageSquare, Terminal } from "lucide-react";
import { useTranslation } from "react-i18next";
import { cn } from "@/lib/utils";
import type { AIMode } from "@/types/aichat";

interface ModeCycleButtonProps {
  currentMode: AIMode;
  onModeChange: (mode: AIMode) => void;
  disabled?: boolean;
  variant?: "header" | "toolbar" | "mobile";
  isAdmin?: boolean;
}

/**
 * Mode Cycle Button - 三态模式切换按钮
 *
 * 循环顺序：普通 → 极客 → 进化 → 普通
 *
 * 设计原则：
 * - 单按钮循环切换，简洁直观
 * - 图标 + 状态标签清晰展示当前模式
 * - PC 端和移动端体验一致
 * - 颜色区分：普通(灰) → 极客(绿) → 进化(紫)
 */
export function ModeCycleButton({ currentMode, onModeChange, disabled = false, variant = "header", isAdmin = true }: ModeCycleButtonProps) {
  const { t } = useTranslation();

  const isToolbar = variant === "toolbar";
  const isMobile = variant === "mobile";

  // 点击切换到下一个模式
  const handleCycle = () => {
    const modes: AIMode[] = ["normal", "geek", "evolution"];
    const currentIndex = modes.indexOf(currentMode);
    const nextIndex = (currentIndex + 1) % modes.length;
    const nextMode = modes[nextIndex];
    onModeChange(nextMode);
  };

  // 根据当前模式获取配置
  const modeConfig = {
    normal: {
      icon: MessageSquare,
      label: t("ai.mode.normal"),
      tooltip: t("ai.mode.normal_tooltip"),
      color: "text-muted-foreground",
      bgColor: "hover:bg-muted",
      borderColor: "",
    },
    geek: {
      icon: Terminal,
      label: t("ai.mode.geek"),
      tooltip: t("ai.mode.geek_tooltip"),
      color: "text-green-600 dark:text-green-400",
      bgColor: "bg-green-500/10 hover:bg-green-500/15",
      borderColor: "border-green-500/30",
    },
    evolution: {
      icon: Dna,
      label: isAdmin ? t("ai.mode.evolution") : t("ai.mode.evolution_locked"),
      tooltip: isAdmin ? t("ai.mode.evolution_tooltip") : t("ai.mode.evolution_locked_tooltip"),
      color: isAdmin ? "text-purple-600 dark:text-purple-400" : "text-muted-foreground opacity-50",
      bgColor: isAdmin ? "bg-purple-500/10 hover:bg-purple-500/15" : "bg-muted/50",
      borderColor: isAdmin ? "border-purple-500/30" : "",
    },
  };

  const config = modeConfig[currentMode];
  const Icon = config.icon;

  // 移动端变体 - 紧凑样式
  if (isMobile) {
    return (
      <button
        onClick={handleCycle}
        disabled={disabled}
        className={cn(
          "flex items-center gap-1.5 px-3 py-2 rounded-lg transition-all",
          "disabled:opacity-50 disabled:cursor-not-allowed",
          "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring",
          config.color,
          config.bgColor,
        )}
        title={config.tooltip}
        aria-label={config.tooltip}
      >
        <Icon className="w-4 h-4 shrink-0" />
        <span className="text-sm font-medium">{config.label}</span>
      </button>
    );
  }

  // 工具栏变体 - 更紧凑
  if (isToolbar) {
    return (
      <button
        onClick={handleCycle}
        disabled={disabled}
        className={cn(
          "flex items-center gap-1 h-7 px-2 rounded transition-all",
          "disabled:opacity-50 disabled:cursor-not-allowed",
          "focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring",
          config.color,
          config.bgColor,
        )}
        title={config.tooltip}
        aria-label={config.tooltip}
      >
        <Icon className="w-3.5 h-3.5 shrink-0" />
        <span className="text-xs whitespace-nowrap">{config.label}</span>
      </button>
    );
  }

  // 头部变体 - 默认
  return (
    <button
      onClick={handleCycle}
      disabled={disabled}
      className={cn(
        "flex items-center gap-1.5 px-2.5 py-1.5 rounded-lg transition-all",
        "disabled:opacity-50 disabled:cursor-not-allowed",
        "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2",
        currentMode !== "normal" && "border",
        config.color,
        config.bgColor,
        config.borderColor,
      )}
      title={config.tooltip}
      aria-label={config.tooltip}
      aria-pressed={currentMode !== "normal"}
    >
      <Icon className="w-4 h-4 shrink-0" />
      <span className="text-sm font-medium hidden sm:inline whitespace-nowrap">{config.label}</span>
    </button>
  );
}
