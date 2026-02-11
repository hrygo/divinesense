/**
 * MemoEditor types - 笔记编辑器类型定义
 */

export interface MemoEditorProps {
  className?: string;
  cacheKey?: string;
  placeholder?: string;
  memoName?: string;
  parentMemoName?: string;
  autoFocus?: boolean;
  onConfirm?: (memoName: string) => void;
  onCancel?: () => void;
  onSubmit?: (content: string) => void;
}
