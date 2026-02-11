/**
 * useCachingCaretCoordinates - 带缓存的光标位置计算
 *
 * 性能优化策略：
 * 1. 缓存光标位置，避免重复计算
 * 2. 只有当光标位置或内容长度变化时才重新计算
 * 3. 使用 WeakMap 存储计算结果
 */

import { useCallback, useRef } from "react";
import getCaretCoordinates from "textarea-caret";

interface CaretInfo {
  top: number;
  left: number;
  height: number;
  cursorPosition: number;
  contentLength: number;
  timestamp?: number;
}

interface UseCachingCaretCoordinatesOptions {
  /** 缓存 TTL (ms) */
  cacheTTL?: number;
}

/**
 * 优化的光标位置计算 Hook
 *
 * @example
 * ```tsx
 * const { getCaretPosition, scrollToCaret } = useCachingCaretCoordinates(editorRef);
 * ```
 */
export function useCachingCaretCoordinates(
  textareaRef: React.RefObject<HTMLTextAreaElement>,
  options: UseCachingCaretCoordinatesOptions = {},
) {
  const { cacheTTL = 100 } = options;

  const cacheRef = useRef<{
    caretInfo: CaretInfo | null;
    timestamp: number;
  }>({
    caretInfo: null,
    timestamp: 0,
  });

  /**
   * 获取光标位置（带缓存）
   */
  const getCaretPosition = useCallback((): CaretInfo | null => {
    const textarea = textareaRef.current;
    if (!textarea) return null;

    const cursorPosition = textarea.selectionStart;
    const contentLength = textarea.value.length;
    const now = Date.now();

    // 检查缓存是否有效
    if (
      cacheRef.current.caretInfo &&
      cacheRef.current.caretInfo.cursorPosition === cursorPosition &&
      cacheRef.current.caretInfo.contentLength === contentLength &&
      now - cacheRef.current.timestamp < cacheTTL
    ) {
      return cacheRef.current.caretInfo;
    }

    // 计算新的光标位置
    const coords = getCaretCoordinates(textarea, cursorPosition);
    const caretInfo: CaretInfo = {
      top: coords.top,
      left: coords.left,
      height: coords.height,
      cursorPosition,
      contentLength,
    };

    // 更新缓存
    cacheRef.current = {
      caretInfo,
      timestamp: now,
    };

    return caretInfo;
  }, [textareaRef, cacheTTL]);

  /**
   * 滚动到光标位置（优化版）
   */
  const scrollToCaret = useCallback(
    (options: { force?: boolean } = {}) => {
      const textarea = textareaRef.current;
      if (!textarea) return;

      const caretInfo = getCaretPosition();
      if (!caretInfo) return;

      const { force = false } = options;
      const lineHeight = parseFloat(getComputedStyle(textarea).lineHeight) || 24;
      const viewportBottom = textarea.scrollTop + textarea.clientHeight;

      // 只有当光标接近或超出底部边缘时才滚动
      if (force || caretInfo.top + lineHeight * 2 > viewportBottom) {
        textarea.scrollTop = Math.max(0, caretInfo.top - textarea.clientHeight / 2);
      }
    },
    [textareaRef, getCaretPosition],
  );

  /**
   * 清除缓存
   */
  const invalidateCache = useCallback(() => {
    cacheRef.current.caretInfo = null;
    cacheRef.current.timestamp = 0;
  }, []);

  return {
    getCaretPosition,
    scrollToCaret,
    invalidateCache,
  };
}

/**
 * 批量光标位置计算器
 * 用于需要频繁查询光标位置的场景
 */
export class CaretPositionBatcher {
  private cache = new Map<number, CaretInfo>();
  private textarea: HTMLTextAreaElement;
  private readonly cacheTTL: number;

  constructor(textarea: HTMLTextAreaElement, cacheTTL = 100) {
    this.textarea = textarea;
    this.cacheTTL = cacheTTL;
  }

  /**
   * 批量获取多个位置的坐标
   */
  getPositions(positions: number[]): Map<number, CaretInfo> {
    const result = new Map<number, CaretInfo>();
    const now = Date.now();

    for (const pos of positions) {
      const cached = this.cache.get(pos);
      if (cached && cached.timestamp && now - cached.timestamp < this.cacheTTL) {
        result.set(pos, cached);
      } else {
        const coords = getCaretCoordinates(this.textarea, pos);
        const caretInfo: CaretInfo & { timestamp: number } = {
          top: coords.top,
          left: coords.left,
          height: coords.height,
          cursorPosition: pos,
          contentLength: this.textarea.value.length,
          timestamp: now,
        };
        this.cache.set(pos, caretInfo);
        result.set(pos, caretInfo);
      }
    }

    return result;
  }

  /**
   * 清除所有缓存
   */
  clear() {
    this.cache.clear();
  }
}
