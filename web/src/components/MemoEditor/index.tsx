import { Sparkles } from "lucide-react";
import { forwardRef, useCallback, useEffect, useRef, useState } from "react";
import { Textarea } from "@/components/ui/textarea";
import { useAuth } from "@/contexts/AuthContext";
import useCurrentUser from "@/hooks/useCurrentUser";
import useMediaQuery from "@/hooks/useMediaQuery";
import { cn } from "@/lib/utils";
import type { Memo } from "@/types/proto/api/v1/memo_service_pb";
import { Visibility } from "@/types/proto/api/v1/memo_service_pb";
import { useTranslate } from "@/utils/i18n";
import { convertVisibilityFromString } from "@/utils/memo";
import { EditorMetadata, EditorToolbar, FocusModeExitButton, FocusModeOverlay, LinkMemoDialog, MobileToolsSheet } from "./components";
import { FOCUS_MODE_STYLES } from "./constants";
import { useAutoSave, useFocusMode, useKeyboard, useMemoInit } from "./hooks";
import { EditorProvider, useEditorContext } from "./state";
import type { EditorRefActions } from "./types/editor";
import type { MemoEditorProps } from "./types/memo-editor";

/**
 * Enhanced MemoEditor - Full-featured editor with toolbar
 *
 * Design inspired by AIChat ChatInput:
 * - Positioned at bottom of main content area (not fixed to viewport)
 * - Uses existing components: EditorToolbar, EditorMetadata
 * - Auto-growing textarea with max height
 * - Mobile keyboard adaptation
 * - Focus mode for distraction-free writing
 */

const MemoEditor = (props: MemoEditorProps) => {
  const { className, cacheKey, memoName, parentMemoName, autoFocus, placeholder, onSubmit, onConfirm, onCancel } = props;

  return (
    <EditorProvider>
      <MemoEditorImpl
        className={className}
        cacheKey={cacheKey}
        memoName={memoName}
        parentMemoName={parentMemoName}
        autoFocus={autoFocus}
        placeholder={placeholder}
        onSubmit={onSubmit}
        onConfirm={onConfirm}
        onCancel={onCancel}
      />
    </EditorProvider>
  );
};

const MemoEditorImpl = forwardRef<HTMLDivElement, MemoEditorProps>(
  ({ className, cacheKey, memoName, autoFocus, placeholder, onSubmit, onConfirm, onCancel: _onCancel }, ref) => {
    const t = useTranslate();
    const currentUser = useCurrentUser();
    const textareaRef = useRef<HTMLTextAreaElement>(null);
    const editorRef = useRef<EditorRefActions>(null);
    const { state, actions, dispatch } = useEditorContext();
    const { userGeneralSetting } = useAuth();
    const [isSaving, setIsSaving] = useState(false);
    const [keyboardHeight, setKeyboardHeight] = useState(0);
    const lastHeightRef = useRef(0);
    const [linkDialogOpen, setLinkDialogOpen] = useState(false);
    const [linkSearchText, setLinkSearchText] = useState("");
    const [filteredMemos] = useState<Memo[]>([]);
    const [mobileToolsOpen, setMobileToolsOpen] = useState(false);
    const rafIdRef = useRef<ReturnType<typeof requestAnimationFrame> | null>(null);

    // Get default visibility from user settings
    const defaultVisibility = userGeneralSetting?.memoVisibility
      ? convertVisibilityFromString(userGeneralSetting.memoVisibility)
      : undefined;

    useMemoInit(editorRef, memoName, cacheKey, currentUser?.name ?? "", autoFocus, defaultVisibility);

    // Auto-save content to localStorage
    useAutoSave(state.content, currentUser?.name ?? "", cacheKey);

    // Focus mode management
    useFocusMode(state.ui.isFocusMode);

    // Handle keyboard save (Ctrl/Cmd + Enter)
    const handleKeyboardSave = useCallback(() => {
      handleSave();
    }, []);

    useKeyboard(editorRef, { onSave: handleKeyboardSave });

    // Sync editorRef with textareaRef for external access
    useEffect(() => {
      if (textareaRef.current && editorRef.current) {
        editorRef.current.focus = () => textareaRef.current?.focus();
      }
    }, [editorRef, textareaRef]);

    // Handle mobile keyboard visibility with debouncing
    useEffect(() => {
      if (typeof window === "undefined" || !window.visualViewport) return;

      let timeoutId: ReturnType<typeof setTimeout> | null = null;
      let lastHeight = 0;

      const handleResize = () => {
        const viewport = window.visualViewport;
        if (!viewport) return;

        const currentHeight = viewport.height;
        if (Math.abs(currentHeight - lastHeight) < 10) {
          return;
        }
        lastHeight = currentHeight;

        if (timeoutId) clearTimeout(timeoutId);
        timeoutId = setTimeout(() => {
          const windowHeight = window.innerHeight;
          const keyboardVisible = currentHeight < windowHeight * 0.85;
          const newKeyboardHeight = keyboardVisible ? windowHeight - currentHeight : 0;
          setKeyboardHeight(newKeyboardHeight);
        }, 100);
      };

      window.visualViewport.addEventListener("resize", handleResize);
      return () => {
        if (timeoutId) clearTimeout(timeoutId);
        window.visualViewport?.removeEventListener("resize", handleResize);
      };
    }, []);

    // Auto-resize textarea based on content
    const handleInput = useCallback((e: React.FormEvent<HTMLTextAreaElement>) => {
      const target = e.target as HTMLTextAreaElement;

      if (rafIdRef.current) {
        cancelAnimationFrame(rafIdRef.current);
      }

      rafIdRef.current = requestAnimationFrame(() => {
        if (!target || !textareaRef.current) return;

        const currentScrollHeight = target.scrollHeight;
        const maxHeight = 120;
        const newHeight = Math.min(currentScrollHeight, maxHeight);

        if (newHeight !== lastHeightRef.current) {
          lastHeightRef.current = newHeight;
          target.style.height = `${newHeight}px`;
        }

        rafIdRef.current = null;
      });
    }, []);

    // Reset height when value changes externally
    useEffect(() => {
      if (textareaRef.current && !state.content) {
        textareaRef.current.style.height = "auto";
      }
    }, [state.content]);

    const handleToggleFocusMode = () => {
      dispatch(actions.toggleFocusMode());
    };

    const handleSave = async () => {
      if (isSaving) return;
      setIsSaving(true);
      try {
        if (onSubmit) {
          onSubmit(state.content);
        }
        if (onConfirm) {
          onConfirm(memoName || "");
        }
      } finally {
        setIsSaving(false);
      }
    };

    const handleKeyDown = useCallback(
      (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
        if (e.key === "Enter") {
          if (e.ctrlKey || e.metaKey) {
            e.preventDefault();
            handleSave();
          }
        }
      },
      [handleSave],
    );

    const handleUploadAttachment = () => {
      // TODO: Implement file upload
      console.log("Upload attachment");
    };

    const handleLinkMemo = () => {
      setLinkDialogOpen(true);
    };

    const handleSelectMemo = (memo: Memo) => {
      // Add relation to state
      // TODO: Implement proper relation creation
      console.log("Link memo:", memo);
      setLinkDialogOpen(false);
    };

    const md = useMediaQuery("md");

    const handleAddLocation = () => {
      // TODO: Implement location picker
      console.log("Add location");
    };

    const handleVisibilityChange = (visibility: Visibility) => {
      dispatch(actions.setMetadata({ visibility }));
    };

    return (
      <>
        <FocusModeOverlay isActive={state.ui.isFocusMode} onToggle={handleToggleFocusMode} />

        {/*
          Bottom editor - inspired by ChatInput design
          - Top border with gradient background
          - shrink-0 to stay at bottom of flex container
          - left-16 to avoid sidebar on mobile
        */}
        <div
          ref={ref}
          className={cn(
            "shrink-0 border-t border-border/50",
            "bg-gradient-to-b from-background/95 to-background",
            FOCUS_MODE_STYLES.transition,
            state.ui.isFocusMode && cn(FOCUS_MODE_STYLES.container.base, FOCUS_MODE_STYLES.container.spacing),
            className,
          )}
          style={{
            paddingBottom: keyboardHeight > 0 ? `${keyboardHeight}px` : undefined,
          }}
        >
          {/* Exit button in focus mode */}
          <FocusModeExitButton isActive={state.ui.isFocusMode} onToggle={handleToggleFocusMode} title={t("editor.exit-focus-mode")} />

          <div className="mx-auto max-w-3xl lg:max-w-4xl xl:max-w-5xl 2xl:max-w-6xl px-4 sm:px-6 py-3 sm:py-4">
            {/* Metadata - attachments, relations, location */}
            <EditorMetadata memoName={memoName} />

            {/* Toolbar - with attachment and focus mode buttons */}
            {!state.ui.isFocusMode && (
              <EditorToolbar
                onCancel={_onCancel}
                onUploadAttachment={handleUploadAttachment}
                onLinkMemo={handleLinkMemo}
                onToggleFocusMode={handleToggleFocusMode}
                onVisibilityChange={(visibility) => {
                  dispatch(actions.setMetadata({ visibility }));
                }}
                onOpenMobileTools={() => setMobileToolsOpen(true)}
                currentVisibility={state.metadata.visibility}
              />
            )}

            {/* Input Box */}
            <div
              className={cn(
                "flex items-end gap-2 md:gap-3 p-2.5 md:p-3 rounded-lg border shadow-sm transition-colors",
                "bg-muted/30 border-border/50",
                "focus-within:ring-2 focus-within:ring-primary/20 focus-within:border-primary/50",
              )}
              style={{ contain: "layout" }}
            >
              <Textarea
                ref={textareaRef}
                value={state.content}
                onChange={(e) => {
                  dispatch(actions.updateContent(e.target.value));
                  handleInput(e);
                }}
                onKeyDown={handleKeyDown}
                placeholder={placeholder ?? t("editor.any-thoughts")}
                className={cn(
                  "flex-1 min-h-[44px] max-h-[120px] bg-transparent border-0 outline-none resize-none",
                  "text-sm leading-relaxed transition-colors",
                  "text-foreground placeholder:text-muted-foreground/60",
                  "focus:ring-0",
                )}
                rows={1}
              />

              {/* Save button */}
              <button
                type="button"
                onClick={handleSave}
                disabled={!state.content.trim() || isSaving}
                className={cn(
                  "shrink-0 h-11 px-4 min-w-[60px] rounded-lg transition-all",
                  "hover:scale-105 active:scale-95",
                  "text-sm font-medium",
                  isSaving
                    ? "bg-muted text-muted-foreground"
                    : state.content.trim()
                      ? "bg-primary text-primary-foreground shadow-md shadow-primary/20 hover:bg-primary/90"
                      : "bg-muted text-muted-foreground hover:bg-muted/80",
                  "disabled:opacity-50 disabled:cursor-not-allowed disabled:hover:scale-100",
                )}
                aria-label={t("editor.save")}
              >
                {isSaving ? <Sparkles className="w-5 h-5 opacity-50 animate-pulse mx-auto" /> : <span>{t("editor.save")}</span>}
              </button>
            </div>
          </div>
        </div>

        {/* Link Memo Dialog */}
        <LinkMemoDialog
          open={linkDialogOpen}
          onOpenChange={setLinkDialogOpen}
          searchText={linkSearchText}
          onSearchChange={setLinkSearchText}
          filteredMemos={filteredMemos}
          isFetching={false}
          onSelectMemo={handleSelectMemo}
        />

        {/* Mobile Tools Sheet - only show on mobile */}
        {!md && (
          <MobileToolsSheet
            open={mobileToolsOpen}
            onOpenChange={setMobileToolsOpen}
            onUploadFile={handleUploadAttachment}
            onLinkMemo={handleLinkMemo}
            onAddLocation={handleAddLocation}
            onVisibilityChange={handleVisibilityChange}
            keyboardHeight={keyboardHeight}
          />
        )}
      </>
    );
  },
);

MemoEditorImpl.displayName = "MemoEditorImpl";

export default MemoEditor;

// Re-export for compatibility
export { default as FocusModeEditor } from "./FocusModeEditor";
export type { EditorRefActions } from "./types/editor";
export type { MemoEditorProps } from "./types/memo-editor";
