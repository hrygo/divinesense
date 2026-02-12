/**
 * EnhancedEditor - 增强的编辑器组件
 *
 * 在原生 textarea 基础上增加：
 * - 光标位置追踪（像素级）
 * - 上下文获取（用于命令/建议）
 * - 虚拟高度自动调整
 * - IME 组合状态处理
 */
import { forwardRef, memo, useCallback, useEffect, useImperativeHandle, useRef, useState } from "react";
import type { EditorProps } from "../types/components";
import type { CursorContext, CursorPosition, EnhancedEditorRefActions, VisibleRange } from "./editor-types";

// 缓存常量
const MIRROR_DIV_STYLES = [
  "direction",
  "boxSizing",
  "width",
  "height",
  "overflowX",
  "overflowY",
  "borderTopWidth",
  "borderRightWidth",
  "borderBottomWidth",
  "borderLeftWidth",
  "paddingTop",
  "paddingRight",
  "paddingBottom",
  "paddingLeft",
  "fontStyle",
  "fontVariant",
  "fontWeight",
  "fontStretch",
  "fontSize",
  "lineHeight",
  "fontFamily",
  "textAlign",
  "textTransform",
  "textDecoration",
  "letterSpacing",
  "wordSpacing",
  "lineBreak",
  "whiteSpace",
  "wordWrap",
] as const;

/**
 * 创建镜像 div 用于计算光标位置
 */
function createMirrorDiv(textarea: HTMLTextAreaElement): HTMLDivElement {
  const mirror = document.createElement("div");
  const styles = window.getComputedStyle(textarea);

  // 复制所有相关样式
  MIRROR_DIV_STYLES.forEach((prop) => {
    // biome-ignore lint/suspicious/noExplicitAny: CSSStyleDeclaration prop access requires any
    mirror.style[prop as any] = styles.getPropertyValue(prop);
  });

  mirror.style.position = "absolute";
  mirror.style.visibility = "hidden";
  mirror.style.whiteSpace = "pre-wrap";
  mirror.style.wordBreak = "break-word";
  mirror.style.top = "0";
  mirror.style.left = "0";

  document.body.appendChild(mirror);
  return mirror;
}

/**
 * Enhanced Editor Component
 */
const EnhancedEditorComponent = forwardRef<EnhancedEditorRefActions, EditorProps>(
  ({ className, initialContent, placeholder, onContentChange, onPaste, onCompositionStart, onCompositionEnd }, ref) => {
    const textareaRef = useRef<HTMLTextAreaElement>(null);
    const mirrorDivRef = useRef<HTMLDivElement | null>(null);
    const [content, setContent] = useState(initialContent);

    // 获取或创建镜像 div
    const getMirrorDiv = useCallback(() => {
      if (!mirrorDivRef.current && textareaRef.current) {
        mirrorDivRef.current = createMirrorDiv(textareaRef.current);
      }
      return mirrorDivRef.current;
    }, []);

    /**
     * 计算光标像素位置
     */
    const calculateCursorPosition = useCallback((): CursorPosition | null => {
      const textarea = textareaRef.current;
      const mirror = getMirrorDiv();
      if (!textarea || !mirror) return null;

      const cursorPos = textarea.selectionStart;
      if (cursorPos === null) return null;

      const textBeforeCursor = content.slice(0, cursorPos);
      mirror.textContent = textBeforeCursor;

      // 创建一个临时 span 来精确定位
      const span = document.createElement("span");
      span.textContent = "|";
      mirror.appendChild(span);

      const spanRect = span.getBoundingClientRect();
      const textareaRect = textarea.getBoundingClientRect();

      // 计算行和列信息
      const lines = textBeforeCursor.split("\n");
      const currentLine = lines[lines.length - 1] || "";
      const column = currentLine.length;
      const line = lines.length - 1;

      // 清理 span
      mirror.removeChild(span);

      return {
        line,
        column,
        top: spanRect.top - textareaRect.top + textarea.scrollTop,
        left: spanRect.left - textareaRect.left,
        height: spanRect.height,
      };
    }, [content, getMirrorDiv]);

    /**
     * 获取光标上下文
     */
    const getContextAtCursor = useCallback((): CursorContext | null => {
      const textarea = textareaRef.current;
      if (!textarea) return null;

      const cursorPos = textarea.selectionStart;
      const fullText = content;

      // 获取光标前的文本
      const textBeforeCursor = fullText.slice(0, cursorPos);
      // �获取光标后的文本
      const textAfterCursor = fullText.slice(cursorPos);

      // 获取当前单词（简单匹配字母数字下划线）
      const wordMatch = textBeforeCursor.match(/\w+$/);
      const word = wordMatch ? wordMatch[0] : "";

      // 获取当前行
      const lastNewlineIndex = textBeforeCursor.lastIndexOf("\n");
      const line = lastNewlineIndex >= 0 ? textBeforeCursor.slice(lastNewlineIndex + 1) : textBeforeCursor;
      const lineStart = lastNewlineIndex >= 0 ? lastNewlineIndex + 1 : 0;

      return {
        before: textBeforeCursor,
        after: textAfterCursor,
        word,
        line,
        lineStart,
        lineEnd: cursorPos,
      };
    }, [content]);

    /**
     * 获取可见范围（用于虚拟滚动）
     */
    const getVisibleRange = useCallback((): VisibleRange => {
      const textarea = textareaRef.current;
      if (!textarea) return { start: 0, end: content.length };

      const lineHeight = parseInt(window.getComputedStyle(textarea).lineHeight) || 20;
      const scrollTop = textarea.scrollTop;
      const clientHeight = textarea.clientHeight;

      const startLine = Math.max(0, Math.floor(scrollTop / lineHeight));
      const visibleLines = Math.ceil(clientHeight / lineHeight);

      // 找到起始位置
      let startPos = 0;
      let currentLine = 0;
      const allLines = content.split("\n");

      while (currentLine < startLine && currentLine < allLines.length) {
        startPos += allLines[currentLine]?.length ?? 0;
        startPos += 1; // newline character
        currentLine++;
      }

      // 找到结束位置
      let endPos = startPos;
      const endLine = Math.min(allLines.length, startLine + visibleLines + 2); // +2 for overscan

      while (currentLine < endLine && currentLine < allLines.length) {
        endPos += allLines[currentLine]?.length ?? 0;
        endPos += 1;
        currentLine++;
      }

      return { start: startPos, end: Math.min(endPos, content.length) };
    }, [content]);

    /**
     * 获取所有文本行
     */
    const getLines = useCallback((): string[] => {
      return content.split("\n");
    }, [content]);

    /**
     * 获取指定行
     */
    const getLine = useCallback(
      (lineNumber: number): string | null => {
        const lines = content.split("\n");
        return lines[lineNumber] ?? null;
      },
      [content],
    );

    /**
     * 在光标位置插入文本
     */
    const insertText = useCallback(
      (text: string): void => {
        const textarea = textareaRef.current;
        if (!textarea) return;

        const start = textarea.selectionStart;
        const before = content.slice(0, start);
        const after = content.slice(textarea.selectionEnd);
        const newContent = before + text + after;

        setContent(newContent);
        onContentChange(newContent);

        // 设置光标位置
        const newCursorPos = start + text.length;
        setTimeout(() => {
          const newEl = textareaRef.current;
          if (newEl) {
            newEl.selectionStart = newCursorPos;
            newEl.selectionEnd = newCursorPos;
            newEl.focus();
          }
        }, 0);
      },
      [content, onContentChange],
    );

    /**
     * 在光标位置替换文本
     */
    const replaceTextAtCursor = useCallback(
      (searchText: string, replacement: string, options?: { selectAfter?: boolean; scrollIntoView?: boolean }): void => {
        const textarea = textareaRef.current;
        if (!textarea) return;

        const start = textarea.selectionStart;
        const searchStart = start === null ? 0 : start;
        const beforeSearch = content.slice(0, searchStart);
        const searchIndex = beforeSearch.toLowerCase().lastIndexOf(searchText.toLowerCase());

        const replaceStart = searchIndex !== -1 && beforeSearch[searchIndex - 1] === searchText ? start : searchIndex + searchText.length;

        const newContent = beforeSearch + replacement + content.slice(replaceStart);
        setContent(newContent);
        onContentChange(newContent);

        // 滚动到替换位置（如果需要）
        if (options?.selectAfter) {
          const newEl = textareaRef.current;
          if (newEl) {
            newEl.selectionStart = replaceStart + replacement.length;
            newEl.selectionEnd = replaceStart + replacement.length;
            newEl.focus();
          }
        }
      },
      [content, onContentChange],
    );

    /**
     * 滚动到指定行
     */
    const scrollToLine = useCallback(
      (lineNumber: number) => {
        const textarea = textareaRef.current;
        if (!textarea) return;

        const lines = content.split("\n");
        if (lineNumber < 0 || lineNumber >= lines.length) return;

        let position = 0;
        for (let i = 0; i < lineNumber; i++) {
          position += (lines[i]?.length ?? 0) + 1; // +1 for newline
        }
        textarea.scrollTop = position;
        textarea.focus();
      },
      [content],
    );

    /**
     * 自动调整高度（虚拟高度）
     */
    const adjustHeight = useCallback(() => {
      const textarea = textareaRef.current;
      if (!textarea) return;

      // 重置高度以计算 scrollHeight
      textarea.style.height = "auto";
      const scrollHeight = textarea.scrollHeight;

      // 应用计算的高度（有最小值限制）
      const newHeight = Math.max(60, Math.min(scrollHeight, 800)); // 最小 60px，最大 800px
      textarea.style.height = `${newHeight}px`;
    }, [content]);

    // 处理内容变化
    const handleChange = useCallback(
      (e: React.ChangeEvent<HTMLTextAreaElement>) => {
        const newContent = e.target.value;
        setContent(newContent);
        onContentChange(newContent);
        adjustHeight();
      },
      [onContentChange, adjustHeight],
    );

    // 处理粘贴事件
    const handlePaste = useCallback(
      (e: React.ClipboardEvent<HTMLTextAreaElement>) => {
        if (onPaste) {
          onPaste(e);
        }
      },
      [onPaste],
    );

    // 处理键盘事件
    const handleKeyDown = useCallback(
      (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
        // Tab 键支持（插入两个空格）
        if (e.key === "Tab") {
          e.preventDefault();
          insertText("  ");
        }
      },
      [insertText],
    );

    // 自动调整高度（内容变化时）
    useEffect(() => {
      adjustHeight();
    }, [content, adjustHeight]);

    /**
     * 暴露增强的方法
     */
    useImperativeHandle(
      ref,
      () => ({
        // EnhancedEditorRefActions
        getCursorPosition: calculateCursorPosition,
        getContextAtCursor,
        insertText,
        replaceTextAtCursor,
        getVisibleRange,
        scrollToLine,
        getLines,
        getLine,
        insertAndSelect: insertText,
        // BaseEditorRefActions
        focus: () => {
          textareaRef.current?.focus();
        },
        getSelection: () => {
          const textarea = textareaRef.current;
          if (!textarea) return null;
          return {
            start: textarea.selectionStart,
            end: textarea.selectionEnd,
          };
        },
        setSelection: (start: number, end: number) => {
          const textarea = textareaRef.current;
          if (!textarea) return;
          textarea.selectionStart = start;
          textarea.selectionEnd = end;
        },
        getContent: () => content,
        setContent: (newContent: string) => {
          setContent(newContent);
          onContentChange(newContent);
        },
      }),
      [
        calculateCursorPosition,
        getContextAtCursor,
        insertText,
        replaceTextAtCursor,
        getVisibleRange,
        scrollToLine,
        getLines,
        getLine,
        insertText,
        content,
        onContentChange,
      ],
    );

    return (
      <textarea
        ref={textareaRef}
        className={className}
        value={content}
        placeholder={placeholder}
        onChange={handleChange}
        onPaste={handlePaste}
        onKeyDown={handleKeyDown}
        onCompositionStart={onCompositionStart}
        onCompositionEnd={onCompositionEnd}
        style={{
          width: "100%",
          minHeight: "60px",
          resize: "none",
          overflow: "auto",
        }}
      />
    );
  },
);

EnhancedEditorComponent.displayName = "EnhancedEditor";

const EnhancedEditor = memo(EnhancedEditorComponent);

export default EnhancedEditor;
