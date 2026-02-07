/**
 * useBlockCollapse Hook
 *
 * Manages collapse state for individual blocks.
 * Implements Phase 5 performance optimization: state sinking.
 *
 * Each block manages its own collapse state internally
 * rather than being managed by a parent component.
 */

import { useCallback, useEffect, useState } from "react";
import { COLLAPSE_PREVIEW_MAX_CHARS } from "../constants";

export interface UseBlockCollapseOptions {
  /** Whether this is the latest block (should be expanded by default) */
  isLatest: boolean;
  /** Whether the block content is streaming */
  isStreaming?: boolean;
  /** Optional external control for collapse state */
  externalCollapsed?: boolean;
  /** Callback when collapse state changes */
  onCollapseChange?: (collapsed: boolean) => void;
}

export interface UseBlockCollapseReturn {
  /** Current collapse state */
  collapsed: boolean;
  /** Toggle collapse state */
  toggleCollapse: () => void;
  /** Set collapse state */
  setCollapsed: (value: boolean) => void;
  /** Generate preview text from content */
  generatePreview: (content: string) => string;
}

/**
 * Hook for managing block collapse state
 *
 * Automatically expands the latest block and collapses historical blocks.
 * Supports external control via props for parent component override.
 */
export function useBlockCollapse({
  isLatest,
  isStreaming = false,
  externalCollapsed,
  onCollapseChange,
}: UseBlockCollapseOptions): UseBlockCollapseReturn {
  // Internal state - defaults to expanded for latest, collapsed for history
  const [internalCollapsed, setInternalCollapsed] = useState(() => !isLatest);

  // Use external value if provided, otherwise use internal state
  const collapsed = externalCollapsed ?? internalCollapsed;

  // Auto-expand when streaming starts
  useEffect(() => {
    if (isStreaming && collapsed) {
      setInternalCollapsed(false);
    }
  }, [isStreaming, collapsed]);

  const toggleCollapse = useCallback(() => {
    setInternalCollapsed((prev) => {
      const newValue = !prev;
      onCollapseChange?.(newValue);
      return newValue;
    });
  }, [onCollapseChange]);

  const setCollapsed = useCallback(
    (value: boolean) => {
      setInternalCollapsed(() => {
        onCollapseChange?.(value);
        return value;
      });
    },
    [onCollapseChange],
  );

  const generatePreview = useCallback((content: string) => {
    if (!content) return "";

    // Strip markdown and HTML for preview
    const plainText = content
      .replace(/```[\s\S]*?```/g, "[Code]")
      .replace(/\[.*?\]\(.*?\)/g, "[Link]")
      .replace(/[#*_`~[\]]/g, "")
      .replace(/\s+/g, " ")
      .trim();

    if (plainText.length <= COLLAPSE_PREVIEW_MAX_CHARS) {
      return plainText;
    }

    return plainText.slice(0, COLLAPSE_PREVIEW_MAX_CHARS) + "...";
  }, []);

  return {
    collapsed,
    toggleCollapse,
    setCollapsed,
    generatePreview,
  };
}
