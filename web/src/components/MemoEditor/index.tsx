import { useRef } from "react";
import { MEMO_EDITOR_CARD } from "@/components/ui/card/constants";
import { useAuth } from "@/contexts/AuthContext";
import useCurrentUser from "@/hooks/useCurrentUser";
import { cn } from "@/lib/utils";
import { useTranslate } from "@/utils/i18n";
import { convertVisibilityFromString } from "@/utils/memo";
import { EditorContent, EditorMetadata, FocusModeExitButton, FocusModeOverlay } from "./components";
import { FOCUS_MODE_STYLES } from "./constants";
import { useAutoSave, useFocusMode, useKeyboard, useMemoInit, useVirtualKeyboard } from "./hooks";
import { EditorProvider, useEditorContext } from "./state";
import type { EditorRefActions } from "./types/editor";
import type { MemoEditorProps } from "./types/memo-editor";

/**
 * 新的简化版 MemoEditor - 使用插件系统
 */

const MemoEditor = (props: MemoEditorProps) => {
  const { className, cacheKey, memoName, parentMemoName, autoFocus, placeholder, onSubmit } = props;

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
      />
    </EditorProvider>
  );
};

const MemoEditorImpl: React.FC<MemoEditorProps> = ({ className, cacheKey, memoName, autoFocus, placeholder, onSubmit }) => {
  const t = useTranslate();
  const currentUser = useCurrentUser();
  const editorRef = useRef<EditorRefActions>(null);
  const { state, actions, dispatch } = useEditorContext();
  const { userGeneralSetting } = useAuth();

  // Get default visibility from user settings
  const defaultVisibility = userGeneralSetting?.memoVisibility ? convertVisibilityFromString(userGeneralSetting.memoVisibility) : undefined;

  useMemoInit(editorRef, memoName, cacheKey, currentUser?.name ?? "", autoFocus, defaultVisibility);

  // Auto-save content to localStorage
  useAutoSave(state.content, currentUser?.name ?? "", cacheKey);

  // Track virtual keyboard height for mobile
  const keyboardHeight = useVirtualKeyboard();
  // Focus mode management with body scroll lock
  useFocusMode(state.ui.isFocusMode);

  const handleToggleFocusMode = () => {
    dispatch(actions.toggleFocusMode());
  };

  // Create a save wrapper that calls onSubmit with current content
  const handleKeyboardSave = () => {
    onSubmit?.(state.content);
  };

  useKeyboard(editorRef, { onSave: handleKeyboardSave });

  return (
    <>
      <FocusModeOverlay isActive={state.ui.isFocusMode} onToggle={handleToggleFocusMode} />

      {/*
        Layout structure:
        - Uses justify-between to push content to top and bottom
        - In focus mode: becomes fixed with specific spacing, editor grows to fill space
        - In normal mode: stays relative with max-height constraint
      */}
      <div
        className={cn(
          MEMO_EDITOR_CARD,
          FOCUS_MODE_STYLES.transition,
          state.ui.isFocusMode && cn(FOCUS_MODE_STYLES.container.base, FOCUS_MODE_STYLES.container.spacing),
          className,
        )}
        style={{
          paddingBottom: keyboardHeight > 0 ? `${keyboardHeight + 16}px` : undefined,
        }}
      >
        {/* Exit button is absolutely positioned in top-right corner when active */}
        <FocusModeExitButton isActive={state.ui.isFocusMode} onToggle={handleToggleFocusMode} title={t("editor.exit-focus-mode")} />

        {/* Editor content grows to fill available space in focus mode */}
        <EditorContent ref={editorRef} placeholder={placeholder ?? ""} />

        {/* Metadata and toolbar grouped together at bottom */}
        <div className="w-full flex flex-col gap-2">
          <EditorMetadata memoName={memoName} />
        </div>
      </div>
    </>
  );
};

export default MemoEditor;

// 重新导出 FocusModeEditor 以保持兼容
export { default as FocusModeEditor } from "./FocusModeEditor";
export type { EditorMode } from "./hooks/useEditorMode";
export { useEditorMode } from "./hooks/useEditorMode";
export type { EditorRefActions } from "./types/editor";
export type { MemoEditorProps } from "./types/memo-editor";
