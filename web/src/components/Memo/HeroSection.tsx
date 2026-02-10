/**
 * HeroSection - Memo Home Page Hero Section
 *
 * Displays:
 * - Greeting based on time of day
 * - Quick stats (today, this week, streak)
 * - Quick action buttons
 */

import { Calendar, FileText, Flame, Plus, Search } from "lucide-react";
import { memo, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";
import useCurrentUser from "@/hooks/useCurrentUser";
import { useFilteredMemoStats } from "@/hooks/useFilteredMemoStats";
import { cn } from "@/lib/utils";

export interface HeroSectionProps {
  onCreateMemo?: () => void;
  onSearch?: () => void;
  className?: string;
}

export const HeroSection = memo(function HeroSection({ onCreateMemo, onSearch, className }: HeroSectionProps) {
  const { t } = useTranslation();
  const currentUser = useCurrentUser();

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
    // Calculate total from activity stats
    const total = Object.values(statistics?.activityStats || {}).reduce((sum, count) => sum + count, 0);

    return {
      total,
      tags: Object.keys(tags).length,
    };
  }, [statistics, tags]);

  return (
    <div className={cn("border-b bg-background/95 backdrop-blur-sm", "border-amber-200 dark:border-amber-800", className)}>
      <div className="mx-auto max-w-4xl px-4 py-8">
        {/* Greeting */}
        <h1 className="text-3xl font-bold text-foreground mb-1">
          {greeting}, {currentUser?.displayName || currentUser?.name?.split("/").pop() || t("common.user")}
        </h1>
        <p className="text-muted-foreground mb-6">{t("memo.hero.subtitle") || "Capture your thoughts and ideas"}</p>

        {/* Quick Stats */}
        <div className="grid grid-cols-3 gap-4 mb-6">
          <StatCard icon={FileText} label={t("memo.hero.total_memos")} value={stats.total} />
          <StatCard icon={Calendar} label={t("memo.hero.tags")} value={stats.tags} />
          <StatCard
            icon={Flame}
            label={t("memo.hero.streak")}
            value={"—"} // TODO: Implement streak calculation from backend data
          />
        </div>

        {/* Quick Actions */}
        <div className="flex flex-wrap gap-2">
          <Button onClick={onCreateMemo} className="gap-2" size="lg">
            <Plus className="w-4 h-4" />
            {t("memo.hero.new_memo")}
          </Button>
          <Button variant="outline" onClick={onSearch} className="gap-2" size="lg">
            <Search className="w-4 h-4" />
            {t("memo.hero.search")}
          </Button>
        </div>
      </div>
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
    <div className="flex items-center gap-3 p-3 rounded-lg bg-amber-50 dark:bg-amber-900/20 border border-amber-200 dark:border-amber-800">
      <div className="p-2 rounded-full bg-amber-100 dark:bg-amber-900/30">
        <Icon className="w-4 h-4 text-amber-600 dark:text-amber-400" />
      </div>
      <div>
        <div className="text-xl font-bold text-foreground">{value}</div>
        <div className="text-xs text-muted-foreground">{label}</div>
      </div>
    </div>
  );
}
