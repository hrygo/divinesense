/**
 * useVirtualHeight - 优化的 textarea 高度自适应
 *
 * 性能优化策略：
 * 1. 缓存上次高度，避免不必要的 DOM 操作
 * 2. 使用 ResizeObserver 代替 scrollHeight 轮询
 * 3. 防抖处理输入事件
 */

import { useCallback, useEffect, useRef } from "react";

interface UseVirtualHeightOptions {
  /** 最小高度 (px) */
  minHeight?: number;
  /** 最大高度 (px) */
  maxHeight?: number;
  /** 是否启用防抖 (默认: true) */
  debounce?: boolean;
  /** 防抖延迟 (ms) */
  debounceDelay?: number;
}

interface UseVirtualHeightReturn {
  /** 更新高度的回调 */
  updateHeight: () => void;
  /** 重置高度到自动 */
  resetHeight: () => void;
  /** 获取当前高度 */
  getCurrentHeight: () => number;
}

/**
 * 优化的高度自适应 Hook
 *
 * @example
 * ```tsx
 * const { updateHeight, resetHeight } = useVirtualHeight({
 *   minHeight: 44,
 *   maxHeight: 400,
 * });
 * ```
 */
export function useVirtualHeight(
  textareaRef: React.RefObject<HTMLTextAreaElement>,
  options: UseVirtualHeightOptions = {},
): UseVirtualHeightReturn {
  const { minHeight = 44, maxHeight = 400, debounce = true, debounceDelay = 50 } = options;

  const lastHeightRef = useRef(0);
  const rafIdRef = useRef<ReturnType<typeof requestAnimationFrame> | null>(null);
  const timeoutIdRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  // 带缓存的高度更新
  const updateHeight = useCallback(() => {
    const textarea = textareaRef.current;
    if (!textarea) return;

    // 使用 RAF 确保在浏览器重绘前执行
    if (rafIdRef.current) {
      cancelAnimationFrame(rafIdRef.current);
    }

    rafIdRef.current = requestAnimationFrame(() => {
      // 重置高度以获取真实的 scrollHeight
      textarea.style.height = "auto";

      const newHeight = Math.max(minHeight, Math.min(maxHeight, textarea.scrollHeight));

      // 只有高度真正变化时才更新 DOM
      if (newHeight !== lastHeightRef.current) {
        textarea.style.height = `${newHeight}px`;
        lastHeightRef.current = newHeight;
      }

      rafIdRef.current = null;
    });
  }, [textareaRef, minHeight, maxHeight]);

  // 防抖版本的高度更新
  const updateHeightDebounced = useCallback(() => {
    if (timeoutIdRef.current) {
      clearTimeout(timeoutIdRef.current);
    }

    timeoutIdRef.current = setTimeout(() => {
      updateHeight();
      timeoutIdRef.current = null;
    }, debounceDelay);
  }, [updateHeight, debounceDelay]);

  // 重置高度
  const resetHeight = useCallback(() => {
    const textarea = textareaRef.current;
    if (!textarea) return;

    textarea.style.height = "auto";
    lastHeightRef.current = minHeight;
  }, [textareaRef, minHeight]);

  // 获取当前高度
  const getCurrentHeight = useCallback(() => {
    return lastHeightRef.current;
  }, []);

  // 清理
  useEffect(() => {
    return () => {
      if (rafIdRef.current) {
        cancelAnimationFrame(rafIdRef.current);
      }
      if (timeoutIdRef.current) {
        clearTimeout(timeoutIdRef.current);
      }
    };
  }, []);

  return {
    updateHeight: debounce ? updateHeightDebounced : updateHeight,
    resetHeight,
    getCurrentHeight,
  };
}

/**
 * 使用 ResizeObserver 的更精确版本
 * 适用于需要响应外部尺寸变化的场景
 */
export function useResizeObserverHeight(
  textareaRef: React.RefObject<HTMLTextAreaElement>,
  options: UseVirtualHeightOptions = {},
): UseVirtualHeightReturn {
  const { minHeight = 44, maxHeight = 400 } = options;
  const lastHeightRef = useRef(0);

  const updateHeight = useCallback(() => {
    const textarea = textareaRef.current;
    if (!textarea) return;

    textarea.style.height = "auto";
    const newHeight = Math.max(minHeight, Math.min(maxHeight, textarea.scrollHeight));

    if (newHeight !== lastHeightRef.current) {
      textarea.style.height = `${newHeight}px`;
      lastHeightRef.current = newHeight;
    }
  }, [textareaRef, minHeight, maxHeight]);

  const resetHeight = useCallback(() => {
    const textarea = textareaRef.current;
    if (!textarea) return;

    textarea.style.height = "auto";
    lastHeightRef.current = minHeight;
  }, [textareaRef, minHeight]);

  const getCurrentHeight = useCallback(() => lastHeightRef.current, []);

  useEffect(() => {
    const textarea = textareaRef.current;
    if (!textarea) return;

    // 使用 ResizeObserver 监听内容区域变化
    const resizeObserver = new ResizeObserver((entries) => {
      for (const entry of entries) {
        if (entry.target === textarea) {
          updateHeight();
        }
      }
    });

    resizeObserver.observe(textarea);

    return () => {
      resizeObserver.disconnect();
    };
  }, [textareaRef, updateHeight]);

  return {
    updateHeight,
    resetHeight,
    getCurrentHeight,
  };
}
