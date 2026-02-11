/**
 * useAutoSave - 优化的自动保存 Hook
 *
 * 性能优化：
 * - 只在内容真正变化时触发保存
 * - 使用 useCallback 减少函数重建
 * - 利用 cacheService 内置的防抖机制
 */

import { useCallback, useEffect, useRef } from "react";
import { cacheService } from "../services";

export const useAutoSave = (content: string, username: string, cacheKey: string | undefined) => {
  // 缓存上次保存的内容，避免重复保存
  const lastSavedRef = useRef(content);

  useEffect(() => {
    // 只在内容真正变化时才触发保存
    if (content !== lastSavedRef.current) {
      lastSavedRef.current = content;
      const key = cacheService.key(username, cacheKey);
      cacheService.save(key, content);
    }
  }, [content, username, cacheKey]);
};

/**
 * useAutoSaveWithFlush - 带立即刷新功能的自动保存
 *
 * 用于需要在组件卸载时立即保存的场景
 */
export const useAutoSaveWithFlush = (content: string, username: string, cacheKey: string | undefined) => {
  const lastSavedRef = useRef(content);
  const pendingSaveRef = useRef<{ key: string; content: string } | null>(null);

  // 正常的防抖保存
  useEffect(() => {
    if (content !== lastSavedRef.current) {
      lastSavedRef.current = content;
      const key = cacheService.key(username, cacheKey);
      // 保存待写入内容，用于 flush
      pendingSaveRef.current = { key, content };
      cacheService.save(key, content);
    }
  }, [content, username, cacheKey]);

  // 立即保存（绕过防抖），用于组件卸载时
  const flush = useCallback(() => {
    if (pendingSaveRef.current) {
      const { key, content: pendingContent } = pendingSaveRef.current;
      // 直接写入 localStorage，不经过防抖
      if (pendingContent.trim()) {
        localStorage.setItem(key, pendingContent);
      } else {
        localStorage.removeItem(key);
      }
      pendingSaveRef.current = null;
    }
  }, []);

  return { flush };
};
