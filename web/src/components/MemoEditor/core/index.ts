/**
 * Core Layer - 编辑器核心层
 *
 * 导出所有核心类型和组件
 */

// 导出插件
export {
  createSlashCommandPlugin,
  slashCommandPlugin,
  slashCommandTriggerConfig,
} from "../plugins";
export { default as EnhancedEditor } from "./EnhancedEditor";
export type {
  CursorContext,
  CursorPosition,
  EditorConfig,
  EditorEvents,
  EnhancedEditorRefActions,
  PluginContext,
  SuggestionItem,
  SuggestionMenuState,
  TriggerConfig,
  TriggerType,
  VisibleRange,
} from "./editor-types";
export type {
  SuggestionMenuProps,
  SuggestionMenuRefActions,
  SuggestionMenuStyles,
} from "./suggestion-menu-types";

// 移除重复导出 - 已在第 21-25 行导出
