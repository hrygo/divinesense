import type { Location, Memo, Visibility } from "@/types/proto/api/v1/memo_service_pb";
import type { EditorRefActions } from "./editor";
import type { LocationState } from "./insert-menu";

export interface MemoEditorProps {
  className?: string;
  cacheKey?: string;
  placeholder?: string;
  memoName?: string;
  parentMemoName?: string;
  autoFocus?: boolean;
  onConfirm?: (memoName: string) => void;
  onCancel?: () => void;
}

export interface EditorContentProps {
  placeholder?: string;
  autoFocus?: boolean;
}

export interface EditorToolbarProps {
  onSave?: () => void;
  onCancel?: () => void;
  onUploadAttachment?: () => void;
  onLinkMemo?: () => void;
  onToggleFocusMode?: () => void;
  onVisibilityChange?: (visibility: import("@/types/proto/api/v1/memo_service_pb").Visibility) => void;
  onOpenMobileTools?: () => void;
  onInsertTags?: (tags: string[]) => void;
  onFormatContent?: (formattedContent: string) => void;
  currentVisibility?: import("@/types/proto/api/v1/memo_service_pb").Visibility;
  memoName?: string;
}

export interface EditorMetadataProps {
  memoName?: string;
}

export interface FocusModeOverlayProps {
  isActive: boolean;
  onToggle: () => void;
}

export interface FocusModeExitButtonProps {
  isActive: boolean;
  onToggle: () => void;
  title: string;
}

export interface LinkMemoDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  searchText: string;
  onSearchChange: (text: string) => void;
  filteredMemos: Memo[];
  isFetching: boolean;
  onSelectMemo: (memo: Memo) => void;
}

export interface LocationDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  state: LocationState;
  locationInitialized: boolean;
  onPositionChange: (position: { lat: number; lng: number }) => void;
  onUpdateCoordinate: (type: "lat" | "lng", value: string) => void;
  onPlaceholderChange: (placeholder: string) => void;
  onCancel: () => void;
  onConfirm: () => void;
}

export interface InsertMenuProps {
  isUploading?: boolean;
  location?: Location;
  onLocationChange: (location?: Location) => void;
  onToggleFocusMode?: () => void;
  memoName?: string;
}

export interface TagSuggestionsProps {
  editorRef: React.RefObject<HTMLTextAreaElement>;
  editorActions: React.ForwardedRef<EditorRefActions>;
}

export interface SlashCommandsProps {
  editorRef: React.RefObject<HTMLTextAreaElement>;
  editorActions: React.ForwardedRef<EditorRefActions>;
  commands: unknown[];
}

export interface CompactEditorProps {
  placeholder?: string;
  onSave?: () => void;
  onExpand?: () => void;
  keyboardHeight?: number;
}

export interface EditorProps {
  className: string;
  initialContent: string;
  placeholder: string;
  onContentChange: (content: string) => void;
  // biome-ignore lint/suspicious/noExplicitAny: Event types from textarea
  onPaste: (event: any) => void;
  // biome-ignore lint/suspicious/noExplicitAny: KeyboardEvent from textarea
  onKeyDown?: (e: any) => void;
  onCompositionStart?: () => void;
  onCompositionEnd?: () => void;
  // biome-ignore lint/suspicious/noExplicitAny: Ref types vary
  ref?: React.RefObject<any>;
}

export interface VisibilitySelectorProps {
  value: Visibility;
  onChange: (visibility: Visibility) => void;
  onOpenChange?: (open: boolean) => void;
}

export interface MobileToolsSheetProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onUploadFile: () => void;
  onLinkMemo: () => void;
  onAddLocation: () => void;
  onVisibilityChange: (visibility: Visibility) => void;
  keyboardHeight?: number;
}
