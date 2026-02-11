/**
 * HeroSection - 神识主页首屏
 *
 * 设计哲学：「禅意智识」
 * - 呼吸感：呼应 logo 的 gentle breathe 动效
 * - 留白：东方美学的空灵意境
 * - 层次：信息渐进式呈现
 * - 智识：AI 搜索的视觉隐喻（光晕、扩散）
 *
 * 交互：
 * - 搜索框聚焦时产生「意识场」效果
 * - 结果卡片如思绪般浮现
 * - 统计数据以禅意方式呈现
 *
 * 设计规范：
 * - 间距：使用 --spacing-* 变量系统 (xs:4px, sm:8px, md:16px, lg:24px, xl:32px)
 * - 圆角：使用 --radius-* 变量系统 (sm:6px, md:8px, lg:12px, xl:16px)
 * - 字体：base:16px, sm:14px, lg:18px, xl:20px, 2xl:24px, 3xl:30px
 * - 行高：snug:1.375, normal:1.5, relaxed:1.625
 */

import { Activity, AlertCircle, Calendar, FileText, Loader2, Search, Sparkles, X } from "lucide-react";
import { memo, useEffect, useMemo, useRef, useState } from "react";
import { createPortal } from "react-dom";
import { useTranslation } from "react-i18next";
import MemoFilters from "@/components/MemoFilters";
import { useMemoFilterContext } from "@/contexts/MemoFilterContext";
import { useSemanticSearch } from "@/hooks/useAIQueries";
import useCurrentUser from "@/hooks/useCurrentUser";
import { useFilteredMemoStats } from "@/hooks/useFilteredMemoStats";
import useNavigateTo from "@/hooks/useNavigateTo";
import { cn } from "@/lib/utils";

export interface HeroSectionProps {
  className?: string;
}

const STAGGER_DELAY = 80;
const BREATH_DURATION = 3000; // 与 logo-breathe-gentle 同步

// 间距常量 - 使用项目规范变量
const SPACING = {
  section: "py-12 sm:py-16", // 区块垂直间距
  inner: "space-y-4", // 内部元素间距 (与 MemoList gap-4 一致)
  compact: "gap-2 sm:gap-3", // 紧凑间距
} as const;

export const HeroSection = memo(function HeroSection({ className }: HeroSectionProps) {
  const { t } = useTranslation();
  const currentUser = useCurrentUser();
  const { addFilter } = useMemoFilterContext();
  const navigateTo = useNavigateTo();

  // Search state
  const [searchQuery, setSearchQuery] = useState("");
  const [isSemantic, setIsSemantic] = useState(true);
  const [isFocused, setIsFocused] = useState(false);
  const [hasAnimated, setHasAnimated] = useState(false);
  const [dropdownPosition, setDropdownPosition] = useState({ top: 0, left: 0, width: 0 });

  const containerRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLInputElement>(null);
  const searchContainerRef = useRef<HTMLDivElement>(null);

  // Get memo stats
  const { statistics, tags } = useFilteredMemoStats({
    userName: currentUser?.name,
  });

  // AI Semantic Search
  const {
    data: semanticResults,
    isLoading,
    isError,
  } = useSemanticSearch(searchQuery, {
    enabled: isSemantic && searchQuery.length > 1,
  });

  // Time-based greeting with poetic touch
  const greeting = useMemo(() => {
    const hour = new Date().getHours();
    const greetings = {
      night: t("greeting.night"),
      morning: t("greeting.morning"),
      afternoon: t("greeting.afternoon"),
      evening: t("greeting.evening"),
    };

    if (hour < 6) return greetings.night;
    if (hour < 12) return greetings.morning;
    if (hour < 18) return greetings.afternoon;
    return greetings.evening;
  }, [t]);

  // Stats
  const stats = useMemo(() => {
    const total = Object.values(statistics?.activityStats || {}).reduce((sum, count) => sum + count, 0);
    return { total, tags: Object.keys(tags).length };
  }, [statistics, tags]);

  // Update dropdown position
  useEffect(() => {
    if (searchContainerRef.current && (isFocused || searchQuery.length > 1)) {
      const rect = searchContainerRef.current.getBoundingClientRect();
      setDropdownPosition({
        top: rect.bottom + 12,
        left: rect.left,
        width: rect.width,
      });
    }
  }, [isFocused, searchQuery]);

  // Handle search
  const handleSearchChange = (value: string) => {
    setSearchQuery(value);
  };

  const handleKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === "Enter") {
      e.preventDefault();
      if (!isSemantic) {
        const trimmedText = searchQuery.trim();
        if (trimmedText !== "") {
          const words = trimmedText.split(/\s+/);
          words.forEach((word) => {
            addFilter({ factor: "contentSearch", value: word });
          });
          setSearchQuery("");
        }
      } else if (semanticResults?.results?.[0]) {
        onMemoClick(semanticResults.results[0].name);
      }
    }
    if (e.key === "Escape") {
      setIsFocused(false);
      inputRef.current?.blur();
    }
  };

  const onMemoClick = (memoId: string) => {
    const id = memoId.split("/").pop();
    if (id) {
      navigateTo(`/m/${id}`);
      setSearchQuery("");
      setIsFocused(false);
    }
  };

  const handleClearSearch = () => {
    setSearchQuery("");
    inputRef.current?.focus();
  };

  const handleToggleMode = () => {
    setIsSemantic(!isSemantic);
    inputRef.current?.focus();
  };

  // Keyboard shortcut
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key === "k") {
        e.preventDefault();
        inputRef.current?.focus();
      }
    };
    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, []);

  // Click outside
  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (containerRef.current && !containerRef.current.contains(event.target as Node)) {
        setIsFocused(false);
      }
    };
    document.addEventListener("mousedown", handleClickOutside);
    return () => document.removeEventListener("mousedown", handleClickOutside);
  }, []);

  // Entry animation
  useEffect(() => {
    if (!hasAnimated) {
      setHasAnimated(true);
    }
  }, [hasAnimated]);

  const showDropdown = isSemantic && searchQuery.length > 1 && isFocused;
  const hasResults = semanticResults?.results && semanticResults.results.length > 0;

  // Dropdown Content - 使用统一的圆角和间距规范
  const DropdownContent = (
    <div className="overflow-hidden rounded-xl border border-border/50 bg-background/95 backdrop-blur-xl shadow-2xl shadow-primary/5">
      {/* Loading */}
      {isLoading && (
        <div className="flex items-center justify-center gap-3 px-6 py-8 text-muted-foreground">
          <Loader2 className="h-5 w-5 animate-spin text-primary" />
          <span className="text-sm">{t("search.ai-searching")}</span>
        </div>
      )}

      {/* Error */}
      {isError && (
        <div className="px-6 py-8">
          <div className="mb-4 flex items-center justify-center text-amber-600 dark:text-amber-400">
            <AlertCircle className="mr-2 h-5 w-5" />
            <span className="text-sm font-medium">{t("search.ai-error")}</span>
          </div>
          <button
            onClick={handleToggleMode}
            className="w-full rounded-lg bg-primary/10 py-2.5 px-4 text-sm text-primary transition-colors hover:bg-primary/20"
          >
            {t("search.fallback-to-keyword")}
          </button>
        </div>
      )}

      {/* No Results */}
      {!isLoading && !isError && !hasResults && (
        <div className="px-6 py-8 text-center">
          <div className="mx-auto mb-4 flex h-12 w-12 items-center justify-center rounded-full bg-muted/50">
            <Search className="h-5 w-5 text-muted-foreground" />
          </div>
          <p className="text-sm text-muted-foreground">{t("search.no-results")}</p>
        </div>
      )}

      {/* Results */}
      {!isLoading && !isError && hasResults && (
        <div className="max-h-96 overflow-y-auto">
          {(semanticResults?.results || []).map((result, index) => (
            <div
              key={String(result.name)}
              className={cn(
                "group border-b border-border/50 last:border-0",
                "cursor-pointer transition-all duration-200",
                "hover:bg-muted/30 hover:pl-5",
                "flex items-start gap-4 px-5 py-4",
              )}
              onClick={() => onMemoClick(result.name)}
              style={{ animationDelay: `${index * 25}ms` }}
            >
              {/* Result icon with glow */}
              <div className="mt-0.5 flex shrink-0">
                <div
                  className={cn(
                    "flex h-9 w-9 items-center justify-center rounded-lg",
                    "bg-primary/10 transition-all duration-300",
                    "group-hover:bg-primary/20 group-hover:scale-110",
                  )}
                >
                  <FileText className="h-4 w-4 text-primary" />
                </div>
              </div>

              {/* Result content */}
              <div className="min-w-0 flex-1">
                {/* Match score with elegant badge */}
                <div className="mb-2 flex items-center gap-2">
                  <span
                    className={cn(
                      "inline-flex items-center rounded-full px-2 py-0.5",
                      "bg-primary/10 text-primary",
                      "text-xs font-medium tabular-nums",
                    )}
                  >
                    {Math.round(Number(result.score) * 100)}%
                  </span>
                  <span className="text-xs text-muted-foreground">匹配度</span>
                </div>
                <p className="text-sm leading-normal text-foreground/80 line-clamp-2">{String(result.snippet || "")}</p>
              </div>
            </div>
          ))}
        </div>
      )}

      {/* Footer hint */}
      {!isLoading && !isError && hasResults && (
        <div className="border-t border-border/50 bg-muted/20 px-4 py-2.5">
          <p className="text-center text-xs text-muted-foreground">{t("search.navigate-hint")}</p>
        </div>
      )}
    </div>
  );

  return (
    <div className={cn("relative pb-8", className)} ref={containerRef}>
      {/* Ambient background - subtle "consciousness field" */}
      <div className="pointer-events-none absolute inset-0 -z-10 overflow-hidden">
        {/* Primary ambient glow */}
        <div
          className="absolute -right-32 -top-32 h-96 w-96 rounded-full bg-primary/3 blur-3xl"
          style={{ animation: `breathe ${BREATH_DURATION}ms ease-in-out infinite` }}
        />
        {/* Secondary ambient glow */}
        <div
          className="absolute -left-24 top-32 h-64 w-64 rounded-full bg-primary/2 blur-3xl"
          style={{ animation: `breathe ${BREATH_DURATION}ms ease-in-out infinite`, animationDelay: `${BREATH_DURATION / 2}ms` }}
        />
      </div>

      {/* Main content - 使用统一间距 */}
      <div className={SPACING.inner}>
        {/* Greeting section - poetic and minimal */}
        <div
          className={cn("transition-all duration-1000 ease-out", hasAnimated ? "opacity-100 translate-y-0" : "opacity-0 -translate-y-4")}
        >
          {/* Date with subtle accent */}
          <div className="mb-4 flex items-center gap-3">
            <div className="h-px flex-1 bg-gradient-to-r from-transparent to-border" />
            <div className="flex items-center gap-2 text-xs font-medium tracking-widest text-muted-foreground">
              <Sparkles className="h-3.5 w-3.5 text-primary" />
              <span>
                {new Date().toLocaleDateString("zh-CN", {
                  month: "long",
                  day: "numeric",
                  weekday: "long",
                })}
              </span>
            </div>
            <div className="h-px flex-1 bg-gradient-to-l from-transparent to-border" />
          </div>

          {/* Main greeting - 响应式字体大小 */}
          <h1 className="text-center font-semibold text-2xl sm:text-3xl lg:text-4xl xl:text-5xl text-foreground leading-snug">
            {String(greeting)}，{" "}
            <span className="text-primary">
              {String(currentUser?.displayName || currentUser?.name?.split("/")?.pop() || t("common.user"))}
            </span>
          </h1>

          {/* Subtitle - 使用 max-w-[28rem] 避免 Tailwind v4 bug */}
          <p
            style={{ opacity: isFocused ? 0.3 : 1 }}
            className="mx-auto max-w-[28rem] text-center text-sm text-muted-foreground transition-opacity duration-300"
          >
            {t("memo.hero.subtitle")}
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
          <ZenStatCard icon={FileText} label={t("memo.hero.total_memos")} value={stats.total} delay={0} />
          <ZenStatCard icon={Calendar} label={t("memo.hero.tags")} value={stats.tags} delay={1} />
          <ZenStatCard icon={Activity} label={t("memo.hero.streak")} value="—" delay={2} />
        </div>

        {/* Search - the "consciousness field" */}
        <div
          ref={searchContainerRef}
          className={cn("transition-all duration-1000 ease-out", hasAnimated ? "opacity-100 translate-y-0" : "opacity-0 translate-y-4")}
          style={{ transitionDelay: hasAnimated ? `${STAGGER_DELAY * 2}ms` : "0ms" }}
        >
          {/* Search container with "aura" effect */}
          <div className="group relative transition-all duration-500 ease-out">
            {/* Focused aura */}
            <div
              className={cn(
                "absolute -inset-1 rounded-2xl opacity-0 transition-opacity duration-500",
                "bg-gradient-to-r from-primary/20 via-primary/10 to-primary/20",
                "blur-xl",
                isFocused && "opacity-100",
              )}
            />

            {/* Search input */}
            <div
              className={cn(
                "relative flex items-center gap-3 rounded-2xl",
                "border-2 transition-all duration-300",
                "bg-background/80 backdrop-blur-md",
                isFocused ? "border-primary/50 shadow-xl shadow-primary/10" : "border-border/50 shadow-sm",
                "px-5 py-4",
              )}
            >
              {/* Mode toggle */}
              <button
                onClick={handleToggleMode}
                className={cn(
                  "flex h-9 w-9 shrink-0 items-center justify-center rounded-xl",
                  "transition-all duration-200",
                  "hover:scale-105 active:scale-95",
                  isSemantic
                    ? "bg-primary text-primary-foreground shadow-lg shadow-primary/25"
                    : "bg-muted text-muted-foreground hover:bg-muted/80",
                )}
                aria-label={isSemantic ? "切换到关键词搜索" : "切换到智能搜索"}
              >
                {isSemantic ? <Sparkles className="h-4 w-4" /> : <Search className="h-4 w-4" />}
              </button>

              {/* Input */}
              <input
                ref={inputRef}
                type="text"
                value={searchQuery}
                onChange={(e) => handleSearchChange(e.currentTarget.value)}
                onKeyDown={handleKeyDown}
                onFocus={() => setIsFocused(true)}
                placeholder={isSemantic ? t("search.ai-placeholder") : t("memo.search_placeholder")}
                className={cn(
                  "flex-1 bg-transparent text-base outline-none",
                  "placeholder:text-muted-foreground/50",
                  "transition-colors duration-200",
                )}
              />

              {/* Clear button */}
              {searchQuery && (
                <button
                  onClick={handleClearSearch}
                  className={cn(
                    "flex h-7 w-7 shrink-0 items-center justify-center rounded-lg",
                    "text-muted-foreground transition-all duration-200",
                    "hover:bg-muted hover:text-foreground",
                    "active:scale-90",
                  )}
                  aria-label="清空搜索"
                >
                  <X className="h-3.5 w-3.5" />
                </button>
              )}

              {/* Keyboard shortcut hint */}
              {!searchQuery && !isFocused && (
                <div className="hidden sm:flex items-center gap-1.5 shrink-0">
                  <kbd className="rounded-md bg-muted/50 px-2 py-1 text-[10px] font-mono text-muted-foreground">⌘ K</kbd>
                </div>
              )}
            </div>
          </div>
        </div>
      </div>

      {/* Dropdown portal */}
      {showDropdown &&
        createPortal(
          <div
            className="fixed z-50 animate-in fade-in slide-in-from-top-2 duration-200"
            style={{
              top: dropdownPosition.top,
              left: dropdownPosition.left,
              width: dropdownPosition.width,
            }}
          >
            {DropdownContent}
          </div>,
          document.body,
        )}

      {/* Active filters */}
      <div
        className={cn(
          "mt-2 sm:mt-3 transition-all duration-1000 ease-out",
          hasAnimated ? "opacity-100 translate-y-0" : "opacity-0 translate-y-4",
        )}
        style={{ transitionDelay: hasAnimated ? `${STAGGER_DELAY * 3}ms` : "0ms" }}
      >
        <MemoFilters />
      </div>
    </div>
  );
});

HeroSection.displayName = "HeroSection";

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
   Global breath animation (syncs with logo)
   ============================================================ */

// Inject breath keyframes if not exists
if (typeof document !== "undefined") {
  const styleId = "hero-breath-animation";
  if (!document.getElementById(styleId)) {
    const style = document.createElement("style");
    style.id = styleId;
    style.textContent = `
      @keyframes breathe {
        0%, 100% { opacity: 0.6; transform: scale(1); }
        50% { opacity: 1; transform: scale(1.05); }
      }
    `;
    document.head.appendChild(style);
  }
}
