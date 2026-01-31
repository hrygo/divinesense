import { Maximize2, Minimize2, Sparkles } from "lucide-react";
import { useTranslation } from "react-i18next";
import { AnimatedAvatar } from "@/components/AIChat/AnimatedAvatar";
import { cn } from "@/lib/utils";
import type { AIMode } from "@/types/aichat";
import { CapabilityStatus, CapabilityType } from "@/types/capability";
import { ModeCycleButton } from "./ModeCycleButton";

interface ChatHeaderProps {
  isThinking?: boolean;
  className?: string;
  currentCapability?: CapabilityType;
  capabilityStatus?: CapabilityStatus;
  currentMode?: AIMode;
  onModeChange?: (mode: AIMode) => void;
  immersiveMode?: boolean;
  onImmersiveModeToggle?: (enabled: boolean) => void;
  isAdmin?: boolean;
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
function getActionDescription(capability: CapabilityType, status: CapabilityStatus, t: (key: string) => string): string | null {
  if (status === "idle") return null;

  if (status === "thinking") {
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
 * 根据当前模式获取样式配置
 */
function getModeStyle(mode: AIMode, t: (key: string) => string) {
  switch (mode) {
    case "geek":
      return {
        border: "border-green-500/20",
        bg: "bg-green-50/50 dark:bg-green-950/20",
        text: "text-green-600 dark:text-green-400",
        name: "font-mono",
        avatarBorder: "border-green-500/30",
        avatarBg: "bg-green-500/10",
        statusText: "text-green-600 dark:text-green-400",
        statusDot: "bg-green-500",
        statusPrefix: t("ai.mode.geek_status"),
        thinking: "text-green-600 dark:text-green-400",
      };
    case "evolution":
      return {
        border: "border-purple-500/30",
        bg: "bg-purple-50/50 dark:bg-purple-950/20",
        text: "text-purple-600 dark:text-purple-400",
        name: "bg-gradient-to-r from-purple-600 to-blue-600 bg-clip-text text-transparent",
        avatarBorder: "border-purple-500/40",
        avatarBg: "bg-purple-500/15",
        statusText: "text-purple-600 dark:text-purple-400",
        statusDot: "bg-purple-500",
        statusPrefix: t("ai.mode.evolution_status"),
        thinking: "text-purple-600 dark:text-purple-400",
      };
    default:
      return {
        border: "",
        bg: "",
        text: "",
        name: "",
        avatarBorder: "",
        avatarBg: "",
        statusText: "",
        statusDot: "",
        statusPrefix: "",
        thinking: "text-primary",
      };
  }
}

export function ChatHeader({
  isThinking = false,
  className,
  currentCapability = CapabilityType.AUTO,
  capabilityStatus = "idle",
  currentMode = "normal",
  onModeChange,
  immersiveMode = false,
  onImmersiveModeToggle,
  isAdmin = true,
}: ChatHeaderProps) {
  const { t } = useTranslation();
  const assistantName = t("ai.assistant-name");
  const actionDescription = getActionDescription(currentCapability, capabilityStatus, t);
  const modeStyle = getModeStyle(currentMode, t);

  return (
    <header
      className={cn(
        "flex items-center justify-between px-4 h-14 shrink-0 transition-all",
        "border-b border-border/80",
        "bg-background/80 backdrop-blur-sm",
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

      {/* Right Section - Immersive Mode Toggle + Mode Cycle Button + Thinking indicator */}
      <div className="flex items-center gap-2">
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
        <ModeCycleButton currentMode={currentMode} onModeChange={onModeChange ?? (() => {})} variant="header" isAdmin={isAdmin} />
        {isThinking && (
          <div className="flex items-center gap-1.5 text-sm">
            <Sparkles className={cn("w-4 h-4 animate-pulse", modeStyle.thinking)} />
          </div>
        )}
      </div>
    </header>
  );
}
