/**
 * useKeyboardNav Hook
 *
 * Manages keyboard navigation for UnifiedMessageBlock.
 * Implements Phase 4 accessibility: Tab focus chain and shortcuts.
 *
 * Phase 4: Accessibility Improvement
 */

import { useCallback, useEffect, useRef } from "react";

export interface UseKeyboardNavOptions {
  /** Block ID for navigation */
  blockId: string;
  /** Whether this block can receive keyboard focus */
  isFocusable?: boolean;
  /** Callback when keyboard shortcut is triggered */
  onShortcut?: (action: string) => void;
  /** Reference to the block element */
  blockRef?: React.RefObject<HTMLElement>;
}

export interface UseKeyboardNavReturn {
  /** Props to spread on the block container */
  keyboardProps: {
    tabIndex: number;
    onKeyDown: (e: React.KeyboardEvent) => void;
    "data-block-id"?: string;
  };
  /** Programmatically focus this block */
  focusBlock: () => void;
}

/**
 * Mapping of keyboard shortcuts to actions
 */
const SHORTCUT_MAP: Record<string, string> = {
  c: "copy",
  e: "edit",
  Enter: "send",
  Escape: "cancel",
} as const;

/**
 * Hook for keyboard navigation within blocks
 *
 * Enables:
 * - Tab/Shift+Tab navigation between blocks
 * - Ctrl+C to copy
 * - Ctrl+E to edit
 * - Ctrl+Enter to send
 * - Escape to cancel
 */
export function useKeyboardNav({ blockId, isFocusable = true, onShortcut, blockRef }: UseKeyboardNavOptions): UseKeyboardNavReturn {
  const internalRef = useRef<HTMLElement>(null);
  const ref = blockRef || internalRef;

  const focusBlock = useCallback(() => {
    if (ref.current && isFocusable) {
      ref.current.focus();
    }
  }, [ref, isFocusable]);

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      const key = e.key;
      const isCtrlOrCmd = e.ctrlKey || e.metaKey;

      // Handle Ctrl/Cmd shortcuts
      if (isCtrlOrCmd) {
        const action = SHORTCUT_MAP[key.toLowerCase()];
        if (action) {
          e.preventDefault();
          onShortcut?.(action);
          return;
        }

        // Ctrl+Enter for send
        if (key === "Enter") {
          e.preventDefault();
          onShortcut?.("send");
          return;
        }
      }

      // Handle Escape
      if (key === "Escape") {
        onShortcut?.("cancel");
        return;
      }
    },
    [onShortcut],
  );

  // Handle global keyboard events for block-to-block navigation
  useEffect(() => {
    if (!isFocusable) return;

    const handleGlobalKeyDown = (e: KeyboardEvent) => {
      // Tab navigation is handled natively by browser
      // This is for any additional global shortcuts
      if (e.key === "Tab" && ref.current) {
        // Ensure proper tab order by checking tabindex
        const currentFocus = document.activeElement;
        if (currentFocus === ref.current) {
          // We're leaving this block
          ref.current?.setAttribute("data-leaving", "true");
        }
      }
    };

    document.addEventListener("keydown", handleGlobalKeyDown);
    return () => document.removeEventListener("keydown", handleGlobalKeyDown);
  }, [isFocusable, ref]);

  const keyboardProps = {
    tabIndex: isFocusable ? 0 : -1,
    onKeyDown: handleKeyDown,
    "data-block-id": blockId,
  };

  return {
    keyboardProps,
    focusBlock,
  };
}
