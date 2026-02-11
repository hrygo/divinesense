/**
 * Plugins Index - 插件导出
 *
 * 集中导出所有编辑器插件
 */

// Re-export types for convenience
export type { EditorPlugin, PluginContext, SuggestionItem, TriggerConfig } from "../core/editor-types";

// List Autocomplete Plugin
export {
  createListAutocompletePlugin,
  listAutocompletePlugin,
  listAutocompleteTriggerConfig,
} from "./list-autocomplete/plugin";
// Slash Commands Plugin
export {
  createSlashCommandPlugin,
  slashCommandPlugin,
  slashCommandTriggerConfig,
} from "./slash-commands/plugin";
// Tag Suggestions Plugin
export {
  createTagSuggestionPlugin,
  tagSuggestionPlugin,
  tagSuggestionTriggerConfig,
} from "./tag-suggestions/plugin";
