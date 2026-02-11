/**
 * Editor types - 编辑器核心类型定义
 */

export interface EditorRefActions {
  focus(): void;
  insertText(text: string): void;
  getSelection(): { start: number; end: number } | null;
  setSelection(start: number, end: number): void;
  getContent(): string;
  setContent(content: string): void;
}
