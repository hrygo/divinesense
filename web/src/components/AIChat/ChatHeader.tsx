import { Maximize2, Minimize2, Sparkles } from "lucide-react";
import { useTranslation } from "react-i18next";
import { cn } from "@/lib/utils";
import { CapabilityStatus, CapabilityType } from "@/types/capability";
import { EvolutionModeToggle } from "./EvolutionModeToggle";
import { GeekModeToggle } from "./GeekModeToggle";

interface ChatHeaderProps {
  isThinking?: boolean;
  className?: string;
  currentCapability?: CapabilityType;
  capabilityStatus?: CapabilityStatus;
  geekMode?: boolean;
  onGeekModeToggle?: (enabled: boolean) => void;
  evolutionMode?: boolean;
  onEvolutionModeToggle?: (enabled: boolean) => void;
  immersiveMode?: boolean;
  onImmersiveModeToggle?: (enabled: boolean) => void;
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
    return t("ai.thinking");
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

export function ChatHeader({
  isThinking = false,
  className,
  currentCapability = CapabilityType.AUTO,
  capabilityStatus = "idle",
  geekMode = false,
  onGeekModeToggle,
  evolutionMode = false,
  onEvolutionModeToggle,
  immersiveMode = false,
  onImmersiveModeToggle,
}: ChatHeaderProps) {
  const { t } = useTranslation();
  const assistantName = t("ai.assistant-name");
  const actionDescription = getActionDescription(currentCapability, capabilityStatus, t);

  return (
    <header
      className={cn(
        "flex items-center justify-between px-4 h-14 shrink-0 transition-all",
        "border-b border-border/80",
        "bg-background/80 backdrop-blur-sm",
        // Geek mode: subtle green border and background
        geekMode && "border-green-500/20 bg-green-50/50 dark:bg-green-950/20",
        // Evolution mode: purple border and background
        evolutionMode && "border-purple-500/30 bg-purple-50/50 dark:bg-purple-950/20",
        className,
      )}
    >
      {/* Left Section */}
      <div className="flex items-center gap-2.5">
        {/* Avatar with mode-specific border */}
        <div
          className={cn(
            "w-9 h-9 flex items-center justify-center rounded-lg transition-all",
            geekMode && "border border-green-500/30 bg-green-500/10",
            evolutionMode && "border border-purple-500/40 bg-purple-500/15",
          )}
        >
          <img src="/assistant-avatar.webp" alt={assistantName} className="h-9 w-auto object-contain" />
        </div>
        <div className="flex flex-col">
          <h1
            className={cn(
              "font-semibold text-foreground text-sm leading-tight",
              geekMode && "font-mono",
              evolutionMode && "bg-gradient-to-r from-purple-600 to-blue-600 bg-clip-text text-transparent",
            )}
          >
            {assistantName}
          </h1>
          {/* Status */}
          {actionDescription ? (
            <span
              className={cn(
                "text-xs flex items-center gap-1.5",
                geekMode ? "text-green-600 dark:text-green-400" : "",
                evolutionMode ? "text-purple-600 dark:text-purple-400" : "",
              )}
            >
              {(geekMode || evolutionMode) && (
                <span
                  className={cn("w-1.5 h-1.5 rounded-full animate-pulse", geekMode && "bg-green-500", evolutionMode && "bg-purple-500")}
                />
              )}
              {actionDescription}
            </span>
          ) : (
            <span className={cn("text-xs text-muted-foreground", geekMode && "font-mono", evolutionMode && "font-medium")}>
              {geekMode ? "$ ready" : evolutionMode ? "⚡ 进化就绪" : t("ai.ready")}
            </span>
          )}
        </div>
      </div>

      {/* Right Section - Immersive Mode Toggle + Evolution Mode Toggle + Geek Mode Toggle + Thinking indicator */}
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
        <EvolutionModeToggle enabled={evolutionMode} onToggle={onEvolutionModeToggle ?? (() => {})} variant="header" />
        <GeekModeToggle enabled={geekMode} onToggle={onGeekModeToggle ?? (() => {})} variant="header" />
        {isThinking && (
          <div className="flex items-center gap-1.5 text-sm">
            <Sparkles
              className={cn(
                "w-4 h-4 animate-pulse",
                geekMode ? "text-green-600 dark:text-green-400" : "",
                evolutionMode ? "text-purple-600 dark:text-purple-400" : "text-primary",
              )}
            />
          </div>
        )}
      </div>
    </header>
  );
}
