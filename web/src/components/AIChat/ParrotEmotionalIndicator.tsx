import { cn } from "@/lib/utils";
import { EmotionalState, PARROT_THEMES, ParrotAgentType } from "@/types/parrot";

interface ParrotEmotionalIndicatorProps {
  active?: boolean;
  parrotId?: ParrotAgentType;
  mood?: EmotionalState;
  soundEffect?: string;
  size?: "sm" | "md" | "lg";
  showText?: boolean;
  className?: string;
}

/**
 * Mood to emoji/icon mapping
 * 情感状态到表情/图标的映射
 */
const MOOD_ICONS: Record<EmotionalState, string> = {
  focused: "🎯",
  curious: "🔍",
  excited: "✨",
  thoughtful: "🤔",
  confused: "❓",
  happy: "😊",
  delighted: "🎉",
  helpful: "💡",
  alert: "⚠️",
};

/**
 * Size class mappings
 */
const SIZE_CLASSES = {
  sm: "w-5 h-5 text-sm",
  md: "w-6 h-6 text-base",
  lg: "w-8 h-8 text-lg",
} as const;

const TEXT_SIZE_CLASSES = {
  sm: "text-[11px]",
  md: "text-xs",
  lg: "text-sm",
} as const;

/**
 * Extract base color class from theme
 * 从主题中提取基础颜色类
 */
function getBaseColorClass(colorClass: string): string {
  // Handle multiple space-separated classes
  const classes = colorClass.split(" ");
  for (const cls of classes) {
    if (cls.startsWith("text-") && !cls.includes("dark:")) {
      return cls;
    }
  }
  return "text-zinc-500";
}

/**
 * Get glow background class from icon background
 */
function getGlowBgClass(iconBg: string): string {
  // Extract the base color without variants
  const baseColor = iconBg.split(" ")[0];
  return baseColor.replace("bg-", "bg-").replace("/100", "").replace("/50", "");
}

/**
 * Parrot Emotional Indicator
 * 鹦鹉情感指示器 - 显示鹦鹉当前的情感状态
 *
 * 设计原则：
 * - 轻量动画，使用 Tailwind CSS
 * - 无外部动画库依赖
 * - 简洁的视觉设计
 */
export function ParrotEmotionalIndicator({
  active = true,
  parrotId,
  mood = "focused",
  soundEffect,
  size = "md",
  showText = false,
  className,
}: ParrotEmotionalIndicatorProps) {
  if (!active) return null;

  const theme = parrotId ? PARROT_THEMES[parrotId] || PARROT_THEMES.AMAZING : PARROT_THEMES.AMAZING;

  const baseColor = getBaseColorClass(theme.iconText);
  const glowBg = getGlowBgClass(theme.iconBg);

  return (
    <div className={cn("inline-flex items-center gap-1.5", className)}>
      {/* Mood Icon */}
      <div
        className={cn(
          "relative flex items-center justify-center rounded-full transition-all duration-300",
          SIZE_CLASSES[size],
          theme.iconBg,
          mood === "excited" || mood === "delighted" ? "scale-110" : "scale-100",
        )}
      >
        <span
          className={cn(
            "transition-transform duration-500",
            mood === "confused" && "rotate-12",
            mood === "thoughtful" && "-rotate-12",
            (mood === "excited" || mood === "delighted") && "scale-125",
          )}
        >
          {MOOD_ICONS[mood]}
        </span>

        {/* Subtle glow effect for positive moods */}
        {(mood === "happy" || mood === "delighted" || mood === "excited") && (
          <span className={cn("absolute inset-0 rounded-full opacity-30 animate-ping", glowBg)} />
        )}
      </div>

      {/* Sound effect text (optional) */}
      {showText && soundEffect && (
        <span
          className={cn("font-medium transition-all duration-300", TEXT_SIZE_CLASSES[size], baseColor, mood === "excited" && "font-bold")}
        >
          {soundEffect}
        </span>
      )}
    </div>
  );
}

/**
 * Compact mood badge - for inline display
 * 紧凑情感徽章 - 用于内联显示
 */
interface MoodBadgeProps {
  mood: EmotionalState;
  parrotId?: ParrotAgentType;
  className?: string;
}

export function MoodBadge({ mood, parrotId, className }: MoodBadgeProps) {
  const theme = parrotId ? PARROT_THEMES[parrotId] || PARROT_THEMES.AMAZING : PARROT_THEMES.AMAZING;

  return (
    <span
      className={cn(
        "inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-[11px] font-medium",
        theme.iconBg,
        theme.iconText,
        "border border-transparent",
        className,
      )}
    >
      <span>{MOOD_ICONS[mood]}</span>
      <span className="capitalize">{mood}</span>
    </span>
  );
}

/**
 * Sound effect bubble - animated popup for parrot sounds
 * 拟声词气泡 - 鹦鹉声音的动画弹出效果
 */
interface SoundBubbleProps {
  sound: string;
  parrotId?: ParrotAgentType;
  active?: boolean;
  className?: string;
}

export function SoundBubble({ sound, parrotId, active = true, className }: SoundBubbleProps) {
  if (!active || !sound) return null;

  const theme = parrotId ? PARROT_THEMES[parrotId] || PARROT_THEMES.AMAZING : PARROT_THEMES.AMAZING;

  return (
    <span
      className={cn(
        "inline-flex items-center justify-center px-2 py-1 rounded-lg text-xs font-medium",
        "animate-in fade-in slide-in-from-bottom-2 duration-300",
        "transition-all duration-300 hover:scale-105",
        theme.bubbleBg,
        theme.bubbleBorder,
        theme.textSecondary,
        "shadow-sm",
        className,
      )}
    >
      {sound}
    </span>
  );
}

export default ParrotEmotionalIndicator;
