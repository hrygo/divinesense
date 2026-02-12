/**
 * Enhanced Editor Types - 编辑器核心类型定义
 *
 * 扩展现有的 EditorRefActions，增加光标位置、上下文获取等能力
 */

import type { EditorRefActions as BaseEditorRefActions } from "../types/editor";

/**
 * 光标位置信息 - 用于定位建议菜单
 */
export interface CursorPosition {
  line: number; // 行号（从 0 开始）
  column: number; // 列号（从 0 开始）
  top: number; // 相对于编辑器的像素位置
  left: number; // 相对于编辑器的像素位置
  height: number; // 行高
}

/**
 * 光标上下文 - 用于命令和建议
 */
export interface CursorContext {
  before: string; // 光标前的文本
  after: string; // 光标后的文本
  word: string; // 当前单词
  line: string; // 当前行
  lineStart: number; // 当前行开始位置
  lineEnd: number; // 当前行结束位置（光标位置）
}

/**
 * 编辑器可见范围 - 用于虚拟滚动优化
 */
export interface VisibleRange {
  start: number;
  end: number;
}

/**
 * 增强的编辑器 Ref Actions
 * 在原有接口基础上扩展
 */
export interface EnhancedEditorRefActions extends BaseEditorRefActions {
  /**
   * 获取光标像素位置（用于定位弹出菜单）
   */
  getCursorPosition(): CursorPosition | null;

  /**
   * 获取光标所在单词/上下文
   */
  getContextAtCursor(): CursorContext | null;

  /**
   * 在光标位置替换文本
   */
  replaceTextAtCursor(searchText: string, replacement: string, options?: { selectAfter?: boolean; scrollIntoView?: boolean }): void;

  /**
   * 获取可见行范围（用于虚拟滚动优化）
   */
  getVisibleRange(): VisibleRange;

  /**
   * 滚动到指定行
   */
  scrollToLine(lineNumber: number): void;

  /**
   * 获取所有文本行
   */
  getLines(): string[];

  /**
   * 获取指定行的文本
   */
  getLine(lineNumber: number): string | null;

  /**
   * 在指定位置插入文本
   */
  insertAndSelect(text: string): void;
}

/**
 * 触发器类型
 */
export const enum TriggerType {
  SLASH = "/", // 斜杠命令
  HASH = "#", // 标签建议
  BRACKET = "[", // 链接建议
  AT = "@", // 提及建议
  CUSTOM = "custom", // 自定义触发器
}

/**
 * 触发器配置
 */
export interface TriggerConfig {
  type: TriggerType;
  char: string;
  minChars?: number; // 最小触发字符数
  maxItems?: number; // 最大建议数量
}

/**
 * 建议菜单状态
 */
export interface SuggestionMenuState {
  isOpen: boolean;
  trigger: TriggerType | null;
  query: string;
  items: SuggestionItem[];
  selectedIndex: number;
  position: { top: number; left: number } | null;
}

/**
 * 插件上下文 - 提供编辑器状态访问
 */
export type EditorDispatch = React.Dispatch<{
  type: string;
  payload?: unknown;
}>;

export interface PluginContext {
  editor: EnhancedEditorRefActions | null | undefined;
  content: string;
  cursor: CursorPosition | null;
  selection: { start: number; end: number } | null;
  suggestions?: SuggestionMenuState; // 可选的建议菜单状态
  dispatch: EditorDispatch;
}

/**
 * 编辑器事件
 */
export interface EditorEvents {
  onChange?: (content: string) => void;
  onCursorChange?: (position: CursorPosition) => void;
  onSelectionChange?: (selection: { start: number; end: number }) => void;
  onPaste?: (event: React.ClipboardEvent) => void;
  onKeyDown?: (event: React.KeyboardEvent) => void;
  onCompositionStart?: () => void;
  onCompositionEnd?: () => void;
}

/**
 * 编辑器配置
 */
export interface EditorConfig {
  placeholder?: string;
  autoFocus?: boolean;
  minHeight?: number;
  maxHeight?: number;
  enableTab?: boolean;
  tabSize?: number;
  enableAutoResize?: boolean;
  enableSyntaxHighlight?: boolean;
  mobileOptimized?: boolean;
}

/**
 * 触发器配置
 */
export interface TriggerConfig {
  type: TriggerType;
  char: string;
  minChars?: number;
  maxItems?: number;
}

/**
 * 建议项配置
 */
export interface SuggestionItem {
  id: string;
  label: string;
  description?: string;
  icon?: React.ReactNode;
  keywords?: string[];
  action: (editor: EnhancedEditorRefActions) => void | Promise<void>;
  shortcut?: string;
}

/**
 * 编辑器插件接口
 */
export interface EditorPlugin {
  id: string;
  name: string;
  priority?: number;
  triggers?: TriggerConfig[];
  getSuggestions?: (context: PluginContext, query: string) => SuggestionItem[] | Promise<SuggestionItem[]>;
  onKeyDown?: (context: PluginContext, event: KeyboardEvent) => boolean;
  onContentChange?: (context: PluginContext, content: string) => void;
  onPaste?: (context: PluginContext, event: ClipboardEvent) => boolean;
  onCompositionStart?: (context: PluginContext) => void;
  onCompositionEnd?: (context: PluginContext) => void;
}
