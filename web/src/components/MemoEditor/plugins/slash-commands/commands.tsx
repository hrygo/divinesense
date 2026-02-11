/**
 * Slash Commands - 斜杠命令定义
 *
 * 定义所有可用的斜杠命令及其行为
 */

import {
  BoldIcon,
  CheckSquare,
  Code2Icon,
  Heading1Icon,
  Heading2Icon,
  Heading3Icon,
  ItalicIcon,
  Link2Icon,
  ListIcon,
  ListOrderedIcon,
  ListTodoIcon,
  QuoteIcon,
  StrikethroughIcon,
  TableIcon,
  TextIcon,
  TextWrapIcon,
  XIcon,
} from "lucide-react";
import React from "react";
import type { SuggestionItem } from "@/components/MemoEditor/core/editor-types";

/**
 * 命令类型
 */
export enum CommandType {
  // 文本格式
  BOLD = "bold",
  ITALIC = "italic",
  STRIKETHROUGH = "strikethrough",
  CODE = "code",
  CODE_BLOCK = "code_block",
  // 标题
  HEADING_1 = "heading_1",
  HEADING_2 = "heading_2",
  HEADING_3 = "heading_3",
  // 列表
  BULLET_LIST = "bullet_list",
  NUMBERED_LIST = "numbered_list",
  TODO_LIST = "todo_list",
  TODO_CHECKED = "todo_checked",
  // 其他
  QUOTE = "quote",
  LINK = "link",
  TABLE = "table",
  DIVIDER = "divider",
  ALIGN_LEFT = "align_left",
  ALIGN_CENTER = "align_center",
  ALIGN_RIGHT = "align_right",
  CLEAR = "clear",
}

/**
 * 命令定义
 */
export interface Command {
  id: string;
  type: CommandType;
  label: string;
  labelKey: string;
  description?: string;
  descriptionKey?: string;
  icon: React.ReactNode;
  keywords?: string[];
  shortcut?: string;
  /**
   * 执行命令
   * @param editor - 编辑器实例
   * @returns 插入的文本或操作完成后的回调
   */
  // biome-ignore lint/suspicious/noExplicitAny: editor can be different editor types
  execute: (editor: any) => string | void | Promise<string | void>;
}

// Helper function to get markdown symbols safely
// Using char codes to avoid TypeScript JSX parsing issues
const getMD = () => ({
  ast2: "**",
  ast1: "*",
  tilde: "~~",
  backtick: "`",
  hash1: String("# ") || "# ",
  hash2: String("## ") || "## ",
  hash3: String("### ") || "### ",
  dash: "- ",
  dot1: "1. ",
  todo: String("-[ ] ") || "- [ ] ",
  todoDone: String("-[x] ") || "- [x] ",
  gt: String.fromCharCode(62) + " ", // '>'
  link: "[描述](url)",
  hr: "---",
  table: String("| 列1 | 列2 |") + String.fromCharCode(10) + "| --- | --- |" + String.fromCharCode(10) + "| 内容 | 内容 |",
});

// Type-safe accessors for markdown symbols
const MD = getMD();
const BOLD_MARKER = MD.ast2;
const ITALIC_MARKER = MD.ast1;
const STRIKETHROUGH_MARKER = MD.tilde;
const CODE_MARKER = MD.backtick;
const H1_MARKER = MD.hash1;
const H2_MARKER = MD.hash2;
const H3_MARKER = MD.hash3;
const BULLET_MARKER = MD.dash;
const NUMBERED_MARKER = MD.dot1;
const TODO_MARKER = MD.todo;
const TODO_DONE_MARKER = MD.todoDone;
const QUOTE_MARKER = MD.gt;
const LINK_MARKER = MD.link;

/**
 * 文本格式命令
 */
const BOLD_COMMAND: Command = {
  id: "bold",
  type: CommandType.BOLD,
  label: "粗体",
  labelKey: "editor.command.bold",
  description: "将选中文本变为粗体",
  descriptionKey: "editor.command.bold-desc",
  icon: <BoldIcon className="h-4 w-4" />,
  keywords: ["bold", "b", "粗体"],
  execute: () => BOLD_MARKER,
};

const ITALIC_COMMAND: Command = {
  id: "italic",
  type: CommandType.ITALIC,
  label: "斜体",
  labelKey: "editor.command.italic",
  description: "将选中文本变为斜体",
  descriptionKey: "editor.command.italic-desc",
  icon: <ItalicIcon className="h-4 w-4" />,
  keywords: ["italic", "i", "斜体"],
  execute: () => ITALIC_MARKER,
};

const STRIKETHROUGH_COMMAND: Command = {
  id: "strikethrough",
  type: CommandType.STRIKETHROUGH,
  label: "删除线",
  labelKey: "editor.command.strikethrough",
  description: "给选中文本添加删除线",
  descriptionKey: "editor.command.strikethrough-desc",
  icon: <StrikethroughIcon className="h-4 w-4" />,
  keywords: ["strikethrough", "s", "删除线"],
  execute: () => STRIKETHROUGH_MARKER,
};

const CODE_COMMAND: Command = {
  id: "code",
  type: CommandType.CODE,
  label: "行内代码",
  labelKey: "editor.command.code",
  description: "插入行内代码标记",
  descriptionKey: "editor.command.code-desc",
  icon: <Code2Icon className="h-4 w-4" />,
  keywords: ["code", "c", "代码", "inline"],
  execute: () => CODE_MARKER,
};

const CODE_BLOCK_COMMAND: Command = {
  id: "code_block",
  type: CommandType.CODE_BLOCK,
  label: "代码块",
  labelKey: "editor.command.code-block",
  description: "插入代码块",
  descriptionKey: "editor.command.code-block-desc",
  icon: <TextIcon className="h-4 w-4" />,
  keywords: ["codeblock", "cb", "代码块", "block"],
  execute: () => {
    const lines = ["```", "", ""];
    return lines.join("\n");
  },
};

const HEADING_1_COMMAND: Command = {
  id: "heading_1",
  type: CommandType.HEADING_1,
  label: "一级标题",
  labelKey: "editor.command.heading-1",
  description: "插入一级标题（#）",
  descriptionKey: "editor.command.heading-1-desc",
  icon: <Heading1Icon className="h-4 w-4" />,
  keywords: ["h1", "heading1", "一级标题", "大标题"],
  shortcut: "1",
  execute: () => H1_MARKER,
};

const HEADING_2_COMMAND: Command = {
  id: "heading_2",
  type: CommandType.HEADING_2,
  label: "二级标题",
  labelKey: "editor.command.heading-2",
  description: "插入二级标题（##）",
  descriptionKey: "editor.command.heading-2-desc",
  icon: <Heading2Icon className="h-4 w-4" />,
  keywords: ["h2", "heading2", "二级标题"],
  shortcut: "2",
  execute: () => H2_MARKER,
};

const HEADING_3_COMMAND: Command = {
  id: "heading_3",
  type: CommandType.HEADING_3,
  label: "三级标题",
  labelKey: "editor.command.heading-3",
  description: "插入三级标题（###）",
  descriptionKey: "editor.command.heading-3-desc",
  icon: <Heading3Icon className="h-4 w-4" />,
  keywords: ["h3", "heading3", "三级标题"],
  shortcut: "3",
  execute: () => H3_MARKER,
};

const BULLET_LIST_COMMAND: Command = {
  id: "bullet_list",
  type: CommandType.BULLET_LIST,
  label: "无序列表",
  labelKey: "editor.command.bullet-list",
  description: "插入无序列表项",
  descriptionKey: "editor.command.bullet-list-desc",
  icon: <ListIcon className="h-4 w-4" />,
  keywords: ["bullet", "ul", "无序列表", "列表", "b"],
  shortcut: "-",
  execute: () => BULLET_MARKER,
};

const NUMBERED_LIST_COMMAND: Command = {
  id: "numbered_list",
  type: CommandType.NUMBERED_LIST,
  label: "有序列表",
  labelKey: "editor.command.numbered-list",
  description: "插入有序列表项",
  descriptionKey: "editor.command.numbered-list-desc",
  icon: <ListOrderedIcon className="h-4 w-4" />,
  keywords: ["numbered", "ol", "有序列表", "数字列表"],
  shortcut: "1.",
  execute: () => NUMBERED_MARKER,
};

const TODO_LIST_COMMAND: Command = {
  id: "todo_list",
  type: CommandType.TODO_LIST,
  label: "待办列表",
  labelKey: "editor.command.todo-list",
  description: "插入待办事项",
  descriptionKey: "editor.command.todo-list-desc",
  icon: <ListTodoIcon className="h-4 w-4" />,
  keywords: ["todo", "task", "待办", "任务", "[]"],
  shortcut: "[]",
  execute: () => TODO_MARKER,
};

const TODO_CHECKED_COMMAND: Command = {
  id: "todo_checked",
  type: CommandType.TODO_CHECKED,
  label: "已完成待办",
  labelKey: "editor.command.todo-checked",
  description: "插入已完成事项",
  descriptionKey: "editor.command.todo-checked-desc",
  icon: <CheckSquare className="h-4 w-4" />,
  keywords: ["done", "checked", "已完成", "x"],
  shortcut: "[x]",
  execute: () => TODO_DONE_MARKER,
};

const QUOTE_COMMAND: Command = {
  id: "quote",
  type: CommandType.QUOTE,
  label: "引用",
  labelKey: "editor.command.quote",
  description: "插入引用块",
  descriptionKey: "editor.command.quote-desc",
  icon: <QuoteIcon className="h-4 w-4" />,
  keywords: ["quote", "q", "引用"],
  shortcut: ">",
  execute: () => QUOTE_MARKER,
};

const LINK_COMMAND: Command = {
  id: "link",
  type: CommandType.LINK,
  label: "链接",
  labelKey: "editor.command.link",
  description: "插入链接",
  descriptionKey: "editor.command.link-desc",
  icon: <Link2Icon className="h-4 w-4" />,
  keywords: ["link", "url", "链接"],
  execute: () => LINK_MARKER,
};

const DIVIDER_COMMAND: Command = {
  id: "divider",
  type: CommandType.DIVIDER,
  label: "分割线",
  labelKey: "editor.command.divider",
  description: "插入水平分割线",
  descriptionKey: "editor.command.divider-desc",
  icon: <TextWrapIcon className="h-4 w-4" />,
  keywords: ["divider", "hr", "分割线", "---"],
  execute: () => {
    return "---\n\n\n";
  },
};

const TABLE_COMMAND: Command = {
  id: "table",
  type: CommandType.TABLE,
  label: "表格",
  labelKey: "editor.command.table",
  description: "插入表格",
  descriptionKey: "editor.command.table-desc",
  icon: <TableIcon className="h-4 w-4" />,
  keywords: ["table", "表格"],
  execute: () => MD.table,
};

const CLEAR_COMMAND: Command = {
  id: "clear",
  type: CommandType.CLEAR,
  label: "清空",
  labelKey: "editor.command.clear",
  description: "清空编辑器内容",
  descriptionKey: "editor.command.clear-desc",
  icon: <XIcon className="h-4 w-4" />,
  keywords: ["clear", "清空", "删除全部"],
  // biome-ignore lint/suspicious/noExplicitAny: editor can be different editor types
  execute: async (editor: any) => {
    // 特殊处理：清空内容
    if (editor.setContent) {
      editor.setContent(String());
    }
    return "";
  },
};

/**
 * 所有斜杠命令
 */
export const SLASH_COMMANDS: Command[] = [
  BOLD_COMMAND,
  ITALIC_COMMAND,
  STRIKETHROUGH_COMMAND,
  CODE_COMMAND,
  CODE_BLOCK_COMMAND,
  HEADING_1_COMMAND,
  HEADING_2_COMMAND,
  HEADING_3_COMMAND,
  BULLET_LIST_COMMAND,
  NUMBERED_LIST_COMMAND,
  TODO_LIST_COMMAND,
  TODO_CHECKED_COMMAND,
  QUOTE_COMMAND,
  LINK_COMMAND,
  DIVIDER_COMMAND,
  TABLE_COMMAND,
  CLEAR_COMMAND,
];

/**
 * 按类型分组命令
 */
export const COMMAND_GROUPS = {
  formatting: [BOLD_COMMAND, ITALIC_COMMAND, STRIKETHROUGH_COMMAND, CODE_COMMAND],
  headings: [HEADING_1_COMMAND, HEADING_2_COMMAND, HEADING_3_COMMAND],
  lists: [BULLET_LIST_COMMAND, NUMBERED_LIST_COMMAND, TODO_LIST_COMMAND, TODO_CHECKED_COMMAND],
  other: [QUOTE_COMMAND, LINK_COMMAND, DIVIDER_COMMAND, TABLE_COMMAND, CLEAR_COMMAND],
} as const;

/**
 * 将命令转换为建议项
 */
export function commandToSuggestionItem(command: Command): SuggestionItem {
  return {
    id: command.id,
    label: command.label,
    description: command.description,
    icon: command.icon,
    keywords: command.keywords,
    // biome-ignore lint/suspicious/noExplicitAny: editor can be different editor types
    action: async (editor: any) => {
      const result = command.execute(editor);
      if (typeof result === "string") {
        editor.insertText(result);
      } else if (result instanceof Promise) {
        const text = await result;
        if (text && editor.insertText) {
          editor.insertText(text);
        }
      }
    },
    shortcut: command.shortcut,
  };
}

/**
 * 搜索命令
 */
export function searchCommands(query: string): Command[] {
  if (!query) return SLASH_COMMANDS;

  const lowerQuery = query.toLowerCase();
  return SLASH_COMMANDS.filter((cmd) => {
    // 匹配 ID
    if (cmd.id.includes(lowerQuery)) return true;
    // 匹配关键词
    if (cmd.keywords?.some((kw) => kw.toLowerCase().includes(lowerQuery))) return true;
    // 匹配标签
    if (cmd.label.toLowerCase().includes(lowerQuery)) return true;
    return false;
  });
}
