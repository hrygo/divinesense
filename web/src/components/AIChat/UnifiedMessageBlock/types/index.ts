/**
 * UnifiedMessageBlock Type Definitions
 */

/** Timeline node types */
export type TimelineNodeType = "user" | "thinking" | "tool" | "answer" | "error";

/** Props for TimelineNode component */
export interface TimelineNodeProps {
  /** Node type determines styling */
  type: TimelineNodeType;
  /** Optional icon component to render inside node */
  icon?: React.ReactNode;
  /** Optional additional CSS classes */
  className?: string;
  /** Optional click handler */
  onClick?: () => void;
}

/** Props for StreamingProgressBar component */
export interface StreamingProgressBarProps {
  /** Progress percentage (0-100) */
  progress: number;
  /** Whether the stream is active */
  isActive: boolean;
  /** Optional additional CSS classes */
  className?: string;
}

/** Props for PendingSkeleton component */
export interface PendingSkeletonProps {
  /** Optional custom message to display */
  message?: string;
  /** Optional additional CSS classes */
  className?: string;
}

/** Tool call data structure */
export interface ToolCallData {
  toolName: string;
  toolId?: string;
  input?: Record<string, unknown>;
  output?: string;
  exitCode?: number;
  isError?: boolean;
  duration?: number;
}

/** Props for ToolCallCard component */
export interface ToolCallCardProps {
  data: ToolCallData;
  className?: string;
}

/** Props for InlineToolCall component */
export interface InlineToolCallProps {
  toolName: string;
  isError?: boolean;
  className?: string;
  inputSummary?: string;
  filePath?: string;
}

/** Options for useBlockCollapse hook */
export interface UseBlockCollapseOptions {
  /** Whether this is the latest block (should be expanded by default) */
  isLatest: boolean;
  /** Whether the block content is streaming */
  isStreaming?: boolean;
  /** Optional external control for collapse state */
  externalCollapsed?: boolean;
  /** Callback when collapse state changes */
  onCollapseChange?: (collapsed: boolean) => void;
}

/** Return value for useBlockCollapse hook */
export interface UseBlockCollapseReturn {
  /** Current collapse state */
  collapsed: boolean;
  /** Toggle collapse state */
  toggleCollapse: () => void;
  /** Set collapse state */
  setCollapsed: (value: boolean) => void;
  /** Generate preview text from content */
  generatePreview: (content: string) => string;
}

/** Options for useStreamingProgress hook */
export interface UseStreamingProgressOptions {
  /** Whether streaming is active */
  isStreaming: boolean;
  /** Current content being streamed */
  content: string;
  /** Expected max content length for progress calculation (optional) */
  expectedMaxLength?: number;
}

/** Return value for useStreamingProgress hook */
export interface UseStreamingProgressReturn {
  /** Current progress percentage (0-100) */
  progress: number;
  /** Whether progress is being shown */
  isShowingProgress: boolean;
}

/** Options for useKeyboardNav hook */
export interface UseKeyboardNavOptions {
  /** Block ID for navigation */
  blockId: string;
  /** Whether this block can receive keyboard focus */
  isFocusable?: boolean;
  /** Callback when keyboard shortcut is triggered */
  onShortcut?: (action: string) => void;
  /** Reference to the block element */
  blockRef?: React.RefObject<HTMLElement>;
}

/** Return value for useKeyboardNav hook */
export interface UseKeyboardNavReturn {
  /** Props to spread on the block container */
  keyboardProps: {
    tabIndex: number;
    onKeyDown: (e: React.KeyboardEvent) => void;
    "data-block-id"?: string;
  };
  /** Programmatically focus this block */
  focusBlock: () => void;
}
