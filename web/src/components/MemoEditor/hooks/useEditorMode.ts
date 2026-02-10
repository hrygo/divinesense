import { useCallback, useMemo, useState } from "react";
import useMediaQuery from "@/hooks/useMediaQuery";

/**
 * Editor mode types for progressive disclosure
 * 快速模式: Minimal input for quick capture
 * 标准模式: Full toolbar with all features
 * 聚焦模式: Fullscreen distraction-free editing
 */
export type EditorMode = "quick" | "standard" | "focus";

interface UseEditorModeOptions {
  /** Default mode (auto-detected by platform if not specified) */
  defaultMode?: EditorMode;
  /** Whether to enable focus mode (default: true) */
  enableFocusMode?: boolean;
}

interface EditorModeState {
  /** Current editor mode */
  mode: EditorMode;
  /** Whether mobile toolbar sheet is open */
  isMobileToolbarOpen: boolean;
  /** Whether current device is mobile (< md breakpoint) */
  isMobile: boolean;
  /** Set editor mode */
  setMode: (mode: EditorMode) => void;
  /** Expand to standard mode */
  expandToStandard: () => void;
  /** Collapse to quick mode */
  collapseToQuick: () => void;
  /** Toggle focus mode */
  toggleFocusMode: () => void;
  /** Open mobile toolbar */
  openMobileToolbar: () => void;
  /** Close mobile toolbar */
  closeMobileToolbar: () => void;
  /** Toggle mobile toolbar */
  toggleMobileToolbar: () => void;
}

/**
 * Hook for managing memo editor mode state
 * Manages progressive disclosure: quick → standard → focus
 *
 * @example
 * ```tsx
 * const { mode, setMode, expandToStandard, isMobile } = useEditorMode();
 * ```
 */
export function useEditorMode(options: UseEditorModeOptions = {}): EditorModeState {
  const { defaultMode, enableFocusMode = true } = options;

  // Mobile detection: < md breakpoint (768px)
  // useMediaQuery returns true when breakpoint matches (min-width)
  // So we need to negate it: isMobile = NOT (matches md breakpoint)
  const mdMatches = useMediaQuery("md");
  const isMobile = !mdMatches;

  // Auto-detect default mode based on platform
  const autoDefaultMode: EditorMode = useMemo(() => {
    if (defaultMode) return defaultMode;
    // Mobile defaults to quick mode, PC to standard mode
    return isMobile ? "quick" : "standard";
  }, [defaultMode, isMobile]);

  const [mode, setModeState] = useState<EditorMode>(autoDefaultMode);
  const [isMobileToolbarOpen, setMobileToolbarOpen] = useState(false);

  // Update mode when platform changes (responsive default)
  const resolvedMode = useMemo(() => {
    // If user hasn't explicitly set a mode (or we want auto behavior)
    if (isMobile && mode === "standard") {
      // Allow standard mode on mobile if user expanded it
      return mode;
    }
    return mode;
  }, [mode, isMobile]);

  const setMode = useCallback(
    (newMode: EditorMode) => {
      if (!enableFocusMode && newMode === "focus") {
        // Focus mode disabled, default to standard
        setModeState("standard");
        return;
      }
      setModeState(newMode);
    },
    [enableFocusMode],
  );

  const expandToStandard = useCallback(() => {
    setMode("standard");
  }, [setMode]);

  const collapseToQuick = useCallback(() => {
    setMode("quick");
    // Close mobile toolbar when collapsing
    setMobileToolbarOpen(false);
  }, []);

  const toggleFocusMode = useCallback(() => {
    setModeState((prev) => (prev === "focus" ? "standard" : "focus"));
  }, []);

  const openMobileToolbar = useCallback(() => {
    setMobileToolbarOpen(true);
  }, []);

  const closeMobileToolbar = useCallback(() => {
    setMobileToolbarOpen(false);
  }, []);

  const toggleMobileToolbar = useCallback(() => {
    setMobileToolbarOpen((prev) => !prev);
  }, []);

  return {
    mode: resolvedMode,
    isMobileToolbarOpen,
    isMobile,
    setMode,
    expandToStandard,
    collapseToQuick,
    toggleFocusMode,
    openMobileToolbar,
    closeMobileToolbar,
    toggleMobileToolbar,
  };
}
