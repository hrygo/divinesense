import { memo, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { AnimatedAvatar } from "@/components/AIChat/AnimatedAvatar";
import { cn } from "@/lib/utils";
import type { AIMode } from "@/types/aichat";

interface PartnerGreetingProps {
  userName?: string;
  recentMemoCount?: number;
  upcomingScheduleCount?: number;
  conversationCount?: number;
  onSendMessage?: (message: string) => void;
  onSendComplete?: () => void;
  className?: string;
  currentMode?: AIMode;
}

/**
 * 时间段类型
 */
type TimeOfDay = "morning" | "afternoon" | "evening" | "night";

/**
 * 示例问题分类
 */
type PromptCategory = "memo" | "schedule" | "create" | "amazing";

/**
 * 获取时间段相关配置
 */
function getTimeConfig(): {
  timeOfDay: TimeOfDay;
  greetingKey: string;
  hintKey: string;
} {
  const hour = new Date().getHours();

  if (hour >= 5 && hour < 9) {
    return {
      timeOfDay: "morning",
      greetingKey: "ai.parrot.partner.greeting-early-morning",
      hintKey: "ai.parrot.partner.hint-early-morning",
    };
  }
  if (hour >= 9 && hour < 12) {
    return {
      timeOfDay: "morning",
      greetingKey: "ai.parrot.partner.greeting-morning",
      hintKey: "ai.parrot.partner.hint-morning",
    };
  }
  if (hour >= 12 && hour < 14) {
    return {
      timeOfDay: "afternoon",
      greetingKey: "ai.parrot.partner.greeting-noon",
      hintKey: "ai.parrot.partner.hint-noon",
    };
  }
  if (hour >= 14 && hour < 18) {
    return {
      timeOfDay: "afternoon",
      greetingKey: "ai.parrot.partner.greeting-afternoon",
      hintKey: "ai.parrot.partner.hint-afternoon",
    };
  }
  if (hour >= 18 && hour < 21) {
    return {
      timeOfDay: "evening",
      greetingKey: "ai.parrot.partner.greeting-evening",
      hintKey: "ai.parrot.partner.hint-evening",
    };
  }
  return {
    timeOfDay: "night",
    greetingKey: "ai.parrot.partner.greeting-night",
    hintKey: "ai.parrot.partner.hint-night",
  };
}

/**
 * 示例问题接口
 */
interface SuggestedPrompt {
  icon: string;
  category: PromptCategory;
  promptKey: string;
  prompt: string;
}

/**
 * 获取极客模式专属配置
 */
function getGeekModePrompts(t: (key: string) => string): SuggestedPrompt[] {
  return [
    {
      icon: "⚡",
      category: "amazing",
      promptKey: "ai.parrot.geek.prompt-refactor",
      prompt: t("ai.parrot.geek.prompt-refactor"),
    },
    {
      icon: "🧪",
      category: "create",
      promptKey: "ai.parrot.geek.prompt-test",
      prompt: t("ai.parrot.geek.prompt-test"),
    },
    {
      icon: "🐛",
      category: "memo",
      promptKey: "ai.parrot.geek.prompt-fix",
      prompt: t("ai.parrot.geek.prompt-fix"),
    },
    {
      icon: "📊",
      category: "amazing",
      promptKey: "ai.parrot.geek.prompt-analyze",
      prompt: t("ai.parrot.geek.prompt-analyze"),
    },
  ];
}

/**
 * 获取进化模式专属配置
 */
function getEvolutionModePrompts(t: (key: string) => string): SuggestedPrompt[] {
  return [
    {
      icon: "🚀",
      category: "create",
      promptKey: "ai.parrot.evolution.prompt-feature",
      prompt: t("ai.parrot.evolution.prompt-feature"),
    },
    {
      icon: "⚡",
      category: "amazing",
      promptKey: "ai.parrot.evolution.prompt-optimize",
      prompt: t("ai.parrot.evolution.prompt-optimize"),
    },
    {
      icon: "🐛",
      category: "memo",
      promptKey: "ai.parrot.evolution.prompt-fix",
      prompt: t("ai.parrot.evolution.prompt-fix"),
    },
    {
      icon: "🔧",
      category: "schedule",
      promptKey: "ai.parrot.evolution.prompt-refactor",
      prompt: t("ai.parrot.evolution.prompt-refactor"),
    },
  ];
}

/**
 * 获取时间段特定的示例问题
 */
function getTimeSpecificPrompts(t: (key: string) => string, timeOfDay: TimeOfDay): SuggestedPrompt[] {
  // 早上（5-12点）：侧重今日计划
  if (timeOfDay === "morning") {
    return [
      {
        icon: "📋",
        category: "schedule",
        promptKey: "ai.parrot.partner.prompt-today-schedule",
        prompt: t("ai.parrot.partner.prompt-today-schedule"),
      },
      {
        icon: "📝",
        category: "memo",
        promptKey: "ai.parrot.partner.prompt-recent-memos",
        prompt: t("ai.parrot.partner.prompt-recent-memos"),
      },
      {
        icon: "➕",
        category: "create",
        promptKey: "ai.parrot.partner.prompt-create-meeting",
        prompt: t("ai.parrot.partner.prompt-create-meeting"),
      },
      {
        icon: "📊",
        category: "amazing",
        promptKey: "ai.parrot.partner.prompt-today-overview",
        prompt: t("ai.parrot.partner.prompt-today-overview"),
      },
    ];
  }

  // 下午（12-18点）：侧重查询和创建
  if (timeOfDay === "afternoon") {
    return [
      {
        icon: "🔍",
        category: "memo",
        promptKey: "ai.parrot.partner.prompt-search-memo",
        prompt: t("ai.parrot.partner.prompt-search-memo"),
      },
      {
        icon: "⏰",
        category: "schedule",
        promptKey: "ai.parrot.partner.prompt-afternoon-free",
        prompt: t("ai.parrot.partner.prompt-afternoon-free"),
      },
      {
        icon: "📅",
        category: "create",
        promptKey: "ai.parrot.partner.prompt-create-tomorrow",
        prompt: t("ai.parrot.partner.prompt-create-tomorrow"),
      },
      {
        icon: "🔗",
        category: "amazing",
        promptKey: "ai.parrot.partner.prompt-connect-info",
        prompt: t("ai.parrot.partner.prompt-connect-info"),
      },
    ];
  }

  // 晚上（18-21点）：侧重回顾
  if (timeOfDay === "evening") {
    return [
      {
        icon: "📝",
        category: "memo",
        promptKey: "ai.parrot.partner.prompt-today-learned",
        prompt: t("ai.parrot.partner.prompt-today-learned"),
      },
      {
        icon: "📅",
        category: "schedule",
        promptKey: "ai.parrot.partner.prompt-tomorrow-plan",
        prompt: t("ai.parrot.partner.prompt-tomorrow-plan"),
      },
      {
        icon: "✅",
        category: "create",
        promptKey: "ai.parrot.partner.prompt-create-reminder",
        prompt: t("ai.parrot.partner.prompt-create-reminder"),
      },
      {
        icon: "📊",
        category: "amazing",
        promptKey: "ai.parrot.partner.prompt-day-summary",
        prompt: t("ai.parrot.partner.prompt-day-summary"),
      },
    ];
  }

  // 深夜（21-5点）：侧重快速查询
  return [
    {
      icon: "🔍",
      category: "memo",
      promptKey: "ai.parrot.partner.prompt-quick-search",
      prompt: t("ai.parrot.partner.prompt-quick-search"),
    },
    {
      icon: "📅",
      category: "schedule",
      promptKey: "ai.parrot.partner.prompt-tomorrow-check",
      prompt: t("ai.parrot.partner.prompt-tomorrow-check"),
    },
    { icon: "💡", category: "memo", promptKey: "ai.parrot.partner.prompt-find-idea", prompt: t("ai.parrot.partner.prompt-find-idea") },
    {
      icon: "🌟",
      category: "amazing",
      promptKey: "ai.parrot.partner.prompt-week-summary",
      prompt: t("ai.parrot.partner.prompt-week-summary"),
    },
  ];
}

/**
 * 获取默认示例问题（当时间特定问题不可用时）
 */
function getDefaultPrompts(t: (key: string) => string): SuggestedPrompt[] {
  return [
    { icon: "🔍", category: "memo", promptKey: "ai.parrot.partner.prompt-search-memo", prompt: t("ai.parrot.partner.prompt-search-memo") },
    {
      icon: "📅",
      category: "schedule",
      promptKey: "ai.parrot.partner.prompt-today-schedule",
      prompt: t("ai.parrot.partner.prompt-today-schedule"),
    },
    {
      icon: "➕",
      category: "create",
      promptKey: "ai.parrot.partner.prompt-create-meeting",
      prompt: t("ai.parrot.partner.prompt-create-meeting"),
    },
    {
      icon: "📊",
      category: "amazing",
      promptKey: "ai.parrot.partner.prompt-day-summary",
      prompt: t("ai.parrot.partner.prompt-day-summary"),
    },
  ];
}

/**
 * Partner Greeting - 统一入口设计
 *
 * UX/UI 设计原则：
 * - 示例提问根据时间段动态调整，更贴近实际使用场景
 * - 覆盖所有能力类型：笔记查询、日程查询、日程创建、综合分析
 * - 用户无需理解系统内部能力边界，点击即可直接使用
 */
export const PartnerGreeting = memo(function PartnerGreeting({
  onSendMessage,
  onSendComplete,
  recentMemoCount,
  upcomingScheduleCount,
  className,
  currentMode = "normal",
}: PartnerGreetingProps) {
  const { t } = useTranslation();
  const [isSending, setIsSending] = useState(false);
  const timeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  // 模式感知的问候语和提示
  const { greetingText, timeHint } = useMemo(() => {
    if (currentMode === "geek") {
      return {
        greetingText: t("ai.parrot.geek.greeting"),
        timeHint: t("ai.parrot.geek.hint"),
      };
    }
    if (currentMode === "evolution") {
      return {
        greetingText: t("ai.parrot.evolution.greeting"),
        timeHint: t("ai.parrot.evolution.hint"),
      };
    }
    // 普通模式使用时间感知问候
    const timeConfig = getTimeConfig();
    return {
      greetingText: t(timeConfig.greetingKey),
      timeHint: t(timeConfig.hintKey),
    };
  }, [currentMode, t]);

  // 根据模式获取示例问题
  const suggestedPrompts = useMemo(() => {
    if (currentMode === "geek") {
      return getGeekModePrompts(t);
    }
    if (currentMode === "evolution") {
      return getEvolutionModePrompts(t);
    }
    // 普通模式使用时间感知问题
    const timeConfig = getTimeConfig();
    const prompts = getTimeSpecificPrompts(t, timeConfig.timeOfDay);
    const hasMissingTranslation = prompts.some((p) => p.prompt === p.promptKey);
    if (hasMissingTranslation) {
      return getDefaultPrompts(t);
    }
    return prompts;
  }, [currentMode, t]);

  // 获取统计信息文本
  const statsText = useMemo(() => {
    const parts: string[] = [];
    if (recentMemoCount !== undefined && recentMemoCount > 0) {
      parts.push(t("ai.parrot.partner.memo-count", { count: recentMemoCount }));
    }
    if (upcomingScheduleCount !== undefined && upcomingScheduleCount > 0) {
      parts.push(t("ai.parrot.partner.schedule-count", { count: upcomingScheduleCount }));
    }
    return parts.join(" · ");
  }, [recentMemoCount, upcomingScheduleCount, t]);

  const handlePromptClick = (prompt: SuggestedPrompt) => {
    if (isSending) return;
    setIsSending(true);
    onSendMessage?.(prompt.prompt);
    // Clear any existing timeout
    if (timeoutRef.current) {
      clearTimeout(timeoutRef.current);
    }
    const delay = onSendComplete ? 3000 : 500;
    timeoutRef.current = setTimeout(() => setIsSending(false), delay);
  };

  // Cleanup timeout on unmount
  useEffect(() => {
    return () => {
      if (timeoutRef.current) {
        clearTimeout(timeoutRef.current);
      }
    };
  }, []);

  return (
    <div className={cn("flex flex-col items-center justify-center h-full w-full px-6 py-8", className)}>
      {/* 主图标 - 带悬浮动画 */}
      <div className="mb-8 animate-in fade-in zoom-in duration-500">
        <AnimatedAvatar src="/assistant-avatar.webp" alt="AI Agent" size="xl" isThinking={!isSending} />
      </div>

      {/* 问候语区域 */}
      <div className="text-center mb-8">
        <h2 className="text-xl font-semibold text-foreground mb-2">{greetingText}</h2>
        <p className="text-sm text-muted-foreground">{timeHint}</p>
        {statsText && <p className="text-xs text-muted-foreground mt-2">{statsText}</p>}
      </div>

      {/* 示例提问 - 点击直接发送 */}
      <div className="grid grid-cols-2 lg:grid-cols-4 gap-3 lg:gap-4 w-full max-w-3xl lg:max-w-4xl xl:max-w-5xl 2xl:max-w-6xl mb-8">
        {suggestedPrompts.map((item) => (
          <button
            key={item.promptKey}
            disabled={isSending}
            onClick={() => handlePromptClick(item)}
            className={cn(
              "flex flex-row items-center gap-3 p-3 rounded-xl",
              "bg-card",
              "border border-border",
              "hover:border-primary/50",
              "hover:bg-accent",
              "transition-all duration-200",
              "active:scale-95",
              "min-h-[56px]",
              isSending && "opacity-50 cursor-not-allowed active:scale-100",
            )}
            title={item.prompt}
          >
            <span className="text-2xl shrink-0">{item.icon}</span>
            <span className="text-sm font-medium text-foreground text-left leading-tight line-clamp-2">{item.prompt}</span>
          </button>
        ))}
      </div>
    </div>
  );
});

/**
 * 简化版伙伴问候 - 用于对话列表中展示
 */
interface MiniPartnerGreetingProps {
  message?: string;
  className?: string;
}

export const MiniPartnerGreeting = memo(function MiniPartnerGreeting({ message, className }: MiniPartnerGreetingProps) {
  const { t } = useTranslation();
  const timeConfig = getTimeConfig();
  const greetingText = t(timeConfig.greetingKey);

  return (
    <div className={cn("flex items-start gap-3 p-4", className)}>
      <div className="w-9 h-9 md:w-10 md:h-10 rounded-xl bg-primary flex items-center justify-center text-lg shrink-0 shadow-sm">
        <span>🦜</span>
      </div>
      <div className="flex-1 min-w-0">
        <p className="font-medium text-foreground mb-1">{greetingText}</p>
        <p className="text-xs text-muted-foreground line-clamp-2">{message || t("ai.parrot.partner.default-hint")}</p>
      </div>
    </div>
  );
});
