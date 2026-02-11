/**
 * useFocusModeEnhanced - 增强的专注模式 Hook
 *
 * UX 优化策略：
 * 1. 支持多种退出方式（ESC、点击遮罩、手势）
 * 2. 进入/退出动画状态管理
 * 3. 保存进入焦点模式前的滚动位置
 * 4. 支持键盘快捷键配置
 */

import { useCallback, useEffect, useRef, useState } from "react";

interface FocusModeState {
  /** 是否在焦点模式中 */
  isActive: boolean;
  /** 是否正在进入动画 */
  isEntering: boolean;
  /** 是否正在退出动画 */
  isExiting: boolean;
}

interface UseFocusModeEnhancedOptions {
  /** ESC 键退出（默认: true） */
  exitOnEscape?: boolean;
  /** 点击遮罩退出（默认: true） */
  exitOnBackdropClick?: boolean;
  /** 进入动画时长 (ms) */
  enterDuration?: number;
  /** 退出动画时长 (ms) */
  exitDuration?: number;
  /** 进入焦点模式时的回调 */
  onEnter?: () => void;
  /** 退出焦点模式时的回调 */
  onExit?: () => void;
  /** 状态变化回调 */
  onStateChange?: (state: FocusModeState) => void;
}

/**
 * 增强的焦点模式 Hook
 *
 * @example
 * ```tsx
 * const focusMode = useFocusModeEnhanced({
 *   onEnter: () => document.body.style.overflow = 'hidden',
 *   onExit: () => document.body.style.overflow = '',
 * });
 * ```
 */
export function useFocusModeEnhanced(options: UseFocusModeEnhancedOptions = {}): FocusModeState & {
  /** 进入焦点模式 */
  enter: () => void;
  /** 退出焦点模式 */
  exit: () => void;
  /** 切换焦点模式 */
  toggle: () => void;
  /** 处理遮罩点击 */
  handleBackdropClick: (e: React.MouseEvent) => void;
} {
  const {
    exitOnEscape = true,
    exitOnBackdropClick = true,
    enterDuration = 300,
    exitDuration = 200,
    onEnter,
    onExit,
    onStateChange,
  } = options;

  const [isActive, setIsActive] = useState(false);
  const [isEntering, setIsEntering] = useState(false);
  const [isExiting, setIsExiting] = useState(false);

  // 保存进入焦点模式前的滚动位置
  const scrollPositionRef = useRef(0);

  // 退出动画定时器
  const exitTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  // 当前状态对象
  const currentState: FocusModeState = {
    isActive,
    isEntering,
    isExiting,
  };

  // 通知状态变化
  useEffect(() => {
    onStateChange?.(currentState);
  }, [currentState, onStateChange]);

  /**
   * 进入焦点模式
   */
  const enter = useCallback(() => {
    if (isActive) return;

    // 保存当前滚动位置
    scrollPositionRef.current = window.scrollY;

    setIsActive(true);
    setIsEntering(true);

    // 锁定背景滚动
    document.body.style.overflow = "hidden";

    onEnter?.();

    // 进入动画完成后
    const timer = setTimeout(() => {
      setIsEntering(false);
    }, enterDuration);

    return () => clearTimeout(timer);
  }, [isActive, enterDuration, onEnter]);

  /**
   * 退出焦点模式
   */
  const exit = useCallback(() => {
    if (!isActive) return;

    setIsExiting(true);

    // 退出动画完成后
    if (exitTimerRef.current) {
      clearTimeout(exitTimerRef.current);
    }

    exitTimerRef.current = setTimeout(() => {
      setIsActive(false);
      setIsExiting(false);

      // 恢复背景滚动
      document.body.style.overflow = "";

      // 恢复滚动位置
      window.scrollTo(0, scrollPositionRef.current);

      onExit?.();
      exitTimerRef.current = null;
    }, exitDuration);
  }, [isActive, exitDuration, onExit]);

  /**
   * 切换焦点模式
   */
  const toggle = useCallback(() => {
    if (isActive) {
      exit();
    } else {
      enter();
    }
  }, [isActive, enter, exit]);

  /**
   * 处理遮罩点击
   */
  const handleBackdropClick = useCallback(
    (e: React.MouseEvent) => {
      if (exitOnBackdropClick && e.target === e.currentTarget) {
        exit();
      }
    },
    [exitOnBackdropClick, exit],
  );

  /**
   * ESC 键退出
   */
  useEffect(() => {
    if (!exitOnEscape) return;

    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === "Escape" && isActive && !isEntering) {
        e.preventDefault();
        exit();
      }
    };

    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [exitOnEscape, isActive, isEntering, exit]);

  // 清理定时器
  useEffect(() => {
    return () => {
      if (exitTimerRef.current) {
        clearTimeout(exitTimerRef.current);
      }
    };
  }, []);

  return {
    ...currentState,
    enter,
    exit,
    toggle,
    handleBackdropClick,
  };
}

/**
 * 焦点模式动画状态
 */
export const focusModeAnimationState = {
  enter: "data-[state=open]:animate-in data-[state=open]:fade-in-0 data-[state=open]:zoom-in-95",
  exit: "data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=closed]:zoom-out-95",
  content:
    "data-[state=open]:animate-in data-[state=open]:fade-in-0 data-[state=open]:slide-in-from-bottom-4 data-[state=open]:duration-300",
} as const;

/**
 * 焦点模式 CSS 类
 */
export const focusModeClasses = {
  backdrop: "fixed inset-0 bg-black/80 backdrop-blur-sm z-50 transition-opacity duration-300",
  container: "fixed inset-4 z-50 flex items-center justify-center",
  content: "w-full max-w-5xl max-h-full overflow-hidden bg-background rounded-lg shadow-2xl border",
  header: "flex items-center justify-between p-4 border-b",
  body: "p-4 overflow-y-auto",
  footer: "flex items-center justify-between p-4 border-t",
} as const;
