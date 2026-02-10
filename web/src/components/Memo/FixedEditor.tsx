/**
 * FixedEditor - Fixed Bottom Memo Editor
 *
 * Adapted from ChatInput pattern for memo editing.
 * Features:
 * - Sticky at bottom of main content area
 * - Auto-resize textarea
 * - Keyboard shortcuts (Ctrl/Cmd+Enter to send)
 * - Mobile keyboard adaptation
 * - Always visible as part of the page
 */

import { memo, useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import MemoEditor from "@/components/MemoEditor";
import { cn } from "@/lib/utils";

export interface FixedEditorProps {
  placeholder?: string;
  className?: string;
}

export const FixedEditor = memo(function FixedEditor({ placeholder, className }: FixedEditorProps) {
  const { t } = useTranslation();
  const [keyboardHeight, setKeyboardHeight] = useState(0);
  const containerRef = useRef<HTMLDivElement>(null);

  // Handle mobile keyboard height
  useEffect(() => {
    if (typeof window === "undefined" || !window.visualViewport) return;

    const handleResize = () => {
      const currentHeight = window.visualViewport?.height ?? window.innerHeight;
      const windowHeight = window.innerHeight;
      const diff = windowHeight - currentHeight;

      // Only update if significant change (avoid jitter)
      if (diff > 50) {
        setKeyboardHeight(diff);
      } else if (diff < 10) {
        setKeyboardHeight(0);
      }
    };

    window.visualViewport.addEventListener("resize", handleResize);
    return () => window.visualViewport?.removeEventListener("resize", handleResize);
  }, []);

  return (
    <div
      ref={containerRef}
      className={cn(
        "sticky bottom-0 left-0 right-0 z-50",
        "bg-background border-t border-border",
        // Padding for mobile keyboard
        keyboardHeight > 0 && "pb-safe",
        className,
      )}
      style={{ paddingBottom: keyboardHeight > 0 ? `${keyboardHeight}px` : undefined }}
    >
      <div className="mx-auto max-w-[100rem] px-4 py-3">
        {/* Memo Editor */}
        <MemoEditor
          placeholder={placeholder || t("editor.any-thoughts")}
          onConfirm={() => {
            // Trigger memo list refresh
            window.dispatchEvent(new Event("memo-created"));
          }}
        />
      </div>
    </div>
  );
});

FixedEditor.displayName = "FixedEditor";
