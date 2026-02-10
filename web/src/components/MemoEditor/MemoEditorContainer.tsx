import { create } from "@bufbuild/protobuf";
import { useMutation } from "@tanstack/react-query";
import { useCallback, useEffect, useState } from "react";
import { toast } from "react-hot-toast";
import { memoServiceClient } from "@/connect";
import { handleError } from "@/lib/error";
import { cn } from "@/lib/utils";
import { MemoSchema, Visibility } from "@/types/proto/api/v1/memo_service_pb";
import { FocusModeEditor } from "./FocusModeEditor";
import { useEditorMode } from "./hooks/useEditorMode";
import { MobileToolbarSheet } from "./MobileToolbarSheet";
import { QuickInput } from "./QuickInput";
import { StandardEditor } from "./StandardEditor";

interface MemoEditorContainerProps {
  /** Optional initial content for editing */
  initialContent?: string;
  /** Callback when memo is successfully created/updated */
  onSuccess?: (memoName: string) => void;
  /** Custom placeholder text */
  placeholder?: string;
  /** Additional CSS classes */
  className?: string;
}

/**
 * MemoEditorContainer - Responsive editor with progressive disclosure
 *
 * Mobile: QuickInput + MobileToolbarSheet (expandable)
 * PC: StandardEditor with full toolbar
 *
 * Modes:
 * - quick: Minimal input for fast capture
 * - standard: Full toolbar with all features
 * - focus: Fullscreen distraction-free editing
 */
export function MemoEditorContainer({ initialContent = "", onSuccess, placeholder, className }: MemoEditorContainerProps) {
  const [content, setContent] = useState(initialContent);
  const { mode, isMobile, isMobileToolbarOpen, expandToStandard, openMobileToolbar, closeMobileToolbar, collapseToQuick, toggleFocusMode } =
    useEditorMode();

  // Handle ESC key to exit focus mode
  useEffect(() => {
    const handleEscape = (e: KeyboardEvent) => {
      if (e.key === "Escape" && mode === "focus") {
        collapseToQuick();
      }
    };
    window.addEventListener("keydown", handleEscape);
    return () => window.removeEventListener("keydown", handleEscape);
  }, [mode, collapseToQuick]);

  // Memo creation mutation
  const createMemo = useMutation({
    mutationFn: async (contentParam: string) => {
      const memo = create(MemoSchema, {
        content: contentParam,
        visibility: Visibility.PRIVATE,
      });

      const response = await memoServiceClient.createMemo({ memo });
      return response;
    },
    onSuccess: (data) => {
      setContent("");
      onSuccess?.(data.name);
    },
    onError: (error) => {
      handleError(error, toast.error, {
        context: "Failed to create memo",
        fallbackMessage: "创建笔记失败，请重试",
      });
    },
  });

  const handleSend = () => {
    if (!content.trim()) return;
    createMemo.mutate(content);
  };

  const handleExpand = useCallback(() => {
    if (isMobile) {
      // Mobile: open toolbar sheet
      openMobileToolbar();
    } else {
      // PC: expand to standard mode
      expandToStandard();
    }
  }, [isMobile, openMobileToolbar, expandToStandard]);

  // Handle mobile toolbar actions - switch to standard mode for advanced features
  const handleToolbarAction = useCallback(() => {
    // Close the mobile toolbar sheet and switch to standard mode
    closeMobileToolbar();
    expandToStandard();
  }, [closeMobileToolbar, expandToStandard]);

  // Show FocusModeEditor for focus mode
  if (mode === "focus") {
    return (
      <FocusModeEditor
        initialContent={content}
        placeholder={placeholder}
        onExit={collapseToQuick}
        onSuccess={(memoName) => {
          setContent("");
          onSuccess?.(memoName);
        }}
      />
    );
  }

  // Show StandardEditor for standard mode
  if (mode === "standard") {
    return (
      <div className={cn("memo-editor-container max-w-3xl mx-auto", className)}>
        <StandardEditor
          initialContent={content}
          placeholder={placeholder}
          onSuccess={(memoName) => {
            setContent("");
            onSuccess?.(memoName);
            collapseToQuick(); // Return to quick mode after save
          }}
          onCancel={() => {
            collapseToQuick(); // Return to quick mode on cancel
          }}
          onToggleFocusMode={toggleFocusMode}
        />
      </div>
    );
  }

  // Quick mode (default)
  return (
    <div className={cn("memo-editor-container", className)}>
      <QuickInput
        value={content}
        onChange={setContent}
        onSend={handleSend}
        onExpand={handleExpand}
        disabled={createMemo.isPending}
        placeholder={placeholder}
        showExpandButton
      />

      {/* Mobile toolbar sheet - only shown on mobile */}
      {isMobile && (
        <MobileToolbarSheet
          open={isMobileToolbarOpen}
          onOpenChange={(open) => {
            if (!open) {
              closeMobileToolbar();
            }
          }}
          onUploadFile={handleToolbarAction}
          onLinkMemo={handleToolbarAction}
          onAddLocation={handleToolbarAction}
        />
      )}
    </div>
  );
}
