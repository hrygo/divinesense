import { useQueryClient } from "@tanstack/react-query";
import { useRef, useState } from "react";
import { toast } from "react-hot-toast";
import { MEMO_EDITOR_CARD } from "@/components/ui/card/constants";
import { useAuth } from "@/contexts/AuthContext";
import useCurrentUser from "@/hooks/useCurrentUser";
import { memoKeys } from "@/hooks/useMemoQueries";
import { userKeys } from "@/hooks/useUserQueries";
import { handleError } from "@/lib/error";
import { cn } from "@/lib/utils";
import { useTranslate } from "@/utils/i18n";
import { convertVisibilityFromString } from "@/utils/memo";
import { EditorContent, EditorMetadata, EditorToolbar, FocusModeExitButton, FocusModeOverlay, LinkMemoDialog } from "./components";
import { FOCUS_MODE_STYLES } from "./constants";
import type { EnhancedEditorRefActions } from "./core/editor-types";
import { useAutoSave, useFocusMode, useKeyboard, useLinkMemo, useMemoInit, useVirtualKeyboard } from "./hooks";
import { cacheService, errorService, memoService, validationService } from "./services";
import { EditorProvider, useEditorContext } from "./state";
import type { MemoEditorProps } from "./types";

const MemoEditor = (props: MemoEditorProps) => {
  const { className, cacheKey, memoName, parentMemoName, autoFocus, placeholder, onConfirm, onCancel } = props;

  return (
    <EditorProvider>
      <MemoEditorImpl
        className={className}
        cacheKey={cacheKey}
        memoName={memoName}
        parentMemoName={parentMemoName}
        autoFocus={autoFocus}
        placeholder={placeholder}
        onConfirm={onConfirm}
        onCancel={onCancel}
      />
    </EditorProvider>
  );
};

const MemoEditorImpl: React.FC<MemoEditorProps> = ({
  className,
  cacheKey,
  memoName,
  parentMemoName,
  autoFocus,
  placeholder,
  onConfirm,
  onCancel,
}) => {
  const t = useTranslate();
  const queryClient = useQueryClient();
  const currentUser = useCurrentUser();
  const editorRef = useRef<EnhancedEditorRefActions>(null);
  const { state, actions, dispatch } = useEditorContext();
  const { userGeneralSetting } = useAuth();

  // Link memo dialog state
  const [linkMemoDialogOpen, setLinkMemoDialogOpen] = useState(false);

  // Get default visibility from user settings
  const defaultVisibility = userGeneralSetting?.memoVisibility ? convertVisibilityFromString(userGeneralSetting.memoVisibility) : undefined;

  useMemoInit(editorRef, memoName, cacheKey, currentUser?.name ?? "", autoFocus, defaultVisibility);

  // Auto-save content to localStorage
  useAutoSave(state.content, currentUser?.name ?? "", cacheKey);

  // Track virtual keyboard height for mobile
  const keyboardHeight = useVirtualKeyboard();

  // Focus mode management with body scroll lock
  useFocusMode(state.ui.isFocusMode);

  // Link memo hook
  const { searchText, setSearchText, isFetching, filteredMemos, addMemoRelation } = useLinkMemo({
    isOpen: linkMemoDialogOpen,
    currentMemoName: memoName,
    existingRelations: state.metadata.relations,
    onAddRelation: (relation) => {
      dispatch(actions.addRelation(relation));
    },
  });

  const handleToggleFocusMode = () => {
    dispatch(actions.toggleFocusMode());
  };

  // Handle AI tag insertion
  const handleInsertTags = (tags: string[]) => {
    if (!editorRef.current || tags.length === 0) return;

    // Insert tags at cursor position
    const tagString = tags.map((tag) => `#${tag}`).join(" ");
    editorRef.current.insertAndSelect(tagString);
  };

  // Handle AI format content
  const handleFormatContent = (formattedContent: string) => {
    if (!editorRef.current) return;

    // Replace entire content with formatted content
    editorRef.current.setContent(formattedContent);
  };

  // Handle visibility change
  const handleVisibilityChange = (visibility: typeof state.metadata.visibility) => {
    dispatch(actions.setMetadata({ visibility }));
  };

  // Handle link memo
  const handleLinkMemo = () => {
    setLinkMemoDialogOpen(true);
  };

  useKeyboard(editorRef, { onSave: handleSave });

  async function handleSave() {
    // Validate before saving
    const { valid, reason } = validationService.canSave(state);
    if (!valid) {
      toast.error(reason || "Cannot save");
      return;
    }

    dispatch(actions.setLoading("saving", true));

    try {
      const result = await memoService.save(state, { memoName, parentMemoName });

      if (!result.hasChanges) {
        toast.error(t("editor.no-changes-detected"));
        onCancel?.();
        return;
      }

      // Clear localStorage cache on successful save
      cacheService.clear(cacheService.key(currentUser?.name ?? "", cacheKey));

      // Invalidate React Query cache to refresh memo lists across the app
      const invalidationPromises = [
        queryClient.invalidateQueries({ queryKey: memoKeys.lists() }),
        queryClient.invalidateQueries({ queryKey: userKeys.stats() }),
      ];

      // If this was a comment, also invalidate comments query for parent memo
      if (parentMemoName) {
        invalidationPromises.push(queryClient.invalidateQueries({ queryKey: memoKeys.comments(parentMemoName) }));
      }

      await Promise.all(invalidationPromises);

      // Reset editor state to initial values
      dispatch(actions.reset());

      // Notify parent component of successful save
      onConfirm?.(result.memoName);
    } catch (error) {
      handleError(error, toast.error, {
        context: "Failed to save memo",
        fallbackMessage: errorService.getErrorMessage(error),
      });
    } finally {
      dispatch(actions.setLoading("saving", false));
    }
  }

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
          state.ui.isFocusMode
            ? cn(FOCUS_MODE_STYLES.container.base, FOCUS_MODE_STYLES.container.spacing, "flex flex-col bg-background")
            : className,
        )}
        style={{
          paddingBottom: !state.ui.isFocusMode && keyboardHeight > 0 ? `${keyboardHeight + 16}px` : undefined,
        }}
      >
        {/* Exit button is absolutely positioned in top-right corner when active */}
        <FocusModeExitButton isActive={state.ui.isFocusMode} onToggle={handleToggleFocusMode} title={t("editor.exit-focus-mode")} />

        {/* Editor content grows to fill available space in focus mode */}
        <EditorContent ref={editorRef} placeholder={placeholder} autoFocus={autoFocus} />

        {/* Metadata and toolbar grouped together at bottom */}
        <div className="w-full flex flex-col gap-2 shrink-0">
          <EditorMetadata memoName={memoName} />
          <EditorToolbar
            onSave={handleSave}
            onCancel={onCancel}
            memoName={memoName}
            onInsertTags={handleInsertTags}
            onFormatContent={handleFormatContent}
            onVisibilityChange={handleVisibilityChange}
            onToggleFocusMode={handleToggleFocusMode}
            onLinkMemo={handleLinkMemo}
          />
        </div>
      </div>

      {/* Link Memo Dialog */}
      <LinkMemoDialog
        open={linkMemoDialogOpen}
        onOpenChange={setLinkMemoDialogOpen}
        searchText={searchText}
        onSearchChange={setSearchText}
        filteredMemos={filteredMemos}
        isFetching={isFetching}
        onSelectMemo={addMemoRelation}
      />
    </>
  );
};

export default MemoEditor;

export { default as FocusModeEditor } from "./FocusModeEditor";
export type { EditorMode } from "./hooks/useEditorMode";
export { useEditorMode } from "./hooks/useEditorMode";
