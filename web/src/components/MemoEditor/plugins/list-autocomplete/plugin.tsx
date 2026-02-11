/**
 * List Autocomplete Plugin - 列表自动完成插件
 *
 * 检测 - 、1. 等列表标记并自动格式化下一行
 */

import { ListIcon } from "lucide-react";
import React from "react";
import type { EditorPlugin, PluginContext, TriggerConfig } from "../../core/editor-types";
import { TriggerType } from "../../core/editor-types";

/**
 * 列表自动完成触发器配置
 */
export const listAutocompleteTriggerConfig: TriggerConfig = {
  type: TriggerType.CUSTOM, // 自定义触发器
  char: "", // 空字符串，由代码检测
  minChars: 0,
  maxItems: 5,
};

/**
 * 列表项建议
 */
interface ListSuggestionItem {
  id: string;
  label: string;
  description: string;
  icon: React.ReactNode;
  insertText: string; // 要插入的文本（不包括列表标记）
  // biome-ignore lint/suspicious/noExplicitAny: Editor ref type from plugin context
  action: (editor: any) => void;
}

// 建议项定义
const LIST_SUGGESTIONS: ListSuggestionItem[] = [
  {
    id: "bullet",
    label: "无序列表",
    description: "插入无序列表项",
    icon: <ListIcon className="h-4 w-4" />,
    insertText: "- ",
    // biome-ignore lint/suspicious/noExplicitAny: Editor ref type
    action: (editor: any) => editor?.insertText("- "),
  },
  {
    id: "numbered",
    label: "有序列表",
    description: "插入有序列表项",
    icon: <ListIcon className="h-4 w-4" />,
    insertText: "1. ",
    // biome-ignore lint/suspicious/noExplicitAny: Editor ref type
    action: (editor: any) => editor?.insertText("1. "),
  },
  {
    id: "todo",
    label: "待办列表",
    description: "插入待办列表项",
    icon: <ListIcon className="h-4 w-4" />,
    insertText: "- [ ] ",
    // biome-ignore lint/suspicious/noExplicitAny: Editor ref type
    action: (editor: any) => editor?.insertText("- [ ] "),
  },
  {
    id: "todo-done",
    label: "已完成",
    description: "插入已完成项",
    icon: <ListIcon className="h-4 w-4" />,
    insertText: "- [x] ",
    // biome-ignore lint/suspicious/noExplicitAny: Editor ref type
    action: (editor: any) => editor?.insertText("- [x] "),
  },
];

/**
 * 检测当前行是否应该显示列表建议
 */
function shouldShowListSuggestions(line: string): boolean {
  const trimmed = line.trim();
  // 空行或已有列表标记
  if (!trimmed || /^[-*+]\s/.test(trimmed) || /^\d+\.\s/.test(trimmed)) {
    return false;
  }
  return true;
}

/**
 * 创建列表自动完成插件实例
 * @param t - i18n 翻译函数（由调用方提供）
 */
// biome-ignore lint/suspicious/noExplicitAny: Plugin accepts any translation function type
export function createListAutocompletePlugin(t?: any): EditorPlugin {
  return {
    id: "list-autocomplete",
    name: "List Autocomplete",
    priority: 80, // 优先级低于 slash 和 tag 插件
    triggers: [listAutocompleteTriggerConfig],

    getSuggestions: async (_context: PluginContext, _query: string) => {
      // query 在这个插件中被忽略（通过检测当前行）
      return LIST_SUGGESTIONS.map((item) => ({
        ...item,
        // 翻译
        label: t ? t(`editor.list.${item.id}`) : item.label,
        description: t ? t(`editor.list.${item.id}-desc`) : item.description,
      }));
    },

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

      // 导航
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
            // 执行插入操作
            // biome-ignore lint/suspicious/noExplicitAny: Suggestion action type
            (selectedItem as any).action(context.editor);
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

      return false;
    },

    onContentChange: (context: PluginContext, content: string) => {
      const cursorContext = context.editor?.getContextAtCursor();
      if (!cursorContext) return;

      const { lineStart } = cursorContext;
      const currentLineText = content.slice(lineStart);

      // 只在行首且应该显示建议时才显示
      if (lineStart === 0 && shouldShowListSuggestions(currentLineText)) {
        context.dispatch({
          type: "SET_SUGGESTIONS",
          payload: {
            isOpen: true,
            trigger: TriggerType.CUSTOM,
            query: "",
            items: LIST_SUGGESTIONS.map((item) => ({
              ...item,
              label: t ? t(`editor.list.${item.id}`) : item.label,
              description: t ? t(`editor.list.${item.id}-desc`) : item.description,
            })),
            selectedIndex: 0,
            position: context.editor?.getCursorPosition?.() || null,
          },
        });
      } else {
        // 其他情况关闭菜单
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
      }
    },
  };
}

/**
 * 默认导出 - 插件实例（不使用 i18n）
 */
export const listAutocompletePlugin = createListAutocompletePlugin();
