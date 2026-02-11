/**
 * Tag Suggestions Plugin - 标签建议插件
 *
 * 处理 # 触发的标签建议
 */

import type { EditorPlugin, PluginContext, TriggerConfig } from "../../core/editor-types";
import { TriggerType } from "../../core/editor-types";
import { COMMAND_GROUPS, commandToSuggestionItem, SLASH_COMMANDS } from "../slash-commands/commands";

// 从 Slash Commands 中提取命令用作标签参考
const TAG_COMMANDS = SLASH_COMMANDS.filter((cmd) => {
  const group = COMMAND_GROUPS.formatting?.includes(cmd);
  return group && cmd.id !== "clear"; // 排除清空命令
});

/**
 * 标签建议触发器配置
 */
export const tagSuggestionTriggerConfig: TriggerConfig = {
  type: TriggerType.HASH,
  char: "#",
  minChars: 0,
  maxItems: 10,
};

/**
 * 创建标签建议插件实例
 * @param t - i18n 翻译函数（由调用方提供）
 */
export function createTagSuggestionPlugin(t?: (key: string) => string): EditorPlugin {
  return {
    id: "tag-suggestions",
    name: "Tag Suggestions",
    priority: 90, // 低于 slash commands，高于其他插件

    triggers: [tagSuggestionTriggerConfig],

    /**
     * 获取建议项
     */
    getSuggestions: async (_context: PluginContext, _query: string) => {
      // 将命令转换为标签建议项
      const suggestions = TAG_COMMANDS.slice(0, tagSuggestionTriggerConfig.maxItems).map((cmd) => {
        const item = commandToSuggestionItem(cmd);

        // 翻译标签
        if (t) {
          item.label = t(cmd.labelKey);
          if (cmd.descriptionKey) {
            item.description = t(cmd.descriptionKey);
          }
        }

        // 将命令 ID 转换为标签格式（#tag）
        item.id = `tag-${cmd.id}`;
        // 添加 # 前缀
        item.label = `#${item.label}`;

        return item;
      });

      return suggestions;
    },

    /**
     * 键盘事件处理
     * 返回 true 表示事件已被处理
     */
    onKeyDown: (context: PluginContext, event: KeyboardEvent) => {
      // ESC 关闭建议菜单
      if (event.key === "Escape") {
        // biome-ignore lint/suspicious/noExplicitAny: Plugin context extensions
        const suggestionsState = (context as any).suggestions;
        if (suggestionsState?.isOpen) {
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

      // 导航建议（复用 slash commands 逻辑）
      // biome-ignore lint/suspicious/noExplicitAny: Plugin context extensions
      const suggestionsState = (context as any).suggestions;
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
        if (event.key === "Enter" || event.key === "Tab") {
          event.preventDefault();
          const selectedItem = suggestionsState.items[suggestionsState.selectedIndex];
          if (selectedItem) {
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

      // 检测 # 触发
      if (event.key === "#") {
        const cursorPos = context.editor?.getContextAtCursor();
        if (cursorPos) {
          context.dispatch({
            type: "SET_SUGGESTIONS",
            payload: {
              isOpen: true,
              trigger: TriggerType.HASH,
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
      // biome-ignore lint/suspicious/noExplicitAny: Plugin context extensions
      const suggestionsState = (context as any).suggestions;

      // 如果建议菜单打开，更新查询
      if (suggestionsState?.isOpen && suggestionsState.trigger === TriggerType.HASH) {
        const triggerIndex = content.lastIndexOf("#");
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
export const tagSuggestionPlugin = createTagSuggestionPlugin();
