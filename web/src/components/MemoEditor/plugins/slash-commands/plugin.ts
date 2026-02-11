/**
 * Slash Commands Plugin - 斜杠命令插件
 *
 * 处理 / 触发的斜杠命令
 */

import type { EditorPlugin, PluginContext, TriggerConfig } from "../../core/editor-types";
import { TriggerType } from "../../core/editor-types";
import { commandToSuggestionItem, searchCommands } from "./commands";

/**
 * 斜杠命令插件配置
 */
export const slashCommandTriggerConfig: TriggerConfig = {
  type: TriggerType.SLASH,
  char: "/",
  minChars: 0, // 立即触发，显示所有命令
  maxItems: 10,
};

/**
 * 创建斜杠命令插件实例
 * @param t - i18n 翻译函数（由调用方提供）
 */
// biome-ignore lint/suspicious/noExplicitAny: Plugin accepts any translation function type
export function createSlashCommandPlugin(t?: any): EditorPlugin {
  return {
    id: "slash-commands",
    name: "Slash Commands",
    priority: 100, // 最高优先级，优先于其他触发器

    triggers: [slashCommandTriggerConfig],

    /**
     * 获取建议项
     */
    getSuggestions: async (_context: PluginContext, query: string) => {
      const commands = searchCommands(query);

      // 将命令转换为建议项，并翻译标签
      return commands.slice(0, slashCommandTriggerConfig.maxItems).map((cmd) => {
        const item = commandToSuggestionItem(cmd);

        // 翻译标签（如果提供了 t 函数）
        if (t) {
          item.label = t(cmd.labelKey);
          if (cmd.descriptionKey) {
            item.description = t(cmd.descriptionKey);
          }
        }

        return item;
      });
    },

    /**
     * 键盘事件处理
     * 返回 true 表示事件已被处理
     */
    onKeyDown: (context: PluginContext, event: KeyboardEvent) => {
      // ESC 关闭建议菜单
      if (event.key === "Escape") {
        context.dispatch({
          type: "SET_SUGGESTIONS",
          payload: {
            isOpen: false,
            trigger: null,
            query: "",
            items: [],
            selectedIndex: -1,
          },
        });
        return true;
      }

      // 导航建议
      const suggestionsState = context.suggestions;
      if (suggestionsState?.isOpen) {
        if (event.key === "ArrowDown") {
          event.preventDefault();
          context.dispatch({
            type: "SET_SUGGESTIONS",
            payload: {
              selectedIndex: Math.min(suggestionsState.selectedIndex + 1, suggestionsState.items.length - 1),
            },
          });
          return true;
        }
        if (event.key === "ArrowUp") {
          event.preventDefault();
          context.dispatch({
            type: "SET_SUGGESTIONS",
            payload: {
              selectedIndex: Math.max(suggestionsState.selectedIndex - 1, 0),
            },
          });
          return true;
        }
        if (event.key === "Enter") {
          event.preventDefault();
          const selectedItem = suggestionsState.items[suggestionsState.selectedIndex];
          if (selectedItem && context.editor) {
            selectedItem.action(context.editor);
            // 关闭菜单
            context.dispatch({
              type: "SET_SUGGESTIONS",
              payload: {
                isOpen: false,
                trigger: null,
                query: "",
                items: [],
                selectedIndex: -1,
              },
            });
          }
          return true;
        }
        // Tab 键也确认选择
        if (event.key === "Tab") {
          event.preventDefault();
          const selectedItem = suggestionsState.items[suggestionsState.selectedIndex];
          if (selectedItem && context.editor) {
            selectedItem.action(context.editor);
            // 关闭菜单
            context.dispatch({
              type: "SET_SUGGESTIONS",
              payload: {
                isOpen: false,
                trigger: null,
                query: "",
                items: [],
                selectedIndex: -1,
              },
            });
          }
          return true;
        }
      }

      // 检测斜杠触发
      if (event.key === "/") {
        // 触发建议菜单
        const cursorPos = context.editor?.getContextAtCursor();
        if (cursorPos) {
          context.dispatch({
            type: "SET_SUGGESTIONS",
            payload: {
              isOpen: true,
              trigger: TriggerType.SLASH,
              query: "",
              items: [], // 将由 getSuggestions 填充
              selectedIndex: 0,
              position: context.editor?.getCursorPosition?.() || null,
            },
          });
        }
      }

      return false;
    },

    /**
     * 内容变化处理
     */
    onContentChange: (context: PluginContext, content: string) => {
      const suggestionsState = context.suggestions;

      // 如果建议菜单打开，更新查询
      if (suggestionsState?.isOpen && suggestionsState.trigger === TriggerType.SLASH) {
        // 从内容中提取 / 后的查询文本
        const triggerIndex = content.lastIndexOf("/");
        if (triggerIndex !== -1) {
          const query = content.slice(triggerIndex + 1);
          context.dispatch({
            type: "SET_SUGGESTIONS",
            payload: {
              query,
            },
          });
        }
      }
    },
  };
}

/**
 * 默认导出 - 插件实例（不使用 i18n）
 */
export const slashCommandPlugin = createSlashCommandPlugin();
