import { Activity, AlertTriangle, BarChart3, Clock, TrendingUp } from "lucide-react";
import { useEffect, useState } from "react";
import useIsMobile from "@/hooks/useIsMobile";
import { cn } from "@/lib/utils";
import { type MetricsOverview, metricsService } from "@/services/metrics";
import { useTranslate } from "@/utils/i18n";

export { MetricsDashboard };

function MetricsDashboard() {
  const t = useTranslate();
  const isMobile = useIsMobile();
  const [timeRange, setTimeRange] = useState("24h");
  const [metrics, setMetrics] = useState<MetricsOverview | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    setIsLoading(true);
    setError(null);
    metricsService
      .getOverview(timeRange)
      .then(setMetrics)
      .catch((err) => {
        console.error("Failed to fetch metrics:", err);
        setError(err instanceof Error ? err.message : "Failed to load metrics");
      })
      .finally(() => setIsLoading(false));
  }, [timeRange]);

  const timeRanges = [
    { value: "1h", label: "1H" },
    { value: "24h", label: "24H" },
    { value: "7d", label: "7D" },
    { value: "30d", label: "30D" },
  ];

  // Translation keys (must exist in i18n files)
  const tTitle = t("setting.metrics-section.overview");
  const tRequests = t("setting.metrics-section.requests");
  const tSuccessRate = t("setting.metrics-section.success-rate");
  const tLatency = t("setting.metrics-section.latency");
  const tP95 = t("setting.metrics-section.p95");
  const tErrors = t("setting.metrics-section.errors");

  return (
    <div className="space-y-4">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <BarChart3 className="w-5 h-5 text-muted-foreground" />
          <h3 className="text-lg font-semibold">{tTitle}</h3>
        </div>
        {!isMobile && (
          <div className="flex gap-1 bg-muted rounded-lg p-1">
            {timeRanges.map((range) => (
              <button
                key={range.value}
                onClick={() => setTimeRange(range.value)}
                className={cn(
                  "px-3 py-1 text-xs font-medium rounded-md transition-colors",
                  timeRange === range.value ? "bg-background text-foreground shadow-sm" : "text-muted-foreground hover:text-foreground",
                )}
              >
                {range.label}
              </button>
            ))}
          </div>
        )}
      </div>

      {/* Mobile: Time range selector as horizontal scroll */}
      {isMobile && (
        <div className="flex gap-2 overflow-x-auto pb-2 scrollbar-hide">
          {timeRanges.map((range) => (
            <button
              key={range.value}
              onClick={() => setTimeRange(range.value)}
              className={cn(
                "px-4 py-2 text-sm font-medium rounded-full whitespace-nowrap transition-colors",
                timeRange === range.value ? "bg-primary text-primary-foreground" : "bg-muted text-muted-foreground hover:bg-muted/80",
              )}
            >
              {range.label}
            </button>
          ))}
        </div>
      )}

      {error && (
        <div className="p-4 bg-destructive/10 border border-destructive/20 rounded-lg">
          <p className="text-sm text-destructive">{error}</p>
        </div>
      )}

      {isLoading ? (
        <div className={cn("grid gap-4", isMobile ? "grid-cols-2" : "grid-cols-1 md:grid-cols-4")}>
          {[...Array(4)].map((_, i) => (
            <div key={i} className={cn("bg-muted/30 rounded-lg animate-pulse", isMobile ? "h-24" : "h-28")} />
          ))}
        </div>
      ) : (
        <div className={cn("grid gap-4", isMobile ? "grid-cols-2" : "grid-cols-1 md:grid-cols-4")}>
          <MetricCard
            title={tRequests}
            value={metrics?.total_requests ?? 0}
            icon={<Activity className="w-4 h-4" />}
            color="text-blue-600"
            bgColor="bg-blue-500/10"
          />
          <MetricCard
            title={tSuccessRate}
            value={`${((metrics?.success_rate ?? 0) * 100).toFixed(1)}%`}
            icon={<TrendingUp className="w-4 h-4" />}
            color="text-green-600"
            bgColor="bg-green-500/10"
          />
          <MetricCard
            title={tLatency}
            value={`${metrics?.p50_latency_ms ?? 0}ms`}
            icon={<Clock className="w-4 h-4" />}
            color="text-amber-600"
            bgColor="bg-amber-500/10"
          />
          <MetricCard
            title={tP95}
            value={`${metrics?.p95_latency_ms ?? 0}ms`}
            icon={<Clock className="w-4 h-4" />}
            color="text-purple-600"
            bgColor="bg-purple-500/10"
          />
        </div>
      )}

      {metrics && metrics.error_count > 0 && (
        <div className={cn("flex items-start gap-3 rounded-lg border", "bg-destructive/10 border-destructive/20")}>
          <AlertTriangle className="w-5 h-5 text-destructive shrink-0 mt-0.5" />
          <div className="flex-1">
            <p className="text-sm text-destructive font-medium">{tErrors}</p>
            <p className="text-xs text-destructive/80 mt-1">
              {metrics.error_count} {t("setting.metrics-section.errors")}
            </p>
          </div>
        </div>
      )}

      {metrics && metrics.is_mock && (
        <div className={cn("flex items-start gap-3 rounded-lg border p-3", "bg-amber-500/10 border-amber-500/20")}>
          <AlertTriangle className="w-4 h-4 text-amber-600 dark:text-amber-400 shrink-0 mt-0.5" />
          <p className="text-xs text-amber-600 dark:text-amber-400">{t("setting.metrics-section.mock-data")}</p>
        </div>
      )}
    </div>
  );
}

interface MetricCardProps {
  title: string;
  value: number | string;
  icon: React.ReactNode;
  color: string;
  bgColor: string;
}

function MetricCard({ title, value, icon, color, bgColor }: MetricCardProps) {
  const isMobile = useIsMobile();

  return (
    <div className={cn("bg-card border border-border rounded-lg transition-shadow", isMobile ? "p-3" : "p-4 hover:shadow-sm")}>
      <div className="flex items-center justify-between mb-2">
        <span className={cn("text-muted-foreground", isMobile ? "text-xs" : "text-sm")}>{title}</span>
        <div className={cn("p-1.5 rounded-lg", bgColor, isMobile && "p-1")}>
          <div className={cn(color, isMobile && "w-3.5 h-3.5")}>{icon}</div>
        </div>
      </div>
      <p className={cn("font-semibold text-foreground", isMobile ? "text-lg" : "text-2xl")}>{value}</p>
    </div>
  );
}
