/**
 * HeroSection - Memo Home Page Hero Section with Inline Search
 *
 * Displays:
 * - Greeting based on time of day
 * - Quick stats (total memos, tags)
 * - Inline search bar with real-time filtering
 * - Active filters display (MemoFilters)
 * - Layout control button (immersive mode)
 */

import { Calendar, FileText, Flame, Maximize2, Minimize2, Search, X } from "lucide-react";
import { memo, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import MemoFilters from "@/components/MemoFilters";
import useCurrentUser from "@/hooks/useCurrentUser";
import { useFilteredMemoStats } from "@/hooks/useFilteredMemoStats";
import { useMemoLayout } from "@/layouts/MemoLayout";
import { cn } from "@/lib/utils";

export interface HeroSectionProps {
  onSearchChange?: (query: string) => void;
  className?: string;
}

export const HeroSection = memo(function HeroSection({ onSearchChange, className }: HeroSectionProps) {
  const { t } = useTranslation();
  const currentUser = useCurrentUser();
  const { immersiveMode, toggleImmersiveMode } = useMemoLayout();

  // Search state
  const [searchQuery, setSearchQuery] = useState("");
  const [isFocused, setIsFocused] = useState(false);

  // Get memo stats for current user
  const { statistics, tags } = useFilteredMemoStats({
    userName: currentUser?.name,
  });

  // Calculate greeting based on time of day
  const greeting = useMemo(() => {
    const hour = new Date().getHours();
    if (hour < 6) return t("greeting.night") || "Good night";
    if (hour < 12) return t("greeting.morning") || "Good morning";
    if (hour < 18) return t("greeting.afternoon") || "Good afternoon";
    return t("greeting.evening") || "Good evening";
  }, [t]);

  // Calculate stats
  const stats = useMemo(() => {
    const total = Object.values(statistics?.activityStats || {}).reduce((sum, count) => sum + count, 0);
    return { total, tags: Object.keys(tags).length };
  }, [statistics, tags]);

  // Handle search input changes
  const handleSearchChange = (value: string) => {
    setSearchQuery(value);
    onSearchChange?.(value);
  };

  // Clear search
  const handleClearSearch = () => {
    setSearchQuery("");
    onSearchChange?.("");
  };

  // Handle keyboard shortcut
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      // Cmd/Ctrl + K to focus search
      if ((e.metaKey || e.ctrlKey) && e.key === "k") {
        e.preventDefault();
        document.getElementById("memo-search-input")?.focus();
      }
      // Escape to clear search
      if (e.key === "Escape" && searchQuery) {
        handleClearSearch();
      }
    };

    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [searchQuery]);

  return (
    <div className={cn("bg-background/95 backdrop-blur-sm relative py-8", className)}>
      {/* Immersive Mode Toggle Button - Desktop only, fixed at top-right of container */}
      <div className="absolute top-4 right-4 hidden lg:flex z-10">
        <button
          onClick={() => toggleImmersiveMode(!immersiveMode)}
          className={cn(
            "flex items-center justify-center w-8 h-8 rounded-md transition-all",
            "text-muted-foreground hover:text-foreground hover:bg-muted",
            immersiveMode && "text-primary bg-primary/10",
          )}
          title={t("ai.exit-immersive") || "Exit immersive"}
        >
          {immersiveMode ? <Minimize2 className="w-4 h-4" /> : <Maximize2 className="w-4 h-4" />}
        </button>
      </div>

      {/* Greeting */}
      <h1 className="text-3xl font-bold text-foreground mb-1">
        {greeting}, {currentUser?.displayName || currentUser?.name?.split("/").pop() || t("common.user")}
      </h1>
      <p className="text-muted-foreground mb-6">{t("memo.hero.subtitle") || "Capture your thoughts and ideas"}</p>

      {/* Quick Stats */}
      <div className="grid grid-cols-3 gap-4 mb-6">
        <StatCard icon={FileText} label={t("memo.hero.total_memos") || "Total"} value={stats.total} />
        <StatCard icon={Calendar} label={t("memo.hero.tags") || "Tags"} value={stats.tags} />
        <StatCard
          icon={Flame}
          label={t("memo.hero.streak") || "Streak"}
          value={"—"} // TODO: Implement streak calculation
        />
      </div>

      {/* Inline Search Bar */}
      <div className="relative mb-4">
        <div
          className={cn(
            "flex items-center gap-3 px-4 py-3 rounded-lg border-2 transition-all duration-200",
            isFocused || searchQuery
              ? "border-violet-500 bg-violet-50/50 dark:bg-violet-950/20 shadow-sm"
              : "border-zinc-200 dark:border-zinc-700 bg-zinc-100/50 dark:bg-zinc-900/50 hover:border-zinc-300 dark:hover:border-zinc-600",
          )}
        >
          <Search className={cn("w-5 h-5 shrink-0", isFocused || searchQuery ? "text-violet-600 dark:text-violet-400" : "text-zinc-400")} />
          <input
            id="memo-search-input"
            type="text"
            value={searchQuery}
            onChange={(e) => handleSearchChange(e.target.value)}
            onFocus={() => setIsFocused(true)}
            onBlur={() => setIsFocused(false)}
            placeholder={t("memo.hero.search_placeholder") || "Search memos..."}
            className="flex-1 bg-transparent border-0 outline-none text-foreground placeholder:text-zinc-400 text-sm"
          />
          {searchQuery && (
            <button
              onClick={handleClearSearch}
              className="p-1 rounded-md hover:bg-zinc-200 dark:hover:bg-zinc-800 transition-colors"
              aria-label="Clear search"
            >
              <X className="w-4 h-4 text-zinc-400" />
            </button>
          )}
          {!searchQuery && (
            <div className="hidden sm:flex items-center gap-1 text-xs text-zinc-400">
              <kbd className="px-1.5 py-0.5 rounded bg-zinc-200 dark:bg-zinc-700 font-mono">⌘</kbd>
              <kbd className="px-1.5 py-0.5 rounded bg-zinc-200 dark:bg-zinc-700 font-mono">K</kbd>
            </div>
          )}
        </div>
      </div>

      {/* Active Filters Display - moved from MemoList */}
      <MemoFilters />
    </div>
  );
});

HeroSection.displayName = "HeroSection";

interface StatCardProps {
  icon: React.ComponentType<{ className?: string }>;
  label: string;
  value: number | string;
}

function StatCard({ icon: Icon, label, value }: StatCardProps) {
  return (
    <div className="flex items-center gap-3 p-3 rounded-md bg-zinc-100/50 dark:bg-zinc-900/50 border border-zinc-200 dark:border-zinc-800">
      <div className="p-2 rounded-md bg-zinc-200 dark:bg-zinc-800">
        <Icon className="w-4 h-4 text-zinc-600 dark:text-zinc-400" />
      </div>
      <div>
        <div className="text-xl font-bold text-foreground">{value}</div>
        <div className="text-xs text-muted-foreground">{label}</div>
      </div>
    </div>
  );
}
