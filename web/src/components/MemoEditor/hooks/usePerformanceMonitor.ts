/**
 * usePerformanceMonitor - 编辑器性能监控 Hook
 *
 * 用途：
 * 1. 监控输入延迟
 * 2. 追踪渲染时间
 * 3. 检测性能退化
 * 4. 上报性能指标
 */

import { useCallback, useEffect, useRef } from "react";

export interface PerformanceMetrics {
  /** 平均输入延迟 (ms) */
  averageInputLatency: number;
  /** 最大输入延迟 (ms) */
  maxInputLatency: number;
  /** 渲染次数 */
  renderCount: number;
  /** 输入次数 */
  inputCount: number;
  /** 上次输入时间戳 */
  lastInputTime: number;
}

interface UsePerformanceMonitorOptions {
  /** 是否启用性能监控 */
  enabled?: boolean;
  /** 采样率 (0-1) */
  sampleRate?: number;
  /** 性能退化阈值 (ms) */
  degradationThreshold?: number;
  /** 性能指标回调 */
  onMetrics?: (metrics: PerformanceMetrics) => void;
  /** 性能退化警告回调 */
  onDegradation?: (metrics: PerformanceMetrics) => void;
}

/**
 * 性能监控 Hook
 *
 * @example
 * ```tsx
 * const { trackInput, trackRender, getMetrics } = usePerformanceMonitor({
 *   enabled: import.meta.env.DEV,
 *   onDegradation: (metrics) => console.warn('Performance degraded:', metrics),
 * });
 * ```
 */
export function usePerformanceMonitor(options: UsePerformanceMonitorOptions = {}) {
  const { enabled = import.meta.env.DEV, sampleRate = 1, degradationThreshold = 50, onMetrics, onDegradation } = options;

  const inputLatenciesRef = useRef<number[]>([]);
  const renderTimesRef = useRef<number[]>([]);
  const inputCountRef = useRef(0);
  const renderCountRef = useRef(0);
  const lastInputTimeRef = useRef(0);
  const renderStartRef = useRef(0);

  /**
   * 追踪输入延迟
   */
  const trackInput = useCallback(() => {
    if (!enabled || Math.random() > sampleRate) return;

    const now = performance.now();
    if (lastInputTimeRef.current > 0) {
      const latency = now - lastInputTimeRef.current;
      inputLatenciesRef.current.push(latency);
      inputCountRef.current++;

      // 保持最近 100 次记录
      if (inputLatenciesRef.current.length > 100) {
        inputLatenciesRef.current.shift();
      }

      // 检测性能退化
      if (latency > degradationThreshold) {
        onDegradation?.({
          averageInputLatency: getAverageLatency(),
          maxInputLatency: getMaxLatency(),
          renderCount: renderCountRef.current,
          inputCount: inputCountRef.current,
          lastInputTime: now,
        });
      }
    }

    lastInputTimeRef.current = now;
  }, [enabled, sampleRate, degradationThreshold, onDegradation]);

  /**
   * 追踪渲染时间
   */
  const trackRender = useCallback(() => {
    if (!enabled || Math.random() > sampleRate) return;

    const renderTime = performance.now() - renderStartRef.current;
    renderTimesRef.current.push(renderTime);
    renderCountRef.current++;

    if (renderTimesRef.current.length > 100) {
      renderTimesRef.current.shift();
    }
  }, [enabled, sampleRate]);

  /**
   * 获取平均延迟
   */
  const getAverageLatency = useCallback(() => {
    const latencies = inputLatenciesRef.current;
    if (latencies.length === 0) return 0;
    const sum = latencies.reduce((a, b) => a + b, 0);
    return sum / latencies.length;
  }, []);

  /**
   * 获取最大延迟
   */
  const getMaxLatency = useCallback(() => {
    const latencies = inputLatenciesRef.current;
    if (latencies.length === 0) return 0;
    return Math.max(...latencies);
  }, []);

  /**
   * 获取性能指标
   */
  const getMetrics = useCallback((): PerformanceMetrics => {
    return {
      averageInputLatency: getAverageLatency(),
      maxInputLatency: getMaxLatency(),
      renderCount: renderCountRef.current,
      inputCount: inputCountRef.current,
      lastInputTime: lastInputTimeRef.current,
    };
  }, [getAverageLatency, getMaxLatency]);

  /**
   * 重置指标
   */
  const resetMetrics = useCallback(() => {
    inputLatenciesRef.current = [];
    renderTimesRef.current = [];
    inputCountRef.current = 0;
    renderCountRef.current = 0;
    lastInputTimeRef.current = 0;
  }, []);

  // 定期上报性能指标
  useEffect(() => {
    if (!enabled) return;

    const interval = setInterval(() => {
      const metrics = getMetrics();
      onMetrics?.(metrics);
    }, 5000); // 每 5 秒上报一次

    return () => clearInterval(interval);
  }, [enabled, getMetrics, onMetrics]);

  // 渲染开始时间（在 effect 中设置）
  useEffect(() => {
    renderStartRef.current = performance.now();
    requestAnimationFrame(() => {
      trackRender();
    });
  });

  return {
    trackInput,
    trackRender,
    getMetrics,
    resetMetrics,
  };
}
