import { forwardRef, memo, useEffect, useImperativeHandle, useRef, useState } from "react";
import type { EditorProps } from "../types/components";
import type { EditorRefActions } from "../types/editor";

const EditorComponent = (
  { className, initialContent, placeholder, onContentChange, onPaste, onCompositionStart, onCompositionEnd }: EditorProps,
  ref: React.ForwardedRef<EditorRefActions>,
) => {
  const textareaRef = useRef<HTMLTextAreaElement>(null);
  const [content, setContent] = useState(initialContent);

  // Expose methods via imperative handle
  useImperativeHandle(
    ref,
    () => ({
      focus: () => textareaRef.current?.focus(),
      insertText: (text: string) => {
        const el = textareaRef.current;
        if (!el) return;
        const start = el.selectionStart;
        const end = el.selectionEnd;
        const before = content.slice(0, start);
        const after = content.slice(end);
        setContent(before + text + after);
        setTimeout(() => {
          const newEl = textareaRef.current;
          if (newEl) {
            newEl.selectionStart = newEl.selectionEnd = start + text.length;
          }
        }, 0);
      },
      getSelection: () => {
        const el = textareaRef.current;
        if (!el) return null;
        return { start: el.selectionStart, end: el.selectionEnd };
      },
      setSelection: (start: number, end: number) => {
        const el = textareaRef.current;
        if (!el) return;
        el.selectionStart = start;
        el.selectionEnd = end;
        el.focus();
      },
      getContent: () => content,
      setContent: setContent,
    }),
    [content],
  );

  useEffect(() => {
    setContent(initialContent);
  }, [initialContent]);

  const handleChange = (e: React.ChangeEvent<HTMLTextAreaElement>) => {
    const newContent = e.target.value;
    setContent(newContent);
    onContentChange(newContent);
  };

  const handlePaste = (e: React.ClipboardEvent) => {
    onPaste?.(e);
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    // Allow tab to insert 2 spaces instead of changing focus
    if (e.key === "Tab") {
      e.preventDefault();
      const el = textareaRef.current;
      if (!el) return;
      const start = el.selectionStart;
      const end = el.selectionEnd;
      const before = content.slice(0, start);
      const after = content.slice(end);
      setContent(before + "  " + after);
      setTimeout(() => {
        const newEl = textareaRef.current;
        if (newEl) {
          newEl.selectionStart = newEl.selectionEnd = start + 2;
          newEl.focus();
        }
      }, 0);
    }
  };

  return (
    <textarea
      ref={textareaRef}
      value={content}
      onChange={handleChange}
      onPaste={handlePaste}
      onKeyDown={handleKeyDown}
      onCompositionStart={onCompositionStart}
      onCompositionEnd={onCompositionEnd}
      placeholder={placeholder}
      className={className}
      style={{
        resize: "none",
        overflow: "auto",
      }}
    />
  );
};

export const Editor = memo(forwardRef<EditorRefActions, EditorProps>(EditorComponent));
Editor.displayName = "Editor";
