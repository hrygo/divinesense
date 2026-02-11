/**
 * PerformanceMetricsPanel - 性能指标可视化组件
 *
 * 用于在开发环境中展示编辑器性能指标
 */

import type { PerformanceMetrics } from "./usePerformanceMonitor";

interface PerformanceMetricsPanelProps {
  metrics: PerformanceMetrics;
}

/**
 * 性能指标可视化组件
 *
 * @example
 * ```tsx
 * <PerformanceMetricsPanel metrics={metrics} />
 * ```
 */
export function PerformanceMetricsPanel({ metrics }: PerformanceMetricsPanelProps) {
  if (!import.meta.env.DEV) return null;

  return (
    <div className="fixed bottom-4 right-4 bg-background border rounded-lg p-3 shadow-lg text-xs">
      <h3 className="font-semibold mb-2">Performance Metrics</h3>
      <div className="space-y-1">
        <div className="flex justify-between gap-4">
          <span className="text-muted-foreground">Avg Latency:</span>
          <span className={metrics.averageInputLatency > 50 ? "text-red-500" : "text-green-500"}>
            {metrics.averageInputLatency.toFixed(2)}ms
          </span>
        </div>
        <div className="flex justify-between gap-4">
          <span className="text-muted-foreground">Max Latency:</span>
          <span className={metrics.maxInputLatency > 100 ? "text-red-500" : "text-green-500"}>{metrics.maxInputLatency.toFixed(2)}ms</span>
        </div>
        <div className="flex justify-between gap-4">
          <span className="text-muted-foreground">Inputs:</span>
          <span>{metrics.inputCount}</span>
        </div>
        <div className="flex justify-between gap-4">
          <span className="text-muted-foreground">Renders:</span>
          <span>{metrics.renderCount}</span>
        </div>
      </div>
    </div>
  );
}
