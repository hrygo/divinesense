import { Dna, MessageSquare, Terminal } from "lucide-react";
import { useTranslation } from "react-i18next";
import { cn } from "@/lib/utils";
import type { AIMode } from "@/types/aichat";
import { PARROT_THEMES } from "@/types/parrot";

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

  // 根据当前模式获取配置 - 使用 PARROT_THEMES
  const normalTheme = PARROT_THEMES.NORMAL;
  const geekTheme = PARROT_THEMES.GEEK;
  const evolutionTheme = PARROT_THEMES.EVOLUTION;

  const modeConfig = {
    normal: {
      icon: MessageSquare,
      label: t("ai.mode.normal"),
      tooltip: t("ai.mode.normal_tooltip"),
      color: normalTheme.text,
      bgColor: normalTheme.inputBg,
      borderColor: normalTheme.inputBorder,
    },
    geek: {
      icon: Terminal,
      label: t("ai.mode.geek"),
      tooltip: t("ai.mode.geek_tooltip"),
      color: geekTheme.text,
      bgColor: geekTheme.inputBg,
      borderColor: geekTheme.inputBorder,
    },
    evolution: {
      icon: Dna,
      label: isAdmin ? t("ai.mode.evolution") : t("ai.mode.evolution_locked"),
      tooltip: isAdmin ? t("ai.mode.evolution_tooltip") : t("ai.mode.evolution_locked_tooltip"),
      color: isAdmin ? evolutionTheme.text : "text-muted-foreground opacity-50",
      bgColor: isAdmin ? evolutionTheme.inputBg : "bg-muted/50",
      borderColor: isAdmin ? evolutionTheme.inputBorder : "",
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
          "flex items-center gap-1.5 px-3 py-2 rounded-lg transition-colors",
          "disabled:opacity-50 disabled:cursor-not-allowed",
          "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring",
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
        <span className="text-sm font-medium">{config.label}</span>
      </button>
    );
  }

  // 工具栏变体 - 更紧凑，用于输入框工具栏
  if (isToolbar) {
    return (
      <button
        onClick={handleCycle}
        disabled={disabled}
        className={cn(
          "flex items-center gap-1 h-7 px-2 rounded transition-colors",
          "disabled:opacity-50 disabled:cursor-not-allowed",
          "focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring",
          currentMode !== "normal" && "border",
          config.color,
          config.bgColor,
          config.borderColor,
        )}
        title={config.tooltip}
        aria-label={config.tooltip}
        aria-pressed={currentMode !== "normal"}
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
        "flex items-center gap-1.5 px-2.5 py-1.5 rounded-lg transition-colors",
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
