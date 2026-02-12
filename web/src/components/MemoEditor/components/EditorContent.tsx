import { forwardRef } from "react";
import type { EnhancedEditorRefActions } from "../core/editor-types";
import { useDragAndDrop } from "../hooks";
import type { EditorContentProps } from "../types";
import { EditorWithPlugins } from "./EditorWithPlugins";

export const EditorContent = forwardRef<EnhancedEditorRefActions, EditorContentProps>(({ placeholder }, ref) => {
  // Handle file drops (no-op for now, can be implemented later)
  const handleDrop = (_files: FileList) => {
    // TODO: implement file drop handling
  };

  const dragHandlers = useDragAndDrop(handleDrop);

  return (
    <div className="w-full flex flex-col flex-1 relative" {...dragHandlers.dragHandlers}>
      <EditorWithPlugins ref={ref} placeholder={placeholder ?? ""} />
    </div>
  );
});

EditorContent.displayName = "EditorContent";
