import { Maximize2, Minimize2, Sparkles } from "lucide-react";
import { useMemo } from "react";
import { useTranslation } from "react-i18next";
import { AnimatedAvatar } from "@/components/AIChat/AnimatedAvatar";
import { HeaderSessionStats } from "@/components/AIChat/HeaderSessionStats";
import { cn } from "@/lib/utils";
import type { AIMode } from "@/types/aichat";
import type { Block as AIBlock } from "@/types/block";
import { CapabilityStatus, CapabilityType } from "@/types/capability";
import { PARROT_THEMES, ParrotAgentType } from "@/types/parrot";

interface ChatHeaderProps {
  isThinking?: boolean;
  className?: string;
  currentCapability?: CapabilityType;
  capabilityStatus?: CapabilityStatus;
  currentMode?: AIMode;
  immersiveMode?: boolean;
  onImmersiveModeToggle?: (enabled: boolean) => void;
  /** Phase 4: Blocks for session stats display */
  blocks?: AIBlock[];
}

/**
 * Chat Header - 简洁状态显示
 *
 * 设计原则：
 * - 极客模式下保持清爽，不使用过度特效
 * - 通过颜色、字体、状态点传达状态
 * - 移动端和桌面端体验一致
 */

/**
 * 根据当前能力和状态获取动作描述
 */
function getActionDescription(
  capability: CapabilityType,
  status: CapabilityStatus,
  mode: AIMode,
  t: (key: string) => string,
): string | null {
  if (status === "idle") return null;

  if (status === "thinking") {
    // Use mode-specific thinking text
    if (mode === "geek") {
      return t("ai.geek_mode.thinking");
    }
    if (mode === "evolution") {
      return t("ai.evolution_mode.thinking");
    }
    return t("ai.states.thinking");
  }

  if (status === "processing") {
    switch (capability) {
      case CapabilityType.MEMO:
        return t("ai.parrot.status.searching-memos");
      case CapabilityType.SCHEDULE:
        return t("ai.parrot.status.querying-schedule");
      case CapabilityType.AMAZING:
        return t("ai.parrot.status.analyzing");
      default:
        return t("ai.processing");
    }
  }

  return null;
}

/**
 * 根据当前模式获取 ParrotAgentType
 */
function modeToParrotType(mode: AIMode): ParrotAgentType {
  switch (mode) {
    case "geek":
      return ParrotAgentType.GEEK;
    case "evolution":
      return ParrotAgentType.EVOLUTION;
    default:
      return ParrotAgentType.AMAZING; // Normal mode uses AMAZING theme
  }
}

/**
 * 根据当前模式获取样式配置 - 使用 PARROT_THEMES
 */
function getModeStyle(mode: AIMode, t: (key: string) => string) {
  const parrotType = modeToParrotType(mode);
  const theme = PARROT_THEMES[parrotType];

  switch (mode) {
    case "geek":
      return {
        border: "border-sky-200 dark:border-slate-700",
        bg: theme.headerBg,
        text: theme.text,
        name: "font-mono",
        avatarBorder: "border-sky-500/30",
        avatarBg: "bg-sky-500/10",
        statusText: theme.text,
        statusDot: "bg-sky-500",
        statusPrefix: t("ai.mode.geek_status"),
        thinking: theme.text,
      };
    case "evolution":
      return {
        border: "border-emerald-200 dark:border-emerald-700",
        bg: theme.headerBg,
        text: theme.text,
        name: "bg-gradient-to-r from-emerald-600 to-teal-600 bg-clip-text text-transparent",
        avatarBorder: "border-emerald-500/40",
        avatarBg: "bg-emerald-500/15",
        statusText: theme.text,
        statusDot: "bg-emerald-500",
        statusPrefix: t("ai.mode.evolution_status"),
        thinking: theme.text,
      };
    default:
      // Normal mode - use amber theme
      return {
        border: "border-amber-200 dark:border-amber-700",
        bg: PARROT_THEMES.NORMAL.headerBg,
        text: PARROT_THEMES.NORMAL.text,
        name: "",
        avatarBorder: "",
        avatarBg: "",
        statusText: PARROT_THEMES.NORMAL.text,
        statusDot: "bg-amber-500",
        statusPrefix: "",
        thinking: PARROT_THEMES.NORMAL.text,
      };
  }
}

export function ChatHeader({
  isThinking = false,
  className,
  currentCapability = CapabilityType.AUTO,
  capabilityStatus = "idle",
  currentMode = "normal",
  immersiveMode = false,
  onImmersiveModeToggle,
  blocks,
}: ChatHeaderProps) {
  const { t } = useTranslation();
  const assistantName = t("ai.assistant-name");

  // Memoize action description to avoid recalculation during rapid state changes
  const actionDescription = useMemo(
    () => getActionDescription(currentCapability, capabilityStatus, currentMode, t),
    [currentCapability, capabilityStatus, currentMode, t],
  );

  const modeStyle = getModeStyle(currentMode, t);

  return (
    <header
      className={cn(
        "flex items-center justify-between px-4 h-14 shrink-0 transition-colors border-b border-border/80 backdrop-blur-sm",
        modeStyle.border,
        modeStyle.bg,
        className,
      )}
    >
      {/* Left Section */}
      <div className="flex items-center gap-2.5">
        {/* Avatar with animated effects */}
        <AnimatedAvatar
          src="/assistant-avatar.webp"
          alt={assistantName}
          size="sm"
          isThinking={isThinking}
          isTyping={isThinking}
          className={cn("rounded-lg", modeStyle.avatarBorder, modeStyle.avatarBg)}
        />
        <div className="flex flex-col">
          <h1 className={cn("font-semibold text-foreground text-sm leading-tight", modeStyle.name)}>{assistantName}</h1>
          {/* Status */}
          {actionDescription ? (
            <span className={cn("text-xs flex items-center gap-1.5", modeStyle.statusText)}>
              {currentMode !== "normal" && <span className={cn("w-1.5 h-1.5 rounded-full animate-pulse", modeStyle.statusDot)} />}
              {actionDescription}
            </span>
          ) : (
            <span className={cn("text-xs text-muted-foreground", modeStyle.name, currentMode !== "normal" && "font-medium")}>
              {modeStyle.statusPrefix || t("ai.ready")}
            </span>
          )}
        </div>
      </div>

      {/* Right Section - Session Stats + Immersive Toggle + Thinking indicator */}
      <div className="flex items-center gap-2">
        {/* Session Stats - PC only */}
        <HeaderSessionStats blocks={blocks} mode={currentMode} />

        {/* Immersive Mode Toggle - Desktop only */}
        {onImmersiveModeToggle && (
          <button
            onClick={() => onImmersiveModeToggle(!immersiveMode)}
            className={cn(
              "hidden lg:flex items-center justify-center w-8 h-8 rounded-lg transition-all",
              "text-muted-foreground hover:text-foreground hover:bg-muted",
              immersiveMode && "text-primary bg-primary/10",
            )}
            title={immersiveMode ? t("ai.exit-immersive") : t("ai.enter-immersive")}
          >
            {immersiveMode ? <Minimize2 className="w-4 h-4" /> : <Maximize2 className="w-4 h-4" />}
          </button>
        )}
        {isThinking && (
          <div className="flex items-center gap-1.5 text-sm">
            <Sparkles className={cn("w-4 h-4 animate-pulse", modeStyle.thinking)} />
          </div>
        )}
      </div>
    </header>
  );
}
