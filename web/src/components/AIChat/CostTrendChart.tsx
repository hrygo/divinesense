import { useTranslation } from "react-i18next";
import { cn } from "@/lib/utils";

interface CostDataPoint {
  date: string; // YYYY-MM-DD
  costUSD: number;
  sessionCount?: number;
}

interface CostTrendChartProps {
  data: CostDataPoint[];
  className?: string;
  days?: number; // Number of days to show
}

/**
 * CostTrendChart - Simple SVG line chart for visualizing cost trends
 *
 * Features:
 * - Auto-scaling Y-axis based on data
 * - Responsive SVG rendering
 * - Hover tooltips for data points
 * - Minimal dependencies (no external chart library)
 */
export function CostTrendChart({ data, className, days = 7 }: CostTrendChartProps) {
  const { t } = useTranslation();

  if (!data || data.length === 0) {
    return (
      <div className={cn("flex items-center justify-center h-48 text-muted-foreground", className)}>
        <p>{t("ai.session-stats.title")} - No data</p>
      </div>
    );
  }

  // Filter to last N days and sort by date
  const chartData = data.slice(-days).sort((a, b) => a.date.localeCompare(b.date));

  // Calculate chart dimensions
  const maxCost = Math.max(...chartData.map((d) => d.costUSD), 0.01);
  const padding = { top: 20, right: 20, bottom: 30, left: 50 };
  const width = 600;
  const height = 200;
  const chartWidth = width - padding.left - padding.right;
  const chartHeight = height - padding.top - padding.bottom;

  // Generate Y-axis ticks (5 ticks)
  const yTicks = Array.from({ length: 5 }, (_, i) => (maxCost * i) / 4);

  // Format date for display
  const formatDate = (dateStr: string) => {
    const date = new Date(dateStr);
    return date.toLocaleDateString("en-US", { month: "short", day: "numeric" });
  };

  // Format cost for display
  const formatCost = (cost: number) => `$${cost.toFixed(4)}`;

  // Generate path for the line
  const linePath = chartData
    .map((point, i) => {
      const x = padding.left + (i / (chartData.length - 1 || 1)) * chartWidth;
      const y = padding.top + chartHeight - (point.costUSD / maxCost) * chartHeight;
      return `${i === 0 ? "M" : "L"} ${x} ${y}`;
    })
    .join(" ");

  // Generate area path (for gradient fill)
  const areaPath = `${linePath} L ${padding.left + chartWidth} ${padding.top + chartHeight} L ${padding.left} ${padding.top + chartHeight} Z`;

  return (
    <div className={cn("w-full", className)}>
      <svg viewBox={`0 0 ${width} ${height}`} className="w-full h-auto" preserveAspectRatio="xMidYMid meet">
        {/* Grid lines */}
        {yTicks.map((value, i) => {
          const y = padding.top + chartHeight - (value / maxCost) * chartHeight;
          return (
            <g key={`grid-${i}`}>
              <line x1={padding.left} y1={y} x2={width - padding.right} y2={y} className="stroke-border/30" strokeWidth={1} />
              <text x={padding.left - 10} y={y + 4} className="text-[10px] fill-muted-foreground text-right" textAnchor="end">
                {formatCost(value)}
              </text>
            </g>
          );
        })}

        {/* X-axis labels */}
        {chartData.map((point, i) => {
          const x = padding.left + (i / (chartData.length - 1 || 1)) * chartWidth;
          const showLabel = chartData.length <= 10 || i % Math.ceil(chartData.length / 7) === 0;
          return (
            <text
              key={`label-${i}`}
              x={x}
              y={height - 5}
              className={cn("text-[10px] fill-muted-foreground", !showLabel && "hidden sm:block")}
              textAnchor="middle"
            >
              {formatDate(point.date)}
            </text>
          );
        })}

        {/* Area fill */}
        <path d={areaPath} className="fill-primary/10" />

        {/* Line */}
        <path d={linePath} className="fill-none stroke-primary" strokeWidth={2} strokeLinecap="round" strokeLinejoin="round" />

        {/* Data points */}
        {chartData.map((point, i) => {
          const x = padding.left + (i / (chartData.length - 1 || 1)) * chartWidth;
          const y = padding.top + chartHeight - (point.costUSD / maxCost) * chartHeight;

          return (
            <g key={`point-${i}`}>
              <circle cx={x} cy={y} r={4} className="fill-background stroke-primary stroke-2 hover:r-5 transition-all cursor-pointer">
                <title>
                  {formatDate(point.date)}: {formatCost(point.costUSD)}
                  {point.sessionCount !== undefined && ` (${point.sessionCount} sessions)`}
                </title>
              </circle>
            </g>
          );
        })}
      </svg>

      {/* Legend */}
      <div className="flex items-center justify-center gap-4 mt-2 text-xs text-muted-foreground">
        <div className="flex items-center gap-1">
          <div className="w-3 h-3 rounded-full bg-primary" />
          <span>{t("ai.session-stats.daily-cost")}</span>
        </div>
      </div>
    </div>
  );
}

/**
 * CompactCostTrend - Mini sparkline version for cards
 */
interface CompactCostTrendProps {
  data: CostDataPoint[];
  className?: string;
}

export function CompactCostTrend({ data, className }: CompactCostTrendProps) {
  if (!data || data.length < 2) {
    return null;
  }

  const chartData = data.slice(-7); // Last 7 data points
  const maxCost = Math.max(...chartData.map((d) => d.costUSD), 0.01);
  const minCost = Math.min(...chartData.map((d) => d.costUSD));
  const range = maxCost - minCost || 1;

  const width = 100;
  const height = 30;
  const padding = 2;

  const linePath = chartData
    .map((point, i) => {
      const x = padding + (i / (chartData.length - 1)) * (width - 2 * padding);
      const normalizedCost = (point.costUSD - minCost) / range;
      const y = height - padding - normalizedCost * (height - 2 * padding);
      return `${i === 0 ? "M" : "L"} ${x} ${y}`;
    })
    .join(" ");

  // Determine trend color based on last vs first
  const trend = chartData[chartData.length - 1].costUSD >= chartData[0].costUSD ? "text-green-500" : "text-red-500";
  const strokeColor = chartData[chartData.length - 1].costUSD >= chartData[0].costUSD ? "#22c55e" : "#ef4444";

  return (
    <svg viewBox={`0 0 ${width} ${height}`} className={cn("w-full h-auto", trend, className)} preserveAspectRatio="none">
      <path d={linePath} fill="none" stroke={strokeColor} strokeWidth={2} strokeLinecap="round" strokeLinejoin="round" />
    </svg>
  );
}
