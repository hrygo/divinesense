import type { Memo } from "@/types/proto/api/v1/memo_service_pb";

export interface MemoViewProps {
  memo: Memo;
  compact?: boolean;
  showCreator?: boolean;
  showVisibility?: boolean;
  showPinned?: boolean;
  showNsfwContent?: boolean;
  hideActionMenu?: boolean; // 隐藏操作菜单（用于 MemoBlock 等已有操作按钮的场景）
  hideInteractionButtons?: boolean; // 隐藏互动按钮（表情、评论），用于 MemoBlock
  className?: string;
  parentPage?: string;
}

export interface MemoHeaderProps {
  showCreator?: boolean;
  showVisibility?: boolean;
  showPinned?: boolean;
  hideActionMenu?: boolean; // 隐藏操作菜单（用于 MemoBlock 等已有操作按钮的场景）
  hideInteractionButtons?: boolean; // 隐藏互动按钮（表情、评论），用于 MemoBlock
  onEdit: () => void;
  onGotoDetail: () => void;
  onUnpin: () => void;
  onToggleNsfwVisibility?: () => void;
}

export interface MemoBodyProps {
  compact?: boolean;
  onContentClick: (e: React.MouseEvent) => void;
  onContentDoubleClick: (e: React.MouseEvent) => void;
  onToggleNsfwVisibility: () => void;
}
