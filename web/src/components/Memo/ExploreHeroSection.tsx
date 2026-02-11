/**
 * ExploreHeroSection - 探索页首屏
 *
 * 设计哲学：「禅意智识」
 * - 呼吸感：呼应 logo 的 gentle breathe 动效
 * - 留白：东方美学的空灵意境
 * - 层次：信息渐进式呈现
 * - 探索：发现公开内容的视觉隐喻（星光、扩散）
 *
 * 交互：
 * - 统计数据以禅意方式呈现
 * - 分类入口卡片悬浮效果
 *
 * 设计规范：
 * - 间距：使用 --spacing-* 变量系统 (xs:4px, sm:8px, md:16px, lg:24px, xl:32px)
 * - 圆角：使用 --radius-* 变量系统 (sm:6px, md:8px, lg:12px, xl:16px)
 * - 字体：base:16px, sm:14px, lg:18px, xl:20px, 2xl:24px, 3xl:30px
 * - 行高：snug:1.375, normal:1.5, relaxed:1.625
 */

import { Compass, Eye, FileText, Globe, Sparkles, Users } from "lucide-react";
import { memo, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { useNavigate } from "react-router-dom";
import { cn } from "@/lib/utils";

export interface ExploreHeroSectionProps {
  className?: string;
  totalMemos?: number;
  totalUsers?: number;
}

const STAGGER_DELAY = 80;
const BREATH_DURATION = 3000; // 与 logo-breathe-gentle 同步

// 间距常量 - 使用项目规范变量
const SPACING = {
  section: "py-12 sm:py-16", // 区块垂直间距
  inner: "space-y-4", // 内部元素间距
  compact: "gap-2 sm:gap-3", // 紧凑间距
} as const;

/**
 * 快捷筛选卡片 - 禅意风格
 */
interface QuickFilterCardProps {
  icon: React.ComponentType<{ className?: string }>;
  label: string;
  description: string;
  href: string;
  delay: number;
  color: "primary" | "secondary" | "accent";
}

const QuickFilterCard = memo(function QuickFilterCard({ icon: Icon, label, description, href, delay, color }: QuickFilterCardProps) {
  const navigate = useNavigate();

  const colorStyles = {
    primary: "group-hover:border-primary/40 group-hover:bg-primary/5",
    secondary: "group-hover:border-muted-foreground/20 group-hover:bg-muted/50",
    accent: "group-hover:border-accent/40 group-hover:bg-accent/5",
  };

  const iconColors = {
    primary: "text-primary",
    secondary: "text-muted-foreground",
    accent: "text-accent",
  };

  return (
    <button
      onClick={() => navigate(href)}
      className={cn(
        "group relative overflow-hidden rounded-xl border border-border/50 bg-background/50 p-4 sm:p-5",
        "text-left transition-all duration-300 hover:shadow-md",
        "active:scale-[0.98]",
        colorStyles[color],
      )}
      style={{ animationDelay: `${delay * 50}ms` }}
    >
      {/* Subtle background pattern */}
      <div className="absolute inset-0 opacity-0 transition-opacity duration-300 group-hover:opacity-100">
        <div
          className={cn(
            "absolute inset-0 bg-gradient-to-br from-transparent to-transparent",
            color === "primary" && "from-primary/5",
            color === "secondary" && "from-muted-foreground/5",
            color === "accent" && "from-accent/5",
          )}
        />
      </div>

      {/* Content */}
      <div className="relative">
        {/* Icon with glow */}
        <div
          className={cn(
            "flex h-10 w-10 sm:h-11 sm:w-11 items-center justify-center rounded-xl mb-3",
            "bg-muted/50 transition-all duration-300",
            "group-hover:scale-110",
          )}
        >
          <Icon className={cn("h-5 w-5 sm:h-5 sm:w-5", iconColors[color])} />
        </div>

        {/* Title and description */}
        <h3 className="text-sm font-semibold text-foreground/90 mb-1">{label}</h3>
        <p className="text-xs text-muted-foreground leading-relaxed">{description}</p>
      </div>
    </button>
  );
});

/**
 * 探索页首屏组件
 */
export const ExploreHeroSection = memo(function ExploreHeroSection({ className, totalMemos = 0, totalUsers = 0 }: ExploreHeroSectionProps) {
  const { t } = useTranslation();
  const [hasAnimated, setHasAnimated] = useState(false);

  const containerRef = useRef<HTMLDivElement>(null);

  // Entry animation
  useEffect(() => {
    if (!hasAnimated) {
      setHasAnimated(true);
    }
  }, [hasAnimated]);

  // 格式化数字
  const formatNumber = (num: number): string => {
    if (num >= 10000) return `${(num / 10000).toFixed(1)}w`;
    if (num >= 1000) return `${(num / 1000).toFixed(1)}k`;
    return String(num);
  };

  // 分类入口数据
  const categories = useMemo(
    () => [
      {
        icon: Globe,
        label: t("explore.categories.all"),
        description: t("explore.categories.all_desc"),
        href: "/explore",
        color: "primary" as const,
      },
      {
        icon: Users,
        label: t("explore.categories.users"),
        description: t("explore.categories.users_desc"),
        href: "/explore/users",
        color: "secondary" as const,
      },
      {
        icon: Eye,
        label: t("explore.categories.trending"),
        description: t("explore.categories.trending_desc"),
        href: "/explore?trending",
        color: "accent" as const,
      },
    ],
    [t],
  );

  return (
    <div className={cn("relative pb-8", className)} ref={containerRef}>
      {/* Ambient background - subtle "exploration field" */}
      <div className="pointer-events-none absolute inset-0 -z-10 overflow-hidden">
        {/* Primary ambient glow - 右侧 */}
        <div
          className="absolute -right-32 -top-32 h-96 w-96 rounded-full bg-primary/2 blur-3xl"
          style={{ animation: `breathe ${BREATH_DURATION}ms ease-in-out infinite` }}
        />
        {/* Secondary ambient glow - 左侧 */}
        <div
          className="absolute -left-24 top-32 h-64 w-64 rounded-full bg-accent/2 blur-3xl"
          style={{
            animation: `breathe ${BREATH_DURATION}ms ease-in-out infinite`,
            animationDelay: `${BREATH_DURATION / 2}ms`,
          }}
        />
        {/* Stars effect - 随机分布的小光点 */}
        <div className="absolute inset-0">
          {Array.from({ length: 6 }).map((_, i) => (
            <div
              key={i}
              className="absolute h-1 w-1 rounded-full bg-primary/30"
              style={{
                left: `${10 + i * 18}%`,
                top: `${15 + (i % 3) * 25}%`,
                animation: `twinkle ${2000 + i * 300}ms ease-in-out infinite`,
                animationDelay: `${i * 200}ms`,
              }}
            />
          ))}
        </div>
      </div>

      {/* Main content - 使用统一间距 */}
      <div className={SPACING.inner}>
        {/* Page header - 禅意风格 */}
        <div
          className={cn("transition-all duration-1000 ease-out", hasAnimated ? "opacity-100 translate-y-0" : "opacity-0 -translate-y-4")}
        >
          {/* Date with subtle accent */}
          <div className="mb-4 flex items-center gap-3">
            <div className="h-px flex-1 bg-gradient-to-r from-transparent to-border" />
            <div className="flex items-center gap-2 text-xs font-medium tracking-widest text-muted-foreground">
              <Compass className="h-3.5 w-3.5 text-primary" />
              <span>{t("explore.title")}</span>
            </div>
            <div className="h-px flex-1 bg-gradient-to-l from-transparent to-border" />
          </div>

          {/* Main title - 响应式字体大小 */}
          <h1 className="text-center font-semibold text-2xl sm:text-3xl lg:text-4xl xl:text-5xl text-foreground leading-snug">
            {t("explore.hero.title_prefix")} <span className="text-primary">{t("explore.hero.title_highlight")}</span>
          </h1>

          {/* Subtitle */}
          <p className="mx-auto max-w-[28rem] text-center text-sm text-muted-foreground transition-opacity duration-300">
            {t("explore.hero.subtitle")}
          </p>
        </div>

        {/* Stats - minimal zen-style cards */}
        <div
          className={cn(
            "grid grid-cols-3 gap-3 sm:gap-4 transition-all duration-1000 ease-out",
            hasAnimated ? "opacity-100 translate-y-0" : "opacity-0 translate-y-4",
          )}
          style={{ transitionDelay: hasAnimated ? `${STAGGER_DELAY}ms` : "0ms" }}
        >
          <ZenStatCard icon={FileText} label={t("explore.stats.memos")} value={formatNumber(totalMemos)} delay={0} />
          <ZenStatCard icon={Users} label={t("explore.stats.users")} value={formatNumber(totalUsers)} delay={1} />
          <ZenStatCard icon={Sparkles} label={t("explore.stats.growing")} value="—" delay={2} />
        </div>

        {/* Quick filter categories - 渐进展示 */}
        <div
          className={cn(
            "grid grid-cols-1 sm:grid-cols-3 gap-3 sm:gap-4 pt-4",
            "transition-all duration-1000 ease-out",
            hasAnimated ? "opacity-100 translate-y-0" : "opacity-0 translate-y-4",
          )}
          style={{ transitionDelay: hasAnimated ? `${STAGGER_DELAY * 2}ms` : "0ms" }}
        >
          {categories.map((category, index) => (
            <QuickFilterCard key={category.label} {...category} delay={index} />
          ))}
        </div>
      </div>
    </div>
  );
});

ExploreHeroSection.displayName = "ExploreHeroSection";

/* ============================================================
   ZenStatCard - Minimal stat card with subtle animation
   设计规范：统一间距、圆角、字体大小
   ============================================================ */

interface ZenStatCardProps {
  icon: React.ComponentType<{ className?: string }>;
  label: string;
  value: number | string;
  delay: number;
}

function ZenStatCard({ icon: Icon, label, value, delay }: ZenStatCardProps) {
  return (
    <div
      className="group relative overflow-hidden rounded-xl border border-border/50 bg-background/50 p-4 sm:p-5 transition-all duration-300 hover:border-primary/30 hover:bg-background"
      style={{ animationDelay: `${delay * 50}ms` }}
    >
      {/* Subtle background pattern */}
      <div className="absolute inset-0 opacity-0 transition-opacity duration-300 group-hover:opacity-100">
        <div className="absolute inset-0 bg-gradient-to-br from-primary/5 to-transparent" />
      </div>

      {/* Content */}
      <div className="relative flex items-center gap-3 sm:gap-4">
        {/* Icon with glow */}
        <div
          className={cn(
            "flex h-10 w-10 sm:h-11 sm:w-11 items-center justify-center rounded-xl",
            "bg-muted/50 transition-all duration-300",
            "group-hover:bg-primary/10 group-hover:scale-110",
          )}
        >
          <Icon className="h-4 w-4 sm:h-5 sm:w-5 text-muted-foreground transition-colors duration-300 group-hover:text-primary" />
        </div>

        {/* Value and label */}
        <div className="min-w-0 flex-1">
          <div className="text-xl sm:text-2xl font-semibold tabular-nums text-foreground">{String(value)}</div>
          <div className="mt-0.5 text-xs font-medium uppercase tracking-wide text-muted-foreground">{label}</div>
        </div>
      </div>
    </div>
  );
}

/* ============================================================
   Global animations (syncs with logo)
   ============================================================ */

// Inject animations if not exists
if (typeof document !== "undefined") {
  const styleId = "explore-hero-animations";
  if (!document.getElementById(styleId)) {
    const style = document.createElement("style");
    style.id = styleId;
    style.textContent = `
      @keyframes breathe {
        0%, 100% { opacity: 0.3; transform: scale(1); }
        50% { opacity: 0.6; transform: scale(1.05); }
      }

      @keyframes twinkle {
        0%, 100% { opacity: 0.1; transform: scale(0.8); }
        50% { opacity: 0.6; transform: scale(1.2); }
      }
    `;
    document.head.appendChild(style);
  }
}
