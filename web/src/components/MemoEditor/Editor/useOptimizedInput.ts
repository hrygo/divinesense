/**
 * useOptimizedInput - 优化的输入事件处理
 *
 * UX 优化策略：
 * 1. 立即更新本地状态，保持输入响应性
 * 2. 延迟执行副作用（高度计算、自动保存等）
 * 3. 使用 Transition API 标记非紧急更新
 */

import { startTransition, useCallback, useEffect, useRef } from "react";

interface UseOptimizedInputOptions {
  /** 输入回调 */
  onInput: (value: string) => void;
  /** 延迟执行的副作用回调 */
  onDeferredUpdate?: (value: string) => void;
  /** 延迟时间 (ms) */
  deferDelay?: number;
  /** 是否使用 startTransition 包装更新 */
  useTransition?: boolean;
}

interface UseOptimizedInputReturn {
  /** 处理输入事件（立即响应） */
  handleInput: (e: React.FormEvent<HTMLTextAreaElement>) => void;
  /** 立即执行所有待处理的更新 */
  flushPendingUpdates: () => void;
  /** 取消待处理的更新 */
  cancelPendingUpdates: () => void;
}

/**
 * 优化的输入处理 Hook
 *
 * @example
 * ```tsx
 * const { handleInput, flushPendingUpdates } = useOptimizedInput({
 *   onInput: (value) => setContent(value),
 *   onDeferredUpdate: (value) => saveToCache(value),
 *   deferDelay: 300,
 * });
 * ```
 */
export function useOptimizedInput(options: UseOptimizedInputOptions): UseOptimizedInputReturn {
  const { onInput, onDeferredUpdate, deferDelay = 150, useTransition = true } = options;

  const timeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const lastValueRef = useRef<string>("");
  const pendingValueRef = useRef<string | null>(null);

  /**
   * 立即更新本地状态（保持输入响应性）
   * 延迟执行副作用（自动保存、高度计算等）
   */
  const handleInput = useCallback(
    (e: React.FormEvent<HTMLTextAreaElement>) => {
      const target = e.target as HTMLTextAreaElement;
      const newValue = target.value;

      // 立即更新本地状态
      if (useTransition) {
        startTransition(() => {
          onInput(newValue);
        });
      } else {
        onInput(newValue);
      }

      lastValueRef.current = newValue;
      pendingValueRef.current = newValue;

      // 清除之前的定时器
      if (timeoutRef.current) {
        clearTimeout(timeoutRef.current);
      }

      // 延迟执行副作用
      if (onDeferredUpdate) {
        timeoutRef.current = setTimeout(() => {
          if (pendingValueRef.current !== null) {
            onDeferredUpdate(pendingValueRef.current);
            pendingValueRef.current = null;
          }
          timeoutRef.current = null;
        }, deferDelay);
      }
    },
    [onInput, onDeferredUpdate, deferDelay, useTransition],
  );

  /**
   * 立即执行所有待处理的更新
   */
  const flushPendingUpdates = useCallback(() => {
    if (timeoutRef.current) {
      clearTimeout(timeoutRef.current);
      timeoutRef.current = null;
    }

    if (pendingValueRef.current !== null && onDeferredUpdate) {
      onDeferredUpdate(pendingValueRef.current);
      pendingValueRef.current = null;
    }
  }, [onDeferredUpdate]);

  /**
   * 取消待处理的更新
   */
  const cancelPendingUpdates = useCallback(() => {
    if (timeoutRef.current) {
      clearTimeout(timeoutRef.current);
      timeoutRef.current = null;
    }
    pendingValueRef.current = null;
  }, []);

  // 组件卸载时清理
  useEffect(() => {
    return () => {
      if (timeoutRef.current) {
        clearTimeout(timeoutRef.current);
      }
    };
  }, []);

  return {
    handleInput,
    flushPendingUpdates,
    cancelPendingUpdates,
  };
}

/**
 * 使用 requestIdleCallback 的更激进的优化版本
 * 在浏览器空闲时执行非关键任务
 */
export function useIdleInput(options: UseOptimizedInputOptions): UseOptimizedInputReturn {
  const { onInput, onDeferredUpdate, deferDelay = 150 } = options;

  const timeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const rafIdRef = useRef<ReturnType<typeof requestAnimationFrame> | null>(null);
  const idleCallbackIdRef = useRef<number | null>(null);

  const handleInput = useCallback(
    (e: React.FormEvent<HTMLTextAreaElement>) => {
      const target = e.target as HTMLTextAreaElement;
      const newValue = target.value;

      // 立即更新本地状态（使用 RAF 确保在下一帧渲染）
      if (rafIdRef.current) {
        cancelAnimationFrame(rafIdRef.current);
      }

      rafIdRef.current = requestAnimationFrame(() => {
        onInput(newValue);
        rafIdRef.current = null;
      });

      // 延迟执行副作用
      if (onDeferredUpdate) {
        if (timeoutRef.current) {
          clearTimeout(timeoutRef.current);
        }
        if (idleCallbackIdRef.current !== null) {
          cancelIdleCallback(idleCallbackIdRef.current);
        }

        // 使用 setTimeout 作为后备（如果 requestIdleCallback 不可用）
        timeoutRef.current = setTimeout(() => {
          const scheduleIdleWork = () => {
            if ("requestIdleCallback" in window) {
              idleCallbackIdRef.current = requestIdleCallback(
                () => {
                  onDeferredUpdate(newValue);
                  idleCallbackIdRef.current = null;
                },
                { timeout: deferDelay },
              );
            } else {
              onDeferredUpdate(newValue);
            }
          };

          scheduleIdleWork();
          timeoutRef.current = null;
        }, deferDelay);
      }
    },
    [onInput, onDeferredUpdate, deferDelay],
  );

  const flushPendingUpdates = useCallback(() => {
    if (timeoutRef.current) {
      clearTimeout(timeoutRef.current);
      timeoutRef.current = null;
    }
    if (rafIdRef.current) {
      cancelAnimationFrame(rafIdRef.current);
      rafIdRef.current = null;
    }
    if (idleCallbackIdRef.current !== null) {
      cancelIdleCallback(idleCallbackIdRef.current);
      idleCallbackIdRef.current = null;
    }
  }, []);

  const cancelPendingUpdates = useCallback(() => {
    if (timeoutRef.current) {
      clearTimeout(timeoutRef.current);
      timeoutRef.current = null;
    }
    if (rafIdRef.current) {
      cancelAnimationFrame(rafIdRef.current);
      rafIdRef.current = null;
    }
    if (idleCallbackIdRef.current !== null) {
      cancelIdleCallback(idleCallbackIdRef.current);
      idleCallbackIdRef.current = null;
    }
  }, []);

  return {
    handleInput,
    flushPendingUpdates,
    cancelPendingUpdates,
  };
}
