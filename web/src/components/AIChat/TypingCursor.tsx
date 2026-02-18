import { useTranslation } from "react-i18next";
import { cn } from "@/lib/utils";
import { PARROT_THEMES, ParrotAgentType } from "@/types/parrot";
import "./animations.css";

interface TypingCursorProps {
  active?: boolean;
  parrotId?: ParrotAgentType;
  variant?: "cursor" | "dots" | "wave" | "parrot";
}

/**
 * Parrot-specific animation name mapping
 * 鹦鹉特定的动画名称映射
 */
const PARROT_ANIMATIONS: Record<string, string> = {
  [ParrotAgentType.MEMO]: "memoFloat",
  [ParrotAgentType.SCHEDULE]: "scheduleTick",
  [ParrotAgentType.GENERAL]: "generalSpin",
};

/**
 * AI Native Typing Indicator
 * Creates an intelligent, animated typing indicator with multiple variations
 */
const TypingCursor = ({ active = true, parrotId, variant = "dots" }: TypingCursorProps) => {
  const theme = parrotId ? PARROT_THEMES[parrotId] || PARROT_THEMES.GENERAL : PARROT_THEMES.GENERAL;

  if (!active) return null;

  if (variant === "cursor") {
    return (
      <span className="inline-flex items-center ml-1">
        <span
          className={cn("inline-block w-0.5 h-4 rounded-full animate-pulse", theme.iconText.replace("text-", "bg-"))}
          style={{ animationDuration: "800ms" }}
        />
      </span>
    );
  }

  if (variant === "wave") {
    return (
      <span className="inline-flex items-center gap-0.5 ml-2">
        {[0, 1, 2].map((i) => (
          <span
            key={i}
            className={cn("w-1 h-3 rounded-full", theme.iconBg)}
            style={{
              animation: "wave 1.2s ease-in-out infinite",
              animationDelay: `${i * 0.15}s`,
            }}
          />
        ))}
      </span>
    );
  }

  // Parrot variant - uses parrot-specific animations
  if (variant === "parrot") {
    const animationName = PARROT_ANIMATIONS[parrotId || ParrotAgentType.GENERAL];
    return (
      <span className="inline-flex items-center gap-1 ml-2">
        {[0, 1, 2].map((i) => (
          <span
            key={i}
            className={cn("w-2 h-2 rounded-full inline-block", theme.iconBg)}
            style={{
              animation: `${animationName} 1.2s ease-in-out infinite`,
              animationDelay: `${i * 0.2}s`,
            }}
          />
        ))}
      </span>
    );
  }

  // Default: Intelligent dots with glow
  return (
    <span className="inline-flex items-center gap-1 ml-2">
      {[0, 1, 2].map((i) => (
        <span
          key={i}
          className={cn("w-2 h-2 rounded-full", theme.iconBg, "animate-pulse")}
          style={{
            animationDuration: "1s",
            animationDelay: `${i * 0.2}s`,
          }}
        />
      ))}
    </span>
  );
};

/**
 * AI Thinking Indicator - Shows AI is "thinking" with a more sophisticated animation
 */
interface AIThinkingIndicatorProps {
  active?: boolean;
  parrotId?: ParrotAgentType;
  size?: "sm" | "md" | "lg";
}

export function AIThinkingIndicator({ active = true, parrotId, size = "md" }: AIThinkingIndicatorProps) {
  if (!active) return null;

  const theme = parrotId ? PARROT_THEMES[parrotId] || PARROT_THEMES.GENERAL : PARROT_THEMES.GENERAL;

  const sizeClasses = {
    sm: "w-4 h-4",
    md: "w-6 h-6",
    lg: "w-8 h-8",
  };

  return (
    <div className="inline-flex items-center justify-center">
      <div className={cn("relative", sizeClasses[size])}>
        {/* Outer ring */}
        <div
          className={cn("absolute inset-0 rounded-full opacity-25", theme.iconBg)}
          style={{
            animation: "ping 1.5s cubic-bezier(0, 0, 0.2, 1) infinite",
          }}
        />
        {/* Inner dot */}
        <div className={cn("absolute inset-0 rounded-full flex items-center justify-center", theme.iconBg)}>
          <span className="text-white text-xs">✨</span>
        </div>
      </div>
    </div>
  );
}

/**
 * Streaming Indicator - Shows content is being streamed in real-time
 */
interface StreamingIndicatorProps {
  active?: boolean;
  parrotId?: ParrotAgentType;
}

export function StreamingIndicator({ active = true, parrotId }: StreamingIndicatorProps) {
  const { t } = useTranslation();
  if (!active) return null;

  const theme = parrotId ? PARROT_THEMES[parrotId] || PARROT_THEMES.GENERAL : PARROT_THEMES.GENERAL;

  return (
    <span className="inline-flex items-center gap-1 ml-2">
      <span className={cn("w-1.5 h-1.5 rounded-full", theme.iconBg, "animate-pulse")} />
      <span className="text-xs text-muted-foreground">{t("ai.typing_indicator")}</span>
    </span>
  );
}

export default TypingCursor;
