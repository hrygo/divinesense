import { forwardRef } from "react";
import { useDragAndDrop } from "../hooks";
import { useEditorContext } from "../state/context";
import { Editor } from "../state/Editor";
import type { EditorContentProps } from "../types";
import type { EditorRefActions } from "../types/editor";

export const EditorContent = forwardRef<EditorRefActions, EditorContentProps>(({ placeholder }, ref) => {
  const { state, actions, dispatch } = useEditorContext();

  // Handle file drops (no-op for now, can be implemented later)
  const handleDrop = (_files: FileList) => {
    // TODO: implement file drop handling
  };

  const dragHandlers = useDragAndDrop(handleDrop);

  const handleCompositionStart = () => {
    dispatch(actions.setComposing(true));
  };

  const handleCompositionEnd = () => {
    dispatch(actions.setComposing(false));
  };

  const handleContentChange = (content: string) => {
    dispatch(actions.updateContent(content));
  };

  const handlePaste = (_e: React.ClipboardEvent) => {
    // Paste handling is managed by Editor component internally
  };

  const handleKeyDown = (_e: React.KeyboardEvent) => {
    // Keyboard handling is managed externally
  };

  return (
    <div className="w-full flex flex-col flex-1" {...dragHandlers.dragHandlers}>
      <Editor
        ref={ref}
        className="memo-editor-content"
        initialContent={state.content}
        placeholder={placeholder ?? ""}
        onContentChange={handleContentChange}
        onPaste={handlePaste}
        onKeyDown={handleKeyDown}
        onCompositionStart={handleCompositionStart}
        onCompositionEnd={handleCompositionEnd}
      />
    </div>
  );
});

EditorContent.displayName = "EditorContent";
